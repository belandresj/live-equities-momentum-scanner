package engine

import (
	"context"
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
		if correctedMark == nil || correctedMark.close != 12 {
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
		if advancedMark == nil || advancedMark.close != futureClose {
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
