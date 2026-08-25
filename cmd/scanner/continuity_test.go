package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

func TestTerminalSinksLatchOutputFailuresWithoutRecursiveShutdownPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stderr bytes.Buffer
	stdout, errorOutput := newTerminalSinks(failingWriter{}, &stderr)
	renderer := newOperatorRenderer(stdout, errorOutput)
	if err := renderer.Render(warmupOperatorSample(time.Now().UTC()), true); err != nil {
		t.Fatalf("initial output error=%v", err)
	}
	if !stdout.Failed() || errorOutput.Failed() {
		t.Fatalf("sink health stdout=%t stderr=%t", stdout.Failed(), errorOutput.Failed())
	}
	if got := strings.Count(stderr.String(), "terminal stdout became unavailable"); got != 1 {
		t.Fatalf("stdout failure report count=%d output=%q", got, stderr.String())
	}
	if err := renderer.Render(warmupOperatorSample(time.Now().UTC().Add(time.Second)), true); err != nil {
		t.Fatalf("latched output error=%v", err)
	}
	if got := strings.Count(stderr.String(), "terminal stdout became unavailable"); got != 1 {
		t.Fatalf("latched stdout reported repeatedly count=%d output=%q", got, stderr.String())
	}
	if ctx.Err() != nil {
		t.Fatal("terminal output failure canceled the scanner context")
	}

	var healthyStdout bytes.Buffer
	_, failedStderr := newTerminalSinks(&healthyStdout, failingWriter{})
	if err := encodeSnapshotMappingFailure(json.NewEncoder(failedStderr), snapshotapi.MappingFailure{Invariant: "capture_coherence"}); err != nil {
		t.Fatalf("mapping diagnostic failure escaped=%v", err)
	}
	if !failedStderr.Failed() || strings.Count(healthyStdout.String(), "terminal stderr became unavailable") != 1 {
		t.Fatalf("stderr failure did not latch/report once: failed=%t stdout=%q", failedStderr.Failed(), healthyStdout.String())
	}
}

func TestBrokenPipeOutputCannotTerminateScannerProcess(t *testing.T) {
	if os.Getenv("SCANNER_CONTINUITY_BROKEN_PIPE_CHILD") == "1" {
		installScannerSignalHandling()
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		original, err := syscall.Dup(int(os.Stdout.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		defer syscall.Close(original)
		_ = reader.Close()
		if err := syscall.Dup2(int(writer.Fd()), int(os.Stdout.Fd())); err != nil {
			t.Fatal(err)
		}
		_, err = fmt.Fprint(os.Stdout, "operator status\n")
		if restoreErr := syscall.Dup2(original, int(os.Stdout.Fd())); restoreErr != nil {
			t.Fatal(restoreErr)
		}
		_ = writer.Close()
		if !errors.Is(err, syscall.EPIPE) {
			t.Fatalf("broken pipe write error=%v", err)
		}
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestBrokenPipeOutputCannotTerminateScannerProcess$")
	command.Env = append(os.Environ(), "SCANNER_CONTINUITY_BROKEN_PIPE_CHILD=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("scanner subprocess terminated on broken pipe: %v output=%s", err, output)
	}
}

func TestSuperviseLiveContainsOutputAndMappingDiagnosticsWhileRuntimeAndAPIContinue(t *testing.T) {
	runtime, binding, clockNanos := newMutableContinuityRuntime(t)
	now := binding.SessionStart()
	proof, stopProof := context.WithTimeout(context.Background(), time.Second)
	defer stopProof()
	continuityAcknowledgeAggregate(t, proof, runtime.Engine(), binding, 1, now)
	continuityCompleteHydration(t, proof, runtime.Engine(), binding, engine.HydrationFreshBootstrap, 1, now)
	beforeCapture, err := runtime.CaptureSnapshot()
	if err != nil {
		shutdownContinuityRuntime(t, runtime)
		t.Fatal(err)
	}
	before, valid := operations.InspectSnapshotCapture(beforeCapture)
	if !valid {
		shutdownContinuityRuntime(t, runtime)
		t.Fatal("initial runtime capture was invalid")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := newFakeSnapshotServer()
	supervisor, err := newAPISupervisor(fakeCaptureSource{}, snapshotapi.ServerConfig{Address: "127.0.0.1:8080"}, func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error) { return server, nil }, nil)
	if err != nil {
		shutdownContinuityRuntime(t, runtime)
		t.Fatal(err)
	}
	stdout, stderr := newTerminalSinks(failingWriter{}, failingWriter{})
	renderer := newOperatorRenderer(stdout, stderr)
	ticks := make(chan time.Time, 2)
	done := make(chan error)
	captured := make(chan struct{}, 2)
	mapped := make(chan struct{}, 1)
	shutdowns := 0
	captures := 0
	result := make(chan error, 1)
	go func() {
		result <- superviseLive(liveSupervisorConfig{
			context: ctx, api: supervisor, done: done, ticks: ticks,
			capture: func() (liveOperatorSample, error) {
				captures++
				captured <- struct{}{}
				if captures == 1 {
					return warmupOperatorSample(time.Now().UTC()), nil
				}
				return captureLiveOperatorSample(runtime)
			},
			render: renderer.Render,
			record: func(*operations.IngressIncident, bool) error { return nil },
			encodeMapping: func(failure snapshotapi.MappingFailure) error {
				mapped <- struct{}{}
				return encodeSnapshotMappingFailure(json.NewEncoder(stderr), failure)
			},
			shutdown: func(bool) error {
				shutdowns++
				apiCtx, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				return errors.Join(supervisor.Shutdown(apiCtx), runtime.Shutdown(apiCtx))
			},
		})
	}()
	ticks <- time.Now()
	<-captured
	clockNanos.Store(now.Add(time.Second).UnixNano())
	input := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: "AAA",
		WindowStart: now, WindowEnd: now.Add(time.Second), Values: engine.AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 100, VWAP: 12, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
		DeliveryTime: now.Add(time.Second), Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 2, ArrayIndex: 1}}
	if admission, completion := runtime.Engine().AdmitAggregate(proof, input); admission != engine.AdmissionAdmitted {
		t.Fatalf("aggregate after terminal output failure admission=%s", admission)
	} else if result := continuityAwait(t, proof, completion); result.Code != engine.DispositionAggregateInserted {
		t.Fatalf("aggregate after terminal output failure result=%+v", result)
	}
	if admission, completion := runtime.Engine().AdmitTimer(proof); admission != engine.AdmissionAdmitted || continuityAwait(t, proof, completion).Code != engine.DispositionTimerApplied {
		t.Fatalf("aggregate evaluation after terminal output failure admission=%s", admission)
	}
	server.mapping <- snapshotapi.MappingFailure{Invariant: "capture_coherence"}
	<-mapped
	ticks <- time.Now()
	<-captured
	afterCapture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	after, valid := operations.InspectSnapshotCapture(afterCapture)
	if !stdout.Failed() || !stderr.Failed() || !runtime.Status().ProcessLive || supervisor.Done() != server.done || !valid || after.Engine.Publication.LastEngineSequence <= before.Engine.Publication.LastEngineSequence {
		t.Fatalf("diagnostic containment failed stdout=%t stderr=%t processLive=%t apiDone=%v", stdout.Failed(), stderr.Failed(), runtime.Status().ProcessLive, supervisor.Done())
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) || shutdowns != 1 {
		t.Fatalf("supervisor stop err=%v shutdowns=%d", err, shutdowns)
	}
}

func TestSuperviseLiveRuntimeTerminalStillUsesShutdownPath(t *testing.T) {
	server := newFakeSnapshotServer()
	supervisor, err := newAPISupervisor(fakeCaptureSource{}, snapshotapi.ServerConfig{Address: "127.0.0.1:8080"}, func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error) { return server, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	done <- errors.New("engine terminal")
	shutdowns := 0
	err = superviseLive(liveSupervisorConfig{
		context: context.Background(), api: supervisor, done: done,
		capture: func() (liveOperatorSample, error) { return liveOperatorSample{}, nil }, render: func(liveOperatorSample, bool) error { return nil },
		record: func(*operations.IngressIncident, bool) error { return nil }, encodeMapping: func(snapshotapi.MappingFailure) error { return nil },
		shutdown: func(liveJoined bool) error {
			if !liveJoined {
				t.Fatal("runtime terminal did not mark live composition joined")
			}
			shutdowns++
			shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			return supervisor.Shutdown(shutdown)
		},
	})
	if err == nil || !strings.Contains(err.Error(), "engine terminal") || shutdowns != 1 {
		t.Fatalf("runtime terminal err=%v shutdowns=%d", err, shutdowns)
	}
}

func TestAPISupervisorReplacementServesSameRuntimePublication(t *testing.T) {
	runtime := newContinuityRuntime(t)
	defer shutdownContinuityRuntime(t, runtime)
	timers := &manualAPITimers{}
	supervisor, err := newAPISupervisor(runtime, snapshotapi.ServerConfig{Address: "127.0.0.1:0"}, func(source snapshotapi.CaptureSource, config snapshotapi.ServerConfig) (snapshotServer, error) {
		return snapshotapi.Listen(source, config)
	}, timers.new)
	if err != nil {
		t.Fatal(err)
	}
	firstResponse := continuitySnapshot(t, supervisor.server.Address())
	first := supervisor.server.(*snapshotapi.Server)
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := first.Shutdown(shutdown); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	<-supervisor.Done()
	supervisor.UnexpectedTermination()
	timers.fire(0)
	if err := supervisor.RestartDue(); err != nil {
		t.Fatal(err)
	}
	secondResponse := continuitySnapshot(t, supervisor.server.Address())
	if firstResponse.SchemaVersion != secondResponse.SchemaVersion || firstResponse.Publication.ID != secondResponse.Publication.ID || firstResponse.Publication.BindingIdentity != secondResponse.Publication.BindingIdentity || secondResponse.Sample.ID <= firstResponse.Sample.ID {
		t.Fatalf("replacement did not preserve publication/source continuity: first=%+v second=%+v", firstResponse, secondResponse)
	}
	shutdown, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestInitialAPIBindConflictFailsBeforeLiveWorkStarts(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	startedLiveWork := false
	api, done, err := startLiveAfterAPI(func() (*apiSupervisor, error) {
		return newAPISupervisor(fakeCaptureSource{}, snapshotapi.ServerConfig{Address: listener.Addr().String()}, func(source snapshotapi.CaptureSource, config snapshotapi.ServerConfig) (snapshotServer, error) {
			return snapshotapi.Listen(source, config)
		}, nil)
	}, func() <-chan error {
		startedLiveWork = true
		return make(chan error)
	})
	if err == nil || api != nil || done != nil || startedLiveWork {
		t.Fatalf("API bind conflict err=%v liveStarted=%t", err, startedLiveWork)
	}
}

func newContinuityRuntime(t *testing.T) *operations.Runtime {
	t.Helper()
	binding := scannerTestBinding(t)
	now := binding.SessionStart()
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	startup, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runtime, err := operations.New(startup, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func newMutableContinuityRuntime(t *testing.T) (*operations.Runtime, reference.Binding, *atomic.Int64) {
	t.Helper()
	binding := scannerTestBinding(t)
	clockNanos := &atomic.Int64{}
	clockNanos.Store(binding.SessionStart().UnixNano())
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	startup, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runtime, err := operations.New(startup, binding, config, func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	return runtime, binding, clockNanos
}

func shutdownContinuityRuntime(t *testing.T, runtime *operations.Runtime) {
	t.Helper()
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

type continuitySnapshotView struct {
	SchemaVersion string `json:"schema_version"`
	Publication   struct {
		ID              string `json:"id"`
		BindingIdentity string `json:"binding_identity"`
	} `json:"publication"`
	Sample struct {
		ID string `json:"id"`
	} `json:"sample"`
}

func continuitySnapshot(t *testing.T, address string) continuitySnapshotView {
	t.Helper()
	response, err := (&http.Client{Timeout: time.Second}).Get("http://" + address + "/api/v2/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("snapshot status=%d", response.StatusCode)
	}
	var payload continuitySnapshotView
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestAPISupervisorReplacesOneGenerationUsingTheSameSource(t *testing.T) {
	source := fakeCaptureSource{}
	first, second := newFakeSnapshotServer(), newFakeSnapshotServer()
	first.address = "127.0.0.1:43001"
	second.address = first.address
	timers := &manualAPITimers{}
	var sources []snapshotapi.CaptureSource
	var configs []snapshotapi.ServerConfig
	factory := func(got snapshotapi.CaptureSource, config snapshotapi.ServerConfig) (snapshotServer, error) {
		sources, configs = append(sources, got), append(configs, config)
		if len(sources) == 1 {
			return first, nil
		}
		return second, nil
	}
	config := snapshotapi.ServerConfig{Address: "127.0.0.1:8080", AllowedOrigins: []string{"http://127.0.0.1:4173"}}
	supervisor, err := newAPISupervisor(source, config, factory, timers.new)
	if err != nil {
		t.Fatal(err)
	}
	first.finish(errors.New("listener failed"))
	<-supervisor.Done()
	supervisor.UnexpectedTermination()
	if len(sources) != 1 || len(timers.delays) != 1 || timers.delays[0] != 250*time.Millisecond {
		t.Fatalf("replacement started before first backoff: sources=%d delays=%v", len(sources), timers.delays)
	}
	timers.fire(0)
	if err := supervisor.RestartDue(); err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 || sources[0] != source || sources[1] != source || configs[0].Address != config.Address || configs[1].Address != first.address || configs[1].AllowedOrigins[0] != config.AllowedOrigins[0] {
		t.Fatalf("replacement source/config drifted: sources=%v configs=%+v", sources, configs)
	}
	if supervisor.Done() != second.done || supervisor.MappingFailures() != second.mapping {
		t.Fatal("active API generation did not replace its event channels")
	}
}

func TestAPISupervisorUsesFiniteExactBackoffAndShutdownCancelsPendingRetry(t *testing.T) {
	runtime := newContinuityRuntime(t)
	defer shutdownContinuityRuntime(t, runtime)
	var source snapshotapi.CaptureSource = runtime
	initial := newFakeSnapshotServer()
	timers := &manualAPITimers{}
	calls := 0
	supervisor, err := newAPISupervisor(source, snapshotapi.ServerConfig{Address: "127.0.0.1:8080"}, func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error) {
		calls++
		if calls == 1 {
			return initial, nil
		}
		return nil, errors.New("bind remains unavailable")
	}, timers.new)
	if err != nil {
		t.Fatal(err)
	}
	initial.finish(errors.New("listener failed"))
	<-supervisor.Done()
	supervisor.UnexpectedTermination()
	for index := range apiRestartBackoff {
		timers.fire(index)
		if restartErr := supervisor.RestartDue(); restartErr == nil {
			t.Fatalf("attempt %d unexpectedly succeeded", index+1)
		}
	}
	if !supervisor.Unavailable() || calls != 6 || len(timers.delays) != len(apiRestartBackoff) || !runtime.Status().ProcessLive {
		t.Fatalf("exhaustion unavailable=%t calls=%d processLive=%t delays=%v", supervisor.Unavailable(), calls, runtime.Status().ProcessLive, timers.delays)
	}
	for index, want := range apiRestartBackoff {
		if timers.delays[index] != want {
			t.Fatalf("backoff %d=%s want %s", index, timers.delays[index], want)
		}
	}

	active := newFakeSnapshotServer()
	timers = &manualAPITimers{}
	supervisor, err = newAPISupervisor(source, snapshotapi.ServerConfig{Address: "127.0.0.1:8080"}, func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error) { return active, nil }, timers.new)
	if err != nil {
		t.Fatal(err)
	}
	active.finish(errors.New("listener failed"))
	<-supervisor.Done()
	supervisor.UnexpectedTermination()
	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if !timers.timers[0].stopped || supervisor.RestartTimer() != nil {
		t.Fatal("controlled stop left an API retry timer active")
	}

	active = newFakeSnapshotServer()
	supervisor, err = newAPISupervisor(source, snapshotapi.ServerConfig{Address: "127.0.0.1:8080"}, func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error) { return active, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	active.mu.Lock()
	stops := active.stops
	active.mu.Unlock()
	if stops != 1 {
		t.Fatalf("active API shutdown count=%d", stops)
	}
}

type fakeCaptureSource struct{}

func (fakeCaptureSource) CaptureSnapshot() (operations.SnapshotCapture, error) {
	return operations.SnapshotCapture{}, nil
}

type fakeSnapshotServer struct {
	done    chan error
	mapping chan snapshotapi.MappingFailure
	address string
	once    sync.Once
	mu      sync.Mutex
	stops   int
}

func newFakeSnapshotServer() *fakeSnapshotServer {
	return &fakeSnapshotServer{done: make(chan error, 1), mapping: make(chan snapshotapi.MappingFailure, 1)}
}

func (server *fakeSnapshotServer) Done() <-chan error { return server.done }
func (server *fakeSnapshotServer) MappingFailures() <-chan snapshotapi.MappingFailure {
	return server.mapping
}
func (server *fakeSnapshotServer) Address() string { return server.address }
func (server *fakeSnapshotServer) finish(err error) {
	server.once.Do(func() { server.done <- err; close(server.done) })
}
func (server *fakeSnapshotServer) Shutdown(context.Context) error {
	server.mu.Lock()
	server.stops++
	server.mu.Unlock()
	server.finish(nil)
	return nil
}

type manualAPITimer struct {
	channel chan time.Time
	stopped bool
}

func (timer *manualAPITimer) Chan() <-chan time.Time { return timer.channel }
func (timer *manualAPITimer) Stop() bool {
	timer.stopped = true
	return true
}

type manualAPITimers struct {
	delays []time.Duration
	timers []*manualAPITimer
}

func (timers *manualAPITimers) new(delay time.Duration) apiRestartTimer {
	timer := &manualAPITimer{channel: make(chan time.Time, 1)}
	timers.delays, timers.timers = append(timers.delays, delay), append(timers.timers, timer)
	return timer
}

func (timers *manualAPITimers) fire(index int) { timers.timers[index].channel <- time.Now() }

func continuityAcknowledgeAggregate(t *testing.T, ctx context.Context, owner *engine.Engine, binding reference.Binding, epoch uint64, at time.Time) {
	t.Helper()
	for _, input := range []engine.ConnectionControlInput{
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.ConnectionAttempt, ConnectionEpoch: epoch, CommandToken: 1, ReceiptTime: at, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateCommandWriteResult, ConnectionEpoch: epoch, CommandToken: 2, ReceiptTime: at, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateSubscriptionResult, ConnectionEpoch: epoch, CommandToken: 2, Position: engine.LivePosition{ConnectionEpoch: epoch, FrameSequence: 1}, ReceiptTime: at, Outcome: engine.ControlSucceeded},
	} {
		admission, completion := owner.AdmitConnectionControl(ctx, input)
		if admission != engine.AdmissionAdmitted || continuityAwait(t, ctx, completion).Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("connection control %s admission=%s", input.Kind, admission)
		}
	}
}

func continuityCompleteHydration(t *testing.T, ctx context.Context, owner *engine.Engine, binding reference.Binding, purpose engine.HydrationPurpose, epoch uint64, at time.Time) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if admission != engine.AdmissionAdmitted {
		t.Fatalf("hydration plan admission=%s", admission)
	}
	plan := continuityAwait(t, ctx, completion)
	if plan.Code != engine.DispositionHydrationPlanApplied {
		t.Fatalf("hydration plan=%+v", plan)
	}
	var fence engine.HydrationFenceCommand
	if command, ok := plan.Plan.FenceCommand(); ok {
		fence = command
	}
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, terminalCompletion := owner.AdmitHydrationTerminal(ctx, terminal)
		result := continuityAwait(t, ctx, terminalCompletion)
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal=%+v", result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("hydration produced no fence")
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 1, 1, at)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := owner.AdmitAggregateIngressFence(ctx, fenceInput)
	if result := continuityAwait(t, ctx, fenceCompletion); result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence=%+v", result)
	}
}

func continuityAwait[T any](t *testing.T, ctx context.Context, completion <-chan T) T {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatalf("integrated proof completion exceeded deadline: %v", ctx.Err())
		var zero T
		return zero
	case result, ok := <-completion:
		if !ok {
			t.Fatal("integrated proof completion closed without a result")
		}
		return result
	}
}
