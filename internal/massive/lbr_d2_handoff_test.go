package massive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// TestPLBRD2Handoff is the bounded-ring portion of P-LBR-D2-HANDOFF. It
// distinguishes decoded ownership and reserved causal markers from the former
// raw queue, and safe T/Q-only saturation from aggregate/control loss. Socket,
// heartbeat, command-boundary, terminal, and recovery branches remain covered
// by the focused transport/operations regressions selected with this proof.
func TestPLBRD2Handoff(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	received := binding.SessionStart().Add(10 * time.Second)
	queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 2, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes}, binding)

	enqueue := func(raw []byte) (queuedLiveFrame, FrameAdmissionReason) {
		t.Helper()
		sequence, reason, _ := queue.beginDecode(1, socketMessageText, received.Add(time.Duration(queue.next)*time.Microsecond), len(raw))
		if reason != FrameAdmitted {
			return queuedLiveFrame{}, reason
		}
		at := queue.lastReceipt
		batch := decodeLiveFrame(LiveFrame{Binding: binding, ConnectionEpoch: 1, FrameSequence: sequence, ReceivedAt: at, Data: raw}, nil, LiveNormalizationOptions{})
		raw[0] = ' '
		frame, reason, _ := queue.tryEnqueueDecoded(1, sequence, at, batch.EncodedBytes, batch)
		return frame, reason
	}

	window := binding.SessionStart().Add(time.Second)
	mixed := []byte(fmt.Sprintf(`[%s,{"ev":"T","sym":"AAA","x":4,"i":"t1","p":10,"s":1,"t":%d}]`, aggregateLiveJSON("AAA", window, ""), window.UnixMilli()))
	first, reason := enqueue(mixed)
	if reason != FrameAdmitted || first.batch.Len() != 2 || first.batch.EncodedBytes == 0 || first.batch.results[0].Kind != LiveResultAggregate || first.batch.results[1].Kind != LiveResultTrade {
		t.Fatalf("decoded mixed admission = frame=%+v reason=%s", first, reason)
	}
	trade := func(id string) []byte {
		return []byte(fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":%q,"p":10,"s":1,"t":%d}]`, id, window.UnixMilli()))
	}
	second, reason := enqueue(trade("t2"))
	if reason != FrameAdmitted || second.sequence != first.sequence+1 {
		t.Fatalf("second batch = %+v/%s", second, reason)
	}
	if _, reason = enqueue(trade("t3")); reason != FrameShedTQCapacity {
		t.Fatalf("T/Q-only saturation was not safely shed: %s", reason)
	}
	snapshot := queue.snapshot()
	if snapshot.FramesShedTQCapacity != 1 || snapshot.TQCapacityShedTrades != 1 || snapshot.TQCapacityMarkersQueued != 1 || snapshot.FramesRejectedCapacity != 0 || !snapshot.Reconciles() {
		t.Fatalf("T/Q shed accounting = %+v", snapshot)
	}
	for _, want := range []queuedLiveKind{queuedLiveDecodedBatch, queuedLiveDecodedBatch, queuedLiveTQCapacity} {
		entry, ok := queue.pop(context.Background())
		if !ok || entry.kind != want {
			t.Fatalf("causal FIFO want=%d entry=%+v ok=%t", want, entry, ok)
		}
		queue.complete(entry, false)
	}
	final := queue.snapshot()
	if final.FramesQueued != 0 || final.QueuedBytes != 0 || final.TQCapacityMarkersDispositioned != 1 || !final.Reconciles() {
		t.Fatalf("final decoded-ring accounting = %+v", final)
	}
}

// TestPLBRD2ProductionTopology excludes the superseded production handoffs.
// Compatibility helpers for historical proofs may remain in _test.go files,
// but no supported runtime path can retain raw frames, incrementally expand a
// frame, or admit live results through the engine's general FIFO.
func TestPLBRD2ProductionTopology(t *testing.T) {
	queueSource, err := os.ReadFile("live_queue.go")
	if err != nil {
		t.Fatal(err)
	}
	transportSource, err := os.ReadFile("live_transport.go")
	if err != nil {
		t.Fatal(err)
	}
	queueText, transportText := string(queueSource), string(transportSource)
	for _, removed := range []string{"queuedLive" + "Raw", "current" + "Frame", "current" + "Cursor", "newLiveFrame" + "Cursor(", "liveBatch" + "Cursor", "handshake" + "Deliveries"} {
		if strings.Contains(queueText, removed) || strings.Contains(transportText, removed) {
			t.Errorf("superseded live handoff remains reachable in production: %s", removed)
		}
	}
	if strings.Contains(transportText, "state."+"Admit") {
		t.Error("production Massive transport still admits live results through the engine FIFO")
	}
	if !strings.Contains(transportText, "state."+"ConsumeLiveBatch") {
		t.Error("production Massive transport does not use the decoded-batch engine handoff")
	}
	if strings.Contains(transportText, "func (a *LiveAttempt) Handshake(") || !strings.Contains(transportText, "HandshakeAndDeliver") {
		t.Error("production handshake can still expose or postpone per-result deliveries")
	}
	if !strings.Contains(queueText, "batch"+" DecodedBatch") {
		t.Error("bounded ingress ring does not own decoded batches")
	}
	production := newLiveFrameQueue(LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: MaximumLiveFrameBytes, TotalFrameBytes: MaximumLiveQueueBytes})
	if len(production.frames) != 4096 || production.decodedSlotCapacity()+liveMarkerReserveSlots != 4096 ||
		production.decodedByteCapacity()+liveMarkerReserveBytes != 64<<20 {
		t.Error("production decoded-batch capacity plus causal-marker reserve is not the authorized 4,096 entries/64 MiB")
	}
}

func TestPLBRD2DecodeCancellationAndSocketWriteSerialization(t *testing.T) {
	t.Run("gate close during decode reconciles", func(t *testing.T) {
		binding := component4TestBinding(t, []string{"AAA"})
		at := binding.SessionStart()
		queue := newLiveFrameQueue(LiveQueueConfig{FrameSlots: 2, MaxFrameBytes: 1024, TotalFrameBytes: 4096}, binding)
		sequence, reason, _ := queue.beginDecode(1, socketMessageText, at, 2)
		if reason != FrameAdmitted {
			t.Fatal(reason)
		}
		queue.closeGate()
		batch := decodeLiveFrame(LiveFrame{Binding: binding, ConnectionEpoch: 1, FrameSequence: sequence, ReceivedAt: at, Data: []byte("[]")}, nil, LiveNormalizationOptions{})
		if _, reason, accounting := queue.tryEnqueueDecoded(1, sequence, at, 2, batch); reason != FrameRejectedGate || accounting.FramesDecoding != 0 || !accounting.Reconciles() {
			t.Fatalf("decode cancellation reason=%s accounting=%+v", reason, accounting)
		}
	})

	t.Run("write and heartbeat ping share one critical section", func(t *testing.T) {
		socket := newFakeLiveSocket()
		socket.mu.Lock()
		socket.blockWrite, socket.writeStarted, socket.writeRelease = true, make(chan struct{}, 1), make(chan struct{})
		socket.blockPing, socket.pingStarted, socket.pingRelease = true, make(chan struct{}, 1), make(chan struct{})
		socket.mu.Unlock()
		attemptCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		attempt := &LiveAttempt{ctx: attemptCtx, connection: socket, durations: OperationalDurations{Write: time.Second}}
		writeDone, pingDone := make(chan error, 1), make(chan error, 1)
		go func() { writeDone <- attempt.write(context.Background(), []byte("auth")) }()
		select {
		case <-socket.writeStarted:
		case <-time.After(time.Second):
			t.Fatal("write did not enter socket")
		}
		go func() { pingDone <- attempt.ping(context.Background()) }()
		select {
		case <-socket.pingStarted:
			t.Fatal("ping entered socket while write was active")
		case <-time.After(10 * time.Millisecond):
		}
		close(socket.writeRelease)
		if err := <-writeDone; err != nil {
			t.Fatal(err)
		}
		select {
		case <-socket.pingStarted:
		case <-time.After(time.Second):
			t.Fatal("serialized ping did not enter socket")
		}
		close(socket.pingRelease)
		if err := <-pingDone; err != nil {
			t.Fatal(err)
		}
		socket.mu.Lock()
		maximum := socket.maximumIO
		socket.mu.Unlock()
		if maximum != 1 {
			t.Fatalf("maximum concurrent socket write/ping = %d", maximum)
		}
	})
}

func TestPLBRD2HandshakeAmbiguityPreservesIngressFirstCause(t *testing.T) {
	cases := []struct {
		name, phase string
		controls    uint64
	}{
		{name: "connected", phase: "connected", controls: 3},
		{name: "authentication", phase: "auth_success", controls: 4},
		{name: "aggregate subscription", phase: "success", controls: 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			socket := newFakeLiveSocket()
			if tc.phase != "connected" {
				socket.send(socketMessageText, `[{"ev":"status","status":"connected"}]`)
			}
			if tc.phase == "success" {
				socket.send(socketMessageText, `[{"ev":"status","status":"auth_success"}]`)
			}
			socket.send(socketMessageText, `[{"ev":"status","status":"`+tc.phase+`"},{"ev":"A","ev":"T"}]`)
			adapter, command := testLiveAdapter(t, socket, []string{"AAA"})
			binding := adapter.binding
			now := binding.SessionStart()
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
			baseline := state.ObserveOperational().LastEngineSequence
			attempt, started, err := adapter.Start(context.Background(), command)
			if err != nil {
				t.Fatal(err)
			}
			if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
				t.Fatalf("start=%+v err=%v", result, err)
			}
			if err := attempt.HandshakeAndDeliver(context.Background(), state); !errors.Is(err, errTransportFailed) {
				t.Fatalf("ambiguity handshake err=%v", err)
			}
			wait, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := attempt.Wait(wait); err != nil {
				t.Fatal(err)
			}
			terminal, ok := attempt.TerminalResult()
			attempt.mu.Lock()
			ingressIntegrity := attempt.terminal != nil && attempt.terminal.ingressIntegrity
			attempt.mu.Unlock()
			view := state.ObserveOperational()
			if !ok || terminal.Reason != TerminalIngressAmbiguity || terminal.Source != TerminalProtocol || !ingressIntegrity ||
				terminal.CausalPosition.ArrayIndex != 1 || view.Lifecycle != "suppressed" || view.Connection.Integrity != 1 ||
				view.Connection.Consumed != tc.controls || view.LastEngineSequence-baseline != tc.controls {
				t.Fatalf("terminal=%+v ingress=%t connection=%+v lifecycle=%s sequence_delta=%d", terminal, ingressIntegrity, view.Connection, view.Lifecycle, view.LastEngineSequence-baseline)
			}
		})
	}
}

func TestPLBRD2DialCancellationFirstCause(t *testing.T) {
	testCase := func(t *testing.T, connector *fakeLiveConnector, cancelBefore, cancelBlocked, cancelLifetime bool, want TerminalReason) {
		t.Helper()
		socket := newFakeLiveSocket()
		adapter, command := testLiveAdapter(t, socket, []string{"AAA"})
		adapter.connector = connector
		binding := adapter.binding
		now := binding.SessionStart()
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
		lifetime, cancelLifetimeContext := context.WithCancel(context.Background())
		defer cancelLifetimeContext()
		attempt, started, err := adapter.Start(lifetime, command)
		if err != nil {
			t.Fatal(err)
		}
		if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("start=%+v err=%v", result, err)
		}
		operation, cancelOperation := context.WithCancel(context.Background())
		if cancelBefore {
			if cancelLifetime {
				cancelLifetimeContext()
			} else {
				cancelOperation()
			}
		}
		handshakeDone := make(chan error, 1)
		go func() { handshakeDone <- attempt.HandshakeAndDeliver(operation, state) }()
		if cancelBlocked {
			select {
			case <-connector.dialStarted:
			case <-time.After(time.Second):
				t.Fatal("dial did not block")
			}
			if cancelLifetime {
				cancelLifetimeContext()
			} else {
				cancelOperation()
			}
		}
		if err := <-handshakeDone; !errors.Is(err, errTransportFailed) {
			t.Fatalf("handshake err=%v", err)
		}
		cancelOperation()
		wait, cancelWait := context.WithTimeout(context.Background(), time.Second)
		defer cancelWait()
		if err := attempt.Wait(wait); err != nil {
			t.Fatal(err)
		}
		terminal, ok := attempt.TerminalResult()
		if !ok || terminal.Source != TerminalReader || terminal.Reason != want {
			t.Fatalf("terminal=%+v ok=%t", terminal, ok)
		}
		result, delivered, err := attempt.DeliverNextToEngine(context.Background(), state)
		if err != nil || !delivered || result.Terminal == nil || result.Terminal.Reason != want ||
			(result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			t.Fatalf("engine terminal result=%+v delivered=%t err=%v", result, delivered, err)
		}
		if !attempt.QueueAccounting().Reconciles() || !adapter.Accounting().Reconciles() || !state.ObserveOperational().Admissions.Reconciles(0) {
			t.Fatalf("accounting queue=%+v adapter=%+v engine=%+v", attempt.QueueAccounting(), adapter.Accounting(), state.ObserveOperational().Admissions)
		}
	}

	t.Run("blocked dial operation cancellation", func(t *testing.T) {
		testCase(t, &fakeLiveConnector{dialStarted: make(chan struct{}, 1), dialRelease: make(chan struct{})}, false, true, false, TerminalContextCanceled)
	})
	t.Run("blocked dial lifetime cancellation", func(t *testing.T) {
		testCase(t, &fakeLiveConnector{dialStarted: make(chan struct{}, 1), dialRelease: make(chan struct{})}, false, true, true, TerminalContextCanceled)
	})
	t.Run("already canceled operation", func(t *testing.T) {
		testCase(t, &fakeLiveConnector{dialStarted: make(chan struct{}, 1), dialRelease: make(chan struct{})}, true, false, false, TerminalContextCanceled)
	})
	t.Run("genuine dial failure", func(t *testing.T) {
		testCase(t, &fakeLiveConnector{fail: true}, false, false, false, TerminalDialFailed)
	})
}
