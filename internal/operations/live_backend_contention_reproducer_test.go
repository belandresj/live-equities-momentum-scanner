package operations

import (
	"context"
	"crypto/sha256"
	"errors"
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
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	liveContentionPopulation         = 6_000
	liveContentionFrames             = 882
	liveContentionAggregatesPerFrame = 2
	liveContentionValidAggregates    = 1_274
	liveContentionUnknownAggregates  = 490
	liveContentionDuration           = 6_140 * time.Millisecond
	liveContentionTailDrain          = 250 * time.Millisecond
	liveContentionQueueStop          = 384
)

var liveContentionIntervals = [...]struct {
	duration time.Duration
	frames   int
}{
	{time.Second, 130},
	{1_035 * time.Millisecond, 149},
	{1_473 * time.Millisecond, 312},
	{2_632 * time.Millisecond, 291},
}

type liveContentionFixture struct {
	frames       []string
	digest       [32]byte
	valid        int
	unknown      int
	minimumBytes int
	maximumBytes int
}

type liveContentionBoundary struct {
	elapsed       time.Duration
	framesRead    uint64
	dispositioned uint64
	queued        uint64
	oldest        time.Duration
}

type liveContentionProbeResult struct {
	maximumQueued uint64
	maximumOldest time.Duration
	stopReason    string
}

type liveContentionResult struct {
	variant                                         string
	elapsed                                         time.Duration
	framesSent, framesRead, framesAdmitted          uint64
	framesDispositioned, framesQueued, framesFenced uint64
	framesRejected                                  uint64
	aggregatesConsumed, aggregatesInserted          uint64
	aggregatesRejected, aggregatesRevised           uint64
	aggregatesFenced                                uint64
	maximumQueued                                   uint64
	maximumOldest, meanDelay, maximumDelay          time.Duration
	deliveries                                      uint64
	averageCPUCores                                 float64
	tailDrained, accountingReconciled, saturated    bool
	stopReason                                      string
	boundaries                                      []liveContentionBoundary
}

// TestLiveBackendContentionReproducer compares the actual paced C5-to-engine
// pipeline with and without T1's 10 ms heavyweight metrics probe. It is a
// deterministic, credential-free diagnostic and changes no production path.
func TestLiveBackendContentionReproducer(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in live backend contention reproducer")
	}
	symbols := make([]string, liveContentionPopulation)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	windowStart := binding.SessionStart().Add(8 * time.Hour).Truncate(time.Second)
	fixture := buildLiveContentionFixture(t, symbols, windowStart)
	validateLiveContentionFixture(t, binding, fixture)

	for _, variant := range []struct {
		name        string
		legacyProbe bool
	}{
		{name: "control"},
		{name: "legacy_probe", legacyProbe: true},
	} {
		t.Run(variant.name, func(t *testing.T) {
			result := runLiveContentionVariant(t, binding, fixture, variant.name, variant.legacyProbe)
			for index, boundary := range result.boundaries {
				t.Logf("variant=%s boundary=%d elapsed=%s frames_read=%d frames_dispositioned=%d queued=%d oldest=%s",
					result.variant, index+1, boundary.elapsed, boundary.framesRead, boundary.dispositioned, boundary.queued, boundary.oldest)
			}
			t.Logf("variant=%s outcome=%s stop=%s elapsed=%s frames_sent=%d frames_read=%d frames_admitted=%d frames_dispositioned=%d frames_queued=%d frames_fenced=%d frames_rejected=%d aggregates_consumed=%d aggregates_inserted=%d aggregates_rejected=%d aggregates_revised=%d aggregates_fenced=%d max_queued=%d max_oldest=%s deliveries=%d mean_delay=%s max_delay=%s frames_per_s=%.2f aggregates_per_s=%.2f avg_cpu_cores=%.3f tail_drained=%t accounting_reconciled=%t",
				result.variant, liveContentionOutcome(result), result.stopReason, result.elapsed, result.framesSent, result.framesRead, result.framesAdmitted,
				result.framesDispositioned, result.framesQueued, result.framesFenced, result.framesRejected, result.aggregatesConsumed,
				result.aggregatesInserted, result.aggregatesRejected, result.aggregatesRevised, result.aggregatesFenced,
				result.maximumQueued, result.maximumOldest, result.deliveries, result.meanDelay, result.maximumDelay,
				float64(result.framesDispositioned)/result.elapsed.Seconds(), float64(result.aggregatesConsumed)/result.elapsed.Seconds(),
				result.averageCPUCores, result.tailDrained, result.accountingReconciled)
			if !result.accountingReconciled {
				t.Fatal("contention reproducer accounting did not reconcile")
			}
			if variant.name == "control" && liveContentionOutcome(result) != "kept_up" {
				t.Fatalf("unprobed control outcome = %s, want kept_up", liveContentionOutcome(result))
			}
		})
	}
}

func buildLiveContentionFixture(t *testing.T, symbols []string, windowStart time.Time) liveContentionFixture {
	t.Helper()
	fixture := liveContentionFixture{frames: make([]string, liveContentionFrames), minimumBytes: int(^uint(0) >> 1)}
	validIndex, unknownIndex := 0, 0
	for frameIndex := range fixture.frames {
		items := make([]string, liveContentionAggregatesPerFrame)
		for itemIndex := range items {
			aggregateIndex := frameIndex*liveContentionAggregatesPerFrame + itemIndex
			validThrough := (aggregateIndex + 1) * liveContentionValidAggregates / (liveContentionFrames * liveContentionAggregatesPerFrame)
			validBefore := aggregateIndex * liveContentionValidAggregates / (liveContentionFrames * liveContentionAggregatesPerFrame)
			symbol := ""
			if validThrough > validBefore {
				symbol = symbols[validIndex]
				validIndex++
				fixture.valid++
			} else {
				symbol = fmt.Sprintf("U%05d", unknownIndex)
				unknownIndex++
				fixture.unknown++
			}
			second := frameIndex * 6 / liveContentionFrames
			window := windowStart.Add(time.Duration(second) * time.Second)
			items[itemIndex] = liveContentionAggregateJSON(symbol, window, aggregateIndex)
		}
		fixture.frames[frameIndex] = "[" + strings.Join(items, ",") + "]"
		fixture.minimumBytes = min(fixture.minimumBytes, len(fixture.frames[frameIndex]))
		fixture.maximumBytes = max(fixture.maximumBytes, len(fixture.frames[frameIndex]))
	}
	fixture.digest = sha256.Sum256([]byte(strings.Join(fixture.frames, "\n")))
	return fixture
}

func liveContentionAggregateJSON(symbol string, window time.Time, ordinal int) string {
	open := 10 + float64(ordinal%100)/100
	closeValue := open + 0.05
	return fmt.Sprintf(`{"ev":"A","sym":%q,"s":%d,"e":%d,"o":%.2f,"h":%.2f,"l":%.2f,"c":%.2f,"v":100,"z":10,"vw":%.3f,"fixture_pad":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`,
		symbol, window.UnixMilli(), window.Add(time.Second).UnixMilli(), open, closeValue+0.05, open-0.05, closeValue, (open+closeValue)/2)
}

func validateLiveContentionFixture(t *testing.T, binding reference.Binding, fixture liveContentionFixture) {
	t.Helper()
	intervalFrames := 0
	for _, interval := range liveContentionIntervals {
		intervalFrames += interval.frames
	}
	if len(binding.UniverseSymbols()) != liveContentionPopulation || len(fixture.frames) != liveContentionFrames ||
		intervalFrames != liveContentionFrames || fixture.valid != liveContentionValidAggregates || fixture.unknown != liveContentionUnknownAggregates ||
		fixture.valid+fixture.unknown != liveContentionFrames*liveContentionAggregatesPerFrame || fixture.minimumBytes < 350 || fixture.maximumBytes > 450 {
		t.Fatalf("fixture manifest population=%d frames=%d intervals=%d valid=%d unknown=%d min_bytes=%d max_bytes=%d",
			len(binding.UniverseSymbols()), len(fixture.frames), intervalFrames, fixture.valid, fixture.unknown, fixture.minimumBytes, fixture.maximumBytes)
	}
	if fixture.digest != sha256.Sum256([]byte(strings.Join(fixture.frames, "\n"))) {
		t.Fatal("fixture digest is not stable")
	}
}

func runLiveContentionVariant(t *testing.T, binding reference.Binding, fixture liveContentionFixture, variant string, legacyProbe bool) liveContentionResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	server := newLiveContentionServer(t, fixture.frames)
	defer server.server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(server.server.URL, "http"), Credential: "fixture-credential",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	now := binding.SessionStart().Add(8*time.Hour + 10*time.Second).Truncate(time.Second)
	run, err := New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	if result, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), started); deliverErr != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection attempt=%+v err=%v", result, deliverErr)
	}
	handshake, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		result, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), delivery)
		if deliverErr != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("handshake=%+v err=%v", result, deliverErr)
		}
	}
	run.setLiveSources(attempt, adapter)
	queueBaseline := attempt.QueueAccounting()
	engineBaseline := run.Engine().ObserveOperational()
	deliveryBaseline := run.deliveryCount.Load()
	deliveryNanosBaseline := run.deliveryTotalNanos.Load()
	deliveryMaxBaseline := run.deliveryMaxNanos.Load()
	cpuBaseline := goRuntimeCPUSeconds()
	baseline := liveDiagnosticBaseline{deliveries: deliveryBaseline, deliveryNanos: deliveryNanosBaseline, deliveryMax: deliveryMaxBaseline, cpuSeconds: cpuBaseline}
	if legacyProbe {
		baseline.metrics = run.Metrics()
	}

	trialCtx, cancelTrial := context.WithCancel(ctx)
	defer cancelTrial()
	deliveryDone := make(chan error, 1)
	go func() {
		for {
			startedDelivery := time.Now()
			result, ok, deliveryErr := attempt.DeliverNextToEngine(trialCtx, run.Engine())
			if ok {
				run.observeDelivery(startedDelivery, result)
			}
			if deliveryErr != nil || !ok {
				deliveryDone <- deliveryErr
				return
			}
		}
	}()

	plannedStart := time.Now().Add(50 * time.Millisecond)
	baseline.started = plannedStart
	server.start <- plannedStart
	probeStop := make(chan struct{})
	probeDone := make(chan liveContentionProbeResult, 1)
	probeSafety := make(chan string, 1)
	if legacyProbe {
		go runLegacyContentionProbe(run, baseline, probeStop, probeDone, probeSafety)
	}

	result := liveContentionResult{variant: variant, boundaries: make([]liveContentionBoundary, 0, len(liveContentionIntervals))}
	boundaryAt := time.Duration(0)
	for _, interval := range liveContentionIntervals {
		boundaryAt += interval.duration
		boundaryTimer := time.NewTimer(max(time.Until(plannedStart.Add(boundaryAt)), 0))
		if legacyProbe {
			select {
			case reason := <-probeSafety:
				if !boundaryTimer.Stop() {
					<-boundaryTimer.C
				}
				result.saturated, result.stopReason = true, reason
			case <-boundaryTimer.C:
			}
		} else {
			<-boundaryTimer.C
		}
		if result.saturated {
			break
		}
		queue := attempt.QueueAccounting()
		result.boundaries = append(result.boundaries, liveContentionBoundary{elapsed: time.Since(plannedStart), framesRead: queue.FramesRead - queueBaseline.FramesRead,
			dispositioned: queue.FramesDispositioned - queueBaseline.FramesDispositioned, queued: queue.FramesQueued, oldest: queue.OldestWaitingFrameAge})
		result.maximumQueued = max(result.maximumQueued, queue.FramesQueued)
		result.maximumOldest = max(result.maximumOldest, queue.OldestWaitingFrameAge)
		if liveContentionFrameRejections(queue, queueBaseline) > 0 || queue.FramesQueued >= liveContentionQueueStop {
			result.saturated, result.stopReason = true, "boundary_safety_stop"
			break
		}
	}

	if !result.saturated {
		select {
		case sendErr := <-server.done:
			if sendErr != nil {
				t.Fatalf("paced fake WebSocket: %v", sendErr)
			}
		case <-ctx.Done():
			t.Fatal("paced fake WebSocket did not finish")
		}
		drainDeadline := time.Now().Add(liveContentionTailDrain)
		for time.Now().Before(drainDeadline) {
			queue := attempt.QueueAccounting()
			if queue.FramesDispositioned-queueBaseline.FramesDispositioned == liveContentionFrames && queue.FramesQueued == 0 && queue.FramesClassifying == 0 {
				result.tailDrained = true
				break
			}
			time.Sleep(time.Millisecond)
		}
		if !result.tailDrained {
			result.saturated, result.stopReason = true, "tail_drain_timeout"
		}
	}

	result.elapsed = time.Since(plannedStart)
	cpuAtBoundary := goRuntimeCPUSeconds()
	timedQueue := attempt.QueueAccounting()
	timedFramesSent := server.sent.Load()

	closeCommand := massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 101, Cause: massive.CloseControlledStop}
	if err := attempt.Close(closeCommand); err != nil {
		t.Fatalf("close contention ingress at measured boundary: %v", err)
	}
	cancelTrial()
	select {
	case <-deliveryDone:
	case <-time.After(10 * time.Second):
		t.Fatal("contention delivery loop did not join")
	}
	timedEngine := run.Engine().ObserveOperational()
	timedDeliveries := run.deliveryCount.Load() - deliveryBaseline
	timedDeliveryNanos := run.deliveryTotalNanos.Load() - deliveryNanosBaseline
	timedDeliveryMax := run.deliveryMaxNanos.Load()
	if legacyProbe {
		close(probeStop)
		probe := <-probeDone
		result.maximumQueued = max(result.maximumQueued, probe.maximumQueued)
		result.maximumOldest = max(result.maximumOldest, probe.maximumOldest)
		if probe.stopReason != "" {
			result.saturated, result.stopReason = true, probe.stopReason
		}
	}
	run.closeAndDrain(attempt, closeCommand.CommandToken, closeCommand.Cause)
	reconciledMetrics := run.Metrics()
	reconciledQueue := reconciledMetrics.LiveQueue
	reconciledEngine := reconciledMetrics.Engine
	result.framesSent = timedFramesSent
	result.framesRead = timedQueue.FramesRead - queueBaseline.FramesRead
	result.framesAdmitted = timedQueue.FramesAdmitted - queueBaseline.FramesAdmitted
	result.framesDispositioned = timedQueue.FramesDispositioned - queueBaseline.FramesDispositioned
	result.framesQueued = timedQueue.FramesQueued
	result.framesFenced = timedQueue.FramesFenced - queueBaseline.FramesFenced
	result.framesRejected = liveContentionFrameRejections(timedQueue, queueBaseline)
	result.aggregatesConsumed = timedEngine.Aggregates.Consumed - engineBaseline.Aggregates.Consumed
	result.aggregatesInserted = timedEngine.Aggregates.Inserted - engineBaseline.Aggregates.Inserted
	result.aggregatesRejected = timedEngine.Aggregates.Rejected - engineBaseline.Aggregates.Rejected
	result.aggregatesRevised = timedEngine.Aggregates.Revised - engineBaseline.Aggregates.Revised
	result.aggregatesFenced = timedEngine.Aggregates.Fenced - engineBaseline.Aggregates.Fenced
	result.maximumQueued = max(result.maximumQueued, reconciledMetrics.QueueHighFrames)
	result.maximumOldest = max(result.maximumOldest, timedQueue.OldestWaitingFrameAge)
	result.deliveries = timedDeliveries
	if result.deliveries > 0 {
		result.meanDelay = time.Duration(timedDeliveryNanos / result.deliveries)
	}
	if timedDeliveryMax > deliveryMaxBaseline {
		result.maximumDelay = time.Duration(timedDeliveryMax)
	}
	cpuSeconds := cpuAtBoundary - cpuBaseline
	if result.elapsed > 0 && cpuSeconds >= 0 {
		result.averageCPUCores = cpuSeconds / result.elapsed.Seconds()
	}
	reconciledRead := reconciledQueue.FramesRead - queueBaseline.FramesRead
	reconciledAdmitted := reconciledQueue.FramesAdmitted - queueBaseline.FramesAdmitted
	reconciledDispositioned := reconciledQueue.FramesDispositioned - queueBaseline.FramesDispositioned
	reconciledFenced := reconciledQueue.FramesFenced - queueBaseline.FramesFenced
	reconciledRejected := liveContentionFrameRejections(reconciledQueue, queueBaseline)
	reconciledConsumed := reconciledEngine.Aggregates.Consumed - engineBaseline.Aggregates.Consumed
	reconciledInserted := reconciledEngine.Aggregates.Inserted - engineBaseline.Aggregates.Inserted
	reconciledAggregateRejected := reconciledEngine.Aggregates.Rejected - engineBaseline.Aggregates.Rejected
	reconciledRevised := reconciledEngine.Aggregates.Revised - engineBaseline.Aggregates.Revised
	reconciledAggregateFenced := reconciledEngine.Aggregates.Fenced - engineBaseline.Aggregates.Fenced
	result.accountingReconciled = reconciledQueue.Reconciles() && reconciledMetrics.Adapter.Reconciles() && reconciledEngine.Aggregates.Reconciles() &&
		reconciledEngine.Admissions.Reconciles(reconciledEngine.QueueOccupancy) && reconciledEngine.Transitions.Reconciles() && reconciledMetrics.AccountingValid &&
		reconciledRead <= liveContentionFrames && reconciledAdmitted+reconciledRejected == reconciledRead &&
		reconciledAdmitted == reconciledDispositioned+reconciledFenced && reconciledQueue.FramesQueued == 0 &&
		reconciledConsumed == reconciledInserted+reconciledAggregateRejected+reconciledRevised+reconciledAggregateFenced
	if !result.saturated {
		result.accountingReconciled = result.accountingReconciled && result.framesSent == liveContentionFrames && result.framesRead == liveContentionFrames &&
			result.framesAdmitted == liveContentionFrames && result.framesRejected == 0 && result.framesDispositioned == liveContentionFrames &&
			result.aggregatesConsumed == liveContentionFrames*liveContentionAggregatesPerFrame && result.aggregatesInserted == liveContentionValidAggregates &&
			result.aggregatesRejected == liveContentionUnknownAggregates && result.aggregatesRevised == 0 && result.aggregatesFenced == 0
	}
	if !result.accountingReconciled {
		t.Logf("reconciliation detail queue_ok=%t adapter_ok=%t aggregates_ok=%t admissions_ok=%t transitions_ok=%t metrics_ok=%t queue=%+v adapter=%+v admissions=%+v transitions=%+v",
			reconciledQueue.Reconciles(), reconciledMetrics.Adapter.Reconciles(), reconciledEngine.Aggregates.Reconciles(), reconciledEngine.Admissions.Reconciles(reconciledEngine.QueueOccupancy),
			reconciledEngine.Transitions.Reconciles(), reconciledMetrics.AccountingValid, reconciledQueue, reconciledMetrics.Adapter, reconciledEngine.Admissions, reconciledEngine.Transitions)
	}
	if result.maximumQueued >= liveContentionQueueStop || result.maximumOldest >= 2*time.Second || result.framesRejected > 0 ||
		timedEngine.Connection.Integrity+timedEngine.Aggregates.Integrity+timedEngine.Transitions.IntegrityFailure > 0 {
		result.saturated = true
		if result.stopReason == "" {
			result.stopReason = "safety_threshold"
		}
	}
	if result.stopReason == "" {
		result.stopReason = "none"
	}

	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	return result
}

func runLegacyContentionProbe(run *Runtime, baseline liveDiagnosticBaseline, stop <-chan struct{}, done chan<- liveContentionProbeResult, safety chan<- string) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	result := liveContentionProbeResult{}
	for {
		select {
		case <-stop:
			done <- result
			return
		case <-ticker.C:
			sample := captureLiveDiagnosticSample(run, baseline, 0)
			result.maximumQueued = max(result.maximumQueued, sample.CurrentQueuedFrames, sample.MaximumQueuedFrames)
			result.maximumOldest = max(result.maximumOldest, time.Duration(sample.OldestLiveFrameNanoseconds))
			switch {
			case sample.RejectedCapacity+sample.RejectedReceipt+sample.RejectedOversize+sample.RejectedGate > 0:
				result.stopReason = "probe_frame_rejection"
			case sample.IngressIntegrity > 0:
				result.stopReason = "probe_ingress_terminal"
			case sample.CurrentQueuedFrames >= liveContentionQueueStop:
				result.stopReason = "probe_queue_safety_stop"
			}
			if result.stopReason != "" {
				select {
				case safety <- result.stopReason:
				default:
				}
			}
		}
	}
}

func liveContentionFrameRejections(current, baseline massive.LiveQueueAccounting) uint64 {
	return current.FramesRejectedOversize - baseline.FramesRejectedOversize + current.FramesRejectedCapacity - baseline.FramesRejectedCapacity +
		current.FramesRejectedReceipt - baseline.FramesRejectedReceipt + current.FramesRejectedGateOrClose - baseline.FramesRejectedGateOrClose
}

func liveContentionOutcome(result liveContentionResult) string {
	if result.saturated || !result.tailDrained {
		return "saturated"
	}
	return "kept_up"
}

func waitUntil(target time.Time) {
	if delay := time.Until(target); delay > 0 {
		timer := time.NewTimer(delay)
		<-timer.C
	}
}

type liveContentionServer struct {
	server *httptest.Server
	start  chan time.Time
	done   chan error
	sent   atomic.Uint64
	once   sync.Once
}

func newLiveContentionServer(t *testing.T, frames []string) *liveContentionServer {
	t.Helper()
	result := &liveContentionServer{start: make(chan time.Time, 1), done: make(chan error, 1)}
	result.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		result.once.Do(func() {
			result.serve(request, writer, frames)
		})
	}))
	return result
}

func (s *liveContentionServer) serve(request *http.Request, writer http.ResponseWriter, frames []string) {
	ctx := request.Context()
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		s.done <- err
		return
	}
	defer connection.CloseNow()
	write := func(value string) error { return connection.Write(ctx, websocket.MessageText, []byte(value)) }
	if err := write(`[{"ev":"status","status":"connected"}]`); err != nil {
		s.done <- err
		return
	}
	if _, _, err := connection.Read(ctx); err != nil {
		s.done <- err
		return
	}
	if err := write(`[{"ev":"status","status":"auth_success"}]`); err != nil {
		s.done <- err
		return
	}
	if _, raw, err := connection.Read(ctx); err != nil || !strings.Contains(string(raw), "A.*") {
		if err == nil {
			err = errors.New("aggregate subscription was not received")
		}
		s.done <- err
		return
	}
	if err := write(`[{"ev":"status","status":"success"}]`); err != nil {
		s.done <- err
		return
	}
	plannedStart := <-s.start
	frameIndex := 0
	intervalStart := time.Duration(0)
	for _, interval := range liveContentionIntervals {
		for index := 0; index < interval.frames; index++ {
			target := plannedStart.Add(intervalStart + time.Duration(index+1)*interval.duration/time.Duration(interval.frames))
			waitUntil(target)
			if err := write(frames[frameIndex]); err != nil {
				s.done <- err
				return
			}
			s.sent.Add(1)
			frameIndex++
		}
		intervalStart += interval.duration
	}
	if frameIndex != liveContentionFrames || intervalStart != liveContentionDuration {
		s.done <- errors.New("paced stream manifest changed")
		return
	}
	s.done <- nil
	<-ctx.Done()
}
