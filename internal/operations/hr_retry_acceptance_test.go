package operations

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/coder/websocket"
)

type hrRetryProvider struct {
	mu                sync.Mutex
	connections       int
	logicalActive     int
	highLogicalActive int
	acceptedAt        []time.Time
	retiredAt         []time.Time
	disconnectInitial chan struct{}
	succeedRecovery   int
}

func (p *hrRetryProvider) handler(writer http.ResponseWriter, request *http.Request) {
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		return
	}
	defer connection.CloseNow()

	p.mu.Lock()
	p.connections++
	number := p.connections
	p.logicalActive++
	if p.logicalActive > p.highLogicalActive {
		p.highLogicalActive = p.logicalActive
	}
	p.acceptedAt = append(p.acceptedAt, time.Now())
	p.retiredAt = append(p.retiredAt, time.Time{})
	p.mu.Unlock()
	retire := func() {
		p.mu.Lock()
		if p.retiredAt[number-1].IsZero() {
			p.retiredAt[number-1] = time.Now()
			p.logicalActive--
		}
		p.mu.Unlock()
	}

	ctx := request.Context()
	write := func(payload string) bool { return connection.Write(ctx, websocket.MessageText, []byte(payload)) == nil }
	if !write(`[{"ev":"status","status":"connected"}]`) {
		retire()
		return
	}
	if _, _, err = connection.Read(ctx); err != nil {
		retire()
		return
	}

	recoveryOrdinal := number - 1
	succeeds := number == 1 || (p.succeedRecovery > 0 && recoveryOrdinal >= p.succeedRecovery)
	if !succeeds {
		if !write(`[{"ev":"status","status":"auth_failed","message":"UNTRUSTED-PROVIDER-PROSE"}]`) {
			retire()
			return
		}
		retire()
		_, _, _ = connection.Read(ctx)
		return
	}
	if !write(`[{"ev":"status","status":"auth_success"}]`) {
		retire()
		return
	}
	if _, _, err = connection.Read(ctx); err != nil {
		retire()
		return
	}
	if !write(`[{"ev":"status","status":"success"}]`) {
		retire()
		return
	}
	if number == 1 {
		select {
		case <-ctx.Done():
		case <-p.disconnectInitial:
		}
		retire()
		return
	}
	<-ctx.Done()
	retire()
}

func (p *hrRetryProvider) snapshot() (connections, high int, accepted, retired []time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connections, p.highLogicalActive, append([]time.Time(nil), p.acceptedAt...), append([]time.Time(nil), p.retiredAt...)
}

func TestPHRRetryProviderTraversal(t *testing.T) {
	t.Run("temporary handshake rejection then exact recovery", func(t *testing.T) {
		runPHRRetryProviderBranch(t, 3)
	})
	t.Run("persistent failure exhausts without another dial", func(t *testing.T) {
		runPHRRetryProviderBranch(t, 0)
	})
}

func runPHRRetryProviderBranch(t *testing.T, succeedRecovery int) {
	t.Helper()
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute)
	started := time.Now()
	clock := func() time.Time { return base.Add(time.Since(started)).UTC() }
	config := DefaultConfig()
	config.RecoveryAttempts = 5
	config.RecoveryBackoffInitial = time.Second
	config.RecoveryBackoffMax = 30 * time.Second
	config.EvaluationDelay = 10 * time.Millisecond
	config.SampleCadence = 10 * time.Millisecond
	config.ConnectionAttemptDeadline = 3 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}

	provider := &hrRetryProvider{disconnectInitial: make(chan struct{}), succeedRecovery: succeedRecovery}
	websocketServer := httptest.NewServer(http.HandlerFunc(provider.handler))
	defer websocketServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`)
	}))
	defer hydrationServer.Close()
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600, Durations: capacityDurations()}
	ctx, cancel := context.WithCancel(context.Background())
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(ctx, components) }()
	waitForOperational(t, run, joined, 1, true)
	close(provider.disconnectInitial)

	if succeedRecovery > 0 {
		deadline := time.After(15 * time.Second)
		for {
			status := run.Status()
			if status.BackendReady && run.Engine().ObserveOperational().Connection.Epoch >= uint64(succeedRecovery+1) {
				break
			}
			select {
			case err := <-joined:
				t.Fatalf("temporary recovery ended process: %v", err)
			case <-deadline:
				t.Fatalf("temporary recovery did not return current: %+v", run.Engine().ObserveOperational())
			case <-time.After(10 * time.Millisecond):
			}
		}
		view := run.Engine().ObserveOperational()
		if view.Connection.RecoveryAttempts != 0 || view.Lifecycle != "live" || view.Hydration.Purpose != engine.HydrationGapRecovery {
			t.Fatalf("successful recovery did not traverse the existing fence: %+v", view)
		}
	} else {
		deadline := time.After(40 * time.Second)
		for run.Engine().ObserveOperational().LifecycleReason != "recovery_exhausted" {
			select {
			case err := <-joined:
				t.Fatalf("persistent failure ended process: %v", err)
			case <-deadline:
				t.Fatalf("persistent failure did not exhaust: %+v", run.Engine().ObserveOperational())
			case <-time.After(10 * time.Millisecond):
			}
		}
	}

	connections, providerHigh, accepted, retired := provider.snapshot()
	wantConnections := 1 + succeedRecovery
	if succeedRecovery == 0 {
		wantConnections = 6
	}
	if connections != wantConnections || providerHigh != 1 || adapter.Accounting().HighAttemptsActive != 1 {
		t.Fatalf("attempt ordering connections=%d want=%d provider_high=%d adapter_high=%d accounting=%+v", connections, wantConnections, providerHigh, adapter.Accounting().HighAttemptsActive, adapter.Accounting())
	}
	for recovery := 1; recovery < connections; recovery++ {
		want := recoveryDelay(time.Second, 30*time.Second, uint64(recovery))
		gap := accepted[recovery].Sub(retired[recovery-1])
		if retired[recovery-1].IsZero() || gap+5*time.Millisecond < want {
			t.Fatalf("recovery ordinal %d gap=%s want>=%s accepted=%v retired=%v", recovery, gap, want, accepted, retired)
		}
	}
	if succeedRecovery == 0 {
		before := connections
		time.Sleep(1100 * time.Millisecond)
		after, _, _, _ := provider.snapshot()
		outcome := run.LatestRecoveryAttempt()
		if after != before || outcome == nil || outcome.ConsecutiveAttempts != 5 || outcome.Lifecycle != "suppressed" || !outcome.NextEligibleAt.IsZero() {
			t.Fatalf("post-exhaustion state connections=%d->%d outcome=%+v", before, after, outcome)
		}
	}

	cancel()
	select {
	case <-joined:
	case <-time.After(3 * time.Second):
		t.Fatal("live runtime did not join")
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}
