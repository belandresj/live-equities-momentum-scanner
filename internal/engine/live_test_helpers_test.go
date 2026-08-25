package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func testBinding(t *testing.T) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-07-29")
	if err != nil {
		t.Fatal(err)
	}
	priorTimestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v3/reference/tickers":
			_ = json.NewEncoder(writer).Encode(map[string]any{"results": []map[string]any{{"ticker": "AAA", "active": true, "market": "stocks", "locale": "us", "type": "CS"}, {"ticker": "BAD", "active": true, "market": "stocks", "locale": "us", "type": "ADRC"}, {"ticker": "MISSING", "active": true, "market": "stocks", "locale": "us", "type": "CS"}}})
		case "/v2/aggs/grouped/locale/us/market/stocks/2026-07-28":
			_, _ = fmt.Fprintf(writer, `{"status":"OK","adjusted":true,"resultsCount":2,"results":[{"T":"AAA","c":10.25,"t":%d},{"T":"BAD","c":0,"t":%d}]}`, priorTimestamp, priorTimestamp)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := &reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(directory, "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) }, Sleep: func(context.Context, time.Duration) error { return nil }}
	universe, err := resolver.Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priorResolver := &reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(directory, "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) }, Sleep: func(context.Context, time.Duration) error { return nil }}
	priors, err := priorResolver.Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func testEngine(t *testing.T, now time.Time, capacity, reserve int) *Engine {
	t.Helper()
	delay := time.Duration(0)
	e, err := New(Config{Clock: func() time.Time { return now }, Capacity: capacity, RequiredReserve: reserve, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func durationPointer(value time.Duration) *time.Duration { return &value }
func validBindingInput(binding reference.Binding) BindingInstall {
	return BindingInstall{SchemaVersion: BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding}
}
func admitValidBinding(t *testing.T, e *Engine, binding reference.Binding) <-chan Disposition {
	t.Helper()
	result, completion := e.AdmitBinding(context.Background(), validBindingInput(binding))
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("binding admission = %s", result)
	}
	return completion
}
func awaitDisposition(t *testing.T, completion <-chan Disposition) Disposition {
	t.Helper()
	select {
	case disposition := <-completion:
		return disposition
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for disposition")
		return Disposition{}
	}
}
func closeAndWait(t *testing.T, e *Engine) {
	t.Helper()
	e.Close()
	if err := e.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
}
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func mutateIdentity(identity string) string {
	last := "0"
	if identity[len(identity)-1:] == last {
		last = "1"
	}
	return identity[:len(identity)-1] + last
}

func awaitTimerDisposition(t *testing.T, completion <-chan TimerDisposition) TimerDisposition {
	t.Helper()
	select {
	case disposition := <-completion:
		return disposition
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for timer disposition")
		return TimerDisposition{}
	}
}

func installEvaluatorMarkOnSymbol(symbol *coreSymbol, at time.Time, price float64, status qualificationStatus) {
	window := at.Add(-time.Second)
	record := &canonicalAggregate{identity: aggregateIdentity{symbol: symbol.symbol, start: window.Unix()}, windowStart: window, windowEnd: at, values: AggregateValues{Open: price, High: price, Low: price, Close: price, Volume: 1, VWAP: price, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage}}
	qualification := &qualificationState{finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{}, accountedThrough: at, result: qualificationResult{at: at, status: status}}
	if status == qualificationProvisional {
		qualification.proofs[at.Unix()] = struct{}{}
	}
	if status == qualificationUnresolved {
		qualification.accountedThrough = time.Time{}
	}
	if status == qualificationFinalized {
		qualification.finalized = true
		qualification.finalProofEnd = at.Add(-time.Minute)
		qualification.result.finalProofEnd = at.Add(-time.Minute)
	}
	qualification.unresolvedOrigin = uncertaintyBootstrapOrigin
	qualification.result.unresolvedOrigin = uncertaintyBootstrapOrigin
	symbol.aggregates = &symbolAggregateState{tail: map[int64]*canonicalAggregate{window.Unix(): record}, latest: &latestAggregateMark{record: *record}, qualification: qualification}
}

type evaluatorSymbol struct {
	symbol        string
	priorStatus   reference.PriorCloseStatus
	prior, mark   float64
	qualification qualificationStatus
}

func evaluatorProofEngine(at time.Time, specs []evaluatorSymbol) *Engine {
	start := at.Add(-2 * time.Hour)
	binding := &installedBinding{identity: "proof-binding", tradingDate: "2026-07-29", sessionStart: start, sessionEnd: start.Add(16 * time.Hour), symbols: make([]coreSymbol, len(specs)), index: make(map[string]int, len(specs))}
	for i, spec := range specs {
		binding.index[spec.symbol] = i
		binding.symbols[i] = coreSymbol{symbol: spec.symbol, prior: frozenPriorClose{symbol: spec.symbol, status: spec.priorStatus, close: spec.prior}}
		if spec.mark != 0 {
			installEvaluatorMarkOnSymbol(&binding.symbols[i], at, spec.mark, spec.qualification)
			coverageStart := binding.sessionStart
			if spec.qualification == qualificationUnresolved {
				coverageStart = coverageStart.Add(time.Second)
			}
			installExactCoverage(binding.symbols[i].aggregates, binding, coverageStart, at, nil)
			state := binding.symbols[i].aggregates
			ensurePriceRangeState(state).result = evaluateTestPriceRangeFeatures(binding, &binding.symbols[i], at)
			ensureMVPMeasurementState(state).result = evaluateMVPMeasurements(binding, state, at, nil)
		}
	}
	e := &Engine{state: &engineState{binding: binding, lifecycle: lifecycleLive, committedT: immutableTime(at), clockMonotonic: true}}
	e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
	for i, spec := range specs {
		if spec.mark == 0 && spec.priorStatus == reference.PriorCloseValid {
			e.state.aggregateEvaluator.coverage[i] = coverageUnknownFailureOrFence
		}
	}
	return e
}

func assertCanonicalClose(t *testing.T, e *Engine, symbol string, window time.Time, close float64) {
	t.Helper()
	if got := aggregateRecord(t, e, symbol, window).values.Close; got != close {
		t.Fatalf("close=%v want=%v", got, close)
	}
}

func aggregateEngine(t *testing.T, binding reference.Binding, now *time.Time) *Engine {
	t.Helper()
	delay := time.Duration(0)
	e, err := New(Config{Clock: func() time.Time { return *now }, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}
	return e
}

func liveAggregate(binding reference.Binding, symbol string, window time.Time, epoch, frame uint64) AggregateInput {
	return AggregateInput{SchemaVersion: AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: AggregateSourceLive,
		Symbol: symbol, WindowStart: window.UTC(), WindowEnd: window.Add(time.Second).UTC(),
		Values:       AggregateValues{Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage},
		DeliveryTime: window.Add(time.Second).UTC(), Live: LivePosition{ConnectionEpoch: epoch, FrameSequence: frame}}
}

func historicalAggregate(binding reference.Binding, symbol string, window time.Time, ordinal uint64) AggregateInput {
	input := liveAggregate(binding, symbol, window, 1, 1)
	input.Source, input.Live = AggregateSourceHistorical, LivePosition{}
	input.Historical = HistoricalPosition{Generation: 7, RequestToken: "token-a", RecordOrdinal: ordinal}
	input.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
	return input
}

func proofFor(binding reference.Binding, input AggregateInput, start, end time.Time) historicalProofContext {
	return historicalProofContext{bindingID: binding.Identity(), generation: input.Historical.Generation, token: input.Historical.RequestToken,
		symbol: input.Symbol, intervalStart: start.UTC(), intervalEnd: end.UTC(),
		result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})}}
}

func changedClose(input AggregateInput, close float64) AggregateInput {
	input.Values.Close = close
	if close > input.Values.High {
		input.Values.High = close
	}
	if close < input.Values.Low {
		input.Values.Low = close
	}
	return input
}

func applyAggregate(t *testing.T, e *Engine, input AggregateInput, code DispositionCode, reason DispositionReason) AggregateDisposition {
	t.Helper()
	result, completion := e.admitAggregateForProof(input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("admission = %s", result)
	}
	got := awaitAggregateDisposition(t, completion)
	if got.Code != code || got.Reason != reason {
		t.Fatalf("disposition = %+v, want %s/%s", got, code, reason)
	}
	return got
}

func applyHistorical(t *testing.T, e *Engine, input AggregateInput, proof historicalProofContext, code DispositionCode, reason DispositionReason) AggregateDisposition {
	t.Helper()
	result, completion := e.admitHistoricalAggregateForProof(input, proof)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("historical admission = %s", result)
	}
	got := awaitAggregateDisposition(t, completion)
	if got.Code != code || got.Reason != reason {
		t.Fatalf("historical disposition = %+v, want %s/%s", got, code, reason)
	}
	advanceHistoricalProofResult(proof.result, input, got.Code)
	return got
}

func advanceHistoricalProofResult(result *historicalProofResult, input AggregateInput, code DispositionCode) {
	if result == nil {
		return
	}
	identity := aggregateIdentity{symbol: input.Symbol, start: input.WindowStart.Unix()}
	switch code {
	case DispositionAggregateInserted, DispositionAggregateRevised:
		result.records[identity] = canonicalAggregate{identity: identity, windowStart: input.WindowStart, windowEnd: input.WindowEnd, values: input.Values,
			first: aggregateEvidence{source: input.Source, deliveryTime: input.DeliveryTime, historical: input.Historical}, authority: aggregateEvidence{source: input.Source, deliveryTime: input.DeliveryTime, historical: input.Historical}}
	case DispositionAggregateWithdrawn:
		delete(result.records, identity)
		result.conflicts[identity] = struct{}{}
	}
}

func awaitAggregateDisposition(t *testing.T, completion <-chan AggregateDisposition) AggregateDisposition {
	t.Helper()
	select {
	case got := <-completion:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for aggregate disposition")
		return AggregateDisposition{}
	}
}

func aggregateState(t *testing.T, e *Engine, symbol string) *symbolAggregateState {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index, ok := e.state.binding.index[symbol]
	if !ok || e.state.binding.symbols[index].aggregates == nil {
		t.Fatalf("no aggregate state for %s", symbol)
	}
	return e.state.binding.symbols[index].aggregates
}

func aggregateRecord(t *testing.T, e *Engine, symbol string, window time.Time) canonicalAggregate {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.state.binding.symbols[e.state.binding.index[symbol]].aggregates
	if state != nil && state.tail[window.Unix()] != nil {
		return *state.tail[window.Unix()]
	}
	if state != nil && state.latest != nil && state.latest.record.identity.start == window.Unix() {
		return state.latest.record
	}
	t.Fatalf("no aggregate record for %s/%s", symbol, window)
	return canonicalAggregate{}
}

func assertNoIdentity(t *testing.T, e *Engine, symbol string, window time.Time) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.state.binding.symbols[e.state.binding.index[symbol]].aggregates
	if state != nil && state.tail[window.Unix()] != nil {
		t.Fatal("identity remained in tail")
	}
	if state != nil && state.latest != nil && state.latest.record.identity.start == window.Unix() {
		t.Fatal("identity remained latest")
	}
}

type controlledClock struct {
	mu    sync.Mutex
	now   time.Time
	reads int
}

func (c *controlledClock) read() time.Time   { c.mu.Lock(); defer c.mu.Unlock(); c.reads++; return c.now }
func (c *controlledClock) set(now time.Time) { c.mu.Lock(); c.now = now; c.mu.Unlock() }
func (c *controlledClock) count() int        { c.mu.Lock(); defer c.mu.Unlock(); return c.reads }
func newS3Engine(t *testing.T, clock Clock, capacity, reserve int, delay time.Duration) *Engine {
	t.Helper()
	e, err := New(Config{Clock: clock, Capacity: capacity, RequiredReserve: reserve, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func admitIllegalForProof(e *Engine) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: inputIllegal}, false)
}
