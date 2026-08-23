package engine

import "time"

// EvaluationCycleState is a fixed-cardinality, publication-only diagnostic
// view. It deliberately omits rows, symbols, and mutable engine state so the
// runtime can bracket one evaluation cycle without taking the engine mutex or
// cloning the full T/Q view.
type EvaluationCycleState struct {
	PublicationID, LastEngineSequence uint64
	RunMode                           RunMode
	Lifecycle                         string
	Watermark                         time.Time
	WatermarkPresent                  bool
	GeneratedAt                       time.Time
	TQPressure                        TQPressureMode
	TQPressureCause                   TQPressureCause
}

// ObserveEvaluationCycle reads the current immutable publication once.
// It is diagnostic-only and cannot provide a readiness or watermark decision.
func (e *Engine) ObserveEvaluationCycle() EvaluationCycleState {
	if e == nil {
		return EvaluationCycleState{}
	}
	publication := e.publication.Load()
	if publication == nil {
		return EvaluationCycleState{}
	}
	result := EvaluationCycleState{
		PublicationID: publication.publicationID, LastEngineSequence: publication.lastEngineSequence,
		RunMode: publication.mode, Lifecycle: string(publication.lifecycle),
		GeneratedAt: publication.generatedAt, TQPressure: publication.tq.Pressure, TQPressureCause: publication.tq.PressureCause,
	}
	if publication.watermark != nil {
		result.Watermark, result.WatermarkPresent = *publication.watermark, true
	}
	return result
}
