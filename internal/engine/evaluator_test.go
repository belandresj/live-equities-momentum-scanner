package engine

import (
	"context"
	"fmt"
	"math"
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
	operational := runtime.ObserveOperational()
	if view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonAccountingIntegrity || view.currentMarketClaim ||
		operational.IntegrityFailure == nil || operational.IntegrityFailure.Category != EvaluatorPopulationAccounting || operational.IntegrityFailure.EngineSequence == 0 ||
		operational.IntegrityFailure.FirstField != "accounting.population" || operational.IntegrityFailure.FirstReason != "identity_mismatch" {
		t.Fatalf("accounting mismatch escaped containment: %+v", view)
	}
	closeAndWait(t, runtime)
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
	if exact.mode != rankingQualifiedCurrent || len(exact.rows) != 1 || !exact.rows[0].tqIntentEligible || exact.rows[0].activity30s.status != featureCurrent {
		t.Fatalf("qualified independent fields = %+v", exact)
	}
	degradedEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}, {"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved}})
	degradedEngine.state.lifecycle = lifecycleHydrating
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
	qualificationOnlyEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}})
	qualificationOnlyEngine.state.lifecycle = lifecycleHydrating
	qualificationOnly := qualificationOnlyEngine.stageAggregateEvaluationLocked(at)
	if qualificationOnly.mode != rankingDegradedBootstrap || qualificationOnly.reason != rankingReasonQualificationPending {
		t.Fatalf("qualification-only degradation = %+v", qualificationOnly)
	}
	populationOnlyEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationNotYetPassed}, {"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved}})
	populationOnlyEngine.state.lifecycle = lifecycleHydrating
	populationOnly := populationOnlyEngine.stageAggregateEvaluationLocked(at)
	if populationOnly.mode != rankingDegradedBootstrap || populationOnly.reason != rankingReasonIncompletePopulation {
		t.Fatalf("population-only degradation = %+v", populationOnly)
	}
	invalidFieldEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional}})
	invalidFieldEngine.state.binding.symbols[0].aggregates.activity = &activityFeatureState{boundExceeded: true}
	invalidField := invalidFieldEngine.stageAggregateEvaluationLocked(at)
	if invalidField.mode != rankingQualifiedCurrent || len(invalidField.rows) != 1 || invalidField.rows[0].activity30s.status != featureCurrent {
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

func TestSlice2LocalUncertaintyPublishesCurrentPartialRanking(t *testing.T) {
	at := time.Date(2026, 8, 13, 15, 15, 0, 0, time.UTC)
	postBootstrap := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}}).stageAggregateEvaluationLocked(at)
	if postBootstrap.mode != rankingDegradedCurrent || postBootstrap.uncertainty.bootstrapOrigin != 1 || validateAggregateEvaluation(postBootstrap) != nil {
		t.Fatalf("live lifecycle retained startup-only degraded_bootstrap: %+v", postBootstrap)
	}
	laterMarkEngine := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}})
	laterMarkState := laterMarkEngine.state.binding.symbols[0].aggregates
	position := LivePosition{ConnectionEpoch: 1, FrameSequence: 2}
	laterMarkState.latest.record.authority = aggregateEvidence{source: AggregateSourceLive, live: position}
	laterMarkState.latest.record.greatestLiveSupport = &position
	for _, record := range laterMarkState.tail {
		record.authority = aggregateEvidence{source: AggregateSourceLive, live: position}
		record.greatestLiveSupport = &position
	}
	laterMarkEngine.state.aggregateEvaluator.coverage[0] = coverageUnknownFailureOrFence
	laterMark := laterMarkEngine.stageAggregateEvaluationLocked(at)
	if laterMark.mode != rankingDegradedCurrent || laterMark.population.trustedRankableMark != 1 || laterMark.population.unknownDueFailureOrFence != 0 ||
		laterMark.qualification.unresolved != 1 || laterMark.populationTransition.trustedByLaterLiveMark != 1 || len(laterMark.rows) != 1 || validateAggregateEvaluation(laterMark) != nil {
		t.Fatalf("later trusted mark did not restore primary trust while retaining qualification uncertainty: %+v", laterMark)
	}

	qualificationOnlyEngine := evaluatorProofEngine(at, []evaluatorSymbol{
		{"AAA", reference.PriorCloseValid, 10, 14, qualificationNotYetPassed},
		{"BBB", reference.PriorCloseValid, 10, 12, qualificationUnresolved},
	})
	qualificationOnlyEngine.mode = RunModeLive
	qualification := qualificationOnlyEngine.state.binding.symbols[1].aggregates.qualification
	qualification.invalid = true
	qualification.unresolvedOrigin = uncertaintyLocalInvalid
	qualification.result.unresolvedOrigin = uncertaintyLocalInvalid
	qualificationOnly := qualificationOnlyEngine.stageAggregateEvaluationLocked(at)
	if qualificationOnly.mode != rankingDegradedCurrent || qualificationOnly.reason != rankingReasonQualificationPending ||
		qualificationOnly.population.unresolvedPopulation != 0 || qualificationOnly.qualification.unresolved != 1 ||
		qualificationOnly.uncertainty.localInvalid != 1 || len(qualificationOnly.rows) != 2 ||
		qualificationOnly.rows[0].symbol != "AAA" || qualificationOnly.rows[1].symbol != "BBB" ||
		qualificationOnly.rows[0].tqIntentEligible || qualificationOnly.rows[1].tqIntentEligible || validateAggregateEvaluation(qualificationOnly) != nil {
		t.Fatalf("qualification-local partial projection = %+v", qualificationOnly)
	}

	populationEngine := evaluatorProofEngine(at, []evaluatorSymbol{
		{"AAA", reference.PriorCloseValid, 10, 14, qualificationNotYetPassed},
		{"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved},
	})
	populationEngine.mode = RunModeLive
	populationEngine.state.aggregateEvaluator.coverage[1] = aggregateCoverageConsequence{outcome: coverageOutcomeUnknown, origin: uncertaintyLocalInvalid}
	population := populationEngine.stageAggregateEvaluationLocked(at)
	if population.mode != rankingDegradedCurrent || population.reason != rankingReasonIncompletePopulation ||
		population.population.unresolvedPopulation != 1 || population.qualification.unresolved != 0 || population.uncertainty.localInvalid != 1 ||
		len(population.rows) != 1 || population.rows[0].symbol != "AAA" || population.rows[0].tqIntentEligible || validateAggregateEvaluation(population) != nil {
		t.Fatalf("population-local partial projection = %+v", population)
	}

	bad := cloneAggregateEvaluation(population)
	bad.rows[0].tqIntentEligible = true
	if validation := validateAggregateEvaluation(bad); validation == nil || validation.Category != EvaluatorTQIntent {
		t.Fatalf("partial projection promoted T/Q: %+v", validation)
	}
	replay := evaluatorProofEngine(at, []evaluatorSymbol{
		{"AAA", reference.PriorCloseValid, 10, 14, qualificationNotYetPassed},
		{"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved},
	})
	replay.mode = RunModeReplay
	replay.state.aggregateEvaluator.coverage[1] = aggregateCoverageConsequence{outcome: coverageOutcomeUnknown, origin: uncertaintyLocalInvalid}
	replayEvaluation := replay.stageAggregateEvaluationLocked(at)
	if replayEvaluation.mode != rankingUnavailable || replayEvaluation.reason != rankingReasonIncompletePopulation || len(replayEvaluation.rows) != 0 {
		t.Fatalf("live partial mode changed replay semantics: %+v", replayEvaluation)
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
	failedAdvance.mode = RunModeLive
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
		}
	}
	engine := &Engine{mode: RunModeLive, state: &engineState{binding: binding, lifecycle: lifecycleLive, committedT: immutableTime(at), clockMonotonic: true}}
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
