package engine

import (
	"context"
	"testing"
	"time"
)

func TestC8LiveCoverageFenceAuthorityAndAdvance(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleLive)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = 1
	e.state.hydration.fenceThrough = 1
	e.state.hydration.fenceMarkerOrdinal = 1
	e.state.hydration.supportedThrough = immutableTime(now)
	e.mu.Unlock()
	now = now.Add(2 * time.Second)
	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, 1, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("coverage admission=%s", admission)
	}
	if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("coverage disposition=%+v", got)
	}
	if got, err := command.Wait(context.Background()); err != nil || got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("command completion=%+v err=%v", got, err)
	}
	if admission, timer := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || timer == nil || (<-timer).Code != DispositionTimerApplied {
		t.Fatal("coverage timer was not applied")
	}
	if view := e.ObserveOperational(); view.Watermark == nil || *view.Watermark != now {
		t.Fatalf("coverage did not support exact T: %+v", view)
	}

	// A retained command cannot be replayed, and a forged command cannot claim
	// completion or replace the current engine-issued authority.
	_, replayCompletion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if got := <-replayCompletion; got.Code != DispositionLiveCoverageFenceFenced {
		t.Fatalf("replayed command=%+v", got)
	}
	foreign := command
	foreign.token++
	foreign.done = make(chan LiveCoverageFenceDisposition, 1)
	foreignFact, _ := NewLiveCoverageFenceInput(foreign, LiveCoverageFenceComplete, 1, 3, now)
	_, foreignCompletion := e.AdmitLiveCoverageFence(context.Background(), foreignFact)
	if got := <-foreignCompletion; got.Code != DispositionLiveCoverageFenceFenced {
		t.Fatalf("foreign command=%+v", got)
	}
	select {
	case got := <-foreign.done:
		t.Fatalf("foreign command received engine authority: %+v", got)
	case <-time.After(10 * time.Millisecond):
	}

	cancelCommand, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	_, canceled := e.AdmitLiveCoverageFenceCancellation(context.Background(), cancelCommand)
	if got := <-canceled; got.Code != DispositionLiveCoverageFenceRejected {
		t.Fatalf("cancellation=%+v", got)
	}
	if _, err := cancelCommand.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	finalCommand, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatalf("cancellation left command stuck: %v", err)
	}
	_, finalCancellation := e.AdmitLiveCoverageFenceCancellation(context.Background(), finalCommand)
	<-finalCancellation
}
