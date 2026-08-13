package engine

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func TestENGAGG01ValidationMergeRetentionMatrix(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("validation binding source and causal fences are mutation isolated", func(t *testing.T) {
		now := start.Add(10 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		base := liveAggregate(binding, "AAA", start, 1, 1)
		result, completion := e.AdmitAggregate(context.Background(), base)
		if result != AdmissionAdmitted || completion == nil {
			t.Fatalf("production aggregate admission = %s", result)
		}
		if got := awaitAggregateDisposition(t, completion); got.Code != DispositionAggregateRejected || got.Reason != ReasonLifecycle {
			t.Fatalf("pre-S3 production lifecycle disposition = %+v", got)
		}
		assertAggregateAccounting(t, e)
		assertAggregateBinDelta(t, aggregateAccounting{}, aggregateAccountingSnapshot(e), DispositionAggregateRejected)
		assertNoAggregateState(t, e, "AAA")
		cases := []struct {
			name   string
			mutate func(*AggregateInput)
			code   DispositionCode
			reason DispositionReason
		}{
			{"schema", func(v *AggregateInput) { v.SchemaVersion = "other" }, DispositionAggregateRejected, ReasonSchema},
			{"binding fence", func(v *AggregateInput) { v.BindingIdentity = mutateIdentity(binding.Identity()) }, DispositionAggregateFenced, ReasonBinding},
			{"run source", func(v *AggregateInput) {
				v.Source = AggregateSourceReplay
				v.Live = LivePosition{}
				v.Replay = ReplayPosition{ArtifactID: "a", RecordOrdinal: 1}
			}, DispositionAggregateRejected, ReasonRunSource},
			{"unknown symbol", func(v *AggregateInput) { v.Symbol = "ZZZ" }, DispositionAggregateRejected, ReasonSymbol},
			{"before session", func(v *AggregateInput) { v.WindowStart = start.Add(-time.Second); v.WindowEnd = start }, DispositionAggregateRejected, ReasonWindow},
			{"at session end", func(v *AggregateInput) {
				v.WindowStart = binding.SessionEnd()
				v.WindowEnd = binding.SessionEnd().Add(time.Second)
			}, DispositionAggregateRejected, ReasonWindow},
			{"not one second", func(v *AggregateInput) { v.WindowEnd = v.WindowStart.Add(2 * time.Second) }, DispositionAggregateRejected, ReasonWindow},
			{"not whole second", func(v *AggregateInput) {
				v.WindowStart = v.WindowStart.Add(time.Nanosecond)
				v.WindowEnd = v.WindowEnd.Add(time.Nanosecond)
			}, DispositionAggregateRejected, ReasonWindow},
			{"nonpositive price", func(v *AggregateInput) { v.Values.Open = 0 }, DispositionAggregateRejected, ReasonStructural},
			{"nonfinite price", func(v *AggregateInput) { v.Values.High = math.NaN() }, DispositionAggregateRejected, ReasonStructural},
			{"negative volume", func(v *AggregateInput) { v.Values.Volume = -1 }, DispositionAggregateRejected, ReasonStructural},
			{"negative ats", func(v *AggregateInput) { v.Values.AverageTradeSize = -1 }, DispositionAggregateRejected, ReasonStructural},
			{"unknown provenance", func(v *AggregateInput) { v.Values.ATSProvenance = "other" }, DispositionAggregateRejected, ReasonStructural},
			{"invalid ohlc", func(v *AggregateInput) { v.Values.Low = v.Values.High + 1 }, DispositionAggregateRejected, ReasonStructural},
			{"delivery evidence", func(v *AggregateInput) { v.DeliveryTime = time.Time{} }, DispositionAggregateRejected, ReasonDeliveryEvidence},
			{"live position", func(v *AggregateInput) { v.Live.FrameSequence = 0 }, DispositionAggregateRejected, ReasonSourcePosition},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := base
				tc.mutate(&input)
				applyAggregate(t, e, input, tc.code, tc.reason)
				assertNoAggregateState(t, e, "AAA")
			})
		}

		accepted := liveAggregate(binding, "AAA", start, 2, 2)
		applyAggregate(t, e, accepted, DispositionAggregateInserted, ReasonNone)
		stale := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 3)
		applyAggregate(t, e, stale, DispositionAggregateFenced, ReasonStaleLiveEpoch)
		assertCanonicalClose(t, e, "AAA", start, accepted.Values.Close)

		historical := historicalAggregate(binding, "AAA", start.Add(2*time.Second), 1)
		applyAggregate(t, e, historical, DispositionAggregateFenced, ReasonHistoricalContext)
		validProof := proofFor(binding, historical, start, start.Add(5*time.Second))
		wrongGeneration := validProof
		wrongGeneration.generation++
		applyHistorical(t, e, historical, wrongGeneration, DispositionAggregateFenced, ReasonHistoricalContext)
		wrongToken := validProof
		wrongToken.token = "stale-token"
		applyHistorical(t, e, historical, wrongToken, DispositionAggregateFenced, ReasonHistoricalContext)
		wrongSymbol := validProof
		wrongSymbol.symbol = "BAD"
		applyHistorical(t, e, historical, wrongSymbol, DispositionAggregateFenced, ReasonHistoricalContext)
		outsideInterval := validProof
		outsideInterval.intervalStart = historical.WindowEnd
		outsideInterval.intervalEnd = historical.WindowEnd.Add(time.Second)
		applyHistorical(t, e, historical, outsideInterval, DispositionAggregateFenced, ReasonHistoricalContext)
		closeAndWait(t, e)
	})

	t.Run("exact equality preserves first evidence and advances greatest live support", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		first := liveAggregate(binding, "AAA", start, 1, 1)
		first.Values.Volume = 0
		applyAggregate(t, e, first, DispositionAggregateInserted, ReasonNone)
		later := first
		later.Values.Volume = math.Copysign(0, -1)
		later.DeliveryTime = later.DeliveryTime.Add(time.Second)
		later.Live.FrameSequence = 4
		applyAggregate(t, e, later, DispositionAggregateExactDuplicate, ReasonNone)
		record := aggregateRecord(t, e, "AAA", start)
		if record.first.deliveryTime != first.DeliveryTime || record.authority.deliveryTime != later.DeliveryTime ||
			record.greatestLiveSupport == nil || *record.greatestLiveSupport != later.Live || !aggregateValuesEqual(record.values, first.Values) {
			t.Fatalf("duplicate evidence = %+v", record)
		}
		lowerEqual := later
		lowerEqual.Live.FrameSequence = 2
		lowerEqual.DeliveryTime = lowerEqual.DeliveryTime.Add(time.Second)
		applyAggregate(t, e, lowerEqual, DispositionAggregateExactDuplicate, ReasonNone)
		record = aggregateRecord(t, e, "AAA", start)
		if record.authority.deliveryTime != later.DeliveryTime || *record.greatestLiveSupport != later.Live {
			t.Fatal("lower equal evidence regressed live support")
		}
		closeAndWait(t, e)
	})

	t.Run("inclusive correction horizon future time and out of order insertion", func(t *testing.T) {
		window := start.Add(time.Minute)
		now := window.Add(time.Second).Add(correctionHorizon)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		first := liveAggregate(binding, "AAA", window, 1, 1)
		applyAggregate(t, e, first, DispositionAggregateInserted, ReasonNone)
		revision := first
		revision.Live.FrameSequence = 2
		revision.Values.Close, revision.Values.Low = 9.9, 9.9
		applyAggregate(t, e, revision, DispositionAggregateRevised, ReasonNone)
		if got := aggregateRecord(t, e, "AAA", window); got.values.Close != 9.9 || got.first.deliveryTime != first.DeliveryTime {
			t.Fatalf("inclusive revision = %+v", got)
		}
		now = now.Add(time.Nanosecond)
		lateRevision := revision
		lateRevision.Live.FrameSequence = 3
		lateRevision.Values.Close, lateRevision.Values.Low = 9.8, 9.8
		applyAggregate(t, e, lateRevision, DispositionAggregateRejected, ReasonTooLate)
		lateFirst := liveAggregate(binding, "BAD", window, 1, 1)
		applyAggregate(t, e, lateFirst, DispositionAggregateRejected, ReasonTooLate)
		closeAndWait(t, e)

		now = window
		futureEngine := aggregateEngine(t, binding, RunModeLive, &now)
		future := liveAggregate(binding, "BAD", window, 1, 2)
		applyAggregate(t, futureEngine, future, DispositionAggregateRejected, ReasonFutureEventTime)
		closeAndWait(t, futureEngine)

		now = start.Add(20 * time.Second)
		e = aggregateEngine(t, binding, RunModeLive, &now)
		newer := liveAggregate(binding, "BAD", start.Add(10*time.Second), 1, 3)
		older := liveAggregate(binding, "BAD", start.Add(5*time.Second), 1, 4)
		applyAggregate(t, e, newer, DispositionAggregateInserted, ReasonNone)
		applyAggregate(t, e, older, DispositionAggregateInserted, ReasonNone)
		assertCanonicalClose(t, e, "BAD", older.WindowStart, older.Values.Close)
		if state := aggregateState(t, e, "BAD"); state.latest == nil || state.latest.record.windowStart != newer.WindowStart {
			t.Fatalf("out-of-order identity regressed latest mark: %+v", state.latest)
		}
		closeAndWait(t, e)
	})

	t.Run("live and replay precedence reject lower and contain repeated contradictions", func(t *testing.T) {
		now := start.Add(5 * time.Second)
		liveEngine := aggregateEngine(t, binding, RunModeLive, &now)
		base := liveAggregate(binding, "AAA", start, 2, 2)
		base.Live.ArrayIndex = 1
		applyAggregate(t, liveEngine, base, DispositionAggregateInserted, ReasonNone)
		arrayGreater := changedClose(base, 10.1)
		arrayGreater.Live.ArrayIndex = 2
		applyAggregate(t, liveEngine, arrayGreater, DispositionAggregateRevised, ReasonNone)
		arrayLower := changedClose(arrayGreater, 10.05)
		arrayLower.Live.ArrayIndex = 0
		applyAggregate(t, liveEngine, arrayLower, DispositionAggregateRejected, ReasonNonprecedent)
		greater := changedClose(base, 10.2)
		greater.Live.FrameSequence = 3
		applyAggregate(t, liveEngine, greater, DispositionAggregateRevised, ReasonNone)
		lower := changedClose(greater, 10.1)
		lower.Live.FrameSequence = 1
		applyAggregate(t, liveEngine, lower, DispositionAggregateRejected, ReasonNonprecedent)
		epochGreater := changedClose(greater, 10.25)
		epochGreater.Live.ConnectionEpoch = 3
		epochGreater.Live.FrameSequence = 1
		applyAggregate(t, liveEngine, epochGreater, DispositionAggregateRevised, ReasonNone)
		staleEpoch := changedClose(epochGreater, 10.27)
		staleEpoch.Live.ConnectionEpoch = 2
		staleEpoch.Live.FrameSequence = 99
		applyAggregate(t, liveEngine, staleEpoch, DispositionAggregateFenced, ReasonStaleLiveEpoch)
		repeated := changedClose(epochGreater, 10.3)
		gotIntegrity := applyAggregate(t, liveEngine, repeated, DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal)
		if gotIntegrity.SuppressionDisposition != SuppressionCleanReinitializationRequired {
			t.Fatalf("live canonical suppression disposition = %q", gotIntegrity.SuppressionDisposition)
		}
		assertNoIdentity(t, liveEngine, "AAA", start)
		reentry := liveAggregate(binding, "AAA", start, 3, 2)
		if result, completion := liveEngine.admitAggregateForProof(reentry); result != AdmissionNotAdmittedClosed || completion != nil {
			t.Fatalf("live canonical failure left admission open: %s/%v", result, completion)
		}
		assertNoIdentity(t, liveEngine, "AAA", start)
		closeAndWait(t, liveEngine)

		replayEngine := aggregateEngine(t, binding, RunModeReplay, &now)
		rbase := replayAggregate(binding, "AAA", start, 2)
		applyAggregate(t, replayEngine, rbase, DispositionAggregateInserted, ReasonNone)
		rgreater := changedClose(rbase, 10.2)
		rgreater.Replay.RecordOrdinal = 3
		applyAggregate(t, replayEngine, rgreater, DispositionAggregateRevised, ReasonNone)
		rlower := changedClose(rgreater, 10.1)
		rlower.Replay.RecordOrdinal = 1
		applyAggregate(t, replayEngine, rlower, DispositionAggregateRejected, ReasonNonprecedent)
		rrepeat := changedClose(rgreater, 10.3)
		gotIntegrity = applyAggregate(t, replayEngine, rrepeat, DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal)
		if gotIntegrity.SuppressionDisposition != SuppressionTerminalReplayFailure {
			t.Fatalf("replay canonical suppression disposition = %q", gotIntegrity.SuppressionDisposition)
		}
		assertNoIdentity(t, replayEngine, "AAA", start)
		rreentry := replayAggregate(binding, "AAA", start, 4)
		if result, completion := replayEngine.admitAggregateForProof(rreentry); result != AdmissionNotAdmittedClosed || completion != nil {
			t.Fatalf("replay canonical failure left admission open: %s/%v", result, completion)
		}
		assertNoIdentity(t, replayEngine, "AAA", start)
		closeAndWait(t, replayEngine)

		artifactEngine := aggregateEngine(t, binding, RunModeReplay, &now)
		applyAggregate(t, artifactEngine, rbase, DispositionAggregateInserted, ReasonNone)
		otherArtifact := replayAggregate(binding, "BAD", start, 1)
		otherArtifact.Replay.ArtifactID = "artifact-b"
		applyAggregate(t, artifactEngine, otherArtifact, DispositionAggregateFenced, ReasonReplayArtifact)
		closeAndWait(t, artifactEngine)
	})

	t.Run("historical proof is fill only and conflicts have no arrival order winner", func(t *testing.T) {
		now := start.Add(30 * time.Minute)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		old := historicalAggregate(binding, "AAA", start.Add(time.Minute), 1)
		proof := proofFor(binding, old, start, start.Add(10*time.Minute))
		applyHistorical(t, e, old, proof, DispositionAggregateInserted, ReasonNone)
		state := aggregateState(t, e, "AAA")
		if len(state.tail) != 0 || state.presence == nil || !state.presence.has(sessionSlot(e.state.binding, old.WindowStart)) || state.latest == nil {
			t.Fatalf("old historical fill state = %+v", state)
		}
		conflict := changedClose(old, 10.4)
		conflict.Historical.RecordOrdinal = 2
		applyHistorical(t, e, conflict, proof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		state = aggregateState(t, e, "AAA")
		if state.latest != nil || state.presence.has(sessionSlot(e.state.binding, old.WindowStart)) || state.historicalConflict == nil || !state.historicalConflict.has(sessionSlot(e.state.binding, old.WindowStart)) || exactAggregateCoverage(state, e.state.binding, old.WindowStart, old.WindowEnd) {
			t.Fatalf("historical withdrawal = %+v", state)
		}
		if got, want := (len(*state.presence)+len(*state.historicalConflict))*8, 2*(sessionSeconds/8); got != want {
			t.Fatalf("two lazy session bitmaps bytes=%d want=%d", got, want)
		}

		firstOld := historicalAggregate(binding, "MISSING", start.Add(2*time.Minute), 1)
		multiProof := proofFor(binding, firstOld, start, start.Add(10*time.Minute))
		applyHistorical(t, e, firstOld, multiProof, DispositionAggregateInserted, ReasonNone)
		newerOld := historicalAggregate(binding, "MISSING", start.Add(3*time.Minute), 2)
		applyHistorical(t, e, newerOld, multiProof, DispositionAggregateInserted, ReasonNone)
		firstEqual := firstOld
		firstEqual.Historical.RecordOrdinal = 3
		before := aggregateAccountingSnapshot(e)
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		e.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		result, completion := e.admitHistoricalAggregateForProof(firstEqual, multiProof)
		if result != AdmissionAdmitted {
			t.Fatalf("immutable historical context admission = %s", result)
		}
		<-entered
		identity := aggregateIdentity{symbol: firstOld.Symbol, start: firstOld.WindowStart.Unix()}
		originalContextRecord := multiProof.result.records[identity]
		mutatedContextRecord := originalContextRecord
		mutatedContextRecord.values = changedClose(firstOld, 99).Values
		multiProof.result.records[identity] = mutatedContextRecord
		close(release)
		if got := awaitAggregateDisposition(t, completion); got.Code != DispositionAggregateExactDuplicate || got.Reason != ReasonNone {
			t.Fatalf("caller mutation changed admitted historical decision = %+v", got)
		}
		e.beforeConsume = nil
		multiProof.result.records[identity] = originalContextRecord
		assertAggregateBinDelta(t, before, aggregateAccountingSnapshot(e), DispositionAggregateExactDuplicate)
		firstConflict := changedClose(firstOld, 10.8)
		firstConflict.Historical.RecordOrdinal = 4
		applyHistorical(t, e, firstConflict, multiProof, DispositionAggregateWithdrawn, ReasonHistoricalHistoricalConflict)
		missingState := aggregateState(t, e, "MISSING")
		if missingState.presence.has(sessionSlot(e.state.binding, firstOld.WindowStart)) || !missingState.presence.has(sessionSlot(e.state.binding, newerOld.WindowStart)) {
			t.Fatal("multi-identity historical result selected an arrival-order winner")
		}

		live := liveAggregate(binding, "BAD", start.Add(20*time.Minute), 1, 10)
		applyAggregate(t, e, live, DispositionAggregateInserted, ReasonNone)
		equalHistorical := historicalAggregate(binding, "BAD", live.WindowStart, 1)
		equalHistorical.Values = live.Values
		proof = proofFor(binding, equalHistorical, live.WindowStart, live.WindowEnd)
		applyHistorical(t, e, equalHistorical, proof, DispositionAggregateExactDuplicate, ReasonNone)
		unequalHistorical := changedClose(equalHistorical, 10.7)
		unequalHistorical.Historical.RecordOrdinal = 2
		applyHistorical(t, e, unequalHistorical, proof, DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		if got := aggregateRecord(t, e, "BAD", live.WindowStart); got.values != live.Values || got.authority.source != AggregateSourceLive {
			t.Fatalf("historical overwrote live = %+v", got)
		}
		badState := aggregateState(t, e, "BAD")
		if !exactAggregateCoverage(badState, e.state.binding, live.WindowStart, live.WindowEnd) ||
			badState.historicalConflict != nil && badState.historicalConflict.has(sessionSlot(e.state.binding, live.WindowStart)) {
			t.Fatal("resolved historical/live discrepancy poisoned exact live coverage")
		}
		conflictEvidence := badState.lastConflict
		if conflictEvidence == nil || conflictEvidence.current.source != AggregateSourceLive || conflictEvidence.incoming.source != AggregateSourceHistorical {
			t.Fatalf("historical/live conflict provenance = %+v", conflictEvidence)
		}
		closeAndWait(t, e)

		now = start.Add(5 * time.Minute)
		interleaved := aggregateEngine(t, binding, RunModeLive, &now)
		row := historicalAggregate(binding, "AAA", start.Add(4*time.Minute), 1)
		rowProof := proofFor(binding, row, start.Add(4*time.Minute), start.Add(6*time.Minute))
		applyHistorical(t, interleaved, row, rowProof, DispositionAggregateInserted, ReasonNone)
		liveRevision := liveAggregate(binding, "AAA", row.WindowStart, 2, 1)
		liveRevision.Values = changedClose(liveRevision, 10.4).Values
		applyAggregate(t, interleaved, liveRevision, DispositionAggregateRevised, ReasonNone)
		now = start.Add(25 * time.Minute)
		trigger := liveAggregate(binding, "AAA", now.Add(-time.Second), 2, 2)
		applyAggregate(t, interleaved, trigger, DispositionAggregateInserted, ReasonNone)
		lateHistorical := changedClose(row, 10.8)
		lateHistorical.Historical.RecordOrdinal = 2
		applyHistorical(t, interleaved, lateHistorical, rowProof, DispositionAggregateRejected, ReasonHistoricalLiveConflict)
		interleavedState := aggregateState(t, interleaved, "AAA")
		if interleavedState.presence == nil || !interleavedState.presence.has(sessionSlot(interleaved.state.binding, row.WindowStart)) {
			t.Fatal("same-result historical conflict cleared compacted live presence")
		}
		if interleavedState.historicalConflict != nil && interleavedState.historicalConflict.has(sessionSlot(interleaved.state.binding, row.WindowStart)) {
			t.Fatal("resolved historical/live discrepancy installed a conflict bit over compacted live authority")
		}
		closeAndWait(t, interleaved)
	})

	t.Run("latest recomputation finalization compacted presence and exact bounds", func(t *testing.T) {
		now := start.Add(time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		older := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, e, older, DispositionAggregateInserted, ReasonNone)
		now = start.Add(correctionHorizon + 2*time.Second)
		latest := liveAggregate(binding, "AAA", start.Add(correctionHorizon), 1, 2)
		applyAggregate(t, e, latest, DispositionAggregateInserted, ReasonNone)
		corrected := changedClose(latest, 11.2)
		corrected.Live.FrameSequence = 3
		applyAggregate(t, e, corrected, DispositionAggregateRevised, ReasonNone)
		state := aggregateState(t, e, "AAA")
		if state.latest == nil || state.latest.record.values.Close != 11.2 {
			t.Fatal("latest correction was not installed")
		}
		contradiction := changedClose(corrected, 11.3)
		applyAggregate(t, e, contradiction, DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal)
		state = aggregateState(t, e, "AAA")
		if state.latest == nil || state.latest.record.identity.start != older.WindowStart.Unix() {
			t.Fatalf("latest withdrawal recomputation = %+v", state.latest)
		}
		closeAndWait(t, e)

		now = start.Add(time.Second)
		finalized := aggregateEngine(t, binding, RunModeLive, &now)
		applyAggregate(t, finalized, older, DispositionAggregateInserted, ReasonNone)
		now = start.Add(correctionHorizon + 3*time.Second)
		trigger := liveAggregate(binding, "AAA", now.Add(-time.Second), 1, 4)
		applyAggregate(t, finalized, trigger, DispositionAggregateInserted, ReasonNone)
		state = aggregateState(t, finalized, "AAA")
		if state.presence == nil || !state.presence.has(0) || historicalRegistrationAllowed(t, finalized, "AAA", start, start.Add(time.Second)) {
			t.Fatal("compacted finalized presence was relabeled missing")
		}
		closeAndWait(t, finalized)

		now = start.Add(time.Second)
		long := aggregateEngine(t, binding, RunModeLive, &now)
		frame := uint64(1)
		for second := 0; second < 1_200; second++ {
			window := start.Add(time.Duration(second) * time.Second)
			now = window.Add(time.Second)
			input := liveAggregate(binding, "AAA", window, 1, frame)
			frame++
			input.Values.Close = 10 + float64(second%10)/100
			input.Values.High = input.Values.Close
			applyAggregate(t, long, input, DispositionAggregateInserted, ReasonNone)
			current := input
			if second%100 == 0 {
				current = changedClose(input, input.Values.Close+0.01)
				current.Live.FrameSequence = frame
				frame++
				applyAggregate(t, long, current, DispositionAggregateRevised, ReasonNone)
			}
			if second%137 == 0 {
				duplicate := current
				duplicate.Live.FrameSequence = frame
				frame++
				duplicate.DeliveryTime = duplicate.DeliveryTime.Add(time.Nanosecond)
				applyAggregate(t, long, duplicate, DispositionAggregateExactDuplicate, ReasonNone)
			}
		}
		state = aggregateState(t, long, "AAA")
		if len(state.tail) != maximumTailRecords || state.presence == nil || state.latest == nil || state.latest.record.windowStart != start.Add(1199*time.Second) {
			t.Fatalf("long trace tail=%d presence=%v latest=%v", len(state.tail), state.presence != nil, state.latest)
		}
		if state.olderLatest == nil || state.olderLatest.windowStart != start.Add(238*time.Second) {
			t.Fatalf("older latest mark = %+v", state.olderLatest)
		}
		if got, want := len(*state.presence)*8, sessionSeconds/8; got != want {
			t.Fatalf("presence bytes=%d want=%d", got, want)
		}
		if got, want := 2*len(slotBitmap{})*8, 2*(sessionSeconds/8); got != want {
			t.Fatalf("charged presence+historical-conflict ceiling=%d want=%d", got, want)
		}
		closeAndWait(t, long)
	})

	t.Run("aggregate path preserves sequence exhaustion and S1 closure", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		e.mu.Lock()
		e.lastReserved = math.MaxUint64 - 2
		e.nextSequence = math.MaxUint64 - 1
		e.mu.Unlock()
		result, completion := e.admitAggregateForProof(liveAggregate(binding, "AAA", start, 1, 1))
		if result != AdmissionAdmitted {
			t.Fatalf("last aggregate admission = %s", result)
		}
		result, extra := e.admitAggregateForProof(liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2))
		if result != AdmissionSequenceBudgetExhausted || extra != nil {
			t.Fatalf("exhaustion = %s", result)
		}
		got := awaitAggregateDisposition(t, completion)
		if got.EngineSequence != math.MaxUint64-1 || got.Code != DispositionAggregateInserted {
			t.Fatalf("last aggregate = %+v", got)
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		e.mu.Lock()
		terminal := e.terminal
		e.mu.Unlock()
		if terminal == nil || terminal.EngineSequence != math.MaxUint64 {
			t.Fatalf("terminal = %+v", terminal)
		}
		assertAggregateAccounting(t, e)
	})
}

func aggregateEngine(t *testing.T, binding reference.Binding, mode RunMode, now *time.Time) *Engine {
	t.Helper()
	delay := time.Duration(0)
	e, err := New(Config{Mode: mode, Clock: func() time.Time { return *now }, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}
	return e
}

func liveAggregate(binding reference.Binding, symbol string, window time.Time, epoch, frame uint64) AggregateInput {
	return AggregateInput{
		SchemaVersion: AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: AggregateSourceLive,
		Symbol: symbol, WindowStart: window.UTC(), WindowEnd: window.Add(time.Second).UTC(),
		Values:       AggregateValues{Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage},
		DeliveryTime: window.Add(time.Second).UTC(), Live: LivePosition{ConnectionEpoch: epoch, FrameSequence: frame},
	}
}

func replayAggregate(binding reference.Binding, symbol string, window time.Time, ordinal uint64) AggregateInput {
	input := liveAggregate(binding, symbol, window, 1, 1)
	input.Source, input.Live = AggregateSourceReplay, LivePosition{}
	input.Replay = ReplayPosition{ArtifactID: "artifact-a", RecordOrdinal: ordinal}
	input.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
	return input
}

func historicalAggregate(binding reference.Binding, symbol string, window time.Time, ordinal uint64) AggregateInput {
	input := liveAggregate(binding, symbol, window, 1, 1)
	input.Source, input.Live = AggregateSourceHistorical, LivePosition{}
	input.Historical = HistoricalPosition{Generation: 7, RequestToken: "token-a", RecordOrdinal: ordinal}
	input.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
	return input
}

func proofFor(binding reference.Binding, input AggregateInput, start, end time.Time) historicalProofContext {
	return historicalProofContext{
		bindingID: binding.Identity(), generation: input.Historical.Generation, token: input.Historical.RequestToken,
		symbol: input.Symbol, intervalStart: start.UTC(), intervalEnd: end.UTC(),
		result: &historicalProofResult{records: make(map[aggregateIdentity]canonicalAggregate), conflicts: make(map[aggregateIdentity]struct{})},
	}
}

func changedClose(input AggregateInput, close float64) AggregateInput {
	input.Values.Close = close
	if close > input.Values.High {
		input.Values.High = close
	}
	if close < input.Values.Low {
		input.Values.Low = close
	}
	return input
}

func applyAggregate(t *testing.T, e *Engine, input AggregateInput, code DispositionCode, reason DispositionReason) AggregateDisposition {
	t.Helper()
	before := aggregateAccountingSnapshot(e)
	beforeRecomputations := aggregateRecomputations(e)
	beforeIntegrity := aggregateIntegritySnapshot(e)
	result, completion := e.admitAggregateForProof(input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("admission = %s", result)
	}
	got := awaitAggregateDisposition(t, completion)
	if got.Code != code || got.Reason != reason {
		t.Fatalf("disposition = %+v, want %s/%s", got, code, reason)
	}
	assertAggregateAccounting(t, e)
	assertAggregateBinDelta(t, before, aggregateAccountingSnapshot(e), code)
	assertRecomputationDelta(t, beforeRecomputations, aggregateRecomputations(e), code, beforeIntegrity)
	return got
}

func applyHistorical(t *testing.T, e *Engine, input AggregateInput, proof historicalProofContext, code DispositionCode, reason DispositionReason) AggregateDisposition {
	t.Helper()
	before := aggregateAccountingSnapshot(e)
	beforeRecomputations := aggregateRecomputations(e)
	beforeIntegrity := aggregateIntegritySnapshot(e)
	result, completion := e.admitHistoricalAggregateForProof(input, proof)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("historical admission = %s", result)
	}
	got := awaitAggregateDisposition(t, completion)
	if got.Code != code || got.Reason != reason {
		t.Fatalf("historical disposition = %+v, want %s/%s", got, code, reason)
	}
	advanceHistoricalProofResult(proof.result, input, got.Code)
	assertAggregateAccounting(t, e)
	assertAggregateBinDelta(t, before, aggregateAccountingSnapshot(e), code)
	assertRecomputationDelta(t, beforeRecomputations, aggregateRecomputations(e), code, beforeIntegrity)
	return got
}

func advanceHistoricalProofResult(result *historicalProofResult, input AggregateInput, code DispositionCode) {
	if result == nil {
		return
	}
	identity := aggregateIdentity{symbol: input.Symbol, start: input.WindowStart.Unix()}
	switch code {
	case DispositionAggregateInserted, DispositionAggregateRevised:
		result.records[identity] = canonicalAggregate{
			identity: identity, windowStart: input.WindowStart, windowEnd: input.WindowEnd,
			values: input.Values, first: aggregateEvidence{source: input.Source, deliveryTime: input.DeliveryTime, historical: input.Historical},
			authority: aggregateEvidence{source: input.Source, deliveryTime: input.DeliveryTime, historical: input.Historical},
		}
	case DispositionAggregateWithdrawn:
		delete(result.records, identity)
		result.conflicts[identity] = struct{}{}
	}
}

func awaitAggregateDisposition(t *testing.T, completion <-chan AggregateDisposition) AggregateDisposition {
	t.Helper()
	select {
	case got, ok := <-completion:
		if !ok {
			t.Fatal("aggregate completion closed without a disposition")
		}
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for aggregate disposition")
		return AggregateDisposition{}
	}
}

func assertAggregateAccounting(t *testing.T, e *Engine) {
	t.Helper()
	e.mu.Lock()
	accounting := e.state.aggregates
	e.mu.Unlock()
	if !accounting.reconciles() {
		t.Fatalf("aggregate accounting does not reconcile: %+v", accounting)
	}
}

func aggregateAccountingSnapshot(e *Engine) aggregateAccounting {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state.aggregates
}

func assertAggregateBinDelta(t *testing.T, before, after aggregateAccounting, code DispositionCode) {
	t.Helper()
	delta := aggregateAccounting{
		consumed: after.consumed - before.consumed, inserted: after.inserted - before.inserted,
		revised: after.revised - before.revised, withdrawnConflict: after.withdrawnConflict - before.withdrawnConflict,
		exactDuplicate: after.exactDuplicate - before.exactDuplicate, rejected: after.rejected - before.rejected,
		fenced: after.fenced - before.fenced, integrity: after.integrity - before.integrity,
	}
	want := aggregateAccounting{consumed: 1}
	switch code {
	case DispositionAggregateInserted:
		want.inserted = 1
	case DispositionAggregateRevised:
		want.revised = 1
	case DispositionAggregateWithdrawn:
		want.withdrawnConflict = 1
	case DispositionAggregateExactDuplicate:
		want.exactDuplicate = 1
	case DispositionAggregateRejected:
		want.rejected = 1
	case DispositionAggregateFenced:
		want.fenced = 1
	case DispositionAggregateIntegrity:
		want.integrity = 1
	default:
		t.Fatalf("non-aggregate code %s", code)
	}
	if delta != want {
		t.Fatalf("accounting delta = %+v, want %+v", delta, want)
	}
}

func aggregateRecomputations(e *Engine) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.binding == nil {
		return 0
	}
	var total uint64
	for index := range e.state.binding.symbols {
		if state := e.state.binding.symbols[index].aggregates; state != nil {
			total += state.recomputations
		}
	}
	return total
}

func assertRecomputationDelta(t *testing.T, before, after uint64, code DispositionCode, integrityAlreadySet bool) {
	t.Helper()
	want := uint64(0)
	if code == DispositionAggregateInserted || code == DispositionAggregateRevised || code == DispositionAggregateWithdrawn || (code == DispositionAggregateIntegrity && !integrityAlreadySet) {
		want = 1
	}
	if after-before != want {
		t.Fatalf("recomputation delta=%d want=%d for %s", after-before, want, code)
	}
}

func TestExactAggregateCoverageWordRanges(t *testing.T) {
	binding := testBinding(t)
	installed := installedBindingForQualification(t, binding)
	start := installed.sessionStart.Add(3 * time.Second)
	end := start.Add(130 * time.Second)
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate)}
	presentAt := start.Add(61 * time.Second)
	tailAt := start.Add(64 * time.Second)
	conflictAt := start.Add(65 * time.Second)
	ensurePresence(state).set(sessionSlot(installed, presentAt))
	state.tail[tailAt.Unix()] = qualificationRecord("AAA", tailAt, 10, 100, 10, 10, ATSLiveProviderAverage)
	ensureHistoricalConflict(state).set(sessionSlot(installed, conflictAt))

	if !installExactCoverage(state, installed, start, end, nil) {
		t.Fatal("non-word-aligned coverage installation failed")
	}
	if state.provenAbsent.has(sessionSlot(installed, presentAt)) || state.provenAbsent.has(sessionSlot(installed, tailAt)) ||
		state.provenAbsent.has(sessionSlot(installed, conflictAt)) || state.provenAbsent.has(sessionSlot(installed, start.Add(-time.Second))) ||
		state.provenAbsent.has(sessionSlot(installed, end)) {
		t.Fatal("coverage range fabricated absence over present, conflicting, or out-of-range slots")
	}
	if exactAggregateCoverage(state, installed, start, end) {
		t.Fatal("historical conflict reached exact coverage")
	}
	state.historicalConflict.clear(sessionSlot(installed, conflictAt))
	if exactAggregateCoverage(state, installed, start, end) {
		t.Fatal("cleared conflict became covered without new evidence")
	}
	if !installExactCoverage(state, installed, start, end, nil) || !exactAggregateCoverage(state, installed, start, end) {
		t.Fatal("word-range absence plus canonical presence did not establish exact coverage")
	}
}

func aggregateIntegritySnapshot(e *Engine) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state.aggregateIntegrity
}

func aggregateState(t *testing.T, e *Engine, symbol string) *symbolAggregateState {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index, ok := e.state.binding.index[symbol]
	if !ok || e.state.binding.symbols[index].aggregates == nil {
		t.Fatalf("no aggregate state for %s", symbol)
	}
	return e.state.binding.symbols[index].aggregates
}

func aggregateRecord(t *testing.T, e *Engine, symbol string, window time.Time) canonicalAggregate {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	state := e.state.binding.symbols[index].aggregates
	if state != nil && state.tail != nil && state.tail[window.Unix()] != nil {
		return *state.tail[window.Unix()]
	}
	if state != nil && state.latest != nil && state.latest.record.identity.start == window.Unix() {
		return state.latest.record
	}
	t.Fatalf("no aggregate record for %s/%s", symbol, window)
	return canonicalAggregate{}
}

func assertCanonicalClose(t *testing.T, e *Engine, symbol string, window time.Time, close float64) {
	t.Helper()
	if got := aggregateRecord(t, e, symbol, window).values.Close; got != close {
		t.Fatalf("close=%v want=%v", got, close)
	}
}

func assertNoAggregateState(t *testing.T, e *Engine, symbol string) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	if e.state.binding.symbols[index].aggregates != nil {
		t.Fatalf("rejected input allocated aggregate state for %s", symbol)
	}
}

func assertNoIdentity(t *testing.T, e *Engine, symbol string, window time.Time) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.state.binding.symbols[e.state.binding.index[symbol]].aggregates
	if state != nil && state.tail[window.Unix()] != nil {
		t.Fatalf("identity remained in tail")
	}
	if state != nil && state.latest != nil && state.latest.record.identity.start == window.Unix() {
		t.Fatalf("identity remained latest")
	}
}

func historicalRegistrationAllowed(t *testing.T, e *Engine, symbol string, start, end time.Time) bool {
	t.Helper()
	return e.historicalRegistrationAllowedForProof(symbol, start, end)
}
