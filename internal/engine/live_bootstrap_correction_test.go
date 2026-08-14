package engine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestLiveBootstrapIntegritySmallInterleaving is the first rung of
// P-LIVE-BOOTSTRAP-INTEGRITY. It exercises the real plan/chunk/terminal/fence
// path with live evidence admitted while hydration remains active.
func TestLiveBootstrapIntegritySmallInterleaving(t *testing.T) {
	symbols := []string{"S0000", "S0001", "S0002", "S0003", "S0004", "S0005"}
	binding := hydrationPopulationBinding(t, symbols)
	start := binding.SessionStart()
	now := start.Add(10 * time.Second)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.delay = 4 * time.Second
	e.mu.Unlock()
	plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	requests := plan.Plan.Requests()
	if len(requests) != len(symbols) {
		t.Fatalf("planned=%d want=%d", len(requests), len(symbols))
	}

	// A live row that exactly repeats later live evidence, a REST/live unequal
	// overlap, and a live-only tail all arrive before REST becomes terminal.
	now = start.Add(12 * time.Second)
	firstLive := liveAggregate(binding, symbols[0], start.Add(time.Second), 1, 2)
	admission, completion := e.AdmitAggregate(context.Background(), firstLive)
	if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateInserted {
		t.Fatal("first live overlay was not installed")
	}
	duplicate := firstLive
	duplicate.Live = LivePosition{ConnectionEpoch: 1, FrameSequence: 3}
	admission, completion = e.AdmitAggregate(context.Background(), duplicate)
	if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateExactDuplicate {
		t.Fatal("equal live evidence did not deduplicate")
	}
	unequal := liveAggregate(binding, symbols[1], start.Add(2*time.Second), 1, 4)
	unequal.Values.Close, unequal.Values.Open, unequal.Values.High, unequal.Values.Low, unequal.Values.VWAP = 11, 11, 11, 11, 11
	admission, completion = e.AdmitAggregate(context.Background(), unequal)
	if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateInserted {
		t.Fatal("unequal live overlay was not installed")
	}
	tail := liveAggregate(binding, symbols[2], start.Add(10*time.Second), 1, 5)
	admission, completion = e.AdmitAggregate(context.Background(), tail)
	if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateInserted {
		t.Fatal("live tail was not installed")
	}

	var fence HydrationFenceCommand
	for index, token := range requests {
		rows := []HydrationRow(nil)
		switch index {
		case 0:
			rows = []HydrationRow{hydrationRow(t, symbols[index], start.Add(time.Second), 10)}
		case 1:
			rows = []HydrationRow{hydrationRow(t, symbols[index], start.Add(2*time.Second), 10)}
		case 3:
			rows = []HydrationRow{hydrationRow(t, symbols[index], start.Add(3*time.Second), 13)}
		}
		if len(rows) > 0 {
			chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, int64(len(rows)), rows)
			if err != nil {
				t.Fatal(err)
			}
			result := admitHydrationChunk(t, e, chunk)
			if result.Code != DispositionHydrationChunkApplied {
				t.Fatalf("chunk[%d]=%+v", index, result)
			}
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 100, int64(len(rows)), 1, int64(len(rows)))
			if err != nil {
				t.Fatal(err)
			}
			fence = admitHydrationTerminal(t, e, terminal).FenceCommand
		} else {
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			fence = admitHydrationTerminal(t, e, terminal).FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("terminal work did not allocate a fence")
	}
	now = start.Add(15 * time.Second)
	fact, err := NewAggregateIngressFenceInput(fence, AggregateIngressFenceComplete, 5, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fact)
	result := awaitHydrationDisposition(t, fenceCompletion)
	if result.Code == DispositionAccountingIntegrity {
		t.Fatalf("small interleaving reproduced %s: %+v", formatIntegrity(e.state.evaluatorIntegrity), result)
	}
	if result.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive || e.state.committedT == nil || !e.state.committedT.Equal(start.Add(11*time.Second)) {
		t.Fatalf("fence=%+v lifecycle=%s committed=%v diagnostic=%s", result, e.state.lifecycle, e.state.committedT, formatIntegrity(e.state.evaluatorIntegrity))
	}
	if e.state.evaluatorIntegrity != nil {
		t.Fatalf("valid completion latched diagnostic: %s", formatIntegrity(e.state.evaluatorIntegrity))
	}
	a := e.state.hydration.generation.accounting
	if a != (HydrationAccounting{Planned: 6, CompletedValue: 3, CompletedEmpty: 3}) || !a.reconciles() || !e.state.hydration.generation.rowAccounting.reconciles() {
		t.Fatalf("terminal accounting=%+v rows=%+v", a, e.state.hydration.generation.rowAccounting)
	}
}

// TestLiveBootstrapIntegrityObservedShape is the second diagnostic rung. It is
// explicitly selected because it recreates the observed population/work/row
// bins and should not enlarge ordinary repository verification.
func TestLiveBootstrapIntegrityObservedShape(t *testing.T) {
	if testing.Short() {
		t.Skip("observed-shape bootstrap diagnostic is not ordinary verification")
	}
	const (
		population = 5_540
		values     = 1_742
		rowsTotal  = 68_154
		conflicts  = 24
	)
	symbols := make([]string, population)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%04d", index)
	}
	binding := hydrationPopulationBinding(t, symbols)
	start := binding.SessionStart()
	now := start.Add(60 * time.Second)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.delay = 4 * time.Second
	e.mu.Unlock()
	budgets := generousHydrationBudgets()
	budgets.MaximumResponseBytes = 2 << 20
	budgets.MaximumNormalizedRecords = population * 60
	budgets.MaximumResidentRecords = population * 60
	plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, budgets)
	requests := plan.Plan.Requests()
	if len(requests) != population {
		t.Fatalf("planned=%d want=%d", len(requests), population)
	}

	now = start.Add(64 * time.Second)
	for index := 0; index < conflicts; index++ {
		live := liveAggregate(binding, symbols[index], start.Add(time.Duration(index%39)*time.Second), 1, uint64(index+2))
		live.Values.Open, live.Values.High, live.Values.Low, live.Values.Close, live.Values.VWAP = 20, 20, 20, 20, 20
		admission, completion := e.AdmitAggregate(context.Background(), live)
		if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateInserted {
			t.Fatalf("live conflict seed[%d] failed", index)
		}
	}
	// One post-handoff live-only identity distinguishes the final fence from
	// REST completion and makes throughFrameSequence exact.
	tail := liveAggregate(binding, symbols[values], start.Add(60*time.Second), 1, conflicts+2)
	admission, completion := e.AdmitAggregate(context.Background(), tail)
	if admission != AdmissionAdmitted || awaitAggregateDisposition(t, completion).Code != DispositionAggregateInserted {
		t.Fatal("live tail failed")
	}

	remainingRows := rowsTotal
	var fence HydrationFenceCommand
	for index, token := range requests {
		if index < values {
			rowCount := rowsTotal / values
			if index < rowsTotal%values {
				rowCount++
			}
			rows := make([]HydrationRow, rowCount)
			for rowIndex := range rows {
				rows[rowIndex] = hydrationRow(t, symbols[index], start.Add(time.Duration(rowIndex)*time.Second), 10+float64(index%7)/10)
			}
			chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, int64(rowCount), rows)
			if err != nil {
				t.Fatal(err)
			}
			if result := admitHydrationChunk(t, e, chunk); result.Code != DispositionHydrationChunkApplied {
				t.Fatalf("chunk[%d]=%+v diagnostic=%s", index, result, formatIntegrity(e.state.evaluatorIntegrity))
			}
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 100, int64(rowCount), 1, int64(rowCount))
			if err != nil {
				t.Fatal(err)
			}
			fence = admitHydrationTerminal(t, e, terminal).FenceCommand
			remainingRows -= rowCount
		} else {
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			fence = admitHydrationTerminal(t, e, terminal).FenceCommand
		}
	}
	if remainingRows != 0 || fence.CommandToken() == 0 {
		t.Fatalf("shape construction rows_remaining=%d fence=%+v", remainingRows, fence)
	}
	now = start.Add(65 * time.Second)
	fact, err := NewAggregateIngressFenceInput(fence, AggregateIngressFenceComplete, conflicts+2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fact)
	var result HydrationDisposition
	select {
	case result = <-fenceCompletion:
	case <-time.After(2 * time.Minute):
		t.Fatal("timed out waiting for observed-shape fence evaluation")
	}
	if result.Code == DispositionAccountingIntegrity {
		t.Fatalf("observed shape reproduced %s: %+v", formatIntegrity(e.state.evaluatorIntegrity), result)
	}
	if result.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive || e.state.evaluatorIntegrity != nil {
		t.Fatalf("fence=%+v lifecycle=%s diagnostic=%s", result, e.state.lifecycle, formatIntegrity(e.state.evaluatorIntegrity))
	}
	work, rows := e.state.hydration.generation.accounting, e.state.hydration.generation.rowAccounting
	if work != (HydrationAccounting{Planned: population, CompletedValue: values, CompletedEmpty: population - values}) ||
		rows.Consumed != rowsTotal || rows.ConflictOrWithdrawal != conflicts || !work.reconciles() || !rows.reconciles() {
		t.Fatalf("work=%+v rows=%+v", work, rows)
	}
}

func formatIntegrity(value *EvaluatorIntegrityView) string {
	if value == nil {
		return "none"
	}
	return fmt.Sprintf("category=%s input=%s sequence=%d candidate=%s expected=%s field=%s symbol=%s reason=%s population=%d/%d/%d/%d/%d/%d/%d",
		value.Category, value.InputKind, value.EngineSequence, value.CandidateTime.Format(time.RFC3339), value.ExpectedTime.Format(time.RFC3339),
		value.FirstField, value.FirstSymbol, value.FirstReason, value.UniverseTotal, value.TrustedRankableMark, value.TrustedBelowPriceMark,
		value.NoPrintThroughT, value.InvalidMark, value.UnknownDueFailureOrFence, value.QualificationUnresolved)
}
