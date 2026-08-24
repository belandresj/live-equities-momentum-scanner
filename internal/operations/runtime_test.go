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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
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
	transition, transitionOK := runtime.ObserveWatermarkStaleTransition()
	if !transitionOK || !transition.Previous.BackendReady || transition.Current.BackendReady ||
		transition.Current.Reason != ReasonWatermarkStale || !transition.Current.SampledAt.Equal(staleView.SampledAt) {
		t.Fatalf("snapshot capture did not retain readiness crossing: ok=%t transition=%+v", transitionOK, transition)
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

func TestSlice2DegradedCurrentIsReadyWithoutTQPromotion(t *testing.T) {
	binding := operationsBinding(t)
	config := DefaultConfig()
	now := binding.SessionStart().Add(2 * time.Minute)
	target := now.Add(-config.EvaluationDelay).Truncate(time.Second)
	view := engine.OperationalView{
		PublicationID: 9, BindingIdentity: binding.Identity(), RunMode: engine.RunModeLive,
		Lifecycle: "live", RankingMode: "degraded_current", CurrentMarketClaim: true, Watermark: &target,
		Connection: engine.OperationalConnection{Epoch: 1, Active: true, Acknowledged: true},
		Hydration:  engine.OperationalHydration{FenceReconciled: true},
	}
	status := deriveStatus(true, binding, config, now, view)
	if !status.BackendReady || !status.RankingCurrent || status.Reason != ReasonNone || status.RankingMode != "degraded_current" || status.TQAvailable {
		t.Fatalf("degraded-current readiness = %+v", status)
	}

	view.CurrentMarketClaim = false
	status = deriveStatus(true, binding, config, now, view)
	if status.BackendReady || status.RankingCurrent || status.Reason != ReasonRankingNoncurrent {
		t.Fatalf("mode string bypassed owner currentness = %+v", status)
	}
}

func TestCKHOTUsableCompletionReturnsToEngine(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(30 * time.Second)
	store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: binding.Identity(), ArtifactByteLimit: 8 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := checkpoint.NewWriter(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	config.EvaluationDelay = 0
	run, err := NewWithCheckpoint(context.Background(), binding, config, func() time.Time { return now }, writer)
	if err != nil {
		t.Fatal(err)
	}
	ackAt := now.Add(-config.EvaluationDelay)
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	var captureCount atomic.Uint64
	var captureMaxNanos atomic.Int64
	captureStop, captureDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(captureDone)
		for {
			select {
			case <-captureStop:
				return
			default:
			}
			started := time.Now()
			capture, err := run.CaptureSnapshot()
			elapsed := time.Since(started).Nanoseconds()
			for elapsed > captureMaxNanos.Load() && !captureMaxNanos.CompareAndSwap(captureMaxNanos.Load(), elapsed) {
			}
			if err == nil {
				if _, ok := InspectSnapshotCapture(capture); ok {
					captureCount.Add(1)
				}
			}
			time.Sleep(time.Millisecond)
		}
	}()
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, 1, now)

	deadline := time.Now().Add(5 * time.Second)
	for {
		writerAccounting, engineAccounting := writer.Accounting(), run.Engine().CheckpointOperations()
		if writerAccounting.Completed == 1 && engineAccounting.Completed == 1 && engineAccounting.Outstanding == 0 {
			if writerAccounting.LastSuccessfulT0 != now || engineAccounting.LastSuccessfulT0 != now || writerAccounting.LastArtifactBytes <= 0 ||
				writerAccounting.LastEncodeDuration <= 0 || writerAccounting.LastReopenValidationDuration <= 0 {
				t.Fatalf("incomplete usable diagnostics writer=%+v engine=%+v", writerAccounting, engineAccounting)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("usable completion not reconciled writer=%+v engine=%+v", writerAccounting, engineAccounting)
		}
		time.Sleep(time.Millisecond)
	}
	close(captureStop)
	<-captureDone
	finalCapture, err := run.CaptureSnapshot()
	finalView, ok := InspectSnapshotCapture(finalCapture)
	if err != nil || !ok || captureCount.Load() == 0 || time.Duration(captureMaxNanos.Load()) > time.Second || !finalView.Status.BackendReady ||
		finalView.Status.Watermark == nil || !finalView.Status.Watermark.Equal(now) || !finalView.Metrics.AccountingValid {
		t.Fatalf("checkpoint API responsiveness captures=%d max=%s err=%v ok=%t status=%+v metrics=%+v", captureCount.Load(), time.Duration(captureMaxNanos.Load()), err, ok, finalView.Status, finalView.Metrics)
	}
	if loaded := store.Load(context.Background()); loaded.Disposition != checkpoint.LoadedLatest || loaded.Candidate.Image.T0 != now {
		t.Fatalf("completed artifact not discoverable: %+v", loaded)
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestRunLiveWaitsForSessionBeforeHydration(t *testing.T) {
	binding := operationsBinding(t)
	start := binding.SessionStart()
	clockNanos := &atomic.Int64{}
	clockNanos.Store(start.Add(-time.Second).UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	config := DefaultConfig()
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}

	disconnect := make(chan struct{})
	var connections atomic.Int32
	var failNew atomic.Bool
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, disconnect)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("empty session-start hydration made a provider request")
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20,
		MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run.RunLive(ctx, components) }()

	deadline := time.After(2 * time.Second)
	for {
		view := run.Engine().ObserveOperational()
		if view.Lifecycle == "awaiting_session" && view.Connection.Active && view.Connection.Acknowledged {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("pre-session live composition exited: %v", err)
		case <-deadline:
			t.Fatalf("pre-session connection was not retained: %+v", view)
		case <-time.After(time.Millisecond):
		}
	}
	select {
	case err := <-done:
		t.Fatalf("pre-session live composition exited before 04:00: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	clockNanos.Store(start.UnixNano())
	deadline = time.After(2 * time.Second)
	for !run.Status().BackendReady {
		select {
		case err := <-done:
			t.Fatalf("live composition exited after 04:00 transition: %v", err)
		case <-deadline:
			t.Fatalf("session-start hydration/fence did not become ready: status=%+v engine=%+v", run.Status(), run.Engine().ObserveOperational())
		case <-time.After(time.Millisecond):
		}
	}
	if got := run.Engine().ObserveOperational(); got.Lifecycle != "live" || got.Hydration.Purpose != engine.HydrationFreshBootstrap || !got.Hydration.FenceReconciled {
		t.Fatalf("session-start live state=%+v", got)
	}

	cancel()
	shutdown, stop := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer stop()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-session live shutdown=%v", err)
		}
	default:
		t.Fatal("shutdown returned before pre-session RunLive joined")
	}
}

func TestRunLivePreSessionDisconnectUsesBoundedRecovery(t *testing.T) {
	binding := operationsBinding(t)
	start := binding.SessionStart()
	clock := func() time.Time { return start.Add(-time.Second) }
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = time.Second
	config.ShutdownDeadline = 2 * time.Second
	config.RecoveryAttempts = 1
	config.RecoveryBackoffInitial = 5 * time.Second
	config.RecoveryBackoffMax = 5 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}

	disconnect := make(chan struct{})
	var connections atomic.Int32
	var failNew atomic.Bool
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, disconnect)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("pre-session disconnect reached hydration")
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20,
		MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	done := make(chan error, 1)
	go func() { done <- run.RunLive(context.Background(), components) }()

	deadline := time.After(2 * time.Second)
	for !run.Engine().ObserveOperational().Connection.Acknowledged {
		select {
		case err := <-done:
			t.Fatalf("pre-session composition exited before disconnect: %v", err)
		case <-deadline:
			t.Fatalf("pre-session aggregate acknowledgement missing: %+v", run.Engine().ObserveOperational())
		case <-time.After(time.Millisecond):
		}
	}
	close(disconnect)
	deadline = time.After(3 * time.Second)
	for run.Engine().ObserveOperational().Lifecycle != "suppressed" {
		select {
		case err := <-done:
			t.Fatalf("recoverable pre-session suppression ended composition: %v", err)
		case <-deadline:
			t.Fatalf("pre-session disconnect did not suppress: %+v", run.Engine().ObserveOperational())
		case <-time.After(time.Millisecond):
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancel()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if got := adapter.Accounting(); !got.Reconciles() || got.AttemptsActive != 0 || got.AttemptsConnected != 0 {
		t.Fatalf("pre-session disconnect cleanup=%+v", got)
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

	sample := engine.TQPressureSample{WaitingFrames: 10, FrameCapacity: 100, ByteCapacity: 1000, TQLocalAccountingHealthy: true}
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
	cycle(base.Add(time.Second))
	if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureDegraded || !view.ShedTradesQuotes {
		t.Fatalf("runtime degradation = %+v", view)
	}
	sample.TQLocalAccountingHealthy = false
	cycle(base.Add(2 * time.Second))
	if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureAggregateOnly || !view.AggregateOnly {
		t.Fatalf("runtime aggregate-only = %+v", view)
	}
	readyAfter := run.Status()
	if !readyAfter.RankingCurrent || readyAfter.Watermark == nil || readyBefore.Watermark == nil || readyAfter.Watermark.Before(*readyBefore.Watermark) {
		t.Fatalf("T/Q pressure changed aggregate readiness: before=%+v after=%+v", readyBefore, readyAfter)
	}
}

func TestPTQRPressureShedsBeforeQuarterQueueAndHardCapacity(t *testing.T) {
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
		_ = run.Shutdown(ctx)
	}()
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, base)
	run.pressureSampler = func(Metrics) engine.TQPressureSample {
		return engine.TQPressureSample{WaitingFrames: 4096, FrameCapacity: massive.MaximumLiveFrameSlots, ByteCapacity: 1, TQLocalAccountingHealthy: true}
	}
	for index := 0; index < 2; index++ {
		clockNanos.Store(base.Add(time.Duration(index) * time.Second).UnixNano())
		admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
		if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
			t.Fatal("pressure tick")
		}
		run.syncTQPressure(context.Background())
	}
	view := run.Engine().ObserveTQ()
	if view.Pressure != engine.TQPressureDegraded || !view.ShedTradesQuotes || view.AggregateOnly || view.PressureCause != engine.TQPressureCauseWaitingFrames {
		t.Fatalf("12.5%% queue pressure did not shed T/Q below quarter capacity: %+v", view)
	}
}

func TestPC9BroadQueueAccountingLossSuppressesInvalidAggregatePath(t *testing.T) {
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
		return Metrics{LiveQueue: massive.LiveQueueAccounting{FramesRead: 1, CapacityFrames: 100, CapacityBytes: 100}, TQNormalization: massive.TQNormalizationAccounting{}}
	}
	admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
		t.Fatal("pressure tick")
	}
	run.syncTQPressure(context.Background())
	operational := run.Engine().ObserveOperational()
	if operational.Lifecycle != "suppressed" || operational.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
		t.Fatalf("broad queue accounting loss escaped aggregate containment: %+v", operational)
	}
}

func TestPTQRAccountingPartitionClosesOnlyTQ(t *testing.T) {
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
		_ = run.Shutdown(ctx)
	}()
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, now)
	partitioned := massive.AdapterAccounting{CommandsStarted: 1, TQCommandsStarted: 1}
	if !partitioned.TransportReconciles() || partitioned.TQReconciles() || partitioned.Reconciles() {
		t.Fatalf("test accounting did not isolate the T/Q partition: %+v", partitioned)
	}
	run.metricsSnapshot = func() Metrics {
		return Metrics{LiveQueue: massive.LiveQueueAccounting{CapacityFrames: 100, CapacityBytes: 100}, Adapter: partitioned}
	}
	admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
		t.Fatal("pressure tick")
	}
	run.syncTQPressure(context.Background())
	operational := run.Engine().ObserveOperational()
	view := run.Engine().ObserveTQ()
	if operational.Lifecycle == "suppressed" || !operational.Connection.Active || !view.AggregateOnly || view.PressureCause != engine.TQPressureCauseTransportAccounting {
		t.Fatalf("T/Q accounting escaped its failure domain: operational=%+v tq=%+v", operational, view)
	}
}

// TestC8RUNTIME02HydrationOutlivesConnectionDeadline proves the connection
// establishment deadline neither cancels finite hydration nor prevents the
// subscribed live tail from being consumed while REST work remains active.
func TestC8RUNTIME02HydrationOutlivesConnectionDeadline(t *testing.T) {
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	startedClock := time.Now()
	clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
	config := DefaultConfig()
	config.EvaluationDelay = 50 * time.Millisecond
	config.ReadinessTolerance = time.Second
	config.SampleCadence = 20 * time.Millisecond
	config.ConnectionAttemptDeadline = 250 * time.Millisecond
	ctx := context.Background()
	run, err := New(ctx, binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	releaseLive := make(chan struct{})
	liveWindow := base.Add(-config.EvaluationDelay).Truncate(time.Second)
	websocketServer := capacityWebSocketServer(t, releaseLive, []string{"[" + capacityAggregateJSON("AAA", liveWindow, 11, 12) + "]"})
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	releaseHydration := make(chan struct{})
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
			return
		case <-releaseHydration:
		}
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
	close(releaseLive)
	deliveryDeadline := time.After(2 * time.Second)
	for run.Engine().ObserveOperational().Aggregates.Inserted == 0 {
		select {
		case err := <-joined:
			t.Fatalf("live composition ended during hydration: %v", err)
		case <-deliveryDeadline:
			t.Fatalf("live tail was not consumed during hydration: %+v", run.Metrics().LiveQueue)
		case <-time.After(10 * time.Millisecond):
		}
	}
	time.Sleep(config.ConnectionAttemptDeadline + 100*time.Millisecond)
	if got := run.Engine().ObserveOperational(); got.Lifecycle != "hydrating" || !got.Connection.Active {
		t.Fatalf("connection deadline canceled active hydration: %+v", got)
	}
	close(releaseHydration)
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
	time.Sleep(config.ConnectionAttemptDeadline + 100*time.Millisecond)
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

func TestLBRRuntimeOneWorkerDisconnectDuringHydration(t *testing.T) {
	symbols := make([]string, 8)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%02d", index)
	}
	binding := capacityBinding(t, symbols)
	base := binding.SessionStart().Add(20 * time.Minute)
	startedClock := time.Now()
	clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 1
	config.RecoveryBackoffInitial = 5 * time.Second
	config.RecoveryBackoffMax = 5 * time.Second
	config.ConnectionAttemptDeadline = 2 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	allActive := make(chan struct{})
	var active atomic.Int32
	var activeOnce sync.Once
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		if active.Add(1) == 1 {
			activeOnce.Do(func() { close(allActive) })
		}
		defer active.Add(-1)
		<-request.Context().Done()
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	var failNew atomic.Bool
	websocketServer := recoverableWebSocketServer(t, &connections, &failNew, allActive)
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 8 << 20, MaximumNormalizedRecords: int64(len(symbols)) * 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	done := make(chan error, 1)
	go func() { done <- run.RunLive(context.Background(), components) }()
	deadline := time.After(5 * time.Second)
	for run.Engine().ObserveOperational().Lifecycle != "suppressed" {
		select {
		case err := <-done:
			t.Fatalf("recoverable suppression ended the process: %v", err)
		case <-deadline:
			t.Fatalf("disconnect did not reach bounded suppression: active=%d view=%+v", active.Load(), run.Engine().ObserveOperational())
		case <-time.After(10 * time.Millisecond):
		}
	}
	select {
	case err := <-done:
		t.Fatalf("recoverable suppression ended before its scheduled command: %v", err)
	default:
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	waitForNoActiveHydration(t, &active)
	if got := adapter.Accounting(); !got.Reconciles() || got.AttemptsActive != 0 || got.AttemptsConnected != 0 || active.Load() != 0 {
		t.Fatalf("disconnect cleanup did not join: adapter=%+v active_hydration=%d", got, active.Load())
	}
}

func TestC8RUNTIME02ShutdownJoinsBlockedHydrationAndAttempt(t *testing.T) {
	symbols := make([]string, 8)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%02d", index)
	}
	binding := capacityBinding(t, symbols)
	base := binding.SessionStart().Add(20 * time.Minute)
	startedClock := time.Now()
	clock := func() time.Time { return base.Add(time.Since(startedClock)).UTC() }
	config := DefaultConfig()
	config.ConnectionAttemptDeadline = 2 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	allActive := make(chan struct{})
	var active atomic.Int32
	var activeOnce sync.Once
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		if active.Add(1) == 1 {
			activeOnce.Do(func() { close(allActive) })
		}
		defer active.Add(-1)
		<-request.Context().Done()
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
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
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 8 << 20, MaximumNormalizedRecords: int64(len(symbols)) * 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	done := make(chan error, 1)
	go func() { done <- run.RunLive(context.Background(), components) }()
	select {
	case <-allActive:
	case err := <-done:
		t.Fatalf("live composition ended before shutdown: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatalf("hydration workers did not all start: active=%d", active.Load())
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown live result = %v", err)
		}
	default:
		t.Fatal("Shutdown returned before RunLive joined")
	}
	waitForNoActiveHydration(t, &active)
	metrics := run.Metrics()
	if got := adapter.Accounting(); !run.joined.Load() || !got.Reconciles() || got.AttemptsActive != 0 || got.AttemptsConnected != 0 || active.Load() != 0 || !metrics.LiveQueue.Reconciles() {
		t.Fatalf("shutdown cleanup did not join: joined=%t adapter=%+v active_hydration=%d queue=%+v", run.joined.Load(), got, active.Load(), metrics.LiveQueue)
	}
}

func waitForNoActiveHydration(t *testing.T, active *atomic.Int32) {
	t.Helper()
	deadline := time.After(time.Second)
	for active.Load() != 0 {
		select {
		case <-deadline:
			t.Fatalf("hydration request handlers did not exit: active=%d", active.Load())
		case <-time.After(time.Millisecond):
		}
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
			config.RecoveryBackoffInitial = 5 * time.Second
			config.RecoveryBackoffMax = 5 * time.Second
			config.EvaluationDelay = 20 * time.Millisecond
			config.SampleCadence = 10 * time.Millisecond
			config.ConnectionAttemptDeadline = 3 * time.Second
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
				for run.Engine().ObserveOperational().Lifecycle != "suppressed" {
					select {
					case err := <-joined:
						t.Fatalf("recoverable exhaustion ended process: %v", err)
					case <-deadline:
						t.Fatalf("exhaustion did not suppress: %+v", run.Status())
					case <-time.After(10 * time.Millisecond):
					}
				}
				view := run.Engine().ObserveOperational()
				if connections.Load() != 2 || view.Lifecycle != "suppressed" || view.LifecycleReason != "recovery_exhausted" || view.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
					t.Fatalf("exhaustion connections=%d view=%+v", connections.Load(), view)
				}
				select {
				case err := <-joined:
					t.Fatalf("suppression stopped live composition: %v", err)
				default:
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

func TestLiveTerminalStateCannotEnterReconnectLoop(t *testing.T) {
	for _, test := range []struct {
		name string
		view engine.OperationalView
		want bool
	}{
		{"hydrating", engine.OperationalView{Lifecycle: "hydrating"}, false},
		{"recovering", engine.OperationalView{Lifecycle: "recovering"}, false},
		{"suppressed", engine.OperationalView{Lifecycle: "suppressed", LifecycleReason: "accounting_integrity", Suppression: engine.SuppressionRestartRequired}, true},
		{"ended", engine.OperationalView{Lifecycle: "ended", LifecycleReason: "controlled_stop"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := terminalLiveStateError(test.view); (got != nil) != test.want {
				t.Fatalf("terminal state error = %v, want=%t", got, test.want)
			}
		})
	}
}

func TestPHRGapPostLiveRecoveryTraversesFence(t *testing.T) {
	binding := operationsBinding(t)
	supported := binding.SessionStart().Add(70 * time.Second)
	clockStarted := time.Now()
	clock := func() time.Time { return supported.Add(time.Since(clockStarted)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 2
	config.RecoveryBackoffInitial = 20 * time.Millisecond
	config.RecoveryBackoffMax = 20 * time.Millisecond
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = 3 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	firstLoss := make(chan struct{})
	secondLoss := make(chan struct{})
	secondFrameWritten := make(chan struct{})
	replacementFrameWritten := make(chan struct{})
	thirdHydrationActive := make(chan struct{})
	var connections atomic.Int32
	websocketServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		epoch := connections.Add(1)
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
		switch epoch {
		case 1:
			<-firstLoss
		case 2:
			if !write("[" + capacityAggregateJSON("AAA", supported.Add(-2*time.Second), 12, 12) + "]") {
				return
			}
			close(secondFrameWritten)
			<-secondLoss
		case 3:
			select {
			case <-thirdHydrationActive:
			case <-ctx.Done():
				return
			}
			if !write("[" + capacityAggregateJSON("AAA", supported.Add(-time.Second), 13, 13) + "]") {
				return
			}
			close(replacementFrameWritten)
			<-ctx.Done()
		}
	}))
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	secondHydrationActive := make(chan struct{})
	secondHydrationCanceled := make(chan struct{})
	releaseThirdHydration := make(chan struct{})
	var releaseThirdOnce sync.Once
	defer releaseThirdOnce.Do(func() { close(releaseThirdHydration) })
	var requests atomic.Int32
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch requests.Add(1) {
		case 1:
		case 2:
			close(secondHydrationActive)
			<-request.Context().Done()
			close(secondHydrationCanceled)
			return
		case 3:
			close(thirdHydrationActive)
			select {
			case <-releaseThirdHydration:
			case <-request.Context().Done():
				return
			}
		default:
			http.Error(writer, "unexpected hydration generation", http.StatusInternalServerError)
			return
		}
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
	initial := run.Engine().ObserveReplayDeterministic()
	if initial.Publication.Watermark == nil || initial.Publication.Lifecycle != "live" || !initial.Publication.CurrentMarketClaim {
		t.Fatalf("initial committed publication=%+v", initial.Publication)
	}
	close(firstLoss)
	select {
	case <-secondHydrationActive:
	case err := <-joined:
		t.Fatalf("runtime ended before first gap generation: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("first gap generation did not start")
	}
	select {
	case <-secondFrameWritten:
	case <-time.After(time.Second):
		t.Fatal("old recovery epoch did not deliver its live aggregate")
	}
	waitForSlice1Coverage(t, run, joined, func() bool {
		return run.Engine().ObserveOperational().Aggregates.Inserted == 1
	}, "old recovery epoch aggregate")
	close(secondLoss)
	select {
	case <-secondHydrationCanceled:
	case err := <-joined:
		t.Fatalf("runtime ended before old generation cancellation: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("old gap generation request was not canceled")
	}
	select {
	case <-replacementFrameWritten:
	case err := <-joined:
		t.Fatalf("runtime ended before replacement live tail: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("replacement epoch did not deliver live tail during hydration")
	}
	waitForSlice1Coverage(t, run, joined, func() bool {
		return run.Engine().ObserveOperational().Aggregates.Inserted == 2
	}, "replacement live-tail aggregate")
	duringReplacement := run.Engine().ObserveOperational()
	if duringReplacement.Lifecycle == "live" || duringReplacement.CurrentMarketClaim || duringReplacement.Hydration.Generation != 3 ||
		duringReplacement.Hydration.Purpose != engine.HydrationGapRecovery || duringReplacement.Hydration.FenceReconciled {
		t.Fatalf("replacement became current before its fence: %+v", duringReplacement)
	}
	releaseThirdOnce.Do(func() { close(releaseThirdHydration) })
	waitForOperational(t, run, joined, 3, true)
	final := run.Engine().ObserveReplayDeterministic()
	view := run.Engine().ObserveOperational()
	canonical := slice1CanonicalSymbol(t, final, "AAA")
	if connections.Load() != 3 || requests.Load() != 3 || view.Connection.RecoveryAttempts != 0 || view.Lifecycle != "live" ||
		view.Hydration.Generation != 3 || view.Hydration.Purpose != engine.HydrationGapRecovery || !view.Hydration.FenceReconciled ||
		final.Publication.Watermark == nil || view.Hydration.SupportedThrough == nil || *final.Publication.Watermark != *view.Hydration.SupportedThrough || !final.Publication.CurrentMarketClaim ||
		len(canonical.Records) != 2 || canonical.Records[0].WindowStart != supported.Add(-2*time.Second) || canonical.Records[0].Values.Close != 12 ||
		canonical.Records[1].WindowStart != supported.Add(-time.Second) || canonical.Records[1].Values.Close != 13 {
		t.Fatalf("gap recovery did not preserve old tail and fence replacement exactly: connections=%d requests=%d view=%+v publication=%+v canonical=%+v",
			connections.Load(), requests.Load(), view, final.Publication, canonical)
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
	t.Skip("superseded generic-success acknowledgement integration trace")
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
	if view.Rows[0].Tape.Status != engine.TQCurrent || view.Rows[0].Spread.Quality != "known_special" {
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
	pressureSample := engine.TQPressureSample{WaitingFrames: 10, FrameCapacity: 100, ByteCapacity: 1000, TQLocalAccountingHealthy: true}
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
	pressureTick(pressureStart.Add(time.Second))
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
	pressureTick(pressureStart.Add(2 * time.Second))
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
		view.Commands.Issued != view.Commands.Pending+view.Commands.Written+view.Commands.Failed+view.Commands.Fenced {
		t.Fatalf("aggregate-only zero/accounting = %+v", view)
	}
	if metrics := run.Metrics(); !metrics.TQNormalization.Reconciles() || metrics.TQNormalization.Rejected == 0 || metrics.TQNormalization.NormalizedTrades == 0 || metrics.TQNormalization.NormalizedQuotes == 0 || !metrics.Adapter.Reconciles() || !metrics.LiveQueue.Reconciles() {
		t.Fatalf("mixed-frame/runtime accounting = %+v", metrics)
	}
	if status := run.Status(); !status.RankingCurrent {
		t.Fatalf("T/Q pressure changed aggregate readiness: %+v", status)
	}
	_ = attempt.Close(massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: command.CommandToken() + 100, Cause: massive.CloseControlledStop})
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestPC9TQFreshEpochSubscribesAllRankedRowsInOneCommand(t *testing.T) {
	t.Skip("superseded acknowledgement-created coverage integration trace")
	binding := capacityBinding(t, []string{"AAA", "BBB"})
	now := binding.SessionStart().Add(20 * time.Minute)
	config := DefaultConfig()
	config.EvaluationDelay, config.SampleCadence = 0, 10*time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	// Remove timer-driven command opportunities. One explicit synchronization
	// must carry the complete fresh-epoch desired set.
	run.timerCancel()
	<-run.timerDone

	commands := make(chan string, 2)
	server := tqInitialBatchServer(t, commands)
	defer server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(server.URL, "http"), Credential: "fixture", Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	attempt, started, err := adapter.Start(context.Background(), massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	run.setLiveSources(attempt, adapter)
	if result, deliverErr := massive.DeliverToEngine(context.Background(), run.Engine(), started); deliverErr != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("start = %+v/%v", result, deliverErr)
	}
	handshake, err := attempt.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		result, deliverErr := massive.DeliverToEngine(context.Background(), run.Engine(), delivery)
		if deliverErr != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake = %+v/%v", result, deliverErr)
		}
	}
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, attempt.Epoch(), now)

	base := now
	position := uint32(0)
	for second := 0; second < 60; second++ {
		window := base.Add(time.Duration(second) * time.Second)
		now = window.Add(time.Second)
		for _, symbol := range []string{"AAA", "BBB"} {
			position++
			input := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: symbol,
				WindowStart: window, WindowEnd: window.Add(time.Second), Values: engine.AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 10_000, VWAP: 12, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
				DeliveryTime: now, Live: engine.LivePosition{ConnectionEpoch: attempt.Epoch(), FrameSequence: 4, ArrayIndex: position}}
			admission, completion := run.Engine().AdmitAggregate(context.Background(), input)
			if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionAggregateInserted {
				t.Fatalf("aggregate %s/%d admission=%s", symbol, second, admission)
			}
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
	if desired := run.Engine().ObserveTQ().Desired; len(desired) != 2 || desired[0] != "AAA" || desired[1] != "BBB" {
		t.Fatalf("desired rank order = %v", desired)
	}

	run.syncTQCommand(context.Background())
	select {
	case payload := <-commands:
		if !strings.Contains(payload, `"params":"T.AAA,Q.AAA,T.BBB,Q.BBB"`) {
			t.Fatalf("initial batch = %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("initial subscription batch was not written")
	}
	startedDelivery := time.Now()
	result, ok, err := attempt.DeliverNextToEngine(context.Background(), run.Engine())
	if err != nil || !ok {
		t.Fatalf("batch status = %+v/%v/%v", result, ok, err)
	}
	run.observeDelivery(startedDelivery, result)
	if result.TQDisposition.Code != engine.DispositionTQApplied {
		t.Fatalf("batch acknowledgement = %+v", result)
	}
	view := run.Engine().ObserveTQ()
	if view.CommandPending || view.Commands.Issued != 1 || view.Commands.Written != 1 || view.Accounting.KnownPresent != 2 || len(view.Rows) != 2 ||
		!view.Rows[0].TradeCoverage || !view.Rows[1].TradeCoverage || view.Rows[0].Tape.Status != engine.TQWarming || view.Rows[1].Tape.Status != engine.TQWarming {
		t.Fatalf("completed initial batch = %+v", view)
	}
	select {
	case payload := <-commands:
		t.Fatalf("unexpected second command = %s", payload)
	case <-time.After(25 * time.Millisecond):
	}

	_ = attempt.Close(massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 100, Cause: massive.CloseControlledStop})
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

func tqInitialBatchServer(t *testing.T, commands chan<- string) *httptest.Server {
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
		_, payload, err := connection.Read(ctx)
		if err != nil {
			return
		}
		commands <- string(payload)
		if !write(`[{"ev":"status","status":"success"},{"ev":"status","status":"success"},{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`) {
			return
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
