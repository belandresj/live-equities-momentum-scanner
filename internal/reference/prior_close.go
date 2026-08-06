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
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const (
	PriorClosePolicyVersion = "massive-adjusted-prior-close-v1"

	priorCloseIdentitySchema = "prior-close-v1"
	maximumGroupedRows       = 100_000
)

// PriorCloseStatus is the exhaustive status of one eligible symbol's prior close.
type PriorCloseStatus string

const (
	PriorCloseValid   PriorCloseStatus = "valid"
	PriorCloseMissing PriorCloseStatus = "missing"
	PriorCloseInvalid PriorCloseStatus = "invalid"
)

// PriorCloseInvalidReason is present only for an invalid symbol-local fact.
type PriorCloseInvalidReason string

const (
	PriorCloseMalformed   PriorCloseInvalidReason = "malformed"
	PriorCloseDuplicate   PriorCloseInvalidReason = "duplicate"
	PriorCloseWrongDate   PriorCloseInvalidReason = "wrong_date"
	PriorCloseNonpositive PriorCloseInvalidReason = "nonpositive"
	PriorCloseNonfinite   PriorCloseInvalidReason = "nonfinite"
)

// PriorCloseFact contains no trusted numeric value unless Status is valid.
type PriorCloseFact struct {
	symbol string
	status PriorCloseStatus
	close  float64
	reason PriorCloseInvalidReason
}

func (f PriorCloseFact) Symbol() string                  { return f.symbol }
func (f PriorCloseFact) Status() PriorCloseStatus        { return f.status }
func (f PriorCloseFact) Reason() PriorCloseInvalidReason { return f.reason }
func (f PriorCloseFact) Close() (float64, bool) {
	return f.close, f.status == PriorCloseValid
}

// PriorCloseAccounting is a closed partition of the bound universe.
type PriorCloseAccounting struct {
	UniverseTotal                int
	ValidPriorClose              int
	MissingPriorClose            int
	InvalidPriorClose            int
	InvalidOrMissingPriorClose   int
	UnattributableDiagnosticRows int
}

func (a PriorCloseAccounting) valid() bool {
	return a.UniverseTotal >= 0 && a.ValidPriorClose >= 0 && a.MissingPriorClose >= 0 &&
		a.InvalidPriorClose >= 0 && a.UnattributableDiagnosticRows >= 0 &&
		a.UniverseTotal == a.ValidPriorClose+a.MissingPriorClose+a.InvalidPriorClose &&
		a.InvalidOrMissingPriorClose == a.MissingPriorClose+a.InvalidPriorClose
}

// PriorCloses is one immutable exact-date, exact-policy normalized dataset.
type PriorCloses struct {
	priorSessionDate  string
	policyVersion     string
	identity          string
	facts             []PriorCloseFact
	accounting        PriorCloseAccounting
	source            Source
	persistenceReason string
	diagnostics       AcquisitionDiagnostics
}

func (p PriorCloses) PriorSessionDate() string            { return p.priorSessionDate }
func (p PriorCloses) PolicyVersion() string               { return p.policyVersion }
func (p PriorCloses) Identity() string                    { return p.identity }
func (p PriorCloses) Accounting() PriorCloseAccounting    { return p.accounting }
func (p PriorCloses) Source() Source                      { return p.source }
func (p PriorCloses) PersistenceReason() string           { return p.persistenceReason }
func (p PriorCloses) Diagnostics() AcquisitionDiagnostics { return p.diagnostics }
func (p PriorCloses) Facts() []PriorCloseFact             { return slices.Clone(p.facts) }

// PriorCloseResolver performs one grouped request and returns facts only.
type PriorCloseResolver struct {
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
	publish  func(string, normalizedPriorCloses) error
	prune    func(string) error
	cacheOps *cacheFileOps
}

type groupedRow struct {
	Symbol json.RawMessage `json:"T"`
	Close  json.RawMessage `json:"c"`
	Start  json.RawMessage `json:"t"`
}

type normalizedPriorCloses struct {
	priorSessionDate string
	retrievedAt      time.Time
	identity         string
	facts            []PriorCloseFact
	accounting       PriorCloseAccounting
}

// Resolve validates the accepted schedule and current universe before using
// fresh or exact-date cached prior-close evidence.
func (r *PriorCloseResolver) Resolve(ctx context.Context, facts session.Facts, universe Universe) (PriorCloses, error) {
	tracker := newAcquisitionTracker()
	if err := validateBindingInputs(r.Schedule, facts, universe); err != nil {
		return PriorCloses{}, acquisitionFailure(tracker, TerminalReasonInvalidRequest, err)
	}

	directory, cacheBoundaryErr := inspectCacheDirectory(r.DataDir, "prior-close")
	var cached normalizedPriorCloses
	var cacheErr error
	if cacheBoundaryErr == nil {
		cached, cacheErr = r.readPriorCloseCache(directory, facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols)
	} else {
		cacheErr = cacheBoundaryErr
	}

	configurationErr := r.configurationError()
	var normalized normalizedPriorCloses
	var fetchErr error
	if configurationErr == nil {
		operationContextFactory := r.operationContext
		if operationContextFactory == nil {
			operationContextFactory = context.WithTimeout
		}
		operationContext, cancel := operationContextFactory(ctx, operationTimeout)
		normalized, fetchErr = r.fetch(operationContext, facts.PriorSessionDate, universe.symbols, tracker)
		cancel()
	} else {
		fetchErr = configurationErr
	}

	if fetchErr == nil {
		tracker.diagnostics.Source = SourceFresh
		result := priorClosesFromNormalized(normalized, SourceFresh)
		result.diagnostics = tracker.snapshot()
		if cacheBoundaryErr != nil {
			preparer := r.prepare
			if preparer == nil {
				preparer = preparePriorCloseCache
			}
			directory, cacheBoundaryErr = preparer(r.DataDir)
		}
		if cacheBoundaryErr != nil {
			result.persistenceReason = persistenceCachePrepareFailed
			return result, nil
		}
		pruner := r.prune
		if pruner == nil {
			pruner = func(directory string) error { return prunePriorCloseCachesAt(directory, r.now()) }
		}
		if err := pruner(directory); err != nil {
			result.persistenceReason = persistenceCachePruneFailed
			return result, nil
		}
		publisher := r.publish
		if publisher == nil {
			publisher = func(directory string, data normalizedPriorCloses) error {
				return publishPriorCloseCacheWithOps(directory, data, r.cacheOps)
			}
		}
		if err := publisher(directory, normalized); err != nil {
			result.persistenceReason = persistenceCachePublicationFailed
			return result, nil
		}
		if err := pruner(directory); err != nil {
			result.persistenceReason = persistenceCachePruneFailed
		}
		return result, nil
	}

	if cacheErr != nil {
		reason := classifyTerminalReason(fetchErr, TerminalReasonSourceUnavailable)
		return PriorCloses{}, acquisitionFailure(tracker, reason, fmt.Errorf("prior-close retrieval failed and no validated exact-date cache exists: %w", fetchErr))
	}
	result := priorClosesFromNormalized(cached, SourceCurrentCache)
	tracker.diagnostics.Source = SourceCurrentCache
	result.diagnostics = tracker.snapshot()
	return result, nil
}

func (r *PriorCloseResolver) configurationError() error {
	if r.APIKey == "" {
		return fmt.Errorf("%w: prior-close credential is required", errAcquisitionConfiguration)
	}
	if _, err := validatedBaseURL(r.BaseURL); err != nil {
		return fmt.Errorf("%w: %v", errAcquisitionConfiguration, err)
	}
	return nil
}

func validateBindingInputs(schedule *session.Schedule, facts session.Facts, universe Universe) error {
	if schedule == nil {
		return errors.New("schedule is required")
	}
	expected, err := schedule.ForTradingDate(facts.TradingDate)
	if err != nil || expected != facts {
		return errors.New("session facts do not match the accepted schedule")
	}
	if !universe.IsCurrent() || universe.referenceDate != facts.TradingDate || universe.policyVersion != EligibilityPolicyVersion ||
		len(universe.symbols) == 0 || universe.accounting.EligibleRecords != len(universe.symbols) || !universe.accounting.valid() {
		return errors.New("current exact-date universe is required")
	}
	identity, err := universeIdentity(universe.referenceDate, universe.policyVersion, universe.symbols)
	if err != nil || identity != universe.identity {
		return errors.New("universe identity mismatch")
	}
	return nil
}

func priorClosesFromNormalized(data normalizedPriorCloses, source Source) PriorCloses {
	return PriorCloses{
		priorSessionDate: data.priorSessionDate,
		policyVersion:    PriorClosePolicyVersion,
		identity:         data.identity,
		facts:            slices.Clone(data.facts),
		accounting:       data.accounting,
		source:           source,
	}
}

func (r *PriorCloseResolver) fetch(ctx context.Context, date string, symbols []string, tracker *acquisitionTracker) (normalizedPriorCloses, error) {
	base, err := validatedBaseURL(r.BaseURL)
	if err != nil {
		return normalizedPriorCloses{}, fmt.Errorf("%w: %v", errAcquisitionConfiguration, err)
	}
	target := base.ResolveReference(&url.URL{Path: "/v2/aggs/grouped/locale/us/market/stocks/" + date})
	query := target.Query()
	query.Set("adjusted", "true")
	query.Set("include_otc", "false")
	target.RawQuery = query.Encode()
	body, err := r.get(ctx, target, tracker, false)
	if err != nil {
		return normalizedPriorCloses{}, err
	}

	rows, err := parseGroupedEnvelope(body)
	if err != nil {
		return normalizedPriorCloses{}, fmt.Errorf("%w: invalid grouped-daily response envelope: %v", errSourceAmbiguous, err)
	}

	eligible := make(map[string]int, len(symbols))
	facts := make([]PriorCloseFact, len(symbols))
	for index, symbol := range symbols {
		eligible[symbol] = index
		facts[index] = PriorCloseFact{symbol: symbol, status: PriorCloseMissing}
	}
	diagnostics := 0
	seen := make(map[string]struct{}, len(rows))
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return normalizedPriorCloses{}, errors.New("load America/New_York timezone")
	}
	for _, raw := range rows {
		members, err := objectMembers(raw)
		if err != nil {
			diagnostics++
			continue
		}
		symbolRaw, err := requiredMember(members, "T")
		if err != nil {
			diagnostics++
			continue
		}
		symbol, err := decodeMember[string](symbolRaw)
		if err != nil || !validSymbol(symbol) {
			diagnostics++
			continue
		}
		index, isEligible := eligible[symbol]
		if !isEligible {
			diagnostics++
			continue
		}
		if _, duplicate := seen[symbol]; duplicate {
			facts[index] = PriorCloseFact{symbol: symbol, status: PriorCloseInvalid, reason: PriorCloseDuplicate}
			continue
		}
		seen[symbol] = struct{}{}
		closeRaw, closeErr := requiredMember(members, "c")
		startRaw, startErr := requiredMember(members, "t")
		if closeErr != nil || startErr != nil {
			facts[index] = PriorCloseFact{symbol: symbol, status: PriorCloseInvalid, reason: PriorCloseMalformed}
			continue
		}
		row := groupedRow{Symbol: symbolRaw, Close: closeRaw, Start: startRaw}
		status, closeValue, reason := normalizePriorRow(row, date, location)
		facts[index] = PriorCloseFact{symbol: symbol, status: status, close: closeValue, reason: reason}
	}

	accounting := accountPriorCloses(facts, diagnostics)
	tracker.diagnostics.UnattributablePriorRows = diagnostics
	identity, err := priorCloseIdentity(date, PriorClosePolicyVersion, facts)
	if err != nil {
		return normalizedPriorCloses{}, err
	}
	return normalizedPriorCloses{
		priorSessionDate: date,
		retrievedAt:      r.now().UTC(),
		identity:         identity,
		facts:            facts,
		accounting:       accounting,
	}, nil
}

func normalizePriorRow(row groupedRow, date string, location *time.Location) (PriorCloseStatus, float64, PriorCloseInvalidReason) {
	closeValue, reason := parseClose(row.Close)
	if reason != "" {
		return PriorCloseInvalid, 0, reason
	}
	var timestampNumber json.Number
	decoder := json.NewDecoder(bytes.NewReader(row.Start))
	decoder.UseNumber()
	if err := decoder.Decode(&timestampNumber); err != nil {
		return PriorCloseInvalid, 0, PriorCloseMalformed
	}
	milliseconds, err := strconv.ParseInt(timestampNumber.String(), 10, 64)
	if err != nil {
		return PriorCloseInvalid, 0, PriorCloseMalformed
	}
	if time.UnixMilli(milliseconds).In(location).Format("2006-01-02") != date {
		return PriorCloseInvalid, 0, PriorCloseWrongDate
	}
	return PriorCloseValid, closeValue, ""
}

func parseClose(raw json.RawMessage) (float64, PriorCloseInvalidReason) {
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		var text string
		if json.Unmarshal(raw, &text) == nil && (text == "NaN" || text == "Infinity" || text == "-Infinity") {
			return 0, PriorCloseNonfinite
		}
		return 0, PriorCloseMalformed
	}
	value, err := number.Float64()
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, PriorCloseNonfinite
	}
	if value <= 0 {
		return 0, PriorCloseNonpositive
	}
	return value, ""
}

func accountPriorCloses(facts []PriorCloseFact, diagnostics int) PriorCloseAccounting {
	accounting := PriorCloseAccounting{UniverseTotal: len(facts), UnattributableDiagnosticRows: diagnostics}
	for _, fact := range facts {
		switch fact.status {
		case PriorCloseValid:
			accounting.ValidPriorClose++
		case PriorCloseMissing:
			accounting.MissingPriorClose++
		case PriorCloseInvalid:
			accounting.InvalidPriorClose++
		}
	}
	accounting.InvalidOrMissingPriorClose = accounting.MissingPriorClose + accounting.InvalidPriorClose
	return accounting
}

func priorCloseIdentity(date, policy string, facts []PriorCloseFact) (string, error) {
	type identityFact struct {
		Symbol    string `json:"symbol"`
		Status    string `json:"status"`
		CloseBits string `json:"close_bits_or_empty"`
	}
	pairs := make([]identityFact, len(facts))
	for index, fact := range facts {
		bits := ""
		if fact.status == PriorCloseValid {
			bits = fmt.Sprintf("%016x", math.Float64bits(fact.close))
		}
		pairs[index] = identityFact{fact.symbol, string(fact.status), bits}
	}
	payload := struct {
		Schema            string         `json:"schema"`
		PriorSessionDate  string         `json:"prior_session_date"`
		PriorClosePolicy  string         `json:"prior_close_policy"`
		Adjusted          bool           `json:"adjusted"`
		IncludeOTC        bool           `json:"include_otc"`
		Locale            string         `json:"locale"`
		Market            string         `json:"market"`
		SortedSymbolFacts []identityFact `json:"sorted_per_universe_symbol"`
	}{priorCloseIdentitySchema, date, policy, true, false, "us", "stocks", pairs}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode prior-close identity: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return priorCloseIdentitySchema + ":" + hex.EncodeToString(sum[:]), nil
}

func (r *PriorCloseResolver) get(ctx context.Context, target *url.URL, tracker *acquisitionTracker, page bool) ([]byte, error) {
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
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return nil, errors.New("construct prior-close request")
		}
		request.Header.Set("Authorization", "Bearer "+r.APIKey)
		attemptContextFactory := r.attemptContext
		if attemptContextFactory == nil {
			attemptContextFactory = context.WithTimeout
		}
		attemptContext, cancel := attemptContextFactory(request.Context(), requestAttemptTimeout)
		request = request.WithContext(attemptContext)
		response, requestErr := safeClient.Do(request)
		if requestErr == nil {
			body, readErr := readBounded(response.Body, maximumReferenceBody)
			closeErr := response.Body.Close()
			cancel()
			if readErr == nil && closeErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
				return body, nil
			}
			if readErr != nil && !errors.Is(readErr, errResponseRead) {
				return nil, readErr
			}
			if readErr == nil && (response.StatusCode < 200 || response.StatusCode >= 300) {
				retryable := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
				if !retryable {
					return nil, fmt.Errorf("prior-close request failed with HTTP %d", response.StatusCode)
				}
			}
		} else {
			cancel()
			lastAttemptErr = requestErr
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if attempt == maximumAttempts-1 {
			if lastAttemptErr != nil {
				return nil, lastAttemptErr
			}
			return nil, errors.New("prior-close request failed after three attempts")
		}
		maximumDelay := 250 * time.Millisecond * time.Duration(1<<attempt)
		delay := time.Duration(0)
		if maximumDelay > 0 {
			delay = time.Duration(r.now().UnixNano() % (int64(maximumDelay) + 1))
		}
		if err := r.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
	panic("unreachable")
}

func parseGroupedEnvelope(body []byte) ([]json.RawMessage, error) {
	members, err := objectMembers(body)
	if err != nil {
		return nil, err
	}
	statusRaw, err := requiredMember(members, "status")
	if err != nil {
		return nil, err
	}
	status, err := decodeMember[string](statusRaw)
	if err != nil || status != "OK" {
		return nil, errors.New("grouped status is not OK")
	}
	adjustedRaw, err := requiredMember(members, "adjusted")
	if err != nil {
		return nil, err
	}
	adjusted, err := decodeMember[bool](adjustedRaw)
	if err != nil || !adjusted {
		return nil, errors.New("grouped adjustment is not explicitly true")
	}
	countRaw, err := requiredMember(members, "resultsCount")
	if err != nil {
		return nil, err
	}
	count, err := decodeMember[int](countRaw)
	if err != nil || count < 0 {
		return nil, errors.New("grouped result count is invalid")
	}
	resultsRaw, err := requiredMember(members, "results")
	if err != nil || bytes.Equal(resultsRaw, []byte("null")) {
		return nil, errors.New("grouped results are missing")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(resultsRaw, &rows); err != nil || count != len(rows) || len(rows) > maximumGroupedRows {
		return nil, errors.New("grouped results do not match their count or bound")
	}
	return rows, nil
}

func (r *PriorCloseResolver) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (r *PriorCloseResolver) sleep(ctx context.Context, delay time.Duration) error {
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

func jsonEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("trailing JSON value")
	}
	return err
}
