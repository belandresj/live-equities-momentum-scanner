package operations

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
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
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact/playback"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const (
	cachedFenceArtifact        = "/Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/aggregate-replay/aggregate-replay-fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a.jsonl"
	cachedFenceReference       = "/Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/reference"
	cachedFenceArtifactID      = "sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a"
	cachedFenceRowsPerChunk    = 256
	cachedFenceObservedPeriod  = 30 * time.Second / 5_798
	cachedFenceMaximumInFlight = 128
)

type cachedFenceManifest struct {
	binding               reference.Binding
	start, end            time.Time
	metadata              replayartifact.Metadata
	handle                *replayartifact.Handle
	cursor                *playback.Cursor
	rowsBySymbol          map[string]int64
	rows, values, empties int64
	preflight             time.Duration
}

type cachedFenceRunResult struct {
	rate                                                        int
	historical, terminals, fence, firstReady, tailDrain         time.Duration
	rows, values, empties                                       uint64
	framesSent, framesRead, framesAdmitted, framesDispositioned uint64
	framesRejected, maximumQueued                               uint64
	rowsObserved, rankingRows                                   int
	lifecycle, rankingMode                                      string
	accounting, ready, fenceReconciled                          bool
	heapBefore, heapAtFence, heapAfter                          uint64
	tq                                                          engine.TQView
	cycleStage, cycleApply, cyclePublication, cycleLock         [10]time.Duration
	cycleAlloc                                                  [10]uint64
	cyclePublicationID                                          [10]uint64
}

type cachedArtifactLine struct {
	Kind        string `json:"kind"`
	Symbol      string `json:"symbol"`
	WindowStart string `json:"window_start"`
	WindowEnd   string `json:"window_end"`
}

func TestCachedHydrationFencePreflight(t *testing.T) {
	if testing.Short() || os.Getenv("CACHED_HYDRATION_PREFLIGHT") != "1" {
		t.Skip("set CACHED_HYDRATION_PREFLIGHT=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	started := time.Now()
	info, err := os.Stat(cachedFenceArtifact)
	if err != nil || info.Size() != 2_584_011_150 {
		t.Fatalf("artifact stat size=%v err=%v", func() int64 {
			if info == nil {
				return -1
			}
			return info.Size()
		}(), err)
	}
	hashFile, err := os.Open(cachedFenceArtifact)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, err = io.Copy(digest, hashFile)
	closeErr := hashFile.Close()
	if err != nil || closeErr != nil || fmt.Sprintf("%x", digest.Sum(nil)) != "e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189" {
		t.Fatalf("artifact digest err=%v close=%v sha=%x", err, closeErr, digest.Sum(nil))
	}
	t.Logf("preflight file bytes/digest validated in %s", time.Since(started))
	for _, path := range []string{cachedFenceReference + "/prior-close/2026-08-06.json", cachedFenceReference + "/universe/2026-08-07.json"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("reference %s: %v", path, err)
		}
	}
	binding, err := cachedFenceBinding(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("preflight binding/references validated in %s", time.Since(started))
	if binding.Identity() != "session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d" || len(binding.UniverseSymbols()) != 5_691 ||
		binding.SessionStart() != time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC) || binding.SessionEnd() != time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("recorded manifest binding=%s population=%d interval=[%s,%s)", binding.Identity(), len(binding.UniverseSymbols()), binding.SessionStart(), binding.SessionEnd())
	}
	t.Logf("GATE_E_PREFLIGHT path=%s bytes=%d file_sha256=%x artifact=%s binding=%s records=%d symbols=%d reference_dates=2026-08-06,2026-08-07 interval=[%s,%s) duration=%s", cachedFenceArtifact, info.Size(), digest.Sum(nil), cachedFenceArtifactID, binding.Identity(), 7_671_171, 5_691, binding.SessionStart().Format(time.RFC3339), binding.SessionEnd().Format(time.RFC3339), time.Since(started))
}

// TestCachedHydrationFencePreparedPlayback proves only that the corrected
// production reader prepares one trusted 17:15 prefix and completes its
// same-open streaming/suffix pass for the exact artifact. It constructs no
// engine, hydration runtime, live adapter, fence, publication, or Gate E cycle.
func TestCachedHydrationFencePreparedPlayback(t *testing.T) {
	if testing.Short() || os.Getenv("CACHED_HYDRATION_PREPARED_PLAYBACK") != "1" {
		t.Skip("set CACHED_HYDRATION_PREPARED_PLAYBACK=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute+45*time.Second)
	defer cancel()
	manifest := preflightCachedFence(t, ctx)
	defer manifest.handle.Close()
	if manifest.rows != 7_581_690 || manifest.values != 5_439 || manifest.empties != 63 {
		t.Fatalf("prepared prefix rows=%d values=%d empty=%d", manifest.rows, manifest.values, manifest.empties)
	}
	counts := make(map[string]int64, len(manifest.rowsBySymbol))
	for symbol := range manifest.rowsBySymbol {
		counts[symbol] = 0
	}
	started := time.Now()
	rows, err := scanCachedFencePlayback(ctx, manifest.cursor, manifest.start, manifest.end, counts, nil)
	if err != nil || rows != manifest.rows {
		t.Fatalf("prepared playback rows=%d expected=%d err=%v", rows, manifest.rows, err)
	}
	values := int64(0)
	for _, count := range counts {
		if count > 0 {
			values++
		}
	}
	if values != manifest.values {
		t.Fatalf("prepared playback values=%d expected=%d", values, manifest.values)
	}
	t.Logf("GATE_E_READER_ONLY_PREPARED rows=%d values=%d empty=%d prepare=%s stream_and_suffix=%s total=%s", manifest.rows,
		manifest.values, manifest.empties, manifest.preflight, time.Since(started), manifest.preflight+time.Since(started))
}

// TestCachedHydrationFenceAcceptance is the explicitly selected, no-network
// production-path acceptance in docs/live-fence-finalization-cached-hydration-correction.md.
// It is intentionally excluded from ordinary and generic non-short test runs.
func TestCachedHydrationFenceAcceptance(t *testing.T) {
	if testing.Short() || os.Getenv("CACHED_HYDRATION_ACCEPTANCE") != "1" {
		t.Skip("set CACHED_HYDRATION_ACCEPTANCE=1 to run the sealed cached-hydration acceptance")
	}
	rate := 1
	if raw := os.Getenv("CACHED_HYDRATION_RATE_MULTIPLIER"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || (value != 1 && value != 2) {
			t.Fatal("CACHED_HYDRATION_RATE_MULTIPLIER must be 1 or 2")
		}
		rate = value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute+30*time.Second)
	defer cancel()
	manifest := preflightCachedFence(t, ctx)
	defer manifest.handle.Close()
	result := runCachedFence(t, ctx, manifest, rate)
	assertCachedFence(t, manifest, result)
	t.Logf("cached_fence rate=%dx binding=%s artifact=%s interval=[%s,%s) population=%d rows=%d values=%d empty=%d preflight=%s historical=%s terminals=%s fence=%s first_ready=%s tail_drain=%s lifecycle=%s ranking=%s ranking_rows=%d queue_high=%d rejections=%d frames=%d/%d/%d/%d heap_before=%d heap_fence=%d heap_after_gc=%d tq_mode=%s tq_rows=%d accounting=%t ready=%t",
		result.rate, manifest.binding.Identity(), manifest.metadata.ArtifactID, manifest.start.Format(time.RFC3339), manifest.end.Format(time.RFC3339),
		len(manifest.rowsBySymbol), manifest.rows, manifest.values, manifest.empties, manifest.preflight, result.historical, result.terminals,
		result.fence, result.firstReady, result.tailDrain, result.lifecycle, result.rankingMode, result.rankingRows, result.maximumQueued,
		result.framesRejected, result.framesSent, result.framesRead, result.framesAdmitted, result.framesDispositioned,
		result.heapBefore, result.heapAtFence, result.heapAfter, result.tq.Pressure, len(result.tq.Rows), result.accounting, result.ready)
}

// TestCachedHydrationFenceAutomaticTimerRehearsal is the compact prerequisite
// for another full Gate E execution. It uses the production runtime and live
// adapter, completes real hydration terminals and the adapter ingress fence,
// observes readiness, advances ten distinct logical seconds only through the
// runtime-owned automatic timer, and exercises the same deterministic teardown.
func TestCachedHydrationFenceAutomaticTimerRehearsal(t *testing.T) {
	productionCadence := os.Getenv("CACHED_HYDRATION_REHEARSAL_PRODUCTION_CADENCE") == "1"
	rehearsalTimeout := 10 * time.Second
	if productionCadence {
		rehearsalTimeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), rehearsalTimeout)
	defer cancel()
	binding := operationsBinding(t)
	base := binding.SessionStart().Add(20 * time.Minute).UTC()
	clock := &cachedFenceClock{now: base}
	server := newCachedFenceServer(t, []string{"AAA"}, base, 1)
	defer server.server.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint:   "ws" + strings.TrimPrefix(server.server.URL, "http"),
		Credential: "cached-rehearsal-fixture",
		Queue:      massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20},
		Clock:      clock.read,
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.SampleCadence = 20 * time.Millisecond
	if productionCadence {
		config.SampleCadence = time.Second
	}
	run, err := New(ctx, binding, config, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	run.Engine().ArmEvaluationTimingForTest(time.Now)
	cleanup := &cachedFenceRuntimeCleanup{run: run, binding: binding}
	defer cleanup.shutdown(t)
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	cleanup.attempt = attempt
	if delivered, err := massive.DeliverToEngine(ctx, run.Engine(), started); err != nil || delivered.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("rehearsal open delivery=%+v err=%v", delivered, err)
	}
	handshake, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		got, deliveryErr := massive.DeliverToEngine(ctx, run.Engine(), delivery)
		if deliveryErr != nil || (got.ControlDisposition.Code != engine.DispositionConnectionControlApplied && got.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("rehearsal handshake delivery=%+v err=%v", got, deliveryErr)
		}
	}
	run.setLiveSources(attempt, adapter)
	queueBase := attempt.QueueAccounting()
	server.framesRead = func() uint64 { return attempt.QueueAccounting().FramesRead - queueBase.FramesRead }
	deliveryCtx, cancelDelivery := context.WithCancel(ctx)
	deliveryDone := make(chan error, 1)
	cleanup.deliveryCancel, cleanup.deliveryDone = cancelDelivery, deliveryDone
	fenceDone := make(chan engine.HydrationDisposition, 1)
	go func() {
		for {
			started := time.Now()
			delivery, ok, deliveryErr := attempt.DeliverNextToEngine(deliveryCtx, run.Engine())
			if ok {
				run.observeDelivery(started, delivery)
				if delivery.HydrationDisposition.Code != "" {
					select {
					case fenceDone <- delivery.HydrationDisposition:
					default:
					}
				}
			}
			if deliveryErr != nil || !ok {
				deliveryDone <- deliveryErr
				return
			}
		}
	}()
	planAdmission, planCompletion := run.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{
		SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(),
		Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600},
	})
	if planAdmission != engine.AdmissionAdmitted || planCompletion == nil {
		t.Fatalf("rehearsal hydration plan admission=%s", planAdmission)
	}
	plan := <-planCompletion
	if plan.Code != engine.DispositionHydrationPlanApplied || len(plan.Plan.Requests()) != 1 {
		t.Fatalf("rehearsal hydration plan=%+v", plan)
	}
	if err := clock.advance(base.Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	close(server.start)
	var fenceCommand engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 1, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		admission, completion := run.Engine().AdmitHydrationTerminal(ctx, terminal)
		if admission != engine.AdmissionAdmitted || completion == nil {
			t.Fatalf("rehearsal hydration terminal admission=%s", admission)
		}
		got := <-completion
		if got.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("rehearsal hydration terminal=%s/%s", got.Code, got.Reason)
		}
		if got.FenceCommand.CommandToken() != 0 {
			fenceCommand = got.FenceCommand
		}
	}
	if fenceCommand.CommandToken() == 0 {
		t.Fatal("rehearsal hydration did not issue an ingress fence")
	}
	capture, err := massive.CaptureAggregateIngressFenceCommandFromEngine(fenceCommand)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.CaptureAggregateIngressFence(ctx, run.Engine(), capture); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case disposition := <-fenceDone:
		if disposition.Code != engine.DispositionAggregateIngressFenceApplied {
			t.Fatalf("rehearsal fence=%s/%s", disposition.Code, disposition.Reason)
		}
	}
	for !run.Status().BackendReady {
		select {
		case <-ctx.Done():
			t.Fatalf("rehearsal readiness: %v engine=%+v", ctx.Err(), run.Engine().ObserveOperational())
		case <-time.After(time.Millisecond):
		}
	}
	close(server.pause)
	select {
	case err := <-server.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("rehearsal producer did not pause")
	}
	for {
		queue := attempt.QueueAccounting()
		if queue.FramesQueued == 0 && queue.FramesClassifying == 0 && queue.FramesRead-queueBase.FramesRead == queue.FramesDispositioned-queueBase.FramesDispositioned {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("rehearsal tail did not drain: %+v", queue)
		case <-time.After(time.Millisecond):
		}
	}
	priorTimingSequence := run.Engine().ObserveEvaluationTiming().EngineSequence
	priorPublicationID := run.Engine().ObserveOperational().PublicationID
	automaticTimer := observeCachedFenceAutomaticTimers(run)
	defer run.setAutomaticTimerObserver(nil)
	cycleTimeout := time.Second
	if config.SampleCadence == time.Second {
		cycleTimeout = 5 * time.Second
	}
	for cycle := 0; cycle < 10; cycle++ {
		cycleCtx, stopCycle := context.WithTimeout(ctx, cycleTimeout)
		automatic, err := awaitCachedFenceAutomaticCycle(cycleCtx, run, clock, automaticTimer, base.Add(time.Duration(cycle+6)*time.Second), priorTimingSequence, priorPublicationID)
		stopCycle()
		if err != nil {
			t.Fatalf("rehearsal cycle %d: %v", cycle+1, err)
		}
		view := automatic.capture
		if view.Engine.Publication.PublicationID != automatic.publicationID || automatic.total >= 2*time.Second {
			t.Fatalf("rehearsal cycle %d timer_publication=%d API_publication=%d total=%s", cycle+1, automatic.publicationID, view.Engine.Publication.PublicationID, automatic.total)
		}
		priorTimingSequence, priorPublicationID = automatic.timing.EngineSequence, automatic.publicationID
	}
	metrics := run.Metrics()
	if !metrics.AccountingValid || !metrics.LiveQueue.Reconciles() || !metrics.Adapter.Reconciles() || run.Engine().ObserveOperational().Lifecycle != "live" {
		t.Fatalf("rehearsal final accounting metrics=%+v engine=%+v", metrics, run.Engine().ObserveOperational())
	}
	cleanup.shutdown(t)
	if status := run.Status(); status.ProcessLive || status.BackendReady || status.Reason != ReasonRuntimeUnavailable {
		t.Fatalf("rehearsal deterministic shutdown status=%+v", status)
	}
}

func preflightCachedFence(t *testing.T, ctx context.Context) cachedFenceManifest {
	t.Helper()
	started := time.Now()
	binding, err := cachedFenceBinding(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start := binding.SessionStart()
	end := time.Date(2026, 8, 7, 21, 15, 0, 0, time.UTC) // 17:15 America/New_York.
	handle, err := replayartifact.OpenValidatedContext(ctx, cachedFenceArtifact, replayartifact.ValidationPlan{
		Binding: binding, Start: binding.SessionStart(), End: binding.SessionEnd(), ExpectedMode: replayartifact.CompleteFinalBars,
		MaximumBytes: 3 << 30, MaximumRecords: 8_000_000,
	})
	if err != nil {
		t.Fatalf("validate sealed artifact: %v", err)
	}
	prepared := false
	defer func() {
		if !prepared {
			_ = handle.Close()
		}
	}()
	metadata := handle.Metadata()
	if metadata.ArtifactID != cachedFenceArtifactID || metadata.AggregateRecords != 7_671_171 || metadata.CoverageEntries != 5_691 || metadata.EmptySymbols != 172 ||
		metadata.BindingIdentity != binding.Identity() || metadata.ReplayStart != binding.SessionStart() || metadata.ReplayEnd != binding.SessionEnd() {
		t.Fatalf("sealed artifact identity mismatch: %+v binding=%s", metadata, binding.Identity())
	}
	cursor, err := handle.BeginPlaybackThroughContext(ctx, end)
	if err != nil {
		t.Fatalf("prepare trusted requested prefix: %v", err)
	}
	prefix, err := cursor.RequestedPrefix()
	if err != nil || !prefix.Valid() || prefix.ArtifactID() != metadata.ArtifactID || prefix.BindingID() != binding.Identity() || prefix.RequestedEnd() != end {
		t.Fatalf("trusted requested prefix invalid: valid=%t artifact=%s binding=%s requested_end=%s records=%d err=%v",
			prefix.Valid(), prefix.ArtifactID(), prefix.BindingID(), prefix.RequestedEnd(), prefix.PrefixRecords(), err)
	}
	prefixCounts := prefix.RecordCounts()
	rowsBySymbol := make(map[string]int64)
	rows := int64(0)
	for _, fact := range binding.PriorCloseFacts() {
		if fact.Status() == reference.PriorCloseValid {
			count, ok := prefixCounts[fact.Symbol()]
			if !ok || count < 0 {
				t.Fatalf("trusted requested prefix lacks %s", fact.Symbol())
			}
			rowsBySymbol[fact.Symbol()] = count
			rows += count
		}
	}
	if len(rowsBySymbol) != 5_502 {
		t.Fatalf("valid-prior population=%d want=5502", len(rowsBySymbol))
	}
	values := int64(0)
	for _, count := range rowsBySymbol {
		if count > 0 {
			values++
		}
	}
	t.Logf("GATE_E_PREPARED artifact=%s requested_end=%s prefix_records=%d selected_rows=%d values=%d empty=%d duration=%s", metadata.ArtifactID,
		end.Format(time.RFC3339), prefix.PrefixRecords(), rows, values, int64(len(rowsBySymbol))-values, time.Since(started))
	prepared = true
	return cachedFenceManifest{binding: binding, start: start, end: end, metadata: metadata, handle: handle, cursor: cursor, rowsBySymbol: rowsBySymbol,
		rows: rows, values: values, empties: int64(len(rowsBySymbol)) - values, preflight: time.Since(started)}
}

func cachedFenceBinding(ctx context.Context) (reference.Binding, error) {
	schedule, err := session.Load()
	if err != nil {
		return reference.Binding{}, err
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		return reference.Binding{}, err
	}
	universe, err := (&reference.Resolver{DataDir: cachedFenceReference, Schedule: schedule}).Resolve(ctx, facts)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("cached universe: %w", err)
	}
	priors, err := (&reference.PriorCloseResolver{DataDir: cachedFenceReference, Schedule: schedule}).Resolve(ctx, facts, universe)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("cached prior closes: %w", err)
	}
	return reference.AssembleBinding(facts, universe, priors)
}

func scanCachedFencePlayback(ctx context.Context, cursor *playback.Cursor, start, end time.Time, counts map[string]int64, consume func(playback.RecordEvidence) error) (int64, error) {
	if evidence, err := cursor.StartContext(ctx); err != nil || !evidence.Valid() || !evidence.Complete() {
		return 0, fmt.Errorf("playback start: %v", err)
	}
	var total int64
	for group := start; !group.After(end); group = group.Add(time.Second) {
		for {
			record, ok, err := cursor.NextRecordContext(ctx, group)
			if err != nil {
				return total, err
			}
			if !ok {
				break
			}
			if _, selected := counts[record.Symbol()]; selected && record.WindowStart().Before(end) {
				counts[record.Symbol()]++
				total++
				if consume != nil {
					if err := consume(record); err != nil {
						return total, err
					}
				}
			}
		}
		if _, err := cursor.FinishGroupContext(ctx, group); err != nil {
			return total, err
		}
	}
	if evidence, err := cursor.RequestedEndContext(ctx); err != nil || !evidence.Valid() || evidence.RequestedEnd() != end {
		return total, fmt.Errorf("playback requested end: %v", err)
	}
	return total, nil
}

type cachedFenceClock struct {
	mu  sync.RWMutex
	now time.Time
}

func TestCachedFenceClockRejectsRegression(t *testing.T) {
	start := time.Date(2026, 8, 7, 21, 15, 5, 0, time.UTC)
	clock := &cachedFenceClock{now: start}
	if err := clock.advance(start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := clock.advance(start); err == nil {
		t.Fatal("cached fence clock admitted a regression")
	}
	if got := clock.read(); got != start.Add(time.Second) {
		t.Fatalf("rejected regression mutated clock to %s", got)
	}
}

func (c *cachedFenceClock) read() time.Time { c.mu.RLock(); defer c.mu.RUnlock(); return c.now }

func (c *cachedFenceClock) advance(value time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value.Before(c.now) {
		return fmt.Errorf("cached fence clock regression: %s -> %s", c.now.Format(time.RFC3339Nano), value.Format(time.RFC3339Nano))
	}
	c.now = value
	return nil
}

type cachedFenceRuntimeCleanup struct {
	run            *Runtime
	attempt        *massive.LiveAttempt
	binding        reference.Binding
	deliveryCancel context.CancelFunc
	deliveryDone   <-chan error
	deliveryJoined bool
	once           sync.Once
}

func (c *cachedFenceRuntimeCleanup) shutdown(t *testing.T) {
	t.Helper()
	c.once.Do(func() {
		deadline, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if c.attempt != nil {
			closeCommand := massive.CloseEpochCommand{BindingIdentity: c.binding.Identity(), ConnectionEpoch: c.attempt.Epoch(), CommandToken: 999, Cause: massive.CloseControlledStop}
			closeErr := c.attempt.Close(closeCommand)
			if closeErr != nil && c.deliveryCancel != nil {
				c.deliveryCancel()
			}
			if c.deliveryDone != nil && !c.deliveryJoined {
				deliveryDeadline := time.NewTimer(4 * time.Second)
				select {
				case <-c.deliveryDone:
					c.deliveryJoined = true
					if !deliveryDeadline.Stop() {
						select {
						case <-deliveryDeadline.C:
						default:
						}
					}
				case <-deliveryDeadline.C:
					if c.deliveryCancel != nil {
						c.deliveryCancel()
					}
					cancelJoinDeadline := time.NewTimer(time.Second)
					select {
					case <-c.deliveryDone:
						c.deliveryJoined = true
						if !cancelJoinDeadline.Stop() {
							select {
							case <-cancelJoinDeadline.C:
							default:
							}
						}
					case <-cancelJoinDeadline.C:
						t.Errorf("delivery loop did not join during cleanup")
					}
				case <-deadline.Done():
					t.Errorf("delivery loop did not join during cleanup")
				}
			}
			if c.deliveryCancel != nil {
				c.deliveryCancel()
			}
		}
		if err := c.run.Shutdown(deadline); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
}

type cachedFenceCycle struct {
	timing        engine.EvaluationTimingView
	total         time.Duration
	publicationID uint64
	capture       SnapshotCaptureView
}

func observeCachedFenceAutomaticTimers(run *Runtime) <-chan automaticTimerObservation {
	observations := make(chan automaticTimerObservation, 2)
	run.setAutomaticTimerObserver(func(observation automaticTimerObservation) {
		select {
		case observations <- observation:
			return
		default:
		}
		select {
		case <-observations:
		default:
		}
		select {
		case observations <- observation:
		default:
		}
	})
	return observations
}

func awaitCachedFenceAutomaticCycle(ctx context.Context, run *Runtime, clock *cachedFenceClock, observations <-chan automaticTimerObservation, sampledAt time.Time, priorTimingSequence, priorPublicationID uint64) (cachedFenceCycle, error) {
	if err := clock.advance(sampledAt); err != nil {
		return cachedFenceCycle{}, err
	}
	target := sampledAt.Add(-run.config.EvaluationDelay).Truncate(time.Second)
	if target.Before(run.binding.SessionStart()) {
		target = run.binding.SessionStart()
	}
	if target.After(run.binding.SessionEnd()) {
		target = run.binding.SessionEnd()
	}
	var lastPublication engine.ReplayPublicationView
	var lastTiming engine.EvaluationTimingView
	for {
		var observation automaticTimerObservation
		select {
		case <-ctx.Done():
			return cachedFenceCycle{}, fmt.Errorf("automatic timer target %s: %w; last publication id=%d sequence=%d disposition=%s watermark=%v timing_sequence=%d prior_sequence=%d prior_publication=%d",
				target.Format(time.RFC3339), ctx.Err(), lastPublication.PublicationID, lastPublication.LastEngineSequence, lastPublication.LastDisposition,
				lastPublication.Watermark, lastTiming.EngineSequence, priorTimingSequence, priorPublicationID)
		case observation = <-observations:
		}
		captured, captureOK := InspectSnapshotCapture(observation.capture)
		timing := observation.timing
		operational := captured.Engine.Operational
		publication := captured.Engine.Publication
		lastPublication, lastTiming = publication, timing
		if operational.Lifecycle == "suppressed" || operational.Lifecycle == "ended" || operational.Suppression != "" {
			return cachedFenceCycle{}, fmt.Errorf("automatic timer reached terminal lifecycle=%s reason=%s suppression=%s", operational.Lifecycle, operational.LifecycleReason, operational.Suppression)
		}
		if observation.captureErr == nil && captureOK && observation.disposition.Code == engine.DispositionTimerApplied &&
			observation.disposition.EngineSequence == timing.EngineSequence && publication.Watermark != nil && publication.Watermark.Equal(target) &&
			publication.LastDisposition == engine.DispositionTimerApplied && publication.LastEngineSequence == timing.EngineSequence &&
			publication.LastEngineSequence > priorTimingSequence && publication.PublicationID > priorPublicationID &&
			operational.PublicationID == publication.PublicationID && operational.LastEngineSequence == publication.LastEngineSequence {
			return cachedFenceCycle{timing: timing, total: timing.Stage + timing.Apply + timing.Publication, publicationID: publication.PublicationID, capture: captured}, nil
		}
	}
}

type cachedFenceServer struct {
	server     *httptest.Server
	start      chan struct{}
	pause      chan struct{}
	done       chan error
	frames     []string
	period     time.Duration
	catchUp    bool
	sent       atomic.Uint64
	framesRead func() uint64
	once       sync.Once
}

func newCachedFenceServer(t *testing.T, symbols []string, at time.Time, rate int) *cachedFenceServer {
	t.Helper()
	frames := make([]string, 64)
	for index := range frames {
		left := symbols[(index*2)%len(symbols)]
		right := symbols[(index*2+1)%len(symbols)]
		frames[index] = "[" + liveContentionAggregateJSON(left, at, index*2) + "," + liveContentionAggregateJSON(right, at, index*2+1) + "]"
	}
	period := cachedFenceObservedPeriod / time.Duration(rate)
	result := &cachedFenceServer{start: make(chan struct{}), pause: make(chan struct{}), done: make(chan error, 1), frames: frames, period: period}
	result.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { result.once.Do(func() { result.serve(w, r) }) }))
	return result
}

func (s *cachedFenceServer) serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connection, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.done <- err
		return
	}
	defer connection.CloseNow()
	write := func(raw string) error { return connection.Write(ctx, websocket.MessageText, []byte(raw)) }
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
			err = errors.New("aggregate subscription missing")
		}
		s.done <- err
		return
	}
	if err := write(`[{"ev":"status","status":"success"}]`); err != nil {
		s.done <- err
		return
	}
	select {
	case <-ctx.Done():
		s.done <- ctx.Err()
		return
	case <-s.start:
	}
	index := 0
	send := func() bool {
		for {
			sent, read := s.sent.Load(), uint64(0)
			if s.framesRead != nil {
				read = s.framesRead()
			}
			if sent <= read || sent-read < cachedFenceMaximumInFlight {
				break
			}
			timer := time.NewTimer(time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				s.done <- ctx.Err()
				return false
			case <-s.pause:
				timer.Stop()
				s.done <- nil
				<-ctx.Done()
				return false
			case <-timer.C:
			}
		}
		if err := write(s.frames[index%len(s.frames)]); err != nil {
			s.done <- err
			return false
		}
		s.sent.Add(1)
		index++
		return true
	}
	if s.catchUp {
		next := time.Now().Add(s.period)
		for {
			wait := time.Until(next)
			if wait <= 0 {
				select {
				case <-ctx.Done():
					s.done <- ctx.Err()
					return
				case <-s.pause:
					s.done <- nil
					<-ctx.Done()
					return
				default:
				}
			} else {
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					s.done <- ctx.Err()
					return
				case <-s.pause:
					timer.Stop()
					s.done <- nil
					<-ctx.Done()
					return
				case <-timer.C:
				}
			}
			if !send() {
				return
			}
			next = next.Add(s.period)
		}
	}
	ticker := time.NewTicker(s.period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.done <- ctx.Err()
			return
		case <-s.pause:
			s.done <- nil
			<-ctx.Done()
			return
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}

type cachedChunkState struct {
	rows    []engine.HydrationRow
	emitted int64
	chunks  int
}

func runCachedFence(t *testing.T, parent context.Context, manifest cachedFenceManifest, rate int) (result cachedFenceRunResult) {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	clock := &cachedFenceClock{now: manifest.end}
	selected := make([]string, 0, 64)
	for _, symbol := range manifest.binding.UniverseSymbols() {
		if _, ok := manifest.rowsBySymbol[symbol]; ok {
			selected = append(selected, symbol)
		}
		if len(selected) == 64 {
			break
		}
	}
	server := newCachedFenceServer(t, selected, manifest.end, rate)
	defer server.server.Close()
	adapter, err := massive.NewLiveAdapter(manifest.binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(server.server.URL, "http"), Credential: "cached-fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 512, MaxFrameBytes: 8 << 20, TotalFrameBytes: 64 << 20}, Clock: clock.read})
	if err != nil {
		t.Fatal(err)
	}
	run, err := New(ctx, manifest.binding, DefaultConfig(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	run.Engine().ArmEvaluationTimingForTest(time.Now)
	cleanup := &cachedFenceRuntimeCleanup{run: run, binding: manifest.binding}
	defer cleanup.shutdown(t)
	attempt, started, err := adapter.Start(ctx, massive.OpenAggregateEpoch{BindingIdentity: manifest.binding.Identity(), CommandToken: 100, Durations: capacityDurations()})
	if err != nil {
		t.Fatal(err)
	}
	cleanup.attempt = attempt
	if delivered, err := massive.DeliverToEngine(ctx, run.Engine(), started); err != nil || delivered.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("open delivery=%+v err=%v", delivered, err)
	}
	handshake, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		got, deliverErr := massive.DeliverToEngine(ctx, run.Engine(), delivery)
		if deliverErr != nil || (got.ControlDisposition.Code != engine.DispositionConnectionControlApplied && got.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("handshake delivery=%+v err=%v", got, deliverErr)
		}
	}
	run.setLiveSources(attempt, adapter)
	queueBase := attempt.QueueAccounting()
	server.framesRead = func() uint64 { return attempt.QueueAccounting().FramesRead - queueBase.FramesRead }
	planAdmission, planCompletion := run.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1,
		BindingIdentity: manifest.binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(),
		Budgets: engine.HydrationPlanBudgets{Workers: 2, RowsPerChunk: cachedFenceRowsPerChunk, MaximumResponseBytes: 2 << 30,
			MaximumNormalizedRecords: int64(len(manifest.rowsBySymbol)) * 57_600, MaximumResidentRecords: 2 * 57_600}})
	if planAdmission != engine.AdmissionAdmitted || planCompletion == nil {
		t.Fatal("production hydration plan not admitted")
	}
	planDisposition := <-planCompletion
	if planDisposition.Code != engine.DispositionHydrationPlanApplied || planDisposition.Plan.Start() != manifest.start || planDisposition.Plan.End() != manifest.end || len(planDisposition.Plan.Requests()) != len(manifest.rowsBySymbol) {
		t.Fatalf("hydration plan=%+v start=%s end=%s requests=%d", planDisposition, planDisposition.Plan.Start(), planDisposition.Plan.End(), len(planDisposition.Plan.Requests()))
	}
	if err := clock.advance(manifest.end.Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	close(server.start)

	deliveryCtx, cancelDelivery := context.WithCancel(ctx)
	deliveryDone := make(chan error, 1)
	cleanup.deliveryCancel, cleanup.deliveryDone = cancelDelivery, deliveryDone
	fenceDone := make(chan engine.HydrationDisposition, 1)
	go func() {
		for {
			started := time.Now()
			delivery, ok, deliveryErr := attempt.DeliverNextToEngine(deliveryCtx, run.Engine())
			if ok {
				run.observeDelivery(started, delivery)
				if delivery.HydrationDisposition.Code != "" {
					select {
					case fenceDone <- delivery.HydrationDisposition:
					default:
					}
				}
			}
			if deliveryErr != nil || !ok {
				deliveryDone <- deliveryErr
				return
			}
		}
	}()

	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	result.rate, result.heapBefore = rate, before.HeapAlloc
	requests := planDisposition.Plan.Requests()
	tokens := make(map[string]engine.HydrationRequestToken, len(requests))
	states := make(map[string]*cachedChunkState, len(requests))
	for _, token := range requests {
		tokens[token.Symbol()] = token
		states[token.Symbol()] = &cachedChunkState{rows: make([]engine.HydrationRow, 0, cachedFenceRowsPerChunk)}
	}
	historicalStarted := time.Now()
	counts := make(map[string]int64, len(manifest.rowsBySymbol))
	for symbol := range manifest.rowsBySymbol {
		counts[symbol] = 0
	}
	flush := func(symbol string) error {
		state, token := states[symbol], tokens[symbol]
		if len(state.rows) == 0 {
			return nil
		}
		total := manifest.rowsBySymbol[symbol]
		totalChunks := int((total + cachedFenceRowsPerChunk - 1) / cachedFenceRowsPerChunk)
		input, err := engine.NewHydrationChunkInput(token, token.ResultID(), state.chunks, totalChunks, state.emitted, total, state.rows)
		if err != nil {
			return err
		}
		admission, completion := run.Engine().AdmitHydrationChunk(ctx, input)
		if admission != engine.AdmissionAdmitted || completion == nil {
			return fmt.Errorf("hydration chunk symbol=%s ordinal=%d emitted=%d total=%d admission=%s context=%v queue=%+v operational=%+v",
				symbol, state.chunks, state.emitted, total, admission, ctx.Err(), attempt.QueueAccounting(), run.Engine().ObserveOperational())
		}
		var got engine.HydrationDisposition
		select {
		case <-ctx.Done():
			return fmt.Errorf("hydration chunk symbol=%s ordinal=%d emitted=%d total=%d completion context=%v queue=%+v operational=%+v",
				symbol, state.chunks, state.emitted, total, ctx.Err(), attempt.QueueAccounting(), run.Engine().ObserveOperational())
		case got = <-completion:
		}
		if got.Code != engine.DispositionHydrationChunkApplied {
			return fmt.Errorf("hydration chunk %s=%s/%s queue=%+v operational=%+v", symbol, got.Code, got.Reason, attempt.QueueAccounting(), run.Engine().ObserveOperational())
		}
		state.emitted += int64(len(state.rows))
		state.chunks++
		state.rows = state.rows[:0]
		return nil
	}
	progressRows := int64(0)
	rows, err := scanCachedFencePlayback(ctx, manifest.cursor, manifest.start, manifest.end, counts, func(value playback.RecordEvidence) error {
		v := value.Values()
		row, err := engine.NewHydrationRow(value.Symbol(), value.WindowStart(), value.WindowEnd(), engine.AggregateValues{Open: v.Open, High: v.High, Low: v.Low, Close: v.Close,
			Volume: v.Volume, VWAP: v.VWAP, AverageTradeSize: v.AverageTradeSize, ATSProvenance: engine.ATSProvenance(v.ATSProvenance)})
		if err != nil {
			return err
		}
		state := states[value.Symbol()]
		state.rows = append(state.rows, row)
		progressRows++
		if progressRows%500_000 == 0 {
			t.Logf("GATE_E_HYDRATION_PROGRESS rows=%d elapsed=%s queue=%+v", progressRows, time.Since(historicalStarted), attempt.QueueAccounting())
		}
		if len(state.rows) == cachedFenceRowsPerChunk {
			return flush(value.Symbol())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for symbol := range states {
		if err := flush(symbol); err != nil {
			t.Fatal(err)
		}
	}
	result.historical = time.Since(historicalStarted)
	t.Logf("GATE_E_HYDRATION rows=%d duration=%s", rows, result.historical)
	if rows != manifest.rows {
		t.Fatalf("timed rows=%d oracle=%d", rows, manifest.rows)
	}

	terminalStarted := time.Now()
	var fenceCommand engine.HydrationFenceCommand
	for _, token := range requests {
		state := states[token.Symbol()]
		terminalState := engine.HydrationCompletedValue
		if state.emitted == 0 {
			terminalState = engine.HydrationCompletedEmpty
		}
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), terminalState, engine.HydrationReasonNone, 1, 1, 1, state.emitted, int64(state.chunks), state.emitted)
		if err != nil {
			t.Fatal(err)
		}
		admission, completion := run.Engine().AdmitHydrationTerminal(ctx, terminal)
		if admission != engine.AdmissionAdmitted || completion == nil {
			t.Fatal("hydration terminal not admitted")
		}
		got := <-completion
		if got.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("terminal %s=%s/%s", token.Symbol(), got.Code, got.Reason)
		}
		if got.FenceCommand.CommandToken() != 0 {
			fenceCommand = got.FenceCommand
		}
	}
	result.terminals = time.Since(terminalStarted)
	if fenceCommand.CommandToken() == 0 {
		t.Fatal("engine did not issue the real ingress fence")
	}
	capture, err := massive.CaptureAggregateIngressFenceCommandFromEngine(fenceCommand)
	if err != nil {
		t.Fatal(err)
	}
	fenceStarted := time.Now()
	if err := attempt.CaptureAggregateIngressFence(ctx, run.Engine(), capture); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case disposition := <-fenceDone:
		if disposition.Code != engine.DispositionAggregateIngressFenceApplied {
			t.Fatalf("fence=%s/%s", disposition.Code, disposition.Reason)
		}
	}
	result.fence = time.Since(fenceStarted)
	readyStarted := time.Now()
	for {
		status := run.Status()
		if status.BackendReady {
			result.firstReady = time.Since(readyStarted)
			break
		}
		if time.Since(readyStarted) > 10*time.Second {
			t.Fatalf("first readiness timeout: %+v engine=%+v", status, run.Engine().ObserveOperational())
		}
		time.Sleep(time.Millisecond)
	}
	var atFence runtime.MemStats
	runtime.ReadMemStats(&atFence)
	result.heapAtFence = atFence.HeapAlloc
	time.Sleep(2 * time.Second)
	close(server.pause)
	select {
	case err := <-server.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("paced producer did not pause")
	}
	tailStarted := time.Now()
	for {
		queue := attempt.QueueAccounting()
		if queue.FramesQueued == 0 && queue.FramesClassifying == 0 && queue.FramesRead-queueBase.FramesRead == queue.FramesDispositioned-queueBase.FramesDispositioned {
			break
		}
		if time.Since(tailStarted) > 10*time.Second {
			t.Fatalf("live tail did not drain: %+v", queue)
		}
		time.Sleep(time.Millisecond)
	}
	result.tailDrain = time.Since(tailStarted)
	priorTimingSequence := run.Engine().ObserveEvaluationTiming().EngineSequence
	priorPublicationID := run.Engine().ObserveOperational().PublicationID
	automaticTimer := observeCachedFenceAutomaticTimers(run)
	defer run.setAutomaticTimerObserver(nil)
	for cycle := 0; cycle < 10; cycle++ {
		runtime.GC()
		var beforeCycle, afterCycle runtime.MemStats
		runtime.ReadMemStats(&beforeCycle)
		cycleCtx, stopCycle := context.WithTimeout(ctx, 5*time.Second)
		automatic, err := awaitCachedFenceAutomaticCycle(cycleCtx, run, clock, automaticTimer, manifest.end.Add(time.Duration(cycle+6)*time.Second), priorTimingSequence, priorPublicationID)
		stopCycle()
		if err != nil {
			t.Fatalf("Gate E cycle %d: %v", cycle+1, err)
		}
		timing := automatic.timing
		result.cycleLock[cycle] = automatic.total
		result.cycleStage[cycle], result.cycleApply[cycle], result.cyclePublication[cycle] = timing.Stage, timing.Apply, timing.Publication
		view := automatic.capture
		result.cyclePublicationID[cycle] = view.Engine.Publication.PublicationID
		if result.cyclePublicationID[cycle] != automatic.publicationID {
			t.Fatalf("Gate E cycle %d API publication=%d automatic timer publication=%d", cycle+1, result.cyclePublicationID[cycle], automatic.publicationID)
		}
		priorTimingSequence, priorPublicationID = timing.EngineSequence, automatic.publicationID
		runtime.ReadMemStats(&afterCycle)
		result.cycleAlloc[cycle] = afterCycle.TotalAlloc - beforeCycle.TotalAlloc
		t.Logf("GATE_E cycle=%d stage=%s apply=%s publication=%s total_lock=%s allocated_bytes=%d publication_id=%d", cycle+1, timing.Stage, timing.Apply, timing.Publication, result.cycleLock[cycle], result.cycleAlloc[cycle], result.cyclePublicationID[cycle])
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	result.heapAfter = after.HeapAlloc
	queue := attempt.QueueAccounting()
	metrics := run.Metrics()
	view := run.Engine().ObserveOperational()
	replay := run.Engine().ObserveReplay()
	status := run.Status()
	result.framesSent = server.sent.Load()
	result.framesRead = queue.FramesRead - queueBase.FramesRead
	result.framesAdmitted = queue.FramesAdmitted - queueBase.FramesAdmitted
	result.framesDispositioned = queue.FramesDispositioned - queueBase.FramesDispositioned
	result.framesRejected = liveContentionFrameRejections(queue, queueBase)
	result.maximumQueued = metrics.QueueHighFrames
	result.rows = view.Hydration.Rows.Consumed
	result.rowsObserved = int(view.Hydration.Rows.Consumed)
	result.values = view.Hydration.Accounting.CompletedValue
	result.empties = view.Hydration.Accounting.CompletedEmpty
	result.rankingRows = replay.Rows
	result.lifecycle, result.rankingMode = view.Lifecycle, view.RankingMode
	result.fenceReconciled, result.ready, result.accounting = view.Hydration.FenceReconciled, status.BackendReady && status.RankingCurrent, metrics.AccountingValid && operationalAccountingValid(view) && queue.Reconciles() && metrics.Adapter.Reconciles()
	result.tq = run.Engine().ObserveTQ()

	return result
}

func assertCachedFence(t *testing.T, manifest cachedFenceManifest, result cachedFenceRunResult) {
	t.Helper()
	problems := make([]string, 0)
	if result.rows != uint64(manifest.rows) || result.values != uint64(manifest.values) || result.empties != uint64(manifest.empties) {
		problems = append(problems, fmt.Sprintf("hydration rows/value/empty=%d/%d/%d want=%d/%d/%d", result.rows, result.values, result.empties, manifest.rows, manifest.values, manifest.empties))
	}
	if result.framesSent == 0 || result.framesRead != result.framesSent || result.framesAdmitted != result.framesSent || result.framesDispositioned != result.framesSent || result.framesRejected != 0 {
		problems = append(problems, fmt.Sprintf("frames sent/read/admitted/dispositioned/rejected=%d/%d/%d/%d/%d", result.framesSent, result.framesRead, result.framesAdmitted, result.framesDispositioned, result.framesRejected))
	}
	if !result.fenceReconciled || !result.ready || result.lifecycle != "live" || result.rankingMode != "qualified_current" || result.rankingRows < 0 || result.rankingRows > 20 {
		problems = append(problems, fmt.Sprintf("lifecycle=%s ranking=%s rows=%d fence=%t ready=%t", result.lifecycle, result.rankingMode, result.rankingRows, result.fenceReconciled, result.ready))
	}
	if !result.accounting || result.maximumQueued >= 512 {
		problems = append(problems, fmt.Sprintf("accounting=%t queue_high=%d", result.accounting, result.maximumQueued))
	}
	for cycle := range result.cycleLock {
		if result.cycleLock[cycle] >= 2*time.Second || result.cyclePublicationID[cycle] == 0 || cycle > 0 && result.cyclePublicationID[cycle] <= result.cyclePublicationID[cycle-1] {
			problems = append(problems, fmt.Sprintf("cycle %d lock=%s publication=%d", cycle+1, result.cycleLock[cycle], result.cyclePublicationID[cycle]))
		}
	}
	if len(problems) != 0 {
		t.Fatalf("cached hydration fence acceptance failed: %s", strings.Join(problems, "; "))
	}
}
