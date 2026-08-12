package replayartifact

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAppendJSONStringMatchesEncodingJSON(t *testing.T) {
	allControlBytes := make([]byte, 0x20)
	for index := range allControlBytes {
		allControlBytes[index] = byte(index)
	}
	tests := []struct {
		name  string
		value string
	}{
		{"ordinary ASCII and HTML left literal", `AAPL <tag>& value`},
		{"ordinary UTF-8", "café 東京 🚀"},
		{"quotes and backslashes", "quote=\" slash=\\"},
		{"named controls", "\b\f\n\r\t"},
		{"all control bytes", string(allControlBytes)},
		{"invalid UTF-8 replacement", string([]byte{'a', 0xff, 0xc0, 'b', 0xe2, 0x82})},
		{"line and paragraph separators", "left\u2028middle\u2029right"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var expected bytes.Buffer
			encoder := json.NewEncoder(&expected)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(test.value); err != nil {
				t.Fatal(err)
			}
			want := expected.Bytes()
			want = want[:len(want)-1]
			got := appendJSONString([]byte("prefix:"), test.value)
			if !bytes.Equal(got, append([]byte("prefix:"), want...)) {
				t.Fatalf("encoding differs\n got: %q\nwant: %q", got, append([]byte("prefix:"), want...))
			}
		})
	}
}

func TestStrictCanonicalAggregateHotPathRejectsAdversarialBytes(t *testing.T) {
	canonical, err := encodeAggregateLine(aggregateLine{
		Kind: "aggregate", Ordinal: 1, LogicalDeliveryTime: "2026-08-06T08:00:01Z", Symbol: "AAA",
		WindowStart: "2026-08-06T08:00:00Z", WindowEnd: "2026-08-06T08:00:01Z",
		Open: 10, High: 11, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1,
		ATSProvenance: "live_provider_average",
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded aggregateLine
	if kind, err := lineKind(canonical); err != nil || kind != "aggregate" {
		t.Fatalf("canonical kind = %q err=%v", kind, err)
	}
	if err := strictCanonicalAggregateLine(canonical, &decoded); err != nil {
		t.Fatalf("canonical aggregate rejected: %v", err)
	}

	tests := []struct {
		name    string
		mutated []byte
	}{
		{"unknown field", bytes.Replace(canonical, []byte(`,"ordinal":1`), []byte(`,"unknown":0,"ordinal":1`), 1)},
		{"duplicate field", bytes.Replace(canonical, []byte(`,"ordinal":1`), []byte(`,"ordinal":1,"ordinal":1`), 1)},
		{"signed zero", bytes.Replace(canonical, []byte(`,"volume":1`), []byte(`,"volume":-0`), 1)},
		{"alternate numeric spelling", bytes.Replace(canonical, []byte(`,"open":10`), []byte(`,"open":1e1`), 1)},
		{"member order", bytes.Replace(canonical, []byte(`,"open":10,"high":11`), []byte(`,"high":11,"open":10`), 1)},
		{"leading whitespace", append([]byte(" "), canonical...)},
		{"duplicate kind", bytes.Replace(canonical, []byte(`{"kind":"aggregate",`), []byte(`{"kind":"aggregate","kind":"aggregate",`), 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kind, kindErr := lineKind(test.mutated)
			if kindErr == nil && kind == "aggregate" {
				var destination aggregateLine
				if err := strictCanonicalAggregateLine(test.mutated, &destination); err == nil {
					t.Fatal("adversarial aggregate reached canonical success")
				}
				return
			}
			if strings.TrimSpace(kind) != "" {
				t.Fatalf("invalid kind result %q err=%v", kind, kindErr)
			}
		})
	}
}
