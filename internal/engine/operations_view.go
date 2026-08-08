package engine

import "time"

// OperationalView is Component 8's defensive, fixed-cardinality projection of
// the existing immutable engine publication. It adds no readiness owner.
type OperationalView struct {
	PublicationID, LastEngineSequence uint64
	BindingIdentity, TradingDate      string
	RunMode                           RunMode
	Lifecycle, LifecycleReason        string
	Suppression                       SuppressionDisposition
	Watermark                         *time.Time
	GeneratedAt                       time.Time
	RankingMode, RankingReason        string
	CurrentMarketClaim                bool
	QueueCapacity, RequiredReserve    int
	QueueOccupancy                    int
	Admissions                        OperationalAdmissions
	Transitions                       OperationalTransitions
	Aggregates                        OperationalAggregates
	Connection                        OperationalConnection
	Hydration                         OperationalHydration
	InstalledCheckpoint               bool
}

type OperationalAdmissions struct {
	Started, InProgress, ResultsCommitted, Admitted, Invalid, Canceled, Closed uint64
	PressureShedOptional, SequenceExhausted, OwnerInProgress, Completed        uint64
}

func (a OperationalAdmissions) Reconciles(queue int) bool {
	classified := a.Admitted + a.Invalid + a.Canceled + a.Closed + a.PressureShedOptional + a.SequenceExhausted
	return a.Started == a.InProgress+a.ResultsCommitted && a.ResultsCommitted == classified &&
		a.Admitted == uint64(queue)+a.OwnerInProgress+a.Completed
}

type OperationalTransitions struct {
	CompletedExternal, CompletedInternal, AppliedMarket, AppliedNonmarket uint64
	ExactDuplicate, Rejected, Fenced, TerminalWorkFact, IntegrityFailure  uint64
}

func (t OperationalTransitions) Reconciles() bool {
	return t.CompletedExternal == t.AppliedMarket+t.AppliedNonmarket+t.ExactDuplicate+t.Rejected+t.Fenced+t.TerminalWorkFact+t.IntegrityFailure
}

type OperationalAggregates struct {
	Consumed, Inserted, Revised, WithdrawnConflict, ExactDuplicate, Rejected, Fenced, Integrity uint64
}

func (a OperationalAggregates) Reconciles() bool {
	return a.Consumed == a.Inserted+a.Revised+a.WithdrawnConflict+a.ExactDuplicate+a.Rejected+a.Fenced+a.Integrity
}

type OperationalConnection struct {
	Epoch, AckFrame                                          uint64
	AckPosition                                              LivePosition
	RecoveryAttempts                                         uint64
	Active, Acknowledged                                     bool
	Consumed, Applied, Deferred, Rejected, Fenced, Integrity uint64
}

func (c OperationalConnection) Reconciles() bool {
	return c.Consumed == c.Applied+c.Deferred+c.Rejected+c.Fenced+c.Integrity
}

type OperationalHydration struct {
	Purpose                                      HydrationPurpose
	Generation                                   uint64
	Start, End                                   time.Time
	Accounting                                   HydrationAccounting
	Rows                                         HydrationRowAccounting
	FenceReconciled                              bool
	FenceEpoch, FenceThrough, FenceMarkerOrdinal uint64
	SupportedThrough                             *time.Time
	PolicyWaiting                                bool
}

func (e *Engine) ObserveOperational() OperationalView {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.observePublication()
	lastSequence := uint64(0)
	if e.nextSequence > 0 {
		lastSequence = e.nextSequence - 1
	}
	counters, transitions, state := e.counters, e.transitions, e.state
	return OperationalView{
		PublicationID: p.publicationID, LastEngineSequence: lastSequence,
		BindingIdentity: p.bindingIdentity, TradingDate: p.tradingDate, RunMode: p.mode,
		Lifecycle: string(p.lifecycle), LifecycleReason: string(p.lifecycleReason), Suppression: p.suppressionDisposition,
		Watermark: immutableTimePointer(p.watermark), GeneratedAt: p.generatedAt,
		RankingMode: string(p.aggregateEvaluation.mode), RankingReason: string(p.aggregateEvaluation.reason), CurrentMarketClaim: p.currentMarketClaim,
		QueueCapacity: e.capacity, RequiredReserve: e.reserve, QueueOccupancy: len(e.queue),
		Admissions:  OperationalAdmissions{counters.started, counters.inProgress, counters.resultsCommitted, counters.admittedExternal, counters.notAdmittedInvalid, counters.notAdmittedCanceled, counters.notAdmittedClosed, counters.pressureShedOptional, counters.sequenceBudgetExhausted, counters.ownerInProgress, counters.completedExternal},
		Transitions: OperationalTransitions{transitions.completedExternal, transitions.completedInternal, transitions.appliedMarket, transitions.appliedNonmarket, transitions.exactDuplicate, transitions.rejected, transitions.fenced, transitions.terminalWorkFact, transitions.integrityFailure},
		Aggregates:  OperationalAggregates{state.aggregates.consumed, state.aggregates.inserted, state.aggregates.revised, state.aggregates.withdrawnConflict, state.aggregates.exactDuplicate, state.aggregates.rejected, state.aggregates.fenced, state.aggregates.integrity},
		Connection: OperationalConnection{Epoch: state.liveEpoch, AckFrame: state.aggregateAckPosition.FrameSequence, AckPosition: state.aggregateAckPosition,
			RecoveryAttempts: state.connectionControl.recoveryAttempts, Active: state.liveEpochActive, Acknowledged: state.aggregateAcknowledged,
			Consumed: state.connectionAccounting.consumed, Applied: state.connectionAccounting.applied, Deferred: state.connectionAccounting.deferred,
			Rejected: state.connectionAccounting.rejected, Fenced: state.connectionAccounting.fenced, Integrity: state.connectionAccounting.integrity},
		Hydration: OperationalHydration{Purpose: state.hydration.generation.purpose, Generation: state.hydration.generation.generation, Start: state.hydration.generation.start, End: state.hydration.generation.end,
			Accounting: state.hydration.generation.accounting, Rows: state.hydration.generation.rowAccounting, FenceReconciled: state.hydration.fenceReconciled,
			FenceEpoch: state.hydration.fenceEpoch, FenceThrough: state.hydration.fenceThrough, FenceMarkerOrdinal: state.hydration.fenceMarkerOrdinal,
			SupportedThrough: immutableTimePointer(state.hydration.supportedThrough), PolicyWaiting: state.hydration.policyWaiting},
		InstalledCheckpoint: state.installedCheckpoint != nil,
	}
}

func operationalViewFromPublication(p *privatePublication) OperationalView {
	if p == nil {
		return OperationalView{}
	}
	return OperationalView{
		PublicationID: p.publicationID, LastEngineSequence: p.lastEngineSequence,
		BindingIdentity: p.bindingIdentity, TradingDate: p.tradingDate, RunMode: p.mode,
		Lifecycle: string(p.lifecycle), LifecycleReason: string(p.lifecycleReason), Suppression: p.suppressionDisposition,
		Watermark: immutableTimePointer(p.watermark), GeneratedAt: p.generatedAt,
		RankingMode: string(p.aggregateEvaluation.mode), RankingReason: string(p.aggregateEvaluation.reason), CurrentMarketClaim: p.currentMarketClaim,
		QueueCapacity: p.queueCapacity, RequiredReserve: p.requiredReserve, QueueOccupancy: p.queueOccupancy,
		Admissions:  OperationalAdmissions{p.admission.started, p.admission.inProgress, p.admission.resultsCommitted, p.admission.admittedExternal, p.admission.notAdmittedInvalid, p.admission.notAdmittedCanceled, p.admission.notAdmittedClosed, p.admission.pressureShedOptional, p.admission.sequenceBudgetExhausted, p.admission.ownerInProgress, p.admission.completedExternal},
		Transitions: OperationalTransitions{p.transitions.completedExternal, p.transitions.completedInternal, p.transitions.appliedMarket, p.transitions.appliedNonmarket, p.transitions.exactDuplicate, p.transitions.rejected, p.transitions.fenced, p.transitions.terminalWorkFact, p.transitions.integrityFailure},
		Aggregates:  OperationalAggregates{p.aggregates.consumed, p.aggregates.inserted, p.aggregates.revised, p.aggregates.withdrawnConflict, p.aggregates.exactDuplicate, p.aggregates.rejected, p.aggregates.fenced, p.aggregates.integrity},
		Connection: OperationalConnection{Epoch: p.connectionEpoch, AckFrame: p.aggregateAckPosition.FrameSequence, AckPosition: p.aggregateAckPosition,
			RecoveryAttempts: p.connectionRecoveryAttempts, Active: p.connectionActive, Acknowledged: p.aggregateAcknowledged,
			Consumed: p.connectionControls.consumed, Applied: p.connectionControls.applied, Deferred: p.connectionControls.deferred,
			Rejected: p.connectionControls.rejected, Fenced: p.connectionControls.fenced, Integrity: p.connectionControls.integrity},
		Hydration: OperationalHydration{Purpose: p.hydrationPurpose, Generation: p.hydrationGeneration, Start: p.hydrationStart, End: p.hydrationEnd,
			Accounting: p.hydrationAccounting, Rows: p.hydrationRows, FenceReconciled: p.hydrationFenceReconciled,
			FenceEpoch: p.hydrationFenceEpoch, FenceThrough: p.hydrationFenceThrough, FenceMarkerOrdinal: p.hydrationFenceMarkerOrdinal,
			SupportedThrough: immutableTimePointer(p.hydrationSupportedThrough), PolicyWaiting: p.hydrationPolicyWaiting},
		InstalledCheckpoint: p.installedCheckpoint,
	}
}
