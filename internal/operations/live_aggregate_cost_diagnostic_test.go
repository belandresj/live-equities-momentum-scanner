package operations

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	liveAggregateCostPopulation = 6_000
	liveAggregateCostFrames     = 400
	liveAggregateCostPerFrame   = 2
)

type liveAggregateCostPhase struct {
	Elapsed       time.Duration
	Frames        uint64
	Aggregates    uint64
	AllocationB   uint64
	Allocations   uint64
	Reconciled    uint64
	NotReconciled uint64
}

// TestLiveAggregateCostAttribution separates pure C5 frame normalization from
// the identical stream's adapter-to-engine path. It is a deterministic,
// credential-free diagnostic and makes no throughput acceptance claim.
func TestLiveAggregateCostAttribution(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in live aggregate cost-attribution diagnostic")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()

	symbols := make([]string, liveAggregateCostPopulation)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	now := binding.SessionStart().Add(20 * time.Minute)
	window := now.Add(-time.Second).Truncate(time.Second)
	frames := liveAggregateCostFramesFor(symbols, window)
	if len(binding.UniverseSymbols()) != liveAggregateCostPopulation || len(frames) != liveAggregateCostFrames {
		t.Fatalf("manifest population=%d frames=%d", len(binding.UniverseSymbols()), len(frames))
	}

	normalization := measureLiveAggregateCost(func() liveAggregateCostPhase {
		return normalizeLiveAggregateCost(binding, frames, now)
	})
	endToEnd := runLiveAggregateCostEndToEnd(t, ctx, binding, frames, now)

	for _, measured := range []struct {
		name  string
		phase liveAggregateCostPhase
	}{{"normalization", normalization}, {"end_to_end", endToEnd}} {
		name, phase := measured.name, measured.phase
		if phase.Frames != liveAggregateCostFrames || phase.Aggregates != liveAggregateCostFrames*liveAggregateCostPerFrame ||
			phase.Reconciled != liveAggregateCostFrames || phase.NotReconciled != 0 {
			t.Fatalf("%s counts=%+v", name, phase)
		}
		t.Logf("phase=%s elapsed=%s frames_per_s=%.2f aggregates_per_s=%.2f allocation_bytes=%d allocations=%d frames=%d aggregates=%d reconciled=%d unreconciled=%d",
			name, phase.Elapsed, float64(phase.Frames)/phase.Elapsed.Seconds(), float64(phase.Aggregates)/phase.Elapsed.Seconds(),
			phase.AllocationB, phase.Allocations, phase.Frames, phase.Aggregates, phase.Reconciled, phase.NotReconciled)
	}
}

func measureLiveAggregateCost(run func() liveAggregateCostPhase) liveAggregateCostPhase {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	result := run()
	result.Elapsed = time.Since(started)
	runtime.ReadMemStats(&after)
	result.AllocationB = after.TotalAlloc - before.TotalAlloc
	result.Allocations = after.Mallocs - before.Mallocs
	return result
}

func normalizeLiveAggregateCost(binding reference.Binding, frames []string, received time.Time) liveAggregateCostPhase {
	result := liveAggregateCostPhase{}
	for index, frame := range frames {
		accounting, consumed := massive.ConsumeLiveFrameForAttribution(massive.LiveFrame{
			Binding: binding, ConnectionEpoch: 1, FrameSequence: uint64(index + 1), ReceivedAt: received, Data: []byte(frame),
		}, nil, massive.LiveNormalizationOptions{})
		result.Frames++
		result.Aggregates += uint64(accounting.NormalizedAggregates)
		if accounting.Reconciles() && consumed == liveAggregateCostPerFrame {
			result.Reconciled++
		} else {
			result.NotReconciled++
		}
	}
	return result
}

func runLiveAggregateCostEndToEnd(t *testing.T, ctx context.Context, binding reference.Binding, frames []string, now time.Time) liveAggregateCostPhase {
	t.Helper()
	release := make(chan struct{})
	server := capacityWebSocketServer(t, release, frames)
	defer server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(server.URL, "http"), Credential: "fixture-credential",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.SampleCadence = 5 * time.Minute
	run, err := New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	if delivered, err := massive.DeliverToEngine(ctx, run.Engine(), started); err != nil || delivered.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection attempt=%+v err=%v", delivered, err)
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
	metricsBaseline := run.Metrics()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	startedPhase := time.Now()
	close(release)

	result := liveAggregateCostPhase{}
	codes := make(map[engine.DispositionCode]uint64)
	for index := 0; index < liveAggregateCostFrames*liveAggregateCostPerFrame; index++ {
		delivery, ok, deliverErr := attempt.DeliverNextToEngine(ctx, run.Engine())
		if deliverErr != nil || !ok {
			t.Fatalf("delivery index=%d ok=%v err=%v", index, ok, deliverErr)
		}
		result.Aggregates++
		codes[delivery.AggregateDisposition.Code]++
	}
	queue := attempt.QueueAccounting()
	metrics := run.Metrics()
	result.Frames = queue.FramesRead - queueBaseline.FramesRead
	if queue.Reconciles() && result.Frames == liveAggregateCostFrames && queue.FramesAdmitted-queueBaseline.FramesAdmitted == liveAggregateCostFrames && queue.FramesQueued == 0 && queue.FramesClassifying == 1 &&
		metrics.AccountingValid && metrics.Engine.Aggregates.Consumed-metricsBaseline.Engine.Aggregates.Consumed == liveAggregateCostFrames*liveAggregateCostPerFrame &&
		codes[engine.DispositionAggregateInserted] == liveAggregateCostFrames*liveAggregateCostPerFrame {
		result.Reconciled = result.Frames
	} else {
		result.NotReconciled = result.Frames
	}
	result.Elapsed = time.Since(startedPhase)
	runtime.ReadMemStats(&after)
	result.AllocationB = after.TotalAlloc - before.TotalAlloc
	result.Allocations = after.Mallocs - before.Mallocs

	run.closeAndDrain(attempt, 101, massive.CloseControlledStop)
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	return result
}

func liveAggregateCostFramesFor(symbols []string, window time.Time) []string {
	frames := make([]string, liveAggregateCostFrames)
	for index := range frames {
		first := index * liveAggregateCostPerFrame
		frames[index] = "[" + capacityAggregateJSON(symbols[first], window, 10, 11) + "," +
			capacityAggregateJSON(symbols[first+1], window, 10, 11) + "]"
	}
	return frames
}
