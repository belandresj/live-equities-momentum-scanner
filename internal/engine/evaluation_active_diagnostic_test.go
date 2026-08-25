package engine

import (
	"context"
	"testing"
	"time"
)

func TestActiveAggregateEvaluationDiagnosticTracksStageApplyPublication(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 21, 43, 0, time.UTC)
	e := &Engine{clock: func() time.Time { return now }, evaluationTimingClock: func() time.Time { return now }}
	node := &queueNode{kind: inputLiveCoverageFence, engineSequence: 42}
	target := now.Add(-4 * time.Second)

	e.startActiveAggregateEvaluationLocked(node, target, AggregateEvaluationPhaseStage)
	assertActiveEvaluationPhase(t, e, AggregateEvaluationPhaseStage, AggregateEvaluationLiveCoverageFence, 42, target, now)
	now = now.Add(700 * time.Millisecond)
	e.advanceActiveAggregateEvaluationLocked(42, AggregateEvaluationPhaseApply)
	assertActiveEvaluationPhase(t, e, AggregateEvaluationPhaseApply, AggregateEvaluationLiveCoverageFence, 42, target, now)
	now = now.Add(30 * time.Millisecond)
	e.advanceActiveAggregateEvaluationLocked(42, AggregateEvaluationPhasePublication)
	assertActiveEvaluationPhase(t, e, AggregateEvaluationPhasePublication, AggregateEvaluationLiveCoverageFence, 42, target, now)

	// A foreign completion cannot erase the active owner; the matching
	// transition completion clears it exactly once.
	e.clearActiveAggregateEvaluation(41)
	if _, ok := e.ObserveActiveAggregateEvaluation(); !ok {
		t.Fatal("foreign sequence cleared active evaluation")
	}
	e.clearActiveAggregateEvaluation(42)
	if value, ok := e.ObserveActiveAggregateEvaluation(); ok {
		t.Fatalf("active evaluation remained after matching completion: %+v", value)
	}
}

func TestActiveAggregateEvaluationDiagnosticCoversProductionMaintenanceScan(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
	now := binding.SessionStart().Add(10 * time.Minute)
	e := aggregateEngine(t, binding, &now)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.state.lifecycle = lifecycleLive
	e.state.committedT = immutableTime(now.Add(-time.Second))
	e.state.hydration.supportedThrough = immutableTime(now)
	e.mu.Unlock()

	entered, release := make(chan struct{}), make(chan struct{})
	e.beforeActiveAggregateEvaluationPhase = func(phase AggregateEvaluationPhase) {
		if phase == AggregateEvaluationPhaseMaintenance {
			close(entered)
			<-release
		}
	}
	admission, completion := e.AdmitMaintenanceTimer(context.Background())
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("maintenance admission=%s completion=%v", admission, completion)
	}
	<-entered
	active, ok := e.ObserveActiveAggregateEvaluation()
	if !ok || active.Phase != AggregateEvaluationPhaseMaintenance || active.Source != AggregateEvaluationTimer {
		t.Fatalf("held production maintenance active=%+v ok=%t", active, ok)
	}
	close(release)
	if disposition := awaitTimerDisposition(t, completion); disposition.Code != DispositionTimerApplied {
		t.Fatalf("maintenance disposition=%+v", disposition)
	}
	if active, ok := e.ObserveActiveAggregateEvaluation(); ok {
		t.Fatalf("completed maintenance remained active: %+v", active)
	}
}

func TestActiveAggregateEvaluationDiagnosticCoversProductionTrustCorrection(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	target := binding.SessionStart().Add(60 * time.Second)
	now := target
	e := aggregateEngine(t, binding, &now)
	defer closeAndWait(t, e)

	e.mu.Lock()
	symbol := &e.state.binding.symbols[0]
	installEvaluatorMarkOnSymbol(symbol, target, 12, qualificationProvisional)
	if !installExactCoverage(symbol.aggregates, e.state.binding, e.state.binding.sessionStart, target, nil) {
		e.mu.Unlock()
		t.Fatal("trust-correction coverage setup failed")
	}
	e.state.lifecycle, e.state.committedT, e.state.latestTarget = lifecycleLive, immutableTime(target), immutableTime(target)
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.state.aggregateAckPosition, e.state.aggregateAckReceivedAt = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, target
	e.state.clockMonotonic, e.state.aggregateProjectionPending = true, true
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(target)
	e.mu.Unlock()

	_, initial := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, initial); got.Code != DispositionTimerApplied {
		t.Fatalf("initial publication=%+v", got)
	}

	entered := make(chan AggregateEvaluationPhase)
	release := make(chan struct{})
	e.beforeActiveAggregateEvaluationPhase = func(phase AggregateEvaluationPhase) {
		entered <- phase
		<-release
	}
	correction := liveAggregate(binding, "AAA", target.Add(-time.Second), 1, 2)
	correction.Values.Open, correction.Values.High, correction.Values.Low, correction.Values.Close, correction.Values.VWAP = 12, 12, 12, 12, 12
	correction.Values.Volume, correction.Values.AverageTradeSize = 1, 0
	admission, completion := e.AdmitAggregate(context.Background(), correction)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("trust-correction admission=%s completion=%v", admission, completion)
	}
	result := make(chan AggregateDisposition, 1)
	go func() { result <- <-completion }()

	for _, want := range []AggregateEvaluationPhase{AggregateEvaluationPhaseStage, AggregateEvaluationPhaseApply, AggregateEvaluationPhasePublication} {
		if got := <-entered; got != want {
			t.Fatalf("production trust-correction phase=%s want=%s", got, want)
		}
		active, ok := e.ObserveActiveAggregateEvaluation()
		if !ok || active.Phase != want || active.Source != AggregateEvaluationTrustCorrection || !active.Target.Equal(target) {
			t.Fatalf("production trust-correction active=%+v ok=%t want_phase=%s", active, ok, want)
		}
		release <- struct{}{}
	}
	if got := <-result; got.Code != DispositionAggregateRevised {
		t.Fatalf("trust-correction result=%+v", got)
	}
	if active, ok := e.ObserveActiveAggregateEvaluation(); ok {
		t.Fatalf("completed trust correction remained active: %+v", active)
	}
}

func TestActiveAggregateEvaluationDiagnosticCoversProductionSelectedTrustClosure(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	target := binding.SessionStart().Add(330 * time.Second)
	now := target
	e := aggregateEngine(t, binding, &now)
	defer closeAndWait(t, e)

	historical := historicalAggregate(binding, "AAA", target.Add(-3*time.Second), 1)
	historical.Values.Open, historical.Values.High, historical.Values.Low, historical.Values.Close, historical.Values.VWAP = 12, 12, 12, 12, 12
	historical.Values.Volume = 7
	applyHistorical(t, e, historical, proofFor(binding, historical, e.state.binding.sessionStart, target), DispositionAggregateInserted, ReasonNone)
	e.mu.Lock()
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = lifecycleLive, 1, true, true
	e.state.aggregateAckPosition, e.state.aggregateAckReceivedAt = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, target
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 100, 1
	e.state.hydration.supportedThrough = immutableTime(target)
	state := ensureAggregateState(&e.state.binding.symbols[0])
	if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, target, nil) {
		e.mu.Unlock()
		t.Fatal("selected-trust-closure coverage setup failed")
	}
	e.mu.Unlock()
	for frame, at := range []time.Time{target.Add(-2 * time.Second), target.Add(-time.Second)} {
		input := liveAggregate(binding, "AAA", at, 1, uint64(frame+2))
		input.Values.Open, input.Values.High, input.Values.Low, input.Values.Close, input.Values.VWAP = 12, 12, 12, 12, 12
		input.Values.Volume = 10
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
			t.Fatalf("selected-trust-closure live insert=%+v", got)
		}
	}
	e.mu.Lock()
	state = e.state.binding.symbols[0].aggregates
	proofEnd := target.Add(-time.Minute)
	state.qualification = &qualificationState{finalized: true, finalProofEnd: proofEnd, finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{},
		accountedThrough: target, result: qualificationResult{at: target, status: qualificationFinalized, finalProofEnd: proofEnd}}
	e.mu.Unlock()
	_, published := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, published); got.Code != DispositionTimerApplied || len(e.observePublication().aggregateEvaluation.rows) != 1 {
		t.Fatalf("selected-trust-closure initial publication=%+v", got)
	}

	e.mu.Lock()
	e.state.lifecycle = lifecycleHydrating
	e.mu.Unlock()
	plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	var token HydrationRequestToken
	for _, request := range plan.Plan.Requests() {
		if request.Symbol() == "AAA" {
			token = request
			break
		}
	}
	values := AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 9, VWAP: 12, AverageTradeSize: 1, ATSProvenance: ATSRESTFloorVolumeOverTrades}
	conflictRow, _ := NewHydrationRow("AAA", target.Add(-3*time.Second), target.Add(-2*time.Second), values)
	conflictChunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{conflictRow})
	e.mu.Lock()
	semanticRevisionBefore, semanticCycleBefore := e.state.trustCorrectionRevision, e.state.lastTrustCorrectionCycle
	e.mu.Unlock()

	entered := make(chan AggregateEvaluationPhase)
	release := make(chan struct{})
	e.beforeActiveAggregateEvaluationPhase = func(phase AggregateEvaluationPhase) {
		entered <- phase
		<-release
	}
	admission, completion := e.AdmitHydrationChunk(context.Background(), conflictChunk)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("selected-trust-closure admission=%s completion=%v", admission, completion)
	}
	for _, want := range []AggregateEvaluationPhase{AggregateEvaluationPhaseStage, AggregateEvaluationPhaseApply, AggregateEvaluationPhasePublication} {
		if got := <-entered; got != want {
			t.Fatalf("selected-trust-closure phase=%s want=%s", got, want)
		}
		active, ok := e.ObserveActiveAggregateEvaluation()
		if !ok || active.Phase != want || active.Source != AggregateEvaluationTrustCorrection || !active.Target.Equal(target) {
			t.Fatalf("selected-trust-closure active=%+v ok=%t want_phase=%s", active, ok, want)
		}
		release <- struct{}{}
	}
	if got := <-completion; got.Code != DispositionHydrationChunkApplied || got.Rows.ConflictOrWithdrawal != 1 {
		t.Fatalf("selected-trust-closure completion=%+v", got)
	}
	e.mu.Lock()
	semanticRevisionAfter, semanticCycleAfter := e.state.trustCorrectionRevision, e.state.lastTrustCorrectionCycle
	e.mu.Unlock()
	if semanticRevisionAfter != semanticRevisionBefore || semanticCycleAfter != semanticCycleBefore {
		t.Fatalf("diagnostic timing changed trust-correction coalescing identity: revision=%d/%d cycle=%+v/%+v", semanticRevisionBefore, semanticRevisionAfter, semanticCycleBefore, semanticCycleAfter)
	}
	if active, ok := e.ObserveActiveAggregateEvaluation(); ok {
		t.Fatalf("completed selected trust closure remained active: %+v", active)
	}
}

func assertActiveEvaluationPhase(t *testing.T, e *Engine, phase AggregateEvaluationPhase, source AggregateEvaluationSource, sequence uint64, target, phaseStartedAt time.Time) {
	t.Helper()
	value, ok := e.ObserveActiveAggregateEvaluation()
	if !ok || value.Phase != phase || value.Source != source || value.EngineSequence != sequence || !value.Target.Equal(target) || !value.PhaseStartedAt.Equal(phaseStartedAt) {
		t.Fatalf("active evaluation phase=%s value=%+v ok=%t", phase, value, ok)
	}
}
