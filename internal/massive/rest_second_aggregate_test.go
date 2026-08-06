package massive

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// TestNormalizeRESTSecondAggregate is P-C4-NORM. It proves the shared row
// mapping boundary only; it does not prove an HTTP or compiler consumer,
// Component 6 fencing/reuse, provider availability, or live ATS parity.
func TestNormalizeRESTSecondAggregate(t *testing.T) {
	t.Run("approved sparse fractional fixture", func(t *testing.T) {
		fixture, err := os.ReadFile("testdata/component4-rest-second-bars.json")
		if err != nil {
			t.Fatal(err)
		}
		var response struct {
			Results []json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(fixture, &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Results) != 3 {
			t.Fatalf("fixture result count = %d", len(response.Results))
		}

		wantStarts := []int64{1785398400000, 1785398401000, 1785398403000}
		wantValues := []engine.AggregateValues{
			{Open: 10, High: 10.2, Low: 9.9, Close: 10.1, Volume: 100.75, VWAP: 10.05, AverageTradeSize: 25, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades},
			{Open: 10.1, High: 10.25, Low: 10.05, Close: 10.2, Volume: 8.5, VWAP: 10.15, AverageTradeSize: 0, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades},
			{Open: 10.2, High: 10.35, Low: 10.2, Close: 10.3, Volume: 50, VWAP: 10.275, AverageTradeSize: 25, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades},
		}
		for index, raw := range response.Results {
			got, rejection := NormalizeRESTSecondAggregate("SYN", raw)
			if rejection != "" {
				t.Fatalf("row %d rejected: %s", index, rejection)
			}
			start := time.UnixMilli(wantStarts[index]).UTC()
			if got.Symbol != "SYN" || got.WindowStart != start || got.WindowEnd != start.Add(time.Second) || got.Values != wantValues[index] {
				t.Fatalf("row %d = %+v", index, got)
			}
		}
		if wantStarts[2]-wantStarts[1] != 2000 {
			t.Fatal("fixture no longer proves a sparse second")
		}
	})

	t.Run("floor ATS and signed zero", func(t *testing.T) {
		positive, rejection := NormalizeRESTSecondAggregate("SYN", []byte(`{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":100.5,"vw":10.25,"n":3}`))
		if rejection != "" || positive.Values.AverageTradeSize != 33 {
			t.Fatalf("fractional floor result = %+v rejection=%s", positive, rejection)
		}
		negativeZero, rejection := NormalizeRESTSecondAggregate("SYN", []byte(`{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":-0,"vw":10.25,"n":3}`))
		if rejection != "" || negativeZero.Values.Volume != 0 || math.Signbit(negativeZero.Values.Volume) || negativeZero.Values.AverageTradeSize != 0 {
			t.Fatalf("signed-zero result = %+v rejection=%s", negativeZero, rejection)
		}
		positiveZero, rejection := NormalizeRESTSecondAggregate("SYN", []byte(`{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":0,"vw":10.25,"n":3}`))
		if rejection != "" || !reflect.DeepEqual(negativeZero, positiveZero) {
			t.Fatalf("signed-zero values differ: negative=%+v positive=%+v rejection=%s", negativeZero, positiveZero, rejection)
		}
		precisionBoundary, rejection := NormalizeRESTSecondAggregate("SYN", []byte(`{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":9007199254740992,"vw":10.25,"n":9007199254740993}`))
		if rejection != "" || precisionBoundary.Values.AverageTradeSize != 0 {
			t.Fatalf("exact-count floor result = %+v rejection=%s", precisionBoundary, rejection)
		}
	})

	valid := `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":100.5,"vw":10.25,"n":3}`
	tests := []struct {
		name      string
		raw       string
		rejection NormalizationRejection
	}{
		{"not object", `[]`, RejectionResponseSyntax},
		{"trailing value", valid + `{}`, RejectionResponseSyntax},
		{"malformed trailing value", valid + `{`, RejectionResponseSyntax},
		{"duplicate timestamp", `{"t":1785398400000,"t":1785398401000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionResponseSyntax},
		{"nonnumeric price", `{"t":1785398400000,"o":"10","h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionResponseSyntax},
		{"missing timestamp", `{"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionTimestamp},
		{"fractional timestamp", `{"t":1785398400000.5,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionTimestamp},
		{"nonsecond timestamp", `{"t":1785398400001,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionTimestamp},
		{"overflow timestamp", `{"t":9223372036854776000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionTimestamp},
		{"missing open", `{"t":1785398400000,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing high", `{"t":1785398400000,"o":10,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing low", `{"t":1785398400000,"o":10,"h":11,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing close", `{"t":1785398400000,"o":10,"h":11,"l":9,"v":1,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing volume", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing VWAP", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"n":1}`, RejectionNumericCount},
		{"nonfinite magnitude", `{"t":1785398400000,"o":1e309,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionNumericCount},
		{"missing transactions", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10}`, RejectionNumericCount},
		{"zero transactions", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":0}`, RejectionNumericCount},
		{"negative transactions", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":-1}`, RejectionNumericCount},
		{"fractional transactions", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1.5}`, RejectionNumericCount},
		{"overflow transactions", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":9223372036854775808}`, RejectionNumericCount},
		{"ATS overflow", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1.7976931348623157e308,"vw":10,"n":1}`, RejectionNumericCount},
		{"zero open", `{"t":1785398400000,"o":0,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"zero high", `{"t":1785398400000,"o":10,"h":0,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"zero low", `{"t":1785398400000,"o":10,"h":11,"l":0,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"zero close", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":0,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"zero VWAP", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":0,"n":1}`, RejectionStructuralAggregate},
		{"negative volume", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":-1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"open below low", `{"t":1785398400000,"o":8,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"open above high", `{"t":1785398400000,"o":12,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"close below low", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":8,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
		{"close above high", `{"t":1785398400000,"o":10,"h":11,"l":9,"c":12,"v":1,"vw":10,"n":1}`, RejectionStructuralAggregate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, rejection := NormalizeRESTSecondAggregate("SYN", []byte(test.raw))
			if rejection != test.rejection {
				t.Fatalf("rejection = %q want %q; value=%+v", rejection, test.rejection, got)
			}
			if got != (RESTSecondAggregate{}) {
				t.Fatalf("rejected row returned partial value: %+v", got)
			}
		})
	}

	t.Run("unknown member is additive", func(t *testing.T) {
		raw := []byte(`{"unknown":{"nested":true},"t":1785398400000,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}`)
		got, rejection := NormalizeRESTSecondAggregate("SYN", raw)
		if rejection != "" || got.Values.AverageTradeSize != 1 {
			t.Fatalf("additive member result = %+v rejection=%s", got, rejection)
		}
	})
}
