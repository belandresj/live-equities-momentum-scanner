package engine

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
)

const (
	DispositionCheckpointTerminalApplied DispositionCode = "checkpoint_terminal_applied"
	DispositionCheckpointTerminalFenced  DispositionCode = "checkpoint_terminal_fenced"
)

// CheckpointOperations is restart-local operational accounting. Its facts do
// not affect committed time, evaluator currentness, or ranking publication.
type CheckpointOperations struct {
	Submitted, Outstanding                            uint64
	Completed, Failed, Canceled, Superseded           uint64
	Fenced, ProjectionRejected, SubmitRejected        uint64
	CadenceIneligible, Eligible                       uint64
	PressureDeferred, ProjectionStarted, Projected    uint64
	ProjectionInProgress                              uint64
	LastAttemptedT0, LastProjectedT0, LastSubmittedT0 time.Time
	LastSuccessfulT0                                  time.Time
	LastProjectionTotal, LastProjectionLock           time.Duration
	LastProjectionSeal                                time.Duration
	LastProjectionFailure, LastSubmitFailure          string
}

type checkpointRequestIdentity struct {
	bindingIdentity  string
	artifactSequence uint64
	t0               time.Time
}

func (o CheckpointOperations) Reconciles() bool {
	return o.ProjectionInProgress <= 1 &&
		o.Eligible == o.PressureDeferred+o.ProjectionStarted &&
		o.ProjectionStarted == o.ProjectionInProgress+o.Projected+o.ProjectionRejected &&
		o.Projected == o.Submitted+o.SubmitRejected &&
		o.Submitted == o.Outstanding+o.Completed+o.Failed+o.Canceled+o.Superseded
}

func (o CheckpointOperations) reconciles() bool { return o.Reconciles() }

// AdmitCheckpointTerminal orders one writer terminal fact through the sole
// engine consumer. Unknown, duplicate, wrong-binding, and stale facts fence.
func (e *Engine) AdmitCheckpointTerminal(ctx context.Context, result checkpoint.TerminalResult) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || result.BindingIdentity == "" || result.RequestID == 0 || result.ArtifactSequence == 0 || result.T0.IsZero() {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputCheckpointTerminal, checkpointTerminal: result}, false)
}

func (e *Engine) CheckpointOperations() CheckpointOperations {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state.checkpointOperations
}

func (e *Engine) maybeSubmitCheckpointLocked(createdAt time.Time) {
	if e.checkpointSubmitter == nil {
		return
	}
	if e.state.checkpointProjectionActive {
		if e.state.binding != nil && e.state.committedT != nil {
			t0 := *e.state.committedT
			offset := t0.Sub(e.state.binding.sessionStart)
			if offset > 0 && offset%(30*time.Second) == 0 && (e.state.checkpointLastAttempted == nil || t0.After(*e.state.checkpointLastAttempted)) {
				e.state.checkpointLastAttempted = immutableTime(t0)
				e.state.checkpointOperations.Eligible++
				e.state.checkpointOperations.PressureDeferred++
				e.state.checkpointOperations.LastAttemptedT0 = t0
			}
		}
		e.enqueueCheckpointProjectionLocked()
		return
	}
	if e.mode != RunModeLive || e.state.binding == nil || e.state.lifecycle != lifecycleLive || e.state.committedT == nil ||
		!e.state.aggregateEvaluator.current.at.Equal(*e.state.committedT) {
		e.state.checkpointOperations.CadenceIneligible++
		return
	}
	t0 := *e.state.committedT
	offset := t0.Sub(e.state.binding.sessionStart)
	if offset <= 0 || offset%(30*time.Second) != 0 || e.state.checkpointLastAttempted != nil && !t0.After(*e.state.checkpointLastAttempted) {
		e.state.checkpointOperations.CadenceIneligible++
		return
	}
	e.state.checkpointLastAttempted = immutableTime(t0)
	e.state.checkpointOperations.Eligible++
	e.state.checkpointOperations.LastAttemptedT0 = t0
	e.state.checkpointOperations.ProjectionStarted++
	if e.state.checkpointSequence == math.MaxUint64 || e.state.checkpointRequestSequence == math.MaxUint64 {
		e.state.checkpointOperations.ProjectionRejected++
		e.state.checkpointOperations.LastProjectionFailure = "sequence_exhausted"
		return
	}
	attemptStarted := diagnosticMonotonicClock()
	e.state.checkpointProjectionActive = true
	e.state.checkpointOperations.ProjectionInProgress = 1
	e.state.checkpointProjectionDirty = false
	e.state.checkpointProjectionT0 = t0
	builder, ok := checkpoint.NewProjectionBuilder(checkpoint.SchemaV1, checkpoint.ProducerLive, projectCheckpointBinding(e.state.binding), t0, createdAt, e.state.checkpointSequence+1, uint64(len(e.state.binding.symbols)))
	if !ok {
		sealElapsed := diagnosticMonotonicClock().Sub(attemptStarted)
		e.state.checkpointOperations.LastProjectionSeal = sealElapsed
		e.state.checkpointOperations.LastProjectionLock = sealElapsed
		e.state.checkpointProjectionActive = false
		e.state.checkpointOperations.ProjectionInProgress = 0
		e.state.checkpointOperations.ProjectionRejected++
		e.state.checkpointOperations.LastProjectionFailure = "projection_invariant"
		return
	}
	evaluator := checkpointProjectionEvaluator(e.state.aggregateEvaluator)
	committedMarkers := checkpointProjectionCommittedMarkers(e.state.binding)
	sealElapsed := diagnosticMonotonicClock().Sub(attemptStarted)
	e.state.checkpointOperations.LastProjectionSeal = sealElapsed
	e.state.checkpointOperations.LastProjectionLock = sealElapsed
	e.state.checkpointProjection = &checkpointProjectionWork{
		bindingIdentity: e.state.binding.identity, t0: t0, started: attemptStarted,
		evaluator: evaluator, committedMarkers: committedMarkers,
		builder: builder, sequence: e.state.checkpointSequence + 1, population: len(e.state.binding.symbols),
		maxLock: sealElapsed,
	}
	e.enqueueCheckpointProjectionLocked()
}

type checkpointProjectionWork struct {
	bindingIdentity  string
	t0, started      time.Time
	evaluator        aggregateEvaluatorState
	committedMarkers []checkpointProjectionCommittedMarker
	builder          *checkpoint.ProjectionBuilder
	sequence         uint64
	population       int
	nextSymbol       int
	maxLock          time.Duration
}

type checkpointProjectionCommittedMarker struct {
	mark    committedAggregateMark
	present bool
}

func checkpointProjectionCommittedMarkers(binding *installedBinding) []checkpointProjectionCommittedMarker {
	markers := make([]checkpointProjectionCommittedMarker, len(binding.symbols))
	for index := range binding.symbols {
		state := binding.symbols[index].aggregates
		if state == nil || state.committedLatest == nil {
			continue
		}
		markers[index] = checkpointProjectionCommittedMarker{mark: *state.committedLatest, present: true}
	}
	return markers
}

func checkpointProjectionEvaluator(source aggregateEvaluatorState) aggregateEvaluatorState {
	result := aggregateEvaluatorState{
		invalidMarks: make(map[int]invalidMarkEvidence, len(source.invalidMarks)),
		coverage:     make(map[int]aggregateCoverageConsequence, len(source.coverage)),
	}
	for index, value := range source.invalidMarks {
		result.invalidMarks[index] = value
	}
	for index, value := range source.coverage {
		result.coverage[index] = value
	}
	return result
}

func (e *Engine) enqueueCheckpointProjectionLocked() {
	if !e.state.checkpointProjectionActive || e.state.checkpointProjectionQueued || len(e.queue) >= e.capacity {
		return
	}
	e.queue = append(e.queue, &queueNode{kind: inputCheckpointProjectionContinue})
	e.internalQueued++
	e.state.checkpointProjectionQueued = true
	e.broadcastLocked()
}

// continueCheckpointProjectionLocked copies exactly one symbol from the sole
// owner, then the consumer places the continuation at the FIFO tail. Accepted
// market inputs therefore interleave between bounded projection slices.
func (e *Engine) continueCheckpointProjectionLocked() {
	work := e.state.checkpointProjection
	if !e.state.checkpointProjectionActive || work == nil {
		return
	}
	started := diagnosticMonotonicClock()
	if e.sealed {
		e.rejectCheckpointProjectionLocked(work, "shutdown")
		return
	}
	if e.state.binding == nil || e.state.binding.identity != work.bindingIdentity {
		e.rejectCheckpointProjectionLocked(work, "ownership_invalidated")
		return
	}
	if e.state.lifecycle != lifecycleLive {
		e.rejectCheckpointProjectionLocked(work, "ownership_invalidated")
		return
	}
	if work.nextSymbol >= work.population || len(work.committedMarkers) != work.population {
		e.rejectCheckpointProjectionLocked(work, "projection_invariant")
		return
	}
	projected, err := projectCheckpointSymbol(e.state.binding, &e.state.binding.symbols[work.nextSymbol], work.evaluator, work.committedMarkers[work.nextSymbol], work.nextSymbol, work.t0)
	elapsed := diagnosticMonotonicClock().Sub(started)
	if elapsed > work.maxLock {
		work.maxLock = elapsed
	}
	if err != nil {
		reason := "projection_invariant"
		if errors.Is(err, errCheckpointSealedCommittedMarker) {
			reason = "sealed_t0_marker_invalid"
		}
		e.rejectCheckpointProjectionLocked(work, reason)
		return
	}
	if !work.builder.AddSymbol(work.nextSymbol, projected) {
		e.rejectCheckpointProjectionLocked(work, "projection_invariant")
		return
	}
	work.nextSymbol++
	if work.nextSymbol < work.population {
		return
	}
	if e.state.checkpointProjectionDirty {
		e.rejectCheckpointProjectionLocked(work, "pre_t0_mutation")
		return
	}
	if e.state.checkpointSequence+1 != work.sequence {
		e.rejectCheckpointProjectionLocked(work, "sequence_changed")
		return
	}
	e.state.checkpointProjectionActive = false
	e.state.checkpointOperations.ProjectionInProgress = 0
	e.state.checkpointProjection = nil
	e.state.checkpointOperations.LastProjectionTotal = diagnosticMonotonicClock().Sub(work.started)
	e.state.checkpointOperations.LastProjectionLock = work.maxLock
	e.state.checkpointSequence = work.sequence
	e.state.checkpointOperations.Projected++
	e.state.checkpointOperations.LastProjectedT0 = work.t0
	e.state.checkpointRequestSequence++
	request, ok := work.builder.Finish(work.bindingIdentity, e.state.checkpointRequestSequence)
	if !ok {
		e.state.checkpointOperations.SubmitRejected++
		e.state.checkpointOperations.LastSubmitFailure = "request_invalid"
		return
	}
	result := e.checkpointSubmitter.Submit(request)
	if result.Disposition != checkpoint.SubmitAccepted {
		e.state.checkpointOperations.SubmitRejected++
		e.state.checkpointOperations.LastSubmitFailure = "writer_rejected"
		return
	}
	if e.state.checkpointOutstanding == nil {
		e.state.checkpointOutstanding = make(map[uint64]checkpointRequestIdentity)
	}
	e.state.checkpointOutstanding[request.RequestID] = checkpointRequestIdentity{bindingIdentity: request.BindingIdentity, artifactSequence: request.ArtifactSequence, t0: request.T0}
	e.state.checkpointLastSubmitted = immutableTime(work.t0)
	e.state.checkpointOperations.LastSubmittedT0 = work.t0
	e.state.checkpointOperations.Submitted++
	e.state.checkpointOperations.Outstanding++
	if result.Superseded != nil {
		e.applyCheckpointTerminalLocked(*result.Superseded)
	}
}

func (e *Engine) rejectCheckpointProjectionLocked(work *checkpointProjectionWork, reason string) {
	if work != nil {
		e.state.checkpointOperations.LastProjectionTotal = diagnosticMonotonicClock().Sub(work.started)
		e.state.checkpointOperations.LastProjectionLock = work.maxLock
	}
	e.state.checkpointProjectionActive = false
	e.state.checkpointOperations.ProjectionInProgress = 0
	e.state.checkpointProjectionQueued = false
	e.state.checkpointProjection = nil
	e.state.checkpointOperations.ProjectionRejected++
	e.state.checkpointOperations.LastProjectionFailure = reason
	if e.state.checkpointProjectionDirty {
		e.state.checkpointProjectionDirty = false
	}
	return
}

func (e *Engine) applyCheckpointTerminalLocked(result checkpoint.TerminalResult) (DispositionCode, DispositionReason) {
	request, ok := e.state.checkpointOutstanding[result.RequestID]
	if !ok || e.state.binding == nil || result.BindingIdentity != e.state.binding.identity || result.BindingIdentity != request.bindingIdentity || result.ArtifactSequence != request.artifactSequence || result.T0 != request.t0 || !checkpoint.ValidTerminalResult(result) {
		e.state.checkpointOperations.Fenced++
		return DispositionCheckpointTerminalFenced, ReasonBinding
	}
	delete(e.state.checkpointOutstanding, result.RequestID)
	e.state.checkpointOperations.Outstanding--
	switch result.Disposition {
	case checkpoint.TerminalCompleted:
		e.state.checkpointOperations.Completed++
		e.state.checkpointOperations.LastSuccessfulT0 = result.T0
	case checkpoint.TerminalFailed:
		e.state.checkpointOperations.Failed++
	case checkpoint.TerminalCanceled:
		e.state.checkpointOperations.Canceled++
	case checkpoint.TerminalSuperseded:
		e.state.checkpointOperations.Superseded++
	default:
		e.state.checkpointOperations.Fenced++
		return DispositionCheckpointTerminalFenced, ReasonStructural
	}
	if !e.state.checkpointOperations.reconciles() {
		return DispositionAccountingIntegrity, ReasonAccounting
	}
	return DispositionCheckpointTerminalApplied, ReasonNone
}
