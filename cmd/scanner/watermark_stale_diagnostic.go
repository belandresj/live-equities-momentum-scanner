package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

const watermarkStaleDiagnosticSchema = 1

// The launcher and dashboard poll readiness once per second. Drain the shared
// runtime transition latch at a finer cadence so persistence follows promptly;
// correctness does not depend on this timer landing inside the stale interval.
const watermarkStaleReadinessCadence = 100 * time.Millisecond

type watermarkStaleStatusSample struct {
	ProcessLive, BackendReady bool
	Reason                    operations.ReadinessReason
	Lifecycle                 string
	RunMode                   engine.RunMode
	SampledAt                 time.Time
	WatermarkLag              time.Duration
	CausalTarget              *time.Time
}

// watermarkStaleDiagnosticRecorder owns only scanner-local transition memory
// and persistence. It does not own readiness or engine lifecycle state.
type watermarkStaleDiagnosticRecorder struct {
	previous  watermarkStaleStatusSample
	hasPrior  bool
	latched   bool
	persisted bool
	attempts  int
}

func (r *watermarkStaleDiagnosticRecorder) observe(sample liveOperatorSample,
	evidence func() operations.WatermarkStallEvidence, directory string, output io.Writer,
	persist func(string, *watermarkStaleDiagnosticDocument) (string, error)) error {
	if r == nil {
		return nil
	}
	current := watermarkStaleStatusFromSample(sample)
	previous := r.previous
	trigger := r.hasPrior && watermarkStaleTriggerMatches(previous, current)
	r.previous, r.hasPrior = current, true
	if !trigger {
		return nil
	}
	return r.persistTransition(previous, current, evidence, directory, output, persist)
}

func (r *watermarkStaleDiagnosticRecorder) observeTransition(transition operations.WatermarkStaleTransition,
	evidence func() operations.WatermarkStallEvidence, directory string, output io.Writer,
	persist func(string, *watermarkStaleDiagnosticDocument) (string, error)) error {
	if r == nil {
		return nil
	}
	previous := watermarkStaleStatusFromStatus(transition.Previous)
	current := watermarkStaleStatusFromStatus(transition.Current)
	if !watermarkStaleTriggerMatches(previous, current) {
		return nil
	}
	r.previous, r.hasPrior = current, true
	if transition.ActiveObserved {
		baseEvidence := evidence
		evidence = func() operations.WatermarkStallEvidence {
			var result operations.WatermarkStallEvidence
			if baseEvidence != nil {
				result = baseEvidence()
			}
			active := transition.Active
			result.Active = &active
			return result
		}
	}
	return r.persistTransition(previous, current, evidence, directory, output, persist)
}

func (r *watermarkStaleDiagnosticRecorder) persistTransition(previous, current watermarkStaleStatusSample,
	evidence func() operations.WatermarkStallEvidence, directory string, output io.Writer,
	persist func(string, *watermarkStaleDiagnosticDocument) (string, error)) error {
	if r.latched {
		return nil
	}
	// Latch before taking the detached ring snapshot and before filesystem I/O.
	// A slow, failing, or colliding diagnostic write cannot be retried by a
	// later status sample and cannot become a runtime control result.
	r.latched = true
	r.attempts++
	var tail operations.WatermarkStallEvidence
	if evidence != nil {
		tail = evidence()
	}
	document := newWatermarkStaleDiagnosticDocument(previous, current, tail)
	if persist == nil {
		return r.reportPersistenceFailure(output, errors.New("watermark-stale diagnostic persistence is unavailable"))
	}
	path, err := persist(directory, &document)
	if err != nil {
		return r.reportPersistenceFailure(output, err)
	}
	r.persisted = true
	if output != nil {
		_, _ = fmt.Fprintf(output, "Watermark-stale diagnostic persisted · %s\n", path)
	}
	return nil
}

func (r *watermarkStaleDiagnosticRecorder) observeStatus(status operations.Status,
	evidence func() operations.WatermarkStallEvidence, directory string, output io.Writer,
	persist func(string, *watermarkStaleDiagnosticDocument) (string, error)) error {
	return r.observe(liveOperatorSample{
		Status: status,
		Metrics: operations.Metrics{Engine: engine.OperationalView{
			// deriveStatus can return ready or watermark_stale only after its
			// RunModeLive guard has passed. Retain that already-proved fact for
			// the diagnostic trigger/document without a second engine read.
			RunMode:   engine.RunModeLive,
			Lifecycle: status.Lifecycle,
		}},
	}, evidence, directory, output, persist)
}

func watermarkStaleStatusFromStatus(status operations.Status) watermarkStaleStatusSample {
	return watermarkStaleStatusSample{
		ProcessLive: status.ProcessLive, BackendReady: status.BackendReady,
		Reason: status.Reason, Lifecycle: status.Lifecycle, RunMode: engine.RunModeLive,
		SampledAt: status.SampledAt, WatermarkLag: status.WatermarkLag, CausalTarget: cloneDiagnosticTime(status.CausalTarget),
	}
}

func (r *watermarkStaleDiagnosticRecorder) reportPersistenceFailure(output io.Writer, err error) error {
	if output != nil {
		_, _ = fmt.Fprintf(output, "Watermark-stale diagnostic persistence failed · attempt %d/1 · %v\n", r.attempts, err)
	}
	// Persistence is observability-local. Deliberately do not return the error
	// to the live supervisor's shutdown or runtime-control path.
	return nil
}

func watermarkStaleStatusFromSample(sample liveOperatorSample) watermarkStaleStatusSample {
	return watermarkStaleStatusSample{
		ProcessLive: sample.Status.ProcessLive, BackendReady: sample.Status.BackendReady,
		Reason: sample.Status.Reason, Lifecycle: sample.Status.Lifecycle,
		RunMode: sample.Metrics.Engine.RunMode, SampledAt: sample.Status.SampledAt,
		WatermarkLag: sample.Status.WatermarkLag, CausalTarget: cloneDiagnosticTime(sample.Status.CausalTarget),
	}
}

func watermarkStaleTriggerMatches(previous, current watermarkStaleStatusSample) bool {
	return previous.ProcessLive && previous.BackendReady && previous.RunMode == engine.RunModeLive && previous.Lifecycle == "live" &&
		current.ProcessLive && !current.BackendReady && current.RunMode == engine.RunModeLive && current.Lifecycle == "live" &&
		current.Reason == operations.ReasonWatermarkStale
}

type watermarkStaleDiagnosticDocument struct {
	Schema       int                                  `json:"schema"`
	Trigger      watermarkStaleDiagnosticTrigger      `json:"trigger"`
	RingCapacity int                                  `json:"ring_capacity"`
	RecordCount  int                                  `json:"record_count"`
	Cycles       []watermarkStaleDiagnosticCycle      `json:"cycles"`
	ActiveCycle  *watermarkStaleDiagnosticActiveCycle `json:"active_cycle,omitempty"`
}

type watermarkStaleDiagnosticActiveCycle struct {
	Sequence                 uint64     `json:"sequence"`
	StartedAt                time.Time  `json:"started_at"`
	PhaseStartedAt           time.Time  `json:"phase_started_at"`
	ObservedAt               time.Time  `json:"observed_at"`
	ElapsedNS                int64      `json:"elapsed_ns"`
	PhaseElapsedNS           int64      `json:"phase_elapsed_ns"`
	Phase                    string     `json:"phase"`
	EvaluationActive         bool       `json:"evaluation_active"`
	EvaluationEngineSequence uint64     `json:"evaluation_engine_sequence"`
	EvaluationSource         string     `json:"evaluation_source"`
	EvaluationTarget         *time.Time `json:"evaluation_target,omitempty"`
	EvaluationPhase          string     `json:"evaluation_phase"`
	EvaluationStartedAt      *time.Time `json:"evaluation_started_at,omitempty"`
	EvaluationPhaseStartedAt *time.Time `json:"evaluation_phase_started_at,omitempty"`
	EvaluationElapsedNS      int64      `json:"evaluation_elapsed_ns"`
	EvaluationPhaseElapsedNS int64      `json:"evaluation_phase_elapsed_ns"`
}

type watermarkStaleDiagnosticTrigger struct {
	PreviousBackendReady  bool       `json:"previous_backend_ready"`
	CurrentBackendReady   bool       `json:"current_backend_ready"`
	CurrentReason         string     `json:"current_reason"`
	RunMode               string     `json:"run_mode"`
	Lifecycle             string     `json:"lifecycle"`
	PreviousSampledAt     time.Time  `json:"previous_sampled_at"`
	CurrentSampledAt      time.Time  `json:"current_sampled_at"`
	CurrentWatermarkLagNS int64      `json:"current_watermark_lag_ns"`
	CurrentCausalTarget   *time.Time `json:"current_causal_target,omitempty"`
}

type watermarkStaleDiagnosticCycle struct {
	CycleStartedAt              time.Time  `json:"cycle_started_at"`
	CycleCompletedAt            time.Time  `json:"cycle_completed_at"`
	CycleDurationNS             int64      `json:"cycle_duration_ns"`
	DiagnosticSampledAt         time.Time  `json:"diagnostic_sampled_at"`
	CoverageFenceDisposition    string     `json:"coverage_fence_disposition"`
	LiveCoverageEnqueueNS       int64      `json:"live_coverage_enqueue_ns"`
	LiveCoverageCompletionNS    int64      `json:"live_coverage_completion_ns"`
	CoverageFenceFinalizationNS int64      `json:"coverage_fence_finalization_ns"`
	CoverageFenceTotalNS        int64      `json:"coverage_fence_total_ns"`
	CoverageFenceValid          bool       `json:"coverage_fence_valid"`
	TimerPolicy                 string     `json:"timer_policy"`
	EvaluationObserved          bool       `json:"evaluation_observed"`
	EvaluationSource            string     `json:"evaluation_source"`
	EvaluationTarget            *time.Time `json:"evaluation_target,omitempty"`
	EvaluationStageNS           int64      `json:"evaluation_stage_ns"`
	EvaluationApplyNS           int64      `json:"evaluation_apply_ns"`
	EvaluationPublicationNS     int64      `json:"evaluation_publication_ns"`
	WatermarkBefore             *time.Time `json:"watermark_before,omitempty"`
	WatermarkAfter              *time.Time `json:"watermark_after,omitempty"`
	CausalTargetBefore          *time.Time `json:"causal_target_before,omitempty"`
	CausalTargetAfter           *time.Time `json:"causal_target_after,omitempty"`
	WatermarkLagBeforeNS        int64      `json:"watermark_lag_before_ns"`
	WatermarkLagAfterNS         int64      `json:"watermark_lag_after_ns"`
	PublicationIDBefore         uint64     `json:"publication_id_before"`
	PublicationIDAfter          uint64     `json:"publication_id_after"`
	EngineSequenceBefore        uint64     `json:"engine_sequence_before"`
	EngineSequenceAfter         uint64     `json:"engine_sequence_after"`
	PublicationGeneratedBefore  *time.Time `json:"publication_generated_before,omitempty"`
	PublicationGeneratedAfter   *time.Time `json:"publication_generated_after,omitempty"`
	QueueCurrentFrames          uint64     `json:"queue_current_frames"`
	QueueHighFrames             uint64     `json:"queue_high_frames"`
	QueueCurrentBytes           int        `json:"queue_current_bytes"`
	QueueHighBytes              int        `json:"queue_high_bytes"`
	QueueOldestWaitingAgeNS     int64      `json:"queue_oldest_waiting_age_ns"`
	ProcessingDelayNS           int64      `json:"processing_delay_ns"`
	ProcessingDelayOneSecondNS  int64      `json:"processing_delay_one_second_ns"`
	TQPressure                  string     `json:"tq_pressure"`
	TQPressureCause             string     `json:"tq_pressure_cause"`
	HeapAllocBytes              uint64     `json:"heap_alloc_bytes"`
	HeapInUseBytes              uint64     `json:"heap_in_use_bytes"`
	Goroutines                  int        `json:"goroutines"`
	GCCycles                    uint32     `json:"gc_cycles"`
	GCPauseTotalNS              uint64     `json:"gc_pause_total_ns"`
	LastGCPauseNS               uint64     `json:"last_gc_pause_ns"`
}

func newWatermarkStaleDiagnosticDocument(previous, current watermarkStaleStatusSample, evidence operations.WatermarkStallEvidence) watermarkStaleDiagnosticDocument {
	records := evidence.Records
	cutoff := current.SampledAt
	if !cutoff.IsZero() {
		kept := records[:0]
		for _, record := range records {
			if !record.CycleCompletedAt.After(cutoff) {
				kept = append(kept, record)
			}
		}
		records = kept
	}
	if len(records) > operations.WatermarkStallRingCapacity {
		records = records[len(records)-operations.WatermarkStallRingCapacity:]
	}
	cycles := make([]watermarkStaleDiagnosticCycle, len(records))
	for index, record := range records {
		cycles[index] = watermarkStaleDiagnosticCycleFromEvidence(record)
	}
	document := watermarkStaleDiagnosticDocument{
		Schema: watermarkStaleDiagnosticSchema, Trigger: watermarkStaleDiagnosticTrigger{
			PreviousBackendReady: previous.BackendReady, CurrentBackendReady: current.BackendReady,
			CurrentReason: string(current.Reason), RunMode: string(current.RunMode), Lifecycle: current.Lifecycle,
			PreviousSampledAt: previous.SampledAt.UTC(), CurrentSampledAt: current.SampledAt.UTC(),
			CurrentWatermarkLagNS: int64(nonnegativeDiagnosticDuration(current.WatermarkLag)), CurrentCausalTarget: current.CausalTarget,
		}, RingCapacity: operations.WatermarkStallRingCapacity, RecordCount: len(cycles), Cycles: cycles,
	}
	if evidence.Active != nil {
		document.ActiveCycle = watermarkStaleDiagnosticActiveCycleFromEvidence(*evidence.Active)
	}
	return document
}

func watermarkStaleDiagnosticActiveCycleFromEvidence(value operations.WatermarkStallActiveCycleEvidence) *watermarkStaleDiagnosticActiveCycle {
	result := &watermarkStaleDiagnosticActiveCycle{
		Sequence: value.Sequence, StartedAt: value.StartedAt.UTC(), PhaseStartedAt: value.PhaseStartedAt.UTC(), ObservedAt: value.ObservedAt.UTC(),
		ElapsedNS: int64(nonnegativeDiagnosticDuration(value.Elapsed)), PhaseElapsedNS: int64(nonnegativeDiagnosticDuration(value.PhaseElapsed)), Phase: string(value.Phase),
		EvaluationActive: value.EvaluationActive, EvaluationElapsedNS: int64(nonnegativeDiagnosticDuration(value.EvaluationElapsed)), EvaluationPhaseElapsedNS: int64(nonnegativeDiagnosticDuration(value.EvaluationPhaseElapsed)),
	}
	if value.EvaluationActive {
		target, started, phaseStarted := value.Evaluation.Target.UTC(), value.Evaluation.StartedAt.UTC(), value.Evaluation.PhaseStartedAt.UTC()
		result.EvaluationEngineSequence, result.EvaluationSource, result.EvaluationPhase = value.Evaluation.EngineSequence, string(value.Evaluation.Source), string(value.Evaluation.Phase)
		result.EvaluationTarget, result.EvaluationStartedAt, result.EvaluationPhaseStartedAt = &target, &started, &phaseStarted
	}
	return result
}

func watermarkStaleDiagnosticCycleFromEvidence(record operations.WatermarkStallCycleEvidence) watermarkStaleDiagnosticCycle {
	evaluationTarget := optionalDiagnosticTime(record.EvaluationTarget, record.EvaluationObserved)
	return watermarkStaleDiagnosticCycle{
		CycleStartedAt: record.CycleStartedAt.UTC(), CycleCompletedAt: record.CycleCompletedAt.UTC(),
		CycleDurationNS: int64(nonnegativeDiagnosticDuration(record.CycleDuration)), DiagnosticSampledAt: record.DiagnosticSampledAt.UTC(),
		CoverageFenceDisposition:    nonemptyDiagnosticCode(string(record.CoverageFenceDisposition), "not_observed"),
		LiveCoverageEnqueueNS:       int64(nonnegativeDiagnosticDuration(record.LiveCoverageEnqueue)),
		LiveCoverageCompletionNS:    int64(nonnegativeDiagnosticDuration(record.LiveCoverageCompletion)),
		CoverageFenceFinalizationNS: int64(nonnegativeDiagnosticDuration(record.CoverageFenceFinalization)), CoverageFenceTotalNS: int64(nonnegativeDiagnosticDuration(record.CoverageFenceTotal)), CoverageFenceValid: record.CoverageFenceValid,
		TimerPolicy: string(record.TimerPolicy), EvaluationObserved: record.EvaluationObserved, EvaluationSource: string(record.EvaluationSource), EvaluationTarget: evaluationTarget,
		EvaluationStageNS: int64(nonnegativeDiagnosticDuration(record.EvaluationStage)), EvaluationApplyNS: int64(nonnegativeDiagnosticDuration(record.EvaluationApply)), EvaluationPublicationNS: int64(nonnegativeDiagnosticDuration(record.EvaluationPublication)),
		WatermarkBefore: optionalDiagnosticTime(record.WatermarkBefore, record.WatermarkBeforePresent), WatermarkAfter: optionalDiagnosticTime(record.WatermarkAfter, record.WatermarkAfterPresent),
		CausalTargetBefore: optionalDiagnosticTime(record.CausalTargetBefore, record.CausalTargetBeforePresent), CausalTargetAfter: optionalDiagnosticTime(record.CausalTargetAfter, record.CausalTargetAfterPresent),
		WatermarkLagBeforeNS: int64(nonnegativeDiagnosticDuration(record.WatermarkLagBefore)), WatermarkLagAfterNS: int64(nonnegativeDiagnosticDuration(record.WatermarkLagAfter)),
		PublicationIDBefore: record.PublicationIDBefore, PublicationIDAfter: record.PublicationIDAfter, EngineSequenceBefore: record.EngineSequenceBefore, EngineSequenceAfter: record.EngineSequenceAfter,
		PublicationGeneratedBefore: optionalDiagnosticTime(record.PublicationGeneratedBefore, !record.PublicationGeneratedBefore.IsZero()), PublicationGeneratedAfter: optionalDiagnosticTime(record.PublicationGeneratedAfter, !record.PublicationGeneratedAfter.IsZero()),
		QueueCurrentFrames: record.QueueCurrentFrames, QueueHighFrames: record.QueueHighFrames, QueueCurrentBytes: record.QueueCurrentBytes, QueueHighBytes: record.QueueHighBytes,
		QueueOldestWaitingAgeNS: int64(nonnegativeDiagnosticDuration(record.QueueOldestWaitingAge)), ProcessingDelayNS: int64(nonnegativeDiagnosticDuration(record.ProcessingDelay)), ProcessingDelayOneSecondNS: int64(nonnegativeDiagnosticDuration(record.ProcessingDelayOneSecond)),
		TQPressure: string(record.TQPressure), TQPressureCause: string(record.TQPressureCause), HeapAllocBytes: record.HeapAllocBytes, HeapInUseBytes: record.HeapInUseBytes, Goroutines: record.Goroutines,
		GCCycles: record.GCCycles, GCPauseTotalNS: record.GCPauseTotalNS, LastGCPauseNS: record.LastGCPauseNS,
	}
}

func persistWatermarkStaleDiagnostic(directory string, document *watermarkStaleDiagnosticDocument) (string, error) {
	if directory == "" || document == nil {
		return "", errors.New("watermark-stale diagnostic persistence requires directory and document")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", errors.New("create watermark-stale diagnostic directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", errors.New("protect watermark-stale diagnostic directory")
	}
	at := document.Trigger.CurrentSampledAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	path := filepath.Join(directory, fmt.Sprintf("watermark-stale-%s.json", at.UTC().Format("20060102T150405.000000000Z")))
	file, err := os.CreateTemp(directory, ".watermark-stale-*.tmp")
	if err != nil {
		return "", errors.New("create watermark-stale diagnostic temporary file")
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return "", errors.New("protect watermark-stale diagnostic temporary file")
	}
	encodeErr := json.NewEncoder(file).Encode(document)
	syncErr := file.Sync()
	closeErr := file.Close()
	if encodeErr != nil || syncErr != nil || closeErr != nil {
		return "", errors.Join(errors.New("persist watermark-stale diagnostic"), encodeErr, syncErr, closeErr)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		return "", errors.New("publish watermark-stale diagnostic without overwrite")
	}
	if err := os.Remove(temporaryPath); err != nil {
		return "", errors.New("remove published watermark-stale diagnostic temporary file")
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return "", errors.New("open watermark-stale diagnostic directory for sync")
	}
	directorySyncErr := directoryFile.Sync()
	directoryCloseErr := directoryFile.Close()
	if directorySyncErr != nil || directoryCloseErr != nil {
		return "", errors.Join(errors.New("sync watermark-stale diagnostic directory"), directorySyncErr, directoryCloseErr)
	}
	return path, nil
}

func cloneDiagnosticTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}

func optionalDiagnosticTime(value time.Time, present bool) *time.Time {
	if !present || value.IsZero() {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}

func nonnegativeDiagnosticDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}

func nonemptyDiagnosticCode(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
