package engine

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPC5ENGINEConnectionControlLifecycle is P-C5-ENGINE. It proves the closed
// control family enters the existing FIFO, only a current post-write A.* ack
// opens the live aggregate path, stale epochs fence, loss takes the Phase 1
// lifecycle edges, T/Q-only failure is deferred, and ingress ambiguity fails
// closed without a second transition/publication owner.
func TestPC5ENGINEConnectionControlLifecycle(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("ordered current ack opens only the post-ack aggregate path", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)

		position := func(frame uint64, index uint32) LivePosition {
			return LivePosition{ConnectionEpoch: 1, FrameSequence: frame, ArrayIndex: index}
		}
		facts := []ConnectionControlInput{
			controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded),
			controlFact(binding.Identity(), ConnectionEstablished, 1, position(1, 0), now, 1, ControlSucceeded),
			controlFact(binding.Identity(), AuthenticationResult, 1, position(2, 0), now, 1, ControlSucceeded),
			controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, now, 2, ControlSucceeded),
		}
		for index, fact := range facts {
			got := admitConnectionControl(t, e, fact)
			if got.Code != DispositionConnectionControlApplied || got.EngineSequence != uint64(index+2) {
				t.Fatalf("fact %d disposition = %+v", index, got)
			}
		}

		beforeAck := liveAggregate(binding, "AAA", start, 1, 2)
		beforeAck.Live.ArrayIndex = 1
		if got := admitProductionAggregate(t, e, beforeAck); got.Code != DispositionAggregateRejected || got.Reason != ReasonLifecycle {
			t.Fatalf("aggregate before ack = %+v", got)
		}

		ack := controlFact(binding.Identity(), AggregateSubscriptionResult, 1, position(3, 0), now, 2, ControlSucceeded)
		gotAck := admitConnectionControl(t, e, ack)
		if gotAck.Code != DispositionConnectionControlApplied || e.observeTimeLifecycle().Lifecycle != lifecycleHydrating {
			t.Fatalf("ack = %+v lifecycle=%s", gotAck, e.observeTimeLifecycle().Lifecycle)
		}
		regressing := controlFact(binding.Identity(), AuthenticationResult, 1, position(2, 1), now, 1, ControlSucceeded)
		if got := admitConnectionControl(t, e, regressing); got.Code != DispositionConnectionControlFenced || got.Reason != ReasonNonprecedent {
			t.Fatalf("regressing control position = %+v", got)
		}

		atAck := liveAggregate(binding, "AAA", start, 1, 3)
		if got := admitProductionAggregate(t, e, atAck); got.Code != DispositionAggregateRejected || got.Reason != ReasonAggregateBeforeAck {
			t.Fatalf("aggregate at ack = %+v", got)
		}
		postAck := liveAggregate(binding, "AAA", start, 1, 3)
		postAck.Live.ArrayIndex = 1
		if got := admitProductionAggregate(t, e, postAck); got.Code != DispositionAggregateInserted {
			t.Fatalf("post-ack aggregate = %+v", got)
		}
		duplicateAck := controlFact(binding.Identity(), AggregateSubscriptionResult, 1, position(5, 0), now, 2, ControlSucceeded)
		if got := admitConnectionControl(t, e, duplicateAck); got.Code != DispositionConnectionControlApplied || e.state.aggregateAckPosition != position(3, 0) {
			t.Fatalf("duplicate ack moved handoff: disposition=%+v handoff=%+v", got, e.state.aggregateAckPosition)
		}
		laterAggregate := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 6)
		if got := admitProductionAggregate(t, e, laterAggregate); got.Code != DispositionAggregateInserted {
			t.Fatalf("aggregate after diagnostic duplicate ack = %+v", got)
		}

		view := e.observePublication()
		if view.connectionEpoch != 1 || !view.connectionActive || !view.aggregateAcknowledged || view.aggregateAckPosition != position(3, 0) ||
			view.lifecycle != lifecycleHydrating || view.lastEngineSequence == 0 || !view.connectionControls.reconciles() {
			t.Fatalf("atomic publication/control accounting = %+v", view)
		}
	})

	t.Run("intent or acknowledgement without successful write cannot hand off", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		ack := controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 2, ControlSucceeded)
		if got := admitConnectionControl(t, e, ack); got.Code != DispositionConnectionControlRejected || got.Reason != ReasonControlSequence {
			t.Fatalf("write-less ack = %+v", got)
		}
		if e.state.aggregateAcknowledged || e.observeTimeLifecycle().Lifecycle != lifecycleAwaitingAggregateAck {
			t.Fatal("write-less ack changed handoff state")
		}
	})

	t.Run("rejected and fenced positions do not mutate accepted causal authority", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		wrongBinding := controlFact("session-binding-v1:wrong", AuthenticationResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 10}, now, 1, ControlSucceeded)
		if got := admitConnectionControl(t, e, wrongBinding); got.Code != DispositionConnectionControlFenced || got.Reason != ReasonBinding {
			t.Fatalf("wrong-binding fact = %+v", got)
		}
		malformed := controlFact(binding.Identity(), AuthenticationResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 11}, now, 0, ControlSucceeded)
		if got := admitConnectionControl(t, e, malformed); got.Code != DispositionConnectionControlRejected || got.Reason != ReasonStructural {
			t.Fatalf("malformed fact = %+v", got)
		}
		if e.state.connectionControl.greatestPosition != (LivePosition{}) {
			t.Fatalf("invalid facts advanced causal authority: %+v", e.state.connectionControl.greatestPosition)
		}
		admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, now, 2, ControlSucceeded))
		ack := controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 2, ControlSucceeded)
		if got := admitConnectionControl(t, e, ack); got.Code != DispositionConnectionControlApplied || e.observeTimeLifecycle().Lifecycle != lifecycleHydrating {
			t.Fatalf("valid lower-position ack after invalid evidence = %+v lifecycle=%s", got, e.observeTimeLifecycle().Lifecycle)
		}
	})

	t.Run("second active epoch and stale old ack are contained", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		if got := admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 2, LivePosition{}, now, 2, ControlSucceeded)); got.Code != DispositionConnectionControlRejected || got.Reason != ReasonLifecycle {
			t.Fatalf("second active epoch = %+v", got)
		}
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 0, ControlFailed))
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 2, LivePosition{}, now, 3, ControlSucceeded))
		admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, 2, LivePosition{}, now, 4, ControlSucceeded))
		stale := controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 2, ControlSucceeded)
		if got := admitConnectionControl(t, e, stale); got.Code != DispositionConnectionControlFenced || got.Reason != ReasonStaleLiveEpoch {
			t.Fatalf("stale ack = %+v", got)
		}
		if e.state.liveEpoch != 2 || e.state.aggregateAcknowledged {
			t.Fatalf("stale ack changed epoch/ack: epoch=%d ack=%t", e.state.liveEpoch, e.state.aggregateAcknowledged)
		}
	})

	t.Run("loss guards cover awaiting hydrating live and recovering", func(t *testing.T) {
		cases := []struct {
			name, state, want string
		}{
			{"awaiting", string(lifecycleAwaitingAggregateAck), string(lifecycleAwaitingAggregateAck)},
			{"hydrating LIFE-T13", string(lifecycleHydrating), string(lifecycleAwaitingAggregateAck)},
			{"live LIFE-T16", string(lifecycleLive), string(lifecycleRecovering)},
			{"recovering LIFE-T21", string(lifecycleRecovering), string(lifecycleRecovering)},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				now := start.Add(10 * time.Second)
				e := aggregateEngine(t, binding, RunModeLive, &now)
				defer closeAndWait(t, e)
				admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
				e.mu.Lock()
				e.state.lifecycle = lifecycle(tc.state)
				if lifecycle(tc.state) == lifecycleLive {
					supported := start.Add(5 * time.Second)
					e.state.committedT = &supported
					e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(supported)
				}
				e.mu.Unlock()
				loss := controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 0, ControlFailed)
				if got := admitConnectionControl(t, e, loss); got.Code != DispositionConnectionControlApplied || string(e.observeTimeLifecycle().Lifecycle) != tc.want {
					t.Fatalf("loss = %+v lifecycle=%s want=%s", got, e.observeTimeLifecycle().Lifecycle, tc.want)
				}
				if e.state.liveEpochActive || e.state.aggregateAcknowledged {
					t.Fatal("loss retained active acknowledgement")
				}
				staleAggregate := liveAggregate(binding, "AAA", start, 1, 2)
				if got := admitProductionAggregate(t, e, staleAggregate); got.Code != DispositionAggregateFenced || got.Reason != ReasonStaleLiveEpoch {
					t.Fatalf("old-epoch aggregate after loss = %+v", got)
				}
			})
		}
	})

	t.Run("pre-session ack survives to S unless its epoch is lost", func(t *testing.T) {
		for _, lose := range []bool{false, true} {
			now := start.Add(-10 * time.Second)
			e := aggregateEngine(t, binding, RunModeLive, &now)
			admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
			admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, now, 2, ControlSucceeded))
			admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 2, ControlSucceeded))
			if lose {
				admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
			}
			now = start
			_, timer := e.AdmitTimer(context.Background())
			if got := awaitTimerDisposition(t, timer); got.Code != DispositionTimerApplied {
				t.Fatalf("timer = %+v", got)
			}
			want := lifecycleHydrating
			if lose {
				want = lifecycleAwaitingAggregateAck
			}
			if got := e.observeTimeLifecycle().Lifecycle; got != want {
				t.Fatalf("lose=%t lifecycle=%s want=%s", lose, got, want)
			}
			closeAndWait(t, e)
		}
	})

	t.Run("TQ-only failure is deferred and ingress ambiguity suppresses atomically", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		tq := controlFact(binding.Identity(), TradeQuoteSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 9, ControlFailed)
		if got := admitConnectionControl(t, e, tq); got.Code != DispositionConnectionControlDeferred || e.observeTimeLifecycle().Lifecycle != lifecycleAwaitingAggregateAck {
			t.Fatalf("T/Q failure = %+v lifecycle=%s", got, e.observeTimeLifecycle().Lifecycle)
		}
		ingress := controlFact(binding.Identity(), IngressIntegrityFailure, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlAmbiguous)
		got := admitConnectionControl(t, e, ingress)
		view := e.observePublication()
		if got.Code != DispositionIngressIntegrity || got.SuppressionDisposition != SuppressionSameBindingRecoveryAllowed ||
			e.observeTimeLifecycle().Lifecycle != lifecycleSuppressed || view.kind != publicationNormal || view.bindingIdentity != binding.Identity() ||
			view.lifecycleReason != lifecycleReasonIngressIntegrity || view.lastDisposition != DispositionIngressIntegrity {
			t.Fatalf("ingress disposition=%+v publication=%+v", got, view)
		}
	})

	inspectC5EngineOwnership(t)
}

func TestC8RecoveryExhaustionRequiresEngineOwnedAttemptHistory(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)
	foreign := binding.Identity()
	if foreign[len(foreign)-1] == '0' {
		foreign = foreign[:len(foreign)-1] + "1"
	} else {
		foreign = foreign[:len(foreign)-1] + "0"
	}

	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
	for name, input := range map[string]RecoveryExhaustionInput{
		"active epoch":    {SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: binding.Identity(), Attempts: 1},
		"foreign binding": {SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: foreign, Attempts: 1},
	} {
		admission, completion := e.AdmitRecoveryExhaustion(context.Background(), input)
		if admission != AdmissionAdmitted || completion == nil {
			t.Fatalf("%s admission=%s", name, admission)
		}
		if got := <-completion; got.Code != DispositionConnectionControlRejected || e.state.lifecycle == lifecycleSuppressed {
			t.Fatalf("%s exhaustion=%+v lifecycle=%s", name, got, e.state.lifecycle)
		}
	}
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 0, ControlFailed))
	wrongCount := RecoveryExhaustionInput{SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: binding.Identity(), Attempts: 2}
	_, wrongCompletion := e.AdmitRecoveryExhaustion(context.Background(), wrongCount)
	if got := <-wrongCompletion; got.Code != DispositionConnectionControlRejected || e.state.lifecycle == lifecycleSuppressed {
		t.Fatalf("wrong-count exhaustion=%+v lifecycle=%s", got, e.state.lifecycle)
	}
	exact := RecoveryExhaustionInput{SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: binding.Identity(), Attempts: 1}
	_, completion := e.AdmitRecoveryExhaustion(context.Background(), exact)
	got := <-completion
	view := e.ObserveOperational()
	if got.Code != DispositionRecoveryExhausted || got.Reason != ReasonRecoveryExhausted || got.SuppressionDisposition != SuppressionSameBindingRecoveryAllowed ||
		view.Lifecycle != string(lifecycleSuppressed) || view.LifecycleReason != string(lifecycleReasonRecoveryExhausted) {
		t.Fatalf("exact exhaustion=%+v view=%+v", got, view)
	}
}

func TestPHRRetryEnginePacesEveryAttemptAndPreservesFacts(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(20 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)

	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
	e.mu.Lock()
	e.state.lifecycle = lifecycleRecovering
	e.state.liveEpochActive = false
	e.state.connectionControl.recoveryAttempts = 0
	e.scheduleRecoveryLocked(nil)
	e.mu.Unlock()

	for ordinal := uint64(1); ordinal <= 5; ordinal++ {
		command, err := e.IssueScheduledRecoveryCommand()
		if err != nil || command.RetryOrdinal() != ordinal || command.FailedEpoch() != ordinal ||
			command.EarliestAt().Sub(command.IssuedAt()) != newRecoveryPolicy(time.Second, 30*time.Second).delay(ordinal) {
			t.Fatalf("ordinal %d command=%+v err=%v", ordinal, command, err)
		}
		now = command.EarliestAt()
		input, err := NewScheduledRecoveryInput(command)
		if err != nil {
			t.Fatal(err)
		}
		_, completion := e.AdmitScheduledRecovery(context.Background(), input)
		if got := <-completion; got.Code != DispositionRecoveryScheduled {
			t.Fatalf("ordinal %d schedule=%+v", ordinal, got)
		}
		if got := e.ObserveOperational().Connection.RecoveryAttempts; got != ordinal-1 {
			t.Fatalf("ordinal %d schedule reset attempts to %d", ordinal, got)
		}
		epoch := ordinal + 1
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, epoch, LivePosition{}, now, epoch*10, ControlSucceeded))
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionEstablished, epoch, LivePosition{ConnectionEpoch: epoch, FrameSequence: 1}, now, epoch*10+1, ControlSucceeded))
		admitConnectionControl(t, e, controlFact(binding.Identity(), AuthenticationResult, epoch, LivePosition{ConnectionEpoch: epoch, FrameSequence: 2}, now, epoch*10+2, ControlFailed))
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, epoch, LivePosition{ConnectionEpoch: epoch, FrameSequence: 3}, now, 0, ControlFailed))
		if got := e.ObserveOperational().Connection.RecoveryAttempts; got != ordinal {
			t.Fatalf("ordinal %d consecutive attempts=%d", ordinal, got)
		}
	}

	_, exhausted := e.AdmitRecoveryExhaustion(context.Background(), RecoveryExhaustionInput{SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: binding.Identity(), Attempts: 5})
	if got := <-exhausted; got.Code != DispositionRecoveryExhausted {
		t.Fatalf("exhaustion=%+v", got)
	}
	if _, err := e.IssueScheduledRecoveryCommand(); err == nil {
		t.Fatal("exhaustion retained automatic recovery authority")
	}
	rejected := admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 7, LivePosition{}, now, 100, ControlSucceeded))
	if rejected.Code != DispositionConnectionControlRejected {
		t.Fatalf("sixth automatic attempt=%+v", rejected)
	}
}

func controlFact(binding string, kind ConnectionControlKind, epoch uint64, position LivePosition, at time.Time, token uint64, outcome ConnectionControlOutcome) ConnectionControlInput {
	return ConnectionControlInput{SchemaVersion: ConnectionControlSchemaV1, BindingIdentity: binding, Kind: kind,
		ConnectionEpoch: epoch, Position: position, ReceiptTime: at.UTC(), CommandToken: token, Outcome: outcome}
}

func admitConnectionControl(t *testing.T, e *Engine, input ConnectionControlInput) ConnectionControlDisposition {
	t.Helper()
	result, completion := e.AdmitConnectionControl(context.Background(), input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("control admission = %s", result)
	}
	select {
	case got := <-completion:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for control disposition")
		return ConnectionControlDisposition{}
	}
}

func admitProductionAggregate(t *testing.T, e *Engine, input AggregateInput) AggregateDisposition {
	t.Helper()
	result, completion := e.AdmitAggregate(context.Background(), input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("aggregate admission = %s", result)
	}
	return awaitAggregateDisposition(t, completion)
}

func inspectC5EngineOwnership(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	set := token.NewFileSet()
	transitionDefinitions, publicationDefinitions, queueFields := 0, 0, 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, filepath.Join(".", entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if strings.Contains(imp.Path.Value, "/internal/massive") {
				t.Fatalf("engine imports adapter package in %s", entry.Name())
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.FuncDecl:
				if value.Name.Name == "transitionLifecycleLocked" {
					transitionDefinitions++
				}
				if value.Name.Name == "storePublication" {
					publicationDefinitions++
				}
				if value.Recv != nil && ast.IsExported(value.Name.Name) && (strings.Contains(value.Name.Name, "SetLifecycle") || strings.Contains(value.Name.Name, "SetEpoch") || strings.Contains(value.Name.Name, "TransitionLifecycle")) {
					t.Fatalf("exported transition authority %s", value.Name.Name)
				}
			case *ast.TypeSpec:
				if value.Name.Name == "Engine" {
					if structure, ok := value.Type.(*ast.StructType); ok {
						for _, field := range structure.Fields.List {
							for _, name := range field.Names {
								if name.Name == "queue" {
									queueFields++
								}
							}
						}
					}
				}
			}
			return true
		})
	}
	if transitionDefinitions != 1 || publicationDefinitions != 1 || queueFields != 1 {
		t.Fatalf("ownership definitions: lifecycle=%d publication=%d FIFO=%d", transitionDefinitions, publicationDefinitions, queueFields)
	}
}
