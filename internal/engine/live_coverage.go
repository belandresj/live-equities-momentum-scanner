package engine

import (
	"context"
	"errors"
	"math"
	"time"
)

const LiveCoverageFenceSchemaV1 = "engine-live-coverage-fence-v1"

const (
	DispositionLiveCoverageFenceApplied  DispositionCode   = "live_coverage_fence_applied"
	DispositionLiveCoverageFenceRejected DispositionCode   = "live_coverage_fence_rejected"
	DispositionLiveCoverageFenceFenced   DispositionCode   = "live_coverage_fence_fenced"
	ReasonLiveCoverageFence              DispositionReason = "live_coverage_fence"
)

type LiveCoverageFenceState string

const (
	LiveCoverageFenceComplete LiveCoverageFenceState = "complete"
	LiveCoverageFenceCanceled LiveCoverageFenceState = "canceled"
)

// LiveCoverageFenceCommand is an engine-issued, unforgeable request for the
// current C5 queue to capture a concrete raw-frame boundary.
type LiveCoverageFenceCommand struct {
	bindingID string
	epoch     uint64
	token     uint64
	done      chan LiveCoverageFenceDisposition
}

func (c LiveCoverageFenceCommand) BindingIdentity() string { return c.bindingID }
func (c LiveCoverageFenceCommand) ConnectionEpoch() uint64 { return c.epoch }
func (c LiveCoverageFenceCommand) CommandToken() uint64    { return c.token }

func (c LiveCoverageFenceCommand) Wait(ctx context.Context) (LiveCoverageFenceDisposition, error) {
	if ctx == nil || c.done == nil {
		return LiveCoverageFenceDisposition{}, errors.New("live coverage fence wait requires engine command and context")
	}
	select {
	case <-ctx.Done():
		return LiveCoverageFenceDisposition{}, ctx.Err()
	case result := <-c.done:
		return result, nil
	}
}

type LiveCoverageFenceDisposition struct {
	EngineSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
}

type LiveCoverageFenceInput struct {
	schemaVersion                       string
	command                             LiveCoverageFenceCommand
	state                               LiveCoverageFenceState
	throughFrameSequence, markerOrdinal uint64
	capturedAt                          time.Time
}

type frozenLiveCoverageFenceInput struct{ LiveCoverageFenceInput }

type liveCoverageState struct {
	lastToken   uint64
	outstanding LiveCoverageFenceCommand
}

func NewLiveCoverageFenceInput(command LiveCoverageFenceCommand, state LiveCoverageFenceState, throughFrameSequence, markerOrdinal uint64, capturedAt time.Time) (LiveCoverageFenceInput, error) {
	if !validLiveCoverageFenceCommand(command) || (state != LiveCoverageFenceComplete && state != LiveCoverageFenceCanceled) || markerOrdinal == 0 || capturedAt.IsZero() || capturedAt != capturedAt.UTC() {
		return LiveCoverageFenceInput{}, errors.New("invalid live coverage fence fact")
	}
	return LiveCoverageFenceInput{schemaVersion: LiveCoverageFenceSchemaV1, command: command, state: state, throughFrameSequence: throughFrameSequence, markerOrdinal: markerOrdinal, capturedAt: capturedAt}, nil
}

func validLiveCoverageFenceCommand(command LiveCoverageFenceCommand) bool {
	return validIdentityShape(command.bindingID) && command.epoch > 0 && command.token > 0 && command.done != nil
}

// IssueLiveCoverageFence allocates command authority only. Coverage and T can
// change solely after the returned command comes back through the engine FIFO.
func (e *Engine) IssueLiveCoverageFence() (LiveCoverageFenceCommand, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := &e.state.liveCoverage
	if e.state.binding == nil || e.state.lifecycle != lifecycleLive || !e.state.liveEpochActive || !e.state.aggregateAcknowledged ||
		!e.state.hydration.fenceReconciled || e.state.hydration.supportedThrough == nil || e.state.hydration.generation.active ||
		state.outstanding.token != 0 || state.lastToken == math.MaxUint64 {
		return LiveCoverageFenceCommand{}, errors.New("live coverage fence is not currently issuable")
	}
	state.lastToken++
	command := LiveCoverageFenceCommand{bindingID: e.state.binding.identity, epoch: e.state.liveEpoch, token: state.lastToken, done: make(chan LiveCoverageFenceDisposition, 1)}
	state.outstanding = command
	return command, nil
}

func (e *Engine) AdmitLiveCoverageFence(ctx context.Context, input LiveCoverageFenceInput) (AdmissionResult, <-chan LiveCoverageFenceDisposition) {
	e.beginAdmission()
	if ctx == nil || input.schemaVersion != LiveCoverageFenceSchemaV1 || !validLiveCoverageFenceCommand(input.command) ||
		(input.state != LiveCoverageFenceComplete && input.state != LiveCoverageFenceCanceled) || input.markerOrdinal == 0 || input.capturedAt.IsZero() || input.capturedAt != input.capturedAt.UTC() {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputLiveCoverageFence, liveCoverageFence: frozenLiveCoverageFenceInput{input}}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.liveCoverageCompletion
}

func (e *Engine) AdmitLiveCoverageFenceCancellation(ctx context.Context, command LiveCoverageFenceCommand) (AdmissionResult, <-chan LiveCoverageFenceDisposition) {
	e.beginAdmission()
	if ctx == nil || !validLiveCoverageFenceCommand(command) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	input := LiveCoverageFenceInput{schemaVersion: LiveCoverageFenceSchemaV1, command: command, state: LiveCoverageFenceCanceled, markerOrdinal: 1, capturedAt: e.clock().UTC()}
	node := &queueNode{kind: inputLiveCoverageFence, liveCoverageFence: frozenLiveCoverageFenceInput{input}}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.liveCoverageCompletion
}

func (e *Engine) applyLiveCoverageFenceLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.liveCoverageFence.LiveCoverageFenceInput
	outstanding := e.state.liveCoverage.outstanding
	if input.command != outstanding || e.state.binding == nil || input.command.bindingID != e.state.binding.identity || input.command.epoch != e.state.liveEpoch {
		return DispositionLiveCoverageFenceFenced, ReasonLiveCoverageFence
	}
	node.signalLiveCoverage = true
	e.state.liveCoverage.outstanding = LiveCoverageFenceCommand{}
	if input.state != LiveCoverageFenceComplete {
		return DispositionLiveCoverageFenceRejected, ReasonLiveCoverageFence
	}
	if e.state.lifecycle != lifecycleLive || !e.state.liveEpochActive || !e.state.aggregateAcknowledged || e.state.hydration.generation.active ||
		input.throughFrameSequence < e.state.aggregateAckPosition.FrameSequence ||
		(e.state.greatestIngressPosition.ConnectionEpoch == e.state.liveEpoch && input.throughFrameSequence < e.state.greatestIngressPosition.FrameSequence) ||
		input.markerOrdinal <= e.state.hydration.fenceMarkerOrdinal || input.capturedAt.Before(e.state.aggregateAckReceivedAt) || input.capturedAt.After(node.admissionTime) {
		return DispositionLiveCoverageFenceFenced, ReasonLiveCoverageFence
	}
	target := timerTarget(input.capturedAt, e.delay, e.state.binding.sessionStart, e.state.binding.sessionEnd)
	if e.state.hydration.supportedThrough != nil && target.Before(*e.state.hydration.supportedThrough) {
		return DispositionLiveCoverageFenceRejected, ReasonLiveCoverageFence
	}
	if e.state.hydration.supportedThrough != nil && target.After(*e.state.hydration.supportedThrough) {
		e.extendOrdinaryLiveCoverageLocked(*e.state.hydration.supportedThrough, target)
	}
	e.state.hydration.supportedThrough = immutableTime(target)
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = e.state.liveEpoch
	e.state.hydration.fenceThrough = input.throughFrameSequence
	e.state.hydration.fenceMarkerOrdinal = input.markerOrdinal
	// The accepted fence is complete run support through this exact target.
	// Carry that boundary into the sole evaluator path so a later timer clock
	// sample cannot strand committed T one second behind supported coverage.
	e.state.latestTarget = immutableTime(target)
	e.state.hydration.revision++
	return DispositionLiveCoverageFenceApplied, ReasonNone
}

// extendOrdinaryLiveCoverageLocked installs the per-symbol consequence of one
// accepted wildcard-stream fence into the existing canonical coverage
// bitmaps. A pre-existing unresolved interval keeps its closed origin; only a
// newly discovered inability to establish exact coverage for the fenced
// interval is labeled a post-bootstrap gap.
func (e *Engine) extendOrdinaryLiveCoverageLocked(start, end time.Time) {
	if e.state.aggregateEvaluator.coverage == nil {
		e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence, len(e.state.binding.symbols))
	}
	for index := range e.state.binding.symbols {
		state := ensureAggregateState(&e.state.binding.symbols[index])
		invalid, hasInvalid := e.state.aggregateEvaluator.invalidMarks[index]
		var invalidEvidence *invalidMarkEvidence
		if hasInvalid {
			invalidEvidence = &invalid
		}
		installed := installExactCoverage(state, e.state.binding, start, end, invalidEvidence)
		invalidInInterval := hasInvalid && !invalid.windowStart.Before(start) && invalid.windowStart.Before(end)
		coverageExactThroughT := installed && !invalidInInterval && exactAggregateCoverage(state, e.state.binding, e.state.binding.sessionStart, end)
		prior, exists := e.state.aggregateEvaluator.coverage[index]
		if exists && prior.outcome == coverageOutcomeUnknown && prior.origin != uncertaintyNone {
			continue
		}
		if !coverageExactThroughT {
			if !exists || prior.outcome != coverageOutcomeUnknown || prior.origin == uncertaintyNone {
				e.state.aggregateEvaluator.coverage[index] = coverageUnknownPostBootstrap
			}
			continue
		}
		if _, hasMark := latestMarkBefore(state, end); hasMark {
			delete(e.state.aggregateEvaluator.coverage, index)
		} else {
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
		}
	}
}

func (e *Engine) finishLiveCoverageCommand(node *queueNode, result LiveCoverageFenceDisposition) {
	e.mu.Lock()
	if !node.signalLiveCoverage && e.state.liveCoverage.outstanding == node.liveCoverageFence.command {
		e.state.liveCoverage.outstanding = LiveCoverageFenceCommand{}
		node.signalLiveCoverage = true
	}
	e.mu.Unlock()
	if node.signalLiveCoverage {
		node.liveCoverageFence.command.done <- result
		close(node.liveCoverageFence.command.done)
	}
}
