package engine

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func TestC9ReviewedTradeConditionFixture(t *testing.T) {
	for _, test := range []struct {
		name                 string
		shape                bool
		conditions           []int64
		classified, eligible bool
	}{
		{"empty", true, nil, true, true}, {"volume", true, []int64{14}, true, true}, {"nonvolume", true, []int64{15}, true, false},
		{"unreviewed", true, []int64{57}, false, false}, {"unknown", true, []int64{999}, false, false}, {"malformed", false, []int64{14}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			classified, eligible := classifyTradeConditionEvidence(test.shape, test.conditions)
			if classified != test.classified || eligible != test.eligible {
				t.Fatalf("classification=%v/%v", classified, eligible)
			}
		})
	}
}

// TestPC9TAQ proves engine-owned desired membership, post-ack causal coverage,
// Tape Rate identity/deduplication, event-time quote duration, and aggregate
// independence on the compact ordinary path.
func TestPC9TAQ(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	delay := time.Duration(0)
	e, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 4, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}

	e.mu.Lock()
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive = lifecycleLive, 1, true
	e.state.aggregateAcknowledged = true
	e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.committedT = immutableTime(now)
	index := e.state.binding.index["AAA"]
	installEvaluatorMarkOnSymbol(&e.state.binding.symbols[index], now, 12, qualificationProvisional)
	installExactCoverage(e.state.binding.symbols[index].aggregates, e.state.binding, e.state.binding.sessionStart, now)
	e.mu.Unlock()

	applyTQTimer(t, e)
	view := e.ObserveTQ()
	if len(view.Desired) != 1 || view.Desired[0] != "AAA" || !view.CommandPending || view.PendingAction != TQSubscribe {
		t.Fatalf("desired/command = %+v", view)
	}
	command := issueTQForTest(t, e)

	pre := baseTrade(binding, now.Add(-500*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: 9, ArrayIndex: 1})
	if got := admitTradeForTest(t, e, pre); got.Code != DispositionTQFenced {
		t.Fatalf("pre-ack trade = %+v", got)
	}

	ackAt := now.Add(-10 * time.Second)
	ack := tqResultForTest(t, command, LivePosition{ConnectionEpoch: 1, FrameSequence: 10}, ackAt, ControlSucceeded)
	if got := admitTQResultForTest(t, e, ack); got.Code != DispositionTQApplied {
		t.Fatalf("ack = %+v", got)
	}

	trade := baseTrade(binding, now.Add(-500*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: 11, ArrayIndex: 1})
	if got := admitTradeForTest(t, e, trade); got.Code != DispositionTQApplied {
		t.Fatalf("trade = %+v", got)
	}
	regressing := trade
	regressing.TradeID = "same-position-other-identity"
	if got := admitTradeForTest(t, e, regressing); got.Code != DispositionTQFenced {
		t.Fatalf("duplicate causal position = %+v", got)
	}
	duplicate := trade
	duplicate.Live.FrameSequence = 12
	if got := admitTradeForTest(t, e, duplicate); got.Code != DispositionTQDuplicate {
		t.Fatalf("duplicate = %+v", got)
	}
	nonvolume := baseTrade(binding, now.Add(-2*time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 13, ArrayIndex: 1})
	nonvolume.TradeID, nonvolume.Conditions = "nonvolume", []int64{15}
	if got := admitTradeForTest(t, e, nonvolume); got.Code != DispositionTQApplied {
		t.Fatalf("classified nonvolume = %+v", got)
	}

	for i, offset := range []time.Duration{-9 * time.Second, -7 * time.Second, -5 * time.Second, -3 * time.Second, -time.Second} {
		ask := 10.02
		if offset < -spreadWindow {
			// These wide quotes would change the old ten-second median but must
			// not contribute to the five-second Spread window.
			ask = 10.10
		}
		quote := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(), Symbol: "AAA",
			SIPTime: now.Add(offset), ReceiptTime: now.Add(offset), BidPrice: 10, AskPrice: ask,
			BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
			Live: LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(20 + i), ArrayIndex: 1}}
		if got := admitQuoteForTest(t, e, quote); got.Code != DispositionTQApplied {
			t.Fatalf("quote %d = %+v", i, got)
		}
	}
	late := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(), Symbol: "AAA", SIPTime: now.Add(-4 * time.Second), ReceiptTime: now,
		BidPrice: 10, AskPrice: 10.04, BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true, Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 25}}
	if got := admitQuoteForTest(t, e, late); got.Code != DispositionTQApplied {
		t.Fatalf("out-of-order quote = %+v", got)
	}
	regressingQuote := late
	regressingQuote.SIPTime = now.Add(-2 * time.Second)
	if got := admitQuoteForTest(t, e, regressingQuote); got.Code != DispositionTQFenced {
		t.Fatalf("duplicate quote causal position = %+v", got)
	}

	applyTQTimer(t, e)
	view = e.ObserveTQ()
	if len(view.Rows) != 1 || !view.Rows[0].TradeCoverage || !view.Rows[0].QuoteCoverage || view.Rows[0].Tape.OneSecond != 1 ||
		view.Rows[0].Tape.FiveSecond != .2 || view.Rows[0].Tape.OneSecondStatus != TQCurrent || view.Rows[0].Tape.FiveSecondStatus != TQCurrent ||
		view.Rows[0].Spread.Status != TQCurrent || math.Abs(view.Rows[0].Spread.Cents-2) > 1e-9 || view.Rows[0].Spread.ValidDuration != 5*time.Second || view.Rows[0].Spread.Quality != "reviewed_ordinary" {
		t.Fatalf("feature view = %+v", view.Rows)
	}
	if view.Accounting.Consumed != view.Accounting.Applied+view.Accounting.Duplicate+view.Accounting.Rejected+view.Accounting.Fenced+view.Accounting.PressureShed+view.Accounting.Integrity {
		t.Fatalf("accounting = %+v", view.Accounting)
	}
	publicationBeforeTQOnly := e.ObserveSnapshot()
	changedConditions := trade
	changedConditions.Live.FrameSequence = 26
	changedConditions.Conditions = []int64{15}
	if got := admitTradeForTest(t, e, changedConditions); got.Code != DispositionTQRejected {
		t.Fatalf("changed-condition repeat = %+v", got)
	}
	if got := e.ObserveTQ().Rows[0].Tape; got.Status != TQInvalid || got.Reason != "unequal_repeat" {
		t.Fatalf("unequal repeat status = %+v", got)
	}
	publicationAfterTQOnly := e.ObserveSnapshot()
	if publicationAfterTQOnly.Publication.PublicationID <= publicationBeforeTQOnly.Publication.PublicationID || publicationBeforeTQOnly.Publication.Watermark == nil ||
		publicationAfterTQOnly.Publication.Watermark == nil || *publicationAfterTQOnly.Publication.Watermark != *publicationBeforeTQOnly.Publication.Watermark ||
		len(publicationAfterTQOnly.TQ.Rows) != 1 || publicationAfterTQOnly.TQ.Rows[0].Tape.Status != TQInvalid {
		t.Fatalf("T/Q-only publication replacement: before=%+v after=%+v", publicationBeforeTQOnly, publicationAfterTQOnly)
	}
	oneSided := late
	oneSided.Live.FrameSequence, oneSided.SIPTime, oneSided.AskPresent, oneSided.AskPrice = 27, now.Add(-500*time.Millisecond), false, 0
	if got := admitQuoteForTest(t, e, oneSided); got.Code != DispositionTQApplied {
		t.Fatalf("one-sided quote = %+v", got)
	}
	if got := e.ObserveTQ().Rows[0].Spread; got.Status != TQUnavailable || got.Reason != "one_sided_quote" {
		t.Fatalf("one-sided spread = %+v", got)
	}

	before := e.ObserveTQ()
	if got := admitTQResultForTest(t, e, ack); got.Code != DispositionTQFenced {
		t.Fatalf("completed-command replay = %+v", got)
	}
	after := e.ObserveTQ()
	if !after.Rows[0].TradeCoverage || !after.Rows[0].ProviderPresent || before.CommandPending || after.CommandPending {
		t.Fatalf("completed-command replay mutated current coverage: before=%+v after=%+v", before, after)
	}

	e.mu.Lock()
	qualifiedEvaluation := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
	e.state.aggregateEvaluator.current.mode = rankingStale
	e.state.aggregateEvaluator.current.reason = ""
	e.state.aggregateEvaluator.current.rows = nil
	e.reconcileTQLocked(now)
	e.mu.Unlock()
	view = e.ObserveTQ()
	if len(view.Desired) != 0 || !view.CommandPending || view.PendingAction != TQUnsubscribe || len(view.Rows) != 0 {
		t.Fatalf("rank removal = %+v", view)
	}
	remove := issueTQForTest(t, e)
	removeResult := tqResultForTest(t, remove, LivePosition{ConnectionEpoch: 1, FrameSequence: 30}, now, ControlSucceeded)
	if got := admitTQResultForTest(t, e, removeResult); got.Code != DispositionTQApplied {
		t.Fatalf("remove ack = %+v", got)
	}
	e.mu.Lock()
	if len(e.state.tq.members) != 0 {
		t.Fatalf("quiescent non-desired membership retained: %+v", e.state.tq.members)
	}
	e.mu.Unlock()

	e.mu.Lock()
	e.state.aggregateEvaluator.current = qualifiedEvaluation
	e.reconcileTQLocked(now)
	e.mu.Unlock()
	view = e.ObserveTQ()
	if !view.CommandPending || view.PendingAction != TQSubscribe {
		t.Fatalf("re-add command = %+v", view)
	}
	readd := issueTQForTest(t, e)
	ambiguous := tqResultForTest(t, readd, LivePosition{ConnectionEpoch: 1, FrameSequence: 31}, now, ControlAmbiguous)
	if got := admitTQResultForTest(t, e, ambiguous); got.Code != DispositionTQRejected {
		t.Fatalf("ambiguous subscribe = %+v", got)
	}
	view = e.ObserveTQ()
	if !view.CommandPending || view.PendingAction != TQUnsubscribe || !view.Rows[0].ProviderMembershipUnknown || view.Accounting.Unknown != 1 {
		t.Fatalf("unknown cleanup command = %+v", view)
	}
	cleanup := issueTQForTest(t, e)
	failedCleanup := tqResultForTest(t, cleanup, LivePosition{}, now, ControlFailed)
	if got := admitTQResultForTest(t, e, failedCleanup); got.Code != DispositionTQRejected {
		t.Fatalf("failed cleanup = %+v", got)
	}
	view = e.ObserveTQ()
	if view.CommandPending || !view.Rows[0].ProviderMembershipUnknown || view.Accounting.KnownPresent != 0 || view.Accounting.Unknown != 1 {
		t.Fatalf("failed cleanup false-zero/loop = %+v", view)
	}

	writerDone := make(chan string, 1)
	go func() {
		for i := 0; i < 32; i++ {
			admission, completion := e.AdmitTimer(context.Background())
			if admission != AdmissionAdmitted || completion == nil {
				writerDone <- "concurrent timer was not admitted"
				return
			}
			if disposition := <-completion; disposition.Code != DispositionTimerApplied {
				writerDone <- "concurrent timer was not applied"
				return
			}
		}
		writerDone <- ""
	}()
	observations := 0
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case failure := <-writerDone:
			if failure != "" {
				t.Fatal(failure)
			}
			if observations == 0 {
				t.Fatal("concurrent publication replacement was not observed")
			}
			finalSnapshot := e.ObserveSnapshot()
			if finalSnapshot.Publication.PublicationID != finalSnapshot.Operational.PublicationID || finalSnapshot.Publication.PublicationID != finalSnapshot.TQ.PublicationID {
				t.Fatal("final snapshot mixed publication identities")
			}
			goto concurrencyComplete
		case <-deadline.C:
			t.Fatal("concurrent publication replacement timed out")
		default:
			snapshot := e.ObserveSnapshot()
			if snapshot.Publication.PublicationID == 0 || snapshot.Publication.PublicationID != snapshot.Operational.PublicationID || snapshot.Publication.PublicationID != snapshot.TQ.PublicationID || len(snapshot.TQ.Rows) != len(snapshot.TQ.Desired) {
				t.Fatalf("mixed concurrent snapshot = %+v", snapshot)
			}
			if len(snapshot.TQ.Desired) != 0 {
				snapshot.TQ.Desired[0] = "MUTATED"
				if again := e.ObserveSnapshot(); len(again.TQ.Desired) == 0 || again.TQ.Desired[0] == "MUTATED" {
					t.Fatal("snapshot slices alias the publication cell")
				}
			}
			observations++
		}
	}

concurrencyComplete:
	closeAndWait(t, e)
}

func TestSpreadViewFiveSecondCoverageBoundary(t *testing.T) {
	target := testBinding(t).SessionStart().Add(time.Hour)
	quote := func(offset time.Duration, sequence uint64) tqQuote {
		return tqQuote{at: target.Add(offset), receipt: target.Add(offset), position: LivePosition{ConnectionEpoch: 1, FrameSequence: sequence},
			bid: 10, ask: 10.02, bidPresent: true, askPresent: true, quality: "reviewed_ordinary"}
	}
	for _, test := range []struct {
		name   string
		quotes []tqQuote
		status TQFieldStatus
		reason string
		valid  time.Duration
	}{
		{"four of five seconds is current", []tqQuote{quote(-4*time.Second, 1), quote(-2*time.Second, 2)}, TQCurrent, "", 4 * time.Second},
		{"three of five seconds is unavailable", []tqQuote{quote(-3*time.Second, 1), quote(-time.Second, 2)}, TQUnavailable, "insufficient_coverage", 3 * time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &tqSymbolState{quoteCoverage: tqCoverage{active: true, start: target.Add(-spreadWindow)}, quotes: test.quotes}
			got := spreadView(state, &target)
			if got.Status != test.status || got.Reason != test.reason || got.ValidDuration != test.valid {
				t.Fatalf("spread = %+v", got)
			}
		})
	}
}

func TestPC9TAQScaledGlobalBoundContainment(t *testing.T) {
	if maximumTradesPerSymbol != 50_000 || maximumTradesGlobal != 500_000 || maximumTradeFingerprints != 100_000 || maximumFingerprintsGlobal != 1_000_000 ||
		maximumQuotesPerSymbol != 20_000 || maximumQuotesGlobal != 400_000 {
		t.Fatal("production T/Q bounds changed")
	}
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(now, []evaluatorSymbol{{"AAA", "valid", 10, 12, qualificationProvisional}, {"BBB", "valid", 10, 11, qualificationProvisional}})
	e.state.liveEpoch, e.state.liveEpochActive = 1, true
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(now)
	e.state.tq = tqState{epoch: 1, nextToken: 1, desired: []string{"AAA", "BBB"}, members: map[string]*tqSymbolState{
		"AAA": {present: true, tradeCoverage: tqCoverage{active: true, epoch: 1, start: now.Add(-time.Minute), ack: LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, greatest: LivePosition{ConnectionEpoch: 1, FrameSequence: 1}}, quoteCoverage: tqCoverage{active: true}},
		"BBB": {present: true, tradeCoverage: tqCoverage{active: true}, quoteCoverage: tqCoverage{active: true}},
	}, tradeCount: 1}
	e.tqLimits = tqRetentionLimits{tradesPerSymbol: 2, tradesGlobal: 1, fingerprintsPerSymbol: 2, fingerprintsGlobal: 2, quotesPerSymbol: 2, quotesGlobal: 2}
	input := baseTrade(testTQBinding{"proof-binding", "2026-07-29"}, now.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 2})
	node := &queueNode{trade: frozenTradeInput{input}, admissionTime: now}
	if code, reason := e.applyTradeLocked(node); code != DispositionTQRejected || reason != ReasonAccounting {
		t.Fatalf("overflow = %s/%s", code, reason)
	}
	e.reconcileTQLocked(now)
	view := e.ObserveTQ()
	if !view.Bounds || !view.AggregateOnly || view.Pressure != TQPressureAggregateOnly || !view.ShedTradesQuotes || !view.CommandPending || view.PendingAction != TQUnsubscribe ||
		e.state.tq.members["AAA"].tradeCoverage.active || e.state.tq.members["BBB"].quoteCoverage.active {
		t.Fatalf("global containment = %+v state=%+v", view, e.state.tq)
	}
}

func TestPC9TAQAggregateOnlyRetiresPendingAdditions(t *testing.T) {
	setup := func(t *testing.T) (*Engine, reference.Binding, time.Time) {
		t.Helper()
		binding := testBinding(t)
		now := binding.SessionStart().Add(20 * time.Minute)
		delay := time.Duration(0)
		e, err := New(Config{Mode: RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 4, EvaluationDelay: &delay})
		if err != nil {
			t.Fatal(err)
		}
		if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
			t.Fatalf("binding = %+v", got)
		}
		e.mu.Lock()
		e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive = lifecycleLive, 1, true
		e.state.committedT = immutableTime(now)
		e.state.aggregateEvaluator.current = pressureQualifiedEvaluation(now)
		e.state.tq.members = map[string]*tqSymbolState{"AAA": {present: true,
			tradeCoverage: tqCoverage{active: true, epoch: 1, greatest: LivePosition{ConnectionEpoch: 1, FrameSequence: 10}},
			quoteCoverage: tqCoverage{active: true, epoch: 1, greatest: LivePosition{ConnectionEpoch: 1, FrameSequence: 10}},
			fingerprints:  make(map[string]tqFingerprint)}}
		e.state.tq.epoch, e.state.tq.nextToken = 1, 1
		e.reconcileTQLocked(now)
		e.mu.Unlock()
		if view := e.ObserveTQ(); !view.CommandPending || view.PendingAction != TQSubscribe || view.PendingSymbol != "MISSING" {
			t.Fatalf("pending addition = %+v", view)
		}
		return e, binding, now
	}

	admitDrop := func(t *testing.T, e *Engine, binding reference.Binding, now time.Time) Disposition {
		t.Helper()
		input := TQDropInput{SchemaVersion: TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(), Family: "Q",
			DropReason: "symbol", Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 11}}
		admission, completion := e.AdmitTQDrop(context.Background(), input)
		if admission != AdmissionAdmitted {
			t.Fatalf("drop admission = %s", admission)
		}
		return awaitDisposition(t, completion)
	}

	t.Run("undispatched subscribe is retired before removal", func(t *testing.T) {
		e, binding, now := setup(t)
		defer closeAndWait(t, e)
		if got := admitDrop(t, e, binding, now); got.Code != DispositionTQRejected {
			t.Fatalf("drop = %+v", got)
		}
		view := e.ObserveTQ()
		if !view.AggregateOnly || !view.CommandPending || view.PendingAction != TQUnsubscribe || view.PendingSymbol != "AAA" || view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage {
			t.Fatalf("containment = %+v", view)
		}
	})

	t.Run("dispatched subscribe becomes cleanup liability only", func(t *testing.T) {
		e, binding, now := setup(t)
		defer closeAndWait(t, e)
		addition := issueTQForTest(t, e)
		if got := admitDrop(t, e, binding, now); got.Code != DispositionTQRejected {
			t.Fatalf("drop = %+v", got)
		}
		result := tqResultForTest(t, addition, LivePosition{ConnectionEpoch: 1, FrameSequence: 12}, now, ControlSucceeded)
		if got := admitTQResultForTest(t, e, result); got.Code != DispositionTQApplied {
			t.Fatalf("late subscribe result = %+v", got)
		}
		view := e.ObserveTQ()
		if !view.AggregateOnly || !view.CommandPending || view.PendingAction != TQUnsubscribe {
			t.Fatalf("cleanup after applied addition = %+v", view)
		}
		for _, row := range view.Rows {
			if row.TradeCoverage || row.QuoteCoverage {
				t.Fatalf("late addition reopened coverage = %+v", view)
			}
		}
	})
}

func pressureQualifiedEvaluation(at time.Time) aggregateEvaluationResult {
	counts := featureDimensionAccounting{}
	counts.statuses[1], counts.statuses[2] = 2, 1
	counts.reasons[0], counts.reasons[9] = 2, 1
	counts.pairs[1][0], counts.pairs[2][9] = 2, 1
	features := featureAccounting{dayPercent: counts, from4AMPercent: counts, hodDrawdown: counts, sessionRange: counts, rolling30: counts, rolling60: counts, activity: counts}
	current := aggregateFeatureField{status: featureCurrent}
	return aggregateEvaluationResult{
		at: at, mode: rankingQualifiedCurrent,
		population:    populationAccounting{universeTotal: 3, validPriorClose: 2, invalidOrMissingPriorClose: 1, trustedRankableMark: 2, coveredPopulation: 3},
		qualification: qualificationAccounting{provisional: 2}, features: features, totalPassers: 2, knownRankableCount: 2, tqIntentAvailable: true,
		rows: []aggregateRankingRow{
			{rank: 1, symbol: "AAA", last: 2, dayPercent: 1, from4AMPercent: current, hodDrawdown: current, sessionRange: current, rolling30: current, rolling60: current, activity: current, tqIntentEligible: true},
			{rank: 2, symbol: "MISSING", last: 1, dayPercent: 0, from4AMPercent: current, hodDrawdown: current, sessionRange: current, rolling30: current, rolling60: current, activity: current, tqIntentEligible: true},
		},
	}
}

func TestPC9TAQScaledSymbolBoundUnsubscribes(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(now, []evaluatorSymbol{{"AAA", "valid", 10, 12, qualificationProvisional}})
	e.state.liveEpoch, e.state.liveEpochActive = 1, true
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(now)
	position := LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.tq = tqState{epoch: 1, nextToken: 1, desired: []string{"AAA"}, members: map[string]*tqSymbolState{
		"AAA": {present: true, tradeCoverage: tqCoverage{active: true, epoch: 1, start: now.Add(-time.Minute), ack: position, greatest: position}, quoteCoverage: tqCoverage{active: true, epoch: 1, ack: position, greatest: position}},
	}}
	e.tqLimits = tqRetentionLimits{tradesPerSymbol: 0, tradesGlobal: 10, fingerprintsPerSymbol: 10, fingerprintsGlobal: 10, quotesPerSymbol: 10, quotesGlobal: 10}
	input := baseTrade(testTQBinding{"proof-binding", "2026-07-29"}, now.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 2})
	if code, reason := e.applyTradeLocked(&queueNode{trade: frozenTradeInput{input}, admissionTime: now}); code != DispositionTQRejected || reason != ReasonAccounting {
		t.Fatalf("symbol overflow = %s/%s", code, reason)
	}
	e.reconcileTQLocked(now)
	view := e.ObserveTQ()
	if view.AggregateOnly || !view.CommandPending || view.PendingAction != TQUnsubscribe || e.state.tq.members["AAA"].tradeCoverage.active {
		t.Fatalf("symbol containment = %+v state=%+v", view, e.state.tq.members["AAA"])
	}
}

func TestC9DesiredMembershipExistsDuringQualifiedHydrationWithoutIssuance(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(now, []evaluatorSymbol{{"AAA", "valid", 10, 12, qualificationProvisional}})
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive = lifecycleHydrating, 1, true
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationLocked(now)
	e.reconcileTQLocked(now)
	view := e.ObserveTQ()
	if len(view.Desired) != 1 || view.Desired[0] != "AAA" || view.CommandPending {
		t.Fatalf("hydrating desired/issuance = %+v", view)
	}
}

func baseTrade(binding interface {
	Identity() string
	TradingDate() string
}, at time.Time, position LivePosition) TradeInput {
	return TradeInput{SchemaVersion: TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(), Symbol: "AAA", TradeID: "trade-1",
		Exchange: 4, Price: 10.01, EconomicSize: 100, EventTime: at, ReceiptTime: at, TimestampBasis: "participant",
		ConditionsClassified: true, IdentityClassified: true, Lifecycle: "original", Live: position}
}

type testTQBinding struct{ identity, date string }

func (v testTQBinding) Identity() string    { return v.identity }
func (v testTQBinding) TradingDate() string { return v.date }

func applyTQTimer(t *testing.T, e *Engine) {
	t.Helper()
	admission, completion := e.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted {
		t.Fatalf("timer admission = %s", admission)
	}
	if got := awaitTimerDisposition(t, completion); got.Code != DispositionTimerApplied {
		t.Fatalf("timer = %+v", got)
	}
}

func admitTQResultForTest(t *testing.T, e *Engine, input TQCommandResultInput) Disposition {
	t.Helper()
	admission, completion := e.AdmitTQCommandResult(context.Background(), input)
	if admission != AdmissionAdmitted {
		t.Fatalf("TQ result admission = %s", admission)
	}
	return awaitDisposition(t, completion)
}

func issueTQForTest(t *testing.T, e *Engine) TQCommand {
	t.Helper()
	command, err := e.IssueTQCommand()
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func tqResultForTest(t *testing.T, command TQCommand, position LivePosition, at time.Time, outcome ConnectionControlOutcome) TQCommandResultInput {
	t.Helper()
	input, err := NewTQCommandResultInput(command, position, at, outcome)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func admitTradeForTest(t *testing.T, e *Engine, input TradeInput) Disposition {
	t.Helper()
	admission, completion := e.AdmitTrade(context.Background(), input)
	if admission != AdmissionAdmitted {
		t.Fatalf("trade admission = %s", admission)
	}
	return awaitDisposition(t, completion)
}

func admitQuoteForTest(t *testing.T, e *Engine, input QuoteInput) Disposition {
	t.Helper()
	admission, completion := e.AdmitQuote(context.Background(), input)
	if admission != AdmissionAdmitted {
		t.Fatalf("quote admission = %s", admission)
	}
	return awaitDisposition(t, completion)
}
