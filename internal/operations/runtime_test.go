package operations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// TestC8RUNTIME01LifecycleReadinessShutdown is P-C8-RUNTIME's compact S1
// trace. The engine owns every lifecycle/readiness input; operations only
// derives a defensive process view and bounds shutdown.
func TestC8RUNTIME01LifecycleReadinessShutdown(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	config := DefaultConfig()
	startup, cancelStartup := context.WithTimeout(context.Background(), time.Second)
	defer cancelStartup()
	runtime, err := New(startup, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if got := runtime.Status(); !got.ProcessLive || got.BackendReady || got.Reason != ReasonLifecycle {
		t.Fatalf("process live fabricated readiness: %+v", got)
	}
	owner := runtime.Engine()
	ackAt := now.Add(-config.EvaluationDelay)
	applyControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	completeHydration(t, owner, binding, engine.HydrationFreshBootstrap, 1, now)
	if got := runtime.Status(); !got.BackendReady || !got.RankingCurrent || got.Reason != ReasonNone || got.TQAvailable {
		t.Fatalf("current aggregate state not ready or TQ became a gate: %+v", got)
	}
	firstCapture, err := runtime.CaptureSnapshot()
	firstView, firstOK := InspectSnapshotCapture(firstCapture)
	if err != nil || !firstOK || !firstView.Status.BackendReady || firstView.Engine.Publication.PublicationID == 0 {
		t.Fatalf("first immutable response capture = %+v/%v", firstCapture, err)
	}
	firstView.Status.BackendReady = false
	if firstView.Engine.Publication.Watermark != nil {
		*firstView.Engine.Publication.Watermark = time.Time{}
	}
	sealedAgain, sealedOK := InspectSnapshotCapture(firstCapture)
	if !sealedOK || !sealedAgain.Status.BackendReady || sealedAgain.Engine.Publication.Watermark == nil || sealedAgain.Engine.Publication.Watermark.IsZero() {
		t.Fatal("inspection copy mutated the sealed capture")
	}
	firstView = sealedAgain
	now = now.Add(3 * time.Second)
	staleCapture, err := runtime.CaptureSnapshot()
	staleView, staleOK := InspectSnapshotCapture(staleCapture)
	if err != nil || !staleOK || staleView.SampleID <= firstView.SampleID || staleView.Engine.Publication.PublicationID != firstView.Engine.Publication.PublicationID ||
		staleView.Status.BackendReady || staleView.Status.Reason != ReasonWatermarkStale {
		t.Fatalf("unchanged-publication readiness expiry = first=%+v stale=%+v err=%v", firstView, staleView, err)
	}
	now = now.Add(-3 * time.Second)
	applyControl(t, owner, binding, engine.ConnectionLost, 1, 0, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now)
	if got := runtime.Status(); got.BackendReady || got.RankingCurrent || got.Reason != ReasonLifecycle || got.Lifecycle != "recovering" {
		t.Fatalf("disconnect remained ready: %+v", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if got := runtime.Status(); got.ProcessLive || got.BackendReady || got.Reason != ReasonRuntimeUnavailable {
		t.Fatalf("joined runtime reported live: %+v", got)
	}
}

func TestC8RUNTIME01CheckpointPrePlanCannotReportReady(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute).UTC()
	watermark := now.Add(-4 * time.Second).Truncate(time.Second)
	view := engine.OperationalView{
		BindingIdentity: binding.Identity(), RunMode: engine.RunModeLive,
		Lifecycle: "hydrating", CurrentMarketClaim: true, Watermark: &watermark,
		InstalledCheckpoint: true,
		Connection:          engine.OperationalConnection{Epoch: 1, AckFrame: 1, Active: true, Acknowledged: true},
		Hydration:           engine.OperationalHydration{Generation: 0, FenceReconciled: false},
	}
	status := deriveStatus(true, binding, DefaultConfig(), now, view)
	if status.BackendReady || status.Reason != ReasonFencePending || !status.RankingCurrent {
		t.Fatalf("current checkpoint became ready before catch-up plan/fence: %+v", status)
	}
}

func TestPC9PressureRuntimeCommandAndSampling(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(15 * time.Minute)
	clockNanos := &atomic.Int64{}
	clockNanos.Store(base.UnixNano())
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	ackAt := base.Add(-config.EvaluationDelay)
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, 1, base)
	readyBefore := run.Status()
	if !readyBefore.BackendReady || !readyBefore.RankingCurrent {
		t.Fatalf("pressure proof requires ready aggregate control: %+v", readyBefore)
	}

	sample := engine.TQPressureSample{QueueCurrentFrames: 50, QueueCapacityFrames: 100, TQLocalAccountingHealthy: true, Goroutines: 1}
	run.pressureSampler = func(Metrics) engine.TQPressureSample { return sample }
	cycle := func(at time.Time) {
		clockNanos.Store(at.UnixNano())
		admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
		if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
			t.Fatal("pressure timer")
		}
		run.syncTQPressure(context.Background())
	}
	run.deliveryWindowMu.Lock()
	run.deliveryOneSecondMaxNanos = uint64(125 * time.Millisecond)
	run.deliveryWindowVersion++
	run.deliveryWindowMu.Unlock()
	cycle(base)
	run.deliveryWindowMu.Lock()
	windowMax := run.deliveryOneSecondMaxNanos
	run.deliveryWindowMu.Unlock()
	if windowMax != 0 || run.Engine().ObserveTQ().Pressure != engine.TQPressureNormal {
		t.Fatalf("first runtime pressure sample = %+v max=%d", run.Engine().ObserveTQ(), windowMax)
	}
	cycle(base.Add(500 * time.Millisecond))
	if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureDegraded || !view.ShedTradesQuotes {
		t.Fatalf("runtime degradation = %+v", view)
	}
	sample.TQLocalAccountingHealthy = false
	cycle(base.Add(time.Second))
	if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureAggregateOnly || !view.AggregateOnly {
		t.Fatalf("runtime aggregate-only = %+v", view)
	}
	readyAfter := run.Status()
	if !readyAfter.BackendReady || !readyAfter.RankingCurrent || readyAfter.Watermark == nil || readyBefore.Watermark == nil || readyAfter.Watermark.Before(*readyBefore.Watermark) {
		t.Fatalf("T/Q pressure changed aggregate readiness: before=%+v after=%+v", readyBefore, readyAfter)
	}
}

func TestPC9PressureBroadAccountingRoutesGlobalIngressIntegrity(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(15 * time.Minute)
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, now)
	run.metricsSnapshot = func() Metrics {
		return Metrics{LiveQueue: massive.LiveQueueAccounting{FramesRead: 1, CapacityFrames: 100}, TQNormalization: massive.TQNormalizationAccounting{}}
	}
	admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
		t.Fatal("pressure tick")
	}
	run.syncTQPressure(context.Background())
	operational := run.Engine().ObserveOperational()
	if operational.Lifecycle != "suppressed" || operational.LifecycleReason != "ingress_integrity" || operational.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
		t.Fatalf("broad accounting did not route globally: %+v", operational)
	}
	if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureNormal || view.AggregateOnly {
		t.Fatalf("broad accounting was mislabeled T/Q-local: %+v", view)
	}
}

// TestC8RUNTIME02HealthyLifetimeOutlivesRecoveryDeadline proves the bounded
// establishment context is not the lifetime of a healthy live connection.
func TestC8RUNTIME02HealthyLifetimeOutlivesRecoveryDeadline(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	startedClock := time.Now()
	clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
	config := DefaultConfig()
	config.EvaluationDelay = 50 * time.Millisecond
	config.ReadinessTolerance = time.Second
	config.SampleCadence = 20 * time.Millisecond
	config.RecoveryAttemptDeadline = 2 * time.Second
	ctx := context.Background()
	run, err := New(ctx, binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	close(release)
	websocketServer := capacityWebSocketServer(t, release, nil)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
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
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(ctx, components) }()
	deadline := time.After(3 * time.Second)
	for !run.Status().BackendReady {
		select {
		case err := <-joined:
			t.Fatalf("live composition ended before ready: %v", err)
		case <-deadline:
			t.Fatalf("live composition did not become ready: %+v", run.Status())
		case <-time.After(10 * time.Millisecond):
		}
	}
	time.Sleep(config.RecoveryAttemptDeadline + 100*time.Millisecond)
	if got := run.Status(); !got.BackendReady || got.Lifecycle != "live" {
		t.Fatalf("healthy connection inherited establishment deadline: %+v", got)
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-joined:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("runtime-owned live cancellation=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Shutdown returned before the live composition joined")
	}
}

// TestC8RUNTIME03InitialRetryAndExhaustion proves establishment retries keep
// bootstrap semantics until a live baseline exists, and terminal exhaustion
// becomes an engine-owned suppression fact rather than a supervisor-only error.
func TestC8RUNTIME03InitialRetryAndExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name      string
		succeedOn int32
		wantReady bool
	}{
		{name: "initial handshake retry remains fresh bootstrap", succeedOn: 2, wantReady: true},
		{name: "bounded establishment exhaustion suppresses", succeedOn: 0, wantReady: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := operationsBinding(t)
			base := binding.SessionStart().Add(20 * time.Minute)
			startedClock := time.Now()
			clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
			config := DefaultConfig()
			config.RecoveryAttempts = 2
			config.EvaluationDelay = 20 * time.Millisecond
			config.SampleCadence = 10 * time.Millisecond
			config.RecoveryAttemptDeadline = 3 * time.Second
			config.ShutdownDeadline = 2 * time.Second
			run, err := New(context.Background(), binding, config, clock)
			if err != nil {
				t.Fatal(err)
			}
			var connections atomic.Int32
			websocketServer := retryWebSocketServer(t, &connections, tc.succeedOn)
			defer websocketServer.Close()
			adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
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
			components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
			joined := make(chan error, 1)
			go func() { joined <- run.RunLive(context.Background(), components) }()

			deadline := time.After(8 * time.Second)
			if tc.wantReady {
				for !run.Status().BackendReady {
					select {
					case err := <-joined:
						t.Fatalf("retry ended before ready: %v", err)
					case <-deadline:
						t.Fatalf("retry did not become ready: %+v", run.Status())
					case <-time.After(10 * time.Millisecond):
					}
				}
				view := run.Engine().ObserveOperational()
				if connections.Load() != 2 || view.Hydration.Purpose != engine.HydrationFreshBootstrap || view.Lifecycle != "live" {
					t.Fatalf("retry connections=%d view=%+v", connections.Load(), view)
				}
			} else {
				select {
				case err := <-joined:
					if err == nil || !strings.Contains(err.Error(), "recovery attempts exhausted") {
						t.Fatalf("exhaustion error=%v", err)
					}
				case <-deadline:
					t.Fatalf("exhaustion did not terminate: %+v", run.Status())
				}
				view := run.Engine().ObserveOperational()
				if connections.Load() != 2 || view.Lifecycle != "suppressed" || view.LifecycleReason != "recovery_exhausted" || view.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
					t.Fatalf("exhaustion connections=%d view=%+v", connections.Load(), view)
				}
			}
			shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
			defer cancelShutdown()
			if err := run.Shutdown(shutdown); err != nil {
				t.Fatal(err)
			}
			if tc.wantReady {
				select {
				case err := <-joined:
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("ready retry shutdown=%v", err)
					}
				default:
					t.Fatal("Shutdown did not join retry composition")
				}
			}
		})
	}
}

func TestC8RUNTIME04SuccessfulRecoveryResetsBudget(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	startedClock := time.Now()
	clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 2
	config.EvaluationDelay = 20 * time.Millisecond
	config.SampleCadence = 10 * time.Millisecond
	config.RecoveryAttemptDeadline = 3 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	var failNew atomic.Bool
	disconnect := make(chan struct{})
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, disconnect)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
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
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(context.Background(), components) }()
	waitForOperational(t, run, joined, 1, true)
	for recoveredEpoch := int32(2); recoveredEpoch <= 3; recoveredEpoch++ {
		disconnect <- struct{}{}
		waitForOperational(t, run, joined, recoveredEpoch-1, false)
		waitForOperational(t, run, joined, recoveredEpoch, true)
		view := run.Engine().ObserveOperational()
		if view.Connection.RecoveryAttempts != 0 || view.Lifecycle != "live" || view.Hydration.Purpose != engine.HydrationGapRecovery {
			t.Fatalf("epoch %d did not reset consecutive budget: %+v", recoveredEpoch, view)
		}
	}
	failNew.Store(true)
	disconnect <- struct{}{}
	select {
	case err := <-joined:
		if err == nil || !strings.Contains(err.Error(), "recovery attempts exhausted") {
			t.Fatalf("post-live recovery exhaustion=%v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("post-live recovery did not exhaust: %+v", run.Status())
	}
	view := run.Engine().ObserveOperational()
	if connections.Load() != 5 || view.Connection.RecoveryAttempts != 2 || view.Lifecycle != "suppressed" || view.LifecycleReason != "recovery_exhausted" {
		t.Fatalf("post-live exhaustion connections=%d view=%+v", connections.Load(), view)
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestC8RUNTIME05ShutdownTimeoutDoesNotClaimJoined(t *testing.T) {
	binding := operationsBinding(t)
	config := DefaultConfig()
	config.ShutdownDeadline = 20 * time.Millisecond
	now := binding.SessionStart().Add(10 * time.Second)
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	_, finish, err := run.beginLive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	err = run.Shutdown(deadline)
	cancel()
	if err == nil || !strings.Contains(err.Error(), "live composition shutdown deadline exceeded") || run.joined.Load() {
		t.Fatalf("shutdown timeout err=%v joined=%t", err, run.joined.Load())
	}
	finish()
	retry, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	if err := run.Shutdown(retry); err != nil || !run.joined.Load() {
		t.Fatalf("shutdown retry err=%v joined=%t", err, run.joined.Load())
	}
}

func TestPC9TAQOpaqueEngineCommandThroughC5Ack(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	eventAt := now.Add(60*time.Second + 100*time.Millisecond)
	config := DefaultConfig()
	config.EvaluationDelay, config.SampleCadence = 0, 10*time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	server := tqAckWebSocketServer(t, eventAt)
	defer server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(server.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	attempt, startDelivery, err := adapter.Start(context.Background(), massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	run.setLiveSources(attempt, adapter)
	if result, err := massive.DeliverToEngine(context.Background(), run.Engine(), startDelivery); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("start = %+v/%v", result, err)
	}
	handshake, err := attempt.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		result, err := massive.DeliverToEngine(context.Background(), run.Engine(), delivery)
		if err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake = %+v/%v", result, err)
		}
	}
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, attempt.Epoch(), now)
	base := now
	for index := 0; index < 60; index++ {
		window := base.Add(time.Duration(index) * time.Second)
		now = window.Add(time.Second)
		input := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: "AAA",
			WindowStart: window, WindowEnd: window.Add(time.Second), Values: engine.AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 10_000, VWAP: 12, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
			DeliveryTime: now, Live: engine.LivePosition{ConnectionEpoch: attempt.Epoch(), FrameSequence: 4, ArrayIndex: uint32(index + 1)}}
		admission, completion := run.Engine().AdmitAggregate(context.Background(), input)
		if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionAggregateInserted {
			t.Fatalf("aggregate %d admission=%s", index, admission)
		}
	}
	coverageCommand, err := run.Engine().IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	coverageInput, err := engine.NewLiveCoverageFenceInput(coverageCommand, engine.LiveCoverageFenceComplete, 4, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, coverageCompletion := run.Engine().AdmitLiveCoverageFence(context.Background(), coverageInput)
	if admission != engine.AdmissionAdmitted || (<-coverageCompletion).Code != engine.DispositionLiveCoverageFenceApplied {
		t.Fatal("live coverage advance")
	}
	admission, timer := run.Engine().AdmitTimer(context.Background())
	if admission != engine.AdmissionAdmitted || (<-timer).Code != engine.DispositionTimerApplied {
		t.Fatal("qualification timer")
	}
	command, err := run.Engine().IssueTQCommand()
	if err != nil {
		t.Fatalf("engine command: %v view=%+v evaluation=%+v", err, run.Engine().ObserveTQ(), run.Engine().ObserveReplayDeterministic().Evaluation)
	}
	adapterCommand, err := massive.ChangeTQCommandFromEngine(command)
	if err != nil || adapterCommand.Symbols[0] != "AAA" || adapterCommand.CommandToken != command.CommandToken() {
		t.Fatalf("C5 conversion = %+v/%v", adapterCommand, err)
	}
	write, err := attempt.ChangeTQ(context.Background(), adapterCommand)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := massive.DeliverToEngine(context.Background(), run.Engine(), write); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred {
		t.Fatalf("write = %+v/%v", result, err)
	}
	time.Sleep(10 * time.Millisecond)
	result, ok, err := attempt.DeliverNextToEngine(context.Background(), run.Engine())
	if err != nil || !ok || result.TQDisposition.Code != engine.DispositionTQApplied {
		t.Fatalf("ack = %+v ok=%v err=%v", result, ok, err)
	}
	view := run.Engine().ObserveTQ()
	if len(view.Rows) != 1 || !view.Rows[0].ProviderPresent || !view.Rows[0].TradeCoverage || !view.Rows[0].QuoteCoverage || view.CommandPending {
		t.Fatalf("coverage = %+v", view)
	}
	for index := 0; index < 4; index++ {
		result, ok, err := attempt.DeliverNextToEngine(context.Background(), run.Engine())
		if err != nil || !ok || result.TQDisposition.Code != engine.DispositionTQApplied {
			t.Fatalf("T/Q evidence %d = %+v/%v/%v", index, result, ok, err)
		}
	}
	now = now.Add(time.Second)
	coverageCommand, err = run.Engine().IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	coverageInput, err = engine.NewLiveCoverageFenceInput(coverageCommand, engine.LiveCoverageFenceComplete, 9, 3, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, coverageCompletion = run.Engine().AdmitLiveCoverageFence(context.Background(), coverageInput)
	if admission != engine.AdmissionAdmitted || (<-coverageCompletion).Code != engine.DispositionLiveCoverageFenceApplied {
		t.Fatal("T/Q coverage advance")
	}
	admission, timer = run.Engine().AdmitTimer(context.Background())
	if admission != engine.AdmissionAdmitted || (<-timer).Code != engine.DispositionTimerApplied {
		t.Fatal("T/Q feature timer")
	}
	view = run.Engine().ObserveTQ()
	if view.Rows[0].Tape.OneSecond != 1 || view.Rows[0].Tape.OneSecondStatus != engine.TQCurrent || view.Rows[0].Spread.Quality != "known_special" {
		t.Fatalf("C5 semantic evidence = %+v", view.Rows[0])
	}
	foreignBinding := binding.Identity()[:len(binding.Identity())-1] + "0"
	if foreignBinding == binding.Identity() {
		foreignBinding = binding.Identity()[:len(binding.Identity())-1] + "1"
	}
	foreignDrop := massive.AdapterDelivery{Kind: massive.DeliveryNormalizationDrop, Rejection: massive.LiveRejection{
		BindingIdentity: foreignBinding, TradingDate: binding.TradingDate(), Family: massive.LiveFamilyQuote, Symbol: "AAA",
		Reason: massive.LiveRejectQuotePrice, Position: engine.LivePosition{ConnectionEpoch: attempt.Epoch(), FrameSequence: 1_000},
	}}
	foreignResult, err := massive.DeliverToEngine(context.Background(), run.Engine(), foreignDrop)
	if err != nil || foreignResult.TQDisposition.Code != engine.DispositionTQFenced {
		t.Fatalf("foreign-binding drop = %+v/%v", foreignResult, err)
	}
	view = run.Engine().ObserveTQ()
	if !view.Rows[0].TradeCoverage || !view.Rows[0].QuoteCoverage || view.AggregateOnly {
		t.Fatalf("foreign-binding drop mutated coverage = %+v", view)
	}
	pressureSample := engine.TQPressureSample{QueueCurrentFrames: 50, QueueCapacityFrames: 100, TQLocalAccountingHealthy: true, Goroutines: 1}
	run.pressureSampler = func(Metrics) engine.TQPressureSample { return pressureSample }
	pressureTick := func(at time.Time) {
		now = at
		admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
		if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
			t.Fatal("integrated pressure tick")
		}
		run.syncTQPressure(context.Background())
		run.syncTQCommand(context.Background())
	}
	pressureStart := now
	pressureTick(pressureStart)
	pressureTick(pressureStart.Add(500 * time.Millisecond))
	view = run.Engine().ObserveTQ()
	if view.Pressure != engine.TQPressureDegraded || !view.ShedTradesQuotes || view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage {
		t.Fatalf("runtime degraded containment = %+v", view)
	}
	result, ok, err = attempt.DeliverNextToEngine(context.Background(), run.Engine())
	if err != nil || !ok || result.TQDisposition.Code != engine.DispositionTQRejected {
		t.Fatalf("early T/Q shed = %+v/%v/%v", result, ok, err)
	}
	result, ok, err = attempt.DeliverNextToEngine(context.Background(), run.Engine())
	if err != nil || !ok || result.AggregateDisposition.Code != engine.DispositionAggregateInserted {
		t.Fatalf("aggregate hidden by mixed-frame shed = %+v/%v/%v", result, ok, err)
	}
	pressureSample.TQLocalAccountingHealthy = false
	pressureTick(pressureStart.Add(time.Second))
	time.Sleep(10 * time.Millisecond)
	if pendingView, accounting := run.Engine().ObserveTQ(), adapter.Accounting(); !pendingView.CommandPending || pendingView.PendingAction != engine.TQUnsubscribe || accounting.CommandsPendingAck != 1 {
		t.Fatalf("unsubscribe was not pending at both boundaries: view=%+v adapter=%+v", pendingView, accounting)
	}
	result, ok, err = attempt.DeliverNextToEngine(context.Background(), run.Engine())
	if err != nil || !ok || result.TQDisposition.Code != engine.DispositionTQApplied {
		t.Fatalf("unsubscribe control after shedding = %+v/%v/%v", result, ok, err)
	}
	view = run.Engine().ObserveTQ()
	if view.Pressure != engine.TQPressureAggregateOnly || !view.AggregateOnly || view.Accounting.KnownPresent != 0 || view.Accounting.Unknown != 0 || view.CommandPending ||
		view.Commands.Issued != view.Commands.Pending+view.Commands.Acknowledged+view.Commands.Failed+view.Commands.Fenced {
		t.Fatalf("aggregate-only zero/accounting = %+v", view)
	}
	if metrics := run.Metrics(); !metrics.TQNormalization.Reconciles() || metrics.TQNormalization.Rejected == 0 || !metrics.Adapter.Reconciles() || !metrics.LiveQueue.Reconciles() {
		t.Fatalf("mixed-frame/runtime accounting = %+v", metrics)
	}
	if status := run.Status(); !status.BackendReady || !status.RankingCurrent {
		t.Fatalf("T/Q pressure changed aggregate readiness: %+v", status)
	}
	_ = attempt.Close(massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: command.CommandToken() + 100, Cause: massive.CloseControlledStop})
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func tqAckWebSocketServer(t *testing.T, eventAt time.Time) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		ctx := request.Context()
		write := func(value string) bool { return connection.Write(ctx, websocket.MessageText, []byte(value)) == nil }
		if !write(`[{"ev":"status","status":"connected"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"auth_success"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil {
			return
		}
		_ = write(`[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`)
		_ = write(fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"eligible","p":12,"s":10,"t":%d,"c":[14]}]`, eventAt.UnixMilli()))
		_ = write(fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"nonvolume","p":12,"s":10,"t":%d,"c":[15]}]`, eventAt.UnixMilli()))
		_ = write(fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"unreviewed","p":12,"s":10,"t":%d,"c":[57]}]`, eventAt.UnixMilli()))
		_ = write(fmt.Sprintf(`[{"ev":"Q","sym":"AAA","t":%d,"bp":12,"ap":12.02,"c":[1],"i":[2]}]`, eventAt.UnixMilli()))
		window := eventAt.Truncate(time.Second)
		_ = write(fmt.Sprintf(`[{"ev":"Q","t":%d,"bp":12,"ap":12.02},{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":12,"h":12,"l":12,"c":12,"v":100,"vw":12,"z":1}]`,
			eventAt.UnixMilli(), window.UnixMilli(), window.Add(time.Second).UnixMilli()))
		if _, _, err := connection.Read(ctx); err == nil {
			_ = write(`[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`)
		}
		<-ctx.Done()
	}))
}

func waitForOperational(t *testing.T, run *Runtime, joined <-chan error, minimumConnections int32, ready bool) {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for {
		status := run.Status()
		if status.BackendReady == ready {
			if run.Engine().ObserveOperational().Connection.Epoch >= uint64(minimumConnections) {
				return
			}
		}
		select {
		case err := <-joined:
			t.Fatalf("live composition ended while waiting for ready=%t: %v", ready, err)
		case <-deadline:
			t.Fatalf("timed out waiting for ready=%t connections>=%d: %+v", ready, minimumConnections, status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func retryWebSocketServer(t *testing.T, connections *atomic.Int32, succeedOn int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		attempt := connections.Add(1)
		if succeedOn == 0 || attempt < succeedOn {
			return
		}
		ctx := request.Context()
		write := func(value string) bool { return connection.Write(ctx, websocket.MessageText, []byte(value)) == nil }
		if !write(`[{"ev":"status","status":"connected"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"auth_success"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		<-ctx.Done()
	}))
}

func recoverableWebSocketServer(t *testing.T, connections *atomic.Int32, failNew *atomic.Bool, disconnect <-chan struct{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		connections.Add(1)
		if failNew.Load() {
			return
		}
		ctx := request.Context()
		write := func(value string) bool { return connection.Write(ctx, websocket.MessageText, []byte(value)) == nil }
		if !write(`[{"ev":"status","status":"connected"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"auth_success"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		select {
		case <-ctx.Done():
		case <-disconnect:
		}
	}))
}

func applyControl(t *testing.T, owner *engine.Engine, binding reference.Binding, kind engine.ConnectionControlKind, epoch, token uint64, position engine.LivePosition, at time.Time) {
	t.Helper()
	outcome := engine.ControlSucceeded
	if kind == engine.ConnectionLost {
		outcome = engine.ControlFailed
	}
	admission, completion := owner.AdmitConnectionControl(context.Background(), engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind, ConnectionEpoch: epoch, CommandToken: token, Position: position, ReceiptTime: at, Outcome: outcome})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("control %s admission=%s", kind, admission)
	}
	if got := <-completion; got.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("control %s disposition=%+v", kind, got)
	}
}

func completeHydration(t *testing.T, owner *engine.Engine, binding reference.Binding, purpose engine.HydrationPurpose, epoch uint64, at time.Time) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan")
	}
	planResult := <-completion
	var fence engine.HydrationFenceCommand
	for _, token := range planResult.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, terminalCompletion := owner.AdmitHydrationTerminal(context.Background(), terminal)
		terminalResult := <-terminalCompletion
		if terminalResult.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal=%+v", terminalResult)
		}
		if terminalResult.FenceCommand.CommandToken() != 0 {
			fence = terminalResult.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatalf("hydration produced no fence: plan=%+v", planResult)
	}
	through := max(uint64(1), owner.ObserveOperational().Connection.AckFrame)
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, through, 1, at)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := owner.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if got := <-fenceCompletion; got.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence=%+v", got)
	}
}

func operationsBinding(t *testing.T) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v3/reference/tickers":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "count": 1, "results": []map[string]any{{"ticker": "AAA", "active": true, "market": "stocks", "locale": "us", "type": "CS"}}})
		case strings.HasPrefix(r.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": 1, "results": []map[string]any{{"T": "AAA", "c": 10.0, "t": facts.PriorRegularClose.UnixMilli()}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}
