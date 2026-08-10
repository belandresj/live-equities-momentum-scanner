package massive

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/coder/websocket"
)

const (
	maximumEndpointBytes   = 2048
	maximumCredentialBytes = 4096
	maximumTQSymbols       = 20
)

var (
	errAdapterConfig   = errors.New("massive live adapter configuration is invalid")
	errCommand         = errors.New("massive live adapter command is invalid")
	errAttemptActive   = errors.New("massive live adapter already has an active attempt")
	errAttemptState    = errors.New("massive live attempt state is invalid")
	errTransportFailed = errors.New("massive live transport failed")
	errCommandInFlight = errors.New("massive live command already in flight")
	errCommandWrite    = errors.New("massive live command write failed")
)

type OperationalDurations struct {
	Dial, HandshakeStep, HandshakeTotal  time.Duration
	HeartbeatInterval, HeartbeatDeadline time.Duration
	Write, Close                         time.Duration
}

func (d OperationalDurations) valid() bool {
	return d.Dial > 0 && d.HandshakeStep > 0 && d.HandshakeTotal > 0 &&
		d.HeartbeatInterval > 0 && d.HeartbeatDeadline > 0 && d.Write > 0 && d.Close > 0
}

type LiveAdapterConfig struct {
	Endpoint   string
	Credential string
	Queue      LiveQueueConfig
	Clock      func() time.Time
}

type OpenAggregateEpoch struct {
	BindingIdentity string
	CommandToken    uint64
	Durations       OperationalDurations
}

type TQCommandAction string

const (
	TQSubscribe   TQCommandAction = "subscribe"
	TQUnsubscribe TQCommandAction = "unsubscribe"
)

type ChangeTQCommand struct {
	BindingIdentity string
	ConnectionEpoch uint64
	CommandToken    uint64
	Action          TQCommandAction
	Symbols         []string
	engineCommand   engine.TQCommand
}

func ChangeTQCommandFromEngine(command engine.TQCommand) (ChangeTQCommand, error) {
	action := TQSubscribe
	if command.Action() == engine.TQUnsubscribe {
		action = TQUnsubscribe
	} else if command.Action() != engine.TQSubscribe {
		return ChangeTQCommand{}, errCommand
	}
	if command.BindingIdentity() == "" || command.ConnectionEpoch() == 0 || command.CommandToken() == 0 || command.Symbol() == "" {
		return ChangeTQCommand{}, errCommand
	}
	return ChangeTQCommand{BindingIdentity: command.BindingIdentity(), ConnectionEpoch: command.ConnectionEpoch(), CommandToken: command.CommandToken(), Action: action, Symbols: []string{command.Symbol()}, engineCommand: command}, nil
}

type CloseCause string

const (
	CloseSessionEnd     CloseCause = "session_end"
	CloseControlledStop CloseCause = "controlled_stop"
	CloseSuperseded     CloseCause = "superseded"
	CloseIntegrityLoss  CloseCause = "integrity_loss"
)

type CloseEpochCommand struct {
	BindingIdentity string
	ConnectionEpoch uint64
	CommandToken    uint64
	Cause           CloseCause
}

type TerminalSource string

const (
	TerminalReader      TerminalSource = "reader"
	TerminalWriter      TerminalSource = "writer"
	TerminalHeartbeat   TerminalSource = "heartbeat"
	TerminalProtocol    TerminalSource = "protocol"
	TerminalEngineClose TerminalSource = "engine_close"
)

type TerminalReason string

const (
	TerminalDialFailed                   TerminalReason = "dial_failed"
	TerminalReadFailed                   TerminalReason = "read_failed"
	TerminalUnsupportedMessage           TerminalReason = "unsupported_message"
	TerminalHeartbeatFailureUnclassified TerminalReason = "heartbeat_failure_unclassified"
	TerminalFrameOversize                TerminalReason = "frame_oversize"
	TerminalFrameCapacity                TerminalReason = "frame_capacity"
	TerminalReceiptRegression            TerminalReason = "receipt_regression"
	TerminalStatusAmbiguous              TerminalReason = "status_ambiguous"
	TerminalIngressAmbiguity             TerminalReason = "ingress_ambiguity"
	TerminalHandshakeDeadline            TerminalReason = "handshake_deadline"
	TerminalWriteFailed                  TerminalReason = "write_failed"
	TerminalContextCanceled              TerminalReason = "context_canceled"
	TerminalCloseRequested               TerminalReason = "close_requested"
)

type DeliveryKind string

const (
	DeliveryControl               DeliveryKind = "control"
	DeliveryAggregate             DeliveryKind = "aggregate"
	DeliveryTrade                 DeliveryKind = "trade_consumer_deferred"
	DeliveryQuote                 DeliveryKind = "quote_consumer_deferred"
	DeliveryNormalizationDrop     DeliveryKind = "normalization_rejection"
	DeliveryTerminal              DeliveryKind = "terminal"
	DeliveryAggregateIngressFence DeliveryKind = "aggregate_ingress_fence"
	DeliveryLiveCoverageFence     DeliveryKind = "live_coverage_fence"
)

type TerminalResult struct {
	BindingIdentity            string
	ConnectionEpoch            uint64
	Position                   engine.LivePosition
	CompletedAt                time.Time
	Source                     TerminalSource
	Reason                     TerminalReason
	CloseCause                 CloseCause
	FramesFenced               uint64
	PendingCommandToken        uint64
	PendingExpectedStatusCount int
	PendingCommandOutcome      engine.ConnectionControlOutcome
}

type AdapterDelivery struct {
	Kind                                     DeliveryKind
	Position                                 engine.LivePosition
	Control                                  engine.ConnectionControlInput
	Aggregate                                engine.AggregateInput
	Trade                                    NormalizedTrade
	Quote                                    NormalizedQuote
	Rejection                                LiveRejection
	Terminal                                 TerminalResult
	AggregateIngressFence                    AggregateIngressFenceFact
	LiveCoverageFence                        LiveCoverageFenceFact
	ExpectedStatusCount, ObservedStatusCount int
	TQAction                                 TQCommandAction
	TQSymbols                                []string
	tqCommand                                engine.TQCommand
}

type EngineDeliveryResult struct {
	Admission               engine.AdmissionResult
	ControlDisposition      engine.ConnectionControlDisposition
	AggregateDisposition    engine.AggregateDisposition
	HydrationDisposition    engine.HydrationDisposition
	LiveCoverageDisposition engine.LiveCoverageFenceDisposition
	TQDisposition           engine.Disposition
	ConsumerDeferred        bool
}

type AdapterAccounting struct {
	ConnectionAttempts                                        uint64
	AttemptsActive, AttemptsConnected                         uint64
	AttemptsFailed, AttemptsCanceled                          uint64
	CommandsStarted, CommandsPendingWrite, CommandsPendingAck uint64
	CommandsAcknowledged, CommandsFailed, CommandsAmbiguous   uint64
	CommandsCanceledOrFenced                                  uint64
}

type TQNormalizationAccounting struct {
	Classified, Normalized, Rejected uint64
}

func (a TQNormalizationAccounting) Reconciles() bool {
	return a.Classified == a.Normalized+a.Rejected
}

func (a AdapterAccounting) Reconciles() bool {
	return a.ConnectionAttempts == a.AttemptsActive+a.AttemptsConnected+a.AttemptsFailed+a.AttemptsCanceled &&
		a.CommandsStarted == a.CommandsPendingWrite+a.CommandsPendingAck+a.CommandsAcknowledged+a.CommandsFailed+a.CommandsAmbiguous+a.CommandsCanceledOrFenced
}

type liveSocket interface {
	Read(context.Context) (socketMessageType, []byte, error)
	Write(context.Context, socketMessageType, []byte) error
	Ping(context.Context) error
	Close(context.Context) error
}

type liveConnector interface {
	Dial(context.Context, string, int64) (liveSocket, error)
}

type coderConnector struct{}

func (coderConnector) Dial(ctx context.Context, endpoint string, readLimit int64) (liveSocket, error) {
	connection, _, err := websocket.Dial(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	connection.SetReadLimit(readLimit)
	return coderSocket{connection: connection}, nil
}

type coderSocket struct{ connection *websocket.Conn }

func (s coderSocket) Read(ctx context.Context) (socketMessageType, []byte, error) {
	kind, data, err := s.connection.Read(ctx)
	if err != nil {
		return 0, nil, err
	}
	switch kind {
	case websocket.MessageText:
		return socketMessageText, data, nil
	case websocket.MessageBinary:
		return socketMessageBinary, data, nil
	default:
		return 0, data, errTransportFailed
	}
}

func (s coderSocket) Write(ctx context.Context, kind socketMessageType, data []byte) error {
	messageType := websocket.MessageText
	if kind == socketMessageBinary {
		messageType = websocket.MessageBinary
	}
	return s.connection.Write(ctx, messageType, data)
}

func (s coderSocket) Ping(ctx context.Context) error { return s.connection.Ping(ctx) }

func (s coderSocket) Close(context.Context) error { return s.connection.CloseNow() }

type LiveAdapter struct {
	mu         sync.Mutex
	binding    reference.Binding
	endpoint   string
	credential string
	queue      LiveQueueConfig
	connector  liveConnector
	clock      func() time.Time
	nextEpoch  uint64
	lastToken  uint64
	active     *LiveAttempt
	accounting AdapterAccounting
}

func (*LiveAdapter) String() string   { return "massive.LiveAdapter{credential:redacted}" }
func (*LiveAdapter) GoString() string { return "massive.LiveAdapter{credential:redacted}" }

func NewLiveAdapter(binding reference.Binding, config LiveAdapterConfig) (*LiveAdapter, error) {
	if binding.Identity() == "" || config.Endpoint == "" || len(config.Endpoint) > maximumEndpointBytes || !utf8.ValidString(config.Endpoint) ||
		config.Credential == "" || len(config.Credential) > maximumCredentialBytes || !utf8.ValidString(config.Credential) ||
		!validateLiveQueueConfig(config.Queue) {
		return nil, errAdapterConfig
	}
	clock := config.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &LiveAdapter{binding: binding, endpoint: config.Endpoint, credential: config.Credential, queue: config.Queue, connector: coderConnector{}, clock: clock}, nil
}

func (a *LiveAdapter) now() time.Time { return a.clock().UTC() }

func (a *LiveAdapter) Start(ctx context.Context, command OpenAggregateEpoch) (*LiveAttempt, AdapterDelivery, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx == nil || command.BindingIdentity != a.binding.Identity() || command.CommandToken == 0 || command.CommandToken <= a.lastToken || !command.Durations.valid() {
		return nil, AdapterDelivery{}, errCommand
	}
	if a.active != nil {
		return nil, AdapterDelivery{}, errAttemptActive
	}
	if a.nextEpoch == ^uint64(0) {
		return nil, AdapterDelivery{}, errAttemptState
	}
	a.nextEpoch++
	a.lastToken = command.CommandToken
	a.accounting.ConnectionAttempts++
	a.accounting.AttemptsActive++
	a.accounting.CommandsStarted++
	a.accounting.CommandsPendingWrite++
	attempt := &LiveAttempt{
		adapter: a, binding: a.binding, epoch: a.nextEpoch, openToken: command.CommandToken,
		durations: command.Durations, endpoint: a.endpoint, credential: a.credential,
		connector: a.connector, queue: newLiveFrameQueue(a.queue), cleanupDone: make(chan struct{}), handshakeDone: make(chan struct{}), openCommandPending: true,
	}
	attempt.queue.now = a.now
	attempt.ctx, attempt.cancel = context.WithCancel(ctx)
	attempt.started = true
	a.active = attempt
	at := a.now()
	delivery := controlDelivery(a.binding.Identity(), attempt.epoch, engine.ConnectionAttempt, command.CommandToken, engine.ControlSucceeded, engine.LivePosition{}, at)
	go attempt.runHandshake()
	return attempt, delivery, nil
}

type pendingCommand struct {
	kind          CommandKind
	token         uint64
	expectedCount int
	afterSequence uint64
	deadline      time.Time
	writeFailed   bool
	writeDone     chan struct{}
	accounted     bool
	statusOutcome engine.ConnectionControlOutcome
	action        TQCommandAction
	symbols       []string
	engineCommand engine.TQCommand
}

type terminalCause struct {
	source           TerminalSource
	reason           TerminalReason
	fenceAfter       uint64
	ingressIntegrity bool
	at               time.Time
	closeCause       CloseCause
}

type LiveAttempt struct {
	mu         sync.Mutex
	nextMu     sync.Mutex
	deliveryMu sync.Mutex
	adapter    *LiveAdapter
	binding    reference.Binding
	epoch      uint64
	openToken  uint64
	durations  OperationalDurations
	endpoint   string
	credential string
	connector  liveConnector
	connection liveSocket
	queue      *liveFrameQueue

	started, handshaken, finished bool
	ctx                           context.Context
	cancel                        context.CancelFunc
	workers                       sync.WaitGroup
	terminal                      *terminalCause
	cleanupDone                   chan struct{}
	pending                       *pendingCommand
	currentFrame                  *queuedLiveFrame
	currentCursor                 *liveFrameCursor
	currentFrameCompleted         bool
	openCommandPending            bool
	openCommandPendingAck         bool
	connected                     bool
	closeCommand                  *CloseEpochCommand
	terminalDelivery              AdapterDelivery
	terminalReturned              bool
	captureToken                  uint64
	handshakeDone                 chan struct{}
	handshakeDeliveries           []AdapterDelivery
	handshakeErr                  error
	shedTQ                        atomic.Bool
	tqAccountingMu                sync.Mutex
	tqAccounting                  TQNormalizationAccounting
}

func (*LiveAttempt) String() string   { return "massive.LiveAttempt{credential:redacted}" }
func (*LiveAttempt) GoString() string { return "massive.LiveAttempt{credential:redacted}" }

func (a *LiveAttempt) Epoch() uint64 { return a.epoch }

func (a *LiveAttempt) SetTQShedding(enabled bool) { a.shedTQ.Store(enabled) }

func (a *LiveAttempt) TQNormalizationAccounting() TQNormalizationAccounting {
	a.tqAccountingMu.Lock()
	defer a.tqAccountingMu.Unlock()
	return a.tqAccounting
}

func (a *LiveAttempt) accountTQ(normalized bool) {
	a.tqAccountingMu.Lock()
	a.tqAccounting.Classified++
	if normalized {
		a.tqAccounting.Normalized++
	} else {
		a.tqAccounting.Rejected++
	}
	a.tqAccountingMu.Unlock()
}

func (a *LiveAttempt) runHandshake() {
	deliveries, err := a.performHandshake()
	a.mu.Lock()
	a.handshakeDeliveries = append([]AdapterDelivery(nil), deliveries...)
	a.handshakeErr = err
	close(a.handshakeDone)
	a.mu.Unlock()
}

func (a *LiveAttempt) Handshake(ctx context.Context) ([]AdapterDelivery, error) {
	if ctx == nil {
		return nil, errAttemptState
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-a.handshakeDone:
		a.mu.Lock()
		defer a.mu.Unlock()
		return append([]AdapterDelivery(nil), a.handshakeDeliveries...), a.handshakeErr
	}
}

func (a *LiveAttempt) performHandshake() ([]AdapterDelivery, error) {
	totalCtx, totalCancel := context.WithTimeout(a.ctx, a.durations.HandshakeTotal)
	defer totalCancel()
	dialCtx, dialCancel := context.WithTimeout(totalCtx, a.durations.Dial)
	connection, err := a.connector.Dial(dialCtx, a.endpoint, int64(a.queue.config.MaxFrameBytes))
	dialCancel()
	if err != nil {
		a.triggerTerminal(TerminalReader, TerminalDialFailed, 0, false)
		return nil, errTransportFailed
	}
	a.mu.Lock()
	if a.terminal != nil || a.finished {
		a.mu.Unlock()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), a.durations.Close)
		_ = connection.Close(closeCtx)
		closeCancel()
		return nil, errTransportFailed
	}
	a.connection = connection
	a.startWorkers()
	a.mu.Unlock()

	deliveries := make([]AdapterDelivery, 0, 4)
	connected, err := a.awaitHandshakeStatus(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseConnected, CommandKind: CommandConnection, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.ConnectionEstablished)
	if err != nil {
		if connected.Kind != "" {
			deliveries = append(deliveries, connected)
		}
		return deliveries, err
	}
	deliveries = append(deliveries, connected)
	a.accountConnected()

	authPayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "auth", Params: a.credential})
	if err := a.write(totalCtx, authPayload); err != nil {
		a.triggerTerminal(TerminalWriter, TerminalWriteFailed, 0, false)
		return deliveries, errCommandWrite
	}
	authenticated, err := a.awaitHandshakeStatus(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseAuthSuccess, CommandKind: CommandAuthentication, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.AuthenticationResult)
	if err != nil {
		if authenticated.Kind != "" {
			deliveries = append(deliveries, authenticated)
		}
		return deliveries, err
	}
	deliveries = append(deliveries, authenticated)

	aggregatePayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "subscribe", Params: "A.*"})
	if err := a.write(totalCtx, aggregatePayload); err != nil {
		deliveries = append(deliveries, controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlFailed, engine.LivePosition{}, a.adapter.now()))
		a.accountOpenCommand(engine.ControlFailed, false)
		a.triggerTerminal(TerminalWriter, TerminalWriteFailed, 0, false)
		return deliveries, errCommandWrite
	}
	a.accountOpenWriteSucceeded()
	deliveries = append(deliveries, controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlSucceeded, engine.LivePosition{}, a.adapter.now()))
	acknowledged, err := a.awaitHandshakeStatus(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.AggregateSubscriptionResult)
	if err != nil {
		if acknowledged.Kind != "" {
			deliveries = append(deliveries, acknowledged)
		}
		return deliveries, err
	}
	deliveries = append(deliveries, acknowledged)
	a.accountOpenCommand(acknowledged.Control.Outcome, true)
	a.mu.Lock()
	a.handshaken = true
	a.mu.Unlock()
	return deliveries, nil
}

func (a *LiveAttempt) accountConnected() {
	a.mu.Lock()
	if a.connected {
		a.mu.Unlock()
		return
	}
	a.connected = true
	a.mu.Unlock()
	a.adapter.mu.Lock()
	if a.adapter.accounting.AttemptsActive > 0 {
		a.adapter.accounting.AttemptsActive--
		a.adapter.accounting.AttemptsConnected++
	}
	a.adapter.mu.Unlock()
}

func (a *LiveAttempt) accountOpenWriteSucceeded() {
	a.mu.Lock()
	if !a.openCommandPending || a.openCommandPendingAck {
		a.mu.Unlock()
		return
	}
	a.openCommandPendingAck = true
	a.mu.Unlock()
	a.adapter.mu.Lock()
	a.adapter.accounting.CommandsPendingWrite--
	a.adapter.accounting.CommandsPendingAck++
	a.adapter.mu.Unlock()
}

func (a *LiveAttempt) accountOpenCommand(outcome engine.ConnectionControlOutcome, fromAck bool) {
	a.mu.Lock()
	if !a.openCommandPending {
		a.mu.Unlock()
		return
	}
	pendingAck := a.openCommandPendingAck
	a.openCommandPending = false
	a.openCommandPendingAck = false
	a.mu.Unlock()
	a.adapter.mu.Lock()
	if pendingAck || fromAck {
		if a.adapter.accounting.CommandsPendingAck > 0 {
			a.adapter.accounting.CommandsPendingAck--
		}
	} else if a.adapter.accounting.CommandsPendingWrite > 0 {
		a.adapter.accounting.CommandsPendingWrite--
	}
	switch outcome {
	case engine.ControlSucceeded:
		a.adapter.accounting.CommandsAcknowledged++
	case engine.ControlFailed:
		a.adapter.accounting.CommandsFailed++
	default:
		a.adapter.accounting.CommandsAmbiguous++
	}
	a.adapter.mu.Unlock()
}

func (a *LiveAttempt) startWorkers() {
	a.workers.Add(2)
	go a.readWorker()
	go a.heartbeatWorker()
}

func (a *LiveAttempt) readWorker() {
	defer a.workers.Done()
	for {
		kind, data, err := a.connection.Read(a.ctx)
		if err != nil {
			reason := TerminalReadFailed
			if a.ctx.Err() != nil {
				reason = TerminalContextCanceled
			}
			a.triggerTerminal(TerminalReader, reason, 0, false)
			return
		}
		if kind != socketMessageText && kind != socketMessageBinary {
			a.queue.tryEnqueue(a.epoch, kind, a.adapter.now(), nil)
			a.triggerTerminal(TerminalProtocol, TerminalUnsupportedMessage, 0, false)
			return
		}
		receivedAt := a.adapter.now()
		_, reason := a.queue.tryEnqueue(a.epoch, kind, receivedAt, data)
		if reason != FrameAdmitted {
			switch reason {
			case FrameRejectedOversize:
				a.triggerTerminal(TerminalProtocol, TerminalFrameOversize, 0, true)
			case FrameRejectedCapacity:
				a.triggerTerminal(TerminalProtocol, TerminalFrameCapacity, 0, true)
			case FrameRejectedReceipt:
				a.triggerTerminal(TerminalProtocol, TerminalReceiptRegression, 0, true)
			default:
				if a.ctx.Err() == nil {
					a.triggerTerminal(TerminalProtocol, TerminalContextCanceled, 0, false)
				}
			}
			return
		}
	}
}

func (a *LiveAttempt) heartbeatWorker() {
	defer a.workers.Done()
	ticker := time.NewTicker(a.durations.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(a.ctx, a.durations.HeartbeatDeadline)
			err := a.connection.Ping(pingCtx)
			cancel()
			if err != nil {
				reason := TerminalHeartbeatFailureUnclassified
				if a.ctx.Err() != nil {
					reason = TerminalContextCanceled
				}
				a.triggerTerminal(TerminalHeartbeat, reason, 0, false)
				return
			}
		}
	}
}

func (a *LiveAttempt) write(parent context.Context, payload []byte) error {
	if parent == nil {
		return errCommandWrite
	}
	writeCtx, cancel := context.WithTimeout(a.ctx, a.durations.Write)
	stopParent := context.AfterFunc(parent, cancel)
	defer stopParent()
	defer cancel()
	if err := a.connection.Write(writeCtx, socketMessageText, payload); err != nil {
		return errCommandWrite
	}
	return nil
}

func (a *LiveAttempt) awaitHandshakeStatus(parent context.Context, status StatusContext, kind engine.ConnectionControlKind) (AdapterDelivery, error) {
	stepCtx, cancel := context.WithTimeout(parent, a.durations.HandshakeStep)
	defer cancel()
	frame, ok := a.queue.pop(stepCtx)
	if !ok {
		a.triggerTerminal(TerminalProtocol, TerminalHandshakeDeadline, 0, false)
		return AdapterDelivery{}, errTransportFailed
	}
	if frame.terminal {
		delivery := a.finishTerminal(frame)
		return delivery, errTransportFailed
	}
	cursor := newLiveFrameCursor(LiveFrame{Binding: a.binding, ConnectionEpoch: a.epoch, FrameSequence: frame.sequence, ReceivedAt: frame.receivedAt, Data: frame.data}, &status, LiveNormalizationOptions{})
	result, first := cursor.Next()
	_, extra := cursor.Next()
	a.queue.complete(frame, false)
	if !first || extra || result.Kind != LiveResultStatus || !cursor.Accounting().Reconciles() {
		a.triggerTerminal(TerminalProtocol, TerminalStatusAmbiguous, frame.sequence, true)
		position := engine.LivePosition{ConnectionEpoch: a.epoch, FrameSequence: frame.sequence}
		return controlDelivery(a.binding.Identity(), a.epoch, engine.IngressIntegrityFailure, 0, engine.ControlAmbiguous, position, frame.receivedAt), errTransportFailed
	}
	outcome := engine.ControlSucceeded
	if result.Status.Disposition == StatusFailed {
		outcome = engine.ControlFailed
	} else if result.Status.Disposition != StatusAcknowledged {
		outcome = engine.ControlAmbiguous
	}
	delivery := controlDelivery(a.binding.Identity(), a.epoch, kind, a.openToken, outcome, result.Position, result.Status.ReceiptTime)
	if outcome != engine.ControlSucceeded {
		a.triggerTerminal(TerminalProtocol, TerminalStatusAmbiguous, frame.sequence, true)
		return delivery, errTransportFailed
	}
	return delivery, nil
}

func (a *LiveAttempt) ChangeTQ(ctx context.Context, command ChangeTQCommand) (AdapterDelivery, error) {
	a.mu.Lock()
	if ctx == nil || !a.handshaken || a.finished || a.terminal != nil || a.pending != nil {
		a.mu.Unlock()
		return AdapterDelivery{}, errCommandInFlight
	}
	if !a.validTQCommand(command) {
		a.mu.Unlock()
		return AdapterDelivery{}, errCommand
	}
	kind := CommandTradeQuoteSubscribe
	engineKind := engine.TradeQuoteCommandWriteResult
	if command.Action == TQUnsubscribe {
		kind = CommandTradeQuoteUnsubscribe
	}
	// Operational deadlines use the process clock, not the injected market
	// receipt clock. A deterministic or replay clock may be static or skewed
	// relative to wall time and must not shorten or extend command I/O bounds.
	a.pending = &pendingCommand{kind: kind, token: command.CommandToken, expectedCount: 2 * len(command.Symbols), afterSequence: a.queue.snapshot().FramesRead, deadline: time.Now().Add(a.durations.HandshakeStep), writeDone: make(chan struct{}), action: command.Action, symbols: append([]string(nil), command.Symbols...), engineCommand: command.engineCommand}
	a.workers.Add(1)
	a.adapter.mu.Lock()
	a.adapter.lastToken = command.CommandToken
	a.adapter.accounting.CommandsStarted++
	a.adapter.accounting.CommandsPendingWrite++
	a.adapter.mu.Unlock()
	a.mu.Unlock()
	defer a.workers.Done()

	pairs := make([]string, 0, 2*len(command.Symbols))
	for _, symbol := range command.Symbols {
		pairs = append(pairs, "T."+symbol, "Q."+symbol)
	}
	payload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: string(command.Action), Params: strings.Join(pairs, ",")})
	err := a.write(ctx, payload)
	outcome := engine.ControlSucceeded
	a.mu.Lock()
	if a.pending != nil && a.pending.token == command.CommandToken {
		a.pending.writeFailed = err != nil
		a.pending.accounted = err != nil
		close(a.pending.writeDone)
	}
	a.mu.Unlock()
	a.adapter.mu.Lock()
	a.adapter.accounting.CommandsPendingWrite--
	if err != nil {
		outcome = engine.ControlFailed
		a.adapter.accounting.CommandsFailed++
	} else {
		a.adapter.accounting.CommandsPendingAck++
	}
	a.adapter.mu.Unlock()
	// next may have entered an empty-queue wait before this command existed.
	// Wake it after write accounting is settled so it can install the command
	// deadline even when the provider sends no acknowledgement or later frame.
	a.queue.requestRecheck()
	delivery := controlDelivery(a.binding.Identity(), a.epoch, engineKind, command.CommandToken, outcome, engine.LivePosition{}, a.adapter.now())
	delivery.TQAction = command.Action
	delivery.TQSymbols = append([]string(nil), command.Symbols...)
	delivery.tqCommand = command.engineCommand
	if err != nil {
		return delivery, errCommandWrite
	}
	return delivery, nil
}

func (a *LiveAttempt) validTQCommand(command ChangeTQCommand) bool {
	if command.BindingIdentity != a.binding.Identity() || command.ConnectionEpoch != a.epoch || command.CommandToken == 0 ||
		(command.Action != TQSubscribe && command.Action != TQUnsubscribe) || len(command.Symbols) == 0 || len(command.Symbols) > maximumTQSymbols {
		return false
	}
	if command.engineCommand.CommandToken() != 0 && (command.engineCommand.BindingIdentity() != command.BindingIdentity || command.engineCommand.ConnectionEpoch() != command.ConnectionEpoch ||
		command.engineCommand.CommandToken() != command.CommandToken || len(command.Symbols) != 1 || command.engineCommand.Symbol() != command.Symbols[0] ||
		command.engineCommand.Action() == engine.TQSubscribe != (command.Action == TQSubscribe)) {
		return false
	}
	a.adapter.mu.Lock()
	validToken := command.CommandToken > a.adapter.lastToken
	a.adapter.mu.Unlock()
	if !validToken || !slices.IsSorted(command.Symbols) {
		return false
	}
	for index, symbol := range command.Symbols {
		if symbol == "" || len(symbol) > 64 || (index > 0 && symbol == command.Symbols[index-1]) || !bindingHasSymbol(a.binding, symbol) {
			return false
		}
	}
	return true
}

func bindingHasSymbol(binding reference.Binding, symbol string) bool {
	symbols := binding.UniverseSymbols()
	index, found := slices.BinarySearch(symbols, symbol)
	return found && index >= 0
}

func (a *LiveAttempt) Close(command CloseEpochCommand) error {
	a.mu.Lock()
	if a.closeCommand != nil {
		accepted := *a.closeCommand == command
		a.mu.Unlock()
		if accepted {
			return nil
		}
		return errCommand
	}
	if !a.started || a.finished || a.terminal != nil || command.BindingIdentity != a.binding.Identity() || command.ConnectionEpoch != a.epoch || command.CommandToken == 0 || !validCloseCause(command.Cause) {
		a.mu.Unlock()
		return errCommand
	}
	a.adapter.mu.Lock()
	if command.CommandToken <= a.adapter.lastToken {
		a.adapter.mu.Unlock()
		a.mu.Unlock()
		return errCommand
	}
	a.adapter.lastToken = command.CommandToken
	a.adapter.accounting.CommandsStarted++
	a.adapter.accounting.CommandsAcknowledged++
	a.adapter.mu.Unlock()
	a.closeCommand = &CloseEpochCommand{BindingIdentity: command.BindingIdentity, ConnectionEpoch: command.ConnectionEpoch, CommandToken: command.CommandToken, Cause: command.Cause}
	cause := &terminalCause{source: TerminalEngineClose, reason: TerminalCloseRequested, closeCause: command.Cause, at: a.adapter.now()}
	a.beginTerminalLocked(cause)
	a.mu.Unlock()
	return nil
}

func validCloseCause(cause CloseCause) bool {
	return cause == CloseSessionEnd || cause == CloseControlledStop || cause == CloseSuperseded || cause == CloseIntegrityLoss
}

// next is deliberately package-private. Production delivery must use
// DeliverNextToEngine so dequeue, engine admission, and completion are one
// serialized causal operation.
func (a *LiveAttempt) next(ctx context.Context) (AdapterDelivery, bool) {
	if ctx == nil || ctx.Err() != nil {
		return AdapterDelivery{}, false
	}
	a.nextMu.Lock()
	defer a.nextMu.Unlock()
	for {
		if a.currentCursor != nil {
			result, ok := a.currentCursor.Next()
			if ok {
				delivery, emit := a.mapResult(result, *a.currentFrame)
				if emit {
					return delivery, true
				}
				continue
			}
			if !a.currentFrameCompleted {
				a.queue.complete(*a.currentFrame, false)
			}
			a.currentFrame, a.currentCursor, a.currentFrameCompleted = nil, nil, false
			continue
		}
		a.mu.Lock()
		if a.finished && a.terminalReturned {
			a.mu.Unlock()
			return AdapterDelivery{}, false
		}
		if !a.handshaken && a.terminal == nil {
			a.mu.Unlock()
			return AdapterDelivery{}, false
		}
		var statusContext *StatusContext
		if a.pending != nil {
			if !time.Now().Before(a.pending.deadline) {
				pending := a.pending
				a.mu.Unlock()
				if !a.detachAndAccountPending(pending, engine.ControlAmbiguous) {
					continue
				}
				delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.TradeQuoteSubscriptionResult, pending.token, engine.ControlAmbiguous, engine.LivePosition{}, a.adapter.now())
				delivery.ExpectedStatusCount = pending.expectedCount
				return delivery, true
			}
			statusContext = &StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseSuccess, CommandKind: a.pending.kind, CommandToken: strconv.FormatUint(a.pending.token, 10), ExpectedCount: a.pending.expectedCount}
		}
		terminal := a.terminal
		a.mu.Unlock()

		popCtx := ctx
		var cancel context.CancelFunc
		if statusContext != nil {
			a.mu.Lock()
			deadline := a.pending.deadline
			a.mu.Unlock()
			popCtx, cancel = context.WithDeadline(ctx, deadline)
		}
		frame, ok, recheck := a.queue.popOrRecheck(popCtx)
		if cancel != nil {
			cancel()
		}
		if recheck {
			continue
		}
		if !ok {
			a.mu.Lock()
			terminalNow := a.terminal != nil
			a.mu.Unlock()
			if terminalNow {
				a.nextMu.Unlock()
				select {
				case <-ctx.Done():
					a.nextMu.Lock()
					return AdapterDelivery{}, false
				case <-a.cleanupDone:
				}
				a.nextMu.Lock()
				marker, markerOK := a.queue.pop(ctx)
				if markerOK && marker.terminal {
					return a.finishTerminal(marker), true
				}
				return AdapterDelivery{}, false
			}
			if statusContext != nil && ctx.Err() == nil {
				a.mu.Lock()
				pending := a.pending
				a.mu.Unlock()
				if pending != nil && a.detachAndAccountPending(pending, engine.ControlAmbiguous) {
					delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.TradeQuoteSubscriptionResult, pending.token, engine.ControlAmbiguous, engine.LivePosition{}, a.adapter.now())
					delivery.ExpectedStatusCount = pending.expectedCount
					return delivery, true
				}
			}
			return AdapterDelivery{}, false
		}
		if frame.terminal {
			return a.finishTerminal(frame), true
		}
		if frame.kind == queuedLiveIngressFence {
			a.queue.complete(frame, false)
			return AdapterDelivery{Kind: DeliveryAggregateIngressFence, AggregateIngressFence: frame.ingressFence}, true
		}
		if frame.kind == queuedLiveCoverageFence {
			a.queue.complete(frame, false)
			return AdapterDelivery{Kind: DeliveryLiveCoverageFence, LiveCoverageFence: frame.liveCoverageFence}, true
		}
		if terminal != nil && terminal.fenceAfter > 0 && frame.sequence > terminal.fenceAfter {
			a.queue.complete(frame, true)
			continue
		}
		// A delivery loop may already be blocked in queue.pop when another
		// goroutine writes a T/Q command. Re-evaluate the pending command after
		// the frame arrives so its response receives correlation context, while
		// never correlating a frame that was read before the command write.
		statusContext = nil
		a.mu.Lock()
		pendingNow := a.pending
		if pendingNow != nil && frame.sequence > pendingNow.afterSequence && time.Now().Before(pendingNow.deadline) {
			statusContext = &StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseSuccess, CommandKind: pendingNow.kind, CommandToken: strconv.FormatUint(pendingNow.token, 10), ExpectedCount: pendingNow.expectedCount}
		}
		a.mu.Unlock()
		a.currentFrame = &frame
		a.currentCursor = newLiveFrameCursor(LiveFrame{Binding: a.binding, ConnectionEpoch: a.epoch, FrameSequence: frame.sequence, ReceivedAt: frame.receivedAt, Data: frame.data}, statusContext, LiveNormalizationOptions{ShedTradesQuotes: a.shedTQ.Load()})
		a.currentFrameCompleted = false
	}
}

// DeliverNextToEngine linearizes dequeue, engine admission, and completion so
// concurrent callers cannot admit a later raw item ahead of an earlier fence.
func (a *LiveAttempt) DeliverNextToEngine(ctx context.Context, state *engine.Engine) (EngineDeliveryResult, bool, error) {
	a.deliveryMu.Lock()
	defer a.deliveryMu.Unlock()
	delivery, ok := a.next(ctx)
	if !ok {
		return EngineDeliveryResult{}, false, nil
	}
	// Once dequeue succeeds, ownership has transferred. A separate internal
	// lifetime guarantees admission and completion; caller cancellation may
	// stop waiting for the next item but cannot drop this causal predecessor.
	result, err := DeliverToEngine(context.Background(), state, delivery)
	return result, true, err
}

// nextForProof preserves the accepted C5 adapter-boundary tests without
// exposing an unsafe production dequeue API.
func (a *LiveAttempt) nextForProof(ctx context.Context) (AdapterDelivery, bool) { return a.next(ctx) }

func (a *LiveAttempt) mapResult(result LiveResult, frame queuedLiveFrame) (AdapterDelivery, bool) {
	a.mu.Lock()
	pending := a.pending
	a.mu.Unlock()
	switch result.Kind {
	case LiveResultAggregate:
		return AdapterDelivery{Kind: DeliveryAggregate, Position: result.Position, Aggregate: result.Aggregate}, true
	case LiveResultTrade:
		a.accountTQ(true)
		return AdapterDelivery{Kind: DeliveryTrade, Position: result.Position, Trade: result.Trade}, true
	case LiveResultQuote:
		a.accountTQ(true)
		return AdapterDelivery{Kind: DeliveryQuote, Position: result.Position, Quote: result.Quote}, true
	case LiveResultRejected:
		if result.Rejection.Family == LiveFamilyTrade || result.Rejection.Family == LiveFamilyQuote {
			a.accountTQ(false)
		}
		return AdapterDelivery{Kind: DeliveryNormalizationDrop, Position: result.Position, Rejection: result.Rejection}, true
	case LiveResultAmbiguous:
		delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.IngressIntegrityFailure, 0, engine.ControlAmbiguous, result.Position, frame.receivedAt)
		a.triggerTerminal(TerminalProtocol, TerminalIngressAmbiguity, frame.sequence, true)
		return delivery, true
	case LiveResultStatus:
		if pending == nil {
			delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.IngressIntegrityFailure, 0, engine.ControlAmbiguous, result.Position, result.Status.ReceiptTime)
			a.triggerTerminal(TerminalProtocol, TerminalStatusAmbiguous, frame.sequence, true)
			return delivery, true
		}
		<-pending.writeDone
		if !pending.writeFailed {
			switch result.Status.Disposition {
			case StatusFailed:
				pending.statusOutcome = engine.ControlFailed
			case StatusAmbiguous:
				if pending.statusOutcome != engine.ControlFailed {
					pending.statusOutcome = engine.ControlAmbiguous
				}
			}
		}
		if !result.Status.FinalInFrame {
			return AdapterDelivery{}, false
		}
		outcome := engine.ControlSucceeded
		if pending.writeFailed {
			outcome = engine.ControlAmbiguous
		} else if pending.statusOutcome != "" {
			outcome = pending.statusOutcome
		}
		if !a.detachAndAccountPending(pending, outcome) {
			return AdapterDelivery{}, false
		}
		delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.TradeQuoteSubscriptionResult, pending.token, outcome, result.Position, result.Status.ReceiptTime)
		delivery.ExpectedStatusCount = pending.expectedCount
		delivery.ObservedStatusCount = result.Status.ObservedCount
		delivery.TQAction = pending.action
		delivery.TQSymbols = append([]string(nil), pending.symbols...)
		delivery.tqCommand = pending.engineCommand
		return delivery, true
	default:
		return AdapterDelivery{}, false
	}
}

func (a *LiveAttempt) detachAndAccountPending(pending *pendingCommand, outcome engine.ConnectionControlOutcome) bool {
	if pending == nil {
		return false
	}
	a.mu.Lock()
	if a.pending != pending {
		a.mu.Unlock()
		return false
	}
	a.pending = nil
	alreadyAccounted := pending.accounted
	pending.accounted = true
	a.mu.Unlock()
	if alreadyAccounted {
		return true
	}
	a.adapter.mu.Lock()
	if a.adapter.accounting.CommandsPendingAck > 0 {
		a.adapter.accounting.CommandsPendingAck--
	}
	switch outcome {
	case engine.ControlSucceeded:
		a.adapter.accounting.CommandsAcknowledged++
	case engine.ControlFailed:
		a.adapter.accounting.CommandsFailed++
	default:
		a.adapter.accounting.CommandsAmbiguous++
	}
	a.adapter.mu.Unlock()
	return true
}

func (a *LiveAttempt) triggerTerminal(source TerminalSource, reason TerminalReason, fenceAfter uint64, ingress bool) {
	a.mu.Lock()
	if a.terminal != nil || a.finished {
		a.mu.Unlock()
		return
	}
	cause := &terminalCause{source: source, reason: reason, fenceAfter: fenceAfter, ingressIntegrity: ingress, at: a.adapter.now()}
	a.beginTerminalLocked(cause)
	a.mu.Unlock()
}

func (a *LiveAttempt) beginTerminalLocked(cause *terminalCause) {
	a.terminal = cause
	if a.cancel != nil {
		a.cancel()
	}
	a.queue.closeGate()
	go a.cleanupTerminal(cause)
}

func (a *LiveAttempt) cleanupTerminal(cause *terminalCause) {
	a.mu.Lock()
	connection := a.connection
	a.mu.Unlock()
	if connection != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), a.durations.Close)
		_ = connection.Close(closeCtx)
		cancel()
	}
	a.workers.Wait()
	a.nextMu.Lock()
	if a.currentFrame != nil && !a.currentFrameCompleted {
		a.queue.complete(*a.currentFrame, true)
		a.currentFrameCompleted = true
	}
	a.currentFrame, a.currentCursor = nil, nil
	a.nextMu.Unlock()
	markerCtx, markerCancel := context.WithTimeout(context.Background(), a.durations.Close)
	marker, admitted := a.queue.enqueueTerminal(markerCtx, a.epoch, cause.at)
	markerCancel()
	if !admitted {
		marker = a.queue.fenceQueuedAndEnqueueTerminal(a.epoch, cause.at)
	}
	a.mu.Lock()
	connected := a.connected
	openPending, openPendingAck := a.openCommandPending, a.openCommandPendingAck
	pending := a.pending
	pendingUnaccounted := pending != nil && !pending.accounted
	if pendingUnaccounted {
		pending.accounted = true
	}
	a.finished = true
	a.pending = nil
	a.openCommandPending = false
	a.openCommandPendingAck = false
	a.credential = ""
	a.mu.Unlock()
	a.adapter.mu.Lock()
	if a.adapter.active == a {
		a.adapter.active = nil
	}
	if connected {
		if a.adapter.accounting.AttemptsConnected > 0 {
			a.adapter.accounting.AttemptsConnected--
		}
	} else if a.adapter.accounting.AttemptsActive > 0 {
		a.adapter.accounting.AttemptsActive--
	}
	if cause.source == TerminalEngineClose || cause.reason == TerminalContextCanceled {
		a.adapter.accounting.AttemptsCanceled++
	} else {
		a.adapter.accounting.AttemptsFailed++
	}
	if openPending {
		if openPendingAck && a.adapter.accounting.CommandsPendingAck > 0 {
			a.adapter.accounting.CommandsPendingAck--
		} else if !openPendingAck && a.adapter.accounting.CommandsPendingWrite > 0 {
			a.adapter.accounting.CommandsPendingWrite--
		}
		switch {
		case cause.source == TerminalEngineClose || cause.reason == TerminalContextCanceled:
			a.adapter.accounting.CommandsCanceledOrFenced++
		case cause.reason == TerminalStatusAmbiguous || cause.reason == TerminalIngressAmbiguity:
			a.adapter.accounting.CommandsAmbiguous++
		default:
			a.adapter.accounting.CommandsFailed++
		}
	}
	if pendingUnaccounted {
		if a.adapter.accounting.CommandsPendingAck > 0 {
			a.adapter.accounting.CommandsPendingAck--
		} else if a.adapter.accounting.CommandsPendingWrite > 0 {
			a.adapter.accounting.CommandsPendingWrite--
		}
		a.adapter.accounting.CommandsCanceledOrFenced++
	}
	a.adapter.mu.Unlock()
	accounting := a.queue.snapshot()
	position := engine.LivePosition{ConnectionEpoch: a.epoch, FrameSequence: marker.sequence}
	terminal := TerminalResult{BindingIdentity: a.binding.Identity(), ConnectionEpoch: a.epoch, Position: position, CompletedAt: cause.at, Source: cause.source, Reason: cause.reason, CloseCause: cause.closeCause, FramesFenced: accounting.FramesFenced}
	if pending != nil {
		terminal.PendingCommandToken = pending.token
		terminal.PendingExpectedStatusCount = pending.expectedCount
		terminal.PendingCommandOutcome = engine.ControlAmbiguous
	}
	controlKind, outcome := engine.ConnectionLost, engine.ControlFailed
	if cause.ingressIntegrity {
		controlKind, outcome = engine.IngressIntegrityFailure, engine.ControlAmbiguous
	}
	delivery := AdapterDelivery{Kind: DeliveryTerminal, Position: position, Control: engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: a.binding.Identity(), Kind: controlKind, ConnectionEpoch: a.epoch, Position: position, ReceiptTime: cause.at, Outcome: outcome}, Terminal: terminal}
	a.mu.Lock()
	a.terminalDelivery = delivery
	a.mu.Unlock()
	close(a.cleanupDone)
}

func (a *LiveAttempt) finishTerminal(frame queuedLiveFrame) AdapterDelivery {
	<-a.cleanupDone
	a.queue.complete(frame, false)
	a.mu.Lock()
	a.terminalReturned = true
	delivery := a.terminalDelivery
	a.mu.Unlock()
	return delivery
}

func (a *LiveAttempt) QueueAccounting() LiveQueueAccounting { return a.queue.snapshot() }

func (a *LiveAttempt) Wait(ctx context.Context) error {
	if ctx == nil {
		return errAttemptState
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.cleanupDone:
		return nil
	}
}

func (a *LiveAdapter) Accounting() AdapterAccounting {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.accounting
}

func controlDelivery(bindingID string, epoch uint64, kind engine.ConnectionControlKind, token uint64, outcome engine.ConnectionControlOutcome, position engine.LivePosition, at time.Time) AdapterDelivery {
	return AdapterDelivery{Kind: DeliveryControl, Position: position, Control: engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: bindingID, Kind: kind, ConnectionEpoch: epoch, Position: position, ReceiptTime: at.UTC(), CommandToken: token, Outcome: outcome}}
}

func DeliverToEngine(ctx context.Context, state *engine.Engine, delivery AdapterDelivery) (EngineDeliveryResult, error) {
	if ctx == nil || state == nil {
		return EngineDeliveryResult{}, errAttemptState
	}
	switch delivery.Kind {
	case DeliveryControl, DeliveryTerminal:
		if (delivery.Control.Kind == engine.TradeQuoteSubscriptionResult ||
			delivery.Control.Kind == engine.TradeQuoteCommandWriteResult && delivery.Control.Outcome != engine.ControlSucceeded) && len(delivery.TQSymbols) == 1 {
			input, inputErr := engine.NewTQCommandResultInput(delivery.tqCommand, delivery.Control.Position, delivery.Control.ReceiptTime, delivery.Control.Outcome)
			if delivery.tqCommand.CommandToken() != 0 && inputErr != nil {
				return EngineDeliveryResult{}, inputErr
			}
			if inputErr == nil {
				admission, completion := state.AdmitTQCommandResult(ctx, input)
				result := EngineDeliveryResult{Admission: admission}
				if admission != engine.AdmissionAdmitted || completion == nil {
					return result, nil
				}
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case result.TQDisposition = <-completion:
					return result, nil
				}
			}
		}
		admission, completion := state.AdmitConnectionControl(ctx, delivery.Control)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.ControlDisposition = <-completion:
			return result, nil
		}
	case DeliveryAggregate:
		admission, completion := state.AdmitAggregate(ctx, delivery.Aggregate)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.AggregateDisposition = <-completion:
			return result, nil
		}
	case DeliveryAggregateIngressFence:
		input, err := EngineAggregateIngressFence(delivery.AggregateIngressFence)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		admission, completion := state.AdmitAggregateIngressFence(ctx, input)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.HydrationDisposition = <-completion:
			return result, nil
		}
	case DeliveryLiveCoverageFence:
		input, err := EngineLiveCoverageFence(delivery.LiveCoverageFence)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		admission, completion := state.AdmitLiveCoverageFence(ctx, input)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.LiveCoverageDisposition = <-completion:
			return result, nil
		}
	case DeliveryTrade:
		input := engine.TradeInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Trade.BindingIdentity, TradingDate: delivery.Trade.TradingDate,
			Symbol: delivery.Trade.Symbol, TradeID: delivery.Trade.TradeID, Exchange: delivery.Trade.Exchange, TRFPresent: delivery.Trade.TRFPresent, TRFID: delivery.Trade.TRFID,
			Price: delivery.Trade.Price, EconomicSize: delivery.Trade.EconomicSize, EventTime: delivery.Trade.EventTime, ReceiptTime: delivery.Trade.ReceiptTime,
			TimestampBasis: string(delivery.Trade.TimestampBasis), Conditions: delivery.Trade.Conditions.Slice(), ConditionsClassified: delivery.Trade.Conditions.Classified,
			IdentityClassified: delivery.Trade.IdentityClassified, Lifecycle: string(delivery.Trade.Lifecycle), Live: delivery.Trade.Live}
		admission, completion := state.AdmitTrade(ctx, input)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.TQDisposition = <-completion:
			return result, nil
		}
	case DeliveryQuote:
		input := engine.QuoteInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Quote.BindingIdentity, TradingDate: delivery.Quote.TradingDate, Symbol: delivery.Quote.Symbol,
			SIPTime: delivery.Quote.SIPTime, ReceiptTime: delivery.Quote.ReceiptTime, BidPrice: delivery.Quote.BidPrice, AskPrice: delivery.Quote.AskPrice,
			BidPresent: delivery.Quote.BidPresent, AskPresent: delivery.Quote.AskPresent, Conditions: delivery.Quote.Conditions.Slice(), Indicators: delivery.Quote.Indicators.Slice(),
			ConditionsClassified: delivery.Quote.Conditions.Classified, IndicatorsClassified: delivery.Quote.Indicators.Classified, Live: delivery.Quote.Live}
		admission, completion := state.AdmitQuote(ctx, input)
		result := EngineDeliveryResult{Admission: admission}
		if admission != engine.AdmissionAdmitted || completion == nil {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case result.TQDisposition = <-completion:
			return result, nil
		}
	case DeliveryNormalizationDrop:
		if delivery.Rejection.Family == LiveFamilyTrade || delivery.Rejection.Family == LiveFamilyQuote {
			input := engine.TQDropInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Rejection.BindingIdentity, TradingDate: delivery.Rejection.TradingDate,
				Family: string(delivery.Rejection.Family), Symbol: delivery.Rejection.Symbol,
				DropReason: string(delivery.Rejection.Reason), Live: delivery.Rejection.Position}
			admission, completion := state.AdmitTQDrop(ctx, input)
			result := EngineDeliveryResult{Admission: admission}
			if admission != engine.AdmissionAdmitted || completion == nil {
				return result, nil
			}
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case result.TQDisposition = <-completion:
				return result, nil
			}
		}
		return EngineDeliveryResult{ConsumerDeferred: true}, nil
	default:
		return EngineDeliveryResult{}, errAttemptState
	}
}
