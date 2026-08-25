package engine

import (
	"context"
	"testing"
	"time"
)

// TestPLBRA1Canonical is the primary LBR-A1 proof. Its expected projection is
// intentionally test-only: production has no baseline fallback or second
// canonical owner.
func TestPLBRA1Canonical(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("exact horizon out-of-order and foreign binding are contained", func(t *testing.T) {
		window := start.Add(time.Minute)
		now := window.Add(time.Second).Add(correctionHorizon)
		e := aggregateEngine(t, binding, &now)
		atBoundary := liveAggregate(binding, "AAA", window, 1, 2)
		applyAggregate(t, e, atBoundary, DispositionAggregateInserted, ReasonNone)
		atBoundaryRevision := changedClose(atBoundary, 10.2)
		atBoundaryRevision.Live.FrameSequence++
		applyAggregate(t, e, atBoundaryRevision, DispositionAggregateRevised, ReasonNone)
		now = now.Add(time.Nanosecond)
		oneTickLate := changedClose(atBoundaryRevision, 10.3)
		oneTickLate.Live.FrameSequence++
		applyAggregate(t, e, oneTickLate, DispositionAggregateRejected, ReasonTooLate)
		lateEqualWithLiveSupport := atBoundaryRevision
		lateEqualWithLiveSupport.Live.FrameSequence = oneTickLate.Live.FrameSequence + 1
		applyAggregate(t, e, lateEqualWithLiveSupport, DispositionAggregateExactDuplicate, ReasonNone)

		wholeNow := now.Truncate(time.Second)
		newer := liveAggregate(binding, "BAD", wholeNow.Add(-time.Second), 1, 5)
		older := liveAggregate(binding, "BAD", wholeNow.Add(-5*time.Second), 1, 6)
		applyAggregate(t, e, newer, DispositionAggregateInserted, ReasonNone)
		applyAggregate(t, e, older, DispositionAggregateInserted, ReasonNone)
		state := aggregateState(t, e, "BAD")
		if state.latest == nil || state.latest.record.windowStart != newer.WindowStart {
			t.Fatalf("ordinary out-of-order insert regressed latest: %+v", state.latest)
		}
		foreign := liveAggregate(binding, "MISSING", wholeNow.Add(-time.Second), 1, 7)
		foreign.BindingIdentity = mutateIdentity(binding.Identity())
		applyAggregate(t, e, foreign, DispositionAggregateFenced, ReasonBinding)
		if _, ok := e.state.binding.index["NOT-BOUND"]; ok {
			t.Fatal("foreign symbol acquired a canonical slot")
		}
		closeAndWait(t, e)
	})

	t.Run("bounded tail prefix correction and immutable affected views", func(t *testing.T) {
		now := start.Add(time.Second)
		e := aggregateEngine(t, binding, &now)
		const records = 1000
		for second := 0; second < records; second++ {
			window := start.Add(time.Duration(second) * time.Second)
			now = window.Add(time.Second)
			input := liveAggregate(binding, "AAA", window, 1, uint64(second+1))
			input.Values.Open = 9.5
			input.Values.High = 10 + float64(second)/1000
			input.Values.Low = 9 - float64(second)/2000
			input.Values.Close = 9.75 + float64(second)/2000
			input.Values.VWAP = input.Values.Close
			applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
			state := aggregateState(t, e, "AAA")
			if len(state.tail) > maximumTailRecords || state.tailBoundHits != 0 {
				t.Fatalf("tail bound at second %d: records=%d hits=%d", second, len(state.tail), state.tailBoundHits)
			}
		}

		state := aggregateState(t, e, "AAA")
		if got, want := len(state.tail), maximumTailRecords; got != want {
			t.Fatalf("inclusive 16-minute tail=%d want=%d", got, want)
		}
		if got, want := state.prefix.printCount, uint32(records-maximumTailRecords); got != want {
			t.Fatalf("sealed prefix prints=%d want=%d", got, want)
		}
		if got, want := state.prefix.volume, float64(records-maximumTailRecords)*100; got != want {
			t.Fatalf("sealed prefix volume=%v want=%v", got, want)
		}
		if !state.prefix.firstOpenTrusted || !state.prefix.extremaTrusted || !state.prefix.latestTrusted {
			t.Fatalf("sealed sufficient inputs unexpectedly untrusted: %+v", state.prefix)
		}

		latestWindow := start.Add((records - 1) * time.Second)
		beforeRevision := state.canonicalRevision
		revision := liveAggregate(binding, "AAA", latestWindow, 1, records+1)
		revision.Values = changedClose(revision, 10.75).Values
		applyAggregate(t, e, revision, DispositionAggregateRevised, ReasonNone)
		duplicate := revision
		duplicate.Live.FrameSequence++
		applyAggregate(t, e, duplicate, DispositionAggregateExactDuplicate, ReasonNone)

		selection := e.observeSelectionState()
		if len(selection) != len(binding.UniverseSymbols()) || selection[0].CanonicalRevision != beforeRevision+1 ||
			selection[0].Affected.Proofs != aggregateProofAll || selection[1].CanonicalRevision != 0 {
			t.Fatalf("symbol-local revision/notification = %+v other=%+v", selection[0], selection[1])
		}
		sealed := historicalAggregate(binding, "AAA", start, 1)
		sealed.Values = changedClose(sealed, 11).Values
		sealedProof := proofFor(binding, sealed, start, start.Add(time.Second))
		prefixBefore := aggregateState(t, e, "AAA").prefix
		applyHistorical(t, e, sealed, sealedProof, DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		if got := aggregateState(t, e, "AAA").prefix; got != prefixBefore {
			t.Fatal("historical mismatch revised a sealed live identity")
		}
		closeAndWait(t, e)
	})

	t.Run("old historical fill withdraws only its prefix effect and sealed live never reopens", func(t *testing.T) {
		now := start.Add(30 * time.Minute)
		e := aggregateEngine(t, binding, &now)
		old := historicalAggregate(binding, "MISSING", start.Add(time.Minute), 1)
		old.Values.Open, old.Values.High, old.Values.Low, old.Values.Close, old.Values.VWAP, old.Values.Volume = 9, 12, 8, 11, 10, 250
		proof := proofFor(binding, old, start, start.Add(5*time.Minute))
		applyHistorical(t, e, old, proof, DispositionAggregateInserted, ReasonNone)
		state := aggregateState(t, e, "MISSING")
		if len(state.tail) != 0 || state.prefix.printCount != 1 || state.prefix.volume != 250 ||
			coverageClassAt(state, e.state.binding, old.WindowStart) != AggregateCoveragePresent {
			t.Fatalf("direct-fold old fill = %+v", state)
		}

		equalTooLateLive := liveAggregate(binding, "MISSING", old.WindowStart, 1, 1)
		equalTooLateLive.Values = old.Values
		applyAggregate(t, e, equalTooLateLive, DispositionAggregateRejected, ReasonTooLate)
		if state.latest == nil || state.latest.record.authority.source != AggregateSourceHistorical {
			t.Fatal("too-late equal live evidence advanced historical-only authority")
		}

		second := historicalAggregate(binding, "MISSING", old.WindowStart.Add(time.Second), 2)
		second.Values.Open, second.Values.High, second.Values.Low, second.Values.Close, second.Values.VWAP, second.Values.Volume = 10, 11, 9, 10.5, 10, 125
		applyHistorical(t, e, second, proof, DispositionAggregateInserted, ReasonNone)
		conflict := changedClose(old, 11.5)
		conflict.Historical.RecordOrdinal = 3
		applyHistorical(t, e, conflict, proof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		state = aggregateState(t, e, "MISSING")
		if state.prefix.printCount != 1 || state.prefix.volume != second.Values.Volume || state.prefix.firstOpenTrusted ||
			state.prefix.extremaTrusted || !state.prefix.latestTrusted || state.prefix.latestAt != second.WindowStart.Unix() ||
			coverageClassAt(state, e.state.binding, old.WindowStart) != AggregateCoverageConflict {
			t.Fatalf("folded historical withdrawal did not localize uncertainty: %+v", state)
		}
		third := historicalAggregate(binding, "MISSING", second.WindowStart.Add(time.Second), 4)
		third.Values.Open, third.Values.High, third.Values.Low, third.Values.Close, third.Values.VWAP, third.Values.Volume = 20, 25, 7, 22, 21, 500
		applyHistorical(t, e, third, proof, DispositionAggregateInserted, ReasonNone)
		state = aggregateState(t, e, "MISSING")
		if state.prefix.firstOpenTrusted || state.prefix.extremaTrusted || !state.prefix.latestTrusted || state.prefix.latestAt != third.WindowStart.Unix() {
			t.Fatalf("unrelated fold restored conflicted first-open/extrema trust: %+v", state.prefix)
		}
		lateLive := liveAggregate(binding, "MISSING", old.WindowStart, 1, 2)
		applyAggregate(t, e, lateLive, DispositionAggregateRejected, ReasonTooLate)
		if state.prefix.printCount != 2 || len(state.tail) != 0 {
			t.Fatal("too-late live input reopened sealed identity")
		}
		closeAndWait(t, e)

		now = start.Add(30 * time.Minute)
		latest := aggregateEngine(t, binding, &now)
		sole := historicalAggregate(binding, "BAD", start.Add(5*time.Minute), 1)
		soleProof := proofFor(binding, sole, sole.WindowStart, sole.WindowStart.Add(5*time.Second))
		applyHistorical(t, latest, sole, soleProof, DispositionAggregateInserted, ReasonNone)
		soleConflict := changedClose(sole, 11)
		soleConflict.Historical.RecordOrdinal = 2
		applyHistorical(t, latest, soleConflict, soleProof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		later := historicalAggregate(binding, "BAD", sole.WindowStart.Add(time.Second), 3)
		applyHistorical(t, latest, later, soleProof, DispositionAggregateInserted, ReasonNone)
		latestState := aggregateState(t, latest, "BAD")
		selection := latest.observeSelectionState()[latest.state.binding.index["BAD"]]
		if latestState.prefix.firstOpenTrusted || latestState.prefix.extremaTrusted || !latestState.prefix.latestTrusted ||
			latestState.latest == nil || latestState.latest.record.identity.start != later.WindowStart.Unix() || !selection.MarkAvailable || selection.TrustedMarkAt != later.WindowStart {
			t.Fatalf("later actual mark did not restore latest only: prefix=%+v latest=%+v selection=%+v", latestState.prefix, latestState.latest, selection)
		}
		closeAndWait(t, latest)
	})

	t.Run("sealed live keeps no comparison outside an active exact request", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e := aggregateEngine(t, binding, &now)
		live := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, e, live, DispositionAggregateInserted, ReasonNone)
		laterLive := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2)
		applyAggregate(t, e, laterLive, DispositionAggregateInserted, ReasonNone)
		now = start.Add(20 * time.Minute)
		e.mu.Lock()
		state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
		e.compactSymbolLocked(state, e.state.binding, "AAA", now)
		e.mu.Unlock()
		equal := historicalAggregate(binding, "AAA", start, 1)
		equal.Values = live.Values
		applyHistorical(t, e, equal, proofFor(binding, equal, start, start.Add(time.Second)), DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		if state.prefix.volume != live.Values.Volume+laterLive.Values.Volume || state.prefix.printCount != 2 {
			t.Fatalf("unclaimed historical evidence changed sealed live state: %+v", state.prefix)
		}
		closeAndWait(t, e)
	})

	t.Run("production hydration ledger accepts sealed duplicate and localizes mismatch", func(t *testing.T) {
		for _, mismatch := range []bool{false, true} {
			now := start.Add(3 * time.Second)
			e, token := plannedHydrationEngine(t, binding, &now)
			live := liveAggregate(binding, "AAA", start, 1, 2)
			live.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
			live.Live.ArrayIndex = 1
			if got := admitProductionAggregate(t, e, live); got.Code != DispositionAggregateInserted {
				t.Fatalf("live = %+v", got)
			}
			laterLive := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 3)
			if got := admitProductionAggregate(t, e, laterLive); got.Code != DispositionAggregateInserted {
				t.Fatalf("later live = %+v", got)
			}
			now = start.Add(20 * time.Minute)
			e.mu.Lock()
			state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
			e.compactSymbolLocked(state, e.state.binding, "AAA", now)
			e.mu.Unlock()
			beforeChunk := e.observeHydration().Requests[0]
			if len(beforeChunk.reconciliation) != 2 || len(beforeChunk.reconciliation) > maximumTailRecords || cap(beforeChunk.reconciliation) > maximumTailRecords {
				t.Fatalf("active request reconciliation bound = len:%d cap:%d hits:%d", len(beforeChunk.reconciliation), cap(beforeChunk.reconciliation), beforeChunk.reconciliationBoundHits)
			}
			values := live.Values
			if mismatch {
				values.Close, values.High = 11, 11
			}
			row, err := NewHydrationRow("AAA", start, start.Add(time.Second), values)
			if err != nil {
				t.Fatal(err)
			}
			chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
			if err != nil {
				t.Fatal(err)
			}
			got := admitHydrationChunk(t, e, chunk)
			want := HydrationRowAccounting{Consumed: 1, Duplicate: 1}
			if mismatch {
				want = HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}
			}
			if got.Code != DispositionHydrationChunkApplied || got.Rows != want || !got.Rows.reconciles() {
				t.Fatalf("sealed production hydration mismatch=%t disposition=%+v want=%+v", mismatch, got, want)
			}
			if afterChunk := e.observeHydration().Requests[0]; len(afterChunk.reconciliation) != 1 {
				t.Fatalf("matching row did not consume reconciliation evidence: %+v", afterChunk)
			}
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			terminalDisposition := admitHydrationTerminal(t, e, terminal)
			if terminalDisposition.Code != DispositionHydrationTerminalApplied {
				t.Fatalf("sealed production terminal = %+v", terminalDisposition)
			}
			if observation := e.observeHydration(); observation.Requests[0].coverage != hydrationCoverageCandidateComplete || len(observation.Requests[0].reconciliation) != 0 {
				t.Fatalf("trusted sealed live support poisoned terminal coverage: %+v", observation)
			}
			fence, err := NewAggregateIngressFenceInput(terminalDisposition.FenceCommand, AggregateIngressFenceComplete, 3, 1, now)
			if err != nil {
				t.Fatal(err)
			}
			admission, completion := e.AdmitAggregateIngressFence(context.Background(), fence)
			if admission != AdmissionAdmitted || awaitHydrationDisposition(t, completion).Code != DispositionAggregateIngressFenceApplied || len(e.observeHydration().Requests[0].reconciliation) != 0 {
				t.Fatal("successful fence retained reconciliation evidence")
			}
			closeAndWait(t, e)
		}
		now := start.Add(3 * time.Second)
		canceled, _ := plannedHydrationEngine(t, binding, &now)
		live := liveAggregate(binding, "AAA", start, 1, 2)
		live.Live.ArrayIndex = 1
		admitProductionAggregate(t, canceled, live)
		now = start.Add(20 * time.Minute)
		canceled.mu.Lock()
		state := canceled.state.binding.symbols[canceled.state.binding.index["AAA"]].aggregates
		canceled.compactSymbolLocked(state, canceled.state.binding, "AAA", now)
		canceled.mu.Unlock()
		if len(canceled.observeHydration().Requests[0].reconciliation) != 1 {
			t.Fatal("cancel fixture did not retain active request reconciliation")
		}
		loss := admitConnectionControl(t, canceled, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 3}, now, 0, ControlFailed))
		if loss.Code != DispositionConnectionControlApplied || canceled.observeHydration().Active || len(canceled.observeHydration().Requests[0].reconciliation) != 0 {
			t.Fatal("replacement epoch retained reconciliation evidence")
		}
		closeAndWait(t, canceled)

		now = start.Add(20 * time.Minute)
		bounded, _ := plannedHydrationEngine(t, binding, &now)
		bounded.mu.Lock()
		for second := 0; second <= maximumTailRecords; second++ {
			at := start.Add(time.Duration(second) * time.Second)
			record := qualificationRecord("AAA", at, 10, 100, 10, 10, ATSLiveProviderAverage)
			bounded.retainHydrationReconciliationLocked("AAA", *record)
		}
		entry := bounded.state.hydration.generation.requests[0]
		bounded.mu.Unlock()
		if len(entry.reconciliation) != maximumTailRecords || cap(entry.reconciliation) != maximumTailRecords || entry.reconciliationBoundHits != 1 || entry.coverage != hydrationCoverageUnknown {
			t.Fatalf("temporary reconciliation bound = len:%d cap:%d hits:%d coverage:%d", len(entry.reconciliation), cap(entry.reconciliation), entry.reconciliationBoundHits, entry.coverage)
		}
		closeAndWait(t, bounded)
	})

	t.Run("historical live precedence no-print invalid and unknown remain distinct", func(t *testing.T) {
		now := start.Add(10 * time.Minute)
		e := aggregateEngine(t, binding, &now)
		window := now.Add(-time.Minute)
		historical := historicalAggregate(binding, "BAD", window, 1)
		proof := proofFor(binding, historical, window, window.Add(time.Second))
		applyHistorical(t, e, historical, proof, DispositionAggregateInserted, ReasonNone)
		live := liveAggregate(binding, "BAD", window, 2, 1)
		live.Values = changedClose(live, 10.4).Values
		applyAggregate(t, e, live, DispositionAggregateRevised, ReasonNone)
		mismatch := historical
		mismatch.Historical.RecordOrdinal = 2
		mismatch.Values = changedClose(mismatch, 10.8).Values
		applyHistorical(t, e, mismatch, proof, DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		if got := aggregateRecord(t, e, "BAD", window); got.values != live.Values || got.authority.source != AggregateSourceLive {
			t.Fatalf("historical/live permutation changed trusted live state: %+v", got)
		}

		e.mu.Lock()
		missing := ensureAggregateState(&e.state.binding.symbols[e.state.binding.index["MISSING"]])
		absenceAt := window.Add(10 * time.Second)
		if !installExactCoverage(missing, e.state.binding, absenceAt, absenceAt.Add(time.Second), nil) {
			e.mu.Unlock()
			t.Fatal("install exact no-print coverage")
		}
		if got := coverageClassAt(missing, e.state.binding, absenceAt); got != AggregateCoverageProvenAbsent {
			e.mu.Unlock()
			t.Fatalf("no-print coverage=%v", got)
		}
		unknownAt := absenceAt.Add(time.Second)
		if got := coverageClassAt(missing, e.state.binding, unknownAt); got != AggregateCoverageUnknown {
			e.mu.Unlock()
			t.Fatalf("unknown coverage=%v", got)
		}
		beforeInvalidRevision := missing.canonicalRevision
		e.mu.Unlock()

		invalid := liveAggregate(binding, "MISSING", absenceAt, 2, 2)
		invalid.Values.Volume = -1
		applyAggregate(t, e, invalid, DispositionAggregateRejected, ReasonStructural)
		selection := e.observeSelectionState()
		missingIndex := e.state.binding.index["MISSING"]
		if selection[missingIndex].Coverage != AggregateCoverageInvalid || selection[missingIndex].CanonicalRevision != beforeInvalidRevision+1 ||
			selection[missingIndex].Affected.Proofs != aggregateProofAll {
			t.Fatalf("invalid evidence notification=%+v", selection[missingIndex])
		}
		applyAggregate(t, e, invalid, DispositionAggregateRejected, ReasonStructural)
		if repeated := e.observeSelectionState()[missingIndex]; repeated.CanonicalRevision != beforeInvalidRevision+1 {
			t.Fatalf("repeated invalid evidence double-notified: %+v", repeated)
		}
		valid := liveAggregate(binding, "MISSING", absenceAt, 2, 3)
		applyAggregate(t, e, valid, DispositionAggregateInserted, ReasonNone)
		if cleared := e.observeSelectionState()[missingIndex]; cleared.CanonicalRevision != beforeInvalidRevision+2 || cleared.Coverage != AggregateCoveragePresent {
			t.Fatalf("accepted replacement did not notify exactly once: %+v", cleared)
		}
		closeAndWait(t, e)
	})
}
