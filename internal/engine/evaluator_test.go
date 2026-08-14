package engine

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestC3POP01CompletePartitionTransitionTable is the sole C3-POP-01 proof.
// It distinguishes incomplete bootstrap, attributable invalidity, reconciled
// no-print, failure precedence, below-price, first print, and earlier-print plus
// later-empty while checking both population identities after every state.
func TestC3POP01CompletePartitionTransitionTable(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 0, qualificationUnresolved}, {"BAD", reference.PriorCloseInvalid, 0, 0, qualificationUnresolved}, {"MISSING", reference.PriorCloseMissing, 0, 0, qualificationUnresolved}})
	assertPopulation := func(name string, want populationAccounting) {
		t.Helper()
		got := e.stageAggregateEvaluationLocked(at).population
		if got != want || !got.reconciles() {
			t.Fatalf("%s population = %+v, want %+v", name, got, want)
		}
	}
	base := populationAccounting{universeTotal: 3, validPriorClose: 1, invalidOrMissingPriorClose: 2, unknownDueFailureOrFence: 1, coveredPopulation: 2, unresolvedPopulation: 1}
	assertPopulation("incomplete bootstrap", base)
	e.state.aggregateEvaluator.invalidMarks = map[int]invalidMarkEvidence{0: {windowStart: at.Add(-time.Second)}}
	delete(e.state.aggregateEvaluator.coverage, 0)
	want := base
	want.unknownDueFailureOrFence, want.invalidMark, want.coveredPopulation, want.unresolvedPopulation = 0, 1, 3, 0
	assertPopulation("attributable invalid", want)
	e.state.aggregateEvaluator.coverage = map[int]aggregateCoverageConsequence{0: coverageUnknownFailureOrFence}
	assertPopulation("failure overrides invalid", base)
	e.state.aggregateEvaluator.coverage[0] = coverageNoPrintThroughT
	delete(e.state.aggregateEvaluator.invalidMarks, 0)
	want.invalidMark, want.noPrintThroughT = 0, 1
	assertPopulation("reconciled no print", want)
	delete(e.state.aggregateEvaluator.coverage, 0) // a later empty subinterval is not session no-print proof
	installEvaluatorMark(e, 0, at, .249, qualificationNotYetPassed)
	want.noPrintThroughT, want.trustedBelowPriceMark = 0, 1
	assertPopulation("first below-price print", want)
	e.state.binding.symbols[0].aggregates.tail[at.Add(-time.Second).Unix()].values.Close = 12
	want.trustedBelowPriceMark, want.trustedRankableMark = 0, 1
	assertPopulation("earlier mark survives later empty", want)
}

// TestC3POP02AccountingIntegrityAndOverlap is the sole C3-POP-02 proof.
// Feature/qualification dimensions change without rebalancing primary bins;
// an injected fixed accounting contradiction reaches the existing global
// accounting-integrity sentinel rather than a current publication.
func TestC3POP02AccountingIntegrityAndOverlap(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional}})
	first := e.stageAggregateEvaluationLocked(at)
	qualification := e.state.binding.symbols[0].aggregates.qualification
	qualification.finalized, qualification.finalProofEnd = true, at.Add(-time.Minute)
	clear(qualification.proofs)
	qualification.result = qualificationResult{at: at, status: qualificationFinalized, finalProofEnd: qualification.finalProofEnd}
	second := e.stageAggregateEvaluationLocked(at)
	if first.population != second.population || first.qualification.provisional != 1 || second.qualification.finalized != 1 || second.features.activity.statuses[0] != 1 {
		t.Fatalf("overlap changed primary accounting: first=%+v second=%+v", first, second)
	}
	bad := second
	bad.population.universeTotal++
	if validateAggregateEvaluation(bad) == nil {
		t.Fatal("counter mismatch validated")
	}
	hole := second
	hole.rows = nil
	if validateAggregateEvaluation(hole) == nil {
		t.Fatal("qualified cardinality hole validated")
	}
	overflow := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, math.SmallestNonzeroFloat64, math.MaxFloat64, qualificationProvisional}}).stageAggregateEvaluationLocked(at)
	if overflow.population.trustedRankableMark != 1 || overflow.population.invalidMark != 0 || overflow.dayInvalidRankable != 1 || overflow.features.dayPercent.statuses[3] != 1 || validateAggregateEvaluation(overflow) != nil {
		t.Fatalf("invalid Day field rebalanced primary population: %+v", overflow)
	}

	binding := testBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	runtime := aggregateEngine(t, binding, RunModeLive, &now)
	runtime.mu.Lock()
	runtime.state.committedT = immutableTime(now)
	runtime.state.lifecycle = lifecycleLive
	runtime.evaluationFault = true
	runtime.mu.Unlock()
	admission, completion := runtime.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionAccountingIntegrity {
		t.Fatal("accounting contradiction did not terminate the transition")
	}
	view := runtime.observePublication()
	if view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonAccountingIntegrity || view.currentMarketClaim {
		t.Fatalf("accounting mismatch escaped containment: %+v", view)
	}
	closeAndWait(t, runtime)
}

func TestCompleteReplayFutureFirstPrintPreservesNoPrintAtDelayedTarget(t *testing.T) {
	at := time.Date(2026, 8, 7, 13, 29, 56, 0, time.UTC)
	e := evaluatorProofEngine(at, []evaluatorSymbol{{"LATE", reference.PriorCloseValid, 10, 0, qualificationUnresolved}})
	e.mode = RunModeReplay
	e.state.lifecycle = lifecycleReplaying
	e.state.replay = replayState{complete: true, observationStart: at.Add(4 * time.Second), lastGroup: at.Add(4 * time.Second)}
	delete(e.state.aggregateEvaluator.coverage, 0)

	symbol := &e.state.binding.symbols[0]
	futureStart := at.Add(2 * time.Second)
	future := canonicalAggregate{
		identity:    aggregateIdentity{symbol: symbol.symbol, start: futureStart.Unix()},
		windowStart: futureStart,
		windowEnd:   futureStart.Add(time.Second),
		values:      AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 1, VWAP: 12, AverageTradeSize: 1, ATSProvenance: ATSRESTFloorVolumeOverTrades},
	}
	state := &symbolAggregateState{tail: map[int64]*canonicalAggregate{futureStart.Unix(): &future}, latest: &latestAggregateMark{record: future}}
	state.provenAbsent = &slotBitmap{}
	for slot := 0; slot < sessionSlot(e.state.binding, at); slot++ {
		state.provenAbsent.set(slot)
	}
	symbol.aggregates = state

	result := e.stageAggregateEvaluationAtLocked(at, at.Add(4*time.Second))
	if result.mode != rankingQualifiedCurrent || result.reason != "" || result.population.noPrintThroughT != 1 ||
		result.population.unknownDueFailureOrFence != 0 || len(result.rows) != 0 {
		t.Fatalf("delayed target lost sealed no-print evidence: %+v", result)
	}
}

// TestC3RANK01ExactFilterOrderAndTop20 is the sole C3-RANK-01 proof.
func TestC3RANK01ExactFilterOrderAndTop20(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	for _, count := range []int{0, 1, 19, 20, 21} {
		t.Run(fmt.Sprintf("%d passers", count), func(t *testing.T) {
			symbols := make([]evaluatorSymbol, 0, count+1)
			for i := 0; i < count; i++ {
				symbols = append(symbols, evaluatorSymbol{fmt.Sprintf("P%02d", i), reference.PriorCloseValid, 10, 50 - float64(i), qualificationProvisional})
			}
			symbols = append(symbols, evaluatorSymbol{"ZZZ", reference.PriorCloseValid, 10, 100, qualificationNotYetPassed})
			got := evaluatorProofEngine(at, symbols).stageAggregateEvaluationLocked(at)
			wantRows := min(count, maximumRankingRows)
			if got.mode != rankingQualifiedCurrent || got.totalPassers != uint64(count) || len(got.rows) != wantRows {
				t.Fatalf("cardinality = mode %s passers %d rows %d, want qualified/%d/%d", got.mode, got.totalPassers, len(got.rows), count, wantRows)
			}
		})
	}
	symbols := make([]evaluatorSymbol, 0, 26)
	for i := 0; i < 25; i++ {
		symbols = append(symbols, evaluatorSymbol{fmt.Sprintf("S%02d", i), reference.PriorCloseValid, 10, 30 - float64(i), qualificationProvisional})
	}
	symbols = append(symbols, evaluatorSymbol{"ZZZ", reference.PriorCloseValid, 10, 100, qualificationNotYetPassed})
	// Exact unrounded equality must use symbol only.
	symbols[1].mark, symbols[2].mark = 28, 28
	e := evaluatorProofEngine(at, symbols)
	got := e.stageAggregateEvaluationLocked(at)
	if got.mode != rankingQualifiedCurrent || got.totalPassers != 25 || len(got.rows) != 20 {
		t.Fatalf("rank result = mode %s passers %d rows %d", got.mode, got.totalPassers, len(got.rows))
	}
	for _, row := range got.rows {
		if row.symbol == "ZZZ" {
			t.Fatal("high-return nonpasser displaced a passer")
		}
	}
	if got.rows[1].symbol != "S01" || got.rows[2].symbol != "S02" {
		t.Fatalf("exact tie order = %s/%s", got.rows[1].symbol, got.rows[2].symbol)
	}
	for i, row := range got.rows {
		if row.rank != uint32(i+1) {
			t.Fatalf("rank hole at %d: %+v", i, row)
		}
	}
}

// TestC3PROJ01ModesAndIndependentFields is the sole C3-PROJ-01 proof.
func TestC3PROJ01ModesAndIndependentFields(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	exact := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional}}).stageAggregateEvaluationLocked(at)
	if exact.mode != rankingQualifiedCurrent || len(exact.rows) != 1 || !exact.rows[0].tqIntentEligible || exact.rows[0].activity.status != featureWarming {
		t.Fatalf("qualified independent fields = %+v", exact)
	}
	degradedEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}, {"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved}})
	degraded := degradedEngine.stageAggregateEvaluationLocked(at)
	if degraded.mode != rankingDegradedBootstrap || degraded.population.coveredPopulation != 1 || degraded.population.unresolvedPopulation != 1 || len(degraded.rows) != 1 || degraded.rows[0].tqIntentEligible {
		t.Fatalf("degraded projection = %+v", degraded)
	}
	unavailable := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 0, qualificationUnresolved}}).stageAggregateEvaluationLocked(at)
	if unavailable.mode != rankingUnavailable || len(unavailable.rows) != 0 {
		t.Fatalf("zero-mark projection = %+v", unavailable)
	}
	exactEmpty := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationNotYetPassed}}).stageAggregateEvaluationLocked(at)
	if exactEmpty.mode != rankingQualifiedCurrent || exactEmpty.totalPassers != 0 || len(exactEmpty.rows) != 0 {
		t.Fatalf("exact empty projection = %+v", exactEmpty)
	}
	qualificationOnly := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}}).stageAggregateEvaluationLocked(at)
	if qualificationOnly.mode != rankingDegradedBootstrap || qualificationOnly.reason != rankingReasonQualificationPending {
		t.Fatalf("qualification-only degradation = %+v", qualificationOnly)
	}
	populationOnly := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationNotYetPassed}, {"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved}}).stageAggregateEvaluationLocked(at)
	if populationOnly.mode != rankingDegradedBootstrap || populationOnly.reason != rankingReasonIncompletePopulation {
		t.Fatalf("population-only degradation = %+v", populationOnly)
	}
	invalidFieldEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional}})
	invalidFieldEngine.state.binding.symbols[0].aggregates.activity = &activityFeatureState{boundExceeded: true}
	invalidField := invalidFieldEngine.stageAggregateEvaluationLocked(at)
	if invalidField.mode != rankingQualifiedCurrent || len(invalidField.rows) != 1 || invalidField.rows[0].activity.status != featureInvalid {
		t.Fatalf("invalid field changed ranking = %+v", invalidField)
	}
	suppressedEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional}})
	suppressedEngine.state.lifecycle, suppressedEngine.state.globalFailure = lifecycleSuppressed, true
	suppressed := suppressedEngine.stageAggregateEvaluationLocked(at)
	if suppressed.mode != rankingSuppressed || len(suppressed.rows) != 0 || validateAggregateEvaluation(suppressed) != nil {
		t.Fatalf("suppressed projection = %+v", suppressed)
	}
	stale := cloneAggregateEvaluation(exact)
	stale.mode, stale.rows = rankingStale, nil
	if validateAggregateEvaluation(stale) != nil {
		t.Fatalf("stale consequence rejected: %+v", stale)
	}
	wrongReason := suppressed
	wrongReason.reason = rankingReasonNoTrustedMarks
	if validateAggregateEvaluation(wrongReason) == nil {
		t.Fatal("wrong suppressed reason validated")
	}
}

// TestC3EVAL01AtomicStagingAndPathEquivalence is the sole C3-EVAL-01 proof.
func TestC3EVAL01AtomicStagingAndPathEquivalence(t *testing.T) {
	binding := testBinding(t)
	at := binding.SessionStart().Add(time.Minute)
	now := at
	e := aggregateEngine(t, binding, RunModeLive, &now)
	before := e.state.aggregateEvaluator.current
	applyQualificationTimer(t, e) // Component 2's closed run-support gate rejects the candidate.
	if !aggregateEvaluationEqual(before, e.state.aggregateEvaluator.current) || e.state.committedT != nil {
		t.Fatal("failed commit applied a shadow evaluation")
	}
	closeAndWait(t, e)

	// A failed attempt beyond an already committed T must discard its staged
	// candidate rather than reproject current rows from candidate-mutated support.
	failedAdvance := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	current := failedAdvance.stageAggregateEvaluationLocked(at)
	failedAdvance.state.aggregateEvaluator.current = cloneAggregateEvaluation(current)
	target := at.Add(time.Second)
	failedAdvance.state.latestTarget = immutableTime(target)
	node := &queueNode{kind: inputTimer, admissionTime: target, engineSequence: 2}
	candidate := failedAdvance.runAggregateFeatureContributorLocked(node, DispositionTimerApplied, ReasonNone)
	if !failedAdvance.runAggregateEvaluatorLocked(node, DispositionTimerApplied, ReasonNone, candidate) ||
		!aggregateEvaluationEqual(failedAdvance.state.aggregateEvaluator.current, current) || !failedAdvance.state.committedT.Equal(at) {
		t.Fatal("failed advance replaced the current evaluation")
	}

	// Seed only Component 2's still-unimplemented run-support fact in package
	// state, then drive the ordinary timer/contributor/publication path.
	runtimeNow := at
	runtime := aggregateEngine(t, binding, RunModeLive, &runtimeNow)
	applyAggregate(t, runtime, liveAggregate(binding, "AAA", at.Add(-time.Second), 1, 1), DispositionAggregateInserted, ReasonNone)
	runtime.mu.Lock()
	runtime.state.committedT = immutableTime(at)
	runtime.state.lifecycle = lifecycleLive
	qualification := ensureQualificationState(runtime.state.binding.symbols[runtime.state.binding.index["AAA"]].aggregates)
	qualification.finalized = true
	qualification.finalProofEnd = at.Add(-time.Minute)
	qualification.result = qualificationResult{at: at, status: qualificationFinalized, finalProofEnd: qualification.finalProofEnd}
	runtime.mu.Unlock()
	applyQualificationTimer(t, runtime)
	published := runtime.observePublication()
	if !published.currentMarketClaim || published.aggregateEvaluation.mode != rankingQualifiedCurrent || len(published.aggregateEvaluation.rows) != 1 || published.aggregateEvaluation.rows[0].symbol != "AAA" {
		t.Fatalf("atomic private publication = %+v", published.aggregateEvaluation)
	}
	closeAndWait(t, runtime)

	a := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	b := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	a.state.binding.symbols[0].aggregates.latest.record.authority.source = AggregateSourceLive
	b.state.binding.symbols[0].aggregates.latest.record.authority.source = AggregateSourceReplay
	first, second := a.stageAggregateEvaluationLocked(at), b.stageAggregateEvaluationLocked(at)
	if !aggregateEvaluationEqual(first, second) {
		t.Fatal("source-equivalent canonical states produced different evaluation")
	}
	a.state.aggregateEvaluator.current = cloneAggregateEvaluation(first)
	a.state.binding.symbols[0].aggregates.tail[at.Add(-time.Second).Unix()].values.Close = 13
	corrected := a.stageAggregateEvaluationLocked(at)
	if corrected.at != first.at || corrected.rows[0].dayPercent == first.rows[0].dayPercent {
		t.Fatal("same-T correction did not replace the coherent staged result")
	}
	bad := corrected
	bad.at = at.Add(time.Second)
	if aggregateEvaluationEqual(corrected, bad) {
		t.Fatal("staged T identity was ignored")
	}

	materialized := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	recomputed := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	materializedCandidate := materialized.stageAggregateEvaluationLocked(at)
	_ = recomputed.stageAggregateEvaluationLocked(at) // keep stage-side initialization identical
	materialized.applyAggregateCandidateResultLocked(&materializedCandidate, at)
	recomputed.applyAggregateCandidateLocked(at, at)
	if !reflect.DeepEqual(materialized.replayDeterministicViewLocked().Canonical, recomputed.replayDeterministicViewLocked().Canonical) ||
		!materialized.state.committedT.Equal(*recomputed.state.committedT) {
		t.Fatal("staged feature materialization changed the deterministic apply result")
	}
}

func BenchmarkReplayObservationProjection5691(b *testing.B) {
	const population = 5691
	start := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	at := start.Add(5*time.Hour + 30*time.Minute)
	binding := &installedBinding{identity: "benchmark", tradingDate: "2026-08-07", sessionStart: start, sessionEnd: start.Add(16 * time.Hour),
		symbols: make([]coreSymbol, population), index: make(map[string]int, population)}
	coveredSlots := sessionSlot(binding, at)
	for index := range binding.symbols {
		symbol := fmt.Sprintf("S%04d", index)
		binding.index[symbol] = index
		absence := &slotBitmap{}
		for slot := 0; slot < coveredSlots; slot++ {
			absence.set(slot)
		}
		windowStart := at.Add(-time.Second)
		values := AggregateValues{Open: 10, High: 10.1, Low: 9.9, Close: 10, Volume: 10_000, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSRESTFloorVolumeOverTrades}
		record := canonicalAggregate{identity: aggregateIdentity{symbol: symbol, start: windowStart.Unix()}, windowStart: windowStart, windowEnd: at, values: values}
		references := make(map[int64]activityBlockSummary, coveredSlots/30)
		for blockEnd := start.Add(activityBlockDuration); !blockEnd.After(at.Add(-activityBlockDuration)); blockEnd = blockEnd.Add(activityBlockDuration) {
			references[blockEnd.Unix()] = activityBlockSummary{end: blockEnd.Unix(), transactions: 1_000, high: 10.1, low: 9.9, expansionBPS: 200, aggregateCount: 30}
		}
		qualification := &qualificationState{finalizedGateBars: make(map[int64]qualificationGateBar), proofs: make(map[int64]struct{}), dirty: make(map[int64]struct{}),
			accountedThrough: at, finalized: true, finalProofEnd: start.Add(time.Minute), result: qualificationResult{at: at, status: qualificationFinalized, finalProofEnd: start.Add(time.Minute)}}
		state := &symbolAggregateState{tail: map[int64]*canonicalAggregate{windowStart.Unix(): &record}, latest: &latestAggregateMark{record: record}, provenAbsent: absence,
			qualification: qualification,
			priceRange: &priceRangeFeatureState{firstStart: start.Unix(), firstOpen: 10, hasFirst: true,
				sessionLows: []extremaPoint{{windowStart: windowStart.Unix(), value: 9.9}}, sessionHighs: []extremaPoint{{windowStart: windowStart.Unix(), value: 10.1}}},
			activity: &activityFeatureState{references: references, mutable: make(map[int64]activityMutableBlock)}}
		binding.symbols[index] = coreSymbol{symbol: symbol, prior: frozenPriorClose{symbol: symbol, status: reference.PriorCloseValid, close: 9}, aggregates: state}
	}
	e := &Engine{mode: RunModeReplay, replayObservationStart: immutableTime(at), state: &engineState{binding: binding, lifecycle: lifecycleReplaying,
		replay: replayState{complete: true, observationStart: at, lastGroup: at}, aggregateEvaluator: aggregateEvaluatorState{invalidMarks: make(map[int]invalidMarkEvidence), coverage: make(map[int]aggregateCoverageConsequence)}}}
	b.ReportMetric(population, "symbols/op")
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		candidate := e.stageAggregateEvaluationAtLocked(at, at)
		if validateAggregateEvaluation(candidate) != nil {
			b.Fatal("synthetic replay projection became invalid")
		}
		e.applyAggregateCandidateResultLocked(&candidate, at)
	}
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
			installExactCoverage(binding.symbols[i].aggregates, binding, coverageStart, at)
		}
	}
	engine := &Engine{state: &engineState{binding: binding, lifecycle: lifecycleLive, committedT: immutableTime(at), clockMonotonic: true}}
	engine.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
	for i, spec := range specs {
		if spec.mark == 0 && spec.priorStatus == reference.PriorCloseValid {
			engine.state.aggregateEvaluator.coverage[i] = coverageUnknownFailureOrFence
		}
	}
	return engine
}

func installEvaluatorMark(e *Engine, index int, at time.Time, price float64, status qualificationStatus) {
	installEvaluatorMarkOnSymbol(&e.state.binding.symbols[index], at, price, status)
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
		qualification.finalized, qualification.finalProofEnd, qualification.result.finalProofEnd = true, at.Add(-time.Minute), at.Add(-time.Minute)
	}
	qualification.unresolvedOrigin = uncertaintyBootstrapOrigin
	qualification.result.unresolvedOrigin = uncertaintyBootstrapOrigin
	symbol.aggregates = &symbolAggregateState{tail: map[int64]*canonicalAggregate{window.Unix(): record}, latest: &latestAggregateMark{record: *record}, qualification: qualification}
}
