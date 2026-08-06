// Package massive contains provider-specific normalization that does not own
// requests, coverage, recovery, replay, or scanner state.
package massive

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// NormalizationRejection is the bounded reason a Massive REST aggregate row
// could not produce one complete provider-independent value.
type NormalizationRejection string

const (
	RejectionResponseSyntax      NormalizationRejection = "response_syntax"
	RejectionTimestamp           NormalizationRejection = "timestamp"
	RejectionNumericCount        NormalizationRejection = "numeric_count"
	RejectionStructuralAggregate NormalizationRejection = "structural_aggregate"
)

// RESTSecondAggregate is the immutable value returned by the shared Massive
// REST second-aggregate mapper. Source-specific request and delivery evidence
// is deliberately supplied by its later consumers.
type RESTSecondAggregate struct {
	Symbol                 string
	WindowStart, WindowEnd time.Time
	Values                 engine.AggregateValues
}

type restSecondAggregateRow struct {
	t, open, high, low, close, volume, vwap, transactions json.Number
}

// NormalizeRESTSecondAggregate maps one raw Massive REST aggregate row. It is
// stateless and returns either one complete value or a bounded rejection; a
// rejected row always returns the zero aggregate.
func NormalizeRESTSecondAggregate(symbol string, raw []byte) (RESTSecondAggregate, NormalizationRejection) {
	row, rejection := decodeRESTSecondAggregateRow(raw)
	if rejection != "" {
		return RESTSecondAggregate{}, rejection
	}

	startMillis, ok := exactJSONInt64(row.t)
	if !ok || startMillis%1000 != 0 || startMillis > math.MaxInt64-1000 {
		return RESTSecondAggregate{}, RejectionTimestamp
	}
	start := time.UnixMilli(startMillis).UTC()
	end := start.Add(time.Second)
	if start.UnixMilli() != startMillis || end.UnixMilli() != startMillis+1000 || end.Sub(start) != time.Second {
		return RESTSecondAggregate{}, RejectionTimestamp
	}

	open, openOK := finiteJSONFloat(row.open)
	high, highOK := finiteJSONFloat(row.high)
	low, lowOK := finiteJSONFloat(row.low)
	closePrice, closeOK := finiteJSONFloat(row.close)
	volume, volumeOK := finiteJSONFloat(row.volume)
	vwap, vwapOK := finiteJSONFloat(row.vwap)
	transactionCount, countOK := exactJSONInt64(row.transactions)
	if !openOK || !highOK || !lowOK || !closeOK || !volumeOK || !vwapOK || !countOK || transactionCount <= 0 {
		return RESTSecondAggregate{}, RejectionNumericCount
	}

	open = normalizeSignedZero(open)
	high = normalizeSignedZero(high)
	low = normalizeSignedZero(low)
	closePrice = normalizeSignedZero(closePrice)
	volume = normalizeSignedZero(volume)
	vwap = normalizeSignedZero(vwap)
	if !validAggregateStructure(open, high, low, closePrice, volume, vwap) {
		return RESTSecondAggregate{}, RejectionStructuralAggregate
	}

	average, ok := checkedFloorAverageTradeSize(volume, transactionCount)
	if !ok {
		return RESTSecondAggregate{}, RejectionNumericCount
	}

	return RESTSecondAggregate{
		Symbol:      symbol,
		WindowStart: start,
		WindowEnd:   end,
		Values: engine.AggregateValues{
			Open:             open,
			High:             high,
			Low:              low,
			Close:            closePrice,
			Volume:           volume,
			VWAP:             vwap,
			AverageTradeSize: average,
			ATSProvenance:    engine.ATSRESTFloorVolumeOverTrades,
		},
	}, ""
}

func decodeRESTSecondAggregateRow(raw []byte) (restSecondAggregateRow, NormalizationRejection) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return restSecondAggregateRow{}, RejectionResponseSyntax
	}

	var row restSecondAggregateRow
	seen := make(map[string]struct{}, 8)
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return restSecondAggregateRow{}, RejectionResponseSyntax
		}
		name, ok := nameToken.(string)
		if !ok {
			return restSecondAggregateRow{}, RejectionResponseSyntax
		}
		target := rowField(&row, name)
		if target == nil {
			var ignored json.RawMessage
			if err := decoder.Decode(&ignored); err != nil {
				return restSecondAggregateRow{}, RejectionResponseSyntax
			}
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			return restSecondAggregateRow{}, RejectionResponseSyntax
		}
		seen[name] = struct{}{}
		valueToken, err := decoder.Token()
		number, numeric := valueToken.(json.Number)
		if err != nil || !numeric {
			return restSecondAggregateRow{}, RejectionResponseSyntax
		}
		*target = number
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return restSecondAggregateRow{}, RejectionResponseSyntax
	}
	if decoder.More() {
		return restSecondAggregateRow{}, RejectionResponseSyntax
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return restSecondAggregateRow{}, RejectionResponseSyntax
	}

	return row, ""
}

func rowField(row *restSecondAggregateRow, name string) *json.Number {
	switch name {
	case "t":
		return &row.t
	case "o":
		return &row.open
	case "h":
		return &row.high
	case "l":
		return &row.low
	case "c":
		return &row.close
	case "v":
		return &row.volume
	case "vw":
		return &row.vwap
	case "n":
		return &row.transactions
	default:
		return nil
	}
}

func exactJSONInt64(number json.Number) (int64, bool) {
	if number == "" {
		return 0, false
	}
	value, ok := new(big.Rat).SetString(string(number))
	if !ok || !value.IsInt() || !value.Num().IsInt64() {
		return 0, false
	}
	return value.Num().Int64(), true
}

func finiteJSONFloat(number json.Number) (float64, bool) {
	if number == "" {
		return 0, false
	}
	value, err := number.Float64()
	return value, err == nil && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func normalizeSignedZero(value float64) float64 {
	if value == 0 {
		return 0
	}
	return value
}

func checkedFloorAverageTradeSize(volume float64, transactionCount int64) (int64, bool) {
	volumeRatio := new(big.Rat).SetFloat64(volume)
	if volumeRatio == nil || volumeRatio.Sign() < 0 || transactionCount <= 0 {
		return 0, false
	}
	ratio := new(big.Rat).Quo(volumeRatio, new(big.Rat).SetInt64(transactionCount))
	floor := new(big.Int).Quo(ratio.Num(), ratio.Denom())
	if !floor.IsInt64() {
		return 0, false
	}
	return floor.Int64(), true
}

func validAggregateStructure(open, high, low, closePrice, volume, vwap float64) bool {
	return open > 0 && high > 0 && low > 0 && closePrice > 0 && vwap > 0 && volume >= 0 &&
		low <= open && open <= high && low <= closePrice && closePrice <= high
}
