package engine

import (
	"context"
	"errors"
	"time"
)

const (
	defaultRecoveryBackoffInitial = time.Second
	defaultRecoveryBackoffMaximum = 30 * time.Second
)

type recoveryPolicy struct {
	initial time.Duration
	maximum time.Duration
}

func newRecoveryPolicy(initial, maximum time.Duration) recoveryPolicy {
	if initial == 0 && maximum == 0 {
		initial, maximum = defaultRecoveryBackoffInitial, defaultRecoveryBackoffMaximum
	}
	return recoveryPolicy{initial: initial, maximum: maximum}
}

func (p recoveryPolicy) valid() bool {
	return p.initial > 0 && p.maximum >= p.initial && p.maximum <= 5*time.Minute
}

func (p recoveryPolicy) delay(ordinal uint64) time.Duration {
	delay := p.initial
	for step := uint64(1); step < ordinal && delay < p.maximum; step++ {
		if delay > p.maximum/2 {
			return p.maximum
		}
		delay *= 2
	}
	if delay > p.maximum {
		return p.maximum
	}
	return delay
}

// ScheduledRecoveryCommand is the engine's opaque, one-shot authority for a
// same-binding continuation. Callers can inspect its finite deadline but
// cannot construct an admissible command or directly select recovering.
type ScheduledRecoveryCommand struct {
	bindingIdentity string
	failedEpoch     uint64
	retryOrdinal    uint64
	issuedAt        time.Time
	earliestAt      time.Time
	done            chan Disposition
}

func (c ScheduledRecoveryCommand) BindingIdentity() string { return c.bindingIdentity }
func (c ScheduledRecoveryCommand) FailedEpoch() uint64     { return c.failedEpoch }
func (c ScheduledRecoveryCommand) RetryOrdinal() uint64    { return c.retryOrdinal }
func (c ScheduledRecoveryCommand) IssuedAt() time.Time     { return c.issuedAt }
func (c ScheduledRecoveryCommand) EarliestAt() time.Time   { return c.earliestAt }

type ScheduledRecoveryInput struct{ command ScheduledRecoveryCommand }
type frozenScheduledRecoveryInput struct{ ScheduledRecoveryInput }

type scheduledRecoveryState struct {
	pending    *ScheduledRecoveryCommand
	dispatched bool
	completed  uint64
	fenced     uint64
}

func validScheduledRecoveryCommand(command ScheduledRecoveryCommand) bool {
	return validIdentityShape(command.bindingIdentity) && command.retryOrdinal > 0 &&
		!command.issuedAt.IsZero() && command.issuedAt == command.issuedAt.UTC() &&
		command.earliestAt.After(command.issuedAt) && command.earliestAt == command.earliestAt.UTC() && command.done != nil
}

func (e *Engine) scheduleRecoveryLocked(node *queueNode) {
	state := &e.state.scheduledRecovery
	if state.pending != nil || e.state.binding == nil {
		return
	}
	issuedAt := e.lastClock
	if node != nil && !node.admissionTime.IsZero() {
		issuedAt = node.admissionTime
	}
	if issuedAt.IsZero() {
		return
	}
	// The next greater epoch is the next consecutive attempt since the last
	// reconciled hydration fence.  A successful fence resets that counter;
	// merely scheduling or admitting this command must not.
	ordinal := e.state.connectionControl.recoveryAttempts + 1
	state.pending = &ScheduledRecoveryCommand{
		bindingIdentity: e.state.binding.identity,
		failedEpoch:     e.state.liveEpoch,
		retryOrdinal:    ordinal,
		issuedAt:        issuedAt,
		earliestAt:      issuedAt.Add(e.recoveryPolicy.delay(ordinal)),
		done:            make(chan Disposition, 1),
	}
	state.dispatched = false
}

// IssueScheduledRecoveryCommand transfers the sole current recovery authority
// once. Merely observing suppression or sleeping does not authorize a socket.
func (e *Engine) IssueScheduledRecoveryCommand() (ScheduledRecoveryCommand, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := &e.state.scheduledRecovery
	recoverableSuppression := e.state.lifecycle == lifecycleSuppressed && e.state.suppressionDisposition == SuppressionSameBindingRecoveryAllowed
	orderedRetry := (e.state.lifecycle == lifecycleAwaitingSession || e.state.lifecycle == lifecycleAwaitingAggregateAck || e.state.lifecycle == lifecycleRecovering) && !e.state.liveEpochActive
	if e.mode != RunModeLive || (!recoverableSuppression && !orderedRetry) ||
		state.pending == nil || state.dispatched || !validScheduledRecoveryCommand(*state.pending) {
		return ScheduledRecoveryCommand{}, errors.New("scheduled recovery command is not issuable")
	}
	state.dispatched = true
	return *state.pending, nil
}

func NewScheduledRecoveryInput(command ScheduledRecoveryCommand) (ScheduledRecoveryInput, error) {
	if !validScheduledRecoveryCommand(command) {
		return ScheduledRecoveryInput{}, errors.New("invalid scheduled recovery command")
	}
	return ScheduledRecoveryInput{command: command}, nil
}

func (e *Engine) AdmitScheduledRecovery(ctx context.Context, input ScheduledRecoveryInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validScheduledRecoveryCommand(input.command) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputScheduledRecovery, scheduledRecovery: frozenScheduledRecoveryInput{input}}, false)
}

func (e *Engine) applyScheduledRecoveryLocked(node *queueNode) (DispositionCode, DispositionReason) {
	command := node.scheduledRecovery.command
	state := &e.state.scheduledRecovery
	recoverableSuppression := e.state.lifecycle == lifecycleSuppressed && e.state.suppressionDisposition == SuppressionSameBindingRecoveryAllowed
	orderedRetry := (e.state.lifecycle == lifecycleAwaitingSession || e.state.lifecycle == lifecycleAwaitingAggregateAck || e.state.lifecycle == lifecycleRecovering) && !e.state.liveEpochActive
	if e.mode != RunModeLive || e.state.binding == nil || (!recoverableSuppression && !orderedRetry) || state.pending == nil || !state.dispatched ||
		command != *state.pending || command.bindingIdentity != e.state.binding.identity || command.failedEpoch != e.state.liveEpoch {
		state.fenced++
		return DispositionConnectionControlFenced, ReasonHistoricalContext
	}
	if node.admissionTime.Before(command.earliestAt) {
		state.fenced++
		return DispositionConnectionControlRejected, ReasonHistoricalContext
	}
	state.pending, state.dispatched = nil, false
	state.completed++
	if recoverableSuppression {
		if !e.transitionLifecycleLocked(lifecycleEventScheduledRecovery, node, lifecycleReasonScheduledRecovery) {
			return DispositionConnectionControlRejected, ReasonLifecycle
		}
		e.state.suppressionDisposition = ""
	}
	return DispositionRecoveryScheduled, ReasonNone
}
