package engine

import (
	"context"
	"time"
)

const ConnectionControlSchemaV1 = "engine-connection-control-v1"
const RecoveryExhaustionSchemaV1 = "engine-recovery-exhaustion-v1"

// ConnectionControlKind is the closed Component 5 fact family admitted by the
// engine. Transport-specific status parsing remains outside this package.
type ConnectionControlKind string

const (
	ConnectionAttempt            ConnectionControlKind = "connection_attempt"
	ConnectionEstablished        ConnectionControlKind = "connection_established"
	AuthenticationResult         ConnectionControlKind = "authentication_result"
	AggregateCommandWriteResult  ConnectionControlKind = "aggregate_command_write_result"
	AggregateSubscriptionResult  ConnectionControlKind = "aggregate_subscription_result"
	TradeQuoteCommandWriteResult ConnectionControlKind = "tq_command_write_result"
	TradeQuoteSubscriptionResult ConnectionControlKind = "tq_subscription_result"
	ConnectionLost               ConnectionControlKind = "connection_lost"
	IngressIntegrityFailure      ConnectionControlKind = "ingress_integrity_failure"
)

type ConnectionControlOutcome string

const (
	ControlSucceeded ConnectionControlOutcome = "succeeded"
	ControlFailed    ConnectionControlOutcome = "failed"
	ControlAmbiguous ConnectionControlOutcome = "ambiguous"
)

// ConnectionControlInput is a provider-independent immutable fact. CommandToken
// is the positive engine command token, not provider text. Position is required
// only for facts classified from the causal provider stream or its terminal
// marker; local attempt/write completions carry a zero position.
type ConnectionControlInput struct {
	SchemaVersion, BindingIdentity string
	Kind                           ConnectionControlKind
	ConnectionEpoch                uint64
	Position                       LivePosition
	ReceiptTime                    time.Time
	CommandToken                   uint64
	Outcome                        ConnectionControlOutcome
}

type ConnectionControlDisposition struct {
	EngineSequence         uint64
	SystemSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
}

// RecoveryExhaustionInput is Component 8's bounded policy fact. The engine
// accepts it only when its own ordered connection facts prove that exactly the
// stated number of attempts ended without an active epoch or hydration.
type RecoveryExhaustionInput struct {
	SchemaVersion, BindingIdentity string
	Attempts                       uint64
}

type frozenRecoveryExhaustionInput struct{ RecoveryExhaustionInput }

const (
	ReasonControlKind        DispositionReason = "control_kind"
	ReasonControlOutcome     DispositionReason = "control_outcome"
	ReasonControlToken       DispositionReason = "control_token"
	ReasonControlSequence    DispositionReason = "control_sequence"
	ReasonAggregateBeforeAck DispositionReason = "aggregate_at_or_before_ack"
	ReasonIngressIntegrity   DispositionReason = "ingress_integrity"
	ReasonRecoveryExhausted  DispositionReason = "recovery_exhausted"
)

type frozenConnectionControlInput struct{ ConnectionControlInput }

type connectionControlState struct {
	latestKind       ConnectionControlKind
	latestOutcome    ConnectionControlOutcome
	latestReason     DispositionReason
	latestEpoch      uint64
	latestPosition   LivePosition
	greatestPosition LivePosition
	revision         uint64
	recoveryAttempts uint64
}

type connectionControlAccounting struct {
	consumed, applied, deferred, rejected, fenced, integrity uint64
}

func (a connectionControlAccounting) reconciles() bool {
	return a.consumed == a.applied+a.deferred+a.rejected+a.fenced+a.integrity
}

func (e *Engine) AdmitConnectionControl(ctx context.Context, input ConnectionControlInput) (AdmissionResult, <-chan ConnectionControlDisposition) {
	e.beginAdmission()
	if ctx == nil || !boundedConnectionControlInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputConnectionControl, connectionControl: frozenConnectionControlInput{ConnectionControlInput: input}}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.controlCompletion
}

func (e *Engine) AdmitRecoveryExhaustion(ctx context.Context, input RecoveryExhaustionInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || input.SchemaVersion != RecoveryExhaustionSchemaV1 || !validIdentityShape(input.BindingIdentity) || input.Attempts == 0 {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputRecoveryExhaustion, recoveryExhaustion: frozenRecoveryExhaustionInput{input}}, false)
}

func boundedConnectionControlInput(input ConnectionControlInput) bool {
	return len(input.SchemaVersion) <= 64 && len(input.BindingIdentity) <= maximumContextBytes &&
		validConnectionControlKind(input.Kind) && validConnectionControlOutcome(input.Outcome) && len(input.Kind) <= 64 && len(input.Outcome) <= 32
}

func validConnectionControlKind(kind ConnectionControlKind) bool {
	switch kind {
	case ConnectionAttempt, ConnectionEstablished, AuthenticationResult,
		AggregateCommandWriteResult, AggregateSubscriptionResult,
		TradeQuoteCommandWriteResult, TradeQuoteSubscriptionResult,
		ConnectionLost, IngressIntegrityFailure:
		return true
	default:
		return false
	}
}

func validConnectionControlOutcome(outcome ConnectionControlOutcome) bool {
	return outcome == ControlSucceeded || outcome == ControlFailed || outcome == ControlAmbiguous
}

func (e *Engine) applyConnectionControlLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.connectionControl.ConnectionControlInput
	e.state.connectionAccounting.consumed++
	code, reason := e.decideConnectionControlLocked(node, input)
	e.recordConnectionControlLocked(input, code, reason)
	switch code {
	case DispositionConnectionControlApplied:
		e.state.connectionAccounting.applied++
	case DispositionConnectionControlDeferred:
		e.state.connectionAccounting.deferred++
	case DispositionConnectionControlFenced:
		e.state.connectionAccounting.fenced++
	case DispositionIngressIntegrity:
		e.state.connectionAccounting.integrity++
	default:
		e.state.connectionAccounting.rejected++
	}
	return code, reason
}

func (e *Engine) recordConnectionControlLocked(input ConnectionControlInput, code DispositionCode, reason DispositionReason) {
	state := &e.state.connectionControl
	state.latestKind = input.Kind
	state.latestOutcome = input.Outcome
	state.latestReason = reason
	state.latestEpoch = input.ConnectionEpoch
	state.latestPosition = input.Position
	causallyAccepted := code == DispositionConnectionControlApplied || code == DispositionConnectionControlDeferred || code == DispositionIngressIntegrity
	if causallyAccepted && input.Position.ConnectionEpoch == e.state.liveEpoch && input.Position.FrameSequence > 0 &&
		(state.greatestPosition.ConnectionEpoch == 0 || compareLive(input.Position, state.greatestPosition) > 0) {
		state.greatestPosition = input.Position
	}
	if causallyAccepted && input.Position.ConnectionEpoch == e.state.liveEpoch && input.Position.FrameSequence > 0 &&
		(e.state.greatestIngressPosition.ConnectionEpoch == 0 || compareLive(input.Position, e.state.greatestIngressPosition) > 0) {
		e.state.greatestIngressPosition = input.Position
	}
	state.revision++
}

func (e *Engine) decideConnectionControlLocked(node *queueNode, input ConnectionControlInput) (DispositionCode, DispositionReason) {
	if input.SchemaVersion != ConnectionControlSchemaV1 {
		return DispositionUnsupportedSchema, ReasonSchema
	}
	if e.mode != RunModeLive || e.state.binding == nil {
		return DispositionIllegalLifecycle, ReasonLifecycle
	}
	if input.BindingIdentity != e.state.binding.identity {
		return DispositionConnectionControlFenced, ReasonBinding
	}
	if input.ConnectionEpoch == 0 || input.ReceiptTime.IsZero() || input.ReceiptTime != input.ReceiptTime.UTC() || !validControlShape(input) {
		return DispositionConnectionControlRejected, ReasonStructural
	}

	if input.Kind == ConnectionAttempt {
		if input.ConnectionEpoch <= e.state.liveEpoch {
			return DispositionConnectionControlFenced, ReasonStaleLiveEpoch
		}
		if e.state.liveEpochActive || !lifecycleAllowsNewEpoch(e.state.lifecycle) {
			return DispositionConnectionControlRejected, ReasonLifecycle
		}
		e.state.liveEpoch = input.ConnectionEpoch
		e.state.greatestIngressPosition = LivePosition{ConnectionEpoch: input.ConnectionEpoch}
		e.state.liveEpochActive = true
		e.state.connectionControl.greatestPosition = LivePosition{}
		e.clearAggregateAcknowledgementLocked()
		e.state.connectionControl.recoveryAttempts++
		return DispositionConnectionControlApplied, ReasonNone
	}

	if input.ConnectionEpoch != e.state.liveEpoch || !e.state.liveEpochActive {
		return DispositionConnectionControlFenced, ReasonStaleLiveEpoch
	}
	if controlHasCausalPosition(input.Kind) && e.state.connectionControl.greatestPosition.ConnectionEpoch == input.ConnectionEpoch &&
		compareLive(input.Position, e.state.connectionControl.greatestPosition) <= 0 {
		return DispositionConnectionControlFenced, ReasonNonprecedent
	}

	switch input.Kind {
	case ConnectionEstablished, AuthenticationResult:
		return DispositionConnectionControlApplied, controlOutcomeReason(input.Outcome)
	case AggregateCommandWriteResult:
		if input.Outcome == ControlSucceeded {
			e.state.aggregateWriteToken = input.CommandToken
		} else if e.state.aggregateWriteToken == input.CommandToken {
			e.state.aggregateWriteToken = 0
		}
		return DispositionConnectionControlApplied, controlOutcomeReason(input.Outcome)
	case AggregateSubscriptionResult:
		if e.state.aggregateAcknowledged {
			// Once the current epoch's handoff is established, later A.*
			// statuses are diagnostics only. They may advance observed control
			// causality but cannot move the immutable handoff boundary.
			return DispositionConnectionControlApplied, controlOutcomeReason(input.Outcome)
		}
		if input.Outcome != ControlSucceeded {
			return DispositionConnectionControlApplied, controlOutcomeReason(input.Outcome)
		}
		if e.state.aggregateWriteToken == 0 || input.CommandToken != e.state.aggregateWriteToken {
			return DispositionConnectionControlRejected, ReasonControlSequence
		}
		if e.state.aggregateAcknowledged && compareLive(input.Position, e.state.aggregateAckPosition) <= 0 {
			return DispositionConnectionControlRejected, ReasonNonprecedent
		}
		if !e.transitionLifecycleLocked(lifecycleEventAggregateAck, node, lifecycleReasonAggregateAck) {
			return DispositionConnectionControlRejected, ReasonLifecycle
		}
		e.state.aggregateAcknowledged = true
		e.state.aggregateAckPosition = input.Position
		e.state.aggregateAckReceivedAt = input.ReceiptTime
		e.clearTQControlQuarantineLocked(input.ConnectionEpoch)
		return DispositionConnectionControlApplied, ReasonNone
	case TradeQuoteCommandWriteResult, TradeQuoteSubscriptionResult:
		return DispositionConnectionControlDeferred, controlOutcomeReason(input.Outcome)
	case ConnectionLost:
		if !lifecycleAllowsAggregateLoss(e.state.lifecycle) {
			return DispositionConnectionControlRejected, ReasonLifecycle
		}
		wasLive := e.state.lifecycle == lifecycleLive
		if wasLive {
			if e.state.committedT == nil {
				return DispositionIngressIntegrity, ReasonIngressIntegrity
			}
			e.state.hydration.supportedT = immutableTime(*e.state.committedT)
			stale := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
			stale.mode, stale.reason, stale.rows, stale.tqIntentAvailable = rankingStale, "", nil, false
			e.state.aggregateEvaluator.current = stale
			e.state.evaluationRevision++
			e.state.exposedRevision++
		}
		if (e.state.lifecycle == lifecycleHydrating || e.state.lifecycle == lifecycleRecovering) && e.state.hydration.generation.active {
			if !e.cancelHydrationGenerationLocked(false) {
				return DispositionAccountingIntegrity, ReasonAccounting
			}
			e.state.hydration.generation.active = false
		}
		e.state.hydration.fenceReconciled = false
		e.state.liveEpochActive = false
		e.clearAggregateAcknowledgementLocked()
		if !e.transitionLifecycleLocked(lifecycleEventAggregateLoss, node, lifecycleReasonAggregateEpochLost) {
			return DispositionConnectionControlRejected, ReasonLifecycle
		}
		e.scheduleRecoveryLocked(node)
		return DispositionConnectionControlApplied, ReasonNone
	case IngressIntegrityFailure:
		if e.state.hydration.generation.active {
			if !e.cancelHydrationGenerationLocked(true) {
				return DispositionAccountingIntegrity, ReasonAccounting
			}
			e.state.hydration.generation.active = false
		}
		if e.routeRecoverableAggregateLossLocked(node) {
			return DispositionIngressIntegrity, ReasonIngressIntegrity
		}
		e.state.hydration.fenceReconciled = false
		e.state.liveEpochActive = false
		e.clearAggregateAcknowledgementLocked()
		disposition := e.enterSuppressionLocked(lifecycleEventIngressIntegrity, node, lifecycleReasonIngressIntegrity)
		if disposition != SuppressionSameBindingRecoveryAllowed {
			return DispositionAccountingIntegrity, ReasonAccounting
		}
		return DispositionIngressIntegrity, ReasonIngressIntegrity
	default:
		return DispositionConnectionControlRejected, ReasonControlKind
	}
}

func (e *Engine) applyRecoveryExhaustionLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.recoveryExhaustion.RecoveryExhaustionInput
	control := &e.state.connectionControl
	validLifecycle := e.state.lifecycle == lifecycleAwaitingSession || e.state.lifecycle == lifecycleAwaitingAggregateAck || e.state.lifecycle == lifecycleRecovering ||
		(e.state.lifecycle == lifecycleSuppressed && e.state.suppressionDisposition == SuppressionSameBindingRecoveryAllowed)
	suppressedIngress := e.state.lifecycle == lifecycleSuppressed && e.state.latestTransition != nil && e.state.latestTransition.Reason == lifecycleReasonIngressIntegrity
	terminalEvidence := ((control.latestKind == ConnectionLost || control.latestKind == IngressIntegrityFailure) &&
		(control.latestOutcome == ControlFailed || control.latestOutcome == ControlAmbiguous)) ||
		suppressedIngress
	if e.mode != RunModeLive || e.state.binding == nil || input.BindingIdentity != e.state.binding.identity ||
		!validLifecycle || e.state.liveEpochActive || e.state.hydration.generation.active ||
		!terminalEvidence ||
		control.recoveryAttempts == 0 || input.Attempts != control.recoveryAttempts {
		return DispositionConnectionControlRejected, ReasonHistoricalContext
	}
	e.state.scheduledRecovery.pending = nil
	e.state.scheduledRecovery.dispatched = false
	e.enterSuppressionLocked(lifecycleEventIngressIntegrity, node, lifecycleReasonRecoveryExhausted)
	return DispositionRecoveryExhausted, ReasonRecoveryExhausted
}

// routeRecoverableAggregateLossLocked converts possible raw aggregate loss
// into the ordinary exact-gap recovery path only when the committed boundary
// is sufficient to name the unsupported suffix. It never fabricates a safe
// boundary during bootstrap or after contradictory canonical evidence.
func (e *Engine) routeRecoverableAggregateLossLocked(node *queueNode) bool {
	if e.mode != RunModeLive || e.state.lifecycle != lifecycleLive || e.state.committedT == nil || e.state.aggregateIntegrity {
		return false
	}
	e.state.hydration.supportedT = immutableTime(*e.state.committedT)
	stale := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	stale.mode, stale.reason, stale.rows, stale.tqIntentAvailable = rankingStale, "", nil, false
	e.state.aggregateEvaluator.current = stale
	e.state.evaluationRevision++
	e.state.exposedRevision++
	e.state.hydration.fenceReconciled = false
	e.state.liveEpochActive = false
	e.clearAggregateAcknowledgementLocked()
	e.state.suppressionDisposition = ""
	if !e.transitionLifecycleLocked(lifecycleEventAggregateLoss, node, lifecycleReasonAggregateEpochLost) {
		return false
	}
	e.scheduleRecoveryLocked(node)
	return true
}

func (e *Engine) clearAggregateAcknowledgementLocked() {
	e.state.aggregateWriteToken = 0
	e.state.aggregateAcknowledged = false
	e.state.aggregateAckPosition = LivePosition{}
	e.state.aggregateAckReceivedAt = time.Time{}
}

func lifecycleAllowsNewEpoch(state lifecycle) bool {
	return state == lifecycleAwaitingSession || state == lifecycleAwaitingAggregateAck || state == lifecycleRecovering
}

func lifecycleAllowsAggregateLoss(state lifecycle) bool {
	return state == lifecycleAwaitingSession || state == lifecycleAwaitingAggregateAck || state == lifecycleHydrating ||
		state == lifecycleLive || state == lifecycleRecovering
}

func controlOutcomeReason(outcome ConnectionControlOutcome) DispositionReason {
	if outcome == ControlSucceeded {
		return ReasonNone
	}
	return ReasonControlOutcome
}

func validControlShape(input ConnectionControlInput) bool {
	hasPosition := input.Position.ConnectionEpoch != 0 || input.Position.FrameSequence != 0 || input.Position.ArrayIndex != 0
	validPosition := input.Position.ConnectionEpoch == input.ConnectionEpoch && input.Position.FrameSequence > 0
	switch input.Kind {
	case ConnectionAttempt:
		return input.CommandToken > 0 && input.Outcome == ControlSucceeded && !hasPosition
	case AggregateCommandWriteResult, TradeQuoteCommandWriteResult:
		return input.CommandToken > 0 && !hasPosition
	case ConnectionEstablished, AuthenticationResult, AggregateSubscriptionResult, TradeQuoteSubscriptionResult:
		return input.CommandToken > 0 && validPosition && (input.Kind != ConnectionEstablished || input.Outcome == ControlSucceeded)
	case ConnectionLost:
		return input.CommandToken == 0 && input.Outcome == ControlFailed && validPosition
	case IngressIntegrityFailure:
		return input.CommandToken == 0 && input.Outcome == ControlAmbiguous && validPosition
	default:
		return false
	}
}

func controlHasCausalPosition(kind ConnectionControlKind) bool {
	switch kind {
	case ConnectionEstablished, AuthenticationResult, AggregateSubscriptionResult, TradeQuoteSubscriptionResult, ConnectionLost, IngressIntegrityFailure:
		return true
	default:
		return false
	}
}
