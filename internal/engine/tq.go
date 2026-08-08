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

const (
	maximumTQSymbols          = 20
	maximumTradesPerSymbol    = 50_000
	maximumTradeFingerprints  = 100_000
	maximumQuotesPerSymbol    = 20_000
	maximumTradesGlobal       = 500_000
	maximumFingerprintsGlobal = 1_000_000
	maximumQuotesGlobal       = 400_000
)

type tqRetentionLimits struct {
	tradesPerSymbol, tradesGlobal, fingerprintsPerSymbol, fingerprintsGlobal, quotesPerSymbol, quotesGlobal int
}

func defaultTQRetentionLimits() tqRetentionLimits {
	return tqRetentionLimits{maximumTradesPerSymbol, maximumTradesGlobal, maximumTradeFingerprints, maximumFingerprintsGlobal, maximumQuotesPerSymbol, maximumQuotesGlobal}
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
	symbol          string
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
func (c TQCommand) Symbol() string          { return c.symbol }

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
	quotes                             []tqQuote
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
	integrity                                                                                  uint64
	tradeCount, quoteCount                                                                     int
	fingerprintCount                                                                           int
	globalBound                                                                                bool
	aggregateOnly                                                                              bool
	pressure                                                                                   tqPressureState
	restoring                                                                                  bool
	restoreNotBefore                                                                           time.Time
	commandsIssued, commandsAcknowledged, commandsFailed, commandsFenced, commandResultsFenced uint64
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
	ValidDuration      time.Duration
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
	KnownPresent, KnownAbsent, Unknown                                      int
	RetainedTrades, RetainedQuotes, RetainedFingerprints                    int
}

type TQCommandAccountingView struct {
	Issued, Pending, Acknowledged, Failed, Fenced, ResultFenced uint64
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
	ShedTradesQuotes                    bool
	PressureMisses                      uint32
	PressureTransitions, PressureFenced uint64
	Accounting                          TQAccountingView
	Commands                            TQCommandAccountingView
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
		(v.action == TQSubscribe || v.action == TQUnsubscribe) && v.symbol != "" && len(v.symbol) <= maximumSymbolBytes && v.done != nil
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
		pressure.nextSequence = maxUint64(1, pressure.nextSequence)
		*s = tqState{
			epoch: e.state.liveEpoch, revision: previous.revision, nextToken: maxUint64(e.state.aggregateWriteToken+1, maxUint64(1, previous.nextToken)), members: make(map[string]*tqSymbolState),
			consumed: previous.consumed, applied: previous.applied, duplicate: previous.duplicate, rejected: previous.rejected, fenced: previous.fenced,
			pressureShed: previous.pressureShed, integrity: previous.integrity, globalBound: previous.globalBound, aggregateOnly: previous.aggregateOnly,
			commandsIssued: previous.commandsIssued, commandsAcknowledged: previous.commandsAcknowledged, commandsFailed: previous.commandsFailed,
			commandsFenced: commandFenced, commandResultsFenced: previous.commandResultsFenced,
			pressure: pressure,
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
				if len(member.trades) == 0 && len(member.quotes) == 0 && len(member.fingerprints) == 0 {
					delete(s.members, symbol)
				}
			}
		}
	}
	if s.pending != nil && s.pending.action == TQSubscribe && !s.dispatched && !desiredSet[s.pending.symbol] {
		s.pending, s.dispatched = nil, false
		s.commandsFenced++
	}
	if s.pending != nil || !e.state.liveEpochActive || e.state.lifecycle != lifecycleLive {
		return
	}
	if s.pressure.mode == TQPressureDegraded {
		return
	}
	if s.aggregateOnly {
		for index := len(desired) - 1; index >= 0; index-- {
			member := s.members[desired[index]]
			if member != nil && (member.present || (member.unknown && !member.cleanupAttempted)) {
				e.issueTQCommandLocked(TQUnsubscribe, desired[index])
				return
			}
		}
	}
	for symbol, member := range s.members {
		if (member.present || (member.unknown && !member.cleanupAttempted)) && (s.aggregateOnly || member.bound || member.resetRequired || !desiredSet[symbol]) {
			e.issueTQCommandLocked(TQUnsubscribe, symbol)
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
	for _, symbol := range desired {
		member := s.member(symbol)
		if member.bound {
			continue
		}
		if member.unknown {
			if !member.cleanupAttempted {
				e.issueTQCommandLocked(TQUnsubscribe, symbol)
			}
			return
		}
		if !member.present {
			if wireMembers >= maximumTQSymbols {
				return
			}
			if s.restoring && now.Before(s.restoreNotBefore) {
				return
			}
			e.issueTQCommandLocked(TQSubscribe, symbol)
			return
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
	s.quoteCount -= len(m.quotes)
	s.fingerprintCount -= len(m.fingerprints)
	m.trades, m.quotes = nil, nil
	m.fingerprints = make(map[string]tqFingerprint)
}

func (s *tqState) prune(target, engineTime time.Time) {
	tradeCutoff, quoteCutoff, fingerprintCutoff := target.Add(-6*time.Second), target.Add(-12*time.Second), engineTime.Add(-16*time.Minute)
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
		quotes := m.quotes[:0]
		for _, item := range m.quotes {
			if item.at.Before(quoteCutoff) {
				s.quoteCount--
				continue
			}
			quotes = append(quotes, item)
		}
		m.quotes = quotes
		for key, item := range m.fingerprints {
			if item.at.Before(fingerprintCutoff) {
				delete(m.fingerprints, key)
				s.fingerprintCount--
			}
		}
	}
}

func (e *Engine) issueTQCommandLocked(action TQAction, symbol string) {
	s := &e.state.tq
	if s.nextToken == 0 {
		e.enterTQGlobalBoundLocked()
		return
	}
	command := &TQCommand{bindingIdentity: e.state.binding.identity, connectionEpoch: e.state.liveEpoch, commandToken: s.nextToken, action: action, symbol: symbol, done: make(chan Disposition, 1)}
	s.nextToken++
	s.pending, s.dispatched = command, false
	s.commandsIssued++
	if action == TQUnsubscribe {
		m := s.member(symbol)
		m.tradeCoverage.active, m.quoteCoverage.active = false, false
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
	e.enterTQAggregateOnlyLocked()
}

func (e *Engine) enterTQAggregateOnlyLocked() {
	s := &e.state.tq
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
	m := s.member(v.command.symbol)
	s.pending, s.dispatched = nil, false
	if v.outcome != ControlSucceeded {
		s.commandsFailed++
		m.present, m.unknown = false, true
		m.tradeCoverage.active, m.quoteCoverage.active = false, false
		if v.command.action == TQUnsubscribe {
			m.cleanupAttempted = true
		}
		s.rejected++
		return DispositionTQRejected, ReasonControlOutcome
	}
	s.commandsAcknowledged++
	if v.command.action == TQUnsubscribe {
		m.present, m.unknown, m.cleanupAttempted = false, false, false
		m.resetRequired = false
		s.releaseMember(m)
		m.unequalRepeat, m.lifecycleObserved = false, false
		if !s.desiredContains(v.command.symbol) {
			delete(s.members, v.command.symbol)
		}
	} else {
		if s.aggregateOnly || m.bound || m.resetRequired || !s.desiredContains(v.command.symbol) {
			// A subscribe already written before containment may still have been
			// applied by the provider. Preserve that cleanup liability, but never
			// reopen causal coverage after additions have stopped.
			m.present, m.unknown, m.cleanupAttempted = true, false, false
			m.tradeCoverage.active, m.quoteCoverage.active = false, false
			s.releaseMember(m)
			s.applied++
			return DispositionTQApplied, ReasonNone
		}
		start := v.receiptTime
		if start.Before(e.state.binding.sessionStart) {
			start = e.state.binding.sessionStart
		}
		if start.After(e.state.binding.sessionEnd) {
			start = e.state.binding.sessionEnd
		}
		m.present, m.unknown, m.cleanupAttempted = true, false, false
		m.tradeCoverage = tqCoverage{active: true, epoch: v.command.connectionEpoch, start: start, ack: v.position, greatest: v.position}
		m.quoteCoverage = m.tradeCoverage
		s.releaseMember(m)
		m.unequalRepeat, m.lifecycleObserved = false, false
		if s.restoring {
			s.restoreNotBefore = node.admissionTime.Add(e.tqPressurePolicy.restoreInterval)
			allPresent := true
			for _, symbol := range s.desired {
				member := s.member(symbol)
				if !member.present || member.resetRequired {
					allPresent = false
					break
				}
			}
			if allPresent {
				s.restoring = false
			}
		}
	}
	s.applied++
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyTQDropLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, s := node.tqDrop.TQDropInput, &e.state.tq
	s.consumed++
	if e.state.binding == nil || v.BindingIdentity != e.state.binding.identity || v.TradingDate != e.state.binding.tradingDate || v.Live.ConnectionEpoch != e.state.liveEpoch {
		s.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if v.Symbol == "" {
		if v.DropReason == "optional_tq_shed" && s.pressure.mode != TQPressureNormal {
			s.pressureShed++
			return DispositionTQRejected, ReasonPressure
		}
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
	cutoff := node.admissionTime.Add(-10 * time.Second)
	if e.state.committedT != nil {
		cutoff = e.state.committedT.Add(-10 * time.Second)
	}
	if v.SIPTime.Before(cutoff) {
		s.rejected++
		return DispositionTQRejected, ReasonHistoricalContext
	}
	limits := e.tqLimits
	if len(m.quotes) >= limits.quotesPerSymbol {
		m.bound = true
		m.tradeCoverage.active, m.quoteCoverage.active = false, false
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	if s.quoteCount >= limits.quotesGlobal {
		e.enterTQGlobalBoundLocked()
		s.integrity++
		return DispositionTQRejected, ReasonAccounting
	}
	quality := quoteQuality(v.ConditionsClassified, v.IndicatorsClassified, v.Conditions, v.Indicators)
	m.quotes = append(m.quotes, tqQuote{at: v.SIPTime, receipt: v.ReceiptTime, position: v.Live, bid: v.BidPrice, ask: v.AskPrice, bidPresent: v.BidPresent, askPresent: v.AskPresent, quality: quality})
	sort.SliceStable(m.quotes, func(i, j int) bool {
		if m.quotes[i].at.Equal(m.quotes[j].at) {
			return compareLive(m.quotes[i].position, m.quotes[j].position) < 0
		}
		return m.quotes[i].at.Before(m.quotes[j].at)
	})
	s.quoteCount++
	s.applied++
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
		Pressure: pressureMode, ShedTradesQuotes: shedding, PressureMisses: s.pressure.consecutiveMisses,
		PressureTransitions: s.pressure.transitions, PressureFenced: s.pressure.fenced}
	if s.pending != nil {
		result.CommandPending, result.PendingAction, result.PendingSymbol = true, s.pending.action, s.pending.symbol
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
	if value.PublicationID == 0 || value.PublicationID != publicationID || len(value.Desired) > maximumTQSymbols || len(value.Rows) != len(value.Desired) || value.Accounting.KnownPresent < 0 || value.Accounting.KnownAbsent < 0 || value.Accounting.Unknown < 0 ||
		value.Accounting.RetainedTrades < 0 || value.Accounting.RetainedQuotes < 0 || value.Accounting.RetainedFingerprints < 0 ||
		value.Accounting.Consumed != value.Accounting.Applied+value.Accounting.Duplicate+value.Accounting.Rejected+value.Accounting.Fenced+value.Accounting.PressureShed+value.Accounting.Integrity ||
		value.Commands.Pending > 1 || value.Commands.Issued != value.Commands.Pending+value.Commands.Acknowledged+value.Commands.Failed+value.Commands.Fenced ||
		value.CommandPending != (value.Commands.Pending == 1) || value.Bounds && !value.AggregateOnly {
		return false
	}
	if value.Pressure != TQPressureNormal && value.Pressure != TQPressureDegraded && value.Pressure != TQPressureAggregateOnly {
		return false
	}
	if value.AggregateOnly != (value.Pressure == TQPressureAggregateOnly) || value.ShedTradesQuotes != (value.Pressure != TQPressureNormal) {
		return false
	}
	if value.Accounting.KnownPresent+value.Accounting.KnownAbsent+value.Accounting.Unknown > 2*maximumTQSymbols ||
		value.Accounting.RetainedTrades > maximumTradesGlobal || value.Accounting.RetainedQuotes > maximumQuotesGlobal || value.Accounting.RetainedFingerprints > maximumFingerprintsGlobal {
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
			!finiteTQ(row.Spread.Cents) || !finiteTQ(row.Spread.BasisPoints) || row.Spread.ValidDuration < 0 {
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
		v.FiveSecondStatus, v.FiveSecondReason = TQWarming, "coverage_warming"
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

type weightedSpread struct {
	cents, bps float64
	duration   time.Duration
}

func spreadView(m *tqSymbolState, target *time.Time) SpreadView {
	v := SpreadView{Status: TQUnavailable, Reason: "coverage"}
	if !m.quoteCoverage.active || target == nil {
		return v
	}
	if target.Sub(m.quoteCoverage.start) < 8*time.Second {
		v.Status, v.Reason = TQWarming, "coverage_warming"
	}
	windowStart := target.Add(-10 * time.Second)
	segments := make([]weightedSpread, 0, len(m.quotes))
	var latest *tqQuote
	for i := range m.quotes {
		q := m.quotes[i]
		if !q.at.Before(*target) {
			continue
		}
		if latest == nil || q.at.After(latest.at) || (q.at.Equal(latest.at) && compareLive(q.position, latest.position) > 0) {
			copy := q
			latest = &copy
		}
		start := q.at
		if start.Before(windowStart) {
			start = windowStart
		}
		if start.Before(m.quoteCoverage.start) {
			start = m.quoteCoverage.start
		}
		end := q.at.Add(2 * time.Second)
		if i+1 < len(m.quotes) && m.quotes[i+1].at.Before(end) {
			end = m.quotes[i+1].at
		}
		if end.After(*target) {
			end = *target
		}
		if !start.Before(end) || !q.bidPresent || !q.askPresent || q.ask < q.bid {
			continue
		}
		cents := 100 * (q.ask - q.bid)
		bps := 10000 * (q.ask - q.bid) / ((q.ask + q.bid) / 2)
		segments = append(segments, weightedSpread{cents: cents, bps: bps, duration: end.Sub(start)})
		v.ValidDuration += end.Sub(start)
	}
	if latest == nil {
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
	if target.Sub(latest.at) > 2*time.Second {
		v.Status, v.Reason = TQStale, "stale_quote"
		return v
	}
	if v.ValidDuration < 8*time.Second {
		if target.Sub(m.quoteCoverage.start) >= 8*time.Second {
			v.Status, v.Reason = TQUnavailable, "insufficient_coverage"
		}
		return v
	}
	v.Status, v.Reason = TQCurrent, ""
	sort.Slice(segments, func(i, j int) bool { return segments[i].cents < segments[j].cents })
	half, accumulated := v.ValidDuration/2, time.Duration(0)
	for _, segment := range segments {
		accumulated += segment.duration
		if accumulated >= half {
			v.Cents = segment.cents
			break
		}
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].bps < segments[j].bps })
	accumulated = 0
	for _, segment := range segments {
		accumulated += segment.duration
		if accumulated >= half {
			v.BasisPoints = segment.bps
			break
		}
	}
	return v
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
