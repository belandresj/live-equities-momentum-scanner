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
	"github.com/coder/websocket"
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

func TestPHRRetryExhaustionStopsAutomaticDial(t *testing.T) {
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
	for run.Engine().ObserveOperational().LifecycleReason != "recovery_exhausted" {
		select {
		case err := <-joined:
			t.Fatalf("stable exhaustion ended the process: %v", err)
		case <-deadline:
			t.Fatalf("recovery did not reach stable exhaustion: adapter=%+v view=%+v", adapter.Accounting(), run.Engine().ObserveOperational())
		case <-time.After(5 * time.Millisecond):
		}
	}
	attempts := adapter.Accounting().ConnectionAttempts
	if attempts != 0 {
		t.Fatalf("exhausted recovery opened a socket: attempts=%d", attempts)
	}
	time.Sleep(150 * time.Millisecond)
	if later := adapter.Accounting().ConnectionAttempts; later != attempts {
		t.Fatalf("scheduler opened a socket after exhaustion: before=%d after=%d", attempts, later)
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

func TestPHRRetryFailedHandshakeFactsPrecedeTerminal(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	// Use a real moving process clock for the engine while keeping the provider
	// fixture deterministic and local.
	started := time.Now()
	clock := func() time.Time { return base.Add(time.Since(started)).UTC() }
	run, err := New(context.Background(), binding, DefaultConfig(), clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = run.Shutdown(deadline)
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, acceptErr := websocket.Accept(writer, request, nil)
		if acceptErr != nil {
			return
		}
		defer connection.CloseNow()
		ctx := request.Context()
		_ = connection.Write(ctx, websocket.MessageText, []byte(`[{"ev":"status","status":"connected"}]`))
		if _, _, readErr := connection.Read(ctx); readErr != nil {
			return
		}
		_ = connection.Write(ctx, websocket.MessageText, []byte(`[{"ev":"status","status":"auth_failed","message":"SECRET-PROVIDER-PROSE"}]`))
		_, _, _ = connection.Read(ctx)
	}))
	defer server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(server.URL, "http"), Credential: "SECRET-CREDENTIAL",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := run.openAttempt(context.Background(), context.Background(), LiveComponents{Adapter: adapter, Durations: capacityDurations()}, 100)
	if err == nil || attempt == nil {
		t.Fatalf("failed handshake attempt=%v err=%v", attempt, err)
	}
	beforeTerminal := run.Engine().ObserveOperational()
	if beforeTerminal.Connection.Applied < 3 || !beforeTerminal.Connection.Active || beforeTerminal.Connection.Acknowledged {
		t.Fatalf("classified handshake facts were not delivered first: %+v", beforeTerminal.Connection)
	}
	run.retireAttempt(attempt, 190, massive.CloseIntegrityLoss, clock())
	outcome := run.LatestRecoveryAttempt()
	if outcome == nil || outcome.Phase != "authentication" || outcome.Outcome != "authentication_failed" || outcome.AggregateAcknowledged {
		t.Fatalf("redacted latest attempt=%+v", outcome)
	}
	if rendered := fmt.Sprintf("%+v %v", outcome, err); strings.Contains(rendered, "SECRET-") {
		t.Fatalf("external prose or credential escaped: %s", rendered)
	}
}

func TestPHRRetryEstablishmentDeadlinePreservesHandshakePrefix(t *testing.T) {
	binding := operationsBinding(t)
	base, started := binding.SessionStart().Add(20*time.Minute), time.Now()
	clock := func() time.Time { return base.Add(time.Since(started)).UTC() }
	run, err := New(context.Background(), binding, DefaultConfig(), clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = run.Shutdown(deadline)
	})
	authReceived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, acceptErr := websocket.Accept(writer, request, nil)
		if acceptErr != nil {
			return
		}
		defer connection.CloseNow()
		ctx := request.Context()
		_ = connection.Write(ctx, websocket.MessageText, []byte(`[{"ev":"status","status":"connected"}]`))
		if _, _, readErr := connection.Read(ctx); readErr == nil {
			close(authReceived)
		}
		_, _, _ = connection.Read(ctx)
	}))
	defer server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(server.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	establishment, cancel := context.WithCancel(context.Background())
	type openResult struct {
		attempt *massive.LiveAttempt
		err     error
	}
	opened := make(chan openResult, 1)
	go func() {
		attempt, openErr := run.openAttempt(context.Background(), establishment, LiveComponents{Adapter: adapter, Durations: capacityDurations()}, 100)
		opened <- openResult{attempt: attempt, err: openErr}
	}()
	select {
	case <-authReceived:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("handshake did not classify connected before deadline")
	}
	result := <-opened
	attempt, err := result.attempt, result.err
	if err == nil || attempt == nil {
		t.Fatalf("deadline attempt=%v err=%v", attempt, err)
	}
	view := run.Engine().ObserveOperational()
	// connection_attempt + connected proves the classified prefix reached the
	// owner while the cancellation terminal is still pending retirement.
	if view.Connection.Applied != 2 || !view.Connection.Active {
		t.Fatalf("deadline discarded handshake prefix: %+v", view.Connection)
	}
	run.retireAttempt(attempt, 190, massive.CloseIntegrityLoss, clock())
	view = run.Engine().ObserveOperational()
	if view.Connection.Applied < 3 || view.Connection.Active {
		t.Fatalf("deadline terminal did not follow prefix: %+v", view.Connection)
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
