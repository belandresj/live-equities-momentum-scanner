package operations

import (
	"errors"
	"math"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// SnapshotCapture is a sealed Component 10 response input. Its only field is
// package-private, so another package can construct only the invalid zero value
// and cannot replace one member of a valid capture.
type SnapshotCapture struct{ sealed *sealedSnapshotCapture }

// SnapshotCaptureView is a detached inspection copy. It cannot be converted
// back into a SnapshotCapture.
type SnapshotCaptureView struct {
	SampleID        uint64
	SampledAt       time.Time
	ProcessLive     bool
	Engine          engine.SnapshotView
	Status          Status
	Metrics         Metrics
	IngressIncident *IngressIncident
	RecoveryAttempt *RecoveryAttemptOutcome
}

type sealedSnapshotCapture struct{ view SnapshotCaptureView }

func InspectSnapshotCapture(capture SnapshotCapture) (SnapshotCaptureView, bool) {
	if capture.sealed == nil {
		return SnapshotCaptureView{}, false
	}
	return cloneSnapshotCaptureView(capture.sealed.view), true
}

func (r *Runtime) CaptureSnapshot() (SnapshotCapture, error) {
	if r == nil || r.engine == nil || r.clock == nil {
		return SnapshotCapture{}, errors.New("runtime snapshot unavailable")
	}
	// This short operations-local critical section is the readiness/TQ-visibility
	// observation linearization. It includes no network, diagnostics, response
	// mapping, or engine mutation, so a slow/canceled HTTP client cannot hold it.
	// Serializing the atomic engine read, readiness derivation, sequence, and
	// latch transition prevents an older ready observation from overtaking a
	// later stale observation and shortening the five-second recovery hold.
	r.captureObservationMu.Lock()
	sampledAt := r.clock().UTC()
	processLive := r.processLive.Load() && !r.joined.Load()
	view := r.engine.ObserveSnapshot()
	status := deriveStatus(processLive, r.binding, r.config, sampledAt, view.Operational)
	sequence, err := r.nextCaptureSequence()
	if err != nil {
		r.captureObservationMu.Unlock()
		return SnapshotCapture{}, err
	}
	status = r.applyTQWatermarkVisibility(sequence, status)
	// Retain the readiness observation immediately after the one atomic engine
	// read, before even bounded diagnostics composition can allow a later
	// scanner sample to overtake this chronology.
	r.recordReadinessObservation(status)
	r.captureObservationMu.Unlock()
	metrics := r.metricsFromDiagnostics(sampledAt, processLive, view)
	return SnapshotCapture{sealed: &sealedSnapshotCapture{view: SnapshotCaptureView{
		SampleID: sequence, SampledAt: sampledAt, ProcessLive: processLive,
		Engine: view, Status: status, Metrics: metrics,
		IngressIncident: r.ingressIncident.get(),
		RecoveryAttempt: r.recoveryAttempt.get(),
	}}}, nil
}

const tqWatermarkRecoveryWindow = 5 * time.Second

type tqWatermarkVisibilityState struct {
	sequence         uint64
	hold             bool
	recoveryStarted  time.Time
	recoveryBoundary *time.Time
}

func (r *Runtime) applyTQWatermarkVisibility(sequence uint64, status Status) Status {
	if r == nil || sequence == 0 || status.SampledAt.IsZero() {
		return status
	}
	for {
		prior := r.tqWatermarkVisibility.Load()
		if prior != nil && sequence <= prior.sequence {
			status.TQWatermarkVisibilityHold = status.Reason == ReasonWatermarkStale || prior.hold
			status.TQWatermarkRecoveryBoundary = cloneTime(prior.recoveryBoundary)
			return status
		}
		next := tqWatermarkVisibilityState{sequence: sequence}
		if prior != nil {
			next.hold = prior.hold
			next.recoveryStarted = prior.recoveryStarted
			next.recoveryBoundary = cloneTime(prior.recoveryBoundary)
		}
		switch {
		case status.Reason == ReasonWatermarkStale:
			next.hold = true
			next.recoveryStarted = time.Time{}
			next.recoveryBoundary = nil
		case next.hold && status.BackendReady && status.Watermark != nil:
			if next.recoveryStarted.IsZero() {
				next.recoveryStarted = status.SampledAt
				next.recoveryBoundary = cloneTime(status.Watermark)
			} else if next.recoveryBoundary != nil && status.SampledAt.Sub(next.recoveryStarted) >= tqWatermarkRecoveryWindow &&
				!status.Watermark.Before(next.recoveryBoundary.Add(tqWatermarkRecoveryWindow)) {
				next.hold = false
			}
		case next.hold:
			next.recoveryStarted = time.Time{}
			next.recoveryBoundary = nil
		}
		if r.tqWatermarkVisibility.CompareAndSwap(prior, &next) {
			status.TQWatermarkVisibilityHold = next.hold
			status.TQWatermarkRecoveryBoundary = cloneTime(next.recoveryBoundary)
			return status
		}
	}
}

func (r *Runtime) nextCaptureSequence() (uint64, error) {
	for {
		current := r.captureSequence.Load()
		if current == math.MaxUint64 {
			return 0, errors.New("runtime snapshot sequence exhausted")
		}
		if r.captureSequence.CompareAndSwap(current, current+1) {
			return current + 1, nil
		}
	}
}

func cloneSnapshotCaptureView(value SnapshotCaptureView) SnapshotCaptureView {
	result := value
	result.Engine.Publication.AggregateEvaluation = cloneEvaluation(value.Engine.Publication.AggregateEvaluation)
	result.Engine.Publication.Watermark = cloneTime(value.Engine.Publication.Watermark)
	result.Engine.Operational.Watermark = cloneTime(value.Engine.Operational.Watermark)
	result.Engine.Operational.Hydration.SupportedThrough = cloneTime(value.Engine.Operational.Hydration.SupportedThrough)
	result.Engine.Operational.IntegrityFailure = cloneIntegrityFailure(value.Engine.Operational.IntegrityFailure)
	result.Engine.TQ.Desired = append([]string(nil), value.Engine.TQ.Desired...)
	result.Engine.TQ.Rows = append([]engine.TQSymbolView(nil), value.Engine.TQ.Rows...)
	result.Status.Watermark = cloneTime(value.Status.Watermark)
	result.Status.CausalTarget = cloneTime(value.Status.CausalTarget)
	result.Status.TQWatermarkRecoveryBoundary = cloneTime(value.Status.TQWatermarkRecoveryBoundary)
	result.Status.IntegrityFailure = cloneIntegrityFailure(value.Status.IntegrityFailure)
	result.Metrics.Engine.Watermark = cloneTime(value.Metrics.Engine.Watermark)
	result.Metrics.Engine.Hydration.SupportedThrough = cloneTime(value.Metrics.Engine.Hydration.SupportedThrough)
	result.Metrics.Engine.IntegrityFailure = cloneIntegrityFailure(value.Metrics.Engine.IntegrityFailure)
	if value.IngressIncident != nil {
		incident := *value.IngressIncident
		incident.Engine.Watermark = cloneTime(value.IngressIncident.Engine.Watermark)
		incident.Engine.Hydration.SupportedThrough = cloneTime(value.IngressIncident.Engine.Hydration.SupportedThrough)
		incident.Engine.IntegrityFailure = cloneIntegrityFailure(value.IngressIncident.Engine.IntegrityFailure)
		incident.PriorEngine.Watermark = cloneTime(value.IngressIncident.PriorEngine.Watermark)
		incident.PriorEngine.Hydration.SupportedThrough = cloneTime(value.IngressIncident.PriorEngine.Hydration.SupportedThrough)
		incident.PriorEngine.IntegrityFailure = cloneIntegrityFailure(value.IngressIncident.PriorEngine.IntegrityFailure)
		incident.LastCoherentProjection = cloneLastCoherentProjection(value.IngressIncident.LastCoherentProjection)
		result.IngressIncident = &incident
	}
	if value.RecoveryAttempt != nil {
		copyValue := *value.RecoveryAttempt
		result.RecoveryAttempt = &copyValue
	}
	return result
}

func cloneIntegrityFailure(value *engine.EvaluatorIntegrityView) *engine.EvaluatorIntegrityView {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneEvaluation(value engine.EvaluationView) engine.EvaluationView {
	result := value
	result.Rows = append([]engine.RankingRowView(nil), value.Rows...)
	for i := range result.Rows {
		if result.Rows[i].Float.Percent != nil {
			percent := *result.Rows[i].Float.Percent
			result.Rows[i].Float.Percent = &percent
		}
	}
	return result
}
