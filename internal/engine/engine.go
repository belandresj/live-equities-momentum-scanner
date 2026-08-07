// Package engine owns the scanner's single ordered state-mutation path.
package engine

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact/playback"
)

// RunMode fixes the evidence model for an engine's lifetime.
type RunMode string

const (
	RunModeLive   RunMode = "live"
	RunModeReplay RunMode = "replay"
)

// Clock supplies engine time. It is sampled by the engine, never by callers.
type Clock func() time.Time

// Config contains the S1 construction parameters. Capacity and RequiredReserve
// intentionally have no production defaults.
type Config struct {
	Mode            RunMode
	Clock           Clock
	Capacity        int
	RequiredReserve int
	EvaluationDelay *time.Duration
}

// AdmissionResult is the exhaustive result of one admission call.
type AdmissionResult string

const (
	AdmissionAdmitted                AdmissionResult = "admitted"
	AdmissionNotAdmittedInvalid      AdmissionResult = "not_admitted_invalid"
	AdmissionNotAdmittedCanceled     AdmissionResult = "not_admitted_canceled"
	AdmissionNotAdmittedClosed       AdmissionResult = "not_admitted_closed"
	AdmissionPressureShedOptional    AdmissionResult = "pressure_shed_optional"
	AdmissionSequenceBudgetExhausted AdmissionResult = "sequence_budget_exhausted"
)

// DispositionCode is the exhaustive S1 transition result.
type DispositionCode string

const (
	DispositionBindingInstalled          DispositionCode = "binding_installed"
	DispositionBindingInvalid            DispositionCode = "rejected_binding_invalid"
	DispositionBindingAlreadyInstalled   DispositionCode = "rejected_binding_already_installed"
	DispositionControlApplied            DispositionCode = "control_applied"
	DispositionSequenceExhausted         DispositionCode = "sequence_exhausted"
	DispositionAggregateInserted         DispositionCode = "aggregate_inserted"
	DispositionAggregateRevised          DispositionCode = "aggregate_revised"
	DispositionAggregateWithdrawn        DispositionCode = "aggregate_withdrawn_conflict"
	DispositionAggregateExactDuplicate   DispositionCode = "aggregate_exact_duplicate"
	DispositionAggregateRejected         DispositionCode = "aggregate_rejected"
	DispositionAggregateFenced           DispositionCode = "aggregate_fenced"
	DispositionAggregateIntegrity        DispositionCode = "aggregate_integrity_failure"
	DispositionTimerApplied              DispositionCode = "timer_applied"
	DispositionClockRegression           DispositionCode = "integrity_failure_clock_regression"
	DispositionPublicationIntegrity      DispositionCode = "integrity_failure_publication"
	DispositionAccountingIntegrity       DispositionCode = "integrity_failure_accounting"
	DispositionUnsupportedSchema         DispositionCode = "rejected_unsupported_schema"
	DispositionIllegalLifecycle          DispositionCode = "rejected_illegal_lifecycle"
	DispositionTerminal                  DispositionCode = "rejected_terminal"
	DispositionReplayStarted             DispositionCode = "replay_started"
	DispositionReplayEnded               DispositionCode = "replay_ended"
	DispositionReplayFailed              DispositionCode = "replay_failed"
	DispositionConnectionControlApplied  DispositionCode = "connection_control_applied"
	DispositionConnectionControlDeferred DispositionCode = "connection_control_consumer_deferred"
	DispositionConnectionControlRejected DispositionCode = "connection_control_rejected"
	DispositionConnectionControlFenced   DispositionCode = "connection_control_fenced"
	DispositionIngressIntegrity          DispositionCode = "ingress_integrity_failure"
)

// Disposition is an immutable completion value for one admitted input.
type Disposition struct {
	EngineSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
}

// TimerDisposition is the immutable completion of one admitted engine-owned
// timer fact. Callers choose neither its time nor either sequence.
type TimerDisposition struct {
	EngineSequence         uint64
	SystemSequence         uint64
	AdmissionTime          time.Time
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
}

const BindingInstallSchemaV1 = "engine-binding-install-v1"

// BindingInstall is the closed S1 binding-install payload. Kind, source, and
// system position are fixed by the typed admission method and assigned by the
// engine rather than selected by the caller.
type BindingInstall struct {
	SchemaVersion   string
	BindingIdentity string
	Binding         reference.Binding
}

type lifecycle string

const (
	lifecycleInitializing         lifecycle = "initializing"
	lifecycleAwaitingSession      lifecycle = "awaiting_session"
	lifecycleAwaitingAggregateAck lifecycle = "awaiting_aggregate_ack"
	lifecycleHydrating            lifecycle = "hydrating"
	lifecycleLive                 lifecycle = "live"
	lifecycleRecovering           lifecycle = "recovering"
	lifecycleReplaying            lifecycle = "replaying"
	lifecycleEnded                lifecycle = "ended"
	lifecycleSuppressed           lifecycle = "suppressed"
)

type lifecycleReason string

const (
	lifecycleReasonBindingBeforeSession lifecycleReason = "binding_before_session"
	lifecycleReasonBindingInSession     lifecycleReason = "binding_in_session"
	lifecycleReasonBindingAfterSession  lifecycleReason = "binding_after_session"
	lifecycleReasonSessionStart         lifecycleReason = "session_start_without_aggregate_ack"
	lifecycleReasonSessionEnd           lifecycleReason = "session_end"
	lifecycleReasonControlledStop       lifecycleReason = "controlled_stop"
	lifecycleReasonSequenceExhaustion   lifecycleReason = "sequence_exhaustion"
	lifecycleReasonClockRegression      lifecycleReason = "clock_regression"
	lifecycleReasonCanonicalIntegrity   lifecycleReason = "canonical_integrity"
	lifecycleReasonPublicationIntegrity lifecycleReason = "publication_integrity"
	lifecycleReasonAccountingIntegrity  lifecycleReason = "accounting_integrity"
	lifecycleReasonClosed               lifecycleReason = "closed"
	lifecycleReasonReplayStart          lifecycleReason = "replay_start"
	lifecycleReasonReplayEnd            lifecycleReason = "replay_end"
	lifecycleReasonReplayFailure        lifecycleReason = "replay_failure"
	lifecycleReasonAggregateAck         lifecycleReason = "aggregate_acknowledged"
	lifecycleReasonAggregateAckAtStart  lifecycleReason = "aggregate_acknowledged_at_session_start"
	lifecycleReasonAggregateEpochLost   lifecycleReason = "aggregate_epoch_lost"
	lifecycleReasonIngressIntegrity     lifecycleReason = "ingress_integrity"
	lifecycleReasonHydrationComplete    lifecycleReason = "hydration_complete"
	lifecycleReasonRecoveryExhausted    lifecycleReason = "recovery_exhausted"
)

// SuppressionDisposition is the exhaustive recovery requirement attached to
// every transition into suppressed. The empty value is valid only when the
// transition is not a suppression outcome.
type SuppressionDisposition string

const (
	SuppressionSameBindingRecoveryAllowed    SuppressionDisposition = "same_binding_recovery_allowed"
	SuppressionCleanReinitializationRequired SuppressionDisposition = "clean_reinitialization_required"
	SuppressionRestartRequired               SuppressionDisposition = "restart_required"
	SuppressionTerminalReplayFailure         SuppressionDisposition = "terminal_replay_failure"
)

type lifecycleEvent uint8

const (
	lifecycleEventBinding lifecycleEvent = iota + 1
	lifecycleEventTimer
	lifecycleEventStop
	lifecycleEventSequenceExhaustion
	lifecycleEventClockRegression
	lifecycleEventCanonicalIntegrity
	lifecycleEventPublicationIntegrity
	lifecycleEventAccountingIntegrity
	lifecycleEventClose
	lifecycleEventReplayStart
	lifecycleEventReplayEnd
	lifecycleEventReplayFailure
	lifecycleEventAggregateAck
	lifecycleEventAggregateLoss
	lifecycleEventIngressIntegrity
	lifecycleEventHydrationComplete
)

type transitionRecord struct {
	Previous, Next         lifecycle
	Reason                 lifecycleReason
	SuppressionDisposition SuppressionDisposition
	BindingIdentity        string
	Epoch, Generation      uint64
	AdmissionTime          time.Time
	EngineSequence         uint64
	CommittedT             *time.Time
}

type inputKind uint8

const (
	inputBinding inputKind = iota + 1
	inputControl
	inputStop
	inputAggregate
	inputTimer
	inputIllegal
	inputUnsupportedSchema
	inputReplayStart
	inputReplayGroup
	inputReplayEnd
	inputReplayFailure
	inputConnectionControl
	inputHydrationPlan
	inputHydrationChunk
	inputHydrationTerminal
	inputHydrationCancelProof
	inputAggregateIngressFence
	inputHydrationPolicyAction
)

type queueNode struct {
	kind                  inputKind
	binding               frozenBinding
	bindingID             string
	aggregate             frozenAggregateInput
	admissionTime         time.Time
	ordinal               uint64
	engineSequence        uint64
	systemSequence        uint64
	clockRegression       bool
	sealOnLink            bool
	completion            chan Disposition
	aggregateCompletion   chan AggregateDisposition
	timerCompletion       chan TimerDisposition
	replayStart           playback.StartEvidence
	replayGroup           playback.GroupEvidence
	replayEnd             playback.EndEvidence
	replayFailure         ReplayFailureInput
	connectionControl     frozenConnectionControlInput
	controlCompletion     chan ConnectionControlDisposition
	hydrationPlan         frozenHydrationPlanInput
	hydrationChunk        frozenHydrationChunkInput
	hydrationTerminal     frozenHydrationTerminalInput
	hydrationCancel       frozenHydrationCancelInput
	aggregateIngressFence frozenAggregateIngressFenceInput
	hydrationPolicy       frozenHydrationPolicyActionInput
	hydrationCompletion   chan HydrationDisposition
}

type engineState struct {
	lifecycle               lifecycle
	binding                 *installedBinding
	aggregates              aggregateAccounting
	liveEpoch               uint64
	liveEpochActive         bool
	aggregateWriteToken     uint64
	aggregateAcknowledged   bool
	aggregateAckPosition    LivePosition
	aggregateAckReceivedAt  time.Time
	greatestIngressPosition LivePosition
	connectionControl       connectionControlState
	connectionAccounting    connectionControlAccounting
	replayArtifact          string
	aggregateIntegrity      bool
	globalFailure           bool
	exposedRevision         uint64
	evaluationRevision      uint64
	aggregateEvaluator      aggregateEvaluatorState
	clockMonotonic          bool
	committedT              *time.Time
	latestTarget            *time.Time
	latestTransition        *transitionRecord
	suppressionDisposition  SuppressionDisposition
	replay                  replayState
	hydration               hydrationState
}

type admissionCounters struct {
	started                 uint64
	inProgress              uint64
	resultsCommitted        uint64
	admittedExternal        uint64
	notAdmittedInvalid      uint64
	notAdmittedCanceled     uint64
	notAdmittedClosed       uint64
	pressureShedOptional    uint64
	sequenceBudgetExhausted uint64
	ownerInProgress         uint64
	completedExternal       uint64
}

type transitionCounters struct {
	completedExternal uint64
	completedInternal uint64
	appliedMarket     uint64
	appliedNonmarket  uint64
	exactDuplicate    uint64
	rejected          uint64
	fenced            uint64
	terminalWorkFact  uint64
	integrityFailure  uint64
}

func (c transitionCounters) reconciles() bool {
	return c.completedExternal == c.appliedMarket+c.appliedNonmarket+c.exactDuplicate+c.rejected+c.fenced+c.terminalWorkFact+c.integrityFailure
}

type publicationCounters struct {
	completedDecisions   uint64
	noExposedChange      uint64
	publicationReplaced  uint64
	publicationIntegrity uint64
}

func (c publicationCounters) reconciles(completedTransitions uint64) bool {
	return c.completedDecisions == completedTransitions &&
		c.completedDecisions == c.noExposedChange+c.publicationReplaced+c.publicationIntegrity
}

// Engine is the sole FIFO, sequence authority, mutable state owner, and private
// publication owner. Its mutable graph never crosses the immutable read cell.
type Engine struct {
	mu       sync.Mutex
	mode     RunMode
	clock    Clock
	capacity int
	reserve  int
	delay    time.Duration

	queue     []*queueNode
	changed   chan struct{}
	done      chan struct{}
	sealed    bool
	exhausted bool

	lastReserved uint64
	nextSequence uint64
	lastSystem   uint64
	lastClock    time.Time
	hasClock     bool
	state        *engineState
	counters     admissionCounters
	transitions  transitionCounters
	publications publicationCounters
	publication  atomic.Pointer[privatePublication]
	sentinels    [8]*privatePublication
	lastPubID    uint64

	// Test-only fault/pause points are package-private and have no production
	// constructor or exported mutation path.
	buildCandidate   func(frozenBinding) (*installedBinding, error)
	beforeConsume    func(*queueNode)
	terminal         *Disposition
	publicationFault publicationFault
	evaluationFault  bool
}

// New constructs an unbound engine shell and starts its sole consumer.
func New(config Config) (*Engine, error) {
	if (config.Mode != RunModeLive && config.Mode != RunModeReplay) || config.Clock == nil || config.EvaluationDelay == nil || *config.EvaluationDelay < 0 ||
		config.Capacity <= 1 || config.RequiredReserve < 1 || config.RequiredReserve >= config.Capacity {
		return nil, errors.New("engine requires live/replay mode, a clock, explicit nonnegative evaluation delay, and finite 1 <= R < C")
	}
	e := &Engine{
		mode: config.Mode, clock: config.Clock, capacity: config.Capacity, reserve: config.RequiredReserve, delay: *config.EvaluationDelay,
		queue: make([]*queueNode, 0, config.Capacity), changed: make(chan struct{}), done: make(chan struct{}),
		nextSequence: 1, state: &engineState{lifecycle: lifecycleInitializing, clockMonotonic: true},
	}
	e.buildCandidate = buildInstalledBinding
	e.installInitialPublication()
	go e.consume()
	return e, nil
}

// AdmitBinding blocks for required capacity, respecting cancellation. A
// successful result transfers ownership and returns the only completion for
// the linked immutable node.
func (e *Engine) AdmitBinding(ctx context.Context, input BindingInstall) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || input.SchemaVersion != BindingInstallSchemaV1 || !validIdentityShape(input.BindingIdentity) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	frozen := freezeBinding(input.Binding)
	return e.admit(ctx, &queueNode{kind: inputBinding, binding: frozen, bindingID: input.BindingIdentity}, false)
}

// AdmitAggregate transfers one normalized aggregate value into the same S1
// FIFO. Historical acceptance additionally requires the package-private S2
// proof context or Component 6's production hydration token ledger.
func (e *Engine) AdmitAggregate(ctx context.Context, input AggregateInput) (AdmissionResult, <-chan AggregateDisposition) {
	e.beginAdmission()
	if ctx == nil || !boundedAggregateInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admitAggregate(ctx, &queueNode{kind: inputAggregate, aggregate: freezeAggregateInput(input)})
}

// AdmitTimer admits intent to evaluate engine time. The engine supplies the
// serialized time sample, binding context, and positive run-local position.
func (e *Engine) AdmitTimer(ctx context.Context) (AdmissionResult, <-chan TimerDisposition) {
	e.beginAdmission()
	if ctx == nil {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputTimer}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.timerCompletion
}

// Stop admits the S1 controlled-stop variant. Its successful linkage seals the
// FIFO in the same critical section, so it is the last external node and its
// ordered transition applies LIFE-T05 after all earlier nodes.
func (e *Engine) Stop(ctx context.Context) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputStop, sealOnLink: true}, false)
}

func (e *Engine) beginAdmission() {
	e.mu.Lock()
	e.counters.started++
	e.counters.inProgress++
	e.mu.Unlock()
}

func (e *Engine) finishNonAdmission(result AdmissionResult) AdmissionResult {
	e.mu.Lock()
	e.commitNonAdmissionLocked(result)
	e.mu.Unlock()
	return result
}

func (e *Engine) admit(ctx context.Context, node *queueNode, optional bool) (AdmissionResult, <-chan Disposition) {
	result := e.admitNode(ctx, node, optional)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.completion
}

func (e *Engine) admitAggregate(ctx context.Context, node *queueNode) (AdmissionResult, <-chan AggregateDisposition) {
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.aggregateCompletion
}

func (e *Engine) admitNode(ctx context.Context, node *queueNode, optional bool) AdmissionResult {
	for {
		e.mu.Lock()
		if ctx.Err() != nil {
			e.commitNonAdmissionLocked(AdmissionNotAdmittedCanceled)
			e.mu.Unlock()
			return AdmissionNotAdmittedCanceled
		}
		if e.sealed {
			e.commitNonAdmissionLocked(AdmissionNotAdmittedClosed)
			e.mu.Unlock()
			return AdmissionNotAdmittedClosed
		}
		if e.lastReserved == math.MaxUint64-1 {
			e.sealed = true
			e.exhausted = true
			e.commitNonAdmissionLocked(AdmissionSequenceBudgetExhausted)
			e.broadcastLocked()
			e.mu.Unlock()
			return AdmissionSequenceBudgetExhausted
		}
		if (node.kind == inputTimer || node.kind == inputReplayGroup) && e.lastSystem == math.MaxUint64 {
			e.sealed = true
			e.exhausted = true
			e.commitNonAdmissionLocked(AdmissionSequenceBudgetExhausted)
			e.broadcastLocked()
			e.mu.Unlock()
			return AdmissionSequenceBudgetExhausted
		}
		if optional && len(e.queue) >= e.capacity-e.reserve {
			e.commitNonAdmissionLocked(AdmissionPressureShedOptional)
			e.mu.Unlock()
			return AdmissionPressureShedOptional
		}
		if len(e.queue) < e.capacity {
			actual := e.clock().UTC()
			if e.hasClock && actual.Before(e.lastClock) {
				node.clockRegression = true
				node.admissionTime = e.lastClock
			} else {
				e.lastClock = actual
				e.hasClock = true
				node.admissionTime = actual
			}
			e.lastReserved++
			node.ordinal = e.lastReserved
			if node.kind == inputAggregate {
				node.aggregateCompletion = make(chan AggregateDisposition, 1)
			} else if node.kind == inputConnectionControl {
				node.controlCompletion = make(chan ConnectionControlDisposition, 1)
			} else if hydrationInputKind(node.kind) {
				node.hydrationCompletion = make(chan HydrationDisposition, 1)
			} else if node.kind == inputTimer || node.kind == inputReplayGroup {
				if e.state.binding != nil {
					node.bindingID = e.state.binding.identity
				}
				e.lastSystem++
				node.systemSequence = e.lastSystem
				node.timerCompletion = make(chan TimerDisposition, 1)
			} else {
				node.completion = make(chan Disposition, 1)
			}
			e.queue = append(e.queue, node)
			e.counters.inProgress--
			e.counters.resultsCommitted++
			e.counters.admittedExternal++
			if node.sealOnLink {
				e.sealed = true
			}
			e.broadcastLocked()
			e.mu.Unlock()
			return AdmissionAdmitted
		}
		changed := e.changed
		e.mu.Unlock()
		select {
		case <-ctx.Done():
			e.mu.Lock()
			// Cancellation wins only if it is observed before linkage under
			// the admission serialization lock.
			e.commitNonAdmissionLocked(AdmissionNotAdmittedCanceled)
			e.mu.Unlock()
			return AdmissionNotAdmittedCanceled
		case <-changed:
		}
	}
}

func (e *Engine) commitNonAdmissionLocked(result AdmissionResult) {
	e.counters.inProgress--
	e.counters.resultsCommitted++
	switch result {
	case AdmissionNotAdmittedInvalid:
		e.counters.notAdmittedInvalid++
	case AdmissionNotAdmittedCanceled:
		e.counters.notAdmittedCanceled++
	case AdmissionNotAdmittedClosed:
		e.counters.notAdmittedClosed++
	case AdmissionPressureShedOptional:
		e.counters.pressureShedOptional++
	case AdmissionSequenceBudgetExhausted:
		e.counters.sequenceBudgetExhausted++
	}
}

// Close seals the FIFO immediately, wakes blocked admissions, and lets the
// sole consumer drain every node linked before the seal boundary.
func (e *Engine) Close() {
	e.mu.Lock()
	if !e.sealed {
		e.sealed = true
		e.broadcastLocked()
	}
	e.mu.Unlock()
}

// Wait waits for the captured FIFO boundary to drain. Close must be called by
// the owner (or sequence exhaustion must have sealed the engine).
func (e *Engine) Wait(ctx context.Context) error {
	if ctx == nil {
		return errors.New("wait context is required")
	}
	select {
	case <-e.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) consume() {
	defer close(e.done)
	for {
		e.mu.Lock()
		for len(e.queue) == 0 && !e.sealed {
			changed := e.changed
			e.mu.Unlock()
			<-changed
			e.mu.Lock()
		}
		if len(e.queue) == 0 && e.sealed {
			if e.exhausted {
				e.applyExhaustionLocked()
			} else if !e.state.globalFailure && e.state.lifecycle != lifecycleEnded {
				before := e.publicationFingerprintLocked()
				if e.state.hydration.generation.active {
					e.cancelHydrationGenerationLocked(false)
					e.state.hydration.generation.active = false
				}
				e.transitionLifecycleLocked(lifecycleEventClose, nil, lifecycleReasonClosed)
				e.finishInternalTransitionLocked(e.nextSequence-1, before, false)
			}
			e.mu.Unlock()
			return
		}
		node := e.queue[0]
		copy(e.queue, e.queue[1:])
		e.queue[len(e.queue)-1] = nil
		e.queue = e.queue[:len(e.queue)-1]
		node.engineSequence = e.nextSequence
		e.nextSequence++
		e.counters.ownerInProgress++
		e.broadcastLocked()
		e.mu.Unlock()

		if e.beforeConsume != nil {
			e.beforeConsume(node)
		}
		disposition := e.transition(node)

		if node.kind == inputAggregate {
			node.aggregateCompletion <- AggregateDisposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
			close(node.aggregateCompletion)
		} else if node.kind == inputConnectionControl {
			node.controlCompletion <- ConnectionControlDisposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
			close(node.controlCompletion)
		} else if hydrationInputKind(node.kind) {
			node.hydrationCompletion <- HydrationDisposition{
				EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason,
				SuppressionDisposition: disposition.SuppressionDisposition, Plan: disposition.hydrationPlan,
				Rows: disposition.hydrationRows, Accounting: disposition.hydrationAccounting,
				FenceCommand: disposition.hydrationFenceCommand,
			}
			close(node.hydrationCompletion)
		} else if node.kind == inputTimer || node.kind == inputReplayGroup {
			node.timerCompletion <- TimerDisposition{EngineSequence: disposition.EngineSequence, SystemSequence: node.systemSequence, AdmissionTime: node.admissionTime, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
			close(node.timerCompletion)
		} else {
			node.completion <- Disposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
			close(node.completion)
		}
	}
}

type transitionDisposition struct {
	EngineSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
	hydrationPlan          HydrationPlanResult
	hydrationRows          HydrationRowAccounting
	hydrationAccounting    HydrationAccounting
	hydrationFenceCommand  HydrationFenceCommand
}

func (e *Engine) transition(node *queueNode) transitionDisposition {
	code := DispositionIllegalLifecycle
	reason := ReasonLifecycle
	var stagedHydrationPlan HydrationPlanResult
	var stagedHydrationRows HydrationRowAccounting
	var stagedHydrationAccounting HydrationAccounting
	var stagedHydrationFenceCommand HydrationFenceCommand
	e.mu.Lock()
	before := e.publicationFingerprintLocked()
	if e.state.lifecycle == lifecycleEnded {
		disposition := transitionDisposition{EngineSequence: node.engineSequence, Code: DispositionTerminal, Reason: ReasonTerminal}
		return e.finishTransitionLocked(node, disposition, before, false)
	}
	if e.state.globalFailure {
		if hydrationInputKind(node.kind) {
			disposition := transitionDisposition{
				EngineSequence: node.engineSequence, Code: DispositionHydrationFenced, Reason: ReasonHistoricalContext,
				SuppressionDisposition: e.state.suppressionDisposition,
				hydrationAccounting:    e.state.hydration.generation.accounting,
			}
			return e.finishTransitionLocked(node, disposition, before, false)
		}
		disposition := transitionDisposition{EngineSequence: node.engineSequence, Code: DispositionTerminal, Reason: ReasonTerminal, SuppressionDisposition: e.state.suppressionDisposition}
		return e.finishTransitionLocked(node, disposition, before, false)
	}
	if node.clockRegression {
		e.state.clockMonotonic = false
		dispositionValue := e.enterSuppressionLocked(lifecycleEventClockRegression, node, lifecycleReasonClockRegression)
		disposition := transitionDisposition{EngineSequence: node.engineSequence, Code: DispositionClockRegression, Reason: ReasonClockRegression, SuppressionDisposition: dispositionValue}
		return e.finishTransitionLocked(node, disposition, before, true)
	}
	e.mu.Unlock()
	if node.kind == inputControl {
		code, reason = DispositionControlApplied, ReasonNone
	} else if node.kind == inputStop {
		e.mu.Lock()
		if e.state.hydration.generation.active {
			e.cancelHydrationGenerationLocked(false)
			e.state.hydration.generation.active = false
		}
		e.transitionLifecycleLocked(lifecycleEventStop, node, lifecycleReasonControlledStop)
		e.mu.Unlock()
		code, reason = DispositionControlApplied, ReasonNone
	} else if node.kind == inputBinding {
		reason = ReasonNone
		candidate, err := e.buildCandidate(node.binding)
		e.mu.Lock()
		current := e.state
		switch {
		case current.binding != nil:
			code = DispositionBindingAlreadyInstalled
		case err != nil || node.bindingID != node.binding.identity:
			code = DispositionBindingInvalid
		case current.lifecycle != lifecycleInitializing:
			code, reason = DispositionIllegalLifecycle, ReasonLifecycle
		default:
			e.state.binding = candidate
			if e.transitionLifecycleLocked(lifecycleEventBinding, node, "") {
				code = DispositionBindingInstalled
			} else {
				e.state.binding = nil
				code, reason = DispositionIllegalLifecycle, ReasonLifecycle
			}
		}
		e.mu.Unlock()
	} else if node.kind == inputAggregate {
		e.mu.Lock()
		code, reason = e.applyAggregateLocked(node.aggregate, node.admissionTime)
		if node.aggregate.Source == AggregateSourceLive && node.aggregate.Live.ConnectionEpoch == e.state.liveEpoch && node.aggregate.Live.FrameSequence > 0 &&
			(e.state.greatestIngressPosition.ConnectionEpoch == 0 || compareLive(node.aggregate.Live, e.state.greatestIngressPosition) > 0) {
			e.state.greatestIngressPosition = node.aggregate.Live
		}
		if code == DispositionAggregateInserted || code == DispositionAggregateRevised || code == DispositionAggregateWithdrawn {
			e.state.exposedRevision++
		}
		if node.aggregate.Source == AggregateSourceLive && (code == DispositionAggregateInserted || code == DispositionAggregateRevised) {
			if index, ok := e.state.binding.index[node.aggregate.Symbol]; ok && e.state.aggregateEvaluator.coverage[index] == coverageNoPrintThroughT {
				delete(e.state.aggregateEvaluator.coverage, index)
			}
		}
		if code == DispositionAggregateIntegrity {
			e.enterSuppressionLocked(lifecycleEventCanonicalIntegrity, node, lifecycleReasonCanonicalIntegrity)
		}
		e.mu.Unlock()
	} else if node.kind == inputTimer {
		e.mu.Lock()
		code, reason = e.applyTimerLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputConnectionControl {
		e.mu.Lock()
		code, reason = e.applyConnectionControlLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputHydrationPlan {
		e.mu.Lock()
		var plan HydrationPlanResult
		code, reason, plan = e.applyHydrationPlanLocked(node)
		if code == DispositionHydrationIntegrity {
			if !e.cancelHydrationGenerationLocked(true) {
				reason = ReasonAccounting
			}
			e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
		}
		nodeResult := e.state.hydration.generation.accounting
		e.mu.Unlock()
		stagedHydrationPlan, stagedHydrationAccounting = plan, nodeResult
	} else if node.kind == inputHydrationChunk {
		e.mu.Lock()
		var rows HydrationRowAccounting
		code, reason, rows = e.applyHydrationChunkLocked(node)
		if code == DispositionHydrationIntegrity {
			if !e.cancelHydrationGenerationLocked(true) {
				reason = ReasonAccounting
			}
			e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
		}
		nodeRows, nodeResult := rows, e.state.hydration.generation.accounting
		e.mu.Unlock()
		stagedHydrationRows, stagedHydrationAccounting = nodeRows, nodeResult
	} else if node.kind == inputHydrationTerminal {
		e.mu.Lock()
		code, reason = e.applyHydrationTerminalLocked(node)
		if code == DispositionHydrationIntegrity {
			if !e.cancelHydrationGenerationLocked(true) {
				reason = ReasonAccounting
			}
			e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
		}
		nodeResult := e.state.hydration.generation.accounting
		if code == DispositionHydrationTerminalApplied {
			stagedHydrationFenceCommand = e.state.hydration.generation.fenceCommand
		}
		e.mu.Unlock()
		stagedHydrationAccounting = nodeResult
	} else if node.kind == inputAggregateIngressFence {
		e.mu.Lock()
		code, reason = e.applyAggregateIngressFenceLocked(node)
		nodeResult := e.state.hydration.generation.accounting
		e.mu.Unlock()
		stagedHydrationAccounting = nodeResult
	} else if node.kind == inputHydrationPolicyAction {
		e.mu.Lock()
		code, reason = e.applyHydrationPolicyActionLocked(node)
		nodeResult := e.state.hydration.generation.accounting
		e.mu.Unlock()
		stagedHydrationAccounting = nodeResult
	} else if node.kind == inputHydrationCancelProof {
		e.mu.Lock()
		code, reason = e.applyHydrationCancelProofLocked(node)
		if code == DispositionHydrationIntegrity {
			if !e.cancelHydrationGenerationLocked(true) {
				reason = ReasonAccounting
			}
			e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
		}
		nodeResult := e.state.hydration.generation.accounting
		e.mu.Unlock()
		stagedHydrationAccounting = nodeResult
	} else if node.kind == inputReplayStart {
		e.mu.Lock()
		code, reason = e.applyReplayStartLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputReplayGroup {
		e.mu.Lock()
		code, reason = e.applyReplayGroupLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputReplayEnd {
		e.mu.Lock()
		code, reason = e.applyReplayEndLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputReplayFailure {
		e.mu.Lock()
		code, reason = e.applyReplayFailureLocked(node)
		e.mu.Unlock()
	} else if node.kind == inputIllegal {
		code, reason = DispositionIllegalLifecycle, ReasonLifecycle
	} else if node.kind == inputUnsupportedSchema {
		code, reason = DispositionUnsupportedSchema, ReasonSchema
	}
	if code == DispositionReplayFailed {
		e.mu.Lock()
		if !e.state.globalFailure {
			e.state.replay.terminal = true
			if e.state.replay.failureReason == "" {
				e.state.replay.failureReason = ReplayFailureEngine
				e.state.replay.failureLogical = e.state.replay.lastGroup
				if e.state.replay.nextOrdinal > 0 {
					e.state.replay.failureOrdinal = e.state.replay.nextOrdinal - 1
				}
			}
			e.enterSuppressionLocked(lifecycleEventReplayFailure, node, lifecycleReasonReplayFailure)
		}
		e.mu.Unlock()
	}
	// Future approved contributors may be inserted only here as explicit,
	// statically named synchronous calls in fixed source order. Each call must
	// receive a copied transition projection plus its named bounded engine-owned
	// substate and return a bounded typed value for the owner to copy/apply. S4
	// deliberately installs no collection, callback, or registry. Component 3's
	// aggregate-feature contributor is the first statically named call at this
	// seam; C3-S2 extends that same call rather than adding another path.
	e.mu.Lock()
	stagedEvaluation := e.runAggregateFeatureContributorLocked(node, code, reason)
	if !e.runAggregateEvaluatorLocked(node, code, reason, stagedEvaluation) {
		code, reason = DispositionAccountingIntegrity, ReasonAccounting
		e.enterSuppressionLocked(lifecycleEventAccountingIntegrity, node, lifecycleReasonAccountingIntegrity)
	}
	disposition := transitionDisposition{EngineSequence: node.engineSequence, Code: code, Reason: reason,
		hydrationPlan: stagedHydrationPlan, hydrationRows: stagedHydrationRows, hydrationAccounting: stagedHydrationAccounting,
		hydrationFenceCommand: stagedHydrationFenceCommand}
	if code == DispositionAggregateIntegrity || code == DispositionAccountingIntegrity || code == DispositionReplayFailed || code == DispositionIngressIntegrity || code == DispositionHydrationIntegrity {
		disposition.SuppressionDisposition = e.state.suppressionDisposition
	}
	forceUnavailable := code == DispositionAggregateIntegrity || code == DispositionAccountingIntegrity || code == DispositionReplayFailed || code == DispositionIngressIntegrity || code == DispositionHydrationIntegrity
	return e.finishTransitionLocked(node, disposition, before, forceUnavailable)
}

func (e *Engine) applyExhaustionLocked() {
	before := e.publicationFingerprintLocked()
	dispositionValue := e.enterSuppressionLocked(lifecycleEventSequenceExhaustion, &queueNode{engineSequence: math.MaxUint64, admissionTime: e.lastClock}, lifecycleReasonSequenceExhaustion)
	disposition := Disposition{EngineSequence: math.MaxUint64, Code: DispositionSequenceExhausted, SuppressionDisposition: dispositionValue}
	e.terminal = &disposition
	e.finishInternalTransitionLocked(math.MaxUint64, before, true)
}

func (e *Engine) broadcastLocked() {
	close(e.changed)
	e.changed = make(chan struct{})
}
