package massive

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/trace"
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
	symbols := command.Symbols()
	if command.BindingIdentity() == "" || command.ConnectionEpoch() == 0 || command.CommandToken() == 0 || len(symbols) == 0 {
		return ChangeTQCommand{}, errCommand
	}
	return ChangeTQCommand{BindingIdentity: command.BindingIdentity(), ConnectionEpoch: command.ConnectionEpoch(), CommandToken: command.CommandToken(), Action: action, Symbols: symbols, engineCommand: command}, nil
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
	TerminalConnectedDeadline            TerminalReason = "connected_deadline"
	TerminalConnectedFailed              TerminalReason = "connected_failed"
	TerminalConnectedAmbiguous           TerminalReason = "connected_ambiguous"
	TerminalAuthenticationDeadline       TerminalReason = "authentication_deadline"
	TerminalAuthenticationFailed         TerminalReason = "authentication_failed"
	TerminalAuthenticationAmbiguous      TerminalReason = "authentication_ambiguous"
	TerminalAggregateSubscribeDeadline   TerminalReason = "aggregate_subscribe_deadline"
	TerminalAggregateSubscribeFailed     TerminalReason = "aggregate_subscribe_failed"
	TerminalAggregateSubscribeAmbiguous  TerminalReason = "aggregate_subscribe_ambiguous"
	TerminalReadFailed                   TerminalReason = "read_failed"
	TerminalUnsupportedMessage           TerminalReason = "unsupported_message"
	TerminalHeartbeatDeadlineNoProgress  TerminalReason = "heartbeat_deadline_without_inbound_progress"
	TerminalHeartbeatTransportFailure    TerminalReason = "heartbeat_transport_failure"
	TerminalHeartbeatFailureWithProgress TerminalReason = "heartbeat_failure_with_inbound_progress"
	TerminalFrameOversize                TerminalReason = "frame_oversize"
	TerminalFrameSlotCapacity            TerminalReason = "frame_slot_capacity"
	TerminalFrameByteCapacity            TerminalReason = "frame_byte_capacity"
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
	DeliveryDecodedBatch          DeliveryKind = "decoded_batch"
	DeliveryControl               DeliveryKind = "control"
	DeliveryAggregate             DeliveryKind = "aggregate"
	DeliveryTrade                 DeliveryKind = "trade_consumer_deferred"
	DeliveryQuote                 DeliveryKind = "quote_consumer_deferred"
	DeliveryNormalizationDrop     DeliveryKind = "normalization_rejection"
	DeliveryTerminal              DeliveryKind = "terminal"
	DeliveryAggregateIngressFence DeliveryKind = "aggregate_ingress_fence"
	DeliveryLiveCoverageFence     DeliveryKind = "live_coverage_fence"
	DeliveryTQControlError        DeliveryKind = "tq_control_error"
)

type TerminalResult struct {
	BindingIdentity           string
	ConnectionEpoch           uint64
	Position                  engine.LivePosition
	CausalPosition            engine.LivePosition
	PositionApplicable        bool
	ArrayIndexApplicable      bool
	QueueAtCause              LiveQueueAccounting
	AdapterAtCause            AdapterAccounting
	CauseAccountingCapturedAt time.Time
	IncomingFrameBytes        int
	ActiveDeliveryKind        DeliveryKind
	ActiveDeliveryStartedAt   time.Time
	ActiveDeliveryAgeAtCause  time.Duration
	CompletedAt               time.Time
	AggregateAcknowledged     bool
	Source                    TerminalSource
	Reason                    TerminalReason
	CloseCause                CloseCause
	FramesFenced              uint64
	PendingCommandToken       uint64
	PendingCommandOutcome     engine.ConnectionControlOutcome
}

type AdapterDelivery struct {
	Kind                  DeliveryKind
	Position              engine.LivePosition
	Control               engine.ConnectionControlInput
	Aggregate             engine.AggregateInput
	Trade                 NormalizedTrade
	Quote                 NormalizedQuote
	Rejection             LiveRejection
	Terminal              TerminalResult
	AggregateIngressFence AggregateIngressFenceFact
	LiveCoverageFence     LiveCoverageFenceFact
	TQAction              TQCommandAction
	TQSymbols             []string
	TQWriteBoundary       engine.LivePosition
	tqCommand             engine.TQCommand
	TQControlError        engine.TQControlErrorInput
}

type EngineDeliveryResult struct {
	Admission               engine.AdmissionResult
	ControlDisposition      engine.ConnectionControlDisposition
	AggregateDisposition    engine.AggregateDisposition
	HydrationDisposition    engine.HydrationDisposition
	LiveCoverageDisposition engine.LiveCoverageFenceDisposition
	TQDisposition           engine.Disposition
	ConsumerDeferred        bool
	// Terminal is the adapter-owned immutable first terminal cause carried
	// alongside (never inferred from) the engine's lifecycle disposition.
	Terminal *TerminalResult
	// PriorEngine is the engine publication immediately before a terminal
	// control is admitted. Together with the post-completion publication it
	// identifies the first invalid transition without inferring adapter cause.
	PriorEngine           engine.OperationalView
	PriorEngineApplicable bool
	PriorPublication      engine.ReplayPublicationView
}

type ActiveDeliveryDiagnostic struct {
	Kind      DeliveryKind
	StartedAt time.Time
	Age       time.Duration
}

type AdapterAccounting struct {
	ConnectionAttempts                                                       uint64
	AttemptsActive, AttemptsConnected, HighAttemptsActive                    uint64
	AttemptsFailed, AttemptsCanceled                                         uint64
	CommandsStarted, CommandsPendingWrite, CommandsPendingAck                uint64
	CommandsAcknowledged, CommandsFailed, CommandsAmbiguous                  uint64
	CommandsCanceledOrFenced                                                 uint64
	TQCommandsStarted, TQCommandsPendingWrite, TQCommandsWritten             uint64
	TQCommandsFailed                                                         uint64
	TQCommandsCanceledOrFenced                                               uint64
	HandshakeStatuses                                                        uint64
	UnsupportedFamilies                                                      uint64
	HeartbeatInboundProgressAt                                               time.Time
	HeartbeatStartedAt                                                       time.Time
	HeartbeatLatestOutcome                                                   TerminalReason
	HeartbeatCapturedFrameSequence, HeartbeatReadFrameSequence               uint64
	HeartbeatInboundProgressOccurrences, HeartbeatInboundProgressConsecutive uint64
}

type TQNormalizationAccounting struct {
	Classified, Normalized, Rejected                     uint64
	ClassifiedTrades, NormalizedTrades, RejectedTrades   uint64
	ClassifiedQuotes, NormalizedQuotes, RejectedQuotes   uint64
	PressureShed, PressureShedTrades, PressureShedQuotes uint64
}

func (a TQNormalizationAccounting) Reconciles() bool {
	classification := a.Classified == a.Normalized+a.Rejected && a.Classified == a.ClassifiedTrades+a.ClassifiedQuotes &&
		a.Normalized == a.NormalizedTrades+a.NormalizedQuotes && a.Rejected == a.RejectedTrades+a.RejectedQuotes &&
		a.ClassifiedTrades == a.NormalizedTrades+a.RejectedTrades && a.ClassifiedQuotes == a.NormalizedQuotes+a.RejectedQuotes
	return classification && a.PressureShed == a.PressureShedTrades+a.PressureShedQuotes &&
		a.PressureShedTrades <= a.RejectedTrades && a.PressureShedQuotes <= a.RejectedQuotes
}

func (a AdapterAccounting) Reconciles() bool {
	return a.TransportReconciles() && a.TQReconciles()
}

func (a AdapterAccounting) TransportReconciles() bool {
	if a.CommandsStarted < a.TQCommandsStarted || a.CommandsPendingWrite < a.TQCommandsPendingWrite ||
		a.CommandsAcknowledged < a.TQCommandsWritten || a.CommandsFailed < a.TQCommandsFailed ||
		a.CommandsCanceledOrFenced < a.TQCommandsCanceledOrFenced {
		return false
	}
	return a.ConnectionAttempts == a.AttemptsActive+a.AttemptsConnected+a.AttemptsFailed+a.AttemptsCanceled &&
		a.CommandsStarted-a.TQCommandsStarted == a.CommandsPendingWrite-a.TQCommandsPendingWrite+
			a.CommandsPendingAck+a.CommandsAcknowledged-a.TQCommandsWritten+
			a.CommandsFailed-a.TQCommandsFailed+a.CommandsAmbiguous+
			a.CommandsCanceledOrFenced-a.TQCommandsCanceledOrFenced
}

func (a AdapterAccounting) TQReconciles() bool {
	return a.TQCommandsStarted == a.TQCommandsPendingWrite+a.TQCommandsWritten+
		a.TQCommandsFailed+a.TQCommandsCanceledOrFenced
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
	if a.accounting.AttemptsActive > a.accounting.HighAttemptsActive {
		a.accounting.HighAttemptsActive = a.accounting.AttemptsActive
	}
	a.accounting.CommandsStarted++
	a.accounting.CommandsPendingWrite++
	attempt := &LiveAttempt{
		adapter: a, binding: a.binding, epoch: a.nextEpoch, openToken: command.CommandToken,
		durations: command.Durations, endpoint: a.endpoint, credential: a.credential,
		connector: a.connector, queue: newLiveFrameQueue(a.queue, a.binding), cleanupDone: make(chan struct{}), handshakeStart: make(chan handshakeRequest), handshakeOwnerDone: make(chan struct{}), openCommandPending: true,
		classificationClock: time.Now,
	}
	attempt.queue.now = a.now
	attempt.ctx, attempt.cancel = context.WithCancel(ctx)
	attempt.started = true
	a.active = attempt
	at := a.now()
	delivery := controlDelivery(a.binding.Identity(), attempt.epoch, engine.ConnectionAttempt, command.CommandToken, engine.ControlSucceeded, engine.LivePosition{}, at)
	attempt.workers.Add(1)
	go attempt.runHandshakeOwner()
	return attempt, delivery, nil
}

type pendingCommand struct {
	kind          CommandKind
	token         uint64
	writeFailed   bool
	writeDone     chan struct{}
	accounted     bool
	action        TQCommandAction
	symbols       []string
	engineCommand engine.TQCommand
}

type handshakeRequest struct {
	state    *engine.Engine
	complete chan error
}

type terminalCause struct {
	source                                   TerminalSource
	reason                                   TerminalReason
	fenceAfter                               uint64
	ingressIntegrity                         bool
	at                                       time.Time
	closeCause                               CloseCause
	position                                 engine.LivePosition
	positionApplicable, arrayIndexApplicable bool
	queueAtCause                             LiveQueueAccounting
	adapterAtCause                           AdapterAccounting
	accountingCapturedAt                     time.Time
	incomingFrameBytes                       int
	activeDeliveryKind                       DeliveryKind
	activeDeliveryStartedAt                  time.Time
	activeDeliveryAgeAtCause                 time.Duration
	aggregateAcknowledged                    bool
}

type LiveAttempt struct {
	mu         sync.Mutex
	deliveryMu sync.Mutex
	writeMu    sync.Mutex
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
	openCommandPending            bool
	openCommandPendingAck         bool
	connected                     bool
	closeCommand                  *CloseEpochCommand
	terminalDelivery              AdapterDelivery
	terminalReturned              bool
	activeDeliveryKind            DeliveryKind
	activeDeliveryStartedAt       time.Time
	beforeEngineDelivery          func(AdapterDelivery)
	captureToken                  uint64
	handshakeStart                chan handshakeRequest
	handshakeOwnerDone            chan struct{}
	handshakeClaimed              bool
	shedTQ                        atomic.Bool
	tqAccountingMu                sync.Mutex
	tqAccounting                  TQNormalizationAccounting
	classificationClock           func() time.Time
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

func (a *LiveAttempt) ActiveDeliveryDiagnostic() ActiveDeliveryDiagnostic {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := ActiveDeliveryDiagnostic{Kind: a.activeDeliveryKind, StartedAt: a.activeDeliveryStartedAt}
	if !result.StartedAt.IsZero() {
		result.Age = time.Since(result.StartedAt)
		if result.Age < 0 {
			result.Age = 0
		}
	}
	return result
}

func (a *LiveAttempt) accountTQ(family LiveFamily, normalized, pressureShed bool) {
	a.tqAccountingMu.Lock()
	a.tqAccounting.Classified++
	if family == LiveFamilyTrade {
		a.tqAccounting.ClassifiedTrades++
	} else {
		a.tqAccounting.ClassifiedQuotes++
	}
	if normalized {
		a.tqAccounting.Normalized++
		if family == LiveFamilyTrade {
			a.tqAccounting.NormalizedTrades++
		} else {
			a.tqAccounting.NormalizedQuotes++
		}
	} else {
		a.tqAccounting.Rejected++
		if family == LiveFamilyTrade {
			a.tqAccounting.RejectedTrades++
		} else {
			a.tqAccounting.RejectedQuotes++
		}
		if pressureShed {
			a.tqAccounting.PressureShed++
			if family == LiveFamilyTrade {
				a.tqAccounting.PressureShedTrades++
			} else {
				a.tqAccounting.PressureShedQuotes++
			}
		}
	}
	a.tqAccountingMu.Unlock()
}

func (a *LiveAttempt) accountTQCapacityShed(trades, quotes uint64) {
	a.tqAccountingMu.Lock()
	total := trades + quotes
	a.tqAccounting.Classified += total
	a.tqAccounting.Rejected += total
	a.tqAccounting.PressureShed += total
	a.tqAccounting.ClassifiedTrades += trades
	a.tqAccounting.RejectedTrades += trades
	a.tqAccounting.PressureShedTrades += trades
	a.tqAccounting.ClassifiedQuotes += quotes
	a.tqAccounting.RejectedQuotes += quotes
	a.tqAccounting.PressureShedQuotes += quotes
	a.tqAccountingMu.Unlock()
}

func (a *LiveAttempt) HandshakeAndDeliver(ctx context.Context, state *engine.Engine) error {
	if ctx == nil || state == nil {
		return errAttemptState
	}
	a.mu.Lock()
	if a.handshakeClaimed {
		a.mu.Unlock()
		return errAttemptState
	}
	a.handshakeClaimed = true
	a.mu.Unlock()
	stopOperation := context.AfterFunc(ctx, a.cancel)
	defer stopOperation()
	request := handshakeRequest{state: state, complete: make(chan error)}
	select {
	case <-a.ctx.Done():
		return errTransportFailed
	case a.handshakeStart <- request:
	}
	return <-request.complete
}

func (a *LiveAttempt) runHandshakeOwner() {
	defer a.workers.Done()
	defer close(a.handshakeOwnerDone)
	select {
	case request, ok := <-a.handshakeStart:
		if !ok {
			return
		}
		request.complete <- a.performHandshakeAndDeliver(request.state)
	case <-a.ctx.Done():
		a.triggerTerminal(TerminalReader, TerminalContextCanceled, 0, false)
	}
}
func (a *LiveAttempt) performHandshakeAndDeliver(state *engine.Engine) error {
	totalCtx, totalCancel := context.WithTimeout(a.ctx, a.durations.HandshakeTotal)
	defer totalCancel()
	dialCtx, dialCancel := context.WithTimeout(totalCtx, a.durations.Dial)
	connection, err := a.connector.Dial(dialCtx, a.endpoint, int64(a.queue.config.MaxFrameBytes))
	dialCancel()
	if err != nil {
		reason := TerminalDialFailed
		if a.ctx.Err() != nil {
			reason = TerminalContextCanceled
		}
		a.triggerTerminal(TerminalReader, reason, 0, false)
		return errTransportFailed
	}
	a.mu.Lock()
	if a.terminal != nil || a.finished {
		a.mu.Unlock()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), a.durations.Close)
		_ = connection.Close(closeCtx)
		closeCancel()
		return errTransportFailed
	}
	a.connection = connection
	a.startWorkers()
	a.mu.Unlock()

	connectedContext := StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseConnected, CommandKind: CommandConnection, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}
	if err := a.awaitHandshakeStatusAndDeliver(totalCtx, state, connectedContext, engine.ConnectionEstablished); err != nil {
		return err
	}
	a.accountConnected()

	authPayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "auth", Params: a.credential})
	if err := a.write(totalCtx, authPayload); err != nil {
		a.triggerTerminal(TerminalWriter, TerminalAuthenticationFailed, 0, false)
		return errCommandWrite
	}
	authContext := StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseAuthSuccess, CommandKind: CommandAuthentication, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}
	if err := a.awaitHandshakeStatusAndDeliver(totalCtx, state, authContext, engine.AuthenticationResult); err != nil {
		return err
	}

	aggregatePayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "subscribe", Params: "A.*"})
	if err := a.write(totalCtx, aggregatePayload); err != nil {
		failed := controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlFailed, engine.LivePosition{}, a.adapter.now())
		_, _ = DeliverToEngine(context.Background(), state, failed)
		a.accountOpenCommand(engine.ControlFailed, false)
		a.triggerTerminal(TerminalWriter, TerminalAggregateSubscribeFailed, 0, false)
		return errCommandWrite
	}
	a.accountOpenWriteSucceeded()
	written := controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlSucceeded, engine.LivePosition{}, a.adapter.now())
	if result, err := DeliverToEngine(context.Background(), state, written); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		a.triggerTerminal(TerminalWriter, TerminalAggregateSubscribeFailed, 0, false)
		return errTransportFailed
	}
	ackContext := StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}
	if err := a.awaitHandshakeStatusAndDeliver(totalCtx, state, ackContext, engine.AggregateSubscriptionResult); err != nil {
		return err
	}
	a.accountOpenCommand(engine.ControlSucceeded, true)
	a.mu.Lock()
	a.handshaken = true
	a.mu.Unlock()
	return nil
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
			position, applicable := a.queue.lastRawPosition(a.epoch)
			a.triggerTerminalAt(TerminalReader, reason, 0, false, position, applicable, false)
			return
		}
		if kind != socketMessageText && kind != socketMessageBinary {
			_, _, _ = a.queue.beginDecode(a.epoch, kind, a.adapter.now(), len(data))
			a.triggerTerminal(TerminalProtocol, TerminalUnsupportedMessage, 0, false)
			return
		}
		receivedAt := a.adapter.now()
		sequence, reason, accountingAtCause := a.queue.beginDecode(a.epoch, kind, receivedAt, len(data))
		if reason != FrameAdmitted {
			position, applicable := a.queue.lastRawPosition(a.epoch)
			switch reason {
			case FrameRejectedOversize:
				a.triggerTerminalAt(TerminalProtocol, TerminalFrameOversize, 0, true, position, applicable, false)
			case FrameRejectedReceipt:
				a.triggerTerminalAt(TerminalProtocol, TerminalReceiptRegression, 0, true, position, applicable, false)
			default:
				if a.ctx.Err() == nil {
					a.triggerTerminal(TerminalProtocol, TerminalContextCanceled, 0, false)
				}
			}
			return
		}
		shed := a.shedTQ.Load()
		batch := decodeLiveFrame(LiveFrame{Binding: a.binding, ConnectionEpoch: a.epoch, FrameSequence: sequence, ReceivedAt: receivedAt, Data: data}, nil,
			LiveNormalizationOptions{ShedTradesQuotes: shed, ClassificationClock: a.classificationClock, FrameTQBudget: FrameLocalTQBudget})
		data = nil
		// The attempt lock is the active-delivery ownership boundary. Hold it
		// across queue admission and capacity-cause construction so a concurrent
		// engine delivery cannot finish or change between rejection and capture.
		a.mu.Lock()
		beforeQueue := a.queue.snapshot()
		_, reason, accountingAtCause = a.queue.tryEnqueueDecoded(a.epoch, sequence, receivedAt, batch.EncodedBytes, batch)
		if reason == FrameShedTQCapacity {
			afterQueue := a.queue.snapshot()
			a.accountTQCapacityShed(afterQueue.TQCapacityShedTrades-beforeQueue.TQCapacityShedTrades, afterQueue.TQCapacityShedQuotes-beforeQueue.TQCapacityShedQuotes)
			a.mu.Unlock()
			continue
		}
		if reason == FrameRejectedSlotCapacity || reason == FrameRejectedByteCapacity {
			position, applicable := a.queue.lastRawPosition(a.epoch)
			terminalReason := TerminalFrameSlotCapacity
			if reason == FrameRejectedByteCapacity {
				terminalReason = TerminalFrameByteCapacity
			}
			cause := a.reserveCapacityTerminalLocked(terminalReason, batch.EncodedBytes, position, applicable, accountingAtCause)
			a.mu.Unlock()
			a.completeReservedTerminal(cause)
			return
		}
		a.mu.Unlock()
		if reason != FrameAdmitted {
			position, applicable := a.queue.lastRawPosition(a.epoch)
			switch reason {
			case FrameRejectedOversize:
				a.triggerTerminalAt(TerminalProtocol, TerminalFrameOversize, 0, true, position, applicable, false)
			case FrameRejectedReceipt:
				a.triggerTerminalAt(TerminalProtocol, TerminalReceiptRegression, 0, true, position, applicable, false)
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
			startedAt := time.Now().UTC()
			capturedSequence := a.queue.snapshot().FramesRead
			pingCtx, cancel := context.WithTimeout(a.ctx, a.durations.HeartbeatDeadline)
			err := a.ping(pingCtx)
			deadline := errors.Is(pingCtx.Err(), context.DeadlineExceeded)
			cancel()
			if err != nil {
				if a.ctx.Err() != nil {
					a.triggerTerminal(TerminalHeartbeat, TerminalContextCanceled, 0, false)
					return
				}
				readSequence := a.queue.snapshot().FramesRead
				if readSequence > capturedSequence {
					a.recordHeartbeatInboundProgress(time.Now().UTC(), startedAt, capturedSequence, readSequence)
					continue
				}
				reason := TerminalHeartbeatTransportFailure
				if deadline {
					reason = TerminalHeartbeatDeadlineNoProgress
				}
				a.triggerTerminal(TerminalHeartbeat, reason, 0, false)
				return
			}
			a.clearHeartbeatInboundProgressConsecutive()
		}
	}
}

func (a *LiveAttempt) recordHeartbeatInboundProgress(at, startedAt time.Time, captured, read uint64) {
	a.adapter.mu.Lock()
	a.adapter.accounting.HeartbeatInboundProgressAt = at
	a.adapter.accounting.HeartbeatStartedAt = startedAt
	a.adapter.accounting.HeartbeatLatestOutcome = TerminalHeartbeatFailureWithProgress
	a.adapter.accounting.HeartbeatCapturedFrameSequence = captured
	a.adapter.accounting.HeartbeatReadFrameSequence = read
	a.adapter.accounting.HeartbeatInboundProgressOccurrences++
	a.adapter.accounting.HeartbeatInboundProgressConsecutive++
	a.adapter.mu.Unlock()
}

func (a *LiveAttempt) clearHeartbeatInboundProgressConsecutive() {
	a.adapter.mu.Lock()
	a.adapter.accounting.HeartbeatInboundProgressConsecutive = 0
	a.adapter.mu.Unlock()
}

func (a *LiveAttempt) write(parent context.Context, payload []byte) error {
	if parent == nil {
		return errCommandWrite
	}
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	writeCtx, cancel := context.WithTimeout(a.ctx, a.durations.Write)
	stopParent := context.AfterFunc(parent, cancel)
	defer stopParent()
	defer cancel()
	if err := a.connection.Write(writeCtx, socketMessageText, payload); err != nil {
		return errCommandWrite
	}
	return nil
}

func (a *LiveAttempt) ping(ctx context.Context) error {
	if ctx == nil {
		return errCommandWrite
	}
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	return a.connection.Ping(ctx)
}

func (a *LiveAttempt) awaitHandshakeStatusAndDeliver(parent context.Context, state *engine.Engine, status StatusContext, kind engine.ConnectionControlKind) error {
	stepCtx, cancel := context.WithTimeout(parent, a.durations.HandshakeStep)
	defer cancel()
	for {
		frame, ok := a.queue.pop(stepCtx)
		if !ok {
			a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, engine.ControlAmbiguous, true), 0, false)
			return errTransportFailed
		}
		if frame.terminal {
			delivery := a.finishTerminal(frame)
			_, _ = DeliverToEngine(context.Background(), state, delivery)
			return errTransportFailed
		}

		inputs := make([]engine.LiveInput, 0, frame.batch.Len())
		controlInputs := make(map[int]bool)
		found, failed := false, false
		failureOutcome := engine.ControlAmbiguous
		var ingressAmbiguity *engine.LivePosition
		for _, decoded := range frame.batch.results {
			if decoded.Kind == LiveResultStatus {
				decoded.Status.CommandKind, decoded.Status.CommandToken = status.CommandKind, status.CommandToken
				if validStatusContext(&status, a.epoch) && decoded.Status.Phase == status.ExpectedPhase && decoded.Status.ObservedCount <= status.ExpectedCount {
					decoded.Status.Disposition, decoded.Status.Reason = StatusAcknowledged, ""
				}
				a.adapter.mu.Lock()
				a.adapter.accounting.HandshakeStatuses++
				a.adapter.mu.Unlock()
				outcome := engine.ControlSucceeded
				if decoded.Status.Disposition == StatusFailed {
					outcome = engine.ControlFailed
				} else if decoded.Status.Disposition != StatusAcknowledged || found {
					outcome = engine.ControlAmbiguous
				}
				control := controlDelivery(a.binding.Identity(), a.epoch, kind, a.openToken, outcome, decoded.Position, decoded.Status.ReceiptTime)
				input, inputErr := engine.NewLiveConnectionControl(control.Control)
				if inputErr != nil {
					a.queue.complete(frame, false)
					return inputErr
				}
				controlInputs[len(inputs)] = false
				inputs = append(inputs, input)
				found = outcome == engine.ControlSucceeded
				if outcome != engine.ControlSucceeded {
					failed, failureOutcome = true, outcome
				}
				continue
			}

			input, emit, inputErr := a.decodedLiveInput(decoded, frame)
			if inputErr != nil {
				a.queue.complete(frame, false)
				return inputErr
			}
			if emit {
				if decoded.Kind == LiveResultAmbiguous {
					controlInputs[len(inputs)] = true
				}
				inputs = append(inputs, input)
			}
			if decoded.Kind == LiveResultAmbiguous {
				failed, failureOutcome = true, engine.ControlAmbiguous
				position := decoded.Position
				ingressAmbiguity = &position
			}
		}
		if !frame.batch.Accounting().Reconciles() {
			a.queue.complete(frame, false)
			a.triggerTerminalAt(TerminalProtocol, TerminalIngressAmbiguity, frame.sequence, true, engine.LivePosition{ConnectionEpoch: a.epoch, FrameSequence: frame.sequence}, true, false)
			return errTransportFailed
		}
		if len(inputs) > 0 {
			results, err := state.ConsumeLiveBatch(context.Background(), inputs)
			if err != nil || len(results) != len(inputs) {
				a.queue.complete(frame, false)
				if err != nil {
					return err
				}
				return errTransportFailed
			}
			for index, result := range results {
				if result.Admission != engine.AdmissionAdmitted {
					a.queue.complete(frame, false)
					a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, engine.ControlAmbiguous, false), frame.sequence, false)
					return errTransportFailed
				}
				integrity, control := controlInputs[index]
				validControl := !integrity && (result.Control.Code == engine.DispositionConnectionControlApplied || result.Control.Code == engine.DispositionConnectionControlDeferred) ||
					integrity && result.Control.Code == engine.DispositionIngressIntegrity
				if control && !validControl {
					a.queue.complete(frame, false)
					a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, engine.ControlAmbiguous, false), frame.sequence, false)
					return errTransportFailed
				}
			}
		}
		a.queue.complete(frame, false)
		if ingressAmbiguity != nil {
			a.triggerTerminalAt(TerminalProtocol, TerminalIngressAmbiguity, frame.sequence, true, *ingressAmbiguity, true, true)
			return errTransportFailed
		}
		if failed {
			a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, failureOutcome, false), frame.sequence, false)
			return errTransportFailed
		}
		if found {
			return nil
		}
	}
}

func (a *LiveAttempt) decodedLiveInput(result LiveResult, frame queuedLiveFrame) (engine.LiveInput, bool, error) {
	switch result.Kind {
	case LiveResultAggregate:
		input, err := engine.NewLiveAggregate(result.Aggregate)
		return input, true, err
	case LiveResultTrade:
		a.accountTQ(LiveFamilyTrade, true, false)
		input, err := engine.NewLiveTrade(engine.TradeInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: result.Trade.BindingIdentity, TradingDate: result.Trade.TradingDate,
			Symbol: result.Trade.Symbol, TradeID: result.Trade.TradeID, Exchange: result.Trade.Exchange, TRFPresent: result.Trade.TRFPresent, TRFID: result.Trade.TRFID,
			Price: result.Trade.Price, EconomicSize: result.Trade.EconomicSize, EventTime: result.Trade.EventTime, ReceiptTime: result.Trade.ReceiptTime,
			TimestampBasis: string(result.Trade.TimestampBasis), Conditions: result.Trade.Conditions.Slice(), ConditionsClassified: result.Trade.Conditions.Classified,
			IdentityClassified: result.Trade.IdentityClassified, Lifecycle: string(result.Trade.Lifecycle), Live: result.Trade.Live})
		return input, true, err
	case LiveResultQuote:
		a.accountTQ(LiveFamilyQuote, true, false)
		input, err := engine.NewLiveQuote(engine.QuoteInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: result.Quote.BindingIdentity, TradingDate: result.Quote.TradingDate, Symbol: result.Quote.Symbol,
			SIPTime: result.Quote.SIPTime, ReceiptTime: result.Quote.ReceiptTime, BidPrice: result.Quote.BidPrice, AskPrice: result.Quote.AskPrice,
			BidPresent: result.Quote.BidPresent, AskPresent: result.Quote.AskPresent, Conditions: result.Quote.Conditions.Slice(), Indicators: result.Quote.Indicators.Slice(),
			ConditionsClassified: result.Quote.Conditions.Classified, IndicatorsClassified: result.Quote.Indicators.Classified, Live: result.Quote.Live})
		return input, true, err
	case LiveResultRejected:
		if result.Rejection.Family == LiveFamilyTrade || result.Rejection.Family == LiveFamilyQuote {
			a.accountTQ(result.Rejection.Family, false, result.Rejection.Reason == LiveRejectOptionalShed)
			input, err := engine.NewLiveTQDrop(engine.TQDropInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: result.Rejection.BindingIdentity, TradingDate: result.Rejection.TradingDate,
				Family: string(result.Rejection.Family), Symbol: result.Rejection.Symbol, DropReason: string(result.Rejection.Reason), Live: result.Rejection.Position})
			return input, true, err
		}
		if result.Rejection.Family == LiveFamilyUnsupported {
			a.adapter.mu.Lock()
			a.adapter.accounting.UnsupportedFamilies++
			a.adapter.mu.Unlock()
		}
		return engine.LiveInput{}, false, nil
	case LiveResultAmbiguous:
		control := controlDelivery(a.binding.Identity(), a.epoch, engine.IngressIntegrityFailure, 0, engine.ControlAmbiguous, result.Position, frame.receivedAt)
		input, err := engine.NewLiveConnectionControl(control.Control)
		return input, true, err
	default:
		return engine.LiveInput{}, false, nil
	}
}

func handshakeTerminalReason(phase StatusPhase, outcome engine.ConnectionControlOutcome, deadline bool) TerminalReason {
	if deadline {
		switch phase {
		case StatusPhaseConnected:
			return TerminalConnectedDeadline
		case StatusPhaseAuthSuccess:
			return TerminalAuthenticationDeadline
		default:
			return TerminalAggregateSubscribeDeadline
		}
	}
	failed := outcome == engine.ControlFailed
	switch phase {
	case StatusPhaseConnected:
		if failed {
			return TerminalConnectedFailed
		}
		return TerminalConnectedAmbiguous
	case StatusPhaseAuthSuccess:
		if failed {
			return TerminalAuthenticationFailed
		}
		return TerminalAuthenticationAmbiguous
	default:
		if failed {
			return TerminalAggregateSubscribeFailed
		}
		return TerminalAggregateSubscribeAmbiguous
	}
}

func (a *LiveAttempt) ChangeTQ(ctx context.Context, command ChangeTQCommand) (AdapterDelivery, error) {
	a.deliveryMu.Lock()
	defer a.deliveryMu.Unlock()
	return a.changeTQLocked(ctx, command)
}

func (a *LiveAttempt) changeTQLocked(ctx context.Context, command ChangeTQCommand) (AdapterDelivery, error) {
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
	// A command is serialized only through its local socket write. Massive does
	// not document a topic-correlated success status, so no status deadline or
	// expected-status accumulator follows this write.
	a.pending = &pendingCommand{kind: kind, token: command.CommandToken, writeDone: make(chan struct{}), action: command.Action, symbols: append([]string(nil), command.Symbols...), engineCommand: command.engineCommand}
	a.workers.Add(1)
	a.adapter.mu.Lock()
	a.adapter.lastToken = command.CommandToken
	a.adapter.accounting.CommandsStarted++
	a.adapter.accounting.CommandsPendingWrite++
	a.adapter.accounting.TQCommandsStarted++
	a.adapter.accounting.TQCommandsPendingWrite++
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
	pending := a.pending
	boundary := engine.LivePosition{}
	if pending != nil && pending.token == command.CommandToken {
		pending.writeFailed, pending.accounted = err != nil, true
		if err == nil {
			boundary, _ = a.queue.lastRawPosition(a.epoch)
			if boundary == (engine.LivePosition{}) {
				boundary.ConnectionEpoch = a.epoch
			}
		}
		close(pending.writeDone)
		a.pending = nil
	}
	a.mu.Unlock()
	a.adapter.mu.Lock()
	a.adapter.accounting.CommandsPendingWrite--
	a.adapter.accounting.TQCommandsPendingWrite--
	if err != nil {
		outcome = engine.ControlFailed
		a.adapter.accounting.CommandsFailed++
		a.adapter.accounting.TQCommandsFailed++
	} else {
		a.adapter.accounting.CommandsAcknowledged++
		a.adapter.accounting.TQCommandsWritten++
	}
	a.adapter.mu.Unlock()
	// Wake a waiting consumer after the local write is accounted. Later status
	// frames are informational and data frames are fenced by the returned
	// complete-frame boundary.
	a.queue.requestRecheck()
	delivery := controlDelivery(a.binding.Identity(), a.epoch, engineKind, command.CommandToken, outcome, engine.LivePosition{}, a.adapter.now())
	delivery.TQWriteBoundary = boundary
	delivery.TQAction = command.Action
	delivery.TQSymbols = append([]string(nil), command.Symbols...)
	delivery.tqCommand = command.engineCommand
	if err != nil {
		return delivery, errCommandWrite
	}
	return delivery, nil
}

// ChangeTQAndDeliver serializes the local socket write, exact admitted-frame
// boundary capture, and engine completion against raw dequeue. The read worker
// does not take deliveryMu and may continue queueing later complete frames.
func (a *LiveAttempt) ChangeTQAndDeliver(ctx context.Context, state *engine.Engine, command ChangeTQCommand) (EngineDeliveryResult, bool, error) {
	a.deliveryMu.Lock()
	defer a.deliveryMu.Unlock()
	delivery, writeErr := a.changeTQLocked(ctx, command)
	if delivery.Kind == "" {
		return EngineDeliveryResult{}, false, writeErr
	}
	a.mu.Lock()
	beforeEngineDelivery := a.beforeEngineDelivery
	a.mu.Unlock()
	if beforeEngineDelivery != nil {
		beforeEngineDelivery(delivery)
	}
	result, deliveryErr := a.deliverOneThroughHandoff(state, delivery)
	if deliveryErr != nil {
		return result, true, errors.Join(writeErr, deliveryErr)
	}
	return result, true, writeErr
}

func (a *LiveAttempt) validTQCommand(command ChangeTQCommand) bool {
	if command.BindingIdentity != a.binding.Identity() || command.ConnectionEpoch != a.epoch || command.CommandToken == 0 ||
		(command.Action != TQSubscribe && command.Action != TQUnsubscribe) || len(command.Symbols) == 0 || len(command.Symbols) > maximumTQSymbols {
		return false
	}
	if command.engineCommand.CommandToken() != 0 && (command.engineCommand.BindingIdentity() != command.BindingIdentity || command.engineCommand.ConnectionEpoch() != command.ConnectionEpoch ||
		command.engineCommand.CommandToken() != command.CommandToken || !slices.Equal(command.engineCommand.Symbols(), command.Symbols) ||
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
	cause := &terminalCause{source: TerminalEngineClose, reason: TerminalCloseRequested, closeCause: command.Cause, at: a.adapter.now(), aggregateAcknowledged: a.handshaken}
	a.beginTerminalLocked(cause)
	a.mu.Unlock()
	return nil
}

func validCloseCause(cause CloseCause) bool {
	return cause == CloseSessionEnd || cause == CloseControlledStop || cause == CloseSuperseded || cause == CloseIntegrityLoss
}

// nextBatch transfers exactly one complete decoded batch or causal marker.
// deliveryMu prevents another consumer from splitting the handoff.
func (a *LiveAttempt) nextBatch(ctx context.Context) (queuedLiveFrame, bool) {
	for {
		a.mu.Lock()
		if a.finished && a.terminalReturned {
			a.mu.Unlock()
			return queuedLiveFrame{}, false
		}
		if !a.handshaken && a.terminal == nil {
			a.mu.Unlock()
			return queuedLiveFrame{}, false
		}
		terminal := a.terminal
		a.mu.Unlock()
		frame, ok, recheck := a.queue.popOrRecheck(ctx)
		if recheck {
			continue
		}
		if !ok {
			if terminal == nil {
				return queuedLiveFrame{}, false
			}
			select {
			case <-ctx.Done():
				return queuedLiveFrame{}, false
			case <-a.cleanupDone:
			}
			frame, ok = a.queue.pop(ctx)
			return frame, ok
		}
		if terminal != nil && terminal.fenceAfter > 0 && frame.kind == queuedLiveDecodedBatch && frame.sequence > terminal.fenceAfter {
			a.queue.complete(frame, true)
			continue
		}
		return frame, true
	}
}

// DeliverNextToEngine linearizes dequeue, engine admission, and completion so
// concurrent callers cannot admit a later raw item ahead of an earlier fence.
func (a *LiveAttempt) DeliverNextToEngine(ctx context.Context, state *engine.Engine) (EngineDeliveryResult, bool, error) {
	a.deliveryMu.Lock()
	defer a.deliveryMu.Unlock()
	var frame queuedLiveFrame
	var ok bool
	trace.WithRegion(ctx, "websocket_decoded_batch_dequeue", func() { frame, ok = a.nextBatch(ctx) })
	if !ok {
		return EngineDeliveryResult{}, false, nil
	}
	if frame.terminal {
		delivery := a.finishTerminal(frame)
		result, err := a.deliverOneThroughHandoff(state, delivery)
		return result, true, err
	}
	deliveryKind := DeliveryDecodedBatch
	if frame.kind == queuedLiveTQCapacity {
		deliveryKind = DeliveryTQControlError
	} else if frame.kind == queuedLiveIngressFence {
		deliveryKind = DeliveryAggregateIngressFence
	} else if frame.kind == queuedLiveCoverageFence {
		deliveryKind = DeliveryLiveCoverageFence
	}
	a.mu.Lock()
	a.activeDeliveryKind, a.activeDeliveryStartedAt = deliveryKind, time.Now()
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.activeDeliveryKind, a.activeDeliveryStartedAt = "", time.Time{}
		a.mu.Unlock()
	}()
	result, err := a.consumeQueuedEntry(state, frame)
	a.queue.complete(frame, false)
	return result, true, err
}

func (a *LiveAttempt) consumeQueuedEntry(state *engine.Engine, frame queuedLiveFrame) (EngineDeliveryResult, error) {
	inputs := make([]engine.LiveInput, 0, max(1, frame.batch.Len()))
	kinds := make([]DeliveryKind, 0, cap(inputs))
	appendInput := func(input engine.LiveInput, kind DeliveryKind) {
		inputs, kinds = append(inputs, input), append(kinds, kind)
	}
	if frame.kind == queuedLiveTQCapacity {
		input := engine.TQControlErrorInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: a.binding.Identity(), ConnectionEpoch: a.epoch,
			Position: frame.tqCapacityPosition, ReceiptTime: frame.receivedAt.UTC()}
		live, err := engine.NewLiveTQControlError(input)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		appendInput(live, DeliveryTQControlError)
	} else if frame.kind == queuedLiveIngressFence {
		frame.ingressFence.deliveryStartedAt = a.activeDeliveryStartedAt
		a.mu.Lock()
		beforeEngineDelivery := a.beforeEngineDelivery
		a.mu.Unlock()
		if beforeEngineDelivery != nil {
			beforeEngineDelivery(AdapterDelivery{Kind: DeliveryAggregateIngressFence, AggregateIngressFence: frame.ingressFence})
		}
		input, err := EngineAggregateIngressFence(frame.ingressFence)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		live, err := engine.NewLiveAggregateIngressFence(input)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		appendInput(live, DeliveryAggregateIngressFence)
	} else if frame.kind == queuedLiveCoverageFence {
		a.mu.Lock()
		beforeEngineDelivery := a.beforeEngineDelivery
		a.mu.Unlock()
		if beforeEngineDelivery != nil {
			beforeEngineDelivery(AdapterDelivery{Kind: DeliveryLiveCoverageFence, LiveCoverageFence: frame.liveCoverageFence})
		}
		input, err := EngineLiveCoverageFence(frame.liveCoverageFence)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		live, err := engine.NewLiveCoverageFence(input)
		if err != nil {
			return EngineDeliveryResult{}, err
		}
		appendInput(live, DeliveryLiveCoverageFence)
	} else {
		for _, decoded := range frame.batch.results {
			delivery, emit := a.mapResult(decoded, frame)
			if !emit {
				continue
			}
			if delivery.Kind == DeliveryNormalizationDrop && delivery.Rejection.Family != LiveFamilyTrade && delivery.Rejection.Family != LiveFamilyQuote {
				continue
			}
			a.mu.Lock()
			beforeEngineDelivery := a.beforeEngineDelivery
			a.mu.Unlock()
			if beforeEngineDelivery != nil {
				beforeEngineDelivery(delivery)
			}
			live, err := engineLiveInput(delivery)
			if err != nil {
				return EngineDeliveryResult{}, err
			}
			appendInput(live, delivery.Kind)
		}
		if !frame.batch.Accounting().Reconciles() {
			return EngineDeliveryResult{}, errTransportFailed
		}
	}
	if len(inputs) == 0 {
		return EngineDeliveryResult{}, nil
	}
	var results []engine.LiveResult
	var err error
	trace.WithRegion(context.Background(), "engine_live_batch_handoff", func() { results, err = state.ConsumeLiveBatch(context.Background(), inputs) })
	if err != nil {
		return EngineDeliveryResult{}, err
	}
	combined := EngineDeliveryResult{}
	for index, item := range results {
		combined.Admission = item.Admission
		switch kinds[index] {
		case DeliveryAggregate:
			combined.AggregateDisposition = item.Aggregate
		case DeliveryControl:
			combined.ControlDisposition = item.Control
		case DeliveryAggregateIngressFence:
			combined.HydrationDisposition = item.Hydration
		case DeliveryLiveCoverageFence:
			combined.LiveCoverageDisposition = item.LiveCoverage
		default:
			combined.TQDisposition = item.Disposition
		}
	}
	return combined, nil
}

func (a *LiveAttempt) deliverOneThroughHandoff(state *engine.Engine, delivery AdapterDelivery) (EngineDeliveryResult, error) {
	return DeliverToEngine(context.Background(), state, delivery)
}

func engineLiveInput(delivery AdapterDelivery) (engine.LiveInput, error) {
	switch delivery.Kind {
	case DeliveryAggregate:
		return engine.NewLiveAggregate(delivery.Aggregate)
	case DeliveryTrade:
		return engine.NewLiveTrade(engine.TradeInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Trade.BindingIdentity, TradingDate: delivery.Trade.TradingDate,
			Symbol: delivery.Trade.Symbol, TradeID: delivery.Trade.TradeID, Exchange: delivery.Trade.Exchange, TRFPresent: delivery.Trade.TRFPresent, TRFID: delivery.Trade.TRFID,
			Price: delivery.Trade.Price, EconomicSize: delivery.Trade.EconomicSize, EventTime: delivery.Trade.EventTime, ReceiptTime: delivery.Trade.ReceiptTime,
			TimestampBasis: string(delivery.Trade.TimestampBasis), Conditions: delivery.Trade.Conditions.Slice(), ConditionsClassified: delivery.Trade.Conditions.Classified,
			IdentityClassified: delivery.Trade.IdentityClassified, Lifecycle: string(delivery.Trade.Lifecycle), Live: delivery.Trade.Live})
	case DeliveryQuote:
		return engine.NewLiveQuote(engine.QuoteInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Quote.BindingIdentity, TradingDate: delivery.Quote.TradingDate, Symbol: delivery.Quote.Symbol,
			SIPTime: delivery.Quote.SIPTime, ReceiptTime: delivery.Quote.ReceiptTime, BidPrice: delivery.Quote.BidPrice, AskPrice: delivery.Quote.AskPrice,
			BidPresent: delivery.Quote.BidPresent, AskPresent: delivery.Quote.AskPresent, Conditions: delivery.Quote.Conditions.Slice(), Indicators: delivery.Quote.Indicators.Slice(),
			ConditionsClassified: delivery.Quote.Conditions.Classified, IndicatorsClassified: delivery.Quote.Indicators.Classified, Live: delivery.Quote.Live})
	case DeliveryNormalizationDrop:
		return engine.NewLiveTQDrop(engine.TQDropInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: delivery.Rejection.BindingIdentity, TradingDate: delivery.Rejection.TradingDate,
			Family: string(delivery.Rejection.Family), Symbol: delivery.Rejection.Symbol, DropReason: string(delivery.Rejection.Reason), Live: delivery.Rejection.Position})
	case DeliveryTQControlError:
		return engine.NewLiveTQControlError(delivery.TQControlError)
	case DeliveryAggregateIngressFence:
		input, err := EngineAggregateIngressFence(delivery.AggregateIngressFence)
		if err != nil {
			return engine.LiveInput{}, err
		}
		return engine.NewLiveAggregateIngressFence(input)
	case DeliveryLiveCoverageFence:
		input, err := EngineLiveCoverageFence(delivery.LiveCoverageFence)
		if err != nil {
			return engine.LiveInput{}, err
		}
		return engine.NewLiveCoverageFence(input)
	case DeliveryControl, DeliveryTerminal:
		if delivery.Control.Kind == engine.TradeQuoteCommandWriteResult && len(delivery.TQSymbols) > 0 {
			input, err := engine.NewTQCommandResultInput(delivery.tqCommand, delivery.TQWriteBoundary, delivery.Control.ReceiptTime, delivery.Control.Outcome)
			if err != nil {
				return engine.LiveInput{}, err
			}
			return engine.NewLiveTQCommandResult(input)
		}
		return engine.NewLiveConnectionControl(delivery.Control)
	default:
		return engine.LiveInput{}, errAttemptState
	}
}

// AcknowledgeTerminalObservation closes the diagnostic interval only after
// operations has latched the immutable adapter terminal. It changes no queue,
// engine, or lifecycle state.
func (a *LiveAttempt) AcknowledgeTerminalObservation() {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.activeDeliveryKind, a.activeDeliveryStartedAt = "", time.Time{}
	a.mu.Unlock()
}

func (a *LiveAttempt) mapResult(result LiveResult, frame queuedLiveFrame) (AdapterDelivery, bool) {
	switch result.Kind {
	case LiveResultAggregate:
		return AdapterDelivery{Kind: DeliveryAggregate, Position: result.Position, Aggregate: result.Aggregate}, true
	case LiveResultTrade:
		a.accountTQ(LiveFamilyTrade, true, false)
		return AdapterDelivery{Kind: DeliveryTrade, Position: result.Position, Trade: result.Trade}, true
	case LiveResultQuote:
		a.accountTQ(LiveFamilyQuote, true, false)
		return AdapterDelivery{Kind: DeliveryQuote, Position: result.Position, Quote: result.Quote}, true
	case LiveResultRejected:
		if result.Rejection.Family == LiveFamilyTrade || result.Rejection.Family == LiveFamilyQuote {
			a.accountTQ(result.Rejection.Family, false, result.Rejection.Reason == LiveRejectOptionalShed)
		} else if result.Rejection.Family == LiveFamilyUnsupported {
			a.adapter.mu.Lock()
			a.adapter.accounting.UnsupportedFamilies++
			a.adapter.mu.Unlock()
		}
		return AdapterDelivery{Kind: DeliveryNormalizationDrop, Position: result.Position, Rejection: result.Rejection}, true
	case LiveResultAmbiguous:
		delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.IngressIntegrityFailure, 0, engine.ControlAmbiguous, result.Position, frame.receivedAt)
		if terminal := a.triggerTerminalAt(TerminalProtocol, TerminalIngressAmbiguity, frame.sequence, true, result.Position, true, true); terminal != nil {
			delivery.Terminal = *terminal
		}
		return delivery, true
	case LiveResultStatus:
		if result.Status.Phase == StatusPhaseAuthFailed {
			delivery := controlDelivery(a.binding.Identity(), a.epoch, engine.AuthenticationResult, 0, engine.ControlFailed, result.Position, result.Status.ReceiptTime)
			a.triggerTerminalAt(TerminalProtocol, TerminalStatusAmbiguous, frame.sequence, false, result.Position, true, true)
			return delivery, true
		}
		a.adapter.mu.Lock()
		switch result.Status.Phase {
		case StatusPhaseError:
			// The bounded error fact below closes T/Q additions for this epoch.
		case StatusPhaseSuccess:
			// Generic and late success statuses are informational only.
		default:
			a.adapter.accounting.UnsupportedFamilies++
		}
		a.adapter.mu.Unlock()
		if result.Status.Phase == StatusPhaseError {
			return a.tqControlErrorDelivery(result.Position, result.Status.ReceiptTime), true
		}
		return AdapterDelivery{}, false
	default:
		return AdapterDelivery{}, false
	}
}

func (a *LiveAttempt) tqControlErrorDelivery(position engine.LivePosition, at time.Time) AdapterDelivery {
	input := engine.TQControlErrorInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: a.binding.Identity(), ConnectionEpoch: a.epoch,
		Position: position, ReceiptTime: at.UTC()}
	return AdapterDelivery{Kind: DeliveryTQControlError, Position: position, TQControlError: input}
}

func (a *LiveAttempt) triggerTerminal(source TerminalSource, reason TerminalReason, fenceAfter uint64, ingress bool) {
	a.triggerTerminalAt(source, reason, fenceAfter, ingress, engine.LivePosition{}, false, false)
}

func (a *LiveAttempt) triggerTerminalAt(source TerminalSource, reason TerminalReason, fenceAfter uint64, ingress bool, position engine.LivePosition, positionApplicable, arrayIndexApplicable bool) *TerminalResult {
	return a.triggerTerminalAtWithCapacity(source, reason, fenceAfter, ingress, position, positionApplicable, arrayIndexApplicable, 0, nil)
}

// reserveCapacityTerminalLocked requires a.mu. The read worker uses it while
// still holding the same lock that covered queue rejection, making the active
// engine delivery fields exact at the capacity-cause linearization point.
func (a *LiveAttempt) reserveCapacityTerminalLocked(reason TerminalReason, incomingFrameBytes int, position engine.LivePosition, positionApplicable bool, queueAtCause LiveQueueAccounting) *terminalCause {
	return a.reserveTerminalCauseLocked(TerminalProtocol, reason, a.queue.lastAdmittedSequence(), true, position, positionApplicable, false, incomingFrameBytes, &queueAtCause)
}

func (a *LiveAttempt) triggerTerminalAtWithCapacity(source TerminalSource, reason TerminalReason, fenceAfter uint64, ingress bool, position engine.LivePosition, positionApplicable, arrayIndexApplicable bool, incomingFrameBytes int, queueAtCause *LiveQueueAccounting) *TerminalResult {
	a.mu.Lock()
	cause := a.reserveTerminalCauseLocked(source, reason, fenceAfter, ingress, position, positionApplicable, arrayIndexApplicable, incomingFrameBytes, queueAtCause)
	a.mu.Unlock()
	return a.completeReservedTerminal(cause)
}

func (a *LiveAttempt) reserveTerminalCauseLocked(source TerminalSource, reason TerminalReason, fenceAfter uint64, ingress bool, position engine.LivePosition, positionApplicable, arrayIndexApplicable bool, incomingFrameBytes int, queueAtCause *LiveQueueAccounting) *terminalCause {
	if a.terminal != nil || a.finished {
		return nil
	}
	if fenceAfter == 0 {
		fenceAfter = a.queue.lastAdmittedSequence()
	}
	cause := &terminalCause{source: source, reason: reason, fenceAfter: fenceAfter, ingressIntegrity: ingress, at: a.adapter.now(),
		position: position, positionApplicable: positionApplicable, arrayIndexApplicable: arrayIndexApplicable,
		incomingFrameBytes: incomingFrameBytes, activeDeliveryKind: a.activeDeliveryKind, activeDeliveryStartedAt: a.activeDeliveryStartedAt}
	cause.aggregateAcknowledged = a.handshaken
	if !cause.activeDeliveryStartedAt.IsZero() {
		cause.activeDeliveryAgeAtCause = time.Since(cause.activeDeliveryStartedAt)
		if cause.activeDeliveryAgeAtCause < 0 {
			cause.activeDeliveryAgeAtCause = 0
		}
	}
	// Each accounting family is internally coherent under its owner's lock.
	// Capture both before cancellation and cleanup mutate either family.
	if queueAtCause != nil {
		cause.queueAtCause = *queueAtCause
	} else {
		cause.queueAtCause = a.queue.snapshot()
	}
	// Reserve the immutable first cause before releasing the attempt owner.
	// Cleanup does not start until the adapter owner's coherent accounting is
	// captured, so neither accounting family is sampled after cleanup mutation.
	a.terminal = cause
	return cause
}

func (a *LiveAttempt) completeReservedTerminal(cause *terminalCause) *TerminalResult {
	if cause == nil {
		return nil
	}
	cause.adapterAtCause = a.adapter.Accounting()
	cause.accountingCapturedAt = a.adapter.now()
	a.mu.Lock()
	a.beginTerminalLocked(cause)
	a.mu.Unlock()
	terminal := terminalResultAtCause(a, cause)
	return &terminal
}

func terminalResultAtCause(a *LiveAttempt, cause *terminalCause) TerminalResult {
	return TerminalResult{BindingIdentity: a.binding.Identity(), ConnectionEpoch: a.epoch,
		CausalPosition: cause.position, PositionApplicable: cause.positionApplicable, ArrayIndexApplicable: cause.arrayIndexApplicable,
		QueueAtCause: cause.queueAtCause, AdapterAtCause: cause.adapterAtCause, CauseAccountingCapturedAt: cause.accountingCapturedAt,
		IncomingFrameBytes: cause.incomingFrameBytes, ActiveDeliveryKind: cause.activeDeliveryKind,
		ActiveDeliveryStartedAt: cause.activeDeliveryStartedAt, ActiveDeliveryAgeAtCause: cause.activeDeliveryAgeAtCause,
		CompletedAt: cause.at, AggregateAcknowledged: cause.aggregateAcknowledged, Source: cause.source, Reason: cause.reason, CloseCause: cause.closeCause}
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
	markerCtx, markerCancel := context.WithTimeout(context.Background(), a.durations.Close)
	marker, admitted := a.queue.enqueueTerminal(markerCtx, a.epoch, cause.at, cause.fenceAfter)
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
			if a.adapter.accounting.TQCommandsPendingWrite > 0 {
				a.adapter.accounting.TQCommandsPendingWrite--
			}
		}
		a.adapter.accounting.CommandsCanceledOrFenced++
		a.adapter.accounting.TQCommandsCanceledOrFenced++
	}
	a.adapter.mu.Unlock()
	accounting := a.queue.snapshot()
	position := engine.LivePosition{ConnectionEpoch: a.epoch, FrameSequence: marker.sequence}
	terminal := terminalResultAtCause(a, cause)
	terminal.Position, terminal.FramesFenced = position, accounting.FramesFenced
	if pending != nil {
		terminal.PendingCommandToken = pending.token
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

// TerminalResult returns the attempt's bounded, redacted terminal fact only
// after cleanup has reconciled the queue, workers, socket, and adapter owner.
func (a *LiveAttempt) TerminalResult() (TerminalResult, bool) {
	if a == nil {
		return TerminalResult{}, false
	}
	select {
	case <-a.cleanupDone:
	default:
		return TerminalResult{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminalDelivery.Terminal.Reason == "" {
		return TerminalResult{}, false
	}
	return a.terminalDelivery.Terminal, true
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
	if delivery.Kind == DeliveryNormalizationDrop && delivery.Rejection.Family != LiveFamilyTrade && delivery.Rejection.Family != LiveFamilyQuote {
		return EngineDeliveryResult{ConsumerDeferred: true}, nil
	}

	prior := state.ObserveSnapshot()
	input, err := engineLiveInput(delivery)
	if err != nil {
		return EngineDeliveryResult{}, err
	}
	results, err := state.ConsumeLiveBatch(ctx, []engine.LiveInput{input})
	if err != nil {
		return EngineDeliveryResult{}, err
	}
	if len(results) != 1 {
		return EngineDeliveryResult{}, errAttemptState
	}
	item := results[0]
	result := EngineDeliveryResult{
		Admission:               item.Admission,
		ControlDisposition:      item.Control,
		AggregateDisposition:    item.Aggregate,
		HydrationDisposition:    item.Hydration,
		LiveCoverageDisposition: item.LiveCoverage,
		TQDisposition:           item.Disposition,
	}
	if delivery.Terminal.Reason != "" {
		terminal := delivery.Terminal
		result.Terminal = &terminal
		result.PriorEngine = prior.Operational
		result.PriorEngineApplicable = true
		result.PriorPublication = prior.Publication
	}
	return result, nil
}
