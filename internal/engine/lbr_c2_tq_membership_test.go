package engine

import (
	"testing"
	"time"
)

// TestPLBRC2TQMembership is the engine-owned portion of the Capability C2
// primary proof. Adapter raw-boundary and mixed-frame classification are
// covered at their direct seams in internal/massive.
func TestPLBRC2TQMembership(t *testing.T) {
	e, _, clock, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	rows := func(symbols ...string) aggregateEvaluationResult {
		result := aggregateEvaluationResult{mode: rankingQualifiedCurrent}
		for index, symbol := range symbols {
			result.rows = append(result.rows, aggregateRankingRow{rank: uint32(index + 1), symbol: symbol, tqIntentEligible: true})
		}
		return result
	}
	applyResult := func(input TQCommandResultInput, at time.Time) DispositionCode {
		e.mu.Lock()
		code, _ := e.applyTQCommandResultLocked(&queueNode{tqCommandResult: frozenTQCommandResultInput{input}, admissionTime: at})
		e.reconcileTQLocked(at, false)
		e.mu.Unlock()
		return code
	}

	e.mu.Lock()
	e.state.tq = tqState{epoch: 1, nextToken: 1, nextGeneration: 1, members: make(map[string]*tqSymbolState), pressure: tqPressureState{mode: TQPressureNormal, nextSequence: 1}}
	e.state.aggregateEvaluator.current = rows("AAA", "MISSING")
	e.reconcileTQLocked(start, true)
	e.mu.Unlock()
	fresh := issueTQForTest(t, e)
	if fresh.Action() != TQSubscribe || !equalTQSymbols(fresh.Symbols(), []string{"AAA", "MISSING"}) {
		t.Fatalf("fresh full sorted batch = %s %v", fresh.Action(), fresh.Symbols())
	}
	if got := applyResult(tqResultForTest(t, fresh, LivePosition{ConnectionEpoch: 1, FrameSequence: 10}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("fresh write = %s", got)
	}

	// Rank loss closes AAA immediately. A symbol selected after the last cadence
	// cannot be added as a continuation of that old cadence.
	e.mu.Lock()
	e.state.aggregateEvaluator.current = rows("MISSING", "NEW")
	e.reconcileTQLocked(start.Add(100*time.Millisecond), false)
	if member := e.state.tq.members["AAA"]; member == nil || member.tradeCoverage.active || member.quoteCoverage.active {
		e.mu.Unlock()
		t.Fatal("rank removal did not close local coverage")
	}
	e.mu.Unlock()
	remove := issueTQForTest(t, e)
	if remove.Action() != TQUnsubscribe || !equalTQSymbols(remove.Symbols(), []string{"AAA"}) {
		t.Fatalf("batched removal = %s %v", remove.Action(), remove.Symbols())
	}
	if got := applyResult(tqResultForTest(t, remove, LivePosition{ConnectionEpoch: 1, FrameSequence: 11}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("removal write = %s", got)
	}
	if _, err := e.IssueTQCommand(); err == nil {
		t.Fatal("post-cadence rank addition escaped before the next cadence")
	}
	e.mu.Lock()
	e.reconcileTQLocked(start.Add(time.Second), true)
	e.mu.Unlock()
	addition := issueTQForTest(t, e)
	if addition.Action() != TQSubscribe || !equalTQSymbols(addition.Symbols(), []string{"NEW"}) {
		t.Fatalf("cadence addition = %s %v", addition.Action(), addition.Symbols())
	}
	if got := applyResult(tqResultForTest(t, addition, LivePosition{ConnectionEpoch: 1, FrameSequence: 12}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("addition write = %s", got)
	}

	// One ordinary cadence owns both phases of its diff: the sorted removal is
	// written first, then the additions batch becomes issuable only after that
	// write result.
	e.mu.Lock()
	e.state.aggregateEvaluator.current = rows("AAA", "MISSING")
	e.reconcileTQLocked(start.Add(2*time.Second), true)
	e.mu.Unlock()
	phaseOne := issueTQForTest(t, e)
	if phaseOne.Action() != TQUnsubscribe || !equalTQSymbols(phaseOne.Symbols(), []string{"NEW"}) {
		t.Fatalf("ordinary diff removal phase = %s %v", phaseOne.Action(), phaseOne.Symbols())
	}
	if got := applyResult(tqResultForTest(t, phaseOne, LivePosition{ConnectionEpoch: 1, FrameSequence: 13}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("ordinary removal result = %s", got)
	}
	phaseTwo := issueTQForTest(t, e)
	if phaseTwo.Action() != TQSubscribe || !equalTQSymbols(phaseTwo.Symbols(), []string{"AAA"}) {
		t.Fatalf("ordinary diff addition phase = %s %v", phaseTwo.Action(), phaseTwo.Symbols())
	}
	if got := applyResult(tqResultForTest(t, phaseTwo, LivePosition{ConnectionEpoch: 1, FrameSequence: 14}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("ordinary addition result = %s", got)
	}
	e.mu.Lock()
	preShedGeneration := e.state.tq.members["AAA"].generation
	e.mu.Unlock()

	// Aggregate-only pressure removes the complete provider membership in one
	// batch. Five healthy samples restore policy, but additions wait for the
	// next combined cadence and then advance one highest-ranked symbol.
	e.mu.Lock()
	e.setTQPressureModeLocked(TQPressureAggregateOnly, start.Add(3*time.Second), TQPressureCauseWaitingFrames)
	e.reconcileTQLocked(start.Add(3*time.Second), false)
	e.mu.Unlock()
	shed := issueTQForTest(t, e)
	if shed.Action() != TQUnsubscribe || !equalTQSymbols(shed.Symbols(), []string{"AAA", "MISSING"}) {
		t.Fatalf("aggregate-only membership removal = %s %v", shed.Action(), shed.Symbols())
	}
	if got := applyResult(tqResultForTest(t, shed, LivePosition{ConnectionEpoch: 1, FrameSequence: 15}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("aggregate-only removal write = %s", got)
	}
	if view := e.ObserveTQ(); view.Accounting.KnownPresent != 0 || view.Accounting.Unknown != 0 || view.CommandPending {
		t.Fatalf("aggregate-only membership did not reach zero: %+v", view)
	}
	healthy := healthyTQPressureSample()
	for second := 4; second < 9; second++ {
		at := start.Add(time.Duration(second) * time.Second)
		clock.Store(at.UnixNano())
		e.mu.Lock()
		e.applyTQPressureSampleLocked(healthy, at)
		e.reconcileTQLocked(at, false)
		e.mu.Unlock()
	}
	if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.CommandPending {
		t.Fatalf("recovery issued additions outside combined cadence: %+v", view)
	}
	e.mu.Lock()
	e.reconcileTQLocked(start.Add(9*time.Second), true)
	e.mu.Unlock()
	restore := issueTQForTest(t, e)
	if restore.Action() != TQSubscribe || !equalTQSymbols(restore.Symbols(), []string{"AAA"}) {
		t.Fatalf("first ranked restoration = %s %v", restore.Action(), restore.Symbols())
	}
	if got := applyResult(tqResultForTest(t, restore, LivePosition{ConnectionEpoch: 1, FrameSequence: 16}, start, ControlSucceeded), start); got != DispositionTQApplied {
		t.Fatalf("first restoration write = %s", got)
	}
	e.mu.Lock()
	restoredGeneration := e.state.tq.members["AAA"].generation
	e.mu.Unlock()
	if restoredGeneration <= preShedGeneration {
		t.Fatalf("restored symbol reused generation: before=%d after=%d", preShedGeneration, restoredGeneration)
	}
	if _, err := e.IssueTQCommand(); err == nil {
		t.Fatal("restoration issued more than one addition per cadence")
	}
	e.mu.Lock()
	e.reconcileTQLocked(start.Add(10*time.Second), true)
	e.mu.Unlock()
	failedAddition := issueTQForTest(t, e)
	if !equalTQSymbols(failedAddition.Symbols(), []string{"MISSING"}) {
		t.Fatalf("second restoration request = %v", failedAddition.Symbols())
	}
	if got := applyResult(tqResultForTest(t, failedAddition, LivePosition{}, start, ControlFailed), start); got != DispositionTQRejected {
		t.Fatalf("failed write = %s", got)
	}
	if view := e.ObserveTQ(); !view.ControlClosed || view.AggregateOnly || view.CommandPending || view.Rows[0].Tape.Reason != "control_error" {
		t.Fatalf("write failure escaped T/Q-only containment: %+v", view)
	}
}
