// Package replayartifact owns the closed Component 4 aggregate replay artifact
// representation and validation boundary. It owns no replay clock or engine state.
package replayartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	SchemaV1                       = "aggregate-replay-jsonl-v1"
	CompleteFinalBars ArtifactMode = "complete_final_bars"
	PartialSynthetic  ArtifactMode = "partial_synthetic"
	MassiveProvider                = "massive"
	SyntheticProvider              = "synthetic"
	MassiveEndpoint                = "/v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}"
	MassivePolicy                  = "massive-rest-second-aggregate-v1"
	SyntheticPolicy                = "synthetic-declared-v1"
	CompleteCoverage               = "complete_interval"
	PartialCoverage                = "declared_partial"
)

type ArtifactMode string

type Metadata struct {
	Mode             ArtifactMode
	ArtifactID       string
	BindingIdentity  string
	UniverseIdentity string
	TradingDate      string
	SessionStart     time.Time
	SessionEnd       time.Time
	ReplayStart      time.Time
	ReplayEnd        time.Time
	AggregateRecords int64
	CoverageEntries  int64
	EmptySymbols     int64
	SealedBytes      int64
}

type SyntheticRecord struct {
	LogicalDeliveryTime time.Time
	Symbol              string
	WindowStart         time.Time
	WindowEnd           time.Time
	Values              engine.AggregateValues
}

type PartialInput struct {
	Binding         reference.Binding
	Start, End      time.Time
	DeclaredSymbols []string
	Records         []SyntheticRecord
	MaximumBytes    int64
	MaximumRecords  int64
}

type artifactContext struct {
	mode                        ArtifactMode
	bindingID, universeID, date string
	sessionStart, sessionEnd    time.Time
	replayStart, replayEnd      time.Time
}

type canonicalRecord struct {
	logicalDeliveryTime time.Time
	symbol              string
	windowStart         time.Time
	windowEnd           time.Time
	values              engine.AggregateValues
}

type headerLine struct {
	Kind                string `json:"kind"`
	Schema              string `json:"schema"`
	ArtifactMode        string `json:"artifact_mode"`
	BindingID           string `json:"binding_id"`
	UniverseID          string `json:"universe_id"`
	TradingDate         string `json:"trading_date"`
	SessionStart        string `json:"session_start"`
	SessionEnd          string `json:"session_end"`
	ReplayStart         string `json:"replay_start"`
	ReplayEnd           string `json:"replay_end"`
	Provider            string `json:"provider"`
	Endpoint            string `json:"endpoint"`
	NormalizationPolicy string `json:"normalization_policy"`
	CompileFormat       string `json:"compile_format"`
}

type aggregateLine struct {
	Kind                string  `json:"kind"`
	Ordinal             int64   `json:"ordinal"`
	LogicalDeliveryTime string  `json:"logical_delivery_time"`
	Symbol              string  `json:"symbol"`
	WindowStart         string  `json:"window_start"`
	WindowEnd           string  `json:"window_end"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	VWAP                float64 `json:"vwap"`
	AverageTradeSize    int64   `json:"average_trade_size"`
	ATSProvenance       string  `json:"ats_provenance"`
}

type coverageLine struct {
	Kind        string `json:"kind"`
	Symbol      string `json:"symbol"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Class       string `json:"class"`
	RecordCount int64  `json:"record_count"`
}

type summaryLine struct {
	Kind             string `json:"kind"`
	AggregateRecords int64  `json:"aggregate_records"`
	CoverageEntries  int64  `json:"coverage_entries"`
	EmptySymbols     int64  `json:"empty_symbols"`
	BodyBytes        int64  `json:"body_bytes"`
}

type sealLine struct {
	Kind        string `json:"kind"`
	ArtifactID  string `json:"artifact_id"`
	SealedBytes int64  `json:"sealed_bytes"`
}

func buildComplete(binding reference.Binding, start, end time.Time, download massive.DownloadResult, maximumBytes, maximumRecords int64) ([]byte, Metadata, error) {
	if !download.Complete() || download.BindingIdentity() != binding.Identity() || download.Start() != start || download.End() != end {
		return nil, Metadata{}, errors.New("complete artifact requires matching sealed download evidence")
	}
	records := download.Records()
	if int64(len(records)) > maximumRecords {
		return nil, Metadata{}, errors.New("aggregate record budget exceeded")
	}
	canonical := make([]canonicalRecord, len(records))
	for index, value := range records {
		canonical[index] = canonicalRecord{value.WindowEnd, value.Symbol, value.WindowStart, value.WindowEnd, value.Values}
	}
	context := contextForBinding(CompleteFinalBars, binding, start, end)
	return buildCanonical(context, binding.UniverseSymbols(), canonical, maximumBytes, maximumRecords)
}

func BuildPartial(input PartialInput) ([]byte, Metadata, error) {
	if input.Binding.Identity() == "" || input.MaximumBytes <= 0 || input.MaximumRecords <= 0 ||
		!validReplayInterval(input.Binding, input.Start, input.End) || len(input.DeclaredSymbols) == 0 || int64(len(input.Records)) > input.MaximumRecords {
		return nil, Metadata{}, errors.New("invalid partial artifact input")
	}
	symbols := slices.Clone(input.DeclaredSymbols)
	slices.Sort(symbols)
	compacted := slices.Compact(slices.Clone(symbols))
	if len(compacted) != len(input.DeclaredSymbols) {
		return nil, Metadata{}, errors.New("partial coverage symbols must be unique")
	}
	bindingSymbols := input.Binding.UniverseSymbols()
	for _, symbol := range symbols {
		if _, found := slices.BinarySearch(bindingSymbols, symbol); !found {
			return nil, Metadata{}, errors.New("partial coverage symbol is outside binding")
		}
	}
	records := make([]canonicalRecord, len(input.Records))
	for index, record := range input.Records {
		records[index] = canonicalRecord{record.LogicalDeliveryTime, record.Symbol, record.WindowStart, record.WindowEnd, record.Values}
	}
	context := contextForBinding(PartialSynthetic, input.Binding, input.Start, input.End)
	return buildCanonical(context, symbols, records, input.MaximumBytes, input.MaximumRecords)
}

func contextForBinding(mode ArtifactMode, binding reference.Binding, start, end time.Time) artifactContext {
	return artifactContext{mode: mode, bindingID: binding.Identity(), universeID: binding.UniverseIdentity(), date: binding.TradingDate(), sessionStart: binding.SessionStart(), sessionEnd: binding.SessionEnd(), replayStart: start, replayEnd: end}
}

func buildCanonical(context artifactContext, coverageSymbols []string, records []canonicalRecord, maximumBytes, maximumRecords int64) ([]byte, Metadata, error) {
	if maximumBytes <= 0 || maximumRecords <= 0 || int64(len(records)) > maximumRecords || len(coverageSymbols) == 0 ||
		context.bindingID == "" || context.universeID == "" || !validContextTimes(context) {
		return nil, Metadata{}, errors.New("invalid artifact context")
	}
	records = slices.Clone(records)
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].logicalDeliveryTime != records[j].logicalDeliveryTime {
			return records[i].logicalDeliveryTime.Before(records[j].logicalDeliveryTime)
		}
		if records[i].windowStart != records[j].windowStart {
			return records[i].windowStart.Before(records[j].windowStart)
		}
		return records[i].symbol < records[j].symbol
	})
	coverageSymbols = slices.Clone(coverageSymbols)
	slices.Sort(coverageSymbols)
	if len(slices.Compact(slices.Clone(coverageSymbols))) != len(coverageSymbols) {
		return nil, Metadata{}, errors.New("duplicate artifact coverage symbol")
	}
	coverage := make(map[string]int64, len(coverageSymbols))
	for _, symbol := range coverageSymbols {
		coverage[symbol] = 0
	}
	previousIdentity := ""
	var previousLogical time.Time
	for _, record := range records {
		if _, covered := coverage[record.symbol]; !covered || !validRecord(context, record) {
			return nil, Metadata{}, errors.New("invalid artifact aggregate record")
		}
		if context.mode == CompleteFinalBars {
			identity := record.symbol + "\x00" + record.windowStart.Format(time.RFC3339Nano)
			if identity == previousIdentity {
				return nil, Metadata{}, errors.New("duplicate complete aggregate identity")
			}
			previousIdentity = identity
		}
		if !previousLogical.IsZero() && record.logicalDeliveryTime.Before(previousLogical) {
			return nil, Metadata{}, errors.New("logical delivery time regressed")
		}
		previousLogical = record.logicalDeliveryTime
		coverage[record.symbol]++
	}

	provider, endpoint, policy, coverageClass := SyntheticProvider, "", SyntheticPolicy, PartialCoverage
	if context.mode == CompleteFinalBars {
		provider, endpoint, policy, coverageClass = MassiveProvider, MassiveEndpoint, MassivePolicy, CompleteCoverage
	} else if context.mode != PartialSynthetic {
		return nil, Metadata{}, errors.New("unknown artifact mode")
	}
	header := headerLine{"header", SchemaV1, string(context.mode), context.bindingID, context.universeID, context.date,
		canonicalTime(context.sessionStart), canonicalTime(context.sessionEnd), canonicalTime(context.replayStart), canonicalTime(context.replayEnd),
		provider, endpoint, policy, SchemaV1}
	var artifact []byte
	var err error
	artifact, err = appendCanonicalLine(artifact, header, maximumBytes)
	if err != nil {
		return nil, Metadata{}, err
	}
	for index, record := range records {
		values := normalizeValues(record.values)
		line := aggregateLine{"aggregate", int64(index + 1), canonicalTime(record.logicalDeliveryTime), record.symbol,
			canonicalTime(record.windowStart), canonicalTime(record.windowEnd), values.Open, values.High, values.Low, values.Close,
			values.Volume, values.VWAP, values.AverageTradeSize, string(values.ATSProvenance)}
		artifact, err = appendCanonicalLine(artifact, line, maximumBytes)
		if err != nil {
			return nil, Metadata{}, err
		}
	}
	empty := int64(0)
	for _, symbol := range coverageSymbols {
		count := coverage[symbol]
		if context.mode == CompleteFinalBars && count == 0 {
			empty++
		}
		line := coverageLine{"coverage", symbol, canonicalTime(context.replayStart), canonicalTime(context.replayEnd), coverageClass, count}
		artifact, err = appendCanonicalLine(artifact, line, maximumBytes)
		if err != nil {
			return nil, Metadata{}, err
		}
	}
	bodyBytes := int64(len(artifact))
	summary := summaryLine{"summary", int64(len(records)), int64(len(coverageSymbols)), empty, bodyBytes}
	artifact, err = appendCanonicalLine(artifact, summary, maximumBytes)
	if err != nil {
		return nil, Metadata{}, err
	}
	sealedBytes := int64(len(artifact))
	digest := sha256.Sum256(artifact)
	artifactID := "sha256:" + hex.EncodeToString(digest[:])
	artifact, err = appendCanonicalLine(artifact, sealLine{"seal", artifactID, sealedBytes}, maximumBytes)
	if err != nil {
		return nil, Metadata{}, err
	}
	metadata := Metadata{context.mode, artifactID, context.bindingID, context.universeID, context.date, context.sessionStart, context.sessionEnd,
		context.replayStart, context.replayEnd, int64(len(records)), int64(len(coverageSymbols)), empty, sealedBytes}
	return artifact, metadata, nil
}

func appendCanonicalLine(destination []byte, value any, maximum int64) ([]byte, error) {
	line, err := encodeCanonicalLine(value)
	if err != nil {
		return nil, err
	}
	if int64(len(destination)) > maximum-int64(len(line)) || int64(len(destination))+int64(len(line)) > math.MaxInt64 {
		return nil, errors.New("artifact byte budget exceeded")
	}
	return append(destination, line...), nil
}

func encodeCanonicalLine(value any) ([]byte, error) {
	switch line := value.(type) {
	case aggregateLine:
		return encodeAggregateLine(line)
	case *aggregateLine:
		return encodeAggregateLine(*line)
	default:
		var encoded bytes.Buffer
		encoder := json.NewEncoder(&encoded)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			return nil, err
		}
		return encoded.Bytes(), nil
	}
}

func encodeAggregateLine(line aggregateLine) ([]byte, error) {
	var encoded []byte
	encoded = append(encoded, `{"kind":"aggregate","ordinal":`...)
	encoded = strconv.AppendInt(encoded, line.Ordinal, 10)
	encoded = append(encoded, `,"logical_delivery_time":`...)
	encoded = appendJSONString(encoded, line.LogicalDeliveryTime)
	encoded = append(encoded, `,"symbol":`...)
	encoded = appendJSONString(encoded, line.Symbol)
	encoded = append(encoded, `,"window_start":`...)
	encoded = appendJSONString(encoded, line.WindowStart)
	encoded = append(encoded, `,"window_end":`...)
	encoded = appendJSONString(encoded, line.WindowEnd)
	for _, field := range []struct {
		name  string
		value float64
	}{{"open", line.Open}, {"high", line.High}, {"low", line.Low}, {"close", line.Close}, {"volume", line.Volume}, {"vwap", line.VWAP}} {
		encoded = append(encoded, ',')
		encoded = appendJSONString(encoded, field.name)
		encoded = append(encoded, ':')
		encoded = appendCanonicalFloat(encoded, field.value)
	}
	encoded = append(encoded, `,"average_trade_size":`...)
	encoded = strconv.AppendInt(encoded, line.AverageTradeSize, 10)
	encoded = append(encoded, `,"ats_provenance":`...)
	encoded = appendJSONString(encoded, line.ATSProvenance)
	encoded = append(encoded, '}', '\n')
	return encoded, nil
}

func appendJSONString(destination []byte, value string) []byte {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	bytes := encoded.Bytes()
	return append(destination, bytes[:len(bytes)-1]...)
}

func appendCanonicalFloat(destination []byte, value float64) []byte {
	value = normalizeZero(value)
	return strconv.AppendFloat(destination, value, 'g', -1, 64)
}

func validContextTimes(context artifactContext) bool {
	return validWholeSecondTime(context.sessionStart) && validWholeSecondTime(context.sessionEnd) && validWholeSecondTime(context.replayStart) && validWholeSecondTime(context.replayEnd) &&
		context.sessionStart.Before(context.sessionEnd) && !context.replayStart.Before(context.sessionStart) && context.replayStart.Before(context.replayEnd) && !context.replayEnd.After(context.sessionEnd)
}

func validReplayInterval(binding reference.Binding, start, end time.Time) bool {
	return binding.Identity() != "" && validWholeSecondTime(start) && validWholeSecondTime(end) && !start.Before(binding.SessionStart()) && start.Before(end) && !end.After(binding.SessionEnd())
}

func validRecord(context artifactContext, record canonicalRecord) bool {
	values := normalizeValues(record.values)
	if !validUTCTime(record.logicalDeliveryTime) || !validWholeSecondTime(record.windowStart) || !validWholeSecondTime(record.windowEnd) ||
		record.windowEnd.Sub(record.windowStart) != time.Second || record.windowStart.Before(context.replayStart) || !record.windowStart.Before(context.replayEnd) || record.windowEnd.After(context.replayEnd) ||
		record.symbol == "" || !validValues(values) {
		return false
	}
	if context.mode == CompleteFinalBars {
		return record.logicalDeliveryTime == record.windowEnd && values.ATSProvenance == engine.ATSRESTFloorVolumeOverTrades
	}
	return !record.logicalDeliveryTime.Before(record.windowEnd) &&
		(values.ATSProvenance == engine.ATSRESTFloorVolumeOverTrades || values.ATSProvenance == engine.ATSLiveProviderAverage)
}

func validValues(values engine.AggregateValues) bool {
	finitePositive := func(value float64) bool { return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) }
	finiteNonnegative := func(value float64) bool { return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0) }
	return finitePositive(values.Open) && finitePositive(values.High) && finitePositive(values.Low) && finitePositive(values.Close) &&
		finiteNonnegative(values.Volume) && finitePositive(values.VWAP) && values.AverageTradeSize >= 0 &&
		(values.ATSProvenance == engine.ATSRESTFloorVolumeOverTrades || values.ATSProvenance == engine.ATSLiveProviderAverage) &&
		values.Low <= values.Open && values.Open <= values.High && values.Low <= values.Close && values.Close <= values.High
}

func normalizeValues(values engine.AggregateValues) engine.AggregateValues {
	values.Open = normalizeZero(values.Open)
	values.High = normalizeZero(values.High)
	values.Low = normalizeZero(values.Low)
	values.Close = normalizeZero(values.Close)
	values.Volume = normalizeZero(values.Volume)
	values.VWAP = normalizeZero(values.VWAP)
	return values
}

func normalizeZero(value float64) float64 {
	if value == 0 {
		return 0
	}
	return value
}

func validUTCTime(value time.Time) bool { return !value.IsZero() && value == value.UTC() }
func validWholeSecondTime(value time.Time) bool {
	return validUTCTime(value) && value.Nanosecond() == 0
}

func canonicalTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseCanonicalTime(value string) (time.Time, error) {
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, errors.New("timestamp is not canonical UTC")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || canonicalTime(parsed) != value || !validUTCTime(parsed) {
		return time.Time{}, errors.New("timestamp is not canonical")
	}
	return parsed, nil
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}
