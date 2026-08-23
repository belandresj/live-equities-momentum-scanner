package engine

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	liveRetainedTailPopulation           = 5_694
	liveRetainedTailRecordsPerSymbol     = 961
	liveRetainedTailActivityWarming      = 1_478
	liveRetainedTailActivityCurrent      = 417
	liveRetainedTailActivityNoTarget     = 2_370
	liveRetainedTailActivityIncomplete   = 920
	liveRetainedTailActivityBound        = 509
	liveRetainedTailTransitionTrusted    = 190
	liveRetainedTailTransitionIncomplete = 647
)

type liveRetainedTailTrial struct {
	stage, apply, total time.Duration
	allocatedBytes      uint64
}

// TestLiveRetainedTailCostAttribution is a deterministic, credential-free
// diagnostic for the state shape absent from the accepted cached-hydration
// fixture. It compares the same market/evaluator result with old records
// represented by folded presence versus 961 canonical tail entries per symbol.
// The 509 terminal-invalid Activity states deliberately retain invalid lookup
// acceleration so apply measures the production rebuild path independently of
// stage. This test makes no acceptance claim and changes no production state.
func TestLiveRetainedTailCostAttribution(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in full-population retained-tail cost diagnostic")
	}

	e, target := generatedLiveRetainedTailEngine(t)
	validateLiveRetainedTailManifest(t, e, target, false)
	folded, foldedEvaluation := measureLiveRetainedTailTrial(t, e, target, "folded")

	uniqueRecords := os.Getenv("LIVE_RETAINED_TAIL_UNIQUE_RECORDS") == "1"
	installLiveRetainedTailShape(t, e, target, uniqueRecords)
	validateLiveRetainedTailManifest(t, e, target, true)
	retained, retainedEvaluation := measureLiveRetainedTailTrial(t, e, target, "retained")

	if !reflect.DeepEqual(foldedEvaluation, retainedEvaluation) {
		t.Fatalf("folded and retained-tail evaluations differ: folded population=%+v transitions=%+v retained population=%+v transitions=%+v",
			foldedEvaluation.population, foldedEvaluation.populationTransition,
			retainedEvaluation.population, retainedEvaluation.populationTransition)
	}

	stageRatio := durationRatio(retained.stage, folded.stage)
	applyRatio := durationRatio(retained.apply, folded.apply)
	t.Logf("RETAINED_TAIL phase=folded stage=%s apply=%s total=%s allocated_bytes=%d",
		folded.stage, folded.apply, folded.total, folded.allocatedBytes)
	t.Logf("RETAINED_TAIL phase=retained stage=%s apply=%s total=%s allocated_bytes=%d stage_ratio=%.2f apply_ratio=%.2f",
		retained.stage, retained.apply, retained.total, retained.allocatedBytes, stageRatio, applyRatio)
	t.Logf("RETAINED_TAIL manifest symbols=%d tail_records_per_symbol=%d tail_record_visits_per_full_scan=%d unique_records=%t activity_warming=%d activity_current=%d activity_unavailable=%d activity_bound_invalid=%d bootstrap_unknown=%d trusted_later_live=%d incomplete_post_mark=%d",
		liveRetainedTailPopulation, liveRetainedTailRecordsPerSymbol,
		liveRetainedTailPopulation*liveRetainedTailRecordsPerSymbol,
		uniqueRecords,
		liveRetainedTailActivityWarming, liveRetainedTailActivityCurrent,
		liveRetainedTailActivityNoTarget+liveRetainedTailActivityIncomplete,
		liveRetainedTailActivityBound,
		liveRetainedTailTransitionTrusted+liveRetainedTailTransitionIncomplete,
		liveRetainedTailTransitionTrusted, liveRetainedTailTransitionIncomplete)
	if stageRatio >= 5 && applyRatio >= 5 {
		t.Log("RETAINED_TAIL decision=matched_state_cost_dominates_both_stage_and_apply")
	} else {
		t.Log("RETAINED_TAIL decision=retained_tail_not_yet_established_as_joint_dominant_cost")
	}
}

func TestLiveRetainedTailSixtyOneSecondCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in full-population retained-tail sixty-cycle proof")
	}
	e, initial := generatedLiveRetainedTailEngine(t)
	installLiveRetainedTailShape(t, e, initial, true)
	final := initial.Add(60 * time.Second)
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		if !installExactCoverage(state, e.state.binding, initial, final, nil) {
			t.Fatalf("symbol=%d future coverage", index)
		}
	}
	runtime.GC()
	var stageTotal, applyTotal, total time.Duration
	var maximum time.Duration
	for cycle := 1; cycle <= 60; cycle++ {
		// Let the runtime pace collection as it does in the scanner. Forcing a
		// full collection immediately before every one-second cycle leaves the
		// scavenger competing with the evaluator and is not a production cadence.
		at := initial.Add(time.Duration(cycle) * time.Second)
		started := time.Now()
		e.mu.Lock()
		stageStarted := time.Now()
		staged := e.stageAggregateEvaluationAtLocked(at, at)
		stageDone := time.Now()
		if validation := validateAggregateEvaluation(staged); validation != nil {
			e.mu.Unlock()
			t.Fatalf("cycle=%d evaluation=%v", cycle, validation)
		}
		e.applyStagedAggregateCandidateLocked(staged, at)
		done := time.Now()
		e.mu.Unlock()
		stage, apply, elapsed := stageDone.Sub(stageStarted), done.Sub(stageDone), done.Sub(started)
		stageTotal += stage
		applyTotal += apply
		total += elapsed
		maximum = max(maximum, elapsed)
		if elapsed > 500*time.Millisecond {
			t.Logf("RETAINED_TAIL_CYCLE_OUTLIER cycle=%d total=%s stage=%s apply=%s", cycle, elapsed, stage, apply)
		}
	}
	mean := total / 60
	t.Logf("RETAINED_TAIL_CYCLES cycles=60 stage_total=%s apply_total=%s mean=%s maximum=%s", stageTotal, applyTotal, mean, maximum)
	if mean > 500*time.Millisecond || maximum > time.Second {
		t.Fatalf("cycle latency mean=%s maximum=%s want mean<=500ms maximum<=1s", mean, maximum)
	}
}

func measureLiveRetainedTailTrial(t *testing.T, e *Engine, target time.Time, phase string) (liveRetainedTailTrial, aggregateEvaluationResult) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	started := time.Now()
	e.mu.Lock()
	stageStarted := time.Now()
	var staged aggregateEvaluationResult
	pprof.Do(context.Background(), pprof.Labels("phase", phase+"_stage"), func(context.Context) {
		staged = e.stageAggregateEvaluationAtLocked(target, target)
	})
	stageDone := time.Now()
	if validation := validateAggregateEvaluation(staged); validation != nil {
		e.mu.Unlock()
		t.Fatalf("staged evaluation invalid: %v", validation)
	}
	pprof.Do(context.Background(), pprof.Labels("phase", phase+"_apply"), func(context.Context) {
		e.applyStagedAggregateCandidateLocked(staged, target)
	})
	applyDone := time.Now()
	e.mu.Unlock()
	runtime.ReadMemStats(&after)
	return liveRetainedTailTrial{
		stage: stageDone.Sub(stageStarted), apply: applyDone.Sub(stageDone),
		total: applyDone.Sub(started), allocatedBytes: after.TotalAlloc - before.TotalAlloc,
	}, staged
}

func generatedLiveRetainedTailEngine(t *testing.T) (*Engine, time.Time) {
	t.Helper()
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	sessionStart := time.Date(2026, 8, 12, 4, 0, 0, 0, location).UTC()
	target := time.Date(2026, 8, 12, 17, 15, 0, 0, location).UTC()
	sessionEnd := time.Date(2026, 8, 12, 20, 0, 0, 0, location).UTC()
	binding := &installedBinding{
		identity: "live-retained-tail-2026-08-12", tradingDate: "2026-08-12",
		sessionStart: sessionStart, sessionEnd: sessionEnd,
		symbols: make([]coreSymbol, liveRetainedTailPopulation),
		index:   make(map[string]int, liveRetainedTailPopulation),
	}
	recent, stale := liveRetainedTailRecords(target, target.Add(-time.Second)), liveRetainedTailRecords(target, target.Add(-31*time.Second))
	for index := range binding.symbols {
		symbol := fmt.Sprintf("S%04d", index)
		binding.index[symbol] = index
		records := recent
		if liveRetainedTailNoTargetIndex(index) {
			records = stale
		}
		state := foldedLiveRetainedTailSymbol(binding, symbol, target, records, index)
		binding.symbols[index] = coreSymbol{
			symbol:     symbol,
			prior:      frozenPriorClose{symbol: symbol, status: reference.PriorCloseValid, close: 10},
			aggregates: state,
		}
	}
	e := &Engine{
		mode: RunModeLive, capacity: 8192, reserve: 128,
		state: &engineState{
			binding: binding, lifecycle: lifecycleLive,
			committedT: immutableTime(target), latestTarget: immutableTime(target), clockMonotonic: true,
			liveEpoch: 1, liveEpochActive: true, aggregateAcknowledged: true,
			aggregateAckPosition: LivePosition{ConnectionEpoch: 1, FrameSequence: 1},
			hydration:            hydrationState{fenceReconciled: true, fenceEpoch: 1, fenceThrough: 1, fenceMarkerOrdinal: 1, supportedThrough: immutableTime(target)},
		},
	}
	e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence, liveRetainedTailTransitionTrusted+liveRetainedTailTransitionIncomplete)
	trustedStart := liveRetainedTailActivityWarming + liveRetainedTailActivityCurrent
	incompleteStart := trustedStart + liveRetainedTailActivityNoTarget
	for index := trustedStart; index < trustedStart+liveRetainedTailTransitionTrusted; index++ {
		e.state.aggregateEvaluator.coverage[index] = coverageUnknownFailureOrFence
	}
	for index := incompleteStart; index < incompleteStart+liveRetainedTailTransitionIncomplete; index++ {
		e.state.aggregateEvaluator.coverage[index] = coverageUnknownFailureOrFence
	}
	return e, target
}

func liveRetainedTailRecords(target, last time.Time) []*canonicalAggregate {
	// These are immutable construction templates. The default low-memory control
	// shares them and therefore biases pointee reads downward; the explicitly
	// selected unique-record mode copies them into one symbol-owned contiguous
	// slice before timing the incident-shaped working set.
	records := make([]*canonicalAggregate, liveRetainedTailRecordsPerSymbol)
	position := LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	for index := range records {
		start := last.Add(-time.Duration(len(records)-1-index) * time.Second)
		records[index] = &canonicalAggregate{
			identity:    aggregateIdentity{symbol: "SHARED", start: start.Unix()},
			windowStart: start, windowEnd: start.Add(time.Second),
			values:              AggregateValues{Open: 10, High: 10.1, Low: 9.9, Close: 10, Volume: 1_500, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage},
			authority:           aggregateEvidence{source: AggregateSourceLive, live: position},
			greatestLiveSupport: &position,
		}
	}
	return records
}

func foldedLiveRetainedTailSymbol(binding *installedBinding, symbol string, target time.Time, records []*canonicalAggregate, index int) *symbolAggregateState {
	latest := records[len(records)-1]
	state := &symbolAggregateState{
		tail:   map[int64]*canonicalAggregate{latest.identity.start: latest},
		latest: &latestAggregateMark{record: *latest}, committedLatest: committedMark(*latest),
		presence: &slotBitmap{}, provenAbsent: &slotBitmap{},
		priceRange: &priceRangeFeatureState{
			firstStart: binding.sessionStart.Unix(), firstOpen: 10, hasFirst: true,
			sessionHigh: 10.1, sessionLow: 9.9, hasSessionExtrema: true, finalizedThrough: target.Unix(),
			sessionHighs: []extremaPoint{{windowStart: target.Add(-60 * time.Minute).Unix(), value: 10.1}},
			sessionLows:  []extremaPoint{{windowStart: target.Add(-60 * time.Minute).Unix(), value: 9.9}},
			result:       unavailablePriceRangeResult(time.Time{}),
		},
		qualification: &qualificationState{
			finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{},
			accountedThrough: target, finalized: true, finalProofEnd: target.Add(-time.Minute),
			result: qualificationResult{at: target, status: qualificationFinalized, finalProofEnd: target.Add(-time.Minute)},
		},
	}
	installExactCoverage(state, binding, binding.sessionStart, target, nil)
	for _, record := range records {
		state.provenAbsent.clear(sessionSlot(binding, record.windowStart))
		state.presence.set(sessionSlot(binding, record.windowStart))
	}

	activity := &activityFeatureState{result: unavailableActivityResult(time.Time{})}
	switch {
	case index < liveRetainedTailActivityWarming:
		activity.referenceLookup = liveRetainedTailLookup(target, 5)
		installLiveRetainedTailFoldedTarget(activity, binding, records, target)
	case index < liveRetainedTailActivityWarming+liveRetainedTailActivityCurrent:
		activity.referenceLookup = liveRetainedTailLookup(target, minimumActivityReferences)
		installLiveRetainedTailFoldedTarget(activity, binding, records, target)
	case index < liveRetainedTailActivityWarming+liveRetainedTailActivityCurrent+liveRetainedTailActivityNoTarget:
		activity.referenceLookup = liveRetainedTailLookup(target, minimumActivityReferences)
	case index < liveRetainedTailPopulation-liveRetainedTailActivityBound:
		activity.referenceLookup = liveRetainedTailLookup(target, minimumActivityReferences)
		gap := target.Add(-2 * time.Second)
		state.presence.clear(sessionSlot(binding, gap))
		state.provenAbsent.clear(sessionSlot(binding, gap))
	default:
		activity.boundExceeded = true
	}
	state.activity = activity

	trustedStart := liveRetainedTailActivityWarming + liveRetainedTailActivityCurrent
	incompleteStart := trustedStart + liveRetainedTailActivityNoTarget
	if index >= trustedStart && index < trustedStart+liveRetainedTailTransitionTrusted ||
		index >= incompleteStart && index < incompleteStart+liveRetainedTailTransitionIncomplete {
		ensureHistoricalConflict(state).set(sessionSlot(binding, binding.sessionStart))
	}
	return state
}

func liveRetainedTailLookup(at time.Time, references int) activityReferenceLookup {
	return activityReferenceLookup{
		at: at, valid: true,
		transactions: make([]float64, references), expansions: make([]float64, references),
	}
}

func installLiveRetainedTailFoldedTarget(activity *activityFeatureState, binding *installedBinding, records []*canonicalAggregate, target time.Time) {
	for _, record := range records {
		if record.windowStart.Before(target.Add(-activityBlockDuration)) {
			continue
		}
		retainFoldedActivityTarget(activity, binding, *record)
	}
}

func installLiveRetainedTailShape(t *testing.T, e *Engine, target time.Time, uniqueRecords bool) {
	t.Helper()
	recent, stale := liveRetainedTailRecords(target, target.Add(-time.Second)), liveRetainedTailRecords(target, target.Add(-31*time.Second))
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		records := recent
		if liveRetainedTailNoTargetIndex(index) {
			records = stale
		}
		state.tail = make(map[int64]*canonicalAggregate, len(records))
		// Replace the folded control's mutable-tail index rather than appending to
		// it. Production mutation maintains exactly one extrema point per
		// canonical tail identity; a mismatched length deliberately selects the
		// containment fallback and is not a valid production-shape capacity proof.
		features := ensurePriceRangeState(state)
		features.highs = nil
		features.lows = nil
		measurements := ensureMVPMeasurementState(state)
		measurements.tail = nil
		var owned []canonicalAggregate
		if uniqueRecords {
			owned = make([]canonicalAggregate, len(records))
		}
		var latest *canonicalAggregate
		for recordIndex, template := range records {
			record := template
			if uniqueRecords {
				owned[recordIndex] = *template
				owned[recordIndex].identity.symbol = e.state.binding.symbols[index].symbol
				record = &owned[recordIndex]
			}
			state.tail[record.identity.start] = record
			retainMutablePriceRangeEvidence(features, *record)
			retainMutableMVPMeasurement(state, *record)
			state.presence.clear(sessionSlot(e.state.binding, record.windowStart))
			latest = record
		}
		state.latest = &latestAggregateMark{record: *latest}
		state.committedLatest = committedMark(*latest)
		if state.activity != nil {
			state.activity.foldedTargets = nil
			state.activity.foldedTargetContributions = 0
			if state.activity.boundExceeded {
				// The folded control's apply attempted the same rebuild and may
				// have left a valid derived lookup despite terminal invalidity.
				// Restore the incident-shaped invalid acceleration before timing
				// the retained-tail apply phase.
				state.activity.referenceLookup = activityReferenceLookup{}
			}
		}
		rebuildTailCoverage(state, e.state.binding)
	}
}

func liveRetainedTailNoTargetIndex(index int) bool {
	return index >= liveRetainedTailActivityWarming+liveRetainedTailActivityCurrent &&
		index < liveRetainedTailPopulation-liveRetainedTailActivityBound
}

func validateLiveRetainedTailManifest(t *testing.T, e *Engine, target time.Time, retained bool) {
	t.Helper()
	if len(e.state.binding.symbols) != liveRetainedTailPopulation ||
		liveRetainedTailActivityWarming+liveRetainedTailActivityCurrent+liveRetainedTailActivityNoTarget+
			liveRetainedTailActivityIncomplete+liveRetainedTailActivityBound != liveRetainedTailPopulation {
		t.Fatalf("invalid population manifest")
	}
	counts := struct{ warming, current, noTarget, incomplete, bound, tail, trusted, transitionIncomplete int }{}
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		counts.tail += len(state.tail)
		result := evaluateActivityFeatures(e.state.binding, state, target)
		switch {
		case result.activity.status == featureWarming && result.activity.reason == featureReasonReferenceWarmup:
			counts.warming++
		case result.activity.status == featureCurrent:
			counts.current++
		case result.activity.status == featureUnavailable && result.activity.reason == featureReasonNoAggregateInTarget:
			counts.noTarget++
		case result.activity.status == featureUnavailable && result.activity.reason == featureReasonHistoryIncomplete:
			counts.incomplete++
		case result.activity.status == featureInvalid && result.activity.reason == featureReasonStateBoundExceeded:
			counts.bound++
		default:
			t.Fatalf("symbol=%d unexpected Activity result=%+v", index, result)
		}
		if coverage, ok := e.state.aggregateEvaluator.coverage[index]; ok {
			mark, hasMark := latestMarkBefore(state, target)
			switch classifyPopulationTransition(e.state.binding, state, mark, hasMark, target, coverage, nil) {
			case populationTransitionTrustedByLaterLiveMark:
				counts.trusted++
			case populationTransitionIncompletePostMarkCoverage:
				counts.transitionIncomplete++
			default:
				t.Fatalf("symbol=%d unexpected population transition", index)
			}
		}
	}
	wantTail := liveRetainedTailPopulation
	if retained {
		wantTail *= liveRetainedTailRecordsPerSymbol
	}
	if counts.warming != liveRetainedTailActivityWarming || counts.current != liveRetainedTailActivityCurrent ||
		counts.noTarget != liveRetainedTailActivityNoTarget || counts.incomplete != liveRetainedTailActivityIncomplete ||
		counts.bound != liveRetainedTailActivityBound || counts.tail != wantTail ||
		counts.trusted != liveRetainedTailTransitionTrusted || counts.transitionIncomplete != liveRetainedTailTransitionIncomplete {
		t.Fatalf("manifest counts=%+v want_tail=%d", counts, wantTail)
	}
}

func durationRatio(numerator, denominator time.Duration) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
