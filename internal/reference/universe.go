// Package reference resolves immutable, exact-date reference facts without
// owning scanner lifecycle, readiness, or canonical market state.
package reference

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const (
	EligibilityPolicyVersion = "massive-us-common-stocks-eligibility-v2"

	universeIdentitySchema  = "universe-v1"
	maximumReferencePages   = 100
	maximumReferenceRecords = 100_000
	maximumReferenceBody    = 4 << 20
	maximumPageRecords      = 1_000
	maximumSymbolBytes      = 64
	maximumAttempts         = 3
	requestAttemptTimeout   = 15 * time.Second
	operationTimeout        = 2 * time.Minute
)

var errResponseRead = errors.New("read reference response")

// Source states whether the returned facts are usable as the requested-date
// universe. ObservablePriorCache is evidence only and is never current.
type Source string

const (
	SourceNone                 Source = "none"
	SourceFresh                Source = "fresh"
	SourceCurrentCache         Source = "current_same_date_cache"
	SourceObservablePriorCache Source = "observable_prior_date_cache"
)

// Accounting is the closed, mutually exclusive classification of every
// decoded reference record. Its fields reconcile to RawReferenceRecords.
type Accounting struct {
	RawReferenceRecords   int
	EligibleRecords       int
	InactiveRecords       int
	WrongMarketRecords    int
	WrongLocaleRecords    int
	IneligibleTypeRecords int
}

func (a Accounting) valid() bool {
	if a.RawReferenceRecords < 0 || a.EligibleRecords < 0 || a.InactiveRecords < 0 ||
		a.WrongMarketRecords < 0 || a.WrongLocaleRecords < 0 || a.IneligibleTypeRecords < 0 {
		return false
	}
	return a.RawReferenceRecords == a.EligibleRecords+a.InactiveRecords+
		a.WrongMarketRecords+a.WrongLocaleRecords+a.IneligibleTypeRecords
}

// Universe is an immutable normalized reference result. Symbols returns a
// defensive copy; no writable map or slice is exposed.
type Universe struct {
	referenceDate     string
	policyVersion     string
	identity          string
	symbols           []string
	accounting        Accounting
	source            Source
	cacheAgeDays      int
	persistenceReason string
	diagnostics       AcquisitionDiagnostics
}

func (u Universe) ReferenceDate() string               { return u.referenceDate }
func (u Universe) PolicyVersion() string               { return u.policyVersion }
func (u Universe) Identity() string                    { return u.identity }
func (u Universe) Accounting() Accounting              { return u.accounting }
func (u Universe) Source() Source                      { return u.source }
func (u Universe) CacheAgeDays() int                   { return u.cacheAgeDays }
func (u Universe) PersistenceReason() string           { return u.persistenceReason }
func (u Universe) Diagnostics() AcquisitionDiagnostics { return u.diagnostics }
func (u Universe) IsCurrent() bool {
	return u.source == SourceFresh || u.source == SourceCurrentCache
}
func (u Universe) Symbols() []string { return slices.Clone(u.symbols) }

// Resolver performs one bounded exact-date universe operation. Schedule facts
// are accepted only when they exactly match Schedule.ForTradingDate.
type Resolver struct {
	BaseURL    string
	APIKey     string
	DataDir    string
	HTTPClient *http.Client
	Schedule   *session.Schedule
	Now        func() time.Time
	Sleep      func(context.Context, time.Duration) error

	operationContext func(context.Context, time.Duration) (context.Context, context.CancelFunc)
	attemptContext   func(context.Context, time.Duration) (context.Context, context.CancelFunc)

	prepare  func(string) (string, error)
	publish  func(string, normalizedUniverse) error
	prune    func(string) error
	cacheOps *cacheFileOps
}

type tickerRecord struct {
	Ticker string `json:"ticker"`
	Active bool   `json:"active"`
	Market string `json:"market"`
	Locale string `json:"locale"`
	Type   string `json:"type"`
}

type normalizedUniverse struct {
	referenceDate string
	retrievedAt   time.Time
	identity      string
	symbols       []string
	accounting    Accounting
}

// Resolve prefers a fresh complete result, falls back to an exact same-date
// cache as current, and otherwise returns the newest validated cache no more
// than seven calendar days old as observable-only evidence.
func (r *Resolver) Resolve(ctx context.Context, facts session.Facts) (Universe, error) {
	tracker := newAcquisitionTracker()
	if err := r.validateFacts(facts); err != nil {
		return Universe{}, acquisitionFailure(tracker, TerminalReasonInvalidRequest, err)
	}

	cacheDirectory, cacheBoundaryErr := inspectCacheDirectory(r.DataDir, "universe")
	var fallback Universe
	var fallbackErr error
	if cacheBoundaryErr == nil {
		fallback, fallbackErr = r.loadFallback(cacheDirectory, facts.TradingDate)
	} else {
		fallbackErr = cacheBoundaryErr
	}

	configurationErr := r.configurationError()
	var normalized normalizedUniverse
	var fetchErr error
	if configurationErr == nil {
		operationContextFactory := r.operationContext
		if operationContextFactory == nil {
			operationContextFactory = context.WithTimeout
		}
		operationContext, cancel := operationContextFactory(ctx, operationTimeout)
		normalized, fetchErr = r.fetch(operationContext, facts.TradingDate, tracker)
		cancel()
	} else {
		fetchErr = configurationErr
	}

	if fetchErr == nil {
		tracker.diagnostics.Source = SourceFresh
		result := universeFromNormalized(normalized, SourceFresh, 0)
		result.diagnostics = tracker.snapshot()
		if cacheBoundaryErr != nil {
			preparer := r.prepare
			if preparer == nil {
				preparer = prepareUniverseCache
			}
			cacheDirectory, cacheBoundaryErr = preparer(r.DataDir)
		}
		if cacheBoundaryErr != nil {
			result.persistenceReason = persistenceCachePrepareFailed
			return result, nil
		}
		pruner := r.prune
		if pruner == nil {
			pruner = func(directory string) error { return pruneUniverseCachesAt(directory, r.now()) }
		}
		if err := pruner(cacheDirectory); err != nil {
			result.persistenceReason = persistenceCachePruneFailed
			return result, nil
		}
		publisher := r.publish
		if publisher == nil {
			publisher = func(directory string, data normalizedUniverse) error {
				return publishUniverseCacheWithOps(directory, data, r.cacheOps)
			}
		}
		if err := publisher(cacheDirectory, normalized); err != nil {
			result.persistenceReason = persistenceCachePublicationFailed
			return result, nil
		}
		if err := pruner(cacheDirectory); err != nil {
			result.persistenceReason = persistenceCachePruneFailed
		}
		return result, nil
	}

	if fallbackErr != nil {
		reason := classifyTerminalReason(fetchErr, TerminalReasonSourceUnavailable)
		return Universe{}, acquisitionFailure(tracker, reason, fmt.Errorf("current universe retrieval failed and no validated cache exists: %w", fetchErr))
	}
	tracker.diagnostics.Source = fallback.source
	fallback.diagnostics = tracker.snapshot()
	return fallback, nil
}

func (r *Resolver) configurationError() error {
	if r.APIKey == "" {
		return fmt.Errorf("%w: universe credential is required", errAcquisitionConfiguration)
	}
	if _, err := validatedBaseURL(r.BaseURL); err != nil {
		return fmt.Errorf("%w: %v", errAcquisitionConfiguration, err)
	}
	return nil
}

func (r *Resolver) validateFacts(facts session.Facts) error {
	if r.Schedule == nil {
		return errors.New("schedule is required")
	}
	expected, err := r.Schedule.ForTradingDate(facts.TradingDate)
	if err != nil {
		return fmt.Errorf("validate requested trading date: %w", err)
	}
	if expected != facts {
		return errors.New("session facts do not match the accepted schedule")
	}
	return nil
}

func universeFromNormalized(data normalizedUniverse, source Source, age int) Universe {
	return Universe{
		referenceDate: data.referenceDate,
		policyVersion: EligibilityPolicyVersion,
		identity:      data.identity,
		symbols:       slices.Clone(data.symbols),
		accounting:    data.accounting,
		source:        source,
		cacheAgeDays:  age,
	}
}

func (r *Resolver) fetch(ctx context.Context, tradingDate string, tracker *acquisitionTracker) (normalizedUniverse, error) {
	if r.APIKey == "" {
		return normalizedUniverse{}, fmt.Errorf("%w: universe credential is required", errAcquisitionConfiguration)
	}
	base, err := validatedBaseURL(r.BaseURL)
	if err != nil {
		return normalizedUniverse{}, fmt.Errorf("%w: %v", errAcquisitionConfiguration, err)
	}
	first := base.ResolveReference(&url.URL{Path: "/v3/reference/tickers"})
	query := first.Query()
	query.Set("date", tradingDate)
	query.Set("active", "true")
	query.Set("market", "stocks")
	query.Set("limit", "1000")
	query.Set("sort", "ticker")
	query.Set("order", "asc")
	first.RawQuery = query.Encode()

	next := first.String()
	visited := make(map[string]struct{})
	seenSymbols := make(map[string]struct{})
	symbols := make([]string, 0, 10_000)
	accounting := Accounting{}
	for page := 0; next != ""; page++ {
		if page >= maximumReferencePages {
			return normalizedUniverse{}, errors.New("reference pagination exceeded 100 pages")
		}
		target, err := validatePageURL(next, base, tradingDate)
		if err != nil {
			return normalizedUniverse{}, err
		}
		key := target.String()
		if _, exists := visited[key]; exists {
			return normalizedUniverse{}, errors.New("reference pagination cycle")
		}
		visited[key] = struct{}{}

		body, err := r.get(ctx, target, tracker, true)
		if err != nil {
			return normalizedUniverse{}, err
		}
		records, nextURL, err := parseUniversePage(body)
		if err != nil {
			return normalizedUniverse{}, fmt.Errorf("%w: decode ticker page: %v", errSourceAmbiguous, err)
		}
		if len(records) > maximumPageRecords {
			return normalizedUniverse{}, errors.New("ticker page exceeds 1000 records")
		}
		for _, record := range records {
			accounting.RawReferenceRecords++
			if accounting.RawReferenceRecords > maximumReferenceRecords {
				return normalizedUniverse{}, errors.New("reference population exceeded 100000 records")
			}
			if !validSymbol(record.Ticker) {
				return normalizedUniverse{}, errors.New("ticker identity is empty, invalid UTF-8, or exceeds 64 bytes")
			}
			if _, exists := seenSymbols[record.Ticker]; exists {
				return normalizedUniverse{}, errors.New("duplicate ticker identity")
			}
			seenSymbols[record.Ticker] = struct{}{}
			switch {
			case !record.Active:
				accounting.InactiveRecords++
			case record.Market != "stocks":
				accounting.WrongMarketRecords++
			case record.Locale != "us":
				accounting.WrongLocaleRecords++
			case !eligibleType(record.Type):
				accounting.IneligibleTypeRecords++
			default:
				accounting.EligibleRecords++
				symbols = append(symbols, record.Ticker)
			}
		}
		next = nextURL
	}
	if len(symbols) == 0 {
		return normalizedUniverse{}, errors.New("eligible universe is empty")
	}
	if !accounting.valid() {
		return normalizedUniverse{}, errors.New("reference record accounting does not reconcile")
	}
	slices.Sort(symbols)
	identity, err := universeIdentity(tradingDate, EligibilityPolicyVersion, symbols)
	if err != nil {
		return normalizedUniverse{}, err
	}
	return normalizedUniverse{
		referenceDate: tradingDate,
		retrievedAt:   r.now().UTC(),
		identity:      identity,
		symbols:       symbols,
		accounting:    accounting,
	}, nil
}

func validatedBaseURL(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil || !base.IsAbs() || base.Scheme != "https" || base.Host == "" || base.User != nil || base.Fragment != "" {
		return nil, errors.New("reference base URL must be credential-free HTTPS")
	}
	return base, nil
}

func validatePageURL(raw string, base *url.URL, tradingDate string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil || !target.IsAbs() {
		return nil, errors.New("reference pagination URL is invalid")
	}
	if target.Scheme != "https" {
		return nil, errors.New("reference pagination URL must remain HTTPS")
	}
	if !strings.EqualFold(target.Host, base.Host) || target.User != nil || target.Fragment != "" {
		return nil, errors.New("reference pagination URL changed origin or contains credentials")
	}
	for key := range target.Query() {
		if strings.EqualFold(key, "apiKey") {
			return nil, errors.New("reference pagination URL contains credentials")
		}
	}
	if dates, exists := target.Query()["date"]; exists && (len(dates) != 1 || dates[0] != tradingDate) {
		return nil, errors.New("reference pagination URL changed request date")
	}
	return target, nil
}

func (r *Resolver) get(ctx context.Context, target *url.URL, tracker *acquisitionTracker, page bool) ([]byte, error) {
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tracker.beginRequest(page)
	var lastAttemptErr error
	for attempt := 0; attempt < maximumAttempts; attempt++ {
		lastAttemptErr = nil
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tracker.beginAttempt()
		requestURL := *target
		query := requestURL.Query()
		for key := range query {
			if strings.EqualFold(key, "apiKey") {
				return nil, errors.New("reference request URL unexpectedly contains credentials")
			}
		}
		query.Set("apiKey", r.APIKey)
		requestURL.RawQuery = query.Encode()

		attemptContextFactory := r.attemptContext
		if attemptContextFactory == nil {
			attemptContextFactory = context.WithTimeout
		}
		attemptContext, cancel := attemptContextFactory(ctx, requestAttemptTimeout)
		request, err := http.NewRequestWithContext(attemptContext, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			cancel()
			return nil, errors.New("construct reference request")
		}
		response, requestErr := safeClient.Do(request)
		if requestErr == nil {
			body, readErr := readBounded(response.Body, maximumReferenceBody)
			closeErr := response.Body.Close()
			cancel()
			switch {
			case readErr == nil && closeErr == nil && response.StatusCode >= 200 && response.StatusCode < 300:
				return body, nil
			case readErr != nil && !errors.Is(readErr, errResponseRead):
				return nil, readErr
			case response.StatusCode < 200 || response.StatusCode >= 300:
				retryable := response.StatusCode == http.StatusRequestTimeout ||
					response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
				if !retryable {
					return nil, fmt.Errorf("reference request failed with HTTP %d", response.StatusCode)
				}
			}
		} else {
			cancel()
			lastAttemptErr = requestErr
		}
		if attempt == maximumAttempts-1 {
			if lastAttemptErr != nil {
				return nil, lastAttemptErr
			}
			return nil, errors.New("reference request failed after three attempts")
		}
		maximumDelay := 250 * time.Millisecond * time.Duration(1<<attempt)
		delay := time.Duration(rand.Int64N(int64(maximumDelay) + 1))
		if err := r.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
	panic("unreachable")
}

func parseUniversePage(body []byte) ([]tickerRecord, string, error) {
	members, err := objectMembers(body)
	if err != nil {
		return nil, "", err
	}
	if raw, present, err := optionalMember(members, "status"); err != nil {
		return nil, "", err
	} else if present {
		status, err := decodeMember[string](raw)
		if err != nil || status != "OK" {
			return nil, "", errors.New("ticker page status is not OK")
		}
	}
	resultsRaw, err := requiredMember(members, "results")
	if err != nil || bytes.Equal(resultsRaw, []byte("null")) {
		return nil, "", errors.New("ticker page has incomplete results")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(resultsRaw, &rows); err != nil || len(rows) == 0 {
		return nil, "", errors.New("ticker page results must be a nonempty array")
	}
	if raw, present, err := optionalMember(members, "count"); err != nil {
		return nil, "", err
	} else if present {
		count, err := decodeMember[int](raw)
		if err != nil || count != len(rows) {
			return nil, "", errors.New("ticker page count does not match results")
		}
	}
	nextURL := ""
	if raw, present, err := optionalMember(members, "next_url"); err != nil {
		return nil, "", err
	} else if present {
		nextURL, err = decodeMember[string](raw)
		if err != nil {
			return nil, "", errors.New("ticker page next_url is malformed")
		}
	}
	records := make([]tickerRecord, len(rows))
	for index, raw := range rows {
		record, err := parseTickerRecord(raw)
		if err != nil {
			return nil, "", fmt.Errorf("ticker record %d is ambiguous: %w", index, err)
		}
		records[index] = record
	}
	return records, nextURL, nil
}

func parseTickerRecord(raw []byte) (tickerRecord, error) {
	members, err := objectMembers(raw)
	if err != nil {
		return tickerRecord{}, err
	}
	var record tickerRecord
	for _, field := range []struct {
		name   string
		decode func(json.RawMessage) error
	}{
		{"ticker", func(raw json.RawMessage) error { record.Ticker, err = decodeMember[string](raw); return err }},
		{"active", func(raw json.RawMessage) error { record.Active, err = decodeMember[bool](raw); return err }},
		{"market", func(raw json.RawMessage) error { record.Market, err = decodeMember[string](raw); return err }},
		{"locale", func(raw json.RawMessage) error { record.Locale, err = decodeMember[string](raw); return err }},
		{"type", func(raw json.RawMessage) error { record.Type, err = decodeMember[string](raw); return err }},
	} {
		value, memberErr := requiredMember(members, field.name)
		if memberErr != nil {
			return tickerRecord{}, memberErr
		}
		if err := field.decode(value); err != nil {
			return tickerRecord{}, fmt.Errorf("member %q has the wrong type", field.name)
		}
	}
	return record, nil
}

func (r *Resolver) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (r *Resolver) sleep(ctx context.Context, delay time.Duration) error {
	if r.Sleep != nil {
		return r.Sleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readBounded(reader io.Reader, maximum int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, errResponseRead
	}
	if int64(len(body)) > maximum {
		return nil, errors.New("reference response exceeds 4 MiB")
	}
	return body, nil
}

func decodeJSON(body []byte, target any, strict bool) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func eligibleType(code string) bool { return code == "CS" || code == "ADRC" }

func validSymbol(symbol string) bool {
	return symbol != "" && len(symbol) <= maximumSymbolBytes && utf8.ValidString(symbol)
}

func universeIdentity(tradingDate, policy string, symbols []string) (string, error) {
	payload := struct {
		Schema            string   `json:"schema"`
		TradingDate       string   `json:"trading_date"`
		EligibilityPolicy string   `json:"eligibility_policy"`
		SortedSymbols     []string `json:"sorted_symbols"`
	}{universeIdentitySchema, tradingDate, policy, symbols}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode universe identity: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return universeIdentitySchema + ":" + hex.EncodeToString(sum[:]), nil
}
