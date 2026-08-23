package engine

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestPLBRB1Selection is P-LBR-B1-SELECTION. It composes the incremental
// qualification, strict as-of mark, compact ordering/accounting, cycle
// coalescing, and owner-sequence boundaries. Display-field calculation and
// immutable snapshot capture are intentionally outside this proof.
func TestPLBRB1Selection(t *testing.T) {
	t.Run("gate correction equality and strict finalization", func(t *testing.T) {
		binding := testBinding(t)
		start := binding.SessionStart()
		proofEnd := start.Add(60 * time.Second)
		seed := func(now *time.Time) *Engine {
			e := aggregateEngine(t, binding, RunModeLive, now)
			for second := 0; second < 60; second++ {
				volume := 497.5
				if second == 59 {
					volume = 510
				}
				applyAggregate(t, e, qualificationInput(binding, start.Add(time.Duration(second)*time.Second), uint64(second+1), volume), DispositionAggregateInserted, ReasonNone)
			}
			proveAggregateCoverage(t, e, "AAA", start, proofEnd)
			setCommittedQualificationTime(e, proofEnd)
			applyQualificationTimer(t, e)
			if got := qualificationForSymbol(t, e, "AAA"); got.result.status != qualificationProvisional || len(got.proofs) != 1 {
				t.Fatalf("approved gate did not create the sole provisional proof: %+v", got)
			}
			return e
		}

		now := proofEnd.Add(time.Second)
		mutable := seed(&now)
		now = proofEnd.Add(correctionHorizon)
		correction := qualificationInput(binding, proofEnd.Add(-time.Second), 61, 510)
		correction.Values.AverageTradeSize = 0
		if got := applyAggregate(t, mutable, correction, DispositionAggregateRevised, ReasonNone); got.Code != DispositionAggregateRevised {
			t.Fatalf("equality correction = %+v", got)
		}
		if got := qualificationForSymbol(t, mutable, "AAA"); got.finalized || got.result.status != qualificationNotYetPassed || got.revoked != 1 {
			t.Fatalf("equality correction did not revoke mutable proof: %+v", got)
		}
		closeAndWait(t, mutable)

		now = proofEnd.Add(time.Second)
		finalized := seed(&now)
		now = proofEnd.Add(correctionHorizon)
		applyQualificationTimer(t, finalized)
		if got := qualificationForSymbol(t, finalized, "AAA"); got.finalized {
			t.Fatalf("proof finalized at equality: %+v", got)
		}
		now = now.Add(time.Nanosecond)
		applyQualificationTimer(t, finalized)
		if got := qualificationForSymbol(t, finalized, "AAA"); !got.finalized || got.result.status != qualificationFinalized || !got.finalProofEnd.Equal(proofEnd) {
			t.Fatalf("strict finalization = %+v", got)
		}
		now = now.Add(time.Minute)
		applyQualificationTimer(t, finalized)
		if got := qualificationForSymbol(t, finalized, "AAA"); !got.finalized || !got.finalProofEnd.Equal(proofEnd) {
			t.Fatalf("quiet timer revoked finalized proof: %+v", got)
		}
		closeAndWait(t, finalized)
	})

	t.Run("strict as-of mark survives a correction-horizon stall", func(t *testing.T) {
		candidate := time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)
		e := evaluatorProofEngine(candidate, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
		state := e.state.binding.symbols[0].aggregates
		preT := *state.tail[candidate.Add(-time.Second).Unix()]
		preT.authority = aggregateEvidence{source: AggregateSourceLive, live: LivePosition{ConnectionEpoch: 1, FrameSequence: 1}}
		preT.greatestLiveSupport = &preT.authority.live
		state.tail[preT.identity.start] = &preT
		state.latest.record = preT
		state.committedLatest = committedMark(preT)
		advanceSelectionMark(state, e.state.binding, candidate)

		for offset, closeValue := range []float64{80, 90} {
			start := candidate.Add(time.Duration(offset) * time.Second)
			record := qualificationRecord("AAA", start, closeValue, 1000, closeValue, 10, ATSLiveProviderAverage)
			record.authority = aggregateEvidence{source: AggregateSourceLive, live: LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(offset + 2)}}
			record.greatestLiveSupport = &record.authority.live
			state.tail[start.Unix()] = record
			considerLatestMark(state, *record)
		}
		advanceSelectionMark(state, e.state.binding, candidate.Add(time.Second))
		atEquality := e.selectionStateViewAtLocked(0, candidate.Add(time.Second))
		if !atEquality.MarkAvailable || !atEquality.TrustedMarkAt.Equal(candidate) || atEquality.TrustedMark.Close != 80 {
			t.Fatalf("first later print was not selected at its first eligible boundary: %+v", atEquality)
		}
		e.compactSymbolLocked(state, e.state.binding, "AAA", candidate.Add(correctionHorizon+2*time.Minute))

		view := e.selectionStateViewAtLocked(0, candidate)
		got := e.stageAggregateEvaluationLocked(candidate)
		if !view.MarkAvailable || !view.TrustedMarkAt.Equal(candidate.Add(-time.Second)) || view.TrustedMark.Close != 12 ||
			got.mode != rankingQualifiedCurrent || len(got.rows) != 1 || got.rows[0].last != 12 {
			t.Fatalf("future-to-T mark displaced retained predecessor: view=%+v evaluation=%+v", view, got)
		}
	})

	t.Run("committed mark revision and withdrawal survive multiple post-T marks", func(t *testing.T) {
		binding := testBinding(t)
		target := binding.SessionStart().Add(10 * time.Second)
		now := target.Add(3 * time.Second)
		values := func(start time.Time, closeValue float64) AggregateInput {
			input := liveAggregate(binding, "AAA", start, 1, uint64(start.Sub(binding.SessionStart())/time.Second)+2)
			input.Values.Open, input.Values.High, input.Values.Low, input.Values.Close, input.Values.VWAP = closeValue, closeValue, closeValue, closeValue, closeValue
			return input
		}
		commit := func(e *Engine) {
			e.mu.Lock()
			candidate := e.stageAggregateEvaluationAtLocked(target, now)
			e.applyStagedAggregateCandidateLocked(candidate, now)
			e.mu.Unlock()
		}

		live := aggregateEngine(t, binding, RunModeLive, &now)
		live.mu.Lock()
		live.state.lifecycle, live.state.liveEpoch, live.state.liveEpochActive = lifecycleLive, 1, true
		live.state.aggregateAcknowledged = true
		live.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
		live.mu.Unlock()
		for _, input := range []AggregateInput{values(target.Add(-2*time.Second), 11), values(target.Add(-time.Second), 12)} {
			if got := admitProductionAggregate(t, live, input); got.Code != DispositionAggregateInserted {
				t.Fatalf("live seed %s = %+v", input.WindowStart, got)
			}
		}
		commit(live)
		for _, input := range []AggregateInput{values(target, 80), values(target.Add(time.Second), 90), values(target.Add(2*time.Second), 100)} {
			if got := admitProductionAggregate(t, live, input); got.Code != DispositionAggregateInserted {
				t.Fatalf("post-T seed %s = %+v", input.WindowStart, got)
			}
		}
		revision := values(target.Add(-time.Second), 13)
		revision.Live.FrameSequence = 20
		if got := admitProductionAggregate(t, live, revision); got.Code != DispositionAggregateRevised {
			t.Fatalf("committed pre-T revision = %+v", got)
		}
		live.mu.Lock()
		revised := live.stageAggregateEvaluationAtLocked(target, now)
		revisedView := live.selectionStateViewAtLocked(live.state.binding.index["AAA"], target)
		priorClose := live.state.binding.symbols[live.state.binding.index["AAA"]].prior.close
		live.mu.Unlock()
		wantDay := percentChange(13, priorClose)
		if len(revised.rows) != 1 || revised.rows[0].last != 13 || revised.rows[0].dayPercent != wantDay.value || revisedView.TrustedMark.Close != 13 {
			t.Fatalf("revision left stale compact mark or divergent Day%%: %+v", revised)
		}
		live.mu.Lock()
		advanceSelectionMark(live.state.binding.symbols[live.state.binding.index["AAA"]].aggregates, live.state.binding, target.Add(time.Second))
		middle := live.stageAggregateEvaluationAtLocked(target.Add(time.Second), now)
		live.mu.Unlock()
		if len(middle.rows) != 1 || middle.rows[0].last != 80 {
			t.Fatalf("candidate-relative step did not select T-1 between retained future rows: %+v", middle)
		}
		closeAndWait(t, live)

		historical := aggregateEngine(t, binding, RunModeLive, &now)
		inputs := []AggregateInput{values(target.Add(-3*time.Second), 10), values(target.Add(-2*time.Second), 11), values(target.Add(-time.Second), 12), values(target, 80), values(target.Add(time.Second), 90)}
		for ordinal := 0; ordinal < 3; ordinal++ {
			input := inputs[ordinal]
			input.Source, input.Live = AggregateSourceHistorical, LivePosition{}
			input.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
			input.Historical = HistoricalPosition{Generation: 1, RequestToken: "selection-withdrawal", RecordOrdinal: uint64(ordinal + 1)}
			applyHistorical(t, historical, input, proofFor(binding, input, binding.SessionStart(), now), DispositionAggregateInserted, ReasonNone)
		}
		commit(historical)
		for ordinal := 3; ordinal < len(inputs); ordinal++ {
			input := inputs[ordinal]
			input.Source, input.Live = AggregateSourceHistorical, LivePosition{}
			input.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
			input.Historical = HistoricalPosition{Generation: 1, RequestToken: "selection-withdrawal", RecordOrdinal: uint64(ordinal + 1)}
			applyHistorical(t, historical, input, proofFor(binding, input, binding.SessionStart(), now), DispositionAggregateInserted, ReasonNone)
		}
		withdraw := inputs[2]
		withdraw.Source, withdraw.Live = AggregateSourceHistorical, LivePosition{}
		withdraw.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
		withdraw.Historical = HistoricalPosition{Generation: 1, RequestToken: "selection-withdrawal", RecordOrdinal: 6}
		withdraw.Values.Open, withdraw.Values.High, withdraw.Values.Low, withdraw.Values.Close, withdraw.Values.VWAP = 14, 14, 14, 14, 14
		applyHistorical(t, historical, withdraw, proofFor(binding, withdraw, binding.SessionStart(), now), DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		historical.mu.Lock()
		view := historical.selectionStateViewAtLocked(historical.state.binding.index["AAA"], target)
		historical.mu.Unlock()
		if !view.MarkAvailable || !view.TrustedMarkAt.Equal(target.Add(-2*time.Second)) || view.TrustedMark.Close != 11 {
			t.Fatalf("withdrawal did not reveal next eligible pre-T mark: %+v", view)
		}
		withdrawAgain := inputs[1]
		withdrawAgain.Source, withdrawAgain.Live = AggregateSourceHistorical, LivePosition{}
		withdrawAgain.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
		withdrawAgain.Historical = HistoricalPosition{Generation: 1, RequestToken: "selection-withdrawal", RecordOrdinal: 7}
		withdrawAgain.Values.Open, withdrawAgain.Values.High, withdrawAgain.Values.Low, withdrawAgain.Values.Close, withdrawAgain.Values.VWAP = 15, 15, 15, 15, 15
		applyHistorical(t, historical, withdrawAgain, proofFor(binding, withdrawAgain, binding.SessionStart(), now), DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		historical.mu.Lock()
		view = historical.selectionStateViewAtLocked(historical.state.binding.index["AAA"], target)
		historical.mu.Unlock()
		if !view.MarkAvailable || !view.TrustedMarkAt.Equal(target.Add(-3*time.Second)) || view.TrustedMark.Close != 10 {
			t.Fatalf("second withdrawal did not repair exact next eligible mark: %+v", view)
		}
		closeAndWait(t, historical)
	})

	t.Run("exact cardinality ties and independent absence", func(t *testing.T) {
		at := time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)
		e := evaluatorProofEngine(at, []evaluatorSymbol{
			{"AAA", reference.PriorCloseValid, 10, 12, qualificationProvisional},
			{"AAB", reference.PriorCloseValid, 10, 12, qualificationFinalized},
			{"HIGH", reference.PriorCloseValid, 10, 99, qualificationNotYetPassed},
			{"BAD", reference.PriorCloseInvalid, 0, 0, qualificationUnresolved},
			{"NONE", reference.PriorCloseValid, 10, 0, qualificationUnresolved},
		})
		// T/Q and display-only state are deliberately absent from every symbol.
		got := e.stageAggregateEvaluationLocked(at)
		if got.mode != rankingDegradedCurrent || got.population.universeTotal != 5 || !got.population.reconciles() ||
			got.totalPassers != 2 || len(got.rows) != 3 || got.rows[0].symbol != "HIGH" || got.rows[1].symbol != "AAA" || got.rows[2].symbol != "AAB" ||
			got.rows[0].tqIntentEligible || got.rows[1].tqIntentEligible || got.rows[2].tqIntentEligible {
			t.Fatalf("partial population/tie/independence result = %+v", got)
		}
	})

	t.Run("one cycle per unchanged supported prefix", func(t *testing.T) {
		binding := testBinding(t)
		at := binding.SessionStart().Add(2 * time.Second)
		now := at
		e := aggregateEngine(t, binding, RunModeLive, &now)
		defer closeAndWait(t, e)
		e.mu.Lock()
		e.state.lifecycle = lifecycleLive
		e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
		e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
		e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
		e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 10, 1
		e.state.hydration.supportedThrough = immutableTime(at)
		e.mu.Unlock()

		input := liveAggregate(binding, "AAA", at.Add(-time.Second), 1, 2)
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
			t.Fatalf("seed aggregate = %+v", got)
		}
		_, first := e.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, first); got.Code != DispositionTimerApplied {
			t.Fatalf("first timer = %+v", got)
		}
		starts := e.ObserveEvaluationTiming().Starts
		_, duplicate := e.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, duplicate); got.Code != DispositionTimerApplied || e.ObserveEvaluationTiming().Starts != starts {
			t.Fatalf("unchanged prefix repeated selection: disposition=%+v before=%+v after=%+v", got, starts, e.ObserveEvaluationTiming().Starts)
		}

		input.Live.FrameSequence = 3
		input.Values.Close, input.Values.High, input.Values.VWAP = 11, 11, 11
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateRevised {
			t.Fatalf("same-T correction = %+v", got)
		}
		_, corrected := e.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, corrected); got.Code != DispositionTimerApplied || e.ObserveEvaluationTiming().Starts.Timer != starts.Timer+1 {
			t.Fatalf("canonical revision did not receive exactly one cycle: disposition=%+v timing=%+v", got, e.ObserveEvaluationTiming())
		}
	})

	t.Run("engine-owned timer and control sequences never reuse", func(t *testing.T) {
		binding := testBinding(t)
		at := binding.SessionStart().Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &at)
		defer closeAndWait(t, e)
		attempt := controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, at, 1, ControlSucceeded)
		attemptAdmission, firstControl := e.AdmitConnectionControl(context.Background(), attempt)
		controlOne := <-firstControl
		_, timer := e.AdmitTimer(context.Background())
		tick := awaitTimerDisposition(t, timer)
		established := controlFact(binding.Identity(), ConnectionEstablished, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, at, 1, ControlSucceeded)
		establishedAdmission, secondControl := e.AdmitConnectionControl(context.Background(), established)
		controlTwo := <-secondControl
		if attemptAdmission != AdmissionAdmitted || establishedAdmission != AdmissionAdmitted ||
			controlOne.Code != DispositionConnectionControlApplied || controlTwo.Code != DispositionConnectionControlApplied ||
			controlOne.SystemSequence != 1 || tick.SystemSequence != 2 || controlTwo.SystemSequence != 3 ||
			controlOne.EngineSequence == tick.EngineSequence || tick.EngineSequence == controlTwo.EngineSequence {
			t.Fatalf("production control positions reused or caller-shaped: control1=%+v timer=%+v control2=%+v", controlOne, tick, controlTwo)
		}
		if controlTwo.SystemSequence == established.Position.FrameSequence {
			t.Fatalf("system sequence impersonated provider position: control=%+v input=%+v", controlTwo, established.Position)
		}
	})
}
