package engine

import (
	"context"
	"testing"
	"time"
)

// TestTQDataConfirmationSeparatesWriteFromCoverage is the focused P1/P2
// engine proof: a successful paired write has a frame boundary, but each
// channel becomes usable only on its own later structurally valid event.
func TestTQDataConfirmationSeparatesWriteFromCoverage(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(20 * time.Minute)
	delay := time.Duration(0)
	e, err := New(Config{Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 4, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { e.Close(); _ = e.Wait(context.Background()) }()
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
	installExactCoverage(e.state.binding.symbols[index].aggregates, e.state.binding, e.state.binding.sessionStart, now, nil)
	e.mu.Unlock()
	applyTQTimer(t, e)

	command := issueTQForTest(t, e)
	boundary := LivePosition{ConnectionEpoch: 1, FrameSequence: 10}
	if got := admitTQResultForTest(t, e, tqResultForTest(t, command, boundary, now, ControlSucceeded)); got.Code != DispositionTQApplied {
		t.Fatalf("write result = %+v", got)
	}
	view := e.ObserveTQ()
	if view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage || view.Rows[0].ProviderPresent || !view.Rows[0].ProviderMembershipUnknown ||
		view.Rows[0].Tape.Reason != "channel_unconfirmed" || view.Rows[0].Spread.Reason != "channel_unconfirmed" || view.Commands.Written != 1 || view.Commands.Pending != 0 {
		t.Fatalf("write falsely created coverage: %+v", view)
	}
	atBoundary := baseTrade(binding, now.Add(50*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: 10, ArrayIndex: 99})
	if got := admitTradeForTest(t, e, atBoundary); got.Code != DispositionTQFenced {
		t.Fatalf("frame at B confirmed coverage: %+v", got)
	}

	// A non-volume-updating trade is still real delivery evidence. It confirms
	// only T and leaves Tape warming rather than inventing a zero-rate channel.
	trade := baseTrade(binding, now.Add(100*time.Millisecond), LivePosition{ConnectionEpoch: 1, FrameSequence: 11, ArrayIndex: 0})
	trade.Conditions = []int64{15}
	if got := admitTradeForTest(t, e, trade); got.Code != DispositionTQApplied {
		t.Fatalf("trade confirmation = %+v", got)
	}
	view = e.ObserveTQ()
	if !view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage || view.Rows[0].ProviderPresent || !view.Rows[0].ProviderMembershipUnknown ||
		view.Rows[0].Tape.Status != TQWarming || view.Rows[0].Spread.Reason != "channel_unconfirmed" {
		t.Fatalf("independent trade confirmation = %+v", view)
	}

	quote := QuoteInput{SchemaVersion: TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(), Symbol: "AAA",
		SIPTime: now.Add(200 * time.Millisecond), ReceiptTime: now.Add(200 * time.Millisecond), BidPrice: 10, AskPrice: 10.02,
		BidPresent: true, AskPresent: true, ConditionsClassified: true, IndicatorsClassified: true,
		Live: LivePosition{ConnectionEpoch: 1, FrameSequence: 12, ArrayIndex: 0}}
	if got := admitQuoteForTest(t, e, quote); got.Code != DispositionTQApplied {
		t.Fatalf("quote confirmation = %+v", got)
	}
	view = e.ObserveTQ()
	if !view.Rows[0].ProviderPresent || view.Rows[0].ProviderMembershipUnknown || view.Rows[0].Spread.Status != TQCurrent || view.Rows[0].Spread.Cents < 1.99 || view.Rows[0].Spread.Cents > 2.01 {
		t.Fatalf("quote confirmation = %+v", view)
	}

	e.mu.Lock()
	e.state.committedT = immutableTime(now.Add(6 * time.Second))
	e.mu.Unlock()
	view = e.ObserveTQ()
	if view.Rows[0].Tape.Status != TQCurrent || view.Rows[0].Tape.FiveSecond != 0 {
		t.Fatalf("covered quiet tape is not zero: %+v", view.Rows[0].Tape)
	}
}
