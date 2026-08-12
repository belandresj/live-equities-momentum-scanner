package playback

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
	for _, value := range []string{
		`AAPL <tag>& value`, "café 東京 🚀", "quote=\" slash=\\", "\b\f\n\r\t", string(allControlBytes),
		string([]byte{'a', 0xff, 0xc0, 'b', 0xe2, 0x82}), "left\u2028middle\u2029right",
	} {
		var expected bytes.Buffer
		encoder := json.NewEncoder(&expected)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			t.Fatal(err)
		}
		want := expected.Bytes()
		want = want[:len(want)-1]
		if got := appendJSONString(nil, value); !bytes.Equal(got, want) {
			t.Fatalf("encoding differs\n got: %q\nwant: %q", got, want)
		}
	}
}

func TestStrictAggregateHotPathRejectsAdversarialBytes(t *testing.T) {
	canonical := []byte(`{"kind":"aggregate","ordinal":1,"logical_delivery_time":"2026-08-06T08:00:01Z","symbol":"AAA","window_start":"2026-08-06T08:00:00Z","window_end":"2026-08-06T08:00:01Z","open":10,"high":11,"low":9,"close":10,"volume":1,"vwap":10,"average_trade_size":1,"ats_provenance":"rest_floor_volume_over_transactions"}` + "\n")
	var decoded aggregateLine
	if kind, err := kindOf(canonical); err != nil || kind != "aggregate" || strictAggregate(canonical, &decoded) != nil {
		t.Fatalf("canonical aggregate rejected kind=%q err=%v", kind, err)
	}
	mutations := [][]byte{
		bytes.Replace(canonical, []byte(`,"ordinal":1`), []byte(`,"unknown":0,"ordinal":1`), 1),
		bytes.Replace(canonical, []byte(`,"ordinal":1`), []byte(`,"ordinal":1,"ordinal":1`), 1),
		bytes.Replace(canonical, []byte(`,"volume":1`), []byte(`,"volume":-0`), 1),
		bytes.Replace(canonical, []byte(`,"open":10`), []byte(`,"open":1e1`), 1),
		bytes.Replace(canonical, []byte(`,"open":10,"high":11`), []byte(`,"high":11,"open":10`), 1),
		append([]byte(" "), canonical...),
		bytes.Replace(canonical, []byte(`{"kind":"aggregate",`), []byte(`{"kind":"aggregate","kind":"aggregate",`), 1),
	}
	for index, mutated := range mutations {
		kind, kindErr := kindOf(mutated)
		if kindErr == nil && kind == "aggregate" {
			var destination aggregateLine
			if err := strictAggregate(mutated, &destination); err == nil {
				t.Fatalf("mutation %d reached canonical success", index)
			}
			continue
		}
		if strings.TrimSpace(kind) != "" {
			t.Fatalf("mutation %d invalid kind=%q err=%v", index, kind, kindErr)
		}
	}
}
