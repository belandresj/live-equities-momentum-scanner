package operations

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

func TestPHRStartupLossCancelsAndReplansThroughFence(t *testing.T) {
	symbols := []string{"AAA", "BBB", "CCC", "DDD"}
	binding := capacityBinding(t, symbols)
	base, started := binding.SessionStart().Add(20*time.Minute), time.Now()
	clock := func() time.Time { return base.Add(time.Since(started)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 3
	config.RecoveryBackoffInitial = 20 * time.Millisecond
	config.RecoveryBackoffMax = 20 * time.Millisecond
	config.EvaluationDelay = 10 * time.Millisecond
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = 2 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}

	var firstActive, firstCanceled atomic.Int32
	var replacement atomic.Bool
	allFirstActive := make(chan struct{})
	var activeOnce sync.Once
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if replacement.Load() {
			parts := strings.Split(request.URL.Path, "/")
			if len(parts) <= 4 {
				http.Error(writer, "bad fixture path", http.StatusBadRequest)
				return
			}
			_, _ = fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, parts[4])
			return
		}
		if firstActive.Add(1) == int32(len(symbols)) {
			activeOnce.Do(func() { close(allFirstActive) })
		}
		<-request.Context().Done()
		if firstCanceled.Add(1) == int32(len(symbols)) {
			replacement.Store(true)
		}
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}

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
		if epoch == 1 {
			<-ctx.Done()
			return
		}
		for {
			if _, _, err := connection.Read(ctx); err != nil {
				return
			}
		}
	}))
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 16, MaxFrameBytes: 4096, TotalFrameBytes: 32768}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	durations := capacityDurations()
	durations.HeartbeatInterval = 250 * time.Millisecond
	durations.HeartbeatDeadline = 100 * time.Millisecond
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: len(symbols), RowsPerChunk: 1,
		MaximumResponseBytes: 4 << 20, MaximumNormalizedRecords: int64(len(symbols)) * 57_600, MaximumResidentRecords: int64(len(symbols)) * 57_600, Durations: durations}
	ctx, cancel := context.WithCancel(context.Background())
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(ctx, components) }()
	select {
	case <-allFirstActive:
	case err := <-joined:
		t.Fatalf("startup ended before loss: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatalf("first generation did not fully open: active=%d", firstActive.Load())
	}
	deadline := time.After(5 * time.Second)
	for !run.Status().BackendReady {
		select {
		case err := <-joined:
			t.Fatalf("startup recovery ended process: %v", err)
		case <-deadline:
			t.Fatalf("startup recovery did not become current: %+v", run.Engine().ObserveOperational())
		case <-time.After(10 * time.Millisecond):
		}
	}
	view := run.Engine().ObserveOperational()
	incident := run.FirstIngressIncident()
	if connections.Load() != 2 || firstCanceled.Load() != int32(len(symbols)) || view.Lifecycle != "live" || view.Hydration.Purpose != engine.HydrationFreshBootstrap ||
		view.Hydration.Generation != 2 || !view.Hydration.FenceReconciled || view.Hydration.Accounting.CompletedEmpty != uint64(len(symbols)) ||
		view.Hydration.Accounting.Open != 0 || view.Connection.RecoveryAttempts != 0 || incident == nil || incident.Source != string(massive.TerminalHeartbeat) ||
		incident.Reason != string(massive.TerminalHeartbeatDeadlineNoProgress) || incident.Epoch != 1 {
		t.Fatalf("startup replacement traversal connections=%d canceled=%d incident=%+v view=%+v", connections.Load(), firstCanceled.Load(), incident, view)
	}
	cancel()
	select {
	case <-joined:
	case <-time.After(3 * time.Second):
		t.Fatal("startup runtime did not join")
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}
