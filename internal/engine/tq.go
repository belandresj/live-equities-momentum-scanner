package engine

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"
)

const TQSchemaV1 = "engine-tq-v1"

// TQControlErrorInput is the one bounded post-handshake provider error fact.
// Generic success and late status messages are intentionally not inputs.
type TQControlErrorInput struct {
	SchemaVersion, BindingIdentity string
	ConnectionEpoch                uint64
	Position                       LivePosition
	ReceiptTime                    time.Time
}

type frozenTQControlErrorInput struct{ TQControlErrorInput }

const (
	maximumTQSymbols          = 20
	maximumTradesPerSymbol    = 10_000
	maximumTradeFingerprints  = 25_000
	maximumTradesGlobal       = 100_000
	maximumFingerprintsGlobal = 400_000
	maximumTQBytesPerSymbol   = 4 << 20
	maximumTQBytesGlobal      = 64 << 20
	tqMemberByteCharge        = 1 << 10
	tqTradeByteCharge         = 64
	tqFingerprintByteCharge   = 384
	tqQuoteByteCharge         = 192
	spreadStaleAge            = 2 * time.Second
)

type tqRetentionLimits struct {
	tradesPerSymbol, tradesGlobal, fingerprintsPerSymbol, fingerprintsGlobal int
	bytesPerSymbol, bytesGlobal                                              int64
}

func defaultTQRetentionLimits() tqRetentionLimits {
	return tqRetentionLimits{
		tradesPerSymbol: maximumTradesPerSymbol, tradesGlobal: maximumTradesGlobal,
		fingerprintsPerSymbol: maximumTradeFingerprints, fingerprintsGlobal: maximumFingerprintsGlobal,
		bytesPerSymbol: maximumTQBytesPerSymbol, bytesGlobal: maximumTQBytesGlobal,
	}
}

type TQAction string

const (
	TQSubscribe   TQAction = "subscribe"
	TQUnsubscribe TQAction = "unsubscribe"
)

type TQCommand struct {
	bindingIdentity string
	connectionEpoch uint64
	commandToken    uint64
	action          TQAction
	symbols         [maximumTQSymbols]string
	symbolCount     int
	done            chan Disposition
}

type TQCommandResultInput struct {
	command     TQCommand
	position    LivePosition
	receiptTime time.Time
	outcome     ConnectionControlOutcome
}

func (c TQCommand) BindingIdentity() string { return c.bindingIdentity }
func (c TQCommand) ConnectionEpoch() uint64 { return c.connectionEpoch }
func (c TQCommand) CommandToken() uint64    { return c.commandToken }
func (c TQCommand) Action() TQAction        { return c.action }
func (c TQCommand) Symbol() string {
	if c.symbolCount == 0 {
		return ""
	}
	return c.symbols[0]
}
func (c TQCommand) Symbols() []string { return append([]string(nil), c.symbols[:c.symbolCount]...) }

func NewTQCommandResultInput(command TQCommand, position LivePosition, receiptTime time.Time, outcome ConnectionControlOutcome) (TQCommandResultInput, error) {
	hasPosition := position.ConnectionEpoch == command.connectionEpoch
	if !validTQCommand(command) || receiptTime.IsZero() || receiptTime != receiptTime.UTC() || !validConnectionControlOutcome(outcome) ||
		!(outcome != ControlSucceeded && position == (LivePosition{}) || hasPosition) {
		return TQCommandResultInput{}, errors.New("invalid T/Q command result")
	}
	return TQCommandResultInput{command: command, position: position, receiptTime: receiptTime, outcome: outcome}, nil
}

type TradeInput struct {
	SchemaVersion, BindingIdentity, TradingDate string
	Symbol, TradeID                             string
	Exchange                                    int64
	TRFPresent                                  bool
	TRFID                                       int64
	Price, EconomicSize                         float64
	EventTime, ReceiptTime                      time.Time
	TimestampBasis                              string
	Conditions                                  []int64
	ConditionsClassified, IdentityClassified    bool
	Lifecycle                                   string
	Live                                        LivePosition
}

type QuoteInput struct {
	SchemaVersion, BindingIdentity, TradingDate string
	Symbol                                      string
	SIPTime, ReceiptTime                        time.Time
	BidPrice, AskPrice                          float64
	BidPresent, AskPresent                      bool
	Conditions, Indicators                      []int64
	ConditionsClassified, IndicatorsClassified  bool
	Live                                        LivePosition
}

type TQDropInput struct {
	SchemaVersion, BindingIdentity, TradingDate string
	Family, Symbol, DropReason                  string
	Live                                        LivePosition
}

type frozenTQCommandResultInput struct{ TQCommandResultInput }
type frozenTradeInput struct{ TradeInput }
type frozenQuoteInput struct{ QuoteInput }
type frozenTQDropInput struct{ TQDropInput }

type tqCoverage struct {
	active                  bool
	epoch                   uint64
	start                   time.Time
	boundary, ack, greatest LivePosition // ack is retained only for legacy test construction.
}

type tqTrade struct {
	at         time.Time
	qualifying bool
	basis      string
	lifecycle  bool
}

type tqTradeIdentity struct {
	exchange   int64
	trfID      int64
	tradeID    string
	trfPresent bool
}

type tqTradeFingerprint struct {
	eventUnixNano                    int64
	priceBits, economicSizeBits      uint64
	conditions                       [16]int64
	conditionCount                   uint8
	basis, lifecycle                 uint8
	conditionsClassified, identified bool
}

type tqFingerprint struct {
	value        tqTradeFingerprint
	firstReceipt time.Time
}

type tqQuote struct {
	at, receipt            time.Time
	position               LivePosition
	bid, ask               float64
	bidPresent, askPresent bool
	quality                string
}

type tqSymbolState struct {
	// requested is local write evidence. present is deliberately stronger: both
	// channels have supplied post-write data for this generation.
	requested, present, unknown, cleanupAttempted bool
	generation                                    uint64
	bound                                         bool
	resetRequired                                 bool
	tradeCoverage, quoteCoverage                  tqCoverage
	trades                                        []tqTrade
	latestQuote, latestValidQuote                 *tqQuote
	fingerprints                                  map[tqTradeIdentity]tqFingerprint
	unequalRepeat                                 bool
	retainedBytes                                 int64
}

type tqState struct {
	epoch uint64
	// canonicalMutations advances for every ordered T/Q/time fact that is
	// classified on the engine path. It is deliberately not part of the global
	// publication fingerprint: canonical progress is synchronous even when the
	// public projection waits for a cadence boundary.
	canonicalMutations uint64
	// publicProjectionRevision advances only for a trust transition that must
	// replace the combined immutable publication immediately.
	publicProjectionRevision                                                              uint64
	projectionDirty                                                                       bool
	projectionDirtyMutations                                                              uint64
	projectionDirtied                                                                     uint64
	projectionFlushedByCadence                                                            uint64
	immediateTrustTransitionPublications                                                  uint64
	immediateTrustTransitionMutations                                                     uint64
	coalescedOrdinaryMutations                                                            uint64
	immediateProjectionPending                                                            bool
	nextToken                                                                             uint64
	nextGeneration                                                                        uint64
	desired                                                                               []string
	cadenceDesired                                                                        []string
	members                                                                               map[string]*tqSymbolState
	pending                                                                               *TQCommand
	dispatched                                                                            bool
	additionPermit, freshEpoch, restoring                                                 bool
	consumed, applied                                                                     uint64
	duplicate, rejected                                                                   uint64
	fenced, pressureShed                                                                  uint64
	tradeApplied, quoteApplied, tradePressureShed, quotePressureShed                      uint64
	integrity                                                                             uint64
	tradeCount, quoteCount                                                                int
	fingerprintCount                                                                      int
	retainedBytes                                                                         int64
	globalBound                                                                           bool
	aggregateOnly                                                                         bool
	pressure                                                                              tqPressureState
	commandsIssued, commandsWritten, commandsFailed, commandsFenced, commandResultsFenced uint64
	controlClosed                                                                         bool
	controlClosedEpoch                                                                    uint64
	controlClosedPosition                                                                 LivePosition
	controlErrors, controlErrorsFenced                                                    uint64
}

type TQFieldStatus string

const (
	TQUnselected   TQFieldStatus = "unselected"
	TQWarming      TQFieldStatus = "warming"
	TQCurrent      TQFieldStatus = "current"
	TQStale        TQFieldStatus = "stale"
	TQUnavailable  TQFieldStatus = "unavailable"
	TQInvalid      TQFieldStatus = "invalid"
	TQPressureShed TQFieldStatus = "pressure_shed"
)

type TapeRateView struct {
	Status                   TQFieldStatus
	Reason                   string
	FiveSecond               float64
	TimestampBasis           string
	LifecycleRecordsObserved bool
}

type SpreadView struct {
	Status             TQFieldStatus
	Reason             string
	Cents, BasisPoints float64
	QuoteAge           time.Duration
	Quality            string
	ObservedAt         time.Time
}

type TQSymbolView struct {
	Symbol                       string
	Desired, ProviderPresent     bool
	ProviderMembershipUnknown    bool
	TradeCoverage, QuoteCoverage bool
	Tape                         TapeRateView
	Spread                       SpreadView
}

type TQAccountingView struct {
	Consumed, Applied, Duplicate, Rejected, Fenced, PressureShed, Integrity uint64
	AppliedTrades, AppliedQuotes, PressureShedTrades, PressureShedQuotes    uint64
	KnownPresent, KnownAbsent, Unknown                                      int
	RetainedTrades, RetainedQuotes, RetainedFingerprints                    int
	RetainedBytes                                                           int64
}

type TQCommandAccountingView struct {
	Issued, Pending, Written, Failed, Fenced, ResultFenced uint64
}

type TQPressureSampleView struct {
	Observed                     bool
	WaitingFrames, FrameCapacity uint64
	WaitingBytes, ByteCapacity   uint64
	OldestWaitingFrameAge        time.Duration
	AggregateWatermarkLag        time.Duration
	RecoveryHealthy              bool
}

type TQView struct {
	PublicationID                       uint64
	Desired                             []string
	CommandPending                      bool
	PendingAction                       TQAction
	PendingSymbol                       string
	Rows                                []TQSymbolView
	Bounds                              bool
	AggregateOnly                       bool
	Pressure                            TQPressureMode
	PressureCause                       TQPressureCause
	ShedTradesQuotes                    bool
	PressureMisses                      uint32
	PressureTransitions, PressureFenced uint64
	PressureSample                      TQPressureSampleView
	RecoveryHealthySamples              uint8
	RecoveryRequiredSamples             uint8
	Accounting                          TQAccountingView
	Commands                            TQCommandAccountingView
	ControlClosed                       bool
	ControlClosedPosition               LivePosition
	ControlErrors, ControlErrorsFenced  uint64
}

// TQPublicationAccountingView is fixed-cardinality engine observation for the
// publication-coalescing proof. It is intentionally outside the strict v2
// snapshot schema: the counters describe projection mechanics, not market
// state or product fields.
type TQPublicationAccountingView struct {
	CanonicalMutations                   uint64
	ProjectionDirtied                    uint64
	ProjectionFlushedByCadence           uint64
	ImmediateTrustTransitionPublications uint64
	ImmediateTrustTransitionMutations    uint64
	CoalescedOrdinaryMutations           uint64
	PendingProjectionMutations           uint64
	ProjectionDirty                      bool
}

// ObserveTQPublicationAccounting returns bounded publication-coalescing
// diagnostics from the sole engine owner. It never returns mutable state and
// does not create a second publication or T/Q authority.
func (e *Engine) ObserveTQPublicationAccounting() TQPublicationAccountingView {
	if e == nil {
		return TQPublicationAccountingView{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.state.tq
	return TQPublicationAccountingView{
		CanonicalMutations:                   s.canonicalMutations,
		ProjectionDirtied:                    s.projectionDirtied,
		ProjectionFlushedByCadence:           s.projectionFlushedByCadence,
		ImmediateTrustTransitionPublications: s.immediateTrustTransitionPublications,
		ImmediateTrustTransitionMutations:    s.immediateTrustTransitionMutations,
		CoalescedOrdinaryMutations:           s.coalescedOrdinaryMutations,
		PendingProjectionMutations:           s.projectionDirtyMutations,
		ProjectionDirty:                      s.projectionDirty,
	}
}

func (e *Engine) AdmitTQControlError(ctx context.Context, input TQControlErrorInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validTQControlErrorInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQControlError, tqControlError: frozenTQControlErrorInput{input}}, false)
}

func validTQControlErrorInput(v TQControlErrorInput) bool {
	hasPosition := v.Position != (LivePosition{})
	validPosition := v.Position.ConnectionEpoch == v.ConnectionEpoch && v.Position.FrameSequence > 0
	return v.SchemaVersion == TQSchemaV1 && validIdentityShape(v.BindingIdentity) && v.ConnectionEpoch > 0 &&
		!v.ReceiptTime.IsZero() && v.ReceiptTime == v.ReceiptTime.UTC() && (!hasPosition || validPosition)
}

func (e *Engine) AdmitTQCommandResult(ctx context.Context, input TQCommandResultInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validTQCommand(input.command) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQCommandResult, tqCommandResult: frozenTQCommandResultInput{input}}, false)
}

func (e *Engine) AdmitTrade(ctx context.Context, input TradeInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validTradeInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	frozen := input
	frozen.Conditions = append([]int64(nil), input.Conditions...)
	sort.Slice(frozen.Conditions, func(i, j int) bool { return frozen.Conditions[i] < frozen.Conditions[j] })
	return e.admit(ctx, &queueNode{kind: inputTrade, trade: frozenTradeInput{frozen}}, false)
}

func (e *Engine) AdmitQuote(ctx context.Context, input QuoteInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validQuoteInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	frozen := input
	frozen.Conditions, frozen.Indicators = append([]int64(nil), input.Conditions...), append([]int64(nil), input.Indicators...)
	return e.admit(ctx, &queueNode{kind: inputQuote, quote: frozenQuoteInput{frozen}}, false)
}

func (e *Engine) AdmitTQDrop(ctx context.Context, input TQDropInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || input.SchemaVersion != TQSchemaV1 || !validIdentityShape(input.BindingIdentity) || (input.Family != "T" && input.Family != "Q") ||
		input.TradingDate == "" || len(input.Symbol) > maximumSymbolBytes || input.DropReason == "" || len(input.DropReason) > 64 || input.Live.ConnectionEpoch == 0 || input.Live.FrameSequence == 0 {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQDrop, tqDrop: frozenTQDropInput{input}}, false)
}

func validTQCommand(v TQCommand) bool {
	return validIdentityShape(v.bindingIdentity) && v.connectionEpoch > 0 && v.commandToken > 0 &&
		(v.action == TQSubscribe || v.action == TQUnsubscribe) && validTQCommandSymbols(v) && v.done != nil
}

func validTQCommandSymbols(v TQCommand) bool {
	if v.symbolCount <= 0 || v.symbolCount > maximumTQSymbols {
		return false
	}
	for index := 0; index < v.symbolCount; index++ {
		if v.symbols[index] == "" || len(v.symbols[index]) > maximumSymbolBytes || index > 0 && v.symbols[index] <= v.symbols[index-1] {
			return false
		}
	}
	for index := v.symbolCount; index < len(v.symbols); index++ {
		if v.symbols[index] != "" {
			return false
		}
	}
	return true
}

func validTradeInput(v TradeInput) bool {
	return v.SchemaVersion == TQSchemaV1 && validIdentityShape(v.BindingIdentity) && v.TradingDate != "" && v.Symbol != "" && len(v.Symbol) <= maximumSymbolBytes &&
		v.TradeID != "" && len(v.TradeID) <= 128 && v.Exchange > 0 && (!v.TRFPresent || v.TRFID > 0) && v.Price > 0 && v.EconomicSize > 0 &&
		!math.IsNaN(v.Price) && !math.IsInf(v.Price, 0) && !math.IsNaN(v.EconomicSize) && !math.IsInf(v.EconomicSize, 0) &&
		!v.EventTime.IsZero() && v.EventTime == v.EventTime.UTC() && !v.ReceiptTime.IsZero() && v.ReceiptTime == v.ReceiptTime.UTC() &&
		(v.TimestampBasis == "participant" || v.TimestampBasis == "sip_fallback") && len(v.Conditions) <= 16 &&
		(v.Lifecycle == "original" || v.Lifecycle == "unsupported_lifecycle") && v.Live.ConnectionEpoch > 0 && v.Live.FrameSequence > 0
}

func validQuoteInput(v QuoteInput) bool {
	return v.SchemaVersion == TQSchemaV1 && validIdentityShape(v.BindingIdentity) && v.TradingDate != "" && v.Symbol != "" && len(v.Symbol) <= maximumSymbolBytes &&
		!v.SIPTime.IsZero() && v.SIPTime == v.SIPTime.UTC() && !v.ReceiptTime.IsZero() && v.ReceiptTime == v.ReceiptTime.UTC() &&
		(v.BidPresent || v.AskPresent) && (!v.BidPresent || v.BidPrice > 0 && !math.IsNaN(v.BidPrice) && !math.IsInf(v.BidPrice, 0)) &&
		(!v.AskPresent || v.AskPrice > 0 && !math.IsNaN(v.AskPrice) && !math.IsInf(v.AskPrice, 0)) && len(v.Conditions) <= 16 && len(v.Indicators) <= 16 &&
		v.Live.ConnectionEpoch > 0 && v.Live.FrameSequence > 0
}

func (e *Engine) reconcileTQLocked(now time.Time, cadence ...bool) {
	s := &e.state.tq
	cadenceBoundary := len(cadence) == 0 || cadence[0]
	if s.members == nil {
		s.members = make(map[string]*tqSymbolState)
		s.nextToken = 1
		s.nextGeneration = 1
	}
	if s.epoch != e.state.liveEpoch {
		previous := *s
		commandFenced := previous.commandsFenced
		if previous.pending != nil {
			commandFenced++
		}
		pressure := previous.pressure
		pressure.pending, pressure.dispatched = nil, false
		pressure.capacityObserved, pressure.lastSlotDrops, pressure.lastByteDrops = false, 0, 0
		pressure.nextSequence = maxUint64(1, pressure.nextSequence)
		*s = tqState{
			epoch: e.state.liveEpoch, canonicalMutations: previous.canonicalMutations, publicProjectionRevision: previous.publicProjectionRevision,
			projectionDirty: previous.projectionDirty, projectionDirtyMutations: previous.projectionDirtyMutations,
			projectionDirtied: previous.projectionDirtied, projectionFlushedByCadence: previous.projectionFlushedByCadence,
			immediateTrustTransitionPublications: previous.immediateTrustTransitionPublications,
			immediateTrustTransitionMutations:    previous.immediateTrustTransitionMutations,
			coalescedOrdinaryMutations:           previous.coalescedOrdinaryMutations,
			immediateProjectionPending:           previous.immediateProjectionPending,
			nextToken:                            maxUint64(e.state.aggregateWriteToken+1, maxUint64(1, previous.nextToken)), nextGeneration: 1, members: make(map[string]*tqSymbolState),
			consumed: previous.consumed, applied: previous.applied, duplicate: previous.duplicate, rejected: previous.rejected, fenced: previous.fenced,
			pressureShed: previous.pressureShed, integrity: previous.integrity, tradeApplied: previous.tradeApplied, quoteApplied: previous.quoteApplied,
			tradePressureShed: previous.tradePressureShed, quotePressureShed: previous.quotePressureShed,
			globalBound: previous.globalBound, aggregateOnly: previous.aggregateOnly,
			commandsIssued: previous.commandsIssued, commandsWritten: previous.commandsWritten, commandsFailed: previous.commandsFailed,
			commandsFenced: commandFenced, commandResultsFenced: previous.commandResultsFenced,
			pressure:      pressure,
			freshEpoch:    true,
			controlErrors: previous.controlErrors, controlErrorsFenced: previous.controlErrorsFenced,
		}
		if previousTQPublicationState(previous) {
			e.markTQTrustTransitionLocked()
		}
	}
	if s.nextToken <= e.state.aggregateWriteToken {
		s.nextToken = e.state.aggregateWriteToken + 1
	}
	desired := make([]string, 0, maximumTQSymbols)
	if e.state.aggregateEvaluator.current.mode == rankingQualifiedCurrent {
		for _, row := range e.state.aggregateEvaluator.current.rows {
			if len(desired) == maximumTQSymbols {
				break
			}
			desired = append(desired, row.symbol)
		}
	}
	previousDesired := s.desired
	s.desired = desired
	if !equalTQSymbols(previousDesired, desired) {
		e.markTQTrustTransitionLocked()
	}
	if cadenceBoundary || s.freshEpoch {
		s.cadenceDesired = append(s.cadenceDesired[:0], desired...)
		s.additionPermit = true
		s.freshEpoch = false
	}
	target := now
	if e.state.committedT != nil {
		target = *e.state.committedT
	}
	s.prune(target, now)
	desiredSet := make(map[string]bool, len(desired))
	for _, symbol := range desired {
		desiredSet[symbol] = true
	}
	for symbol, member := range s.members {
		if !desiredSet[symbol] {
			member.tradeCoverage.active, member.quoteCoverage.active = false, false
			// A removal invalidates this generation before its best-effort wire
			// unsubscribe completes. If rank churn selects it again first, the
			// existing removal is completed before a new write can allocate a
			// boundary, so old traffic cannot confirm the return.
			member.resetRequired = member.requested || member.present || member.unknown
			member.present, member.unknown = false, member.requested
			if !member.requested {
				member.bound = false
				if len(member.trades) == 0 && member.latestQuote == nil && member.latestValidQuote == nil && len(member.fingerprints) == 0 {
					s.deleteMember(symbol)
				}
			}
		}
	}
	if s.pending != nil && s.pending.action == TQSubscribe && !s.dispatched {
		for _, symbol := range s.pending.symbols[:s.pending.symbolCount] {
			if !desiredSet[symbol] {
				s.pending, s.dispatched = nil, false
				s.commandsFenced++
				break
			}
		}
	}
	if s.pending != nil || s.controlClosed || !e.state.liveEpochActive || e.state.lifecycle != lifecycleLive {
		return
	}
	removals := make([]string, 0, maximumTQSymbols)
	for symbol, member := range s.members {
		if (member.requested || member.present || member.unknown) && (s.aggregateOnly || member.bound || member.resetRequired || !desiredSet[symbol]) {
			removals = append(removals, symbol)
		}
	}
	if len(removals) > 0 {
		e.issueTQCommandLocked(TQUnsubscribe, removals)
		return
	}
	if s.aggregateOnly {
		return
	}
	if s.pressure.mode == TQPressureDegraded || !s.additionPermit {
		return
	}
	wireMembers := 0
	for _, member := range s.members {
		if member.requested || member.present || member.unknown {
			wireMembers++
		}
	}
	missing := make([]string, 0, len(desired))
	cadenceSet := make(map[string]bool, len(s.cadenceDesired))
	for _, symbol := range s.cadenceDesired {
		cadenceSet[symbol] = true
	}
	for _, symbol := range desired {
		if !cadenceSet[symbol] {
			continue
		}
		member := s.member(symbol)
		if member.bound {
			continue
		}
		if !member.requested {
			if wireMembers+len(missing) < maximumTQSymbols {
				missing = append(missing, symbol)
			}
		}
	}
	if len(missing) > 0 {
		s.additionPermit = false
		if s.restoring {
			e.issueTQCommandLocked(TQSubscribe, missing[:1])
		} else {
			e.issueTQCommandLocked(TQSubscribe, missing)
		}
		return
	}
	s.additionPermit = false
	s.restoring = false
	_ = now
}

func (s *tqState) member(symbol string) *tqSymbolState {
	m := s.members[symbol]
	if m == nil {
		m = &tqSymbolState{fingerprints: make(map[tqTradeIdentity]tqFingerprint), retainedBytes: tqMemberByteCharge}
		s.members[symbol] = m
		s.retainedBytes += tqMemberByteCharge
	}
	return m
}

func (s *tqState) deleteMember(symbol string) {
	m := s.members[symbol]
	if m == nil {
		return
	}
	s.releaseMember(m)
	s.retainedBytes -= m.retainedBytes
	m.retainedBytes = 0
	delete(s.members, symbol)
}

func previousTQPublicationState(s tqState) bool {
	if len(s.desired) > 0 || s.pending != nil || s.controlClosed || s.aggregateOnly || s.globalBound {
		return true
	}
	for _, member := range s.members {
		if member.requested || member.present || member.unknown || member.tradeCoverage.active || member.quoteCoverage.active {
			return true
		}
	}
	return false
}

func equalTQSymbols(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// markTQTrustTransitionLocked records that the current ordered input changed
// whether a T/Q field or membership claim can be trusted. The mutation is
// materialized into the public projection once at the end of the input.
func (e *Engine) markTQTrustTransitionLocked() {
	e.state.tq.immediateProjectionPending = true
}

// recordTQProjectionInputLocked accounts one ordered T/Q/time input after all
// of its canonical mutation and reconciliation work has completed. Ordinary
// inputs dirtied here do not advance publicProjectionRevision.
func (e *Engine) recordTQProjectionInputLocked() {
	s := &e.state.tq
	s.canonicalMutations++
	s.projectionDirtied++
	s.projectionDirty = true
	s.projectionDirtyMutations++
	if s.immediateProjectionPending {
		s.publicProjectionRevision++
		s.immediateTrustTransitionMutations++
	} else {
		s.coalescedOrdinaryMutations++
	}
}

// flushTQProjectionLocked clears only the dirty work represented by the
// combined publication just stored. The sole engine owner holds the lock, so
// no later input can be cleared accidentally.
func (e *Engine) flushTQProjectionLocked() {
	s := &e.state.tq
	if !s.projectionDirty {
		return
	}
	if s.immediateProjectionPending {
		s.immediateTrustTransitionPublications++
	} else {
		s.projectionFlushedByCadence++
	}
	s.projectionDirty = false
	s.projectionDirtyMutations = 0
	s.immediateProjectionPending = false
}

func (s *tqState) releaseMember(m *tqSymbolState) {
	s.tradeCount -= len(m.trades)
	s.quoteCount -= retainedQuoteCount(m)
	s.fingerprintCount -= len(m.fingerprints)
	released := int64(len(m.trades))*tqTradeByteCharge + int64(len(m.fingerprints))*tqFingerprintByteCharge + int64(retainedQuoteCount(m))*tqQuoteByteCharge
	s.retainedBytes -= released
	m.retainedBytes -= released
	m.trades, m.latestQuote, m.latestValidQuote = nil, nil, nil
	m.fingerprints = make(map[tqTradeIdentity]tqFingerprint)
}

func (s *tqState) prune(target, engineTime time.Time) {
	tradeCutoff := target.Add(-5 * time.Second)
	// A stalled aggregate watermark cannot pin future-to-T contributions
	// forever. Thirty seconds is the maximum accepted T/Q disorder horizon;
	// once monotonic engine time passes it, an event cannot participate in the
	// next honest five-second projection after recovery.
	engineTradeCutoff := engineTime.Add(-30 * time.Second)
	if engineTradeCutoff.After(tradeCutoff) {
		tradeCutoff = engineTradeCutoff
	}
	for _, m := range s.members {
		trades := m.trades[:0]
		for _, item := range m.trades {
			if item.at.Before(tradeCutoff) {
				s.tradeCount--
				s.retainedBytes -= tqTradeByteCharge
				m.retainedBytes -= tqTradeByteCharge
				continue
			}
			trades = append(trades, item)
		}
		m.trades = trades
		for key, item := range m.fingerprints {
			// Equality remains retained. Cleanup occurs only on the first engine
			// tick strictly after first receipt plus thirty seconds.
			if engineTime.After(item.firstReceipt.Add(30 * time.Second)) {
				delete(m.fingerprints, key)
				s.fingerprintCount--
				s.retainedBytes -= tqFingerprintByteCharge
				m.retainedBytes -= tqFingerprintByteCharge
			}
		}
	}
}

func (e *Engine) issueTQCommandLocked(action TQAction, symbols []string) {
	s := &e.state.tq
	if s.nextToken == 0 || len(symbols) == 0 || len(symbols) > maximumTQSymbols {
		e.enterTQGlobalBoundLocked()
		return
	}
	symbols = append([]string(nil), symbols...)
	sort.Strings(symbols)
	command := &TQCommand{bindingIdentity: e.state.binding.identity, connectionEpoch: e.state.liveEpoch, commandToken: s.nextToken, action: action, symbolCount: len(symbols), done: make(chan Disposition, 1)}
	copy(command.symbols[:], symbols)
	s.nextToken++
	s.pending, s.dispatched = command, false
	s.commandsIssued++
	if action == TQUnsubscribe {
		for _, symbol := range symbols {
			m := s.member(symbol)
			m.tradeCoverage.active, m.quoteCoverage.active = false, false
		}
	}
}

// IssueTQCommand transfers the exact current opaque command to the C5
// composition once. It does not change membership or coverage.
func (e *Engine) IssueTQCommand() (TQCommand, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &e.state.tq
	if s.pending == nil || s.dispatched || !validTQCommand(*s.pending) || (s.aggregateOnly && s.pending.action == TQSubscribe) {
		return TQCommand{}, errors.New("T/Q command is not issuable")
	}
	s.dispatched = true
	return *s.pending, nil
}

func (e *Engine) enterTQGlobalBoundLocked() {
	if !e.state.tq.globalBound {
		e.markTQTrustTransitionLocked()
	}
	e.state.tq.globalBound = true
	e.state.tq.pressure.cause = TQPressureCauseRetentionBound
	e.enterTQAggregateOnlyLocked()
}

func (e *Engine) enterTQAggregateOnlyLocked() {
	s := &e.state.tq
	wasTrustworthy := !s.aggregateOnly && s.pressure.mode != TQPressureAggregateOnly
	if !wasTrustworthy {
		for _, member := range s.members {
			if member.present || member.tradeCoverage.active || member.quoteCoverage.active {
				wasTrustworthy = true
				break
			}
		}
	}
	if wasTrustworthy {
		e.markTQTrustTransitionLocked()
	}
	s.pressure.streaks.healthy = 0
	s.pressure.lastSampleRecoveryHealthy = false
	if s.pressure.mode != TQPressureAggregateOnly {
		s.pressure.mode = TQPressureAggregateOnly
		s.pressure.transitions++
		if s.pressure.degradedAt.IsZero() {
			s.pressure.degradedAt = e.lastClock
		}
	}
	s.aggregateOnly = true
	e.stopTQAdditionsLocked()
	e.closeAllTQCoverageLocked()
}

func (e *Engine) stopTQAdditionsLocked() {
	s := &e.state.tq
	s.additionPermit = false
	s.cadenceDesired = s.cadenceDesired[:0]
	if s.pending != nil && s.pending.action == TQSubscribe && !s.dispatched {
		s.pending = nil
		s.dispatched = false
		s.commandsFenced++
	}
}

func (e *Engine) closeAllTQCoverageLocked() {
	s := &e.state.tq
	for _, member := range s.members {
		member.requested = member.requested || member.present || member.unknown
		member.tradeCoverage.active, member.quoteCoverage.active = false, false
		member.present, member.unknown = false, member.requested
	}
}

func (e *Engine) applyTQCommandResultLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.tqCommandResult.TQCommandResultInput, &e.state.tq
	s.consumed++
	if e.state.binding == nil || v.command.bindingIdentity != e.state.binding.identity || v.command.connectionEpoch != e.state.liveEpoch || !e.state.liveEpochActive || s.pending == nil ||
		v.command != *s.pending || !s.dispatched {
		s.fenced++
		s.commandResultsFenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	// A current command result changes requested/provider-membership trust even
	// when the local write failed; the complete input publishes that transition
	// once after all cleanup and reconciliation are done.
	e.markTQTrustTransitionLocked()
	commandSymbols := v.command.symbols[:v.command.symbolCount]
	s.pending, s.dispatched = nil, false
	if v.outcome != ControlSucceeded {
		s.commandsFailed++
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			if v.command.action == TQUnsubscribe {
				m.requested, m.present, m.unknown = true, false, true
				m.resetRequired = true
			} else {
				m.requested, m.present, m.unknown = false, false, false
			}
			m.tradeCoverage.active, m.quoteCoverage.active = false, false
			if v.command.action == TQUnsubscribe {
				m.cleanupAttempted = true
			}
		}
		// A failed local write gives no request boundary and no evidence about
		// which provider-side topics changed. Contain only T/Q for this epoch;
		// aggregate processing remains live and no silence-based retry is issued.
		s.controlClosed, s.controlClosedEpoch = true, e.state.liveEpoch
		s.controlErrors++
		e.stopTQAdditionsLocked()
		e.closeAllTQCoverageLocked()
		s.rejected++
		return DispositionTQRejected, ReasonControlOutcome
	}
	s.commandsWritten++
	if v.command.action == TQUnsubscribe {
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			m.present, m.unknown, m.cleanupAttempted = false, false, false
			m.requested = false
			m.resetRequired = false
			s.releaseMember(m)
			m.unequalRepeat = false
			if !s.desiredContains(symbol) {
				s.deleteMember(symbol)
			}
		}
	} else {
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			pressureContains := s.pressure.mode != "" && s.pressure.mode != TQPressureNormal
			if pressureContains || s.aggregateOnly || m.bound || m.resetRequired || !s.desiredContains(symbol) {
				// A subscribe already written before containment may still have been
				// applied by the provider. Preserve that cleanup liability, but never
				// reopen causal coverage after additions have stopped.
				m.requested, m.present, m.unknown, m.cleanupAttempted = true, false, true, false
				m.resetRequired = true
				m.tradeCoverage.active, m.quoteCoverage.active = false, false
				s.releaseMember(m)
				continue
			}
			m.requested, m.present, m.unknown, m.cleanupAttempted = true, false, true, false
			if s.nextGeneration == 0 {
				e.enterTQGlobalBoundLocked()
				continue
			}
			m.generation = s.nextGeneration
			s.nextGeneration++
			m.tradeCoverage = tqCoverage{epoch: v.command.connectionEpoch, boundary: v.position}
			m.quoteCoverage = tqCoverage{epoch: v.command.connectionEpoch, boundary: v.position}
			s.releaseMember(m)
			m.unequalRepeat = false
		}
	}
	s.applied++
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyTQControlErrorLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.tqControlError.TQControlErrorInput, &e.state.tq
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.ConnectionEpoch != e.state.liveEpoch || !e.state.liveEpochActive {
		s.controlErrorsFenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if v.Position != (LivePosition{}) && s.controlClosedPosition != (LivePosition{}) && compareLive(v.Position, s.controlClosedPosition) <= 0 {
		s.controlErrorsFenced++
		return DispositionTQFenced, ReasonNonprecedent
	}
	if !s.controlClosed {
		e.markTQTrustTransitionLocked()
		s.controlClosed = true
		s.controlClosedEpoch = v.ConnectionEpoch
		s.controlClosedPosition = v.Position
	} else if v.Position != (LivePosition{}) {
		s.controlClosedPosition = v.Position
	}
	s.controlErrors++
	if s.pending != nil {
		action := s.pending.action
		for _, symbol := range s.pending.symbols[:s.pending.symbolCount] {
			member := s.member(symbol)
			if action == TQUnsubscribe {
				member.requested, member.present, member.unknown = true, false, true
				member.resetRequired = true
			} else {
				member.requested, member.present, member.unknown = false, false, false
			}
			member.tradeCoverage.active, member.quoteCoverage.active = false, false
		}
		s.pending, s.dispatched = nil, false
		s.commandsFailed++
	}
	for _, member := range s.members {
		if member.requested || member.present || member.unknown {
			member.present, member.unknown = false, true
		}
		member.tradeCoverage.active, member.quoteCoverage.active = false, false
		s.releaseMember(member)
	}
	e.stopTQAdditionsLocked()
	e.closeAllTQCoverageLocked()
	return DispositionTQRejected, ReasonControlOutcome
}

func (e *Engine) clearTQControlErrorLocked(epoch uint64) {
	s := &e.state.tq
	if !s.controlClosed || epoch <= s.controlClosedEpoch {
		return
	}
	// A strictly newer aggregate acknowledgement restores the public T/Q trust
	// domain. The connection-control fingerprint would publish the combined
	// cell anyway, but this transition must also be represented by the T/Q
	// projection accounting and immediate-transition counters.
	e.markTQTrustTransitionLocked()
	s.controlClosed = false
	s.controlClosedEpoch, s.controlClosedPosition = 0, LivePosition{}
}

func (e *Engine) applyTQDropLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.tqDrop.TQDropInput, &e.state.tq
	s.consumed++
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || v.Live.ConnectionEpoch != e.state.liveEpoch {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if v.DropReason == "optional_tq_shed" {
		s.pressureShed++
		if v.Family == "T" {
			s.tradePressureShed++
		} else {
			s.quotePressureShed++
		}
		if v.Symbol == "" {
			changed := false
			for _, member := range s.members {
				if v.Family == "T" {
					changed = changed || member.tradeCoverage.active
					member.tradeCoverage.active = false
				} else {
					changed = changed || member.quoteCoverage.active
					member.quoteCoverage.active = false
				}
				if member.present {
					member.resetRequired = true
				}
			}
			if changed {
				e.markTQTrustTransitionLocked()
			}
			return DispositionTQRejected, ReasonPressure
		}
		m, ok := s.members[v.Symbol]
		if !ok || !m.present {
			s.fenced++
			s.pressureShed--
			if v.Family == "T" {
				s.tradePressureShed--
			} else {
				s.quotePressureShed--
			}
			return DispositionTQFenced, ReasonHistoricalContext
		}
		coverage := &m.tradeCoverage
		if v.Family == "Q" {
			coverage = &m.quoteCoverage
		}
		if !coverage.active || compareLive(v.Live, coverage.greatest) <= 0 {
			s.fenced++
			s.pressureShed--
			if v.Family == "T" {
				s.tradePressureShed--
			} else {
				s.quotePressureShed--
			}
			return DispositionTQFenced, ReasonHistoricalContext
		}
		if coverage.active || !m.resetRequired {
			e.markTQTrustTransitionLocked()
		}
		coverage.greatest = v.Live
		m.tradeCoverage.active, m.quoteCoverage.active, m.resetRequired = false, false, true
		return DispositionTQRejected, ReasonPressure
	}
	if v.Symbol == "" {
		var greatest LivePosition
		for _, member := range s.members {
			coverage := member.tradeCoverage
			if v.Family == "Q" {
				coverage = member.quoteCoverage
			}
			if coverage.active && compareLive(coverage.greatest, greatest) > 0 {
				greatest = coverage.greatest
			}
		}
		if greatest.ConnectionEpoch == 0 || compareLive(v.Live, greatest) <= 0 {
			s.fenced++
			return DispositionTQFenced, ReasonHistoricalContext
		}
		s.pressure.cause = TQPressureCauseTransportAccounting
		e.enterTQAggregateOnlyLocked()
		s.integrity++
		return DispositionTQRejected, ReasonStructural
	}
	m, ok := s.members[v.Symbol]
	if !ok || !m.present {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	coverage := &m.tradeCoverage
	if v.Family == "Q" {
		coverage = &m.quoteCoverage
	}
	if !coverage.active || compareLive(v.Live, coverage.greatest) <= 0 {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if coverage.active || !m.resetRequired {
		e.markTQTrustTransitionLocked()
	}
	coverage.greatest = v.Live
	m.tradeCoverage.active, m.quoteCoverage.active, m.resetRequired = false, false, true
	s.rejected++
	return DispositionTQRejected, ReasonStructural
}

func (e *Engine) applyTradeLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.trade.TradeInput, &e.state.tq
	s.consumed++
	cleanupTarget := node.admissionTime
	if e.state.committedT != nil {
		cleanupTarget = *e.state.committedT
	}
	s.prune(cleanupTarget, node.admissionTime)
	m, ok := s.members[v.Symbol]
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || !ok || (!m.requested && !m.present && !m.unknown) || !s.desiredContains(v.Symbol) ||
		v.Live.ConnectionEpoch != m.tradeCoverage.epoch || v.Live.FrameSequence <= m.tradeCoverage.boundary.FrameSequence ||
		m.tradeCoverage.active && compareLive(v.Live, m.tradeCoverage.greatest) <= 0 {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if !m.tradeCoverage.active {
		e.markTQTrustTransitionLocked()
		start := v.ReceiptTime
		if start.Before(e.state.binding.sessionStart) {
			start = e.state.binding.sessionStart
		}
		if start.After(e.state.binding.sessionEnd) {
			start = e.state.binding.sessionEnd
		}
		m.tradeCoverage.active, m.tradeCoverage.start, m.tradeCoverage.greatest = true, start, v.Live
		m.present, m.unknown = m.quoteCoverage.active, !m.quoteCoverage.active
	} else {
		m.tradeCoverage.greatest = v.Live
	}
	if v.EventTime.Before(e.state.binding.sessionStart) || !v.EventTime.Before(e.state.binding.sessionEnd) ||
		e.state.committedT != nil && v.EventTime.Before(e.state.committedT.Add(-30*time.Second)) {
		s.rejected++
		return DispositionTQRejected, ReasonHistoricalContext
	}
	key := makeTQTradeIdentity(v)
	fingerprint := makeTQTradeFingerprint(v)
	if prior, exists := m.fingerprints[key]; exists {
		if prior.value == fingerprint {
			s.duplicate++
			return DispositionTQDuplicate, ReasonNone
		}
		e.markTQTrustTransitionLocked()
		m.unequalRepeat = true
		s.rejected++
		return DispositionTQRejected, ReasonStructural
	}
	limits := e.tqLimits
	incomingBytes := int64(tqTradeByteCharge + tqFingerprintByteCharge)
	if len(m.trades) >= limits.tradesPerSymbol || len(m.fingerprints) >= limits.fingerprintsPerSymbol || m.retainedBytes+incomingBytes > limits.bytesPerSymbol {
		e.markTQTrustTransitionLocked()
		m.bound = true
		m.tradeCoverage.active, m.quoteCoverage.active = false, false
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	if s.tradeCount >= limits.tradesGlobal || s.fingerprintCount >= limits.fingerprintsGlobal || s.retainedBytes+incomingBytes > limits.bytesGlobal {
		e.enterTQGlobalBoundLocked()
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	conditionClassified, conditionEligible := classifyTradeConditionEvidence(v.ConditionsClassified, v.Conditions)
	lifecycleObserved := v.Lifecycle != "original"
	m.fingerprints[key] = tqFingerprint{value: fingerprint, firstReceipt: v.ReceiptTime}
	m.trades = append(m.trades, tqTrade{at: v.EventTime, qualifying: v.Lifecycle == "original" && v.IdentityClassified && conditionClassified && conditionEligible, basis: v.TimestampBasis, lifecycle: lifecycleObserved})
	s.tradeCount++
	s.fingerprintCount++
	m.retainedBytes += incomingBytes
	s.retainedBytes += incomingBytes
	s.applied++
	s.tradeApplied++
	return DispositionTQApplied, ReasonNone
}

func makeTQTradeIdentity(v TradeInput) tqTradeIdentity {
	result := tqTradeIdentity{exchange: v.Exchange, tradeID: v.TradeID, trfPresent: v.TRFPresent}
	if v.TRFPresent {
		result.trfID = v.TRFID
	}
	return result
}

func makeTQTradeFingerprint(v TradeInput) tqTradeFingerprint {
	result := tqTradeFingerprint{
		eventUnixNano: v.EventTime.UnixNano(), priceBits: math.Float64bits(v.Price), economicSizeBits: math.Float64bits(v.EconomicSize),
		conditionCount: uint8(len(v.Conditions)), conditionsClassified: v.ConditionsClassified, identified: v.IdentityClassified,
	}
	if v.TimestampBasis == "sip_fallback" {
		result.basis = 1
	}
	if v.Lifecycle == "unsupported_lifecycle" {
		result.lifecycle = 1
	}
	copy(result.conditions[:], v.Conditions)
	return result
}

func (e *Engine) applyQuoteLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.quote.QuoteInput, &e.state.tq
	s.consumed++
	m, ok := s.members[v.Symbol]
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || !ok || (!m.requested && !m.present && !m.unknown) || !s.desiredContains(v.Symbol) ||
		v.Live.ConnectionEpoch != m.quoteCoverage.epoch || v.Live.FrameSequence <= m.quoteCoverage.boundary.FrameSequence ||
		m.quoteCoverage.active && compareLive(v.Live, m.quoteCoverage.greatest) <= 0 {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if !m.quoteCoverage.active {
		e.markTQTrustTransitionLocked()
		m.quoteCoverage.active, m.quoteCoverage.greatest = true, v.Live
		m.present, m.unknown = m.tradeCoverage.active, !m.tradeCoverage.active
	} else {
		m.quoteCoverage.greatest = v.Live
	}
	if v.SIPTime.Before(e.state.binding.sessionStart) || !v.SIPTime.Before(e.state.binding.sessionEnd) ||
		e.state.committedT != nil && v.SIPTime.Before(e.state.committedT.Add(-30*time.Second)) {
		s.rejected++
		return DispositionTQRejected, ReasonHistoricalContext
	}
	priorStatus := spreadView(m, e.state.committedT).Status
	quality := quoteQuality(v.ConditionsClassified, v.IndicatorsClassified, v.Conditions, v.Indicators)
	quote := &tqQuote{at: v.SIPTime, receipt: v.ReceiptTime, position: v.Live, bid: v.BidPrice, ask: v.AskPrice, bidPresent: v.BidPresent, askPresent: v.AskPresent, quality: quality}
	before := retainedQuoteCount(m)
	previousLatest, previousValid := m.latestQuote, m.latestValidQuote
	m.latestQuote = quote
	if quote.bidPresent && quote.askPresent && quote.ask >= quote.bid {
		m.latestValidQuote = quote
	}
	after := retainedQuoteCount(m)
	quoteDelta := after - before
	limits := e.tqLimits
	if quoteDelta > 0 && (m.retainedBytes+int64(quoteDelta)*tqQuoteByteCharge > limits.bytesPerSymbol || s.retainedBytes+int64(quoteDelta)*tqQuoteByteCharge > limits.bytesGlobal) {
		m.latestQuote, m.latestValidQuote = previousLatest, previousValid
		if m.retainedBytes+int64(quoteDelta)*tqQuoteByteCharge > limits.bytesPerSymbol {
			m.bound = true
			m.tradeCoverage.active, m.quoteCoverage.active = false, false
			e.markTQTrustTransitionLocked()
		} else {
			e.enterTQGlobalBoundLocked()
		}
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	s.quoteCount += quoteDelta
	m.retainedBytes += int64(quoteDelta) * tqQuoteByteCharge
	s.retainedBytes += int64(quoteDelta) * tqQuoteByteCharge
	newStatus := spreadView(m, e.state.committedT).Status
	if (priorStatus == TQCurrent || priorStatus == TQStale) && newStatus != TQCurrent && newStatus != TQStale {
		e.markTQTrustTransitionLocked()
	}
	s.applied++
	s.quoteApplied++
	return DispositionTQApplied, ReasonNone
}

func quoteQuality(conditionsClassified, indicatorsClassified bool, conditions, indicators []int64) string {
	if !conditionsClassified || !indicatorsClassified {
		return "unclassified"
	}
	for _, values := range [][]int64{conditions, indicators} {
		for _, value := range values {
			if value <= 0 {
				return "unclassified"
			}
		}
	}
	if len(conditions) == 0 && len(indicators) == 0 {
		return "reviewed_ordinary"
	}
	return "known_special"
}

func (e *Engine) ObserveTQ() TQView {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := e.tqViewLocked()
	if publication := e.publication.Load(); publication != nil {
		result.PublicationID = publication.publicationID
	}
	return result
}

func (e *Engine) tqViewLocked() TQView {
	s := &e.state.tq
	pressureMode := s.pressure.mode
	if pressureMode == "" {
		pressureMode = TQPressureNormal
	}
	if s.aggregateOnly {
		pressureMode = TQPressureAggregateOnly
	}
	shedding := s.aggregateOnly || pressureMode != TQPressureNormal
	result := TQView{Desired: append([]string(nil), s.desired...), Bounds: s.globalBound, AggregateOnly: s.aggregateOnly,
		Pressure: pressureMode, PressureCause: s.pressure.cause, ShedTradesQuotes: shedding, PressureMisses: s.pressure.consecutiveMisses,
		PressureTransitions: s.pressure.transitions, PressureFenced: s.pressure.fenced,
		PressureSample: TQPressureSampleView{Observed: s.pressure.sampleObserved, WaitingFrames: s.pressure.lastSample.WaitingFrames,
			FrameCapacity: s.pressure.lastSample.FrameCapacity, WaitingBytes: s.pressure.lastSample.WaitingBytes, ByteCapacity: s.pressure.lastSample.ByteCapacity,
			OldestWaitingFrameAge: s.pressure.lastSample.OldestWaitingFrameAge, AggregateWatermarkLag: s.pressure.lastSample.AggregateWatermarkLag,
			RecoveryHealthy: s.pressure.lastSampleRecoveryHealthy},
		RecoveryHealthySamples: s.pressure.streaks.healthy, RecoveryRequiredSamples: e.tqPressurePolicy.recoverySamples,
		ControlClosed: s.controlClosed, ControlClosedPosition: s.controlClosedPosition,
		ControlErrors: s.controlErrors, ControlErrorsFenced: s.controlErrorsFenced}
	if s.pending != nil {
		result.CommandPending, result.PendingAction, result.PendingSymbol = true, s.pending.action, s.pending.Symbol()
	}
	desired := make(map[string]bool, len(s.desired))
	for _, symbol := range s.desired {
		desired[symbol] = true
	}
	for _, symbol := range s.desired {
		m := s.members[symbol]
		row := TQSymbolView{Symbol: symbol, Desired: true, Tape: TapeRateView{Status: TQUnselected}, Spread: SpreadView{Status: TQUnselected}}
		if m != nil {
			row.ProviderPresent, row.ProviderMembershipUnknown = m.present, m.unknown
			row.TradeCoverage, row.QuoteCoverage = m.tradeCoverage.active, m.quoteCoverage.active
			row.Tape = tapeView(m, e.state.committedT)
			row.Spread = spreadView(m, e.state.committedT)
		}
		if s.controlClosed {
			row.Tape = TapeRateView{Status: TQUnavailable, Reason: "control_error"}
			row.Spread = SpreadView{Status: TQUnavailable, Reason: "control_error"}
		}
		if shedding {
			row.Tape = TapeRateView{Status: TQPressureShed, Reason: "pressure", LifecycleRecordsObserved: row.Tape.LifecycleRecordsObserved}
			row.Spread = SpreadView{Status: TQPressureShed, Reason: "pressure"}
		}
		result.Rows = append(result.Rows, row)
	}
	knownPresent, unknown := 0, 0
	for _, m := range s.members {
		if m.requested && m.present {
			knownPresent++
		}
		if m.requested && (!m.present || m.unknown) {
			unknown++
		}
	}
	result.Accounting = TQAccountingView{Consumed: s.consumed, Applied: s.applied, Duplicate: s.duplicate, Rejected: s.rejected, Fenced: s.fenced, PressureShed: s.pressureShed, Integrity: s.integrity,
		AppliedTrades: s.tradeApplied, AppliedQuotes: s.quoteApplied, PressureShedTrades: s.tradePressureShed, PressureShedQuotes: s.quotePressureShed,
		KnownPresent: knownPresent, KnownAbsent: len(s.members) - knownPresent - unknown, Unknown: unknown, RetainedTrades: s.tradeCount, RetainedQuotes: s.quoteCount, RetainedFingerprints: s.fingerprintCount,
		RetainedBytes: s.retainedBytes}
	pending := uint64(0)
	if s.pending != nil {
		pending = 1
	}
	result.Commands = TQCommandAccountingView{Issued: s.commandsIssued, Pending: pending, Written: s.commandsWritten, Failed: s.commandsFailed,
		Fenced: s.commandsFenced, ResultFenced: s.commandResultsFenced}
	return result
}

func cloneTQView(value TQView) TQView {
	result := value
	result.Desired = append([]string(nil), value.Desired...)
	result.Rows = append([]TQSymbolView(nil), value.Rows...)
	return result
}

func validTQPublication(value TQView, publicationID uint64, evaluation aggregateEvaluationResult) bool {
	pressureSample := value.PressureSample
	if value.PublicationID == 0 || value.PublicationID != publicationID || len(value.Desired) > maximumTQSymbols || len(value.Rows) != len(value.Desired) || value.Accounting.KnownPresent < 0 || value.Accounting.KnownAbsent < 0 || value.Accounting.Unknown < 0 ||
		value.Accounting.RetainedTrades < 0 || value.Accounting.RetainedQuotes < 0 || value.Accounting.RetainedFingerprints < 0 || value.Accounting.RetainedBytes < 0 ||
		value.Accounting.Consumed != value.Accounting.Applied+value.Accounting.Duplicate+value.Accounting.Rejected+value.Accounting.Fenced+value.Accounting.PressureShed+value.Accounting.Integrity ||
		value.Commands.Pending > 1 || value.Commands.Issued != value.Commands.Pending+value.Commands.Written+value.Commands.Failed+value.Commands.Fenced ||
		value.CommandPending != (value.Commands.Pending == 1) || value.Bounds && !value.AggregateOnly ||
		value.RecoveryRequiredSamples == 0 || value.RecoveryHealthySamples > value.RecoveryRequiredSamples ||
		value.Pressure != TQPressureNormal && value.RecoveryHealthySamples >= value.RecoveryRequiredSamples ||
		pressureSample.OldestWaitingFrameAge < 0 || pressureSample.AggregateWatermarkLag < 0 ||
		pressureSample.Observed && (pressureSample.FrameCapacity == 0 || pressureSample.WaitingFrames > pressureSample.FrameCapacity || pressureSample.ByteCapacity == 0 || pressureSample.WaitingBytes > pressureSample.ByteCapacity) ||
		!pressureSample.Observed && (pressureSample.WaitingFrames != 0 || pressureSample.FrameCapacity != 0 || pressureSample.WaitingBytes != 0 || pressureSample.ByteCapacity != 0 || pressureSample.OldestWaitingFrameAge != 0 || pressureSample.AggregateWatermarkLag != 0 || pressureSample.RecoveryHealthy) {
		return false
	}
	if value.Pressure != TQPressureNormal && value.Pressure != TQPressureDegraded && value.Pressure != TQPressureAggregateOnly {
		return false
	}
	if value.Pressure == TQPressureNormal && value.PressureCause != TQPressureCauseNone || value.Pressure != TQPressureNormal && !validTQPressureCause(value.PressureCause) ||
		value.Accounting.AppliedTrades+value.Accounting.AppliedQuotes > value.Accounting.Applied ||
		value.Accounting.PressureShedTrades+value.Accounting.PressureShedQuotes != value.Accounting.PressureShed {
		return false
	}
	if value.AggregateOnly != (value.Pressure == TQPressureAggregateOnly) || value.ShedTradesQuotes != (value.Pressure != TQPressureNormal) {
		return false
	}
	if value.Accounting.KnownPresent+value.Accounting.KnownAbsent+value.Accounting.Unknown > 2*maximumTQSymbols ||
		value.Accounting.RetainedTrades > maximumTradesGlobal || value.Accounting.RetainedQuotes > 2*maximumTQSymbols || value.Accounting.RetainedFingerprints > maximumFingerprintsGlobal ||
		value.Accounting.RetainedBytes > maximumTQBytesGlobal {
		return false
	}
	wantDesired := evaluation.mode == rankingQualifiedCurrent
	if !wantDesired && len(value.Desired) != 0 || wantDesired && len(value.Desired) != len(evaluation.rows) {
		return false
	}
	seen := make(map[string]struct{}, len(value.Desired))
	for index, symbol := range value.Desired {
		if symbol == "" || value.Rows[index].Symbol != symbol || !value.Rows[index].Desired || wantDesired && evaluation.rows[index].symbol != symbol {
			return false
		}
		if _, exists := seen[symbol]; exists {
			return false
		}
		seen[symbol] = struct{}{}
		row := value.Rows[index]
		if !validTQFieldStatus(row.Tape.Status) || !validTQFieldStatus(row.Spread.Status) || !finiteTQ(row.Tape.FiveSecond) ||
			!finiteTQ(row.Spread.Cents) || !finiteTQ(row.Spread.BasisPoints) || row.Spread.QuoteAge < 0 ||
			!row.Spread.ObservedAt.IsZero() && row.Spread.ObservedAt != row.Spread.ObservedAt.UTC() {
			return false
		}
	}
	return true
}

func validTQFieldStatus(value TQFieldStatus) bool {
	switch value {
	case TQUnselected, TQWarming, TQCurrent, TQStale, TQUnavailable, TQInvalid, TQPressureShed:
		return true
	default:
		return false
	}
}

func validTQPressureCause(value TQPressureCause) bool {
	switch value {
	case TQPressureCauseNone, TQPressureCauseWaitingFrames, TQPressureCauseWaitingBytes, TQPressureCauseOldestWaitingFrame,
		TQPressureCauseWatermarkLag, TQPressureCauseCapacityDrop, TQPressureCauseRetentionBound, TQPressureCauseTransportAccounting:
		return true
	default:
		return false
	}
}

func finiteTQ(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func (s *tqState) desiredContains(symbol string) bool {
	for _, desired := range s.desired {
		if desired == symbol {
			return true
		}
	}
	return false
}

func tapeView(m *tqSymbolState, target *time.Time) TapeRateView {
	v := TapeRateView{Status: TQUnavailable, Reason: "channel_unconfirmed"}
	if !m.tradeCoverage.active || target == nil {
		if !m.requested {
			v.Reason = "coverage"
		}
		return v
	}
	if m.unequalRepeat {
		v.Status, v.Reason = TQInvalid, "unequal_repeat"
		return v
	}
	age := target.Sub(m.tradeCoverage.start)
	if age < time.Second {
		v.Status, v.Reason = TQWarming, "coverage_warming"
		return v
	}
	five, participant, sip := 0, false, false
	for _, trade := range m.trades {
		if trade.at.Before(target.Add(-5*time.Second)) || !trade.at.Before(*target) {
			continue
		}
		v.LifecycleRecordsObserved = v.LifecycleRecordsObserved || trade.lifecycle
		if !trade.qualifying {
			continue
		}
		five++
		participant = participant || trade.basis == "participant"
		sip = sip || trade.basis == "sip_fallback"
	}
	if age < 5*time.Second {
		v.Status, v.Reason = TQWarming, "five_second_warming"
	} else {
		v.Status, v.Reason = TQCurrent, "qualifying_original_prints"
		v.FiveSecond = float64(five) / 5
	}
	switch {
	case participant && sip:
		v.TimestampBasis = "mixed"
	case participant:
		v.TimestampBasis = "participant"
	case sip:
		v.TimestampBasis = "sip_fallback"
	default:
		v.TimestampBasis = "none"
	}
	return v
}

func spreadView(m *tqSymbolState, target *time.Time) SpreadView {
	v := SpreadView{Status: TQUnavailable, Reason: "channel_unconfirmed"}
	if !m.quoteCoverage.active || target == nil {
		if !m.requested {
			v.Reason = "coverage"
		}
		return v
	}
	latest := m.latestQuote
	if latest == nil {
		v.Status, v.Reason = TQWarming, "coverage_warming"
		return v
	}
	v.ObservedAt = latest.at
	v.Quality = latest.quality
	if !latest.bidPresent || !latest.askPresent {
		v.Status, v.Reason = TQUnavailable, "one_sided_quote"
		return v
	}
	if latest.ask < latest.bid {
		v.Status, v.Reason = TQInvalid, "crossed_quote"
		return v
	}
	valid := m.latestValidQuote
	if valid == nil || valid.position != latest.position {
		return v
	}
	v.QuoteAge = target.Sub(valid.at)
	if v.QuoteAge < 0 {
		v.QuoteAge = 0
	}
	v.Cents = 100 * (valid.ask - valid.bid)
	v.BasisPoints = 10000 * (valid.ask - valid.bid) / ((valid.ask + valid.bid) / 2)
	if v.QuoteAge > spreadStaleAge {
		v.Status, v.Reason = TQStale, "stale_quote"
		return v
	}
	v.Status, v.Reason = TQCurrent, ""
	return v
}

func retainedQuoteCount(m *tqSymbolState) int {
	if m == nil || m.latestQuote == nil {
		return 0
	}
	if m.latestValidQuote == nil || m.latestValidQuote.position == m.latestQuote.position {
		return 1
	}
	return 2
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
