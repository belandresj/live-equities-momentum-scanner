package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
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

// TestC8LOAD01CurrentHostMixedLoad is P-C8-LOAD. It is current-host evidence,
// not an SLA: the real C5 queue transports a deterministic 6,000-symbol
// aggregate/correction/TQ-deferred stream into the sole engine owner.
func TestC8LOAD01CurrentHostMixedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("C8 controlled-load acceptance proof")
	}
	const (
		population        = 6_000
		initialRecords    = 100
		corrections       = 20
		exactDuplicates   = 20
		engineRejections  = 20
		consumerDeferred  = 60
		itemsPerFrame     = 100
		trialTimeout      = 5 * time.Minute
		maximumHeapGrowth = 512 << 20
	)
	ctx, cancel := context.WithTimeout(context.Background(), trialTimeout)
	defer cancel()
	symbols := make([]string, population)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	now := binding.SessionStart().Add(20 * time.Minute)
	target := now.Add(-4 * time.Second).Truncate(time.Second)
	var adapterNanos atomic.Int64
	adapterNanos.Store(target.UnixNano())
	window := target.Add(-time.Second)
	frames := capacityFrames(symbols, initialRecords, window, corrections, exactDuplicates, consumerDeferred, itemsPerFrame)
	wantFrames := (initialRecords + corrections + exactDuplicates + consumerDeferred) / itemsPerFrame
	if len(binding.UniverseSymbols()) != population || len(frames) != wantFrames || initialRecords <= 0 || initialRecords > population || corrections > 961 ||
		(initialRecords+corrections+exactDuplicates+consumerDeferred)%itemsPerFrame != 0 {
		t.Fatalf("load manifest population=%d frames=%d/%d initial=%d corrections=%d duplicates=%d rejected=%d deferred=%d", len(binding.UniverseSymbols()), len(frames), wantFrames, initialRecords, corrections, exactDuplicates, engineRejections, consumerDeferred)
	}

	releaseFrames := make(chan struct{})
	server := capacityWebSocketServer(t, releaseFrames, frames)
	defer server.Close()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: endpoint, Credential: "fixture-credential", Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: func() time.Time { return time.Unix(0, adapterNanos.Load()).UTC() }})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.SampleCadence = 5 * time.Minute // the proof drives its exact timer after the fixed stream
	run, err := New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	baseline := run.Metrics()
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := massive.DeliverToEngine(ctx, run.Engine(), started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection attempt=%+v err=%v", result, err)
	}
	handshake, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		if result, err := massive.DeliverToEngine(ctx, run.Engine(), delivery); err != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("handshake=%+v err=%v", result, err)
		}
	}
	adapterNanos.Store(now.UnixNano())
	completeCapacityHydration(t, ctx, run, attempt, binding, now)
	run.setLiveSources(attempt, adapter)
	close(releaseFrames)

	startedLoad := time.Now()
	codes := map[engine.DispositionCode]uint64{}
	for item := 0; item < initialRecords+corrections+exactDuplicates+consumerDeferred; item++ {
		if item%itemsPerFrame == 0 {
			_ = run.Metrics() // sample the real queue once per source frame
		}
		startedDelivery := time.Now()
		result, ok, err := attempt.DeliverNextToEngine(ctx, run.Engine())
		if err != nil || !ok {
			t.Fatalf("delivery item=%d ok=%v err=%v", item, ok, err)
		}
		run.observeDelivery(startedDelivery, result)
		if result.ConsumerDeferred {
			continue
		}
		codes[result.AggregateDisposition.Code]++
	}
	settleCtx, cancelSettle := context.WithTimeout(ctx, 10*time.Millisecond)
	if _, ok, err := attempt.DeliverNextToEngine(settleCtx, run.Engine()); err != nil || ok {
		t.Fatalf("frame settlement ok=%v err=%v", ok, err)
	}
	cancelSettle()
	for index := 0; index < engineRejections; index++ {
		input := capacityAggregate(binding, "UNKNOWN", window, 10, uint64(wantFrames+index+1))
		startedDelivery := time.Now()
		admission, completion := run.Engine().AdmitAggregate(ctx, input)
		if admission != engine.AdmissionAdmitted || completion == nil {
			t.Fatalf("rejection admission=%s", admission)
		}
		disposition := <-completion
		run.observeDelivery(startedDelivery, massive.EngineDeliveryResult{Admission: admission, AggregateDisposition: disposition})
		codes[disposition.Code]++
	}
	if admission, completion := run.Engine().AdmitTimer(ctx); admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTimerApplied {
		t.Fatal("load evaluation timer was not applied")
	}
	elapsed := time.Since(startedLoad)

	if codes[engine.DispositionAggregateInserted] != initialRecords || codes[engine.DispositionAggregateRevised] != corrections ||
		codes[engine.DispositionAggregateExactDuplicate] != exactDuplicates || codes[engine.DispositionAggregateRejected] != engineRejections {
		t.Fatalf("aggregate oracle codes=%v", codes)
	}
	view := run.Engine().ObserveReplayDeterministic()
	if len(view.Canonical) != population || len(view.Canonical[0].Records) != 1 || view.Canonical[0].LatestValues.Close != capacityCorrectionClose(corrections-1) ||
		view.Evaluation.Population.UniverseTotal != population || view.Evaluation.Population.TrustedRankableMark != initialRecords ||
		view.Evaluation.Population.NoPrintThroughT != population-initialRecords || view.Evaluation.KnownRankableCount != initialRecords ||
		view.Evaluation.TotalPassers != 0 || len(view.Evaluation.Rows) != 0 {
		t.Fatalf("ranking/canonical oracle population=%d first=%+v evaluation=%+v", len(view.Canonical), view.Canonical[0], view.Evaluation)
	}
	status := run.Status()
	metrics := run.Metrics()
	wantConsumed := uint64(initialRecords + corrections + exactDuplicates + engineRejections)
	if !status.BackendReady || !status.RankingCurrent || status.TQAvailable || metrics.ConsumerDeferred != consumerDeferred ||
		metrics.Engine.Aggregates.Consumed != wantConsumed || !metrics.AccountingValid || metrics.QueueHighFrames == 0 || metrics.QueueHighFrames > 512 ||
		metrics.LiveQueue.FramesRead != uint64(wantFrames+3) || metrics.LiveQueue.FramesDispositioned != uint64(wantFrames+3) || metrics.QueueCurrentFrames != 0 ||
		metrics.HeapAllocBytes > baseline.HeapAllocBytes+maximumHeapGrowth {
		t.Fatalf("load status=%+v metrics=%+v baseline_heap=%d", status, metrics, baseline.HeapAllocBytes)
	}
	throughput := float64(initialRecords+corrections+exactDuplicates+engineRejections+consumerDeferred) / elapsed.Seconds()
	t.Logf("host=%s/%s go=%s cpus=%d population=%d frames=%d aggregates=%d corrections=%d duplicates=%d rejected=%d tq_deferred=%d elapsed=%s throughput=%.0f_items/s mean_delay=%s max_delay=%s queue_high_frames=%d queue_high_bytes=%d heap_growth=%d goroutine_growth=%d watermark_lag=%s",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU(), population, wantFrames, initialRecords, corrections, exactDuplicates, engineRejections, consumerDeferred, elapsed, throughput,
		metrics.MeanProcessingDelay, metrics.MaxProcessingDelay, metrics.QueueHighFrames, metrics.QueueHighBytes, int64(metrics.HeapAllocBytes)-int64(baseline.HeapAllocBytes), metrics.Goroutines-baseline.Goroutines, metrics.WatermarkLag)

	run.closeAndDrain(ctx, attempt, 190, massive.CloseControlledStop)
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func capacityFrames(symbols []string, initial int, window time.Time, corrections, duplicates, deferred, perFrame int) []string {
	items := make([]string, 0, initial+corrections+duplicates+deferred)
	for index, symbol := range symbols[:initial] {
		closeValue := 11 + float64(index)
		items = append(items, capacityAggregateJSON(symbol, window, 10+float64(index), closeValue))
	}
	for index := 0; index < corrections; index++ {
		items = append(items, capacityAggregateJSON(symbols[0], window, 10, capacityCorrectionClose(index)))
	}
	for index := 0; index < duplicates; index++ {
		items = append(items, capacityAggregateJSON(symbols[0], window, 10, capacityCorrectionClose(corrections-1)))
	}
	for index := 0; index < deferred; index++ {
		if index%2 == 0 {
			items = append(items, fmt.Sprintf(`{"ev":"T","sym":%q,"x":4,"i":%q,"p":10,"s":1,"t":%d}`, symbols[index%len(symbols)], fmt.Sprintf("trade-%d", index), window.UnixMilli()))
		} else {
			items = append(items, fmt.Sprintf(`{"ev":"Q","sym":%q,"t":%d,"bp":10,"ap":10.1}`, symbols[index%len(symbols)], window.UnixMilli()))
		}
	}
	frames := make([]string, 0, len(items)/perFrame)
	for start := 0; start < len(items); start += perFrame {
		frames = append(frames, "["+strings.Join(items[start:start+perFrame], ",")+"]")
	}
	return frames
}

func capacityAggregateJSON(symbol string, window time.Time, openValue, closeValue float64) string {
	return fmt.Sprintf(`{"ev":"A","sym":%q,"s":%d,"e":%d,"o":%g,"h":%g,"l":%g,"c":%g,"v":100,"z":10,"vw":%g}`,
		symbol, window.UnixMilli(), window.Add(time.Second).UnixMilli(), openValue, max(openValue, closeValue)+1, min(openValue, closeValue)-1, closeValue, (openValue+closeValue)/2)
}

func capacityCorrectionClose(index int) float64 { return 20 + float64(index)/1000 }

func capacityAggregate(binding reference.Binding, symbol string, window time.Time, closeValue float64, frame uint64) engine.AggregateInput {
	return engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: symbol,
		WindowStart: window, WindowEnd: window.Add(time.Second), Values: engine.AggregateValues{Open: closeValue, High: closeValue + 1, Low: closeValue - 1, Close: closeValue, Volume: 100, VWAP: closeValue, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
		DeliveryTime: window.Add(time.Second), Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: frame}}
}

func capacityWebSocketServer(t *testing.T, release <-chan struct{}, frames []string) *httptest.Server {
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
		case <-release:
		}
		for _, frame := range frames {
			if !write(frame) {
				return
			}
		}
		<-ctx.Done()
	}))
}

func capacityDurations() massive.OperationalDurations {
	return massive.OperationalDurations{Dial: 5 * time.Second, HandshakeStep: 5 * time.Second, HandshakeTotal: 20 * time.Second, HeartbeatInterval: 10 * time.Minute, HeartbeatDeadline: 5 * time.Second, Write: 5 * time.Second, Close: 5 * time.Second}
}

func completeCapacityHydration(t *testing.T, ctx context.Context, run *Runtime, attempt *massive.LiveAttempt, binding reference.Binding, now time.Time) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 8, RowsPerChunk: 256, MaximumResponseBytes: 64 << 20, MaximumNormalizedRecords: int64(len(binding.UniverseSymbols())) * 57_600, MaximumResidentRecords: 8 * 57_600}
	admission, completion := run.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("capacity hydration plan was not admitted")
	}
	plan := <-completion
	if plan.Code != engine.DispositionHydrationPlanApplied || len(plan.Plan.Requests()) != len(binding.UniverseSymbols()) {
		t.Fatalf("capacity hydration plan=%+v", plan)
	}
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		admission, completed := run.Engine().AdmitHydrationTerminal(ctx, terminal)
		if admission != engine.AdmissionAdmitted || completed == nil {
			t.Fatal("capacity hydration terminal was not admitted")
		}
		result := <-completed
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("capacity hydration terminal=%+v", result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("capacity hydration omitted fence command")
	}
	if err := run.finishFence(ctx, attempt, fence, plan.Plan.End()); err != nil {
		t.Fatalf("capacity hydration fence did not reconcile: %v", err)
	}
	if got := run.Status(); !got.BackendReady || got.Watermark == nil || *got.Watermark != now.Add(-4*time.Second).Truncate(time.Second) {
		t.Fatalf("capacity bootstrap status=%+v", got)
	}
}

func capacityBinding(t *testing.T, symbols []string) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			offset, _ := strconv.Atoi(request.URL.Query().Get("cursor"))
			end := min(offset+1000, len(symbols))
			records := make([]map[string]any, end-offset)
			for index, symbol := range symbols[offset:end] {
				records[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			response := map[string]any{"status": "OK", "count": len(records), "results": records}
			if end < len(symbols) {
				query := request.URL.Query()
				query.Del("apiKey")
				query.Set("cursor", strconv.Itoa(end))
				response["next_url"] = server.URL + request.URL.Path + "?" + query.Encode()
			}
			_ = json.NewEncoder(writer).Encode(response)
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			rows := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				rows[index] = map[string]any{"T": symbol, "c": 10 + index, "t": facts.PriorRegularClose.UnixMilli()}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(rows), "results": rows})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "fixture", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "fixture", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil || !slices.Equal(binding.UniverseSymbols(), symbols) {
		t.Fatalf("capacity binding: %v", err)
	}
	return binding
}
