package operations

import (
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// WatermarkStallRingCapacity is the fixed maximum ordinary-cycle tail retained
// for the private watermark-stale diagnostic.
const WatermarkStallRingCapacity = 120

type WatermarkStallTimerPolicy string

const (
	WatermarkStallTimerMaintenanceOnly    WatermarkStallTimerPolicy = "maintenance_only"
	WatermarkStallTimerEvaluationFallback WatermarkStallTimerPolicy = "evaluation_fallback"
)

// WatermarkStallCycleEvidence contains only fixed-cardinality, symbol-free
// facts. It is not part of SnapshotCapture or the public API schema.
type WatermarkStallCycleEvidence struct {
	CycleStartedAt, CycleCompletedAt time.Time
	CycleDuration                    time.Duration
	DiagnosticSampledAt              time.Time

	CoverageFenceDisposition  engine.DispositionCode
	CoverageFenceFinalization time.Duration
	CoverageFenceTotal        time.Duration
	CoverageFenceValid        bool
	TimerPolicy               WatermarkStallTimerPolicy

	EvaluationObserved    bool
	EvaluationSource      engine.AggregateEvaluationSource
	EvaluationTarget      time.Time
	EvaluationStage       time.Duration
	EvaluationApply       time.Duration
	EvaluationPublication time.Duration

	WatermarkBefore, WatermarkAfter       time.Time
	WatermarkBeforePresent                bool
	WatermarkAfterPresent                 bool
	CausalTargetBefore, CausalTargetAfter time.Time
	CausalTargetBeforePresent             bool
	CausalTargetAfterPresent              bool
	WatermarkLagBefore, WatermarkLagAfter time.Duration

	PublicationIDBefore, PublicationIDAfter               uint64
	EngineSequenceBefore, EngineSequenceAfter             uint64
	PublicationGeneratedBefore, PublicationGeneratedAfter time.Time

	QueueCurrentFrames, QueueHighFrames       uint64
	QueueCurrentBytes, QueueHighBytes         int
	QueueOldestWaitingAge                     time.Duration
	ProcessingDelay, ProcessingDelayOneSecond time.Duration

	TQPressure                     engine.TQPressureMode
	TQPressureCause                engine.TQPressureCause
	HeapAllocBytes, HeapInUseBytes uint64
	Goroutines                     int
	GCCycles                       uint32
	GCPauseTotalNS, LastGCPauseNS  uint64
}

// WatermarkStallEvidence is a detached chronological copy of the ring. The
// slice is allocated only when a recorder asks for the incident tail.
type WatermarkStallEvidence struct {
	Capacity int
	Records  []WatermarkStallCycleEvidence
}

// WatermarkStaleTransition is the first exact runtime-observed crossing from
// ready to watermark_stale. Snapshot capture may contribute the observation,
// but it performs no persistence and does not wait for the scanner recorder.
type WatermarkStaleTransition struct {
	Previous Status
	Current  Status
}

type readinessObservationState struct {
	latest     Status
	hasLatest  bool
	transition *WatermarkStaleTransition
}

// recordReadinessObservation retains a crossing even when the caller that saw
// it was /readyz and the scanner's independent timer sampled on another phase.
// The CAS is fixed-cardinality and lock-free; it cannot affect the returned
// readiness value or invoke filesystem/terminal work.
func (r *Runtime) recordReadinessObservation(status Status) {
	if r == nil || status.SampledAt.IsZero() {
		return
	}
	status = cloneReadinessDiagnosticStatus(status)
	for {
		prior := r.readinessObservations.Load()
		if prior != nil && prior.transition != nil {
			return
		}
		next := &readinessObservationState{latest: status, hasLatest: true}
		if prior != nil && prior.hasLatest && readinessTransitionMatches(prior.latest, status) {
			transition := WatermarkStaleTransition{Previous: cloneReadinessDiagnosticStatus(prior.latest), Current: status}
			next.transition = &transition
		}
		if r.readinessObservations.CompareAndSwap(prior, next) {
			return
		}
	}
}

func readinessTransitionMatches(previous, current Status) bool {
	return previous.ProcessLive && previous.BackendReady && previous.Lifecycle == "live" &&
		current.ProcessLive && !current.BackendReady && current.Lifecycle == "live" && current.Reason == ReasonWatermarkStale
}

// ObserveWatermarkStaleTransition returns a detached copy of the retained
// first crossing. Observation does not consume it; cmd/scanner owns the
// process-local one-persistence latch.
func (r *Runtime) ObserveWatermarkStaleTransition() (WatermarkStaleTransition, bool) {
	if r == nil {
		return WatermarkStaleTransition{}, false
	}
	state := r.readinessObservations.Load()
	if state == nil || state.transition == nil {
		return WatermarkStaleTransition{}, false
	}
	return cloneWatermarkStaleTransition(*state.transition), true
}

func cloneWatermarkStaleTransition(value WatermarkStaleTransition) WatermarkStaleTransition {
	return WatermarkStaleTransition{Previous: cloneReadinessDiagnosticStatus(value.Previous), Current: cloneReadinessDiagnosticStatus(value.Current)}
}

func cloneReadinessDiagnosticStatus(value Status) Status {
	result := value
	result.Watermark = cloneTime(value.Watermark)
	result.CausalTarget = cloneTime(value.CausalTarget)
	result.IntegrityFailure = cloneIntegrityFailure(value.IntegrityFailure)
	return result
}

type watermarkStallRing struct {
	mu      sync.Mutex
	records [WatermarkStallRingCapacity]WatermarkStallCycleEvidence
	next    int
	count   int
}

func (r *watermarkStallRing) add(record WatermarkStallCycleEvidence) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.records[r.next] = record
	r.next = (r.next + 1) % len(r.records)
	if r.count < len(r.records) {
		r.count++
	}
	r.mu.Unlock()
}

func (r *watermarkStallRing) snapshot() WatermarkStallEvidence {
	if r == nil {
		return WatermarkStallEvidence{Capacity: WatermarkStallRingCapacity}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result := WatermarkStallEvidence{Capacity: len(r.records), Records: make([]WatermarkStallCycleEvidence, r.count)}
	start := (r.next - r.count + len(r.records)) % len(r.records)
	for index := range result.Records {
		result.Records[index] = r.records[(start+index)%len(r.records)]
	}
	return result
}

// ObserveWatermarkStallEvidence returns a detached, bounded diagnostic tail.
// It has no effect on readiness, publication, cancellation, or engine state.
func (r *Runtime) ObserveWatermarkStallEvidence() WatermarkStallEvidence {
	if r == nil {
		return WatermarkStallEvidence{Capacity: WatermarkStallRingCapacity}
	}
	return r.watermarkStallRing.snapshot()
}

func (r *Runtime) recordWatermarkStallCycle(startedAt, completedAt time.Time, cycleDuration time.Duration,
	fenceDisposition engine.DispositionCode, timerPolicy WatermarkStallTimerPolicy,
	before, after engine.EvaluationCycleState, fenceTiming engine.FenceTimingView,
	timerDisposition engine.TimerDisposition, timing engine.EvaluationTimingView, metrics Metrics) {
	if r == nil {
		return
	}
	if before.RunMode != engine.RunModeLive || before.Lifecycle != "live" || after.RunMode != engine.RunModeLive || after.Lifecycle != "live" {
		return
	}
	causalBefore, hasCausalBefore := watermarkDiagnosticTarget(r.binding, r.config, startedAt)
	causalAfter, hasCausalAfter := watermarkDiagnosticTarget(r.binding, r.config, completedAt)
	timingObserved := timing.EngineSequence == timerDisposition.EngineSequence
	if !timingObserved && fenceTiming.EngineSequence != 0 {
		timingObserved = timing.EngineSequence == fenceTiming.EngineSequence
	}
	evaluationSource := engine.AggregateEvaluationSource("")
	var evaluationTarget time.Time
	var evaluationStage, evaluationApply, evaluationPublication time.Duration
	if timingObserved {
		evaluationSource, evaluationTarget = timing.Source, timing.Target
		evaluationStage, evaluationApply, evaluationPublication = timing.Stage, timing.Apply, timing.Publication
	}
	record := WatermarkStallCycleEvidence{
		CycleStartedAt: startedAt.UTC(), CycleCompletedAt: completedAt.UTC(), CycleDuration: nonnegativeDuration(cycleDuration),
		DiagnosticSampledAt:      metrics.SampledAt,
		CoverageFenceDisposition: fenceDisposition, TimerPolicy: timerPolicy,
		CoverageFenceFinalization: nonnegativeDuration(fenceTiming.CoverageFinalization), CoverageFenceTotal: nonnegativeDuration(fenceTiming.Total),
		CoverageFenceValid: fenceTiming.Valid && fenceTiming.EngineSequence != 0,
		EvaluationObserved: timingObserved, EvaluationSource: evaluationSource, EvaluationTarget: evaluationTarget,
		EvaluationStage: nonnegativeDuration(evaluationStage), EvaluationApply: nonnegativeDuration(evaluationApply), EvaluationPublication: nonnegativeDuration(evaluationPublication),
		WatermarkBefore: before.Watermark, WatermarkAfter: after.Watermark, WatermarkBeforePresent: before.WatermarkPresent, WatermarkAfterPresent: after.WatermarkPresent,
		CausalTargetBefore: causalBefore, CausalTargetAfter: causalAfter, CausalTargetBeforePresent: hasCausalBefore, CausalTargetAfterPresent: hasCausalAfter,
		WatermarkLagBefore:  diagnosticWatermarkLag(before.Watermark, before.WatermarkPresent, causalBefore, hasCausalBefore),
		WatermarkLagAfter:   diagnosticWatermarkLag(after.Watermark, after.WatermarkPresent, causalAfter, hasCausalAfter),
		PublicationIDBefore: before.PublicationID, PublicationIDAfter: after.PublicationID,
		EngineSequenceBefore: before.LastEngineSequence, EngineSequenceAfter: after.LastEngineSequence,
		PublicationGeneratedBefore: before.GeneratedAt, PublicationGeneratedAfter: after.GeneratedAt,
		QueueCurrentFrames: metrics.QueueCurrentFrames, QueueHighFrames: metrics.QueueHighFrames,
		QueueCurrentBytes: metrics.QueueCurrentBytes, QueueHighBytes: metrics.QueueHighBytes,
		QueueOldestWaitingAge: nonnegativeDuration(metrics.LiveQueue.OldestWaitingFrameAge),
		ProcessingDelay:       nonnegativeDuration(metrics.MaxProcessingDelay), ProcessingDelayOneSecond: nonnegativeDuration(metrics.MaxProcessingDelayOneSecond),
		TQPressure: after.TQPressure, TQPressureCause: after.TQPressureCause,
		HeapAllocBytes: metrics.HeapAllocBytes, HeapInUseBytes: metrics.HeapInUseBytes, Goroutines: metrics.Goroutines,
		GCCycles: metrics.GCCycles, GCPauseTotalNS: metrics.GCPauseTotalNS, LastGCPauseNS: metrics.LastGCPauseNS,
	}
	r.watermarkStallRing.add(record)
}

func watermarkDiagnosticTarget(binding reference.Binding, config Config, now time.Time) (time.Time, bool) {
	if binding.Identity() == "" || now.IsZero() {
		return time.Time{}, false
	}
	return readinessCausalTarget(binding, config, now), true
}

func diagnosticWatermarkLag(watermark time.Time, watermarkPresent bool, target time.Time, targetPresent bool) time.Duration {
	if !watermarkPresent || !targetPresent || !target.After(watermark) {
		return 0
	}
	return target.Sub(watermark)
}

func nonnegativeDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}
