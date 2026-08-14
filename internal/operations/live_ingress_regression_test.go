package operations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

// Regression for the 2026-08-11 frame-capacity terminal. C5 maps that terminal
// to ingress_integrity; once the engine has contained it, RunLive must not keep
// creating epochs that the suppressed lifecycle can only reject. The direct
// ingress fact makes this test deterministic and independent of socket timing.
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
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("ingress failure was not admitted")
	}
	if result := <-completion; result.Code != engine.DispositionIngressIntegrity {
		t.Fatalf("ingress failure was not contained: %+v", result)
	}

	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws://127.0.0.1:1", Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 1024, TotalFrameBytes: 1024},
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker("https://127.0.0.1", func() (string, error) { return "fixture", nil }, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
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
		t.Fatalf("suppression drifted after containment: %+v", view)
	}
}

func TestPTQRScheduledRecoveryIsTheOnlySameBindingContinuation(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	started := time.Now()
	clock := func() time.Time { return base.Add(time.Since(started)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 1
	config.RecoveryBackoffInitial = 100 * time.Millisecond
	config.RecoveryBackoffMax = 100 * time.Millisecond
	config.SampleCadence = 10 * time.Minute
	config.ConnectionAttemptDeadline = 20 * time.Millisecond
	config.ShutdownDeadline = time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, clock())
	admission, completion := run.Engine().AdmitOperationalIngressIntegrity(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionIngressIntegrity {
		t.Fatal("ingress suppression was not installed")
	}
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws://127.0.0.1:1", Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 1024, TotalFrameBytes: 8192}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker("https://127.0.0.1", func() (string, error) { return "fixture", nil }, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	ctx, cancel := context.WithCancel(context.Background())
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(ctx, components) }()

	time.Sleep(50 * time.Millisecond)
	if attempts := adapter.Accounting().ConnectionAttempts; attempts != 0 {
		t.Fatalf("socket opened before engine deadline: attempts=%d", attempts)
	}
	deadline := time.After(500 * time.Millisecond)
	for adapter.Accounting().ConnectionAttempts < 1 || run.Engine().ObserveOperational().Lifecycle != "suppressed" {
		select {
		case err := <-joined:
			t.Fatalf("recoverable scheduler ended the process: %v", err)
		case <-deadline:
			t.Fatalf("scheduled recovery did not make one bounded attempt: adapter=%+v view=%+v", adapter.Accounting(), run.Engine().ObserveOperational())
		case <-time.After(5 * time.Millisecond):
		}
	}
	attempts := adapter.Accounting().ConnectionAttempts
	if attempts == 0 || attempts > 2 {
		t.Fatalf("scheduler did not remain bounded at the first deadline: attempts=%d", attempts)
	}
	time.Sleep(50 * time.Millisecond)
	if later := adapter.Accounting().ConnectionAttempts; later != attempts {
		t.Fatalf("scheduler opened another socket before the next deadline: before=%d after=%d", attempts, later)
	}
	status := run.Status()
	if !status.ProcessLive || status.BackendReady || status.Reason != ReasonSuppressed {
		t.Fatalf("recoverable suppression process status=%+v", status)
	}
	cancel()
	select {
	case err := <-joined:
		if err == nil {
			t.Fatal("canceled live composition returned nil")
		}
	case <-time.After(time.Second):
		t.Fatal("scheduled recovery did not stop on cancellation")
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), time.Second)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestPTQRQuietLiveSocketCannotMaskSessionEnd(t *testing.T) {
	binding := operationsBinding(t)
	var clockNanos atomic.Int64
	clockNanos.Store(binding.SessionStart().Add(20 * time.Minute).UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	config := DefaultConfig()
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = time.Second
	config.ShutdownDeadline = time.Second
	run, err := New(context.Background(), binding, config, clock)
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

	var connections atomic.Int32
	var failNew atomic.Bool
	quiet := make(chan struct{})
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, quiet)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`)
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(context.Background(), components) }()
	waitForOperational(t, run, joined, 1, true)

	clockNanos.Store(binding.SessionEnd().UnixNano())
	select {
	case err := <-joined:
		if err != nil {
			t.Fatalf("session end returned an error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("quiet socket masked session end: view=%+v adapter=%+v", run.Engine().ObserveOperational(), adapter.Accounting())
	}
	view := run.Engine().ObserveOperational()
	accounting := adapter.Accounting()
	if view.Lifecycle != "ended" || accounting.AttemptsActive != 0 || !accounting.Reconciles() {
		t.Fatalf("session-end cleanup did not reconcile: view=%+v adapter=%+v", view, accounting)
	}
}

func TestPTQRQuietLiveSocketCannotMaskRestartRequiredSuppression(t *testing.T) {
	binding := operationsBinding(t)
	initial := binding.SessionStart().Add(20 * time.Minute)
	var clockNanos atomic.Int64
	clockNanos.Store(initial.UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	config := DefaultConfig()
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = time.Second
	config.ShutdownDeadline = time.Second
	run, err := New(context.Background(), binding, config, clock)
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

	var connections atomic.Int32
	var failNew atomic.Bool
	quiet := make(chan struct{})
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, quiet)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`)
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(context.Background(), components) }()
	waitForOperational(t, run, joined, 1, true)

	clockNanos.Store(initial.Add(-time.Second).UnixNano())
	select {
	case err := <-joined:
		if err == nil || !strings.Contains(err.Error(), "aggregate runtime requires restart") {
			t.Fatalf("restart-required suppression returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("quiet socket masked restart-required suppression: view=%+v adapter=%+v", run.Engine().ObserveOperational(), adapter.Accounting())
	}
	view := run.Engine().ObserveOperational()
	accounting := adapter.Accounting()
	if view.Lifecycle != "suppressed" || view.Suppression != engine.SuppressionRestartRequired || accounting.AttemptsActive != 0 || !accounting.Reconciles() {
		t.Fatalf("restart-required cleanup did not reconcile: view=%+v adapter=%+v", view, accounting)
	}
}

func TestPTQRQuietLiveSocketHonorsParentDeadline(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	config := DefaultConfig()
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = time.Second
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

	var connections atomic.Int32
	var failNew atomic.Bool
	quiet := make(chan struct{})
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, quiet)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`)
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	lifetime, cancelLifetime := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancelLifetime()
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(lifetime, components) }()
	waitForOperational(t, run, joined, 1, true)

	select {
	case err := <-joined:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("parent deadline returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("quiet socket masked parent deadline: view=%+v adapter=%+v", run.Engine().ObserveOperational(), adapter.Accounting())
	}
	accounting := adapter.Accounting()
	if accounting.AttemptsActive != 0 || !accounting.Reconciles() {
		t.Fatalf("parent-deadline cleanup did not reconcile: %+v", accounting)
	}
}
