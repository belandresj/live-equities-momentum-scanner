// Package massive contains provider-specific normalization that does not own
// requests, coverage, recovery, replay, or scanner state.
package massive

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	MaximumLiveFrameBytes       = 8 << 20
	maximumTradeIDBytes         = 128
	maximumScalarBytes          = 32
	maximumMetadataValues       = 16
	maximumCommandBytes         = 256
	providerFutureSkew          = 250 * time.Millisecond
	FrameLocalTQBudget          = 500 * time.Millisecond
	MaximumDecodedBatchElements = 65_536
	MaximumDecodedBatchCharge   = 32 << 20
	decodedBatchInitialCapacity = 1
)

type LiveResultKind string

const (
	LiveResultAggregate LiveResultKind = "aggregate"
	LiveResultTrade     LiveResultKind = "trade"
	LiveResultQuote     LiveResultKind = "quote"
	LiveResultStatus    LiveResultKind = "status"
	LiveResultRejected  LiveResultKind = "rejected"
	LiveResultAmbiguous LiveResultKind = "ingress_ambiguity"
)

type LiveFamily string

const (
	LiveFamilyAggregate   LiveFamily = "A"
	LiveFamilyTrade       LiveFamily = "T"
	LiveFamilyQuote       LiveFamily = "Q"
	LiveFamilyStatus      LiveFamily = "status"
	LiveFamilyUnsupported LiveFamily = "unsupported"
)

type LiveRejectionReason string

const (
	LiveRejectFrameBounds       LiveRejectionReason = "frame_bounds"
	LiveRejectFrameSyntax       LiveRejectionReason = "frame_syntax"
	LiveRejectBatchElements     LiveRejectionReason = "batch_elements"
	LiveRejectBatchCharge       LiveRejectionReason = "batch_charge"
	LiveRejectEventFamily       LiveRejectionReason = "event_family"
	LiveRejectDuplicateMember   LiveRejectionReason = "duplicate_recognized_member"
	LiveRejectSymbol            LiveRejectionReason = "symbol"
	LiveRejectTimestamp         LiveRejectionReason = "timestamp"
	LiveRejectNumeric           LiveRejectionReason = "numeric"
	LiveRejectAggregate         LiveRejectionReason = "aggregate_structure"
	LiveRejectTradeIdentity     LiveRejectionReason = "trade_identity"
	LiveRejectTradeSize         LiveRejectionReason = "trade_size"
	LiveRejectQuotePrice        LiveRejectionReason = "quote_price"
	LiveRejectOptionalShed      LiveRejectionReason = "optional_tq_shed"
	LiveRejectStatusCorrelation LiveRejectionReason = "status_correlation"
)

type StatusPhase string

const (
	StatusPhaseConnected   StatusPhase = "connected"
	StatusPhaseAuthSuccess StatusPhase = "auth_success"
	StatusPhaseSuccess     StatusPhase = "success"
	StatusPhaseAuthFailed  StatusPhase = "auth_failed"
	StatusPhaseError       StatusPhase = "error"
	StatusPhaseUnknown     StatusPhase = "unknown"
)

type StatusDisposition string

const (
	StatusAcknowledged StatusDisposition = "acknowledged"
	StatusFailed       StatusDisposition = "failed"
	StatusAmbiguous    StatusDisposition = "ambiguous"
)

type CommandKind string

const (
	CommandConnection            CommandKind = "connection"
	CommandAuthentication        CommandKind = "authentication"
	CommandAggregateSubscribe    CommandKind = "aggregate_subscribe"
	CommandTradeQuoteSubscribe   CommandKind = "trade_quote_subscribe"
	CommandTradeQuoteUnsubscribe CommandKind = "trade_quote_unsubscribe"
)

type TimestampBasis string

const (
	TimestampParticipant TimestampBasis = "participant"
	TimestampSIPFallback TimestampBasis = "sip_fallback"
)

type TradeLifecycle string

const (
	TradeOriginal             TradeLifecycle = "original"
	TradeUnsupportedLifecycle TradeLifecycle = "unsupported_lifecycle"
)

type MetadataShape string

const (
	MetadataAbsent       MetadataShape = "absent"
	MetadataNull         MetadataShape = "null"
	MetadataScalar       MetadataShape = "scalar"
	MetadataArray        MetadataShape = "array"
	MetadataObject       MetadataShape = "object"
	MetadataUnclassified MetadataShape = "unclassified"
)

// LiveFrame is one copied, bounded provider frame plus its immutable binding
// and receipt-order evidence. The cursor retains it only for one frame.
type LiveFrame struct {
	Binding         reference.Binding
	ConnectionEpoch uint64
	FrameSequence   uint64
	ReceivedAt      time.Time
	Data            []byte
}

// StatusContext is transport-supplied correlation evidence for the sole
// expected provider phase. It owns no command state and makes no coverage claim.
type StatusContext struct {
	ConnectionEpoch uint64
	ExpectedPhase   StatusPhase
	CommandKind     CommandKind
	CommandToken    string
	ExpectedCount   int
}

type LiveNormalizationOptions struct {
	ShedTradesQuotes    bool
	ClassificationClock func() time.Time
	FrameTQBudget       time.Duration
}

type OptionalInt64 struct {
	Present    bool
	Classified bool
	Value      int64
}

type BoundedIntVector struct {
	Shape      MetadataShape
	Classified bool
	Count      uint8
	Values     [maximumMetadataValues]int64
}

func (v BoundedIntVector) Slice() []int64 {
	return append([]int64(nil), v.Values[:v.Count]...)
}

type LifecycleEvidence struct {
	Present    bool
	Classified bool
	JSONType   MetadataShape
	Scalar     string
}

type NormalizedTrade struct {
	SchemaVersion, BindingIdentity, TradingDate string
	Symbol, TradeID                             string
	Exchange                                    int64
	TRFPresent, IdentityClassified              bool
	TRFID                                       int64
	Price, EconomicSize                         float64
	BaseSize                                    int64
	Conditions                                  BoundedIntVector
	ParticipantTime, SIPTime                    time.Time
	ParticipantValid, SIPValid                  bool
	EventTime                                   time.Time
	TimestampBasis                              TimestampBasis
	ReceiptTime                                 time.Time
	Sequence, Tape                              OptionalInt64
	Lifecycle                                   TradeLifecycle
	LifecycleEvidence                           LifecycleEvidence
	Live                                        engine.LivePosition
}

type NormalizedQuote struct {
	SchemaVersion, BindingIdentity, TradingDate string
	Symbol                                      string
	SIPTime, ReceiptTime                        time.Time
	BidPrice, AskPrice                          float64
	BidPresent, AskPresent                      bool
	BidExchange, AskExchange                    OptionalInt64
	BidSize, AskSize                            OptionalInt64
	Conditions, Indicators                      BoundedIntVector
	Sequence, Tape                              OptionalInt64
	Live                                        engine.LivePosition
}

type NormalizedStatus struct {
	BindingIdentity string
	ConnectionEpoch uint64
	Position        engine.LivePosition
	ReceiptTime     time.Time
	Phase           StatusPhase
	CommandKind     CommandKind
	CommandToken    string
	Disposition     StatusDisposition
	Reason          LiveRejectionReason
	ObservedCount   int
	FinalInFrame    bool
}

type LiveRejection struct {
	BindingIdentity string
	TradingDate     string
	Family          LiveFamily
	Symbol          string
	Reason          LiveRejectionReason
	Position        engine.LivePosition
}

type LiveResult struct {
	Kind      LiveResultKind
	Position  engine.LivePosition
	Aggregate engine.AggregateInput
	Trade     NormalizedTrade
	Quote     NormalizedQuote
	Status    NormalizedStatus
	Rejection LiveRejection
}

type LiveFrameAccounting struct {
	ArrayCardinalityKnown bool
	DeclaredArrayElements int
	ArrayElementsExamined int
	NormalizedAggregates  int
	NormalizedTrades      int
	NormalizedQuotes      int
	NormalizedControls    int
	AttributableRejected  int
	PressureShedTrades    int
	PressureShedQuotes    int
	IngressAmbiguity      int
	FencedRemainder       int
	FrameIngressAmbiguity int
}

// DecodedBatch is the sole-owned immutable result of one streaming pass over
// one provider frame. The temporary D1 bridge reads its private backing in
// order; callers cannot retain or mutate the source frame through this value.
type DecodedBatch struct {
	BindingIdentity string
	ConnectionEpoch uint64
	FrameSequence   uint64
	ReceivedAt      time.Time
	EncodedBytes    int
	RetainedCharge  int
	accounting      LiveFrameAccounting
	results         []LiveResult
	parsePasses     uint8
}

func (b DecodedBatch) Accounting() LiveFrameAccounting { return b.accounting }
func (b DecodedBatch) Len() int                        { return len(b.results) }

func (a LiveFrameAccounting) Reconciles() bool {
	classified := a.ArrayElementsExamined == a.NormalizedAggregates+a.NormalizedTrades+
		a.NormalizedQuotes+a.NormalizedControls+a.AttributableRejected+a.IngressAmbiguity &&
		a.FrameIngressAmbiguity >= 0 && a.FrameIngressAmbiguity <= 1
	if !classified || a.PressureShedTrades < 0 || a.PressureShedQuotes < 0 || a.PressureShedTrades+a.PressureShedQuotes > a.AttributableRejected {
		return false
	}
	if a.ArrayCardinalityKnown {
		return a.DeclaredArrayElements == a.ArrayElementsExamined+a.FencedRemainder
	}
	return a.DeclaredArrayElements == 0
}

// ConsumeLiveFrameForAttribution drains the ordinary live-normalization cursor
// without retaining normalized market values. It is a read-only diagnostic
// seam for comparing the existing bounded frame accounting with engine
// admission; production delivery remains LiveAttempt.DeliverNextToEngine.
func ConsumeLiveFrameForAttribution(frame LiveFrame, statusContext *StatusContext, options LiveNormalizationOptions) (LiveFrameAccounting, int) {
	batch := decodeLiveFrame(frame, statusContext, options)
	return batch.accounting, len(batch.results)
}

type wireValue struct {
	shape      MetadataShape
	kind       wireValueKind
	scalar     string
	vector     [maximumMetadataValues]int64
	count      uint8
	classified bool
}

type wireValueKind uint8

const (
	wireNull wireValueKind = iota + 1
	wireString
	wireNumber
	wireBool
	wireArray
	wireObject
)

type objectMembers struct {
	values map[string]wireValue
	counts map[string]int
}

func decodeLiveFrame(frame LiveFrame, statusContext *StatusContext, options LiveNormalizationOptions) DecodedBatch {
	batch := DecodedBatch{BindingIdentity: frame.Binding.Identity(), ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence,
		ReceivedAt: frame.ReceivedAt, EncodedBytes: len(frame.Data), parsePasses: 1}
	started := classificationNow(options)
	if !validFrameContext(frame) {
		appendBatchResult(&batch, ambiguity(engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence}, LiveRejectFrameBounds))
		batch.accounting.FrameIngressAmbiguity = 1
		return batch
	}
	if !utf8.Valid(frame.Data) || len(bytes.TrimSpace(frame.Data)) == 0 {
		appendBatchResult(&batch, ambiguity(engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence}, LiveRejectFrameSyntax))
		batch.accounting.FrameIngressAmbiguity = 1
		return batch
	}
	decoder := json.NewDecoder(bytes.NewReader(frame.Data))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		appendBatchResult(&batch, ambiguity(engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence}, LiveRejectFrameSyntax))
		batch.accounting.FrameIngressAmbiguity = 1
		return batch
	}
	statusIndexes := make([]int, 0, 2)
	ambiguous := false
	for index := 0; decoder.More(); index++ {
		position := engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence, ArrayIndex: uint32(index)}
		if ambiguous {
			if err := skipJSONValue(decoder); err != nil {
				batch.accounting.ArrayCardinalityKnown = false
				return finalizeDecodedBatch(batch, statusIndexes, statusContext)
			}
			batch.accounting.FencedRemainder++
			continue
		}
		members, ok := decodeObjectMembers(decoder)
		if !ok {
			appendClassifiedResult(&batch, ambiguity(position, LiveRejectEventFamily))
			ambiguous = true
			continue
		}
		elementOptions := options
		if classificationBudgetExceeded(options, started) {
			elementOptions.ShedTradesQuotes = true
		}
		result, terminal := normalizeElement(frame, position, members, statusContext, elementOptions)
		chargeReason := ambiguity(position, LiveRejectBatchCharge)
		if len(batch.results)+2 > MaximumDecodedBatchElements || !batchCanAppendPair(&batch, result, chargeReason) {
			reason := LiveRejectBatchCharge
			if len(batch.results)+2 > MaximumDecodedBatchElements {
				reason = LiveRejectBatchElements
			}
			appendClassifiedResult(&batch, ambiguity(position, reason))
			ambiguous = true
			continue
		}
		appendClassifiedResult(&batch, result)
		if result.Kind == LiveResultStatus {
			statusIndexes = append(statusIndexes, len(batch.results)-1)
		}
		ambiguous = terminal
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim(']') {
		position := engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence, ArrayIndex: uint32(batch.accounting.ArrayElementsExamined)}
		if !ambiguous {
			appendClassifiedResult(&batch, ambiguity(position, LiveRejectFrameSyntax))
		}
		return finalizeDecodedBatch(batch, statusIndexes, statusContext)
	}
	batch.accounting.ArrayCardinalityKnown = true
	batch.accounting.DeclaredArrayElements = batch.accounting.ArrayElementsExamined + batch.accounting.FencedRemainder
	if _, trailingErr := decoder.Token(); !errors.Is(trailingErr, io.EOF) {
		if !ambiguous {
			position := engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence, ArrayIndex: uint32(batch.accounting.DeclaredArrayElements)}
			appendBatchResult(&batch, ambiguity(position, LiveRejectFrameSyntax))
		}
		batch.accounting.FrameIngressAmbiguity = 1
	}
	return finalizeDecodedBatch(batch, statusIndexes, statusContext)
}

func finalizeDecodedBatch(batch DecodedBatch, statusIndexes []int, context *StatusContext) DecodedBatch {
	for ordinal, index := range statusIndexes {
		status := &batch.results[index].Status
		status.ObservedCount = len(statusIndexes)
		status.FinalInFrame = ordinal == len(statusIndexes)-1
		if (context == nil || len(statusIndexes) > context.ExpectedCount) && status.Disposition == StatusAcknowledged {
			status.Disposition, status.Reason = StatusAmbiguous, LiveRejectStatusCorrelation
		}
	}
	return batch
}

func appendBatchResult(batch *DecodedBatch, result LiveResult) {
	if len(batch.results) == cap(batch.results) {
		oldCapacity := cap(batch.results)
		capacity := nextDecodedBatchCapacity(oldCapacity)
		backing := make([]LiveResult, len(batch.results), capacity)
		copy(backing, batch.results)
		batch.results = backing
		batch.RetainedCharge += (capacity - oldCapacity) * int(unsafe.Sizeof(LiveResult{}))
	}
	batch.results = append(batch.results, result)
	batch.RetainedCharge += retainedResultDynamicCharge(result)
}

func appendClassifiedResult(batch *DecodedBatch, result LiveResult) {
	appendBatchResult(batch, result)
	batch.accounting.ArrayElementsExamined++
	switch result.Kind {
	case LiveResultAggregate:
		batch.accounting.NormalizedAggregates++
	case LiveResultTrade:
		batch.accounting.NormalizedTrades++
	case LiveResultQuote:
		batch.accounting.NormalizedQuotes++
	case LiveResultStatus:
		batch.accounting.NormalizedControls++
	case LiveResultRejected:
		batch.accounting.AttributableRejected++
		if result.Rejection.Reason == LiveRejectOptionalShed {
			if result.Rejection.Family == LiveFamilyTrade {
				batch.accounting.PressureShedTrades++
			} else if result.Rejection.Family == LiveFamilyQuote {
				batch.accounting.PressureShedQuotes++
			}
		}
	case LiveResultAmbiguous:
		batch.accounting.IngressAmbiguity++
	}
}

func retainedResultCharge(result LiveResult) int {
	return int(unsafe.Sizeof(result)) + retainedResultDynamicCharge(result)
}

func retainedResultDynamicCharge(result LiveResult) int {
	charge := 0
	charge += len(result.Aggregate.SchemaVersion) + len(result.Aggregate.BindingIdentity) + len(result.Aggregate.Symbol)
	charge += len(result.Trade.SchemaVersion) + len(result.Trade.BindingIdentity) + len(result.Trade.TradingDate) + len(result.Trade.Symbol) + len(result.Trade.TradeID) + len(result.Trade.LifecycleEvidence.Scalar)
	charge += len(result.Quote.SchemaVersion) + len(result.Quote.BindingIdentity) + len(result.Quote.TradingDate) + len(result.Quote.Symbol)
	charge += len(result.Status.BindingIdentity) + len(result.Status.CommandToken)
	charge += len(result.Rejection.BindingIdentity) + len(result.Rejection.TradingDate) + len(result.Rejection.Symbol)
	return charge
}

func batchCanAppendPair(batch *DecodedBatch, first, second LiveResult) bool {
	length, capacity, charge := len(batch.results), cap(batch.results), batch.RetainedCharge
	for _, result := range [...]LiveResult{first, second} {
		if length == capacity {
			newCapacity := nextDecodedBatchCapacity(capacity)
			charge += (newCapacity - capacity) * int(unsafe.Sizeof(LiveResult{}))
			capacity = newCapacity
		}
		charge += retainedResultDynamicCharge(result)
		length++
	}
	return charge <= MaximumDecodedBatchCharge
}

func nextDecodedBatchCapacity(current int) int {
	if current == 0 {
		return decodedBatchInitialCapacity
	}
	return min(current*2, MaximumDecodedBatchElements)
}

func decodeObjectMembers(decoder *json.Decoder) (objectMembers, bool) {
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return objectMembers{}, false
	}
	members := objectMembers{values: make(map[string]wireValue), counts: make(map[string]int)}
	for decoder.More() {
		nameToken, err := decoder.Token()
		name, named := nameToken.(string)
		if err != nil || !named {
			return objectMembers{}, false
		}
		if !recognizedWireMember(name) {
			if err := skipJSONValue(decoder); err != nil {
				return objectMembers{}, false
			}
			continue
		}
		value, ok := decodeWireValue(decoder)
		if !ok {
			return objectMembers{}, false
		}
		members.counts[name]++
		if members.counts[name] == 1 {
			members.values[name] = value
		}
	}
	closing, err := decoder.Token()
	return members, err == nil && closing == json.Delim('}')
}

func recognizedWireMember(name string) bool {
	switch name {
	case "ev", "sym", "s", "e", "o", "h", "l", "c", "dv", "v", "vw", "z", "x", "i", "p", "ds", "pt", "t", "q", "trfi", "trft", "bx", "ax", "bp", "ap", "bs", "as", "status":
		return true
	default:
		return false
	}
}

func decodeWireValue(decoder *json.Decoder) (wireValue, bool) {
	token, err := decoder.Token()
	if err != nil {
		return wireValue{}, false
	}
	switch value := token.(type) {
	case nil:
		return wireValue{shape: MetadataNull, kind: wireNull, scalar: "null", classified: true}, true
	case string:
		return wireValue{shape: MetadataScalar, kind: wireString, scalar: value, classified: utf8.ValidString(value)}, true
	case json.Number:
		return wireValue{shape: MetadataScalar, kind: wireNumber, scalar: string(value), classified: true}, true
	case bool:
		return wireValue{shape: MetadataScalar, kind: wireBool, scalar: strconv.FormatBool(value), classified: true}, true
	case json.Delim:
		switch value {
		case '[':
			result := wireValue{shape: MetadataArray, kind: wireArray, classified: true}
			for decoder.More() {
				element, ok := decodeWireValue(decoder)
				if !ok {
					return wireValue{}, false
				}
				integer, exact := wireExactInt64(element)
				if !exact || result.count == maximumMetadataValues {
					result.classified = false
				} else if result.classified {
					result.vector[result.count], result.count = integer, result.count+1
				}
			}
			closing, err := decoder.Token()
			return result, err == nil && closing == json.Delim(']')
		case '{':
			for decoder.More() {
				if _, err := decoder.Token(); err != nil || skipJSONValue(decoder) != nil {
					return wireValue{}, false
				}
			}
			closing, err := decoder.Token()
			return wireValue{shape: MetadataObject, kind: wireObject}, err == nil && closing == json.Delim('}')
		}
	}
	return wireValue{}, false
}

func skipJSONValue(decoder *json.Decoder) error {
	_, ok := decodeWireValue(decoder)
	if !ok {
		return strconv.ErrSyntax
	}
	return nil
}

func validFrameContext(frame LiveFrame) bool {
	return frame.ConnectionEpoch > 0 && frame.FrameSequence > 0 && !frame.ReceivedAt.IsZero() &&
		frame.ReceivedAt == frame.ReceivedAt.UTC() &&
		len(frame.Data) <= MaximumLiveFrameBytes && frame.Binding.Identity() != "" &&
		frame.Binding.TradingDate() != "" && !frame.Binding.SessionStart().IsZero() &&
		frame.Binding.SessionStart().Before(frame.Binding.SessionEnd())
}

func classificationNow(options LiveNormalizationOptions) time.Time {
	if options.ClassificationClock == nil {
		return time.Time{}
	}
	return options.ClassificationClock()
}

func classificationBudgetExceeded(options LiveNormalizationOptions, started time.Time) bool {
	if options.ShedTradesQuotes || options.ClassificationClock == nil || options.FrameTQBudget <= 0 || started.IsZero() {
		return false
	}
	return options.ClassificationClock().Sub(started) >= options.FrameTQBudget
}

func normalizeElement(frame LiveFrame, position engine.LivePosition, members objectMembers, statusContext *StatusContext, options LiveNormalizationOptions) (LiveResult, bool) {
	if members.counts["ev"] != 1 {
		return ambiguity(position, LiveRejectEventFamily), true
	}
	event, ok := rawString(members.values["ev"])
	if !ok {
		return ambiguity(position, LiveRejectEventFamily), true
	}
	switch event {
	case string(LiveFamilyAggregate):
		return normalizeAggregate(frame, position, members)
	case string(LiveFamilyTrade):
		if options.ShedTradesQuotes {
			return rejection(frame, LiveFamilyTrade, pressureShedSymbol(members), position, LiveRejectOptionalShed), false
		}
		return normalizeTrade(frame, position, members)
	case string(LiveFamilyQuote):
		if options.ShedTradesQuotes {
			return rejection(frame, LiveFamilyQuote, pressureShedSymbol(members), position, LiveRejectOptionalShed), false
		}
		return normalizeQuote(frame, position, members)
	case string(LiveFamilyStatus):
		return normalizeStatus(frame, position, members, statusContext), false
	default:
		return rejection(frame, LiveFamilyUnsupported, "", position, LiveRejectEventFamily), false
	}
}

func pressureShedSymbol(members objectMembers) string {
	if members.counts["sym"] != 1 {
		return ""
	}
	symbol, ok := rawString(members.values["sym"])
	if !ok || len(symbol) > 64 {
		return ""
	}
	return symbol
}

func ambiguity(position engine.LivePosition, reason LiveRejectionReason) LiveResult {
	result := LiveResult{Kind: LiveResultAmbiguous, Position: position}
	result.Rejection = LiveRejection{Reason: reason, Position: position}
	return result
}

func rejection(frame LiveFrame, family LiveFamily, symbol string, position engine.LivePosition, reason LiveRejectionReason) LiveResult {
	result := LiveResult{Kind: LiveResultRejected, Position: position}
	result.Rejection = LiveRejection{BindingIdentity: frame.Binding.Identity(), TradingDate: frame.Binding.TradingDate(), Family: family, Symbol: symbol, Reason: reason, Position: position}
	return result
}

func duplicateRecognized(members objectMembers, recognized ...string) bool {
	for _, name := range recognized {
		if members.counts[name] > 1 {
			return true
		}
	}
	return false
}

func normalizeAggregate(frame LiveFrame, position engine.LivePosition, members objectMembers) (LiveResult, bool) {
	if members.counts["sym"] != 1 {
		return ambiguity(position, LiveRejectSymbol), true
	}
	symbol, ok := boundedSymbol(members.values["sym"])
	if !ok {
		return ambiguity(position, LiveRejectSymbol), true
	}
	if duplicateRecognized(members, "s", "e", "o", "h", "l", "c", "dv", "v", "vw", "z") {
		return rejection(frame, LiveFamilyAggregate, symbol, position, LiveRejectDuplicateMember), false
	}
	startMillis, startOK := rawExactInt64(members.values["s"])
	endMillis, endOK := rawExactInt64(members.values["e"])
	if !startOK || !endOK || startMillis%1000 != 0 || endMillis-startMillis != 1000 {
		return rejection(frame, LiveFamilyAggregate, symbol, position, LiveRejectTimestamp), false
	}
	start, end := time.UnixMilli(startMillis).UTC(), time.UnixMilli(endMillis).UTC()
	if start.UnixMilli() != startMillis || end.UnixMilli() != endMillis || start.Before(frame.Binding.SessionStart()) || end.After(frame.Binding.SessionEnd()) {
		return rejection(frame, LiveFamilyAggregate, symbol, position, LiveRejectTimestamp), false
	}
	open, openOK := rawFiniteFloat(members.values["o"])
	high, highOK := rawFiniteFloat(members.values["h"])
	low, lowOK := rawFiniteFloat(members.values["l"])
	closePrice, closeOK := rawFiniteFloat(members.values["c"])
	vwap, vwapOK := rawFiniteFloat(members.values["vw"])
	var volume float64
	var volumeOK bool
	if members.counts["dv"] == 1 {
		volume, volumeOK = rawDecimalString(members.values["dv"], false, 0, MaximumLiveFrameBytes)
	} else {
		volume, volumeOK = rawFiniteFloat(members.values["v"])
	}
	average, averageOK := rawExactInt64(members.values["z"])
	if !openOK || !highOK || !lowOK || !closeOK || !vwapOK || !volumeOK || !averageOK || average < 0 {
		return rejection(frame, LiveFamilyAggregate, symbol, position, LiveRejectNumeric), false
	}
	open, high, low, closePrice = normalizeSignedZero(open), normalizeSignedZero(high), normalizeSignedZero(low), normalizeSignedZero(closePrice)
	volume, vwap = normalizeSignedZero(volume), normalizeSignedZero(vwap)
	if !validAggregateStructure(open, high, low, closePrice, volume, vwap) {
		return rejection(frame, LiveFamilyAggregate, symbol, position, LiveRejectAggregate), false
	}
	input := engine.AggregateInput{
		SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: frame.Binding.Identity(),
		Source: engine.AggregateSourceLive, Symbol: symbol, WindowStart: start, WindowEnd: end,
		Values:       engine.AggregateValues{Open: open, High: high, Low: low, Close: closePrice, Volume: volume, VWAP: vwap, AverageTradeSize: average, ATSProvenance: engine.ATSLiveProviderAverage},
		DeliveryTime: frame.ReceivedAt.UTC(), Live: position,
	}
	return LiveResult{Kind: LiveResultAggregate, Position: position, Aggregate: input}, false
}

func normalizeTrade(frame LiveFrame, position engine.LivePosition, members objectMembers) (LiveResult, bool) {
	recognized := []string{"sym", "x", "i", "p", "s", "ds", "c", "pt", "t", "q", "z", "trfi", "trft", "e"}
	if duplicateRecognized(members, recognized...) {
		return rejection(frame, LiveFamilyTrade, bestEffortSymbol(members), position, LiveRejectDuplicateMember), false
	}
	symbol, symbolOK := boundedSymbol(members.values["sym"])
	exchange, exchangeOK := rawExactInt64(members.values["x"])
	tradeID, tradeIDOK := rawString(members.values["i"])
	price, priceOK := rawFiniteFloat(members.values["p"])
	if !symbolOK || !exchangeOK || exchange <= 0 || !tradeIDOK || tradeID == "" || len(tradeID) > maximumTradeIDBytes || !priceOK || price <= 0 || price > 1e9 {
		return rejection(frame, LiveFamilyTrade, bestEffortSymbol(members), position, LiveRejectTradeIdentity), false
	}
	baseSize, sizeOK := rawExactInt64(members.values["s"])
	if !sizeOK || baseSize < 0 {
		return rejection(frame, LiveFamilyTrade, symbol, position, LiveRejectTradeSize), false
	}
	economicSize := float64(baseSize)
	if members.counts["ds"] == 1 {
		value, ok := rawDecimalString(members.values["ds"], true, 1e12, maximumScalarBytes)
		if !ok || math.Floor(value) != float64(baseSize) {
			return rejection(frame, LiveFamilyTrade, symbol, position, LiveRejectTradeSize), false
		}
		economicSize = value
	} else if baseSize == 0 {
		return rejection(frame, LiveFamilyTrade, symbol, position, LiveRejectTradeSize), false
	}
	conditions := decodeConditions(members, "c", false)
	participant, participantPresent, participantValid := decodeEventTime(members, "pt", frame)
	sip, sipPresent, sipValid := decodeEventTime(members, "t", frame)
	_ = participantPresent
	_ = sipPresent
	var eventTime time.Time
	basis := TimestampParticipant
	if participantValid && (!sipValid || !participant.After(sip)) {
		eventTime = participant
	} else if sipValid {
		eventTime, basis = sip, TimestampSIPFallback
	} else {
		return rejection(frame, LiveFamilyTrade, symbol, position, LiveRejectTimestamp), false
	}
	sequence := decodeOptionalInt(members, "q", 0, math.MaxInt64)
	tape := decodeOptionalInt(members, "z", 1, 3)
	trade := NormalizedTrade{
		SchemaVersion: "normalized-trade-v1", BindingIdentity: frame.Binding.Identity(), TradingDate: frame.Binding.TradingDate(),
		Symbol: symbol, TradeID: tradeID, Exchange: exchange, Price: normalizeSignedZero(price), BaseSize: baseSize,
		EconomicSize: normalizeSignedZero(economicSize), Conditions: conditions, ParticipantTime: participant,
		SIPTime: sip, ParticipantValid: participantValid, SIPValid: sipValid, EventTime: eventTime,
		TimestampBasis: basis, ReceiptTime: frame.ReceivedAt.UTC(), Sequence: sequence, Tape: tape,
		Lifecycle: TradeOriginal, IdentityClassified: true, Live: position,
	}
	trfIDRaw, hasTRFID := members.values["trfi"]
	trfTimeRaw, hasTRFTime := members.values["trft"]
	if hasTRFID && hasTRFTime {
		trfID, idOK := rawExactInt64(trfIDRaw)
		trfTime, timeOK := rawExactInt64(trfTimeRaw)
		if idOK && timeOK && trfID > 0 && trfTime > 0 {
			trade.TRFPresent, trade.TRFID = true, trfID
		} else {
			trade.IdentityClassified = false
		}
	} else if hasTRFID || hasTRFTime {
		trade.IdentityClassified = false
	}
	if lifecycleRaw, present := members.values["e"]; present {
		trade.Lifecycle = TradeUnsupportedLifecycle
		trade.LifecycleEvidence = boundedLifecycleEvidence(lifecycleRaw)
	}
	return LiveResult{Kind: LiveResultTrade, Position: position, Trade: trade}, false
}

func normalizeQuote(frame LiveFrame, position engine.LivePosition, members objectMembers) (LiveResult, bool) {
	recognized := []string{"sym", "t", "bx", "ax", "bp", "ap", "bs", "as", "c", "i", "q", "z"}
	if duplicateRecognized(members, recognized...) {
		return rejection(frame, LiveFamilyQuote, bestEffortSymbol(members), position, LiveRejectDuplicateMember), false
	}
	symbol, symbolOK := boundedSymbol(members.values["sym"])
	sip, _, sipValid := decodeEventTime(members, "t", frame)
	bidRaw, bidPresent := members.values["bp"]
	askRaw, askPresent := members.values["ap"]
	bid, bidOK := rawFiniteFloat(bidRaw)
	ask, askOK := rawFiniteFloat(askRaw)
	if !symbolOK {
		return rejection(frame, LiveFamilyQuote, bestEffortSymbol(members), position, LiveRejectSymbol), false
	}
	if !sipValid {
		return rejection(frame, LiveFamilyQuote, symbol, position, LiveRejectTimestamp), false
	}
	if bidPresent && (!bidOK || bid <= 0 || bid > 1e9) || askPresent && (!askOK || ask <= 0 || ask > 1e9) {
		return rejection(frame, LiveFamilyQuote, symbol, position, LiveRejectQuotePrice), false
	}
	if !bidPresent && !askPresent {
		return rejection(frame, LiveFamilyQuote, symbol, position, LiveRejectQuotePrice), false
	}
	quote := NormalizedQuote{
		SchemaVersion: "normalized-quote-v1", BindingIdentity: frame.Binding.Identity(), TradingDate: frame.Binding.TradingDate(), Symbol: symbol,
		SIPTime: sip, ReceiptTime: frame.ReceivedAt.UTC(), BidPrice: normalizeSignedZero(bid), AskPrice: normalizeSignedZero(ask), BidPresent: bidPresent, AskPresent: askPresent,
		BidExchange: decodeOptionalInt(members, "bx", 1, math.MaxInt64), AskExchange: decodeOptionalInt(members, "ax", 1, math.MaxInt64),
		BidSize: decodeOptionalInt(members, "bs", 1, math.MaxInt64), AskSize: decodeOptionalInt(members, "as", 1, math.MaxInt64),
		Conditions: decodeConditions(members, "c", true), Indicators: decodeConditions(members, "i", false),
		Sequence: decodeOptionalInt(members, "q", 0, math.MaxInt64), Tape: decodeOptionalInt(members, "z", 1, 3), Live: position,
	}
	return LiveResult{Kind: LiveResultQuote, Position: position, Quote: quote}, false
}

func normalizeStatus(frame LiveFrame, position engine.LivePosition, members objectMembers, context *StatusContext) LiveResult {
	status := NormalizedStatus{BindingIdentity: frame.Binding.Identity(), ConnectionEpoch: frame.ConnectionEpoch, Position: position, ReceiptTime: frame.ReceivedAt.UTC(), Disposition: StatusAmbiguous, Reason: LiveRejectStatusCorrelation}
	if context != nil {
		status.CommandKind, status.CommandToken = context.CommandKind, context.CommandToken
	}
	if duplicateRecognized(members, "status") {
		return LiveResult{Kind: LiveResultStatus, Position: position, Status: status}
	}
	value, ok := rawString(members.values["status"])
	if !ok || value == "" {
		return LiveResult{Kind: LiveResultStatus, Position: position, Status: status}
	}
	switch StatusPhase(value) {
	case StatusPhaseConnected, StatusPhaseAuthSuccess, StatusPhaseSuccess:
		status.Phase = StatusPhase(value)
	case StatusPhaseAuthFailed, StatusPhaseError:
		status.Phase = StatusPhase(value)
		status.Disposition, status.Reason = StatusFailed, ""
		return LiveResult{Kind: LiveResultStatus, Position: position, Status: status}
	default:
		status.Phase = StatusPhaseUnknown
		status.Disposition, status.Reason = StatusFailed, ""
		return LiveResult{Kind: LiveResultStatus, Position: position, Status: status}
	}
	if validStatusContext(context, frame.ConnectionEpoch) && status.Phase == context.ExpectedPhase {
		status.Disposition, status.Reason = StatusAcknowledged, ""
	}
	return LiveResult{Kind: LiveResultStatus, Position: position, Status: status}
}

func validStatusContext(context *StatusContext, epoch uint64) bool {
	if context == nil || context.ConnectionEpoch != epoch || context.ExpectedCount <= 0 || len(context.CommandToken) > maximumCommandBytes {
		return false
	}
	switch context.CommandKind {
	case CommandConnection:
		return context.ExpectedPhase == StatusPhaseConnected
	case CommandAuthentication:
		return context.ExpectedPhase == StatusPhaseAuthSuccess
	case CommandAggregateSubscribe, CommandTradeQuoteSubscribe, CommandTradeQuoteUnsubscribe:
		return context.ExpectedPhase == StatusPhaseSuccess && context.CommandToken != ""
	default:
		return false
	}
}

func rawString(raw wireValue) (string, bool) {
	return raw.scalar, raw.kind == wireString && raw.classified && utf8.ValidString(raw.scalar)
}

func boundedSymbol(raw wireValue) (string, bool) {
	symbol, ok := rawString(raw)
	return symbol, ok && symbol != "" && len(symbol) <= 64
}

func bestEffortSymbol(members objectMembers) string {
	if members.counts["sym"] != 1 {
		return ""
	}
	symbol, _ := boundedSymbol(members.values["sym"])
	return symbol
}

func rawExactInt64(raw wireValue) (int64, bool) {
	return wireExactInt64(raw)
}

func wireExactInt64(raw wireValue) (int64, bool) {
	if raw.kind != wireNumber || !raw.classified || raw.scalar == "" {
		return 0, false
	}
	return exactJSONInt64(json.Number(raw.scalar))
}

func rawFiniteFloat(raw wireValue) (float64, bool) {
	if raw.kind != wireNumber || !raw.classified || raw.scalar == "" {
		return 0, false
	}
	return finiteJSONFloat(json.Number(raw.scalar))
}

func rawDecimalString(raw wireValue, positive bool, maximum float64, maximumBytes int) (float64, bool) {
	text, ok := rawString(raw)
	if !ok || len(text) == 0 || len(text) > maximumBytes || !plainDecimal(text) {
		return 0, false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || (positive && value <= 0) || (maximum > 0 && value > maximum) {
		return 0, false
	}
	return value, true
}

func plainDecimal(value string) bool {
	dot := false
	digits := 0
	for index, character := range value {
		switch {
		case character >= '0' && character <= '9':
			digits++
		case character == '.' && !dot && index > 0 && index < len(value)-1:
			dot = true
		default:
			return false
		}
	}
	return digits > 0
}

func decodeEventTime(members objectMembers, name string, frame LiveFrame) (time.Time, bool, bool) {
	raw, present := members.values[name]
	if !present {
		return time.Time{}, false, false
	}
	milliseconds, ok := rawExactInt64(raw)
	if !ok {
		return time.Time{}, true, false
	}
	value := time.UnixMilli(milliseconds).UTC()
	valid := value.UnixMilli() == milliseconds && !value.Before(frame.Binding.SessionStart()) && value.Before(frame.Binding.SessionEnd()) && !value.After(frame.ReceivedAt.UTC().Add(providerFutureSkew))
	return value, true, valid
}

func decodeOptionalInt(members objectMembers, name string, minimum, maximum int64) OptionalInt64 {
	raw, present := members.values[name]
	if !present || raw.shape == MetadataNull {
		return OptionalInt64{Present: present, Classified: !present}
	}
	value, ok := rawExactInt64(raw)
	if !ok || value < minimum || value > maximum {
		return OptionalInt64{Present: true}
	}
	return OptionalInt64{Present: true, Classified: true, Value: value}
}

func decodeConditions(members objectMembers, name string, scalarAllowed bool) BoundedIntVector {
	raw, present := members.values[name]
	if !present {
		return BoundedIntVector{Shape: MetadataAbsent, Classified: true}
	}
	if raw.shape == MetadataNull {
		return BoundedIntVector{Shape: MetadataNull, Classified: true}
	}
	if scalarAllowed {
		if value, ok := rawExactInt64(raw); ok {
			result := BoundedIntVector{Shape: MetadataScalar, Classified: true}
			result.Values[0], result.Count = value, 1
			return result
		}
	}
	if raw.shape != MetadataArray {
		return BoundedIntVector{Shape: MetadataUnclassified}
	}
	if !raw.classified {
		return BoundedIntVector{Shape: MetadataUnclassified}
	}
	// Classified means the provider shape is syntactically bounded and exact;
	// only the engine-owned C9 fixture decides semantic feature eligibility.
	result := BoundedIntVector{Shape: MetadataArray, Classified: true, Count: raw.count, Values: raw.vector}
	return result
}

func boundedLifecycleEvidence(raw wireValue) LifecycleEvidence {
	shape := raw.shape
	if raw.kind == wireNull {
		shape = MetadataScalar
	}
	evidence := LifecycleEvidence{Present: true, JSONType: shape}
	if (raw.kind == wireString || raw.kind == wireNumber || raw.kind == wireBool || raw.kind == wireNull) && raw.classified && len(raw.scalar) <= maximumScalarBytes {
		evidence.Classified, evidence.Scalar = true, raw.scalar
	}
	return evidence
}
