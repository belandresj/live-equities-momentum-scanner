package engine

import (
	"errors"
	"math"
	"time"
)

const privatePublicationSchemaV1 = "engine-private-publication-v1"

type publicationKind uint8

const (
	publicationInitial publicationKind = iota + 1
	publicationNormal
	publicationUnavailableSentinel
)

type publicationDecision uint8

const (
	decisionNoExposedChange publicationDecision = iota + 1
	decisionReplaced
	decisionIntegrityFailure
)

// publicationFault is the narrow fixed package-private S4 proof seam. It is
// neither configurable by production callers nor a callback/registration
// mechanism, and it can model only the two contracted publication failures.
type publicationFault uint8

const (
	publicationFaultNone publicationFault = iota
	publicationFaultBuild
	publicationFaultValidation
)

type publicationFingerprint struct {
	bindingIdentity      string
	lifecycle            lifecycle
	exposedRevision      uint64
	aggregateIntegrity   bool
	clockMonotonic       bool
	globalFailure        bool
	evaluationRevision   uint64
	controlRevision      uint64
	hydrationRevision    uint64
	tqProjectionRevision uint64
}

type privatePublication struct {
	kind                                                                    publicationKind
	schemaVersion                                                           string
	publicationID                                                           uint64
	bindingIdentity                                                         string
	tradingDate                                                             string
	mode                                                                    RunMode
	lifecycle                                                               lifecycle
	lifecycleReason                                                         lifecycleReason
	suppressionDisposition                                                  SuppressionDisposition
	lastDisposition                                                         DispositionCode
	dispositionReason                                                       DispositionReason
	lastEngineSequence                                                      uint64
	watermark                                                               *time.Time
	generatedAt                                                             time.Time
	queueCapacity                                                           int
	requiredReserve                                                         int
	queueOccupancy                                                          int
	admission                                                               admissionCounters
	transitions                                                             transitionCounters
	publications                                                            publicationCounters
	aggregates                                                              aggregateAccounting
	connectionControls                                                      connectionControlAccounting
	connectionRecoveryAttempts                                              uint64
	connectionEpoch                                                         uint64
	connectionActive                                                        bool
	aggregateAcknowledged                                                   bool
	aggregateAckPosition                                                    LivePosition
	latestControlKind                                                       ConnectionControlKind
	latestControlOutcome                                                    ConnectionControlOutcome
	latestControlReason                                                     DispositionReason
	aggregateIntegrity                                                      bool
	clockMonotonic                                                          bool
	currentMarketClaim                                                      bool
	aggregateEvaluation                                                     aggregateEvaluationResult
	hydrationPurpose                                                        HydrationPurpose
	hydrationActive                                                         bool
	hydrationGeneration                                                     uint64
	hydrationStart                                                          time.Time
	hydrationEnd                                                            time.Time
	hydrationAccounting                                                     HydrationAccounting
	hydrationRows                                                           HydrationRowAccounting
	hydrationFenceReconciled                                                bool
	hydrationFenceEpoch, hydrationFenceThrough, hydrationFenceMarkerOrdinal uint64
	hydrationSupportedThrough                                               *time.Time
	hydrationPolicyAction                                                   HydrationPolicyAction
	hydrationPolicyToken                                                    uint64
	hydrationPolicyWaiting                                                  bool
	installedCheckpoint                                                     bool
	evaluatorIntegrity                                                      *EvaluatorIntegrityView
	tq                                                                      TQView
}

// publicationView is a defensive package-private read projection. Component
// 10 owns any eventual exported or serialized schema.
type publicationView privatePublication

func (e *Engine) installInitialPublication() {
	initial := &privatePublication{
		kind: publicationInitial, schemaVersion: privatePublicationSchemaV1,
		mode: e.mode, lifecycle: lifecycleInitializing, clockMonotonic: true,
	}
	for _, reason := range []lifecycleReason{
		lifecycleReasonSequenceExhaustion,
		lifecycleReasonClockRegression,
		lifecycleReasonCanonicalIntegrity,
		lifecycleReasonPublicationIntegrity,
		lifecycleReasonAccountingIntegrity,
		lifecycleReasonReplayFailure,
		lifecycleReasonIngressIntegrity,
		lifecycleReasonRecoveryExhausted,
	} {
		index, _ := sentinelIndex(reason)
		e.sentinels[index] = newUnavailableSentinel(e.mode, reason)
	}
	e.storePublication(initial)
}

func sentinelIndex(reason lifecycleReason) (int, bool) {
	switch reason {
	case lifecycleReasonSequenceExhaustion:
		return 0, true
	case lifecycleReasonClockRegression:
		return 1, true
	case lifecycleReasonCanonicalIntegrity:
		return 2, true
	case lifecycleReasonPublicationIntegrity:
		return 3, true
	case lifecycleReasonAccountingIntegrity:
		return 4, true
	case lifecycleReasonReplayFailure:
		return 5, true
	case lifecycleReasonIngressIntegrity:
		return 6, true
	case lifecycleReasonRecoveryExhausted:
		return 7, true
	default:
		return 4, false
	}
}

func newUnavailableSentinel(mode RunMode, reason lifecycleReason) *privatePublication {
	result := &privatePublication{
		kind: publicationUnavailableSentinel, schemaVersion: privatePublicationSchemaV1,
		mode: mode, lifecycle: lifecycleSuppressed, lifecycleReason: reason,
		suppressionDisposition: suppressionDispositionFor(mode, reason), clockMonotonic: true,
		aggregateEvaluation: aggregateEvaluationResult{mode: rankingSuppressed, reason: rankingReasonGlobalSuppression},
	}
	switch reason {
	case lifecycleReasonSequenceExhaustion:
		result.lastDisposition = DispositionSequenceExhausted
	case lifecycleReasonClockRegression:
		result.lastDisposition, result.dispositionReason = DispositionClockRegression, ReasonClockRegression
		result.clockMonotonic = false
	case lifecycleReasonCanonicalIntegrity:
		result.lastDisposition, result.dispositionReason = DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal
		result.aggregateIntegrity = true
	case lifecycleReasonPublicationIntegrity:
		result.lastDisposition, result.dispositionReason = DispositionPublicationIntegrity, ReasonPublication
	case lifecycleReasonAccountingIntegrity:
		result.lastDisposition, result.dispositionReason = DispositionAccountingIntegrity, ReasonAccounting
	case lifecycleReasonReplayFailure:
		result.lastDisposition, result.dispositionReason = DispositionReplayFailed, ReasonReplayEvidence
	case lifecycleReasonIngressIntegrity:
		result.lastDisposition, result.dispositionReason = DispositionIngressIntegrity, ReasonIngressIntegrity
	case lifecycleReasonRecoveryExhausted:
		result.lastDisposition, result.dispositionReason = DispositionIngressIntegrity, ReasonAggregateIngressFence
	}
	return result
}

func (e *Engine) unavailableSentinelLocked() *privatePublication {
	if e.state.latestTransition == nil || e.state.latestTransition.Next != lifecycleSuppressed {
		return e.sentinels[4]
	}
	index, _ := sentinelIndex(e.state.latestTransition.Reason)
	return e.sentinels[index]
}

// storePublication is the sole atomic-cell replacement path for initial,
// normal, and unavailable publications.
func (e *Engine) storePublication(value *privatePublication) { e.publication.Store(value) }

func (e *Engine) observePublication() publicationView {
	publication := e.publication.Load()
	copyValue := publicationView(*publication)
	copyValue.watermark = immutableTimePointer(publication.watermark)
	copyValue.aggregateEvaluation = cloneAggregateEvaluation(publication.aggregateEvaluation)
	copyValue.evaluatorIntegrity = cloneEvaluatorIntegrity(publication.evaluatorIntegrity)
	copyValue.tq = cloneTQView(publication.tq)
	return copyValue
}

func (e *Engine) publicationFingerprintLocked() publicationFingerprint {
	result := publicationFingerprint{
		lifecycle: e.state.lifecycle, exposedRevision: e.state.exposedRevision,
		aggregateIntegrity: e.state.aggregateIntegrity,
		clockMonotonic:     e.state.clockMonotonic, globalFailure: e.state.globalFailure,
		evaluationRevision:   e.state.evaluationRevision,
		controlRevision:      e.state.connectionControl.revision,
		hydrationRevision:    e.state.hydration.revision,
		tqProjectionRevision: e.state.tq.publicProjectionRevision,
	}
	if e.state.binding != nil {
		result.bindingIdentity = e.state.binding.identity
	}
	return result
}

func (e *Engine) finishTransitionLocked(node *queueNode, disposition transitionDisposition, before publicationFingerprint, forceUnavailable bool) transitionDisposition {
	changed := before != e.publicationFingerprintLocked()
	if disposition.Code == DispositionBindingInvalid && e.publication.Load().kind == publicationInitial {
		changed = true
	}
	disposition = e.completePublicationDecisionLocked(node, disposition, changed, forceUnavailable, false)
	e.broadcastLocked()
	e.mu.Unlock()
	return disposition
}

func (e *Engine) finishInternalTransitionLocked(sequence uint64, before publicationFingerprint, forceUnavailable bool) {
	changed := before != e.publicationFingerprintLocked()
	node := &queueNode{engineSequence: sequence, admissionTime: e.lastClock}
	code := DispositionControlApplied
	if forceUnavailable {
		code = DispositionSequenceExhausted
	}
	e.completePublicationDecisionLocked(node,
		transitionDisposition{EngineSequence: sequence, Code: code},
		changed, forceUnavailable, true)
}

// completePublicationDecisionLocked is the one external/internal publication
// decision path. The caller holds the sole owner lock; this function returns
// only after counters and the cell/sentinel decision are final.
func (e *Engine) completePublicationDecisionLocked(node *queueNode, disposition transitionDisposition, changed, forceUnavailable, internal bool) transitionDisposition {
	prospectiveAdmission := e.counters
	prospectiveTransitions := e.transitions
	if internal {
		prospectiveTransitions.completedInternal++
	} else {
		prospectiveAdmission.ownerInProgress--
		prospectiveAdmission.completedExternal++
		classifyCompletedTransition(&prospectiveTransitions, disposition.Code)
	}
	prospectivePublications := e.publications

	decision := decisionNoExposedChange
	if forceUnavailable {
		decision = decisionIntegrityFailure
	} else if changed {
		decision = decisionReplaced
	}

	var candidate *privatePublication
	if decision == decisionReplaced {
		generatedAt := e.clock().UTC()
		if e.hasClock && generatedAt.Before(e.lastClock) {
			e.state.clockMonotonic = false
			disposition.SuppressionDisposition = e.enterSuppressionLocked(lifecycleEventClockRegression, node, lifecycleReasonClockRegression)
			disposition.Code, disposition.Reason = DispositionClockRegression, ReasonClockRegression
			if !internal {
				prospectiveTransitions = e.transitions
				classifyCompletedTransition(&prospectiveTransitions, disposition.Code)
			}
			decision = decisionIntegrityFailure
		} else {
			e.lastClock = generatedAt
			e.hasClock = true
			if e.lastPubID == math.MaxUint64 {
				disposition.Code, disposition.Reason = DispositionPublicationIntegrity, ReasonPublication
				disposition.SuppressionDisposition = e.enterSuppressionLocked(lifecycleEventPublicationIntegrity, node, lifecycleReasonPublicationIntegrity)
				if !internal {
					prospectiveTransitions = e.transitions
					classifyCompletedTransition(&prospectiveTransitions, disposition.Code)
				}
				decision = decisionIntegrityFailure
			} else {
				nextID := e.lastPubID + 1
				prospectivePublications = advancePublicationCounters(prospectivePublications, decisionReplaced)
				var err error
				if e.state.evaluationTiming.EngineSequence == node.engineSequence {
					e.advanceActiveAggregateEvaluationLocked(node.engineSequence, AggregateEvaluationPhasePublication)
				}
				publicationStarted := e.evaluationTimingStart()
				candidate, err = e.buildPublicationLocked(nextID, node.engineSequence, disposition, generatedAt,
					prospectiveAdmission, prospectiveTransitions, prospectivePublications)
				if e.state.evaluationTiming.EngineSequence == node.engineSequence {
					e.state.evaluationTiming.Publication = e.evaluationTimingElapsed(publicationStarted)
				}
				if node.kind == inputAggregateIngressFence {
					e.state.fenceTiming.Publication = e.state.evaluationTiming.Publication
				}
				if err != nil || e.publicationFault == publicationFaultBuild ||
					e.publicationFault == publicationFaultValidation || validatePublication(candidate) != nil {
					disposition.Code, disposition.Reason = DispositionPublicationIntegrity, ReasonPublication
					disposition.SuppressionDisposition = e.enterSuppressionLocked(lifecycleEventPublicationIntegrity, node, lifecycleReasonPublicationIntegrity)
					if !internal {
						prospectiveTransitions = e.transitions
						classifyCompletedTransition(&prospectiveTransitions, disposition.Code)
					}
					decision = decisionIntegrityFailure
					candidate = nil
				}
			}
		}
	}

	if decision == decisionIntegrityFailure {
		prospectivePublications = advancePublicationCounters(e.publications, decisionIntegrityFailure)
	} else if decision == decisionNoExposedChange {
		prospectivePublications = advancePublicationCounters(e.publications, decisionNoExposedChange)
	}

	if !prospectiveAccountingCoherent(e, prospectiveAdmission, prospectiveTransitions, prospectivePublications) {
		disposition.Code, disposition.Reason = DispositionAccountingIntegrity, ReasonAccounting
		if !internal {
			prospectiveTransitions = e.transitions
			classifyCompletedTransition(&prospectiveTransitions, disposition.Code)
		}
		prospectivePublications = advancePublicationCounters(e.publications, decisionIntegrityFailure)
		disposition.SuppressionDisposition = e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
		decision = decisionIntegrityFailure
		candidate = nil
	}

	e.counters = prospectiveAdmission
	e.transitions = prospectiveTransitions
	e.publications = prospectivePublications
	switch decision {
	case decisionReplaced:
		e.lastPubID = candidate.publicationID
		e.storePublication(candidate)
		e.flushTQProjectionLocked()
		if candidate.lifecycle != lifecycleSuppressed && candidate.kind == publicationNormal {
			e.state.lastCoherentPublication = candidate
		}
	case decisionIntegrityFailure:
		if terminal := e.buildSuppressedPublicationLocked(node, disposition, prospectiveAdmission, prospectiveTransitions, prospectivePublications); terminal != nil {
			e.lastPubID = terminal.publicationID
			e.storePublication(terminal)
		} else {
			e.storePublication(e.unavailableSentinelLocked())
		}
	}
	if node.kind == inputAggregateIngressFence {
		e.finishFenceTimingLocked()
	}
	e.publicationFault = publicationFaultNone
	return disposition
}

func (e *Engine) finishFenceTimingLocked() {
	timing := &e.state.fenceTiming
	timing.PublicationID = e.lastPubID
	timing.Total = e.evaluationTimingElapsed(e.state.fenceTimingStarted)
	e.state.fenceTimingStarted = time.Time{}
	parts := []time.Duration{timing.CoverageFinalization, timing.SymbolMaintenance, timing.EvaluationStage, timing.EvaluationApply, timing.Publication}
	var attributed time.Duration
	for _, part := range parts {
		if part < 0 || attributed > time.Duration(math.MaxInt64)-part {
			timing.Valid = false
			timing.InvalidReason = "nonmonotonic_or_overflow"
			return
		}
		attributed += part
	}
	if timing.Total < 0 || attributed > timing.Total {
		timing.Valid = false
		timing.InvalidReason = "nonmonotonic_or_overlap"
		return
	}
	timing.ResidualOrderedOverhead = timing.Total - attributed
}

func classifyCompletedTransition(counters *transitionCounters, code DispositionCode) {
	counters.completedExternal++
	switch code {
	case DispositionAggregateInserted, DispositionAggregateRevised, DispositionAggregateWithdrawn:
		counters.appliedMarket++
	case DispositionBindingInstalled, DispositionControlApplied, DispositionTimerApplied, DispositionReplayStarted, DispositionReplayEnded, DispositionReplayRequestedEnd,
		DispositionConnectionControlApplied, DispositionConnectionControlDeferred, DispositionHydrationPlanApplied, DispositionHydrationChunkApplied,
		DispositionAggregateIngressFenceApplied, DispositionCheckpointProjected, DispositionCheckpointInstalled, DispositionCheckpointTerminalApplied:
		counters.appliedNonmarket++
	case DispositionLiveCoverageFenceApplied:
		counters.appliedNonmarket++
	case DispositionHydrationPolicyApplied:
		counters.appliedNonmarket++
	case DispositionRecoveryScheduled:
		counters.appliedNonmarket++
	case DispositionAggregateExactDuplicate:
		counters.exactDuplicate++
	case DispositionAggregateFenced, DispositionConnectionControlFenced, DispositionHydrationFenced, DispositionAggregateIngressFenceFenced, DispositionCheckpointTerminalFenced, DispositionLiveCoverageFenceFenced:
		counters.fenced++
	case DispositionHydrationTerminalApplied:
		counters.terminalWorkFact++
	case DispositionClockRegression, DispositionAggregateIntegrity, DispositionPublicationIntegrity, DispositionAccountingIntegrity, DispositionReplayFailed, DispositionIngressIntegrity, DispositionHydrationIntegrity, DispositionRecoveryExhausted:
		counters.integrityFailure++
	default:
		counters.rejected++
	}
}

func advancePublicationCounters(counters publicationCounters, decision publicationDecision) publicationCounters {
	counters.completedDecisions++
	switch decision {
	case decisionNoExposedChange:
		counters.noExposedChange++
	case decisionReplaced:
		counters.publicationReplaced++
	case decisionIntegrityFailure:
		counters.publicationIntegrity++
	}
	return counters
}

func prospectiveAccountingCoherent(e *Engine, admission admissionCounters, transitions transitionCounters, publications publicationCounters) bool {
	classified := admission.admittedExternal + admission.notAdmittedInvalid + admission.notAdmittedCanceled +
		admission.notAdmittedClosed + admission.pressureShedOptional + admission.sequenceBudgetExhausted
	return e.state.aggregates.reconciles() && e.state.connectionAccounting.reconciles() && transitions.reconciles() &&
		admission.started == admission.inProgress+admission.resultsCommitted &&
		admission.resultsCommitted == classified &&
		admission.admittedExternal == uint64(e.externalQueueOccupancyLocked())+admission.ownerInProgress+admission.completedExternal &&
		publications.reconciles(transitions.completedExternal+transitions.completedInternal)
}

func (e *Engine) buildPublicationLocked(id, sequence uint64, disposition transitionDisposition, generatedAt time.Time,
	admission admissionCounters, transitions transitionCounters, publications publicationCounters) (*privatePublication, error) {
	if e.publicationFault == publicationFaultBuild {
		return nil, errors.New("injected publication construction failure")
	}
	candidate := &privatePublication{
		kind: publicationNormal, schemaVersion: privatePublicationSchemaV1, publicationID: id,
		mode: e.mode, lifecycle: e.state.lifecycle, lastDisposition: disposition.Code,
		dispositionReason: disposition.Reason, lastEngineSequence: sequence,
		watermark: immutableTimePointer(e.state.committedT), generatedAt: generatedAt,
		queueCapacity: e.capacity, requiredReserve: e.reserve, queueOccupancy: e.externalQueueOccupancyLocked(),
		admission: admission, transitions: transitions, publications: publications,
		aggregates: e.state.aggregates, aggregateIntegrity: e.state.aggregateIntegrity,
		connectionControls: e.state.connectionAccounting, connectionEpoch: e.state.liveEpoch,
		connectionRecoveryAttempts: e.state.connectionControl.recoveryAttempts,
		connectionActive:           e.state.liveEpochActive, aggregateAcknowledged: e.state.aggregateAcknowledged,
		aggregateAckPosition: e.state.aggregateAckPosition,
		latestControlKind:    e.state.connectionControl.latestKind, latestControlOutcome: e.state.connectionControl.latestOutcome,
		latestControlReason: e.state.connectionControl.latestReason,
		clockMonotonic:      e.state.clockMonotonic,
		aggregateEvaluation: cloneAggregateEvaluation(e.state.aggregateEvaluator.current),
		hydrationPurpose:    e.state.hydration.generation.purpose,
		hydrationActive:     e.state.hydration.generation.active,
		hydrationGeneration: e.state.hydration.generation.generation,
		hydrationStart:      e.state.hydration.generation.start, hydrationEnd: e.state.hydration.generation.end,
		hydrationAccounting:      e.state.hydration.generation.accounting,
		hydrationRows:            e.state.hydration.generation.rowAccounting,
		hydrationFenceReconciled: e.state.hydration.fenceReconciled,
		hydrationFenceEpoch:      e.state.hydration.fenceEpoch, hydrationFenceThrough: e.state.hydration.fenceThrough,
		hydrationFenceMarkerOrdinal: e.state.hydration.fenceMarkerOrdinal,
		hydrationSupportedThrough:   immutableTimePointer(e.state.hydration.supportedThrough),
		hydrationPolicyAction:       e.state.hydration.policyAction,
		hydrationPolicyToken:        e.state.hydration.lastPolicyToken,
		hydrationPolicyWaiting:      e.state.hydration.policyWaiting,
		installedCheckpoint:         e.state.installedCheckpoint != nil,
		evaluatorIntegrity:          cloneEvaluatorIntegrity(e.state.evaluatorIntegrity),
		tq:                          cloneTQView(e.tqViewLocked()),
	}
	candidate.currentMarketClaim = candidate.aggregateEvaluation.mode == rankingQualifiedCurrent ||
		candidate.aggregateEvaluation.mode == rankingDegradedBootstrap ||
		candidate.aggregateEvaluation.mode == rankingDegradedCurrent
	candidate.tq.PublicationID = id
	if e.state.binding != nil {
		candidate.bindingIdentity = e.state.binding.identity
		candidate.tradingDate = e.state.binding.tradingDate
	}
	if e.state.latestTransition != nil {
		candidate.lifecycleReason = e.state.latestTransition.Reason
	}
	candidate.suppressionDisposition = e.state.suppressionDisposition
	if e.publicationFault == publicationFaultValidation {
		candidate.publicationID = 0
	}
	return candidate, nil
}

// buildSuppressedPublicationLocked preserves an installed binding's terminal
// integrity state as one coherent sealed capture. If even this construction
// cannot validate, the no-claim sentinel remains the fail-closed fallback.
func (e *Engine) buildSuppressedPublicationLocked(node *queueNode, disposition transitionDisposition, admission admissionCounters, transitions transitionCounters, publications publicationCounters) *privatePublication {
	if e.state.binding == nil || e.state.lifecycle != lifecycleSuppressed ||
		(e.state.evaluatorIntegrity == nil && e.state.suppressionDisposition != SuppressionSameBindingRecoveryAllowed) ||
		e.lastPubID == math.MaxUint64 || node == nil || node.admissionTime.IsZero() {
		return nil
	}
	candidate, err := e.buildPublicationLocked(e.lastPubID+1, node.engineSequence, disposition, node.admissionTime.UTC(), admission, transitions, publications)
	if err != nil {
		return nil
	}
	evaluation := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	evaluation.mode, evaluation.reason, evaluation.rows, evaluation.tqIntentAvailable = rankingSuppressed, rankingReasonGlobalSuppression, nil, false
	evaluation.enrichedRows = 0
	candidate.aggregateEvaluation = evaluation
	candidate.currentMarketClaim = false
	candidate.tq = cloneTQView(e.tqViewLocked())
	candidate.tq.PublicationID = candidate.publicationID
	if validatePublication(candidate) != nil {
		return nil
	}
	return candidate
}

func cloneEvaluatorIntegrity(value *EvaluatorIntegrityView) *EvaluatorIntegrityView {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func validatePublication(candidate *privatePublication) error {
	if candidate == nil {
		return errors.New("nil private publication")
	}
	classifiedAdmissions := candidate.admission.admittedExternal + candidate.admission.notAdmittedInvalid + candidate.admission.notAdmittedCanceled +
		candidate.admission.notAdmittedClosed + candidate.admission.pressureShedOptional + candidate.admission.sequenceBudgetExhausted
	if candidate.kind != publicationNormal || candidate.schemaVersion != privatePublicationSchemaV1 ||
		candidate.publicationID == 0 || (candidate.mode != RunModeLive && candidate.mode != RunModeReplay) ||
		candidate.generatedAt.IsZero() || candidate.generatedAt != candidate.generatedAt.UTC() ||
		candidate.queueCapacity <= 1 || candidate.requiredReserve < 1 || candidate.requiredReserve >= candidate.queueCapacity ||
		candidate.queueOccupancy < 0 || candidate.queueOccupancy > candidate.queueCapacity ||
		candidate.admission.started != candidate.admission.inProgress+candidate.admission.resultsCommitted ||
		candidate.admission.resultsCommitted != classifiedAdmissions ||
		candidate.admission.admittedExternal != uint64(candidate.queueOccupancy)+candidate.admission.ownerInProgress+candidate.admission.completedExternal ||
		candidate.transitions.completedExternal != candidate.admission.completedExternal ||
		!candidate.transitions.reconciles() || !candidate.publications.reconciles(candidate.transitions.completedExternal+candidate.transitions.completedInternal) ||
		!candidate.aggregates.reconciles() || !candidate.connectionControls.reconciles() ||
		(candidate.hydrationActive && candidate.hydrationGeneration == 0) ||
		(candidate.hydrationActive && candidate.hydrationFenceReconciled) ||
		(candidate.hydrationGeneration > 0 && (!validHydrationPurpose(candidate.hydrationPurpose) ||
			candidate.hydrationStart.After(candidate.hydrationEnd) || !candidate.hydrationAccounting.reconciles() || !candidate.hydrationRows.reconciles())) {
		return errors.New("invalid private publication")
	}
	if candidate.watermark != nil && candidate.generatedAt.Before(*candidate.watermark) {
		return errors.New("publication generated before committed watermark")
	}
	if candidate.hydrationFenceReconciled && (candidate.hydrationFenceEpoch == 0 || candidate.hydrationFenceMarkerOrdinal == 0 || candidate.hydrationSupportedThrough == nil) {
		return errors.New("invalid private hydration fence")
	}
	if (candidate.hydrationPolicyAction == "") != (candidate.hydrationPolicyToken == 0) ||
		(candidate.hydrationPolicyAction != "" && !validHydrationPolicyAction(candidate.hydrationPolicyAction)) {
		return errors.New("invalid private hydration policy state")
	}
	if candidate.watermark == nil {
		if !candidate.aggregateEvaluation.at.IsZero() {
			return errors.New("evaluation without committed watermark")
		}
	} else if !candidate.aggregateEvaluation.at.Equal(*candidate.watermark) {
		return errors.New("evaluation watermark mismatch")
	}
	if (candidate.bindingIdentity == "") != (candidate.tradingDate == "") ||
		candidate.currentMarketClaim != (candidate.aggregateEvaluation.mode == rankingQualifiedCurrent || candidate.aggregateEvaluation.mode == rankingDegradedBootstrap || candidate.aggregateEvaluation.mode == rankingDegradedCurrent) ||
		validateAggregateEvaluation(candidate.aggregateEvaluation) != nil || !validTQPublication(candidate.tq, candidate.publicationID, candidate.aggregateEvaluation) {
		return errors.New("invalid publication claim")
	}
	if (candidate.connectionActive && candidate.connectionEpoch == 0) ||
		(candidate.aggregateAcknowledged && (!candidate.connectionActive || candidate.aggregateAckPosition.ConnectionEpoch != candidate.connectionEpoch || candidate.aggregateAckPosition.FrameSequence == 0)) ||
		(!candidate.aggregateAcknowledged && candidate.aggregateAckPosition != (LivePosition{})) ||
		(candidate.connectionControls.consumed == 0 && (candidate.latestControlKind != "" || candidate.latestControlOutcome != "")) ||
		(candidate.connectionControls.consumed > 0 && (!validConnectionControlKind(candidate.latestControlKind) || !validConnectionControlOutcome(candidate.latestControlOutcome))) {
		return errors.New("invalid connection/control publication")
	}
	return nil
}
