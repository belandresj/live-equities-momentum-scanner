package replaymode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

// TestReplayWindowRuntime is P-C12-WINDOW and P-C12-RUNTIME. It proves that
// S through O0 are unpaced (including O0's timer), the first observation is
// published only after O0, later groups use cumulative one-second deadlines,
// and successful requested-end state remains immutable until shutdown.
func TestReplayWindowRuntime(t *testing.T) {
	fixture := newReplayFixture(t, 6*time.Second)
	runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
	defer runtime.Close()
	timeline := newTestTimeline(time.Unix(1_800_000_000, 0).UTC(), time.Second, 0)
	runtime.schedule = timeline.schedule()
	var phases []operations.ReplayPhase
	var snapshots []snapshotapi.Snapshot
	var records []StatusRecord
	runtime.afterPublish = func(phase operations.ReplayPhase) {
		capture, err := runtime.CaptureSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := snapshotapi.Map(capture)
		if err != nil {
			t.Fatal(err)
		}
		phases = append(phases, phase)
		snapshots = append(snapshots, snapshot)
	}

	result, err := runtime.RunReporting(context.Background(), func(record StatusRecord) error {
		records = append(records, record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != replay.OutcomeComplete || result.Completion != replay.CompletionArtifactEnd {
		t.Fatalf("result = %+v", result)
	}
	if got, want := phases, []operations.ReplayPhase{operations.ReplayWarming, operations.ReplayWarming, operations.ReplayWarming, operations.ReplayWarming, operations.ReplayWarming,
		operations.ReplayObserving, operations.ReplayObserving, operations.ReplayFinalizing, operations.ReplayRetainedSuccess}; !reflect.DeepEqual(got, want) {
		t.Fatalf("phases = %v want %v", got, want)
	}
	if got, want := statusPhases(records), []operations.ReplayPhase{operations.ReplayWarming, operations.ReplayObserving, operations.ReplayFinalizing, operations.ReplayRetainedSuccess}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status transitions = %v want %v", got, want)
	}
	encoded, marshalErr := json.Marshal(records)
	if marshalErr != nil || strings.Contains(string(encoded), fixture.artifact) || strings.Contains(string(encoded), finalArtifactID(snapshots)) {
		t.Fatalf("status records exposed private artifact data: body=%s err=%v", encoded, marshalErr)
	}
	if got := timeline.deadlineSnapshot(); len(got) != 2 || got[1].Sub(got[0]) != time.Second {
		t.Fatalf("cumulative deadlines = %v", got)
	}
	for index, snapshot := range snapshots[:5] {
		if snapshot.Replay.Phase != "warming" || len(snapshot.Rows) != 0 || snapshot.Ranking.Mode != "unavailable" || snapshot.Ranking.Reason != "replay_warming" ||
			snapshot.Replay.ScheduleLagMS != nil || snapshot.Replay.Window.ObservationBoundariesPublished != "0" {
			t.Fatalf("warming[%d] = %+v", index, snapshot)
		}
	}
	first := snapshots[5]
	if first.Replay.Phase != "observing" || first.Replay.LogicalTime != timestamp(fixture.start.Add(4*time.Second)) ||
		first.Publication.CommittedT == nil || *first.Publication.CommittedT != timestamp(fixture.start) || first.Replay.ScheduleLagMS == nil || *first.Replay.ScheduleLagMS != 0 ||
		first.Replay.Window.WarmupGroupsCompleted != "5" || first.Replay.Window.ObservationSecondsCompleted != "0" || first.Replay.Window.ObservationBoundariesPublished != "1" {
		t.Fatalf("first observation = %+v", first)
	}
	final := snapshots[len(snapshots)-1]
	assertFinalReplaySnapshot(t, final, replay.CompletionArtifactEnd)
	if final.Publication.ID != fmt.Sprint(result.Status.Publication.PublicationID) || final.Publication.ID == snapshots[len(snapshots)-2].Publication.ID {
		t.Fatalf("terminal publication identity: final=%s prior=%s result=%d", final.Publication.ID, snapshots[len(snapshots)-2].Publication.ID, result.Status.Publication.PublicationID)
	}
	if final.Replay.Source.ArtifactRecords != "2" || final.Replay.Source.CompletedRecordDispositions != "2" || final.Replay.Source.UnreadRecords != "0" ||
		final.Replay.Source.CompletedGroups != "7" || final.Replay.Source.RemainingGroups != "0" || final.Replay.Window.ObservationSecondsCompleted != "2" ||
		final.Replay.Window.ObservationBoundariesPublished != "3" {
		t.Fatalf("final accounting = %+v", final.Replay)
	}
	terminalContext := operations.ReplayCaptureContext{Phase: operations.ReplayRetainedSuccess, ArtifactID: runtime.metadata.ArtifactID, ArtifactEnd: runtime.metadata.ReplayEnd,
		ObservationStart: runtime.observationStart, ObservationEnd: runtime.observationEnd, LogicalTime: runtime.lastLogical, Completion: result.Completion,
		ScheduleLag: runtime.lastLag, Source: result.Accounting, Window: runtime.window}
	if fallbackRetained, err := runtime.operations.PublishReplayTerminal(terminalContext, result); err != nil || !fallbackRetained {
		t.Fatalf("terminal fallback retained=%t err=%v", fallbackRetained, err)
	}
	retained, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	retainedView, ok := operations.InspectSnapshotCapture(retained)
	if !ok || !sameTerminalPublication(retainedView.Engine.Publication, result.Status.Publication) {
		t.Fatalf("terminal fallback publication=%+v result=%+v", retainedView.Engine.Publication, result.Status.Publication)
	}
	retainedSnapshot, err := snapshotapi.Map(retained)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(final.Publication, retainedSnapshot.Publication) || !reflect.DeepEqual(final.Ranking, retainedSnapshot.Ranking) ||
		!reflect.DeepEqual(final.Rows, retainedSnapshot.Rows) || !reflect.DeepEqual(final.Replay, retainedSnapshot.Replay) {
		t.Fatalf("retained state changed: final=%+v retained=%+v", final, retainedSnapshot)
	}
	handler, err := snapshotapi.NewHandler(runtime, snapshotapi.HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("snapshot status=%d body=%s", response.Code, response.Body.String())
	}
	var served snapshotapi.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&served); err != nil {
		t.Fatal(err)
	}
	assertFinalReplaySnapshot(t, served, replay.CompletionArtifactEnd)
	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), `"reason":"not_live_mode"`) {
		t.Fatalf("ready status=%d body=%s", ready.Code, ready.Body.String())
	}
	live := httptest.NewRecorder()
	handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if live.Code != http.StatusOK {
		t.Fatalf("live status=%d body=%s", live.Code, live.Body.String())
	}
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if records[len(records)-1].Phase != operations.ReplayShuttingDown {
		t.Fatalf("final status transition = %v", statusPhases(records))
	}
}

// O0=S and O1<R are the two window edges most likely to acquire an accidental
// warm-up tick or to mistake a validated suffix for applied observation data.
func TestReplayWindowEdges(t *testing.T) {
	fixture := newReplayFixture(t, 8*time.Second)
	runtime := fixture.prepare(t, 0, 6*time.Second)
	defer runtime.Close()
	timeline := newTestTimeline(time.Unix(1_805_000_000, 0).UTC(), time.Second, 0)
	runtime.schedule = timeline.schedule()
	var snapshots []snapshotapi.Snapshot
	runtime.afterPublish = func(operations.ReplayPhase) {
		capture, err := runtime.CaptureSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		mapped, err := snapshotapi.Map(capture)
		if err != nil {
			t.Fatal(err)
		}
		snapshots = append(snapshots, mapped)
	}
	result, err := runtime.Run(context.Background())
	if err != nil || result.Outcome != replay.OutcomeComplete || result.Completion != replay.CompletionRequestedEnd {
		t.Fatalf("requested-end result=%+v err=%v", result, err)
	}
	if len(snapshots) != 9 || snapshots[0].Replay.Phase != "warming" || snapshots[0].Replay.Window.WarmupGroupsCompleted != "0" || snapshots[1].Replay.Phase != "observing" ||
		snapshots[1].Replay.LogicalTime != timestamp(fixture.start) || snapshots[1].Publication.CommittedT == nil || *snapshots[1].Publication.CommittedT != timestamp(fixture.start) ||
		snapshots[1].Replay.Window.WarmupGroupsPlanned != "1" || snapshots[1].Replay.Window.WarmupGroupsCompleted != "1" {
		t.Fatalf("O0=S captures: warming=%+v observing=%+v", snapshots[0], snapshots[1])
	}
	final := snapshots[len(snapshots)-1]
	assertFinalReplaySnapshot(t, final, replay.CompletionRequestedEnd)
	if final.Replay.Source.ArtifactRecords != "2" || final.Replay.Source.CompletedRecordDispositions != "1" ||
		final.Replay.Source.IntentionallyUnappliedSuffixRecords != "1" || final.Replay.Source.UnreadRecords != "0" ||
		final.Replay.Source.PlannedGroups != "7" || final.Replay.Source.CompletedGroups != "7" {
		t.Fatalf("requested-end suffix accounting = %+v", final.Replay.Source)
	}
}

// TestReplayDeterminism is P-C12-DETERMINISM. The same artifact and logical
// window must yield the same market publication, ranking, rows, and accounting
// under on-time, accelerated, and deliberately late wall schedules.
func TestReplayDeterminism(t *testing.T) {
	fixture := newReplayFixture(t, 6*time.Second)
	type runEvidence struct {
		final     snapshotapi.Snapshot
		deadlines []time.Time
		lags      []uint64
	}
	run := func(step, late time.Duration) runEvidence {
		runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
		defer runtime.Close()
		timeline := newTestTimeline(time.Unix(1_810_000_000, 0).UTC(), step, late)
		runtime.schedule = timeline.schedule()
		var final snapshotapi.Snapshot
		var lags []uint64
		runtime.afterPublish = func(phase operations.ReplayPhase) {
			capture, err := runtime.CaptureSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			mapped, err := snapshotapi.Map(capture)
			if err != nil {
				t.Fatal(err)
			}
			if mapped.Replay.ScheduleLagMS != nil {
				lags = append(lags, *mapped.Replay.ScheduleLagMS)
			}
			final = mapped
		}
		result, err := runtime.Run(context.Background())
		if err != nil || result.Outcome != replay.OutcomeComplete {
			t.Fatalf("run: result=%+v err=%v", result, err)
		}
		return runEvidence{final: final, deadlines: timeline.deadlineSnapshot(), lags: lags}
	}
	onTime := run(time.Second, 0)
	accelerated := run(10*time.Millisecond, 0)
	late := run(time.Second, 250*time.Millisecond)
	for name, other := range map[string]runEvidence{"accelerated": accelerated, "late": late} {
		if !reflect.DeepEqual(onTime.final.Publication, other.final.Publication) || !reflect.DeepEqual(onTime.final.Ranking, other.final.Ranking) ||
			!reflect.DeepEqual(onTime.final.Rows, other.final.Rows) || !reflect.DeepEqual(onTime.final.Accounting, other.final.Accounting) ||
			!reflect.DeepEqual(onTime.final.Replay.Source, other.final.Replay.Source) || !reflect.DeepEqual(onTime.final.Replay.Window, other.final.Replay.Window) ||
			onTime.final.Replay.LogicalTime != other.final.Replay.LogicalTime || onTime.final.Replay.Completion != other.final.Replay.Completion {
			t.Fatalf("%s logical result diverged: on-time=%+v other=%+v", name, onTime.final, other.final)
		}
	}
	if len(accelerated.deadlines) != 2 || accelerated.deadlines[1].Sub(accelerated.deadlines[0]) != 10*time.Millisecond {
		t.Fatalf("accelerated deadlines = %v", accelerated.deadlines)
	}
	if len(late.lags) < 3 || late.lags[len(late.lags)-1] != 250 {
		t.Fatalf("late schedule lags = %v", late.lags)
	}
}

// TestReplayContainment is P-C12-CONTAINMENT. Cancellation before a paced
// boundary and validation failure after startup both use the C4 source's sole
// terminal path, clear active accounting, and publish no current rows.
func TestReplayContainment(t *testing.T) {
	fixture := newReplayFixture(t, 6*time.Second)
	t.Run("canceled paced boundary", func(t *testing.T) {
		runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
		defer runtime.Close()
		ctx, cancel := context.WithCancel(context.Background())
		timeline := newTestTimeline(time.Unix(1_820_000_000, 0).UTC(), time.Second, 0)
		timeline.cancel = cancel
		runtime.schedule = timeline.schedule()
		result, err := runtime.Run(ctx)
		if !errors.Is(err, context.Canceled) || result.Outcome != replay.OutcomeCanceled || result.Accounting.CanceledRuns != 1 || result.Accounting.ActiveGroup != 0 {
			t.Fatalf("canceled result=%+v err=%v", result, err)
		}
		capture, captureErr := runtime.CaptureSnapshot()
		if captureErr != nil {
			t.Fatal(captureErr)
		}
		snapshot, mapErr := snapshotapi.Map(capture)
		if mapErr != nil {
			t.Fatal(mapErr)
		}
		if snapshot.Replay.Phase != "canceling" || len(snapshot.Rows) != 0 || snapshot.Status.RankingCurrent || snapshot.Replay.Source.CanceledRuns != "1" {
			t.Fatalf("canceled snapshot = %+v", snapshot)
		}
	})

	t.Run("canceled finalization", func(t *testing.T) {
		runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
		defer runtime.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		runtime.schedule = newTestTimeline(time.Unix(1_825_000_000, 0).UTC(), time.Second, 0).schedule()
		runtime.afterPublish = func(phase operations.ReplayPhase) {
			if phase == operations.ReplayFinalizing {
				cancel()
			}
		}
		result, err := runtime.Run(ctx)
		if !errors.Is(err, context.Canceled) || result.Outcome != replay.OutcomeCanceled || result.Accounting.CanceledRuns != 1 || result.Accounting.ActiveGroup != 0 {
			t.Fatalf("finalization result=%+v err=%v", result, err)
		}
		capture, captureErr := runtime.CaptureSnapshot()
		if captureErr != nil {
			t.Fatal(captureErr)
		}
		snapshot, mapErr := snapshotapi.Map(capture)
		if mapErr != nil {
			t.Fatal(mapErr)
		}
		if snapshot.Replay.Phase != "canceling" || snapshot.Publication.Lifecycle != "ended" || snapshot.Publication.LifecycleReason != "controlled_stop" ||
			snapshot.Replay.Source.CanceledRuns != "1" || snapshot.Replay.Window.ObservationSecondsCompleted != "2" {
			t.Fatalf("finalization cancellation = %+v", snapshot)
		}
	})

	t.Run("artifact changed after validation", func(t *testing.T) {
		runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
		defer runtime.Close()
		runtime.afterStart = func() {
			file, err := os.OpenFile(fixture.artifact, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			_, writeErr := file.WriteString(" ")
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				t.Fatalf("mutate artifact: write=%v close=%v", writeErr, closeErr)
			}
		}
		var records []StatusRecord
		result, runErr := runtime.RunReporting(context.Background(), func(record StatusRecord) error {
			records = append(records, record)
			return nil
		})
		if runErr == nil || result.Outcome != replay.OutcomeFailed || result.Accounting.FailedRuns != 1 || result.Accounting.ActiveGroup != 0 {
			t.Fatalf("failed result=%+v err=%v", result, runErr)
		}
		if len(records) == 0 || records[len(records)-1].Phase != operations.ReplaySuppressed || records[len(records)-1].FailedRuns != "1" {
			t.Fatalf("failed status records = %+v", records)
		}
		if capture, captureErr := runtime.CaptureSnapshot(); captureErr == nil {
			t.Fatalf("terminal failure left prior capture reachable: %+v", capture)
		}
		handler, handlerErr := snapshotapi.NewHandler(runtime, snapshotapi.HandlerConfig{})
		if handlerErr != nil {
			t.Fatal(handlerErr)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("failed snapshot status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestReplayContainmentDuringActiveStep(t *testing.T) {
	const symbols = 512
	fixture := newReplayFixtureSymbols(t, 6*time.Second, symbols)
	runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
	defer runtime.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	runtime.beforeStep = func(group time.Time) {
		if !group.Equal(fixture.start) {
			return
		}
		go func() {
			defer close(started)
			deadline := time.Now().Add(time.Second)
			for time.Now().Before(deadline) {
				if runtime.operations.Engine().ObserveReplay().AggregateInserted > 0 {
					cancel()
					return
				}
				time.Sleep(time.Microsecond)
			}
		}()
	}
	result, err := runtime.Run(ctx)
	<-started
	if !errors.Is(err, context.Canceled) || result.Outcome != replay.OutcomeCanceled || result.Accounting.CanceledRuns != 1 ||
		result.Accounting.CompletedRecordDispositions == 0 || result.Accounting.CompletedRecordDispositions >= symbols || result.Accounting.ActiveGroup != 0 {
		t.Fatalf("active-step result=%+v err=%v", result, err)
	}
	capture, captureErr := runtime.CaptureSnapshot()
	if captureErr != nil {
		t.Fatal(captureErr)
	}
	snapshot, mapErr := snapshotapi.Map(capture)
	if mapErr != nil || snapshot.Replay.Phase != "canceling" || snapshot.Replay.Source.CanceledRuns != "1" || len(snapshot.Rows) != 0 {
		t.Fatalf("active-step snapshot=%+v mapErr=%v", snapshot, mapErr)
	}
}

// TestPrepareUsesOnlyValidatedArtifactAndCaches proves startup validation is
// complete before an engine/API owner is returned and performs no acquisition.
func TestPrepareUsesOnlyValidatedArtifactAndCaches(t *testing.T) {
	fixture := newReplayFixture(t, 6*time.Second)
	requests := fixture.requests.Load()
	runtime := fixture.prepare(t, 4*time.Second, 6*time.Second)
	if fixture.requests.Load() != requests {
		t.Fatalf("prepare made provider requests: before=%d after=%d", requests, fixture.requests.Load())
	}
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []StartupConfig{
		{ArtifactPath: fixture.artifact, ReferenceDirectory: filepath.Join(t.TempDir(), "missing"), ObservationStart: clockText(fixture.start.Add(4 * time.Second)), ObservationEnd: clockText(fixture.start.Add(6 * time.Second))},
		{ArtifactPath: fixture.artifact, ReferenceDirectory: fixture.referenceDirectory, ObservationStart: "04:00:06", ObservationEnd: "04:00:04"},
	} {
		if got, err := Prepare(context.Background(), mutation); err == nil || got != nil {
			t.Fatalf("invalid startup accepted: runtime=%v err=%v", got, err)
		}
	}
	canceled, cancelStartup := context.WithCancel(context.Background())
	cancelStartup()
	if got, err := Prepare(canceled, fixture.config(4*time.Second, 6*time.Second)); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("canceled startup: runtime=%v err=%v", got, err)
	}
}

type replayFixture struct {
	artifact, referenceDirectory string
	start                        time.Time
	requests                     *atomic.Uint64
	symbols                      int
}

func newReplayFixture(t *testing.T, artifactDuration time.Duration) replayFixture {
	return newReplayFixtureSymbols(t, artifactDuration, 1)
}

func newReplayFixtureSymbols(t *testing.T, artifactDuration time.Duration, symbolCount int) replayFixture {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-06")
	if err != nil {
		t.Fatal(err)
	}
	symbols := make([]string, symbolCount)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%04d", index)
	}
	requests := new(atomic.Uint64)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			results := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				results[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "count": len(results), "results": results})
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			results := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				results[index] = map[string]any{"T": symbol, "c": 10 + float64(index)/100, "t": facts.PriorRegularClose.UnixMilli()}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(results), "results": results})
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/ticker/"):
			parts := strings.Split(request.URL.Path, "/")
			if len(parts) < 5 {
				http.NotFound(writer, request)
				return
			}
			symbol := parts[4]
			results := []map[string]any{{"t": facts.SessionStart.UnixMilli(), "o": 10, "h": 11, "l": 9, "c": 10.5, "v": 100, "vw": 10.25, "n": 10}}
			if artifactDuration > time.Second {
				results = append(results, map[string]any{"t": facts.SessionStart.Add(artifactDuration - time.Second).UnixMilli(), "o": 11, "h": 12, "l": 10, "c": 11.5, "v": 110, "vw": 11.25, "n": 11})
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "ticker": symbol, "adjusted": false,
				"results": results})
		default:
			http.NotFound(writer, request)
		}
	}))
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	referenceDirectory := filepath.Join(root, "reference")
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: referenceDirectory, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }}).Resolve(context.Background(), facts)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: referenceDirectory, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	maximumRecords := int64(artifactDuration/time.Second) * int64(symbolCount)
	workers := min(symbolCount, 8)
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: facts.SessionStart, End: facts.SessionStart.Add(artifactDuration), Workers: workers,
		DestinationDirectory: root, Limits: replayartifact.Limits{MaximumNormalizedRecords: maximumRecords, MaximumResponseBytes: 1 << 20, MaximumArtifactBytes: 1 << 20,
			MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: maximumRecords}}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("compile = %+v", compiled)
	}
	return replayFixture{artifact: compiled.Path, referenceDirectory: referenceDirectory, start: facts.SessionStart, requests: requests, symbols: symbolCount}
}

func (f replayFixture) config(startAfter, endAfter time.Duration) StartupConfig {
	return StartupConfig{ArtifactPath: f.artifact, ReferenceDirectory: f.referenceDirectory, ObservationStart: clockText(f.start.Add(startAfter)), ObservationEnd: clockText(f.start.Add(endAfter))}
}

func (f replayFixture) prepare(t *testing.T, startAfter, endAfter time.Duration) *Runtime {
	t.Helper()
	runtime, err := Prepare(context.Background(), f.config(startAfter, endAfter))
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

type testTimeline struct {
	mu        sync.Mutex
	now       time.Time
	wallStep  time.Duration
	late      time.Duration
	deadlines []time.Time
	cancel    context.CancelFunc
}

func newTestTimeline(now time.Time, wallStep, late time.Duration) *testTimeline {
	return &testTimeline{now: now, wallStep: wallStep, late: late}
}

func (t *testTimeline) schedule() schedule {
	return schedule{wallStep: t.wallStep, now: func() time.Time {
		t.mu.Lock()
		defer t.mu.Unlock()
		return t.now
	}, wait: func(ctx context.Context, deadline time.Time) error {
		t.mu.Lock()
		t.deadlines = append(t.deadlines, deadline)
		t.now = deadline.Add(t.late)
		cancel := t.cancel
		t.cancel = nil
		t.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return ctx.Err()
	}}
}

func (t *testTimeline) deadlineSnapshot() []time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]time.Time(nil), t.deadlines...)
}

func assertFinalReplaySnapshot(t *testing.T, snapshot snapshotapi.Snapshot, completion replay.CompletionDisposition) {
	t.Helper()
	if snapshot.Replay == nil || snapshot.Replay.Phase != "retained_success" || snapshot.Replay.Completion != string(completion) ||
		snapshot.Publication.Lifecycle != "ended" || snapshot.Publication.RunMode != "replay" || snapshot.Status.BackendReady || snapshot.Status.ReadinessReason != "not_live_mode" ||
		snapshot.Status.RankingCurrent || snapshot.Replay.Source.CompletedRuns != "1" || snapshot.Replay.Source.FailedRuns != "0" || snapshot.Replay.Source.CanceledRuns != "0" {
		t.Fatalf("final replay snapshot = %+v", snapshot)
	}
	for _, row := range snapshot.Rows {
		if row.Tape5s.Status != "unavailable" || row.Tape5s.Reason != "replay_unavailable" || row.Spread.Status != "unavailable" || row.Spread.Reason != "replay_unavailable" {
			t.Fatalf("replay row exposes live T/Q = %+v", row)
		}
	}
}

func clockText(value time.Time) string { return value.In(mustNewYork()).Format("15:04:05") }

func mustNewYork() *time.Location {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return location
}

func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func statusPhases(records []StatusRecord) []operations.ReplayPhase {
	result := make([]operations.ReplayPhase, len(records))
	for index := range records {
		result[index] = records[index].Phase
	}
	return result
}

func finalArtifactID(snapshots []snapshotapi.Snapshot) string {
	if len(snapshots) == 0 || snapshots[len(snapshots)-1].Replay == nil {
		return "unreachable-artifact-id"
	}
	return snapshots[len(snapshots)-1].Replay.ArtifactID
}

func sameTerminalPublication(left, right engine.ReplayPublicationView) bool {
	leftRows, rightRows := left.AggregateEvaluation.Rows, right.AggregateEvaluation.Rows
	left.AggregateEvaluation.Rows, right.AggregateEvaluation.Rows = nil, nil
	if !reflect.DeepEqual(left, right) || len(leftRows) != len(rightRows) {
		return false
	}
	for index := range leftRows {
		if !reflect.DeepEqual(leftRows[index], rightRows[index]) {
			return false
		}
	}
	return true
}

func TestObservationTimeRejectsNonCanonicalValues(t *testing.T) {
	for _, value := range []string{"4:00:00", "04:00", " 04:00:00", "24:00:00"} {
		if parsed, err := observationTime("2026-08-06", value); err == nil || !parsed.IsZero() {
			t.Errorf("%q accepted as %s", value, parsed)
		}
	}
	if got, err := observationTime("2026-08-06", "04:00:00"); err != nil || got.Format(time.RFC3339) != "2026-08-06T08:00:00Z" {
		t.Fatalf("canonical time = %s err=%v", got, err)
	}
}

func TestFixturePathsRemainPrivateAndTemporary(t *testing.T) {
	fixture := newReplayFixture(t, time.Second)
	if filepath.Ext(fixture.artifact) != ".jsonl" {
		t.Fatal(fmt.Errorf("unexpected artifact path %q", fixture.artifact))
	}
}
