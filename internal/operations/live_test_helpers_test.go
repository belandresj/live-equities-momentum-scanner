package operations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
	"github.com/coder/websocket"
)

func applyControl(t *testing.T, owner *engine.Engine, binding reference.Binding, kind engine.ConnectionControlKind, epoch, token uint64, position engine.LivePosition, at time.Time) {
	t.Helper()
	outcome := engine.ControlSucceeded
	if kind == engine.ConnectionLost {
		outcome = engine.ControlFailed
	}
	admission, completion := owner.AdmitConnectionControl(context.Background(), engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind, ConnectionEpoch: epoch, CommandToken: token, Position: position, ReceiptTime: at, Outcome: outcome})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("control %s admission=%s", kind, admission)
	}
	if got := <-completion; got.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("control %s disposition=%+v", kind, got)
	}
}

func completeHydration(t *testing.T, owner *engine.Engine, binding reference.Binding, purpose engine.HydrationPurpose, epoch uint64, at time.Time) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan")
	}
	plan := <-completion
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, done := owner.AdmitHydrationTerminal(context.Background(), terminal)
		result := <-done
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("terminal=%+v", result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatalf("hydration produced no fence: %+v", plan)
	}
	through := max(uint64(1), owner.ObserveOperational().Connection.AckFrame)
	input, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, through, 1, at)
	if err != nil {
		t.Fatal(err)
	}
	_, done := owner.AdmitAggregateIngressFence(context.Background(), input)
	if got := <-done; got.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence=%+v", got)
	}
}

func operationsBinding(t *testing.T) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v3/reference/tickers":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "count": 1, "results": []map[string]any{{"ticker": "AAA", "active": true, "market": "stocks", "locale": "us", "type": "CS"}}})
		case strings.HasPrefix(r.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": 1, "results": []map[string]any{{"T": "AAA", "c": 10.0, "t": facts.PriorRegularClose.UnixMilli()}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func waitForOperational(t *testing.T, run *Runtime, joined <-chan error, minimumConnections int32, ready bool) {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for {
		status := run.Status()
		if status.BackendReady == ready && run.Engine().ObserveOperational().Connection.Epoch >= uint64(minimumConnections) {
			return
		}
		select {
		case err := <-joined:
			t.Fatalf("live composition ended while waiting for ready=%t: %v", ready, err)
		case <-deadline:
			t.Fatalf("timed out waiting for ready=%t connections>=%d: %+v", ready, minimumConnections, status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func recoverableWebSocketServer(t *testing.T, connections *atomic.Int32, failNew *atomic.Bool, disconnect <-chan struct{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		connections.Add(1)
		if failNew.Load() {
			return
		}
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
		select {
		case <-ctx.Done():
		case <-disconnect:
		}
	}))
}
