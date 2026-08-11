package engine

import "time"

func (e *Engine) applyTimerLocked(node *queueNode) (DispositionCode, DispositionReason) {
	if e.state.binding == nil || node.bindingID != e.state.binding.identity {
		return DispositionIllegalLifecycle, ReasonLifecycle
	}
	target := timerTarget(node.admissionTime, e.delay, e.state.binding.sessionStart, e.state.binding.sessionEnd)
	if !e.transitionLifecycleLocked(lifecycleEventTimer, node, "") {
		return DispositionIllegalLifecycle, ReasonLifecycle
	}
	if e.state.lifecycle == lifecycleEnded && e.state.hydration.generation.active {
		if !e.cancelHydrationGenerationLocked(false) {
			return DispositionAccountingIntegrity, ReasonAccounting
		}
		e.state.hydration.generation.active = false
	}
	e.state.latestTarget = immutableTime(target)
	return DispositionTimerApplied, ReasonNone
}

func timerTarget(now time.Time, delay time.Duration, start, end time.Time) time.Time {
	target := now.Add(-delay).Truncate(time.Second)
	switch {
	case !target.After(start):
		return start
	case !target.Before(end):
		return end
	default:
		return target
	}
}

// candidateTargetSupportedLocked is the read-only central gate evaluated
// before Component 3 applies committed T or any evaluator-current state.
func (e *Engine) candidateTargetSupportedLocked(target time.Time) bool {
	validInstalledBinding := e.state.binding != nil
	monotonicClock := e.state.clockMonotonic
	noUnresolvedGlobalAmbiguity := !e.state.aggregateIntegrity
	coherentCompletedAccounting := e.accountingCoherentLocked()
	if !validInstalledBinding || !monotonicClock || !noUnresolvedGlobalAmbiguity || !coherentCompletedAccounting {
		return false
	}
	if e.state.committedT != nil {
		if target.Before(*e.state.committedT) {
			return false
		}
		if target.Equal(*e.state.committedT) {
			return true
		}
	}

	completeRunSupport := false
	switch e.mode {
	case RunModeLive:
		applicableAcceptedAggregateEpoch := e.state.liveEpochActive && e.state.aggregateAcknowledged && e.state.liveEpoch != 0
		consumedThroughIngressFence := e.state.hydration.fenceReconciled && e.state.hydration.fenceEpoch == e.state.liveEpoch &&
			e.state.hydration.fenceThrough >= e.state.aggregateAckPosition.FrameSequence
		noPriorUnresolvedGlobalTransportGap := !e.state.aggregateIntegrity
		installedContributorPredicates := e.state.hydration.supportedThrough != nil && !e.state.hydration.supportedThrough.Before(target)
		completeRunSupport = applicableAcceptedAggregateEpoch && consumedThroughIngressFence &&
			noPriorUnresolvedGlobalTransportGap && installedContributorPredicates
	case RunModeReplay:
		replay := e.state.replay
		validatedReplayArtifact := replay.validated && replay.complete && !replay.terminal
		provedArtifactCoverageAndOrdering := replay.coveredThrough != nil && !replay.coveredThrough.Before(target) && replay.nextOrdinal > 0
		completedLogicalDeliveryGroup := !replay.lastGroup.IsZero() && !replay.lastGroup.Before(target)
		completeRunSupport = validatedReplayArtifact && provedArtifactCoverageAndOrdering && completedLogicalDeliveryGroup
	default:
		return false
	}
	return completeRunSupport
}

func (e *Engine) accountingCoherentLocked() bool {
	counters := e.counters
	classified := counters.admittedExternal + counters.notAdmittedInvalid + counters.notAdmittedCanceled +
		counters.notAdmittedClosed + counters.pressureShedOptional + counters.sequenceBudgetExhausted
	return e.state.aggregates.reconciles() && e.state.connectionAccounting.reconciles() &&
		counters.started == counters.inProgress+counters.resultsCommitted &&
		counters.resultsCommitted == classified &&
		counters.admittedExternal == uint64(len(e.queue))+counters.ownerInProgress+counters.completedExternal
}

func suppressionDispositionFor(mode RunMode, reason lifecycleReason) SuppressionDisposition {
	if mode == RunModeReplay {
		return SuppressionTerminalReplayFailure
	}
	switch reason {
	case lifecycleReasonIngressIntegrity:
		return SuppressionSameBindingRecoveryAllowed
	case lifecycleReasonRecoveryExhausted:
		return SuppressionSameBindingRecoveryAllowed
	case lifecycleReasonCanonicalIntegrity:
		return SuppressionCleanReinitializationRequired
	case lifecycleReasonClockRegression, lifecycleReasonSequenceExhaustion,
		lifecycleReasonPublicationIntegrity, lifecycleReasonAccountingIntegrity:
		return SuppressionRestartRequired
	default:
		return ""
	}
}

func suppressionRequiresTermination(disposition SuppressionDisposition) bool {
	return disposition == SuppressionCleanReinitializationRequired ||
		disposition == SuppressionRestartRequired ||
		disposition == SuppressionTerminalReplayFailure
}

// enterSuppressionLocked is the sole point that binds a suppression cause to
// its recovery requirement and containment boundary.
func (e *Engine) enterSuppressionLocked(event lifecycleEvent, node *queueNode, reason lifecycleReason) SuppressionDisposition {
	disposition := suppressionDispositionFor(e.mode, reason)
	if disposition == "" {
		event = lifecycleEventAccountingIntegrity
		reason = lifecycleReasonAccountingIntegrity
		disposition = SuppressionRestartRequired
	}
	e.state.suppressionDisposition = disposition
	if suppressionRequiresTermination(disposition) {
		e.state.globalFailure = true
		e.sealed = true
		e.broadcastLocked()
	}
	e.transitionLifecycleLocked(event, node, reason)
	return disposition
}

func (e *Engine) transitionLifecycleLocked(event lifecycleEvent, node *queueNode, fixedReason lifecycleReason) bool {
	previous := e.state.lifecycle
	next := previous
	reason := fixedReason
	admissionTime := time.Time{}
	engineSequence := uint64(0)
	if node != nil {
		admissionTime = node.admissionTime
		engineSequence = node.engineSequence
	}

	switch event {
	case lifecycleEventBinding:
		if previous != lifecycleInitializing || e.state.binding == nil {
			return false
		}
		if e.mode == RunModeReplay {
			return true
		}
		switch {
		case admissionTime.Before(e.state.binding.sessionStart):
			next, reason = lifecycleAwaitingSession, lifecycleReasonBindingBeforeSession
		case admissionTime.Before(e.state.binding.sessionEnd):
			next, reason = lifecycleAwaitingAggregateAck, lifecycleReasonBindingInSession
		default:
			next, reason = lifecycleEnded, lifecycleReasonBindingAfterSession
		}
	case lifecycleEventTimer:
		if e.state.binding == nil {
			return false
		}
		replayEndGroup := e.mode == RunModeReplay && previous == lifecycleReplaying && node != nil &&
			node.kind == inputReplayGroup && admissionTime.Equal(e.state.binding.sessionEnd)
		if !admissionTime.Before(e.state.binding.sessionEnd) && !replayEndGroup {
			next, reason = lifecycleEnded, lifecycleReasonSessionEnd
		} else {
			switch previous {
			case lifecycleAwaitingSession:
				if !admissionTime.Before(e.state.binding.sessionStart) {
					if e.state.liveEpochActive && e.state.aggregateAcknowledged {
						next, reason = lifecycleHydrating, lifecycleReasonAggregateAckAtStart
					} else {
						next, reason = lifecycleAwaitingAggregateAck, lifecycleReasonSessionStart
					}
				}
			case lifecycleAwaitingAggregateAck, lifecycleHydrating, lifecycleLive, lifecycleRecovering, lifecycleReplaying, lifecycleSuppressed:
				// A quiet timer is a legal self-transition and cannot fabricate
				// the later evidence needed to leave any of these states.
			case lifecycleInitializing:
				return false
			default:
				return false
			}
		}
	case lifecycleEventStop:
		next, reason = lifecycleEnded, lifecycleReasonControlledStop
	case lifecycleEventSequenceExhaustion:
		next, reason = lifecycleSuppressed, lifecycleReasonSequenceExhaustion
	case lifecycleEventClockRegression:
		next, reason = lifecycleSuppressed, lifecycleReasonClockRegression
	case lifecycleEventCanonicalIntegrity:
		next, reason = lifecycleSuppressed, lifecycleReasonCanonicalIntegrity
	case lifecycleEventPublicationIntegrity:
		next, reason = lifecycleSuppressed, lifecycleReasonPublicationIntegrity
	case lifecycleEventAccountingIntegrity:
		next, reason = lifecycleSuppressed, lifecycleReasonAccountingIntegrity
	case lifecycleEventClose:
		next, reason = lifecycleEnded, lifecycleReasonClosed
	case lifecycleEventReplayStart:
		if e.mode != RunModeReplay || previous != lifecycleInitializing || !e.state.replay.validated {
			return false
		}
		next, reason = lifecycleReplaying, lifecycleReasonReplayStart
	case lifecycleEventReplayEnd:
		if e.mode != RunModeReplay || previous != lifecycleReplaying || !e.state.replay.terminal {
			return false
		}
		next, reason = lifecycleEnded, lifecycleReasonReplayEnd
	case lifecycleEventReplayRequestedEnd:
		if e.mode != RunModeReplay || previous != lifecycleReplaying || !e.state.replay.terminal {
			return false
		}
		next, reason = lifecycleEnded, lifecycleReasonReplayRequestedEnd
	case lifecycleEventReplayFailure:
		if e.mode != RunModeReplay || previous == lifecycleEnded {
			return false
		}
		next, reason = lifecycleSuppressed, lifecycleReasonReplayFailure
	case lifecycleEventAggregateAck:
		if !e.state.liveEpochActive {
			return false
		}
		switch previous {
		case lifecycleAwaitingSession:
			// Pre-session acknowledgement is retained; the timer at S owns
			// LIFE-T06 and the transition to hydrating.
		case lifecycleAwaitingAggregateAck:
			if admissionTime.Before(e.state.binding.sessionStart) || !admissionTime.Before(e.state.binding.sessionEnd) {
				return false
			}
			next, reason = lifecycleHydrating, lifecycleReasonAggregateAck
		case lifecycleHydrating, lifecycleLive, lifecycleRecovering:
			// Duplicate/control diagnostic in hydrating/live, or LIFE-T18's
			// recovery handoff. Component 6 installs the recovery substep.
		default:
			return false
		}
	case lifecycleEventAggregateLoss:
		switch previous {
		case lifecycleAwaitingSession, lifecycleAwaitingAggregateAck:
		case lifecycleHydrating:
			next, reason = lifecycleAwaitingAggregateAck, lifecycleReasonAggregateEpochLost
		case lifecycleLive:
			next, reason = lifecycleRecovering, lifecycleReasonAggregateEpochLost
		case lifecycleRecovering:
		default:
			return false
		}
	case lifecycleEventIngressIntegrity:
		if previous == lifecycleInitializing || previous == lifecycleReplaying || previous == lifecycleEnded {
			return false
		}
		next = lifecycleSuppressed
		if reason == "" {
			reason = lifecycleReasonIngressIntegrity
		}
	case lifecycleEventHydrationComplete:
		if (previous != lifecycleHydrating && previous != lifecycleRecovering) || e.mode != RunModeLive {
			return false
		}
		next, reason = lifecycleLive, lifecycleReasonHydrationComplete
	default:
		return false
	}

	// A reasonless legal self-transition (most importantly a quiet timer while
	// suppressed) is progress accounting, not new lifecycle evidence. Preserve
	// the fixed cause that established the current state. A later explicit
	// suppression event may still replace the record when its reason or
	// disposition genuinely changes.
	if next == previous && (reason == "" || !(next == lifecycleSuppressed &&
		(e.state.latestTransition == nil || e.state.latestTransition.Reason != reason ||
			e.state.latestTransition.SuppressionDisposition != e.state.suppressionDisposition))) {
		return true
	}
	record := &transitionRecord{
		Previous: previous, Next: next, Reason: reason, AdmissionTime: admissionTime,
		EngineSequence: engineSequence, CommittedT: immutableTimePointer(e.state.committedT),
	}
	if next == lifecycleSuppressed {
		record.SuppressionDisposition = e.state.suppressionDisposition
	}
	if e.state.binding != nil {
		record.BindingIdentity = e.state.binding.identity
	}
	if node != nil {
		switch node.kind {
		case inputAggregate:
			record.Epoch = node.aggregate.Live.ConnectionEpoch
			record.Generation = node.aggregate.Historical.Generation
		case inputConnectionControl:
			record.Epoch = node.connectionControl.ConnectionEpoch
		case inputHydrationPlan:
			record.Epoch = node.hydrationPlan.ConnectionEpoch
		case inputHydrationChunk:
			record.Epoch = node.hydrationChunk.token.connectionEpoch
			record.Generation = node.hydrationChunk.token.generation
		case inputHydrationTerminal:
			record.Epoch = node.hydrationTerminal.token.connectionEpoch
			record.Generation = node.hydrationTerminal.token.generation
		}
	}
	e.state.lifecycle = next
	e.state.latestTransition = record
	if next == lifecycleEnded && !e.sealed {
		e.sealed = true
		e.broadcastLocked()
	}
	return true
}

func immutableTime(value time.Time) *time.Time {
	copyValue := value
	return &copyValue
}

func immutableTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	return immutableTime(*value)
}

type timeLifecycleObservation struct {
	Lifecycle              lifecycle
	ClockMonotonic         bool
	SuppressionDisposition SuppressionDisposition
	LastClock              time.Time
	LatestTarget           *time.Time
	CommittedT             *time.Time
	LatestTransition       *transitionRecord
}

// observeTimeLifecycle returns a narrow immutable package-private proof view.
func (e *Engine) observeTimeLifecycle() timeLifecycleObservation {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := timeLifecycleObservation{
		Lifecycle: e.state.lifecycle, ClockMonotonic: e.state.clockMonotonic,
		SuppressionDisposition: e.state.suppressionDisposition,
		LastClock:              e.lastClock, LatestTarget: immutableTimePointer(e.state.latestTarget),
		CommittedT: immutableTimePointer(e.state.committedT),
	}
	if e.state.latestTransition != nil {
		copyRecord := *e.state.latestTransition
		copyRecord.CommittedT = immutableTimePointer(copyRecord.CommittedT)
		result.LatestTransition = &copyRecord
	}
	return result
}
