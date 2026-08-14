package massive

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// NormalizeLiveFrame is a test-only collector. Production consumes the cursor
// one result at a time and never materializes this slice.
func NormalizeLiveFrame(frame LiveFrame, status *StatusContext, options LiveNormalizationOptions) ([]LiveResult, LiveFrameAccounting) {
	cursor := newLiveFrameCursor(frame, status, options)
	var results []LiveResult
	for {
		result, ok := cursor.Next()
		if !ok {
			break
		}
		results = append(results, result)
	}
	return results, cursor.Accounting()
}

func TestPC5ClassStrictMixedFrameClassification(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	start := binding.SessionStart().Add(10 * time.Second)
	received := start.Add(2 * time.Second)
	aggregate := aggregateLiveJSON("AAA", start, `"dv":"1000.5"`)
	trade := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-1","p":10.1,"s":2,"t":%d}`, start.UnixMilli())
	quote := fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10,"ap":10.2}`, start.UnixMilli())
	status := `{"ev":"status","status":"success","message":"ignored provider prose"}`
	context := &StatusContext{ConnectionEpoch: 7, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: "aggregate-1", ExpectedCount: 1}

	results, accounting := NormalizeLiveFrame(liveFrame(binding, 7, 11, received, "["+aggregate+","+trade+","+quote+","+status+"]"), context, LiveNormalizationOptions{})
	if !accounting.Reconciles() || accounting.DeclaredArrayElements != 4 || len(results) != 4 {
		t.Fatalf("mixed accounting/results = %+v %d", accounting, len(results))
	}
	wantKinds := []LiveResultKind{LiveResultAggregate, LiveResultTrade, LiveResultQuote, LiveResultStatus}
	for index, result := range results {
		if result.Kind != wantKinds[index] || result.Position != (engine.LivePosition{ConnectionEpoch: 7, FrameSequence: 11, ArrayIndex: uint32(index)}) {
			t.Fatalf("result %d = %+v", index, result)
		}
	}

	t.Run("explicit unsupported family is attributable and preserves causal remainder", func(t *testing.T) {
		frame := "[" + aggregate + `,{"ev":"mystery"},` + aggregate + "]"
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 12, received, frame), nil, LiveNormalizationOptions{})
		if len(got) != 3 || got[0].Kind != LiveResultAggregate || got[1].Kind != LiveResultRejected || got[1].Rejection.Family != LiveFamilyUnsupported ||
			got[2].Kind != LiveResultAggregate || counts.FencedRemainder != 0 || !counts.Reconciles() {
			t.Fatalf("unsupported family = %#v %+v", got, counts)
		}
	})

	t.Run("duplicate event and trailing JSON fail closed", func(t *testing.T) {
		for _, data := range []string{`[{"ev":"A","ev":"T"}]`, `[{}]`, `[7]`, `[] {}`} {
			got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 13, received, data), nil, LiveNormalizationOptions{})
			if len(got) != 1 || got[0].Kind != LiveResultAmbiguous || !counts.Reconciles() {
				t.Fatalf("data %q = %#v %+v", data, got, counts)
			}
		}
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 13, received, `{}`), nil, LiveNormalizationOptions{})
		if len(got) != 1 || counts.ArrayCardinalityKnown || counts.DeclaredArrayElements != 0 || counts.ArrayElementsExamined != 0 || counts.FrameIngressAmbiguity != 1 || !counts.Reconciles() {
			t.Fatalf("non-array accounting fabricated cardinality: %#v %+v", got, counts)
		}
	})

	t.Run("later syntax and trailing JSON preserve the causal prefix", func(t *testing.T) {
		for _, data := range []string{
			"[" + aggregate + `] {}`,
			"[" + aggregate + `,{"ev":]`,
			"[" + aggregate + `,]`,
		} {
			got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 13, received, data), nil, LiveNormalizationOptions{})
			if len(got) != 2 || got[0].Kind != LiveResultAggregate || got[1].Kind != LiveResultAmbiguous || got[1].Position.ArrayIndex != 1 || !counts.Reconciles() {
				t.Fatalf("causal syntax prefix for %q = %#v %+v", data, got, counts)
			}
		}
	})

	t.Run("ambiguous aggregate identity fences later facts", func(t *testing.T) {
		for _, item := range []string{`{"ev":"A"}`, `{"ev":"A","sym":"AAA","sym":"BBB"}`} {
			got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 13, received, "["+item+","+aggregate+"]"), nil, LiveNormalizationOptions{})
			if len(got) != 1 || got[0].Kind != LiveResultAmbiguous || got[0].Rejection.Reason != LiveRejectSymbol || counts.FencedRemainder != 1 || !counts.Reconciles() {
				t.Fatalf("ambiguous identity = %#v %+v", got, counts)
			}
		}
	})

	t.Run("empty array is an explicit no-item disposition", func(t *testing.T) {
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 13, received, `[]`), nil, LiveNormalizationOptions{})
		if len(got) != 0 || counts.DeclaredArrayElements != 0 || !counts.Reconciles() {
			t.Fatalf("empty frame = %#v %+v", got, counts)
		}
	})

	t.Run("non UTF-8 and invalid causal context fail closed", func(t *testing.T) {
		frame := liveFrame(binding, 7, 13, received, `[]`)
		frame.Data = []byte{0xff}
		got, counts := NormalizeLiveFrame(frame, nil, LiveNormalizationOptions{})
		if got[0].Kind != LiveResultAmbiguous || !counts.Reconciles() {
			t.Fatalf("non UTF-8 = %#v %+v", got, counts)
		}
		frame = liveFrame(binding, 0, 13, received, `[]`)
		got, counts = NormalizeLiveFrame(frame, nil, LiveNormalizationOptions{})
		if got[0].Kind != LiveResultAmbiguous || got[0].Rejection.Reason != LiveRejectFrameBounds || !counts.Reconciles() {
			t.Fatalf("causal context = %#v %+v", got, counts)
		}
		frame = liveFrame(binding, 7, 13, received.In(time.FixedZone("not-utc", 3600)), `[]`)
		got, counts = NormalizeLiveFrame(frame, nil, LiveNormalizationOptions{})
		if got[0].Rejection.Reason != LiveRejectFrameBounds || counts.FrameIngressAmbiguity != 1 {
			t.Fatalf("non-UTC receipt entered success: %#v %+v", got, counts)
		}
	})

	t.Run("shed TQ remains classifiable and cannot hide later aggregate control", func(t *testing.T) {
		frame := "[" + trade + "," + quote + "," + aggregate + "," + status + "]"
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 14, received, frame), context, LiveNormalizationOptions{ShedTradesQuotes: true})
		if len(got) != 4 || got[0].Rejection.Reason != LiveRejectOptionalShed || got[1].Rejection.Reason != LiveRejectOptionalShed || got[2].Kind != LiveResultAggregate || got[3].Status.Disposition != StatusAcknowledged || counts.AttributableRejected != 2 || !counts.Reconciles() {
			t.Fatalf("shed classification = %#v %+v", got, counts)
		}
	})

	t.Run("frame-local budget sheds only later TQ and preserves aggregate control order", func(t *testing.T) {
		calls := 0
		clockStart := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
		clock := func() time.Time {
			at := clockStart.Add(time.Duration(calls) * 100 * time.Millisecond)
			calls++
			return at
		}
		frame := "[" + trade + "," + quote + "," + aggregate + "," + status + "," + trade + "," + quote + "," + aggregate + "]"
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 15, received, frame), context, LiveNormalizationOptions{
			ClassificationClock: clock, FrameTQBudget: FrameLocalTQBudget,
		})
		want := []LiveResultKind{LiveResultTrade, LiveResultQuote, LiveResultAggregate, LiveResultStatus, LiveResultRejected, LiveResultRejected, LiveResultAggregate}
		if len(got) != len(want) || !counts.Reconciles() || counts.NormalizedTrades != 1 || counts.NormalizedQuotes != 1 || counts.NormalizedAggregates != 2 || counts.NormalizedControls != 1 || counts.AttributableRejected != 2 || counts.PressureShedTrades != 1 || counts.PressureShedQuotes != 1 {
			t.Fatalf("frame-local accounting = %#v %+v", got, counts)
		}
		for index := range want {
			if got[index].Kind != want[index] || got[index].Position.ArrayIndex != uint32(index) {
				t.Fatalf("result %d = %+v", index, got[index])
			}
		}
		if got[4].Rejection.Reason != LiveRejectOptionalShed || got[4].Rejection.Symbol != "AAA" || got[5].Rejection.Reason != LiveRejectOptionalShed || got[5].Rejection.Symbol != "AAA" {
			t.Fatalf("frame-local sheds = %+v %+v", got[4], got[5])
		}
	})

	t.Run("attributable rejection preserves neighbors", func(t *testing.T) {
		bad := fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":0,"h":1,"l":1,"c":1,"v":1,"vw":1,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli())
		got, counts := NormalizeLiveFrame(liveFrame(binding, 7, 15, received, "["+aggregate+","+bad+","+aggregate+"]"), nil, LiveNormalizationOptions{})
		if len(got) != 3 || got[1].Kind != LiveResultRejected || got[2].Kind != LiveResultAggregate || counts.AttributableRejected != 1 || !counts.Reconciles() {
			t.Fatalf("local rejection = %#v %+v", got, counts)
		}
	})
}

func TestPC5AggLiveAggregateNormalization(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	start := binding.SessionStart().Add(time.Minute)
	received := start.Add(2 * time.Second)

	t.Run("exact dv live ATS and provenance", func(t *testing.T) {
		result := oneLiveResult(t, binding, received, aggregateLiveJSON("AAA", start, `"dv":"1000.5"`))
		wantValues := engine.AggregateValues{Open: 10, High: 11, Low: 9, Close: 10.5, Volume: 1000.5, VWAP: 10.25, AverageTradeSize: 0, ATSProvenance: engine.ATSLiveProviderAverage}
		if result.Kind != LiveResultAggregate || result.Aggregate.SchemaVersion != engine.AggregateSchemaV1 || result.Aggregate.Source != engine.AggregateSourceLive || result.Aggregate.BindingIdentity != binding.Identity() || result.Aggregate.Symbol != "AAA" || !result.Aggregate.WindowStart.Equal(start) || !result.Aggregate.WindowEnd.Equal(start.Add(time.Second)) || result.Aggregate.Values != wantValues {
			t.Fatalf("aggregate = %+v", result.Aggregate)
		}
	})

	t.Run("v fallback and provider z is not REST floor", func(t *testing.T) {
		result := oneLiveResult(t, binding, received, aggregateLiveJSON("AAA", start, `"v":200.25,"z":17`))
		if result.Aggregate.Values.Volume != 200.25 || result.Aggregate.Values.AverageTradeSize != 17 || result.Aggregate.Values.ATSProvenance != engine.ATSLiveProviderAverage {
			t.Fatalf("fallback = %+v", result.Aggregate.Values)
		}
	})

	t.Run("frame byte bound is not an economic volume ceiling", func(t *testing.T) {
		result := oneLiveResult(t, binding, received, aggregateLiveJSON("AAA", start, `"dv":"9000000.5","z":12`))
		if result.Kind != LiveResultAggregate || result.Aggregate.Values.Volume != 9000000.5 {
			t.Fatalf("large economic volume = %+v", result)
		}
	})

	t.Run("binding membership remains engine acceptance", func(t *testing.T) {
		result := oneLiveResult(t, binding, received, aggregateLiveJSON("NOT-BOUND", start, `"v":1,"z":1`))
		if result.Kind != LiveResultAggregate || result.Aggregate.Symbol != "NOT-BOUND" || result.Aggregate.BindingIdentity != binding.Identity() {
			t.Fatalf("provider fact did not preserve engine-bound identity evidence: %+v", result)
		}
	})

	t.Run("signed zero volume normalizes", func(t *testing.T) {
		result := oneLiveResult(t, binding, received, aggregateLiveJSON("AAA", start, `"v":-0,"z":0`))
		if result.Kind != LiveResultAggregate || math.Signbit(result.Aggregate.Values.Volume) {
			t.Fatalf("signed zero = %+v", result)
		}
	})

	invalid := map[string]string{
		"duplicate_numeric": fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":10,"o":11,"h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli()),
		"missing_numeric":   fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli()),
		"fractional_time":   fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d.5,"e":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli()),
		"wrong_interval":    fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, start.UnixMilli(), start.Add(2*time.Second).UnixMilli()),
		"nonfinite":         fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":1e999,"h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli()),
		"bad_structure":     fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":12,"h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, start.UnixMilli(), start.Add(time.Second).UnixMilli()),
	}
	for name, raw := range invalid {
		t.Run(name, func(t *testing.T) {
			result := oneLiveResult(t, binding, received, raw)
			if result.Kind != LiveResultRejected || !reflect.DeepEqual(result.Aggregate, engine.AggregateInput{}) {
				t.Fatalf("non-atomic rejection = %+v", result)
			}
		})
	}
}

func TestPC5TradeNormalization(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	sip := binding.SessionStart().Add(time.Minute)
	received := sip.Add(time.Second)
	complete := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-1","p":10.25,"s":2,"ds":"2.5","c":[1,999],"pt":%d,"t":%d,"q":7,"z":2,"trfi":12,"trft":1}`, sip.Add(-time.Millisecond).UnixMilli(), sip.UnixMilli())
	result := oneLiveResult(t, binding, received, complete)
	trade := result.Trade
	if result.Kind != LiveResultTrade || trade.BindingIdentity != binding.Identity() || trade.TradingDate != binding.TradingDate() || trade.EconomicSize != 2.5 || trade.EventTime != trade.ParticipantTime || trade.TimestampBasis != TimestampParticipant || trade.Conditions.Count != 2 || !trade.Conditions.Classified || !trade.TRFPresent || !trade.IdentityClassified || trade.Sequence.Value != 7 || trade.Tape.Value != 2 {
		t.Fatalf("complete trade = %+v", trade)
	}

	t.Run("participant after SIP falls back", func(t *testing.T) {
		raw := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-2","p":10,"s":1,"pt":%d,"t":%d}`, sip.Add(time.Millisecond).UnixMilli(), sip.UnixMilli())
		got := oneLiveResult(t, binding, received, raw).Trade
		if got.TimestampBasis != TimestampSIPFallback || !got.EventTime.Equal(sip) || !got.ParticipantValid || !got.SIPValid {
			t.Fatalf("fallback = %+v", got)
		}
	})

	t.Run("identity and lifecycle uncertainty remain explicit", func(t *testing.T) {
		raw := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-3","p":10,"s":1,"t":%d,"trfi":12,"e":{"action":"correction","id":"untrusted"}}`, sip.UnixMilli())
		got := oneLiveResult(t, binding, received, raw).Trade
		if got.IdentityClassified || got.Lifecycle != TradeUnsupportedLifecycle || !got.LifecycleEvidence.Present || got.LifecycleEvidence.Classified || got.LifecycleEvidence.JSONType != MetadataObject {
			t.Fatalf("uncertainty = %+v", got)
		}
	})

	invalid := map[string]string{
		"decimal_floor_mismatch":  fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade","p":10,"s":2,"ds":"3.1","t":%d}`, sip.UnixMilli()),
		"future_beyond_tolerance": fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade","p":10,"s":1,"t":%d}`, received.Add(providerFutureSkew+time.Millisecond).UnixMilli()),
		"duplicate_price":         fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade","p":10,"p":11,"s":1,"t":%d}`, sip.UnixMilli()),
	}
	for name, raw := range invalid {
		t.Run(name, func(t *testing.T) {
			got := oneLiveResult(t, binding, received, raw)
			if got.Kind != LiveResultRejected || !reflect.DeepEqual(got.Trade, NormalizedTrade{}) {
				t.Fatalf("trade rejection = %+v", got)
			}
		})
	}
}

func TestPC5QuoteNormalization(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	sip := binding.SessionStart().Add(time.Minute)
	received := sip.Add(time.Second)

	tests := []struct {
		name, prices string
	}{
		{"locked", `"bp":10,"ap":10`},
		{"crossed", `"bp":10.2,"ap":10`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,%s,"c":7,"i":[1,2]}`, sip.UnixMilli(), test.prices)
			got := oneLiveResult(t, binding, received, raw)
			if got.Kind != LiveResultQuote || got.Quote.Conditions.Shape != MetadataScalar || got.Quote.Indicators.Shape != MetadataArray || !got.Quote.Conditions.Classified || !got.Quote.Indicators.Classified || got.Quote.BidSize.Present || got.Quote.AskSize.Present {
				t.Fatalf("quote = %+v", got.Quote)
			}
		})
	}

	t.Run("malformed ancillary metadata remains unclassified without zero fabrication", func(t *testing.T) {
		raw := fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10,"ap":10.1,"bx":"bad","bs":null,"c":{"bad":1},"i":"bad"}`, sip.UnixMilli())
		got := oneLiveResult(t, binding, received, raw).Quote
		if !got.BidExchange.Present || got.BidExchange.Classified || !got.BidSize.Present || got.BidSize.Classified || got.Conditions.Shape != MetadataUnclassified || got.Indicators.Shape != MetadataUnclassified {
			t.Fatalf("ancillary evidence = %+v", got)
		}
	})

	t.Run("one-sided quote remains explicit feature evidence", func(t *testing.T) {
		raw := fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10}`, sip.UnixMilli())
		got := oneLiveResult(t, binding, received, raw)
		if got.Kind != LiveResultQuote || !got.Quote.BidPresent || got.Quote.AskPresent || got.Quote.BidPrice != 10 {
			t.Fatalf("one-sided quote = %+v", got)
		}
	})

	invalid := map[string]string{
		"future":         fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10,"ap":10.1}`, received.Add(providerFutureSkew+time.Millisecond).UnixMilli()),
		"missing_symbol": fmt.Sprintf(`{"ev":"Q","t":%d,"bp":10,"ap":10.1}`, sip.UnixMilli()),
		"no_sides":       fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d}`, sip.UnixMilli()),
		"duplicate_core": fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10,"bp":11,"ap":12}`, sip.UnixMilli()),
	}
	for name, raw := range invalid {
		t.Run(name, func(t *testing.T) {
			got := oneLiveResult(t, binding, received, raw)
			if got.Kind != LiveResultRejected || !reflect.DeepEqual(got.Quote, NormalizedQuote{}) || got.Rejection.BindingIdentity != binding.Identity() || got.Rejection.TradingDate != binding.TradingDate() {
				t.Fatalf("quote rejection = %+v", got)
			}
		})
	}
}

func TestPC5StatusNormalizationCorrelation(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	received := binding.SessionStart().Add(time.Minute)
	base := StatusContext{ConnectionEpoch: 5, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: "secret-free-token", ExpectedCount: 1}

	t.Run("exact success is bounded and redacted", func(t *testing.T) {
		got, _ := NormalizeLiveFrame(liveFrame(binding, 5, 1, received, `[{"ev":"status","status":"success","message":"https://provider.example/?apiKey=secret"}]`), &base, LiveNormalizationOptions{})
		if len(got) != 1 || got[0].Status.Disposition != StatusAcknowledged || got[0].Status.CommandToken != base.CommandToken || got[0].Status.Phase != StatusPhaseSuccess {
			t.Fatalf("status = %+v", got)
		}
		if strings.Contains(fmt.Sprintf("%+v", got[0]), "provider.example") {
			t.Fatal("provider prose escaped normalized fact")
		}
	})

	tests := []struct {
		name, data string
		context    StatusContext
	}{
		{"wrong_phase", `[{"ev":"status","status":"connected"}]`, base},
		{"wrong_command_phase", `[{"ev":"status","status":"connected"}]`, func() StatusContext { value := base; value.ExpectedPhase = StatusPhaseConnected; return value }()},
		{"extra", `[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`, base},
		{"wrong_epoch", `[{"ev":"status","status":"success"}]`, func() StatusContext { value := base; value.ConnectionEpoch = 6; return value }()},
		{"duplicate_status", `[{"ev":"status","status":"success","status":"success"}]`, base},
		{"missing_command_token", `[{"ev":"status","status":"success"}]`, func() StatusContext { value := base; value.CommandToken = ""; return value }()},
	}

	t.Run("partial frame remains correlated evidence", func(t *testing.T) {
		partial := base
		partial.ExpectedCount = 2
		got, _ := NormalizeLiveFrame(liveFrame(binding, 5, 4, received, `[{"ev":"status","status":"success"}]`), &partial, LiveNormalizationOptions{})
		if len(got) != 1 || got[0].Status.Disposition != StatusAcknowledged || got[0].Status.ObservedCount != 1 {
			t.Fatalf("partial asynchronous status = %+v", got)
		}
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := NormalizeLiveFrame(liveFrame(binding, 5, 2, received, test.data), &test.context, LiveNormalizationOptions{})
			for _, result := range got {
				if result.Kind != LiveResultStatus || result.Status.Disposition != StatusAmbiguous || result.Status.Reason != LiveRejectStatusCorrelation {
					t.Fatalf("ambiguous status = %+v", got)
				}
			}
		})
	}

	t.Run("provider failure is a fact not retry policy", func(t *testing.T) {
		got, _ := NormalizeLiveFrame(liveFrame(binding, 5, 3, received, `[{"ev":"status","status":"auth_failed","message":"unbounded ignored"}]`), &base, LiveNormalizationOptions{})
		if got[0].Status.Disposition != StatusFailed || got[0].Status.Reason != "" {
			t.Fatalf("failure status = %+v", got[0].Status)
		}
	})
}

func liveFrame(binding reference.Binding, epoch, sequence uint64, received time.Time, data string) LiveFrame {
	return LiveFrame{Binding: binding, ConnectionEpoch: epoch, FrameSequence: sequence, ReceivedAt: received, Data: []byte(data)}
}

func aggregateLiveJSON(symbol string, start time.Time, volumeAndATS string) string {
	if volumeAndATS == "" {
		volumeAndATS = `"v":1000.5,"z":0`
	} else if volumeAndATS == `"dv":"1000.5"` {
		volumeAndATS += `,"z":0`
	}
	return fmt.Sprintf(`{"ev":"A","sym":%q,"s":%d,"e":%d,"o":10,"h":11,"l":9,"c":10.5,%s,"vw":10.25}`, symbol, start.UnixMilli(), start.Add(time.Second).UnixMilli(), volumeAndATS)
}

func oneLiveResult(t *testing.T, binding reference.Binding, received time.Time, raw string) LiveResult {
	t.Helper()
	results, accounting := NormalizeLiveFrame(liveFrame(binding, 1, 1, received, "["+raw+"]"), nil, LiveNormalizationOptions{})
	if len(results) != 1 || !accounting.Reconciles() {
		t.Fatalf("single result = %#v accounting=%+v", results, accounting)
	}
	return results[0]
}
