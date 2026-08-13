package operations_test

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

// TestDeliveryLatencyEmptyWindowMapsAfterPressureReset crosses the production
// recorder, pressure reset, sealed capture, and C10 mapper boundaries. It is
// the regression for the observed intermittent snapshot/readiness 503s.
func TestDeliveryLatencyEmptyWindowMapsAfterPressureReset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, _ := operations.NewDeliveryLatencyRegressionRuntime(t, ctx)
	t.Cleanup(func() {
		shutdown, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		if err := run.Shutdown(shutdown); err != nil {
			t.Fatal(err)
		}
	})

	operations.RecordDeliveryLatencyForTest(run, 7*time.Millisecond, operations.DeliveryLatencyAggregate)
	before := captureAndMapDeliveryLatency(t, run)
	if !before.WindowNonempty || before.MaximumDuration != 7*time.Millisecond || before.MaximumFamily != operations.DeliveryLatencyAggregate {
		t.Fatalf("known-family window = %+v", before)
	}
	beforeDeliveries := before.Total()

	operations.SampleAndResetDeliveryLatencyWindowForTest(t, ctx, run)
	empty := captureAndMapDeliveryLatency(t, run)
	if empty.WindowNonempty || empty.MaximumDuration != 0 || empty.MaximumFamily != operations.DeliveryLatencyUnknown {
		t.Fatalf("reset window = %+v", empty)
	}
	if empty.Total() != beforeDeliveries || empty.Aggregate != before.Aggregate || empty.Unknown != 0 {
		t.Fatalf("reset changed cumulative family accounting: before=%+v after=%+v", before, empty)
	}

	operations.RecordDeliveryLatencyForTest(run, 11*time.Millisecond, operations.DeliveryLatencyCheckpoint)
	after := captureAndMapDeliveryLatency(t, run)
	if !after.WindowNonempty || after.MaximumDuration != 11*time.Millisecond || after.MaximumFamily != operations.DeliveryLatencyCheckpoint {
		t.Fatalf("next delivery did not establish its atomic maximum pair: %+v", after)
	}
	if after.Total() != beforeDeliveries+1 || after.Aggregate != before.Aggregate || after.Checkpoint != before.Checkpoint+1 || after.Unknown != 0 {
		t.Fatalf("cumulative seven-family identity changed: before=%+v after=%+v", before, after)
	}
}

func captureAndMapDeliveryLatency(t *testing.T, run *operations.Runtime) operations.DeliveryLatencyAttribution {
	t.Helper()
	capture, err := run.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	view, ok := operations.InspectSnapshotCapture(capture)
	if !ok {
		t.Fatal("production capture was not sealed")
	}
	if view.Metrics.DeliveryLatencyAttribution.Total() != view.Metrics.Deliveries {
		t.Fatalf("cumulative delivery-family identity = deliveries=%d attribution=%+v", view.Metrics.Deliveries, view.Metrics.DeliveryLatencyAttribution)
	}
	if _, err := snapshotapi.Map(capture); err != nil {
		t.Fatalf("coherent delivery-latency capture failed mapping: %v", err)
	}
	return view.Metrics.DeliveryLatencyAttribution
}
