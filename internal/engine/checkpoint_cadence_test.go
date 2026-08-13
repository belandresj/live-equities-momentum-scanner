package engine

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
)

func TestC7CODEC01CompleteSemanticImageRoundTrip(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	t0 := start.Add(30 * time.Second)
	now := t0.Add(time.Second)
	source := aggregateEngine(t, binding, RunModeLive, &now)
	for i := 0; i < 30; i++ {
		input := liveAggregate(binding, "AAA", start.Add(time.Duration(i)*time.Second), 1, uint64(i+1))
		input.Values.Volume = 10_000
		applyAggregate(t, source, input, DispositionAggregateInserted, ReasonNone)
	}
	proveAggregateCoverage(t, source, "AAA", start, t0)
	source.mu.Lock()
	source.state.lifecycle = lifecycleLive
	source.applyAggregateCandidateLocked(t0, t0)
	source.state.aggregateEvaluator.current = source.stageAggregateEvaluationAtLocked(t0, t0)
	source.mu.Unlock()
	image := projectForTest(t, source).Image
	var encoded bytes.Buffer
	_, checksum, err := checkpoint.Encode(context.Background(), &encoded, image, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := checkpoint.Decode(context.Background(), bytes.NewReader(encoded.Bytes()), 8<<20)
	if err != nil || candidate.Checksum != checksum || !reflect.DeepEqual(candidate.Image, image) {
		t.Fatalf("complete image decode err=%v", err)
	}
	target := freshBoundEngine(t, binding, &now)
	admission, completion := target.AdmitCheckpointInstall(context.Background(), candidate)
	if admission != AdmissionAdmitted || (<-completion).Disposition != CheckpointInstalled {
		t.Fatalf("decoded candidate admission=%s", admission)
	}
	closeAndWait(t, source)
	closeAndWait(t, target)
}

// TestC7CADENCE01EngineWriterTrace is the engine half of P-C7-CADENCE. It
// proves aligned committed-T eligibility, nonblocking owner progress, exact
// outstanding identity, and stale/wrong result fencing. Filesystem durability
// is proved separately by P-C7-STORE.
func TestC7CADENCE01EngineWriterTrace(t *testing.T) {
	binding := testBinding(t)
	store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: binding.Identity(), ArtifactByteLimit: 8 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	store.SetStepHookForTest(func(step checkpoint.WriteStep) error {
		if step == checkpoint.StepPayloadEncode {
			select {
			case <-entered:
			default:
				close(entered)
			}
			<-release
		}
		return nil
	})
	writer, err := checkpoint.NewWriter(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	now := binding.SessionStart()
	delay := time.Duration(0)
	e, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return now }, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay, CheckpointSubmitter: writer})
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatal(got)
	}

	advance := func(at time.Time) {
		now = at
		e.mu.Lock()
		e.state.lifecycle = lifecycleLive
		e.applyAggregateCandidateLocked(at, at)
		e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(at, at)
		e.maybeSubmitCheckpointLocked(at)
		e.mu.Unlock()
		if offset := at.Sub(binding.SessionStart()); offset > 0 && offset%(30*time.Second) == 0 {
			deadline := time.Now().Add(2 * time.Second)
			for {
				operations := e.CheckpointOperations()
				if operations.ProjectionStarted == operations.Projected+operations.ProjectionRejected {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("checkpoint projection did not settle: %+v", operations)
				}
				time.Sleep(time.Millisecond)
			}
		}
	}
	start := binding.SessionStart()
	for _, offset := range []time.Duration{time.Second, 17 * time.Second, 29 * time.Second} {
		advance(start.Add(offset))
		if got := e.CheckpointOperations(); got.Submitted != 0 {
			t.Fatalf("nonaligned %s submitted: %+v", offset, got)
		}
	}
	advance(start.Add(30 * time.Second))
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not pause")
	}
	advance(start.Add(31 * time.Second))
	if got := e.CheckpointOperations(); got.Submitted != 1 {
		t.Fatalf("31s submitted: %+v", got)
	}
	advance(start.Add(45 * time.Second))
	advance(start.Add(59 * time.Second))
	advance(start.Add(60 * time.Second))
	advance(start.Add(90 * time.Second))
	e.mu.Lock()
	committed := *e.state.committedT
	e.mu.Unlock()
	if committed != start.Add(90*time.Second) {
		t.Fatalf("writer backlog blocked T: %s", committed)
	}
	if got := e.CheckpointOperations(); !got.reconciles() || got.Submitted != 3 || got.Outstanding != 2 || got.Superseded != 1 {
		t.Fatalf("paused engine accounting=%+v", got)
	}
	if got := writer.Accounting(); !got.Reconciles() || got.InProgress != 1 || got.Pending != 1 || got.Superseded != 1 {
		t.Fatalf("paused writer accounting=%+v", got)
	}

	close(release)
	results := map[uint64]checkpoint.TerminalResult{}
	for len(results) != 2 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := writer.NextResult(ctx)
		cancel()
		if err != nil {
			t.Fatalf("writer results=%+v err=%v", results, err)
		}
		results[result.RequestID] = result
	}
	wrong := results[3]
	invalidTerminals := []checkpoint.TerminalResult{results[3], results[3], results[3], results[3]}
	invalidTerminals[0].Step = checkpoint.StepFileSync
	invalidTerminals[1].Disposition, invalidTerminals[1].Step, invalidTerminals[1].Reason = checkpoint.TerminalFailed, "", "write_failed"
	invalidTerminals[2].Disposition, invalidTerminals[2].Reason = checkpoint.TerminalSuperseded, "bad_reason"
	invalidTerminals[3].Disposition, invalidTerminals[3].Step, invalidTerminals[3].Reason = checkpoint.TerminalFailed, checkpoint.WriteStep("unknown_step"), "write_failed"
	for _, invalid := range invalidTerminals {
		admission, completion := e.AdmitCheckpointTerminal(context.Background(), invalid)
		if admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionCheckpointTerminalFenced {
			t.Fatalf("inconsistent terminal consumed: %+v", invalid)
		}
		if got := e.CheckpointOperations(); got.Outstanding != 2 {
			t.Fatalf("inconsistent terminal consumed outstanding: %+v", got)
		}
	}
	wrong.BindingIdentity = "wrong"
	if admission, completion := e.AdmitCheckpointTerminal(context.Background(), wrong); admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionCheckpointTerminalFenced {
		t.Fatalf("wrong binding admission=%s", admission)
	}
	for _, id := range []uint64{1, 3} {
		admission, completion := e.AdmitCheckpointTerminal(context.Background(), results[id])
		if admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionCheckpointTerminalApplied {
			t.Fatalf("terminal %d admission=%s", id, admission)
		}
	}
	admission, completion := e.AdmitCheckpointTerminal(context.Background(), results[1])
	if admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionCheckpointTerminalFenced {
		t.Fatalf("duplicate admission=%s", admission)
	}
	if got := e.CheckpointOperations(); !got.reconciles() || got.Submitted != 3 || got.Outstanding != 0 || got.Completed != 2 || got.Superseded != 1 || got.Fenced != 6 || got.LastSuccessfulT0 != start.Add(90*time.Second) {
		t.Fatalf("terminal engine accounting=%+v", got)
	}
	if got := writer.Accounting(); !got.Reconciles() || got.Completed != 2 || got.Superseded != 1 {
		t.Fatalf("terminal writer accounting=%+v", got)
	}

	writer.Close()
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := writer.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}
	closeAndWait(t, e)
}

func TestCKHOTSameT0CorrectionRejectsDetachedProjection(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	t0 := start.Add(30 * time.Second)
	now := t0.Add(time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	input := liveAggregate(binding, "AAA", start, 1, 1)
	applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
	proveAggregateCoverage(t, e, "AAA", start, t0)
	submitter := &checkpointDiscardSubmitter{}
	e.mu.Lock()
	e.checkpointSubmitter = submitter
	e.state.lifecycle = lifecycleLive
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.applyAggregateCandidateLocked(t0, t0)
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(t0, t0)
	e.maybeSubmitCheckpointLocked(now)
	// Hold the test-owned projection before its first continuation.
	e.queue = nil
	e.internalQueued = 0
	e.state.checkpointProjectionQueued = false
	if operations := e.state.checkpointOperations; operations.ProjectionInProgress != 1 || !operations.Reconciles() {
		e.mu.Unlock()
		t.Fatalf("active projection accounting=%+v", operations)
	}
	correction := freezeAggregateInput(input)
	correction.Values.Close, correction.Values.High, correction.Values.VWAP = 11, 11, 11
	correction.Live.FrameSequence = 2
	if code, reason := e.applyAggregateLocked(correction, now); code != DispositionAggregateRevised || reason != ReasonNone {
		e.mu.Unlock()
		t.Fatalf("pre-T0 correction=%s/%s", code, reason)
	}
	for e.state.checkpointProjectionActive {
		e.continueCheckpointProjectionLocked()
	}
	operations := e.state.checkpointOperations
	e.mu.Unlock()
	if operations.ProjectionInProgress != 0 || operations.ProjectionRejected != 1 || operations.LastProjectionFailure != "pre_t0_mutation" || submitter.submitted.Load() != 0 || !operations.Reconciles() {
		t.Fatalf("dirty detached image escaped operations=%+v submitted=%d", operations, submitter.submitted.Load())
	}
	closeAndWait(t, e)
}

func TestCKHOTIncrementalProjectionResolvesMaintainedCommittedMark(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	t0 := start.Add(30 * time.Second)
	now := t0
	e := aggregateEngine(t, binding, RunModeLive, &now)

	pre := liveAggregate(binding, "AAA", t0.Add(-time.Second), 1, 1)
	pre.Values.Open, pre.Values.High, pre.Values.Low, pre.Values.Close, pre.Values.VWAP = 10, 11, 9, 10, 10
	applyAggregate(t, e, pre, DispositionAggregateInserted, ReasonNone)
	proveAggregateCoverage(t, e, "AAA", start, t0)

	e.mu.Lock()
	e.applyAggregateCandidateLocked(t0, t0)
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(t0, t0)
	wantEvaluation := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	e.mu.Unlock()

	post := liveAggregate(binding, "AAA", t0.Add(time.Second), 1, 2)
	post.Values.Open, post.Values.High, post.Values.Low, post.Values.Close, post.Values.VWAP = 20, 21, 19, 20, 20
	now = post.WindowEnd
	applyAggregate(t, e, post, DispositionAggregateInserted, ReasonNone)

	e.mu.Lock()
	symbolIndex := e.state.binding.index["AAA"]
	state := e.state.binding.symbols[symbolIndex].aggregates
	e.compactSymbolLocked(state, e.state.binding, "AAA", post.WindowEnd.Add(correctionHorizon+time.Nanosecond))
	if state.committedLatest == nil || state.committedLatest.windowStart != pre.WindowStart || state.olderLatest == nil || state.olderLatest.windowStart != post.WindowStart || len(state.tail) != 0 {
		e.mu.Unlock()
		t.Fatalf("stalled-T projection shape committed=%+v older=%+v tail=%d", state.committedLatest, state.olderLatest, len(state.tail))
	}
	store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: binding.Identity(), ArtifactByteLimit: 8 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		e.mu.Unlock()
		t.Fatal(err)
	}
	writer, err := checkpoint.NewWriter(context.Background(), store)
	if err != nil {
		e.mu.Unlock()
		t.Fatal(err)
	}
	e.checkpointSubmitter = writer
	e.state.lifecycle = lifecycleLive
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.committedT = immutableTime(t0)
	e.state.aggregateEvaluator.current.at = t0
	// Production cadence starts only after the timer's evaluator transition.
	// Keep the target at the already-committed T0 while the accepted forward
	// state remains in the graph, matching a delayed live evaluation boundary.
	e.delay = now.Sub(t0)
	e.mu.Unlock()
	timerAdmission, timerCompletion := e.AdmitTimer(context.Background())
	timerDisposition := awaitTimerDisposition(t, timerCompletion)
	if timerAdmission != AdmissionAdmitted || timerDisposition.Code != DispositionTimerApplied {
		t.Fatalf("production timer admission=%s disposition=%+v", timerAdmission, timerDisposition)
	}

	resultCtx, resultCancel := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := writer.NextResult(resultCtx)
	resultCancel()
	if err != nil || result.Disposition != checkpoint.TerminalCompleted || result.T0 != t0 {
		t.Fatalf("writer result=%+v err=%v operations=%+v", result, err, e.CheckpointOperations())
	}
	admission, completion := e.AdmitCheckpointTerminal(context.Background(), result)
	if admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionCheckpointTerminalApplied {
		t.Fatalf("terminal admission=%s result=%+v", admission, result)
	}
	loaded := store.Load(context.Background())
	if loaded.Disposition != checkpoint.LoadedLatest || loaded.Candidate.Image.T0 != t0 {
		t.Fatalf("completed artifact not discoverable: %+v", loaded)
	}
	projected := loaded.Candidate.Image.Symbols[symbolIndex]
	wantValues := checkpoint.Values{Open: pre.Values.Open, High: pre.Values.High, Low: pre.Values.Low, Close: pre.Values.Close, Volume: pre.Values.Volume,
		VWAP: pre.Values.VWAP, AverageTradeSize: pre.Values.AverageTradeSize, ATSProvenance: string(pre.Values.ATSProvenance)}
	if projected.OlderMark == nil || projected.OlderMark.WindowStart != pre.WindowStart || projected.OlderMark.Values != wantValues ||
		projected.CommittedMark == nil || projected.CommittedMark.WindowStart != pre.WindowStart || projected.CommittedMark.Values != wantValues {
		t.Fatalf("persisted older=%+v committed=%+v want=%+v", projected.OlderMark, projected.CommittedMark, pre.Values)
	}
	installNow := t0.Add(time.Second)
	target := freshBoundEngine(t, binding, &installNow)
	installAdmission, installCompletion := target.AdmitCheckpointInstall(context.Background(), loaded.Candidate)
	if installAdmission != AdmissionAdmitted {
		t.Fatalf("loaded candidate install admission=%s", installAdmission)
	}
	install := <-installCompletion
	if install.Disposition != CheckpointInstalled || install.Fact.T0 != t0 {
		t.Fatalf("loaded candidate install=%+v", install)
	}
	target.mu.Lock()
	installedT0 := target.state.committedT
	installedEvaluation := cloneAggregateEvaluation(target.state.aggregateEvaluator.current)
	target.mu.Unlock()
	if installedT0 == nil || *installedT0 != t0 || !aggregateEvaluationEqual(installedEvaluation, wantEvaluation) {
		t.Fatalf("installed checkpoint coherence T0=%v evaluation_equal=%t", installedT0, aggregateEvaluationEqual(installedEvaluation, wantEvaluation))
	}
	if operations := e.CheckpointOperations(); operations.ProjectionRejected != 0 || operations.Projected != 1 || operations.Submitted != 1 || operations.Completed != 1 || !operations.Reconciles() {
		t.Fatalf("projection operations=%+v", operations)
	}
	writer.Close()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	if err := writer.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}
	closeAndWait(t, target)
	closeAndWait(t, e)
}

func TestCKHOTNearCapacityInternalQueueAccountingIsConstantShape(t *testing.T) {
	const external = 8191
	e := &Engine{capacity: 8192, changed: make(chan struct{}), state: &engineState{clockMonotonic: true}}
	e.queue = make([]*queueNode, external)
	for index := range e.queue {
		e.queue[index] = &queueNode{kind: inputAggregate}
	}
	e.counters.started = external
	e.counters.resultsCommitted = external
	e.counters.admittedExternal = external
	e.state.checkpointProjectionActive = true
	e.state.checkpointProjection = &checkpointProjectionWork{started: time.Now()}
	e.enqueueCheckpointProjectionLocked()
	if len(e.queue) != 8192 || e.internalQueued != 1 || e.externalQueueOccupancyLocked() != external || !e.accountingCoherentLocked() {
		t.Fatalf("near-capacity continuation accounting queue=%d internal=%d external=%d coherent=%t", len(e.queue), e.internalQueued, e.externalQueueOccupancyLocked(), e.accountingCoherentLocked())
	}
	// Rejection makes the continuation stale but does not misclassify it as an
	// external admission while it remains physically queued.
	e.rejectCheckpointProjectionLocked(e.state.checkpointProjection, "projection_invariant")
	if e.internalQueued != 1 || e.externalQueueOccupancyLocked() != external || !e.accountingCoherentLocked() {
		t.Fatalf("stale continuation accounting internal=%d external=%d coherent=%t", e.internalQueued, e.externalQueueOccupancyLocked(), e.accountingCoherentLocked())
	}
	// Model the consumer's eventual dequeue of the stale tail node.
	e.queue = e.queue[:len(e.queue)-1]
	e.internalQueued--
	if e.internalQueued != 0 || e.externalQueueOccupancyLocked() != external || !e.accountingCoherentLocked() {
		t.Fatalf("dequeued continuation accounting internal=%d external=%d coherent=%t", e.internalQueued, e.externalQueueOccupancyLocked(), e.accountingCoherentLocked())
	}
}
