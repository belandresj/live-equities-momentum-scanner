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

type ingressReproducerFault string

const ingressReproducerFrameSlots = 8192

const (
	ingressHealthy          ingressReproducerFault = "healthy_concurrent"
	ingressCapacityControl  ingressReproducerFault = "frame_capacity_control"
	ingressCapacity         ingressReproducerFault = "frame_capacity"
	ingressOversizeControl  ingressReproducerFault = "frame_oversize_control"
	ingressOversize         ingressReproducerFault = "frame_oversize"
	ingressReceiptControl   ingressReproducerFault = "receipt_regression_control"
	ingressReceipt          ingressReproducerFault = "receipt_regression"
	ingressStatusControl    ingressReproducerFault = "status_ambiguous_control"
	ingressStatus           ingressReproducerFault = "status_ambiguous"
	ingressAmbiguityControl ingressReproducerFault = "ingress_ambiguity_control"
	ingressAmbiguity        ingressReproducerFault = "ingress_ambiguity"
)

// TestLiveIngressFirstCauseProductionReproducer is P-D1-FIRST-CAUSE and
// P-D1-REPRODUCER. Every case uses Runtime.RunLive, the production Massive
// adapter/queue, real engine admission, two fake REST workers, live-tail
// delivery, terminal cleanup, and the sealed operator capture boundary.
func TestLiveIngressFirstCauseProductionReproducer(t *testing.T) {
	if testing.Short() {
		t.Skip("diagnostic production-composition reproducer")
	}
	for _, test := range []struct {
		fault       ingressReproducerFault
		want        string
		wantHealthy bool
	}{
		{ingressHealthy, "", true},
		{ingressCapacityControl, "", true},
		{ingressCapacity, string(massive.TerminalFrameSlotCapacity), false},
		{ingressOversizeControl, "", true},
		// The production WebSocket read ceiling rejects the bytes before queue
		// admission. Pre-D1 behavior is reader/read_failed, not the queue's
		// protocol/frame_oversize ingress-integrity branch.
		{ingressOversize, string(massive.TerminalReadFailed), false},
		{ingressReceiptControl, "", true},
		{ingressReceipt, string(massive.TerminalReceiptRegression), false},
		{ingressStatusControl, "", true},
		{ingressStatus, string(massive.TerminalStatusAmbiguous), false},
		{ingressAmbiguityControl, "", true},
		{ingressAmbiguity, string(massive.TerminalIngressAmbiguity), false},
	} {
		t.Run(string(test.fault), func(t *testing.T) {
			incident, status := runIngressFirstCauseCase(t, test.fault)
			if test.wantHealthy {
				if incident != nil || status.Lifecycle == "suppressed" || status.AccountingValid == false {
					t.Fatalf("healthy control incident=%+v status=%+v", incident, status)
				}
				t.Logf("healthy lifecycle=%s ready=%t accounting=%t", status.Lifecycle, status.BackendReady, status.AccountingValid)
				return
			}
			if incident == nil || incident.Owner != IngressOwnerAdapterTerminal || incident.Reason != test.want {
				t.Fatalf("first cause=%+v want=%s", incident, test.want)
			}
			if incident.PriorEngine.Lifecycle != "hydrating" || incident.PriorEngine.Suppression != "" ||
				incident.Hydration.Accounting.Open > incident.Hydration.Accounting.Planned || incident.Hydration.FenceReconciled {
				t.Fatalf("last coherent engine state=%+v", incident.PriorEngine)
			}
			if test.fault == ingressReceipt {
				if incident.LifecycleReason != "clock_regression" || incident.Suppression != engine.SuppressionRestartRequired {
					t.Fatalf("receipt regression lifecycle=%+v", incident)
				}
			} else if test.fault == ingressOversize {
				if incident.LifecycleReason == "ingress_integrity" || incident.Queue.FramesRejectedOversize != 0 {
					t.Fatalf("websocket oversize changed suppression behavior: %+v", incident)
				}
			} else if incident.Lifecycle != "suppressed" || incident.LifecycleReason != "ingress_integrity" || incident.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
				t.Fatalf("ingress lifecycle=%+v", incident)
			}
			for index := 0; index < incident.IdentityCount; index++ {
				if !incident.Identities[index].Reconciled {
					t.Fatalf("unrelated identity failed: %+v incident=%+v", incident.Identities[index], incident)
				}
			}
			t.Logf("first_owner=%s source=%s reason=%s lifecycle=%s lifecycle_reason=%s suppression=%s epoch=%d position=%+v queue_read=%d queue_high=%d hydration=%d/%d",
				incident.Owner, incident.Source, incident.Reason, incident.Lifecycle, incident.LifecycleReason, incident.Suppression, incident.Epoch, incident.Position,
				incident.Queue.FramesRead, incident.QueueHighFrames, incident.Hydration.Accounting.Planned-incident.Hydration.Accounting.Open, incident.Hydration.Accounting.Planned)
			if incident.CapturedAt.IsZero() || incident.EngineCapturedAt.Before(incident.CapturedAt) {
				t.Fatalf("causal capture timestamps invalid: %+v", incident)
			}
			if test.fault == ingressCapacity && (incident.Queue.FramesRejectedSlotCapacity != 1 || incident.Queue.FramesQueued != ingressReproducerFrameSlots || incident.QueueHighFrames != ingressReproducerFrameSlots) {
				t.Fatalf("capacity causal accounting=%+v high=%d", incident.Queue, incident.QueueHighFrames)
			}
		})
	}
}

func TestLiveIngressRuntimeAccountingGuardCoherentMismatchAndHealthyConcurrency(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		incident, status := runIngressFirstCauseCase(t, ingressHealthy)
		if incident != nil || status.Lifecycle == "suppressed" || !status.AccountingValid {
			t.Fatalf("real concurrent delivery false-suppressed: incident=%+v status=%+v", incident, status)
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		run := newIngressAccountingRuntime(t)
		mismatch := Metrics{SampledAt: run.clock().UTC(), LiveQueue: massive.LiveQueueAccounting{FramesRead: 2, FramesAdmitted: 1, FramesDispositioned: 1, CapacityFrames: 100, CapacityBytes: 100}, Adapter: massive.AdapterAccounting{ConnectionAttempts: 1, AttemptsActive: 1}}
		run.metricsSnapshot = func() Metrics { return mismatch }
		pressureTick(t, run)
		run.syncTQPressure(context.Background())
		incident := run.FirstIngressIncident()
		if incident == nil || incident.Owner != IngressOwnerRuntimeAccounting || incident.Reason != "queue.frames_read" {
			t.Fatalf("runtime accounting incident=%+v", incident)
		}
		found := false
		for index := 0; index < incident.IdentityCount; index++ {
			identity := incident.Identities[index]
			if identity.Name == "queue.frames_read" {
				found = !identity.Reconciled && identity.Left == 2 && identity.Right == 1
			}
		}
		if !found {
			t.Fatalf("exact failed operands absent: %+v", incident.Identities)
		}
		if view := run.Engine().ObserveTQ(); view.Pressure != engine.TQPressureAggregateOnly || !view.AggregateOnly {
			t.Fatalf("runtime accounting loss did not shed/unsubscribe T/Q: %+v", view)
		}
	})
}

func TestIngressIncidentFirstWriteSurvivesConcurrentCleanup(t *testing.T) {
	var latch ingressIncidentLatch
	first := IngressIncident{Owner: IngressOwnerAdapterTerminal, Source: "protocol", Reason: string(massive.TerminalIngressAmbiguity), Epoch: 7}
	latch.set(first)
	var joined sync.WaitGroup
	for index := 0; index < 32; index++ {
		joined.Add(1)
		go func() {
			defer joined.Done()
			latch.set(IngressIncident{Owner: IngressOwnerEngineTransition, Source: "cleanup", Reason: "controlled_stop", Epoch: 7})
		}()
	}
	joined.Wait()
	got := latch.get()
	if got == nil || got.Owner != first.Owner || got.Source != first.Source || got.Reason != first.Reason || got.Epoch != first.Epoch {
		t.Fatalf("first cause overwritten: %+v", got)
	}
}

func TestIngressDiagnosticHistoryIsFixedAndChronological(t *testing.T) {
	var history ingressDiagnosticHistory
	base := time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	for index := 0; index < maximumIngressHistorySamples+7; index++ {
		history.add(IngressDiagnosticSample{CapturedAt: base.Add(time.Duration(index) * time.Second), FramesRead: uint64(index)})
	}
	samples, count := history.snapshot()
	if count != maximumIngressHistorySamples || samples[0].FramesRead != 7 || samples[count-1].FramesRead != maximumIngressHistorySamples+6 {
		t.Fatalf("bounded history count=%d first=%+v last=%+v", count, samples[0], samples[count-1])
	}
}

func TestIngressIncidentCaptureDoesNotInsertOffCadenceHistory(t *testing.T) {
	base := time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	assertPeriodicOnly := func(t *testing.T, incident *IngressIncident) {
		t.Helper()
		if incident == nil || incident.HistoryCount != 1 || incident.History[0].CapturedAt != base || incident.History[0].FramesRead != 7 {
			t.Fatalf("fault capture mutated one-second history: %+v", incident)
		}
	}
	t.Run("runtime accounting guard", func(t *testing.T) {
		run := newIngressAccountingRuntime(t)
		run.ingressHistory.add(IngressDiagnosticSample{CapturedAt: base, FramesRead: 7})
		metrics := run.Metrics()
		metrics.SampledAt = base.Add(500 * time.Millisecond)
		run.recordRuntimeAccountingIncident("queue.frames_read", metrics)
		assertPeriodicOnly(t, run.FirstIngressIncident())
	})
	t.Run("adapter terminal", func(t *testing.T) {
		run := newIngressAccountingRuntime(t)
		run.ingressHistory.add(IngressDiagnosticSample{CapturedAt: base, FramesRead: 7})
		terminal := massive.TerminalResult{BindingIdentity: run.binding.Identity(), ConnectionEpoch: 1,
			Source: massive.TerminalProtocol, Reason: massive.TerminalFrameSlotCapacity,
			CauseAccountingCapturedAt: base.Add(500 * time.Millisecond)}
		prior := run.engine.ObserveOperational()
		run.recordAdapterTerminal(massive.EngineDeliveryResult{Terminal: &terminal, PriorEngine: prior, PriorEngineApplicable: true})
		assertPeriodicOnly(t, run.FirstIngressIncident())
	})
}

func newIngressAccountingRuntime(t *testing.T) *Runtime {
	t.Helper()
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = run.Shutdown(ctx)
	})
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, now)
	return run
}

func pressureTick(t *testing.T, run *Runtime) {
	t.Helper()
	admission, completion := run.Engine().AdmitTQPressureTick(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("pressure tick admission")
	}
	<-completion
}

func runIngressFirstCauseCase(t *testing.T, fault ingressReproducerFault) (*IngressIncident, Status) {
	t.Helper()
	binding := capacityBinding(t, []string{"AAA", "BBB", "CCC", "DDD"})
	base := binding.SessionStart().Add(20 * time.Minute)
	clockNanos := &atomic.Int64{}
	clockNanos.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	hydrationStarted := make(chan struct{})
	capacityControlHydrationRelease := make(chan struct{})
	var capacityControlReleaseOnce sync.Once
	var hydrationOnce sync.Once
	rest := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hydrationOnce.Do(func() { close(hydrationStarted) })
		parts := strings.Split(request.URL.Path, "/")
		symbol := ""
		if len(parts) > 4 {
			symbol = parts[4]
		}
		if symbol == "DDD" && fault == ingressCapacityControl {
			select {
			case <-request.Context().Done():
				return
			case <-capacityControlHydrationRelease:
			}
		} else {
			delay := 10 * time.Millisecond
			if symbol == "DDD" {
				delay = 250 * time.Millisecond
			}
			timer := time.NewTimer(delay)
			select {
			case <-request.Context().Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, symbol)
	}))
	defer rest.Close()
	ws := ingressFirstCauseWebSocketServer(t, fault, hydrationStarted, clockNanos, base)
	defer ws.Close()
	queue := massive.LiveQueueConfig{FrameSlots: ingressReproducerFrameSlots, MaxFrameBytes: 8 << 20, TotalFrameBytes: massive.MaximumLiveQueueBytes}
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(ws.URL, "http"), Credential: "fixture", Queue: queue, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker(rest.URL, func() (string, error) { return "fixture", nil }, rest.Client())
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.RecoveryAttempts = 1
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Minute
	config.ConnectionAttemptDeadline = time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	var pumpReleased atomic.Bool
	if fault == ingressCapacity || fault == ingressCapacityControl {
		run.beforeHydrationPump = func(attempt *massive.LiveAttempt) {
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				accounting := attempt.QueueAccounting()
				if fault == ingressCapacity && accounting.FramesRejectedCapacity == 1 {
					pumpReleased.Store(true)
					return
				}
				if fault == ingressCapacityControl && accounting.FramesQueued == uint64(queue.FrameSlots) {
					pumpReleased.Store(true)
					go func() {
						deadline := time.Now().Add(5 * time.Second)
						for time.Now().Before(deadline) {
							current := attempt.QueueAccounting()
							if current.FramesQueued == 0 && current.FramesClassifying == 0 {
								capacityControlReleaseOnce.Do(func() { close(capacityControlHydrationRelease) })
								return
							}
							time.Sleep(time.Millisecond)
						}
						capacityControlReleaseOnce.Do(func() { close(capacityControlHydrationRelease) })
					}()
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Errorf("production queue boundary not reached: %s %+v", fault, attempt.QueueAccounting())
		}
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 2, RowsPerChunk: 4, MaximumResponseBytes: 1 << 20,
		MaximumNormalizedRecords: int64(len(binding.UniverseSymbols())) * 57_600, MaximumResidentRecords: 2 * 57_600, Durations: capacityDurations()}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	done := make(chan error, 1)
	go func() { done <- run.RunLive(ctx, components) }()
	var healthyStatus Status
	if ingressHealthyControl(fault) {
		deadline := time.After(10 * time.Second)
		for !run.Status().BackendReady {
			select {
			case err := <-done:
				t.Fatalf("healthy RunLive ended: %v", err)
			case <-deadline:
				t.Fatalf("healthy control did not become ready: %+v queue=%+v pump_released=%t", run.Engine().ObserveOperational(), run.Metrics().LiveQueue, pumpReleased.Load())
			case <-time.After(time.Millisecond):
			}
		}
		if fault == ingressHealthy {
			for index := 0; index < 32; index++ {
				pressureTick(t, run)
				run.syncTQPressure(context.Background())
				if incident := run.FirstIngressIncident(); incident != nil {
					t.Fatalf("guard false-suppressed during real delivery at sample %d: %+v", index, incident)
				}
				time.Sleep(time.Millisecond)
			}
		}
		healthyStatus = run.Status()
		cancel()
		<-done
	} else {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatalf("fault did not terminate: %s engine=%+v", fault, run.Engine().ObserveOperational())
		}
		cancel()
	}
	shutdown, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	capture, err := run.CaptureSnapshot()
	view, ok := InspectSnapshotCapture(capture)
	if err != nil || !ok {
		t.Fatalf("operator capture: %v", err)
	}
	if ingressHealthyControl(fault) {
		return view.IngressIncident, healthyStatus
	}
	return view.IngressIncident, view.Status
}

func ingressHealthyControl(fault ingressReproducerFault) bool {
	switch fault {
	case ingressHealthy, ingressCapacityControl, ingressOversizeControl, ingressReceiptControl, ingressStatusControl, ingressAmbiguityControl:
		return true
	default:
		return false
	}
}

func ingressFirstCauseWebSocketServer(t *testing.T, fault ingressReproducerFault, hydrationStarted <-chan struct{}, clockNanos *atomic.Int64, base time.Time) *httptest.Server {
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
		select {
		case <-ctx.Done():
			return
		case <-hydrationStarted:
		}
		timer := time.NewTimer(80 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		valid := fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":10,"h":11,"l":9,"c":10,"v":100,"vw":10,"z":10}`, base.Add(-time.Second).UnixMilli(), base.UnixMilli())
		switch fault {
		case ingressHealthy:
			ticker := time.NewTicker(2 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if !write("[" + valid + "]") {
						return
					}
				}
			}
		case ingressCapacity, ingressCapacityControl:
			count := ingressReproducerFrameSlots
			if fault == ingressCapacity {
				count++
			}
			frame := "[" + valid + "]"
			for index := 0; index < count; index++ {
				if !write(frame) {
					return
				}
			}
			if fault == ingressCapacityControl {
				<-ctx.Done()
			}
		case ingressOversize, ingressOversizeControl:
			size := 8 << 20
			if fault == ingressOversize {
				size++
			}
			_ = write("[" + strings.Repeat(" ", size-2) + "]")
		case ingressReceipt:
			if !write(`[]`) {
				return
			}
			time.Sleep(10 * time.Millisecond)
			clockNanos.Store(base.Add(-time.Second).UnixNano())
			_ = write(`[]`)
		case ingressReceiptControl:
			if !write(`[]`) {
				return
			}
			time.Sleep(10 * time.Millisecond)
			_ = write(`[]`)
		case ingressStatus:
			_ = write(`[{"ev":"status","status":"success"}]`)
		case ingressStatusControl:
			_ = write("[" + valid + "]")
		case ingressAmbiguity:
			_ = write("[{}," + valid + "]")
		case ingressAmbiguityControl:
			_ = write("[" + valid + "]")
		}
		<-ctx.Done()
	}))
}
