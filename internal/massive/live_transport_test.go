package massive

import (
	"context"
	"errors"
	"fmt"
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

func (s *fakeLiveSocket) Ping(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pingError
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
	if binary, ok := attempt.Next(context.Background()); !ok || binary.Kind != DeliveryAggregate {
		t.Fatalf("ordinary binary read = %+v %v", binary, ok)
	}
	socket.failRead()
	terminal, ok := attempt.Next(context.Background())
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
		terminal, ok := failedAttempt.Next(context.Background())
		if !ok || terminal.Terminal.Reason != TerminalStatusAmbiguous {
			t.Fatalf("auth terminal = %+v", terminal)
		}

		dialAdapter, dialCommand := testLiveAdapter(t, newFakeLiveSocket(), []string{"AAA"})
		dialAdapter.connector = &fakeLiveConnector{fail: true}
		dialAttempt, _, _ := dialAdapter.Start(context.Background(), dialCommand)
		if _, err := dialAttempt.Handshake(context.Background()); !errors.Is(err, errTransportFailed) {
			t.Fatalf("dial failure = %v", err)
		}
		terminal, ok = dialAttempt.Next(context.Background())
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
		terminal, ok := heartbeatAttempt.Next(context.Background())
		elapsed := time.Since(startedAt)
		if !ok || terminal.Terminal.Source != TerminalHeartbeat || terminal.Terminal.Reason != TerminalHeartbeatFailureUnclassified || elapsed > 200*time.Millisecond {
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
		terminal, ok := racingAttempt.Next(context.Background())
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
			terminal, ok = unsupportedAttempt.Next(context.Background())
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
		terminal, ok = deadlineAttempt.Next(context.Background())
		if !ok || terminal.Terminal.Reason != TerminalHandshakeDeadline {
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
		if terminal, ok := totalAttempt.Next(context.Background()); !ok || terminal.Terminal.Reason != TerminalHandshakeDeadline {
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
		if terminal, ok := idleAttempt.Next(context.Background()); !ok || terminal.Kind != DeliveryTerminal || !idleAdapter.Accounting().Reconciles() {
			t.Fatalf("unawaited terminal/accounting = %+v %v %+v", terminal, ok, idleAdapter.Accounting())
		}
	})

	t.Run("later ambiguity preserves the delivered causal prefix", func(t *testing.T) {
		prefixSocket := newFakeLiveSocket()
		enqueueHandshake(prefixSocket)
		prefixAdapter, command := testLiveAdapter(t, prefixSocket, []string{"AAA"})
		prefixAttempt, _, _ := startHandshake(t, prefixAdapter, command)
		start := prefixAdapter.binding.SessionStart().Add(10 * time.Second)
		prefixSocket.send(socketMessageText, "["+aggregateLiveJSON("AAA", start, `"v":1,"z":1`)+`,{"ev":"unknown"}]`)
		first, ok := prefixAttempt.Next(context.Background())
		if !ok || first.Kind != DeliveryAggregate || first.Position.ArrayIndex != 0 {
			t.Fatalf("causal prefix = %+v %v", first, ok)
		}
		ingress, ok := prefixAttempt.Next(context.Background())
		if !ok || ingress.Control.Kind != engine.IngressIntegrityFailure || ingress.Position.ArrayIndex != 1 {
			t.Fatalf("ingress fact = %+v %v", ingress, ok)
		}
		terminal, ok := prefixAttempt.Next(context.Background())
		if !ok || terminal.Kind != DeliveryTerminal || !prefixAdapter.Accounting().Reconciles() || !prefixAttempt.QueueAccounting().Reconciles() {
			t.Fatalf("prefix terminal/accounting = %+v %v adapter=%+v queue=%+v", terminal, ok, prefixAdapter.Accounting(), prefixAttempt.QueueAccounting())
		}
	})
}

func TestPC5CommandWriteAcknowledgementLinearization(t *testing.T) {
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapter(t, socket, []string{"AAA", "BBB"})
	attempt, _, _ := startHandshake(t, adapter, open)

	socket.mu.Lock()
	socket.blockWrite = true
	socket.writeStarted = make(chan struct{}, 1)
	socket.writeRelease = make(chan struct{})
	socket.mu.Unlock()
	type commandResult struct {
		delivery AdapterDelivery
		err      error
	}
	result := make(chan commandResult, 1)
	go func() {
		delivery, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA", "BBB"}})
		result <- commandResult{delivery: delivery, err: err}
	}()
	<-socket.writeStarted
	socket.send(socketMessageText, `[{"ev":"status","status":"success"},{"ev":"status","status":"success"},{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`)
	ackResult := make(chan AdapterDelivery, 1)
	go func() {
		ack, _ := attempt.Next(context.Background())
		ackResult <- ack
	}()
	select {
	case premature := <-result:
		t.Fatalf("write returned before release: %+v", premature)
	default:
	}
	select {
	case premature := <-ackResult:
		t.Fatalf("ack completed before write: %+v", premature)
	case <-time.After(5 * time.Millisecond):
	}
	close(socket.writeRelease)
	written := <-result
	if written.err != nil || written.delivery.Control.Outcome != engine.ControlSucceeded {
		t.Fatalf("write result = %+v", written)
	}
	ack := <-ackResult
	if ack.Control.Kind != engine.TradeQuoteSubscriptionResult || ack.Control.Outcome != engine.ControlSucceeded || ack.ExpectedStatusCount != 4 || ack.ObservedStatusCount != 4 {
		t.Fatalf("ack result = %+v", ack)
	}
	writes := socket.written()
	if writes[len(writes)-1] != `{"action":"subscribe","params":"T.AAA,Q.AAA,T.BBB,Q.BBB"}` {
		t.Fatalf("dynamic command = %q", writes[len(writes)-1])
	}

	t.Run("shape token and stale close reject before I/O", func(t *testing.T) {
		before := len(socket.written())
		bad := []ChangeTQCommand{
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch() + 1, CommandToken: 3, Action: TQSubscribe, Symbols: []string{"AAA"}},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Action: TQSubscribe, Symbols: []string{"BBB", "AAA"}},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA"}},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Action: TQSubscribe},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Action: TQSubscribe, Symbols: []string{"AAA", "AAA"}},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Action: TQSubscribe, Symbols: []string{"NOT-BOUND"}},
			{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 3, Action: TQSubscribe, Symbols: makeTwentyOneSymbols()},
		}
		for _, command := range bad {
			if _, err := attempt.ChangeTQ(context.Background(), command); !errors.Is(err, errCommand) {
				t.Fatalf("bad command accepted: %+v err=%v", command, err)
			}
		}
		if err := attempt.Close(CloseEpochCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch() + 1, CommandToken: 3, Cause: CloseControlledStop}); !errors.Is(err, errCommand) {
			t.Fatalf("stale close = %v", err)
		}
		if len(socket.written()) != before {
			t.Fatal("invalid command performed I/O")
		}
	})

	t.Run("partial extra and ack-before-write-failure never complete", func(t *testing.T) {
		for _, test := range []struct {
			token uint64
			data  string
			want  int
		}{
			{3, `[{"ev":"status","status":"success"}]`, 1},
			{4, `[{"ev":"status","status":"success"},{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`, 3},
		} {
			write, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: test.token, Action: TQSubscribe, Symbols: []string{"AAA"}})
			if err != nil || write.Control.Outcome != engine.ControlSucceeded {
				t.Fatalf("write %d = %+v %v", test.token, write, err)
			}
			socket.send(socketMessageText, test.data)
			ack, ok := attempt.Next(context.Background())
			if !ok || ack.Control.Outcome != engine.ControlAmbiguous || ack.ExpectedStatusCount != 2 || ack.ObservedStatusCount != test.want {
				t.Fatalf("ambiguous count %d = %+v", test.token, ack)
			}
		}

		socket.mu.Lock()
		socket.blockWrite = true
		socket.writeError = errors.New("provider prose CREDENTIAL-MUST-NOT-ESCAPE")
		socket.writeStarted = make(chan struct{}, 1)
		socket.writeRelease = make(chan struct{})
		socket.mu.Unlock()
		failed := make(chan commandResult, 1)
		go func() {
			delivery, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 5, Action: TQUnsubscribe, Symbols: []string{"AAA"}})
			failed <- commandResult{delivery: delivery, err: err}
		}()
		<-socket.writeStarted
		socket.send(socketMessageText, `[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`)
		close(socket.writeRelease)
		failure := <-failed
		if !errors.Is(failure.err, errCommandWrite) || failure.delivery.Control.Outcome != engine.ControlFailed || strings.Contains(fmt.Sprint(failure.err), "CREDENTIAL") {
			t.Fatalf("write failure = %+v", failure)
		}
		if accounting := adapter.Accounting(); accounting.CommandsFailed != 1 || !accounting.Reconciles() {
			t.Fatalf("write failure accounting = %+v", accounting)
		}
		ack, ok := attempt.Next(context.Background())
		if !ok || ack.Control.Outcome != engine.ControlAmbiguous {
			t.Fatalf("ack after failed write = %+v", ack)
		}
		socket.mu.Lock()
		socket.blockWrite = false
		socket.writeError = nil
		socket.mu.Unlock()
	})

	attempt.durations.HandshakeStep = 3 * time.Millisecond
	if _, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 6, Action: TQSubscribe, Symbols: []string{"AAA"}}); err != nil {
		t.Fatal(err)
	}
	timedOut, ok := attempt.Next(context.Background())
	if !ok || timedOut.Control.Outcome != engine.ControlAmbiguous || timedOut.Control.Position != (engine.LivePosition{}) {
		t.Fatalf("ack deadline fabricated success/position = %+v", timedOut)
	}
	attempt.durations.HandshakeStep = time.Second
	if _, err := attempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: open.BindingIdentity, ConnectionEpoch: attempt.Epoch(), CommandToken: 7, Action: TQSubscribe, Symbols: []string{"AAA"}}); err != nil {
		t.Fatal(err)
	}
	socket.failRead()
	if terminal, ok := attempt.Next(context.Background()); !ok || terminal.Kind != DeliveryTerminal || terminal.Control.Outcome != engine.ControlFailed || terminal.Terminal.PendingCommandToken != 7 || terminal.Terminal.PendingExpectedStatusCount != 2 || terminal.Terminal.PendingCommandOutcome != engine.ControlAmbiguous {
		t.Fatalf("loss while pending = %+v %v", terminal, ok)
	}
	if accounting := adapter.Accounting(); !accounting.Reconciles() || accounting.CommandsStarted != 7 || accounting.CommandsPendingWrite != 0 || accounting.CommandsPendingAck != 0 {
		t.Fatalf("command accounting = %+v", accounting)
	}

	t.Run("completed acknowledgement cannot be reclassified by terminal cleanup", func(t *testing.T) {
		raceSocket := newFakeLiveSocket()
		enqueueHandshake(raceSocket)
		raceAdapter, command := testLiveAdapter(t, raceSocket, []string{"AAA"})
		raceAttempt, _, _ := startHandshake(t, raceAdapter, command)
		if _, err := raceAttempt.ChangeTQ(context.Background(), ChangeTQCommand{BindingIdentity: command.BindingIdentity, ConnectionEpoch: raceAttempt.Epoch(), CommandToken: 2, Action: TQSubscribe, Symbols: []string{"AAA"}}); err != nil {
			t.Fatal(err)
		}
		raceSocket.send(socketMessageText, `[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`)
		waitForQueuedFrames(t, raceAttempt, 1)
		// Hold the adapter accounting lock so Next can atomically detach the
		// pending command but cannot yet record its acknowledgement. Terminal
		// cleanup then overlaps the exact detach-to-account interval that must
		// not classify the command a second time.
		raceAdapter.mu.Lock()
		type deliveryResult struct {
			delivery AdapterDelivery
			ok       bool
		}
		ackDone := make(chan deliveryResult, 1)
		go func() {
			delivery, ok := raceAttempt.Next(context.Background())
			ackDone <- deliveryResult{delivery: delivery, ok: ok}
		}()
		deadline := time.Now().Add(200 * time.Millisecond)
		for {
			raceAttempt.mu.Lock()
			detached := raceAttempt.pending == nil
			raceAttempt.mu.Unlock()
			if detached {
				break
			}
			if time.Now().After(deadline) {
				raceAdapter.mu.Unlock()
				t.Fatal("pending command was not detached")
			}
			time.Sleep(time.Millisecond)
		}
		raceAttempt.triggerTerminal(TerminalReader, TerminalReadFailed, 0, false)
		raceAdapter.mu.Unlock()
		ackResult := <-ackDone
		ack, ok := ackResult.delivery, ackResult.ok
		if !ok || ack.Control.Outcome != engine.ControlSucceeded {
			t.Fatalf("race acknowledgement = %+v %v", ack, ok)
		}
		if terminal, ok := raceAttempt.Next(context.Background()); !ok || terminal.Kind != DeliveryTerminal || !raceAdapter.Accounting().Reconciles() {
			t.Fatalf("race terminal/accounting = %+v %v %+v", terminal, ok, raceAdapter.Accounting())
		}
		accounting := raceAdapter.Accounting()
		if accounting.CommandsStarted != 2 || accounting.CommandsAcknowledged != 2 || accounting.CommandsCanceledOrFenced != 0 {
			t.Fatalf("completed acknowledgement was reclassified: %+v", accounting)
		}
	})
}

func TestPC5BoundRawFIFOAccountingAndDrain(t *testing.T) {
	if validateLiveQueueConfig(LiveQueueConfig{}) || validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 513, MaxFrameBytes: 1, TotalFrameBytes: 1}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: MaximumLiveFrameBytes + 1, TotalFrameBytes: MaximumLiveQueueBytes}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 2, TotalFrameBytes: 1}) ||
		validateLiveQueueConfig(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 1, TotalFrameBytes: MaximumLiveQueueBytes + 1}) ||
		!validateLiveQueueConfig(LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes}) {
		t.Fatal("queue ceiling validation mismatch")
	}
	queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 2, MaxFrameBytes: 4, TotalFrameBytes: 6})
	at := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	source := []byte("abc")
	first, reason := queue.tryEnqueue(9, socketMessageText, at, source)
	if reason != FrameAdmitted {
		t.Fatal(reason)
	}
	source[0] = 'z'
	second, reason := queue.tryEnqueue(9, socketMessageBinary, at, []byte("def"))
	if reason != FrameAdmitted || second.sequence != first.sequence+1 {
		t.Fatalf("second = %+v %s", second, reason)
	}
	if _, reason = queue.tryEnqueue(9, socketMessageText, at, []byte("x")); reason != FrameRejectedCapacity {
		t.Fatalf("count capacity = %s", reason)
	}
	if string(queue.frames[0].data) != "abc" {
		t.Fatalf("copy-on-admission = %q", queue.frames[0].data)
	}
	if _, reason = queue.tryEnqueue(9, socketMessageText, at.Add(-time.Nanosecond), []byte("x")); reason != FrameRejectedReceipt {
		t.Fatalf("receipt regression = %s", reason)
	}
	if _, reason = queue.tryEnqueue(9, socketMessageText, at, []byte("12345")); reason != FrameRejectedOversize {
		t.Fatalf("oversize = %s", reason)
	}
	queue.closeGate()
	if _, reason = queue.tryEnqueue(9, socketMessageText, at, []byte("x")); reason != FrameRejectedGate {
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
	if !accounting.Reconciles() || accounting.FramesRead != 6 || accounting.FramesAdmitted != 2 || accounting.FramesDispositioned != 1 || accounting.FramesFenced != 1 || accounting.QueuedBytes != 0 {
		t.Fatalf("accounting = %+v", accounting)
	}
	canceledQueue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 1, MaxFrameBytes: 4, TotalFrameBytes: 4})
	_, _ = canceledQueue.tryEnqueue(1, socketMessageText, at, []byte("x"))
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := canceledQueue.enqueueTerminal(canceledCtx, 1, at); ok || !canceledQueue.snapshot().Reconciles() {
		t.Fatalf("canceled terminal linkage mutated accounting: %+v", canceledQueue.snapshot())
	}

	t.Run("byte-only saturation is atomic before slot saturation", func(t *testing.T) {
		byteQueue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 3, MaxFrameBytes: 4, TotalFrameBytes: 5})
		if _, reason := byteQueue.tryEnqueue(1, socketMessageText, at, []byte("abc")); reason != FrameAdmitted {
			t.Fatal(reason)
		}
		before := byteQueue.snapshot()
		if _, reason := byteQueue.tryEnqueue(1, socketMessageText, at, []byte("def")); reason != FrameRejectedCapacity {
			t.Fatalf("byte saturation = %s", reason)
		}
		after := byteQueue.snapshot()
		if after.FramesQueued != before.FramesQueued || after.QueuedBytes != before.QueuedBytes || !after.Reconciles() {
			t.Fatalf("byte rejection mutated queue: before=%+v after=%+v", before, after)
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
	first.triggerTerminal(TerminalHeartbeat, TerminalHeartbeatFailureUnclassified, 0, false)
	first.triggerTerminal(TerminalReader, TerminalReadFailed, 0, false)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	if err := first.Wait(waitCtx); err != nil {
		t.Fatalf("self-contained first cleanup: %v", err)
	}
	waitCancel()
	if !firstSocket.isClosed() || first.QueueAccounting().FramesFenced != 1 {
		t.Fatalf("first cleanup did not close/fence: closed=%v accounting=%+v", firstSocket.isClosed(), first.QueueAccounting())
	}
	terminal, ok := first.Next(context.Background())
	if !ok || terminal.Terminal.Source != TerminalHeartbeat || terminal.Terminal.Reason != TerminalHeartbeatFailureUnclassified {
		t.Fatalf("first cause = %+v", terminal)
	}
	if _, ok := first.Next(context.Background()); ok {
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
	if closed, ok := second.Next(context.Background()); !ok || closed.Terminal.Source != TerminalEngineClose || closed.Terminal.CloseCause != CloseControlledStop {
		t.Fatalf("controlled close = %+v", closed)
	}
	if err := second.Close(closeCommand); err != nil {
		t.Fatalf("post-terminal idempotent close = %v", err)
	}
	firstSocket.send(socketMessageText, `[{"ev":"A"}]`)
	if _, ok := first.Next(context.Background()); ok {
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
	rootTerminal, ok := third.Next(context.Background())
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
			delivery, ok := orderedAttempt.Next(context.Background())
			if ok {
				positions <- callerPosition{caller: 1, position: delivery.Position}
			}
		}()
		waitForNextDrainOwner(t, orderedAttempt)
		callers.Add(1)
		go func() {
			defer callers.Done()
			delivery, ok := orderedAttempt.Next(context.Background())
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
		if _, ok := orderedAttempt.Next(context.Background()); !ok {
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
			delivery, ok := failedAttempt.Next(context.Background())
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
	status, ok := attempt.Next(context.Background())
	if !ok || status.Control.Kind != engine.TradeQuoteSubscriptionResult || status.Control.Outcome != engine.ControlFailed {
		t.Fatalf("mixed TQ status = %+v", status)
	}
	if result, err := DeliverToEngine(context.Background(), state, status); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred {
		t.Fatalf("mixed TQ engine = %+v err=%v", result, err)
	}
	liveAggregate, ok := attempt.Next(context.Background())
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
	terminal, ok := attempt.Next(context.Background())
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
	if _, ok := newAttempt.Next(context.Background()); !ok {
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
