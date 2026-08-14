package engine

import (
	"context"
	"testing"
	"time"
)

// Regression for an unsafe pre-baseline queue-capacity terminal: quiet timers
// preserve its first cause, and only the opaque engine command can leave
// suppression after the finite backoff.
func TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)

	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
	got := admitConnectionControl(t, e, controlFact(binding.Identity(), IngressIntegrityFailure, 1,
		LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlAmbiguous))
	if got.Code != DispositionIngressIntegrity || got.SuppressionDisposition != SuppressionSameBindingRecoveryAllowed {
		t.Fatalf("ingress suppression=%+v", got)
	}
	command, err := e.IssueScheduledRecoveryCommand()
	if err != nil || command.BindingIdentity() != binding.Identity() || command.FailedEpoch() != 1 || command.RetryOrdinal() != 1 ||
		command.EarliestAt().Sub(command.IssuedAt()) != time.Second {
		t.Fatalf("scheduled recovery command=%+v err=%v", command, err)
	}
	_, timer := e.AdmitTimer(context.Background())
	if timer == nil || (<-timer).Code != DispositionTimerApplied {
		t.Fatal("suppressed timer was not dispositioned")
	}
	rejected := admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 2, LivePosition{}, now, 2, ControlSucceeded))
	if rejected.Code != DispositionConnectionControlRejected {
		t.Fatalf("unscheduled reconnect=%+v", rejected)
	}
	view := e.ObserveOperational()
	if view.Lifecycle != "suppressed" || view.LifecycleReason != "ingress_integrity" || view.Suppression != SuppressionSameBindingRecoveryAllowed ||
		view.BindingIdentity != binding.Identity() {
		t.Fatalf("suppression reason/identity drifted: %+v", view)
	}
	input, err := NewScheduledRecoveryInput(command)
	if err != nil {
		t.Fatal(err)
	}
	_, early := e.AdmitScheduledRecovery(context.Background(), input)
	if got := <-early; got.Code != DispositionConnectionControlRejected || e.ObserveOperational().Lifecycle != "suppressed" {
		t.Fatalf("early recovery=%+v view=%+v", got, e.ObserveOperational())
	}
	now = command.EarliestAt()
	_, due := e.AdmitScheduledRecovery(context.Background(), input)
	if got := <-due; got.Code != DispositionRecoveryScheduled {
		t.Fatalf("due recovery=%+v", got)
	}
	view = e.ObserveOperational()
	if view.Lifecycle != "recovering" || view.LifecycleReason != "scheduled_recovery" || view.Suppression != "" || view.Connection.Active {
		t.Fatalf("scheduled recovery did not restore engine ownership: %+v", view)
	}
}

func TestPTQRRecoverableIngressLossRoutesToExactGapRecovery(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(20 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
	e.mu.Lock()
	supported := binding.SessionStart().Add(15 * time.Second)
	e.state.lifecycle = lifecycleLive
	e.state.committedT = immutableTime(supported)
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(supported)
	e.state.hydration.fenceReconciled = true
	e.mu.Unlock()

	got := admitConnectionControl(t, e, controlFact(binding.Identity(), IngressIntegrityFailure, 1,
		LivePosition{ConnectionEpoch: 1, FrameSequence: 9}, now, 0, ControlAmbiguous))
	view := e.ObserveOperational()
	if got.Code != DispositionIngressIntegrity || got.SuppressionDisposition != "" || view.Lifecycle != "recovering" || view.Suppression != "" ||
		view.Hydration.SupportedThrough != nil || view.CurrentMarketClaim || view.Connection.Active {
		t.Fatalf("recoverable ingress loss=%+v view=%+v", got, view)
	}
	e.mu.Lock()
	retained := immutableTimePointer(e.state.hydration.supportedT)
	e.mu.Unlock()
	if retained == nil || !retained.Equal(supported) {
		t.Fatalf("retained exact recovery boundary=%v want=%v", retained, supported)
	}
	if _, err := e.IssueScheduledRecoveryCommand(); err == nil {
		t.Fatal("direct gap recovery incorrectly issued a suppression command")
	}
}
