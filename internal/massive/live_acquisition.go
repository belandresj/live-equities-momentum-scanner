package massive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	liveRequestDeadline = 15 * time.Second
	livePageLimit       = 2
	liveAttemptLimit    = 3
	livePageByteLimit   = int64(16 << 20)
	liveResultLimit     = 50_000
	liveWorkerLimit     = 8
)

type DownloadReason string

const (
	DownloadReasonNone                   DownloadReason = ""
	DownloadReasonRequestConstruction    DownloadReason = "request_construction"
	DownloadReasonTransportDeadline      DownloadReason = "transport_deadline"
	DownloadReasonHTTPRetryExhausted     DownloadReason = "http_retry_exhausted"
	DownloadReasonRedirectContinuation   DownloadReason = "redirect_continuation"
	DownloadReasonResponseSizeSyntax     DownloadReason = "response_size_syntax"
	DownloadReasonEnvelopeIdentityStatus DownloadReason = "envelope_identity_status"
	DownloadReasonTimestamp              DownloadReason = "timestamp"
	DownloadReasonNumericCount           DownloadReason = "numeric_count"
	DownloadReasonStructuralAggregate    DownloadReason = "structural_aggregate"
	DownloadReasonSymbolIntervalOrder    DownloadReason = "symbol_interval_order_duplicate"
	DownloadReasonPlanBudget             DownloadReason = "plan_budget"
	DownloadReasonCanceled               DownloadReason = "canceled"
)

type SymbolOutcomeState string

const (
	SymbolComplete SymbolOutcomeState = "complete"
	SymbolFailed   SymbolOutcomeState = "failed"
	SymbolCanceled SymbolOutcomeState = "canceled"
)

type SymbolOutcome struct {
	Symbol   string
	State    SymbolOutcomeState
	Reason   DownloadReason
	Records  int64
	Pages    int64
	Attempts int64
	Bytes    int64
}

type CredentialSource func() (string, error)

// aggregateRESTClient is the one Massive second-aggregate request, envelope,
// pagination, and normalization path used by production hydration. It owns
// only bounded request-local state.
type aggregateRESTClient struct {
	base       *url.URL
	credential CredentialSource
	client     *http.Client
	sleep      func(context.Context, time.Duration) error
	now        func() time.Time
}

func newAggregateRESTClient(baseURL string, credential CredentialSource, client *http.Client) (*aggregateRESTClient, error) {
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || (base.Path != "" && base.Path != "/") ||
		base.User != nil || base.RawQuery != "" || base.Fragment != "" || credential == nil || client == nil {
		return nil, errors.New("invalid live Massive hydration configuration")
	}
	base.Path = ""
	return &aggregateRESTClient{
		base: base, credential: credential, client: client, sleep: sleepWithContext, now: time.Now,
	}, nil
}

type aggregateRESTRequest struct {
	start, end               time.Time
	maximumNormalizedRecords int64
}

func (d *aggregateRESTClient) acquire(ctx context.Context, token, symbol string, request aggregateRESTRequest, wireBytes *responseBudget, records *atomic.Int64, resident *residentRecordBudget) ([]RESTSecondAggregate, SymbolOutcome) {
	outcome := SymbolOutcome{Symbol: symbol, State: SymbolFailed}
	endpointPath := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/second/%d/%d", url.PathEscape(symbol), request.start.UnixMilli(), request.end.Add(-time.Millisecond).UnixMilli())
	current := *d.base
	current.Path = endpointPath
	setFixedAggregateQuery(&current)
	seenPages := make(map[string]struct{}, livePageLimit)
	seenIdentities := make(map[int64]struct{})
	values := make([]RESTSecondAggregate, 0)
	keepResident := false
	defer func() {
		if resident != nil && !keepResident {
			resident.release(int64(len(values)))
		}
	}()
	for pageNumber := 0; pageNumber < livePageLimit; pageNumber++ {
		pageKey := current.String()
		if _, duplicate := seenPages[pageKey]; duplicate {
			outcome.Reason = DownloadReasonRedirectContinuation
			return nil, outcome
		}
		seenPages[pageKey] = struct{}{}
		page, pageBytes, attempts, reason := d.fetchPage(ctx, token, &current, symbol, wireBytes)
		outcome.Attempts += int64(attempts)
		outcome.Bytes += pageBytes
		if reason != DownloadReasonNone {
			if reason == DownloadReasonCanceled {
				outcome.State = SymbolCanceled
			}
			outcome.Reason = reason
			return nil, outcome
		}
		outcome.Pages++
		for _, raw := range page.results {
			value, rejection := NormalizeRESTSecondAggregate(symbol, raw)
			if rejection != "" {
				outcome.Reason = downloadReasonForNormalization(rejection)
				return nil, outcome
			}
			if value.WindowStart.Before(request.start) || !value.WindowStart.Before(request.end) || value.WindowEnd.After(request.end) {
				outcome.Reason = DownloadReasonSymbolIntervalOrder
				return nil, outcome
			}
			key := value.WindowStart.Unix()
			if _, duplicate := seenIdentities[key]; duplicate || len(values) > 0 && !values[len(values)-1].WindowStart.Before(value.WindowStart) {
				outcome.Reason = DownloadReasonSymbolIntervalOrder
				return nil, outcome
			}
			seenIdentities[key] = struct{}{}
			if records != nil && records.Add(1) > request.maximumNormalizedRecords {
				outcome.Reason = DownloadReasonPlanBudget
				return nil, outcome
			}
			if resident != nil && !resident.reserve() {
				outcome.Reason = DownloadReasonPlanBudget
				return nil, outcome
			}
			values = append(values, value)
		}
		if page.nextURL == "" {
			outcome.State = SymbolComplete
			outcome.Reason = DownloadReasonNone
			outcome.Records = int64(len(values))
			keepResident = true
			return values, outcome
		}
		if pageNumber+1 >= livePageLimit {
			outcome.Reason = DownloadReasonRedirectContinuation
			return nil, outcome
		}
		next, reason := d.validateContinuation(page.nextURL, endpointPath)
		if reason != DownloadReasonNone {
			outcome.Reason = reason
			return nil, outcome
		}
		current = *next
	}
	outcome.Reason = DownloadReasonRedirectContinuation
	return nil, outcome
}

type restPage struct {
	results []json.RawMessage
	nextURL string
}

func (d *aggregateRESTClient) fetchPage(ctx context.Context, token string, pageURL *url.URL, symbol string, budget *responseBudget) (restPage, int64, int, DownloadReason) {
	var bytesRead int64
	var retryDelay time.Duration
	for attempt := 1; attempt <= liveAttemptLimit; attempt++ {
		if attempt > 1 {
			if retryDelay == 0 {
				retryDelay = 25 * time.Millisecond * time.Duration(1<<(attempt-2))
			}
			if err := d.sleep(ctx, retryDelay); err != nil {
				return restPage{}, bytesRead, attempt - 1, DownloadReasonCanceled
			}
			retryDelay = 0
		}
		attemptContext, cancel := context.WithTimeout(ctx, liveRequestDeadline)
		request, err := http.NewRequestWithContext(attemptContext, http.MethodGet, pageURL.String(), nil)
		if err != nil {
			cancel()
			return restPage{}, bytesRead, attempt, DownloadReasonRequestConstruction
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Accept", "application/json")
		client := *d.client
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		response, err := client.Do(request)
		if err != nil {
			cancel()
			if ctx.Err() != nil {
				return restPage{}, bytesRead, attempt, DownloadReasonCanceled
			}
			if attempt == liveAttemptLimit {
				return restPage{}, bytesRead, attempt, DownloadReasonTransportDeadline
			}
			continue
		}
		if response.Request == nil || response.Request.URL.Scheme != d.base.Scheme || response.Request.URL.Host != d.base.Host {
			response.Body.Close()
			cancel()
			return restPage{}, bytesRead, attempt, DownloadReasonRedirectContinuation
		}
		body, readErr := budget.read(response.Body)
		response.Body.Close()
		cancel()
		amount := int64(len(body))
		bytesRead += amount
		if errors.Is(readErr, errResponseBudget) || amount > livePageByteLimit {
			return restPage{}, bytesRead, attempt, DownloadReasonResponseSizeSyntax
		}
		if readErr != nil {
			if attempt == liveAttemptLimit {
				return restPage{}, bytesRead, attempt, DownloadReasonTransportDeadline
			}
			continue
		}
		if response.StatusCode != http.StatusOK {
			if response.StatusCode >= 300 && response.StatusCode < 400 {
				return restPage{}, bytesRead, attempt, DownloadReasonRedirectContinuation
			}
			retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
			if !retryable {
				return restPage{}, bytesRead, attempt, DownloadReasonEnvelopeIdentityStatus
			}
			if attempt == liveAttemptLimit {
				return restPage{}, bytesRead, attempt, DownloadReasonHTTPRetryExhausted
			}
			if response.StatusCode == http.StatusTooManyRequests {
				retryDelay = retryAfter(response.Header.Get("Retry-After"), d.now)
			}
			continue
		}
		page, reason := decodeRESTPage(body, symbol)
		if reason != DownloadReasonNone {
			return restPage{}, bytesRead, attempt, reason
		}
		return page, bytesRead, attempt, DownloadReasonNone
	}
	return restPage{}, bytesRead, liveAttemptLimit, DownloadReasonHTTPRetryExhausted
}

func (d *aggregateRESTClient) validateContinuation(raw, endpointPath string) (*url.URL, DownloadReason) {
	next, err := url.Parse(raw)
	if err != nil || next.Scheme != d.base.Scheme || next.Host != d.base.Host || next.Path != endpointPath || next.User != nil || next.Fragment != "" {
		return nil, DownloadReasonRedirectContinuation
	}
	setFixedAggregateQuery(next)
	return next, DownloadReasonNone
}

func setFixedAggregateQuery(value *url.URL) {
	query := value.Query()
	query.Del("apiKey")
	query.Del("apikey")
	query.Set("adjusted", "false")
	query.Set("sort", "asc")
	query.Set("limit", strconv.Itoa(liveResultLimit))
	value.RawQuery = query.Encode()
}

var errResponseBudget = errors.New("live hydration response byte budget exceeded")

type responseBudget struct {
	mu                 sync.Mutex
	condition          *sync.Cond
	used, reserved     int64
	maximum            int64
	exhausted, proving bool
}

func (b *responseBudget) read(source io.Reader) ([]byte, error) {
	if source == nil || b.maximum <= 0 {
		return nil, errResponseBudget
	}
	return io.ReadAll(&responseBudgetReader{source: source, budget: b, pageRemaining: livePageByteLimit})
}

type responseBudgetReader struct {
	source        io.Reader
	budget        *responseBudget
	pageRemaining int64
}

func (r *responseBudgetReader) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	want := min(int64(len(destination)), r.pageRemaining)
	if want > 0 {
		reserved := r.budget.reserve(want)
		if reserved > 0 {
			read, err := r.source.Read(destination[:reserved])
			r.budget.commit(reserved, int64(read))
			r.pageRemaining -= int64(read)
			return read, err
		}
	}
	return r.proveBoundary()
}

func (r *responseBudgetReader) proveBoundary() (int, error) {
	if !r.budget.beginProof() {
		return 0, errResponseBudget
	}
	defer r.budget.endProof()
	var proof [1]byte
	read, err := r.source.Read(proof[:])
	if read == 0 && errors.Is(err, io.EOF) {
		return 0, io.EOF
	}
	if read == 0 && err != nil {
		return 0, err
	}
	r.budget.fail()
	return 0, errResponseBudget
}

func (b *responseBudget) reserve(want int64) int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.initializeConditionLocked()
	for {
		if b.exhausted {
			return 0
		}
		remaining := b.maximum - b.used - b.reserved
		if remaining <= 0 && b.reserved > 0 {
			b.condition.Wait()
			continue
		}
		if remaining <= 0 {
			return 0
		}
		reserved := min(want, remaining)
		b.reserved += reserved
		return reserved
	}
}

func (b *responseBudget) commit(reserved, read int64) {
	b.mu.Lock()
	b.initializeConditionLocked()
	b.reserved -= reserved
	b.used += read
	b.condition.Broadcast()
	b.mu.Unlock()
}

func (b *responseBudget) beginProof() bool {
	b.mu.Lock()
	b.initializeConditionLocked()
	for b.proving && !b.exhausted {
		b.condition.Wait()
	}
	if b.exhausted {
		b.mu.Unlock()
		return false
	}
	b.proving = true
	b.mu.Unlock()
	return true
}

func (b *responseBudget) endProof() {
	b.mu.Lock()
	b.initializeConditionLocked()
	b.proving = false
	b.condition.Broadcast()
	b.mu.Unlock()
}

func (b *responseBudget) fail() {
	b.mu.Lock()
	b.initializeConditionLocked()
	b.exhausted = true
	b.condition.Broadcast()
	b.mu.Unlock()
}

func (b *responseBudget) initializeConditionLocked() {
	if b.condition == nil {
		b.condition = sync.NewCond(&b.mu)
	}
}

func decodeRESTPage(body []byte, symbol string) (restPage, DownloadReason) {
	members, err := strictObjectMembers(body)
	if err != nil {
		return restPage{}, DownloadReasonResponseSizeSyntax
	}
	for _, name := range []string{"status", "ticker", "adjusted", "results", "next_url", "count", "queryCount", "resultsCount"} {
		if len(members[name]) > 1 {
			return restPage{}, DownloadReasonEnvelopeIdentityStatus
		}
	}
	var status, ticker string
	var adjusted bool
	if len(members["status"]) != 1 || json.Unmarshal(members["status"][0], &status) != nil || status != "OK" ||
		len(members["ticker"]) != 1 || json.Unmarshal(members["ticker"][0], &ticker) != nil || ticker != symbol ||
		len(members["adjusted"]) != 1 || json.Unmarshal(members["adjusted"][0], &adjusted) != nil || adjusted {
		return restPage{}, DownloadReasonEnvelopeIdentityStatus
	}
	page := restPage{results: []json.RawMessage{}}
	if len(members["results"]) == 1 && !bytes.Equal(bytes.TrimSpace(members["results"][0]), []byte("null")) {
		if err := json.Unmarshal(members["results"][0], &page.results); err != nil || len(page.results) > liveResultLimit {
			return restPage{}, DownloadReasonResponseSizeSyntax
		}
	}
	if len(members["next_url"]) == 1 {
		if err := json.Unmarshal(members["next_url"][0], &page.nextURL); err != nil {
			return restPage{}, DownloadReasonRedirectContinuation
		}
	}
	for _, name := range []string{"count", "queryCount", "resultsCount"} {
		if len(members[name]) == 0 {
			continue
		}
		number, ok := strictJSONNumber(members[name][0])
		if !ok {
			return restPage{}, DownloadReasonEnvelopeIdentityStatus
		}
		count, ok := exactJSONInt64(number)
		if !ok || count < 0 || count != int64(len(page.results)) {
			return restPage{}, DownloadReasonEnvelopeIdentityStatus
		}
	}
	return page, DownloadReasonNone
}

func strictJSONNumber(raw []byte) (json.Number, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", false
	}
	number, ok := value.(json.Number)
	if !ok {
		return "", false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", false
	}
	return number, true
}

func strictObjectMembers(raw []byte) (map[string][]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("not an object")
	}
	members := make(map[string][]json.RawMessage)
	for decoder.More() {
		nameToken, err := decoder.Token()
		name, ok := nameToken.(string)
		if err != nil || !ok {
			return nil, errors.New("invalid object member")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		members[name] = append(members[name], value)
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, errors.New("unterminated object")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing JSON")
	}
	return members, nil
}

func downloadReasonForNormalization(reason NormalizationRejection) DownloadReason {
	switch reason {
	case RejectionTimestamp:
		return DownloadReasonTimestamp
	case RejectionNumericCount:
		return DownloadReasonNumericCount
	case RejectionStructuralAggregate:
		return DownloadReasonStructuralAggregate
	default:
		return DownloadReasonResponseSizeSyntax
	}
}

func retryAfter(value string, now func() time.Time) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		delay := time.Duration(seconds) * time.Second
		if delay >= 0 && delay <= liveRequestDeadline {
			return delay
		}
		return 0
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := when.Sub(now())
	if delay >= 0 && delay <= liveRequestDeadline {
		return delay
	}
	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
