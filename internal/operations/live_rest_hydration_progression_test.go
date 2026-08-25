package operations

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
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
	liveRESTProgressionCycles       = 19
	liveRESTProgressionJoinDeadline = 10 * time.Second
)

type liveRESTProgressionFixture struct {
	frames        []string
	digest        [32]byte
	minimumBytes  int
	maximumBytes  int
	validPerCycle int
}

type liveRESTProgressionResult struct {
	workers, population, validPerCycle                              int
	maximumRESTActive                                               int64
	hydrationDuration, elapsed, maximumOldest, maximumDeliveryDelay time.Duration
	tailDrain                                                       time.Duration
	framesSent, framesRead, framesAdmitted, framesDispositioned     uint64
	framesRejected, framesFenced, aggregatesConsumed                uint64
	aggregatesInserted, aggregatesRevised, aggregatesRejected       uint64
	aggregatesFenced, maximumQueued                                 uint64
	hydrationPlanned, hydrationValue, hydrationEmpty                uint64
	hydrationFailed, hydrationCanceled, hydrationFenced             uint64
	hydrationRows, hydrationRowsInserted                            uint64
	ingressFencesStarted, ingressFencesDispositioned                uint64
	publicationID, publicationSequence                              uint64
	averageCPUCores                                                 float64
	fenceApplied, tailDrained, accountingReconciled                 bool
	progressDuringLive, lifecycleReady, publicationCoherent         bool
	stopReason                                                      string
	err                                                             error
	completionOutOfOrder                                            bool
	publicationProblem                                              string
	evaluation                                                      engine.EvaluationView
	tq                                                              engine.TQView
	workerAccounting                                                massive.HydrationWorkerAccounting
	terminalOrder                                                   []uint64
	maximumTerminalSequence, fenceSequence                          uint64
}

// TestPLBRA3ParallelHydration is P-LBR-A3-PARALLEL-HYDRATION. It uses the
// ordinary live composition at every supported worker count. The REST fixture
// barriers the first wave at the configured ceiling and releases it in an
// order different from the engine plan while the real fake WebSocket keeps
// advancing aggregates through the independent live consumer.
func TestPLBRA3ParallelHydration(t *testing.T) {
	binding, fixture := liveRESTProgressionInputs(t, 32)
	var baseline *liveRESTProgressionResult
	for _, workers := range []int{1, 2, 4, 8} {
		t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
			result := runLiveRESTProgression(t, binding, fixture, workers)
			assertLiveRESTProgression(t, result)
			if result.maximumRESTActive != int64(workers) {
				t.Fatalf("configured workers=%d maximum active REST requests=%d", workers, result.maximumRESTActive)
			}
			if workers > 1 && !result.completionOutOfOrder {
				t.Fatal("latency-controlled REST completion retained plan order")
			}
			if workers > 1 {
				terminalOutOfOrder := false
				for index, requestID := range result.terminalOrder {
					if requestID != uint64(index+1) {
						terminalOutOfOrder = true
						break
					}
				}
				if !terminalOutOfOrder {
					t.Fatal("latency-controlled engine terminal order retained plan order")
				}
			}
			if baseline == nil {
				baseline = &result
			} else if evaluationEqual, tqEqual := equalLBREvaluation(result.evaluation, baseline.evaluation), equalLBRTQProduct(result.tq, baseline.tq); !evaluationEqual || !tqEqual {
				t.Fatalf("workers=%d changed public projection relative to one worker: ranking=%t TQ=%t", workers, evaluationEqual, tqEqual)
			}
		})
	}
}

func equalLBREvaluation(left, right engine.EvaluationView) bool {
	left.At, right.At = time.Time{}, time.Time{}
	left.PopulationTransition, right.PopulationTransition = engine.PopulationTransitionDiagnosticView{}, engine.PopulationTransitionDiagnosticView{}
	return reflect.DeepEqual(left, right)
}

func equalLBRTQProduct(left, right engine.TQView) bool {
	left.PublicationID, right.PublicationID = 0, 0
	left.Accounting, right.Accounting = engine.TQAccountingView{}, engine.TQAccountingView{}
	left.Commands, right.Commands = engine.TQCommandAccountingView{}, engine.TQCommandAccountingView{}
	left.PressureSample, right.PressureSample = engine.TQPressureSampleView{}, engine.TQPressureSampleView{}
	return reflect.DeepEqual(left, right)
}

// TestLiveRESTHydrationProgression is the opt-in deterministic progression
// specified by docs/live-rest-hydration-progression.md. Every worker level gets
// a fresh runtime, adapter, engine, HTTP server, WebSocket server, and hydration
// generation. The fixture bytes and pacing are unchanged between levels.
func TestLiveRESTHydrationProgression(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in live plus REST hydration progression")
	}
	binding, fixture := liveRESTProgressionInputs(t, liveContentionPopulation)
	workers := []int{1, 2, 4}
	results := make([]liveRESTProgressionResult, 0, 4)
	for index := 0; index < len(workers); index++ {
		workerCount := workers[index]
		passed := t.Run(fmt.Sprintf("workers_%d", workerCount), func(t *testing.T) {
			result := runLiveRESTProgression(t, binding, fixture, workerCount)
			results = append(results, result)
			assertLiveRESTProgression(t, result)
		})
		if !passed {
			break
		}
		if workerCount == 4 {
			workers = append(workers, 8)
		}
	}
	logLiveRESTProgressionComparison(t, results)
}

// TestLiveRESTHydrationProgressionHarness runs the same one-worker production
// composition under the race detector. It is separate from the named sweep so
// the progression's race command does not execute every concurrency level.
func TestLiveRESTHydrationProgressionHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("opt-in live plus REST hydration progression harness")
	}
	binding, fixture := liveRESTProgressionInputs(t, 128)
	assertLiveRESTProgression(t, runLiveRESTProgression(t, binding, fixture, 1))
}

func liveRESTProgressionInputs(t *testing.T, population int) (reference.Binding, liveRESTProgressionFixture) {
	t.Helper()
	symbols := make([]string, population)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := capacityBinding(t, symbols)
	windowStart := binding.SessionStart().Add(8*time.Hour - 10*time.Second).Truncate(time.Second)
	fixture := buildLiveRESTProgressionFixture(t, symbols, windowStart)
	return binding, fixture
}

func buildLiveRESTProgressionFixture(t *testing.T, symbols []string, windowStart time.Time) liveRESTProgressionFixture {
	t.Helper()
	frameCount := liveContentionFrames * liveRESTProgressionCycles
	validPerCycle := min(len(symbols), liveContentionValidAggregates)
	result := liveRESTProgressionFixture{frames: make([]string, frameCount), minimumBytes: int(^uint(0) >> 1), validPerCycle: validPerCycle}
	for cycle := 0; cycle < liveRESTProgressionCycles; cycle++ {
		validIndex, unknownIndex := 0, 0
		for frameInCycle := 0; frameInCycle < liveContentionFrames; frameInCycle++ {
			items := make([]string, liveContentionAggregatesPerFrame)
			for itemIndex := range items {
				aggregateInCycle := frameInCycle*liveContentionAggregatesPerFrame + itemIndex
				validThrough := (aggregateInCycle + 1) * validPerCycle / (liveContentionFrames * liveContentionAggregatesPerFrame)
				validBefore := aggregateInCycle * validPerCycle / (liveContentionFrames * liveContentionAggregatesPerFrame)
				symbol := ""
				if validThrough > validBefore {
					symbol = symbols[validIndex]
					validIndex++
				} else {
					symbol = fmt.Sprintf("U%05d", unknownIndex)
					unknownIndex++
				}
				second := frameInCycle * 6 / liveContentionFrames
				window := windowStart.Add(time.Duration(cycle*6+second) * time.Second)
				ordinal := cycle*liveContentionFrames*liveContentionAggregatesPerFrame + aggregateInCycle
				items[itemIndex] = liveContentionAggregateJSON(symbol, window, ordinal)
			}
			frameIndex := cycle*liveContentionFrames + frameInCycle
			result.frames[frameIndex] = "[" + strings.Join(items, ",") + "]"
			result.minimumBytes = min(result.minimumBytes, len(result.frames[frameIndex]))
			result.maximumBytes = max(result.maximumBytes, len(result.frames[frameIndex]))
		}
		if validIndex != validPerCycle || unknownIndex != liveContentionFrames*liveContentionAggregatesPerFrame-validPerCycle {
			t.Fatalf("progression cycle %d manifest valid=%d unknown=%d", cycle, validIndex, unknownIndex)
		}
	}
	result.digest = sha256.Sum256([]byte(strings.Join(result.frames, "\n")))
	if len(symbols) == 0 || len(result.frames) != frameCount || result.minimumBytes < 350 || result.maximumBytes > 450 {
		t.Fatalf("progression fixture population=%d frames=%d min_bytes=%d max_bytes=%d", len(symbols), len(result.frames), result.minimumBytes, result.maximumBytes)
	}
	return result
}

type liveRESTProgressionServer struct {
	server *httptest.Server
	limit  int
	start  chan time.Time
	stop   chan struct{}
	done   chan error
	sent   atomic.Uint64
	once   sync.Once
}

func newLiveRESTProgressionServer(t *testing.T, fixture liveRESTProgressionFixture, limit int) *liveRESTProgressionServer {
	t.Helper()
	result := &liveRESTProgressionServer{limit: limit, start: make(chan time.Time, 1), stop: make(chan struct{}), done: make(chan error, 1)}
	result.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		result.once.Do(func() { result.serve(request, writer, fixture.frames) })
	}))
	return result
}

func (s *liveRESTProgressionServer) serve(request *http.Request, writer http.ResponseWriter, frames []string) {
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
	for cycle := 0; cycle < liveRESTProgressionCycles; cycle++ {
		intervalStart := time.Duration(cycle) * liveContentionDuration
		for _, interval := range liveContentionIntervals {
			for index := 0; index < interval.frames; index++ {
				if frameIndex == s.limit {
					s.done <- nil
					ticker := time.NewTicker(20 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-s.stop:
							<-ctx.Done()
							return
						case <-ticker.C:
							// Keep the socket reader crossing transport read
							// boundaries without adding another market/control
							// frame to the fixed semantic fixture. This lets an
							// ingress-fence request linearize even after the last
							// application frame has been sent.
							if err := connection.Ping(ctx); err != nil {
								return
							}
						}
					}
				}
				target := plannedStart.Add(intervalStart + time.Duration(index+1)*interval.duration/time.Duration(interval.frames))
				timer := time.NewTimer(max(time.Until(target), 0))
				select {
				case <-ctx.Done():
					if !timer.Stop() {
						<-timer.C
					}
					s.done <- ctx.Err()
					return
				case <-s.stop:
					if !timer.Stop() {
						<-timer.C
					}
					s.done <- nil
					<-ctx.Done()
					return
				case <-timer.C:
				}
				if err := write(frames[frameIndex]); err != nil {
					s.done <- err
					return
				}
				s.sent.Add(1)
				frameIndex++
			}
			intervalStart += interval.duration
		}
	}
	s.done <- errors.New("paced progression fixture exhausted before hydration completed")
}

type liveRESTFixtureServer struct {
	server                               *httptest.Server
	liveSent                             *atomic.Uint64
	seen                                 []atomic.Uint32
	requests, values, empties, afterLive atomic.Uint64
	maximumBody                          atomic.Uint64
	active, maximumActive                atomic.Int64
	workers                              int
	firstWave                            chan struct{}
	firstWaveOnce                        sync.Once
	mu                                   sync.Mutex
	completionOrder                      []int
	err                                  error
}

func newLiveRESTFixtureServer(t *testing.T, population, workers int, liveSent *atomic.Uint64) *liveRESTFixtureServer {
	t.Helper()
	result := &liveRESTFixtureServer{liveSent: liveSent, seen: make([]atomic.Uint32, population), workers: workers, firstWave: make(chan struct{})}
	result.server = httptest.NewTLSServer(http.HandlerFunc(result.serve))
	return result
}

func (s *liveRESTFixtureServer) serve(writer http.ResponseWriter, request *http.Request) {
	fail := func(err error) {
		s.mu.Lock()
		if s.err == nil {
			s.err = err
		}
		s.mu.Unlock()
		http.Error(writer, "invalid deterministic hydration request", http.StatusBadRequest)
	}
	parts := strings.Split(request.URL.Path, "/")
	if request.Method != http.MethodGet || len(parts) != 10 || parts[1] != "v2" || parts[2] != "aggs" || parts[3] != "ticker" ||
		parts[5] != "range" || parts[6] != "1" || parts[7] != "second" || request.Header.Get("Authorization") != "Bearer progression-fixture" ||
		request.URL.Query().Get("adjusted") != "false" || request.URL.Query().Get("sort") != "asc" || request.URL.Query().Get("limit") != "50000" {
		fail(fmt.Errorf("unexpected request shape path=%q query=%q", request.URL.Path, request.URL.RawQuery))
		return
	}
	index, err := strconv.Atoi(strings.TrimPrefix(parts[4], "S"))
	startMillis, startErr := strconv.ParseInt(parts[8], 10, 64)
	endMillis, endErr := strconv.ParseInt(parts[9], 10, 64)
	if err != nil || startErr != nil || endErr != nil || index < 0 || index >= len(s.seen) || parts[4] != fmt.Sprintf("S%05d", index) || endMillis < startMillis {
		fail(fmt.Errorf("unexpected request identity path=%q", request.URL.Path))
		return
	}
	if s.seen[index].Add(1) != 1 {
		fail(fmt.Errorf("duplicate request for fixture index %d", index))
		return
	}
	active := s.active.Add(1)
	updateLiveRESTMaximum(&s.maximumActive, active)
	defer s.active.Add(-1)
	if len(s.seen) <= 128 {
		if active == int64(s.workers) {
			s.firstWaveOnce.Do(func() { close(s.firstWave) })
		}
		select {
		case <-request.Context().Done():
			return
		case <-s.firstWave:
		}
		delay := time.Duration(1+(s.workers-index%s.workers)) * 5 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-request.Context().Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
	s.requests.Add(1)
	if s.liveSent.Load() > 0 {
		s.afterLive.Add(1)
	}
	body := ""
	if index%2 == 0 {
		s.values.Add(1)
		body = fmt.Sprintf(`{"status":"OK","ticker":%q,"adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":100,"vw":10.25,"n":4}]}`, parts[4], startMillis)
	} else {
		s.empties.Add(1)
		body = fmt.Sprintf(`{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, parts[4])
	}
	for old := s.maximumBody.Load(); uint64(len(body)) > old && !s.maximumBody.CompareAndSwap(old, uint64(len(body))); old = s.maximumBody.Load() {
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(body))
	s.mu.Lock()
	s.completionOrder = append(s.completionOrder, index)
	s.mu.Unlock()
}

func updateLiveRESTMaximum(target *atomic.Int64, value int64) {
	for current := target.Load(); value > current && !target.CompareAndSwap(current, value); current = target.Load() {
	}
}

func (s *liveRESTFixtureServer) failure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

type liveRESTProgressionMonitor struct {
	maximumQueued, maximumOldest uint64
	progressDuringLive           bool
	stopReason                   string
}

func monitorLiveRESTProgression(ctx context.Context, cancel context.CancelFunc, plannedStart time.Time, attempt *massive.LiveAttempt, run *Runtime,
	queueBaseline massive.LiveQueueAccounting, done <-chan struct{}, result chan<- liveRESTProgressionMonitor) {
	monitor := liveRESTProgressionMonitor{}
	boundary := time.Duration(0)
	intervalIndex := 0
	for {
		interval := liveContentionIntervals[intervalIndex%len(liveContentionIntervals)]
		boundary += interval.duration
		intervalIndex++
		timer := time.NewTimer(max(time.Until(plannedStart.Add(boundary)), 0))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			result <- monitor
			return
		case <-done:
			if !timer.Stop() {
				<-timer.C
			}
			result <- monitor
			return
		case <-timer.C:
		}
		queue := attempt.QueueAccounting()
		view := run.Engine().ObserveOperational()
		queued := queue.FramesQueued
		monitor.maximumQueued = max(monitor.maximumQueued, queued)
		monitor.maximumOldest = max(monitor.maximumOldest, uint64(queue.OldestWaitingFrameAge))
		terminal := view.Hydration.Accounting.CompletedValue + view.Hydration.Accounting.CompletedEmpty
		if queue.FramesDispositioned > queueBaseline.FramesDispositioned && terminal > 0 && view.Hydration.Accounting.Open > 0 {
			monitor.progressDuringLive = true
		}
		switch {
		case liveContentionFrameRejections(queue, queueBaseline) > 0:
			monitor.stopReason = "frame_rejection"
		case view.Connection.Integrity+view.Aggregates.Integrity+view.Transitions.IntegrityFailure+view.Hydration.Rows.Integrity > 0:
			monitor.stopReason = "integrity_terminal"
		case queue.FramesQueued >= liveContentionQueueStop:
			monitor.stopReason = "queue_threshold"
		case queue.OldestWaitingFrameAge >= 2*time.Second:
			monitor.stopReason = "oldest_frame_threshold"
		case !queue.Reconciles() || !operationalAccountingValid(view):
			monitor.stopReason = "accounting_divergence"
		case view.Suppression != "" || view.Lifecycle == "suppressed":
			monitor.stopReason = "engine_suppression"
		}
		if monitor.stopReason != "" {
			cancel()
			result <- monitor
			return
		}
	}
}

func runLiveRESTProgression(t *testing.T, binding reference.Binding, fixture liveRESTProgressionFixture, workers int) (result liveRESTProgressionResult) {
	t.Helper()
	population := len(binding.UniverseSymbols())
	result = liveRESTProgressionResult{workers: workers, population: population, validPerCycle: fixture.validPerCycle}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	liveFrameLimit := len(fixture.frames)
	if population <= 128 {
		liveFrameLimit = liveContentionFrames
	}
	live := newLiveRESTProgressionServer(t, fixture, liveFrameLimit)
	defer live.server.Close()
	rest := newLiveRESTFixtureServer(t, population, workers, &live.sent)
	defer rest.server.Close()
	hydrator, err := massive.NewHydrationWorker(rest.server.URL, func() (string, error) { return "progression-fixture", nil }, rest.server.Client())
	if err != nil {
		result.err = err
		return result
	}
	base := binding.SessionStart().Add(8 * time.Hour).Truncate(time.Second)
	var fixedClock atomic.Int64
	var movingClockStart atomic.Int64
	clock := func() time.Time {
		if fixed := fixedClock.Load(); fixed != 0 {
			return time.Unix(0, fixed).UTC()
		}
		if started := movingClockStart.Load(); started != 0 {
			return base.Add(max(time.Since(time.Unix(0, started)), 0)).UTC()
		}
		return base.UTC()
	}
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(live.server.URL, "http"), Credential: "progression-fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: clock,
	})
	if err != nil {
		result.err = err
		return result
	}
	config := DefaultConfig()
	run, err := New(ctx, binding, config, clock)
	if err != nil {
		result.err = err
		return result
	}
	shutdownRun := func() {
		shutdown, cancelShutdown := context.WithTimeout(context.Background(), liveRESTProgressionJoinDeadline)
		defer cancelShutdown()
		if shutdownErr := run.Shutdown(shutdown); shutdownErr != nil && result.err == nil {
			result.err = shutdownErr
		}
	}
	defer shutdownRun()
	var terminalMu sync.Mutex
	type terminalObservation struct{ requestID, engineSequence uint64 }
	terminalObservations := make([]terminalObservation, 0, population)
	run.afterHydrationWorker = func(accounting massive.HydrationWorkerAccounting) { result.workerAccounting = accounting }
	run.afterHydrationTerminal = func(requestID, engineSequence uint64) {
		terminalMu.Lock()
		terminalObservations = append(terminalObservations, terminalObservation{requestID: requestID, engineSequence: engineSequence})
		terminalMu.Unlock()
	}

	establish, cancelEstablish := context.WithTimeout(ctx, config.ConnectionAttemptDeadline)
	attempt, err := run.openAttempt(ctx, establish, LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: workers, RowsPerChunk: 256,
		MaximumResponseBytes: 512 << 20, MaximumNormalizedRecords: int64(population) * 57_600,
		MaximumResidentRecords: int64(workers) * 57_600, Durations: capacityDurations()}, 100)
	cancelEstablish()
	if err != nil {
		result.err = err
		return result
	}
	run.setLiveSources(attempt, adapter)
	components := productionDiagnosticComponents(adapter, hydrator, workers, population)
	queueBaseline := attempt.QueueAccounting()
	engineBaseline := run.Engine().ObserveOperational()
	deliveryMaxBaseline := run.deliveryMaxNanos.Load()
	cpuBaseline := goRuntimeCPUSeconds()
	plannedStart := time.Now()
	movingClockStart.Store(plannedStart.UnixNano())
	live.start <- plannedStart

	trialCtx, cancelTrial := context.WithCancel(ctx)
	defer cancelTrial()
	monitorDone := make(chan struct{})
	monitorResult := make(chan liveRESTProgressionMonitor, 1)
	go monitorLiveRESTProgression(trialCtx, cancelTrial, plannedStart, attempt, run, queueBaseline, monitorDone, monitorResult)

	hydrationStarted := time.Now()
	_, hydrateErr := run.hydrate(trialCtx, components, attempt, engine.HydrationFreshBootstrap)
	result.hydrationDuration = time.Since(hydrationStarted)
	if hydrateErr != nil {
		result.err = hydrateErr
	}
	// Hydration and its real ingress fence have completed. Stop the autonomous
	// sampler before defining the admitted tail boundary so a newly-started
	// periodic evaluation cannot become unbounded post-boundary work. The final
	// production coverage capture and timer are driven explicitly below.
	if result.err == nil {
		run.timerCancel()
		select {
		case <-run.timerDone:
		case <-time.After(liveRESTProgressionJoinDeadline):
			result.err = errors.New("runtime timer did not join at the measured tail boundary")
		}
	}

	drainCtx, cancelDrain := context.WithCancel(trialCtx)
	drainDone := make(chan error, 1)
	go func() {
		for {
			started := time.Now()
			delivery, ok, deliveryErr := attempt.DeliverNextToEngine(drainCtx, run.Engine())
			if ok {
				run.observeDelivery(started, delivery)
			}
			if deliveryErr != nil || !ok {
				drainDone <- deliveryErr
				return
			}
		}
	}()

	if result.err == nil {
		minimumDeadline := time.NewTimer(liveRESTProgressionJoinDeadline)
		for live.sent.Load() < liveContentionFrames && result.err == nil {
			select {
			case <-minimumDeadline.C:
				result.err = errors.New("paced live server did not complete one T1 envelope")
			case <-trialCtx.Done():
				result.err = trialCtx.Err()
			case <-time.After(time.Millisecond):
			}
		}
		if !minimumDeadline.Stop() {
			select {
			case <-minimumDeadline.C:
			default:
			}
		}
	}
	close(live.stop)
	select {
	case liveErr := <-live.done:
		if liveErr != nil && result.err == nil {
			result.err = liveErr
		}
	case <-time.After(liveRESTProgressionJoinDeadline):
		if result.err == nil {
			result.err = errors.New("paced live server did not stop")
		}
	}

	tailStarted := time.Now()
	tailDeadline := tailStarted.Add(liveContentionTailDrain)
	for time.Now().Before(tailDeadline) {
		queue := attempt.QueueAccounting()
		sent := live.sent.Load()
		if queue.FramesRead-queueBaseline.FramesRead == sent && queue.FramesDispositioned-queueBaseline.FramesDispositioned == sent &&
			queue.FramesQueued == 0 && queue.FramesClassifying == 0 {
			result.tailDrained = true
			result.tailDrain = time.Since(tailStarted)
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !result.tailDrained && result.tailDrain == 0 {
		result.tailDrain = time.Since(tailStarted)
	}

	if result.err == nil && result.tailDrained {
		if population <= 128 {
			// The proof compares exact final bytes across independently paced
			// worker runs. Pin the final engine time after the fixed live frame
			// boundary so wall-clock scheduling cannot select an adjacent second.
			fixedClock.Store(base.Add(6500 * time.Millisecond).UnixNano())
		}
		run.captureLiveCoverage(trialCtx)
		admission, completion := run.Engine().AdmitTimer(trialCtx)
		if admission != engine.AdmissionAdmitted || completion == nil {
			result.err = errors.New("final accepted-prefix timer was not admitted")
		} else {
			select {
			case <-trialCtx.Done():
				result.err = trialCtx.Err()
			case disposition := <-completion:
				if disposition.Code != engine.DispositionTimerApplied {
					result.err = fmt.Errorf("final accepted-prefix timer = %s/%s", disposition.Code, disposition.Reason)
				}
			}
		}
	}
	close(monitorDone)
	monitor := <-monitorResult
	result.maximumQueued = monitor.maximumQueued
	result.maximumOldest = time.Duration(monitor.maximumOldest)
	result.progressDuringLive = monitor.progressDuringLive || rest.afterLive.Load() > 0
	result.stopReason = monitor.stopReason

	timedQueue := attempt.QueueAccounting()
	timedSnapshot := run.Engine().ObserveSnapshot()
	timedEngine := timedSnapshot.Operational
	timedStatus := run.Status()
	timedReplay := run.Engine().ObserveSnapshot()
	result.evaluation = timedReplay.Publication.AggregateEvaluation
	result.tq = timedSnapshot.TQ
	result.elapsed = time.Since(plannedStart)
	result.framesSent = live.sent.Load()
	result.framesRead = timedQueue.FramesRead - queueBaseline.FramesRead
	result.framesAdmitted = timedQueue.FramesAdmitted - queueBaseline.FramesAdmitted
	result.framesDispositioned = timedQueue.FramesDispositioned - queueBaseline.FramesDispositioned
	result.framesRejected = liveContentionFrameRejections(timedQueue, queueBaseline)
	result.framesFenced = timedQueue.FramesFenced - queueBaseline.FramesFenced
	// Hydration rows use the same canonical aggregate accounting family. Remove
	// their separately reconciled contribution to isolate generated live facts.
	result.aggregatesConsumed = timedEngine.Aggregates.Consumed - engineBaseline.Aggregates.Consumed - timedEngine.Hydration.Rows.Consumed
	result.aggregatesInserted = timedEngine.Aggregates.Inserted - engineBaseline.Aggregates.Inserted - timedEngine.Hydration.Rows.Inserted
	result.aggregatesRevised = timedEngine.Aggregates.Revised - engineBaseline.Aggregates.Revised
	result.aggregatesRejected = timedEngine.Aggregates.Rejected - engineBaseline.Aggregates.Rejected
	result.aggregatesFenced = timedEngine.Aggregates.Fenced - engineBaseline.Aggregates.Fenced
	result.hydrationPlanned = timedEngine.Hydration.Accounting.Planned
	result.hydrationValue = timedEngine.Hydration.Accounting.CompletedValue
	result.hydrationEmpty = timedEngine.Hydration.Accounting.CompletedEmpty
	result.hydrationFailed = timedEngine.Hydration.Accounting.Failed
	result.hydrationCanceled = timedEngine.Hydration.Accounting.Canceled
	result.hydrationFenced = timedEngine.Hydration.Accounting.Fenced
	result.hydrationRows = timedEngine.Hydration.Rows.Consumed
	result.hydrationRowsInserted = timedEngine.Hydration.Rows.Inserted
	result.ingressFencesStarted = timedQueue.IngressFencesStarted - queueBaseline.IngressFencesStarted
	result.ingressFencesDispositioned = timedQueue.IngressFencesDispositioned - queueBaseline.IngressFencesDispositioned
	result.fenceApplied = timedEngine.Hydration.FenceReconciled
	result.fenceSequence = run.Engine().ObserveFenceTiming().EngineSequence
	result.publicationID = timedEngine.PublicationID
	result.publicationSequence = timedEngine.LastEngineSequence
	result.lifecycleReady = timedStatus.BackendReady && timedStatus.RankingCurrent && timedEngine.Lifecycle == "live" && timedEngine.Suppression == ""
	publication := timedSnapshot.Publication
	evaluation := publication.AggregateEvaluation
	result.publicationCoherent = publication.PublicationID == timedEngine.PublicationID && publication.LastEngineSequence <= timedEngine.LastEngineSequence &&
		publication.Lifecycle == timedEngine.Lifecycle && publication.CurrentMarketClaim && publication.Watermark != nil && timedEngine.Watermark != nil &&
		*publication.Watermark == *timedEngine.Watermark && publication.Lifecycle == "live" &&
		evaluation.Mode == "qualified_current" && evaluation.Reason == "" && evaluation.Population.UniverseTotal == uint64(population) &&
		evaluation.Population.ValidPriorClose == uint64(population) && evaluation.Population.CoveredPopulation == uint64(population) &&
		evaluation.Population.UnresolvedPopulation == 0 && evaluation.TotalPassers == 0 && len(evaluation.Rows) == 0 &&
		timedSnapshot.TQ.PublicationID == publication.PublicationID && len(timedSnapshot.TQ.Desired) == 0 && len(timedSnapshot.TQ.Rows) == 0
	if !result.publicationCoherent {
		result.publicationProblem = fmt.Sprintf("snapshot_pub=%d/%d engine_pub=%d/%d lifecycle=%s/%s current=%t watermark=%v/%v population=%+v",
			publication.PublicationID, publication.LastEngineSequence, timedEngine.PublicationID, timedEngine.LastEngineSequence,
			publication.Lifecycle, timedEngine.Lifecycle, publication.CurrentMarketClaim,
			publication.Watermark, timedEngine.Watermark, publication.AggregateEvaluation.Population)
	}
	if maximum := run.deliveryMaxNanos.Load(); maximum > deliveryMaxBaseline {
		result.maximumDeliveryDelay = time.Duration(maximum)
	}
	cpuSeconds := goRuntimeCPUSeconds() - cpuBaseline
	if result.elapsed > 0 && cpuSeconds >= 0 {
		result.averageCPUCores = cpuSeconds / result.elapsed.Seconds()
	}

	cancelDrain()
	select {
	case <-drainDone:
	case <-time.After(liveRESTProgressionJoinDeadline):
		if result.err == nil {
			result.err = errors.New("post-hydration delivery loop did not join")
		}
	}
	closeCommand := massive.CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 190, Cause: massive.CloseControlledStop}
	if closeErr := attempt.Close(closeCommand); closeErr != nil && result.err == nil {
		result.err = closeErr
	}
	run.closeAndDrain(attempt, closeCommand.CommandToken, closeCommand.Cause)
	reconciled := run.Metrics()
	result.accountingReconciled = reconciled.LiveQueue.Reconciles() && reconciled.Adapter.Reconciles() && reconciled.Engine.Admissions.Reconciles(reconciled.Engine.QueueOccupancy) &&
		reconciled.Engine.Transitions.Reconciles() && reconciled.Engine.Aggregates.Reconciles() && operationalAccountingValid(reconciled.Engine) && reconciled.AccountingValid &&
		reconciled.LiveQueue.FramesQueued == 0 && reconciled.LiveQueue.FramesClassifying == 0 && reconciled.LiveQueue.IngressFencesQueued == 0 &&
		reconciled.LiveQueue.IngressFencesClassifying == 0 && reconciled.Engine.QueueOccupancy == 0
	if restErr := rest.failure(); restErr != nil && result.err == nil {
		result.err = restErr
	}
	result.maximumRESTActive = rest.maximumActive.Load()
	rest.mu.Lock()
	for index, completed := range rest.completionOrder {
		if completed != index {
			result.completionOutOfOrder = true
			break
		}
	}
	rest.mu.Unlock()
	terminalMu.Lock()
	sort.Slice(terminalObservations, func(left, right int) bool {
		return terminalObservations[left].engineSequence < terminalObservations[right].engineSequence
	})
	result.terminalOrder = make([]uint64, len(terminalObservations))
	for index, observation := range terminalObservations {
		result.terminalOrder[index] = observation.requestID
		result.maximumTerminalSequence = max(result.maximumTerminalSequence, observation.engineSequence)
	}
	terminalMu.Unlock()
	if rest.requests.Load() != uint64(population) || rest.values.Load() != uint64((population+1)/2) || rest.empties.Load() != uint64(population/2) || rest.maximumBody.Load() > 256 {
		if result.err == nil {
			result.err = fmt.Errorf("REST fixture accounting requests=%d values=%d empty=%d max_body=%d", rest.requests.Load(), rest.values.Load(), rest.empties.Load(), rest.maximumBody.Load())
		}
	}
	for index := range rest.seen {
		if rest.seen[index].Load() != 1 {
			if result.err == nil {
				result.err = fmt.Errorf("REST fixture symbol index %d requests=%d", index, rest.seen[index].Load())
			}
			break
		}
	}
	return result
}

func assertLiveRESTProgression(t *testing.T, result liveRESTProgressionResult) {
	t.Helper()
	wantValid, wantUnknown := liveRESTProgressionAggregateCounts(result.framesSent, result.validPerCycle)
	problems := make([]string, 0)
	if result.err != nil {
		problems = append(problems, result.err.Error())
	}
	if result.stopReason != "" {
		problems = append(problems, "safety stop="+result.stopReason)
	}
	if result.framesSent < liveContentionFrames || result.framesRead != result.framesSent || result.framesAdmitted != result.framesSent || result.framesDispositioned != result.framesSent || result.framesRejected != 0 || result.framesFenced != 0 {
		problems = append(problems, fmt.Sprintf("live frames sent/read/admitted/dispositioned/rejected/fenced=%d/%d/%d/%d/%d/%d", result.framesSent, result.framesRead, result.framesAdmitted, result.framesDispositioned, result.framesRejected, result.framesFenced))
	}
	if result.aggregatesConsumed != result.framesSent*liveContentionAggregatesPerFrame || result.aggregatesInserted != wantValid || result.aggregatesRejected != wantUnknown || result.aggregatesRevised != 0 || result.aggregatesFenced != 0 {
		problems = append(problems, fmt.Sprintf("aggregates consumed/inserted/rejected/revised/fenced=%d/%d/%d/%d/%d want=%d/%d/%d/0/0", result.aggregatesConsumed, result.aggregatesInserted, result.aggregatesRejected, result.aggregatesRevised, result.aggregatesFenced, result.framesSent*liveContentionAggregatesPerFrame, wantValid, wantUnknown))
	}
	if result.maximumQueued >= liveContentionQueueStop || result.maximumOldest >= 2*time.Second || !result.tailDrained || result.tailDrain > liveContentionTailDrain {
		problems = append(problems, fmt.Sprintf("queue max=%d oldest=%s tail_drained=%t tail=%s", result.maximumQueued, result.maximumOldest, result.tailDrained, result.tailDrain))
	}
	wantValues, wantEmpty := uint64((result.population+1)/2), uint64(result.population/2)
	if result.hydrationPlanned != uint64(result.population) || result.hydrationValue != wantValues || result.hydrationEmpty != wantEmpty ||
		result.hydrationFailed != 0 || result.hydrationCanceled != 0 || result.hydrationFenced != 0 || result.hydrationRows != wantValues || result.hydrationRowsInserted != wantValues {
		problems = append(problems, fmt.Sprintf("hydration planned/value/empty/failed/canceled/fenced/rows/inserted=%d/%d/%d/%d/%d/%d/%d/%d", result.hydrationPlanned, result.hydrationValue, result.hydrationEmpty, result.hydrationFailed, result.hydrationCanceled, result.hydrationFenced, result.hydrationRows, result.hydrationRowsInserted))
	}
	if !result.progressDuringLive || !result.fenceApplied || result.ingressFencesStarted == 0 || result.ingressFencesStarted != result.ingressFencesDispositioned {
		problems = append(problems, fmt.Sprintf("progress=%t fence=%t ingress_fences=%d/%d", result.progressDuringLive, result.fenceApplied, result.ingressFencesStarted, result.ingressFencesDispositioned))
	}
	worker := result.workerAccounting
	if worker.ItemsStarted != int64(result.population) || worker.MaximumActiveWorkers != int64(result.workers) ||
		worker.MaximumResidentRecords > int64(result.workers)*57_600 || worker.ResponseBytes <= 0 || worker.ResponseBytes > 512<<20 ||
		worker.NormalizedRows != int64(wantValues) || len(result.terminalOrder) != result.population ||
		result.maximumTerminalSequence == 0 || result.fenceSequence <= result.maximumTerminalSequence {
		problems = append(problems, fmt.Sprintf("worker accounting=%+v terminal_order=%d terminal_max=%d fence_sequence=%d", worker, len(result.terminalOrder), result.maximumTerminalSequence, result.fenceSequence))
	}
	if !result.lifecycleReady || !result.publicationCoherent || result.publicationID == 0 || result.publicationSequence == 0 {
		problems = append(problems, fmt.Sprintf("ready=%t publication_coherent=%t publication=%d sequence=%d detail=%s", result.lifecycleReady, result.publicationCoherent, result.publicationID, result.publicationSequence, result.publicationProblem))
	}
	if !result.accountingReconciled {
		problems = append(problems, "adapter/queue/engine accounting did not reconcile at shutdown")
	}
	if len(problems) != 0 {
		t.Fatalf("workers=%d progression failed: %s", result.workers, strings.Join(problems, "; "))
	}
}

func liveRESTProgressionAggregateCounts(frames uint64, validPerCycle int) (valid, unknown uint64) {
	aggregatesPerCycle := uint64(liveContentionFrames * liveContentionAggregatesPerFrame)
	aggregates := frames * liveContentionAggregatesPerFrame
	fullCycles, remainder := aggregates/aggregatesPerCycle, aggregates%aggregatesPerCycle
	valid = fullCycles*uint64(validPerCycle) + remainder*uint64(validPerCycle)/aggregatesPerCycle
	return valid, aggregates - valid
}

func logLiveRESTProgressionComparison(t *testing.T, results []liveRESTProgressionResult) {
	t.Helper()
	if len(results) == 0 {
		return
	}
	t.Log("workers | hydration | live frames/s | aggregates/s | max queued | max oldest | max delivery | tail drain | fence | accounting | outcome")
	safe := 0
	for _, result := range results {
		outcome := "pass"
		if result.err != nil || result.stopReason != "" || !result.accountingReconciled || !result.lifecycleReady || !result.publicationCoherent || !result.tailDrained {
			outcome = "fail"
		} else {
			safe = result.workers
		}
		seconds := max(result.elapsed.Seconds(), 0.001)
		t.Logf("%d | %s | %.1f | %.1f | %d | %s | %s | %s | %t | %t | %s (cpu=%.2f cores)", result.workers, result.hydrationDuration,
			float64(result.framesDispositioned)/seconds, float64(result.aggregatesConsumed)/seconds, result.maximumQueued, result.maximumOldest,
			result.maximumDeliveryDelay, result.tailDrain, result.fenceApplied, result.accountingReconciled, outcome, result.averageCPUCores)
	}
	for index := 1; index < len(results); index++ {
		lower, higher := results[index-1], results[index]
		if safe < higher.workers || lower.hydrationDuration == 0 || higher.hydrationDuration < lower.hydrationDuration*9/10 {
			continue
		}
		worseQueue := higher.maximumQueued >= lower.maximumQueued+4 && higher.maximumQueued*4 > lower.maximumQueued*5
		worseAge := higher.maximumOldest >= lower.maximumOldest+10*time.Millisecond && higher.maximumOldest*4 > lower.maximumOldest*5
		worseDelay := higher.maximumDeliveryDelay >= lower.maximumDeliveryDelay+5*time.Millisecond && higher.maximumDeliveryDelay*4 > lower.maximumDeliveryDelay*5
		worseCPU := higher.averageCPUCores > lower.averageCPUCores*1.10
		if worseQueue || worseAge || worseDelay || worseCPU {
			safe = lower.workers
			break
		}
	}
	t.Logf("selected safe local hydration concurrency=%d workers (test evidence only; production default unchanged)", safe)
}
