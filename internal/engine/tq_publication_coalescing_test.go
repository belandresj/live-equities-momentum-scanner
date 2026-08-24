package engine

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type tqPublicationFixture struct {
	e       *Engine
	binding interface {
		Identity() string
		TradingDate() string
		SessionStart() time.Time
		SessionEnd() time.Time
	}
	now   time.Time
	clock *time.Time
}

func newTQPublicationFixture(t *testing.T, symbols []string) tqPublicationFixture {
	t.Helper()
	binding := hydrationPopulationBinding(t, symbols)
	now := binding.SessionStart().Add(20 * time.Minute)
	nowPointer := now
	e := aggregateEngine(t, binding, RunModeLive, &nowPointer)
	t.Cleanup(func() { closeAndWait(t, e) })

	e.mu.Lock()
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive = lifecycleLive, 1, true
	e.state.aggregateAcknowledged = true
	e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.committedT = immutableTime(now)
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(binding.SessionEnd())
	e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence, len(symbols))
	for index := range e.state.binding.symbols {
		installEvaluatorMarkOnSymbol(&e.state.binding.symbols[index], now, 12+float64(index), qualificationFinalized)
		if !installExactCoverage(e.state.binding.symbols[index].aggregates, e.state.binding, e.state.binding.sessionStart, binding.SessionEnd(), nil) {
			e.mu.Unlock()
			t.Fatalf("exact coverage setup failed for %s", symbols[index])
		}
	}
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(now, now)
	e.state.aggregateProjectionPending = true
	e.mu.Unlock()

	admission, completion := e.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("initial aggregate publication did not complete")
	}
	command := issueTQForTest(t, e)
	if got := admitTQResultForTest(t, e, tqResultForTest(t, command, LivePosition{ConnectionEpoch: 1, FrameSequence: 10}, now, ControlSucceeded)); got.Code != DispositionTQApplied {
		t.Fatalf("subscription write result = %+v", got)
	}

	return tqPublicationFixture{e: e, binding: binding, now: now, clock: &nowPointer}
}

func (f tqPublicationFixture) confirmChannels(t *testing.T) {
	t.Helper()
	for index, symbol := range f.e.ObserveTQ().Desired {
		frame := uint64(11 + index*2)
		beforeTrade := f.e.ObserveSnapshot().Publication.PublicationID
		trade := baseTrade(f.binding, f.now.Add(-100*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: frame})
		trade.Symbol, trade.TradeID = symbol, fmt.Sprintf("confirmation-trade-%d", index)
		if got := admitTradeForTest(t, f.e, trade); got.Code != DispositionTQApplied {
			t.Fatalf("trade confirmation %s = %+v", symbol, got)
		}
		if got := f.e.ObserveSnapshot().Publication.PublicationID; got != beforeTrade+1 {
			t.Fatalf("first trade confirmation %s publication=%d want=%d", symbol, got, beforeTrade+1)
		}
		quote := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: f.binding.Identity(), TradingDate: f.binding.TradingDate(), Symbol: symbol,
			SIPTime: f.now.Add(-50 * time.Millisecond), ReceiptTime: f.now.Add(-50 * time.Millisecond), BidPrice: 10, AskPrice: 10.02,
			BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
			Live: LivePosition{ConnectionEpoch: 1, FrameSequence: frame + 1}}
		if got := admitQuoteForTest(t, f.e, quote); got.Code != DispositionTQApplied {
			t.Fatalf("quote confirmation %s = %+v", symbol, got)
		}
		if got := f.e.ObserveSnapshot().Publication.PublicationID; got != beforeTrade+2 {
			t.Fatalf("first quote confirmation %s publication=%d want=%d", symbol, got, beforeTrade+2)
		}
	}
}

func (f tqPublicationFixture) ordinaryTrade(t *testing.T, id string, frame uint64) TradeInput {
	t.Helper()
	trade := baseTrade(f.binding, f.now.Add(-200*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: frame})
	trade.TradeID = id
	return trade
}

func (f tqPublicationFixture) ordinaryQuote(frame uint64) QuoteInput {
	return QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: f.binding.Identity(), TradingDate: f.binding.TradingDate(), Symbol: f.e.ObserveTQ().Desired[0],
		SIPTime: f.now.Add(-150 * time.Millisecond), ReceiptTime: f.now.Add(-150 * time.Millisecond), BidPrice: 10, AskPrice: 10.03,
		BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
		Live: LivePosition{ConnectionEpoch: 1, FrameSequence: frame}}
}

// TestTQPublicationCoalescingDirtyCadence proves that canonical T/Q facts are
// completed synchronously while ordinary changes wait for the next existing
// aggregate publication boundary. The snapshot remains one immutable
// aggregate-and-T/Q cell throughout the dirty interval.
func TestTQPublicationCoalescingDirtyCadence(t *testing.T) {
	fixture := newTQPublicationFixture(t, []string{"AAA"})
	fixture.confirmChannels(t)

	before := fixture.e.ObserveSnapshot()
	beforeAccounting := fixture.e.ObserveTQPublicationAccounting()
	beforeCurrent := fixture.e.ObserveTQ()

	trade := fixture.ordinaryTrade(t, "ordinary-trade", 100)
	if got := admitTradeForTest(t, fixture.e, trade); got.Code != DispositionTQApplied {
		t.Fatalf("ordinary trade = %+v", got)
	}
	duplicate := trade
	duplicate.Live.FrameSequence = 101
	if got := admitTradeForTest(t, fixture.e, duplicate); got.Code != DispositionTQDuplicate {
		t.Fatalf("duplicate trade = %+v", got)
	}
	unequal := trade
	unequal.Live.FrameSequence = 102
	unequal.Price = 10.02
	if got := admitTradeForTest(t, fixture.e, unequal); got.Code != DispositionTQRejected {
		t.Fatalf("unequal repeat = %+v", got)
	}
	if got := admitQuoteForTest(t, fixture.e, fixture.ordinaryQuote(103)); got.Code != DispositionTQApplied {
		t.Fatalf("ordinary quote = %+v", got)
	}

	afterFacts := fixture.e.ObserveSnapshot()
	current := fixture.e.ObserveTQ()
	accounting := fixture.e.ObserveTQPublicationAccounting()
	if afterFacts.Publication.PublicationID != before.Publication.PublicationID ||
		afterFacts.TQ.Accounting != before.TQ.Accounting ||
		current.Accounting.Consumed != beforeCurrent.Accounting.Consumed+4 ||
		current.Rows[0].Tape.Status != TQInvalid || current.Rows[0].Tape.Reason != "unequal_repeat" {
		t.Fatalf("ordinary facts replaced or escaped immutable publication: before=%+v after=%+v current=%+v", before, afterFacts, current)
	}
	if accounting.CanonicalMutations != beforeAccounting.CanonicalMutations+4 ||
		accounting.ProjectionDirtied != beforeAccounting.ProjectionDirtied+4 ||
		accounting.CoalescedOrdinaryMutations != beforeAccounting.CoalescedOrdinaryMutations+4 ||
		accounting.CanonicalMutations != accounting.ImmediateTrustTransitionMutations+accounting.CoalescedOrdinaryMutations ||
		!accounting.ProjectionDirty || accounting.PendingProjectionMutations != 4 {
		t.Fatalf("ordinary mutation accounting = before=%+v after=%+v", beforeAccounting, accounting)
	}

	maintenanceBefore := fixture.e.ObserveSnapshot().Publication.PublicationID
	if admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("maintenance timer did not complete")
	}
	maintenanceAfter := fixture.e.ObserveSnapshot().Publication.PublicationID
	maintenanceAccounting := fixture.e.ObserveTQPublicationAccounting()
	if maintenanceAfter != maintenanceBefore || !maintenanceAccounting.ProjectionDirty || maintenanceAccounting.PendingProjectionMutations != 5 {
		t.Fatalf("maintenance timer flushed dirty T/Q projection: publication=%d/%d accounting=%+v", maintenanceBefore, maintenanceAfter, maintenanceAccounting)
	}

	fixture.e.mu.Lock()
	fixture.now = fixture.now.Add(time.Second)
	fixture.e.state.aggregateProjectionPending = true
	fixture.e.mu.Unlock()
	if admission, completion := fixture.e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("cadence timer did not complete")
	}
	final := fixture.e.ObserveSnapshot()
	finalCurrent := fixture.e.ObserveTQ()
	finalAccounting := fixture.e.ObserveTQPublicationAccounting()
	if final.Publication.PublicationID != before.Publication.PublicationID+1 ||
		final.Publication.PublicationID != final.TQ.PublicationID ||
		!reflect.DeepEqual(final.TQ, finalCurrent) || final.TQ.Rows[0].Tape.Status != TQInvalid ||
		final.TQ.Rows[0].Tape.Reason != "unequal_repeat" || finalAccounting.ProjectionDirty || finalAccounting.PendingProjectionMutations != 0 ||
		finalAccounting.ProjectionFlushedByCadence != beforeAccounting.ProjectionFlushedByCadence+1 {
		t.Fatalf("cadence publication did not flush latest canonical T/Q state: snapshot=%+v current=%+v accounting=%+v", final, finalCurrent, finalAccounting)
	}
}

// TestTQPublicationCoalescingImmediateTrustTransitions proves that coverage
// loss and rank removal replace the combined publication once, while an
// unchanged maintenance input remains coalesced.
func TestTQPublicationCoalescingImmediateTrustTransitions(t *testing.T) {
	fixture := newTQPublicationFixture(t, []string{"AAA"})
	fixture.confirmChannels(t)

	beforeDrop := fixture.e.ObserveSnapshot()
	beforeAccounting := fixture.e.ObserveTQPublicationAccounting()
	drop := TQDropInput{SchemaVersion: TQSchemaV1, BindingIdentity: fixture.binding.Identity(), TradingDate: fixture.binding.TradingDate(), Family: "Q", Symbol: "AAA",
		DropReason: "symbol", Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 200}}
	admission, completion := fixture.e.AdmitTQDrop(context.Background(), drop)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("drop admission = %s", admission)
	}
	if got := <-completion; got.Code != DispositionTQRejected {
		t.Fatalf("drop = %+v", got)
	}
	afterDrop := fixture.e.ObserveSnapshot()
	dropAccounting := fixture.e.ObserveTQPublicationAccounting()
	if afterDrop.Publication.PublicationID != beforeDrop.Publication.PublicationID+1 || afterDrop.TQ.Rows[0].TradeCoverage || afterDrop.TQ.Rows[0].QuoteCoverage ||
		dropAccounting.ImmediateTrustTransitionPublications != beforeAccounting.ImmediateTrustTransitionPublications+1 {
		t.Fatalf("coverage close was not one immediate publication: before=%+v after=%+v accounting=%+v", beforeDrop, afterDrop, dropAccounting)
	}

	unchangedID := afterDrop.Publication.PublicationID
	if admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("unchanged maintenance timer did not complete")
	}
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != unchangedID {
		t.Fatalf("unchanged maintenance input replaced publication: got=%d want=%d", got, unchangedID)
	}
	repeatedDrop := drop
	repeatedDrop.Live.FrameSequence++
	if got := admitTQDropForCoalescing(t, fixture.e, repeatedDrop); got.Code != DispositionTQFenced {
		t.Fatalf("repeated coverage drop = %+v", got)
	}
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != unchangedID {
		t.Fatalf("repeated coverage drop replaced publication: got=%d want=%d", got, unchangedID)
	}

	fixture.e.mu.Lock()
	evaluation := &fixture.e.state.aggregateEvaluator.current
	evaluation.mode, evaluation.reason, evaluation.rows, evaluation.tqIntentAvailable = rankingUnavailable, rankingReasonNoTrustedMarks, nil, false
	evaluation.enrichedRows = 0
	fixture.e.mu.Unlock()
	beforeRemoval := fixture.e.ObserveSnapshot()
	if admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("rank-removal maintenance timer did not complete")
	}
	afterRemoval := fixture.e.ObserveSnapshot()
	removalAccounting := fixture.e.ObserveTQPublicationAccounting()
	if afterRemoval.Publication.PublicationID != beforeRemoval.Publication.PublicationID+1 || len(afterRemoval.TQ.Desired) != 0 ||
		removalAccounting.ImmediateTrustTransitionPublications != dropAccounting.ImmediateTrustTransitionPublications+1 {
		t.Fatalf("rank removal was not one immediate publication: before=%+v after=%+v accounting=%+v", beforeRemoval, afterRemoval, removalAccounting)
	}
	if admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("unchanged post-removal timer did not complete")
	}
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != afterRemoval.Publication.PublicationID {
		t.Fatalf("unchanged post-removal timer replaced publication: got=%d want=%d", got, afterRemoval.Publication.PublicationID)
	}
}

// TestTQPublicationCoalescingPostFenceDirtyState proves the step-1 ordering
// boundary directly: a successful live fence owns the cadence publication,
// ordinary post-fence T/Q facts remain dirty through the maintenance-only
// timer, and the following fence publishes the latest canonical T/Q cell once.
func TestTQPublicationCoalescingPostFenceDirtyState(t *testing.T) {
	fixture := newTQPublicationFixture(t, []string{"AAA"})
	fixture.confirmChannels(t)

	// The fixture starts with support through session end so its initial
	// publication can be installed without a live fence. Move the support
	// boundary back to the current committed T for this real fence sequence.
	fixture.e.mu.Lock()
	fixture.e.state.hydration.supportedThrough = immutableTime(fixture.now)
	fixture.e.mu.Unlock()
	applyLiveCoverageFenceForCoalescing(t, fixture.e, 200, 2, fixture.now)
	fencePublication := fixture.e.ObserveSnapshot()

	if got := admitTradeForTest(t, fixture.e, fixture.ordinaryTrade(t, "post-fence-trade", 300)); got.Code != DispositionTQApplied {
		t.Fatalf("post-fence trade = %+v", got)
	}
	quote := fixture.ordinaryQuote(301)
	if got := admitQuoteForTest(t, fixture.e, quote); got.Code != DispositionTQApplied {
		t.Fatalf("post-fence quote = %+v", got)
	}
	accounting := fixture.e.ObserveTQPublicationAccounting()
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != fencePublication.Publication.PublicationID ||
		!accounting.ProjectionDirty || accounting.PendingProjectionMutations != 2 {
		t.Fatalf("post-fence facts escaped cadence boundary: publication=%d/%d accounting=%+v", got, fencePublication.Publication.PublicationID, accounting)
	}

	if admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatal("post-fence maintenance timer did not complete")
	}
	maintenanceAccounting := fixture.e.ObserveTQPublicationAccounting()
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != fencePublication.Publication.PublicationID ||
		!maintenanceAccounting.ProjectionDirty || maintenanceAccounting.PendingProjectionMutations != 3 {
		t.Fatalf("maintenance timer consumed post-fence T/Q dirtiness: publication=%d/%d accounting=%+v", got, fencePublication.Publication.PublicationID, maintenanceAccounting)
	}

	next := fixture.now.Add(time.Second)
	*fixture.clock = next
	applyLiveCoverageFenceForCoalescing(t, fixture.e, 201, 3, next)
	secondFence := fixture.e.ObserveSnapshot()
	current := fixture.e.ObserveTQ()
	finalAccounting := fixture.e.ObserveTQPublicationAccounting()
	if secondFence.Publication.PublicationID != fencePublication.Publication.PublicationID+1 ||
		secondFence.Publication.PublicationID != secondFence.TQ.PublicationID || !reflect.DeepEqual(secondFence.TQ, current) ||
		finalAccounting.ProjectionDirty || finalAccounting.PendingProjectionMutations != 0 {
		t.Fatalf("next live fence did not publish latest T/Q once: first=%+v second=%+v current=%+v accounting=%+v", fencePublication, secondFence, current, finalAccounting)
	}

	closeInput := TQDropInput{SchemaVersion: TQSchemaV1, BindingIdentity: fixture.binding.Identity(), TradingDate: fixture.binding.TradingDate(), Family: "Q", Symbol: "AAA",
		DropReason: "symbol", Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 302}}
	if got := admitTQDropForCoalescing(t, fixture.e, closeInput); got.Code != DispositionTQRejected {
		t.Fatalf("post-fence coverage close = %+v", got)
	}
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != secondFence.Publication.PublicationID+1 {
		t.Fatalf("post-fence coverage close did not publish immediately: got=%d want=%d", got, secondFence.Publication.PublicationID+1)
	}
}

func assertTQPublicationOracle(t *testing.T, label string, actual, oracle tqPublicationFixture) {
	t.Helper()
	got := actual.e.ObserveSnapshot().TQ
	want := oracle.e.ObserveTQ()
	// The oracle is intentionally eager at the canonical observation boundary;
	// publication identity belongs to each engine's immutable cell, not to the
	// T/Q value comparison. Command selection scans an internal member map, so
	// its single pending symbol is engine-local nondeterministic detail.
	got.PublicationID, want.PublicationID = 0, 0
	got.PendingSymbol, want.PendingSymbol = "", ""
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s T/Q differs from eager canonical oracle: got=%+v want=%+v", label, got, want)
	}
}

func assertTQCanonicalOracle(t *testing.T, label string, actual, oracle tqPublicationFixture) {
	t.Helper()
	got := actual.e.ObserveTQ()
	want := oracle.e.ObserveTQ()
	got.PublicationID, want.PublicationID = 0, 0
	got.PendingSymbol, want.PendingSymbol = "", ""
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s canonical T/Q differs from eager oracle: got=%+v want=%+v", label, got, want)
	}
}

// TestTQPublicationCoalescingCadenceOracle compares the coalesced immutable
// T/Q cell with a second deterministic engine's eager canonical observation at
// each cadence boundary. The trace covers ordinary/duplicate/unequal trades,
// one-sided and crossed quotes, quiet covered time, stale aging, pressure
// samples, and generation fencing.
func TestTQPublicationCoalescingCadenceOracle(t *testing.T) {
	actual := newTQPublicationFixture(t, []string{"AAA", "BBB"})
	oracle := newTQPublicationFixture(t, []string{"AAA", "BBB"})
	actual.confirmChannels(t)
	oracle.confirmChannels(t)

	applyBoth := func(label string, apply func(t *testing.T, fixture tqPublicationFixture) Disposition) {
		t.Helper()
		actualDisposition := apply(t, actual)
		oracleDisposition := apply(t, oracle)
		if actualDisposition != oracleDisposition {
			t.Fatalf("%s disposition actual=%+v oracle=%+v", label, actualDisposition, oracleDisposition)
		}
		assertTQCanonicalOracle(t, label, actual, oracle)
	}

	applyBoth("qualifying trade", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		return admitTradeForTest(t, fixture.e, fixture.ordinaryTrade(t, "oracle-trade", 400))
	})
	applyBoth("duplicate trade", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		trade := fixture.ordinaryTrade(t, "oracle-trade", 401)
		return admitTradeForTest(t, fixture.e, trade)
	})
	applyBoth("unequal repeat", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		trade := fixture.ordinaryTrade(t, "oracle-trade", 402)
		trade.Price = 10.01
		return admitTradeForTest(t, fixture.e, trade)
	})
	applyBoth("one-sided quote", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		quote := fixture.ordinaryQuote(403)
		quote.AskPresent, quote.AskPrice = false, 0
		return admitQuoteForTest(t, fixture.e, quote)
	})
	applyBoth("crossed quote", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		quote := fixture.ordinaryQuote(404)
		quote.BidPrice, quote.AskPrice = 10.05, 10.04
		return admitQuoteForTest(t, fixture.e, quote)
	})
	applyBoth("generation-fenced trade", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		trade := fixture.ordinaryTrade(t, "fenced-trade", 405)
		trade.Live.ConnectionEpoch = 2
		return admitTradeForTest(t, fixture.e, trade)
	})
	applyBoth("healthy pressure sample", func(t *testing.T, fixture tqPublicationFixture) Disposition {
		applyFixturePressureSample(t, fixture, healthyTQPressureSample())
		return Disposition{Code: DispositionTQApplied, Reason: ReasonNone}
	})

	for boundary, target := range []time.Time{actual.now, actual.now.Add(6 * time.Second)} {
		*actual.clock, *oracle.clock = target, target
		for _, fixture := range []*tqPublicationFixture{&actual, &oracle} {
			fixture.e.mu.Lock()
			fixture.e.state.aggregateProjectionPending = true
			fixture.e.mu.Unlock()
			admission, completion := fixture.e.AdmitTimer(context.Background())
			if admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
				t.Fatalf("oracle cadence boundary %d did not complete", boundary)
			}
		}
		assertTQPublicationOracle(t, fmt.Sprintf("cadence boundary %d", boundary), actual, oracle)
		if got := actual.e.ObserveSnapshot().Publication.PublicationID; got != oracle.e.ObserveSnapshot().Publication.PublicationID {
			t.Fatalf("cadence boundary %d publication IDs diverged actual=%d oracle=%d", boundary, got, oracle.e.ObserveSnapshot().Publication.PublicationID)
		}
	}

	// Force the same deterministic rank-removal consequence in both engines;
	// reconciliation must publish the changed desired set once while retaining
	// byte-for-byte agreement with the eager canonical observation.
	for _, fixture := range []*tqPublicationFixture{&actual, &oracle} {
		fixture.e.mu.Lock()
		evaluation := &fixture.e.state.aggregateEvaluator.current
		evaluation.mode, evaluation.reason, evaluation.rows, evaluation.tqIntentAvailable = rankingUnavailable, rankingReasonNoTrustedMarks, nil, false
		evaluation.enrichedRows = 0
		fixture.e.mu.Unlock()
		admission, completion := fixture.e.AdmitMaintenanceTimer(context.Background())
		if admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
			t.Fatal("rank-churn maintenance timer did not complete")
		}
	}
	assertTQPublicationOracle(t, "rank churn", actual, oracle)
}

func admitTQDropForCoalescing(t *testing.T, e *Engine, input TQDropInput) Disposition {
	t.Helper()
	admission, completion := e.AdmitTQDrop(context.Background(), input)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("T/Q drop admission = %s", admission)
	}
	return awaitDisposition(t, completion)
}

func TestTQPublicationCoalescingFamilyDropAndQuarantineRecovery(t *testing.T) {
	t.Run("family drop is edge triggered", func(t *testing.T) {
		fixture := newTQPublicationFixture(t, []string{"AAA", "BBB"})
		fixture.confirmChannels(t)
		before := fixture.e.ObserveSnapshot()
		beforeAccounting := fixture.e.ObserveTQPublicationAccounting()
		drop := TQDropInput{SchemaVersion: TQSchemaV1, BindingIdentity: fixture.binding.Identity(), TradingDate: fixture.binding.TradingDate(), Family: "Q",
			DropReason: "optional_tq_shed", Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 300}}
		if got := admitTQDropForCoalescing(t, fixture.e, drop); got.Code != DispositionTQRejected || got.Reason != ReasonPressure {
			t.Fatalf("family drop = %+v", got)
		}
		afterFirst := fixture.e.ObserveSnapshot()
		if afterFirst.Publication.PublicationID != before.Publication.PublicationID+1 {
			t.Fatalf("first family drop publication=%d want=%d", afterFirst.Publication.PublicationID, before.Publication.PublicationID+1)
		}
		drop.Live.FrameSequence++
		if got := admitTQDropForCoalescing(t, fixture.e, drop); got.Code != DispositionTQRejected || got.Reason != ReasonPressure {
			t.Fatalf("repeated family drop = %+v", got)
		}
		afterSecond := fixture.e.ObserveSnapshot()
		accounting := fixture.e.ObserveTQPublicationAccounting()
		if afterSecond.Publication.PublicationID != afterFirst.Publication.PublicationID ||
			accounting.ImmediateTrustTransitionPublications != beforeAccounting.ImmediateTrustTransitionPublications+1 {
			t.Fatalf("repeated family drop republished unchanged trust state: first=%+v second=%+v accounting=%+v", afterFirst, afterSecond, accounting)
		}
	})

	t.Run("quarantine recovery is accounted as an immediate transition", func(t *testing.T) {
		fixture := newTQPublicationFixture(t, []string{"AAA"})
		fixture.confirmChannels(t)
		before := fixture.e.ObserveTQPublicationAccounting()
		quarantine := TQControlQuarantineInput{SchemaVersion: TQSchemaV1, BindingIdentity: fixture.binding.Identity(), ConnectionEpoch: 1,
			Position: LivePosition{ConnectionEpoch: 1, FrameSequence: 400}, ReceiptTime: fixture.now, Failure: TQControlStatusExtra}
		admission, completion := fixture.e.AdmitTQControlQuarantine(context.Background(), quarantine)
		if admission != AdmissionAdmitted || completion == nil {
			t.Fatalf("quarantine admission = %s", admission)
		}
		if got := awaitDisposition(t, completion); got.Code != DispositionTQRejected {
			t.Fatalf("quarantine = %+v", got)
		}
		afterQuarantine := fixture.e.ObserveTQPublicationAccounting()
		if afterQuarantine.ImmediateTrustTransitionMutations != before.ImmediateTrustTransitionMutations+1 {
			t.Fatalf("quarantine transition accounting=%+v before=%+v", afterQuarantine, before)
		}

		if got := admitConnectionControl(t, fixture.e, controlFact(fixture.binding.Identity(), ConnectionLost, 1,
			LivePosition{ConnectionEpoch: 1, FrameSequence: 401}, fixture.now, 0, ControlFailed)); got.Code != DispositionConnectionControlApplied {
			t.Fatalf("connection loss = %+v", got)
		}
		for _, fact := range []ConnectionControlInput{
			controlFact(fixture.binding.Identity(), ConnectionAttempt, 2, LivePosition{}, fixture.now, 20, ControlSucceeded),
			controlFact(fixture.binding.Identity(), ConnectionEstablished, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 1}, fixture.now, 20, ControlSucceeded),
			controlFact(fixture.binding.Identity(), AuthenticationResult, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 2}, fixture.now, 20, ControlSucceeded),
			controlFact(fixture.binding.Identity(), AggregateCommandWriteResult, 2, LivePosition{}, fixture.now, 21, ControlSucceeded),
			controlFact(fixture.binding.Identity(), AggregateSubscriptionResult, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 3}, fixture.now, 21, ControlSucceeded),
		} {
			if got := admitConnectionControl(t, fixture.e, fact); got.Code != DispositionConnectionControlApplied {
				t.Fatalf("epoch recovery %s = %+v", fact.Kind, got)
			}
		}
		afterRecovery := fixture.e.ObserveTQPublicationAccounting()
		if afterRecovery.ImmediateTrustTransitionMutations < afterQuarantine.ImmediateTrustTransitionMutations+2 ||
			afterRecovery.CanonicalMutations != afterRecovery.ImmediateTrustTransitionMutations+afterRecovery.CoalescedOrdinaryMutations {
			t.Fatalf("quarantine recovery transition was not accounted: before=%+v after=%+v", afterQuarantine, afterRecovery)
		}
		if view := fixture.e.ObserveTQ(); view.Quarantined {
			t.Fatalf("quarantine did not clear on greater acknowledged epoch: %+v", view)
		}
	})
}

// TestTQPublicationCoalescingP1HighRate is the explicit production-shape
// offline load proof. It is intentionally excluded from ordinary short test
// runs; invoke it without -short under the bounded acceptance timeout.
func TestTQPublicationCoalescingP1HighRate(t *testing.T) {
	if testing.Short() {
		t.Skip("P1 production-shape load is an explicit offline acceptance proof")
	}
	symbols := make([]string, 20)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%02d", index)
	}
	fixture := newTQPublicationFixture(t, symbols)
	fixture.confirmChannels(t)
	baseline := fixture.e.ObserveSnapshot()
	baselineAccounting := fixture.e.ObserveTQPublicationAccounting()
	baselineTQ := fixture.e.ObserveTQ()

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	const facts = 100_000
	for index := 0; index < facts/2; index++ {
		symbol := symbols[index%len(symbols)]
		frame := uint64(1_000 + index)
		trade := baseTrade(fixture.binding, fixture.now.Add(-100*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: frame})
		trade.Symbol, trade.TradeID = symbol, fmt.Sprintf("load-trade-%d", index)
		if got := admitTradeForTest(t, fixture.e, trade); got.Code != DispositionTQApplied {
			t.Fatalf("load trade %d/%s = %+v", index, symbol, got)
		}
		quote := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: fixture.binding.Identity(), TradingDate: fixture.binding.TradingDate(), Symbol: symbol,
			SIPTime: fixture.now.Add(-50 * time.Millisecond), ReceiptTime: fixture.now.Add(-50 * time.Millisecond), BidPrice: 10, AskPrice: 10.02,
			BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
			Live: LivePosition{ConnectionEpoch: 1, FrameSequence: frame + 1}}
		if got := admitQuoteForTest(t, fixture.e, quote); got.Code != DispositionTQApplied {
			t.Fatalf("load quote %d/%s = %+v", index, symbol, got)
		}
	}
	loadDuration := time.Since(started)
	afterFacts := fixture.e.ObserveSnapshot()
	if afterFacts.Publication.PublicationID != baseline.Publication.PublicationID {
		t.Fatalf("ordinary load replaced publication before cadence: baseline=%d after=%d", baseline.Publication.PublicationID, afterFacts.Publication.PublicationID)
	}
	loadedTQ := fixture.e.ObserveTQ()
	loadedAccounting := fixture.e.ObserveTQPublicationAccounting()
	if loadedTQ.Accounting.Consumed != baselineTQ.Accounting.Consumed+facts || loadedTQ.Accounting.Applied != baselineTQ.Accounting.Applied+facts ||
		loadedTQ.Accounting.RetainedTrades > maximumTradesGlobal || loadedTQ.Accounting.RetainedQuotes > 2*maximumTQSymbols ||
		loadedTQ.Accounting.RetainedFingerprints > maximumFingerprintsGlobal || loadedAccounting.CanonicalMutations != baselineAccounting.CanonicalMutations+facts ||
		loadedAccounting.ProjectionDirtied != baselineAccounting.ProjectionDirtied+facts || !loadedAccounting.ProjectionDirty {
		t.Fatalf("high-rate canonical/accounting proof failed: baseline=%+v loaded=%+v tq=%+v", baselineAccounting, loadedAccounting, loadedTQ)
	}

	for cycle := 0; cycle < 60; cycle++ {
		fixture.e.mu.Lock()
		fixture.e.state.aggregateProjectionPending = true
		fixture.e.mu.Unlock()
		beforeID := fixture.e.ObserveSnapshot().Publication.PublicationID
		admission, completion := fixture.e.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
			t.Fatalf("cadence %d did not complete", cycle)
		}
		if afterID := fixture.e.ObserveSnapshot().Publication.PublicationID; afterID != beforeID+1 {
			t.Fatalf("cadence %d publication count=%d want=%d", cycle, afterID, beforeID+1)
		}
	}
	runtime.ReadMemStats(&after)
	runtime.GC()
	var afterGC runtime.MemStats
	runtime.ReadMemStats(&afterGC)
	final := fixture.e.ObserveSnapshot()
	finalTQ := fixture.e.ObserveTQ()
	finalAccounting := fixture.e.ObserveTQPublicationAccounting()
	if final.Publication.PublicationID != baseline.Publication.PublicationID+60 ||
		!reflect.DeepEqual(final.Publication.AggregateEvaluation, baseline.Publication.AggregateEvaluation) ||
		final.Publication.CurrentMarketClaim != baseline.Publication.CurrentMarketClaim ||
		!reflect.DeepEqual(final.TQ, finalTQ) || finalAccounting.ProjectionDirty || finalAccounting.PendingProjectionMutations != 0 ||
		finalAccounting.ProjectionFlushedByCadence != baselineAccounting.ProjectionFlushedByCadence+60 ||
		finalAccounting.CanonicalMutations != finalAccounting.ImmediateTrustTransitionMutations+finalAccounting.CoalescedOrdinaryMutations {
		t.Fatalf("P1 final coalescing/control proof failed: baseline=%+v final=%+v tq=%+v accounting=%+v", baseline, final, finalTQ, finalAccounting)
	}
	t.Logf("TQ_PUBLICATION_COALESCING_P1 facts=%d cycles=%d load=%s publications=%d canonical=%d ordinary=%d immediate_mutations=%d immediate_publications=%d cadence_flushes=%d heap_before=%d heap_after=%d heap_after_gc=%d alloc_bytes=%d queue_occupancy=%d", facts, 60, loadDuration,
		final.Publication.PublicationID-baseline.Publication.PublicationID, finalAccounting.CanonicalMutations, finalAccounting.CoalescedOrdinaryMutations,
		finalAccounting.ImmediateTrustTransitionMutations, finalAccounting.ImmediateTrustTransitionPublications, finalAccounting.ProjectionFlushedByCadence,
		before.HeapAlloc, after.HeapAlloc, afterGC.HeapAlloc, after.TotalAlloc-before.TotalAlloc, fixture.e.ObserveOperational().QueueOccupancy)
}

func applyFixturePressureSample(t *testing.T, fixture tqPublicationFixture, sample TQPressureSample) {
	t.Helper()
	fixture.e.mu.Lock()
	fixture.e.advanceTQPressureTimerLocked(fixture.now)
	fixture.e.mu.Unlock()
	command := issuePressureForTest(t, fixture.e)
	input, err := NewTQPressureResultInput(command, sample)
	if err != nil {
		t.Fatal(err)
	}
	if got := admitPressureForTest(t, fixture.e, input); got.Code != DispositionTQApplied {
		t.Fatalf("pressure sample = %+v", got)
	}
}

// TestTQPublicationCoalescingPressureTransitions proves that samples remain
// cadence-coalesced while degraded, aggregate-only, and recovery mode changes
// replace the same combined publication exactly once.
func TestTQPublicationCoalescingPressureTransitions(t *testing.T) {
	fixture := newTQPublicationFixture(t, []string{"AAA"})
	fixture.confirmChannels(t)
	baseline := fixture.e.ObserveSnapshot()
	baselineAccounting := fixture.e.ObserveTQPublicationAccounting()

	healthy := healthyTQPressureSample()
	applyFixturePressureSample(t, fixture, healthy)
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != baseline.Publication.PublicationID {
		t.Fatalf("unchanged pressure sample published: got=%d want=%d", got, baseline.Publication.PublicationID)
	}
	unhealthy := healthy
	unhealthy.WaitingFrames = 10
	applyFixturePressureSample(t, fixture, unhealthy)
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != baseline.Publication.PublicationID {
		t.Fatalf("first degraded dwell sample published: got=%d want=%d", got, baseline.Publication.PublicationID)
	}
	applyFixturePressureSample(t, fixture, unhealthy)
	degraded := fixture.e.ObserveSnapshot()
	if degraded.Publication.PublicationID != baseline.Publication.PublicationID+1 || degraded.TQ.Pressure != TQPressureDegraded {
		t.Fatalf("degraded transition publication=%+v", degraded)
	}

	severe := unhealthy
	severe.WaitingFrames = 25
	applyFixturePressureSample(t, fixture, severe)
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != degraded.Publication.PublicationID {
		t.Fatalf("first aggregate-only dwell sample published: got=%d want=%d", got, degraded.Publication.PublicationID)
	}
	applyFixturePressureSample(t, fixture, severe)
	if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != degraded.Publication.PublicationID {
		t.Fatalf("second aggregate-only dwell sample published: got=%d want=%d", got, degraded.Publication.PublicationID)
	}
	applyFixturePressureSample(t, fixture, severe)
	aggregateOnly := fixture.e.ObserveSnapshot()
	if aggregateOnly.Publication.PublicationID != degraded.Publication.PublicationID+1 || aggregateOnly.TQ.Pressure != TQPressureAggregateOnly || !aggregateOnly.TQ.AggregateOnly {
		t.Fatalf("aggregate-only transition publication=%+v", aggregateOnly)
	}

	for count := 0; count < 4; count++ {
		applyFixturePressureSample(t, fixture, healthy)
		if got := fixture.e.ObserveSnapshot().Publication.PublicationID; got != aggregateOnly.Publication.PublicationID {
			t.Fatalf("early recovery sample %d published: got=%d want=%d", count+1, got, aggregateOnly.Publication.PublicationID)
		}
	}
	applyFixturePressureSample(t, fixture, healthy)
	recovered := fixture.e.ObserveSnapshot()
	accounting := fixture.e.ObserveTQPublicationAccounting()
	if recovered.Publication.PublicationID != aggregateOnly.Publication.PublicationID+1 || recovered.TQ.Pressure != TQPressureNormal || recovered.TQ.AggregateOnly ||
		accounting.ImmediateTrustTransitionPublications != baselineAccounting.ImmediateTrustTransitionPublications+3 ||
		accounting.ImmediateTrustTransitionMutations != baselineAccounting.ImmediateTrustTransitionMutations+3 {
		t.Fatalf("recovery transition publication=%+v accounting=%+v", recovered, accounting)
	}
}
