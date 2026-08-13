package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestFenceAttributionPartitionsProductionBoundary(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(2 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	defer closeAndWait(t, e)

	terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	terminalResult := admitHydrationTerminal(t, e, terminal)

	var timingMu sync.Mutex
	timingNow := time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC)
	e.ArmEvaluationTimingForTest(func() time.Time {
		timingMu.Lock()
		defer timingMu.Unlock()
		result := timingNow
		timingNow = timingNow.Add(time.Millisecond)
		return result
	})
	now = now.Add(5 * time.Second)
	fence, err := NewAggregateIngressFenceInput(terminalResult.FenceCommand, AggregateIngressFenceComplete, 7, 3, now)
	if err != nil {
		t.Fatal(err)
	}
	_, completion := e.AdmitAggregateIngressFence(context.Background(), fence)
	disposition := awaitHydrationDisposition(t, completion)
	if disposition.Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence=%+v", disposition)
	}
	timing := e.ObserveFenceTiming()
	parts := timing.CoverageFinalization + timing.SymbolMaintenance + timing.EvaluationStage + timing.EvaluationApply + timing.Publication + timing.ResidualOrderedOverhead
	if !timing.Valid || timing.InvalidReason != "" || timing.EngineSequence != disposition.EngineSequence || timing.HydrationGeneration != terminalResult.FenceCommand.Generation() ||
		timing.ConnectionEpoch != 1 || timing.ThroughFrameSequence != 7 || timing.MarkerOrdinal != 3 || timing.CommandToken != terminalResult.FenceCommand.CommandToken() ||
		timing.Target != now || timing.PublicationID == 0 || timing.Total <= 0 || timing.Total != parts {
		t.Fatalf("fence timing=%+v parts=%s", timing, parts)
	}
}

func TestFenceDeliveryStartRetainsMonotonicClock(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(2 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	defer closeAndWait(t, e)
	terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	command := admitHydrationTerminal(t, e, terminal).FenceCommand
	start := time.Now()
	input, err := NewAggregateIngressFenceInputAtDelivery(command, AggregateIngressFenceComplete, 1, 1, time.Now().UTC(), start)
	if err != nil || input.deliveryStartedAt != start || !input.deliveryStartedAt.Before(time.Now()) {
		t.Fatalf("delivery start=%v err=%v", input.deliveryStartedAt, err)
	}
}
