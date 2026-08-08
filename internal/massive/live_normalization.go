// Package massive contains provider-specific normalization that does not own
// requests, coverage, recovery, replay, or scanner state.
package massive

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	MaximumLiveFrameBytes = 8 << 20
	maximumTradeIDBytes   = 128
	maximumScalarBytes    = 32
	maximumMetadataValues = 16
	maximumCommandBytes   = 256
	providerFutureSkew    = 250 * time.Millisecond
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
	LiveFamilyAggregate LiveFamily = "A"
	LiveFamilyTrade     LiveFamily = "T"
	LiveFamilyQuote     LiveFamily = "Q"
	LiveFamilyStatus    LiveFamily = "status"
)

type LiveRejectionReason string

const (
	LiveRejectFrameBounds       LiveRejectionReason = "frame_bounds"
	LiveRejectFrameSyntax       LiveRejectionReason = "frame_syntax"
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
	ShedTradesQuotes bool
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
	IngressAmbiguity      int
	FencedRemainder       int
	FrameIngressAmbiguity int
}

func (a LiveFrameAccounting) Reconciles() bool {
	classified := a.ArrayElementsExamined == a.NormalizedAggregates+a.NormalizedTrades+
		a.NormalizedQuotes+a.NormalizedControls+a.AttributableRejected+a.IngressAmbiguity &&
		a.FrameIngressAmbiguity >= 0 && a.FrameIngressAmbiguity <= 1
	if !classified {
		return false
	}
	if a.ArrayCardinalityKnown {
		return a.DeclaredArrayElements == a.ArrayElementsExamined+a.FencedRemainder
	}
	return a.DeclaredArrayElements == 0
}

type liveFrameAnalysis struct {
	arrayKnown, statusExact  bool
	declared, ambiguityIndex int
	statusCount              int
	ambiguityReason          LiveRejectionReason
	frameAmbiguity           bool
	frameAmbiguityReason     LiveRejectionReason
}

type liveFrameCursor struct {
	frame      LiveFrame
	status     *StatusContext
	options    LiveNormalizationOptions
	decoder    *json.Decoder
	analysis   liveFrameAnalysis
	accounting LiveFrameAccounting
	index      int
	statusSeen int
	done       bool
}

// newLiveFrameCursor performs a bounded first pass for exact array/status
// cardinality, then exposes one immutable result at a time. It never retains a
// result slice or a second decoded tree.
func newLiveFrameCursor(frame LiveFrame, statusContext *StatusContext, options LiveNormalizationOptions) *liveFrameCursor {
	if !validFrameContext(frame) {
		return &liveFrameCursor{frame: frame, status: statusContext, options: options, analysis: liveFrameAnalysis{frameAmbiguity: true, frameAmbiguityReason: LiveRejectFrameBounds}}
	}
	data := frame.Data
	if !utf8.Valid(data) || len(bytes.TrimSpace(data)) == 0 {
		return &liveFrameCursor{frame: frame, status: statusContext, options: options, analysis: liveFrameAnalysis{frameAmbiguity: true, frameAmbiguityReason: LiveRejectFrameSyntax}}
	}
	analysis := analyzeLiveFrame(frame, statusContext, options)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		analysis.frameAmbiguity, analysis.frameAmbiguityReason = true, LiveRejectFrameSyntax
		return &liveFrameCursor{frame: frame, status: statusContext, options: options, analysis: analysis}
	}
	accounting := LiveFrameAccounting{ArrayCardinalityKnown: analysis.arrayKnown}
	if analysis.arrayKnown {
		accounting.DeclaredArrayElements = analysis.declared
	}
	return &liveFrameCursor{frame: frame, status: statusContext, options: options, decoder: decoder, analysis: analysis, accounting: accounting}
}

func analyzeLiveFrame(frame LiveFrame, statusContext *StatusContext, options LiveNormalizationOptions) liveFrameAnalysis {
	decoder := json.NewDecoder(bytes.NewReader(frame.Data))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		return liveFrameAnalysis{frameAmbiguity: true, frameAmbiguityReason: LiveRejectFrameSyntax}
	}
	analysis := liveFrameAnalysis{ambiguityIndex: -1, statusExact: true}
	for index := 0; decoder.More(); index++ {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			analysis.ambiguityIndex, analysis.ambiguityReason, analysis.statusExact = index, LiveRejectFrameSyntax, false
			return analysis
		}
		analysis.declared++
		if analysis.ambiguityIndex >= 0 {
			continue
		}
		position := engine.LivePosition{ConnectionEpoch: frame.ConnectionEpoch, FrameSequence: frame.FrameSequence, ArrayIndex: uint32(index)}
		result, ambiguous := normalizeElement(frame, position, raw, statusContext, options)
		if result.Kind == LiveResultStatus {
			analysis.statusCount++
		}
		if ambiguous {
			analysis.ambiguityIndex, analysis.ambiguityReason, analysis.statusExact = index, result.Rejection.Reason, false
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim(']') {
		analysis.ambiguityIndex, analysis.ambiguityReason, analysis.statusExact = analysis.declared, LiveRejectFrameSyntax, false
		return analysis
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		analysis.frameAmbiguity, analysis.frameAmbiguityReason, analysis.statusExact = true, LiveRejectFrameSyntax, false
	}
	analysis.arrayKnown = true
	return analysis
}

func validFrameContext(frame LiveFrame) bool {
	return frame.ConnectionEpoch > 0 && frame.FrameSequence > 0 && !frame.ReceivedAt.IsZero() &&
		frame.ReceivedAt == frame.ReceivedAt.UTC() &&
		len(frame.Data) <= MaximumLiveFrameBytes && frame.Binding.Identity() != "" &&
		frame.Binding.TradingDate() != "" && !frame.Binding.SessionStart().IsZero() &&
		frame.Binding.SessionStart().Before(frame.Binding.SessionEnd())
}

func (c *liveFrameCursor) Next() (LiveResult, bool) {
	if c.done {
		return LiveResult{}, false
	}
	position := engine.LivePosition{ConnectionEpoch: c.frame.ConnectionEpoch, FrameSequence: c.frame.FrameSequence, ArrayIndex: uint32(c.index)}
	if c.decoder == nil {
		c.done = true
		c.accounting.FrameIngressAmbiguity = 1
		return ambiguity(position, c.analysis.frameAmbiguityReason), true
	}
	if c.analysis.ambiguityIndex == c.index {
		c.done = true
		c.accounting.ArrayElementsExamined++
		c.accounting.IngressAmbiguity++
		if c.analysis.arrayKnown {
			c.accounting.FencedRemainder = c.analysis.declared - c.index - 1
		}
		return ambiguity(position, c.analysis.ambiguityReason), true
	}
	if c.decoder.More() {
		var raw json.RawMessage
		if err := c.decoder.Decode(&raw); err != nil {
			c.done = true
			c.accounting.ArrayElementsExamined++
			c.accounting.IngressAmbiguity++
			return ambiguity(position, LiveRejectFrameSyntax), true
		}
		result, ambiguous := normalizeElement(c.frame, position, raw, c.status, c.options)
		c.index++
		c.accounting.ArrayElementsExamined++
		switch result.Kind {
		case LiveResultAggregate:
			c.accounting.NormalizedAggregates++
		case LiveResultTrade:
			c.accounting.NormalizedTrades++
		case LiveResultQuote:
			c.accounting.NormalizedQuotes++
		case LiveResultStatus:
			c.accounting.NormalizedControls++
			c.statusSeen++
			result.Status.ObservedCount = c.analysis.statusCount
			result.Status.FinalInFrame = c.statusSeen == c.analysis.statusCount
			if !c.analysis.statusExact || c.status == nil || c.analysis.statusCount != c.status.ExpectedCount {
				if result.Status.Disposition == StatusAcknowledged {
					result.Status.Disposition, result.Status.Reason = StatusAmbiguous, LiveRejectStatusCorrelation
				}
			}
		case LiveResultRejected:
			c.accounting.AttributableRejected++
		case LiveResultAmbiguous:
			c.accounting.IngressAmbiguity++
		}
		if ambiguous {
			c.done = true
			if c.analysis.arrayKnown {
				c.accounting.FencedRemainder = c.analysis.declared - c.index
			}
		}
		return result, true
	}
	_, _ = c.decoder.Token()
	c.done = true
	if c.analysis.frameAmbiguity {
		c.accounting.FrameIngressAmbiguity = 1
		return ambiguity(position, c.analysis.frameAmbiguityReason), true
	}
	return LiveResult{}, false
}

func (c *liveFrameCursor) Accounting() LiveFrameAccounting { return c.accounting }

type objectMembers struct {
	values map[string]json.RawMessage
	counts map[string]int
}

func decodeObjectMembers(raw json.RawMessage) (objectMembers, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return objectMembers{}, false
	}
	members := objectMembers{values: make(map[string]json.RawMessage), counts: make(map[string]int)}
	for decoder.More() {
		nameToken, err := decoder.Token()
		name, named := nameToken.(string)
		if err != nil || !named {
			return objectMembers{}, false
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return objectMembers{}, false
		}
		members.counts[name]++
		if members.counts[name] == 1 {
			members.values[name] = append(json.RawMessage(nil), value...)
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return objectMembers{}, false
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return objectMembers{}, false
	}
	return members, true
}

func normalizeElement(frame LiveFrame, position engine.LivePosition, raw json.RawMessage, statusContext *StatusContext, options LiveNormalizationOptions) (LiveResult, bool) {
	members, ok := decodeObjectMembers(raw)
	if !ok || members.counts["ev"] != 1 {
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
			return rejection(frame, LiveFamilyTrade, "", position, LiveRejectOptionalShed), false
		}
		return normalizeTrade(frame, position, members)
	case string(LiveFamilyQuote):
		if options.ShedTradesQuotes {
			return rejection(frame, LiveFamilyQuote, "", position, LiveRejectOptionalShed), false
		}
		return normalizeQuote(frame, position, members)
	case string(LiveFamilyStatus):
		return normalizeStatus(frame, position, members, statusContext), false
	default:
		return ambiguity(position, LiveRejectEventFamily), true
	}
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
	default:
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

func rawString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || !utf8.ValidString(value) {
		return "", false
	}
	return value, true
}

func boundedSymbol(raw json.RawMessage) (string, bool) {
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

func rawExactInt64(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	return exactJSONInt64(json.Number(raw))
}

func rawFiniteFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	return finiteJSONFloat(json.Number(raw))
}

func rawDecimalString(raw json.RawMessage, positive bool, maximum float64, maximumBytes int) (float64, bool) {
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
	if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
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
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return BoundedIntVector{Shape: MetadataNull, Classified: true}
	}
	if scalarAllowed {
		if value, ok := rawExactInt64(raw); ok {
			result := BoundedIntVector{Shape: MetadataScalar, Classified: true}
			result.Values[0], result.Count = value, 1
			return result
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		return BoundedIntVector{Shape: MetadataUnclassified}
	}
	result := BoundedIntVector{Shape: MetadataArray}
	for decoder.More() {
		if result.Count == maximumMetadataValues {
			return BoundedIntVector{Shape: MetadataUnclassified}
		}
		var number json.Number
		if err := decoder.Decode(&number); err != nil {
			return BoundedIntVector{Shape: MetadataUnclassified}
		}
		value, ok := exactJSONInt64(number)
		if !ok {
			return BoundedIntVector{Shape: MetadataUnclassified}
		}
		result.Values[result.Count] = value
		result.Count++
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim(']') {
		return BoundedIntVector{Shape: MetadataUnclassified}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return BoundedIntVector{Shape: MetadataUnclassified}
	}
	// Classified means the provider shape is syntactically bounded and exact;
	// only the engine-owned C9 fixture decides semantic feature eligibility.
	result.Classified = true
	return result
}

func boundedLifecycleEvidence(raw json.RawMessage) LifecycleEvidence {
	evidence := LifecycleEvidence{Present: true, JSONType: MetadataUnclassified}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return evidence
	}
	switch trimmed[0] {
	case '"':
		evidence.JSONType = MetadataScalar
		value, ok := rawString(raw)
		if ok && len(value) <= maximumScalarBytes {
			evidence.Classified, evidence.Scalar = true, value
		}
	case 'n', 't', 'f', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		evidence.JSONType = MetadataScalar
		if len(trimmed) <= maximumScalarBytes {
			evidence.Classified, evidence.Scalar = true, string(trimmed)
		}
	case '[':
		evidence.JSONType = MetadataArray
	case '{':
		evidence.JSONType = MetadataObject
	}
	return evidence
}
