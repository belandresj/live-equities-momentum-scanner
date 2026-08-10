package operations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/metrics"
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

const (
	diagnosticStable = iota + 1
	diagnosticSaturated
	diagnosticTerminal
)

const (
	diagnosticStopDeadline = iota + 1
	diagnosticStopRejection
	diagnosticStopTerminal
	diagnosticStopQueue
	diagnosticStopOldest
	diagnosticStopAccounting
)

const (
	diagnosticClassLivePath = iota + 1
	diagnosticClassCrossSource
	diagnosticClassWorkerBoundary
	diagnosticClassOtherStartup
)

type liveDiagnosticConfig struct {
	duration, cadence, safetyCadence, joinDeadline time.Duration
	queueStop                                      uint64
	oldestStop                                     time.Duration
}

func productionLiveDiagnosticConfig() liveDiagnosticConfig {
	return liveDiagnosticConfig{duration: 30 * time.Second, cadence: time.Second, safetyCadence: 10 * time.Millisecond,
		joinDeadline: 10 * time.Second, queueStop: 384, oldestStop: 2 * time.Second}
}

// liveDiagnosticSample deliberately contains only bounded numeric values. It
// cannot retain a credential, URL, provider body, raw frame, symbol, or row.
type liveDiagnosticSample struct {
	ElapsedMilliseconds, Workers                                                           int64
	LiveFramesRead, LiveFramesAdmitted, LiveFramesDispositioned, LiveFramesQueued          uint64
	LiveFramesFenced, RejectedCapacity, RejectedReceipt, RejectedOversize, RejectedGate    uint64
	CurrentQueuedFrames, MaximumQueuedFrames                                               uint64
	CurrentQueuedBytes, MaximumQueuedBytes                                                 int64
	OldestLiveFrameNanoseconds                                                             int64
	LiveDeliveryCount, MeanDeliveryDelayNanoseconds, OneSecondMaxDelayNanoseconds          uint64
	OverallMaxDeliveryDelayNanoseconds                                                     uint64
	EngineQueueOccupancy                                                                   int64
	AggregateConsumed, AggregateInserted, AggregateRevised, AggregateRejected              uint64
	AggregateFenced, IngressIntegrity                                                      uint64
	HydrationOpen, HydrationCompleted, HydrationFailed, HydrationCanceled, HydrationFenced uint64
	HydrationRowsConsumed                                                                  uint64
	HydrationRowsPerSecond, AverageCPUCores                                                float64
	HeapAllocBytes, HeapInUseBytes                                                         uint64
	Goroutines                                                                             int64
	AccountingValid                                                                        int64
}

type liveDiagnosticSummary struct {
	Workers, Outcome, StopReason                                   int64
	DurationMilliseconds                                           int64
	LiveFramesPerSecondIn, LiveFramesPerSecondOut                  float64
	MaximumQueuedFrames                                            uint64
	MaximumOldestFrameNanoseconds, MaximumDeliveryDelayNanoseconds int64
	Rejections                                                     uint64
	HydrationRowsPerSecond, AverageCPUCores                        float64
	Samples                                                        []liveDiagnosticSample
}

type liveDiagnosticArtifact struct {
	Schema, Classification int64
	Trials                 []liveDiagnosticSummary
}

type liveDiagnosticTrial struct {
	binding    reference.Binding
	adapter    *massive.LiveAdapter
	hydrator   *massive.HydrationWorker
	workers    int
	clock      func() time.Time
	config     liveDiagnosticConfig
	components LiveComponents
}

type liveDiagnosticBaseline struct {
	metrics       Metrics
	deliveries    uint64
	deliveryNanos uint64
	deliveryMax   uint64
	cpuSeconds    float64
	started       time.Time
}

// TestLiveHydrationThroughputDiagnostic is an explicitly enabled live
// diagnostic. The request's trading date and the current exact-date reference
// caches bind the run; reference resolution itself is cache-only.
func TestLiveHydrationThroughputDiagnostic(t *testing.T) {
	if os.Getenv("LIVE_HYDRATION_DIAGNOSTIC") != "1" {
		t.Skip("set LIVE_HYDRATION_DIAGNOSTIC=1 after authorizing an exact trading date")
	}
	tradingDate := os.Getenv("LIVE_TRADING_DATE")
	credential := os.Getenv("MASSIVE_API_KEY")
	output := os.Getenv("LIVE_DIAGNOSTIC_OUTPUT")
	if tradingDate == "" || credential == "" || output == "" {
		t.Fatal("LIVE_TRADING_DATE, LIVE_DIAGNOSTIC_OUTPUT, and MASSIVE_API_KEY are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	defer cancel()
	output = validatedLiveDiagnosticOutput(t, output)
	binding := cachedLiveDiagnosticBinding(t, ctx, tradingDate, filepath.Join(filepath.Dir(output), "reference"))

	workers := []int{0, 1, 2, 4}
	results := make([]liveDiagnosticSummary, 0, 5)
	for index := 0; index < len(workers); index++ {
		trial := newProviderLiveDiagnosticTrial(t, binding, credential, workers[index])
		result, err := runLiveDiagnosticTrial(ctx, trial)
		if err != nil {
			t.Fatalf("diagnostic trial %d cleanup: %v", workers[index], err)
		}
		results = append(results, result)
		if result.Outcome != diagnosticStable {
			break
		}
		if workers[index] == 4 {
			workers = append(workers, 8)
		}
	}
	classification := classifyLiveDiagnostic(results)
	artifact := liveDiagnosticArtifact{Schema: 1, Classification: int64(classification), Trials: results}
	if err := writeLiveDiagnosticArtifact(output, artifact); err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		t.Logf("workers=%d duration_ms=%d live_in_per_s=%.2f live_out_per_s=%.2f max_queued=%d max_oldest_ns=%d rejections=%d max_delivery_ns=%d hydration_rows_per_s=%.2f avg_cpu_cores=%.3f outcome=%s stop=%s",
			result.Workers, result.DurationMilliseconds, result.LiveFramesPerSecondIn, result.LiveFramesPerSecondOut, result.MaximumQueuedFrames,
			result.MaximumOldestFrameNanoseconds, result.Rejections, result.MaximumDeliveryDelayNanoseconds, result.HydrationRowsPerSecond, result.AverageCPUCores,
			diagnosticOutcomeName(int(result.Outcome)), diagnosticStopName(int(result.StopReason)))
	}
	t.Logf("classification=%s next_boundary=%s", diagnosticClassificationName(classification), diagnosticNextBoundary(classification, results))
}

// TestLiveHydrationThroughputHarness is the fake-source race proof. It proves
// bounded sampling and a deadline safety stop while a REST worker is blocked,
// then proves cancellation joins the HTTP, WebSocket, delivery, and engine
// work. The safety predicate is separately exercised at every stop boundary.
func TestLiveHydrationThroughputHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in diagnostic harness proof")
	}
	moduleRoot, err := liveDiagnosticModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err != nil {
		t.Fatalf("module-root discovery: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*liveDiagnosticSample)
		want int
	}{
		{"rejection", func(s *liveDiagnosticSample) { s.RejectedCapacity = 1 }, diagnosticStopRejection},
		{"integrity", func(s *liveDiagnosticSample) { s.IngressIntegrity = 1 }, diagnosticStopTerminal},
		{"queue", func(s *liveDiagnosticSample) { s.CurrentQueuedFrames = 384 }, diagnosticStopQueue},
		{"oldest", func(s *liveDiagnosticSample) { s.OldestLiveFrameNanoseconds = int64(2 * time.Second) }, diagnosticStopOldest},
		{"accounting", func(s *liveDiagnosticSample) { s.AccountingValid = 0 }, diagnosticStopAccounting},
	} {
		t.Run("safety_"+test.name, func(t *testing.T) {
			sample := liveDiagnosticSample{AccountingValid: 1}
			test.edit(&sample)
			if got := liveDiagnosticSafetyStop(sample, productionLiveDiagnosticConfig()); got != test.want {
				t.Fatalf("stop=%d want=%d", got, test.want)
			}
		})
	}

	binding := capacityBinding(t, []string{"AAA", "BBB", "CCC", "DDD"})
	var activeREST atomic.Int64
	rest := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		activeREST.Add(1)
		defer activeREST.Add(-1)
		<-request.Context().Done()
	}))
	defer rest.Close()
	ws := liveDiagnosticWebSocketServer(t)
	defer ws.Close()
	hydrator, err := massive.NewHydrationWorker(rest.URL, func() (string, error) { return "fixture", nil }, rest.Client())
	if err != nil {
		t.Fatal(err)
	}
	now := binding.SessionStart().Add(time.Hour)
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(ws.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	config := productionLiveDiagnosticConfig()
	config.duration, config.cadence, config.safetyCadence = 120*time.Millisecond, 20*time.Millisecond, time.Millisecond
	trial := liveDiagnosticTrial{binding: binding, adapter: adapter, hydrator: hydrator, workers: 1, clock: func() time.Time { return now }, config: config}
	trial.components = productionDiagnosticComponents(adapter, hydrator, 1, len(binding.UniverseSymbols()))
	result, err := runLiveDiagnosticTrial(context.Background(), trial)
	if err != nil {
		t.Fatal(err)
	}
	maximumSamples := int(config.duration/config.cadence) + 1
	if result.Outcome != diagnosticStable || result.StopReason != diagnosticStopDeadline || len(result.Samples) < 2 || len(result.Samples) > maximumSamples || activeREST.Load() != 0 {
		t.Fatalf("bounded/joined result=%+v active_rest=%d maximum_samples=%d", result, activeREST.Load(), maximumSamples)
	}
	body, err := json.Marshal(liveDiagnosticArtifact{Schema: 1, Classification: diagnosticClassOtherStartup, Trials: []liveDiagnosticSummary{result}})
	if err != nil || strings.Contains(string(body), "fixture") || strings.Contains(string(body), "AAA") || strings.Contains(string(body), rest.URL) || strings.Contains(string(body), ws.URL) {
		t.Fatalf("numeric artifact boundary violated: err=%v", err)
	}
}

func runLiveDiagnosticTrial(parent context.Context, trial liveDiagnosticTrial) (liveDiagnosticSummary, error) {
	if parent == nil || trial.binding.Identity() == "" || trial.adapter == nil || trial.hydrator == nil || trial.clock == nil || trial.config.duration <= 0 ||
		trial.config.cadence <= 0 || trial.config.safetyCadence <= 0 || trial.config.joinDeadline <= 0 || trial.config.queueStop == 0 || trial.config.oldestStop <= 0 {
		return liveDiagnosticSummary{}, errors.New("invalid live diagnostic trial")
	}
	trialCtx, cancelTrial := context.WithCancel(parent)
	defer cancelTrial()
	runtimeConfig := DefaultConfig()
	run, err := New(trialCtx, trial.binding, runtimeConfig, trial.clock)
	if err != nil {
		return liveDiagnosticSummary{}, err
	}
	components := trial.components
	if !components.valid() {
		components = productionDiagnosticComponents(trial.adapter, trial.hydrator, max(1, trial.workers), len(trial.binding.UniverseSymbols()))
	}
	establish, cancelEstablish := context.WithTimeout(trialCtx, runtimeConfig.ConnectionAttemptDeadline)
	attempt, err := run.openAttempt(trialCtx, establish, components, 100)
	cancelEstablish()
	if err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), trial.config.joinDeadline)
		defer cancelCleanup()
		if attempt != nil {
			run.setLiveSources(attempt, trial.adapter)
			_ = closeLiveDiagnosticTrial(cleanupCtx, run, attempt)
		} else {
			_ = run.Shutdown(cleanupCtx)
		}
		return liveDiagnosticSummary{}, err
	}
	run.setLiveSources(attempt, trial.adapter)
	baselineMetrics := run.Metrics()
	baseline := liveDiagnosticBaseline{metrics: baselineMetrics, deliveries: run.deliveryCount.Load(), deliveryNanos: run.deliveryTotalNanos.Load(),
		deliveryMax: run.deliveryMaxNanos.Load(), cpuSeconds: goRuntimeCPUSeconds(), started: time.Now()}

	workDone := make(chan error, 1)
	go func() {
		if trial.workers > 0 {
			if _, hydrateErr := run.hydrate(trialCtx, components, attempt, engine.HydrationFreshBootstrap); hydrateErr != nil {
				workDone <- hydrateErr
				return
			}
		}
		for {
			started := time.Now()
			result, ok, deliveryErr := attempt.DeliverNextToEngine(trialCtx, run.engine)
			if ok {
				run.observeDelivery(started, result)
			}
			if deliveryErr != nil || !ok {
				workDone <- deliveryErr
				return
			}
		}
	}()

	summary := liveDiagnosticSummary{Workers: int64(trial.workers)}
	trackSample := func(sample liveDiagnosticSample) {
		if sample.MaximumQueuedFrames > summary.MaximumQueuedFrames {
			summary.MaximumQueuedFrames = sample.MaximumQueuedFrames
		}
		if sample.OldestLiveFrameNanoseconds > summary.MaximumOldestFrameNanoseconds {
			summary.MaximumOldestFrameNanoseconds = sample.OldestLiveFrameNanoseconds
		}
		if int64(sample.OverallMaxDeliveryDelayNanoseconds) > summary.MaximumDeliveryDelayNanoseconds {
			summary.MaximumDeliveryDelayNanoseconds = int64(sample.OverallMaxDeliveryDelayNanoseconds)
		}
	}
	appendSample := func() liveDiagnosticSample {
		sample := captureLiveDiagnosticSample(run, baseline, trial.workers)
		summary.Samples = append(summary.Samples, sample)
		trackSample(sample)
		return sample
	}
	initial := appendSample()
	stopReason := liveDiagnosticSafetyStop(initial, trial.config)
	sampleTicker := time.NewTicker(trial.config.cadence)
	safetyTicker := time.NewTicker(trial.config.safetyCadence)
	deadline := time.NewTimer(trial.config.duration)
	defer sampleTicker.Stop()
	defer safetyTicker.Stop()
	defer deadline.Stop()
	workJoined := false
	for stopReason == 0 {
		select {
		case <-parent.Done():
			stopReason = diagnosticStopTerminal
		case <-workDone:
			workJoined = true
			stopReason = diagnosticStopTerminal
		case <-deadline.C:
			stopReason = diagnosticStopDeadline
		case <-sampleTicker.C:
			stopReason = liveDiagnosticSafetyStop(appendSample(), trial.config)
		case <-safetyTicker.C:
			probe := captureLiveDiagnosticSample(run, baseline, trial.workers)
			trackSample(probe)
			stopReason = liveDiagnosticSafetyStop(probe, trial.config)
		}
	}
	boundary := captureLiveDiagnosticSample(run, baseline, trial.workers)
	maximumSamples := int(trial.config.duration/trial.config.cadence) + 1
	if len(summary.Samples) >= maximumSamples {
		summary.Samples[len(summary.Samples)-1] = boundary
	} else {
		summary.Samples = append(summary.Samples, boundary)
	}
	trackSample(boundary)
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), trial.config.joinDeadline)
	defer cancelCleanup()
	cancelTrial()
	if !workJoined {
		select {
		case <-workDone:
		case <-cleanupCtx.Done():
			return summary, errors.New("diagnostic live/hydration work did not join")
		}
	}
	err = closeLiveDiagnosticTrial(cleanupCtx, run, attempt)
	if err != nil {
		return summary, err
	}

	last := summary.Samples[len(summary.Samples)-1]
	seconds := float64(max(last.ElapsedMilliseconds, 1)) / 1000
	summary.StopReason = int64(stopReason)
	summary.DurationMilliseconds = last.ElapsedMilliseconds
	summary.LiveFramesPerSecondIn = float64(last.LiveFramesRead) / seconds
	summary.LiveFramesPerSecondOut = float64(last.LiveFramesDispositioned) / seconds
	summary.Rejections = last.RejectedCapacity + last.RejectedReceipt + last.RejectedOversize + last.RejectedGate
	summary.HydrationRowsPerSecond = float64(last.HydrationRowsConsumed) / seconds
	summary.AverageCPUCores = last.AverageCPUCores
	summary.Outcome = diagnosticStable
	if stopReason != diagnosticStopDeadline {
		summary.Outcome = diagnosticSaturated
	}
	if stopReason == diagnosticStopTerminal {
		summary.Outcome = diagnosticTerminal
	}
	return summary, nil
}

func captureLiveDiagnosticSample(run *Runtime, baseline liveDiagnosticBaseline, workers int) liveDiagnosticSample {
	now := time.Now()
	observed := run.Metrics()
	queue := observed.LiveQueue
	baseQueue := baseline.metrics.LiveQueue
	deliveries := run.deliveryCount.Load() - baseline.deliveries
	deliveryNanos := run.deliveryTotalNanos.Load() - baseline.deliveryNanos
	overallMax := run.deliveryMaxNanos.Load()
	if overallMax <= baseline.deliveryMax {
		overallMax = 0
	}
	hydration := observed.Engine.Hydration
	completed := hydration.Accounting.CompletedValue + hydration.Accounting.CompletedEmpty
	elapsed := now.Sub(baseline.started)
	cpu := goRuntimeCPUSeconds() - baseline.cpuSeconds
	averageCPU := 0.0
	if elapsed > 0 && cpu >= 0 {
		averageCPU = cpu / elapsed.Seconds()
	}
	valid := int64(0)
	if observed.AccountingValid {
		valid = 1
	}
	mean := uint64(0)
	if deliveries > 0 {
		mean = deliveryNanos / deliveries
	}
	return liveDiagnosticSample{
		ElapsedMilliseconds: elapsed.Milliseconds(), Workers: int64(workers),
		LiveFramesRead: subtractCounter(queue.FramesRead, baseQueue.FramesRead), LiveFramesAdmitted: subtractCounter(queue.FramesAdmitted, baseQueue.FramesAdmitted),
		LiveFramesDispositioned: subtractCounter(queue.FramesDispositioned, baseQueue.FramesDispositioned), LiveFramesQueued: queue.FramesQueued,
		LiveFramesFenced: subtractCounter(queue.FramesFenced, baseQueue.FramesFenced), RejectedCapacity: subtractCounter(queue.FramesRejectedCapacity, baseQueue.FramesRejectedCapacity),
		RejectedReceipt: subtractCounter(queue.FramesRejectedReceipt, baseQueue.FramesRejectedReceipt), RejectedOversize: subtractCounter(queue.FramesRejectedOversize, baseQueue.FramesRejectedOversize),
		RejectedGate:        subtractCounter(queue.FramesRejectedGateOrClose, baseQueue.FramesRejectedGateOrClose),
		CurrentQueuedFrames: observed.QueueCurrentFrames, MaximumQueuedFrames: observed.QueueHighFrames, CurrentQueuedBytes: int64(observed.QueueCurrentBytes), MaximumQueuedBytes: int64(observed.QueueHighBytes),
		OldestLiveFrameNanoseconds: int64(queue.OldestFrameAge), LiveDeliveryCount: deliveries, MeanDeliveryDelayNanoseconds: mean,
		OneSecondMaxDelayNanoseconds: uint64(observed.MaxProcessingDelayOneSecond), OverallMaxDeliveryDelayNanoseconds: overallMax,
		EngineQueueOccupancy: int64(observed.Engine.QueueOccupancy), AggregateConsumed: observed.Engine.Aggregates.Consumed, AggregateInserted: observed.Engine.Aggregates.Inserted,
		AggregateRevised: observed.Engine.Aggregates.Revised, AggregateRejected: observed.Engine.Aggregates.Rejected, AggregateFenced: observed.Engine.Aggregates.Fenced,
		IngressIntegrity: observed.Engine.Connection.Integrity + observed.Engine.Aggregates.Integrity + observed.Engine.Transitions.IntegrityFailure,
		HydrationOpen:    hydration.Accounting.Open, HydrationCompleted: completed, HydrationFailed: hydration.Accounting.Failed, HydrationCanceled: hydration.Accounting.Canceled,
		HydrationFenced: hydration.Accounting.Fenced, HydrationRowsConsumed: hydration.Rows.Consumed, HydrationRowsPerSecond: float64(hydration.Rows.Consumed) / max(elapsed.Seconds(), 0.001),
		AverageCPUCores: averageCPU, HeapAllocBytes: observed.HeapAllocBytes, HeapInUseBytes: observed.HeapInUseBytes, Goroutines: int64(observed.Goroutines), AccountingValid: valid,
	}
}

func liveDiagnosticSafetyStop(sample liveDiagnosticSample, config liveDiagnosticConfig) int {
	if sample.RejectedCapacity+sample.RejectedReceipt+sample.RejectedOversize+sample.RejectedGate > 0 {
		return diagnosticStopRejection
	}
	if sample.IngressIntegrity > 0 {
		return diagnosticStopTerminal
	}
	if sample.CurrentQueuedFrames >= config.queueStop {
		return diagnosticStopQueue
	}
	if time.Duration(sample.OldestLiveFrameNanoseconds) >= config.oldestStop {
		return diagnosticStopOldest
	}
	if sample.AccountingValid == 0 {
		return diagnosticStopAccounting
	}
	return 0
}

func closeLiveDiagnosticTrial(ctx context.Context, run *Runtime, attempt *massive.LiveAttempt) error {
	_ = attempt.Close(massive.CloseEpochCommand{BindingIdentity: run.binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 190, Cause: massive.CloseControlledStop})
	for {
		started := time.Now()
		result, ok, err := attempt.DeliverNextToEngine(ctx, run.engine)
		if ok {
			run.observeDelivery(started, result)
		}
		if err != nil || !ok {
			break
		}
	}
	if err := attempt.Wait(ctx); err != nil {
		return errors.New("diagnostic WebSocket cleanup did not join")
	}
	if err := run.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func productionDiagnosticComponents(adapter *massive.LiveAdapter, hydrator *massive.HydrationWorker, workers, population int) LiveComponents {
	return LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: workers, RowsPerChunk: 256, MaximumResponseBytes: 512 << 20,
		MaximumNormalizedRecords: int64(population) * 57_600, MaximumResidentRecords: int64(workers) * 57_600, Durations: defaultDurations()}
}

func newProviderLiveDiagnosticTrial(t *testing.T, binding reference.Binding, credential string, workers int) liveDiagnosticTrial {
	t.Helper()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "wss://socket.massive.com/stocks", Credential: credential,
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker("https://api.massive.com", func() (string, error) { return credential, nil }, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	componentWorkers := max(1, workers)
	return liveDiagnosticTrial{binding: binding, adapter: adapter, hydrator: hydrator, workers: workers, clock: func() time.Time { return time.Now().UTC() },
		config: productionLiveDiagnosticConfig(), components: productionDiagnosticComponents(adapter, hydrator, componentWorkers, len(binding.UniverseSymbols()))}
}

func cachedLiveDiagnosticBinding(t *testing.T, ctx context.Context, tradingDate, dataDirectory string) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal("load accepted exchange schedule")
	}
	facts, err := schedule.ForTradingDate(tradingDate)
	if err != nil {
		t.Fatal("unsupported LIVE_TRADING_DATE")
	}
	// Empty credentials deliberately force the resolvers onto validated exact-
	// date caches without making an additional reference-data provider request.
	universe, err := (&reference.Resolver{DataDir: dataDirectory, Schedule: schedule}).Resolve(ctx, facts)
	if err != nil || universe.Source() != reference.SourceCurrentCache {
		t.Fatal("current exact-date universe cache is required")
	}
	priors, err := (&reference.PriorCloseResolver{DataDir: dataDirectory, Schedule: schedule}).Resolve(ctx, facts, universe)
	if err != nil || priors.Source() != reference.SourceCurrentCache {
		t.Fatal("current exact-date prior-close cache is required")
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal("assemble cached production binding")
	}
	return binding
}

func validatedLiveDiagnosticOutput(t *testing.T, output string) string {
	t.Helper()
	working, err := liveDiagnosticModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(working, "var")
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("LIVE_DIAGNOSTIC_OUTPUT must be a directory below the current worktree's var directory")
	}
	return absolute
}

func liveDiagnosticModuleRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", errors.New("resolve diagnostic working directory")
	}
	for depth := 0; depth < 8; depth++ {
		body, readErr := os.ReadFile(filepath.Join(current, "go.mod"))
		if readErr == nil && strings.Contains(string(body), "module github.com/belandresj/live-equities-momentum-scanner\n") {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("locate diagnostic module root")
}

func writeLiveDiagnosticArtifact(directory string, artifact liveDiagnosticArtifact) error {
	if artifact.Schema != 1 || len(artifact.Trials) == 0 || len(artifact.Trials) > 5 {
		return errors.New("invalid bounded diagnostic artifact")
	}
	for _, trial := range artifact.Trials {
		if len(trial.Samples) == 0 || len(trial.Samples) > 31 {
			return errors.New("invalid bounded diagnostic sample count")
		}
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return errors.New("create diagnostic output directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return errors.New("protect diagnostic output directory")
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil || len(body) > 1<<20 {
		return errors.New("encode bounded diagnostic artifact")
	}
	file, err := os.OpenFile(filepath.Join(directory, "summary.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create diagnostic artifact without overwriting prior evidence")
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return errors.New("write diagnostic artifact")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return errors.New("sync diagnostic artifact")
	}
	return file.Close()
}

func classifyLiveDiagnostic(results []liveDiagnosticSummary) int {
	if len(results) == 0 || results[0].Outcome != diagnosticStable {
		return diagnosticClassLivePath
	}
	if len(results) < 2 || results[1].Outcome != diagnosticStable {
		return diagnosticClassCrossSource
	}
	for _, result := range results[2:] {
		if result.Outcome != diagnosticStable {
			return diagnosticClassWorkerBoundary
		}
	}
	return diagnosticClassOtherStartup
}

func diagnosticOutcomeName(outcome int) string {
	switch outcome {
	case diagnosticStable:
		return "stable"
	case diagnosticSaturated:
		return "saturated"
	case diagnosticTerminal:
		return "terminal"
	default:
		return "invalid"
	}
}

func diagnosticStopName(reason int) string {
	switch reason {
	case diagnosticStopDeadline:
		return "deadline"
	case diagnosticStopRejection:
		return "live_frame_rejection"
	case diagnosticStopTerminal:
		return "connection_or_integrity_terminal"
	case diagnosticStopQueue:
		return "queue_75_percent"
	case diagnosticStopOldest:
		return "oldest_frame_two_seconds"
	case diagnosticStopAccounting:
		return "accounting_invalid"
	default:
		return "invalid"
	}
}

func diagnosticClassificationName(classification int) string {
	switch classification {
	case diagnosticClassLivePath:
		return "live_path_bottleneck"
	case diagnosticClassCrossSource:
		return "cross_source_contention"
	case diagnosticClassWorkerBoundary:
		return "hydration_worker_boundary"
	case diagnosticClassOtherStartup:
		return "other_production_start_interaction"
	default:
		return "invalid"
	}
}

func diagnosticNextBoundary(classification int, results []liveDiagnosticSummary) string {
	switch classification {
	case diagnosticClassLivePath:
		return "live_normalization_admission_evaluation_batching_or_optimization"
	case diagnosticClassCrossSource:
		return "live_priority_engine_admission_and_reduced_hydration_side_work"
	case diagnosticClassWorkerBoundary:
		maximumStable := int64(0)
		for _, result := range results {
			if result.Outcome == diagnosticStable {
				maximumStable = result.Workers
			}
		}
		return "cap_or_throttle_hydration_at_" + strconv.FormatInt(maximumStable, 10) + "_workers"
	case diagnosticClassOtherStartup:
		return "isolate_non_hydration_production_start_interaction"
	default:
		return "invalid"
	}
}

func goRuntimeCPUSeconds() float64 {
	samples := []metrics.Sample{{Name: "/cpu/classes/total:cpu-seconds"}}
	metrics.Read(samples)
	if samples[0].Value.Kind() != metrics.KindFloat64 {
		return 0
	}
	return samples[0].Value.Float64()
}

func subtractCounter(value, baseline uint64) uint64 {
	if value < baseline {
		return 0
	}
	return value - baseline
}

func liveDiagnosticWebSocketServer(t *testing.T) *httptest.Server {
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
		if _, raw, err := connection.Read(ctx); err != nil || !strings.Contains(string(raw), "A.*") || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !write(`[]`) {
					return
				}
			}
		}
	}))
}
