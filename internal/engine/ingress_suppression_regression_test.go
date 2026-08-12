package engine

import (
	"context"
	"testing"
	"time"
)

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
	_, timer := e.AdmitTimer(context.Background())
	if timer == nil || (<-timer).Code != DispositionTimerApplied {
		t.Fatal("suppressed timer was not dispositioned")
	}
	rejected := admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 2, LivePosition{}, now, 2, ControlSucceeded))
	if rejected.Code != DispositionConnectionControlRejected {
		t.Fatalf("suppressed reconnect=%+v", rejected)
	}
	view := e.ObserveOperational()
	if view.Lifecycle != "suppressed" || view.LifecycleReason != "ingress_integrity" || view.Suppression != SuppressionSameBindingRecoveryAllowed || view.BindingIdentity != binding.Identity() {
		t.Fatalf("suppression reason/identity drifted: %+v", view)
	}
}
