package engine

import (
	"context"
	"math"
	"sort"
	"testing"
	"time"
)

// TestC3FEAT01PriceRangeBoundaryTable is the sole C3-FEAT-01 primary proof.
// Its dangerous counterexamples are factor-of-100 drift, inclusion of the T
// bar, fabricated missing seconds, zero-width-as-zero, conflict globalization,
// and nonfinite local arithmetic. Exact fields/statuses are observable here;
// API formatting and freshness overlays are intentionally not proved.
func TestC3FEAT01PriceRangeBoundaryTable(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	at := start.Add(60 * time.Minute)
	now := at.Add(2 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	proof := historicalProofContext{
		bindingID: binding.Identity(), generation: 7, token: "token-a", symbol: "AAA",
		intervalStart: start, intervalEnd: at.Add(time.Second),
		result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})},
	}
	inputs := []AggregateInput{
		featureHistorical(binding, "AAA", start.Add(5*time.Minute), 1, 8, 10, 7, 9),
		featureHistorical(binding, "AAA", at.Add(-30*time.Minute-time.Second), 2, 15, 30, 4, 15),
		featureHistorical(binding, "AAA", at.Add(-30*time.Minute), 3, 15, 20, 5, 15),
		featureHistorical(binding, "AAA", at.Add(-time.Second), 4, 15, 16, 14, 15),
		featureHistorical(binding, "AAA", at, 5, 100, 100, 100, 100),
	}
	for _, input := range inputs {
		applyHistorical(t, e, input, proof, DispositionAggregateInserted, ReasonNone)
	}
	proveAggregateCoverage(t, e, "AAA", start, at.Add(time.Second))
	result := priceRangeResult(t, e, "AAA", at)
	want := priceRangeFeatureResult{
		at:             at,
		dayPercent:     currentField(referencePercent(15, 10.25)),
		from4AMPercent: currentField(referencePercent(15, 8)),
		hodDrawdown:    currentField(referencePercent(15, 30)),
		sessionRange:   currentField(100 * (15.0 - 4.0) / (30.0 - 4.0)),
		rolling30:      currentField(100 * (15.0 - 5.0) / (20.0 - 5.0)),
		rolling60:      currentField(100 * (15.0 - 4.0) / (30.0 - 4.0)),
	}
	assertPriceRangeResult(t, result, want)

	for _, test := range []struct {
		name string
		got  aggregateFeatureField
		want aggregateFeatureField
	}{
		{"genuine zero", rangePosition(5, 5, 10), currentField(0)},
		{"genuine one hundred", rangePosition(10, 5, 10), currentField(100)},
		{"zero width unavailable", rangePosition(10, 10, 10), aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}},
		{"closed range invalid", rangePosition(10, 11, 10), aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}},
		{"nonfinite percent invalid", percentChange(math.MaxFloat64, math.SmallestNonzeroFloat64), aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("field = %+v, want %+v", test.got, test.want)
			}
		})
	}
	closeAndWait(t, e)

	t.Run("missing print is absence and early session windows are intersected", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		empty := aggregateEngine(t, binding, RunModeLive, &now)
		proveAggregateCoverage(t, empty, "AAA", start, start.Add(time.Second))
		got := priceRangeResult(t, empty, "AAA", start.Add(time.Second))
		missing := aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
		if got.dayPercent != missing || got.from4AMPercent != missing || got.rolling30 != missing || got.rolling60 != missing {
			t.Fatalf("missing print fabricated a value: %+v", got)
		}
		closeAndWait(t, empty)

		flat := aggregateEngine(t, binding, RunModeLive, &now)
		input := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, flat, input, DispositionAggregateInserted, ReasonNone)
		proveAggregateCoverage(t, flat, "AAA", start, start.Add(time.Second))
		got = priceRangeResult(t, flat, "AAA", start.Add(time.Second))
		zeroWidth := aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}
		if got.rolling30 != zeroWidth || got.rolling60 != zeroWidth || got.sessionRange != zeroWidth {
			t.Fatalf("elapsed-session warm-up replaced exact intersection: %+v", got)
		}
		closeAndWait(t, flat)
	})

	t.Run("resolved historical live discrepancy preserves price and range fields", func(t *testing.T) {
		now := start.Add(time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		first := liveAggregate(binding, "AAA", start, 1, 1)
		first.Values = AggregateValues{Open: 8, High: 10, Low: 7, Close: 9, Volume: 1, VWAP: 9, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage}
		applyAggregate(t, e, first, DispositionAggregateInserted, ReasonNone)
		conflictAt := start.Add(20 * time.Minute)
		now = conflictAt.Add(time.Second)
		live := liveAggregate(binding, "AAA", conflictAt, 1, 2)
		live.Values = AggregateValues{Open: 12, High: 14, Low: 11, Close: 13, Volume: 1, VWAP: 13, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage}
		applyAggregate(t, e, live, DispositionAggregateInserted, ReasonNone)
		historical := featureHistorical(binding, "AAA", conflictAt, 1, 12, 15, 10, 13)
		proof := proofFor(binding, historical, conflictAt, conflictAt.Add(time.Second))
		applyHistorical(t, e, historical, proof, DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		now = start.Add(60 * time.Minute)
		latest := liveAggregate(binding, "AAA", now.Add(-time.Second), 1, 3)
		latest.Values = AggregateValues{Open: 15, High: 16, Low: 14, Close: 15, Volume: 1, VWAP: 15, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage}
		applyAggregate(t, e, latest, DispositionAggregateInserted, ReasonNone)
		proveAggregateCoverage(t, e, "AAA", start, now)
		got := priceRangeResult(t, e, "AAA", now)
		for name, field := range map[string]aggregateFeatureField{
			"day": got.dayPercent, "from4am": got.from4AMPercent, "hod": got.hodDrawdown,
			"session": got.sessionRange, "rolling30": got.rolling30, "rolling60": got.rolling60,
		} {
			if field.status != featureCurrent {
				t.Fatalf("%s was invalidated by a resolved REST/live discrepancy: %+v all=%+v", name, field, got)
			}
		}
		closeAndWait(t, e)
	})

	t.Run("finite canonical inputs cannot leak nonfinite derived fields", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		input := liveAggregate(binding, "AAA", start, 1, 1)
		input.Values = AggregateValues{Open: 1, High: math.MaxFloat64, Low: 1, Close: math.MaxFloat64, Volume: 1, VWAP: math.MaxFloat64, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage}
		applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
		proveAggregateCoverage(t, e, "AAA", start, start.Add(time.Second))
		got := priceRangeResult(t, e, "AAA", start.Add(time.Second))
		if got.dayPercent.status != featureInvalid || got.from4AMPercent.status != featureInvalid || got.hodDrawdown != currentField(0) ||
			got.sessionRange.status != featureInvalid || got.sessionRange.reason != featureReasonInvalidInput {
			t.Fatalf("local arithmetic containment = %+v", got)
		}
		closeAndWait(t, e)
	})
}

// TestC3FEAT02CorrectionPermutationDifferentialTrace is the sole C3-FEAT-02
// primary proof. It compares every accepted insert/correction and timer against
// full canonical recomputation, then delivers the same 70-minute history in
// reverse order. Stale extrema, delivery-order dependence, hidden raw feature
// history, or a same-T correction missed by the central transition are directly
// observable. It intentionally does not establish production latency or the
// later successful run-support/commit proof.
func TestC3FEAT02CorrectionPermutationDifferentialTrace(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start
	e := aggregateEngine(t, binding, RunModeLive, &now)
	proveAggregateCoverage(t, e, "AAA", start, binding.SessionStart().Add(16*time.Hour))
	reference := make(map[int64]AggregateValues, 70*60)
	var frame uint64
	var evaluationT time.Time

	for second := 0; second < 70*60; second++ {
		window := start.Add(time.Duration(second) * time.Second)
		now = window.Add(2 * time.Second)
		frame++
		price := 10 + float64(second%1800)/10_000
		input := liveAggregate(binding, "AAA", window, 1, frame)
		input.Values = AggregateValues{
			Open: price, High: price + float64(second%13)/1000, Low: price - float64(second%11)/1000,
			Close: price, Volume: float64(second % 1000), VWAP: price, AverageTradeSize: int64(second % 50), ATSProvenance: ATSLiveProviderAverage,
		}
		applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
		reference[window.Unix()] = input.Values
		evaluationT = input.WindowEnd
		assertPriceRangeResult(t, priceRangeResult(t, e, "AAA", evaluationT), fullPriceRangeReference(binding, reference, evaluationT))

		if second >= 30 && second%97 == 0 {
			correctedWindow := window.Add(-30 * time.Second)
			frame++
			corrected := liveAggregate(binding, "AAA", correctedWindow, 1, frame)
			corrected.Values = reference[correctedWindow.Unix()]
			corrected.Values.High += 0.01
			corrected.DeliveryTime = now
			applyAggregate(t, e, corrected, DispositionAggregateRevised, ReasonNone)
			reference[correctedWindow.Unix()] = corrected.Values
			assertPriceRangeResult(t, priceRangeResult(t, e, "AAA", evaluationT), fullPriceRangeReference(binding, reference, evaluationT))
		}
		if second%60 == 0 {
			result, completion := e.AdmitTimer(context.Background())
			if result != AdmissionAdmitted {
				t.Fatalf("timer admission = %s", result)
			}
			if got := awaitTimerDisposition(t, completion); got.Code != DispositionTimerApplied {
				t.Fatalf("timer = %+v", got)
			}
			assertPriceRangeResult(t, priceRangeResult(t, e, "AAA", evaluationT), fullPriceRangeReference(binding, reference, evaluationT))
		}
	}

	// Component 2 intentionally has no successful run-support producer yet.
	// Seed its private committed boundary once, then prove the ordinary aggregate
	// transition—not a test-only evaluator—replaces the result at unchanged T.
	e.mu.Lock()
	symbol := &e.state.binding.symbols[e.state.binding.index["AAA"]]
	seed := evaluatePriceRangeFeatures(e.state.binding, symbol, evaluationT)
	symbol.aggregates.priceRange.result = seed
	e.state.committedT = immutableTime(evaluationT)
	e.mu.Unlock()
	frame++
	lastWindow := evaluationT.Add(-time.Second)
	corrected := liveAggregate(binding, "AAA", lastWindow, 1, frame)
	corrected.Values = reference[lastWindow.Unix()]
	corrected.Values.Close -= 0.005
	corrected.Values.Low = min(corrected.Values.Low, corrected.Values.Close)
	corrected.DeliveryTime = now.Add(time.Second)
	now = corrected.DeliveryTime
	applyAggregate(t, e, corrected, DispositionAggregateRevised, ReasonNone)
	reference[lastWindow.Unix()] = corrected.Values
	want := fullPriceRangeReference(binding, reference, evaluationT)
	e.mu.Lock()
	// B2 removed full-population display-formula evaluation from live aggregate
	// mutation; retain this legacy formula differential as explicit test tooling.
	stored := evaluatePriceRangeFeatures(e.state.binding, symbol, evaluationT)
	e.mu.Unlock()
	assertPriceRangeResult(t, stored, want)

	state := aggregateState(t, e, "AAA")
	if retained := len(state.priceRange.sessionHighs) + len(state.priceRange.sessionLows); retained > 2*sessionSeconds || state.priceRange.boundExceeded {
		t.Fatalf("cutoff-bearing extrema retained %d points (bound %d)", retained, 2*sessionSeconds)
	}
	if len(state.priceRange.highs) > maximumExtremaPointsPerDeque || len(state.priceRange.lows) > maximumExtremaPointsPerDeque {
		t.Fatalf("individual extrema deque exceeded one point/second: high=%d low=%d", len(state.priceRange.highs), len(state.priceRange.lows))
	}

	permutedNow := evaluationT.Add(time.Minute)
	permuted := aggregateEngine(t, binding, RunModeLive, &permutedNow)
	proveAggregateCoverage(t, permuted, "AAA", start, binding.SessionStart().Add(16*time.Hour))
	keys := make([]int64, 0, len(reference))
	for key := range reference {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	proof := historicalProofContext{
		bindingID: binding.Identity(), generation: 7, token: "token-a", symbol: "AAA",
		intervalStart: start, intervalEnd: evaluationT,
		result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})},
	}
	for ordinal, key := range keys {
		window := time.Unix(key, 0).UTC()
		input := historicalAggregate(binding, "AAA", window, uint64(ordinal+1))
		input.Values = reference[key]
		applyHistorical(t, permuted, input, proof, DispositionAggregateInserted, ReasonNone)
	}
	assertPriceRangeResult(t, priceRangeResult(t, permuted, "AAA", evaluationT), want)
	permutedState := aggregateState(t, permuted, "AAA")
	if retained := len(permutedState.priceRange.highs) + len(permutedState.priceRange.lows); retained > maximumPriceRangeDequePoints || permutedState.priceRange.boundExceeded {
		t.Fatalf("permuted shared extrema deques retained %d points", retained)
	}

	boundState := &symbolAggregateState{}
	e.mu.Lock()
	installed := e.state.binding
	e.mu.Unlock()
	for second := 0; second < sessionSeconds; second++ {
		window := start.Add(time.Duration(second) * time.Second)
		high, low := 10_000-float64(second), 1+float64(second)
		foldPriceRangeAggregate(boundState, installed, canonicalAggregate{
			windowStart: window, windowEnd: window.Add(time.Second),
			values: AggregateValues{Open: low, High: high, Low: low, Close: high},
		})
	}
	if got := len(boundState.priceRange.sessionHighs) + len(boundState.priceRange.sessionLows); got != 2*sessionSeconds || boundState.priceRange.boundExceeded {
		t.Fatalf("exact cutoff-bearing extrema boundary = %d exceeded=%v, want %d/false", got, boundState.priceRange.boundExceeded, 2*sessionSeconds)
	}
	closeAndWait(t, permuted)
	closeAndWait(t, e)
}

func featureHistorical(binding interface {
	Identity() string
}, symbol string, window time.Time, ordinal uint64, open, high, low, close float64) AggregateInput {
	return AggregateInput{
		SchemaVersion: AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: AggregateSourceHistorical,
		Symbol: symbol, WindowStart: window.UTC(), WindowEnd: window.Add(time.Second).UTC(),
		Values:       AggregateValues{Open: open, High: high, Low: low, Close: close, Volume: 1, VWAP: close, AverageTradeSize: 1, ATSProvenance: ATSRESTFloorVolumeOverTrades},
		DeliveryTime: window.Add(time.Second).UTC(), Historical: HistoricalPosition{Generation: 7, RequestToken: "token-a", RecordOrdinal: ordinal},
	}
}

func priceRangeResult(t *testing.T, e *Engine, symbol string, at time.Time) priceRangeFeatureResult {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	return evaluatePriceRangeFeatures(e.state.binding, &e.state.binding.symbols[index], at)
}

func proveAggregateCoverage(t *testing.T, e *Engine, symbol string, start, end time.Time) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	state := ensureAggregateState(&e.state.binding.symbols[index])
	if !installExactCoverage(state, e.state.binding, start, end, nil) {
		t.Fatalf("could not install exact coverage %s [%s,%s)", symbol, start, end)
	}
}

func currentField(value float64) aggregateFeatureField {
	return aggregateFeatureField{status: featureCurrent, value: value}
}

func assertPriceRangeResult(t *testing.T, got, want priceRangeFeatureResult) {
	t.Helper()
	if got != want {
		t.Fatalf("price/range result\n got: %+v\nwant: %+v", got, want)
	}
}

func fullPriceRangeReference(binding interface {
	SessionStart() time.Time
}, records map[int64]AggregateValues, at time.Time) priceRangeFeatureResult {
	starts := make([]int64, 0, len(records))
	for start := range records {
		if start < at.Unix() {
			starts = append(starts, start)
		}
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	missing := aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
	result := priceRangeFeatureResult{at: at, dayPercent: missing, from4AMPercent: missing, hodDrawdown: missing, sessionRange: missing, rolling30: missing, rolling60: missing}
	if len(starts) == 0 {
		return result
	}
	first, latest := records[starts[0]], records[starts[len(starts)-1]]
	sessionLow, sessionHigh := first.Low, first.High
	for _, start := range starts[1:] {
		value := records[start]
		sessionLow, sessionHigh = min(sessionLow, value.Low), max(sessionHigh, value.High)
	}
	result.dayPercent = currentField(100 * (latest.Close/10.25 - 1))
	result.from4AMPercent = currentField(100 * (latest.Close/first.Open - 1))
	result.hodDrawdown = currentField(100 * (latest.Close/sessionHigh - 1))
	if sessionHigh == sessionLow {
		result.sessionRange = aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}
	} else {
		result.sessionRange = currentField(100 * (latest.Close - sessionLow) / (sessionHigh - sessionLow))
	}
	result.rolling30 = fullRollingReference(records, at, maxTime(binding.SessionStart(), at.Add(-30*time.Minute)), latest.Close)
	result.rolling60 = fullRollingReference(records, at, maxTime(binding.SessionStart(), at.Add(-60*time.Minute)), latest.Close)
	return result
}

func fullRollingReference(records map[int64]AggregateValues, at, start time.Time, last float64) aggregateFeatureField {
	found := false
	var low, high float64
	for unix, value := range records {
		if unix < start.Unix() || unix >= at.Unix() {
			continue
		}
		if !found {
			low, high, found = value.Low, value.High, true
		} else {
			low, high = min(low, value.Low), max(high, value.High)
		}
	}
	if !found {
		return aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
	}
	if high == low {
		return aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}
	}
	return currentField(100 * (last - low) / (high - low))
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func referencePercent(value, base float64) float64 { return 100 * (value/base - 1) }
