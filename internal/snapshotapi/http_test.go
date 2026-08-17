package snapshotapi

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type countingCaptureSource struct {
	runtime *operations.Runtime
	calls   atomic.Uint64
}

func (source *countingCaptureSource) CaptureSnapshot() (operations.SnapshotCapture, error) {
	source.calls.Add(1)
	return source.runtime.CaptureSnapshot()
}

func TestPC10HTTPRoutesCORSAndLifecycle(t *testing.T) {
	runtime, binding, now := newSnapshotRuntime(t)
	source := &countingCaptureSource{runtime: runtime}
	handler, err := NewHandler(source, HandlerConfig{AllowedOrigins: []string{"http://127.0.0.1:3000"}})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/livez", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || source.calls.Load() != 1 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Length") != strconvLen(response.Body.Bytes()) {
		t.Fatalf("initial liveness = code %d calls %d headers=%v body=%s", response.Code, source.calls.Load(), response.Header(), response.Body.String())
	}
	var live livenessResponse
	if err := json.Unmarshal(response.Body.Bytes(), &live); err != nil || !live.ProcessLive || live.Reason != "" || live.SampleID == "0" {
		t.Fatalf("initial liveness body = %+v/%v", live, err)
	}

	beforeReady := source.calls.Load()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var readiness readinessResponse
	if response.Code != http.StatusServiceUnavailable || source.calls.Load() != beforeReady+1 || json.Unmarshal(response.Body.Bytes(), &readiness) != nil ||
		readiness.PublicationID == nil || readiness.BindingIdentity == nil || readiness.BackendReady || readiness.Reason != "lifecycle_not_ready" {
		t.Fatalf("initial readiness = code %d calls %d body=%s", response.Code, source.calls.Load(), response.Body.String())
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/api/v2/snapshot", http.StatusOK)
	beforeV1 := source.calls.Load()
	v1 := httptest.NewRecorder()
	handler.ServeHTTP(v1, httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil))
	if v1.Code != http.StatusNotFound || source.calls.Load() != beforeV1 {
		t.Fatalf("retired v1 route = code %d calls %d body=%s", v1.Code, source.calls.Load(), v1.Body.String())
	}
	for _, test := range []struct {
		path   string
		status int
	}{{"/readyz", http.StatusServiceUnavailable}, {"/api/v2/snapshot", http.StatusOK}} {
		before := source.calls.Load()
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, test.path, nil))
		if response.Code != test.status || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || source.calls.Load() != before+1 {
			t.Fatalf("initial HEAD %s = code %d calls %d headers=%v body=%q", test.path, response.Code, source.calls.Load(), response.Header(), response.Body.String())
		}
	}

	before := source.calls.Load()
	for _, test := range []struct {
		request *http.Request
		status  int
		allow   string
	}{
		{httptest.NewRequest(http.MethodPost, "/livez", nil), http.StatusMethodNotAllowed, "GET, HEAD"},
		{httptest.NewRequest(http.MethodGet, "/missing", nil), http.StatusNotFound, ""},
		{httptest.NewRequest(http.MethodGet, "/livez?schema=2", nil), http.StatusBadRequest, ""},
		{httptest.NewRequest(http.MethodHead, "/missing", nil), http.StatusNotFound, ""},
	} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, test.request)
		if response.Code != test.status || response.Header().Get("Allow") != test.allow || source.calls.Load() != before ||
			response.Header().Get("Content-Length") == "" || test.request.Method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatalf("invalid route/method = %s %s code=%d allow=%q calls=%d headers=%v body=%q", test.request.Method, test.request.URL, response.Code, response.Header().Get("Allow"), source.calls.Load(), response.Header(), response.Body.String())
		}
	}
	for _, test := range []struct {
		name   string
		origin []string
	}{
		{"unlisted", []string{"https://evil.example"}},
		{"wildcard", []string{"*"}},
		{"null", []string{"null"}},
		{"malformed", []string{"://bad"}},
		{"multiple", []string{"http://127.0.0.1:3000", "http://127.0.0.1:3000"}},
	} {
		request = httptest.NewRequest(http.MethodGet, "/livez", nil)
		for _, origin := range test.origin {
			request.Header.Add("Origin", origin)
		}
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "" || response.Header().Get("Vary") != "" {
			t.Fatalf("%s origin reached capture or received CORS data: code=%d headers=%v", test.name, response.Code, response.Header())
		}
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v2/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" || response.Header().Get("Vary") != "Origin" {
		t.Fatalf("preflight = code %d calls %d headers=%v", response.Code, source.calls.Load(), response.Header())
	}
	request = httptest.NewRequest(http.MethodOptions, "/readyz", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodHead)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Methods") != "GET, HEAD" || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" {
		t.Fatalf("HEAD preflight = code %d calls %d headers=%v", response.Code, source.calls.Load(), response.Header())
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v2/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before {
		t.Fatal("invalid preflight reached capture")
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v2/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "Authorization")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("custom-header preflight received access")
	}
	request = httptest.NewRequest(http.MethodOptions, "/missing", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before {
		t.Fatal("unknown-route preflight was exposed")
	}

	request = httptest.NewRequest(http.MethodHead, "/livez", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" || response.Header().Get("Vary") != "Origin" {
		t.Fatalf("HEAD parity = code %d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}

	makeSnapshotReady(t, runtime.Engine(), binding, *now)
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/readyz", http.StatusOK)
	for _, path := range []string{"/readyz", "/api/v2/snapshot"} {
		before := source.calls.Load()
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, path, nil))
		if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || source.calls.Load() != before+1 {
			t.Fatalf("ready HEAD %s = code %d calls %d headers=%v body=%q", path, response.Code, source.calls.Load(), response.Header(), response.Body.String())
		}
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil)
	response = httptest.NewRecorder()
	before = source.calls.Load()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || source.calls.Load() != before+1 {
		t.Fatalf("snapshot = code %d calls %d body=%s", response.Code, source.calls.Load(), response.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil || snapshot.SchemaVersion != SchemaVersion || snapshot.Publication.ID == "0" || snapshot.Rows == nil {
		t.Fatalf("snapshot body = %+v/%v", snapshot, err)
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := runtime.Shutdown(shutdown); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	request = httptest.NewRequest(http.MethodGet, "/livez", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || json.Unmarshal(response.Body.Bytes(), &live) != nil || live.ProcessLive || live.Reason != "runtime_unavailable" {
		t.Fatalf("terminal liveness = code %d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || json.Unmarshal(response.Body.Bytes(), &readiness) != nil || readiness.PublicationID == nil || readiness.BackendReady || readiness.Reason != "runtime_unavailable" {
		t.Fatalf("terminal readiness = code %d body=%s", response.Code, response.Body.String())
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/api/v2/snapshot", http.StatusOK)
}

// TestPMVPAPITape5sSealedPublicationBoundaries proves that every bounded
// Tape 5s state is produced by the engine, sealed by Runtime, and served by
// the v2 route without API-side coverage interpretation.
func TestPMVPAPITape5sSealedPublicationBoundaries(t *testing.T) {
	cases := []struct {
		name        string
		coverageAge *time.Duration
		mutate      func(*testing.T, *operations.Runtime, reference.Binding, *time.Time)
		status      string
		reason      string
		value       *float64
	}{
		{name: "unavailable coverage", status: "unavailable", reason: "coverage"},
		{name: "below one second", coverageAge: tapeDurationPointer(999 * time.Millisecond), status: "warming", reason: "coverage_warming"},
		{name: "one through below five seconds", coverageAge: tapeDurationPointer(time.Second), status: "warming", reason: "five_second_warming"},
		{name: "five seconds covered silence", coverageAge: tapeDurationPointer(5 * time.Second), status: "current", reason: "qualifying_original_prints", value: floatPointer(0)},
		{name: "unequal repeat", coverageAge: tapeDurationPointer(5 * time.Second), mutate: makeTapeUnequalRepeat, status: "invalid", reason: "unequal_repeat"},
		{name: "pressure shedding", coverageAge: tapeDurationPointer(5 * time.Second), mutate: shedTapeForPressure, status: "pressure_shed", reason: "pressure"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime, binding, now := newQualifiedTapeRuntime(t)
			t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
			owner := runtime.Engine()
			if test.coverageAge != nil {
				command, err := owner.IssueTQCommand()
				if err != nil {
					t.Fatal(err)
				}
				input, err := engine.NewTQCommandResultInput(command, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 100}, now.Add(-operations.DefaultConfig().EvaluationDelay).Add(-*test.coverageAge), engine.ControlSucceeded)
				if err != nil {
					t.Fatal(err)
				}
				admission, completion := owner.AdmitTQCommandResult(context.Background(), input)
				if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
					t.Fatal("T/Q coverage acknowledgement was not applied")
				}
			}
			if test.mutate != nil {
				test.mutate(t, runtime, binding, now)
			}

			capture, err := runtime.CaptureSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			sealed, ok := operations.InspectSnapshotCapture(capture)
			if !ok || sealed.Engine.Publication.PublicationID == 0 || sealed.Engine.Publication.PublicationID != sealed.Engine.TQ.PublicationID || len(sealed.Engine.TQ.Rows) != 1 {
				t.Fatalf("Tape case was not one sealed engine publication: %+v", sealed.Engine)
			}
			source := &countingCaptureSource{runtime: runtime}
			handler, err := NewHandler(source, HandlerConfig{})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
			var snapshot Snapshot
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &snapshot) != nil || len(snapshot.Rows) != 1 {
				t.Fatalf("Tape v2 response=%d body=%s", response.Code, response.Body.String())
			}
			got := snapshot.Rows[0].Tape5s
			if got.Status != test.status || got.Reason != test.reason || (got.TradesPerSecond == nil) != (test.value == nil) ||
				got.TradesPerSecond != nil && *got.TradesPerSecond != *test.value {
				t.Fatalf("Tape 5s v2 tuple=%+v want status=%s reason=%s value=%v", got, test.status, test.reason, test.value)
			}
		})
	}
}

func TestPMVPVolumeRevisionPreservesDerivedBackendReadiness(t *testing.T) {
	runtime, binding, now := newQualifiedTapeRuntime(t)
	t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
	beforeCapture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	before, err := Map(beforeCapture)
	if err != nil {
		t.Fatal(err)
	}
	target := now.Add(-operations.DefaultConfig().EvaluationDelay)
	revision := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: "AAA", WindowStart: target.Add(-time.Second), WindowEnd: target, DeliveryTime: *now,
		Values: engine.AggregateValues{Open: 20, High: 20, Low: 20, Close: 20, Volume: 6_000, VWAP: 20, AverageTradeSize: 5, ATSProvenance: engine.ATSLiveProviderAverage},
		Live:   engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 200}}
	admission, completion := runtime.Engine().AdmitAggregate(context.Background(), revision)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionAggregateRevised {
		t.Fatal("same-identity Volume revision was not applied")
	}
	timerAdmission, timerCompletion := runtime.Engine().AdmitTimer(context.Background())
	if timerAdmission != engine.AdmissionAdmitted || timerCompletion == nil || (<-timerCompletion).Code != engine.DispositionTimerApplied {
		t.Fatal("same-T Volume correction was not sealed")
	}
	afterCapture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	after, err := Map(afterCapture)
	if err != nil {
		t.Fatal(err)
	}
	if !before.Status.BackendReady || !after.Status.BackendReady || !before.Status.RankingCurrent || !after.Status.RankingCurrent ||
		before.Publication.CommittedT == nil || after.Publication.CommittedT == nil || *before.Publication.CommittedT != *after.Publication.CommittedT ||
		before.Publication.ID == after.Publication.ID || len(before.Rows) != 1 || len(after.Rows) != 1 ||
		before.Rows[0].Volume.ValueShares == nil || *before.Rows[0].Volume.ValueShares != 300_000 ||
		after.Rows[0].Volume.ValueShares == nil || *after.Rows[0].Volume.ValueShares != 301_000 ||
		before.Rows[0].Rank != after.Rows[0].Rank || before.Rows[0].Symbol != after.Rows[0].Symbol ||
		before.Rows[0].DayChangeRatio != after.Rows[0].DayChangeRatio || before.Rows[0].LastUSD != after.Rows[0].LastUSD {
		t.Fatalf("Volume revision changed readiness/ranking: before=%+v after=%+v", before, after)
	}
}

// TestSlice2LocalDefectProductionComposition proves the live Runtime -> Engine
// -> sealed snapshot -> HTTP boundary. One trusted live mark remains usable
// after a different symbol contributes attributable structural-invalid
// evidence inside already fenced coverage; the defect is local, readiness
// stays 200, rows carry no T/Q, and a later transport loss still fails closed.
func TestSlice2LocalDefectProductionComposition(t *testing.T) {
	binding := snapshotBinding(t, "AAA", "BBB")
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runtime, err := operations.New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
	owner := runtime.Engine()
	ackAt := now.Add(-config.EvaluationDelay)
	applySnapshotControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)

	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan not admitted")
	}
	plan := <-completion
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) != 2 {
		t.Fatalf("hydration plan=%+v requests=%d", plan, len(requests))
	}
	markStart := now.Add(-10 * time.Second)
	mark := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: "AAA", WindowStart: markStart, WindowEnd: markStart.Add(time.Second), DeliveryTime: markStart.Add(time.Second),
		Values: engine.AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000, VWAP: 12, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
		Live:   engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 2}}
	if admitted, completed := owner.AdmitAggregate(ctx, mark); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionAggregateInserted {
		t.Fatal("trusted live mark was not installed")
	}
	var fence engine.HydrationFenceCommand
	for _, request := range requests {
		terminal, terminalErr := engine.NewHydrationTerminalInput(request, request.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if terminalErr != nil {
			t.Fatal(terminalErr)
		}
		_, terminalCompletion := owner.AdmitHydrationTerminal(ctx, terminal)
		result := <-terminalCompletion
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal=%+v", result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := owner.AdmitAggregateIngressFence(ctx, fenceInput)
	if result := <-fenceCompletion; result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence=%+v", result)
	}

	invalid := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: "BBB", WindowStart: markStart, WindowEnd: markStart.Add(time.Second), DeliveryTime: now,
		Values: engine.AggregateValues{Open: 10, High: 10, Low: 10, Close: 10, Volume: math.NaN(), VWAP: 10, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
		Live:   engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 3}}
	admitted, invalidCompletion := owner.AdmitAggregate(ctx, invalid)
	if admitted != engine.AdmissionAdmitted || invalidCompletion == nil {
		t.Fatal("attributable invalid aggregate was not admitted to the owner FIFO")
	}
	if result := <-invalidCompletion; result.Code != engine.DispositionAggregateRejected || result.Reason != engine.ReasonStructural {
		t.Fatalf("invalid aggregate=%+v", result)
	}
	if admitted, completed := owner.AdmitTimer(ctx); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionTimerApplied {
		t.Fatal("same-target evaluation timer was not applied")
	}

	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	view, ok := operations.InspectSnapshotCapture(capture)
	if !ok || view.Engine.Publication.AggregateEvaluation.Features.SessionVolume.Statuses[3] != 1 ||
		view.Engine.Publication.AggregateEvaluation.Features.Activity30s.Statuses[3] != 1 ||
		view.Engine.Publication.AggregateEvaluation.Features.Move30s.Statuses[3] != 1 {
		t.Fatalf("structurally invalid aggregate was not field-local invalid: %+v", view.Engine.Publication.AggregateEvaluation.Features)
	}
	mapped, err := Map(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !mapped.Status.BackendReady || !mapped.Status.RankingCurrent || mapped.Ranking.Mode != "degraded_current" || mapped.Ranking.Reason != "incomplete_population" ||
		len(mapped.Rows) != 1 || mapped.Rows[0].Symbol != "AAA" || len(mapped.TQ.DesiredSymbols) != 0 || mapped.Rows[0].TQMembership.Desired ||
		mapped.Accounting.Population.TrustedRankableMark != 1 || mapped.Accounting.Population.UnknownDueFailureOrFence != 1 || mapped.Accounting.Uncertainty.PostBootstrapGap != 1 {
		t.Fatalf("local defect snapshot status=%+v ranking=%+v rows=%+v accounting=%+v tq=%+v", mapped.Status, mapped.Ranking, mapped.Rows, mapped.Accounting, mapped.TQ)
	}
	source := &countingCaptureSource{runtime: runtime}
	handler, err := NewHandler(source, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/readyz", http.StatusOK)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
	var wire Snapshot
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &wire) != nil || wire.Ranking.Mode != "degraded_current" || len(wire.Rows) != 1 {
		t.Fatalf("partial snapshot HTTP=%d body=%s", response.Code, response.Body.String())
	}
	lost := engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.ConnectionLost,
		ConnectionEpoch: 1, Position: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 4}, ReceiptTime: now, Outcome: engine.ControlFailed}
	if admitted, completed := owner.AdmitConnectionControl(ctx, lost); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionConnectionControlApplied {
		t.Fatal("global transport loss was not applied")
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/readyz", http.StatusServiceUnavailable)
}

func TestSuppressedEvaluatorIntegrityRemainsServable(t *testing.T) {
	runtime, binding, now := newSnapshotRuntime(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	owner := runtime.Engine()
	makeSnapshotReady(t, owner, binding, *now)
	owner.ArmEvaluatorAccountingFaultForTest()
	admission, completion := owner.AdmitTimer(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("fault timer admission=%s", admission)
	}
	disposition := <-completion
	if disposition.Code != engine.DispositionAccountingIntegrity || disposition.SuppressionDisposition != engine.SuppressionRestartRequired {
		t.Fatalf("evaluator failure=%+v", disposition)
	}
	handler, err := NewHandler(runtime, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	if response := request("/livez"); response.Code != http.StatusOK {
		t.Fatalf("livez=%d %s", response.Code, response.Body.String())
	}
	ready := request("/readyz")
	var readiness readinessResponse
	if ready.Code != http.StatusServiceUnavailable || json.Unmarshal(ready.Body.Bytes(), &readiness) != nil || readiness.Reason != "suppressed" || readiness.PublicationID == nil || readiness.BindingIdentity == nil {
		t.Fatalf("readyz=%d %s", ready.Code, ready.Body.String())
	}
	snapshotResponse := request("/api/v2/snapshot")
	var snapshot Snapshot
	if snapshotResponse.Code != http.StatusOK || json.Unmarshal(snapshotResponse.Body.Bytes(), &snapshot) != nil {
		t.Fatalf("snapshot=%d %s", snapshotResponse.Code, snapshotResponse.Body.String())
	}
	if snapshot.Publication.Lifecycle != "suppressed" || snapshot.Publication.LifecycleReason != "accounting_integrity" || snapshot.Publication.Suppression != "restart_required" ||
		snapshot.Status.BackendReady || snapshot.Status.RankingCurrent || snapshot.Status.ReadinessReason != "suppressed" || snapshot.Ranking.Mode != "suppressed" || len(snapshot.Rows) != 0 || snapshot.Publication.BindingIdentity != binding.Identity() ||
		snapshot.Operations.IntegrityFailure == nil || snapshot.Operations.IntegrityFailure.Category != "population_accounting" || snapshot.Operations.IntegrityFailure.EngineSequence == "0" ||
		snapshot.Operations.IntegrityFailure.FirstSymbol != "" || snapshot.Operations.IntegrityFailure.FirstField != "accounting.population" || snapshot.Operations.IntegrityFailure.FirstReason != "identity_mismatch" {
		t.Fatalf("servable suppression=%+v", snapshot)
	}
}

func TestPC10HTTPBoundsLoopbackCancellationAndProgress(t *testing.T) {
	var nilSource *countingCaptureSource
	if _, err := NewHandler(nilSource, HandlerConfig{}); err == nil {
		t.Fatal("typed-nil source configured")
	}
	if _, err := NewHandler(&countingCaptureSource{}, HandlerConfig{AllowedOrigins: []string{"null"}}); err == nil {
		t.Fatal("null origin configured")
	}
	if _, err := NewHandler(&countingCaptureSource{}, HandlerConfig{AllowedOrigins: []string{"http://127.0.0.1:99999"}}); err == nil {
		t.Fatal("invalid origin port configured")
	}
	for _, address := range []string{"0.0.0.0:0", "localhost:0", "192.0.2.1:8080", ":8080"} {
		if loopbackAddress(address) {
			t.Fatalf("non-explicit-loopback address accepted: %s", address)
		}
	}
	if !loopbackAddress("127.0.0.1:0") || !loopbackAddress("[::1]:8080") {
		t.Fatal("loopback address rejected")
	}
	if _, err := marshalBounded(strings.Repeat("x", maximumResponseBytes)); err == nil {
		t.Fatal("over-bound encoded response accepted")
	}

	runtime, binding, now := newSnapshotRuntime(t)
	makeSnapshotReady(t, runtime.Engine(), binding, *now)
	t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
	source := &countingCaptureSource{runtime: runtime}
	server, err := Listen(source, ServerConfig{Address: "127.0.0.1:0", AllowedOrigins: []string{"http://127.0.0.1:3000"}})
	if err != nil {
		t.Fatal(err)
	}
	if server.http.ReadHeaderTimeout != 5*time.Second || server.http.ReadTimeout != 5*time.Second || server.http.WriteTimeout != 10*time.Second || server.http.IdleTimeout != 30*time.Second || server.http.MaxHeaderBytes != 16<<10 {
		t.Fatalf("server bounds = %+v", server.http)
	}
	response, err := http.Get("http://" + server.Address() + "/api/v2/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || len(body) == 0 || len(body) > maximumResponseBytes {
		t.Fatalf("real loopback response = status %d bytes %d err %v", response.StatusCode, len(body), readErr)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	failures := make(chan string, 33)
	var concurrent sync.WaitGroup
	concurrent.Add(1)
	go func() {
		defer concurrent.Done()
		for i := 0; i < 16; i++ {
			admission, completion := runtime.Engine().AdmitTimer(context.Background())
			if admission != engine.AdmissionAdmitted || completion == nil {
				failures <- "concurrent timer not admitted"
				return
			}
			if disposition := <-completion; disposition.Code != engine.DispositionTimerApplied {
				failures <- "concurrent timer not applied"
				return
			}
		}
	}()
	for i := 0; i < 16; i++ {
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			response, err := client.Get("http://" + server.Address() + "/api/v2/snapshot")
			if err != nil {
				failures <- "concurrent snapshot request failed"
				return
			}
			defer response.Body.Close()
			var snapshot Snapshot
			if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&snapshot) != nil || snapshot.Publication.ID == "0" || snapshot.Status.TQPressureMode != snapshot.TQ.PressureMode {
				failures <- "concurrent snapshot was incoherent"
			}
		}()
	}
	concurrentDone := make(chan struct{})
	go func() { concurrent.Wait(); close(concurrentDone) }()
	select {
	case <-concurrentDone:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent publication reads timed out")
	}
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
	client.CloseIdleConnections()
	http.DefaultClient.CloseIdleConnections()
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := server.Shutdown(shutdown); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-server.Done():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not join")
	}

	handler, err := NewHandler(source, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()
	before := source.calls.Load()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/livez", nil).WithContext(canceled))
	if source.calls.Load() != before {
		t.Fatal("canceled client reached capture")
	}

	blocked := newBlockingWriter()
	handlerDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
		close(handlerDone)
	}()
	select {
	case <-blocked.started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not reach blocked client")
	}
	admission, completion := runtime.Engine().AdmitTimer(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("engine progress was blocked by HTTP writer")
	}
	select {
	case <-completion:
	case <-time.After(2 * time.Second):
		t.Fatal("engine transition blocked behind HTTP writer")
	}
	close(blocked.release)
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("blocked handler did not finish")
	}
}

func TestPTQRRecoverableSuppressionKeepsAPIAndLivenessAvailable(t *testing.T) {
	runtime, binding, now := newSnapshotRuntime(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	applySnapshotControl(t, runtime.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, *now)
	admission, completion := runtime.Engine().AdmitOperationalIngressIntegrity(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionIngressIntegrity {
		t.Fatal("recoverable ingress suppression was not installed")
	}
	if capture, captureErr := runtime.CaptureSnapshot(); captureErr != nil {
		t.Fatal(captureErr)
	} else if _, mapErr := Map(capture); mapErr != nil {
		view, _ := operations.InspectSnapshotCapture(capture)
		t.Fatalf("recoverable suppressed capture did not map: %v view=%+v", mapErr, view)
	}
	handler, err := NewHandler(runtime, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	if live := request("/livez"); live.Code != http.StatusOK {
		t.Fatalf("recoverable livez=%d %s", live.Code, live.Body.String())
	}
	if ready := request("/readyz"); ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), `"reason":"suppressed"`) {
		t.Fatalf("recoverable readyz=%d %s", ready.Code, ready.Body.String())
	}
	snapshotResponse := request("/api/v2/snapshot")
	var snapshot Snapshot
	if snapshotResponse.Code != http.StatusOK || json.Unmarshal(snapshotResponse.Body.Bytes(), &snapshot) != nil ||
		snapshot.Publication.Lifecycle != "suppressed" || snapshot.Publication.Suppression != "same_binding_recovery_allowed" || !snapshot.Status.ProcessLive || snapshot.Status.BackendReady {
		t.Fatalf("recoverable snapshot=%d %+v body=%s", snapshotResponse.Code, snapshot, snapshotResponse.Body.String())
	}

	command, err := runtime.Engine().IssueScheduledRecoveryCommand()
	if err != nil {
		t.Fatal(err)
	}
	*now = command.EarliestAt()
	input, err := engine.NewScheduledRecoveryInput(command)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion = runtime.Engine().AdmitScheduledRecovery(context.Background(), input)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionRecoveryScheduled {
		t.Fatal("scheduled recovery was not applied")
	}
	ready := request("/readyz")
	if ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), `"reason":"lifecycle_not_ready"`) {
		t.Fatalf("scheduled recovery readyz=%d %s", ready.Code, ready.Body.String())
	}
	snapshotResponse = request("/api/v2/snapshot")
	snapshot = Snapshot{}
	if snapshotResponse.Code != http.StatusOK || json.Unmarshal(snapshotResponse.Body.Bytes(), &snapshot) != nil ||
		snapshot.Publication.Lifecycle != "recovering" || snapshot.Publication.LifecycleReason != "scheduled_recovery" || snapshot.Publication.Suppression != "" ||
		!snapshot.Status.ProcessLive || snapshot.Status.BackendReady {
		t.Fatalf("scheduled recovery snapshot=%d %+v body=%s", snapshotResponse.Code, snapshot, snapshotResponse.Body.String())
	}
}

func assertStatusAndOneCapture(t *testing.T, handler http.Handler, source *countingCaptureSource, method, path string, status int) {
	t.Helper()
	before := source.calls.Load()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != status || source.calls.Load() != before+1 {
		t.Fatalf("%s %s = code %d calls %d body=%s", method, path, response.Code, source.calls.Load(), response.Body.String())
	}
}

func newSnapshotRuntime(t *testing.T) (*operations.Runtime, reference.Binding, *time.Time) {
	t.Helper()
	binding := snapshotBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runtime, err := operations.New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return runtime, binding, &now
}

func newQualifiedTapeRuntime(t *testing.T) (*operations.Runtime, reference.Binding, *time.Time) {
	t.Helper()
	binding := snapshotBinding(t, "AAA")
	now := binding.SessionStart().Add(10 * time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	runtime, err := operations.New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	owner := runtime.Engine()
	ackAt := now.Add(-config.EvaluationDelay)
	applySnapshotControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("Tape hydration plan was not admitted")
	}
	plan := <-completion
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) != 1 {
		t.Fatalf("Tape hydration plan=%+v", plan)
	}
	terminal, err := engine.NewHydrationTerminalInput(requests[0], requests[0].ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion = owner.AdmitHydrationTerminal(context.Background(), terminal)
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("Tape hydration terminal was not admitted")
	}
	terminalResult := <-completion
	if terminalResult.Code != engine.DispositionHydrationTerminalApplied || terminalResult.FenceCommand.CommandToken() == 0 {
		t.Fatalf("Tape hydration terminal=%+v", terminalResult)
	}
	fence, err := engine.NewAggregateIngressFenceInput(terminalResult.FenceCommand, engine.AggregateIngressFenceComplete, 1, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion = owner.AdmitAggregateIngressFence(context.Background(), fence)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatal("Tape hydration fence was not applied")
	}
	target := now.Add(-config.EvaluationDelay)
	for second := 0; second < 60; second++ {
		now = now.Add(time.Second)
		start := target.Add(time.Duration(second) * time.Second)
		ordinary := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
			Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), DeliveryTime: now,
			Values: engine.AggregateValues{Open: 20, High: 20, Low: 20, Close: 20, Volume: 5_000, VWAP: 20, AverageTradeSize: 5, ATSProvenance: engine.ATSLiveProviderAverage},
			Live:   engine.LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(second + 2)}}
		admission, aggregateCompletion := owner.AdmitAggregate(context.Background(), ordinary)
		if admission != engine.AdmissionAdmitted || aggregateCompletion == nil || (<-aggregateCompletion).Code != engine.DispositionAggregateInserted {
			t.Fatalf("ordinary qualifying aggregate second %d was not inserted", second)
		}
	}
	continuationCommand, err := owner.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	continuationFence, err := engine.NewLiveCoverageFenceInput(continuationCommand, engine.LiveCoverageFenceComplete, 61, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, liveCompletion := owner.AdmitLiveCoverageFence(context.Background(), continuationFence)
	if admission != engine.AdmissionAdmitted || liveCompletion == nil {
		t.Fatalf("qualifying live continuation fence admission=%s", admission)
	}
	continuationResult := <-liveCompletion
	if continuationResult.Code != engine.DispositionLiveCoverageFenceApplied {
		t.Fatalf("qualifying live continuation fence=%+v", continuationResult)
	}
	timerAdmission, timerCompletion := owner.AdmitTimer(context.Background())
	if timerAdmission != engine.AdmissionAdmitted || timerCompletion == nil || (<-timerCompletion).Code != engine.DispositionTimerApplied {
		t.Fatal("qualifying Tape timer was not applied")
	}
	view := owner.ObserveTQ()
	if len(view.Desired) != 1 || view.Desired[0] != "AAA" || !view.CommandPending {
		t.Fatalf("qualifying row did not request T/Q coverage: tq=%+v snapshot=%+v", view, runtime.Engine().ObserveSnapshot())
	}
	return runtime, binding, &now
}

func makeTapeUnequalRepeat(t *testing.T, runtime *operations.Runtime, binding reference.Binding, now *time.Time) {
	t.Helper()
	target := now.Add(-operations.DefaultConfig().EvaluationDelay)
	base := engine.TradeInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: binding.Identity(), TradingDate: binding.TradingDate(),
		Symbol: "AAA", TradeID: "same", Exchange: 1, Price: 20, EconomicSize: 100, EventTime: target.Add(-500 * time.Millisecond), ReceiptTime: *now,
		TimestampBasis: "participant", ConditionsClassified: true, IdentityClassified: true, Lifecycle: "original",
		Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 101}}
	admission, completion := runtime.Engine().AdmitTrade(context.Background(), base)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
		t.Fatal("qualifying original trade was not applied")
	}
	repeat := base
	repeat.Conditions = []int64{15}
	repeat.Live.FrameSequence = 102
	admission, completion = runtime.Engine().AdmitTrade(context.Background(), repeat)
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQRejected {
		t.Fatal("unequal repeat was not rejected")
	}
}

func shedTapeForPressure(t *testing.T, runtime *operations.Runtime, _ reference.Binding, now *time.Time) {
	t.Helper()
	sample := engine.TQPressureSample{WaitingFrames: 10, FrameCapacity: 100, ByteCapacity: 1_000, DeliveryLatencyAttributed: true, TQLocalAccountingHealthy: true, Goroutines: 1}
	for i := 0; i < 2; i++ {
		*now = now.Add(time.Second)
		admission, completion := runtime.Engine().AdmitTQPressureTick(context.Background())
		if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
			t.Fatal("pressure tick was not applied")
		}
		command, err := runtime.Engine().IssueTQPressureCommand()
		if err != nil {
			t.Fatal(err)
		}
		input, err := engine.NewTQPressureResultInput(command, sample)
		if err != nil {
			t.Fatal(err)
		}
		admission, completion = runtime.Engine().AdmitTQPressureResult(context.Background(), input)
		if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionTQApplied {
			t.Fatal("pressure sample was not applied")
		}
	}
}

func tapeDurationPointer(value time.Duration) *time.Duration {
	copyValue := value
	return &copyValue
}

func makeSnapshotReady(t *testing.T, owner *engine.Engine, binding reference.Binding, at time.Time) {
	t.Helper()
	ackAt := at.Add(-operations.DefaultConfig().EvaluationDelay)
	applySnapshotControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan not admitted")
	}
	plan := <-completion
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, terminalCompletion := owner.AdmitHydrationTerminal(context.Background(), terminal)
		result := <-terminalCompletion
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal = %+v", result)
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
	_, fenceCompletion := owner.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if result := <-fenceCompletion; result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence = %+v", result)
	}
}

func applySnapshotControl(t *testing.T, owner *engine.Engine, binding reference.Binding, kind engine.ConnectionControlKind, epoch, token uint64, position engine.LivePosition, at time.Time) {
	t.Helper()
	admission, completion := owner.AdmitConnectionControl(context.Background(), engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind, ConnectionEpoch: epoch, CommandToken: token, Position: position, ReceiptTime: at, Outcome: engine.ControlSucceeded})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("control %s not admitted", kind)
	}
	if result := <-completion; result.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("control %s = %+v", kind, result)
	}
}

func shutdownSnapshotRuntime(t *testing.T, runtime *operations.Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Errorf("runtime shutdown: %v", err)
	}
}

func strconvLen(value []byte) string { return strconv.Itoa(len(value)) }

type blockingWriter struct {
	header  http.Header
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{header: make(http.Header), started: make(chan struct{}), release: make(chan struct{})}
}

func (writer *blockingWriter) Header() http.Header { return writer.header }
func (writer *blockingWriter) WriteHeader(int)     {}
func (writer *blockingWriter) Write(value []byte) (int, error) {
	writer.once.Do(func() { close(writer.started) })
	<-writer.release
	return len(value), nil
}
