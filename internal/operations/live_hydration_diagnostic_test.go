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
	diagnosticTrialLiveOnly = iota + 1
	diagnosticTrialOneWorker
)

type liveDiagnosticConfig struct {
	liveOnlyDuration, oneWorkerTimeout   time.Duration
	cadence, safetyCadence, joinDeadline time.Duration
	queueStop                            uint64
	oldestStop                           time.Duration
	postReadySamples                     int
}

func productionLiveDiagnosticConfig() liveDiagnosticConfig {
	return liveDiagnosticConfig{liveOnlyDuration: 30 * time.Second, oneWorkerTimeout: 8 * time.Minute, cadence: time.Second, safetyCadence: 50 * time.Millisecond,
		joinDeadline: 10 * time.Second, queueStop: 384, oldestStop: 2 * time.Second, postReadySamples: 10}
}

// liveDiagnosticSample deliberately contains only bounded numeric values. It
// cannot retain a credential, URL, provider body, raw frame, symbol, or row.
type liveDiagnosticSample struct {
	ElapsedMilliseconds, Workers                                                         int64
	LiveFramesRead, LiveFramesAdmitted, LiveFramesDispositioned, LiveFramesQueued        uint64
	LiveFramesFenced, RejectedCapacity, RejectedReceipt, RejectedOversize, RejectedGate  uint64
	ConnectionEpoch, IngressFencesStarted, IngressFencesDispositioned                    uint64
	CurrentQueuedFrames, MaximumQueuedFrames                                             uint64
	CurrentQueuedBytes, MaximumQueuedBytes                                               int64
	OldestLiveFrameNanoseconds                                                           int64
	LiveDeliveryCount, MeanDeliveryDelayNanoseconds, OneSecondMaxDelayNanoseconds        uint64
	OverallMaxDeliveryDelayNanoseconds                                                   uint64
	EngineQueueOccupancy                                                                 int64
	AggregateConsumed, AggregateInserted, AggregateRevised, AggregateDuplicate           uint64
	AggregateRejected, AggregateConflictOrWithdrawn                                      uint64
	AggregateFenced, IngressIntegrity                                                    uint64
	HydrationGeneration, HydrationPlanned, HydrationOpen, HydrationValue, HydrationEmpty uint64
	HydrationFenceMarkerOrdinal                                                          uint64
	HydrationFailed, HydrationCanceled, HydrationFenced                                  uint64
	HydrationRowsConsumed, HydrationRowsInserted, HydrationRowsDuplicate                 uint64
	HydrationRowsConflictOrWithdrawn, HydrationRowsRejected, HydrationRowsFenced         uint64
	HydrationRowsPerSecond, AverageCPUCores                                              float64
	HeapAllocBytes, HeapInUseBytes                                                       uint64
	Goroutines, AccountingValid, Suppressed, FenceReconciled                             int64
	ConnectionAcknowledged                                                               int64
	LifecycleLive, BackendReady, RankingCurrent                                          int64
	WatermarkLagNanoseconds                                                              int64
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
	Schema, Trial int64
	Trials        []liveDiagnosticSummary
}

type liveDiagnosticTrial struct {
	binding         reference.Binding
	adapter         *massive.LiveAdapter
	hydrator        *massive.HydrationWorker
	mode            int
	workers         int
	clock           func() time.Time
	config          liveDiagnosticConfig
	runtimeConfig   Config
	components      LiveComponents
	sampleTransform func(*liveDiagnosticSample)
	queueTransform  func(*liveDiagnosticQueueProbe)
}

type liveDiagnosticQueueProbe struct {
	currentQueued, maximumQueued uint64
	oldest                       time.Duration
	rejections                   uint64
}

type liveDiagnosticHydrationReply struct {
	err                error
	progressedWithLive bool
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
	mode, err := parseLiveHydrationTrial(os.Getenv("LIVE_HYDRATION_TRIAL"))
	if err != nil {
		t.Fatal(err)
	}
	tradingDate := os.Getenv("LIVE_TRADING_DATE")
	credential := os.Getenv("MASSIVE_API_KEY")
	output := os.Getenv("LIVE_DIAGNOSTIC_OUTPUT")
	if tradingDate == "" || credential == "" || output == "" {
		t.Fatal("LIVE_TRADING_DATE, LIVE_DIAGNOSTIC_OUTPUT, and MASSIVE_API_KEY are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute+30*time.Second)
	defer cancel()
	output = validatedLiveDiagnosticOutput(t, output)
	moduleRoot, err := liveDiagnosticModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	binding := cachedLiveDiagnosticBinding(t, ctx, tradingDate, filepath.Join(moduleRoot, "var", "reference"))

	trial := newProviderLiveDiagnosticTrial(t, binding, credential, mode)
	result, err := runLiveDiagnosticTrial(ctx, trial)
	if err != nil {
		t.Fatalf("diagnostic trial cleanup: %v", err)
	}
	artifact := liveDiagnosticArtifact{Schema: 1, Trial: int64(mode), Trials: []liveDiagnosticSummary{result}}
	if err := writeLiveDiagnosticArtifact(output, artifact); err != nil {
		t.Fatal(err)
	}
	t.Logf("workers=%d duration_ms=%d live_in_per_s=%.2f live_out_per_s=%.2f max_queued=%d max_oldest_ns=%d rejections=%d max_delivery_ns=%d hydration_rows_per_s=%.2f avg_cpu_cores=%.3f outcome=%s stop=%s",
		result.Workers, result.DurationMilliseconds, result.LiveFramesPerSecondIn, result.LiveFramesPerSecondOut, result.MaximumQueuedFrames,
		result.MaximumOldestFrameNanoseconds, result.Rejections, result.MaximumDeliveryDelayNanoseconds, result.HydrationRowsPerSecond, result.AverageCPUCores,
		diagnosticOutcomeName(int(result.Outcome)), diagnosticStopName(int(result.StopReason)))
	last := result.Samples[len(result.Samples)-1]
	t.Logf("hydration_terminal=%d/%d planned=%d fence=%d ready=%d ranking_current=%d watermark_lag_ns=%d accounting=%d",
		last.HydrationValue, last.HydrationEmpty, last.HydrationPlanned, last.FenceReconciled, last.BackendReady, last.RankingCurrent, last.WatermarkLagNanoseconds, last.AccountingValid)
	if result.Outcome != diagnosticStable {
		t.Fatal("live hydration trial failed its acceptance boundary")
	}
}

func parseLiveHydrationTrial(value string) (int, error) {
	switch value {
	case "live_only":
		return diagnosticTrialLiveOnly, nil
	case "one_worker":
		return diagnosticTrialOneWorker, nil
	default:
		return 0, errors.New("LIVE_HYDRATION_TRIAL must be exactly live_only or one_worker")
	}
}

// TestLiveHydrationThroughputHarness is the fake-source race proof for mode
// selection, the non-hydrating live-only path, bounded sampling, cancellation,
// cleanup, and numeric artifact containment. The production one-worker path is
// proved by TestLiveRESTHydrationProgressionHarness.
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
	for _, value := range []string{"", "0", "live", "one_worker "} {
		if _, err := parseLiveHydrationTrial(value); err == nil {
			t.Fatalf("trial %q was accepted", value)
		}
	}
	if mode, err := parseLiveHydrationTrial("live_only"); err != nil || mode != diagnosticTrialLiveOnly {
		t.Fatalf("live_only parse=%d err=%v", mode, err)
	}
	if mode, err := parseLiveHydrationTrial("one_worker"); err != nil || mode != diagnosticTrialOneWorker {
		t.Fatalf("one_worker parse=%d err=%v", mode, err)
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
			if got := liveDiagnosticHeavyStop(sample, diagnosticTrialOneWorker, productionLiveDiagnosticConfig()); got != test.want {
				t.Fatalf("stop=%d want=%d", got, test.want)
			}
		})
	}
	for _, test := range []struct {
		name  string
		probe liveDiagnosticQueueProbe
		want  int
	}{
		{"rejection", liveDiagnosticQueueProbe{rejections: 1}, diagnosticStopRejection},
		{"queue", liveDiagnosticQueueProbe{currentQueued: 384}, diagnosticStopQueue},
		{"oldest", liveDiagnosticQueueProbe{oldest: 2 * time.Second}, diagnosticStopOldest},
	} {
		t.Run("queue_safety_"+test.name, func(t *testing.T) {
			if got := liveDiagnosticQueueStop(test.probe, productionLiveDiagnosticConfig()); got != test.want {
				t.Fatalf("stop=%d want=%d", got, test.want)
			}
		})
	}
	body, err := os.ReadFile(filepath.Join(moduleRoot, "internal", "operations", "live_hydration_diagnostic_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	start := strings.LastIndex(source, "func captureLiveDiagnosticQueueProbe")
	end := strings.Index(source[start:], "\n}\n\nfunc liveDiagnosticQueueStop")
	if start < 0 || end < 0 {
		t.Fatal("locate lightweight queue probe source")
	}
	probeSource := source[start : start+end]
	if strings.Contains(probeSource, ".Metrics(") || strings.Contains(probeSource, "ReadMemStats") {
		t.Fatal("50-millisecond queue probe performs a heavyweight runtime sample")
	}

	binding := capacityBinding(t, []string{"AAA", "BBB", "CCC", "DDD"})
	var restRequests atomic.Int64
	rest := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		restRequests.Add(1)
		http.Error(writer, "live_only must not request REST", http.StatusInternalServerError)
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
	config.liveOnlyDuration, config.cadence, config.safetyCadence = 120*time.Millisecond, 20*time.Millisecond, time.Millisecond
	trial := liveDiagnosticTrial{binding: binding, adapter: adapter, hydrator: hydrator, mode: diagnosticTrialLiveOnly, workers: 0, clock: func() time.Time { return now }, config: config}
	trial.components = productionDiagnosticComponents(adapter, hydrator, 1, len(binding.UniverseSymbols()))
	result, err := runLiveDiagnosticTrial(context.Background(), trial)
	if err != nil {
		t.Fatal(err)
	}
	maximumSamples := int(config.liveOnlyDuration/config.cadence) + 2
	last := result.Samples[len(result.Samples)-1]
	if result.Outcome != diagnosticStable || result.StopReason != diagnosticStopDeadline || len(result.Samples) < 2 || len(result.Samples) > maximumSamples || restRequests.Load() != 0 ||
		last.BackendReady != 0 || last.HydrationPlanned+last.HydrationOpen+last.HydrationValue+last.HydrationEmpty+last.HydrationFailed+last.HydrationCanceled+last.HydrationFenced != 0 {
		t.Fatalf("bounded live_only result=%+v rest_requests=%d maximum_samples=%d", result, restRequests.Load(), maximumSamples)
	}
	encoded, err := json.Marshal(liveDiagnosticArtifact{Schema: 1, Trial: diagnosticTrialLiveOnly, Trials: []liveDiagnosticSummary{result}})
	if err != nil || strings.Contains(string(encoded), "fixture") || strings.Contains(string(encoded), "AAA") || strings.Contains(string(encoded), rest.URL) || strings.Contains(string(encoded), ws.URL) {
		t.Fatalf("numeric artifact boundary violated: err=%v", err)
	}
	root := t.TempDir()
	first, second := filepath.Join(root, "live-only"), filepath.Join(root, "one-worker")
	if err := writeLiveDiagnosticArtifact(first, liveDiagnosticArtifact{Schema: 1, Trial: diagnosticTrialLiveOnly, Trials: []liveDiagnosticSummary{result}}); err != nil {
		t.Fatal(err)
	}
	oneWorkerResult := result
	oneWorkerResult.Workers = 1
	if err := writeLiveDiagnosticArtifact(second, liveDiagnosticArtifact{Schema: 1, Trial: diagnosticTrialOneWorker, Trials: []liveDiagnosticSummary{oneWorkerResult}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(first, "summary.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(second, "summary.json")); err != nil {
		t.Fatal(err)
	}
	if err := writeLiveDiagnosticArtifact(first, liveDiagnosticArtifact{Schema: 1, Trial: diagnosticTrialLiveOnly, Trials: []liveDiagnosticSummary{result}}); err == nil {
		t.Fatal("diagnostic artifact overwrote prior evidence")
	}
	for _, stop := range []struct {
		name            string
		want            int
		sampleTransform func(*liveDiagnosticSample)
		queueTransform  func(*liveDiagnosticQueueProbe)
	}{
		{"rejection", diagnosticStopRejection, func(sample *liveDiagnosticSample) { sample.RejectedCapacity = 1 }, nil},
		{"integrity", diagnosticStopTerminal, func(sample *liveDiagnosticSample) { sample.IngressIntegrity = 1 }, nil},
		{"accounting", diagnosticStopAccounting, func(sample *liveDiagnosticSample) { sample.AccountingValid = 0 }, nil},
		{"queue", diagnosticStopQueue, nil, func(probe *liveDiagnosticQueueProbe) { probe.currentQueued = 384 }},
		{"oldest", diagnosticStopOldest, nil, func(probe *liveDiagnosticQueueProbe) { probe.oldest = 2 * time.Second }},
	} {
		t.Run("cancel_join_"+stop.name, func(t *testing.T) {
			stopAdapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(ws.URL, "http"), Credential: "fixture",
				Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			stopConfig := productionLiveDiagnosticConfig()
			stopConfig.liveOnlyDuration, stopConfig.cadence, stopConfig.safetyCadence = 100*time.Millisecond, 5*time.Millisecond, time.Millisecond
			stopTrial := liveDiagnosticTrial{binding: binding, adapter: stopAdapter, hydrator: hydrator, mode: diagnosticTrialLiveOnly, workers: 0,
				clock: func() time.Time { return now }, config: stopConfig, sampleTransform: stop.sampleTransform, queueTransform: stop.queueTransform}
			stopTrial.components = productionDiagnosticComponents(stopAdapter, hydrator, 1, len(binding.UniverseSymbols()))
			stopped, err := runLiveDiagnosticTrial(context.Background(), stopTrial)
			if err != nil || stopped.StopReason != int64(stop.want) || stopped.Outcome == diagnosticStable {
				t.Fatalf("stop=%s result=%+v err=%v", stop.name, stopped, err)
			}
		})
	}

	oneWorkerREST := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		restRequests.Add(1)
		timer := time.NewTimer(3 * time.Millisecond)
		select {
		case <-request.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		symbol := ""
		parts := strings.Split(request.URL.Path, "/")
		if len(parts) > 4 {
			symbol = parts[4]
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"OK","ticker":"` + symbol + `","adjusted":false,"results":[]}`))
	}))
	defer oneWorkerREST.Close()
	oneWorkerWS := liveDiagnosticWebSocketServer(t)
	defer oneWorkerWS.Close()
	oneWorkerHydrator, err := massive.NewHydrationWorker(oneWorkerREST.URL, func() (string, error) { return "fixture", nil }, oneWorkerREST.Client())
	if err != nil {
		t.Fatal(err)
	}
	oneWorkerAdapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(oneWorkerWS.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	oneWorkerConfig := productionLiveDiagnosticConfig()
	oneWorkerConfig.oneWorkerTimeout, oneWorkerConfig.cadence, oneWorkerConfig.safetyCadence, oneWorkerConfig.postReadySamples = time.Second, 10*time.Millisecond, time.Millisecond, 3
	fastRuntime := DefaultConfig()
	fastRuntime.EvaluationDelay, fastRuntime.SampleCadence = 0, 5*time.Millisecond
	oneWorkerTrial := liveDiagnosticTrial{binding: binding, adapter: oneWorkerAdapter, hydrator: oneWorkerHydrator, mode: diagnosticTrialOneWorker, workers: 1,
		clock: func() time.Time { return now }, config: oneWorkerConfig, runtimeConfig: fastRuntime}
	oneWorkerTrial.components = productionDiagnosticComponents(oneWorkerAdapter, oneWorkerHydrator, 1, len(binding.UniverseSymbols()))
	oneWorkerDiagnostic, err := runLiveDiagnosticTrial(context.Background(), oneWorkerTrial)
	if err != nil {
		t.Fatal(err)
	}
	oneWorkerLast := oneWorkerDiagnostic.Samples[len(oneWorkerDiagnostic.Samples)-1]
	if oneWorkerDiagnostic.Outcome != diagnosticStable || oneWorkerDiagnostic.Workers != 1 || oneWorkerLast.HydrationPlanned != uint64(len(binding.UniverseSymbols())) ||
		oneWorkerLast.HydrationOpen != 0 || oneWorkerLast.HydrationFailed+oneWorkerLast.HydrationCanceled+oneWorkerLast.HydrationFenced != 0 ||
		oneWorkerLast.FenceReconciled != 1 || oneWorkerLast.BackendReady != 1 || oneWorkerLast.RankingCurrent != 1 || oneWorkerLast.LifecycleLive != 1 {
		t.Fatalf("one-worker diagnostic did not reach stable readiness: %+v", oneWorkerDiagnostic)
	}
}

func runLiveDiagnosticTrial(parent context.Context, trial liveDiagnosticTrial) (liveDiagnosticSummary, error) {
	if parent == nil || trial.binding.Identity() == "" || trial.adapter == nil || trial.hydrator == nil || trial.clock == nil ||
		(trial.mode != diagnosticTrialLiveOnly && trial.mode != diagnosticTrialOneWorker) ||
		trial.config.liveOnlyDuration <= 0 || trial.config.oneWorkerTimeout <= 0 || trial.config.cadence <= 0 || trial.config.safetyCadence <= 0 ||
		trial.config.joinDeadline <= 0 || trial.config.queueStop == 0 || trial.config.oldestStop <= 0 || trial.config.postReadySamples <= 0 {
		return liveDiagnosticSummary{}, errors.New("invalid live diagnostic trial")
	}
	workers := 0
	if trial.mode == diagnosticTrialOneWorker {
		workers = 1
	}
	if trial.workers != workers {
		return liveDiagnosticSummary{}, errors.New("live diagnostic trial worker count contradicts mode")
	}
	trialCtx, cancelTrial := context.WithCancel(parent)
	defer cancelTrial()
	runtimeConfig := trial.runtimeConfig
	if !runtimeConfig.valid() {
		runtimeConfig = DefaultConfig()
	}
	run, err := New(trialCtx, trial.binding, runtimeConfig, trial.clock)
	if err != nil {
		return liveDiagnosticSummary{}, err
	}
	components := trial.components
	if !components.valid() {
		components = productionDiagnosticComponents(trial.adapter, trial.hydrator, 1, len(trial.binding.UniverseSymbols()))
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

	queueBaseline := attempt.QueueAccounting()
	hydrationDone := make(chan liveDiagnosticHydrationReply, 1)
	workDone := make(chan error, 1)
	go func() {
		if trial.mode == diagnosticTrialOneWorker {
			if _, hydrateErr := run.hydrate(trialCtx, components, attempt, engine.HydrationFreshBootstrap); hydrateErr != nil {
				hydrationDone <- liveDiagnosticHydrationReply{err: hydrateErr}
				workDone <- hydrateErr
				return
			}
			queue := attempt.QueueAccounting()
			hydration := run.engine.ObserveOperational().Hydration.Accounting
			hydrationDone <- liveDiagnosticHydrationReply{progressedWithLive: queue.FramesDispositioned > queueBaseline.FramesDispositioned && hydration.CompletedValue+hydration.CompletedEmpty > 0}
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

	summary := liveDiagnosticSummary{Workers: int64(workers)}
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
		sample := captureLiveDiagnosticSample(run, baseline, workers)
		if trial.sampleTransform != nil {
			trial.sampleTransform(&sample)
		}
		summary.Samples = append(summary.Samples, sample)
		trackSample(sample)
		return sample
	}
	initial := appendSample()
	stopReason := liveDiagnosticHeavyStop(initial, trial.mode, trial.config)
	sampleTicker := time.NewTicker(trial.config.cadence)
	safetyTicker := time.NewTicker(trial.config.safetyCadence)
	duration := trial.config.liveOnlyDuration
	if trial.mode == diagnosticTrialOneWorker {
		duration = trial.config.oneWorkerTimeout
	}
	deadline := time.NewTimer(duration)
	defer sampleTicker.Stop()
	defer safetyTicker.Stop()
	defer deadline.Stop()
	workJoined := false
	hydrationTerminal := trial.mode == diagnosticTrialLiveOnly
	hydrationProgressedWithLive := false
	readyObserved := false
	readySamples := 0
	for stopReason == 0 {
		select {
		case <-parent.Done():
			stopReason = diagnosticStopTerminal
		case <-workDone:
			workJoined = true
			stopReason = diagnosticStopTerminal
		case hydrationReply := <-hydrationDone:
			hydrationTerminal = hydrationReply.err == nil
			hydrationProgressedWithLive = hydrationReply.progressedWithLive
			if hydrationReply.err != nil {
				stopReason = diagnosticStopTerminal
			}
		case <-deadline.C:
			stopReason = diagnosticStopDeadline
		case <-sampleTicker.C:
			sample := appendSample()
			stopReason = liveDiagnosticHeavyStop(sample, trial.mode, trial.config)
			if stopReason == 0 && trial.mode == diagnosticTrialOneWorker && hydrationTerminal {
				if sample.BackendReady == 1 && sample.RankingCurrent == 1 && sample.FenceReconciled == 1 && sample.LifecycleLive == 1 {
					if readyObserved {
						readySamples++
					} else {
						readyObserved = true
					}
				} else {
					readyObserved = false
					readySamples = 0
				}
				if readySamples == trial.config.postReadySamples {
					stopReason = diagnosticStopDeadline
				}
			}
		case <-safetyTicker.C:
			probe := captureLiveDiagnosticQueueProbe(attempt, queueBaseline)
			if trial.queueTransform != nil {
				trial.queueTransform(&probe)
			}
			if probe.maximumQueued > summary.MaximumQueuedFrames {
				summary.MaximumQueuedFrames = probe.maximumQueued
			}
			if int64(probe.oldest) > summary.MaximumOldestFrameNanoseconds {
				summary.MaximumOldestFrameNanoseconds = int64(probe.oldest)
			}
			stopReason = liveDiagnosticQueueStop(probe, trial.config)
		}
	}
	boundary := captureLiveDiagnosticSample(run, baseline, workers)
	summary.Samples = append(summary.Samples, boundary)
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
	if err = drainLiveDiagnosticAttempt(cleanupCtx, run, attempt); err != nil {
		return summary, err
	}
	if err = run.Shutdown(cleanupCtx); err != nil {
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
	acceptedCompletion := stopReason == diagnosticStopDeadline
	if trial.mode == diagnosticTrialOneWorker {
		acceptedCompletion = acceptedCompletion && hydrationTerminal && hydrationProgressedWithLive && readySamples == trial.config.postReadySamples &&
			last.HydrationGeneration == 1 && last.HydrationPlanned == uint64(len(trial.binding.UniverseSymbols())) && last.HydrationOpen == 0 &&
			last.HydrationValue+last.HydrationEmpty == last.HydrationPlanned && last.HydrationFailed+last.HydrationCanceled+last.HydrationFenced == 0 &&
			last.IngressFencesStarted >= 1 && last.IngressFencesStarted == last.IngressFencesDispositioned &&
			last.HydrationFenceMarkerOrdinal != 0 && last.FenceReconciled == 1 &&
			last.ConnectionEpoch != 0 && last.ConnectionAcknowledged == 1 && last.BackendReady == 1 && last.RankingCurrent == 1 &&
			last.LifecycleLive == 1 && last.WatermarkLagNanoseconds <= int64(runtimeConfig.ReadinessTolerance)
	} else {
		acceptedCompletion = acceptedCompletion && last.BackendReady == 0 && last.HydrationGeneration == 0 && last.HydrationPlanned == 0 && last.HydrationOpen == 0 &&
			last.HydrationValue == 0 && last.HydrationEmpty == 0 && last.HydrationFailed == 0 && last.HydrationCanceled == 0 && last.HydrationFenced == 0 &&
			last.IngressFencesStarted == 0 && last.IngressFencesDispositioned == 0 && last.ConnectionEpoch != 0 && last.ConnectionAcknowledged == 1
	}
	acceptedCompletion = acceptedCompletion && last.CurrentQueuedFrames == 0 && liveDiagnosticStableTail(summary.Samples, trial.config.postReadySamples)
	if !acceptedCompletion {
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
	status := run.Status()
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
	suppressed, fenceReconciled, connectionAcknowledged := int64(0), int64(0), int64(0)
	lifecycleLive, backendReady, rankingCurrent := int64(0), int64(0), int64(0)
	if observed.Engine.Suppression != "" || observed.Engine.Lifecycle == "suppressed" {
		suppressed = 1
	}
	if hydration.FenceReconciled {
		fenceReconciled = 1
	}
	if observed.Engine.Lifecycle == "live" {
		lifecycleLive = 1
	}
	if status.BackendReady {
		backendReady = 1
	}
	if status.RankingCurrent {
		rankingCurrent = 1
	}
	if observed.Engine.Connection.Acknowledged {
		connectionAcknowledged = 1
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
		RejectedGate:               subtractCounter(queue.FramesRejectedGateOrClose, baseQueue.FramesRejectedGateOrClose),
		ConnectionEpoch:            observed.Engine.Connection.Epoch,
		IngressFencesStarted:       subtractCounter(queue.IngressFencesStarted, baseQueue.IngressFencesStarted),
		IngressFencesDispositioned: subtractCounter(queue.IngressFencesDispositioned, baseQueue.IngressFencesDispositioned),
		CurrentQueuedFrames:        observed.QueueCurrentFrames, MaximumQueuedFrames: observed.QueueHighFrames, CurrentQueuedBytes: int64(observed.QueueCurrentBytes), MaximumQueuedBytes: int64(observed.QueueHighBytes),
		OldestLiveFrameNanoseconds: int64(queue.OldestFrameAge), LiveDeliveryCount: deliveries, MeanDeliveryDelayNanoseconds: mean,
		OneSecondMaxDelayNanoseconds: uint64(observed.MaxProcessingDelayOneSecond), OverallMaxDeliveryDelayNanoseconds: overallMax,
		EngineQueueOccupancy: int64(observed.Engine.QueueOccupancy), AggregateConsumed: observed.Engine.Aggregates.Consumed, AggregateInserted: observed.Engine.Aggregates.Inserted,
		AggregateRevised: observed.Engine.Aggregates.Revised, AggregateDuplicate: observed.Engine.Aggregates.ExactDuplicate,
		AggregateRejected: observed.Engine.Aggregates.Rejected, AggregateConflictOrWithdrawn: observed.Engine.Aggregates.WithdrawnConflict, AggregateFenced: observed.Engine.Aggregates.Fenced,
		IngressIntegrity:    observed.Engine.Connection.Integrity + observed.Engine.Aggregates.Integrity + observed.Engine.Transitions.IntegrityFailure + hydration.Rows.Integrity,
		HydrationGeneration: hydration.Generation, HydrationPlanned: hydration.Accounting.Planned, HydrationOpen: hydration.Accounting.Open, HydrationValue: hydration.Accounting.CompletedValue, HydrationEmpty: hydration.Accounting.CompletedEmpty,
		HydrationFenceMarkerOrdinal: hydration.FenceMarkerOrdinal,
		HydrationFailed:             hydration.Accounting.Failed, HydrationCanceled: hydration.Accounting.Canceled, HydrationFenced: hydration.Accounting.Fenced,
		HydrationRowsConsumed: hydration.Rows.Consumed, HydrationRowsInserted: hydration.Rows.Inserted, HydrationRowsDuplicate: hydration.Rows.Duplicate,
		HydrationRowsConflictOrWithdrawn: hydration.Rows.ConflictOrWithdrawal, HydrationRowsRejected: hydration.Rows.Rejected, HydrationRowsFenced: hydration.Rows.Fenced,
		HydrationRowsPerSecond: float64(hydration.Rows.Consumed) / max(elapsed.Seconds(), 0.001), AverageCPUCores: averageCPU,
		HeapAllocBytes: observed.HeapAllocBytes, HeapInUseBytes: observed.HeapInUseBytes, Goroutines: int64(observed.Goroutines), AccountingValid: valid,
		Suppressed: suppressed, FenceReconciled: fenceReconciled, ConnectionAcknowledged: connectionAcknowledged,
		LifecycleLive: lifecycleLive, BackendReady: backendReady, RankingCurrent: rankingCurrent,
		WatermarkLagNanoseconds: int64(status.WatermarkLag),
	}
}

func captureLiveDiagnosticQueueProbe(attempt *massive.LiveAttempt, baseline massive.LiveQueueAccounting) liveDiagnosticQueueProbe {
	queue := attempt.QueueAccounting()
	return liveDiagnosticQueueProbe{
		currentQueued: queue.FramesQueued + queue.FramesClassifying,
		maximumQueued: queue.FramesQueued + queue.FramesClassifying,
		oldest:        queue.OldestFrameAge,
		rejections: subtractCounter(queue.FramesRejectedCapacity, baseline.FramesRejectedCapacity) +
			subtractCounter(queue.FramesRejectedReceipt, baseline.FramesRejectedReceipt) +
			subtractCounter(queue.FramesRejectedOversize, baseline.FramesRejectedOversize) +
			subtractCounter(queue.FramesRejectedGateOrClose, baseline.FramesRejectedGateOrClose),
	}
}

func liveDiagnosticQueueStop(probe liveDiagnosticQueueProbe, config liveDiagnosticConfig) int {
	if probe.rejections > 0 {
		return diagnosticStopRejection
	}
	if probe.currentQueued >= config.queueStop {
		return diagnosticStopQueue
	}
	if probe.oldest >= config.oldestStop {
		return diagnosticStopOldest
	}
	return 0
}

func liveDiagnosticHeavyStop(sample liveDiagnosticSample, mode int, config liveDiagnosticConfig) int {
	if sample.RejectedCapacity+sample.RejectedReceipt+sample.RejectedOversize+sample.RejectedGate > 0 {
		return diagnosticStopRejection
	}
	if sample.IngressIntegrity > 0 || sample.Suppressed != 0 {
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
	if mode == diagnosticTrialLiveOnly && (sample.HydrationGeneration+sample.HydrationPlanned+sample.HydrationOpen+sample.HydrationValue+sample.HydrationEmpty+sample.HydrationFailed+sample.HydrationCanceled+sample.HydrationFenced+sample.HydrationRowsConsumed > 0) {
		return diagnosticStopAccounting
	}
	if mode == diagnosticTrialOneWorker && sample.HydrationFailed+sample.HydrationCanceled+sample.HydrationFenced > 0 {
		return diagnosticStopTerminal
	}
	return 0
}

func liveDiagnosticStableTail(samples []liveDiagnosticSample, count int) bool {
	if len(samples) < 2 {
		return false
	}
	start := max(0, len(samples)-count)
	last := samples[len(samples)-1]
	return last.CurrentQueuedFrames <= samples[start].CurrentQueuedFrames &&
		last.OldestLiveFrameNanoseconds <= samples[start].OldestLiveFrameNanoseconds
}

func closeLiveDiagnosticTrial(ctx context.Context, run *Runtime, attempt *massive.LiveAttempt) error {
	if err := drainLiveDiagnosticAttempt(ctx, run, attempt); err != nil {
		return err
	}
	return run.Shutdown(ctx)
}

func drainLiveDiagnosticAttempt(ctx context.Context, run *Runtime, attempt *massive.LiveAttempt) error {
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
	return nil
}

func productionDiagnosticComponents(adapter *massive.LiveAdapter, hydrator *massive.HydrationWorker, workers, population int) LiveComponents {
	return LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: workers, RowsPerChunk: 256, MaximumResponseBytes: 512 << 20,
		MaximumNormalizedRecords: int64(population) * 57_600, MaximumResidentRecords: int64(workers) * 57_600, Durations: defaultDurations()}
}

func newProviderLiveDiagnosticTrial(t *testing.T, binding reference.Binding, credential string, mode int) liveDiagnosticTrial {
	t.Helper()
	workers := 0
	if mode == diagnosticTrialOneWorker {
		workers = 1
	}
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "wss://socket.massive.com/stocks", Credential: credential,
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker("https://api.massive.com", func() (string, error) { return credential, nil }, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	return liveDiagnosticTrial{binding: binding, adapter: adapter, hydrator: hydrator, mode: mode, workers: workers, clock: func() time.Time { return time.Now().UTC() },
		config: productionLiveDiagnosticConfig(), components: productionDiagnosticComponents(adapter, hydrator, 1, len(binding.UniverseSymbols()))}
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
	if artifact.Schema != 1 || (artifact.Trial != diagnosticTrialLiveOnly && artifact.Trial != diagnosticTrialOneWorker) || len(artifact.Trials) != 1 {
		return errors.New("invalid bounded diagnostic artifact")
	}
	for _, trial := range artifact.Trials {
		if len(trial.Samples) == 0 || len(trial.Samples) > 491 {
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
	if err != nil || len(body) > 2<<20 {
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
