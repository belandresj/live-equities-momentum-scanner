package engine

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestPLBRA2Hydration is the engine/lifecycle portion of P-LBR-A2-HYDRATION.
// The Massive worker and real fence conversion are composed by the matching
// proof in internal/massive.
func TestPLBRA2Hydration(t *testing.T) {
	binding := testBinding(t)
	start, end := binding.SessionStart(), binding.SessionEnd()

	t.Run("schedule binding live mode and one-worker legality", func(t *testing.T) {
		now := start.Add(-time.Second)
		before := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleAwaitingSession)
		if view := before.ObserveOperational(); view.RunMode != RunModeLive || view.Lifecycle != string(lifecycleAwaitingSession) || view.Hydration.Active || view.CurrentMarketClaim {
			t.Fatalf("pre-session state = %+v", view)
		}
		plan := admitHydrationPlan(t, before, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if plan.Code != DispositionHydrationRejected || plan.Reason != ReasonLifecycle || before.state.hydration.generation.active {
			t.Fatalf("pre-session fabricated hydration = %+v", plan)
		}
		now = start
		admission, timer := before.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied || before.state.lifecycle != lifecycleHydrating {
			t.Fatal("acknowledged preconnect did not wait for exact S")
		}
		closeAndWait(t, before)

		now = start
		awaiting := aggregateEngine(t, binding, RunModeLive, &now)
		if awaiting.state.lifecycle != lifecycleAwaitingAggregateAck || awaiting.state.hydration.generation.active || awaiting.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("in-session initialization = %+v", awaiting.ObserveOperational())
		}
		closeAndWait(t, awaiting)

		now = end
		ended := aggregateEngine(t, binding, RunModeLive, &now)
		if ended.state.lifecycle != lifecycleEnded || ended.state.hydration.generation.active || ended.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("at-E initialization = %+v", ended.ObserveOperational())
		}
		if result, completion := ended.AdmitHydrationPlan(context.Background(), HydrationPlanInput{SchemaVersion: HydrationPlanSchemaV1,
			BindingIdentity: binding.Identity(), Purpose: HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: generousHydrationBudgets()}); result != AdmissionNotAdmittedClosed || completion != nil {
			t.Fatalf("ended engine admitted hydration = %s", result)
		}

		invalid := testEngine(t, RunModeLive, start, 4, 1)
		result, completion := invalid.AdmitBinding(context.Background(), BindingInstall{SchemaVersion: BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: reference.Binding{}})
		if result != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionBindingInvalid || invalid.state.binding != nil ||
			invalid.state.hydration.generation.active || invalid.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("invalid binding created partial initialization = %s %+v", result, invalid.ObserveOperational())
		}
		closeAndWait(t, invalid)

		now = start.Add(2 * time.Second)
		exhausted := aggregateEngine(t, binding, RunModeLive, &now)
		admitConnectionControl(t, exhausted, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		admitConnectionControl(t, exhausted, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 0, ControlFailed))
		exhaustion := RecoveryExhaustionInput{SchemaVersion: RecoveryExhaustionSchemaV1, BindingIdentity: binding.Identity(), Attempts: 1}
		exhaustionAdmission, exhaustionCompletion := exhausted.AdmitRecoveryExhaustion(context.Background(), exhaustion)
		exhaustionResult := <-exhaustionCompletion
		if exhaustionAdmission != AdmissionAdmitted || exhaustionResult.Code != DispositionRecoveryExhausted || exhausted.state.lifecycle != lifecycleSuppressed ||
			exhausted.state.latestTransition.Reason != lifecycleReasonRecoveryExhausted || exhausted.state.hydration.generation.active || exhausted.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("in-session initialization exhaustion = %+v view=%+v", exhaustionResult, exhausted.ObserveOperational())
		}
		closeAndWait(t, exhausted)
	})

	t.Run("five terminal bins malformed row and epoch fencing", func(t *testing.T) {
		population := hydrationPopulationBinding(t, []string{"AAA", "BBB", "CCC", "DDD", "EEE"})
		now := population.SessionStart().Add(5 * time.Second)
		e := acknowledgedHydrationEngine(t, population, &now, now, lifecycleHydrating)
		plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		requests := plan.Plan.Requests()
		if plan.Code != DispositionHydrationPlanApplied || len(requests) != 5 || plan.Accounting != (HydrationAccounting{Planned: 5, Open: 5}) {
			t.Fatalf("population plan = %+v requests=%d", plan, len(requests))
		}
		bySymbol := make(map[string]HydrationRequestToken, len(requests))
		for _, token := range requests {
			bySymbol[token.Symbol()] = token
		}
		valueToken := bySymbol["AAA"]
		row := hydrationRow(t, "AAA", population.SessionStart(), 10)
		chunk, _ := NewHydrationChunkInput(valueToken, valueToken.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		if got := admitHydrationChunk(t, e, chunk); got.Rows != (HydrationRowAccounting{Consumed: 1, Inserted: 1}) {
			t.Fatalf("value row = %+v", got)
		}
		value, _ := NewHydrationTerminalInput(valueToken, valueToken.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		admitHydrationTerminal(t, e, value)
		emptyToken := bySymbol["BBB"]
		empty, _ := NewHydrationTerminalInput(emptyToken, emptyToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		admitHydrationTerminal(t, e, empty)
		failedToken := bySymbol["CCC"]
		failed, _ := NewHydrationTerminalInput(failedToken, failedToken.ResultID(), HydrationFailed, HydrationReasonRequestConstruction, 0, 0, 0, 0, 0, 0)
		admitHydrationTerminal(t, e, failed)
		canceledToken := bySymbol["DDD"]
		canceled, _ := NewHydrationTerminalInput(canceledToken, canceledToken.ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
		admitHydrationTerminal(t, e, canceled)
		admission, canceledCompletion := e.admitHydrationCancelForProof(context.Background(), population.Identity(), plan.Plan.Generation(), true)
		got := awaitHydrationDisposition(t, canceledCompletion)
		want := HydrationAccounting{Planned: 5, CompletedValue: 1, CompletedEmpty: 1, Failed: 1, Canceled: 1, Fenced: 1}
		if admission != AdmissionAdmitted || got.Code != DispositionHydrationTerminalApplied || got.Accounting != want || !got.Accounting.reconciles() ||
			e.state.hydration.generation.rowAccounting != (HydrationRowAccounting{Consumed: 1, Inserted: 1}) {
			t.Fatalf("five-bin reconciliation = admission:%s disposition:%+v want:%+v", admission, got, want)
		}
		assertCanonicalClose(t, e, "AAA", population.SessionStart(), 10)
		lateToken := bySymbol["EEE"]
		late, _ := NewHydrationTerminalInput(lateToken, lateToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if lateResult := admitHydrationTerminal(t, e, late); lateResult.Code != DispositionHydrationFenced {
			t.Fatalf("fenced request accepted late terminal = %+v", lateResult)
		}
		closeAndWait(t, e)

		now = start.Add(3 * time.Second)
		malformed, malformedToken := plannedHydrationEngine(t, binding, &now)
		badRow := HydrationRow{symbol: "AAA", windowStart: start, windowEnd: start.Add(time.Second), values: AggregateValues{
			Open: 10, High: 10, Low: 10, Close: 10, Volume: -1, VWAP: 10, AverageTradeSize: 1, ATSProvenance: ATSRESTFloorVolumeOverTrades,
		}}
		badChunk, _ := NewHydrationChunkInput(malformedToken, malformedToken.ResultID(), 0, 1, 0, 1, []HydrationRow{badRow})
		badResult := admitHydrationChunk(t, malformed, badChunk)
		state := malformed.state.binding.symbols[malformed.state.binding.index["AAA"]].aggregates
		if badResult.Code != DispositionHydrationIntegrity || badResult.Reason != ReasonHydrationChunkSequence || malformed.state.lifecycle != lifecycleSuppressed ||
			state != nil && (len(state.tail) != 0 || state.prefix.printCount != 0) {
			t.Fatalf("malformed row reached canonical success = %+v state=%+v", badResult, state)
		}
		closeAndWait(t, malformed)

		now = start.Add(3 * time.Second)
		superseded, oldToken := plannedHydrationEngine(t, binding, &now)
		oldRow := hydrationRow(t, "AAA", start, 10)
		oldChunk, _ := NewHydrationChunkInput(oldToken, oldToken.ResultID(), 0, 1, 0, 1, []HydrationRow{oldRow})
		if accepted := admitHydrationChunk(t, superseded, oldChunk); accepted.Rows.Inserted != 1 {
			t.Fatalf("pre-replacement row = %+v", accepted)
		}
		now = start.Add(4 * time.Second)
		admitConnectionControl(t, superseded, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
		if superseded.state.hydration.generation.active || superseded.state.hydration.generation.accounting.Canceled != 1 {
			t.Fatalf("epoch replacement did not terminalize old work: %+v", superseded.state.hydration.generation)
		}
		assertCanonicalClose(t, superseded, "AAA", start, 10)
		ackRecoveryEpoch(t, superseded, binding, 2, 3, 4, start.Add(5*time.Second), &now)
		newPlan := admitHydrationPlan(t, superseded, HydrationFreshBootstrap, 2, generousHydrationBudgets())
		if newPlan.Code != DispositionHydrationPlanApplied || newPlan.Plan.Generation() == oldToken.Generation() {
			t.Fatalf("replacement generation = %+v", newPlan)
		}
		oldTerminal, _ := NewHydrationTerminalInput(oldToken, oldToken.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		if stale := admitHydrationTerminal(t, superseded, oldTerminal); stale.Code != DispositionHydrationFenced {
			t.Fatalf("old terminal crossed generation = %+v", stale)
		}
		if stale := admitHydrationChunk(t, superseded, oldChunk); stale.Code != DispositionHydrationFenced {
			t.Fatalf("old row crossed generation = %+v", stale)
		}
		assertCanonicalClose(t, superseded, "AAA", start, 10)
		closeAndWait(t, superseded)
	})

	t.Run("ordered fence evaluator reset and post-live gap recovery", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		empty, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		terminal := admitHydrationTerminal(t, e, empty)
		if terminal.FenceCommand.CommandToken() == 0 || e.ObserveOperational().CurrentMarketClaim || e.state.lifecycle != lifecycleHydrating {
			t.Fatalf("terminal falsely completed hydration = %+v", e.ObserveOperational())
		}
		e.mu.Lock()
		e.state.connectionControl.recoveryAttempts = 3
		e.mu.Unlock()
		behind, _ := NewAggregateIngressFenceInput(terminal.FenceCommand, AggregateIngressFenceComplete, 0, 1, now)
		_, behindCompletion := e.AdmitAggregateIngressFence(context.Background(), behind)
		if result := awaitHydrationDisposition(t, behindCompletion); result.Code != DispositionAggregateIngressFenceFenced || e.state.connectionControl.recoveryAttempts != 3 || e.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("behind marker reached current/reset = %+v view=%+v", result, e.ObserveOperational())
		}
		fence, _ := NewAggregateIngressFenceInput(terminal.FenceCommand, AggregateIngressFenceComplete, 1, 2, now)
		_, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fence)
		fenceResult := awaitHydrationDisposition(t, fenceCompletion)
		firstCurrent := e.ObserveOperational()
		if fenceResult.Code != DispositionAggregateIngressFenceApplied || !firstCurrent.CurrentMarketClaim || firstCurrent.Lifecycle != string(lifecycleLive) ||
			firstCurrent.Watermark == nil || !firstCurrent.Watermark.Equal(start.Add(3*time.Second)) || firstCurrent.Connection.RecoveryAttempts != 0 ||
			e.state.evaluationAppliedSequence != fenceResult.EngineSequence {
			t.Fatalf("fence/evaluator did not atomically establish current = %+v view=%+v", fenceResult, firstCurrent)
		}

		oldToken := token
		now = start.Add(4 * time.Second)
		loss := admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
		if loss.Code != DispositionConnectionControlApplied || e.state.lifecycle != lifecycleRecovering || e.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("post-live loss = %+v view=%+v", loss, e.ObserveOperational())
		}
		ackRecoveryEpoch(t, e, binding, 2, 3, 4, start.Add(5*time.Second), &now)
		gap := admitHydrationPlan(t, e, HydrationGapRecovery, 2, generousHydrationBudgets())
		if gap.Code != DispositionHydrationPlanApplied || gap.Plan.Start() != *firstCurrent.Watermark || gap.Plan.End() != start.Add(5*time.Second) || len(gap.Plan.Requests()) != 1 {
			t.Fatalf("exact gap plan = %+v", gap)
		}
		gapToken := gap.Plan.Requests()[0]
		gapEmpty, _ := NewHydrationTerminalInput(gapToken, gapToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		gapTerminal := admitHydrationTerminal(t, e, gapEmpty)
		gapFence, _ := NewAggregateIngressFenceInput(gapTerminal.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
		_, gapCompletion := e.AdmitAggregateIngressFence(context.Background(), gapFence)
		if recovered := awaitHydrationDisposition(t, gapCompletion); recovered.Code != DispositionAggregateIngressFenceApplied || !e.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("gap recovery did not restore current = %+v view=%+v", recovered, e.ObserveOperational())
		}
		lateOld, _ := NewHydrationTerminalInput(oldToken, oldToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if stale := admitHydrationTerminal(t, e, lateOld); stale.Code != DispositionHydrationFenced {
			t.Fatalf("superseded generation terminal = %+v", stale)
		}
		now = start.Add(6 * time.Second)
		live := liveAggregate(binding, "AAA", start.Add(5*time.Second), 2, 2)
		if liveResult := admitProductionAggregate(t, e, live); liveResult.Code != DispositionAggregateInserted {
			t.Fatalf("post-recovery live aggregate = %+v", liveResult)
		}
		coverageCommand, err := e.IssueLiveCoverageFence()
		if err != nil {
			t.Fatal(err)
		}
		coverage, _ := NewLiveCoverageFenceInput(coverageCommand, LiveCoverageFenceComplete, 2, 2, now)
		coverageAdmission, coverageCompletion := e.AdmitLiveCoverageFence(context.Background(), coverage)
		if coverageAdmission != AdmissionAdmitted || (<-coverageCompletion).Code != DispositionLiveCoverageFenceApplied {
			t.Fatal("ordinary live coverage fence was not applied")
		}
		admission, tick := e.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || awaitTimerDisposition(t, tick).Code != DispositionTimerApplied || e.state.committedT == nil || !e.state.committedT.Equal(start.Add(6*time.Second)) {
			t.Fatalf("ordinary live evaluation froze after recovery: committed=%v", e.state.committedT)
		}
		closeAndWait(t, e)

		now = start.Add(3 * time.Second)
		faulted, faultToken := plannedHydrationEngine(t, binding, &now)
		faultEmpty, _ := NewHydrationTerminalInput(faultToken, faultToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		faultTerminal := admitHydrationTerminal(t, faulted, faultEmpty)
		faulted.mu.Lock()
		faulted.state.connectionControl.recoveryAttempts = 3
		faulted.evaluationFault = true
		faulted.mu.Unlock()
		faultFence, _ := NewAggregateIngressFenceInput(faultTerminal.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
		_, faultCompletion := faulted.AdmitAggregateIngressFence(context.Background(), faultFence)
		faultResult := awaitHydrationDisposition(t, faultCompletion)
		if faultResult.Code != DispositionAccountingIntegrity || faulted.state.connectionControl.recoveryAttempts != 3 || faulted.ObserveOperational().CurrentMarketClaim {
			t.Fatalf("failed evaluator reset retry/currentness = %+v view=%+v", faultResult, faulted.ObserveOperational())
		}
		closeAndWait(t, faulted)
	})

	t.Run("stalled committed T recovers across compacted sealed live presence", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, initialToken := plannedHydrationEngine(t, binding, &now)
		initialEmpty, _ := NewHydrationTerminalInput(initialToken, initialToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		initialTerminal := admitHydrationTerminal(t, e, initialEmpty)
		initialFence, _ := NewAggregateIngressFenceInput(initialTerminal.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
		_, initialCompletion := e.AdmitAggregateIngressFence(context.Background(), initialFence)
		if got := awaitHydrationDisposition(t, initialCompletion); got.Code != DispositionAggregateIngressFenceApplied {
			t.Fatalf("initial fence = %+v", got)
		}
		committed := *e.state.committedT
		preT := liveAggregate(binding, "AAA", committed.Add(-time.Second), 1, 2)
		preT.Values.Open, preT.Values.High, preT.Values.Low, preT.Values.Close, preT.Values.VWAP = 9, 9, 9, 9, 9
		if got := admitProductionAggregate(t, e, preT); got.Code != DispositionAggregateInserted {
			t.Fatalf("pre-T retained mark = %+v", got)
		}
		const stalledSeconds = 17*60 + 2
		values := make(map[int64]AggregateValues, stalledSeconds)
		for second := 0; second < stalledSeconds; second++ {
			window := committed.Add(time.Duration(second) * time.Second)
			now = window.Add(time.Second)
			live := liveAggregate(binding, "AAA", window, 1, uint64(second+3))
			live.Values.Close, live.Values.High, live.Values.VWAP = 10+float64(second%7)/10, 10+float64(second%7)/10, 10+float64(second%7)/10
			values[window.Unix()] = live.Values
			if got := admitProductionAggregate(t, e, live); got.Code != DispositionAggregateInserted {
				t.Fatalf("stalled live second %d = %+v", second, got)
			}
		}
		r := committed.Add(stalledSeconds * time.Second)
		state := aggregateState(t, e, "AAA")
		for _, at := range []time.Time{committed, committed.Add(time.Second)} {
			slot := sessionSlot(e.state.binding, at)
			if state.tail[at.Unix()] != nil || state.presence == nil || !state.presence.has(slot) || state.sealedLive == nil || !state.sealedLive.has(slot) {
				t.Fatalf("live identity did not compact with sealed source support at %s", at)
			}
		}
		preTView := e.selectionStateViewAtLocked(e.state.binding.index["AAA"], committed)
		if !preTView.MarkAvailable || !preTView.TrustedMarkAt.Equal(committed.Add(-time.Second)) || preTView.TrustedMark.Close != 9 {
			t.Fatalf("compaction replaced the strict pre-T mark: %+v", preTView)
		}
		if e.state.committedT == nil || !e.state.committedT.Equal(committed) || r.Sub(committed) <= correctionHorizon {
			t.Fatalf("committed stall = T:%v R:%s", e.state.committedT, r)
		}

		now = r
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1,
			LivePosition{ConnectionEpoch: 1, FrameSequence: stalledSeconds + 3}, now, 0, ControlFailed))
		ackRecoveryEpoch(t, e, binding, 2, 3, 4, r, &now)
		gap := admitHydrationPlan(t, e, HydrationGapRecovery, 2, generousHydrationBudgets())
		if gap.Code != DispositionHydrationPlanApplied || gap.Plan.Start() != committed || gap.Plan.End() != r || len(gap.Plan.Requests()) != 1 {
			t.Fatalf("compacted exact-gap plan = %+v", gap)
		}
		token := gap.Plan.Requests()[0]
		if len(e.observeHydration().Requests[0].reconciliation) != 0 {
			t.Fatal("pre-request compaction fabricated equality history")
		}
		equalRow, _ := NewHydrationRow("AAA", committed, committed.Add(time.Second), values[committed.Unix()])
		mismatchValues := values[committed.Add(time.Second).Unix()]
		mismatchValues.Close, mismatchValues.High, mismatchValues.VWAP = 99, 99, 99
		mismatchRow, _ := NewHydrationRow("AAA", committed.Add(time.Second), committed.Add(2*time.Second), mismatchValues)
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 2, []HydrationRow{equalRow, mismatchRow})
		rows := admitHydrationChunk(t, e, chunk)
		if rows.Code != DispositionHydrationChunkApplied || rows.Rows != (HydrationRowAccounting{Consumed: 2, ConflictOrWithdrawal: 2}) || !rows.Rows.reconciles() ||
			len(e.observeHydration().Requests[0].reconciliation) != 0 {
			t.Fatalf("conservative sealed-live reconciliation = %+v hydration=%+v", rows, e.observeHydration())
		}
		terminalInput, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 2, 1, 2)
		terminal := admitHydrationTerminal(t, e, terminalInput)
		if terminal.Accounting != (HydrationAccounting{Planned: 1, CompletedValue: 1}) || !terminal.Accounting.reconciles() {
			t.Fatalf("compacted gap terminal accounting = %+v", terminal)
		}
		gapFence, _ := NewAggregateIngressFenceInput(terminal.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
		startsBeforeRecoveryFence := e.ObserveEvaluationTiming().Starts
		_, gapCompletion := e.AdmitAggregateIngressFence(context.Background(), gapFence)
		if got := awaitHydrationDisposition(t, gapCompletion); got.Code != DispositionAggregateIngressFenceApplied || !e.ObserveOperational().CurrentMarketClaim ||
			e.state.committedT == nil || !e.state.committedT.Equal(r) {
			t.Fatalf("compacted gap did not restore current = %+v view=%+v", got, e.ObserveOperational())
		}
		if recoveryTiming := e.ObserveEvaluationTiming(); recoveryTiming.Source != AggregateEvaluationIngressFence ||
			recoveryTiming.Starts.AggregateIngressFence != startsBeforeRecoveryFence.AggregateIngressFence+1 {
			t.Fatalf("same-T recovery ingress fence was deduplicated: before=%+v after=%+v", startsBeforeRecoveryFence, recoveryTiming)
		}
		if state.historicalConflict != nil && (state.historicalConflict.has(sessionSlot(e.state.binding, committed)) ||
			state.historicalConflict.has(sessionSlot(e.state.binding, committed.Add(time.Second)))) {
			t.Fatal("conservative REST disposition poisoned sealed live coverage")
		}
		recoveredPreT := e.selectionStateViewAtLocked(e.state.binding.index["AAA"], committed)
		if !recoveredPreT.MarkAvailable || !recoveredPreT.TrustedMarkAt.Equal(committed.Add(-time.Second)) || recoveredPreT.TrustedMark.Close != 9 {
			t.Fatalf("post-recovery as-of T selected T-or-later mark: %+v", recoveredPreT)
		}
		e.mu.Lock()
		recoveredEvaluation := e.stageAggregateEvaluationAtLocked(committed, now)
		e.mu.Unlock()
		if validation := validateAggregateEvaluation(recoveredEvaluation); validation != nil || !recoveredEvaluation.population.reconciles() ||
			recoveredEvaluation.population.trustedRankableMark != 1 || len(recoveredEvaluation.rows) != 1 ||
			recoveredEvaluation.rows[0].symbol != "AAA" || recoveredEvaluation.rows[0].last != 9 {
			t.Fatalf("post-recovery production evaluator at old T = %+v validation=%v", recoveredEvaluation, validation)
		}
		recoveredRow := recoveredEvaluation.rows[0]
		if recoveredRow.sessionVolume != currentField(100) || recoveredRow.fromOpenPercent != currentField(0) ||
			recoveredRow.dayRange.status != featureUnavailable || recoveredRow.dayRange.reason != featureReasonZeroWidth ||
			recoveredRow.activity30s.status != featureWarming || recoveredRow.activity30s.reason != featureReasonReferenceWarmup ||
			recoveredRow.move30s.status != featureWarming || recoveredRow.move30s.reason != featureReasonRollingWarmup ||
			!state.prefix.foldedThrough.After(committed) {
			t.Fatalf("post-recovery B2 as-of-T fields admitted compacted future evidence: row=%+v foldedThrough=%s T=%s", recoveredRow, state.prefix.foldedThrough, committed)
		}

		now = r.Add(time.Second)
		later := liveAggregate(binding, "AAA", r, 2, 2)
		if got := admitProductionAggregate(t, e, later); got.Code != DispositionAggregateInserted {
			t.Fatalf("later ordinary live = %+v", got)
		}
		coverageCommand, err := e.IssueLiveCoverageFence()
		if err != nil {
			t.Fatal(err)
		}
		coverage, _ := NewLiveCoverageFenceInput(coverageCommand, LiveCoverageFenceComplete, 2, 2, now)
		_, coverageCompletion := e.AdmitLiveCoverageFence(context.Background(), coverage)
		if got := <-coverageCompletion; got.Code != DispositionLiveCoverageFenceApplied {
			t.Fatalf("later coverage = %+v", got)
		}
		_, timer := e.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, timer); got.Code != DispositionTimerApplied || e.state.committedT == nil || !e.state.committedT.Equal(now) {
			t.Fatalf("later timer did not advance after compacted recovery = %+v T=%v", got, e.state.committedT)
		}
		closeAndWait(t, e)
	})
}
