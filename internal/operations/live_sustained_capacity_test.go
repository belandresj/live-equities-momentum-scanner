package operations

import (
	"context"
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
)

const (
	liveSustainedCapacityFPS      = 216
	liveSustainedCapacityDuration = 60 * time.Second
	liveSustainedCapacityFrames   = liveSustainedCapacityFPS * int(liveSustainedCapacityDuration/time.Second)
)

// TestLiveSustainedAggregateCapacity proves the credential-free C5-to-engine
// path can consume the required 1.5x observed live rate for sixty paced
// seconds without sustained queue growth. The retained-tail evaluator cost is
// proved separately in engine; this test isolates normalization, ordered
// admission/completion, ordinary aggregate work, and the production queue.
func TestLiveSustainedAggregateCapacity(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in sixty-second sustained live aggregate capacity proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	symbols := make([]string, liveContentionPopulation)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	now := binding.SessionStart().Add(8*time.Hour + 2*time.Minute).Truncate(time.Second)
	frames := make([]string, liveSustainedCapacityFrames)
	for index := range frames {
		window := now.Add(-time.Minute).Add(time.Duration(index/liveSustainedCapacityFPS) * time.Second)
		first := (2 * index) % len(symbols)
		frames[index] = "[" + capacityAggregateJSON(symbols[first], window, 10, 11) + "," +
			capacityAggregateJSON(symbols[(first+1)%len(symbols)], window, 10, 11) + "]"
	}
	server := newUniformPacedLiveServer(t, frames, liveSustainedCapacityDuration)
	defer server.server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(server.server.URL, "http"), Credential: "fixture-credential",
		Queue: massive.LiveQueueConfig{FrameSlots: massive.MaximumLiveFrameSlots, MaxFrameBytes: 8 << 20, TotalFrameBytes: massive.MaximumLiveQueueBytes},
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	run, err := New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	if delivered, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), started); deliverErr != nil || delivered.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection attempt=%+v err=%v", delivered, deliverErr)
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

	deliveryCtx, cancelDelivery := context.WithCancel(ctx)
	deliveryDone := make(chan error, 1)
	go func() {
		for {
			started := time.Now()
			result, ok, deliverErr := attempt.DeliverNextToEngine(deliveryCtx, run.Engine())
			if ok {
				run.observeDelivery(started, result)
			}
			if deliverErr != nil || !ok {
				deliveryDone <- deliverErr
				return
			}
		}
	}()

	plannedStart := time.Now().Add(100 * time.Millisecond)
	server.start <- plannedStart
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var highFrames uint64
	var maximumOldest time.Duration
	var midpointQueued uint64
	for {
		select {
		case sendErr := <-server.done:
			if sendErr != nil {
				t.Fatalf("paced fake WebSocket: %v", sendErr)
			}
			goto sent
		case <-ticker.C:
			queue := attempt.QueueAccounting()
			highFrames = max(highFrames, queue.FramesQueued)
			maximumOldest = max(maximumOldest, queue.OldestWaitingFrameAge)
			if midpointQueued == 0 && time.Now().After(plannedStart.Add(liveSustainedCapacityDuration/2)) {
				midpointQueued = queue.FramesQueued
			}
			if liveContentionFrameRejections(queue, queueBaseline) != 0 || queue.FramesQueued > 1024 || queue.OldestWaitingFrameAge > 250*time.Millisecond {
				t.Fatalf("capacity safety boundary queue=%+v high=%d oldest=%s", queue, highFrames, maximumOldest)
			}
		case <-ctx.Done():
			t.Fatal("sustained paced stream timed out")
		}
	}

sent:
	drainDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(drainDeadline) {
		queue := attempt.QueueAccounting()
		highFrames = max(highFrames, queue.FramesQueued)
		maximumOldest = max(maximumOldest, queue.OldestWaitingFrameAge)
		if queue.FramesDispositioned-queueBaseline.FramesDispositioned == uint64(liveSustainedCapacityFrames) && queue.FramesQueued == 0 && queue.FramesClassifying == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	timedQueue := attempt.QueueAccounting()
	timedEngine := run.Engine().ObserveOperational()
	finalQueued := timedQueue.FramesQueued
	framesDispositioned := timedQueue.FramesDispositioned - queueBaseline.FramesDispositioned
	framesRejected := liveContentionFrameRejections(timedQueue, queueBaseline)
	consumed := timedEngine.Aggregates.Consumed - engineBaseline.Aggregates.Consumed

	closeCommand := massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 101, Cause: massive.CloseControlledStop}
	if err := attempt.Close(closeCommand); err != nil {
		t.Fatal(err)
	}
	cancelDelivery()
	select {
	case <-deliveryDone:
	case <-time.After(10 * time.Second):
		t.Fatal("sustained delivery loop did not join")
	}
	run.closeAndDrain(attempt, closeCommand.CommandToken, closeCommand.Cause)
	metrics := run.Metrics()
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}

	t.Logf("SUSTAINED_CAPACITY ingress_fps=%d duration=%s frames=%d dispositioned=%d aggregates=%d midpoint_queue=%d final_queue=%d high_queue=%d maximum_oldest=%s rejected=%d",
		liveSustainedCapacityFPS, liveSustainedCapacityDuration, server.sent.Load(), framesDispositioned, consumed,
		midpointQueued, finalQueued, max(highFrames, metrics.QueueHighFrames), maximumOldest, framesRejected)
	if server.sent.Load() != uint64(liveSustainedCapacityFrames) || framesDispositioned != uint64(liveSustainedCapacityFrames) || consumed != uint64(2*liveSustainedCapacityFrames) ||
		framesRejected != 0 || finalQueued != 0 || finalQueued > midpointQueued || max(highFrames, metrics.QueueHighFrames) > 1024 ||
		maximumOldest > 250*time.Millisecond || !metrics.AccountingValid || !metrics.LiveQueue.Reconciles() || !metrics.Adapter.Reconciles() {
		t.Fatalf("sustained capacity acceptance failed queue=%+v engine=%+v", timedQueue, timedEngine.Aggregates)
	}
}

type uniformPacedLiveServer struct {
	server   *httptest.Server
	start    chan time.Time
	done     chan error
	sent     atomic.Uint64
	once     sync.Once
	duration time.Duration
}

func newUniformPacedLiveServer(t *testing.T, frames []string, duration time.Duration) *uniformPacedLiveServer {
	t.Helper()
	result := &uniformPacedLiveServer{start: make(chan time.Time, 1), done: make(chan error, 1), duration: duration}
	result.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		result.once.Do(func() { result.serve(request, writer, frames) })
	}))
	return result
}

func (s *uniformPacedLiveServer) serve(request *http.Request, writer http.ResponseWriter, frames []string) {
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
	for index, frame := range frames {
		waitUntil(plannedStart.Add(time.Duration(index+1) * s.duration / time.Duration(len(frames))))
		if err := write(frame); err != nil {
			s.done <- err
			return
		}
		s.sent.Add(1)
	}
	s.done <- nil
	<-ctx.Done()
}
