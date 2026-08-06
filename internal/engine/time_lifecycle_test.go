package engine

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type controlledClock struct {
	mu    sync.Mutex
	now   time.Time
	reads int
}

func (c *controlledClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	return c.now
}

func (c *controlledClock) set(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

func (c *controlledClock) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

func TestENGTIME01InjectedClockTimerIntegrity(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	clock := &controlledClock{now: start.Add(-time.Minute)}
	e := newS3Engine(t, RunModeLive, clock.read, 2, 1, 250*time.Millisecond)
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}

	entered, release := make(chan struct{}), make(chan struct{})
	var pauseOnce sync.Once
	e.beforeConsume = func(*queueNode) { pauseOnce.Do(func() { close(entered); <-release }) }

	clock.set(start.Add(-30 * time.Second))
	result, first := e.AdmitTimer(context.Background())
	if result != AdmissionAdmitted {
		t.Fatalf("first timer admission = %s", result)
	}
	<-entered
	clock.set(start.Add(-20 * time.Second))
	_, second := e.AdmitTimer(context.Background())
	clock.set(start.Add(-10 * time.Second))
	_, third := e.AdmitTimer(context.Background())
	readsAtCapacity := clock.count()

	blockedResult := make(chan AdmissionResult, 1)
	blockedCompletion := make(chan (<-chan TimerDisposition), 1)
	go func() {
		result, completion := e.AdmitTimer(context.Background())
		blockedResult <- result
		blockedCompletion <- completion
	}()
	waitForInProgress(t, e, 1)
	if got := clock.count(); got != readsAtCapacity {
		t.Fatalf("blocked admission sampled early: reads %d -> %d", readsAtCapacity, got)
	}
	clock.set(start)
	close(release)
	if result := <-blockedResult; result != AdmissionAdmitted {
		t.Fatalf("unblocked timer admission = %s", result)
	}
	blocked := <-blockedCompletion

	gotTimers := []TimerDisposition{
		awaitTimerDisposition(t, first), awaitTimerDisposition(t, second),
		awaitTimerDisposition(t, third), awaitTimerDisposition(t, blocked),
	}
	for index, got := range gotTimers {
		if got.Code != DispositionTimerApplied || got.EngineSequence != uint64(index+2) || got.SystemSequence != uint64(index+1) {
			t.Fatalf("timer[%d] = %+v", index, got)
		}
		if index > 0 && got.AdmissionTime.Before(gotTimers[index-1].AdmissionTime) {
			t.Fatalf("timer samples regressed: %+v", gotTimers)
		}
	}
	if gotTimers[3].AdmissionTime != start {
		t.Fatalf("blocked timer preserved pre-capacity sample: %s", gotTimers[3].AdmissionTime)
	}

	readsBeforeCancel := clock.count()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, canceled := e.AdmitTimer(ctx)
	if result != AdmissionNotAdmittedCanceled || canceled != nil || clock.count() != readsBeforeCancel {
		t.Fatalf("pre-link cancel = %s completion=%v reads=%d/%d", result, canceled, clock.count(), readsBeforeCancel)
	}

	clock.set(start)
	ctx, cancel = context.WithCancel(context.Background())
	result, equal := e.AdmitTimer(ctx)
	if result != AdmissionAdmitted {
		t.Fatalf("equal timer admission = %s", result)
	}
	cancel()
	if got := awaitTimerDisposition(t, equal); got.Code != DispositionTimerApplied || got.AdmissionTime != start {
		t.Fatalf("post-link cancellation/equal clock = %+v", got)
	}

	clock.set(start.Add(-time.Nanosecond))
	result, regressed := e.AdmitTimer(context.Background())
	if result != AdmissionAdmitted {
		t.Fatalf("regression admission = %s", result)
	}
	regression := awaitTimerDisposition(t, regressed)
	if regression.Code != DispositionClockRegression || regression.Reason != ReasonClockRegression || regression.AdmissionTime != start || regression.SuppressionDisposition != SuppressionRestartRequired {
		t.Fatalf("regression disposition = %+v", regression)
	}
	observation := e.observeTimeLifecycle()
	if observation.ClockMonotonic || observation.Lifecycle != lifecycleSuppressed || observation.CommittedT != nil || observation.LatestTransition == nil || observation.LatestTransition.Reason != lifecycleReasonClockRegression {
		t.Fatalf("regression containment = %+v", observation)
	}

	clock.set(start.Add(time.Second))
	if result, later := e.AdmitTimer(context.Background()); result != AdmissionNotAdmittedClosed || later != nil {
		t.Fatalf("post-regression admission = %s/%v", result, later)
	}
	observation = e.observeTimeLifecycle()
	if observation.Lifecycle != lifecycleSuppressed || observation.CommittedT != nil {
		t.Fatalf("ordinary timer restored suppressed state: %+v", observation)
	}
	e.mu.Lock()
	if e.state.aggregates != (aggregateAccounting{}) || e.state.binding.symbols[0].aggregates != nil {
		t.Fatal("quiet timers created aggregate state")
	}
	e.mu.Unlock()
	assertAccounting(t, e)
	closeAndWait(t, e)
}

func TestENGCOMMIT01ClosedBaseGateAndNonauthority(t *testing.T) {
	binding := testBinding(t)
	start, end := binding.SessionStart(), binding.SessionEnd()
	delay := 2500 * time.Millisecond

	if engine, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return start }, Capacity: 4, RequiredReserve: 1, EvaluationDelay: durationPointer(-1)}); err == nil || engine != nil {
		t.Fatal("negative evaluation delay was accepted")
	}

	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"before S", start.Add(-time.Second), start},
		{"at S", start.Add(delay), start},
		{"interior preserves precision until floor", start.Add(10*time.Second + delay + 987*time.Millisecond), start.Add(10 * time.Second)},
		{"at E", end.Add(delay), end},
		{"after E", end.Add(delay + time.Hour), end},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			clock := &controlledClock{now: start.Add(-time.Hour)}
			e := newS3Engine(t, RunModeLive, clock.read, 4, 1, delay)
			if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
				t.Fatalf("binding = %+v", got)
			}
			clock.set(test.now)
			result, completion := e.AdmitTimer(context.Background())
			if result != AdmissionAdmitted {
				t.Fatalf("timer admission = %s", result)
			}
			if got := awaitTimerDisposition(t, completion); got.Code != DispositionTimerApplied {
				t.Fatalf("timer = %+v", got)
			}
			observation := e.observeTimeLifecycle()
			if observation.LatestTarget == nil || *observation.LatestTarget != test.want || observation.CommittedT != nil {
				t.Fatalf("target/T = %+v", observation)
			}
			closeAndWait(t, e)
		})
	}

	for _, mode := range []RunMode{RunModeLive, RunModeReplay} {
		t.Run(string(mode)+" branch remains closed", func(t *testing.T) {
			now := start.Add(time.Minute)
			e := aggregateEngine(t, binding, mode, &now)
			var input AggregateInput
			if mode == RunModeLive {
				input = liveAggregate(binding, "AAA", start, 1, 1)
			} else {
				input = replayAggregate(binding, "AAA", start, 1)
			}
			applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
			result, timer := e.AdmitTimer(context.Background())
			if result != AdmissionAdmitted {
				t.Fatalf("timer admission = %s", result)
			}
			got := awaitTimerDisposition(t, timer)
			if mode == RunModeReplay && got.Code != DispositionIllegalLifecycle {
				t.Fatalf("replay timer without artifact = %+v", got)
			}
			if e.observeTimeLifecycle().CommittedT != nil {
				t.Fatal("canonical aggregate, receipt/delivery evidence, lifecycle, or timer advanced T")
			}
			closeAndWait(t, e)
		})
	}

	t.Run("source ownership", func(t *testing.T) {
		files, err := filepath.Glob("*.go")
		if err != nil {
			t.Fatal(err)
		}
		assignments := 0
		for _, name := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			source, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(source), "time.Now(") {
				t.Fatalf("product wall-clock read in %s", name)
			}
			file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				assignment, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range assignment.Lhs {
					selector, ok := lhs.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "committedT" {
						assignments++
					}
				}
				return true
			})
		}
		if assignments != 1 {
			t.Fatalf("committedT assignment paths = %d, want 1", assignments)
		}
		engineType := reflect.TypeOf((*Engine)(nil))
		for index := 0; index < engineType.NumMethod(); index++ {
			name := engineType.Method(index).Name
			if strings.Contains(name, "Commit") || strings.Contains(name, "Target") || strings.Contains(name, "Lifecycle") {
				t.Fatalf("exported mutation/observation seam %s", name)
			}
		}
	})
}

func TestENGLIFE01RemainingStagedLifecycleOwner(t *testing.T) {
	binding := testBinding(t)
	start, end := binding.SessionStart(), binding.SessionEnd()
	states := []lifecycle{lifecycleInitializing, lifecycleAwaitingSession, lifecycleAwaitingAggregateAck, lifecycleHydrating, lifecycleLive, lifecycleRecovering, lifecycleReplaying, lifecycleSuppressed, lifecycleEnded}
	if got := []string{string(states[0]), string(states[1]), string(states[2]), string(states[3]), string(states[4]), string(states[5]), string(states[6]), string(states[7]), string(states[8])}; !reflect.DeepEqual(got, []string{"initializing", "awaiting_session", "awaiting_aggregate_ack", "hydrating", "live", "recovering", "replaying", "suppressed", "ended"}) {
		t.Fatalf("lifecycle vocabulary = %v", got)
	}

	for _, test := range []struct {
		mode   RunMode
		reason lifecycleReason
		want   SuppressionDisposition
	}{
		{RunModeLive, lifecycleReasonCanonicalIntegrity, SuppressionCleanReinitializationRequired},
		{RunModeLive, lifecycleReasonClockRegression, SuppressionRestartRequired},
		{RunModeLive, lifecycleReasonSequenceExhaustion, SuppressionRestartRequired},
		{RunModeLive, lifecycleReasonPublicationIntegrity, SuppressionRestartRequired},
		{RunModeLive, lifecycleReasonAccountingIntegrity, SuppressionRestartRequired},
		{RunModeReplay, lifecycleReasonCanonicalIntegrity, SuppressionTerminalReplayFailure},
		{RunModeReplay, lifecycleReasonClockRegression, SuppressionTerminalReplayFailure},
		{RunModeReplay, lifecycleReasonSequenceExhaustion, SuppressionTerminalReplayFailure},
		{RunModeReplay, lifecycleReasonPublicationIntegrity, SuppressionTerminalReplayFailure},
		{RunModeReplay, lifecycleReasonAccountingIntegrity, SuppressionTerminalReplayFailure},
	} {
		if got := suppressionDispositionFor(test.mode, test.reason); got != test.want {
			t.Fatalf("suppression mapping %s/%s = %s, want %s", test.mode, test.reason, got, test.want)
		}
	}
	if suppressionRequiresTermination(SuppressionSameBindingRecoveryAllowed) ||
		!suppressionRequiresTermination(SuppressionCleanReinitializationRequired) ||
		!suppressionRequiresTermination(SuppressionRestartRequired) ||
		!suppressionRequiresTermination(SuppressionTerminalReplayFailure) {
		t.Fatal("suppression admission-boundary mapping is not closed")
	}

	for _, test := range []struct {
		name        string
		mode        RunMode
		bindingTime time.Time
		suppress    bool
	}{
		{"awaiting session", RunModeLive, start.Add(-time.Minute), false},
		{"awaiting aggregate ack", RunModeLive, start, false},
		{"replay initializing", RunModeReplay, start, false},
	} {
		t.Run("LIFE-T29 from "+test.name, func(t *testing.T) {
			terminalClock := &controlledClock{now: test.bindingTime}
			terminalEngine := newS3Engine(t, test.mode, terminalClock.read, 3, 1, 0)
			awaitDisposition(t, admitValidBinding(t, terminalEngine, binding))
			if test.suppress {
				terminalClock.set(test.bindingTime.Add(-time.Nanosecond))
				_, regression := terminalEngine.AdmitTimer(context.Background())
				if got := awaitTimerDisposition(t, regression); got.Code != DispositionClockRegression {
					t.Fatalf("suppression setup = %+v", got)
				}
			}
			terminalClock.set(end)
			_, timer := terminalEngine.AdmitTimer(context.Background())
			if got := awaitTimerDisposition(t, timer); got.Code != DispositionTimerApplied {
				t.Fatalf("terminal timer = %+v", got)
			}
			obs := terminalEngine.observeTimeLifecycle()
			if obs.Lifecycle != lifecycleEnded || obs.LatestTransition == nil || obs.LatestTransition.Reason != lifecycleReasonSessionEnd {
				t.Fatalf("LIFE-T29 result = %+v", obs)
			}
			if err := terminalEngine.Wait(testContext(t)); err != nil {
				t.Fatal(err)
			}
		})
	}

	clock := &controlledClock{now: start.Add(-time.Minute)}
	e := newS3Engine(t, RunModeLive, clock.read, 5, 1, 0)
	awaitDisposition(t, admitValidBinding(t, e, binding))
	_, before := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, before); got.Code != DispositionTimerApplied || e.observeTimeLifecycle().Lifecycle != lifecycleAwaitingSession {
		t.Fatalf("pre-S timer = %+v lifecycle=%s", got, e.observeTimeLifecycle().Lifecycle)
	}
	clock.set(start)
	_, atStart := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, atStart); got.Code != DispositionTimerApplied || e.observeTimeLifecycle().Lifecycle != lifecycleAwaitingAggregateAck {
		t.Fatalf("S timer LIFE-T07 = %+v lifecycle=%s", got, e.observeTimeLifecycle().Lifecycle)
	}
	clock.set(start.Add(time.Minute))
	_, waiting := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, waiting); got.Code != DispositionTimerApplied || e.observeTimeLifecycle().Lifecycle != lifecycleAwaitingAggregateAck {
		t.Fatalf("ack wait fabricated progress = %+v lifecycle=%s", got, e.observeTimeLifecycle().Lifecycle)
	}

	entered, release := make(chan struct{}), make(chan struct{})
	var pauseOnce sync.Once
	e.beforeConsume = func(*queueNode) { pauseOnce.Do(func() { close(entered); <-release }) }
	clock.set(end)
	_, terminalTimer := e.AdmitTimer(context.Background())
	<-entered
	result, behind := admitIllegalForProof(e)
	if result != AdmissionAdmitted {
		t.Fatalf("node behind terminal timer = %s", result)
	}
	close(release)
	if got := awaitTimerDisposition(t, terminalTimer); got.Code != DispositionTimerApplied {
		t.Fatalf("terminal timer = %+v", got)
	}
	if got := awaitDisposition(t, behind); got.Code != DispositionTerminal || got.Reason != ReasonTerminal {
		t.Fatalf("node behind terminal timer = %+v", got)
	}
	result, later := e.AdmitTimer(context.Background())
	if result != AdmissionNotAdmittedClosed || later != nil {
		t.Fatalf("post-ended admission = %s", result)
	}
	if err := e.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	observation := e.observeTimeLifecycle()
	if observation.Lifecycle != lifecycleEnded || observation.LatestTransition == nil || observation.LatestTransition.Reason != lifecycleReasonSessionEnd {
		t.Fatalf("terminal state = %+v", observation)
	}

	t.Run("illegal admitted input", func(t *testing.T) {
		now := start
		illegalEngine := testEngine(t, RunModeLive, now, 3, 1)
		awaitDisposition(t, admitValidBinding(t, illegalEngine, binding))
		_, completion := admitIllegalForProof(illegalEngine)
		if got := awaitDisposition(t, completion); got.Code != DispositionIllegalLifecycle || got.Reason != ReasonLifecycle {
			t.Fatalf("illegal input = %+v", got)
		}
		closeAndWait(t, illegalEngine)
	})

	t.Run("suppressed unbound engine cannot install binding", func(t *testing.T) {
		regressionClock := &controlledClock{now: start}
		unbound := newS3Engine(t, RunModeLive, regressionClock.read, 3, 1, 0)
		_, illegalTimer := unbound.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, illegalTimer); got.Code != DispositionIllegalLifecycle {
			t.Fatalf("unbound timer = %+v", got)
		}
		regressionClock.set(start.Add(-time.Nanosecond))
		_, regressedBinding := unbound.AdmitBinding(context.Background(), validBindingInput(binding))
		if got := awaitDisposition(t, regressedBinding); got.Code != DispositionClockRegression {
			t.Fatalf("regressing binding = %+v", got)
		}
		regressionClock.set(start.Add(time.Second))
		if result, laterBinding := unbound.AdmitBinding(context.Background(), validBindingInput(binding)); result != AdmissionNotAdmittedClosed || laterBinding != nil {
			t.Fatalf("post-suppression binding admission = %s/%v", result, laterBinding)
		}
		unbound.mu.Lock()
		installed := unbound.state.binding
		unbound.mu.Unlock()
		if installed != nil || unbound.observeTimeLifecycle().Lifecycle != lifecycleSuppressed {
			t.Fatalf("binding escaped suppression: binding=%v state=%+v", installed, unbound.observeTimeLifecycle())
		}
		closeAndWait(t, unbound)
	})

	t.Run("canonical integrity suppression", func(t *testing.T) {
		now := start.Add(time.Minute)
		integrityEngine := aggregateEngine(t, binding, RunModeLive, &now)
		base := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, integrityEngine, base, DispositionAggregateInserted, ReasonNone)
		conflict := changedClose(base, base.Values.Close+1)
		got := applyAggregate(t, integrityEngine, conflict, DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal)
		obs := integrityEngine.observeTimeLifecycle()
		if got.SuppressionDisposition != SuppressionCleanReinitializationRequired || obs.Lifecycle != lifecycleSuppressed || obs.SuppressionDisposition != SuppressionCleanReinitializationRequired || obs.LatestTransition == nil || obs.LatestTransition.Reason != lifecycleReasonCanonicalIntegrity || obs.LatestTransition.SuppressionDisposition != SuppressionCleanReinitializationRequired || obs.CommittedT != nil {
			t.Fatalf("canonical containment = %+v", obs)
		}
		closeAndWait(t, integrityEngine)
	})

	t.Run("sequence exhaustion and controlled stop", func(t *testing.T) {
		exhausted := testEngine(t, RunModeReplay, start, 2, 1)
		exhausted.mu.Lock()
		exhausted.lastReserved = math.MaxUint64 - 2
		exhausted.nextSequence = math.MaxUint64 - 1
		exhausted.mu.Unlock()
		_, last := admitControl(t, exhausted, false)
		if result, _ := admitControl(t, exhausted, false); result != AdmissionSequenceBudgetExhausted {
			t.Fatalf("sequence exhaustion = %s", result)
		}
		awaitDisposition(t, last)
		if err := exhausted.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		obs := exhausted.observeTimeLifecycle()
		if obs.Lifecycle != lifecycleSuppressed || obs.SuppressionDisposition != SuppressionTerminalReplayFailure || obs.LatestTransition == nil || obs.LatestTransition.Reason != lifecycleReasonSequenceExhaustion || obs.LatestTransition.SuppressionDisposition != SuppressionTerminalReplayFailure {
			t.Fatalf("exhaustion lifecycle = %+v", obs)
		}
		if view := exhausted.observePublication(); view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonSequenceExhaustion || view.suppressionDisposition != SuppressionTerminalReplayFailure || view.lastDisposition != DispositionSequenceExhausted {
			t.Fatalf("exhaustion publication = %+v", view)
		}

		stopped := testEngine(t, RunModeLive, start, 3, 1)
		awaitDisposition(t, admitValidBinding(t, stopped, binding))
		_, stop := stopped.Stop(context.Background())
		if got := awaitDisposition(t, stop); got.Code != DispositionControlApplied {
			t.Fatalf("stop = %+v", got)
		}
		if err := stopped.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		obs = stopped.observeTimeLifecycle()
		if obs.Lifecycle != lifecycleEnded || obs.LatestTransition == nil || obs.LatestTransition.Reason != lifecycleReasonControlledStop {
			t.Fatalf("controlled stop lifecycle = %+v", obs)
		}
	})

	t.Run("replay binding remains initializing", func(t *testing.T) {
		now := start
		replay := testEngine(t, RunModeReplay, now, 3, 1)
		awaitDisposition(t, admitValidBinding(t, replay, binding))
		before := replay.observeTimeLifecycle()
		_, timer := replay.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, timer); got.Code != DispositionIllegalLifecycle || replay.observeTimeLifecycle().Lifecycle != lifecycleInitializing {
			t.Fatalf("unvalidated replay timer = %+v lifecycle=%s", got, replay.observeTimeLifecycle().Lifecycle)
		}
		after := replay.observeTimeLifecycle()
		if before.LatestTarget != nil || after.LatestTarget != nil {
			t.Fatalf("illegal replay timer mutated target: before=%+v after=%+v", before, after)
		}
		closeAndWait(t, replay)
	})
}

func newS3Engine(t *testing.T, mode RunMode, clock Clock, capacity, reserve int, delay time.Duration) *Engine {
	t.Helper()
	e, err := New(Config{Mode: mode, Clock: clock, Capacity: capacity, RequiredReserve: reserve, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func awaitTimerDisposition(t *testing.T, completion <-chan TimerDisposition) TimerDisposition {
	t.Helper()
	select {
	case disposition, ok := <-completion:
		if !ok {
			t.Fatal("timer completion closed without disposition")
		}
		return disposition
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for timer disposition")
		return TimerDisposition{}
	}
}

func admitIllegalForProof(e *Engine) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: inputIllegal}, false)
}
