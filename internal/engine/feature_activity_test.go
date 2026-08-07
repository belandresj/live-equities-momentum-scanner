package engine

import (
	"context"
	"math"
	"math/big"
	"sort"
	"testing"
	"time"
)

// TestC3ACT01TargetReferencePercentileBoundaryTable is the sole C3-ACT-01
// primary proof. Dangerous counterexamples are an all-session/current-target
// reference, fabricated missing seconds, ATS zero becoming zero transactions,
// exclusion of transactions==100, strict tie percentiles, flat-block
// invalidation, nonfinite leakage, and Activity changing independent fields.
// Exact components/counts/percentiles/statuses distinguish conformance. The
// proof intentionally does not establish provider ATS normalization/equality,
// public formatting, freshness, or trading value.
func TestC3ACT01TargetReferencePercentileBoundaryTable(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(30 * time.Minute)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	e.mu.Lock()
	installed := e.state.binding
	index := installed.index["AAA"]
	symbol := &installed.symbols[index]
	state := ensureAggregateState(symbol)
	e.mu.Unlock()

	t.Run("exact arithmetic ten references inclusive ties and current exclusion", func(t *testing.T) {
		resetActivityTestState(state, installed)
		start := installed.sessionStart
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 10; block++ {
			expansion := float64(block)
			values := activityValues(float64(100+block), expansion)
			installActivityTestRecord(state, start.Add(time.Duration(block)*activityBlockDuration), values)
		}
		installActivityTestRecord(state, at.Add(-activityBlockDuration), activityValues(105, 5))
		// A bar beginning at T is outside the target and cannot become reference.
		boundary := activityValues(1_000_000, 1_000)
		boundary.AverageTradeSize = 0
		installActivityTestRecord(state, at, boundary)

		got := evaluateActivityFeatures(installed, state, at)
		if got.activity != currentField(60) || got.targetTransactions != 105 || !closeFloat(got.targetExpansionBPS, 5) ||
			got.referenceCount != 10 || got.transactionPercentile != 60 || got.expansionPercentile != 60 {
			t.Fatalf("exact Activity = %+v", got)
		}
	})

	t.Run("reference floor and nine versus ten availability", func(t *testing.T) {
		resetActivityTestState(state, installed)
		start := installed.sessionStart
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 9; block++ {
			installActivityTestRecord(state, start.Add(time.Duration(block)*activityBlockDuration), activityValues(100, 0))
		}
		installActivityTestRecord(state, start.Add(9*activityBlockDuration), activityValues(99.999, 0))
		installActivityTestRecord(state, at.Add(-activityBlockDuration), activityValues(100, 0))
		got := evaluateActivityFeatures(installed, state, at)
		if got.referenceCount != 9 || got.activity.status != featureWarming || got.activity.reason != featureReasonReferenceWarmup {
			t.Fatalf("nine-reference result = %+v", got)
		}
		state.tail[start.Add(9*activityBlockDuration).Unix()].values = activityValues(100, 0)
		got = evaluateActivityFeatures(installed, state, at)
		if got.referenceCount != 10 || got.activity != currentField(100) {
			t.Fatalf("inclusive 100 boundary = %+v", got)
		}
	})

	t.Run("sparse flat blocks are valid and missing seconds are absence", func(t *testing.T) {
		resetActivityTestState(state, installed)
		start := installed.sessionStart
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 11; block++ {
			// One aggregate per block proves that 29 missing seconds are not
			// synthesized, while a genuine flat block has zero expansion.
			installActivityTestRecord(state, start.Add(time.Duration(block)*activityBlockDuration+29*time.Second), activityValues(100, 0))
		}
		got := evaluateActivityFeatures(installed, state, at)
		if got.activity != currentField(100) || got.targetExpansionBPS != 0 || got.referenceCount != 10 {
			t.Fatalf("sparse flat Activity = %+v", got)
		}
	})

	t.Run("target ATS coverage arithmetic and field-local containment", func(t *testing.T) {
		resetActivityTestState(state, installed)
		start := installed.sessionStart
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 10; block++ {
			installActivityTestRecord(state, start.Add(time.Duration(block)*activityBlockDuration), activityValues(100, 0))
		}
		installActivityTestRecord(state, at.Add(-activityBlockDuration), activityValues(100, 0))
		priceBefore := evaluatePriceRangeFeatures(installed, symbol, at)
		state.tail[at.Add(-activityBlockDuration).Unix()].values.AverageTradeSize = 0
		invalid := evaluateActivityFeatures(installed, state, at)
		priceAfter := evaluatePriceRangeFeatures(installed, symbol, at)
		if invalid.activity.status != featureInvalid || invalid.activity.reason != featureReasonInvalidInput || priceAfter != priceBefore {
			t.Fatalf("ATS containment activity=%+v price before=%+v after=%+v", invalid, priceBefore, priceAfter)
		}
		state.tail[at.Add(-activityBlockDuration).Unix()].values = activityValues(100, 0)
		state.provenAbsent = nil
		unavailable := evaluateActivityFeatures(installed, state, at)
		if unavailable.activity.status != featureUnavailable || unavailable.activity.reason != featureReasonHistoryIncomplete {
			t.Fatalf("coverage result = %+v", unavailable)
		}
	})

	t.Run("finite input overflow and local state bound never leak nonfinite", func(t *testing.T) {
		resetActivityTestState(state, installed)
		start := installed.sessionStart
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 10; block++ {
			installActivityTestRecord(state, start.Add(time.Duration(block)*activityBlockDuration), activityValues(100, 0))
		}
		values := activityValues(math.MaxFloat64, 0)
		installActivityTestRecord(state, at.Add(-2*time.Second), values)
		installActivityTestRecord(state, at.Add(-time.Second), values)
		got := evaluateActivityFeatures(installed, state, at)
		if got.activity.status != featureInvalid || got.activity.reason != featureReasonInvalidInput || !finiteFeature(got.activity.value) {
			t.Fatalf("overflow containment = %+v", got)
		}
		state.activity.boundExceeded = true
		got = evaluateActivityFeatures(installed, state, at)
		if got.activity.status != featureInvalid || got.activity.reason != featureReasonStateBoundExceeded {
			t.Fatalf("bound containment = %+v", got)
		}
	})

	closeAndWait(t, e)
}

// TestC3ACT02CorrectionLongPathDifferentialTrace is the sole C3-ACT-02
// primary proof. It compares insert/revision/reference insert-remove-replace,
// withdrawal/conflict, expiry, timer maintenance, horizon equality/strict
// fold, and forward/reverse delivery with full recomputation after every event.
// Stale membership/percentiles, delivery-order dependence, all-session/raw
// retention, >119 references, >33 mutable IDs, price/range mutation, or rank
// state are observable. Checkpoint/restart equivalence, provider ATS mapping,
// production latency, and Component 8 capacity remain intentionally unproved.
func TestC3ACT02CorrectionLongPathDifferentialTrace(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("insert revision remove replace expiry timer and horizon", func(t *testing.T) {
		now := start
		e := aggregateEngine(t, binding, RunModeLive, &now)
		proveAggregateCoverage(t, e, "AAA", start, start.Add(16*time.Hour))
		reference := make(map[int64]AggregateValues)
		var frame uint64
		at := start.Add(11 * activityBlockDuration)
		for block := 0; block < 11; block++ {
			window := start.Add(time.Duration(block) * activityBlockDuration)
			now = window.Add(time.Second)
			frame++
			input := liveAggregate(binding, "AAA", window, 1, frame)
			input.Values = activityValues(100, float64(block%4))
			input.DeliveryTime = now
			applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
			reference[window.Unix()] = input.Values
			assertActivityResult(t, activityResult(t, e, "AAA", at), fullActivityReference(start, reference, at, true))
		}

		// Remove, insert, then replace one reference through accepted revisions.
		window := start.Add(2 * activityBlockDuration)
		for _, tx := range []float64{99.999, 100, 250} {
			frame++
			now = at
			input := liveAggregate(binding, "AAA", window, 1, frame)
			input.Values = activityValues(tx, 7)
			input.DeliveryTime = now
			applyAggregate(t, e, input, DispositionAggregateRevised, ReasonNone)
			reference[window.Unix()] = input.Values
			assertActivityResult(t, activityResult(t, e, "AAA", at), fullActivityReference(start, reference, at, true))
		}
		// Anchor the accepted same-T result before wall-clock time reaches the
		// correction horizon; folded target evidence then remains fixed-scalar
		// sufficient state rather than a raw Activity bar copy.
		e.mu.Lock()
		e.state.committedT = immutableTime(at)
		activity := ensureActivityState(e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates)
		applyActivityResult(activity, evaluateActivityFeatures(e.state.binding, e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates, at))
		e.mu.Unlock()

		// The oldest identity remains mutable at exact H equality.
		horizonWindow := at.Add(-time.Second)
		frame++
		now = horizonWindow.Add(time.Second).Add(correctionHorizon)
		input := liveAggregate(binding, "AAA", horizonWindow, 1, frame)
		input.Values = activityValues(200, 2)
		input.DeliveryTime = now
		applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
		reference[horizonWindow.Unix()] = input.Values
		assertActivityResult(t, activityResult(t, e, "AAA", at), fullActivityReference(start, reference, at, true))
		state := aggregateState(t, e, "AAA")
		if state.tail[horizonWindow.Unix()] == nil {
			t.Fatal("horizon-equality evidence folded early")
		}

		// One nanosecond strictly beyond H folds through the same timer path.
		now = horizonWindow.Add(time.Second).Add(correctionHorizon).Add(time.Nanosecond)
		result, completion := e.AdmitTimer(context.Background())
		if result != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
			t.Fatalf("strict-fold timer admission=%s", result)
		}
		state = aggregateState(t, e, "AAA")
		if state.tail[horizonWindow.Unix()] != nil {
			t.Fatal("strictly old evidence remained in canonical tail")
		}

		// Advancing T expires old references. Seed only the existing central T
		// field because Component 2 intentionally has no run-support producer yet.
		advanced := start.Add(75 * time.Minute)
		e.mu.Lock()
		e.state.committedT = immutableTime(advanced)
		e.mu.Unlock()
		now = advanced
		result, completion = e.AdmitTimer(context.Background())
		if result != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
			t.Fatalf("expiry timer admission=%s", result)
		}
		state = aggregateState(t, e, "AAA")
		if len(state.activity.references) != 0 {
			t.Fatalf("expired references retained: %d", len(state.activity.references))
		}
		assertActivityBounds(t, state)
		closeAndWait(t, e)
	})

	t.Run("historical withdrawal contains Activity only", func(t *testing.T) {
		at := start.Add(11 * activityBlockDuration)
		now := at
		e := aggregateEngine(t, binding, RunModeLive, &now)
		proveAggregateCoverage(t, e, "AAA", start, start.Add(16*time.Hour))
		proof := historicalProofContext{
			bindingID: binding.Identity(), generation: 7, token: "token-a", symbol: "AAA", intervalStart: start, intervalEnd: at,
			result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})},
		}
		input := historicalAggregate(binding, "AAA", start, 1)
		input.Values = activityValues(100, 0)
		applyHistorical(t, e, input, proof, DispositionAggregateInserted, ReasonNone)
		conflict := input
		conflict.Historical.RecordOrdinal = 2
		conflict.Values = activityValues(200, 1)
		applyHistorical(t, e, conflict, proof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		target := historicalAggregate(binding, "AAA", at.Add(-time.Second), 3)
		target.Values = activityValues(100, 0)
		applyHistorical(t, e, target, proof, DispositionAggregateInserted, ReasonNone)
		got := activityResult(t, e, "AAA", at)
		if got.activity.status != featureUnavailable || got.activity.reason != featureReasonHistoryIncomplete {
			t.Fatalf("withdrawal coverage containment = %+v", got)
		}
		closeAndWait(t, e)
	})

	t.Run("exact mutable block identifier bound fails locally", func(t *testing.T) {
		now := start.Add(2 * time.Hour)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		e.mu.Lock()
		installed := e.state.binding
		symbol := &installed.symbols[installed.index["AAA"]]
		state := ensureAggregateState(symbol)
		activity := ensureActivityState(state)
		for block := 1; block <= maximumMutableActivityBlockIDs; block++ {
			end := start.Add(time.Duration(block) * activityBlockDuration).Unix()
			summary := finishActivitySummary(addActivityAggregate(activityBlockSummary{end: end, low: math.Inf(1)}, activityValues(100, 0)))
			activity.mutable[end] = activityMutableBlock{folded: summary, current: summary}
		}
		if len(activity.mutable) != maximumMutableActivityBlockIDs || activity.boundExceeded {
			t.Fatalf("exact mutable boundary=%d exceeded=%v", len(activity.mutable), activity.boundExceeded)
		}
		priceBefore := evaluatePriceRangeFeatures(installed, symbol, start.Add(20*time.Minute))
		foldActivityAggregate(state, installed, canonicalAggregate{
			windowStart: start.Add(time.Duration(maximumMutableActivityBlockIDs) * activityBlockDuration),
			windowEnd:   start.Add(time.Duration(maximumMutableActivityBlockIDs)*activityBlockDuration + time.Second),
			values:      activityValues(100, 0),
		}, start)
		got := evaluateActivityFeatures(installed, state, start.Add(20*time.Minute))
		priceAfter := evaluatePriceRangeFeatures(installed, symbol, start.Add(20*time.Minute))
		e.mu.Unlock()
		if !activity.boundExceeded || len(activity.mutable) != 0 || len(activity.references) != 0 ||
			got.activity.status != featureInvalid || got.activity.reason != featureReasonStateBoundExceeded || priceAfter != priceBefore {
			t.Fatalf("mutable bound containment activity=%+v refs=%d mutable=%d price before=%+v after=%+v", got, len(activity.references), len(activity.mutable), priceBefore, priceAfter)
		}
		closeAndWait(t, e)
	})

	t.Run("long path and delivery permutation", func(t *testing.T) {
		at := start.Add(70 * time.Minute)
		keys := make([]int64, 0, 420)
		values := make(map[int64]AggregateValues, 420)
		components := [...]float64{33.1, 33.2, 33.7}
		for block := 0; block < 140; block++ {
			for component, transactions := range components {
				window := start.Add(time.Duration(block)*activityBlockDuration + time.Duration(component)*10*time.Second)
				keys = append(keys, window.Unix())
				values[window.Unix()] = activityValues(transactions+float64(block%17)/10, float64(block%23+component))
			}
		}
		forward := runActivityPermutationTrace(t, binding, at, keys, values)
		reverseKeys := append([]int64(nil), keys...)
		sort.Slice(reverseKeys, func(i, j int) bool { return reverseKeys[i] > reverseKeys[j] })
		reverse := runActivityPermutationTrace(t, binding, at, reverseKeys, values)
		assertActivityResult(t, reverse.result, forward.result)
		if reverse.references != forward.references || reverse.mutable != forward.mutable {
			t.Fatalf("retained occupancy differs forward=%d/%d reverse=%d/%d", forward.references, forward.mutable, reverse.references, reverse.mutable)
		}
	})
}

type activityTraceResult struct {
	result              activityFeatureResult
	references, mutable int
}

func runActivityPermutationTrace(t *testing.T, binding interface {
	Identity() string
	SessionStart() time.Time
}, at time.Time, order []int64, values map[int64]AggregateValues) activityTraceResult {
	t.Helper()
	// testBinding's concrete type is needed by the accepted engine helpers.
	concrete := testBinding(t)
	if concrete.Identity() != binding.Identity() || !concrete.SessionStart().Equal(binding.SessionStart()) {
		t.Fatal("unexpected binding fixture identity")
	}
	now := at
	e := aggregateEngine(t, concrete, RunModeLive, &now)
	proveAggregateCoverage(t, e, "AAA", concrete.SessionStart(), concrete.SessionStart().Add(16*time.Hour))
	accepted := make(map[int64]AggregateValues, len(order))
	proof := historicalProofContext{
		bindingID: concrete.Identity(), generation: 7, token: "token-a", symbol: "AAA", intervalStart: concrete.SessionStart(), intervalEnd: at,
		result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})},
	}
	for ordinal, key := range order {
		window := time.Unix(key, 0).UTC()
		input := historicalAggregate(concrete, "AAA", window, uint64(ordinal+1))
		input.Values = values[key]
		applyHistorical(t, e, input, proof, DispositionAggregateInserted, ReasonNone)
		accepted[key] = input.Values
		assertActivityResult(t, activityResult(t, e, "AAA", at), fullActivityReference(concrete.SessionStart(), accepted, at, true))
		state := aggregateState(t, e, "AAA")
		assertActivityBounds(t, state)
		wantReferences, wantMutable := fullActivityOccupancy(concrete.SessionStart(), accepted, at, now)
		if len(state.activity.references) != wantReferences || len(state.activity.mutable) != wantMutable {
			t.Fatalf("retained occupancy after event %d = %d/%d want %d/%d", ordinal, len(state.activity.references), len(state.activity.mutable), wantReferences, wantMutable)
		}
	}
	// Fold the complete canonical tail through an ordered timer while holding
	// target T fixed with the engine's configured delay. This reaches the exact
	// 119-reference boundary and proves target sufficient state survives strict
	// folding without a raw bar copy.
	delay := correctionHorizon + time.Nanosecond
	e.mu.Lock()
	e.delay = delay
	e.state.committedT = immutableTime(at)
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	activity := ensureActivityState(state)
	applyActivityResult(activity, evaluateActivityFeatures(e.state.binding, state, at))
	e.mu.Unlock()
	now = at.Add(delay)
	admission, completion := e.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatalf("long-path fold timer admission=%s", admission)
	}
	assertActivityResult(t, activityResult(t, e, "AAA", at), fullActivityReference(concrete.SessionStart(), accepted, at, true))
	state = aggregateState(t, e, "AAA")
	if len(state.activity.references) > maximumActivityReferences {
		t.Fatalf("retained reference boundary=%d exceeds %d", len(state.activity.references), maximumActivityReferences)
	}
	result := activityTraceResult{result: activityResult(t, e, "AAA", at), references: len(state.activity.references), mutable: len(state.activity.mutable)}
	closeAndWait(t, e)
	return result
}

func resetActivityTestState(state *symbolAggregateState, binding *installedBinding) {
	state.tail = make(map[int64]*canonicalAggregate)
	state.presence = nil
	state.provenAbsent = nil
	state.historicalConflict = nil
	state.latest, state.olderLatest = nil, nil
	state.priceRange = nil
	state.activity = nil
	installExactCoverage(state, binding, binding.sessionStart, binding.sessionEnd)
}

func installActivityTestRecord(state *symbolAggregateState, window time.Time, values AggregateValues) {
	record := &canonicalAggregate{
		identity: aggregateIdentity{symbol: "AAA", start: window.Unix()}, windowStart: window, windowEnd: window.Add(time.Second), values: values,
	}
	state.tail[window.Unix()] = record
	if state.latest == nil || window.After(state.latest.record.windowStart) {
		state.latest = &latestAggregateMark{record: *record}
	}
}

func activityValues(transactions, expansionBPS float64) AggregateValues {
	low := 10.0
	high := low * math.Exp(expansionBPS/10_000)
	return AggregateValues{
		Open: low, High: high, Low: low, Close: high, Volume: transactions, VWAP: low,
		AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage,
	}
}

func activityResult(t *testing.T, e *Engine, symbol string, at time.Time) activityFeatureResult {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	return evaluateActivityFeatures(e.state.binding, e.state.binding.symbols[index].aggregates, at)
}

func assertActivityResult(t *testing.T, got, want activityFeatureResult) {
	t.Helper()
	if got.at != want.at || got.activity != want.activity || got.referenceCount != want.referenceCount ||
		!sameFloat(got.targetTransactions, want.targetTransactions) || !sameFloat(got.targetExpansionBPS, want.targetExpansionBPS) ||
		!sameFloat(got.transactionPercentile, want.transactionPercentile) || !sameFloat(got.expansionPercentile, want.expansionPercentile) {
		t.Fatalf("Activity result\n got: %+v\nwant: %+v", got, want)
	}
}

func assertActivityBounds(t *testing.T, state *symbolAggregateState) {
	t.Helper()
	if state.activity == nil {
		return
	}
	if len(state.activity.references) > maximumActivityReferences || len(state.activity.mutable) > maximumMutableActivityBlockIDs ||
		len(state.activity.foldedTargets) > maximumActivityTargetBlocks || state.activity.foldedTargetContributions > maximumActivityTargetContributions ||
		state.activity.boundExceeded {
		t.Fatalf("Activity bounds refs=%d mutable=%d target_blocks=%d target_contributions=%d exceeded=%v",
			len(state.activity.references), len(state.activity.mutable), len(state.activity.foldedTargets), state.activity.foldedTargetContributions, state.activity.boundExceeded)
	}
}

func fullActivityReference(sessionStart time.Time, records map[int64]AggregateValues, at time.Time, coverage bool) activityFeatureResult {
	result := unavailableActivityResult(at)
	if !coverage {
		result.activity = aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
		return result
	}
	target := fullActivityComponents(records, at.Add(-activityBlockDuration), at)
	if target.invalid {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		return result
	}
	if target.aggregateCount == 0 {
		result.activity = aggregateFeatureField{status: featureUnavailable, reason: featureReasonNoAggregateInTarget}
		return result
	}
	result.targetTransactions, result.targetExpansionBPS = target.transactions, target.expansionBPS
	floor := at.Add(-60 * time.Minute)
	if floor.Before(sessionStart) {
		floor = sessionStart
	}
	upper := at.Add(-activityBlockDuration)
	txReferences := make([]float64, 0, maximumActivityEvaluationReferences)
	expansionReferences := make([]float64, 0, maximumActivityEvaluationReferences)
	for blockStart := sessionStart; blockStart.Add(activityBlockDuration).Compare(upper) <= 0; blockStart = blockStart.Add(activityBlockDuration) {
		if blockStart.Before(floor) {
			continue
		}
		summary := fullActivityComponents(records, blockStart, blockStart.Add(activityBlockDuration))
		if summary.invalid {
			result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
			return result
		}
		if summary.aggregateCount == 0 {
			continue
		}
		if summary.transactions >= minimumActivityReferenceTx {
			txReferences = append(txReferences, summary.transactions)
			expansionReferences = append(expansionReferences, summary.expansionBPS)
		}
	}
	result.referenceCount = len(txReferences)
	if result.referenceCount < minimumActivityReferences {
		result.activity = aggregateFeatureField{status: featureWarming, reason: featureReasonReferenceWarmup}
		return result
	}
	result.transactionPercentile = referencePercentile(txReferences, target.transactions)
	result.expansionPercentile = referencePercentile(expansionReferences, target.expansionBPS)
	value := math.Sqrt(result.transactionPercentile * result.expansionPercentile)
	if !finiteFeature(value) || value < 0 || value > 100 {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		return result
	}
	result.activity = currentField(value)
	return result
}

func fullActivityComponents(records map[int64]AggregateValues, start, end time.Time) activityBlockSummary {
	keys := make([]int64, 0, len(records))
	for key := range records {
		if key >= start.Unix() && key < end.Unix() {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	summary := activityBlockSummary{low: math.Inf(1)}
	var exact big.Rat
	for _, key := range keys {
		values := records[key]
		if values.AverageTradeSize <= 0 {
			summary.invalid = true
			return summary
		}
		component := values.Volume / float64(values.AverageTradeSize)
		if !finiteFeature(component) {
			summary.invalid = true
			return summary
		}
		var term big.Rat
		term.SetFloat64(component)
		exact.Add(&exact, &term)
		if summary.aggregateCount == 0 {
			summary.high, summary.low = values.High, values.Low
		} else {
			summary.high, summary.low = max(summary.high, values.High), min(summary.low, values.Low)
		}
		summary.aggregateCount++
	}
	if summary.aggregateCount == 0 {
		return summary
	}
	summary.transactions, _ = new(big.Float).SetPrec(53).SetMode(big.ToNearestEven).SetRat(&exact).Float64()
	summary.expansionBPS = 10_000 * math.Log(summary.high/summary.low)
	if !finiteFeature(summary.transactions) || !finiteFeature(summary.expansionBPS) || summary.expansionBPS < 0 {
		summary.invalid = true
	}
	return summary
}

func referencePercentile(values []float64, value float64) float64 {
	count := 0
	for _, candidate := range values {
		if candidate <= value {
			count++
		}
	}
	return 100 * float64(count) / float64(len(values))
}

func fullActivityOccupancy(sessionStart time.Time, records map[int64]AggregateValues, at, now time.Time) (references, mutable int) {
	blocks := make(map[int64]struct{})
	for key := range records {
		window := time.Unix(key, 0).UTC()
		offset := window.Sub(sessionStart)
		end := sessionStart.Add((offset/activityBlockDuration + 1) * activityBlockDuration)
		blocks[end.Unix()] = struct{}{}
	}
	floor := at.Add(-60 * time.Minute)
	if floor.Before(sessionStart) {
		floor = sessionStart
	}
	upper := at.Add(-activityBlockDuration)
	for end := range blocks {
		blockEnd := time.Unix(end, 0).UTC()
		if now.After(blockEnd.Add(correctionHorizon)) {
			blockStart := blockEnd.Add(-activityBlockDuration)
			if !blockStart.Before(floor) && !blockEnd.After(upper) {
				references++
			}
		} else {
			mutable++
		}
	}
	return references, mutable
}

func sameFloat(a, b float64) bool {
	return a == b || (math.IsNaN(a) && math.IsNaN(b))
}

func closeFloat(a, b float64) bool { return math.Abs(a-b) <= 1e-9 }
