package engine

import (
	"context"
	"testing"
	"time"
)

// TestPLBRD2HandoffUnbufferedBatchBypassesGeneralFIFO is the engine boundary
// of P-LBR-D2-HANDOFF. It distinguishes a synchronous whole-batch rendezvous
// from enqueueing either the batch or its individual results into the general
// engine FIFO. Transport saturation/recovery is proved in package massive.
func TestPLBRD2HandoffUnbufferedBatchBypassesGeneralFIFO(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := aggregateEngine(t, binding, &now)
	defer closeAndWait(t, e)

	first, err := NewLiveAggregate(liveAggregate(binding, "AAA", binding.SessionStart(), 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewLiveAggregate(liveAggregate(binding, "AAA", binding.SessionStart().Add(time.Second), 1, 2))
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	e.beforeConsume = func(node *queueNode) {
		if node.kind == inputAggregate && node.aggregate.Live.FrameSequence == 1 {
			close(entered)
			<-release
		}
	}
	done := make(chan []LiveResult, 1)
	go func() {
		results, consumeErr := e.ConsumeLiveBatch(context.Background(), []LiveInput{first, second})
		if consumeErr != nil {
			done <- nil
			return
		}
		done <- results
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first live mutation did not reach the sole owner")
	}
	view := e.ObserveOperational()
	if view.QueueOccupancy != 0 || view.Admissions.OwnerInProgress != 1 || view.Admissions.Admitted != view.Admissions.Completed+1 {
		t.Fatalf("live batch accumulated in general FIFO: %+v", view.Admissions)
	}
	select {
	case <-done:
		t.Fatal("batch handoff completed before its first causal mutation")
	default:
	}
	close(release)
	results := <-done
	if len(results) != 2 || results[0].Admission != AdmissionAdmitted || results[1].Admission != AdmissionAdmitted ||
		results[0].Aggregate.EngineSequence+1 != results[1].Aggregate.EngineSequence {
		t.Fatalf("batch results/order = %+v", results)
	}
	final := e.ObserveOperational()
	if final.QueueOccupancy != 0 || final.Admissions.OwnerInProgress != 0 || !final.Admissions.Reconciles(0) || !final.Transitions.Reconciles() {
		t.Fatalf("final handoff accounting = %+v transitions=%+v", final.Admissions, final.Transitions)
	}
}
