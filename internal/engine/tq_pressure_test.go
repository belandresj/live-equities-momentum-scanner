package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func TestPC9PressureTransitionsExpiryAndRankedRestoration(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	e.mu.Lock()
	aggregateBefore := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	committedBefore := *e.state.committedT
	e.mu.Unlock()
	healthy := TQPressureSample{QueueCapacityFrames: 100, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}
	unhealthy := healthy
	unhealthy.QueueCurrentFrames = 50

	applyPressureSample(t, e, clockNanos, start, unhealthy)
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("first unhealthy sample bypassed dwell: %s", got)
	}
	applyPressureSample(t, e, clockNanos, start.Add(500*time.Millisecond), unhealthy)
	view := e.ObserveTQ()
	if view.Pressure != TQPressureDegraded || !view.ShedTradesQuotes || view.Rows[0].TradeCoverage || view.Rows[0].Tape.Status != TQPressureShed || view.CommandPending {
		t.Fatalf("degraded containment = %+v", view)
	}
	applyPressureSample(t, e, clockNanos, start.Add(2*time.Second), unhealthy)
	view = e.ObserveTQ()
	if view.Pressure != TQPressureAggregateOnly || !view.AggregateOnly || !view.CommandPending || view.PendingAction != TQUnsubscribe || view.PendingSymbol != "MISSING" {
		t.Fatalf("aggregate-only removal order = %+v", view)
	}

	position := uint64(100)
	for _, symbol := range []string{"MISSING", "AAA"} {
		command := issueTQForTest(t, e)
		if command.Symbol() != symbol || command.Action() != TQUnsubscribe {
			t.Fatalf("removal %s = %s/%s", symbol, command.Symbol(), command.Action())
		}
		result := tqResultForTest(t, command, LivePosition{ConnectionEpoch: 1, FrameSequence: position}, start.Add(2*time.Second), ControlSucceeded)
		position++
		if got := admitTQResultForTest(t, e, result); got.Code != DispositionTQApplied {
			t.Fatalf("removal result = %+v", got)
		}
	}
	if view = e.ObserveTQ(); view.Accounting.KnownPresent != 0 || view.CommandPending {
		t.Fatalf("provider membership not zero = %+v", view)
	}

	applyPressureSample(t, e, clockNanos, start.Add(3*time.Second), healthy)
	if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
		t.Fatalf("early healthy sample recovered: %s", got)
	}
	applyPressureSample(t, e, clockNanos, start.Add(33*time.Second), healthy)
	if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
		t.Fatalf("sampling blackout manufactured recovery: %s", got)
	}
	for second := 34; second <= 63; second++ {
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), healthy)
	}
	view = e.ObserveTQ()
	if view.Pressure != TQPressureNormal || view.AggregateOnly || !view.CommandPending || view.PendingAction != TQSubscribe || view.PendingSymbol != "AAA" {
		t.Fatalf("ranked recovery start = %+v", view)
	}
	addAAA := issueTQForTest(t, e)
	if got := admitTQResultForTest(t, e, tqResultForTest(t, addAAA, LivePosition{ConnectionEpoch: 1, FrameSequence: position}, start.Add(63*time.Second), ControlSucceeded)); got.Code != DispositionTQApplied {
		t.Fatalf("AAA restoration = %+v", got)
	}
	view = e.ObserveTQ()
	if view.CommandPending || !view.Rows[0].TradeCoverage || view.Rows[1].TradeCoverage || view.Rows[0].Tape.Status != TQWarming || view.Rows[0].Spread.Status != TQWarming {
		t.Fatalf("restoration bypassed five-second gate = %+v", view)
	}
	clockNanos.Store(start.Add(68 * time.Second).UnixNano())
	e.mu.Lock()
	e.reconcileTQLocked(start.Add(68 * time.Second))
	e.mu.Unlock()
	view = e.ObserveTQ()
	if !view.CommandPending || view.PendingAction != TQSubscribe || view.PendingSymbol != "MISSING" {
		t.Fatalf("second ranked restoration = %+v", view)
	}
	if view.Accounting.Consumed != view.Accounting.Applied+view.Accounting.Duplicate+view.Accounting.Rejected+view.Accounting.Fenced+view.Accounting.PressureShed+view.Accounting.Integrity {
		t.Fatalf("T/Q accounting = %+v", view.Accounting)
	}
	if view.Commands.Issued != view.Commands.Pending+view.Commands.Acknowledged+view.Commands.Failed+view.Commands.Fenced {
		t.Fatalf("command accounting = %+v", view.Commands)
	}
	e.mu.Lock()
	if !aggregateEvaluationEqual(aggregateBefore, e.state.aggregateEvaluator.current) || e.state.committedT == nil || !e.state.committedT.Equal(committedBefore) {
		t.Fatalf("pressure changed aggregate state: before=%+v after=%+v", aggregateBefore, e.state.aggregateEvaluator.current)
	}
	e.mu.Unlock()
}

func TestPC9CurrentHostHeapGateBoundaries(t *testing.T) {
	policy := defaultTQPressurePolicy()
	got := [3]uint64{policy.recoveryHeap, policy.degradedHeap, policy.aggregateHeap}
	want := [3]uint64{2560 << 20, 3328 << 20, 4096 << 20}
	if got != want || tqRecoveryHeapBytes != want[0] || tqDegradedHeapBytes != want[1] || tqAggregateHeapBytes != want[2] ||
		!(policy.recoveryHeap < policy.degradedHeap && policy.degradedHeap < policy.aggregateHeap) {
		t.Fatalf("current-host heap policy = recovery=%d degraded=%d aggregate=%d", policy.recoveryHeap, policy.degradedHeap, policy.aggregateHeap)
	}

	t.Run("observed aggregate baseline leaves TQ normal", func(t *testing.T) {
		e, _, clockNanos, start := pressureProofEngine(t)
		defer closeAndWait(t, e)
		sample := TQPressureSample{QueueCapacityFrames: 100, HeapAllocBytes: 1_785_959_440, TQLocalAccountingHealthy: true, Goroutines: 1}
		applyPressureSample(t, e, clockNanos, start, sample)
		applyPressureSample(t, e, clockNanos, start.Add(time.Second), sample)
		applyPressureSample(t, e, clockNanos, start.Add(2*time.Second), sample)
		if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.ShedTradesQuotes {
			t.Fatalf("observed baseline shed T/Q = %+v", view)
		}
	})

	t.Run("degraded heap gate retains dwell boundary", func(t *testing.T) {
		e, _, clockNanos, start := pressureProofEngine(t)
		defer closeAndWait(t, e)
		sample := TQPressureSample{QueueCapacityFrames: 100, HeapAllocBytes: tqDegradedHeapBytes, TQLocalAccountingHealthy: true, Goroutines: 1}
		applyPressureSample(t, e, clockNanos, start, sample)
		if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
			t.Fatalf("degraded heap bypassed dwell = %s", got)
		}
		applyPressureSample(t, e, clockNanos, start.Add(500*time.Millisecond), sample)
		if got := e.ObserveTQ().Pressure; got != TQPressureDegraded {
			t.Fatalf("degraded heap boundary = %s", got)
		}
	})

	t.Run("aggregate heap gate remains immediate", func(t *testing.T) {
		e, _, clockNanos, start := pressureProofEngine(t)
		defer closeAndWait(t, e)
		sample := TQPressureSample{QueueCapacityFrames: 100, HeapAllocBytes: tqAggregateHeapBytes, TQLocalAccountingHealthy: true, Goroutines: 1}
		applyPressureSample(t, e, clockNanos, start, sample)
		if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || !view.AggregateOnly {
			t.Fatalf("aggregate heap boundary = %+v", view)
		}
	})

	t.Run("recovery heap gate remains strict", func(t *testing.T) {
		e, _, clockNanos, start := pressureProofEngine(t)
		defer closeAndWait(t, e)
		e.mu.Lock()
		e.state.tq.members, e.state.tq.desired = nil, nil
		e.state.tq.pressure.mode = TQPressureAggregateOnly
		e.state.tq.aggregateOnly = true
		e.state.tq.pressure.degradedAt = start.Add(-time.Minute)
		e.mu.Unlock()
		atGate := TQPressureSample{QueueCapacityFrames: 100, DeliveryLatencyAttributed: true, HeapAllocBytes: tqRecoveryHeapBytes, TQLocalAccountingHealthy: true, Goroutines: 1}
		for second := 0; second <= 30; second++ {
			applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), atGate)
		}
		if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
			t.Fatalf("recovered at non-strict heap boundary = %s", got)
		}
		belowGate := atGate
		belowGate.HeapAllocBytes--
		for second := 31; second <= 61; second++ {
			applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), belowGate)
		}
		if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
			t.Fatalf("did not recover below heap boundary = %s", got)
		}
	})
}

func TestPC9RecoveryDeliveryGateBoundariesAndAttribution(t *testing.T) {
	policy := defaultTQPressurePolicy()
	if policy.degradedDelivery != 2*time.Second || policy.aggregateDelivery != 5*time.Second || policy.recoveryDelivery != time.Second ||
		policy.degradedDwell != 500*time.Millisecond || policy.aggregateDwell != 2*time.Second || policy.recoveryDwell != 30*time.Second ||
		policy.degradedQueuePercent != 50 || policy.aggregateQueuePercent != 80 || policy.recoveryQueuePercent != 20 ||
		policy.degradedOldest != 250*time.Millisecond || policy.aggregateOldest != 1500*time.Millisecond || policy.recoveryOldest != 100*time.Millisecond ||
		policy.degradedHeap != tqDegradedHeapBytes || policy.aggregateHeap != tqAggregateHeapBytes || policy.recoveryHeap != tqRecoveryHeapBytes ||
		policy.degradedGoroutines != 64 || policy.aggregateGoroutines != 128 || policy.recoveryGoroutines != 48 {
		t.Fatalf("pressure policy changed outside recovery delivery: %+v", policy)
	}

	newRecoveringEngine := func(t *testing.T) (*Engine, *atomic.Int64, time.Time) {
		t.Helper()
		e, _, clockNanos, start := pressureProofEngine(t)
		e.mu.Lock()
		e.state.tq.members, e.state.tq.desired = nil, nil
		e.state.tq.pressure.mode = TQPressureAggregateOnly
		e.state.tq.aggregateOnly = true
		e.state.tq.pressure.degradedAt = start.Add(-time.Minute)
		e.mu.Unlock()
		return e, clockNanos, start
	}
	healthy := TQPressureSample{QueueCapacityFrames: 100, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}

	t.Run("equal one second cannot recover; one nanosecond below can", func(t *testing.T) {
		e, clockNanos, start := newRecoveringEngine(t)
		defer closeAndWait(t, e)
		atGate := healthy
		atGate.MaxDeliveryDelayOneSec = time.Second
		for second := 0; second <= 30; second++ {
			applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), atGate)
		}
		if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
			t.Fatalf("recovered at non-strict delivery boundary = %s", got)
		}

		belowGate := atGate
		belowGate.MaxDeliveryDelayOneSec = time.Second - time.Nanosecond
		for second := 31; second <= 61; second++ {
			applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), belowGate)
		}
		if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
			t.Fatalf("did not recover below delivery boundary = %s", got)
		}
	})

	t.Run("unattributed samples cannot establish recovery dwell", func(t *testing.T) {
		e, clockNanos, start := newRecoveringEngine(t)
		defer closeAndWait(t, e)
		unattributed := healthy
		unattributed.DeliveryLatencyAttributed = false
		unattributed.MaxDeliveryDelayOneSec = time.Second - time.Nanosecond
		for second := 0; second <= 30; second++ {
			applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), unattributed)
		}
		if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
			t.Fatalf("unattributed samples manufactured recovery = %s", got)
		}
	})

	t.Run("delayed attributed result is fenced before recovery", func(t *testing.T) {
		e, clockNanos, start := newRecoveringEngine(t)
		defer closeAndWait(t, e)
		e.mu.Lock()
		e.advanceTQPressureTimerLocked(start)
		e.mu.Unlock()
		command := issuePressureForTest(t, e)
		input, err := NewTQPressureResultInput(command, healthy)
		if err != nil {
			t.Fatal(err)
		}
		clockNanos.Store(start.Add(2*time.Second + time.Nanosecond).UnixNano())
		if got := admitPressureForTest(t, e, input); got.Code != DispositionTQFenced {
			t.Fatalf("delayed attributed result = %+v", got)
		}
		e.mu.Lock()
		recoverySince := e.state.tq.pressure.recoverySince
		e.mu.Unlock()
		if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly || !recoverySince.IsZero() {
			t.Fatalf("delayed result advanced recovery: pressure=%s recovery_since=%s", got, recoverySince)
		}
	})
}

func TestPC9PressureMissingAndLateResultsCannotLeaveNormal(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	healthy := TQPressureSample{QueueCapacityFrames: 100, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}

	e.mu.Lock()
	e.advanceTQPressureTimerLocked(start)
	e.mu.Unlock()
	first := issuePressureForTest(t, e)
	clockNanos.Store(start.Add(2 * time.Second).UnixNano())
	e.mu.Lock()
	e.advanceTQPressureTimerLocked(start.Add(2 * time.Second))
	e.mu.Unlock()
	if view := e.ObserveTQ(); view.Pressure != TQPressureDegraded || view.PressureMisses != 1 {
		t.Fatalf("first expiry = %+v", view)
	}
	second := issuePressureForTest(t, e)
	lateFirst, _ := NewTQPressureResultInput(first, healthy)
	if got := admitPressureForTest(t, e, lateFirst); got.Code != DispositionTQFenced || e.ObserveTQ().Pressure != TQPressureDegraded {
		t.Fatalf("late healthy first result = %+v view=%+v", got, e.ObserveTQ())
	}
	clockNanos.Store(start.Add(4100 * time.Millisecond).UnixNano())
	lateSecond, _ := NewTQPressureResultInput(second, healthy)
	if got := admitPressureForTest(t, e, lateSecond); got.Code != DispositionTQFenced {
		t.Fatalf("late second result = %+v", got)
	}
	view := e.ObserveTQ()
	if view.Pressure != TQPressureAggregateOnly || view.PressureMisses != 2 || !view.ShedTradesQuotes || view.PressureFenced < 2 {
		t.Fatalf("second expiry false normal = %+v", view)
	}
}

func TestPC9PressureTQLocalAccountingFailureIsImmediate(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	sample := TQPressureSample{QueueCapacityFrames: 100, TQLocalAccountingHealthy: false, Goroutines: 1}
	applyPressureSample(t, e, clockNanos, start, sample)
	if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || !view.AggregateOnly {
		t.Fatalf("T/Q-local accounting failure = %+v", view)
	}
}

func TestPC9PressureEpochReplacementPreservesMonotonicAuthority(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	healthy := TQPressureSample{QueueCapacityFrames: 100, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}

	e.mu.Lock()
	e.state.tq.consumed, e.state.tq.applied, e.state.tq.duplicate = 6, 1, 1
	e.state.tq.rejected, e.state.tq.fenced, e.state.tq.pressureShed, e.state.tq.integrity = 1, 1, 1, 1
	e.state.tq.commandsIssued, e.state.tq.commandsAcknowledged, e.state.tq.commandsFailed, e.state.tq.commandsFenced = 3, 1, 1, 1
	e.state.tq.commandResultsFenced = 1
	e.state.tq.pressure.mode, e.state.tq.pressure.transitions, e.state.tq.pressure.fenced = TQPressureDegraded, 2, 1
	e.advanceTQPressureTimerLocked(start)
	e.issueTQCommandLocked(TQUnsubscribe, "AAA")
	e.mu.Unlock()
	oldPressure := issuePressureForTest(t, e)
	oldPressureResult, err := NewTQPressureResultInput(oldPressure, healthy)
	if err != nil {
		t.Fatal(err)
	}
	oldTQ := issueTQForTest(t, e)
	oldTQResult := tqResultForTest(t, oldTQ, LivePosition{ConnectionEpoch: 1, FrameSequence: 100}, start, ControlSucceeded)
	before := e.ObserveTQ()
	if before.Commands.Issued != 4 || before.Commands.Pending != 1 || before.PressureTransitions != 2 || before.PressureFenced != 1 || before.Accounting.PressureShed != 1 ||
		before.Accounting.Consumed != before.Accounting.Applied+before.Accounting.Duplicate+before.Accounting.Rejected+before.Accounting.Fenced+before.Accounting.PressureShed+before.Accounting.Integrity {
		t.Fatalf("pre-replacement command accounting = %+v", before.Commands)
	}

	clockNanos.Store(start.Add(time.Second).UnixNano())
	e.mu.Lock()
	e.state.liveEpoch = 2
	e.state.aggregateEvaluator.current = pressureQualifiedEvaluation(start)
	e.state.aggregateEvaluator.current.mode = rankingStale
	e.state.aggregateEvaluator.current.rows = nil
	e.reconcileTQLocked(start.Add(time.Second))
	e.advanceTQPressureTimerLocked(start.Add(time.Second))
	e.mu.Unlock()
	newPressure := issuePressureForTest(t, e)
	if newPressure.Sequence() <= oldPressure.Sequence() {
		t.Fatalf("pressure sequence regressed across epoch: old=%d new=%d", oldPressure.Sequence(), newPressure.Sequence())
	}
	afterReplacement := e.ObserveTQ()
	if afterReplacement.Pressure != before.Pressure || afterReplacement.PressureTransitions != before.PressureTransitions || afterReplacement.PressureFenced != before.PressureFenced ||
		afterReplacement.Accounting.Consumed != before.Accounting.Consumed || afterReplacement.Accounting.Applied != before.Accounting.Applied ||
		afterReplacement.Accounting.Duplicate != before.Accounting.Duplicate || afterReplacement.Accounting.Rejected != before.Accounting.Rejected ||
		afterReplacement.Accounting.Fenced != before.Accounting.Fenced || afterReplacement.Accounting.PressureShed != before.Accounting.PressureShed ||
		afterReplacement.Accounting.Integrity != before.Accounting.Integrity || afterReplacement.Commands.Issued != before.Commands.Issued ||
		afterReplacement.Commands.Acknowledged != before.Commands.Acknowledged || afterReplacement.Commands.Failed != before.Commands.Failed ||
		afterReplacement.Commands.ResultFenced != before.Commands.ResultFenced || afterReplacement.Commands.Fenced != before.Commands.Fenced+1 || afterReplacement.Commands.Pending != 0 ||
		afterReplacement.Commands.Issued != afterReplacement.Commands.Pending+afterReplacement.Commands.Acknowledged+afterReplacement.Commands.Failed+afterReplacement.Commands.Fenced {
		t.Fatalf("replacement command accounting = before=%+v after=%+v", before.Commands, afterReplacement.Commands)
	}

	if got := admitTQResultForTest(t, e, oldTQResult); got.Code != DispositionTQFenced {
		t.Fatalf("old-epoch T/Q result = %+v", got)
	}
	afterTQFence := e.ObserveTQ()
	if afterTQFence.Accounting.Consumed != afterReplacement.Accounting.Consumed+1 || afterTQFence.Accounting.Fenced != afterReplacement.Accounting.Fenced+1 ||
		afterTQFence.Commands.ResultFenced != afterReplacement.Commands.ResultFenced+1 || afterTQFence.Commands.Issued != afterTQFence.Commands.Pending+afterTQFence.Commands.Acknowledged+afterTQFence.Commands.Failed+afterTQFence.Commands.Fenced ||
		afterTQFence.Accounting.Consumed != afterTQFence.Accounting.Applied+afterTQFence.Accounting.Duplicate+afterTQFence.Accounting.Rejected+afterTQFence.Accounting.Fenced+afterTQFence.Accounting.PressureShed+afterTQFence.Accounting.Integrity ||
		afterTQFence.Accounting.KnownPresent != 0 || afterTQFence.Accounting.Unknown != 0 {
		t.Fatalf("old T/Q result accounting/membership = before=%+v after=%+v", afterReplacement, afterTQFence)
	}

	if got := admitPressureForTest(t, e, oldPressureResult); got.Code != DispositionTQFenced {
		t.Fatalf("old-epoch pressure result = %+v", got)
	}
	afterFence := e.ObserveTQ()
	if afterFence.PressureFenced != afterTQFence.PressureFenced+1 || afterFence.PressureTransitions != before.PressureTransitions ||
		afterFence.Accounting.PressureShed != before.Accounting.PressureShed {
		t.Fatalf("monotonic operational counters = before=%+v replacement=%+v fenced=%+v", before, afterReplacement, afterFence)
	}
}

func pressureProofEngine(t *testing.T) (*Engine, reference.Binding, *atomic.Int64, time.Time) {
	t.Helper()
	binding := testBinding(t)
	start := binding.SessionStart().Add(30 * time.Minute)
	clockNanos := &atomic.Int64{}
	clockNanos.Store(start.UnixNano())
	delay := time.Duration(0)
	e, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }, Capacity: 64, RequiredReserve: 4, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}
	e.mu.Lock()
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive = lifecycleLive, 1, true
	e.state.committedT = immutableTime(start)
	e.state.aggregateEvaluator.current = pressureQualifiedEvaluation(start)
	coverage := tqCoverage{active: true, epoch: 1, start: start.Add(-10 * time.Second), ack: LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, greatest: LivePosition{ConnectionEpoch: 1, FrameSequence: 10}}
	e.state.tq = tqState{epoch: 1, nextToken: 1, desired: []string{"AAA", "MISSING"}, members: map[string]*tqSymbolState{
		"AAA":     {present: true, tradeCoverage: coverage, quoteCoverage: coverage, fingerprints: make(map[string]tqFingerprint)},
		"MISSING": {present: true, tradeCoverage: coverage, quoteCoverage: coverage, fingerprints: make(map[string]tqFingerprint)},
	}, pressure: tqPressureState{mode: TQPressureNormal, nextSequence: 1}}
	e.mu.Unlock()
	return e, binding, clockNanos, start
}

func applyPressureSample(t *testing.T, e *Engine, clock *atomic.Int64, at time.Time, sample TQPressureSample) {
	t.Helper()
	clock.Store(at.UnixNano())
	e.mu.Lock()
	e.advanceTQPressureTimerLocked(at)
	e.mu.Unlock()
	command := issuePressureForTest(t, e)
	input, err := NewTQPressureResultInput(command, sample)
	if err != nil {
		t.Fatal(err)
	}
	if got := admitPressureForTest(t, e, input); got.Code != DispositionTQApplied {
		t.Fatalf("pressure sample = %+v", got)
	}
}

func issuePressureForTest(t *testing.T, e *Engine) TQPressureCommand {
	t.Helper()
	command, err := e.IssueTQPressureCommand()
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func admitPressureForTest(t *testing.T, e *Engine, input TQPressureResultInput) Disposition {
	t.Helper()
	admission, completion := e.AdmitTQPressureResult(context.Background(), input)
	if admission != AdmissionAdmitted {
		t.Fatalf("pressure admission = %s", admission)
	}
	return awaitDisposition(t, completion)
}
