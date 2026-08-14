package snapshotapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

// TestLiveFirstReadyResolvedDiscrepancyIsTransportable covers the production
// C10 boundary for a diagnostic-only REST/live discrepancy at the first ready
// hydration/fence publication.
func TestLiveFirstReadyResolvedDiscrepancyIsTransportable(t *testing.T) {
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

	ackAt := now.Add(-config.EvaluationDelay)
	applySnapshotControl(t, runtime.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, runtime.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, runtime.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)

	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 120, MaximumResidentRecords: 120}
	admission, completion := runtime.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan not admitted")
	}
	plan := <-completion
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) != 2 {
		t.Fatalf("hydration plan=%+v requests=%d", plan, len(requests))
	}

	values := func(price float64, provenance engine.ATSProvenance) engine.AggregateValues {
		return engine.AggregateValues{Open: price, High: price, Low: price, Close: price, Volume: 1000, VWAP: price, AverageTradeSize: 10, ATSProvenance: provenance}
	}
	// BBB's live fact intentionally precedes an unequal REST fact for the same
	// identity. Live remains canonical and exact; the API must still expose the
	// discrepancy through bounded hydration-row accounting.
	live := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: "BBB", WindowStart: requests[1].Start(), WindowEnd: requests[1].Start().Add(time.Second), Values: values(20, engine.ATSLiveProviderAverage),
		DeliveryTime: requests[1].Start().Add(time.Second), Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 2}}
	if admitted, completed := runtime.Engine().AdmitAggregate(ctx, live); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionAggregateInserted {
		t.Fatal("live conflict seed was not installed")
	}

	var fence engine.HydrationFenceCommand
	for _, request := range requests {
		price := 20.0
		if request.Symbol() == "BBB" {
			price = 10
		}
		row, rowErr := engine.NewHydrationRow(request.Symbol(), request.Start(), request.Start().Add(time.Second), values(price, engine.ATSRESTFloorVolumeOverTrades))
		if rowErr != nil {
			t.Fatal(rowErr)
		}
		chunk, chunkErr := engine.NewHydrationChunkInput(request, request.ResultID(), 0, 1, 0, 1, []engine.HydrationRow{row})
		if chunkErr != nil {
			t.Fatal(chunkErr)
		}
		if admitted, completed := runtime.Engine().AdmitHydrationChunk(ctx, chunk); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionHydrationChunkApplied {
			t.Fatalf("hydration chunk for %s failed", request.Symbol())
		}
		terminal, terminalErr := engine.NewHydrationTerminalInput(request, request.ResultID(), engine.HydrationCompletedValue, engine.HydrationReasonNone, 1, 1, 100, 1, 1, 1)
		if terminalErr != nil {
			t.Fatal(terminalErr)
		}
		_, terminalCompletion := runtime.Engine().AdmitHydrationTerminal(ctx, terminal)
		terminalResult := <-terminalCompletion
		if terminalResult.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal for %s=%+v", request.Symbol(), terminalResult)
		}
		if terminalResult.FenceCommand.CommandToken() != 0 {
			fence = terminalResult.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("terminal hydration did not allocate a fence")
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := runtime.Engine().AdmitAggregateIngressFence(ctx, fenceInput)
	if result := <-fenceCompletion; result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence=%+v", result)
	}

	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	mapped, mapErr := Map(capture)
	if mapErr != nil {
		t.Fatalf("first-ready capture rejected: %v", mapErr)
	}
	if !mapped.Status.BackendReady || !mapped.Status.RankingCurrent || mapped.Publication.Lifecycle != "live" ||
		mapped.Ranking.Mode != "qualified_current" || mapped.Ranking.Reason != "" || len(mapped.Rows) != 0 ||
		mapped.Accounting.Population.TrustedRankableMark != 2 || mapped.Accounting.Population.UnknownDueFailureOrFence != 0 ||
		mapped.Accounting.Qualification.NotYetPassed != 2 || mapped.Accounting.Qualification.Unresolved != 0 ||
		mapped.Recovery.Rows.Consumed != "2" || mapped.Recovery.Rows.Inserted != "1" || mapped.Recovery.Rows.ConflictOrWithdrawal != "1" {
		t.Fatalf("first-ready snapshot publication=%+v status=%+v ranking=%+v rows=%d", mapped.Publication, mapped.Status, mapped.Ranking, len(mapped.Rows))
	}

	server, err := Listen(runtime, ServerConfig{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	for _, path := range []string{"/api/v2/snapshot", "/readyz"} {
		response, requestErr := client.Get("http://" + server.Address() + path)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		var body json.RawMessage
		decodeErr := json.NewDecoder(response.Body).Decode(&body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || decodeErr != nil || !json.Valid(body) {
			t.Fatalf("%s=%d decode=%v body=%s", path, response.StatusCode, decodeErr, body)
		}
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	if err := server.Shutdown(shutdown); err != nil {
		cancelShutdown()
		t.Fatal(err)
	}
	cancelShutdown()
	select {
	case err := <-server.Done():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot server did not stop")
	}
}
