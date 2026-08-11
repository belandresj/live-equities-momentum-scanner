package engine

import (
	"context"
	"testing"
	"time"
	"unsafe"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestC3R2FoldedNonAlignedActivityTarget amends C3-ACT-01/02. It proves that
// a post-committed-T aggregate remains semantically available after strict raw
// tail folding when a later non-30-second-aligned candidate target consumes it.
func TestC3R2FoldedNonAlignedActivityTarget(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	oldT := start.Add(11 * activityBlockDuration)
	laterT := oldT.Add(17 * time.Second)
	now := oldT
	e := aggregateEngine(t, binding, RunModeLive, &now)
	accepted := make(map[int64]AggregateValues)
	var frame uint64

	// Ten exact eligible reference blocks make every target statistic and final
	// Activity value observable rather than stopping at reference warm-up.
	for block := 0; block < 10; block++ {
		window := start.Add(time.Duration(block) * activityBlockDuration)
		frame++
		input := liveAggregate(binding, "AAA", window, 1, frame)
		input.Values = activityValues(100+float64(block), float64(block+1))
		applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
		accepted[window.Unix()] = input.Values
	}
	preWindow := oldT.Add(-time.Second)
	frame++
	pre := liveAggregate(binding, "AAA", preWindow, 1, frame)
	pre.Values = activityValues(150, 15)
	applyAggregate(t, e, pre, DispositionAggregateInserted, ReasonNone)
	accepted[preWindow.Unix()] = pre.Values
	proveAggregateCoverage(t, e, "AAA", start, laterT)

	e.mu.Lock()
	e.applyAggregateCandidateLocked(oldT, oldT)
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	oldResult := state.activity.result
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(oldT)
	e.mu.Unlock()
	assertActivityResult(t, oldResult, fullActivityReference(start, accepted, oldT, true))

	// Exercise replacement while the identity is still mutable. Only the
	// revised canonical value may enter the folded target projection.
	postWindow := oldT.Add(5 * time.Second)
	now = postWindow.Add(time.Second)
	frame++
	post := liveAggregate(binding, "AAA", postWindow, 1, frame)
	post.Values = activityValues(120, 4)
	post.DeliveryTime = now
	applyAggregate(t, e, post, DispositionAggregateInserted, ReasonNone)
	frame++
	revised := post
	revised.Live.FrameSequence = frame
	revised.Values = activityValues(240, 40)
	applyAggregate(t, e, revised, DispositionAggregateRevised, ReasonNone)
	accepted[postWindow.Unix()] = revised.Values

	// Advance actual engine time strictly past H for the entire later target.
	// The Component 2 gate remains closed for the later T, so old T and its
	// applied Activity result must stay unchanged while compaction occurs.
	foldNow := laterT.Add(correctionHorizon + activityBlockDuration + time.Nanosecond)
	e.mu.Lock()
	e.delay = foldNow.Sub(laterT)
	e.mu.Unlock()
	now = foldNow
	admission, completion := e.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatalf("strict-fold timer admission=%s", admission)
	}

	e.mu.Lock()
	state = e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	if e.state.committedT == nil || !e.state.committedT.Equal(oldT) || state.activity.result != oldResult {
		e.mu.Unlock()
		t.Fatalf("old-T Activity changed committed=%v got=%+v want=%+v", e.state.committedT, state.activity.result, oldResult)
	}
	if len(state.tail) != 0 || state.tail[postWindow.Unix()] != nil {
		e.mu.Unlock()
		t.Fatalf("target evidence survived in raw tail: %d", len(state.tail))
	}
	blockEnd := activityBlockEnd(e.state.binding, postWindow)
	block, retained := state.activity.foldedTargets[blockEnd.Unix()]
	slot := int(postWindow.Sub(blockEnd.Add(-activityBlockDuration)) / time.Second)
	consumedFoldedEvidence := retained && block.present&(uint32(1)<<uint(slot)) != 0
	oldAfterFold := evaluateActivityFeatures(e.state.binding, state, oldT)
	later := evaluateActivityFeatures(e.state.binding, state, laterT)
	e.applyAggregateCandidateLocked(laterT, foldNow)
	if !consumedFoldedEvidence || e.state.committedT == nil || !e.state.committedT.Equal(laterT) {
		e.mu.Unlock()
		t.Fatalf("folded target was not retained/committed retained=%v committed=%v", consumedFoldedEvidence, e.state.committedT)
	}
	if len(state.activity.foldedTargets) > maximumActivityTargetBlocks ||
		state.activity.foldedTargetContributions > maximumActivityTargetContributions {
		e.mu.Unlock()
		t.Fatalf("folded target bounds blocks=%d contributions=%d", len(state.activity.foldedTargets), state.activity.foldedTargetContributions)
	}
	e.mu.Unlock()

	assertActivityResult(t, oldAfterFold, fullActivityReference(start, accepted, oldT, true))
	wantLater := fullActivityReference(start, accepted, laterT, true)
	assertActivityResult(t, later, wantLater)
	if later.activity.status == featureUnavailable && later.activity.reason == featureReasonNoAggregateInTarget {
		t.Fatalf("folded target regressed to no aggregate: %+v", later)
	}
	if got := unsafe.Sizeof(activityFoldedTargetBlock{}); got != 728 ||
		maximumActivityTargetBlocks != sessionSeconds/30 || maximumActivityTargetContributions != sessionSeconds {
		t.Fatalf("folded target charge size=%d blocks=%d contributions=%d", got, maximumActivityTargetBlocks, maximumActivityTargetContributions)
	}
	closeAndWait(t, e)

	t.Run("permitted historical withdrawal removes folded target identity", func(t *testing.T) {
		now := start.Add(30 * time.Minute)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		window := start.Add(time.Minute)
		input := historicalAggregate(binding, "AAA", window, 1)
		input.Values = activityValues(180, 8)
		proof := proofFor(binding, input, window, window.Add(time.Second))
		applyHistorical(t, e, input, proof, DispositionAggregateInserted, ReasonNone)
		state := aggregateState(t, e, "AAA")
		if state.activity == nil || state.activity.foldedTargetContributions != 1 {
			t.Fatalf("deep historical target contribution=%+v", state.activity)
		}
		conflict := input
		conflict.Historical.RecordOrdinal = 2
		conflict.Values = activityValues(220, 9)
		applyHistorical(t, e, conflict, proof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		state = aggregateState(t, e, "AAA")
		if state.activity.foldedTargetContributions != 0 || len(state.activity.foldedTargets) != 0 ||
			state.historicalConflict == nil || !state.historicalConflict.has(sessionSlot(e.state.binding, window)) {
			t.Fatalf("withdrawal retained folded success evidence activity=%+v", state.activity)
		}
		closeAndWait(t, e)
	})
}

// TestC3R1ExactCoverageConsequences amends C3-FEAT-01, C3-ACT-01, and
// C3-QUAL-01. It distinguishes accepted presence and bitmap allocation from
// exact interval proof, then distinguishes unknown seconds from proven absence.
func TestC3R1ExactCoverageConsequences(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(10 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	applyAggregate(t, e, liveAggregate(binding, "AAA", start, 1, 1), DispositionAggregateInserted, ReasonNone)

	e.mu.Lock()
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	state.historicalConflict = &slotBitmap{} // allocation is not evidence
	price := evaluatePriceRangeFeatures(e.state.binding, &e.state.binding.symbols[e.state.binding.index["AAA"]], now)
	activity := evaluateActivityFeatures(e.state.binding, state, now)
	evaluateQualificationThrough(state, e.state.binding, now, now)
	qualification := state.qualification.result
	if exactAggregateCoverage(state, e.state.binding, start, now) {
		e.mu.Unlock()
		t.Fatal("accepted sparse aggregate or empty conflict bitmap established coverage")
	}
	if price.dayPercent.status != featureCurrent || price.from4AMPercent.status != featureCurrent ||
		price.hodDrawdown.reason != featureReasonHistoryIncomplete || price.rolling30.reason != featureReasonHistoryIncomplete || activity.activity.reason != featureReasonHistoryIncomplete ||
		qualification.status != qualificationUnresolved {
		e.mu.Unlock()
		t.Fatalf("unknown sparse consequences price=%+v activity=%+v qualification=%+v", price, activity, qualification)
	}
	if !installExactCoverage(state, e.state.binding, start, now) || !exactAggregateCoverage(state, e.state.binding, start, now) ||
		!state.provenAbsent.has(sessionSlot(e.state.binding, start.Add(time.Second))) {
		e.mu.Unlock()
		t.Fatal("exact coverage did not install localized proven absence")
	}
	e.mu.Unlock()
	closeAndWait(t, e)
}

// TestC3R1CoveredEmptyActivityTarget amends C3-ACT-01. An older session print
// does not turn a covered empty [T-30s,T) target into before_first_print.
func TestC3R1CoveredEmptyActivityTarget(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	at := start.Add(time.Minute)
	now := at
	e := aggregateEngine(t, binding, RunModeLive, &now)
	applyAggregate(t, e, liveAggregate(binding, "AAA", start, 1, 1), DispositionAggregateInserted, ReasonNone)
	proveAggregateCoverage(t, e, "AAA", start, at)
	got := activityResult(t, e, "AAA", at)
	if got.activity.status != featureUnavailable || got.activity.reason != featureReasonNoAggregateInTarget {
		t.Fatalf("covered empty target = %+v", got)
	}
	closeAndWait(t, e)
}

// TestC3R1CommittedBoundaryRetention amends C3-FEAT-02, C3-ACT-02, and
// C3-QUAL-02. A post-T record may be folded into bounded forward sufficient
// state, but it cannot change the mark/ranges at old T or be discarded before
// a later candidate can consume it.
func TestC3R1CommittedBoundaryRetention(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	oldT := start.Add(20 * time.Second)
	now := oldT
	e := aggregateEngine(t, binding, RunModeLive, &now)
	pre := liveAggregate(binding, "AAA", oldT.Add(-time.Second), 1, 1)
	pre.Values.Open, pre.Values.High, pre.Values.Low, pre.Values.Close = 10, 11, 9, 10
	applyAggregate(t, e, pre, DispositionAggregateInserted, ReasonNone)
	proveAggregateCoverage(t, e, "AAA", start, oldT)

	e.mu.Lock()
	e.applyAggregateCandidateLocked(oldT, oldT)
	before := evaluatePriceRangeFeatures(e.state.binding, &e.state.binding.symbols[e.state.binding.index["AAA"]], oldT)
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(oldT)
	e.mu.Unlock()

	postWindow := oldT.Add(time.Second)
	now = postWindow.Add(time.Second)
	post := liveAggregate(binding, "AAA", postWindow, 1, 2)
	post.Values.Open, post.Values.High, post.Values.Low, post.Values.Close = 100, 110, 90, 100
	post.DeliveryTime = now
	applyAggregate(t, e, post, DispositionAggregateInserted, ReasonNone)

	e.mu.Lock()
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	e.compactSymbolLocked(state, e.state.binding, "AAA", post.WindowEnd.Add(correctionHorizon+time.Nanosecond))
	after := evaluatePriceRangeFeatures(e.state.binding, &e.state.binding.symbols[e.state.binding.index["AAA"]], oldT)
	laterMark, laterAvailable := latestMarkBefore(state, post.WindowEnd)
	activityKey := activityBlockEnd(e.state.binding, postWindow).Unix()
	_, activityReferenceRetained := state.activity.references[activityKey]
	_, activityMutableRetained := state.activity.mutable[activityKey]
	activityRetained := activityReferenceRetained || activityMutableRetained
	_, qualificationRetained := state.qualification.finalizedGateBars[postWindow.Unix()]
	if before != after || state.committedLatest == nil || state.committedLatest.close != 10 || !laterAvailable || laterMark.values.Close != 100 ||
		!activityRetained || !qualificationRetained {
		e.mu.Unlock()
		t.Fatalf("stalled-T retention before=%+v after=%+v committed=%+v later=%+v activity=%v qualification=%v", before, after, state.committedLatest, laterMark, activityRetained, qualificationRetained)
	}
	e.mu.Unlock()
	closeAndWait(t, e)
}

// TestC3R1AtomicCandidateAndPublicationIdentity amends C3-EVAL-01. Candidate-T
// mismatch fails before any applied state, and private publication identity
// requires evaluation.at == watermark.
func TestC3R1AtomicCandidateAndPublicationIdentity(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	e.mode = RunModeReplay
	current := e.stageAggregateEvaluationLocked(at)
	e.state.aggregateEvaluator.current = cloneAggregateEvaluation(current)
	state := e.state.binding.symbols[0].aggregates
	ensurePriceRangeState(state).result = evaluatePriceRangeFeatures(e.state.binding, &e.state.binding.symbols[0], at)
	applyActivityResult(ensureActivityState(state), evaluateActivityFeatures(e.state.binding, state, at))
	beforeT, beforeEvaluation := *e.state.committedT, cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	beforePrice, beforeActivity, beforeQualification := state.priceRange.result, state.activity.result, state.qualification.result
	bad := current
	bad.at = at.Add(time.Second)
	if e.runAggregateEvaluatorLocked(&queueNode{kind: inputAggregate, admissionTime: at}, DispositionAggregateRevised, ReasonNone, &bad) ||
		!e.state.committedT.Equal(beforeT) || !aggregateEvaluationEqual(e.state.aggregateEvaluator.current, beforeEvaluation) ||
		state.priceRange.result != beforePrice || state.activity.result != beforeActivity || state.qualification.result != beforeQualification {
		t.Fatal("mismatched candidate changed applied state")
	}
	target := at.Add(time.Second)
	e.state.latestTarget = immutableTime(target)
	invalid := e.stageAggregateEvaluationAtLocked(target, target)
	invalid.population.universeTotal++
	if e.runAggregateEvaluatorLocked(&queueNode{kind: inputTimer, admissionTime: target}, DispositionTimerApplied, ReasonNone, &invalid) ||
		!e.state.committedT.Equal(beforeT) || !aggregateEvaluationEqual(e.state.aggregateEvaluator.current, beforeEvaluation) ||
		state.priceRange.result != beforePrice || state.activity.result != beforeActivity || state.qualification.result != beforeQualification {
		t.Fatal("invalid advancing candidate changed applied state")
	}

	binding := testBinding(t)
	now := binding.SessionStart()
	runtime := aggregateEngine(t, binding, RunModeLive, &now)
	runtime.mu.Lock()
	base := *runtime.publication.Load()
	evaluation := runtime.stageAggregateEvaluationLocked(now)
	base.watermark = immutableTime(now)
	base.aggregateEvaluation = evaluation
	if err := validatePublication(&base); err != nil {
		runtime.mu.Unlock()
		t.Fatalf("valid watermark/evaluation identity rejected: %v", err)
	}
	base.aggregateEvaluation.at = now.Add(time.Second)
	if err := validatePublication(&base); err == nil {
		runtime.mu.Unlock()
		t.Fatal("publication accepted watermark/evaluation-T mismatch")
	}
	runtime.mu.Unlock()
	closeAndWait(t, runtime)
}

// TestC3R1IndependentFeatureAccountingAndOrigins amends C3-POP-02 and
// C3-PROJ-01: invalid prior close affects Day/rankability only, every feature
// status/reason pair reconciles, unresolved is explicit, and only bootstrap
// uncertainty can produce degraded output.
func TestC3R1IndependentFeatureAccountingAndOrigins(t *testing.T) {
	at := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	invalidPrior := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseInvalid, 0, 12, qualificationNotYetPassed}}).stageAggregateEvaluationLocked(at)
	if invalidPrior.features.dayPercent.reasons[9] != 1 || invalidPrior.features.from4AMPercent.statuses[1] != 1 ||
		invalidPrior.features.hodDrawdown.statuses[1] != 1 || invalidPrior.features.activity.statuses[0] != 1 ||
		validateAggregateEvaluation(invalidPrior) != nil {
		t.Fatalf("invalid-prior independent accounting = %+v", invalidPrior)
	}
	tampered := invalidPrior
	tampered.features.activity.reasons[4]--
	tampered.features.activity.reasons[1]++
	if validateAggregateEvaluation(tampered) == nil {
		t.Fatal("contradictory feature status/reason counters reconciled")
	}

	bootstrap := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationNotYetPassed}, {"BBB", reference.PriorCloseValid, 10, 0, qualificationUnresolved}})
	if got := bootstrap.stageAggregateEvaluationLocked(at); got.mode != rankingDegradedBootstrap || got.uncertainty.bootstrapOrigin == 0 {
		t.Fatalf("bootstrap-origin degradation = %+v", got)
	}
	bootstrap.state.aggregateEvaluator.coverage[1] = coverageUnknownPostBootstrap
	if got := bootstrap.stageAggregateEvaluationLocked(at); got.mode == rankingDegradedBootstrap || got.reason != rankingReasonIncompletePopulation || got.uncertainty.postBootstrapGap == 0 {
		t.Fatalf("post-bootstrap uncertainty was relabeled degraded: %+v", got)
	}

	unresolved := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationUnresolved}}).stageAggregateEvaluationLocked(at)
	total := unresolved.qualification.unresolved + unresolved.qualification.notYetPassed + unresolved.qualification.provisional + unresolved.qualification.finalized
	if unresolved.qualification.unresolved != 1 || total != unresolved.population.trustedRankableMark {
		t.Fatalf("qualification overlap omitted unresolved: %+v", unresolved)
	}
}
