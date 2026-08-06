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
		applicableAcceptedAggregateEpoch := e.state.liveEpoch != 0
		consumedThroughIngressFence := false // no S3 fact can prove this
		noPriorUnresolvedGlobalTransportGap := false
		installedContributorPredicates := false
		completeRunSupport = applicableAcceptedAggregateEpoch && consumedThroughIngressFence &&
			noPriorUnresolvedGlobalTransportGap && installedContributorPredicates
	case RunModeReplay:
		validatedReplayArtifact := false // Component 4 owns this evidence.
		provedArtifactCoverageAndOrdering := false
		completedLogicalDeliveryGroup := false
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
	return e.state.aggregates.reconciles() &&
		counters.started == counters.inProgress+counters.resultsCommitted &&
		counters.resultsCommitted == classified &&
		counters.admittedExternal == uint64(len(e.queue))+counters.ownerInProgress+counters.completedExternal
}

func suppressionDispositionFor(mode RunMode, reason lifecycleReason) SuppressionDisposition {
	if mode == RunModeReplay {
		return SuppressionTerminalReplayFailure
	}
	switch reason {
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
		if !admissionTime.Before(e.state.binding.sessionEnd) {
			next, reason = lifecycleEnded, lifecycleReasonSessionEnd
		} else {
			switch previous {
			case lifecycleAwaitingSession:
				if !admissionTime.Before(e.state.binding.sessionStart) {
					next, reason = lifecycleAwaitingAggregateAck, lifecycleReasonSessionStart
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
	default:
		return false
	}

	if next == previous && !(next == lifecycleSuppressed &&
		(e.state.latestTransition == nil || e.state.latestTransition.Reason != reason ||
			e.state.latestTransition.SuppressionDisposition != e.state.suppressionDisposition)) {
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
	if node != nil && node.kind == inputAggregate {
		record.Epoch = node.aggregate.Live.ConnectionEpoch
		record.Generation = node.aggregate.Historical.Generation
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
