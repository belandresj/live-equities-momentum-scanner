package engine

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
)

type checkpointDiscardSubmitter struct {
	submitted atomic.Uint64
	requests  chan checkpoint.Request
}

func (s *checkpointDiscardSubmitter) Submit(request checkpoint.Request) checkpoint.SubmitResult {
	s.submitted.Add(1)
	if s.requests != nil {
		s.requests <- request
	}
	return checkpoint.SubmitResult{Disposition: checkpoint.SubmitAccepted}
}

func updateDurationMaximum(target *atomic.Int64, value time.Duration) {
	for value > time.Duration(target.Load()) && !target.CompareAndSwap(target.Load(), int64(value)) {
	}
}

// TestPCKHOTLiveMatureProjectionBound is the engine-owned portion of
// P-CKHOT-LIVE. It runs three accepted mature 6,000-symbol Activity
// projections through the production consumer and FIFO while live aggregate
// facts, ordinary timer/evaluator watermark advancement, and both snapshot
// views continue concurrently. Persistence, HTTP mapping, and restart are
// proved at the checkpoint/operations/API boundaries.
func TestPCKHOTLiveMatureProjectionBound(t *testing.T) {
	if testing.Short() {
		t.Skip("generated 6,000-symbol checkpoint hot-path acceptance")
	}
	fixture, target := generatedMatureActivityEngine(t)
	validateMatureActivityManifest(t, fixture, target)
	window := target.Add(-activityBlockDuration)
	duplicateValues := fixture.state.binding.symbols[0].aggregates.tail[window.Unix()].values
	submitter := &checkpointDiscardSubmitter{requests: make(chan checkpoint.Request, 3)}
	clockNanos := atomic.Int64{}
	clockNanos.Store(target.Add(10 * time.Minute).UnixNano())
	delay := time.Duration(0)
	e, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }, Capacity: 8192, RequiredReserve: 128, EvaluationDelay: &delay, CheckpointSubmitter: submitter})
	if err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	e.state = fixture.state
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(target, target)
	if err := validateAggregateEvaluation(e.state.aggregateEvaluator.current); err != nil {
		e.mu.Unlock()
		t.Fatalf("mature current evaluation: %v", err)
	}
	e.mu.Unlock()
	defer func() {
		e.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Wait(ctx); err != nil {
			t.Errorf("engine wait: %v", err)
		}
	}()

	var maxObserverDelay, maxDeliveryDelay atomic.Int64
	var captures atomic.Uint64
	observerStop := make(chan struct{})
	observerDone := make(chan struct{})
	go func() {
		defer close(observerDone)
		for {
			select {
			case <-observerStop:
				return
			default:
			}
			started := time.Now()
			_ = e.ObserveOperational()
			_ = e.ObserveSnapshot()
			updateDurationMaximum(&maxObserverDelay, time.Since(started))
			captures.Add(1)
			time.Sleep(time.Millisecond)
		}
	}()
	defer func() { close(observerStop); <-observerDone }()

	var plateau uint64
	var frame atomic.Uint64
	frame.Store(1)
	for trial := 1; trial <= 3; trial++ {
		at := target.Add(time.Duration(trial-1) * 30 * time.Second)
		clockNanos.Store(at.UnixNano())
		var before, sealed, after runtime.MemStats
		e.mu.Lock()
		e.state.hydration.supportedThrough = immutableTime(at.Add(2 * time.Second))
		if trial == 1 {
			runtime.ReadMemStats(&before)
			e.maybeSubmitCheckpointLocked(at)
			runtime.ReadMemStats(&sealed)
			e.mu.Unlock()
		} else {
			// Advance to the next aligned boundary through the real evaluator,
			// but isolate the subsequent projection seal allocation from that
			// distinct O(N) evaluation by temporarily withholding the submitter.
			e.checkpointSubmitter = nil
			e.mu.Unlock()
			admission, completion := e.AdmitTimer(context.Background())
			if admission != AdmissionAdmitted || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
				t.Fatalf("trial %d boundary timer admission=%s", trial, admission)
			}
			e.mu.Lock()
			e.checkpointSubmitter = submitter
			runtime.ReadMemStats(&before)
			e.maybeSubmitCheckpointLocked(at)
			runtime.ReadMemStats(&sealed)
			e.mu.Unlock()
		}
		if operations := e.CheckpointOperations(); operations.ProjectionInProgress != 1 || operations.LastProjectionSeal <= 0 || operations.LastProjectionLock < operations.LastProjectionSeal {
			t.Fatalf("trial %d projection seal not active/measured: %+v", trial, operations)
		}
		sealedMarkerBytes := uint64(len(fixture.state.binding.symbols)) * uint64(unsafe.Sizeof(checkpointProjectionCommittedMarker{}))
		producerStop := make(chan struct{})
		producerDone := make(chan struct{})
		var produced atomic.Uint64
		var producerFailure atomic.Pointer[string]
		go func() {
			defer close(producerDone)
			for {
				select {
				case <-producerStop:
					return
				default:
				}
				position := frame.Add(1)
				input := AggregateInput{SchemaVersion: AggregateSchemaV1, BindingIdentity: "mature-activity-6000", Source: AggregateSourceLive,
					Symbol: "S0000", WindowStart: window, WindowEnd: window.Add(time.Second), DeliveryTime: window.Add(time.Second),
					Values: duplicateValues,
					Live:   LivePosition{ConnectionEpoch: 1, FrameSequence: position}}
				started := time.Now()
				admission, completion := e.AdmitAggregate(context.Background(), input)
				if admission != AdmissionAdmitted || completion == nil {
					message := fmt.Sprintf("aggregate admission=%s", admission)
					producerFailure.Store(&message)
					return
				}
				result := <-completion
				updateDurationMaximum(&maxDeliveryDelay, time.Since(started))
				if result.Code != DispositionAggregateInserted && result.Code != DispositionAggregateExactDuplicate {
					message := fmt.Sprintf("aggregate disposition=%+v", result)
					producerFailure.Store(&message)
					return
				}
				produced.Add(1)
				time.Sleep(100 * time.Millisecond)
			}
		}()

		for second := 1; second <= 2; second++ {
			advanceTo := at.Add(time.Duration(second) * time.Second)
			clockNanos.Store(advanceTo.UnixNano())
			admission, completion := e.AdmitTimer(context.Background())
			disposition := awaitTimerDisposition(t, completion)
			if admission != AdmissionAdmitted || disposition.Code != DispositionTimerApplied {
				close(producerStop)
				<-producerDone
				t.Fatalf("trial %d active timer %d admission=%s disposition=%+v", trial, second, admission, disposition)
			}
			e.mu.Lock()
			committed := e.state.committedT
			active := e.state.checkpointProjectionActive
			e.mu.Unlock()
			if committed == nil || *committed != advanceTo || !active {
				close(producerStop)
				<-producerDone
				t.Fatalf("trial %d timer %d did not advance during projection: committed=%v active=%t", trial, second, committed, active)
			}
		}

		var request checkpoint.Request
		select {
		case request = <-submitter.requests:
		case <-producerDone:
			e.mu.Lock()
			sealed, lifecycle, failure := e.sealed, e.state.lifecycle, e.state.globalFailure
			e.mu.Unlock()
			if reason := producerFailure.Load(); reason != nil {
				t.Fatalf("trial %d producer stopped: %s sealed=%t lifecycle=%s failure=%t operations=%+v", trial, *reason, sealed, lifecycle, failure, e.CheckpointOperations())
			}
			t.Fatalf("trial %d producer stopped unexpectedly", trial)
		case <-time.After(2 * time.Minute):
			close(producerStop)
			<-producerDone
			t.Fatalf("trial %d projection deadline operations=%+v", trial, e.CheckpointOperations())
		}
		close(producerStop)
		<-producerDone
		if failure := producerFailure.Load(); failure != nil {
			t.Fatalf("trial %d %s", trial, *failure)
		}
		if produced.Load() < 100 {
			t.Fatalf("trial %d aggregate interleaving count=%d", trial, produced.Load())
		}
		terminal := checkpoint.TerminalResult{BindingIdentity: request.BindingIdentity, RequestID: request.RequestID, ArtifactSequence: request.ArtifactSequence, T0: request.T0, Disposition: checkpoint.TerminalCompleted}
		admission, completion := e.AdmitCheckpointTerminal(context.Background(), terminal)
		if admission != AdmissionAdmitted || completion == nil || (<-completion).Code != DispositionCheckpointTerminalApplied {
			t.Fatalf("trial %d terminal admission=%s", trial, admission)
		}
		operations := e.CheckpointOperations()
		view := e.ObserveOperational()
		if operations.ProjectionRejected != 0 || operations.LastProjectionLock > 500*time.Millisecond || time.Duration(maxObserverDelay.Load()) > time.Second || time.Duration(maxDeliveryDelay.Load()) > time.Second ||
			!operations.Reconciles() || !view.Admissions.Reconciles(view.QueueOccupancy) || !view.Transitions.Reconciles() || !view.Aggregates.Reconciles() {
			t.Fatalf("trial %d responsiveness operations=%+v view=%+v observer=%s delivery=%s", trial, operations, view, time.Duration(maxObserverDelay.Load()), time.Duration(maxDeliveryDelay.Load()))
		}
		runtime.ReadMemStats(&after)
		t.Logf("P_CKHOT_LIVE trial=%d projection_total=%s projection_seal=%s max_projection_lock=%s sealed_marker_value_bytes=%d seal_allocation_bytes=%d max_observer_delay=%s max_aggregate_delivery=%s aggregate_completions=%d snapshot_captures=%d allocation_bytes=%d watermark_advanced_seconds=2",
			trial, operations.LastProjectionTotal, operations.LastProjectionSeal, operations.LastProjectionLock, sealedMarkerBytes, sealed.TotalAlloc-before.TotalAlloc, time.Duration(maxObserverDelay.Load()), time.Duration(maxDeliveryDelay.Load()), produced.Load(), captures.Load(), after.TotalAlloc-before.TotalAlloc)
		runtime.GC()
		runtime.ReadMemStats(&after)
		t.Logf("P_CKHOT_LIVE trial=%d post_gc_heap_bytes=%d", trial, after.HeapAlloc)
		if trial == 1 {
			plateau = after.HeapAlloc
		} else if after.HeapAlloc > plateau+plateau/5 {
			t.Fatalf("trial %d post-GC heap grew from %d to %d", trial, plateau, after.HeapAlloc)
		}
		// The forced plateau GC is proof instrumentation between boundaries,
		// not checkpoint work in the next boundary's latency population.
		maxObserverDelay.Store(0)
		maxDeliveryDelay.Store(0)
	}
	if submitter.submitted.Load() != 3 {
		t.Fatalf("accepted ownership transfers=%d", submitter.submitted.Load())
	}
}
