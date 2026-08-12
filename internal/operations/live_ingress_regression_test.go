package operations

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

func TestSuppressedIngressCannotEnterReconnectHotLoop(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	config := DefaultConfig()
	config.RecoveryAttempts = 3
	config.SampleCadence = 10 * time.Minute
	config.ConnectionAttemptDeadline = 50 * time.Millisecond
	config.ShutdownDeadline = time.Second
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(shutdown); err != nil {
			t.Fatal(err)
		}
	})
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, now)
	admission, completion := run.Engine().AdmitOperationalIngressIntegrity(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionIngressIntegrity {
		t.Fatal("ingress failure was not contained")
	}

	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws://127.0.0.1:1", Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 1024, TotalFrameBytes: 1024}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker("https://127.0.0.1", func() (string, error) { return "fixture", nil }, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	err = run.RunLive(ctx, components)
	cancel()
	if err == nil {
		t.Fatal("terminal live state returned nil")
	}
	accounting := adapter.Accounting()
	if accounting.ConnectionAttempts != 0 || accounting.CommandsStarted != 0 {
		t.Fatalf("suppressed engine recreated attempts: %+v", accounting)
	}
	view := run.Engine().ObserveOperational()
	if view.LifecycleReason != "ingress_integrity" || view.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
		t.Fatalf("suppression drifted: %+v", view)
	}
}
