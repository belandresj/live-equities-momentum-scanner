package operations

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

// TestPC8DeliveryLatencyAttribution is the primary proof for the reopened
// C8-MEASURE-01 boundary. It uses only the closed family vocabulary and the
// same recorder used by production delivery observations.
func TestPC8DeliveryLatencyAttribution(t *testing.T) {
	run := &Runtime{deliveryOneSecondMaxFamily: DeliveryLatencyUnknown}
	observations := []struct {
		delay  time.Duration
		family DeliveryLatencyFamily
	}{
		{time.Millisecond, DeliveryLatencyAggregate},
		{2 * time.Millisecond, DeliveryLatencyTQ},
		{3 * time.Millisecond, DeliveryLatencyControl},
		{4 * time.Millisecond, DeliveryLatencyHydrationFence},
		{5 * time.Millisecond, DeliveryLatencyTimer},
		{6 * time.Millisecond, DeliveryLatencyUnknown},
	}
	for _, observation := range observations {
		run.recordDeliveryLatency(observation.delay, observation.family)
	}

	run.deliveryWindowMu.Lock()
	attribution := deliveryLatencyAttribution(run.deliveryFamilyCounts, time.Duration(run.deliveryOneSecondMaxNanos), run.deliveryOneSecondMaxFamily, run.deliveryWindowNonempty)
	deliveries := run.deliveryCount.Load()
	run.deliveryWindowMu.Unlock()
	if attribution.Total() != deliveries || deliveries != uint64(len(observations)) ||
		attribution.Aggregate != 1 || attribution.TQ != 1 || attribution.Control != 1 || attribution.HydrationFence != 1 ||
		attribution.Timer != 1 || attribution.Unknown != 1 {
		t.Fatalf("fixed attribution accounting = deliveries=%d attribution=%+v", deliveries, attribution)
	}
	if attribution.MaximumDuration != 6*time.Millisecond || attribution.MaximumFamily != DeliveryLatencyUnknown {
		t.Fatalf("maximum pair did not come from the winning delivery: %+v", attribution)
	}

	// Equal durations select the lowest closed-family priority independent of
	// arrival order. Every selected pair still corresponds to an observed
	// delivery with that exact duration.
	tied := &Runtime{deliveryOneSecondMaxFamily: DeliveryLatencyUnknown}
	for _, family := range []DeliveryLatencyFamily{DeliveryLatencyUnknown, DeliveryLatencyTimer, DeliveryLatencyHydrationFence, DeliveryLatencyControl, DeliveryLatencyTQ, DeliveryLatencyAggregate} {
		tied.recordDeliveryLatency(9*time.Millisecond, family)
	}
	if tied.deliveryOneSecondMaxNanos != uint64(9*time.Millisecond) || tied.deliveryOneSecondMaxFamily != DeliveryLatencyAggregate {
		t.Fatalf("deterministic tie = duration=%s family=%s", time.Duration(tied.deliveryOneSecondMaxNanos), tied.deliveryOneSecondMaxFamily)
	}

	classified := []struct {
		name   string
		result massive.EngineDeliveryResult
		want   DeliveryLatencyFamily
	}{
		{"aggregate", massive.EngineDeliveryResult{AggregateDisposition: engine.AggregateDisposition{Code: engine.DispositionAggregateInserted}}, DeliveryLatencyAggregate},
		{"tq", massive.EngineDeliveryResult{TQDisposition: engine.Disposition{Code: engine.DispositionTQApplied}}, DeliveryLatencyTQ},
		{"control", massive.EngineDeliveryResult{ControlDisposition: engine.ConnectionControlDisposition{Code: engine.DispositionConnectionControlApplied}}, DeliveryLatencyControl},
		{"hydration", massive.EngineDeliveryResult{HydrationDisposition: engine.HydrationDisposition{Code: engine.DispositionHydrationChunkApplied}}, DeliveryLatencyHydrationFence},
		{"live fence", massive.EngineDeliveryResult{LiveCoverageDisposition: engine.LiveCoverageFenceDisposition{Code: engine.DispositionLiveCoverageFenceApplied}}, DeliveryLatencyHydrationFence},
		{"absent", massive.EngineDeliveryResult{}, DeliveryLatencyUnknown},
		{"deferred without family", massive.EngineDeliveryResult{ConsumerDeferred: true}, DeliveryLatencyUnknown},
		{"mixed", massive.EngineDeliveryResult{AggregateDisposition: engine.AggregateDisposition{Code: engine.DispositionAggregateInserted}, TQDisposition: engine.Disposition{Code: engine.DispositionTQApplied}}, DeliveryLatencyUnknown},
	}
	for _, test := range classified {
		if got := deliveryLatencyFamily(test.result); got != test.want {
			t.Fatalf("%s family = %q, want %q", test.name, got, test.want)
		}
	}

	// Attribution affects only whether a below-threshold delivery value may
	// contribute to C9 recovery dwell. It does not admit engine work or mutate
	// ranking, watermark, readiness, or the degradation thresholds.
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	live, err := New(ctx, binding, DefaultConfig(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	beforeEngine, beforeStatus := live.Engine().ObserveSnapshot(), live.Status()
	beforePressure := defaultTQPressureSample(Metrics{Deliveries: 1, MaxProcessingDelayOneSecond: 999 * time.Millisecond, DeliveryLatencyAttribution: DeliveryLatencyAttribution{Aggregate: 1, MaximumDuration: 999 * time.Millisecond, MaximumFamily: DeliveryLatencyAggregate, WindowNonempty: true}})
	live.recordDeliveryLatency(3*time.Second, DeliveryLatencyUnknown)
	afterEngine, afterStatus := live.Engine().ObserveSnapshot(), live.Status()
	afterPressure := defaultTQPressureSample(Metrics{Deliveries: 1, MaxProcessingDelayOneSecond: 999 * time.Millisecond, DeliveryLatencyAttribution: DeliveryLatencyAttribution{Unknown: 1, MaximumDuration: 999 * time.Millisecond, MaximumFamily: DeliveryLatencyUnknown, WindowNonempty: true}})
	if !reflect.DeepEqual(beforeEngine, afterEngine) || !reflect.DeepEqual(beforeStatus, afterStatus) {
		t.Fatalf("diagnostic attribution mutated engine/status: before=%+v/%+v after=%+v/%+v", beforeEngine, beforeStatus, afterEngine, afterStatus)
	}
	if !reflect.DeepEqual(beforePressure, afterPressure) {
		t.Fatalf("diagnostic attribution leaked into pressure input: before=%+v after=%+v", beforePressure, afterPressure)
	}
	if err := live.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
