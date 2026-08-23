package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

func TestWatermarkStaleRecorderExactTransitionPersistsOneBoundedIncident(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "diagnostics")
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	evidenceBase := base.Add(-4 * time.Second)
	evidence := watermarkTestEvidence(evidenceBase, 5)
	var output bytes.Buffer
	recorder := &watermarkStaleDiagnosticRecorder{}
	if err := recorder.observe(watermarkTestSample(base, true, operations.ReasonNone, "live", engine.RunModeLive), func() operations.WatermarkStallEvidence { return evidence }, directory, &output, persistWatermarkStaleDiagnostic); err != nil {
		t.Fatal(err)
	}
	if err := recorder.observe(watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive), func() operations.WatermarkStallEvidence { return evidence }, directory, &output, persistWatermarkStaleDiagnostic); err != nil {
		t.Fatal(err)
	}
	if recorder.attempts != 1 || !recorder.latched || !recorder.persisted || !strings.Contains(output.String(), "Watermark-stale diagnostic persisted") {
		t.Fatalf("recorder=%+v output=%q", recorder, output.String())
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("incident entries=%v err=%v", entries, err)
	}
	info, err := os.Stat(filepath.Join(directory, entries[0].Name()))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode=%v err=%v", info.Mode().Perm(), err)
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil || directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode=%v err=%v", directoryInfo.Mode().Perm(), err)
	}
	data, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var document watermarkStaleDiagnosticDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("JSON=%s err=%v", data, err)
	}
	if document.Schema != watermarkStaleDiagnosticSchema || document.RingCapacity != operations.WatermarkStallRingCapacity || document.RecordCount != 5 || len(document.Cycles) != 5 ||
		document.Trigger.PreviousBackendReady != true || document.Trigger.CurrentBackendReady || document.Trigger.CurrentReason != string(operations.ReasonWatermarkStale) || document.Trigger.RunMode != string(engine.RunModeLive) || document.Trigger.Lifecycle != "live" {
		t.Fatalf("document=%+v", document)
	}
	if !document.Cycles[0].CycleStartedAt.Equal(evidenceBase) || !document.Cycles[4].CycleStartedAt.Equal(base) ||
		document.Cycles[0].EvaluationStageNS != int64(20*time.Millisecond) || document.Cycles[0].EvaluationApplyNS != int64(3*time.Millisecond) ||
		document.Cycles[0].EvaluationPublicationNS != int64(time.Millisecond) || document.Cycles[0].ProcessingDelayOneSecondNS != int64(7*time.Millisecond) ||
		document.Cycles[0].GCCycles != 11 || document.Cycles[0].LastGCPauseNS != 3 {
		t.Fatalf("cycle evidence=%+v", document.Cycles[0])
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema", "trigger", "ring_capacity", "record_count", "cycles"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing top-level JSON key %q: %s", key, data)
		}
	}
	// Decode the first element with an explicit map so field names remain
	// independently asserted rather than relying only on Go struct tags.
	var cycles []map[string]json.RawMessage
	if decodeErr := json.Unmarshal(raw["cycles"], &cycles); decodeErr != nil || len(cycles) != 5 {
		t.Fatalf("cycle JSON=%s err=%v", raw["cycles"], decodeErr)
	}
	cycleRaw := cycles[0]
	for _, key := range []string{"cycle_started_at", "cycle_completed_at", "coverage_fence_disposition", "timer_policy", "evaluation_stage_ns", "evaluation_apply_ns", "evaluation_publication_ns", "watermark_before", "watermark_after", "queue_current_frames", "queue_high_frames", "queue_current_bytes", "queue_high_bytes", "processing_delay_ns", "processing_delay_one_second_ns", "tq_pressure", "tq_pressure_cause", "heap_alloc_bytes", "heap_in_use_bytes", "gc_cycles", "gc_pause_total_ns", "last_gc_pause_ns"} {
		if _, ok := cycleRaw[key]; !ok {
			t.Fatalf("missing cycle JSON key %q: %s", key, data)
		}
	}
}

func TestWatermarkStaleSharedTransitionLatchCatchesPhaseOffsetTransientMissedByOneSecondScannerSamples(t *testing.T) {
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	staleStart, staleEnd := base.Add(1250*time.Millisecond), base.Add(1300*time.Millisecond)
	statusAt := func(at time.Time) operations.Status {
		if !at.Before(staleStart) && at.Before(staleEnd) {
			return watermarkTestSample(at, false, operations.ReasonWatermarkStale, "live", engine.RunModeLive).Status
		}
		return watermarkTestSample(at, true, operations.ReasonNone, "live", engine.RunModeLive).Status
	}

	// The scanner's former one-second phase samples at 0s, 1s, and 2s and
	// misses the entire 50ms interval. A launcher phase shifted by 250ms sees it
	// at 1.25s, matching the owner-run failure that motivated this correction.
	for _, offset := range []time.Duration{0, time.Second, 2 * time.Second} {
		if got := statusAt(base.Add(offset)); !got.BackendReady {
			t.Fatalf("former scanner phase unexpectedly saw stale at %s", offset)
		}
	}
	if got := statusAt(base.Add(1250 * time.Millisecond)); got.BackendReady || got.Reason != operations.ReasonWatermarkStale {
		t.Fatalf("launcher phase did not see transient: %+v", got)
	}

	// CaptureSnapshot retains the launcher-observed crossing in operations;
	// the scanner recorder may receive it after the backend has already become
	// ready again. No recorder timer has to land inside the 50ms interval.
	recorder := &watermarkStaleDiagnosticRecorder{}
	attempts := 0
	persist := func(string, *watermarkStaleDiagnosticDocument) (string, error) {
		attempts++
		return "watermark-stale-phase-offset.json", nil
	}
	transition := operations.WatermarkStaleTransition{
		Previous: statusAt(base.Add(time.Second)),
		Current:  statusAt(base.Add(1250 * time.Millisecond)),
	}
	if err := recorder.observeTransition(transition, func() operations.WatermarkStallEvidence {
		return watermarkTestEvidence(base, 2)
	}, "diagnostics", nil, persist); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || !recorder.latched || !recorder.persisted {
		t.Fatalf("retained phase-offset transition was not persisted: recorder=%+v attempts=%d", recorder, attempts)
	}
}

func TestWatermarkStaleRecorderOneProcessBoundAcrossRepeatedStaleAndRecovery(t *testing.T) {
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	recorder := &watermarkStaleDiagnosticRecorder{}
	attempts := 0
	persist := func(string, *watermarkStaleDiagnosticDocument) (string, error) {
		attempts++
		return "watermark-stale-test.json", nil
	}
	evidence := func() operations.WatermarkStallEvidence { return watermarkTestEvidence(base, 1) }
	observe := func(sample liveOperatorSample) {
		if err := recorder.observe(sample, evidence, "diagnostics", nil, persist); err != nil {
			t.Fatal(err)
		}
	}
	observe(watermarkTestSample(base, true, operations.ReasonNone, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(2*time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(3*time.Second), true, operations.ReasonNone, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(4*time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive))
	if attempts != 1 || recorder.attempts != 1 {
		t.Fatalf("one-process incident bound attempts=%d recorder=%+v", attempts, recorder)
	}
}

func TestWatermarkStaleRecorderExcludesNonOrdinaryOrNonWatermarkStates(t *testing.T) {
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	current := watermarkTestSample(base, true, operations.ReasonNone, "live", engine.RunModeLive)
	cases := []struct {
		name      string
		candidate liveOperatorSample
	}{
		{"startup", watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "initializing", engine.RunModeLive)},
		{"hydration", watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "hydrating", engine.RunModeLive)},
		{"recovery", watermarkTestSample(base.Add(time.Second), false, operations.ReasonLifecycle, "recovering", engine.RunModeLive)},
		{"replay", watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeReplay)},
		{"shutdown", watermarkTestSampleWithProcess(base.Add(time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive, false)},
		{"tq-only", watermarkTestSample(base.Add(time.Second), false, operations.ReasonRankingNoncurrent, "live", engine.RunModeLive)},
		{"accounting", watermarkTestSample(base.Add(time.Second), false, operations.ReasonAccounting, "live", engine.RunModeLive)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			recorder := &watermarkStaleDiagnosticRecorder{}
			attempts := 0
			persist := func(string, *watermarkStaleDiagnosticDocument) (string, error) { attempts++; return "path", nil }
			if err := recorder.observe(current, nil, "", nil, persist); err != nil {
				t.Fatal(err)
			}
			if err := recorder.observe(test.candidate, nil, "", nil, persist); err != nil {
				t.Fatal(err)
			}
			if attempts != 0 || recorder.latched {
				t.Fatalf("excluded state triggered recorder=%+v attempts=%d", recorder, attempts)
			}
		})
	}
}

func TestWatermarkStalePersistenceFailureIsNonfatalAndLatched(t *testing.T) {
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	recorder := &watermarkStaleDiagnosticRecorder{}
	var output bytes.Buffer
	attempts := 0
	persist := func(string, *watermarkStaleDiagnosticDocument) (string, error) {
		attempts++
		return "", errors.New("read-only diagnostics")
	}
	observe := func(sample liveOperatorSample) {
		if err := recorder.observe(sample, func() operations.WatermarkStallEvidence { return watermarkTestEvidence(base, 2) }, "", &output, persist); err != nil {
			t.Fatalf("persistence error reached scanner control path: %v", err)
		}
	}
	observe(watermarkTestSample(base, true, operations.ReasonNone, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive))
	observe(watermarkTestSample(base.Add(2*time.Second), false, operations.ReasonWatermarkStale, "live", engine.RunModeLive))
	if attempts != 1 || recorder.attempts != 1 || recorder.persisted || !recorder.latched || !strings.Contains(output.String(), "persistence failed") {
		t.Fatalf("failure was not bounded/nonfatal recorder=%+v attempts=%d output=%q", recorder, attempts, output.String())
	}
}

func TestWatermarkStalePersistenceIsProtectedAndCreateWithoutOverwrite(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "diagnostics")
	base := time.Date(2026, 8, 20, 16, 0, 0, 123, time.UTC)
	document := newWatermarkStaleDiagnosticDocument(
		watermarkStaleStatusSample{BackendReady: true, ProcessLive: true, Lifecycle: "live", RunMode: engine.RunModeLive, SampledAt: base},
		watermarkStaleStatusSample{ProcessLive: true, Lifecycle: "live", RunMode: engine.RunModeLive, Reason: operations.ReasonWatermarkStale, SampledAt: base.Add(time.Second)},
		watermarkTestEvidence(base, 1),
	)
	path, err := persistWatermarkStaleDiagnostic(directory, &document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistWatermarkStaleDiagnostic(directory, &document); err == nil {
		t.Fatal("second persistence overwrote existing watermark-stale file")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || filepath.Base(path) != entries[0].Name() {
		t.Fatalf("files=%v path=%s err=%v", entries, path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{"AAA", "symbol", "provider", "credential", "payload", "MASSIVE_API_KEY", "environment", "raw_frame"} {
		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(prohibited)) {
			t.Fatalf("prohibited diagnostic content %q in %s", prohibited, data)
		}
	}
}

func watermarkTestSample(at time.Time, ready bool, reason operations.ReadinessReason, lifecycle string, runMode engine.RunMode) liveOperatorSample {
	return watermarkTestSampleWithProcess(at, ready, reason, lifecycle, runMode, true)
}

func watermarkTestSampleWithProcess(at time.Time, ready bool, reason operations.ReadinessReason, lifecycle string, runMode engine.RunMode, processLive bool) liveOperatorSample {
	target := at.Add(-4 * time.Second)
	return liveOperatorSample{
		Status:  operations.Status{ProcessLive: processLive, BackendReady: ready, RankingCurrent: ready, Reason: reason, Lifecycle: lifecycle, SampledAt: at, WatermarkLag: 3 * time.Second, CausalTarget: &target},
		Metrics: operations.Metrics{Engine: engine.OperationalView{RunMode: runMode, Lifecycle: lifecycle}},
	}
}

func watermarkTestEvidence(base time.Time, count int) operations.WatermarkStallEvidence {
	records := make([]operations.WatermarkStallCycleEvidence, count)
	for index := range records {
		at := base.Add(time.Duration(index) * time.Second)
		target := at.Add(-4 * time.Second)
		records[index] = operations.WatermarkStallCycleEvidence{
			CycleStartedAt: at, CycleCompletedAt: at.Add(50 * time.Millisecond), CycleDuration: 50 * time.Millisecond, DiagnosticSampledAt: at,
			CoverageFenceDisposition: engine.DispositionLiveCoverageFenceApplied, CoverageFenceFinalization: 12 * time.Millisecond, CoverageFenceTotal: 17 * time.Millisecond, CoverageFenceValid: true,
			TimerPolicy: operations.WatermarkStallTimerMaintenanceOnly, EvaluationObserved: true, EvaluationSource: engine.AggregateEvaluationLiveCoverageFence, EvaluationTarget: target,
			EvaluationStage: 20 * time.Millisecond, EvaluationApply: 3 * time.Millisecond, EvaluationPublication: time.Millisecond,
			WatermarkBefore: target.Add(-2 * time.Second), WatermarkAfter: target.Add(-time.Second), WatermarkBeforePresent: true, WatermarkAfterPresent: true,
			CausalTargetBefore: target, CausalTargetAfter: target, CausalTargetBeforePresent: true, CausalTargetAfterPresent: true, WatermarkLagBefore: 2 * time.Second, WatermarkLagAfter: time.Second,
			PublicationIDBefore: uint64(index + 1), PublicationIDAfter: uint64(index + 2), EngineSequenceBefore: uint64(index + 10), EngineSequenceAfter: uint64(index + 11), PublicationGeneratedBefore: at, PublicationGeneratedAfter: at.Add(50 * time.Millisecond),
			QueueCurrentFrames: 2, QueueHighFrames: 8, QueueCurrentBytes: 2048, QueueHighBytes: 8192, QueueOldestWaitingAge: 5 * time.Millisecond, ProcessingDelay: 6 * time.Millisecond, ProcessingDelayOneSecond: 7 * time.Millisecond,
			TQPressure: engine.TQPressureNormal, TQPressureCause: engine.TQPressureCauseNone, HeapAllocBytes: 1000, HeapInUseBytes: 2000, Goroutines: 4, GCCycles: 11, GCPauseTotalNS: 9, LastGCPauseNS: 3,
		}
	}
	return operations.WatermarkStallEvidence{Capacity: operations.WatermarkStallRingCapacity, Records: records}
}
