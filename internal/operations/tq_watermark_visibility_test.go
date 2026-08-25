package operations

import (
	"testing"
	"time"
)

func TestPLBRR1CaptureVisibilityIsAtomicWithReadinessAndFiveSecondRecovery(t *testing.T) {
	runtime := &Runtime{}
	base := time.Date(2026, 8, 25, 19, 0, 0, 0, time.UTC)
	watermark := base
	ready := Status{ProcessLive: true, BackendReady: true, Lifecycle: "live", SampledAt: base, Watermark: &watermark}
	if got := runtime.applyTQWatermarkVisibility(1, ready); got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary != nil {
		t.Fatalf("initial ready visibility = %+v", got)
	}

	stale := ready
	stale.BackendReady, stale.Reason, stale.SampledAt = false, ReasonWatermarkStale, base.Add(time.Second)
	if got := runtime.applyTQWatermarkVisibility(2, stale); !got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary != nil {
		t.Fatalf("stale capture was not masked atomically: %+v", got)
	}

	recoveryBoundary := base.Add(time.Second)
	recovering := ready
	recovering.SampledAt, recovering.Watermark = base.Add(2*time.Second), &recoveryBoundary
	got := runtime.applyTQWatermarkVisibility(3, recovering)
	if !got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary == nil || !got.TQWatermarkRecoveryBoundary.Equal(recoveryBoundary) {
		t.Fatalf("recovery boundary = %+v", got)
	}

	tooEarlyWatermark := recoveryBoundary.Add(4 * time.Second)
	recovering.SampledAt, recovering.Watermark = base.Add(7*time.Second), &tooEarlyWatermark
	if got = runtime.applyTQWatermarkVisibility(4, recovering); !got.TQWatermarkVisibilityHold {
		t.Fatalf("wall time released without five watermark seconds: %+v", got)
	}

	releasedWatermark := recoveryBoundary.Add(5 * time.Second)
	recovering.SampledAt, recovering.Watermark = base.Add(8*time.Second), &releasedWatermark
	if got = runtime.applyTQWatermarkVisibility(5, recovering); got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary == nil {
		t.Fatalf("five-second recovery did not release: %+v", got)
	}

	stale.SampledAt = base.Add(9 * time.Second)
	if got = runtime.applyTQWatermarkVisibility(6, stale); !got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary != nil {
		t.Fatalf("renewed stale crossing did not restart hold: %+v", got)
	}
}

func TestPLBRR1LaterStaleObservationSurvivesAnOvertakenReadyCapture(t *testing.T) {
	runtime := &Runtime{}
	base := time.Date(2026, 8, 25, 19, 0, 0, 0, time.UTC)
	watermark := base
	ready := Status{ProcessLive: true, BackendReady: true, Lifecycle: "live", SampledAt: base, Watermark: &watermark}
	if got := runtime.applyTQWatermarkVisibility(1, ready); got.TQWatermarkVisibilityHold {
		t.Fatalf("initial ready = %+v", got)
	}
	stale := ready
	stale.BackendReady, stale.Reason, stale.SampledAt = false, ReasonWatermarkStale, base.Add(2*time.Second)
	if got := runtime.applyTQWatermarkVisibility(3, stale); !got.TQWatermarkVisibilityHold {
		t.Fatalf("later stale observation = %+v", got)
	}
	overtakenReady := ready
	overtakenReady.SampledAt = base.Add(time.Second)
	if got := runtime.applyTQWatermarkVisibility(2, overtakenReady); !got.TQWatermarkVisibilityHold {
		t.Fatalf("overtaken earlier ready capture erased later stale state: %+v", got)
	}
	nextReady := ready
	nextReady.SampledAt = base.Add(3 * time.Second)
	if got := runtime.applyTQWatermarkVisibility(4, nextReady); !got.TQWatermarkVisibilityHold || got.TQWatermarkRecoveryBoundary == nil {
		t.Fatalf("next observation did not begin guarded recovery: %+v", got)
	}
}
