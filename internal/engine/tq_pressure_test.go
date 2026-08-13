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
	healthy := healthyTQPressureSample()
	unhealthy := healthy
	unhealthy.WaitingFrames = 10

	applyPressureSample(t, e, clockNanos, start, unhealthy)
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("first unhealthy sample bypassed dwell: %s", got)
	}
	applyPressureSample(t, e, clockNanos, start.Add(time.Second), unhealthy)
	view := e.ObserveTQ()
	if view.Pressure != TQPressureDegraded || !view.ShedTradesQuotes || view.Rows[0].TradeCoverage || view.Rows[0].Tape.Status != TQPressureShed || view.CommandPending {
		t.Fatalf("degraded containment = %+v", view)
	}
	severe := unhealthy
	severe.WaitingFrames = 25
	applyPressureSample(t, e, clockNanos, start.Add(2*time.Second), severe)
	applyPressureSample(t, e, clockNanos, start.Add(3*time.Second), severe)
	applyPressureSample(t, e, clockNanos, start.Add(4*time.Second), severe)
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
		result := tqResultForTest(t, command, LivePosition{ConnectionEpoch: 1, FrameSequence: position}, start.Add(4*time.Second), ControlSucceeded)
		position++
		if got := admitTQResultForTest(t, e, result); got.Code != DispositionTQApplied {
			t.Fatalf("removal result = %+v", got)
		}
	}
	if view = e.ObserveTQ(); view.Accounting.KnownPresent != 0 || view.CommandPending {
		t.Fatalf("provider membership not zero = %+v", view)
	}

	applyPressureSample(t, e, clockNanos, start.Add(5*time.Second), healthy)
	if got := e.ObserveTQ().Pressure; got != TQPressureAggregateOnly {
		t.Fatalf("early healthy sample recovered: %s", got)
	}
	for second := 6; second <= 9; second++ {
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), healthy)
	}
	view = e.ObserveTQ()
	if view.Pressure != TQPressureNormal || view.AggregateOnly || !view.CommandPending || view.PendingAction != TQSubscribe || view.PendingSymbol != "AAA" {
		t.Fatalf("ranked recovery start = %+v", view)
	}
	addAAA := issueTQForTest(t, e)
	if got := admitTQResultForTest(t, e, tqResultForTest(t, addAAA, LivePosition{ConnectionEpoch: 1, FrameSequence: position}, start.Add(9*time.Second), ControlSucceeded)); got.Code != DispositionTQApplied {
		t.Fatalf("AAA restoration = %+v", got)
	}
	view = e.ObserveTQ()
	if !view.CommandPending || view.PendingSymbol != "MISSING" || !view.Rows[0].TradeCoverage || view.Rows[1].TradeCoverage || view.Rows[0].Tape.Status != TQWarming || view.Rows[0].Spread.Status != TQWarming {
		t.Fatalf("restoration did not serialize by acknowledgement = %+v", view)
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

func TestPC9DiagnosticsAndQuietWindowsDoNotGatePressure(t *testing.T) {
	policy := defaultTQPressurePolicy()
	if policy.degradedSamples != 2 || policy.aggregateSamples != 3 || policy.watermarkSamples != 2 || policy.recoverySamples != 5 ||
		policy.degradedQueuePercent != 10 || policy.aggregateQueuePercent != 25 || policy.recoveryQueuePercent != 1 ||
		policy.degradedOldest != time.Second || policy.aggregateOldest != 2*time.Second || policy.recoveryOldest != 250*time.Millisecond {
		t.Fatalf("direct pressure policy = %+v", policy)
	}
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	diagnosticSpike := healthyTQPressureSample()
	diagnosticSpike.HeapAllocBytes, diagnosticSpike.Goroutines, diagnosticSpike.MaxDeliveryDelayOneSec = ^uint64(0), 1<<20, time.Hour
	for second, age := range []time.Duration{251 * time.Millisecond, 500 * time.Millisecond, time.Second} {
		diagnosticSpike.ActiveFrameAge = age
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), diagnosticSpike)
	}
	if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.ShedTradesQuotes {
		t.Fatalf("active frame age changed global pressure = %+v", view)
	}
	diagnosticSpike.ActiveFrameAge = time.Hour
	for second := 3; second <= 10; second++ {
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), diagnosticSpike)
	}
	if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.ShedTradesQuotes {
		t.Fatalf("diagnostics changed pressure = %+v", view)
	}
	e.mu.Lock()
	e.state.tq.members, e.state.tq.desired = nil, nil
	e.state.aggregateEvaluator.current.mode, e.state.aggregateEvaluator.current.rows = rankingStale, nil
	e.state.tq.pressure.mode, e.state.tq.pressure.cause, e.state.tq.aggregateOnly = TQPressureAggregateOnly, TQPressureCauseWaitingFrames, true
	e.mu.Unlock()
	quiet := diagnosticSpike
	quiet.WaitingFrames, quiet.OldestWaitingFrameAge = 0, 0
	for second := 11; second <= 16; second++ {
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), quiet)
	}
	if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.PressureCause != TQPressureCauseNone {
		t.Fatalf("empty healthy windows did not recover = %+v", view)
	}
}

func TestPC9PressureMissingAndLateResultsDoNotCreatePressure(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	healthy := healthyTQPressureSample()

	e.mu.Lock()
	e.advanceTQPressureTimerLocked(start)
	e.mu.Unlock()
	first := issuePressureForTest(t, e)
	clockNanos.Store(start.Add(2 * time.Second).UnixNano())
	e.mu.Lock()
	e.advanceTQPressureTimerLocked(start.Add(2 * time.Second))
	e.mu.Unlock()
	if view := e.ObserveTQ(); view.Pressure != TQPressureNormal || view.PressureMisses != 1 {
		t.Fatalf("first expiry = %+v", view)
	}
	second := issuePressureForTest(t, e)
	lateFirst, _ := NewTQPressureResultInput(first, healthy)
	if got := admitPressureForTest(t, e, lateFirst); got.Code != DispositionTQFenced || e.ObserveTQ().Pressure != TQPressureNormal {
		t.Fatalf("late healthy first result = %+v view=%+v", got, e.ObserveTQ())
	}
	clockNanos.Store(start.Add(4100 * time.Millisecond).UnixNano())
	lateSecond, _ := NewTQPressureResultInput(second, healthy)
	if got := admitPressureForTest(t, e, lateSecond); got.Code != DispositionTQFenced {
		t.Fatalf("late second result = %+v", got)
	}
	view := e.ObserveTQ()
	if view.Pressure != TQPressureNormal || view.ShedTradesQuotes || view.PressureFenced < 2 {
		t.Fatalf("sample silence created pressure = %+v", view)
	}
}

func TestPC9PressureTQLocalAccountingFailureIsImmediate(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	sample := healthyTQPressureSample()
	sample.TQLocalAccountingHealthy = false
	applyPressureSample(t, e, clockNanos, start, sample)
	if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || !view.AggregateOnly {
		t.Fatalf("T/Q-local accounting failure = %+v", view)
	}
}

func TestPC9PressureCapacityDropIsImmediateAndCumulativeBaselineRecovers(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	baseline := healthyTQPressureSample()
	applyPressureSample(t, e, clockNanos, start, baseline)
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("zero capacity baseline created pressure = %s", got)
	}
	baseline.SlotCapacityDrops = 1
	applyPressureSample(t, e, clockNanos, start.Add(time.Second), baseline)
	if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || view.PressureCause != TQPressureCauseCapacityDrop {
		t.Fatalf("new capacity drop not decisive = %+v", view)
	}
	for second := 2; second <= 7; second++ {
		applyPressureSample(t, e, clockNanos, start.Add(time.Duration(second)*time.Second), baseline)
	}
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("unchanged cumulative capacity count blocked recovery = %s", got)
	}
}

func TestPC9ExactWaitingPressureBoundaries(t *testing.T) {
	degradedCases := []struct {
		name  string
		set   func(*TQPressureSample)
		cause TQPressureCause
	}{
		{"frames", func(s *TQPressureSample) { s.WaitingFrames = 10 }, TQPressureCauseWaitingFrames},
		{"bytes", func(s *TQPressureSample) { s.WaitingBytes = 100 }, TQPressureCauseWaitingBytes},
		{"oldest", func(s *TQPressureSample) { s.OldestWaitingFrameAge = time.Second }, TQPressureCauseOldestWaitingFrame},
	}
	for _, test := range degradedCases {
		t.Run("degraded_"+test.name, func(t *testing.T) {
			e, _, clock, start := pressureProofEngine(t)
			defer closeAndWait(t, e)
			sample := healthyTQPressureSample()
			test.set(&sample)
			applyPressureSample(t, e, clock, start, sample)
			if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
				t.Fatalf("one exact boundary sample degraded: %s", got)
			}
			applyPressureSample(t, e, clock, start.Add(time.Second), sample)
			if view := e.ObserveTQ(); view.Pressure != TQPressureDegraded || view.PressureCause != test.cause {
				t.Fatalf("two exact boundary samples = %+v", view)
			}
		})
	}

	aggregateCases := []struct {
		name  string
		set   func(*TQPressureSample)
		cause TQPressureCause
	}{
		{"frames", func(s *TQPressureSample) { s.WaitingFrames = 25 }, TQPressureCauseWaitingFrames},
		{"bytes", func(s *TQPressureSample) { s.WaitingBytes = 250 }, TQPressureCauseWaitingBytes},
		{"oldest", func(s *TQPressureSample) { s.OldestWaitingFrameAge = 2 * time.Second }, TQPressureCauseOldestWaitingFrame},
	}
	for _, test := range aggregateCases {
		t.Run("aggregate_"+test.name, func(t *testing.T) {
			e, _, clock, start := pressureProofEngine(t)
			defer closeAndWait(t, e)
			sample := healthyTQPressureSample()
			test.set(&sample)
			for second := 0; second < 2; second++ {
				applyPressureSample(t, e, clock, start.Add(time.Duration(second)*time.Second), sample)
			}
			if got := e.ObserveTQ().Pressure; got == TQPressureAggregateOnly {
				t.Fatalf("two exact severe samples entered aggregate-only")
			}
			applyPressureSample(t, e, clock, start.Add(2*time.Second), sample)
			if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || view.PressureCause != test.cause {
				t.Fatalf("three exact severe samples = %+v", view)
			}
		})
	}
}

func TestPC9WatermarkLagRequiresTQWorkAndGreaterThanTwoSeconds(t *testing.T) {
	e, _, clock, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	sample := healthyTQPressureSample()
	sample.TQWorkPresent = true
	for second, lag := range []time.Duration{0, time.Second, 2 * time.Second} {
		sample.AggregateWatermarkLag = lag
		applyPressureSample(t, e, clock, start.Add(time.Duration(second)*time.Second), sample)
	}
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("lag at or below two seconds triggered: %s", got)
	}
	sample.AggregateWatermarkLag = 2*time.Second + time.Nanosecond
	applyPressureSample(t, e, clock, start.Add(3*time.Second), sample)
	if got := e.ObserveTQ().Pressure; got != TQPressureNormal {
		t.Fatalf("one excessive lag sample triggered: %s", got)
	}
	applyPressureSample(t, e, clock, start.Add(4*time.Second), sample)
	if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || view.PressureCause != TQPressureCauseWatermarkLag {
		t.Fatalf("two excessive lag samples with T/Q work = %+v", view)
	}
}

func TestPC9SlotAndByteDropsIndependentlyEnterAggregateOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		set  func(*TQPressureSample)
	}{
		{"slot", func(s *TQPressureSample) { s.SlotCapacityDrops = 1 }},
		{"byte", func(s *TQPressureSample) { s.ByteCapacityDrops = 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, _, clock, start := pressureProofEngine(t)
			defer closeAndWait(t, e)
			sample := healthyTQPressureSample()
			test.set(&sample)
			applyPressureSample(t, e, clock, start, sample)
			if view := e.ObserveTQ(); view.Pressure != TQPressureAggregateOnly || view.PressureCause != TQPressureCauseCapacityDrop {
				t.Fatalf("drop did not enter aggregate-only: %+v", view)
			}
		})
	}
}

func TestPC9PressureEpochReplacementPreservesMonotonicAuthority(t *testing.T) {
	e, _, clockNanos, start := pressureProofEngine(t)
	defer closeAndWait(t, e)
	healthy := healthyTQPressureSample()

	e.mu.Lock()
	e.state.tq.consumed, e.state.tq.applied, e.state.tq.duplicate = 6, 1, 1
	e.state.tq.rejected, e.state.tq.fenced, e.state.tq.pressureShed, e.state.tq.integrity = 1, 1, 1, 1
	e.state.tq.tradePressureShed = 1
	e.state.tq.commandsIssued, e.state.tq.commandsAcknowledged, e.state.tq.commandsFailed, e.state.tq.commandsFenced = 3, 1, 1, 1
	e.state.tq.commandResultsFenced = 1
	e.state.tq.pressure.mode, e.state.tq.pressure.cause, e.state.tq.pressure.transitions, e.state.tq.pressure.fenced = TQPressureDegraded, TQPressureCauseWaitingFrames, 2, 1
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

func healthyTQPressureSample() TQPressureSample {
	return TQPressureSample{FrameCapacity: 100, ByteCapacity: 1000, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}
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
