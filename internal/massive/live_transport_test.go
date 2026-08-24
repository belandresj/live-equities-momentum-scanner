package massive

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type fakeSocketRead struct {
	kind socketMessageType
	data string
	err  error
}

type fakeLiveSocket struct {
	reads chan fakeSocketRead

	mu           sync.Mutex
	writes       [][]byte
	writeStarted chan struct{}
	writeRelease chan struct{}
	blockWrite   bool
	writeError   error
	pingError    error
	pingStarted  chan struct{}
	pingRelease  chan struct{}
	blockPing    bool
	closeBlock   bool
	closed       bool
}

func newFakeLiveSocket() *fakeLiveSocket {
	return &fakeLiveSocket{reads: make(chan fakeSocketRead, 32)}
}

func (s *fakeLiveSocket) Read(ctx context.Context) (socketMessageType, []byte, error) {
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case read := <-s.reads:
		return read.kind, []byte(read.data), read.err
	}
}

func (s *fakeLiveSocket) Write(ctx context.Context, _ socketMessageType, data []byte) error {
	s.mu.Lock()
	s.writes = append(s.writes, append([]byte(nil), data...))
	block, started, release := s.blockWrite, s.writeStarted, s.writeRelease
	s.mu.Unlock()
	if block {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
	}
	s.mu.Lock()
	err := s.writeError
	s.mu.Unlock()
	return err
}

func (s *fakeLiveSocket) Ping(ctx context.Context) error {
	s.mu.Lock()
	err, block, started, release := s.pingError, s.blockPing, s.pingStarted, s.pingRelease
	s.mu.Unlock()
	if block {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
	}
	return err
}

func (s *fakeLiveSocket) Close(ctx context.Context) error {
	s.mu.Lock()
	block := s.closeBlock
	s.closed = true
	s.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (s *fakeLiveSocket) send(kind socketMessageType, data string) {
	s.reads <- fakeSocketRead{kind: kind, data: data}
}

func (s *fakeLiveSocket) failRead() {
	s.reads <- fakeSocketRead{err: errors.New("private fake failure")}
}

func (s *fakeLiveSocket) written() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.writes))
	for index := range s.writes {
		result[index] = string(s.writes[index])
	}
	return result
}

func (s *fakeLiveSocket) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

type fakeLiveConnector struct {
	mu                sync.Mutex
	sockets           []*fakeLiveSocket
	dials             int
	fail              bool
	dialStarted       chan struct{}
	dialRelease       chan struct{}
	returnAfterCancel bool
}

func (c *fakeLiveConnector) Dial(ctx context.Context, _ string, _ int64) (liveSocket, error) {
	c.mu.Lock()
	c.dials++
	started, release := c.dialStarted, c.dialRelease
	c.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		if c.returnAfterCancel {
			<-release
		} else {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail || len(c.sockets) == 0 {
		return nil, errors.New("secret-bearing dial error")
	}
	socket := c.sockets[0]
	c.sockets = c.sockets[1:]
	return socket, nil
}

func testLiveAdapter(t *testing.T, socket *fakeLiveSocket, symbols []string) (*LiveAdapter, OpenAggregateEpoch) {
	t.Helper()
	binding := component4TestBinding(t, symbols)
	adapter, err := NewLiveAdapter(binding, LiveAdapterConfig{
		Endpoint: "wss://offline.invalid/stocks", Credential: "CREDENTIAL-MUST-NOT-ESCAPE",
		Queue: LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384},
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{socket}}
	durations := OperationalDurations{Dial: time.Second, HandshakeStep: time.Second, HandshakeTotal: 4 * time.Second, HeartbeatInterval: time.Hour, HeartbeatDeadline: time.Second, Write: time.Second, Close: 20 * time.Millisecond}
	return adapter, OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1, Durations: durations}
}

func enqueueHandshake(socket *fakeLiveSocket) {
	socket.send(socketMessageText, `[ {"ev":"status","status":"connected"} ]`)
	socket.send(socketMessageBinary, `[ {"ev":"status","status":"auth_success"} ]`)
	socket.send(socketMessageText, `[ {"ev":"status","status":"success"} ]`)
}

func startHandshake(t *testing.T, adapter *LiveAdapter, open OpenAggregateEpoch) (*LiveAttempt, AdapterDelivery, []AdapterDelivery) {
	t.Helper()
	attempt, started, err := adapter.Start(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	handshake, err := attempt.Handshake(context.Background())
	if err != nil {
		t.Fatalf("handshake: %v deliveries=%+v", err, handshake)
	}
	return attempt, started, handshake
}

func TestPC5TransportOneAttemptHandshakeHeartbeatAndContainment(t *testing.T) {
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapter(t, socket, []string{"AAA"})
	attempt, started, handshake := startHandshake(t, adapter, open)
	if started.Control.Kind != engine.ConnectionAttempt || len(handshake) != 4 {
		t.Fatalf("ordered handshake facts = %+v %+v", started, handshake)
	}
	wantKinds := []engine.ConnectionControlKind{engine.ConnectionEstablished, engine.AuthenticationResult, engine.AggregateCommandWriteResult, engine.AggregateSubscriptionResult}
	for index, delivery := range handshake {
		if delivery.Control.Kind != wantKinds[index] || delivery.Control.Outcome != engine.ControlSucceeded {
			t.Fatalf("handshake %d = %+v", index, delivery)
		}
	}
	writes := socket.written()
	if len(writes) != 2 || writes[1] != `{"action":"subscribe","params":"A.*"}` {
		t.Fatalf("handshake writes = %q", writes)
	}
	binaryStart := adapter.binding.SessionStart().Add(10 * time.Second)
	socket.send(socketMessageBinary, "["+aggregateLiveJSON("AAA", binaryStart, `"v":1,"z":1`)+"]")
	if binary, ok := attempt.nextForProof(context.Background()); !ok || binary.Kind != DeliveryAggregate {
		t.Fatalf("ordinary binary read = %+v %v", binary, ok)
	}
	socket.failRead()
	terminal, ok := attempt.nextForProof(context.Background())
	if !ok || terminal.Kind != DeliveryTerminal || terminal.Terminal.Source != TerminalReader || terminal.Terminal.Reason != TerminalReadFailed {
		t.Fatalf("terminal = %+v %v", terminal, ok)
	}
	if !attempt.QueueAccounting().Reconciles() || !adapter.Accounting().Reconciles() || !socket.isClosed() {
		t.Fatalf("terminal accounting/close = queue=%+v adapter=%+v closed=%v", attempt.QueueAccounting(), adapter.Accounting(), socket.isClosed())
	}
	observable := fmt.Sprintf("%+v %+v %v %+v %#v %+v %#v", started, handshake, terminal, adapter, adapter, attempt, attempt)
	if strings.Contains(observable, "CREDENTIAL-MUST-NOT-ESCAPE") || strings.Contains(observable, "secret-bearing") {
		t.Fatalf("credential/provider prose escaped: %s", observable)
	}

	t.Run("auth failure and dial failure are redacted and terminal", func(t *testing.T) {
		failedSocket := newFakeLiveSocket()
		failedSocket.send(socketMessageText, `[{"ev":"status","status":"connected"}]`)
		failedSocket.send(socketMessageText, `[{"ev":"status","status":"auth_failed","message":"CREDENTIAL-MUST-NOT-ESCAPE wss://private"}]`)
		failedAdapter, command := testLiveAdapter(t, failedSocket, []string{"AAA"})
		failedAttempt, _, err := failedAdapter.Start(context.Background(), command)
		if err != nil {
			t.Fatal(err)
		}
		facts, err := failedAttempt.Handshake(context.Background())
		if err == nil || strings.Contains(fmt.Sprintf("%v %+v", err, facts), "CREDENTIAL-MUST-NOT-ESCAPE") {
			t.Fatalf("auth failure = %v %+v", err, facts)
		}
		terminal, ok := failedAttempt.nextForProof(context.Background())
		if !ok || terminal.Terminal.Reason != TerminalAuthenticationFailed {
			t.Fatalf("auth terminal = %+v", terminal)
		}

		dialAdapter, dialCommand := testLiveAdapter(t, newFakeLiveSocket(), []string{"AAA"})
		dialAdapter.connector = &fakeLiveConnector{fail: true}
		dialAttempt, _, _ := dialAdapter.Start(context.Background(), dialCommand)
		if _, err := dialAttempt.Handshake(context.Background()); !errors.Is(err, errTransportFailed) {
			t.Fatalf("dial failure = %v", err)
		}
		terminal, ok = dialAttempt.nextForProof(context.Background())
		if !ok || terminal.Terminal.Reason != TerminalDialFailed {
			t.Fatalf("dial terminal = %+v", terminal)
		}
	})

	t.Run("heartbeat and bounded hung close join the epoch", func(t *testing.T) {
		heartbeatSocket := newFakeLiveSocket()
		enqueueHandshake(heartbeatSocket)
		heartbeatAdapter, command := testLiveAdapter(t, heartbeatSocket, []string{"AAA"})
		command.Durations.HeartbeatInterval = 2 * time.Millisecond
		command.Durations.HeartbeatDeadline = 2 * time.Millisecond
		command.Durations.Close = 5 * time.Millisecond
		heartbeatSocket.mu.Lock()
		heartbeatSocket.pingError = errors.New("combined ping failure")
		heartbeatSocket.closeBlock = true
		heartbeatSocket.mu.Unlock()
		heartbeatAttempt, _, _ := startHandshake(t, heartbeatAdapter, command)
		startedAt := time.Now()
		terminal, ok := heartbeatAttempt.nextForProof(context.Background())
		elapsed := time.Since(startedAt)
		if !ok || terminal.Terminal.Source != TerminalHeartbeat || terminal.Terminal.Reason != TerminalHeartbeatTransportFailure || elapsed > 200*time.Millisecond {
			t.Fatalf("heartbeat/close = %+v elapsed=%s", terminal, elapsed)
		}
	})

	t.Run("close racing dial handoff cannot leak the returned socket", func(t *testing.T) {
		racingSocket := newFakeLiveSocket()
		racingAdapter, command := testLiveAdapter(t, racingSocket, []string{"AAA"})
		connector := &fakeLiveConnector{
			sockets:           []*fakeLiveSocket{racingSocket},
			dialStarted:       make(chan struct{}, 1),
			dialRelease:       make(chan struct{}),
			returnAfterCancel: true,
		}
		racingAdapter.connector = connector
		racingAttempt, _, err := racingAdapter.Start(context.Background(), command)
		if err != nil {
			t.Fatal(err)
		}
		handshakeResult := make(chan error, 1)
		go func() {
			_, handshakeErr := racingAttempt.Handshake(context.Background())
			handshakeResult <- handshakeErr
		}()
		<-connector.dialStarted
		err = racingAttempt.Close(CloseEpochCommand{
			BindingIdentity: command.BindingIdentity,
			ConnectionEpoch: racingAttempt.Epoch(),
			CommandToken:    2,
			Cause:           CloseControlledStop,
		})
		if err != nil {
			t.Fatalf("close during dial = %v", err)
		}
		waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		if err := racingAttempt.Wait(waitCtx); err != nil {
			t.Fatalf("terminal cleanup did not finish before dial release: %v", err)
		}
		close(connector.dialRelease)
		if err := <-handshakeResult; !errors.Is(err, errTransportFailed) {
			t.Fatalf("handshake after terminal = %v", err)
		}
		if !racingSocket.isClosed() {
			t.Fatal("socket returned after terminal cleanup was not closed")
		}
		terminal, ok := racingAttempt.nextForProof(context.Background())
		if !ok || terminal.Terminal.Source != TerminalEngineClose || terminal.Terminal.CloseCause != CloseControlledStop || !racingAdapter.Accounting().Reconciles() {
			t.Fatalf("racing terminal/accounting = %+v %v %+v", terminal, ok, racingAdapter.Accounting())
		}
	})

	t.Run("unsupported message and handshake deadline fail closed", func(t *testing.T) {
		unsupported := newFakeLiveSocket()
		unsupported.send(socketMessageType(99), "ignored")
		unsupportedAdapter, command := testLiveAdapter(t, unsupported, []string{"AAA"})
		unsupportedAttempt, _, _ := unsupportedAdapter.Start(context.Background(), command)
		facts, err := unsupportedAttempt.Handshake(context.Background())
		if err == nil || len(facts) > 1 {
			t.Fatalf("unsupported = %v %+v", err, facts)
		}
		var terminal AdapterDelivery
		var ok bool
		if len(facts) == 1 {
			terminal, ok = facts[0], true
		} else {
			terminal, ok = unsupportedAttempt.nextForProof(context.Background())
		}
		if !ok || terminal.Terminal.Reason != TerminalUnsupportedMessage {
			t.Fatalf("unsupported terminal = %+v", terminal)
		}

		deadlineSocket := newFakeLiveSocket()
		deadlineAdapter, deadlineCommand := testLiveAdapter(t, deadlineSocket, []string{"AAA"})
		deadlineCommand.Durations.HandshakeStep = 3 * time.Millisecond
		deadlineCommand.Durations.HandshakeTotal = 20 * time.Millisecond
		deadlineAttempt, _, _ := deadlineAdapter.Start(context.Background(), deadlineCommand)
		if _, err := deadlineAttempt.Handshake(context.Background()); !errors.Is(err, errTransportFailed) {
			t.Fatalf("deadline = %v", err)
		}
		terminal, ok = deadlineAttempt.nextForProof(context.Background())
		if !ok || terminal.Terminal.Reason != TerminalConnectedDeadline {
			t.Fatalf("deadline terminal = %+v", terminal)
		}

		totalSocket := newFakeLiveSocket()
		totalAdapter, totalCommand := testLiveAdapter(t, totalSocket, []string{"AAA"})
		totalCommand.Durations.HandshakeStep = 50 * time.Millisecond
		totalCommand.Durations.HandshakeTotal = 3 * time.Millisecond
		totalAttempt, _, _ := totalAdapter.Start(context.Background(), totalCommand)
		if _, err := totalAttempt.Handshake(context.Background()); !errors.Is(err, errTransportFailed) {
			t.Fatalf("whole-handshake deadline = %v", err)
		}
		if terminal, ok := totalAttempt.nextForProof(context.Background()); !ok || terminal.Terminal.Reason != TerminalConnectedDeadline {
			t.Fatalf("whole-handshake terminal = %+v %v", terminal, ok)
		}
	})

	t.Run("start owns progress even when handshake is never awaited", func(t *testing.T) {
		idleSocket := newFakeLiveSocket()
		idleAdapter, command := testLiveAdapter(t, idleSocket, []string{"AAA"})
		ctx, cancel := context.WithCancel(context.Background())
		idleAttempt, _, err := idleAdapter.Start(ctx, command)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer waitCancel()
		if err := idleAttempt.Wait(waitCtx); err != nil {
			t.Fatalf("unawaited handshake did not terminate: %v", err)
		}
		if terminal, ok := idleAttempt.nextForProof(context.Background()); !ok || terminal.Kind != DeliveryTerminal || !idleAdapter.Accounting().Reconciles() {
			t.Fatalf("unawaited terminal/accounting = %+v %v %+v", terminal, ok, idleAdapter.Accounting())
		}
	})

	t.Run("later explicit unsupported family preserves subsequent aggregates", func(t *testing.T) {
		prefixSocket := newFakeLiveSocket()
		enqueueHandshake(prefixSocket)
		prefixAdapter, command := testLiveAdapter(t, prefixSocket, []string{"AAA"})
		prefixAttempt, _, _ := startHandshake(t, prefixAdapter, command)
		start := prefixAdapter.binding.SessionStart().Add(10 * time.Second)
		prefixSocket.send(socketMessageText, "["+aggregateLiveJSON("AAA", start, `"v":1,"z":1`)+`,{"ev":"unknown"}]`)
		first, ok := prefixAttempt.nextForProof(context.Background())
		if !ok || first.Kind != DeliveryAggregate || first.Position.ArrayIndex != 0 {
			t.Fatalf("causal prefix = %+v %v", first, ok)
		}
		unsupported, ok := prefixAttempt.nextForProof(context.Background())
		if !ok || unsupported.Kind != DeliveryNormalizationDrop || unsupported.Rejection.Family != LiveFamilyUnsupported || unsupported.Position.ArrayIndex != 1 {
			t.Fatalf("unsupported fact = %+v %v", unsupported, ok)
		}
		if !prefixAdapter.Accounting().Reconciles() || !prefixAttempt.QueueAccounting().Reconciles() {
			t.Fatalf("unsupported accounting: adapter=%+v queue=%+v", prefixAdapter.Accounting(), prefixAttempt.QueueAccounting())
		}
	})
}

func TestPHRHeartbeatInboundProgressAndQuietDeadline(t *testing.T) {
	t.Run("failed heartbeat with inbound aggregate progress is nonterminal", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		now := binding.SessionStart().Add(20 * time.Second)
		delay := time.Duration(0)
		state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 8, EvaluationDelay: &delay})
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			state.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = state.Wait(ctx)
		}()
		admission, installed := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
		if admission != engine.AdmissionAdmitted || (<-installed).Code != engine.DispositionBindingInstalled {
			t.Fatal("binding install")
		}

		socket := newFakeLiveSocket()
		socket.blockPing, socket.pingStarted, socket.pingRelease = true, make(chan struct{}, 1), make(chan struct{})
		socket.pingError = errors.New("fixture ping failure")
		enqueueHandshake(socket)
		adapter, command := testLiveAdapter(t, socket, []string{"AAA"})
		command.Durations.HeartbeatInterval = 20 * time.Millisecond
		command.Durations.HeartbeatDeadline = 200 * time.Millisecond
		attempt, started, handshake := startHandshake(t, adapter, command)
		defer closeAttemptForTest(t, attempt, 90)
		if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("start=%+v err=%v", result, err)
		}
		for _, delivery := range handshake {
			result, err := DeliverToEngine(context.Background(), state, delivery)
			if err != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
				t.Fatalf("handshake=%+v err=%v", result, err)
			}
		}
		planAdmission, planDone := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}})
		if planAdmission != engine.AdmissionAdmitted || (<-planDone).Code != engine.DispositionHydrationPlanApplied {
			t.Fatal("hydration plan")
		}
		select {
		case <-socket.pingStarted:
		case <-time.After(time.Second):
			t.Fatal("heartbeat did not start")
		}
		window := now.Add(-2 * time.Second).Truncate(time.Second)
		socket.send(socketMessageText, "["+aggregateLiveJSON("AAA", window, "")+"]")
		deadline := time.Now().Add(time.Second)
		for attempt.QueueAccounting().FramesRead < 4 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		close(socket.pingRelease)
		result, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
		if err != nil || !ok || result.AggregateDisposition.Code != engine.DispositionAggregateInserted {
			t.Fatalf("aggregate disposition=%+v ok=%t err=%v", result, ok, err)
		}
		deadline = time.Now().Add(time.Second)
		for adapter.Accounting().HeartbeatInboundProgressOccurrences != 1 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		view, accounting := state.ObserveOperational(), adapter.Accounting()
		if view.Lifecycle != "hydrating" || !view.Hydration.Active || !view.Connection.Active || accounting.HeartbeatCapturedFrameSequence >= accounting.HeartbeatReadFrameSequence ||
			accounting.HeartbeatLatestOutcome != TerminalHeartbeatFailureWithProgress || accounting.HeartbeatInboundProgressOccurrences != 1 || accounting.HeartbeatInboundProgressConsecutive != 1 {
			t.Fatalf("nonterminal heartbeat view=%+v accounting=%+v", view, accounting)
		}
		socket.mu.Lock()
		socket.blockPing, socket.pingError = false, nil
		socket.mu.Unlock()
		deadline = time.Now().Add(time.Second)
		for adapter.Accounting().HeartbeatInboundProgressConsecutive != 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		accounting = adapter.Accounting()
		if accounting.HeartbeatInboundProgressOccurrences != 1 || accounting.HeartbeatInboundProgressConsecutive != 0 || !state.ObserveOperational().Hydration.Active {
			t.Fatalf("successful heartbeat reset=%+v view=%+v", accounting, state.ObserveOperational())
		}
	})

	t.Run("quiet heartbeat deadline terminates exactly once", func(t *testing.T) {
		socket := newFakeLiveSocket()
		socket.blockPing, socket.pingStarted, socket.pingRelease = true, make(chan struct{}, 1), make(chan struct{})
		enqueueHandshake(socket)
		adapter, command := testLiveAdapter(t, socket, []string{"AAA"})
		command.Durations.HeartbeatInterval = 2 * time.Millisecond
		command.Durations.HeartbeatDeadline = 5 * time.Millisecond
		attempt, _, _ := startHandshake(t, adapter, command)
		terminal, ok := attempt.nextForProof(context.Background())
		if !ok || terminal.Terminal.Source != TerminalHeartbeat || terminal.Terminal.Reason != TerminalHeartbeatDeadlineNoProgress {
			t.Fatalf("quiet terminal=%+v", terminal)
		}
		if again, ok := attempt.nextForProof(context.Background()); ok {
			t.Fatalf("duplicate terminal=%+v", again)
		}
	})
}

func TestCapacityTerminalReasonsAndOperandsFollowReaderAdmissionPath(t *testing.T) {
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	binding := component4TestBinding(t, []string{"AAA"})
	adapter, err := NewLiveAdapter(binding, LiveAdapterConfig{Endpoint: "wss://offline.invalid/stocks", Credential: "fixture",
		Queue: LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 4096}})
	if err != nil {
		t.Fatal(err)
	}
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{socket}}
	attempt, _, err := adapter.Start(context.Background(), OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1,
		Durations: OperationalDurations{Dial: time.Second, HandshakeStep: time.Second, HandshakeTotal: 4 * time.Second, HeartbeatInterval: time.Hour, HeartbeatDeadline: time.Second, Write: time.Second, Close: 20 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attempt.Handshake(context.Background()); err != nil {
		t.Fatal(err)
	}
	socket.send(socketMessageText, "["+strings.Repeat(" ", 2998)+"]")
	socket.send(socketMessageText, "["+strings.Repeat(" ", 1998)+"]")
	// Do not start the consumer until the reader has attempted both admissions.
	// Otherwise it can drain the first frame before the second reaches the byte
	// check, making this reader-boundary proof scheduler-dependent.
	rejectionDeadline := time.Now().Add(time.Second)
	for attempt.QueueAccounting().FramesRejectedByteCapacity != 1 {
		if time.Now().After(rejectionDeadline) {
			t.Fatalf("reader did not reach byte-capacity rejection: %+v", attempt.QueueAccounting())
		}
		time.Sleep(time.Millisecond)
	}
	terminalCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	delivery, ok := attempt.nextForProof(terminalCtx)
	if !ok || delivery.Kind != DeliveryTerminal || delivery.Terminal.Reason != TerminalFrameByteCapacity || delivery.Terminal.IncomingFrameBytes != 2000 ||
		delivery.Terminal.QueueAtCause.FramesQueued != 1 || delivery.Terminal.QueueAtCause.QueuedBytes != 3000 || delivery.Terminal.QueueAtCause.CapacityBytes != 4096 ||
		delivery.Terminal.QueueAtCause.FramesRejectedByteCapacity != 1 || delivery.Terminal.QueueAtCause.FramesRejectedSlotCapacity != 0 {
		t.Fatalf("byte capacity terminal=%+v ok=%t", delivery, ok)
	}
}

func closeAttemptForTest(t *testing.T, attempt *LiveAttempt, token uint64) {
	t.Helper()
	if attempt == nil {
		return
	}
	_ = attempt.Close(CloseEpochCommand{BindingIdentity: attempt.binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: token, Cause: CloseControlledStop})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = attempt.Wait(ctx)
}

func TestPC5BoundRawFIFOAccountingAndDrain(t *testing.T) {
	if validateLiveQueueConfig(LiveQueueConfig{}) || validateLiveQueueConfig(LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots + 1, MaxFrameBytes: 1, TotalFrameBytes: 1}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: MaximumLiveFrameBytes + 1, TotalFrameBytes: MaximumLiveQueueBytes}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 2, TotalFrameBytes: 1}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 1, TotalFrameBytes: MaximumLiveQueueBytes + 1}) ||
		!validateLiveQueueConfig(LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes}) {
		t.Fatal("queue ceiling validation mismatch")
	}
	queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 2, MaxFrameBytes: 4, TotalFrameBytes: 6})
	at := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	source := []byte("abc")
	first, reason, _ := queue.tryEnqueue(9, socketMessageText, at, source)
	if reason != FrameAdmitted {
		t.Fatal(reason)
	}
	source[0] = 'z'
	second, reason, _ := queue.tryEnqueue(9, socketMessageBinary, at, []byte("def"))
	if reason != FrameAdmitted || second.sequence != first.sequence+1 {
		t.Fatalf("second = %+v %s", second, reason)
	}
	if _, reason, _ = queue.tryEnqueue(9, socketMessageText, at, []byte("x")); reason != FrameRejectedSlotCapacity {
		t.Fatalf("count capacity = %s", reason)
	}
	if string(queue.frames[0].data) != "abc" {
		t.Fatalf("copy-on-admission = %q", queue.frames[0].data)
	}
	if _, reason, _ = queue.tryEnqueue(9, socketMessageText, at.Add(-time.Nanosecond), []byte("x")); reason != FrameRejectedReceipt {
		t.Fatalf("receipt regression = %s", reason)
	}
	if _, reason, _ = queue.tryEnqueue(9, socketMessageText, at, []byte("12345")); reason != FrameRejectedOversize {
		t.Fatalf("oversize = %s", reason)
	}
	queue.closeGate()
	if _, reason, _ = queue.tryEnqueue(9, socketMessageText, at, []byte("x")); reason != FrameRejectedGate {
		t.Fatalf("gate = %s", reason)
	}
	type markerResult struct {
		frame queuedLiveFrame
		ok    bool
	}
	markerDone := make(chan markerResult, 1)
	go func() {
		frame, ok := queue.enqueueTerminal(context.Background(), 9, at)
		markerDone <- markerResult{frame: frame, ok: ok}
	}()
	select {
	case <-markerDone:
		t.Fatal("full queue admitted terminal marker early")
	case <-time.After(2 * time.Millisecond):
	}
	popped, ok := queue.pop(context.Background())
	if !ok || popped.sequence != first.sequence || string(popped.data) != "abc" {
		t.Fatalf("FIFO sequence = %+v", popped)
	}
	queue.complete(popped, false)
	markerAdmission := <-markerDone
	if !markerAdmission.ok {
		t.Fatal("terminal marker was not admitted after capacity")
	}
	marker, _ := queue.pop(context.Background())
	if !marker.terminal || marker.sequence != markerAdmission.frame.sequence || marker.sequence <= second.sequence {
		t.Fatalf("terminal order = %+v", marker)
	}
	queue.complete(marker, false)
	accounting := queue.snapshot()
	if !accounting.Reconciles() || accounting.FramesRead != 6 || accounting.FramesAdmitted != 2 || accounting.FramesRejectedSlotCapacity != 1 ||
		accounting.FramesDispositioned != 1 || accounting.FramesFenced != 1 || accounting.QueuedBytes != 0 {
		t.Fatalf("accounting = %+v", accounting)
	}
	canceledQueue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 4, TotalFrameBytes: 4})
	_, _, _ = canceledQueue.tryEnqueue(1, socketMessageText, at, []byte("x"))
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := canceledQueue.enqueueTerminal(canceledCtx, 1, at); ok || !canceledQueue.snapshot().Reconciles() {
		t.Fatalf("canceled terminal linkage mutated accounting: %+v", canceledQueue.snapshot())
	}

	t.Run("byte-only saturation is atomic before slot saturation", func(t *testing.T) {
		byteQueue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 3, MaxFrameBytes: 4, TotalFrameBytes: 5})
		if _, reason, _ := byteQueue.tryEnqueue(1, socketMessageText, at, []byte("abc")); reason != FrameAdmitted {
			t.Fatal(reason)
		}
		before := byteQueue.snapshot()
		if _, reason, _ := byteQueue.tryEnqueue(1, socketMessageText, at, []byte("def")); reason != FrameRejectedByteCapacity {
			t.Fatalf("byte saturation = %s", reason)
		}
		after := byteQueue.snapshot()
		if after.FramesQueued != before.FramesQueued || after.QueuedBytes != before.QueuedBytes || after.FramesRejectedByteCapacity != 1 || !after.Reconciles() {
			t.Fatalf("byte rejection mutated queue: before=%+v after=%+v", before, after)
		}
	})

	t.Run("capacity terminal captures exact operands and active engine delivery", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		adapter := &LiveAdapter{binding: binding, clock: func() time.Time { return at }, accounting: AdapterAccounting{ConnectionAttempts: 1, AttemptsConnected: 1}}
		attemptCtx, cancelAttempt := context.WithCancel(context.Background())
		activeStartedAt := time.Now().Add(-2 * time.Second)
		socket := newFakeLiveSocket()
		attempt := &LiveAttempt{adapter: adapter, binding: binding, epoch: 1, ctx: attemptCtx, cancel: cancelAttempt, started: true,
			queue: newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 4, TotalFrameBytes: 4}), connection: socket, cleanupDone: make(chan struct{}),
			durations:          OperationalDurations{HeartbeatInterval: time.Hour, HeartbeatDeadline: time.Second, Close: 10 * time.Millisecond},
			activeDeliveryKind: DeliveryAggregateIngressFence, activeDeliveryStartedAt: activeStartedAt}
		adapter.active = attempt
		socket.send(socketMessageText, "abc")
		socket.send(socketMessageText, "def")
		attempt.startWorkers()
		waitCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		if err := attempt.Wait(waitCtx); err != nil {
			t.Fatal(err)
		}
		attempt.mu.Lock()
		terminal := attempt.terminalDelivery.Terminal
		attempt.mu.Unlock()
		if terminal.Reason != TerminalFrameSlotCapacity || terminal.IncomingFrameBytes != 3 || terminal.QueueAtCause.FramesQueued != 1 ||
			terminal.QueueAtCause.CapacityFrames != 1 || terminal.QueueAtCause.QueuedBytes != 3 || terminal.QueueAtCause.CapacityBytes != 4 ||
			terminal.CausalPosition != (engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}) || !terminal.PositionApplicable ||
			terminal.ActiveDeliveryKind != DeliveryAggregateIngressFence || terminal.ActiveDeliveryStartedAt != activeStartedAt ||
			terminal.ActiveDeliveryAgeAtCause < 2*time.Second || terminal.ActiveDeliveryAgeAtCause > 3*time.Second {
			t.Fatalf("capacity terminal=%+v", terminal)
		}
	})

	t.Run("optional TQ cannot hide later aggregate or control", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		start := binding.SessionStart().Add(10 * time.Second)
		received := start.Add(2 * time.Second)
		trade := fmt.Sprintf(`{"ev":"T","sym":"AAA","x":4,"i":"t1","p":10,"s":1,"t":%d}`, start.UnixMilli())
		aggregate := aggregateLiveJSON("AAA", start, `"dv":"1000.5"`)
		status := `{"ev":"status","status":"success"}`
		statusContext := &StatusContext{ConnectionEpoch: 1, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: "1", ExpectedCount: 1}
		results, counts := NormalizeLiveFrame(LiveFrame{Binding: binding, ConnectionEpoch: 1, FrameSequence: 1, ReceivedAt: received, Data: []byte("[" + trade + "," + aggregate + "," + status + "]")}, statusContext, LiveNormalizationOptions{ShedTradesQuotes: true})
		if len(results) != 3 || results[0].Rejection.Reason != LiveRejectOptionalShed || results[1].Kind != LiveResultAggregate || results[2].Status.Disposition != StatusAcknowledged || !counts.Reconciles() {
			t.Fatalf("mixed shed preservation = %#v %+v", results, counts)
		}
	})

	t.Run("pressure sample exposes current capacity and oldest raw age", func(t *testing.T) {
		queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 4, MaxFrameBytes: 32, TotalFrameBytes: 128})
		at := time.Date(2026, 8, 8, 15, 0, 0, 0, time.UTC)
		queue.now = func() time.Time { return at.Add(300 * time.Millisecond) }
		if _, reason, _ := queue.tryEnqueue(1, socketMessageText, at, []byte("[]")); reason != FrameAdmitted {
			t.Fatal(reason)
		}
		view := queue.snapshot()
		if view.CapacityFrames != 4 || view.CapacityBytes != 128 || view.OldestWaitingFrameAge != 300*time.Millisecond {
			t.Fatalf("queued pressure view = %+v", view)
		}
		frame, ok := queue.pop(context.Background())
		if active := queue.snapshot(); !ok || active.OldestWaitingFrameAge != 0 || active.ActiveFrameAge != 300*time.Millisecond {
			t.Fatalf("waiting/active ages were not separated: %+v", active)
		}
		queue.complete(frame, false)
		if view = queue.snapshot(); view.OldestWaitingFrameAge != 0 || view.ActiveFrameAge != 0 || !view.Reconciles() {
			t.Fatalf("completed pressure view = %+v", view)
		}
	})
}

func TestPLBRC2WriteBoundaryUsesGreatestAdmittedRawFrame(t *testing.T) {
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapter(t, socket, []string{"AAA"})
	attempt, _, _ := startHandshake(t, adapter, open)
	defer closeAttemptForTest(t, attempt, 90)

	received := adapter.now().UTC()
	prior, ok := attempt.queue.lastRawPosition(attempt.Epoch())
	if !ok {
		t.Fatal("handshake did not establish an admitted raw-frame prefix")
	}
	boundarySequence := prior.FrameSequence + 1
	if frame, reason, _ := attempt.queue.tryEnqueue(attempt.Epoch(), socketMessageText, received, []byte(`[]`)); reason != FrameAdmitted || frame.sequence != boundarySequence {
		t.Fatalf("admitted boundary frame = sequence=%d reason=%s", frame.sequence, reason)
	}
	oversize := make([]byte, attempt.queue.config.MaxFrameBytes+1)
	if _, reason, _ := attempt.queue.tryEnqueue(attempt.Epoch(), socketMessageText, received, oversize); reason != FrameRejectedOversize {
		t.Fatalf("diagnostic read attempt was not rejected: %s", reason)
	}
	delivery, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(),
		CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA"}})
	if err != nil {
		t.Fatal(err)
	}
	accounting := attempt.QueueAccounting()
	if accounting.FramesRead != accounting.FramesAdmitted+accounting.FramesRejectedOversize+accounting.FramesRejectedCapacity+accounting.FramesRejectedReceipt+accounting.FramesRejectedGateOrClose ||
		delivery.TQWriteBoundary != (engine.LivePosition{ConnectionEpoch: attempt.Epoch(), FrameSequence: boundarySequence}) {
		t.Fatalf("write boundary followed read attempts: boundary=%+v queue=%+v", delivery.TQWriteBoundary, accounting)
	}
}

func TestPLBRC2GenericStatusIsInformationalAndProviderErrorIsBounded(t *testing.T) {
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapter(t, socket, []string{"AAA"})
	attempt, _, _ := startHandshake(t, adapter, open)
	defer closeAttemptForTest(t, attempt, 90)
	if _, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(),
		CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA"}}); err != nil {
		t.Fatal(err)
	}
	socket.send(socketMessageText, `[{"ev":"status","status":"success"},{"ev":"status","status":"error","message":"must-not-escape"}]`)
	delivery, ok := attempt.nextForProof(context.Background())
	if !ok || delivery.Kind != DeliveryTQControlError || delivery.Position.ArrayIndex != 1 || delivery.TQControlError.Position != delivery.Position {
		t.Fatalf("bounded provider error = %+v ok=%t", delivery, ok)
	}
	if strings.Contains(fmt.Sprint(delivery), "must-not-escape") || !adapter.Accounting().Reconciles() {
		t.Fatalf("provider prose/accounting escaped: delivery=%+v accounting=%+v", delivery, adapter.Accounting())
	}
}

func TestPLBRC2CommandResultPrecedesPostBoundaryRawDelivery(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	now := binding.SessionStart().Add(20 * time.Minute)
	delay := time.Duration(0)
	state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 64, RequiredReserve: 8, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		state.Close()
		waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = state.Wait(waitCtx)
	}()
	admission, completion := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionBindingInstalled {
		t.Fatal("binding install")
	}

	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapterForBinding(t, socket, binding)
	adapter.clock = func() time.Time { return now }
	attempt, started, handshake := startHandshake(t, adapter, open)
	defer closeAttemptForTest(t, attempt, 90)
	if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("connection attempt = %+v/%v", result, err)
	}
	for _, delivery := range handshake {
		if result, err := DeliverToEngine(context.Background(), state, delivery); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake = %+v/%v", result, err)
		}
	}

	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	planAdmission, planCompletion := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: budgets})
	if planAdmission != engine.AdmissionAdmitted {
		t.Fatal("hydration plan")
	}
	plan := <-planCompletion
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, terminalErr := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if terminalErr != nil {
			t.Fatal(terminalErr)
		}
		_, terminalCompletion := state.AdmitHydrationTerminal(context.Background(), terminal)
		terminalResult := <-terminalCompletion
		if terminalResult.FenceCommand.CommandToken() != 0 {
			fence = terminalResult.FenceCommand
		}
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 3, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := state.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if got := <-fenceCompletion; got.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence = %+v", got)
	}

	base := now
	for index := 0; index < 60; index++ {
		window := base.Add(time.Duration(index) * time.Second)
		now = window.Add(time.Second)
		input := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: "AAA",
			WindowStart: window, WindowEnd: window.Add(time.Second), Values: engine.AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 10_000, VWAP: 12, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
			DeliveryTime: now, Live: engine.LivePosition{ConnectionEpoch: attempt.Epoch(), FrameSequence: 4, ArrayIndex: uint32(index + 1)}}
		aggregateAdmission, aggregateCompletion := state.AdmitAggregate(context.Background(), input)
		if aggregateAdmission != engine.AdmissionAdmitted || (<-aggregateCompletion).Code != engine.DispositionAggregateInserted {
			t.Fatalf("aggregate %d", index)
		}
	}
	coverageCommand, err := state.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	coverageInput, err := engine.NewLiveCoverageFenceInput(coverageCommand, engine.LiveCoverageFenceComplete, 4, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	_, coverageCompletion := state.AdmitLiveCoverageFence(context.Background(), coverageInput)
	if got := <-coverageCompletion; got.Code != engine.DispositionLiveCoverageFenceApplied {
		t.Fatalf("coverage fence = %+v", got)
	}
	_, timerCompletion := state.AdmitTimer(context.Background())
	if got := <-timerCompletion; got.Code != engine.DispositionTimerApplied {
		t.Fatalf("qualification timer = %+v", got)
	}
	command, err := state.IssueTQCommand()
	if err != nil {
		t.Fatalf("T/Q command = %v view=%+v", err, state.ObserveTQ())
	}
	adapterCommand, err := ChangeTQCommandFromEngine(command)
	if err != nil {
		t.Fatal(err)
	}

	receivedAt := time.Now().UTC()
	tradeAt := now.Add(-100 * time.Millisecond)
	frameB := fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"at-b","p":12,"s":1,"t":%d}]`, tradeAt.UnixMilli())
	queuedB, reason, _ := attempt.queue.tryEnqueue(attempt.Epoch(), socketMessageText, receivedAt, []byte(frameB))
	if reason != FrameAdmitted {
		t.Fatal(reason)
	}

	beforeEvaluation := state.ObserveReplayDeterministic().Evaluation
	beforeOperational := state.ObserveOperational()
	captured := make(chan engine.LivePosition, 1)
	release := make(chan struct{})
	attempt.mu.Lock()
	attempt.beforeEngineDelivery = func(delivery AdapterDelivery) {
		if delivery.Control.Kind == engine.TradeQuoteCommandWriteResult {
			captured <- delivery.TQWriteBoundary
			<-release
		}
	}
	attempt.mu.Unlock()
	type deliveredResult struct {
		result EngineDeliveryResult
		ok     bool
		err    error
	}
	commandDone := make(chan deliveredResult, 1)
	go func() {
		result, ok, changeErr := attempt.ChangeTQAndDeliver(context.Background(), state, adapterCommand)
		commandDone <- deliveredResult{result: result, ok: ok, err: changeErr}
	}()
	var boundary engine.LivePosition
	select {
	case boundary = <-captured:
	case <-time.After(time.Second):
		t.Fatal("command did not pause after B capture")
	}
	if boundary.FrameSequence != queuedB.sequence {
		t.Fatalf("captured B = %+v queued=%+v", boundary, queuedB)
	}
	frameNext := fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"after-b","p":12,"s":1,"t":%d}]`, tradeAt.Add(time.Millisecond).UnixMilli())
	queuedNext, reason, _ := attempt.queue.tryEnqueue(attempt.Epoch(), socketMessageText, receivedAt.Add(time.Microsecond), []byte(frameNext))
	if reason != FrameAdmitted || queuedNext.sequence != queuedB.sequence+1 {
		t.Fatalf("B+1 = %+v/%s", queuedNext, reason)
	}
	rawDone := make(chan deliveredResult, 1)
	go func() {
		result, ok, deliveryErr := attempt.DeliverNextToEngine(context.Background(), state)
		rawDone <- deliveredResult{result: result, ok: ok, err: deliveryErr}
	}()
	select {
	case result := <-rawDone:
		t.Fatalf("raw delivery crossed paused command result: %+v", result)
	default:
	}
	close(release)
	var commandResult deliveredResult
	select {
	case commandResult = <-commandDone:
	case <-time.After(time.Second):
		t.Fatal("command result did not complete")
	}
	if commandResult.err != nil || !commandResult.ok || commandResult.result.TQDisposition.Code != engine.DispositionTQApplied {
		t.Fatalf("command result = %+v", commandResult)
	}
	var atBoundary deliveredResult
	select {
	case atBoundary = <-rawDone:
	case <-time.After(time.Second):
		t.Fatal("frame B did not complete")
	}
	if atBoundary.err != nil || !atBoundary.ok || atBoundary.result.TQDisposition.Code != engine.DispositionTQFenced ||
		commandResult.result.TQDisposition.EngineSequence >= atBoundary.result.TQDisposition.EngineSequence {
		t.Fatalf("frame B ordering = command=%+v B=%+v", commandResult, atBoundary)
	}
	afterBoundary, ok, deliveryErr := attempt.DeliverNextToEngine(context.Background(), state)
	if deliveryErr != nil || !ok || afterBoundary.TQDisposition.Code != engine.DispositionTQApplied ||
		atBoundary.result.TQDisposition.EngineSequence >= afterBoundary.TQDisposition.EngineSequence {
		t.Fatalf("frame B+1 ordering = B=%+v B+1=%+v/%v/%v", atBoundary, afterBoundary, ok, deliveryErr)
	}
	view := state.ObserveTQ()
	if !view.Rows[0].TradeCoverage || view.Rows[0].QuoteCoverage || view.Rows[0].ProviderPresent {
		t.Fatalf("B+1 did not independently confirm T: %+v", view)
	}
	afterOperational := state.ObserveOperational()
	if !reflect.DeepEqual(beforeEvaluation, state.ObserveReplayDeterministic().Evaluation) ||
		!reflect.DeepEqual(beforeOperational.Aggregates, afterOperational.Aggregates) || !reflect.DeepEqual(beforeOperational.Connection, afterOperational.Connection) ||
		!adapter.Accounting().Reconciles() || !attempt.QueueAccounting().Reconciles() {
		t.Fatalf("aggregate/control/accounting changed: before=%+v after=%+v adapter=%+v queue=%+v", beforeOperational, afterOperational, adapter.Accounting(), attempt.QueueAccounting())
	}
}

func TestMaximumLiveQueueSlotCeilingAdmitsThenFailsClosed(t *testing.T) {
	queue := newLiveFrameQueue(LiveQueueConfig{
		FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes,
	})
	at := time.Date(2026, 8, 12, 21, 0, 0, 0, time.UTC)
	for index := 0; index < MaximumLiveFrameSlots; index++ {
		frame, reason, _ := queue.tryEnqueue(1, socketMessageText, at, []byte("x"))
		if reason != FrameAdmitted || frame.sequence != uint64(index+1) {
			t.Fatalf("admission index=%d sequence=%d reason=%s", index, frame.sequence, reason)
		}
	}
	if _, reason, snapshot := queue.tryEnqueue(1, socketMessageText, at, []byte("x")); reason != FrameRejectedSlotCapacity ||
		snapshot.FramesQueued != MaximumLiveFrameSlots || snapshot.HighFramesQueued != MaximumLiveFrameSlots ||
		snapshot.FramesRejectedSlotCapacity != 1 || snapshot.QueuedBytes != MaximumLiveFrameSlots {
		t.Fatalf("ceiling reason=%s snapshot=%+v", reason, snapshot)
	}
	half := uint64(MaximumLiveFrameSlots / 2)
	for sequence := uint64(1); sequence <= half; sequence++ {
		frame, ok := queue.pop(context.Background())
		if !ok || frame.sequence != sequence {
			t.Fatalf("initial drain sequence=%d frame=%+v ok=%t", sequence, frame, ok)
		}
		queue.complete(frame, false)
	}
	for index := uint64(0); index < half; index++ {
		frame, reason, _ := queue.tryEnqueue(1, socketMessageText, at, []byte("x"))
		if reason != FrameAdmitted || frame.sequence != uint64(MaximumLiveFrameSlots)+index+1 {
			t.Fatalf("wrap admission index=%d sequence=%d reason=%s", index, frame.sequence, reason)
		}
	}
	for sequence := half + 1; sequence <= uint64(MaximumLiveFrameSlots)+half; sequence++ {
		frame, ok := queue.pop(context.Background())
		if !ok || frame.sequence != sequence {
			t.Fatalf("wrapped drain sequence=%d frame=%+v ok=%t", sequence, frame, ok)
		}
		queue.complete(frame, false)
	}
	if accounting := queue.snapshot(); !accounting.Reconciles() || accounting.FramesQueued != 0 || accounting.QueuedBytes != 0 ||
		accounting.FramesDispositioned != uint64(MaximumLiveFrameSlots)+half {
		t.Fatalf("final accounting=%+v", accounting)
	}
}

func TestFenceBurstEnvelopeFIFOAndIndependentCapacityFailures(t *testing.T) {
	const envelope = 7_774
	queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes})
	at := time.Date(2026, 8, 12, 18, 44, 40, 0, time.UTC)
	for index := 0; index < envelope; index++ {
		frame, reason, _ := queue.tryEnqueue(11, socketMessageText, at.Add(time.Duration(index)*time.Microsecond), []byte("[]"))
		if reason != FrameAdmitted || frame.sequence != uint64(index+1) {
			t.Fatalf("envelope admission index=%d sequence=%d reason=%s", index, frame.sequence, reason)
		}
	}
	fact, admitted := queue.enqueueIngressFence(context.Background(), AggregateIngressFenceFact{})
	if !admitted || fact.ThroughFrameSequence != envelope || fact.MarkerOrdinal != 1 {
		t.Fatalf("fence=%+v admitted=%t", fact, admitted)
	}
	for sequence := uint64(1); sequence <= envelope; sequence++ {
		frame, ok := queue.pop(context.Background())
		if !ok || frame.kind != queuedLiveRaw || frame.sequence != sequence {
			t.Fatalf("fifo sequence=%d frame=%+v ok=%t", sequence, frame, ok)
		}
		queue.complete(frame, false)
	}
	marker, ok := queue.pop(context.Background())
	if !ok || marker.kind != queuedLiveIngressFence || marker.ingressFence.ThroughFrameSequence != envelope {
		t.Fatalf("marker=%+v ok=%t", marker, ok)
	}
	queue.complete(marker, false)
	accounting := queue.snapshot()
	if !accounting.Reconciles() || accounting.FramesRead != envelope || accounting.FramesDispositioned != envelope || accounting.FramesQueued != 0 || accounting.QueuedBytes != 0 || accounting.HighFramesQueued != envelope {
		t.Fatalf("burst accounting=%+v", accounting)
	}

	slot := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 2, MaxFrameBytes: 16, TotalFrameBytes: 32})
	_, _, _ = slot.tryEnqueue(1, socketMessageText, at, []byte("a"))
	_, _, _ = slot.tryEnqueue(1, socketMessageText, at, []byte("b"))
	if _, reason, snapshot := slot.tryEnqueue(1, socketMessageText, at, []byte("c")); reason != FrameRejectedSlotCapacity || snapshot.FramesRejectedSlotCapacity != 1 || snapshot.FramesRejectedByteCapacity != 0 {
		t.Fatalf("slot control reason=%s accounting=%+v", reason, snapshot)
	}
	bytes := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 4, MaxFrameBytes: 16, TotalFrameBytes: 16})
	_, _, _ = bytes.tryEnqueue(1, socketMessageText, at, []byte("123456789"))
	if _, reason, snapshot := bytes.tryEnqueue(1, socketMessageText, at, []byte("12345678")); reason != FrameRejectedByteCapacity || snapshot.FramesRejectedByteCapacity != 1 || snapshot.FramesRejectedSlotCapacity != 0 {
		t.Fatalf("byte control reason=%s accounting=%+v", reason, snapshot)
	}
}

func TestFenceBurstBehindActiveRealFencePreservesPrefixAndDrains(t *testing.T) {
	const envelope = 7_774
	baselineGoroutines := runtime.NumGoroutine()
	var memoryBefore, memoryHeld runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	binding := component4TestBinding(t, []string{"AAA"})
	now := binding.SessionStart().Add(30 * time.Second)
	delay := time.Duration(0)
	state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 8, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		state.Close()
		wait, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = state.Wait(wait)
	}()
	installed, completion := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if installed != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionBindingInstalled {
		t.Fatal("binding install")
	}

	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, err := NewLiveAdapter(binding, LiveAdapterConfig{Endpoint: "wss://offline.invalid/stocks", Credential: "fixture",
		Queue: LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes}})
	if err != nil {
		t.Fatal(err)
	}
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{socket}}
	open := OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1, Durations: OperationalDurations{Dial: time.Second, HandshakeStep: time.Second, HandshakeTotal: 4 * time.Second, HeartbeatInterval: time.Hour, HeartbeatDeadline: time.Second, Write: time.Second, Close: 20 * time.Millisecond}}
	attempt, started, handshake := startHandshake(t, adapter, open)
	if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("start=%+v err=%v", result, err)
	}
	for _, delivery := range handshake {
		result, err := DeliverToEngine(context.Background(), state, delivery)
		if err != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("handshake=%+v err=%v", result, err)
		}
	}
	planAdmission, planCompletion := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}})
	if planAdmission != engine.AdmissionAdmitted {
		t.Fatal(planAdmission)
	}
	plan := <-planCompletion
	token := plan.Plan.Requests()[0]
	terminal, _ := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 1, 0, 0, 0)
	_, terminalCompletion := state.AdmitHydrationTerminal(context.Background(), terminal)
	terminalResult := <-terminalCompletion
	command, err := CaptureAggregateIngressFenceCommandFromEngine(terminalResult.FenceCommand)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.CaptureAggregateIngressFence(context.Background(), state, command); err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC().Add(time.Second)

	release := make(chan struct{})
	attempt.beforeEngineDelivery = func(delivery AdapterDelivery) {
		if delivery.Kind == DeliveryAggregateIngressFence {
			<-release
		}
	}
	fenceDone := make(chan EngineDeliveryResult, 1)
	go func() { result, _, _ := attempt.DeliverNextToEngine(context.Background(), state); fenceDone <- result }()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && attempt.ActiveDeliveryDiagnostic().Kind != DeliveryAggregateIngressFence {
		time.Sleep(time.Millisecond)
	}
	if attempt.ActiveDeliveryDiagnostic().Kind != DeliveryAggregateIngressFence {
		t.Fatal("real fence did not become active")
	}
	receivedAt := now.Add(time.Second)
	payload := []byte("[" + aggregateLiveJSON("AAA", binding.SessionStart().Add(10*time.Second), `"dv":"1000.5"`) + "]")
	for index := 0; index < envelope; index++ {
		_, reason, _ := attempt.queue.tryEnqueue(attempt.Epoch(), socketMessageText, receivedAt.Add(time.Duration(index)*time.Microsecond), payload)
		if reason != FrameAdmitted {
			t.Fatalf("later frame %d=%s", index, reason)
		}
	}
	if accounting := attempt.QueueAccounting(); accounting.FramesQueued != envelope || accounting.FramesRejectedCapacity != 0 {
		t.Fatalf("held burst=%+v", accounting)
	}
	runtime.ReadMemStats(&memoryHeld)
	if memoryHeld.TotalAlloc-memoryBefore.TotalAlloc > 64<<20 {
		t.Fatalf("burst allocation exceeded descriptor/payload bound: before=%d held=%d", memoryBefore.TotalAlloc, memoryHeld.TotalAlloc)
	}
	close(release)
	if result := <-fenceDone; result.HydrationDisposition.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence=%+v", result)
	}
	publication := state.ObserveSnapshot().Publication
	if publication.Watermark == nil || publication.LastDisposition != engine.DispositionAggregateIngressFenceApplied || publication.LastEngineSequence == 0 {
		t.Fatalf("fence publication=%+v", publication)
	}
	for index := 0; index < envelope; index++ {
		result, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
		if err != nil || !ok || result.AggregateDisposition.Code == "" || result.AggregateDisposition.EngineSequence <= publication.LastEngineSequence {
			t.Fatalf("later delivery %d=%+v ok=%t err=%v", index, result, ok, err)
		}
	}
	if after := state.ObserveSnapshot().Publication; after.PublicationID != publication.PublicationID || after.LastEngineSequence != publication.LastEngineSequence || after.Watermark == nil || !after.Watermark.Equal(*publication.Watermark) {
		t.Fatalf("post-marker frames changed fence publication before next evaluator boundary: before=%+v after=%+v", publication, after)
	}
	accounting := attempt.QueueAccounting()
	if !accounting.Reconciles() || accounting.FramesQueued != 0 || accounting.QueuedBytes != 0 || accounting.FramesDispositioned < envelope || accounting.HighFramesQueued != envelope {
		t.Fatalf("drain=%+v", accounting)
	}
	if err := attempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Cause: CloseControlledStop}); err != nil {
		t.Fatal(err)
	}
	terminalDelivery, ok := attempt.nextForProof(context.Background())
	if !ok || terminalDelivery.Kind != DeliveryTerminal {
		t.Fatalf("joined terminal=%+v ok=%t", terminalDelivery, ok)
	}
	wait, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := attempt.Wait(wait); err != nil || runtime.NumGoroutine() > baselineGoroutines+2 {
		t.Fatalf("attempt did not join: err=%v goroutines=%d baseline=%d", err, runtime.NumGoroutine(), baselineGoroutines)
	}
}

// TestC6FENCE01RawFrameMarkerEngineFIFOLinearization is P-C6-FENCE. It proves
// the zero-payload marker is ordered after every prior raw frame without
// consuming a raw sequence, and survives terminal cleanup as canceled evidence
// ahead of the pre-existing C5 terminal marker.
func TestC6FENCE01RawFrameMarkerEngineFIFOLinearization(t *testing.T) {
	queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 32, TotalFrameBytes: 32})
	at := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	first, reason, _ := queue.tryEnqueue(7, socketMessageText, at, []byte("[]"))
	if reason != FrameAdmitted {
		t.Fatal(reason)
	}
	type fenceAdmission struct {
		fact AggregateIngressFenceFact
		ok   bool
	}
	fenceDone := make(chan fenceAdmission, 1)
	go func() {
		fact, ok := queue.enqueueIngressFence(context.Background(), AggregateIngressFenceFact{})
		fenceDone <- fenceAdmission{fact, ok}
	}()
	select {
	case <-fenceDone:
		t.Fatal("full configured queue admitted fence marker")
	case <-time.After(2 * time.Millisecond):
	}
	frame, _ := queue.pop(context.Background())
	if frame.kind != queuedLiveRaw {
		t.Fatalf("first item = %+v", frame)
	}
	admittedFence := <-fenceDone
	fact, ok := admittedFence.fact, admittedFence.ok
	if !ok || fact.ThroughFrameSequence != first.sequence || fact.MarkerOrdinal != 1 || queue.next != first.sequence+1 {
		t.Fatalf("marker identity/order = %+v rawNext=%d first=%d", fact, queue.next, first.sequence)
	}
	queue.complete(frame, false)
	marker, _ := queue.pop(context.Background())
	if marker.kind != queuedLiveIngressFence || marker.ingressFence.ThroughFrameSequence != first.sequence {
		t.Fatalf("marker = %+v", marker)
	}
	queue.complete(marker, false)
	secondFact, secondOK := queue.enqueueIngressFence(context.Background(), AggregateIngressFenceFact{})
	if !secondOK || secondFact.MarkerOrdinal != 2 {
		t.Fatalf("later-generation marker = %+v ok=%v", secondFact, secondOK)
	}
	secondMarker, _ := queue.pop(context.Background())
	queue.complete(secondMarker, false)
	if !queue.snapshot().Reconciles() {
		t.Fatalf("accounting = %+v", queue.snapshot())
	}

	loss := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 32, TotalFrameBytes: 32})
	_, _ = loss.enqueueIngressFence(context.Background(), AggregateIngressFenceFact{})
	terminal, admitted := loss.enqueueTerminal(context.Background(), 7, at)
	if !admitted {
		t.Fatal("terminal not admitted")
	}
	queuedFence, _ := loss.pop(context.Background())
	if queuedFence.kind != queuedLiveIngressFence || queuedFence.ingressFence.State != engine.AggregateIngressFenceCanceled {
		t.Fatalf("loss fence = %+v", queuedFence)
	}
	loss.complete(queuedFence, false)
	queuedTerminal, _ := loss.pop(context.Background())
	if !queuedTerminal.terminal || queuedTerminal.sequence != terminal.sequence {
		t.Fatalf("terminal coexistence = %+v", queuedTerminal)
	}
	loss.complete(queuedTerminal, false)
	if !loss.snapshot().Reconciles() {
		t.Fatalf("loss accounting = %+v", loss.snapshot())
	}

	t.Run("canceled full capture synchronously cancels engine generation", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		now := binding.SessionStart().Add(2 * time.Second)
		delay := time.Duration(0)
		state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 16, RequiredReserve: 4, EvaluationDelay: &delay})
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			state.Close()
			wait, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = state.Wait(wait)
		}()
		admission, installed := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
		if admission != engine.AdmissionAdmitted || (<-installed).Code != engine.DispositionBindingInstalled {
			t.Fatal("binding install")
		}
		admitControl := func(kind engine.ConnectionControlKind, position engine.LivePosition, token uint64) {
			input := engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind,
				ConnectionEpoch: 1, Position: position, ReceiptTime: now, CommandToken: token, Outcome: engine.ControlSucceeded}
			got, completion := state.AdmitConnectionControl(context.Background(), input)
			if got != engine.AdmissionAdmitted {
				t.Fatal(got)
			}
			if disposition := <-completion; disposition.Code != engine.DispositionConnectionControlApplied {
				t.Fatalf("%s = %+v", kind, disposition)
			}
		}
		admitControl(engine.ConnectionAttempt, engine.LivePosition{}, 1)
		admitControl(engine.ConnectionEstablished, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, 1)
		admitControl(engine.AuthenticationResult, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, 1)
		admitControl(engine.AggregateCommandWriteResult, engine.LivePosition{}, 2)
		admitControl(engine.AggregateSubscriptionResult, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 3}, 2)
		planAdmission, planCompletion := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1,
			BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1,
			Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 100, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 100_000, MaximumResidentRecords: 100_000}})
		if planAdmission != engine.AdmissionAdmitted {
			t.Fatal(planAdmission)
		}
		plan := <-planCompletion
		token := plan.Plan.Requests()[0]
		terminal, _ := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		_, terminalCompletion := state.AdmitHydrationTerminal(context.Background(), terminal)
		terminalResult := <-terminalCompletion
		command, err := CaptureAggregateIngressFenceCommandFromEngine(terminalResult.FenceCommand)
		if err != nil {
			t.Fatal(err)
		}
		attempt := &LiveAttempt{binding: binding, epoch: 1, started: true, handshaken: true,
			queue: newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 32, TotalFrameBytes: 32})}
		_, _, _ = attempt.queue.tryEnqueue(1, socketMessageText, now, []byte("[]"))
		captureCtx, cancelCapture := context.WithCancel(context.Background())
		cancelCapture()
		if err := attempt.CaptureAggregateIngressFence(captureCtx, state, command); !errors.Is(err, context.Canceled) {
			t.Fatalf("capture cancel = %v", err)
		}
		late, _ := engine.NewAggregateIngressFenceInput(terminalResult.FenceCommand, engine.AggregateIngressFenceComplete, 3, 1, now)
		_, lateCompletion := state.AdmitAggregateIngressFence(context.Background(), late)
		if got := <-lateCompletion; got.Code != engine.DispositionAggregateIngressFenceFenced {
			t.Fatalf("canceled generation accepted late completion: %+v", got)
		}
	})

	t.Run("concurrent delivery preserves raw-before-fence engine order", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		now := binding.SessionStart().Add(30 * time.Second)
		delay := time.Duration(0)
		state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 8, EvaluationDelay: &delay})
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			state.Close()
			wait, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = state.Wait(wait)
		}()
		admission, installed := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
		if admission != engine.AdmissionAdmitted || (<-installed).Code != engine.DispositionBindingInstalled {
			t.Fatal("binding install")
		}

		socket := newFakeLiveSocket()
		enqueueHandshake(socket)
		adapter, open := testLiveAdapterForBinding(t, socket, binding)
		attempt, started, err := adapter.Start(context.Background(), open)
		if err != nil {
			t.Fatal(err)
		}
		type unsafePublicDrain interface {
			Next(context.Context) (AdapterDelivery, bool)
		}
		if _, exposed := any(attempt).(unsafePublicDrain); exposed {
			t.Fatal("unsafe public dequeue path remains exposed")
		}
		if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("start=%+v %v", result, err)
		}
		handshake, err := attempt.Handshake(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, delivery := range handshake {
			if result, err := DeliverToEngine(context.Background(), state, delivery); err != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
				t.Fatalf("handshake=%+v %v", result, err)
			}
		}
		planAdmission, planCompletion := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 100, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 100_000, MaximumResidentRecords: 100_000}})
		if planAdmission != engine.AdmissionAdmitted {
			t.Fatal(planAdmission)
		}
		plan := <-planCompletion
		if len(plan.Plan.Requests()) != 1 {
			t.Fatalf("plan=%+v", plan)
		}
		token := plan.Plan.Requests()[0]
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, terminalCompletion := state.AdmitHydrationTerminal(context.Background(), terminal)
		terminalResult := <-terminalCompletion
		command, err := CaptureAggregateIngressFenceCommandFromEngine(terminalResult.FenceCommand)
		if err != nil {
			t.Fatal(err)
		}

		socket.send(socketMessageText, "["+aggregateLiveJSON("AAA", binding.SessionStart().Add(10*time.Second), `"dv":"1000.5"`)+"]")
		waitForQueuedFrames(t, attempt, 1)
		if err := attempt.CaptureAggregateIngressFence(context.Background(), state, command); err != nil {
			t.Fatal(err)
		}
		now = time.Now().UTC()
		type delivered struct {
			result EngineDeliveryResult
			ok     bool
			err    error
		}
		results := make(chan delivered, 2)
		for range 2 {
			go func() {
				result, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
				results <- delivered{result, ok, err}
			}()
		}
		firstResult, secondResult := <-results, <-results
		if firstResult.err != nil || secondResult.err != nil || !firstResult.ok || !secondResult.ok {
			t.Fatalf("deliveries=%+v %+v", firstResult, secondResult)
		}
		var aggregateSequence, fenceSequence uint64
		for _, got := range []delivered{firstResult, secondResult} {
			if got.result.AggregateDisposition.EngineSequence != 0 {
				aggregateSequence = got.result.AggregateDisposition.EngineSequence
			}
			if got.result.HydrationDisposition.EngineSequence != 0 {
				fenceSequence = got.result.HydrationDisposition.EngineSequence
			}
		}
		if aggregateSequence == 0 || fenceSequence <= aggregateSequence {
			t.Fatalf("engine order aggregate=%d fence=%d", aggregateSequence, fenceSequence)
		}
		if err := attempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Cause: CloseControlledStop}); err != nil {
			t.Fatal(err)
		}
		if _, ok := attempt.nextForProof(context.Background()); !ok {
			t.Fatal("terminal missing")
		}
	})
}

func TestPC5ReconnectFirstCauseMarkerAndExplicitGreaterEpoch(t *testing.T) {
	firstSocket, secondSocket := newFakeLiveSocket(), newFakeLiveSocket()
	enqueueHandshake(firstSocket)
	enqueueHandshake(secondSocket)
	adapter, open := testLiveAdapter(t, firstSocket, []string{"AAA"})
	connector := &fakeLiveConnector{sockets: []*fakeLiveSocket{firstSocket, secondSocket}}
	adapter.connector = connector
	first, _, _ := startHandshake(t, adapter, open)
	firstSocket.send(socketMessageText, `[]`)
	waitForQueuedFrames(t, first, 1)
	first.triggerTerminal(TerminalHeartbeat, TerminalHeartbeatTransportFailure, 0, false)
	first.triggerTerminal(TerminalReader, TerminalReadFailed, 0, false)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	if err := first.Wait(waitCtx); err != nil {
		t.Fatalf("self-contained first cleanup: %v", err)
	}
	waitCancel()
	if !firstSocket.isClosed() || first.QueueAccounting().FramesFenced != 1 {
		t.Fatalf("first cleanup did not close/fence: closed=%v accounting=%+v", firstSocket.isClosed(), first.QueueAccounting())
	}
	terminal, ok := first.nextForProof(context.Background())
	if !ok || terminal.Terminal.Source != TerminalHeartbeat || terminal.Terminal.Reason != TerminalHeartbeatTransportFailure {
		t.Fatalf("first cause = %+v", terminal)
	}
	if _, ok := first.nextForProof(context.Background()); ok {
		t.Fatal("duplicate terminal outcome")
	}
	open.CommandToken = 2
	second, _, handshake := startHandshake(t, adapter, open)
	if second.Epoch() <= first.Epoch() || len(handshake) != 4 {
		t.Fatalf("reopen epoch/handshake = %d -> %d %+v", first.Epoch(), second.Epoch(), handshake)
	}
	if writes := secondSocket.written(); len(writes) != 2 || strings.Contains(writes[1], "T.") || strings.Contains(writes[1], "Q.") {
		t.Fatalf("new epoch did not start A.* only: %q", writes)
	}
	closeCommand := CloseEpochCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: second.Epoch(), CommandToken: 3, Cause: CloseControlledStop}
	if err := second.Close(closeCommand); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(closeCommand); err != nil {
		t.Fatal(err)
	}
	if closed, ok := second.nextForProof(context.Background()); !ok || closed.Terminal.Source != TerminalEngineClose || closed.Terminal.CloseCause != CloseControlledStop {
		t.Fatalf("controlled close = %+v", closed)
	}
	if err := second.Close(closeCommand); err != nil {
		t.Fatalf("post-terminal idempotent close = %v", err)
	}
	firstSocket.send(socketMessageText, `[{"ev":"A"}]`)
	if _, ok := first.nextForProof(context.Background()); ok {
		t.Fatal("late old-reader input escaped after terminal/reopen")
	}

	thirdSocket := newFakeLiveSocket()
	enqueueHandshake(thirdSocket)
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{thirdSocket}}
	open.CommandToken = 4
	rootCtx, rootCancel := context.WithCancel(context.Background())
	third, _, err := adapter.Start(rootCtx, open)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := third.Handshake(rootCtx); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		thirdSocket.send(socketMessageText, `[]`)
	}
	waitForQueuedFrames(t, third, 8)
	rootCancel()
	waitCtx, waitCancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	if err := third.Wait(waitCtx); err != nil {
		t.Fatalf("root-cancel cleanup: %v", err)
	}
	waitCancel()
	rootTerminal, ok := third.nextForProof(context.Background())
	if !ok || rootTerminal.Terminal.Reason != TerminalContextCanceled || third.QueueAccounting().FramesFenced != 8 || !thirdSocket.isClosed() {
		t.Fatalf("root cancellation = terminal=%+v accounting=%+v closed=%v", rootTerminal, third.QueueAccounting(), thirdSocket.isClosed())
	}

	if accounting := adapter.Accounting(); !accounting.Reconciles() || accounting.ConnectionAttempts != 3 || accounting.AttemptsFailed != 1 || accounting.AttemptsCanceled != 2 {
		t.Fatalf("reconnect accounting = %+v", accounting)
	}

	t.Run("concurrent Next callers share one sequential drain", func(t *testing.T) {
		orderedSocket := newFakeLiveSocket()
		enqueueHandshake(orderedSocket)
		orderedAdapter, command := testLiveAdapter(t, orderedSocket, []string{"AAA"})
		orderedAttempt, _, _ := startHandshake(t, orderedAdapter, command)
		start := orderedAdapter.binding.SessionStart().Add(10 * time.Second)
		type callerPosition struct {
			caller   int
			position engine.LivePosition
		}
		positions := make(chan callerPosition, 2)
		var callers sync.WaitGroup
		callers.Add(1)
		go func() {
			defer callers.Done()
			delivery, ok := orderedAttempt.nextForProof(context.Background())
			if ok {
				positions <- callerPosition{caller: 1, position: delivery.Position}
			}
		}()
		waitForNextDrainOwner(t, orderedAttempt)
		callers.Add(1)
		go func() {
			defer callers.Done()
			delivery, ok := orderedAttempt.nextForProof(context.Background())
			if ok {
				positions <- callerPosition{caller: 2, position: delivery.Position}
			}
		}()
		orderedSocket.send(socketMessageText, "["+aggregateLiveJSON("AAA", start, `"v":1,"z":1`)+"]")
		orderedSocket.send(socketMessageText, "["+aggregateLiveJSON("AAA", start.Add(time.Second), `"v":1,"z":1`)+"]")
		callers.Wait()
		close(positions)
		got := make(map[int]engine.LivePosition, 2)
		for result := range positions {
			got[result.caller] = result.position
		}
		if len(got) != 2 || got[1].FrameSequence+1 != got[2].FrameSequence {
			t.Fatalf("concurrent drain = %+v", got)
		}
		if err := orderedAttempt.Close(CloseEpochCommand{BindingIdentity: command.BindingIdentity, ConnectionEpoch: orderedAttempt.Epoch(), CommandToken: 2, Cause: CloseControlledStop}); err != nil {
			t.Fatal(err)
		}
		if _, ok := orderedAttempt.nextForProof(context.Background()); !ok {
			t.Fatal("ordered attempt terminal missing")
		}
	})

	t.Run("blocked Next releases the drain for terminal cleanup", func(t *testing.T) {
		failedSocket := newFakeLiveSocket()
		enqueueHandshake(failedSocket)
		failedAdapter, command := testLiveAdapter(t, failedSocket, []string{"AAA"})
		failedAttempt, _, _ := startHandshake(t, failedAdapter, command)
		type nextResult struct {
			delivery AdapterDelivery
			ok       bool
		}
		nextDone := make(chan nextResult, 1)
		go func() {
			delivery, ok := failedAttempt.nextForProof(context.Background())
			nextDone <- nextResult{delivery: delivery, ok: ok}
		}()
		waitForNextDrainOwner(t, failedAttempt)
		failedSocket.failRead()
		select {
		case result := <-nextDone:
			if !result.ok || result.delivery.Kind != DeliveryTerminal || result.delivery.Terminal.Reason != TerminalReadFailed {
				t.Fatalf("blocked Next terminal = %+v %v", result.delivery, result.ok)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("blocked Next deadlocked terminal cleanup")
		}
		waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		if err := failedAttempt.Wait(waitCtx); err != nil || !failedAdapter.Accounting().Reconciles() || !failedAttempt.QueueAccounting().Reconciles() {
			t.Fatalf("terminal completion/accounting: err=%v adapter=%+v queue=%+v", err, failedAdapter.Accounting(), failedAttempt.QueueAccounting())
		}
	})
}

func waitForNextDrainOwner(t *testing.T, attempt *LiveAttempt) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !attempt.nextMu.TryLock() {
			return
		}
		attempt.nextMu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Next did not acquire the serialized drain")
}

func waitForQueuedFrames(t *testing.T, attempt *LiveAttempt, count uint64) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if attempt.QueueAccounting().FramesQueued == count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("queued frames did not reach %d: %+v", count, attempt.QueueAccounting())
}

func makeTwentyOneSymbols() []string {
	result := make([]string, 21)
	for index := range result {
		result[index] = fmt.Sprintf("S%02d", index)
	}
	return result
}

func TestPC5LiveOfflineComponentsOneThroughFiveCanonicalPath(t *testing.T) {
	t.Skip("superseded generic-success delivery expectation; post-handshake successes are informational and discarded")
	binding := component4TestBinding(t, []string{"AAA"})
	now := binding.SessionStart().Add(30 * time.Second)
	delay := time.Duration(0)
	state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 8, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		state.Close()
		waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := state.Wait(waitCtx); err != nil {
			t.Errorf("engine wait: %v", err)
		}
	}()
	admission, bindingCompletion := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || (<-bindingCompletion).Code != engine.DispositionBindingInstalled {
		t.Fatalf("binding admission = %s", admission)
	}

	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapterForBinding(t, socket, binding)
	attempt, started, err := adapter.Start(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("attempt engine result = %+v err=%v", result, err)
	}

	preAckRaw := "[" + aggregateLiveJSON("AAA", binding.SessionStart().Add(5*time.Second), `"dv":"1000.5"`) + "]"
	preAck, _ := NormalizeLiveFrame(LiveFrame{Binding: binding, ConnectionEpoch: attempt.Epoch(), FrameSequence: 1, ReceivedAt: now, Data: []byte(preAckRaw)}, nil, LiveNormalizationOptions{})
	preAckResult, err := DeliverToEngine(context.Background(), state, AdapterDelivery{Kind: DeliveryAggregate, Aggregate: preAck[0].Aggregate, Position: preAck[0].Position})
	if err != nil || preAckResult.AggregateDisposition.Code != engine.DispositionAggregateRejected {
		t.Fatalf("pre-ack aggregate = %+v err=%v", preAckResult, err)
	}

	handshake, err := attempt.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		result, err := DeliverToEngine(context.Background(), state, delivery)
		if err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake delivery %+v = %+v err=%v", delivery, result, err)
		}
	}

	write, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA"}})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := DeliverToEngine(context.Background(), state, write); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred {
		t.Fatalf("TQ write = %+v err=%v", result, err)
	}
	aggregate := aggregateLiveJSON("AAA", binding.SessionStart().Add(10*time.Second), `"dv":"1000.5"`)
	socket.send(socketMessageText, `[{"ev":"status","status":"failed","message":"ignored"},{"ev":"status","status":"failed"},`+aggregate+`]`)
	status, ok := attempt.nextForProof(context.Background())
	if !ok || status.Kind != DeliveryTQControlError {
		t.Fatalf("mixed TQ status = %+v", status)
	}
	if result, err := DeliverToEngine(context.Background(), state, status); err != nil || result.TQDisposition.Code != engine.DispositionTQRejected {
		t.Fatalf("mixed TQ engine = %+v err=%v", result, err)
	}
	duplicateStatus, ok := attempt.nextForProof(context.Background())
	if !ok || duplicateStatus.Kind != DeliveryTQControlError {
		t.Fatalf("duplicate failed status = %+v", duplicateStatus)
	}
	if result, err := DeliverToEngine(context.Background(), state, duplicateStatus); err != nil || result.TQDisposition.Code != engine.DispositionTQRejected {
		t.Fatalf("duplicate failed status engine = %+v err=%v", result, err)
	}
	liveAggregate, ok := attempt.nextForProof(context.Background())
	if !ok || liveAggregate.Kind != DeliveryAggregate || liveAggregate.Position.ArrayIndex != 2 {
		t.Fatalf("mixed aggregate = %+v", liveAggregate)
	}
	result, err := DeliverToEngine(context.Background(), state, liveAggregate)
	if err != nil || result.AggregateDisposition.Code != engine.DispositionAggregateInserted {
		t.Fatalf("canonical aggregate = %+v err=%v", result, err)
	}
	view := state.ObserveReplayDeterministic()
	if len(view.Canonical) != 1 || view.Canonical[0].Symbol != "AAA" || len(view.Canonical[0].Records) != 1 || view.Canonical[0].Records[0].Values.Close != 10.5 {
		t.Fatalf("canonical/evaluator view = %+v", view)
	}
	if err := attempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Cause: CloseControlledStop}); err != nil {
		t.Fatal(err)
	}
	terminal, ok := attempt.nextForProof(context.Background())
	if !ok || terminal.Kind != DeliveryTerminal {
		t.Fatalf("terminal = %+v", terminal)
	}
	if terminalResult, err := DeliverToEngine(context.Background(), state, terminal); err != nil || terminalResult.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("terminal engine result = %+v err=%v", terminalResult, err)
	}

	newSocket := newFakeLiveSocket()
	enqueueHandshake(newSocket)
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{newSocket}}
	open.CommandToken = 4
	newAttempt, newStarted, err := adapter.Start(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	if newAttempt.Epoch() <= attempt.Epoch() {
		t.Fatalf("epoch did not increase: %d -> %d", attempt.Epoch(), newAttempt.Epoch())
	}
	if startResult, err := DeliverToEngine(context.Background(), state, newStarted); err != nil || startResult.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("new attempt engine result = %+v err=%v", startResult, err)
	}
	newHandshake, err := newAttempt.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range newHandshake {
		if _, err := DeliverToEngine(context.Background(), state, delivery); err != nil {
			t.Fatal(err)
		}
	}
	oldRaw := "[" + aggregateLiveJSON("AAA", binding.SessionStart().Add(20*time.Second), `"dv":"1000.5"`) + "]"
	oldResults, _ := NormalizeLiveFrame(LiveFrame{Binding: binding, ConnectionEpoch: attempt.Epoch(), FrameSequence: 100, ReceivedAt: now, Data: []byte(oldRaw)}, nil, LiveNormalizationOptions{})
	stale, err := DeliverToEngine(context.Background(), state, AdapterDelivery{Kind: DeliveryAggregate, Aggregate: oldResults[0].Aggregate, Position: oldResults[0].Position})
	if err != nil || stale.AggregateDisposition.Code != engine.DispositionAggregateFenced || stale.AggregateDisposition.Reason != engine.ReasonStaleLiveEpoch {
		t.Fatalf("old epoch reached new canonical path = %+v err=%v", stale, err)
	}
	if err := newAttempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: newAttempt.Epoch(), CommandToken: 5, Cause: CloseControlledStop}); err != nil {
		t.Fatal(err)
	}
	if _, ok := newAttempt.nextForProof(context.Background()); !ok {
		t.Fatal("new epoch terminal missing")
	}
}

func testLiveAdapterForBinding(t *testing.T, socket *fakeLiveSocket, binding reference.Binding) (*LiveAdapter, OpenAggregateEpoch) {
	t.Helper()
	adapter, err := NewLiveAdapter(binding, LiveAdapterConfig{
		Endpoint: "wss://offline.invalid/stocks", Credential: "CREDENTIAL-MUST-NOT-ESCAPE",
		Queue: LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384},
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter.connector = &fakeLiveConnector{sockets: []*fakeLiveSocket{socket}}
	durations := OperationalDurations{Dial: time.Second, HandshakeStep: time.Second, HandshakeTotal: 4 * time.Second, HeartbeatInterval: time.Hour, HeartbeatDeadline: time.Second, Write: time.Second, Close: 20 * time.Millisecond}
	return adapter, OpenAggregateEpoch{BindingIdentity: binding.Identity(), CommandToken: 1, Durations: durations}
}
