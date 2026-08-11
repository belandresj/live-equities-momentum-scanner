package operations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

const (
	hydrationContentionPopulation = 6_000
	hydrationContentionChunkRows  = 256
	hydrationContentionChunksEach = 100
	hydrationContentionDuration   = 60 * time.Second
	// The sealed artifact admits approximately 29,600 chunks over 147 seconds:
	// preserve that measured ~200 chunk/s owner load during the focused phase.
	hydrationContentionFeedTime      = hydrationContentionDuration
	hydrationContentionSparseChunks  = 4
	hydrationContentionDenseChunks   = 7
	hydrationContentionMeasuredDense = 2
	hydrationContentionBurstChunks   = 512
	hydrationContentionTailChunks    = hydrationContentionSparseChunks + hydrationContentionDenseChunks
)

type hydrationContentionResult struct {
	rate, chunks                                    int
	rows                                            uint64
	framesRead, framesDispositioned, framesRejected uint64
	maximumQueued, queuedAtStop                     uint64
	chunkMean, chunkMaximum, drain                  time.Duration
	deliveryMean, deliveryMaximum                   time.Duration
	gcCycles                                        uint32
	gcPause                                         time.Duration
	heapDelta                                       int64
	accounting                                      bool
}

// TestHydrationLiveContention is the short-cycle diagnostic for the cached
// hydration fence correction. It uses the real adapter queue, WebSocket
// normalizer, DeliverNextToEngine, sole engine owner, production hydration
// plan/admissions, and 256-row historical chunks. It proves contention
// throughput and accounting, not hydration completion or readiness.
func TestHydrationLiveContention(t *testing.T) {
	if testing.Short() || os.Getenv("HYDRATION_LIVE_CONTENTION") != "1" {
		t.Skip("set HYDRATION_LIVE_CONTENTION=1 to run the focused contention diagnostic")
	}
	for _, rate := range []int{1, 2} {
		t.Run(fmt.Sprintf("%dx", rate), func(t *testing.T) {
			result := runHydrationLiveContention(t, rate)
			t.Logf("hydration_contention rate=%dx duration=%s chunks=%d rows=%d chunk_mean=%s chunk_max=%s frames_read=%d dispositioned=%d rejected=%d queue_high=%d queue_at_stop=%d drain=%s delivery_mean=%s delivery_max=%s gc_cycles=%d gc_pause=%s heap_delta=%d accounting=%t",
				result.rate, hydrationContentionDuration, result.chunks, result.rows, result.chunkMean, result.chunkMaximum,
				result.framesRead, result.framesDispositioned, result.framesRejected, result.maximumQueued, result.queuedAtStop,
				result.drain, result.deliveryMean, result.deliveryMaximum, result.gcCycles, result.gcPause, result.heapDelta, result.accounting)
			if !result.accounting || result.framesRejected != 0 {
				t.Fatalf("contention accounting/rejection = %t/%d", result.accounting, result.framesRejected)
			}
		})
	}
}

func runHydrationLiveContention(t *testing.T, rate int) hydrationContentionResult {
	t.Helper()
	tailMode := os.Getenv("HYDRATION_LIVE_CONTENTION_TAIL") == "1"
	timeout := 75 * time.Second
	if tailMode {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	symbols := make([]string, hydrationContentionPopulation)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	boundary := binding.SessionEnd().Add(-(2*time.Hour + 45*time.Minute))
	clock := &cachedFenceClock{now: boundary}
	server := newCachedFenceServer(t, symbols[:64], boundary, rate)
	server.catchUp = true
	defer server.server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(server.server.URL, "http"), Credential: "contention-fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: clock.read})
	if err != nil {
		t.Fatal(err)
	}
	run, err := New(ctx, binding, DefaultConfig(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 700, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	if delivered, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), started); deliverErr != nil || delivered.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection start=%+v err=%v", delivered, deliverErr)
	}
	handshake, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		got, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), delivery)
		if deliverErr != nil || (got.ControlDisposition.Code != engine.DispositionConnectionControlApplied && got.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("handshake=%+v err=%v", got, deliverErr)
		}
	}
	run.setLiveSources(attempt, adapter)
	queueBase := attempt.QueueAccounting()
	engineBase := run.Engine().ObserveOperational()
	deliveryBase, deliveryNanosBase, deliveryMaxBase := run.deliveryCount.Load(), run.deliveryTotalNanos.Load(), run.deliveryMaxNanos.Load()

	planAdmission, planCompletion := run.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1,
		BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(),
		Budgets: engine.HydrationPlanBudgets{Workers: 2, RowsPerChunk: hydrationContentionChunkRows, MaximumResponseBytes: 2 << 30,
			MaximumNormalizedRecords: int64(len(symbols)) * 57_600, MaximumResidentRecords: 2 * 57_600}})
	if planAdmission != engine.AdmissionAdmitted || planCompletion == nil {
		t.Fatal("hydration plan admission")
	}
	plan := <-planCompletion
	if plan.Code != engine.DispositionHydrationPlanApplied || len(plan.Plan.Requests()) != len(symbols) {
		t.Fatalf("hydration plan=%+v requests=%d", plan, len(plan.Plan.Requests()))
	}
	requests := plan.Plan.Requests()
	normalizationFixture := []byte(`{"kind":"aggregate","symbol":"FIXTURE","window_start":"2026-08-07T20:00:00Z","window_end":"2026-08-07T20:00:01Z","open":10,"high":10.1,"low":9.9,"close":10,"volume":100,"vwap":10,"average_trade_size":10,"ats_provenance":"rest_floor_volume_over_transactions"}`)
	admitChunk := func(admitCtx context.Context, token engine.HydrationRequestToken, ordinal, totalChunks int, base time.Time, spacing time.Duration, normalize bool) (time.Duration, error) {
		rowOffset := int64(ordinal * hydrationContentionChunkRows)
		rowsInput := make([]engine.HydrationRow, hydrationContentionChunkRows)
		var buildErr error
		build := func() {
			for rowIndex := range rowsInput {
				if normalize {
					var kind struct {
						Kind string `json:"kind"`
					}
					var value cachedArtifactLine
					if json.Unmarshal(normalizationFixture, &kind) != nil || json.Unmarshal(normalizationFixture, &value) != nil || kind.Kind != "aggregate" {
						buildErr = errors.New("historical normalization fixture")
						return
					}
					if _, err := time.Parse(time.RFC3339Nano, value.WindowStart); err != nil {
						buildErr = err
						return
					}
					if _, err := time.Parse(time.RFC3339Nano, value.WindowEnd); err != nil {
						buildErr = err
						return
					}
				}
				window := base.Add(time.Duration(rowIndex) * spacing)
				seed := ordinal*hydrationContentionChunkRows + rowIndex
				price := 10 + float64(seed%1000)/1000
				volume := 100 + float64((seed*13)%10_000)/7
				row, rowErr := engine.NewHydrationRow(token.Symbol(), window, window.Add(time.Second), engine.AggregateValues{
					Open: price, High: price + .2, Low: price - .2, Close: price + .05, Volume: volume, VWAP: price,
					AverageTradeSize: 7 + int64((seed*17)%93), ATSProvenance: engine.ATSRESTFloorVolumeOverTrades})
				if rowErr != nil {
					buildErr = rowErr
					return
				}
				rowsInput[rowIndex] = row
			}
		}
		if normalize {
			trace.WithRegion(admitCtx, "historical_normalization", build)
		} else {
			build()
		}
		if buildErr != nil {
			return 0, buildErr
		}
		input, inputErr := engine.NewHydrationChunkInput(token, token.ResultID(), ordinal, totalChunks, rowOffset,
			int64(totalChunks*hydrationContentionChunkRows), rowsInput)
		if inputErr != nil {
			return 0, inputErr
		}
		started := time.Now()
		admission, completion := run.Engine().AdmitHydrationChunk(admitCtx, input)
		if admission != engine.AdmissionAdmitted || completion == nil {
			return 0, fmt.Errorf("chunk admission=%s", admission)
		}
		got := <-completion
		elapsed := time.Since(started)
		if got.Code != engine.DispositionHydrationChunkApplied {
			return elapsed, fmt.Errorf("chunk disposition=%s/%s", got.Code, got.Reason)
		}
		return elapsed, nil
	}
	if tailMode {
		preloadStarted := time.Now()
		preloadChunks := 0
		for ordinal := 0; ordinal < hydrationContentionSparseChunks; ordinal++ {
			for _, token := range requests[:64] {
				base := token.Start().Add(time.Duration(ordinal*hydrationContentionChunkRows) * 30 * time.Second)
				if _, err := admitChunk(ctx, token, ordinal, hydrationContentionTailChunks, base, 30*time.Second, false); err != nil {
					t.Fatalf("tail preload ordinal=%d symbol=%s: %v", ordinal, token.Symbol(), err)
				}
				preloadChunks++
			}
		}
		for denseOrdinal := 0; denseOrdinal < hydrationContentionDenseChunks-hydrationContentionMeasuredDense; denseOrdinal++ {
			for requestIndex, token := range requests {
				ordinal, totalChunks := denseOrdinal, hydrationContentionDenseChunks
				if requestIndex < 64 {
					ordinal, totalChunks = hydrationContentionSparseChunks+denseOrdinal, hydrationContentionTailChunks
				}
				base := token.End().Add(-time.Duration(hydrationContentionDenseChunks*hydrationContentionChunkRows) * time.Second).
					Add(time.Duration(denseOrdinal*hydrationContentionChunkRows) * time.Second)
				if _, err := admitChunk(ctx, token, ordinal, totalChunks, base, time.Second, false); err != nil {
					t.Fatalf("tail preload dense=%d symbol=%s: %v", denseOrdinal, token.Symbol(), err)
				}
				preloadChunks++
			}
		}
		t.Logf("hydration_contention_tail_preload chunks=%d rows=%d elapsed=%s", preloadChunks,
			preloadChunks*hydrationContentionChunkRows, time.Since(preloadStarted))
	}
	clock.set(boundary.Add(5 * time.Second))
	engineBase = run.Engine().ObserveOperational()

	var memoryBefore runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	deliveryCtx, stopDelivery := context.WithCancel(ctx)
	defer stopDelivery()
	hydrationCtx, stopHydration := context.WithCancel(ctx)
	defer stopHydration()
	deliveryDone := make(chan error, 1)
	go pprof.Do(deliveryCtx, pprof.Labels("phase", "live_delivery"), func(labelCtx context.Context) {
		trace.WithRegion(labelCtx, "live_delivery", func() {
			for {
				started := time.Now()
				delivery, ok, deliveryErr := attempt.DeliverNextToEngine(labelCtx, run.Engine())
				if ok {
					run.observeDelivery(started, delivery)
				}
				if deliveryErr != nil || !ok {
					deliveryDone <- deliveryErr
					return
				}
			}
		})
	})

	var chunks atomic.Int64
	var rows atomic.Uint64
	var chunkNanos atomic.Int64
	var chunkMaximum atomic.Int64
	hydrationDone := make(chan error, 1)
	measuredStart := time.Now()
	go pprof.Do(hydrationCtx, pprof.Labels("phase", "historical_chunks"), func(labelCtx context.Context) {
		trace.WithRegion(labelCtx, "historical_chunks", func() {
			loopChunks := hydrationContentionChunksEach
			if tailMode {
				loopChunks = hydrationContentionMeasuredDense
			}
			for loopOrdinal := 0; loopOrdinal < loopChunks; loopOrdinal++ {
				for requestIndex := range requests {
					token := requests[requestIndex]
					select {
					case <-labelCtx.Done():
						hydrationDone <- nil
						return
					default:
					}
					if tailMode {
						position := loopOrdinal*len(requests) + requestIndex
						burstStart := position / hydrationContentionBurstChunks * hydrationContentionBurstChunks
						target := measuredStart.Add(time.Duration(burstStart) * hydrationContentionFeedTime / time.Duration(loopChunks*len(requests)))
						if wait := time.Until(target); wait > 0 {
							timer := time.NewTimer(wait)
							select {
							case <-labelCtx.Done():
								timer.Stop()
								hydrationDone <- nil
								return
							case <-timer.C:
							}
						}
					}
					ordinal, totalChunks := loopOrdinal, hydrationContentionChunksEach
					base := token.Start().Add(time.Duration(loopOrdinal*hydrationContentionChunkRows) * time.Second)
					if tailMode {
						denseOrdinal := hydrationContentionDenseChunks - hydrationContentionMeasuredDense + loopOrdinal
						ordinal, totalChunks = denseOrdinal, hydrationContentionDenseChunks
						if requestIndex < 64 {
							ordinal, totalChunks = hydrationContentionSparseChunks+denseOrdinal, hydrationContentionTailChunks
						}
						base = token.End().Add(-time.Duration(hydrationContentionDenseChunks*hydrationContentionChunkRows) * time.Second).
							Add(time.Duration(denseOrdinal*hydrationContentionChunkRows) * time.Second)
					}
					elapsed, chunkErr := admitChunk(labelCtx, token, ordinal, totalChunks, base, time.Second, true)
					if chunkErr != nil {
						hydrationDone <- chunkErr
						return
					}
					chunks.Add(1)
					rows.Add(hydrationContentionChunkRows)
					chunkNanos.Add(int64(elapsed))
					for prior := chunkMaximum.Load(); int64(elapsed) > prior && !chunkMaximum.CompareAndSwap(prior, int64(elapsed)); prior = chunkMaximum.Load() {
					}
				}
			}
			hydrationDone <- nil
		})
	})

	close(server.start)
	measuredChunkTarget := hydrationContentionChunksEach * len(requests)
	deadline := time.Now().Add(hydrationContentionDuration)
	if tailMode {
		measuredChunkTarget = hydrationContentionMeasuredDense * len(requests)
		// The producer's final 512-chunk burst is scheduled just before the
		// 60-second feed boundary. Give that already-scheduled bounded work a
		// small completion margin so the full-retention claim cannot pass with a
		// silently truncated final burst.
		deadline = deadline.Add(5 * time.Second)
	}
	maximumQueued := uint64(0)
	for time.Now().Before(deadline) {
		queue := attempt.QueueAccounting()
		maximumQueued = max(maximumQueued, queue.FramesQueued)
		if liveContentionFrameRejections(queue, queueBase) != 0 || int(chunks.Load()) == measuredChunkTarget {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopHydration()
	select {
	case err := <-hydrationDone:
		if err != nil {
			t.Logf("hydration producer stopped: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("hydration producer did not join")
	}
	close(server.pause)
	select {
	case <-server.done:
	case <-time.After(5 * time.Second):
		t.Fatal("WebSocket producer did not stop")
	}
	queueAtStop := attempt.QueueAccounting()
	drainStarted := time.Now()
	for time.Since(drainStarted) < 5*time.Second {
		queue := attempt.QueueAccounting()
		maximumQueued = max(maximumQueued, queue.FramesQueued)
		if queue.FramesQueued == 0 && queue.FramesClassifying == 0 && queue.FramesRead-queueBase.FramesRead == queue.FramesDispositioned-queueBase.FramesDispositioned {
			break
		}
		time.Sleep(time.Millisecond)
	}
	timedQueue := attempt.QueueAccounting()
	timedEngine := run.Engine().ObserveOperational()
	var memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryAfter)

	closeCommand := massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 701, Cause: massive.CloseControlledStop}
	_ = attempt.Close(closeCommand)
	stopDelivery()
	select {
	case <-deliveryDone:
	case <-time.After(10 * time.Second):
		t.Fatal("delivery loop did not join")
	}
	run.closeAndDrain(attempt, closeCommand.CommandToken, closeCommand.Cause)
	metrics := run.Metrics()
	reconciledQueue := metrics.LiveQueue
	reconciledEngine := metrics.Engine

	chunkCount := int(chunks.Load())
	result := hydrationContentionResult{rate: rate, chunks: chunkCount, rows: rows.Load(), framesRead: timedQueue.FramesRead - queueBase.FramesRead,
		framesDispositioned: timedQueue.FramesDispositioned - queueBase.FramesDispositioned, framesRejected: liveContentionFrameRejections(timedQueue, queueBase),
		maximumQueued: maximumQueued, queuedAtStop: queueAtStop.FramesQueued, chunkMaximum: time.Duration(chunkMaximum.Load()), drain: time.Since(drainStarted),
		deliveryMaximum: time.Duration(run.deliveryMaxNanos.Load()), gcCycles: memoryAfter.NumGC - memoryBefore.NumGC,
		gcPause: time.Duration(memoryAfter.PauseTotalNs - memoryBefore.PauseTotalNs), heapDelta: int64(memoryAfter.HeapAlloc) - int64(memoryBefore.HeapAlloc)}
	if chunkCount > 0 {
		result.chunkMean = time.Duration(chunkNanos.Load() / int64(chunkCount))
	}
	if tailMode && chunkCount != measuredChunkTarget {
		t.Fatalf("full-retention measured chunks=%d rows=%d want chunks=%d rows=%d", chunkCount, result.rows,
			measuredChunkTarget, measuredChunkTarget*hydrationContentionChunkRows)
	}
	deliveryCount := run.deliveryCount.Load() - deliveryBase
	if deliveryCount > 0 {
		result.deliveryMean = time.Duration((run.deliveryTotalNanos.Load() - deliveryNanosBase) / deliveryCount)
	}
	if result.deliveryMaximum <= time.Duration(deliveryMaxBase) {
		result.deliveryMaximum = 0
	}
	result.accounting = reconciledQueue.Reconciles() && metrics.Adapter.Reconciles() && reconciledEngine.Admissions.Reconciles(reconciledEngine.QueueOccupancy) &&
		reconciledEngine.Transitions.Reconciles() && reconciledEngine.Aggregates.Reconciles() && metrics.AccountingValid &&
		timedEngine.Hydration.Rows.Consumed-engineBase.Hydration.Rows.Consumed == result.rows &&
		reconciledQueue.FramesRead-queueBase.FramesRead == reconciledQueue.FramesDispositioned-queueBase.FramesDispositioned+reconciledQueue.FramesFenced-queueBase.FramesFenced+
			liveContentionFrameRejections(reconciledQueue, queueBase)

	shutdown, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	return result
}
