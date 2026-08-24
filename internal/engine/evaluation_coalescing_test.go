package engine

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestLiveAggregateEvaluationCoalescing proves that live aggregate mutation is
// synchronous and symbol-local while full-population evaluation/publication is
// coalesced at timer and hydration-ingress boundaries.
func TestLiveAggregateEvaluationCoalescing(t *testing.T) {
	t.Run("timer coalesces corrections and respects support", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA", "BBB", "CCC"})
		start := binding.SessionStart()
		t0, t1 := start.Add(2*time.Second), start.Add(3*time.Second)
		now := t0
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)

		e.mu.Lock()
		e.state.lifecycle = lifecycleLive
		e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
		e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
		e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
		e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 100, 1
		e.state.hydration.supportedThrough = immutableTime(t0)
		e.mu.Unlock()

		frame := uint64(1)
		for _, symbol := range []string{"AAA", "BBB", "CCC"} {
			for at := start; at.Before(t0); at = at.Add(time.Second) {
				frame++
				input := liveAggregate(binding, symbol, at, 1, frame)
				input.Values.Close = 10 + float64(frame)/100
				input.Values.High = input.Values.Close
				if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
					t.Fatalf("bootstrap aggregate %s/%s = %+v", symbol, at, got)
				}
			}
		}
		beforeInitial := e.observePublication()
		if !e.state.aggregateProjectionPending {
			t.Fatal("bootstrap aggregate prefix did not mark projection pending")
		}
		admission, timer := e.AdmitTimer(context.Background())
		initialDisposition := awaitTimerDisposition(t, timer)
		if admission != AdmissionAdmitted || initialDisposition.Code != DispositionTimerApplied {
			t.Fatalf("initial supported timer admission=%s disposition=%+v", admission, initialDisposition)
		}
		initial := e.observePublication()
		if initial.publicationID != beforeInitial.publicationID+1 || initial.watermark == nil || !initial.watermark.Equal(t0) ||
			initial.aggregateEvaluation.at != t0 || initial.aggregateEvaluation.population.universeTotal != 3 ||
			initial.aggregateEvaluation.population.trustedRankableMark != 3 || e.state.aggregateProjectionPending {
			t.Fatalf("initial coalesced evaluation = %+v pending=%t", initial.aggregateEvaluation, e.state.aggregateProjectionPending)
		}

		initialID, initialSequence := initial.publicationID, initial.lastEngineSequence
		now = t1
		future := liveAggregate(binding, "AAA", t0, 1, frame+1)
		future.Values.Close, future.Values.High = 15, 15
		if got := admitProductionAggregate(t, e, future); got.Code != DispositionAggregateInserted {
			t.Fatalf("future insert = %+v", got)
		}
		frame++
		futureClose := 15.0
		for revision := 0; revision < 200; revision++ {
			frame++
			revised := future
			revised.Live.FrameSequence = frame
			futureClose = 15 + float64(revision+1)/100
			revised.Values.Close, revised.Values.High = futureClose, futureClose
			if got := admitProductionAggregate(t, e, revised); got.Code != DispositionAggregateRevised {
				t.Fatalf("future revision %d = %+v", revision, got)
			}
		}
		correction := liveAggregate(binding, "AAA", start.Add(time.Second), 1, frame+1)
		correction.Values.Close, correction.Values.High = 12, 12
		if got := admitProductionAggregate(t, e, correction); got.Code != DispositionAggregateRevised {
			t.Fatalf("same-T correction = %+v", got)
		}
		frame++
		unknown := liveAggregate(binding, "ZZZ", start.Add(time.Second), 1, frame+1)
		if got := admitProductionAggregate(t, e, unknown); got.Code != DispositionAggregateRejected || got.Reason != ReasonSymbol {
			t.Fatalf("unknown symbol = %+v", got)
		}
		if got := e.observePublication(); got.publicationID != initialID || got.lastEngineSequence != initialSequence {
			t.Fatalf("aggregate prefix replaced snapshot: before=%d/%d after=%d/%d", initialID, initialSequence, got.publicationID, got.lastEngineSequence)
		}
		if record := aggregateRecord(t, e, "AAA", start.Add(time.Second)); record.values.Close != 12 {
			t.Fatalf("canonical correction not synchronous: %+v", record.values)
		}
		fallbackTiming := e.ObserveEvaluationTiming()
		cancelCommand, err := e.IssueLiveCoverageFence()
		if err != nil {
			t.Fatal(err)
		}
		_, canceled := e.AdmitLiveCoverageFenceCancellation(context.Background(), cancelCommand)
		if got := <-canceled; got.Code != DispositionLiveCoverageFenceRejected || e.ObserveEvaluationTiming() != fallbackTiming || !e.state.aggregateProjectionPending {
			t.Fatalf("canceled fence changed pending evaluation got=%+v timing=%+v pending=%t", got, e.ObserveEvaluationTiming(), e.state.aggregateProjectionPending)
		}
		staleFact, err := NewLiveCoverageFenceInput(cancelCommand, LiveCoverageFenceComplete, frame, frame+1, t1)
		if err != nil {
			t.Fatal(err)
		}
		_, fenced := e.AdmitLiveCoverageFence(context.Background(), staleFact)
		if got := <-fenced; got.Code != DispositionLiveCoverageFenceFenced || e.ObserveEvaluationTiming() != fallbackTiming || !e.state.aggregateProjectionPending {
			t.Fatalf("fenced fence changed pending evaluation got=%+v timing=%+v pending=%t", got, e.ObserveEvaluationTiming(), e.state.aggregateProjectionPending)
		}

		// latestTarget=t1 is unsupported, so pending work must evaluate at t0.
		_, timer = e.AdmitTimer(context.Background())
		fallbackDisposition := awaitTimerDisposition(t, timer)
		fallback := e.observePublication()
		if fallback.publicationID != initialID+1 || fallback.lastEngineSequence != fallbackDisposition.EngineSequence ||
			fallback.watermark == nil || !fallback.watermark.Equal(t0) || fallback.aggregateEvaluation.at != t0 || e.state.aggregateProjectionPending {
			t.Fatalf("same-T fallback publication = %+v pending=%t", fallback, e.state.aggregateProjectionPending)
		}
		if fallback.aggregates.consumed != 209 || fallback.aggregates.inserted != 7 || fallback.aggregates.revised != 201 || fallback.aggregates.rejected != 1 {
			t.Fatalf("coalesced aggregate accounting = %+v", fallback.aggregates)
		}
		e.mu.Lock()
		correctedMark := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates.committedLatest
		e.state.hydration.supportedThrough = immutableTime(t1)
		e.mu.Unlock()
		if correctedMark == nil || correctedMark.values.Close != 12 {
			t.Fatalf("same-T correction not applied: %+v", correctedMark)
		}

		_, timer = e.AdmitTimer(context.Background())
		advancedDisposition := awaitTimerDisposition(t, timer)
		advanced := e.observePublication()
		if advanced.watermark == nil || !advanced.watermark.Equal(t1) || advanced.aggregateEvaluation.at != t1 ||
			advanced.lastEngineSequence != advancedDisposition.EngineSequence || advanced.publicationID != fallback.publicationID+1 {
			t.Fatalf("supported later-T publication = %+v", advanced)
		}
		e.mu.Lock()
		advancedMark := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates.committedLatest
		e.mu.Unlock()
		if advancedMark == nil || advancedMark.values.Close != futureClose {
			t.Fatalf("later target did not include future insert: %+v", advancedMark)
		}
	})

	t.Run("ingress fence consumes one pending prefix", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
		start := binding.SessionStart()
		now := start.Add(2 * time.Second)
		e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
		defer closeAndWait(t, e)
		plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if len(plan.Plan.Requests()) != 2 {
			t.Fatalf("hydration requests = %d, want 2", len(plan.Plan.Requests()))
		}
		for index, symbol := range []string{"AAA", "BBB"} {
			input := liveAggregate(binding, symbol, start.Add(time.Second), 1, uint64(index+2))
			if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
				t.Fatalf("live tail %s = %+v", symbol, got)
			}
		}
		var terminal HydrationDisposition
		for _, token := range plan.Plan.Requests() {
			fact, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			terminal = admitHydrationTerminal(t, e, fact)
		}
		before := e.observePublication()
		now = start.Add(3 * time.Second)
		fence, err := NewAggregateIngressFenceInput(terminal.FenceCommand, AggregateIngressFenceComplete, 3, 1, now)
		if err != nil {
			t.Fatal(err)
		}
		admission, completion := e.AdmitAggregateIngressFence(context.Background(), fence)
		if admission != AdmissionAdmitted {
			t.Fatalf("fence admission = %s", admission)
		}
		got := awaitHydrationDisposition(t, completion)
		after := e.observePublication()
		if got.Code != DispositionAggregateIngressFenceApplied || after.publicationID != before.publicationID+1 ||
			after.lastEngineSequence != got.EngineSequence || after.watermark == nil || !after.watermark.Equal(now) ||
			after.aggregateEvaluation.at != now || after.aggregateEvaluation.population.universeTotal != 2 || e.state.aggregateProjectionPending {
			t.Fatalf("coalesced fence disposition=%+v publication=%+v pending=%t", got, after, e.state.aggregateProjectionPending)
		}
	})

	t.Run("ordinary live fence commits its captured second after admission crosses a second", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
		start := binding.SessionStart()
		t0, fenceTarget, admissionAt := start.Add(2*time.Second), start.Add(3*time.Second), start.Add(4*time.Second)
		now := t0
		e := acknowledgedHydrationEngine(t, binding, &now, t0, lifecycleLive)
		defer closeAndWait(t, e)

		e.mu.Lock()
		e.state.hydration.fenceReconciled = true
		e.state.hydration.fenceEpoch = 1
		e.state.hydration.fenceThrough = 1
		e.state.hydration.fenceMarkerOrdinal = 1
		e.state.hydration.supportedThrough = immutableTime(t0)
		if e.state.aggregateEvaluator.coverage == nil {
			e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
		}
		for index := range e.state.binding.symbols {
			state := ensureAggregateState(&e.state.binding.symbols[index])
			if !installExactCoverage(state, e.state.binding, start, t0, nil) {
				e.mu.Unlock()
				t.Fatal("initial exact coverage setup failed")
			}
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
		}
		e.mu.Unlock()

		admission, timer := e.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
			t.Fatal("initial supported timer was not applied")
		}
		initial := e.observePublication()
		if initial.watermark == nil || !initial.watermark.Equal(t0) || initial.aggregateEvaluation.at != t0 {
			t.Fatalf("initial publication = %+v", initial)
		}

		command, err := e.IssueLiveCoverageFence()
		if err != nil {
			t.Fatal(err)
		}
		fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, 1, 2, fenceTarget)
		if err != nil {
			t.Fatal(err)
		}
		// Production waits for the raw-frame marker before admitting the timer.
		// Model a fence whose exact captured target is one second older than the
		// admission clock; the old path then sampled admissionAt for the timer,
		// found it unsupported, and left committed T frozen at t0 indefinitely.
		now = admissionAt
		admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
		if admission != AdmissionAdmitted || completion == nil {
			t.Fatalf("coverage admission = %s", admission)
		}
		disposition := <-completion
		if disposition.Code != DispositionLiveCoverageFenceApplied {
			t.Fatalf("coverage disposition = %+v", disposition)
		}
		afterFence := e.observePublication()
		fenceTiming := e.ObserveEvaluationTiming()
		if afterFence.watermark == nil || !afterFence.watermark.Equal(fenceTarget) || afterFence.aggregateEvaluation.at != fenceTarget ||
			afterFence.lastEngineSequence != disposition.EngineSequence || afterFence.publicationID != initial.publicationID+1 ||
			fenceTiming.Source != AggregateEvaluationLiveCoverageFence || !fenceTiming.Target.Equal(fenceTarget) {
			t.Fatalf("accepted fence did not commit its exact target: initial=%+v after=%+v", initial, afterFence)
		}

		admission, timer = e.AdmitMaintenanceTimer(context.Background())
		if admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
			t.Fatal("post-fence timer was not applied")
		}
		afterUnsupportedTimer := e.observePublication()
		if afterUnsupportedTimer.watermark == nil || !afterUnsupportedTimer.watermark.Equal(fenceTarget) ||
			afterUnsupportedTimer.aggregateEvaluation.at != fenceTarget || e.ObserveEvaluationTiming() != fenceTiming {
			t.Fatalf("unsupported later timer changed the committed boundary: %+v", afterUnsupportedTimer)
		}
	})

	t.Run("post-fence correction remains pending through maintenance and is consumed by next fence", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA"})
		start := binding.SessionStart()
		t0, t1 := start.Add(2*time.Second), start.Add(3*time.Second)
		now := t0
		e := acknowledgedHydrationEngine(t, binding, &now, t0, lifecycleLive)
		defer closeAndWait(t, e)
		e.mu.Lock()
		e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
		e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
		e.state.hydration.supportedThrough = immutableTime(t0)
		state := ensureAggregateState(&e.state.binding.symbols[0])
		if !installExactCoverage(state, e.state.binding, start, t0, nil) {
			e.mu.Unlock()
			t.Fatal("initial coverage")
		}
		e.mu.Unlock()

		_, initialTimer := e.AdmitTimer(context.Background())
		if awaitTimerDisposition(t, initialTimer).Code != DispositionTimerApplied {
			t.Fatal("initial timer")
		}
		now = t1
		applyLiveCoverageFenceForCoalescing(t, e, 1, 2, t1)
		f1Timing := e.ObserveEvaluationTiming()
		correction := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2)
		correction.Values.Close, correction.Values.High = 13, 13
		if got := admitProductionAggregate(t, e, correction); got.Code != DispositionAggregateInserted {
			t.Fatalf("post-fence aggregate=%+v", got)
		}
		if !e.state.aggregateProjectionPending {
			t.Fatal("post-fence aggregate did not remain pending")
		}
		_, maintenance := e.AdmitMaintenanceTimer(context.Background())
		if awaitTimerDisposition(t, maintenance).Code != DispositionTimerApplied || e.ObserveEvaluationTiming() != f1Timing || !e.state.aggregateProjectionPending {
			t.Fatalf("maintenance consumed pending work timing=%+v pending=%t", e.ObserveEvaluationTiming(), e.state.aggregateProjectionPending)
		}
		applyLiveCoverageFenceForCoalescing(t, e, 2, 3, t1)
		f2Timing := e.ObserveEvaluationTiming()
		committed := e.state.binding.symbols[0].aggregates.committedLatest
		if f2Timing.Starts.LiveCoverageFence != f1Timing.Starts.LiveCoverageFence+1 || f2Timing.Starts.Timer != f1Timing.Starts.Timer ||
			e.state.aggregateProjectionPending || committed == nil || committed.values.Close != 13 {
			t.Fatalf("second fence timing=%+v pending=%t evaluation=%+v", f2Timing, e.state.aggregateProjectionPending, e.observePublication().aggregateEvaluation)
		}
	})

	t.Run("supported same-target timer preserves strict correction-horizon finalization", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA"})
		target := binding.SessionEnd()
		now := target.Add(-time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)

		e.mu.Lock()
		e.state.lifecycle = lifecycleLive
		installEvaluatorMarkOnSymbol(&e.state.binding.symbols[0], target, 12, qualificationProvisional)
		e.state.binding.symbols[0].prior = frozenPriorClose{symbol: "AAA", status: reference.PriorCloseValid, close: 10}
		e.state.committedT = immutableTime(target)
		e.state.latestTarget = immutableTime(target)
		e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(target, target)
		e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
		e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
		e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch, e.state.hydration.fenceThrough = true, 1, 1
		e.state.hydration.fenceMarkerOrdinal = 1
		e.state.hydration.supportedThrough = immutableTime(target)
		deadline := target.Add(correctionHorizon)
		e.state.aggregateEvaluationDeadline = immutableTime(deadline)
		e.mu.Unlock()
		now = deadline.Add(time.Nanosecond)

		qualification := e.state.binding.symbols[0].aggregates.qualification
		if qualification.finalized || qualification.result.status != qualificationProvisional {
			t.Fatalf("precondition qualification = %+v", qualification)
		}
		admission, timer := e.AdmitTimer(context.Background())
		disposition := awaitTimerDisposition(t, timer)
		if admission != AdmissionAdmitted || disposition.Code != DispositionTimerApplied {
			t.Fatalf("same-target timer admission=%s disposition=%+v", admission, disposition)
		}
		qualification = e.state.binding.symbols[0].aggregates.qualification
		if !qualification.finalized || qualification.result.status != qualificationFinalized || !qualification.result.at.Equal(target) {
			t.Fatalf("same-target timer skipped strict horizon finalization: %+v", qualification)
		}
	})

	t.Run("session-end maintenance timer flushes pending same-target work exactly once", func(t *testing.T) {
		binding := hydrationPopulationBinding(t, []string{"AAA"})
		target := binding.SessionEnd()
		now := target.Add(-time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		e.mu.Lock()
		e.state.lifecycle = lifecycleLive
		installEvaluatorMarkOnSymbol(&e.state.binding.symbols[0], target, 12, qualificationFinalized)
		e.state.binding.symbols[0].prior = frozenPriorClose{symbol: "AAA", status: reference.PriorCloseValid, close: 10}
		e.state.committedT, e.state.latestTarget = immutableTime(target), immutableTime(target)
		e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(target, target)
		e.state.aggregateProjectionPending = true
		e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
		e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
		e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch, e.state.hydration.fenceThrough = true, 1, 1
		e.state.hydration.fenceMarkerOrdinal = 1
		e.state.hydration.supportedThrough = immutableTime(target)
		starts := e.state.evaluationTiming.Starts
		e.mu.Unlock()
		now = target.Add(4 * time.Second)

		admission, timer := e.AdmitMaintenanceTimer(context.Background())
		disposition := awaitTimerDisposition(t, timer)
		timing := e.ObserveEvaluationTiming()
		if admission != AdmissionAdmitted || disposition.Code != DispositionTimerApplied || e.state.aggregateProjectionPending ||
			timing.Source != AggregateEvaluationTimer || timing.Starts.Timer != starts.Timer+1 || e.state.lifecycle != lifecycleEnded {
			t.Fatalf("session-end flush admission=%s disposition=%+v timing=%+v pending=%t lifecycle=%s", admission, disposition, timing, e.state.aggregateProjectionPending, e.state.lifecycle)
		}
	})

	t.Run("invalid candidate retains pending work", func(t *testing.T) {
		at := time.Date(2026, 8, 6, 15, 0, 0, 0, time.UTC)
		e := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
		e.mode = RunModeLive
		e.state.aggregateProjectionPending = true
		candidate := e.stageAggregateEvaluationAtLocked(at, at)
		candidate.population.universeTotal++
		node := &queueNode{kind: inputTimer, admissionTime: at}
		if e.runAggregateEvaluatorLocked(node, DispositionTimerApplied, ReasonNone, &candidate) || !e.state.aggregateProjectionPending {
			t.Fatal("invalid candidate consumed pending projection work")
		}
	})
}

func applyLiveCoverageFenceForCoalescing(t *testing.T, e *Engine, through, marker uint64, capturedAt time.Time) {
	t.Helper()
	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, through, marker, capturedAt)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("live fence admission=%s", admission)
	}
	if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("live fence=%+v", got)
	}
}

func TestEvaluationCoalescingSemanticDifferential(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
	start := binding.SessionStart()
	target := start.Add(20 * time.Minute)
	now := target.Add(time.Nanosecond)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)
	historical := historicalAggregate(binding, "BBB", start.Add(time.Minute), 1)
	proof := proofFor(binding, historical, start, target)
	applyHistorical(t, e, historical, proof, DispositionAggregateInserted, ReasonNone)
	conflict := changedClose(historical, historical.Values.Close+1)
	applyHistorical(t, e, conflict, proof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)

	e.mu.Lock()
	e.state.lifecycle = lifecycleHydrating
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(target)
	e.mu.Unlock()

	insert := liveAggregate(binding, "AAA", target.Add(-2*time.Second), 1, 2)
	insert.Values.Close, insert.Values.High = 11, 11
	if got := admitProductionAggregate(t, e, insert); got.Code != DispositionAggregateInserted {
		t.Fatalf("insert=%+v", got)
	}
	revision := insert
	revision.Live.FrameSequence = 3
	revision.Values.Close, revision.Values.High = 12, 12
	if got := admitProductionAggregate(t, e, revision); got.Code != DispositionAggregateRevised {
		t.Fatalf("revision=%+v", got)
	}
	e.mu.Lock()
	e.state.lifecycle = lifecycleLive
	for index := range e.state.binding.symbols {
		state := ensureAggregateState(&e.state.binding.symbols[index])
		if !installExactCoverage(state, e.state.binding, start, target, nil) {
			e.mu.Unlock()
			t.Fatalf("coverage index=%d", index)
		}
	}
	qualification := ensureQualificationState(e.state.binding.symbols[0].aggregates)
	proofEnd := target.Add(-correctionHorizon)
	qualification.proofs[proofEnd.Unix()] = struct{}{}
	qualification.accountedThrough = target
	qualification.result = qualificationResult{at: target, status: qualificationProvisional, currentProofCount: 1}
	// B1 advances qualification in the owner-maintenance phase before staging
	// compact selection. The semantic oracle mirrors that phase explicitly;
	// stageAggregateEvaluationAtLocked is now a nonmutating candidate read.
	for index := range e.state.binding.symbols {
		symbol := &e.state.binding.symbols[index]
		if state := symbol.aggregates; state != nil {
			evaluateQualificationThrough(state, e.state.binding, target, now)
			var invalid *invalidMarkEvidence
			if evidence, ok := e.state.aggregateEvaluator.invalidMarks[index]; ok {
				copyEvidence := evidence
				invalid = &copyEvidence
			}
			maintainCurrentFieldStatuses(e.state.binding, symbol, target, invalid)
		}
	}
	oracle := e.stageAggregateEvaluationAtLocked(target, now)
	if validation := validateAggregateEvaluation(oracle); validation != nil {
		e.mu.Unlock()
		t.Fatalf("oracle=%v", validation)
	}
	accountingBefore := e.state.aggregates
	e.mu.Unlock()

	applyLiveCoverageFenceForCoalescing(t, e, 3, 2, now)
	e.mu.Lock()
	actual := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	accountingAfter := e.state.aggregates
	e.mu.Unlock()
	if !aggregateEvaluationEqual(actual, oracle) || accountingAfter != accountingBefore {
		t.Fatalf("coalesced/oracle mismatch actual=%+v oracle=%+v accounting=%+v/%+v", actual, oracle, accountingBefore, accountingAfter)
	}
	if actual.qualification.finalized != 1 || actual.population.trustedRankableMark != 1 || actual.population.unknownDueFailureOrFence != 1 || actual.uncertainty.localInvalid != 1 {
		t.Fatalf("trace outcomes=%+v", actual)
	}
	timing := e.ObserveEvaluationTiming()
	publication := e.observePublication()
	_, maintenance := e.AdmitMaintenanceTimer(context.Background())
	if awaitTimerDisposition(t, maintenance).Code != DispositionTimerApplied || e.ObserveEvaluationTiming() != timing {
		t.Fatal("maintenance timer repeated semantic-differential evaluation")
	}
	after := e.observePublication()
	if !aggregateEvaluationEqual(after.aggregateEvaluation, publication.aggregateEvaluation) || after.watermark == nil || !after.watermark.Equal(target) {
		t.Fatalf("maintenance changed market meaning before=%+v after=%+v", publication, after)
	}
	if !slices.Equal(e.ObserveTQ().Desired, evaluationOracleDesired(oracle)) {
		t.Fatalf("target T/Q desired=%v oracle=%v", e.ObserveTQ().Desired, evaluationOracleDesired(oracle))
	}

	// Advance through quiet/no-print seconds. The fence owns the one oracle
	// projection at the required target; its following timer retains T/Q/time
	// maintenance without replacing that market evaluation.
	quietTarget := target.Add(5 * time.Second)
	now = quietTarget.Add(time.Nanosecond)
	e.mu.Lock()
	for index := range e.state.binding.symbols {
		if !installExactCoverage(e.state.binding.symbols[index].aggregates, e.state.binding, start, quietTarget, nil) {
			e.mu.Unlock()
			t.Fatalf("quiet coverage index=%d", index)
		}
	}
	e.state.hydration.supportedThrough = immutableTime(quietTarget)
	for index := range e.state.binding.symbols {
		if state := e.state.binding.symbols[index].aggregates; state != nil {
			evaluateQualificationThrough(state, e.state.binding, quietTarget, now)
		}
	}
	quietOracle := e.stageAggregateEvaluationAtLocked(quietTarget, now)
	e.mu.Unlock()
	applyLiveCoverageFenceForCoalescing(t, e, 4, 3, quietTarget)
	quietPublication := e.observePublication()
	quietTiming := e.ObserveEvaluationTiming()
	if !aggregateEvaluationEqual(quietPublication.aggregateEvaluation, quietOracle) || quietPublication.watermark == nil || !quietPublication.watermark.Equal(quietTarget) ||
		!quietPublication.aggregateEvaluation.at.Equal(quietTarget) || !slices.Equal(e.ObserveTQ().Desired, evaluationOracleDesired(quietOracle)) {
		t.Fatalf("quiet coalesced/oracle mismatch publication=%+v oracle=%+v tq=%v", quietPublication, quietOracle, e.ObserveTQ().Desired)
	}
	_, maintenance = e.AdmitMaintenanceTimer(context.Background())
	if awaitTimerDisposition(t, maintenance).Code != DispositionTimerApplied || e.ObserveEvaluationTiming() != quietTiming {
		t.Fatal("quiet maintenance timer repeated aggregate evaluation")
	}

	// The final fence still evaluates its captured market target while live.
	// Its following maintenance timer owns the ordered session-end transition
	// and the required same-target terminal evaluation.
	sessionEnd := binding.SessionEnd()
	now = sessionEnd.Add(4 * time.Second)
	e.mu.Lock()
	for index := range e.state.binding.symbols {
		if !installExactCoverage(e.state.binding.symbols[index].aggregates, e.state.binding, start, sessionEnd, nil) {
			e.mu.Unlock()
			t.Fatalf("session-end coverage index=%d", index)
		}
	}
	e.state.hydration.supportedThrough = immutableTime(sessionEnd)
	for index := range e.state.binding.symbols {
		if state := e.state.binding.symbols[index].aggregates; state != nil {
			evaluateQualificationThrough(state, e.state.binding, sessionEnd, now)
		}
	}
	endFenceOracle := e.stageAggregateEvaluationAtLocked(sessionEnd, now)
	e.mu.Unlock()
	priorPublicationID := e.observePublication().publicationID
	applyLiveCoverageFenceForCoalescing(t, e, 5, 4, sessionEnd)
	endPublication := e.observePublication()
	endTiming := e.ObserveEvaluationTiming()
	if e.state.lifecycle != lifecycleLive || !aggregateEvaluationEqual(endPublication.aggregateEvaluation, endFenceOracle) ||
		endPublication.watermark == nil || !endPublication.watermark.Equal(sessionEnd) || endPublication.publicationID <= priorPublicationID ||
		!slices.Equal(e.ObserveTQ().Desired, evaluationOracleDesired(endFenceOracle)) {
		t.Fatalf("session-end fence/oracle mismatch lifecycle=%s publication=%+v oracle=%+v tq=%v", e.state.lifecycle, endPublication, endFenceOracle, e.ObserveTQ().Desired)
	}
	e.mu.Lock()
	e.state.lifecycle = lifecycleEnded
	endTimerOracle := e.stageAggregateEvaluationAtLocked(sessionEnd, now)
	e.state.lifecycle = lifecycleLive
	e.mu.Unlock()
	_, maintenance = e.AdmitMaintenanceTimer(context.Background())
	if awaitTimerDisposition(t, maintenance).Code != DispositionTimerApplied || e.state.lifecycle != lifecycleEnded ||
		e.ObserveEvaluationTiming().Starts.Timer != endTiming.Starts.Timer+1 ||
		!aggregateEvaluationEqual(e.observePublication().aggregateEvaluation, endTimerOracle) ||
		!slices.Equal(e.ObserveTQ().Desired, evaluationOracleDesired(endTimerOracle)) {
		t.Fatalf("session-end timer/oracle mismatch lifecycle=%s publication=%+v oracle=%+v timing=%+v tq=%v", e.state.lifecycle, e.observePublication(), endTimerOracle, e.ObserveEvaluationTiming(), e.ObserveTQ().Desired)
	}
}

func evaluationOracleDesired(result aggregateEvaluationResult) []string {
	if result.mode != rankingQualifiedCurrent {
		return nil
	}
	desired := make([]string, 0, min(len(result.rows), maximumTQSymbols))
	for _, row := range result.rows {
		if len(desired) == maximumTQSymbols {
			break
		}
		desired = append(desired, row.symbol)
	}
	return desired
}
