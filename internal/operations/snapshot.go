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
	SampleID    uint64
	SampledAt   time.Time
	ProcessLive bool
	Engine      engine.SnapshotView
	Status      Status
	Metrics     Metrics
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
	r.captureMu.Lock()
	defer r.captureMu.Unlock()
	if r.captureSequence == math.MaxUint64 {
		return SnapshotCapture{}, errors.New("runtime snapshot sequence exhausted")
	}
	r.captureSequence++
	sampledAt := r.clock().UTC()
	processLive := r.processLive.Load() && !r.joined.Load()
	view := r.engine.ObserveSnapshot()
	metrics := r.metricsFromPublication(sampledAt, processLive, view.Operational)
	return SnapshotCapture{sealed: &sealedSnapshotCapture{view: SnapshotCaptureView{
		SampleID: r.captureSequence, SampledAt: sampledAt, ProcessLive: processLive,
		Engine: view, Status: deriveStatus(processLive, r.binding, r.config, sampledAt, view.Operational), Metrics: metrics,
	}}}, nil
}

func cloneSnapshotCaptureView(value SnapshotCaptureView) SnapshotCaptureView {
	result := value
	result.Engine.Publication.AggregateEvaluation = cloneReplayEvaluation(value.Engine.Publication.AggregateEvaluation)
	result.Engine.Publication.Watermark = cloneTime(value.Engine.Publication.Watermark)
	result.Engine.Operational.Watermark = cloneTime(value.Engine.Operational.Watermark)
	result.Engine.Operational.Hydration.SupportedThrough = cloneTime(value.Engine.Operational.Hydration.SupportedThrough)
	result.Engine.TQ.Desired = append([]string(nil), value.Engine.TQ.Desired...)
	result.Engine.TQ.Rows = append([]engine.TQSymbolView(nil), value.Engine.TQ.Rows...)
	result.Status.Watermark = cloneTime(value.Status.Watermark)
	result.Status.CausalTarget = cloneTime(value.Status.CausalTarget)
	result.Metrics.Engine.Watermark = cloneTime(value.Metrics.Engine.Watermark)
	result.Metrics.Engine.Hydration.SupportedThrough = cloneTime(value.Metrics.Engine.Hydration.SupportedThrough)
	return result
}

func cloneReplayEvaluation(value engine.ReplayEvaluationView) engine.ReplayEvaluationView {
	result := value
	result.Rows = append([]engine.ReplayRankingRowView(nil), value.Rows...)
	return result
}
