package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

// TestPV1RCVertical is the closed-market release proof. It carries one
// deterministic scanner state through a persisted checkpoint and catch-up,
// engine-owned T/Q enrichment, the loopback C10 response, and the production
// C11 view model. Component proofs own the deeper counterexample matrices.
func TestPV1RCVertical(t *testing.T) {
	if testing.Short() {
		t.Skip("integrated private V1 RC acceptance proof")
	}
	proof, cancelProof := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelProof()
	binding := scannerTestBinding(t)
	start := binding.SessionStart()
	now := start
	config := operations.DefaultConfig()
	config.EvaluationDelay = 0
	config.SampleCadence = 10 * time.Minute

	source, err := operations.New(proof, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	v1RCAcknowledgeAggregate(t, proof, source.Engine(), binding, 1, now)
	v1RCCompleteHydration(t, proof, source.Engine(), binding, engine.HydrationFreshBootstrap, 1, now)
	for index := 0; index < 60; index++ {
		window := start.Add(time.Duration(index) * time.Second)
		now = window.Add(time.Second)
		input := engine.AggregateInput{
			SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: "AAA",
			WindowStart: window, WindowEnd: window.Add(time.Second),
			Values:       engine.AggregateValues{Open: 12, High: 12.1, Low: 11.9, Close: 12, Volume: 10_000, VWAP: 12, AverageTradeSize: 100, ATSProvenance: engine.ATSLiveProviderAverage},
			DeliveryTime: now, Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(index + 2), ArrayIndex: 1},
		}
		admission, completion := source.Engine().AdmitAggregate(proof, input)
		if admission != engine.AdmissionAdmitted || v1RCAwait(t, proof, completion).Code != engine.DispositionAggregateInserted {
			t.Fatalf("source aggregate %d admission=%s", index, admission)
		}
	}
	v1RCAdvanceCoverageAndTimer(t, proof, source.Engine(), 61, 2, now)
	if got := source.Status(); !got.BackendReady || !got.RankingCurrent {
		t.Fatalf("source scanner not ready: %+v", got)
	}
	if rows := source.Engine().ObserveTQ().Desired; len(rows) != 1 || rows[0] != "AAA" {
		t.Fatalf("source ranking did not select AAA: %v", rows)
	}
	admission, projected := source.Engine().AdmitCheckpointProjection(proof)
	if admission != engine.AdmissionAdmitted {
		t.Fatalf("checkpoint projection admission=%s", admission)
	}
	projection := v1RCAwait(t, proof, projected)
	if projection.Disposition != engine.CheckpointProjected || projection.Image.T0 != now {
		t.Fatalf("checkpoint projection=%+v", projection)
	}
	store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: binding.Identity(), ArtifactByteLimit: 8 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if written := store.Write(proof, projection.Image); written.Disposition != checkpoint.WriteCompleted {
		t.Fatalf("checkpoint write=%+v", written)
	}
	loaded := store.Load(proof)
	if loaded.Disposition != checkpoint.LoadedLatest || !loaded.Candidate.Integrity {
		t.Fatalf("checkpoint load=%+v", loaded)
	}
	v1RCShutdown(t, proof, source)

	restartAt := start.Add(70 * time.Second)
	clockNanos := &atomic.Int64{}
	clockNanos.Store(restartAt.UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	facts := make(chan struct{})
	websocketServer := v1RCWebSocketServer(t, clockNanos, facts)
	defer websocketServer.Close()
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "ticker": "AAA", "adjusted": false, "resultsCount": 0, "results": []any{}})
	}))
	defer hydrationServer.Close()
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "local-fixture", Queue: massive.LiveQueueConfig{FrameSlots: 32, MaxFrameBytes: 4096, TotalFrameBytes: 32 * 4096}, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "local-fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	writer, err := checkpoint.NewWriter(proof, store)
	if err != nil {
		t.Fatal(err)
	}
	config.SampleCadence = 50 * time.Millisecond
	config.ConnectionAttemptDeadline = 5 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	restarted, err := operations.NewWithCheckpoint(proof, binding, config, clock, writer)
	if err != nil {
		writer.Close()
		t.Fatal(err)
	}
	defer v1RCShutdown(t, proof, restarted)
	components := operations.LiveComponents{Adapter: adapter, Hydrator: hydrator, Store: store, Workers: 1, RowsPerChunk: 16, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600,
		Durations: massive.OperationalDurations{Dial: time.Second, HandshakeStep: time.Second, HandshakeTotal: 3 * time.Second, HeartbeatInterval: 10 * time.Second, HeartbeatDeadline: time.Second, Write: time.Second, Close: time.Second}}
	liveContext, cancelLive := context.WithCancel(proof)
	defer cancelLive()
	liveDone := make(chan error, 1)
	go func() { liveDone <- restarted.RunLive(liveContext, components) }()
	v1RCWaitFor(t, proof, liveDone, func() bool {
		view := restarted.Engine().ObserveOperational()
		return view.InstalledCheckpoint && view.Hydration.Purpose == engine.HydrationCheckpointCatchup && view.Hydration.FenceReconciled && restarted.Status().BackendReady
	}, "production checkpoint discovery/catch-up readiness")
	v1RCWaitFor(t, proof, liveDone, func() bool {
		rows := restarted.Engine().ObserveTQ().Rows
		return len(rows) == 1 && rows[0].ProviderPresent
	}, "production T/Q subscription acknowledgement")
	clockNanos.Store(restartAt.Add(6 * time.Second).UnixNano())
	close(facts)
	v1RCWaitFor(t, proof, liveDone, func() bool {
		rows := restarted.Engine().ObserveTQ().Rows
		return len(rows) == 1 && rows[0].TradeCoverage && rows[0].QuoteCoverage && rows[0].Tape.FiveSecondStatus == engine.TQCurrent && rows[0].Spread.Status == engine.TQCurrent
	}, "production T/Q facts and feature timer")
	if got := restarted.Status(); !got.BackendReady || !got.RankingCurrent || got.Lifecycle != "live" {
		t.Fatalf("restarted scanner not ready: %+v", got)
	}
	operational := restarted.Engine().ObserveOperational()
	if !operational.InstalledCheckpoint || operational.Hydration.Purpose != engine.HydrationCheckpointCatchup || !operational.Hydration.FenceReconciled {
		t.Fatalf("checkpoint/catch-up facts not preserved: %+v", operational)
	}
	tq := restarted.Engine().ObserveTQ()
	if len(tq.Rows) != 1 || tq.Rows[0].Symbol != "AAA" || !tq.Rows[0].TradeCoverage || !tq.Rows[0].QuoteCoverage || tq.Rows[0].Tape.FiveSecondStatus != engine.TQCurrent || tq.Rows[0].Spread.Status != engine.TQCurrent {
		t.Fatalf("T/Q enrichment not current: %+v", tq)
	}

	api, err := snapshotapi.Listen(restarted, snapshotapi.ServerConfig{Address: "127.0.0.1:0", AllowedOrigins: []string{"http://127.0.0.1:14173"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		deadline, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := api.Shutdown(deadline); err != nil {
			t.Fatal(err)
		}
	}()
	request, err := http.NewRequestWithContext(proof, http.MethodGet, "http://"+api.Address()+"/api/v2/snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://127.0.0.1:14173")
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || response.StatusCode != http.StatusOK || response.Header.Get("Access-Control-Allow-Origin") != "http://127.0.0.1:14173" {
		t.Fatalf("snapshot response status=%d cors=%q read=%v close=%v", response.StatusCode, response.Header.Get("Access-Control-Allow-Origin"), readErr, closeErr)
	}
	var wire struct {
		SchemaVersion string `json:"schema_version"`
		Rows          []struct {
			Symbol string `json:"symbol"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(body, &wire); err != nil || wire.SchemaVersion != "scanner.snapshot.v2" || len(wire.Rows) != 1 || wire.Rows[0].Symbol != "AAA" {
		t.Fatalf("snapshot wire=%+v err=%v", wire, err)
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	const script = `
import { readFileSync } from "node:fs";
import { buildViewModel } from "./ui/model.js";
const model = buildViewModel(JSON.parse(readFileSync(0, "utf8")), "connected");
const checkpoint = model.diagnostics.find(([name]) => name === "checkpoint.installed");
if (!model.current || !model.processLive || !model.backendReady || model.rankingMode !== "qualified_current" || model.rows.length !== 1 || model.rows[0].symbol !== "AAA" || model.rows[0].tape.state !== "current" || model.rows[0].spread.state !== "current" || checkpoint?.[1] !== "true") throw new Error(JSON.stringify(model));
process.stdout.write(JSON.stringify({publication:model.publicationID,sample:model.sampleID,symbol:model.rows[0].symbol,tape:model.rows[0].tape.primary,spread:model.rows[0].spread.primary}));
`
	command := exec.CommandContext(proof, "node", "--input-type=module", "-e", script)
	command.Dir = root
	command.Stdin = bytes.NewReader(body)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("C11 production view model rejected C10 body: %v\n%s", err, output)
	}
	t.Logf("integrated private V1 RC view=%s", output)
	cancelLive()
	if err := v1RCAwait(t, proof, liveDone); !errors.Is(err, context.Canceled) {
		t.Fatalf("production live composition shutdown=%v", err)
	}
}

func v1RCAcknowledgeAggregate(t *testing.T, ctx context.Context, owner *engine.Engine, binding reference.Binding, epoch uint64, at time.Time) {
	t.Helper()
	for _, input := range []engine.ConnectionControlInput{
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.ConnectionAttempt, ConnectionEpoch: epoch, CommandToken: 1, ReceiptTime: at, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateCommandWriteResult, ConnectionEpoch: epoch, CommandToken: 2, ReceiptTime: at, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateSubscriptionResult, ConnectionEpoch: epoch, CommandToken: 2, Position: engine.LivePosition{ConnectionEpoch: epoch, FrameSequence: 1}, ReceiptTime: at, Outcome: engine.ControlSucceeded},
	} {
		admission, completion := owner.AdmitConnectionControl(ctx, input)
		if admission != engine.AdmissionAdmitted || v1RCAwait(t, ctx, completion).Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("connection control %s admission=%s", input.Kind, admission)
		}
	}
}

func v1RCCompleteHydration(t *testing.T, ctx context.Context, owner *engine.Engine, binding reference.Binding, purpose engine.HydrationPurpose, epoch uint64, at time.Time) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if admission != engine.AdmissionAdmitted {
		t.Fatalf("hydration plan admission=%s", admission)
	}
	plan := v1RCAwait(t, ctx, completion)
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
		result := v1RCAwait(t, ctx, terminalCompletion)
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
	if result := v1RCAwait(t, ctx, fenceCompletion); result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence=%+v", result)
	}
}

func v1RCAdvanceCoverageAndTimer(t *testing.T, ctx context.Context, owner *engine.Engine, through, marker uint64, at time.Time) {
	t.Helper()
	command, err := owner.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	input, err := engine.NewLiveCoverageFenceInput(command, engine.LiveCoverageFenceComplete, through, marker, at)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := owner.AdmitLiveCoverageFence(ctx, input)
	if admission != engine.AdmissionAdmitted || v1RCAwait(t, ctx, completion).Code != engine.DispositionLiveCoverageFenceApplied {
		t.Fatalf("live coverage admission=%s", admission)
	}
	admission, timer := owner.AdmitTimer(ctx)
	if admission != engine.AdmissionAdmitted || v1RCAwait(t, ctx, timer).Code != engine.DispositionTimerApplied {
		t.Fatalf("timer admission=%s", admission)
	}
}

func v1RCWebSocketServer(t *testing.T, clock *atomic.Int64, releaseFacts <-chan struct{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		ctx := request.Context()
		write := func(value string) bool { return connection.Write(ctx, websocket.MessageText, []byte(value)) == nil }
		if !write(`[{"ev":"status","status":"connected"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"auth_success"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		_, _, err = connection.Read(ctx)
		if err != nil || !write(`[{"ev":"status","status":"success"},{"ev":"status","status":"success"}]`) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-releaseFacts:
		}
		now := time.Unix(0, clock.Load()).UTC()
		if !write(fmt.Sprintf(`[{"ev":"T","sym":"AAA","x":4,"i":"v1rc-trade","p":12,"s":100,"t":%d,"c":[14]}]`, now.Add(-500*time.Millisecond).UnixMilli())) {
			return
		}
		for _, offset := range []time.Duration{-5 * time.Second, -4 * time.Second, -3 * time.Second, -2 * time.Second, -time.Second, 0} {
			if !write(fmt.Sprintf(`[{"ev":"Q","sym":"AAA","t":%d,"bp":11.99,"ap":12.01,"c":[1],"i":[2]}]`, now.Add(offset).UnixMilli())) {
				return
			}
		}
		<-ctx.Done()
	}))
}

func v1RCWaitFor(t *testing.T, ctx context.Context, liveDone <-chan error, ready func() bool, boundary string) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for !ready() {
		select {
		case <-ctx.Done():
			t.Fatalf("%s exceeded proof deadline: %v", boundary, ctx.Err())
		case err := <-liveDone:
			t.Fatalf("%s ended live composition: %v", boundary, err)
		case <-ticker.C:
		}
	}
}

func v1RCAwait[T any](t *testing.T, ctx context.Context, completion <-chan T) T {
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

func v1RCShutdown(t *testing.T, parent context.Context, runtime *operations.Runtime) {
	t.Helper()
	deadline, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if err := runtime.Shutdown(deadline); err != nil {
		t.Fatal(err)
	}
}
