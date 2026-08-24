package operations

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

func TestWatermarkStallRingCapacityWrapAndDetachedChronology(t *testing.T) {
	var ring watermarkStallRing
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	for index := 0; index < WatermarkStallRingCapacity+5; index++ {
		ring.add(WatermarkStallCycleEvidence{CycleStartedAt: base.Add(time.Duration(index) * time.Second), EngineSequenceBefore: uint64(index)})
	}
	snapshot := ring.snapshot()
	if snapshot.Capacity != WatermarkStallRingCapacity || len(snapshot.Records) != WatermarkStallRingCapacity || cap(snapshot.Records) > WatermarkStallRingCapacity {
		t.Fatalf("ring bound=%+v cap=%d", snapshot, cap(snapshot.Records))
	}
	for index, record := range snapshot.Records {
		want := index + 5
		if !record.CycleStartedAt.Equal(base.Add(time.Duration(want)*time.Second)) || record.EngineSequenceBefore != uint64(want) {
			t.Fatalf("chronology index=%d record=%+v", index, record)
		}
	}
	snapshot.Records[0].EngineSequenceBefore = 999
	if got := ring.snapshot().Records[0].EngineSequenceBefore; got != 5 {
		t.Fatalf("detached snapshot mutation reached ring: %d", got)
	}
	typeOfRecord := reflect.TypeOf(WatermarkStallCycleEvidence{})
	for index := 0; index < typeOfRecord.NumField(); index++ {
		switch typeOfRecord.Field(index).Type.Kind() {
		case reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer:
			t.Fatalf("cycle record retains unbounded/reference field %s", typeOfRecord.Field(index).Name)
		}
	}
}

func TestWatermarkStallRingConcurrentSnapshotsNeverExposePartialRecords(t *testing.T) {
	var ring watermarkStallRing
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	var writers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		writers.Add(1)
		go func(worker int) {
			defer writers.Done()
			for index := 0; index < 500; index++ {
				sequence := uint64(worker*1000 + index + 1)
				ring.add(WatermarkStallCycleEvidence{
					CycleStartedAt: base.Add(time.Duration(sequence) * time.Millisecond), CycleCompletedAt: base.Add(time.Duration(sequence+1) * time.Millisecond),
					EngineSequenceBefore: sequence, EngineSequenceAfter: sequence + 1,
					PublicationIDBefore: sequence, PublicationIDAfter: sequence + 1,
				})
			}
		}(worker)
	}
	for index := 0; index < 500; index++ {
		snapshot := ring.snapshot()
		if len(snapshot.Records) > WatermarkStallRingCapacity {
			t.Fatalf("ring exceeded capacity: %d", len(snapshot.Records))
		}
		for _, record := range snapshot.Records {
			if record.EngineSequenceAfter != record.EngineSequenceBefore+1 || record.PublicationIDAfter != record.PublicationIDBefore+1 || record.CycleCompletedAt.Before(record.CycleStartedAt) {
				t.Fatalf("partial cycle record=%+v", record)
			}
		}
	}
	writers.Wait()
}

func TestWatermarkStallActiveCycleDistinguishesOrderedWaitPhases(t *testing.T) {
	base := time.Date(2026, 8, 24, 17, 21, 43, 0, time.UTC)
	now := base
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 8, RequiredReserve: 1,
		EvaluationDelay: &delay, RecoveryBackoffInitial: time.Second, RecoveryBackoffMaximum: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	runtime := &Runtime{engine: owner, clock: func() time.Time { return now }}
	active := runtime.beginWatermarkStallCycle(base)

	for _, step := range []struct {
		phase WatermarkStallCyclePhase
		at    time.Time
	}{
		{WatermarkStallCycleLiveCoverageEnqueue, base.Add(100 * time.Millisecond)},
		{WatermarkStallCycleLiveCoverageCompletion, base.Add(300 * time.Millisecond)},
		{WatermarkStallCycleTimerAdmission, base.Add(600 * time.Millisecond)},
		{WatermarkStallCycleTimerCompletion, base.Add(900 * time.Millisecond)},
	} {
		now = step.at
		runtime.advanceWatermarkStallCycle(active.Sequence, step.phase)
		now = now.Add(250 * time.Millisecond)
		evidence := runtime.ObserveWatermarkStallEvidence()
		if evidence.Active == nil || evidence.Active.Phase != step.phase || evidence.Active.PhaseElapsed != 250*time.Millisecond || evidence.Active.Elapsed != now.Sub(base) || evidence.Active.EvaluationActive {
			t.Fatalf("phase=%s evidence=%+v", step.phase, evidence.Active)
		}
	}
	runtime.clearWatermarkStallCycle(active.Sequence)
	if evidence := runtime.ObserveWatermarkStallEvidence(); evidence.Active != nil {
		t.Fatalf("completed cycle remained active: %+v", evidence.Active)
	}
}

func TestWatermarkStaleTransitionTimestampsActiveObservationAfterPreemption(t *testing.T) {
	base := time.Date(2026, 8, 24, 17, 21, 43, 0, time.UTC)
	now := base
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 8, RequiredReserve: 1,
		EvaluationDelay: &delay, RecoveryBackoffInitial: time.Second, RecoveryBackoffMaximum: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	runtime := &Runtime{engine: owner, clock: func() time.Time { return now }}
	active := runtime.beginWatermarkStallCycle(base)
	ready := func(at time.Time, backendReady bool) Status {
		watermark, target := at.Add(-5*time.Second), at.Add(-4*time.Second)
		status := Status{ProcessLive: true, BackendReady: backendReady, RankingCurrent: true, Lifecycle: "live", SampledAt: at, Watermark: &watermark, CausalTarget: &target}
		if !backendReady {
			status.Reason, status.WatermarkLag = ReasonWatermarkStale, 3*time.Second
		}
		return status
	}
	runtime.recordReadinessObservation(ready(base, true))

	// The stale snapshot was sampled at t+1, then its goroutine was delayed.
	// The active phase changed at t+2 before the readiness observation CAS.
	// The retained phase must carry its own t+3 observation timestamp rather
	// than borrowing the older snapshot timestamp and clipping durations.
	now = base.Add(2 * time.Second)
	runtime.advanceWatermarkStallCycle(active.Sequence, WatermarkStallCycleLiveCoverageCompletion)
	now = base.Add(3 * time.Second)
	runtime.recordReadinessObservation(ready(base.Add(time.Second), false))
	transition, ok := runtime.ObserveWatermarkStaleTransition()
	if !ok || !transition.ActiveObserved || transition.Active.Phase != WatermarkStallCycleLiveCoverageCompletion ||
		!transition.Active.ObservedAt.Equal(now) || transition.Active.ObservedAt.Before(transition.Active.PhaseStartedAt) || transition.Active.PhaseElapsed != time.Second {
		t.Fatalf("preempted transition=%+v ok=%t", transition, ok)
	}
}

func TestWatermarkStallCycleEvidenceDistinguishesFailurePlanes(t *testing.T) {
	binding := operationsBinding(t)
	runtime := &Runtime{binding: binding, config: DefaultConfig()}
	base := binding.SessionStart().Add(10 * time.Minute)
	target := base.Add(-runtime.config.EvaluationDelay).Truncate(time.Second)
	state := func(sequence, publication uint64, watermark time.Time, pressure engine.TQPressureMode, cause engine.TQPressureCause) engine.EvaluationCycleState {
		return engine.EvaluationCycleState{PublicationID: publication, LastEngineSequence: sequence, RunMode: engine.RunModeLive, Lifecycle: "live", Watermark: watermark, WatermarkPresent: !watermark.IsZero(), GeneratedAt: base, TQPressure: pressure, TQPressureCause: cause}
	}
	metrics := func(oldest, delay time.Duration) Metrics {
		return Metrics{SampledAt: base, QueueCurrentFrames: 2, QueueHighFrames: 8, QueueCurrentBytes: 2048, QueueHighBytes: 8192,
			LiveQueue: massive.LiveQueueAccounting{FramesQueued: 2, QueuedBytes: 2048, OldestWaitingFrameAge: oldest}, MaxProcessingDelay: delay,
			MaxProcessingDelayOneSecond: delay, HeapAllocBytes: 100, HeapInUseBytes: 200, Goroutines: 3, GCCycles: 7, GCPauseTotalNS: 11, LastGCPauseNS: 2}
	}
	cycle := func(index int, policy WatermarkStallTimerPolicy, fence engine.DispositionCode, fenceTotal, stage, delay time.Duration, pressure engine.TQPressureMode, cause engine.TQPressureCause) {
		beforeSequence, afterSequence := uint64(index*2+1), uint64(index*2+2)
		beforeWatermark, afterWatermark := target.Add(-2*time.Second), target.Add(-time.Second)
		timingSequence := afterSequence
		fenceSequence := uint64(0)
		evaluationSource := engine.AggregateEvaluationTimer
		if policy == WatermarkStallTimerMaintenanceOnly {
			timingSequence = beforeSequence
			fenceSequence = beforeSequence
			evaluationSource = engine.AggregateEvaluationLiveCoverageFence
		}
		evaluationTiming := engine.EvaluationTimingView{EngineSequence: timingSequence, Source: evaluationSource, Target: target, Stage: stage, Apply: 10 * time.Millisecond, Publication: 2 * time.Millisecond}
		fenceResult := engine.LiveCoverageFenceDisposition{EngineSequence: fenceSequence, Code: fence}
		observedAfterTimer := evaluationTiming
		if policy == WatermarkStallTimerMaintenanceOnly {
			fenceResult.EvaluationTiming = evaluationTiming
			observedAfterTimer = engine.EvaluationTimingView{EngineSequence: afterSequence + 100, Source: engine.AggregateEvaluationIngressFence, Target: target, Stage: 9 * time.Second}
		}
		runtime.recordWatermarkStallCycle(base.Add(time.Duration(index)*time.Second), base.Add(time.Duration(index+1)*time.Second), 50*time.Millisecond,
			fenceResult, policy, state(beforeSequence, uint64(index*2+1), beforeWatermark, pressure, cause), state(afterSequence, uint64(index*2+2), afterWatermark, pressure, cause),
			engine.FenceTimingView{EngineSequence: fenceSequence, CoverageFinalization: fenceTotal, Total: fenceTotal + stage, Valid: fenceSequence != 0},
			engine.TimerDisposition{EngineSequence: afterSequence}, observedAfterTimer, metrics(delay, delay), 3*time.Millisecond, fenceTotal)
	}
	cycle(0, WatermarkStallTimerEvaluationFallback, engine.DispositionLiveCoverageFenceRejected, 0, 900*time.Millisecond, 0, "", "")
	cycle(1, WatermarkStallTimerMaintenanceOnly, engine.DispositionLiveCoverageFenceApplied, 1500*time.Millisecond, 40*time.Millisecond, 0, "", "")
	cycle(2, WatermarkStallTimerEvaluationFallback, engine.DispositionLiveCoverageFenceFenced, 0, 40*time.Millisecond, 1200*time.Millisecond, engine.TQPressureDegraded, engine.TQPressureCauseOldestWaitingFrame)
	cycle(3, WatermarkStallTimerMaintenanceOnly, engine.DispositionLiveCoverageFenceApplied, 10*time.Millisecond, 20*time.Millisecond, 5*time.Millisecond, engine.TQPressureNormal, engine.TQPressureCauseNone)

	got := runtime.ObserveWatermarkStallEvidence()
	if len(got.Records) != 4 {
		t.Fatalf("cycle count=%d", len(got.Records))
	}
	if got.Records[0].EvaluationStage != 900*time.Millisecond || got.Records[0].TimerPolicy != WatermarkStallTimerEvaluationFallback {
		t.Fatalf("slow evaluator evidence=%+v", got.Records[0])
	}
	if got.Records[1].CoverageFenceFinalization != 1500*time.Millisecond || got.Records[1].EvaluationStage != 40*time.Millisecond || got.Records[1].TimerPolicy != WatermarkStallTimerMaintenanceOnly {
		t.Fatalf("late fence evidence=%+v", got.Records[1])
	}
	if !got.Records[1].EvaluationObserved || got.Records[1].EvaluationSource != engine.AggregateEvaluationLiveCoverageFence || got.Records[1].LiveCoverageEnqueue != 3*time.Millisecond || got.Records[1].LiveCoverageCompletion != 1500*time.Millisecond {
		t.Fatalf("live-coverage correlation=%+v", got.Records[1])
	}
	if got.Records[2].QueueOldestWaitingAge != 1200*time.Millisecond || got.Records[2].ProcessingDelayOneSecond != 1200*time.Millisecond || got.Records[2].TQPressureCause != engine.TQPressureCauseOldestWaitingFrame {
		t.Fatalf("queue/delivery evidence=%+v", got.Records[2])
	}
	if got.Records[3].CoverageFenceFinalization != 10*time.Millisecond || got.Records[3].QueueOldestWaitingAge != 5*time.Millisecond || got.Records[3].TQPressure != engine.TQPressureNormal {
		t.Fatalf("normal evidence=%+v", got.Records[3])
	}
	for _, record := range got.Records {
		if record.CausalTargetAfter.IsZero() || record.WatermarkLagAfter <= 0 || record.PublicationIDAfter <= record.PublicationIDBefore || record.EngineSequenceAfter <= record.EngineSequenceBefore {
			t.Fatalf("missing causal/progress evidence=%+v", record)
		}
	}
}

func TestWatermarkStaleTransitionLatchRetainsShortCrossingAcrossSamplerPhases(t *testing.T) {
	runtime := &Runtime{}
	base := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	ready := func(at time.Time) Status {
		watermark, target := at.Add(-5*time.Second), at.Add(-4*time.Second)
		return Status{ProcessLive: true, BackendReady: true, RankingCurrent: true, Lifecycle: "live", SampledAt: at, Watermark: &watermark, CausalTarget: &target}
	}
	stale := ready(base.Add(1250 * time.Millisecond))
	stale.BackendReady = false
	stale.Reason = ReasonWatermarkStale
	stale.WatermarkLag = 3 * time.Second

	// Former scanner samples at 1s and 2s are both ready. Model the harder
	// arrival order: the launcher seals stale at 1.25s, is preempted, the scanner
	// records recovered-ready at 1.32s, and only then does the launcher finish
	// recording the status it will return. Atomic record order must retain that
	// user-visible ready -> stale crossing rather than discard the older sample.
	runtime.recordReadinessObservation(ready(base.Add(time.Second)))
	runtime.recordReadinessObservation(ready(base.Add(1320 * time.Millisecond)))
	runtime.recordReadinessObservation(stale)
	transition, ok := runtime.ObserveWatermarkStaleTransition()
	if !ok || !transition.Previous.BackendReady || transition.Current.BackendReady || transition.Current.Reason != ReasonWatermarkStale ||
		!transition.Previous.SampledAt.Equal(base.Add(1320*time.Millisecond)) || !transition.Current.SampledAt.Equal(stale.SampledAt) {
		t.Fatalf("retained transition ok=%t value=%+v", ok, transition)
	}
	// Returned pointers are detached, and no later observation can replace the
	// first retained crossing.
	*transition.Current.CausalTarget = time.Time{}
	runtime.recordReadinessObservation(ready(base.Add(2 * time.Second)))
	again, ok := runtime.ObserveWatermarkStaleTransition()
	if !ok || again.Current.CausalTarget == nil || again.Current.CausalTarget.IsZero() || !again.Current.SampledAt.Equal(stale.SampledAt) {
		t.Fatalf("transition alias/order containment failed ok=%t value=%+v", ok, again)
	}
}
