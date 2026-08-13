package operations

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// NewDeliveryLatencyRegressionRuntime exposes only the production Runtime
// boundary needed by the external C8/C10 regression. Keeping this bridge in a
// test file avoids adding a mutable delivery-metrics API to the scanner.
func NewDeliveryLatencyRegressionRuntime(t *testing.T, ctx context.Context) (*Runtime, time.Time) {
	t.Helper()
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(15 * time.Minute)
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ackAt := now.Add(-config.EvaluationDelay)
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, 1, now)
	return run, now
}

// RecordDeliveryLatencyForTest invokes the production recorder without
// exporting that mutable operation from a non-test build.
func RecordDeliveryLatencyForTest(run *Runtime, delay time.Duration, family DeliveryLatencyFamily) {
	run.recordDeliveryLatency(delay, family)
}

// SampleAndResetDeliveryLatencyWindowForTest runs the production pressure
// command/result path. The fixed sample supplies only the queue capacity that
// a live adapter normally contributes; syncTQPressure performs the reset.
func SampleAndResetDeliveryLatencyWindowForTest(t *testing.T, ctx context.Context, run *Runtime) {
	t.Helper()
	admission, completion := run.Engine().AdmitTQPressureTick(ctx)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
		t.Fatal("delivery-latency pressure tick was not applied")
	}
	priorSampler := run.pressureSampler
	run.pressureSampler = func(metrics Metrics) engine.TQPressureSample {
		attribution := metrics.DeliveryLatencyAttribution
		return engine.TQPressureSample{
			FrameCapacity:             100,
			ByteCapacity:              100,
			MaxDeliveryDelayOneSec:    metrics.MaxProcessingDelayOneSecond,
			DeliveryLatencyAttributed: attribution.MaximumFamily != DeliveryLatencyUnknown && attribution.Reconciles(metrics.Deliveries),
			TQLocalAccountingHealthy:  true,
			Goroutines:                1,
		}
	}
	run.syncTQPressure(ctx)
	run.pressureSampler = priorSampler
}
