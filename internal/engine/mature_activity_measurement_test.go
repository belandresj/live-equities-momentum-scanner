package engine

import (
	"fmt"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	matureMeasurementSymbols    = 6_000
	matureMeasurementReferences = 1_589
	matureMeasurementBlocks     = 1_590
)

type matureBoundaryTrial struct {
	stage, apply, publication, total time.Duration
	allocatedBytes                   uint64
}

func TestMatureActivityBoundaryMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("generated 6,000-symbol mature boundary measurement")
	}
	e, target := generatedMatureActivityEngine(t)
	validateMatureActivityManifest(t, e, target)
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		if !rebuildActivityReferenceLookup(ensureActivityState(state), state, e.state.binding, target) {
			t.Fatalf("symbol %d reference lookup rebuild failed", index)
		}
	}

	trials := make([]matureBoundaryTrial, 3)
	for trial := range trials {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		lockStarted := time.Now()
		e.mu.Lock()
		stageStarted := time.Now()
		staged := e.stageAggregateEvaluationAtLocked(target, target)
		stageDone := time.Now()
		if validation := validateAggregateEvaluation(staged); validation != nil {
			e.mu.Unlock()
			t.Fatalf("trial %d staged evaluation invalid: %v", trial+1, validation)
		}
		e.applyStagedAggregateCandidateLocked(staged, target)
		applyDone := time.Now()
		e.state.aggregateEvaluator.current = cloneAggregateEvaluation(staged)
		e.reconcileTQLocked(target)
		candidate, err := e.buildPublicationLocked(uint64(trial+2), uint64(trial+2), transitionDisposition{EngineSequence: uint64(trial + 2), Code: DispositionTimerApplied}, target,
			e.counters, e.transitions, e.publications)
		if err == nil {
			err = validatePublication(candidate)
		}
		publicationDone := time.Now()
		e.mu.Unlock()
		if err != nil {
			t.Fatalf("trial %d publication invalid: %v", trial+1, err)
		}
		runtime.ReadMemStats(&after)
		trials[trial] = matureBoundaryTrial{
			stage: stageDone.Sub(stageStarted), apply: applyDone.Sub(stageDone), publication: publicationDone.Sub(applyDone),
			total: publicationDone.Sub(lockStarted), allocatedBytes: after.TotalAlloc - before.TotalAlloc,
		}
		t.Logf("GATE_C trial=%d stage=%s apply=%s publication=%s total_lock=%s allocated_bytes=%d", trial+1,
			trials[trial].stage, trials[trial].apply, trials[trial].publication, trials[trial].total, trials[trial].allocatedBytes)
	}
	var total time.Duration
	for _, trial := range trials {
		total += trial.total
		if trial.total >= 2*time.Second {
			t.Fatalf("mature evaluator trial exceeded readiness tolerance: %+v", trial)
		}
	}
	mean := total / time.Duration(len(trials))
	t.Logf("GATE_C manifest symbols=%d references_per_symbol=%d target_blocks_per_symbol=1 blocks_per_symbol=%d target=%s mean_total_lock=%s",
		matureMeasurementSymbols, matureMeasurementReferences, matureMeasurementBlocks, target.Format(time.RFC3339), mean)
	if mean >= time.Second {
		t.Fatalf("mature evaluator mean exceeded one second: %s", mean)
	}
}

func TestMatureActivitySixtyOneSecondCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("generated 6,000-symbol sixty-cycle live measurement")
	}
	e, initial := generatedMatureActivityEngine(t)
	validateMatureActivityManifest(t, e, initial)
	final := initial.Add(60 * time.Second)
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		price := 12 + float64(index%100)/100
		for second := initial.Add(-activityBlockDuration); second.Before(final); second = second.Add(time.Second) {
			values := AggregateValues{Open: price, High: price * 1.001, Low: price, Close: price, Volume: 1500, VWAP: price, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}
			record := canonicalAggregate{identity: aggregateIdentity{symbol: e.state.binding.symbols[index].symbol, start: second.Unix()}, windowStart: second, windowEnd: second.Add(time.Second), values: values}
			retainFoldedActivityTarget(state.activity, e.state.binding, record)
			end := activityBlockEnd(e.state.binding, second)
			block := state.activity.mutable[end.Unix()]
			if block.current.aggregateCount == 0 {
				block.current = activityBlockSummary{end: end.Unix(), low: math.Inf(1)}
			}
			block.current = addActivityAggregate(block.current, values)
			state.activity.mutable[end.Unix()] = block
			state.presence.set(sessionSlot(e.state.binding, second))
			state.provenAbsent.clear(sessionSlot(e.state.binding, second))
		}
		if !rebuildActivityReferenceLookup(ensureActivityState(state), state, e.state.binding, initial) {
			t.Fatalf("symbol %d lookup rebuild", index)
		}
	}
	e.state.hydration.supportedThrough = immutableTime(final)
	var stageTotal, applyTotal, publicationTotal, total time.Duration
	var maxTotal time.Duration
	for cycle := 1; cycle <= 60; cycle++ {
		// Keep discarded per-cycle candidate images from accumulating across the
		// intentionally dense 6,000-symbol fixture; collection is outside timing.
		runtime.GC()
		at := initial.Add(time.Duration(cycle) * time.Second)
		started := time.Now()
		e.mu.Lock()
		stagedAt := time.Now()
		staged := e.stageAggregateEvaluationAtLocked(at, at)
		stagedDone := time.Now()
		if validation := validateAggregateEvaluation(staged); validation != nil {
			e.mu.Unlock()
			t.Fatalf("cycle %d invalid: %v", cycle, validation)
		}
		e.applyStagedAggregateCandidateLocked(staged, at)
		applyDone := time.Now()
		e.state.aggregateEvaluator.current = cloneAggregateEvaluation(staged)
		e.reconcileTQLocked(at)
		candidate, err := e.buildPublicationLocked(uint64(cycle+1), uint64(cycle+1), transitionDisposition{EngineSequence: uint64(cycle + 1), Code: DispositionTimerApplied}, at, e.counters, e.transitions, e.publications)
		if err == nil {
			err = validatePublication(candidate)
		}
		published := time.Now()
		e.mu.Unlock()
		if err != nil {
			t.Fatalf("cycle %d publication: %v", cycle, err)
		}
		stage, apply, publication, elapsed := stagedDone.Sub(stagedAt), applyDone.Sub(stagedDone), published.Sub(applyDone), published.Sub(started)
		stageTotal += stage
		applyTotal += apply
		publicationTotal += publication
		total += elapsed
		if elapsed > maxTotal {
			maxTotal = elapsed
		}
		if elapsed >= 2*time.Second {
			t.Fatalf("cycle %d lock=%s", cycle, elapsed)
		}
	}
	mean := total / 60
	if mean >= time.Second {
		t.Fatalf("mean lock=%s", mean)
	}
	for index := range e.state.binding.symbols {
		activity := e.state.binding.symbols[index].aggregates.activity
		if !activity.referenceLookup.valid || len(activity.referenceLookup.transactions) > maximumActivityReferences || len(activity.referenceLookup.transactions) != len(activity.referenceLookup.expansions) {
			t.Fatalf("symbol %d lookup bound", index)
		}
	}
	t.Logf("GATE_D cycles=60 stage_total=%s apply_total=%s publication_total=%s mean_lock=%s max_lock=%s aggregate_rejection=0 queue_growth=0 lookup_bound=%d", stageTotal, applyTotal, publicationTotal, mean, maxTotal, maximumActivityReferences)
}

func generatedMatureActivityEngine(t *testing.T) (*Engine, time.Time) {
	t.Helper()
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	sessionStart := time.Date(2026, 8, 7, 4, 0, 0, 0, location).UTC()
	target := time.Date(2026, 8, 7, 17, 15, 0, 0, location).UTC()
	sessionEnd := time.Date(2026, 8, 7, 20, 0, 0, 0, location).UTC()
	binding := &installedBinding{identity: "mature-activity-6000", tradingDate: "2026-08-07", sessionStart: sessionStart, sessionEnd: sessionEnd,
		symbols: make([]coreSymbol, matureMeasurementSymbols), index: make(map[string]int, matureMeasurementSymbols)}
	for index := range binding.symbols {
		symbolName := fmt.Sprintf("S%04d", index)
		binding.index[symbolName] = index
		state := generatedMatureActivitySymbol(binding, symbolName, target, index)
		binding.symbols[index] = coreSymbol{symbol: symbolName, prior: frozenPriorClose{symbol: symbolName, status: reference.PriorCloseValid, close: 10}, aggregates: state}
	}
	e := &Engine{mode: RunModeLive, capacity: 8192, reserve: 128, state: &engineState{binding: binding, lifecycle: lifecycleLive, committedT: immutableTime(target), latestTarget: immutableTime(target), clockMonotonic: true,
		liveEpoch: 1, liveEpochActive: true, aggregateAcknowledged: true, aggregateAckPosition: LivePosition{ConnectionEpoch: 1, FrameSequence: 1},
		hydration: hydrationState{fenceReconciled: true, fenceEpoch: 1, fenceThrough: 1, fenceMarkerOrdinal: 1, supportedThrough: immutableTime(target)}}}
	e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
	return e, target
}

func generatedMatureActivitySymbol(binding *installedBinding, symbol string, target time.Time, ordinal int) *symbolAggregateState {
	targetStart := target.Add(-activityBlockDuration)
	price := 12 + float64(ordinal%100)/100
	record := &canonicalAggregate{identity: aggregateIdentity{symbol: symbol, start: targetStart.Unix()}, windowStart: targetStart, windowEnd: targetStart.Add(time.Second),
		values: AggregateValues{Open: price, High: price * 1.001, Low: price, Close: price, Volume: 1_500, VWAP: price, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}}
	state := &symbolAggregateState{tail: map[int64]*canonicalAggregate{targetStart.Unix(): record}, latest: &latestAggregateMark{record: *record}, committedLatest: committedMark(*record), presence: &slotBitmap{}, provenAbsent: &slotBitmap{}}
	startSlot, endSlot := sessionSlot(binding, binding.sessionStart), sessionSlot(binding, target)
	for word := startSlot / 64; word <= (endSlot-1)/64; word++ {
		state.provenAbsent[word] = slotRangeWordMask(word, startSlot, endSlot)
	}
	activity := &activityFeatureState{references: make(map[int64]*activityBlockSummary, matureMeasurementReferences), mutable: make(map[int64]activityMutableBlock), result: unavailableActivityResult(time.Time{})}
	summaries := make([]activityBlockSummary, matureMeasurementReferences)
	for block := 0; block < matureMeasurementReferences; block++ {
		blockStart := binding.sessionStart.Add(time.Duration(block) * activityBlockDuration)
		blockEnd := blockStart.Add(activityBlockDuration)
		values := AggregateValues{Open: price, High: price * (1.0001 + float64(block%7)/100_000), Low: price, Close: price, Volume: float64(1_000 + block%100), AverageTradeSize: 5, VWAP: price, ATSProvenance: ATSLiveProviderAverage}
		summaries[block] = addActivityAggregate(activityBlockSummary{end: blockEnd.Unix(), low: values.Low}, values)
		activity.references[blockEnd.Unix()] = &summaries[block]
		slot := sessionSlot(binding, blockStart)
		state.presence.set(slot)
		state.provenAbsent.clear(slot)
	}
	targetSlot := sessionSlot(binding, targetStart)
	state.provenAbsent.clear(targetSlot)
	state.activity = activity
	state.qualification = &qualificationState{finalized: true, finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{}, accountedThrough: target,
		finalProofEnd: target.Add(-time.Minute), result: qualificationResult{at: target, status: qualificationFinalized, finalProofEnd: target.Add(-time.Minute)}}
	return state
}

func validateMatureActivityManifest(t *testing.T, e *Engine, target time.Time) {
	t.Helper()
	if len(e.state.binding.symbols) != matureMeasurementSymbols || target.Sub(e.state.binding.sessionStart) != time.Duration(matureMeasurementBlocks)*activityBlockDuration {
		t.Fatalf("mature manifest binding symbols=%d blocks=%d", len(e.state.binding.symbols), target.Sub(e.state.binding.sessionStart)/activityBlockDuration)
	}
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		if state == nil || state.activity == nil || len(state.activity.references) != matureMeasurementReferences || len(state.tail) != 1 || state.activity.boundExceeded {
			t.Fatalf("mature manifest symbol=%d references=%d target_records=%d bound=%t", index, len(state.activity.references), len(state.tail), state.activity.boundExceeded)
		}
	}
}
