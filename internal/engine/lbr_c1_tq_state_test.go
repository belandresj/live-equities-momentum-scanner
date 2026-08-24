package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// TestPLBRC1TQState is the allocated C1 primary proof. It uses the real owner,
// combined publication cell, and selected-row seam; scaled limits distinguish
// every production containment branch without a capacity run.
func TestPLBRC1TQState(t *testing.T) {
	t.Run("conditional TRF identity", func(t *testing.T) {
		f := newTQPublicationFixture(t, []string{"AAA"})
		trade := baseTrade(f.binding, f.now.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 11})
		trade.TradeID, trade.TRFPresent, trade.TRFID = "conditional-trf", false, 101
		if got := admitTradeForTest(t, f.e, trade); got.Code != DispositionTQApplied {
			t.Fatalf("absent-TRF original = %+v", got)
		}
		absentRepeat := trade
		absentRepeat.Live.FrameSequence, absentRepeat.TRFID = 12, 202
		if got := admitTradeForTest(t, f.e, absentRepeat); got.Code != DispositionTQDuplicate {
			t.Fatalf("irrelevant absent TRF ID split identity = %+v", got)
		}

		present := trade
		present.TradeID, present.TRFPresent, present.TRFID, present.Live.FrameSequence = "present-trf", true, 101, 13
		if got := admitTradeForTest(t, f.e, present); got.Code != DispositionTQApplied {
			t.Fatalf("present TRF original = %+v", got)
		}
		presentDistinct := present
		presentDistinct.TRFID, presentDistinct.Live.FrameSequence = 202, 14
		if got := admitTradeForTest(t, f.e, presentDistinct); got.Code != DispositionTQApplied {
			t.Fatalf("present TRF ID failed to distinguish identity = %+v", got)
		}
	})

	t.Run("causal horizons features and stalled cleanup", func(t *testing.T) {
		f := newTQPublicationFixture(t, []string{"AAA"})

		// The successful write boundary is complete-frame evidence: a later
		// array element in B is still excluded and cannot confirm either channel.
		atBoundary := baseTrade(f.binding, f.now.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 10, ArrayIndex: 99})
		atBoundary.TradeID = "at-boundary"
		if got := admitTradeForTest(t, f.e, atBoundary); got.Code != DispositionTQFenced || f.e.ObserveTQ().Rows[0].TradeCoverage {
			t.Fatalf("complete-frame B was not strict: disposition=%+v view=%+v", got, f.e.ObserveTQ())
		}

		// Equality at T-30s is accepted and independently confirms trade; one
		// nanosecond older is rejected without closing that confirmed channel.
		equality := baseTrade(f.binding, f.now.Add(-30*time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 11})
		equality.TradeID, equality.ReceiptTime = "late-equality", f.now.Add(-4*time.Second)
		if got := admitTradeForTest(t, f.e, equality); got.Code != DispositionTQApplied {
			t.Fatalf("T-30s equality = %+v", got)
		}
		older := equality
		older.TradeID, older.EventTime, older.Live.FrameSequence = "late-older", equality.EventTime.Add(-time.Nanosecond), 12
		if got := admitTradeForTest(t, f.e, older); got.Code != DispositionTQRejected {
			t.Fatalf("older than T-30s = %+v", got)
		}
		if view := f.e.ObserveTQ(); !view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage || view.Rows[0].Tape.Status != TQWarming {
			t.Fatalf("independent trade confirmation/warmup = %+v", view)
		}
		f.e.mu.Lock()
		*f.clock = f.now.Add(time.Second)
		f.e.mu.Unlock()
		applyTQTimer(t, f.e)
		committedT := f.now.Add(time.Second)
		if view := f.e.ObserveTQ(); view.Rows[0].Tape.Status != TQCurrent || view.Rows[0].Tape.FiveSecond != 0 {
			t.Fatalf("covered quiet Tape = %+v", view)
		}
		beforeAggregate := f.e.ObserveReplayDeterministic()
		beforeAggregateBytes, err := json.Marshal([]any{beforeAggregate.Canonical, beforeAggregate.Evaluation})
		if err != nil {
			t.Fatal(err)
		}

		quoteEquality := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: f.binding.Identity(), TradingDate: f.binding.TradingDate(), Symbol: "AAA",
			SIPTime: committedT.Add(-30 * time.Second), ReceiptTime: committedT.Add(-time.Second), BidPrice: 10, AskPrice: 10.02,
			BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
			Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 13}}
		if got := admitQuoteForTest(t, f.e, quoteEquality); got.Code != DispositionTQApplied || !f.e.ObserveTQ().Rows[0].QuoteCoverage || f.e.ObserveTQ().Rows[0].Spread.Status != TQStale {
			t.Fatalf("quote T-30s equality = %+v view=%+v", got, f.e.ObserveTQ())
		}
		quoteOlder := quoteEquality
		quoteOlder.SIPTime, quoteOlder.Live.FrameSequence = quoteEquality.SIPTime.Add(-time.Nanosecond), 14
		if got := admitQuoteForTest(t, f.e, quoteOlder); got.Code != DispositionTQRejected || !f.e.ObserveTQ().Rows[0].QuoteCoverage {
			t.Fatalf("quote older than T-30s = %+v view=%+v", got, f.e.ObserveTQ())
		}

		locked := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: f.binding.Identity(), TradingDate: f.binding.TradingDate(), Symbol: "AAA",
			SIPTime: committedT.Add(-time.Second), ReceiptTime: committedT.Add(-time.Second), BidPrice: 10, AskPrice: 10,
			BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
			Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 15}}
		if got := admitQuoteForTest(t, f.e, locked); got.Code != DispositionTQApplied {
			t.Fatalf("locked quote = %+v", got)
		}
		if spread := f.e.ObserveTQ().Rows[0].Spread; spread.Status != TQCurrent || spread.Cents != 0 || spread.BasisPoints != 0 {
			t.Fatalf("locked spread = %+v", spread)
		}
		oneSided := locked
		oneSided.Live.FrameSequence, oneSided.BidPresent, oneSided.BidPrice = 16, false, 0
		beforeOneSided := f.e.ObserveSnapshot().Publication.PublicationID
		if got := admitQuoteForTest(t, f.e, oneSided); got.Code != DispositionTQApplied || f.e.ObserveTQ().Rows[0].Spread.Reason != "one_sided_quote" {
			t.Fatalf("one-sided quote = %+v view=%+v", got, f.e.ObserveTQ())
		}
		if f.e.ObserveSnapshot().Publication.PublicationID != beforeOneSided+1 {
			t.Fatal("one-sided trust closure did not publish immediately")
		}
		crossed := locked
		crossed.Live.FrameSequence, crossed.BidPrice, crossed.AskPrice = 17, 10.01, 10
		if got := admitQuoteForTest(t, f.e, crossed); got.Code != DispositionTQApplied || f.e.ObserveTQ().Rows[0].Spread.Reason != "crossed_quote" {
			t.Fatalf("crossed quote = %+v view=%+v", got, f.e.ObserveTQ())
		}
		if retained := f.e.ObserveTQ().Accounting.RetainedQuotes; retained != 2 {
			t.Fatalf("quote state is not O(1): retained=%d", retained)
		}

		trade := baseTrade(f.binding, committedT.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 18})
		trade.TradeID, trade.ReceiptTime = "dedupe-boundary", committedT.Add(-time.Second)
		if got := admitTradeForTest(t, f.e, trade); got.Code != DispositionTQApplied {
			t.Fatalf("qualifying trade = %+v", got)
		}
		duplicate := trade
		duplicate.Live.FrameSequence = 19
		if got := admitTradeForTest(t, f.e, duplicate); got.Code != DispositionTQDuplicate {
			t.Fatalf("exact duplicate = %+v", got)
		}
		lifecycle := trade
		lifecycle.TradeID, lifecycle.Live.FrameSequence, lifecycle.Lifecycle = "lifecycle", 20, "unsupported_lifecycle"
		if got := admitTradeForTest(t, f.e, lifecycle); got.Code != DispositionTQApplied || !f.e.ObserveTQ().Rows[0].Tape.LifecycleRecordsObserved {
			t.Fatalf("lifecycle disclosure = %+v view=%+v", got, f.e.ObserveTQ())
		}
		unequal := trade
		unequal.Live.FrameSequence, unequal.Price = 21, 10.02
		beforeUnequal := f.e.ObserveSnapshot().Publication.PublicationID
		if got := admitTradeForTest(t, f.e, unequal); got.Code != DispositionTQRejected || f.e.ObserveTQ().Rows[0].Tape.Status != TQInvalid {
			t.Fatalf("unequal repeat = %+v view=%+v", got, f.e.ObserveTQ())
		}
		if f.e.ObserveSnapshot().Publication.PublicationID != beforeUnequal+1 {
			t.Fatal("unequal-repeat trust closure did not publish immediately")
		}

		// Freeze aggregate support at committed T while engine time advances.
		// Duplicate evidence survives equality and both contribution/duplicate
		// liabilities disappear only on the next monotonic tick.
		f.e.mu.Lock()
		f.e.state.hydration.supportedThrough = immutableTime(committedT)
		*f.clock = trade.ReceiptTime.Add(30 * time.Second)
		f.e.mu.Unlock()
		applyTQTimer(t, f.e)
		f.e.mu.Lock()
		identity := tqTradeIdentity{exchange: trade.Exchange, tradeID: trade.TradeID}
		_, retainedAtEquality := f.e.state.tq.members["AAA"].fingerprints[identity]
		f.e.mu.Unlock()
		if !retainedAtEquality {
			t.Fatal("receipt+30s equality evicted duplicate evidence")
		}
		f.e.mu.Lock()
		*f.clock = trade.ReceiptTime.Add(30*time.Second + time.Nanosecond)
		f.e.mu.Unlock()
		applyTQTimer(t, f.e)
		f.e.mu.Lock()
		_, retainedAfter := f.e.state.tq.members["AAA"].fingerprints[identity]
		committed := *f.e.state.committedT
		f.e.mu.Unlock()
		if retainedAfter || committed != committedT {
			t.Fatalf("strict duplicate cleanup/stalled T: retained=%t committed=%s", retainedAfter, committed)
		}
		afterTQ := f.e.ObserveReplayDeterministic()
		afterAggregateBytes, err := json.Marshal([]any{afterTQ.Canonical, afterTQ.Evaluation})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(beforeAggregate.Canonical, afterTQ.Canonical) || !reflect.DeepEqual(beforeAggregate.Evaluation, afterTQ.Evaluation) || !bytes.Equal(beforeAggregateBytes, afterAggregateBytes) {
			t.Fatal("T/Q mutation changed aggregate projection")
		}
		drop := TQDropInput{SchemaVersion: TQSchemaV1, BindingIdentity: f.binding.Identity(), TradingDate: f.binding.TradingDate(), Family: "Q", Symbol: "AAA",
			DropReason: "structural_quote", Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 22}}
		admission, completion := f.e.AdmitTQDrop(context.Background(), drop)
		if admission != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionTQRejected || f.e.ObserveTQ().Rows[0].QuoteCoverage {
			t.Fatalf("quote gap did not close trust: admission=%s view=%+v", admission, f.e.ObserveTQ())
		}
		f.e.mu.Lock()
		f.e.state.aggregateEvaluator.current.mode = rankingStale
		f.e.state.aggregateEvaluator.current.rows = nil
		f.e.reconcileTQLocked(*f.clock)
		f.e.mu.Unlock()
		if view := f.e.ObserveTQ(); len(view.Desired) != 0 || !view.CommandPending || view.PendingAction != TQUnsubscribe {
			t.Fatalf("rank removal = %+v", view)
		}

		// Epoch loss closes trust and the replacement epoch owns a fresh empty
		// T/Q generation. Aggregate canonical/evaluation bytes remain unchanged.
		if got := admitConnectionControl(t, f.e, controlFact(f.binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 23}, *f.clock, 0, ControlFailed)); got.Code != DispositionConnectionControlApplied {
			t.Fatalf("connection loss = %+v", got)
		}
		for _, fact := range []ConnectionControlInput{
			controlFact(f.binding.Identity(), ConnectionAttempt, 2, LivePosition{}, *f.clock, 20, ControlSucceeded),
			controlFact(f.binding.Identity(), ConnectionEstablished, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 1}, *f.clock, 20, ControlSucceeded),
			controlFact(f.binding.Identity(), AuthenticationResult, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 2}, *f.clock, 20, ControlSucceeded),
			controlFact(f.binding.Identity(), AggregateCommandWriteResult, 2, LivePosition{}, *f.clock, 21, ControlSucceeded),
			controlFact(f.binding.Identity(), AggregateSubscriptionResult, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 3}, *f.clock, 21, ControlSucceeded),
		} {
			if got := admitConnectionControl(t, f.e, fact); got.Code != DispositionConnectionControlApplied {
				t.Fatalf("replacement epoch fact = %+v", got)
			}
		}
		afterAggregate := f.e.ObserveReplayDeterministic()
		if !reflect.DeepEqual(beforeAggregate.Canonical, afterAggregate.Canonical) {
			t.Fatalf("T/Q loss/replacement changed canonical aggregate state: before=%+v after=%+v", beforeAggregate.Canonical, afterAggregate.Canonical)
		}
		view := f.e.ObserveTQ()
		if view.Accounting.RetainedTrades != 0 || view.Accounting.RetainedFingerprints != 0 || view.Accounting.RetainedQuotes != 0 || view.Accounting.RetainedBytes != 0 {
			t.Fatalf("replacement epoch retained old generation = %+v", view.Accounting)
		}
	})

	t.Run("count and byte containment", func(t *testing.T) {
		assertProduction := defaultTQRetentionLimits()
		if assertProduction != (tqRetentionLimits{tradesPerSymbol: 10_000, tradesGlobal: 100_000, fingerprintsPerSymbol: 25_000, fingerprintsGlobal: 400_000, bytesPerSymbol: 4 << 20, bytesGlobal: 64 << 20}) {
			t.Fatalf("production limits = %+v", assertProduction)
		}

		for _, test := range []struct {
			name      string
			limits    tqRetentionLimits
			aggregate bool
		}{
			{"symbol contribution count", tqRetentionLimits{tradesPerSymbol: 0, tradesGlobal: 10, fingerprintsPerSymbol: 10, fingerprintsGlobal: 10, bytesPerSymbol: 4 << 20, bytesGlobal: 64 << 20}, false},
			{"global contribution count", tqRetentionLimits{tradesPerSymbol: 10, tradesGlobal: 0, fingerprintsPerSymbol: 10, fingerprintsGlobal: 10, bytesPerSymbol: 4 << 20, bytesGlobal: 64 << 20}, true},
			{"symbol duplicate count", tqRetentionLimits{tradesPerSymbol: 10, tradesGlobal: 10, fingerprintsPerSymbol: 0, fingerprintsGlobal: 10, bytesPerSymbol: 4 << 20, bytesGlobal: 64 << 20}, false},
			{"global duplicate count", tqRetentionLimits{tradesPerSymbol: 10, tradesGlobal: 10, fingerprintsPerSymbol: 10, fingerprintsGlobal: 0, bytesPerSymbol: 4 << 20, bytesGlobal: 64 << 20}, true},
			{"symbol bytes", tqRetentionLimits{tradesPerSymbol: 10, tradesGlobal: 10, fingerprintsPerSymbol: 10, fingerprintsGlobal: 10, bytesPerSymbol: tqMemberByteCharge + tqTradeByteCharge + tqFingerprintByteCharge - 1, bytesGlobal: 64 << 20}, false},
			{"global bytes", tqRetentionLimits{tradesPerSymbol: 10, tradesGlobal: 10, fingerprintsPerSymbol: 10, fingerprintsGlobal: 10, bytesPerSymbol: 4 << 20, bytesGlobal: tqMemberByteCharge + tqTradeByteCharge + tqFingerprintByteCharge - 1}, true},
		} {
			t.Run(test.name, func(t *testing.T) {
				f := newTQPublicationFixture(t, []string{"AAA"})
				f.e.tqLimits = test.limits
				trade := baseTrade(f.binding, f.now.Add(-time.Second), LivePosition{ConnectionEpoch: 1, FrameSequence: 11})
				trade.TradeID = test.name
				if got := admitTradeForTest(t, f.e, trade); got.Code != DispositionTQRejected || got.Reason != ReasonAccounting {
					t.Fatalf("bound disposition = %+v", got)
				}
				view := f.e.ObserveTQ()
				if test.aggregate != view.AggregateOnly || view.Rows[0].TradeCoverage || view.Rows[0].Tape.Status == TQCurrent || view.Accounting.RetainedBytes > maximumTQBytesGlobal {
					t.Fatalf("bound containment = %+v", view)
				}
			})
		}
	})
}

// Compile-time guard: C1 retains no separate one-second Tape projection.
func TestPLBRC1TapeProjectionShape(t *testing.T) {
	shape := reflect.TypeOf(TapeRateView{})
	for index := 0; index < shape.NumField(); index++ {
		if shape.Field(index).Name == "OneSecond" || shape.Field(index).Name == "OneSecondStatus" || shape.Field(index).Name == "OneSecondReason" {
			t.Fatalf("Tape projection restored a superseded field: %v", shape)
		}
	}
	if _, ok := shape.FieldByName("FiveSecond"); !ok {
		t.Fatalf("Tape 5s projection missing from %v", shape)
	}
}
