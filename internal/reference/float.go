package reference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	floatCacheSchema       = "massive-float-cache-v1"
	floatSourceIdentity    = "massive-stocks-float-experimental"
	maximumFloatPages      = 20
	maximumFloatRecords    = 100_000
	maximumFloatPageBytes  = 4 << 20
	maximumFloatPageResult = 5_000
)

// FloatProvenance distinguishes a successful refresh from a validated fallback.
// Cache values are deliberately stale: cache age is operational, while each
// effective date remains the provider's disclosure date.
type FloatProvenance string

const (
	FloatFresh       FloatProvenance = "fresh"
	FloatCache       FloatProvenance = "cache"
	FloatUnavailable FloatProvenance = "unavailable"
)

// FloatFact is one validated Massive public-free-float observation.
type FloatFact struct {
	Ticker           string
	FreeFloat        float64
	FreeFloatPercent *float64
	EffectiveDate    string
	Provider         string
	RetrievedAt      time.Time
	Provenance       FloatProvenance
}

// FloatLookup is immutable by construction. Facts returns detached values and
// Lookup returns a detached optional percentage.
type FloatLookup struct {
	facts      []FloatFact
	provenance FloatProvenance
	retrieved  time.Time
}

// NewFloatLookup builds the immutable handoff accepted by the engine. It is
// intentionally strict so tests and alternate startup compositions cannot
// bypass the same attribution and provenance constraints as FloatResolver.
func NewFloatLookup(facts []FloatFact, provenance FloatProvenance, retrievedAt time.Time) (FloatLookup, error) {
	if (provenance != FloatFresh && provenance != FloatCache) || retrievedAt.IsZero() || retrievedAt != retrievedAt.UTC() {
		return FloatLookup{}, errors.New("invalid Float lookup provenance")
	}
	result := FloatLookup{provenance: provenance, retrieved: retrievedAt, facts: make([]FloatFact, len(facts))}
	for i, fact := range facts {
		if !validSymbol(fact.Ticker) || fact.FreeFloat <= 0 || math.IsNaN(fact.FreeFloat) || math.IsInf(fact.FreeFloat, 0) ||
			fact.Provider != floatSourceIdentity || !fact.RetrievedAt.Equal(retrievedAt) || fact.Provenance != provenance {
			return FloatLookup{}, errors.New("invalid Float lookup fact")
		}
		if fact.FreeFloatPercent != nil && (*fact.FreeFloatPercent < 0 || *fact.FreeFloatPercent > 100 || math.IsNaN(*fact.FreeFloatPercent) || math.IsInf(*fact.FreeFloatPercent, 0)) {
			return FloatLookup{}, errors.New("invalid Float lookup percent")
		}
		if fact.EffectiveDate != "" {
			if _, err := time.Parse("2006-01-02", fact.EffectiveDate); err != nil {
				return FloatLookup{}, errors.New("invalid Float lookup effective date")
			}
		}
		result.facts[i] = cloneFloatFact(fact)
	}
	sort.Slice(result.facts, func(i, j int) bool { return result.facts[i].Ticker < result.facts[j].Ticker })
	for i := 1; i < len(result.facts); i++ {
		if result.facts[i-1].Ticker == result.facts[i].Ticker {
			return FloatLookup{}, errors.New("ambiguous Float lookup ticker")
		}
	}
	return result, nil
}

func (l FloatLookup) Provenance() FloatProvenance { return l.provenance }
func (l FloatLookup) RetrievedAt() time.Time      { return l.retrieved }
func (l FloatLookup) Len() int                    { return len(l.facts) }
func (l FloatLookup) Facts() []FloatFact {
	result := make([]FloatFact, len(l.facts))
	for i := range l.facts {
		result[i] = cloneFloatFact(l.facts[i])
	}
	return result
}
func (l FloatLookup) Lookup(ticker string) (FloatFact, bool) {
	i := sort.Search(len(l.facts), func(i int) bool { return l.facts[i].Ticker >= ticker })
	if i == len(l.facts) || l.facts[i].Ticker != ticker {
		return FloatFact{}, false
	}
	return cloneFloatFact(l.facts[i]), true
}

func cloneFloatFact(f FloatFact) FloatFact {
	if f.FreeFloatPercent != nil {
		v := *f.FreeFloatPercent
		f.FreeFloatPercent = &v
	}
	return f
}

// FloatResolver performs one bounded startup refresh and validated cache
// fallback. It owns no engine state and makes no readiness decision.
type FloatResolver struct {
	BaseURL    string
	APIKey     string
	DataDir    string
	HTTPClient *http.Client
	Now        func() time.Time
}

type floatCache struct {
	SchemaVersion string           `json:"schema_version"`
	Source        string           `json:"source"`
	RetrievedAt   string           `json:"retrieved_at"`
	Facts         []floatCacheFact `json:"facts"`
}

type floatCacheFact struct {
	Ticker           string   `json:"ticker"`
	FreeFloat        float64  `json:"free_float"`
	FreeFloatPercent *float64 `json:"free_float_percent,omitempty"`
	EffectiveDate    string   `json:"effective_date,omitempty"`
}

// Resolve returns unavailable rather than failing startup when neither fresh
// nor cached Float is usable. The optional enrichment never gates the scanner.
func (r *FloatResolver) Resolve(ctx context.Context, universe Universe) FloatLookup {
	unavailable := FloatLookup{provenance: FloatUnavailable}
	if ctx == nil || !universe.IsCurrent() || len(universe.symbols) == 0 {
		return unavailable
	}
	directory, cacheErr := inspectCacheDirectory(r.DataDir, "float")
	var cached FloatLookup
	if cacheErr == nil {
		cached, cacheErr = r.readCache(filepath.Join(directory, "latest.json"), universe.symbols)
	}
	fresh, err := r.fetch(ctx, universe.symbols)
	if err == nil {
		// Re-prepare even when inspection succeeded so the publication path is
		// independently revalidated immediately before its atomic replace.
		if prepared, prepareErr := prepareCacheDirectory(r.DataDir, "float"); prepareErr == nil {
			directory = prepared
			_ = r.publishCache(directory, fresh)
		}
		return fresh
	}
	if cacheErr == nil {
		return cached
	}
	return unavailable
}

func (r *FloatResolver) fetch(parent context.Context, universe []string) (FloatLookup, error) {
	base, err := validatedBaseURL(r.BaseURL)
	if err != nil || r.APIKey == "" || r.HTTPClient == nil {
		return FloatLookup{}, errors.New("invalid Float acquisition configuration")
	}
	operation, cancel := context.WithTimeout(parent, operationTimeout)
	defer cancel()
	allowed := make(map[string]struct{}, len(universe))
	for _, symbol := range universe {
		allowed[symbol] = struct{}{}
	}
	start := *base
	start.Path = "/stocks/vX/float"
	query := start.Query()
	query.Set("limit", "5000")
	query.Set("sort", "ticker.asc")
	start.RawQuery = query.Encode()
	next := start.String()
	seenURLs := make(map[string]struct{})
	facts := make(map[string]FloatFact)
	ambiguous := make(map[string]struct{})
	total := 0
	retrievedAt := r.now()
	for page := 0; next != ""; page++ {
		if err := validateFloatFetchBounds(page, 0, 0, total); err != nil {
			return FloatLookup{}, err
		}
		requestURL, err := validateFloatPageURL(base, next, seenURLs)
		if err != nil {
			return FloatLookup{}, err
		}
		seenURLs[requestURL.String()] = struct{}{}
		attempt, cancelAttempt := context.WithTimeout(operation, requestAttemptTimeout)
		req, err := http.NewRequestWithContext(attempt, http.MethodGet, requestURL.String(), nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+r.APIKey)
		}
		var response *http.Response
		if err == nil {
			response, err = r.HTTPClient.Do(req)
		}
		if err != nil {
			cancelAttempt()
			return FloatLookup{}, err
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maximumFloatPageBytes+1))
		closeErr := response.Body.Close()
		cancelAttempt()
		if response.StatusCode != http.StatusOK || readErr != nil || closeErr != nil || validateFloatFetchBounds(page, len(body), 0, total) != nil {
			return FloatLookup{}, errors.New("Float response failed bounded acquisition")
		}
		rows, nextURL, err := decodeFloatPage(body)
		if err != nil || validateFloatFetchBounds(page, 0, len(rows), total) != nil {
			return FloatLookup{}, errors.New("Float response is incomplete or invalid")
		}
		total += len(rows)
		for _, raw := range rows {
			fact, ticker, valid := normalizeFloatRow(raw, retrievedAt)
			if ticker == "" {
				continue
			}
			if _, wanted := allowed[ticker]; !wanted {
				continue
			}
			if !valid {
				delete(facts, ticker)
				ambiguous[ticker] = struct{}{}
				continue
			}
			if _, bad := ambiguous[ticker]; bad {
				continue
			}
			if prior, duplicate := facts[ticker]; duplicate {
				_ = prior
				delete(facts, ticker)
				ambiguous[ticker] = struct{}{}
				continue
			}
			facts[ticker] = fact
		}
		next = nextURL
	}
	result := FloatLookup{provenance: FloatFresh, retrieved: retrievedAt, facts: make([]FloatFact, 0, len(facts))}
	for _, fact := range facts {
		result.facts = append(result.facts, fact)
	}
	sort.Slice(result.facts, func(i, j int) bool { return result.facts[i].Ticker < result.facts[j].Ticker })
	return result, nil
}

func validateFloatFetchBounds(page, responseBytes, pageRows, priorRows int) error {
	if page < 0 || page >= maximumFloatPages {
		return errors.New("Float pagination exceeds page bound")
	}
	if responseBytes < 0 || responseBytes > maximumFloatPageBytes {
		return errors.New("Float response exceeds byte bound")
	}
	if pageRows < 0 || pageRows > maximumFloatPageResult {
		return errors.New("Float response exceeds result bound")
	}
	if priorRows < 0 || priorRows > maximumFloatRecords-pageRows {
		return errors.New("Float response exceeds record bound")
	}
	return nil
}

func decodeFloatPage(body []byte) ([]json.RawMessage, string, error) {
	members, err := objectMembers(body)
	if err != nil {
		return nil, "", err
	}
	statusRaw, err := requiredMember(members, "status")
	if err != nil {
		return nil, "", err
	}
	status, err := decodeMember[string](statusRaw)
	if err != nil || status != "OK" {
		return nil, "", errors.New("Float response status is not OK")
	}
	resultsRaw, err := requiredMember(members, "results")
	if err != nil {
		return nil, "", err
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(resultsRaw, &rows); err != nil || rows == nil {
		return nil, "", errors.New("Float results are not an array")
	}
	nextRaw, present, err := optionalMember(members, "next_url")
	if err != nil {
		return nil, "", err
	}
	var next string
	if present {
		next, err = decodeMember[string](nextRaw)
		if err != nil {
			return nil, "", err
		}
	}
	return rows, next, nil
}

func normalizeFloatRow(raw json.RawMessage, retrievedAt time.Time) (FloatFact, string, bool) {
	members, err := objectMembers(raw)
	if err != nil {
		return FloatFact{}, "", false
	}
	tickerRaw, err := requiredMember(members, "ticker")
	if err != nil {
		return FloatFact{}, "", false
	}
	ticker, err := decodeMember[string](tickerRaw)
	if err != nil || !validSymbol(ticker) {
		return FloatFact{}, "", false
	}
	floatRaw, err := requiredMember(members, "free_float")
	if err != nil {
		return FloatFact{}, ticker, false
	}
	number, err := decodeMember[json.Number](floatRaw)
	if err != nil {
		return FloatFact{}, ticker, false
	}
	freeFloat, err := number.Float64()
	if err != nil || freeFloat <= 0 || math.IsNaN(freeFloat) || math.IsInf(freeFloat, 0) {
		return FloatFact{}, ticker, false
	}
	fact := FloatFact{Ticker: ticker, FreeFloat: freeFloat, Provider: floatSourceIdentity, RetrievedAt: retrievedAt, Provenance: FloatFresh}
	if rawPercent, present, memberErr := optionalMember(members, "free_float_percent"); memberErr != nil {
		return FloatFact{}, ticker, false
	} else if present {
		value, decodeErr := decodeMember[json.Number](rawPercent)
		if decodeErr != nil {
			return FloatFact{}, ticker, false
		}
		percent, parseErr := value.Float64()
		if parseErr != nil || percent < 0 || percent > 100 || math.IsNaN(percent) || math.IsInf(percent, 0) {
			return FloatFact{}, ticker, false
		}
		fact.FreeFloatPercent = &percent
	}
	if rawDate, present, memberErr := optionalMember(members, "effective_date"); memberErr != nil {
		return FloatFact{}, ticker, false
	} else if present {
		date, decodeErr := decodeMember[string](rawDate)
		if decodeErr != nil {
			return FloatFact{}, ticker, false
		}
		if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
			return FloatFact{}, ticker, false
		}
		fact.EffectiveDate = date
	}
	return fact, ticker, true
}

func validateFloatPageURL(base *url.URL, raw string, seen map[string]struct{}) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if !parsed.IsAbs() {
		parsed = base.ResolveReference(parsed)
	}
	if parsed.Scheme != base.Scheme || parsed.Host != base.Host || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("Float continuation changed origin")
	}
	query := parsed.Query()
	if query.Get("apiKey") != "" || query.Get("api_key") != "" {
		return nil, errors.New("Float continuation contains a credential")
	}
	parsed.RawQuery = query.Encode()
	if _, duplicate := seen[parsed.String()]; duplicate {
		return nil, errors.New("Float pagination cycle")
	}
	return parsed, nil
}

func (r *FloatResolver) publishCache(directory string, lookup FloatLookup) error {
	cache := floatCache{SchemaVersion: floatCacheSchema, Source: floatSourceIdentity, RetrievedAt: canonicalCacheTime(lookup.retrieved), Facts: make([]floatCacheFact, len(lookup.facts))}
	for i, fact := range lookup.facts {
		cache.Facts[i] = floatCacheFact{Ticker: fact.Ticker, FreeFloat: fact.FreeFloat, FreeFloatPercent: fact.FreeFloatPercent, EffectiveDate: fact.EffectiveDate}
	}
	body, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return publishPrivateCacheFile(directory, "latest.json", ".float-", body, nil)
}

func (r *FloatResolver) readCache(path string, universe []string) (FloatLookup, error) {
	info, err := validatePrivateRegularFile(path)
	if err != nil || info.Size() > maximumCacheBytes {
		return FloatLookup{}, errors.New("Float cache is unavailable")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return FloatLookup{}, err
	}
	var cache floatCache
	if err := decodeStrictCacheJSON(body, &cache); err != nil || cache.SchemaVersion != floatCacheSchema || cache.Source != floatSourceIdentity {
		return FloatLookup{}, errors.New("Float cache schema or source mismatch")
	}
	retrieved, err := parseCanonicalCacheTime(cache.RetrievedAt)
	if err != nil || retrieved.After(r.now()) {
		return FloatLookup{}, errors.New("Float cache retrieval time is invalid")
	}
	allowed := make(map[string]struct{}, len(universe))
	for _, symbol := range universe {
		allowed[symbol] = struct{}{}
	}
	result := FloatLookup{provenance: FloatCache, retrieved: retrieved, facts: make([]FloatFact, 0, len(cache.Facts))}
	previous := ""
	for _, row := range cache.Facts {
		if row.Ticker <= previous {
			return FloatLookup{}, errors.New("Float cache tickers are not strict sorted unique")
		}
		previous = row.Ticker
		if _, ok := allowed[row.Ticker]; !ok || row.FreeFloat <= 0 || math.IsNaN(row.FreeFloat) || math.IsInf(row.FreeFloat, 0) {
			return FloatLookup{}, errors.New("Float cache fact is invalid")
		}
		if row.FreeFloatPercent != nil && (*row.FreeFloatPercent < 0 || *row.FreeFloatPercent > 100 || math.IsNaN(*row.FreeFloatPercent) || math.IsInf(*row.FreeFloatPercent, 0)) {
			return FloatLookup{}, errors.New("Float cache percent is invalid")
		}
		if row.EffectiveDate != "" {
			if _, err := time.Parse("2006-01-02", row.EffectiveDate); err != nil {
				return FloatLookup{}, errors.New("Float cache effective date is invalid")
			}
		}
		fact := FloatFact{Ticker: row.Ticker, FreeFloat: row.FreeFloat, FreeFloatPercent: row.FreeFloatPercent, EffectiveDate: row.EffectiveDate,
			Provider: floatSourceIdentity, RetrievedAt: retrieved, Provenance: FloatCache}
		result.facts = append(result.facts, cloneFloatFact(fact))
	}
	return result, nil
}

func (r *FloatResolver) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}
