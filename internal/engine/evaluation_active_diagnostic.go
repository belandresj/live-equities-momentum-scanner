package engine

import "time"

// AggregateEvaluationPhase identifies the bounded phase of the one aggregate
// evaluation currently owned by the engine. It is diagnostic-only.
type AggregateEvaluationPhase string

const (
	AggregateEvaluationPhaseMaintenance AggregateEvaluationPhase = "maintenance"
	AggregateEvaluationPhaseStage       AggregateEvaluationPhase = "stage"
	AggregateEvaluationPhaseApply       AggregateEvaluationPhase = "apply"
	AggregateEvaluationPhasePublication AggregateEvaluationPhase = "publication"
)

// ActiveAggregateEvaluationView is a fixed-cardinality observation of an
// in-flight full-universe evaluation. It owns no market state and retains no
// symbol or event data.
type ActiveAggregateEvaluationView struct {
	EngineSequence            uint64
	Source                    AggregateEvaluationSource
	Target                    time.Time
	Phase                     AggregateEvaluationPhase
	StartedAt, PhaseStartedAt time.Time
}

func (e *Engine) ObserveActiveAggregateEvaluation() (ActiveAggregateEvaluationView, bool) {
	if e == nil {
		return ActiveAggregateEvaluationView{}, false
	}
	value := e.activeAggregateEvaluation.Load()
	if value == nil {
		return ActiveAggregateEvaluationView{}, false
	}
	return *value, true
}

func (e *Engine) startActiveAggregateEvaluationLocked(node *queueNode, target time.Time, phase AggregateEvaluationPhase) {
	e.startActiveAggregateEvaluationSourceLocked(node, target, phase, aggregateEvaluationSource(node))
}

func (e *Engine) startActiveAggregateEvaluationSourceLocked(node *queueNode, target time.Time, phase AggregateEvaluationPhase, source AggregateEvaluationSource) {
	now := e.evaluationTimingStart().UTC()
	value := &ActiveAggregateEvaluationView{
		EngineSequence: node.engineSequence, Source: source, Target: target,
		Phase: phase, StartedAt: now, PhaseStartedAt: now,
	}
	for {
		current := e.activeAggregateEvaluation.Load()
		if e.activeAggregateEvaluation.CompareAndSwap(current, value) {
			e.pauseActiveAggregateEvaluationPhaseForTest(phase)
			return
		}
	}
}

func (e *Engine) startOrAdvanceActiveAggregateEvaluationLocked(node *queueNode, target time.Time, phase AggregateEvaluationPhase) {
	current := e.activeAggregateEvaluation.Load()
	if current != nil && current.EngineSequence == node.engineSequence {
		e.advanceActiveAggregateEvaluationLocked(node.engineSequence, phase)
		return
	}
	e.startActiveAggregateEvaluationLocked(node, target, phase)
}

func (e *Engine) advanceActiveAggregateEvaluationLocked(sequence uint64, phase AggregateEvaluationPhase) {
	current := e.activeAggregateEvaluation.Load()
	if current == nil || current.EngineSequence != sequence {
		return
	}
	value := *current
	value.Phase, value.PhaseStartedAt = phase, e.evaluationTimingStart().UTC()
	if e.activeAggregateEvaluation.CompareAndSwap(current, &value) {
		e.pauseActiveAggregateEvaluationPhaseForTest(phase)
	}
}

func (e *Engine) pauseActiveAggregateEvaluationPhaseForTest(phase AggregateEvaluationPhase) {
	if e.beforeActiveAggregateEvaluationPhase != nil {
		e.beforeActiveAggregateEvaluationPhase(phase)
	}
}

func (e *Engine) clearActiveAggregateEvaluation(sequence uint64) {
	for {
		current := e.activeAggregateEvaluation.Load()
		if current == nil || current.EngineSequence != sequence {
			return
		}
		if e.activeAggregateEvaluation.CompareAndSwap(current, nil) {
			return
		}
	}
}

func aggregateEvaluationSource(node *queueNode) AggregateEvaluationSource {
	switch node.kind {
	case inputLiveCoverageFence:
		return AggregateEvaluationLiveCoverageFence
	case inputAggregateIngressFence:
		return AggregateEvaluationIngressFence
	case inputReplayGroup:
		return AggregateEvaluationReplay
	default:
		return AggregateEvaluationTimer
	}
}
