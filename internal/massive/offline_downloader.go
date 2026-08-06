package massive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	OfflineRequestDeadline = 15 * time.Second
	OfflinePageLimit       = 2
	OfflineAttemptLimit    = 3
	OfflinePageByteLimit   = int64(16 << 20)
	OfflineResultLimit     = 50_000
	OfflineWorkerLimit     = 8
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

type DownloadAccounting struct {
	PlannedSymbols  int64
	CompleteSymbols int64
	FailedSymbols   int64
	CanceledSymbols int64
	NonemptySymbols int64
	EmptySymbols    int64
	Records         int64
	Pages           int64
	Attempts        int64
	Bytes           int64
}

type DownloadPlan struct {
	Binding                  reference.Binding
	Start, End               time.Time
	Workers                  int
	MaximumNormalizedRecords int64
	MaximumResponseBytes     int64
}

// DownloadResult is sealed by Downloader. Complete is true only when every
// symbol from the immutable binding completed the exact interval successfully.
type DownloadResult struct {
	bindingID  string
	start      time.Time
	end        time.Time
	outcomes   []SymbolOutcome
	records    []RESTSecondAggregate
	accounting DownloadAccounting
	complete   bool
}

func (r DownloadResult) Complete() bool                 { return r.complete }
func (r DownloadResult) BindingIdentity() string        { return r.bindingID }
func (r DownloadResult) Start() time.Time               { return r.start }
func (r DownloadResult) End() time.Time                 { return r.end }
func (r DownloadResult) Outcomes() []SymbolOutcome      { return slices.Clone(r.outcomes) }
func (r DownloadResult) Records() []RESTSecondAggregate { return slices.Clone(r.records) }
func (r DownloadResult) Accounting() DownloadAccounting { return r.accounting }

type CredentialSource func() (string, error)

type OfflineDownloader struct {
	base       *url.URL
	credential CredentialSource
	client     *http.Client
	sleep      func(context.Context, time.Duration) error
	now        func() time.Time
}

func NewOfflineDownloader(baseURL string, credential CredentialSource, client *http.Client) (*OfflineDownloader, error) {
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || (base.Path != "" && base.Path != "/") ||
		base.User != nil || base.RawQuery != "" || base.Fragment != "" || credential == nil || client == nil {
		return nil, errors.New("invalid offline Massive downloader configuration")
	}
	base.Path = ""
	return &OfflineDownloader{base: base, credential: credential, client: client, sleep: sleepWithContext, now: time.Now}, nil
}

func (d *OfflineDownloader) Download(ctx context.Context, plan DownloadPlan) DownloadResult {
	symbols := plan.Binding.UniverseSymbols()
	result := DownloadResult{bindingID: plan.Binding.Identity(), start: plan.Start, end: plan.End}
	if !validDownloadPlan(plan, symbols) {
		return failedPlanResult(result, symbols)
	}
	result.outcomes = make([]SymbolOutcome, len(symbols))
	perSymbolRecords := make([][]RESTSecondAggregate, len(symbols))
	result.accounting.PlannedSymbols = int64(len(symbols))
	if ctx == nil || ctx.Err() != nil {
		for index, symbol := range symbols {
			result.outcomes[index] = SymbolOutcome{Symbol: symbol, State: SymbolCanceled, Reason: DownloadReasonCanceled}
		}
		result.accounting.CanceledSymbols = int64(len(symbols))
		return result
	}
	token, err := d.credential()
	if err != nil || token == "" || strings.ContainsAny(token, "\r\n") {
		for index, symbol := range symbols {
			result.outcomes[index] = SymbolOutcome{Symbol: symbol, State: SymbolFailed, Reason: DownloadReasonRequestConstruction}
		}
		result.accounting.FailedSymbols = int64(len(symbols))
		return result
	}

	workers := min(plan.Workers, len(symbols))
	jobs := make(chan int)
	wireBytes := responseBudget{maximum: plan.MaximumResponseBytes}
	var normalizedRecords atomic.Int64
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for index := range jobs {
				if ctx.Err() != nil {
					result.outcomes[index] = SymbolOutcome{Symbol: symbols[index], State: SymbolCanceled, Reason: DownloadReasonCanceled}
					continue
				}
				result.outcomes[index], perSymbolRecords[index] = d.downloadSymbol(ctx, token, symbols[index], plan, &wireBytes, &normalizedRecords)
			}
		}()
	}
	for index := 0; index < len(symbols); index++ {
		if ctx.Err() != nil {
			for remaining := index; remaining < len(symbols); remaining++ {
				result.outcomes[remaining] = SymbolOutcome{Symbol: symbols[remaining], State: SymbolCanceled, Reason: DownloadReasonCanceled}
			}
			break
		}
		select {
		case jobs <- index:
		case <-ctx.Done():
			for remaining := index; remaining < len(symbols); remaining++ {
				result.outcomes[remaining] = SymbolOutcome{Symbol: symbols[remaining], State: SymbolCanceled, Reason: DownloadReasonCanceled}
			}
			index = len(symbols)
		}
	}
	close(jobs)
	group.Wait()

	for index, outcome := range result.outcomes {
		result.accounting.Pages += outcome.Pages
		result.accounting.Attempts += outcome.Attempts
		result.accounting.Bytes += outcome.Bytes
		switch outcome.State {
		case SymbolComplete:
			result.accounting.CompleteSymbols++
			if outcome.Records == 0 {
				result.accounting.EmptySymbols++
			} else {
				result.accounting.NonemptySymbols++
			}
		case SymbolCanceled:
			result.accounting.CanceledSymbols++
		default:
			result.accounting.FailedSymbols++
		}
		if outcome.State == SymbolComplete {
			result.records = append(result.records, perSymbolRecords[index]...)
		}
	}
	result.accounting.Records = int64(len(result.records))
	result.complete = result.accounting.CompleteSymbols == result.accounting.PlannedSymbols &&
		result.accounting.FailedSymbols == 0 && result.accounting.CanceledSymbols == 0 &&
		result.accounting.CompleteSymbols == result.accounting.NonemptySymbols+result.accounting.EmptySymbols &&
		result.accounting.Records <= plan.MaximumNormalizedRecords
	return result
}

func validDownloadPlan(plan DownloadPlan, symbols []string) bool {
	if plan.Binding.Identity() == "" || len(symbols) == 0 || plan.Workers < 1 || plan.Workers > OfflineWorkerLimit ||
		plan.MaximumNormalizedRecords <= 0 || plan.MaximumResponseBytes <= 0 ||
		plan.Start != plan.Start.UTC() || plan.End != plan.End.UTC() || plan.Start.Nanosecond() != 0 || plan.End.Nanosecond() != 0 ||
		plan.Start.Before(plan.Binding.SessionStart()) || !plan.Start.Before(plan.End) || plan.End.After(plan.Binding.SessionEnd()) {
		return false
	}
	seconds := int64(plan.End.Sub(plan.Start) / time.Second)
	if seconds <= 0 || seconds > 16*60*60 || int64(len(symbols)) > math.MaxInt64/seconds {
		return false
	}
	return int64(len(symbols))*seconds <= plan.MaximumNormalizedRecords
}

func failedPlanResult(result DownloadResult, symbols []string) DownloadResult {
	result.outcomes = make([]SymbolOutcome, len(symbols))
	result.accounting.PlannedSymbols = int64(len(symbols))
	result.accounting.FailedSymbols = int64(len(symbols))
	for index, symbol := range symbols {
		result.outcomes[index] = SymbolOutcome{Symbol: symbol, State: SymbolFailed, Reason: DownloadReasonPlanBudget}
	}
	return result
}

func (d *OfflineDownloader) downloadSymbol(ctx context.Context, token, symbol string, plan DownloadPlan, wireBytes *responseBudget, records *atomic.Int64) (SymbolOutcome, []RESTSecondAggregate) {
	values, outcome := d.downloadSymbolRecords(ctx, token, symbol, plan, wireBytes, records)
	return outcome, values
}

func (d *OfflineDownloader) downloadSymbolRecords(ctx context.Context, token, symbol string, plan DownloadPlan, wireBytes *responseBudget, records *atomic.Int64) ([]RESTSecondAggregate, SymbolOutcome) {
	outcome := SymbolOutcome{Symbol: symbol, State: SymbolFailed}
	endpointPath := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/second/%d/%d", url.PathEscape(symbol), plan.Start.UnixMilli(), plan.End.Add(-time.Millisecond).UnixMilli())
	current := *d.base
	current.Path = endpointPath
	setFixedAggregateQuery(&current)
	seenPages := make(map[string]struct{}, OfflinePageLimit)
	seenIdentities := make(map[int64]struct{})
	values := make([]RESTSecondAggregate, 0)
	for pageNumber := 0; pageNumber < OfflinePageLimit; pageNumber++ {
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
			if value.WindowStart.Before(plan.Start) || !value.WindowStart.Before(plan.End) || value.WindowEnd.After(plan.End) {
				outcome.Reason = DownloadReasonSymbolIntervalOrder
				return nil, outcome
			}
			key := value.WindowStart.Unix()
			if _, duplicate := seenIdentities[key]; duplicate || len(values) > 0 && !values[len(values)-1].WindowStart.Before(value.WindowStart) {
				outcome.Reason = DownloadReasonSymbolIntervalOrder
				return nil, outcome
			}
			seenIdentities[key] = struct{}{}
			if records != nil && records.Add(1) > plan.MaximumNormalizedRecords {
				outcome.Reason = DownloadReasonPlanBudget
				return nil, outcome
			}
			values = append(values, value)
		}
		if page.nextURL == "" {
			outcome.State = SymbolComplete
			outcome.Reason = DownloadReasonNone
			outcome.Records = int64(len(values))
			return values, outcome
		}
		if pageNumber+1 >= OfflinePageLimit {
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

func (d *OfflineDownloader) fetchPage(ctx context.Context, token string, pageURL *url.URL, symbol string, budget *responseBudget) (restPage, int64, int, DownloadReason) {
	var bytesRead int64
	var retryDelay time.Duration
	for attempt := 1; attempt <= OfflineAttemptLimit; attempt++ {
		if attempt > 1 {
			if retryDelay == 0 {
				retryDelay = 25 * time.Millisecond * time.Duration(1<<(attempt-2))
			}
			if err := d.sleep(ctx, retryDelay); err != nil {
				return restPage{}, bytesRead, attempt - 1, DownloadReasonCanceled
			}
			retryDelay = 0
		}
		attemptContext, cancel := context.WithTimeout(ctx, OfflineRequestDeadline)
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
			if attempt == OfflineAttemptLimit {
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
		if errors.Is(readErr, errResponseBudget) || amount > OfflinePageByteLimit {
			return restPage{}, bytesRead, attempt, DownloadReasonResponseSizeSyntax
		}
		if readErr != nil {
			if attempt == OfflineAttemptLimit {
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
			if attempt == OfflineAttemptLimit {
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
	return restPage{}, bytesRead, OfflineAttemptLimit, DownloadReasonHTTPRetryExhausted
}

func (d *OfflineDownloader) validateContinuation(raw, endpointPath string) (*url.URL, DownloadReason) {
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
	query.Set("limit", strconv.Itoa(OfflineResultLimit))
	value.RawQuery = query.Encode()
}

var errResponseBudget = errors.New("offline response byte budget exceeded")

type responseBudget struct {
	mu      sync.Mutex
	used    int64
	maximum int64
}

func (b *responseBudget) read(source io.Reader) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.maximum - b.used
	if remaining <= 0 {
		return nil, errResponseBudget
	}
	limit := min(OfflinePageByteLimit, remaining)
	body, err := io.ReadAll(io.LimitReader(source, limit+1))
	if int64(len(body)) > limit {
		b.used = b.maximum
		return body, errResponseBudget
	}
	b.used += int64(len(body))
	return body, err
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
		if err := json.Unmarshal(members["results"][0], &page.results); err != nil || len(page.results) > OfflineResultLimit {
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
		if delay >= 0 && delay <= OfflineRequestDeadline {
			return delay
		}
		return 0
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := when.Sub(now())
	if delay >= 0 && delay <= OfflineRequestDeadline {
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
