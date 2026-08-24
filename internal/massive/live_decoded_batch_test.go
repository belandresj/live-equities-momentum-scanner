package massive

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

func TestPLBRD1Decode(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	window := binding.SessionStart().Add(time.Minute)
	received := window.Add(2 * time.Second)
	aggregate := aggregateLiveJSON("AAA", window, `"dv":"1000.5"`)
	trade := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-1","p":10.1,"s":2,"t":%d,"unknown":{"nested":[1,2,3]}}`, window.UnixMilli())
	quote := fmt.Sprintf(`{"ev":"Q","sym":"AAA","t":%d,"bp":10,"ap":10.2}`, window.UnixMilli())
	status := `{"ev":"status","status":"success","message":"discarded provider prose"}`
	context := &StatusContext{ConnectionEpoch: 7, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: "aggregate-1", ExpectedCount: 1}

	t.Run("single pass immutable mixed semantic oracle", func(t *testing.T) {
		data := []byte("[" + aggregate + "," + trade + "," + quote + "," + status + `,{"ev":"unsupported","payload":[1,2]}]`)
		frame := LiveFrame{Binding: binding, ConnectionEpoch: 7, FrameSequence: 11, ReceivedAt: received, Data: data}
		batch := decodeLiveFrame(frame, context, LiveNormalizationOptions{})
		if batch.parsePasses != 1 || batch.BindingIdentity != binding.Identity() || batch.ConnectionEpoch != 7 || batch.FrameSequence != 11 || batch.ReceivedAt != received || batch.EncodedBytes != len(data) {
			t.Fatalf("batch envelope = %+v passes=%d", batch, batch.parsePasses)
		}
		if batch.Len() != 5 || batch.RetainedCharge <= 0 || batch.RetainedCharge > MaximumDecodedBatchCharge || !batch.Accounting().Reconciles() {
			t.Fatalf("batch bounds/accounting = len=%d charge=%d accounting=%+v", batch.Len(), batch.RetainedCharge, batch.Accounting())
		}
		want := []LiveResultKind{LiveResultAggregate, LiveResultTrade, LiveResultQuote, LiveResultStatus, LiveResultRejected}
		for index, result := range batch.results {
			position := engine.LivePosition{ConnectionEpoch: 7, FrameSequence: 11, ArrayIndex: uint32(index)}
			if result.Kind != want[index] || result.Position != position {
				t.Fatalf("result %d = %+v", index, result)
			}
		}
		if batch.results[0].Aggregate.Values.Volume != 1000.5 || batch.results[1].Trade.Symbol != "AAA" || batch.results[2].Quote.AskPrice != 10.2 ||
			batch.results[3].Status.Disposition != StatusAcknowledged || batch.results[4].Rejection.Family != LiveFamilyUnsupported {
			t.Fatalf("semantic oracle = %#v", batch.results)
		}
		for index := range data {
			data[index] = 'x'
		}
		if batch.results[0].Aggregate.Symbol != "AAA" || batch.results[1].Trade.TradeID != "trade-1" || batch.results[2].Quote.Symbol != "AAA" {
			t.Fatal("decoded batch retained a mutable raw-frame alias")
		}
	})

	t.Run("earliest ambiguity preserves prefix and fences suffix", func(t *testing.T) {
		batch := decodeLiveFrame(liveFrame(binding, 7, 12, received, "["+aggregate+`,{"sym":"AAA"},`+aggregate+"]"), nil, LiveNormalizationOptions{})
		accounting := batch.Accounting()
		if batch.Len() != 2 || batch.results[0].Kind != LiveResultAggregate || batch.results[1].Kind != LiveResultAmbiguous ||
			batch.results[1].Position.ArrayIndex != 1 || accounting.FencedRemainder != 1 || !accounting.Reconciles() {
			t.Fatalf("ambiguity batch = %#v accounting=%+v", batch.results, accounting)
		}
	})

	t.Run("local rejection continues and duplicate recognized member is bounded", func(t *testing.T) {
		badAggregate := fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":0,"h":1,"l":1,"c":1,"v":1,"vw":1,"z":1}`, window.UnixMilli(), window.Add(time.Second).UnixMilli())
		duplicateTrade := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"trade-2","p":10,"p":11,"s":1,"t":%d}`, window.UnixMilli())
		batch := decodeLiveFrame(liveFrame(binding, 7, 13, received, "["+badAggregate+","+duplicateTrade+","+aggregate+"]"), nil, LiveNormalizationOptions{})
		if batch.Len() != 3 || batch.results[0].Kind != LiveResultRejected || batch.results[0].Rejection.Family != LiveFamilyAggregate ||
			batch.results[1].Rejection.Reason != LiveRejectDuplicateMember || batch.results[2].Kind != LiveResultAggregate || !batch.Accounting().Reconciles() {
			t.Fatalf("localized batch = %#v accounting=%+v", batch.results, batch.Accounting())
		}
	})

	t.Run("wire scalar kinds remain strict", func(t *testing.T) {
		wrongNumericType := fmt.Sprintf(`{"ev":"A","sym":"AAA","s":%d,"e":%d,"o":"10","h":11,"l":9,"c":10,"v":1,"vw":10,"z":1}`, window.UnixMilli(), window.Add(time.Second).UnixMilli())
		wrongTradeIDType := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":7,"p":10,"s":1,"t":%d}`, window.UnixMilli())
		for _, test := range []struct {
			name string
			raw  string
			kind LiveResultKind
		}{
			{name: "numeric event discriminator", raw: `{"ev":7,"sym":"AAA"}`, kind: LiveResultAmbiguous},
			{name: "boolean aggregate symbol", raw: `{"ev":"A","sym":true}`, kind: LiveResultAmbiguous},
			{name: "string numeric aggregate", raw: wrongNumericType, kind: LiveResultRejected},
			{name: "numeric trade identifier", raw: wrongTradeIDType, kind: LiveResultRejected},
		} {
			t.Run(test.name, func(t *testing.T) {
				batch := decodeLiveFrame(liveFrame(binding, 7, 13, received, "["+test.raw+"]"), nil, LiveNormalizationOptions{})
				if batch.Len() != 1 || batch.results[0].Kind != test.kind || !batch.Accounting().Reconciles() {
					t.Fatalf("strict scalar kind = %#v accounting=%+v", batch.results, batch.Accounting())
				}
			})
		}
	})

	t.Run("every trailing outcome is frame ambiguity without fabricated array work", func(t *testing.T) {
		for _, suffix := range []string{" {}", " garbage", " true"} {
			batch := decodeLiveFrame(liveFrame(binding, 7, 13, received, "[]"+suffix), nil, LiveNormalizationOptions{})
			accounting := batch.Accounting()
			if batch.Len() != 1 || batch.results[0].Kind != LiveResultAmbiguous || accounting.FrameIngressAmbiguity != 1 ||
				accounting.ArrayElementsExamined != 0 || accounting.IngressAmbiguity != 0 || accounting.DeclaredArrayElements != 0 || !accounting.ArrayCardinalityKnown || !accounting.Reconciles() {
				t.Fatalf("trailing %q = %#v accounting=%+v", suffix, batch.results, accounting)
			}
		}
		batch := decodeLiveFrame(liveFrame(binding, 7, 13, received, `[{"sym":"AAA"}] garbage`), nil, LiveNormalizationOptions{})
		accounting := batch.Accounting()
		if batch.Len() != 1 || batch.results[0].Kind != LiveResultAmbiguous || batch.results[0].Position.ArrayIndex != 0 ||
			accounting.IngressAmbiguity != 1 || accounting.FrameIngressAmbiguity != 1 || accounting.ArrayElementsExamined != 1 || !accounting.Reconciles() {
			t.Fatalf("composed earliest ambiguity = %#v accounting=%+v", batch.results, accounting)
		}
	})

	t.Run("budget equality sheds TQ but preserves later aggregate and control", func(t *testing.T) {
		calls := 0
		base := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
		clock := func() time.Time {
			calls++
			if calls == 1 {
				return base
			}
			return base.Add(FrameLocalTQBudget)
		}
		batch := decodeLiveFrame(liveFrame(binding, 7, 14, received, "["+trade+","+quote+","+aggregate+","+status+"]"), context,
			LiveNormalizationOptions{ClassificationClock: clock, FrameTQBudget: FrameLocalTQBudget})
		if batch.Len() != 4 || batch.results[0].Rejection.Reason != LiveRejectOptionalShed || batch.results[1].Rejection.Reason != LiveRejectOptionalShed ||
			batch.results[2].Kind != LiveResultAggregate || batch.results[3].Kind != LiveResultStatus || !batch.Accounting().Reconciles() {
			t.Fatalf("budget batch = %#v accounting=%+v", batch.results, batch.Accounting())
		}
	})

	t.Run("frame and retained batch bounds fail closed", func(t *testing.T) {
		oversize := LiveFrame{Binding: binding, ConnectionEpoch: 7, FrameSequence: 15, ReceivedAt: received, Data: make([]byte, MaximumLiveFrameBytes+1)}
		batch := decodeLiveFrame(oversize, nil, LiveNormalizationOptions{})
		if batch.Len() != 1 || batch.results[0].Rejection.Reason != LiveRejectFrameBounds || batch.Accounting().FrameIngressAmbiguity != 1 {
			t.Fatalf("oversize frame = %#v accounting=%+v", batch.results, batch.Accounting())
		}

		probe := rejection(liveFrame(binding, 7, 16, received, "[]"), LiveFamilyUnsupported, "", engine.LivePosition{ConnectionEpoch: 7, FrameSequence: 16}, LiveRejectEventFamily)
		count := MaximumDecodedBatchCharge/retainedResultCharge(probe) + 4
		if count >= MaximumDecodedBatchElements {
			count = MaximumDecodedBatchElements - 1
		}
		var source strings.Builder
		source.Grow(count*22 + 2)
		source.WriteByte('[')
		for index := 0; index < count; index++ {
			if index > 0 {
				source.WriteByte(',')
			}
			source.WriteString(`{"ev":"unsupported"}`)
		}
		source.WriteByte(']')
		batch = decodeLiveFrame(liveFrame(binding, 7, 16, received, source.String()), nil, LiveNormalizationOptions{})
		last := batch.results[len(batch.results)-1]
		if batch.RetainedCharge > MaximumDecodedBatchCharge || batch.Len() > MaximumDecodedBatchElements || last.Kind != LiveResultAmbiguous ||
			(last.Rejection.Reason != LiveRejectBatchCharge && last.Rejection.Reason != LiveRejectBatchElements) || !batch.Accounting().Reconciles() {
			t.Fatalf("retained bound = len=%d charge=%d last=%+v accounting=%+v", batch.Len(), batch.RetainedCharge, last, batch.Accounting())
		}
	})

	t.Run("production source excludes a second parse and raw element retention", func(t *testing.T) {
		source, err := os.ReadFile("live_normalization.go")
		if err != nil {
			t.Fatal(err)
		}
		text := string(source)
		if strings.Count(text, "json.NewDecoder(") != 1 || strings.Contains(text, "json.RawMessage") || strings.Contains(text, "json.Unmarshal(") || strings.Contains(text, "analyzeLiveFrame") {
			t.Fatal("production decoder regained a second JSON parse or raw-element representation")
		}
	})
}

func BenchmarkPLBRD1Decode(b *testing.B) {
	binding := component4TestBinding(b, []string{"AAA"})
	window := binding.SessionStart().Add(time.Minute)
	received := window.Add(2 * time.Second)
	items := make([]string, 0, 256)
	for index := 0; index < 256; index++ {
		items = append(items, aggregateLiveJSON("AAA", window.Add(time.Duration(index)*time.Second), `"v":1000,"z":10`))
	}
	data := []byte("[" + strings.Join(items, ",") + "]")
	frame := LiveFrame{Binding: binding, ConnectionEpoch: 1, FrameSequence: 1, ReceivedAt: received.Add(256 * time.Second), Data: data}
	if batch := decodeLiveFrame(frame, nil, LiveNormalizationOptions{}); batch.Len() != len(items) || !batch.Accounting().Reconciles() {
		b.Fatalf("invalid benchmark fixture: len=%d accounting=%+v", batch.Len(), batch.Accounting())
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		batch := decodeLiveFrame(frame, nil, LiveNormalizationOptions{})
		if batch.Len() != len(items) {
			b.Fatal("decode changed")
		}
	}
}
