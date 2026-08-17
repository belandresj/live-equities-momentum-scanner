package engine

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"time"
)

const TQSchemaV1 = "engine-tq-v1"

type TQControlFailureClass string

const (
	TQControlStatusUnsolicited TQControlFailureClass = "status_unsolicited"
	TQControlStatusFailed      TQControlFailureClass = "status_failed"
	TQControlStatusAmbiguous   TQControlFailureClass = "status_ambiguous"
	TQControlStatusExtra       TQControlFailureClass = "status_extra"
	TQControlStatusDeadline    TQControlFailureClass = "status_deadline"
	TQControlWriteFailed       TQControlFailureClass = "write_failed"
	TQControlAccounting        TQControlFailureClass = "command_accounting"
)

// TQControlQuarantineInput is a bounded, provider-independent T/Q control
// failure fact. It cannot carry provider prose or request a lifecycle change.
type TQControlQuarantineInput struct {
	SchemaVersion, BindingIdentity string
	ConnectionEpoch                uint64
	Position                       LivePosition
	ReceiptTime                    time.Time
	Failure                        TQControlFailureClass
	CommandToken                   uint64
	ExpectedStatuses               int
	ObservedStatuses               int
	Deadline                       bool
}

type frozenTQControlQuarantineInput struct{ TQControlQuarantineInput }

const (
	maximumTQSymbols          = 20
	maximumTradesPerSymbol    = 50_000
	maximumTradeFingerprints  = 100_000
	maximumTradesGlobal       = 500_000
	maximumFingerprintsGlobal = 1_000_000
	spreadStaleAge            = 2 * time.Second
)

type tqRetentionLimits struct {
	tradesPerSymbol, tradesGlobal, fingerprintsPerSymbol, fingerprintsGlobal int
}

func defaultTQRetentionLimits() tqRetentionLimits {
	return tqRetentionLimits{maximumTradesPerSymbol, maximumTradesGlobal, maximumTradeFingerprints, maximumFingerprintsGlobal}
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
	hasPosition := position.ConnectionEpoch == command.connectionEpoch && position.FrameSequence > 0
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
	active        bool
	epoch         uint64
	start         time.Time
	ack, greatest LivePosition
}

type tqTrade struct {
	key, fingerprint string
	at               time.Time
	qualifying       bool
	basis            string
	lifecycle        bool
}

type tqFingerprint struct {
	value string
	at    time.Time
}

type tqQuote struct {
	at, receipt            time.Time
	position               LivePosition
	bid, ask               float64
	bidPresent, askPresent bool
	quality                string
}

type tqSymbolState struct {
	present, unknown, cleanupAttempted bool
	bound                              bool
	resetRequired                      bool
	tradeCoverage, quoteCoverage       tqCoverage
	trades                             []tqTrade
	latestQuote, latestValidQuote      *tqQuote
	fingerprints                       map[string]tqFingerprint
	unequalRepeat, lifecycleObserved   bool
}

type tqState struct {
	epoch                                                                                      uint64
	revision                                                                                   uint64
	nextToken                                                                                  uint64
	desired                                                                                    []string
	members                                                                                    map[string]*tqSymbolState
	pending                                                                                    *TQCommand
	dispatched                                                                                 bool
	consumed, applied                                                                          uint64
	duplicate, rejected                                                                        uint64
	fenced, pressureShed                                                                       uint64
	tradeApplied, quoteApplied, tradePressureShed, quotePressureShed                           uint64
	integrity                                                                                  uint64
	tradeCount, quoteCount                                                                     int
	fingerprintCount                                                                           int
	globalBound                                                                                bool
	aggregateOnly                                                                              bool
	pressure                                                                                   tqPressureState
	commandsIssued, commandsAcknowledged, commandsFailed, commandsFenced, commandResultsFenced uint64
	epochCommandsAcknowledged                                                                  uint64
	quarantined, quarantineRaisedAggregateOnly                                                 bool
	quarantineReason                                                                           TQControlFailureClass
	quarantineEpoch                                                                            uint64
	quarantinePosition                                                                         LivePosition
	quarantineExpected, quarantineObserved                                                     int
	quarantineDeadline                                                                         bool
	quarantines, quarantineFenced                                                              uint64
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
	Status                            TQFieldStatus
	Reason                            string
	OneSecondStatus, FiveSecondStatus TQFieldStatus
	OneSecondReason, FiveSecondReason string
	OneSecond, FiveSecond             float64
	TimestampBasis                    string
	LifecycleRecordsObserved          bool
}

type SpreadView struct {
	Status             TQFieldStatus
	Reason             string
	Cents, BasisPoints float64
	QuoteAge           time.Duration
	Quality            string
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
}

type TQCommandAccountingView struct {
	Issued, Pending, Acknowledged, Failed, Fenced, ResultFenced uint64
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
	Quarantined                         bool
	QuarantineReason                    TQControlFailureClass
	QuarantinePosition                  LivePosition
	QuarantineExpected                  int
	QuarantineObserved                  int
	QuarantineDeadline                  bool
	Quarantines, QuarantineFenced       uint64
}

func (e *Engine) AdmitTQControlQuarantine(ctx context.Context, input TQControlQuarantineInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validTQControlQuarantineInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQControlQuarantine, tqControlQuarantine: frozenTQControlQuarantineInput{input}}, false)
}

func validTQControlQuarantineInput(v TQControlQuarantineInput) bool {
	hasPosition := v.Position != (LivePosition{})
	validPosition := v.Position.ConnectionEpoch == v.ConnectionEpoch && v.Position.FrameSequence > 0
	return v.SchemaVersion == TQSchemaV1 && validIdentityShape(v.BindingIdentity) && v.ConnectionEpoch > 0 &&
		!v.ReceiptTime.IsZero() && v.ReceiptTime == v.ReceiptTime.UTC() && validTQControlFailure(v.Failure) &&
		(!hasPosition || validPosition) && v.ExpectedStatuses >= 0 && v.ExpectedStatuses <= 2*maximumTQSymbols &&
		v.ObservedStatuses >= 0 && v.ObservedStatuses <= 2*maximumTQSymbols+1 &&
		(v.CommandToken != 0 || v.ExpectedStatuses == 0)
}

func validTQControlFailure(v TQControlFailureClass) bool {
	switch v {
	case TQControlStatusUnsolicited, TQControlStatusFailed, TQControlStatusAmbiguous,
		TQControlStatusExtra, TQControlStatusDeadline, TQControlWriteFailed, TQControlAccounting:
		return true
	default:
		return false
	}
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

func (e *Engine) reconcileTQLocked(now time.Time) {
	s := &e.state.tq
	if s.members == nil {
		s.members = make(map[string]*tqSymbolState)
		s.nextToken = 1
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
			epoch: e.state.liveEpoch, revision: previous.revision, nextToken: maxUint64(e.state.aggregateWriteToken+1, maxUint64(1, previous.nextToken)), members: make(map[string]*tqSymbolState),
			consumed: previous.consumed, applied: previous.applied, duplicate: previous.duplicate, rejected: previous.rejected, fenced: previous.fenced,
			pressureShed: previous.pressureShed, integrity: previous.integrity, tradeApplied: previous.tradeApplied, quoteApplied: previous.quoteApplied,
			tradePressureShed: previous.tradePressureShed, quotePressureShed: previous.quotePressureShed,
			globalBound: previous.globalBound, aggregateOnly: previous.aggregateOnly,
			commandsIssued: previous.commandsIssued, commandsAcknowledged: previous.commandsAcknowledged, commandsFailed: previous.commandsFailed,
			commandsFenced: commandFenced, commandResultsFenced: previous.commandResultsFenced,
			pressure:    pressure,
			quarantined: previous.quarantined, quarantineRaisedAggregateOnly: previous.quarantineRaisedAggregateOnly,
			quarantineReason: previous.quarantineReason, quarantineEpoch: previous.quarantineEpoch, quarantinePosition: previous.quarantinePosition,
			quarantineExpected: previous.quarantineExpected, quarantineObserved: previous.quarantineObserved,
			quarantineDeadline: previous.quarantineDeadline, quarantines: previous.quarantines, quarantineFenced: previous.quarantineFenced,
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
	s.desired = desired
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
			if !member.present && !member.unknown {
				member.bound = false
				if len(member.trades) == 0 && member.latestQuote == nil && member.latestValidQuote == nil && len(member.fingerprints) == 0 {
					delete(s.members, symbol)
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
	if s.pending != nil || s.quarantined || !e.state.liveEpochActive || e.state.lifecycle != lifecycleLive {
		return
	}
	if s.pressure.mode == TQPressureDegraded {
		return
	}
	if s.aggregateOnly {
		for index := len(desired) - 1; index >= 0; index-- {
			member := s.members[desired[index]]
			if member != nil && (member.present || (member.unknown && !member.cleanupAttempted)) {
				e.issueTQCommandLocked(TQUnsubscribe, []string{desired[index]})
				return
			}
		}
	}
	for symbol, member := range s.members {
		if (member.present || (member.unknown && !member.cleanupAttempted)) && (s.aggregateOnly || member.bound || member.resetRequired || !desiredSet[symbol]) {
			e.issueTQCommandLocked(TQUnsubscribe, []string{symbol})
			return
		}
	}
	if s.aggregateOnly {
		return
	}
	wireMembers := 0
	for _, member := range s.members {
		if member.present || member.unknown {
			wireMembers++
		}
	}
	missing := make([]string, 0, len(desired))
	for _, symbol := range desired {
		member := s.member(symbol)
		if member.bound {
			continue
		}
		if member.unknown {
			if !member.cleanupAttempted {
				e.issueTQCommandLocked(TQUnsubscribe, []string{symbol})
			}
			return
		}
		if !member.present {
			if wireMembers+len(missing) < maximumTQSymbols {
				missing = append(missing, symbol)
			}
		}
	}
	if len(missing) > 0 {
		// A fresh epoch has no earlier T/Q command whose delayed, untagged
		// provider status could contaminate this acknowledgement. Subscribe the
		// complete displayed set in one command so every selected row begins
		// causal coverage at the same terminal boundary. Later churn and pressure
		// restoration retain the accepted single-symbol command path.
		if wireMembers == 0 && s.epochCommandsAcknowledged == 0 {
			e.issueTQCommandLocked(TQSubscribe, missing)
		} else {
			e.issueTQCommandLocked(TQSubscribe, missing[:1])
		}
	}
	_ = now
}

func (s *tqState) member(symbol string) *tqSymbolState {
	m := s.members[symbol]
	if m == nil {
		m = &tqSymbolState{fingerprints: make(map[string]tqFingerprint)}
		s.members[symbol] = m
	}
	return m
}

func (s *tqState) releaseMember(m *tqSymbolState) {
	s.tradeCount -= len(m.trades)
	s.quoteCount -= retainedQuoteCount(m)
	s.fingerprintCount -= len(m.fingerprints)
	m.trades, m.latestQuote, m.latestValidQuote = nil, nil, nil
	m.fingerprints = make(map[string]tqFingerprint)
}

func (s *tqState) prune(target, engineTime time.Time) {
	tradeCutoff, fingerprintCutoff := target.Add(-6*time.Second), engineTime.Add(-16*time.Minute)
	for _, m := range s.members {
		trades := m.trades[:0]
		for _, item := range m.trades {
			if item.at.Before(tradeCutoff) {
				s.tradeCount--
				continue
			}
			trades = append(trades, item)
		}
		m.trades = trades
		for key, item := range m.fingerprints {
			if item.at.Before(fingerprintCutoff) {
				delete(m.fingerprints, key)
				s.fingerprintCount--
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
	e.state.tq.globalBound = true
	e.state.tq.pressure.cause = TQPressureCauseRetentionBound
	e.enterTQAggregateOnlyLocked()
}

func (e *Engine) enterTQAggregateOnlyLocked() {
	s := &e.state.tq
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
	if s.pending != nil && s.pending.action == TQSubscribe && !s.dispatched {
		s.pending = nil
		s.dispatched = false
		s.commandsFenced++
	}
}

func (e *Engine) closeAllTQCoverageLocked() {
	s := &e.state.tq
	for _, member := range s.members {
		member.tradeCoverage.active, member.quoteCoverage.active = false, false
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
	commandSymbols := v.command.symbols[:v.command.symbolCount]
	s.pending, s.dispatched = nil, false
	if v.outcome != ControlSucceeded {
		s.commandsFailed++
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			m.present, m.unknown = false, true
			m.tradeCoverage.active, m.quoteCoverage.active = false, false
			if v.command.action == TQUnsubscribe {
				m.cleanupAttempted = true
			}
		}
		s.rejected++
		return DispositionTQRejected, ReasonControlOutcome
	}
	s.commandsAcknowledged++
	s.epochCommandsAcknowledged++
	if v.command.action == TQUnsubscribe {
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			m.present, m.unknown, m.cleanupAttempted = false, false, false
			m.resetRequired = false
			s.releaseMember(m)
			m.unequalRepeat, m.lifecycleObserved = false, false
			if !s.desiredContains(symbol) {
				delete(s.members, symbol)
			}
		}
	} else {
		start := v.receiptTime
		if start.Before(e.state.binding.sessionStart) {
			start = e.state.binding.sessionStart
		}
		if start.After(e.state.binding.sessionEnd) {
			start = e.state.binding.sessionEnd
		}
		for _, symbol := range commandSymbols {
			m := s.member(symbol)
			pressureContains := s.pressure.mode != "" && s.pressure.mode != TQPressureNormal
			if pressureContains || s.aggregateOnly || m.bound || m.resetRequired || !s.desiredContains(symbol) {
				// A subscribe already written before containment may still have been
				// applied by the provider. Preserve that cleanup liability, but never
				// reopen causal coverage after additions have stopped.
				m.present, m.unknown, m.cleanupAttempted = true, false, false
				m.resetRequired = true
				m.tradeCoverage.active, m.quoteCoverage.active = false, false
				s.releaseMember(m)
				continue
			}
			m.present, m.unknown, m.cleanupAttempted = true, false, false
			m.tradeCoverage = tqCoverage{active: true, epoch: v.command.connectionEpoch, start: start, ack: v.position, greatest: v.position}
			m.quoteCoverage = m.tradeCoverage
			s.releaseMember(m)
			m.unequalRepeat, m.lifecycleObserved = false, false
		}
	}
	s.applied++
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyTQControlQuarantineLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.tqControlQuarantine.TQControlQuarantineInput, &e.state.tq
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.ConnectionEpoch != e.state.liveEpoch || !e.state.liveEpochActive {
		s.quarantineFenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if v.Position != (LivePosition{}) && s.quarantinePosition != (LivePosition{}) && compareLive(v.Position, s.quarantinePosition) <= 0 {
		s.quarantineFenced++
		return DispositionTQFenced, ReasonNonprecedent
	}
	if !s.quarantined {
		s.quarantined = true
		s.quarantineReason = v.Failure
		s.quarantineEpoch = v.ConnectionEpoch
		s.quarantinePosition = v.Position
		s.quarantineExpected = v.ExpectedStatuses
		s.quarantineObserved = v.ObservedStatuses
		s.quarantineDeadline = v.Deadline
		s.quarantineRaisedAggregateOnly = !s.aggregateOnly
	} else if v.Position != (LivePosition{}) {
		s.quarantinePosition = v.Position
	}
	s.quarantines++
	if s.pending != nil {
		for _, symbol := range s.pending.symbols[:s.pending.symbolCount] {
			member := s.member(symbol)
			member.present, member.unknown = false, true
			member.tradeCoverage.active, member.quoteCoverage.active = false, false
		}
		s.pending, s.dispatched = nil, false
		s.commandsFailed++
	}
	for _, member := range s.members {
		if member.present || member.unknown {
			member.present, member.unknown = false, true
		}
		member.tradeCoverage.active, member.quoteCoverage.active = false, false
		s.releaseMember(member)
	}
	e.enterTQAggregateOnlyLocked()
	return DispositionTQRejected, ReasonControlOutcome
}

// clearTQControlQuarantineLocked is called only when the aggregate handshake
// acknowledges a strictly greater connection epoch. A fresh epoch has no
// provider-membership continuity and therefore must warm from new coverage.
func (e *Engine) clearTQControlQuarantineLocked(epoch uint64) {
	s := &e.state.tq
	if !s.quarantined || epoch <= s.quarantineEpoch {
		return
	}
	if s.quarantineRaisedAggregateOnly {
		s.aggregateOnly = false
		if s.pressure.mode == TQPressureAggregateOnly {
			s.pressure.mode = TQPressureNormal
		}
	}
	s.quarantined, s.quarantineRaisedAggregateOnly = false, false
	s.quarantineReason, s.quarantineEpoch, s.quarantinePosition = "", 0, LivePosition{}
	s.quarantineExpected, s.quarantineObserved, s.quarantineDeadline = 0, 0, false
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
			for _, member := range s.members {
				if v.Family == "T" {
					member.tradeCoverage.active = false
				} else {
					member.quoteCoverage.active = false
				}
				if member.present {
					member.resetRequired = true
				}
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
	coverage.greatest = v.Live
	m.tradeCoverage.active, m.quoteCoverage.active, m.resetRequired = false, false, true
	s.rejected++
	return DispositionTQRejected, ReasonStructural
}

func (e *Engine) applyTradeLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.trade.TradeInput, &e.state.tq
	s.consumed++
	m, ok := s.members[v.Symbol]
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || !ok || !m.tradeCoverage.active ||
		v.Live.ConnectionEpoch != m.tradeCoverage.epoch || compareLive(v.Live, m.tradeCoverage.greatest) <= 0 {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	m.tradeCoverage.greatest = v.Live
	key := v.TradingDate + "\x00" + v.Symbol + "\x00" + strconv.FormatInt(v.Exchange, 10) + "\x00" + v.TradeID
	if v.TRFPresent {
		key += "\x00" + strconv.FormatInt(v.TRFID, 10)
	}
	fingerprint := key + "\x00" + v.EventTime.Format(time.RFC3339Nano) + "\x00" + v.TimestampBasis + "\x00" +
		strconv.FormatFloat(v.Price, 'g', -1, 64) + "\x00" + strconv.FormatFloat(v.EconomicSize, 'g', -1, 64) + "\x00" +
		strconv.FormatBool(v.ConditionsClassified) + "\x00" + strconv.FormatBool(v.IdentityClassified) + "\x00" + v.Lifecycle
	for _, condition := range v.Conditions {
		fingerprint += "\x00" + strconv.FormatInt(condition, 10)
	}
	if prior, exists := m.fingerprints[key]; exists {
		if prior.value == fingerprint {
			s.duplicate++
			return DispositionTQDuplicate, ReasonNone
		}
		m.unequalRepeat = true
		s.rejected++
		return DispositionTQRejected, ReasonStructural
	}
	limits := e.tqLimits
	if len(m.trades) >= limits.tradesPerSymbol || len(m.fingerprints) >= limits.fingerprintsPerSymbol {
		m.bound = true
		m.tradeCoverage.active, m.quoteCoverage.active = false, false
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	if s.tradeCount >= limits.tradesGlobal || s.fingerprintCount >= limits.fingerprintsGlobal {
		e.enterTQGlobalBoundLocked()
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	conditionClassified, conditionEligible := classifyTradeConditionEvidence(v.ConditionsClassified, v.Conditions)
	lifecycleObserved := v.Lifecycle != "original"
	m.fingerprints[key] = tqFingerprint{value: fingerprint, at: node.admissionTime}
	m.trades = append(m.trades, tqTrade{key: key, fingerprint: fingerprint, at: v.EventTime, qualifying: v.Lifecycle == "original" && v.IdentityClassified && conditionClassified && conditionEligible, basis: v.TimestampBasis, lifecycle: lifecycleObserved})
	m.lifecycleObserved = m.lifecycleObserved || lifecycleObserved
	s.tradeCount++
	s.fingerprintCount++
	s.applied++
	s.tradeApplied++
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyQuoteLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.quote.QuoteInput, &e.state.tq
	s.consumed++
	m, ok := s.members[v.Symbol]
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || !ok || !m.quoteCoverage.active ||
		v.Live.ConnectionEpoch != m.quoteCoverage.epoch || compareLive(v.Live, m.quoteCoverage.greatest) <= 0 {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	m.quoteCoverage.greatest = v.Live
	if v.SIPTime.Before(e.state.binding.sessionStart) || v.SIPTime.After(e.state.binding.sessionEnd) {
		s.rejected++
		return DispositionTQRejected, ReasonHistoricalContext
	}
	quality := quoteQuality(v.ConditionsClassified, v.IndicatorsClassified, v.Conditions, v.Indicators)
	quote := &tqQuote{at: v.SIPTime, receipt: v.ReceiptTime, position: v.Live, bid: v.BidPrice, ask: v.AskPrice, bidPresent: v.BidPresent, askPresent: v.AskPresent, quality: quality}
	before := retainedQuoteCount(m)
	m.latestQuote = quote
	if quote.bidPresent && quote.askPresent && quote.ask >= quote.bid {
		m.latestValidQuote = quote
	}
	s.quoteCount += retainedQuoteCount(m) - before
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
		Quarantined: s.quarantined, QuarantineReason: s.quarantineReason, QuarantinePosition: s.quarantinePosition,
		QuarantineExpected: s.quarantineExpected, QuarantineObserved: s.quarantineObserved,
		QuarantineDeadline: s.quarantineDeadline, Quarantines: s.quarantines, QuarantineFenced: s.quarantineFenced}
	if s.pending != nil {
		result.CommandPending, result.PendingAction, result.PendingSymbol = true, s.pending.action, s.pending.Symbol()
	}
	desired := make(map[string]bool, len(s.desired))
	for _, symbol := range s.desired {
		desired[symbol] = true
	}
	for _, symbol := range s.desired {
		m := s.members[symbol]
		row := TQSymbolView{Symbol: symbol, Desired: true, Tape: TapeRateView{Status: TQUnselected, OneSecondStatus: TQUnselected, FiveSecondStatus: TQUnselected}, Spread: SpreadView{Status: TQUnselected}}
		if m != nil {
			row.ProviderPresent, row.ProviderMembershipUnknown = m.present, m.unknown
			row.TradeCoverage, row.QuoteCoverage = m.tradeCoverage.active, m.quoteCoverage.active
			row.Tape = tapeView(m, e.state.committedT)
			row.Spread = spreadView(m, e.state.committedT)
		}
		if shedding {
			row.Tape = TapeRateView{Status: TQPressureShed, Reason: "pressure", OneSecondStatus: TQPressureShed, FiveSecondStatus: TQPressureShed,
				OneSecondReason: "pressure", FiveSecondReason: "pressure", LifecycleRecordsObserved: row.Tape.LifecycleRecordsObserved}
			row.Spread = SpreadView{Status: TQPressureShed, Reason: "pressure"}
		}
		result.Rows = append(result.Rows, row)
	}
	knownPresent, unknown := 0, 0
	for _, m := range s.members {
		if m.present {
			knownPresent++
		}
		if m.unknown {
			unknown++
		}
	}
	result.Accounting = TQAccountingView{Consumed: s.consumed, Applied: s.applied, Duplicate: s.duplicate, Rejected: s.rejected, Fenced: s.fenced, PressureShed: s.pressureShed, Integrity: s.integrity,
		AppliedTrades: s.tradeApplied, AppliedQuotes: s.quoteApplied, PressureShedTrades: s.tradePressureShed, PressureShedQuotes: s.quotePressureShed,
		KnownPresent: knownPresent, KnownAbsent: len(s.members) - knownPresent - unknown, Unknown: unknown, RetainedTrades: s.tradeCount, RetainedQuotes: s.quoteCount, RetainedFingerprints: s.fingerprintCount}
	pending := uint64(0)
	if s.pending != nil {
		pending = 1
	}
	result.Commands = TQCommandAccountingView{Issued: s.commandsIssued, Pending: pending, Acknowledged: s.commandsAcknowledged, Failed: s.commandsFailed,
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
		value.Accounting.RetainedTrades < 0 || value.Accounting.RetainedQuotes < 0 || value.Accounting.RetainedFingerprints < 0 ||
		value.Accounting.Consumed != value.Accounting.Applied+value.Accounting.Duplicate+value.Accounting.Rejected+value.Accounting.Fenced+value.Accounting.PressureShed+value.Accounting.Integrity ||
		value.Commands.Pending > 1 || value.Commands.Issued != value.Commands.Pending+value.Commands.Acknowledged+value.Commands.Failed+value.Commands.Fenced ||
		value.CommandPending != (value.Commands.Pending == 1) || value.Bounds && !value.AggregateOnly || value.Quarantined && !value.AggregateOnly ||
		value.QuarantineExpected < 0 || value.QuarantineExpected > 2*maximumTQSymbols || value.QuarantineObserved < 0 || value.QuarantineObserved > 2*maximumTQSymbols+1 ||
		value.Quarantined != validTQControlFailure(value.QuarantineReason) || value.RecoveryRequiredSamples == 0 || value.RecoveryHealthySamples > value.RecoveryRequiredSamples ||
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
		value.Accounting.RetainedTrades > maximumTradesGlobal || value.Accounting.RetainedQuotes > 2*maximumTQSymbols || value.Accounting.RetainedFingerprints > maximumFingerprintsGlobal {
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
		if !validTQFieldStatus(row.Tape.Status) || !validTQFieldStatus(row.Tape.OneSecondStatus) || !validTQFieldStatus(row.Tape.FiveSecondStatus) ||
			!validTQFieldStatus(row.Spread.Status) || !finiteTQ(row.Tape.OneSecond) || !finiteTQ(row.Tape.FiveSecond) ||
			!finiteTQ(row.Spread.Cents) || !finiteTQ(row.Spread.BasisPoints) || row.Spread.QuoteAge < 0 ||
			row.Tape.Status != row.Tape.FiveSecondStatus || row.Tape.Reason != row.Tape.FiveSecondReason {
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
	v := TapeRateView{Status: TQUnavailable, Reason: "coverage", OneSecondStatus: TQUnavailable, FiveSecondStatus: TQUnavailable, OneSecondReason: "coverage", FiveSecondReason: "coverage", LifecycleRecordsObserved: m.lifecycleObserved}
	if !m.tradeCoverage.active || target == nil {
		return v
	}
	if m.unequalRepeat {
		v.Status, v.Reason = TQInvalid, "unequal_repeat"
		v.OneSecondStatus, v.FiveSecondStatus = TQInvalid, TQInvalid
		v.OneSecondReason, v.FiveSecondReason = "unequal_repeat", "unequal_repeat"
		return v
	}
	age := target.Sub(m.tradeCoverage.start)
	if age < time.Second {
		v.Status, v.Reason = TQWarming, "coverage_warming"
		v.OneSecondStatus, v.FiveSecondStatus = TQWarming, TQWarming
		v.OneSecondReason, v.FiveSecondReason = "coverage_warming", "coverage_warming"
		return v
	}
	one, five, participant, sip := 0, 0, false, false
	for _, trade := range m.trades {
		if !trade.qualifying || trade.at.Before(target.Add(-5*time.Second)) || !trade.at.Before(*target) {
			continue
		}
		five++
		if !trade.at.Before(target.Add(-time.Second)) {
			one++
		}
		participant = participant || trade.basis == "participant"
		sip = sip || trade.basis == "sip_fallback"
	}
	v.OneSecondStatus, v.OneSecondReason, v.OneSecond = TQCurrent, "qualifying_original_prints", float64(one)
	if age < 5*time.Second {
		v.Status, v.Reason = TQWarming, "five_second_warming"
		v.FiveSecondStatus, v.FiveSecondReason = TQWarming, "five_second_warming"
	} else {
		v.Status, v.Reason = TQCurrent, "qualifying_original_prints"
		v.FiveSecondStatus, v.FiveSecondReason = TQCurrent, "qualifying_original_prints"
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
	v := SpreadView{Status: TQUnavailable, Reason: "coverage"}
	if !m.quoteCoverage.active || target == nil {
		return v
	}
	latest := m.latestQuote
	if latest == nil {
		v.Status, v.Reason = TQWarming, "coverage_warming"
		return v
	}
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
