package engine

import (
	"context"
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
	Submitted, Outstanding                     uint64
	Completed, Failed, Canceled, Superseded    uint64
	Fenced, ProjectionRejected, SubmitRejected uint64
	CadenceIneligible                          uint64
	LastSuccessfulT0                           time.Time
}

func (o CheckpointOperations) reconciles() bool {
	return o.Submitted == o.Outstanding+o.Completed+o.Failed+o.Canceled+o.Superseded
}

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
	projection := e.projectCheckpointLocked(createdAt)
	if projection.Disposition != CheckpointProjected {
		e.state.checkpointOperations.ProjectionRejected++
		return
	}
	if e.state.checkpointRequestSequence == ^uint64(0) {
		e.state.checkpointOperations.SubmitRejected++
		return
	}
	e.state.checkpointRequestSequence++
	request := checkpoint.Request{BindingIdentity: e.state.binding.identity, RequestID: e.state.checkpointRequestSequence, ArtifactSequence: projection.Image.Sequence, T0: t0, Image: projection.Image}
	result := e.checkpointSubmitter.Submit(request)
	if result.Disposition != checkpoint.SubmitAccepted {
		e.state.checkpointOperations.SubmitRejected++
		return
	}
	if e.state.checkpointOutstanding == nil {
		e.state.checkpointOutstanding = make(map[uint64]checkpoint.Request)
	}
	e.state.checkpointOutstanding[request.RequestID] = request
	e.state.checkpointLastSubmitted = immutableTime(t0)
	e.state.checkpointOperations.Submitted++
	e.state.checkpointOperations.Outstanding++
	if result.Superseded != nil {
		e.applyCheckpointTerminalLocked(*result.Superseded)
	}
}

func (e *Engine) applyCheckpointTerminalLocked(result checkpoint.TerminalResult) (DispositionCode, DispositionReason) {
	request, ok := e.state.checkpointOutstanding[result.RequestID]
	if !ok || e.state.binding == nil || result.BindingIdentity != e.state.binding.identity || result.BindingIdentity != request.BindingIdentity || result.ArtifactSequence != request.ArtifactSequence || result.T0 != request.T0 || !checkpoint.ValidTerminalResult(result) {
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
