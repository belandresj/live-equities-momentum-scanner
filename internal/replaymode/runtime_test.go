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
	fixture := newReplayFixture(t, 7*time.Second)
	runtime := fixture.prepare(t, 4*time.Second, 7*time.Second)
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
	if got, want := phases, []operations.ReplayPhase{operations.ReplayWarming, operations.ReplayObserving, operations.ReplayObserving, operations.ReplayObserving, operations.ReplayFinalizing, operations.ReplayRetainedSuccess}; !reflect.DeepEqual(got, want) {
		t.Fatalf("phases = %v want %v", got, want)
	}
	if got, want := statusPhases(records), []operations.ReplayPhase{operations.ReplayWarming, operations.ReplayObserving, operations.ReplayFinalizing, operations.ReplayRetainedSuccess}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status transitions = %v want %v", got, want)
	}
	encoded, marshalErr := json.Marshal(records)
	if marshalErr != nil || strings.Contains(string(encoded), fixture.artifact) || strings.Contains(string(encoded), finalArtifactID(snapshots)) {
		t.Fatalf("status records exposed private artifact data: body=%s err=%v", encoded, marshalErr)
	}
	if got := timeline.deadlineSnapshot(); len(got) != 3 || got[1].Sub(got[0]) != time.Second || got[2].Sub(got[1]) != time.Second {
		t.Fatalf("cumulative deadlines = %v", got)
	}
	for index, snapshot := range snapshots[:1] {
		if snapshot.Replay.Phase != "warming" || len(snapshot.Rows) != 0 || snapshot.Ranking.Mode != "unavailable" || snapshot.Ranking.Reason != "replay_warming" ||
			snapshot.Replay.ScheduleLagMS != nil || snapshot.Replay.Window.ObservationBoundariesPublished != "0" {
			t.Fatalf("warming[%d] = %+v", index, snapshot)
		}
	}
	first := snapshots[1]
	if first.Replay.Phase != "observing" || first.Replay.LogicalTime != timestamp(fixture.start.Add(4*time.Second)) ||
		first.Publication.CommittedT == nil || *first.Publication.CommittedT != timestamp(fixture.start) || first.Replay.ScheduleLagMS == nil || *first.Replay.ScheduleLagMS != 0 ||
		first.Replay.Window.WarmupGroupsCompleted != "5" || first.Replay.Window.ObservationSecondsCompleted != "0" || first.Replay.Window.ObservationBoundariesPublished != "1" {
		t.Fatalf("first observation = %+v", first)
	}
	if got := []string{snapshots[1].Publication.ID, snapshots[2].Publication.ID, snapshots[3].Publication.ID}; got[0] == got[1] || got[1] == got[2] || got[0] == got[2] {
		t.Fatalf("three consecutive API publications did not change: %v", got)
	}
	final := snapshots[len(snapshots)-1]
	assertFinalReplaySnapshot(t, final, replay.CompletionArtifactEnd)
	if final.Publication.ID != fmt.Sprint(result.Status.Publication.PublicationID) || final.Publication.ID == snapshots[len(snapshots)-2].Publication.ID {
		t.Fatalf("terminal publication identity: final=%s prior=%s result=%d", final.Publication.ID, snapshots[len(snapshots)-2].Publication.ID, result.Status.Publication.PublicationID)
	}
	if final.Replay.Source.ArtifactRecords != "2" || final.Replay.Source.CompletedRecordDispositions != "2" || final.Replay.Source.UnreadRecords != "0" ||
		final.Replay.Source.CompletedGroups != "8" || final.Replay.Source.RemainingGroups != "0" || final.Replay.Window.ObservationSecondsCompleted != "3" ||
		final.Replay.Window.ObservationBoundariesPublished != "4" {
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
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil))
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

func TestReplayWarmingPublicationRateLimit(t *testing.T) {
	fixture := newReplayFixture(t, 10*time.Second)
	runtime := fixture.prepare(t, 8*time.Second, 10*time.Second)
	defer runtime.Close()
	timeline := newTestTimeline(time.Unix(1_807_000_000, 0).UTC(), time.Second, 0)
	runtime.schedule = timeline.schedule()
	runtime.beforeStep = func(time.Time) { timeline.advance(250 * time.Millisecond) }
	type capture struct {
		phase operations.ReplayPhase
		at    time.Time
	}
	var captures []capture
	runtime.afterPublish = func(phase operations.ReplayPhase) { captures = append(captures, capture{phase, timeline.current()}) }
	result, err := runtime.Run(context.Background())
	if err != nil || result.Outcome != replay.OutcomeComplete {
		t.Fatalf("rate-limited run: %+v err=%v", result, err)
	}
	var warming []time.Time
	for _, value := range captures {
		if value.phase == operations.ReplayWarming {
			warming = append(warming, value.at)
		}
	}
	if len(warming) < 2 || captures[0].phase != operations.ReplayWarming {
		t.Fatalf("warming captures = %+v", captures)
	}
	for index := 1; index < len(warming); index++ {
		if warming[index].Sub(warming[index-1]) < time.Second {
			t.Fatalf("warming publication exceeded 1 Hz: %v", warming)
		}
	}
	observed := false
	for _, value := range captures {
		if value.phase == operations.ReplayObserving {
			observed = true
			break
		}
	}
	if !observed {
		t.Fatalf("mandatory observation-start capture missing: %+v", captures)
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

// The fast-forward path must be a projection optimization only. This fixture
// contains a dense symbol that finalizes qualification, a quiet symbol, and a
// complete no-print symbol across the production four-second delay.
func TestReplayFastForwardFirstObservationEquivalence(t *testing.T) {
	fixture, observationStart, observationEnd := newSemanticReplayFixture(t)
	type observation struct {
		view               engine.ReplayDeterministicView
		status             engine.ReplayStatus
		result             replay.Result
		qualificationTrace [][]engine.ReplayQualificationView
	}
	run := func(fastForward bool) observation {
		binding, err := cachedBinding(context.Background(), "2026-08-06", fixture.referenceDirectory)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := replayartifact.ProbeCandidateHeader(context.Background(), fixture.artifact, MaximumArtifactBytes)
		if err != nil {
			t.Fatal(err)
		}
		handle, err := replayartifact.OpenValidated(fixture.artifact, replayartifact.ValidationPlan{Binding: binding, Start: candidate.ReplayStart, End: candidate.ReplayEnd,
			ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: MaximumArtifactBytes, MaximumRecords: MaximumRecords})
		if err != nil {
			t.Fatal(err)
		}
		defer handle.Close()
		clock, err := replay.NewSimulatedClock(candidate.ReplayStart)
		if err != nil {
			t.Fatal(err)
		}
		delay := 4 * time.Second
		config := engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 256, RequiredReserve: 8, EvaluationDelay: &delay}
		if fastForward {
			config.ReplayObservationStart = &observationStart
		}
		owner, err := engine.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		admission, installed := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
		if admission != engine.AdmissionAdmitted || (<-installed).Code != engine.DispositionBindingInstalled {
			t.Fatal("binding install failed")
		}
		source, err := replay.NewSourceThrough(handle, owner, clock, replay.Unpaced(), observationEnd)
		if err != nil {
			t.Fatal(err)
		}
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		var first engine.ReplayDeterministicView
		var firstStatus engine.ReplayStatus
		var qualificationTrace [][]engine.ReplayQualificationView
		for group := candidate.ReplayStart; !group.After(observationEnd); group = group.Add(time.Second) {
			step, err := source.Step(context.Background())
			if err != nil || step.LogicalTime != group {
				t.Fatalf("step %s: %+v err=%v", group, step, err)
			}
			if !group.After(observationStart) {
				boundary := owner.ObserveReplayDeterministic()
				qualifications := make([]engine.ReplayQualificationView, len(boundary.Canonical))
				for index := range boundary.Canonical {
					qualifications[index] = boundary.Canonical[index].Qualification
				}
				qualificationTrace = append(qualificationTrace, qualifications)
			}
			if group == observationStart {
				first, firstStatus = owner.ObserveReplayDeterministic(), owner.ObserveReplay()
			}
		}
		result, err := source.Finish(context.Background())
		if err != nil || result.Outcome != replay.OutcomeComplete {
			t.Fatalf("finish: %+v err=%v", result, err)
		}
		return observation{view: first, status: firstStatus, result: result, qualificationTrace: qualificationTrace}
	}

	baseline, optimized := run(false), run(true)
	if !reflect.DeepEqual(baseline.qualificationTrace, optimized.qualificationTrace) {
		t.Fatalf("warm-up qualification progression diverged:\nbaseline=%+v\noptimized=%+v", baseline.qualificationTrace, optimized.qualificationTrace)
	}
	if !reflect.DeepEqual(baseline.view.Canonical, optimized.view.Canonical) || !reflect.DeepEqual(baseline.view.Evaluation, optimized.view.Evaluation) ||
		!reflect.DeepEqual(baseline.view.Publication.AggregateEvaluation, optimized.view.Publication.AggregateEvaluation) ||
		!reflect.DeepEqual(baseline.view.Publication.Watermark, optimized.view.Publication.Watermark) || baseline.status.LastLogicalTime != optimized.status.LastLogicalTime ||
		!reflect.DeepEqual(baseline.status.CommittedT, optimized.status.CommittedT) || baseline.status.PresentSlots != optimized.status.PresentSlots || baseline.status.ProvenAbsentSlots != optimized.status.ProvenAbsentSlots {
		t.Fatalf("first observation diverged:\nbaseline=%+v\noptimized=%+v", baseline, optimized)
	}
	if optimized.status.CommittedT == nil || *optimized.status.CommittedT != observationStart.Add(-4*time.Second) || optimized.status.LastLogicalTime != observationStart {
		t.Fatalf("optimized boundary clocks = %+v", optimized.status)
	}
	var dense, quiet, noPrint *engine.ReplayCanonicalSymbol
	for index := range optimized.view.Canonical {
		symbol := &optimized.view.Canonical[index]
		switch symbol.Symbol {
		case "S0000":
			dense = symbol
		case "S0001":
			quiet = symbol
		case "S0002":
			noPrint = symbol
		}
	}
	if dense == nil || quiet == nil || noPrint == nil || dense.Qualification.Status != "finalized" || quiet.Qualification.Status == "finalized" || len(noPrint.Records) != 0 {
		t.Fatalf("dense/quiet/no-print semantic fixture = dense=%+v quiet=%+v noPrint=%+v", dense, quiet, noPrint)
	}
	for name, result := range map[string]replay.Result{"baseline": baseline.result, "optimized": optimized.result} {
		a := result.Accounting
		if a.CompletedRecordDispositions+a.IntentionallyUnappliedSuffixRecords != a.ArtifactRecords || a.CompletedGroups != a.PlannedGroups || a.ActiveGroup != 0 || a.RemainingGroups != 0 {
			t.Fatalf("%s terminal accounting = %+v", name, a)
		}
	}
}

// TestRetainedReplayPerformanceAcceptance is the explicitly selected local
// acceptance tier. It opens the retained artifact in place only when both
// private paths are supplied and reports no artifact path, identity, or rows.
func TestRetainedReplayPerformanceAcceptance(t *testing.T) {
	if testing.Short() {
		t.Skip("retained artifact acceptance is not ordinary verification")
	}
	artifact := os.Getenv("REPLAY_B4_ARTIFACT")
	referenceDirectory := os.Getenv("REPLAY_B4_REFERENCE_DIR")
	if artifact == "" || referenceDirectory == "" {
		t.Skip("retained artifact acceptance paths are not configured")
	}
	info, err := os.Lstat(artifact)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != 2_584_011_150 {
		t.Fatal("retained artifact does not match the approved file manifest")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	validationStarted := time.Now()
	runtime, err := Prepare(ctx, StartupConfig{ArtifactPath: artifact, ReferenceDirectory: referenceDirectory, ObservationStart: "09:30:00", ObservationEnd: "09:35:00"})
	validationDuration := time.Since(validationStarted)
	if err != nil {
		t.Fatalf("retained artifact validation failed: %v", err)
	}
	t.Logf("segment complete: validation=%s", validationDuration)
	defer runtime.Close()
	wantStart, startErr := observationTime("2026-08-07", "04:00:00")
	wantEnd, endErr := observationTime("2026-08-07", "20:00:00")
	metadata := runtime.metadata
	if startErr != nil || endErr != nil || metadata.Mode != replayartifact.CompleteFinalBars || metadata.TradingDate != "2026-08-07" ||
		metadata.ReplayStart != wantStart || metadata.ReplayEnd != wantEnd || metadata.AggregateRecords != 7_671_171 || metadata.CoverageEntries != 5_691 {
		t.Fatal("retained artifact does not match the approved semantic manifest")
	}

	var runStarted, playbackReady, observingAt, finalizingAt, retainedAt time.Time
	var hookErr error
	var priorPublication string
	consecutiveChanges, longestChangingRun := 0, 0
	var maximumLagMS uint64
	runtime.afterStart = func() {
		playbackReady = time.Now()
		t.Logf("segment complete: construction=%s", playbackReady.Sub(runStarted))
	}
	runtime.afterPublish = func(phase operations.ReplayPhase) {
		now := time.Now()
		switch phase {
		case operations.ReplayObserving, operations.ReplayFinalizing:
			capture, captureErr := runtime.CaptureSnapshot()
			if captureErr != nil {
				hookErr = captureErr
				return
			}
			mapped, mapErr := snapshotapi.Map(capture)
			if mapErr != nil {
				hookErr = mapErr
				return
			}
			if mapped.Replay.ScheduleLagMS != nil && *mapped.Replay.ScheduleLagMS > maximumLagMS {
				maximumLagMS = *mapped.Replay.ScheduleLagMS
			}
			if phase == operations.ReplayObserving {
				if mapped.Ranking.Mode != "qualified_current" || mapped.Ranking.Reason != "" || len(mapped.Rows) == 0 ||
					mapped.Accounting.Population.UnresolvedPopulation != 0 {
					hookErr = fmt.Errorf("retained observation did not expose complete ranked rows: mode=%s reason=%s rows=%d unresolved=%d",
						mapped.Ranking.Mode, mapped.Ranking.Reason, len(mapped.Rows), mapped.Accounting.Population.UnresolvedPopulation)
					return
				}
				if observingAt.IsZero() {
					observingAt = now
					t.Logf("segment complete: warmup=%s", observingAt.Sub(playbackReady))
				}
				if priorPublication == "" || mapped.Publication.ID != priorPublication {
					consecutiveChanges++
				} else {
					consecutiveChanges = 1
				}
				if consecutiveChanges > longestChangingRun {
					longestChangingRun = consecutiveChanges
				}
				priorPublication = mapped.Publication.ID
			} else {
				finalizingAt = now
				t.Logf("segment complete: observation=%s", finalizingAt.Sub(observingAt))
			}
		case operations.ReplayRetainedSuccess:
			retainedAt = now
			t.Logf("segment complete: finalization=%s", retainedAt.Sub(finalizingAt))
		}
	}
	runStarted = time.Now()
	result, runErr := runtime.Run(ctx)
	if runErr != nil || hookErr != nil || result.Outcome != replay.OutcomeComplete || result.Completion != replay.CompletionRequestedEnd {
		t.Fatalf("retained replay failed: outcome=%s completion=%s run_error=%v capture_error=%v", result.Outcome, result.Completion, runErr, hookErr)
	}
	if playbackReady.IsZero() || observingAt.IsZero() || finalizingAt.IsZero() || retainedAt.IsZero() {
		t.Fatal("retained replay did not expose every measured phase boundary")
	}
	constructionDuration := playbackReady.Sub(runStarted)
	warmupDuration := observingAt.Sub(playbackReady)
	observationDuration := finalizingAt.Sub(observingAt)
	finalizationDuration := retainedAt.Sub(finalizingAt)
	accounting := result.Accounting
	if accounting.ArtifactRecords != 7_671_171 || accounting.CompletedRecordDispositions+accounting.IntentionallyUnappliedSuffixRecords != accounting.ArtifactRecords ||
		accounting.UnreadRecords != 0 || accounting.CompletedGroups != accounting.PlannedGroups || accounting.ActiveGroup != 0 || accounting.RemainingGroups != 0 {
		t.Fatal("retained replay terminal accounting did not reconcile")
	}
	if longestChangingRun < 3 {
		t.Fatalf("retained observation had only %d consecutive changing API publications", longestChangingRun)
	}
	if validationDuration > 3*time.Minute || constructionDuration > 5*time.Second || warmupDuration > 3*time.Minute ||
		observationDuration < 5*time.Minute || observationDuration > 5*time.Minute+15*time.Second || finalizationDuration > 3*time.Minute {
		t.Fatalf("retained replay missed the acceptance envelope: validation=%s construction=%s warmup=%s observation=%s finalization=%s", validationDuration, constructionDuration, warmupDuration, observationDuration, finalizationDuration)
	}
	shutdown, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatalf("retained replay shutdown failed: %v", err)
	}
	t.Logf("retained manifest: bytes=%d records=%d symbols=%d complete_session=%s..%s", info.Size(), metadata.AggregateRecords, metadata.CoverageEntries, metadata.ReplayStart.Format(time.RFC3339), metadata.ReplayEnd.Format(time.RFC3339))
	t.Logf("segments: validation=%s construction=%s warmup=%s observation=%s finalization=%s; longest_changing_publication_run=%d max_schedule_lag_ms=%d", validationDuration, constructionDuration, warmupDuration, observationDuration, finalizationDuration, longestChangingRun, maximumLagMS)
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
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil))
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

func newSemanticReplayFixture(t *testing.T) (replayFixture, time.Time, time.Time) {
	base := newReplayFixtureSymbols(t, time.Second, 3)
	binding, err := cachedBinding(context.Background(), "2026-08-06", base.referenceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	start := binding.SessionStart()
	observationStart := start.Add(17*time.Minute + 5*time.Second)
	observationEnd := observationStart.Add(3 * time.Second)
	artifactEnd := observationEnd.Add(2 * time.Second)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		parts := strings.Split(request.URL.Path, "/")
		if len(parts) < 5 || !strings.HasPrefix(request.URL.Path, "/v2/aggs/ticker/") {
			http.NotFound(writer, request)
			return
		}
		symbol := parts[4]
		results := make([]map[string]any, 0, 61)
		switch symbol {
		case "S0000":
			for second := 0; second < 61; second++ {
				price := 10 + float64(second)/100
				results = append(results, map[string]any{"t": start.Add(time.Duration(second) * time.Second).UnixMilli(), "o": price, "h": price + .1, "l": price - .1, "c": price, "v": 10_000, "vw": price, "n": 1_000})
			}
		case "S0001":
			for second := 0; second < 60; second += 10 {
				results = append(results, map[string]any{"t": start.Add(time.Duration(second) * time.Second).UnixMilli(), "o": 10, "h": 10.1, "l": 9.9, "c": 10, "v": 100, "vw": 10, "n": 10})
			}
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "ticker": symbol, "adjusted": false, "results": results})
	}))
	defer server.Close()
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test", nil }, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: start, End: artifactEnd, Workers: 3,
		DestinationDirectory: root, Limits: replayartifact.Limits{MaximumNormalizedRecords: 4096, MaximumResponseBytes: 1 << 20, MaximumArtifactBytes: 1 << 20,
			MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 4096}}, downloader)
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("semantic compile = %+v", compiled)
	}
	return replayFixture{artifact: compiled.Path, referenceDirectory: base.referenceDirectory, start: start, symbols: 3}, observationStart, observationEnd
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

func (t *testTimeline) advance(value time.Duration) {
	t.mu.Lock()
	t.now = t.now.Add(value)
	t.mu.Unlock()
}

func (t *testTimeline) current() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.now
}

func assertFinalReplaySnapshot(t *testing.T, snapshot snapshotapi.Snapshot, completion replay.CompletionDisposition) {
	t.Helper()
	if snapshot.Replay == nil || snapshot.Replay.Phase != "retained_success" || snapshot.Replay.Completion != string(completion) ||
		snapshot.Publication.Lifecycle != "ended" || snapshot.Publication.RunMode != "replay" || snapshot.Status.BackendReady || snapshot.Status.ReadinessReason != "not_live_mode" ||
		snapshot.Status.RankingCurrent || snapshot.Replay.Source.CompletedRuns != "1" || snapshot.Replay.Source.FailedRuns != "0" || snapshot.Replay.Source.CanceledRuns != "0" {
		t.Fatalf("final replay snapshot = %+v", snapshot)
	}
	for _, row := range snapshot.Rows {
		if row.TapeRate.Status != "unavailable" || row.TapeRate.Reason != "replay_unavailable" || row.Spread.Status != "unavailable" || row.Spread.Reason != "replay_unavailable" {
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
