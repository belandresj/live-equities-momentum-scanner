package snapshotapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestPLBRA3ParallelHydrationAPIEquivalence closes the API-v2 boundary of
// P-LBR-A3-PARALLEL-HYDRATION. The ordinary production-composition proof owns
// the blocking worker pool and live-tail assertions; this mapper-boundary
// proof independently forces the same hydration facts through out-of-plan
// terminal orders at every supported worker count, then captures, maps, and
// JSON-encodes the resulting API-v2 product.
func TestPLBRA3ParallelHydrationAPIEquivalence(t *testing.T) {
	binding := snapshotBinding(t, "AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG", "HHH")
	var baseline []byte
	for _, workers := range []int{1, 2, 4, 8} {
		t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
			body := lbrA3MappedAPIProduct(t, binding, workers)
			if baseline == nil {
				baseline = body
				return
			}
			if string(body) != string(baseline) {
				t.Fatalf("workers=%d changed mapped API-v2 product\none=%s\ngot=%s", workers, baseline, body)
			}
		})
	}
}

func lbrA3MappedAPIProduct(t *testing.T, binding reference.Binding, workers int) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := binding.SessionStart().Add(8 * time.Hour).UTC()
	runtime, err := operations.New(ctx, binding, operations.DefaultConfig(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownSnapshotRuntime(t, runtime)

	ackAt := now.Add(-operations.DefaultConfig().EvaluationDelay)
	applySnapshotControl(t, runtime.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, runtime.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, runtime.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)

	population := len(binding.UniverseSymbols())
	budgets := engine.HydrationPlanBudgets{
		Workers: workers, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20,
		MaximumNormalizedRecords: int64(population) * 57_600,
		MaximumResidentRecords:   int64(workers) * 57_600,
	}
	admission, completion := runtime.Engine().AdmitHydrationPlan(ctx, engine.HydrationPlanInput{
		SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(),
		Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets,
	})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan not admitted")
	}
	plan := <-completion
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) != population {
		t.Fatalf("hydration plan=%+v requests=%d", plan, len(requests))
	}

	order := make([]int, len(requests))
	for index := range order {
		order[index] = index
	}
	if workers > 1 {
		for left, right := 0, len(order)-1; left < right; left, right = left+1, right-1 {
			order[left], order[right] = order[right], order[left]
		}
	}
	values := engine.AggregateValues{Open: 12, High: 13, Low: 11, Close: 12.5, Volume: 1000, VWAP: 12.25, AverageTradeSize: 10, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades}
	var fence engine.HydrationFenceCommand
	for _, requestIndex := range order {
		request := requests[requestIndex]
		state := engine.HydrationCompletedEmpty
		rows, chunks, emittedRows := int64(0), int64(0), int64(0)
		if requestIndex%2 == 0 {
			row, rowErr := engine.NewHydrationRow(request.Symbol(), request.Start(), request.Start().Add(time.Second), values)
			if rowErr != nil {
				t.Fatal(rowErr)
			}
			chunk, chunkErr := engine.NewHydrationChunkInput(request, request.ResultID(), 0, 1, 0, 1, []engine.HydrationRow{row})
			if chunkErr != nil {
				t.Fatal(chunkErr)
			}
			chunkAdmission, chunkCompletion := runtime.Engine().AdmitHydrationChunk(ctx, chunk)
			if chunkAdmission != engine.AdmissionAdmitted || chunkCompletion == nil || (<-chunkCompletion).Code != engine.DispositionHydrationChunkApplied {
				t.Fatalf("hydration chunk for %s not applied", request.Symbol())
			}
			state, rows, chunks, emittedRows = engine.HydrationCompletedValue, 1, 1, 1
		}
		terminal, terminalErr := engine.NewHydrationTerminalInput(request, request.ResultID(), state, engine.HydrationReasonNone, 1, 1, 100, rows, chunks, emittedRows)
		if terminalErr != nil {
			t.Fatal(terminalErr)
		}
		terminalAdmission, terminalCompletion := runtime.Engine().AdmitHydrationTerminal(ctx, terminal)
		if terminalAdmission != engine.AdmissionAdmitted || terminalCompletion == nil {
			t.Fatalf("hydration terminal for %s not admitted", request.Symbol())
		}
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
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 1, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	fenceAdmission, fenceCompletion := runtime.Engine().AdmitAggregateIngressFence(ctx, fenceInput)
	if fenceAdmission != engine.AdmissionAdmitted || fenceCompletion == nil || (<-fenceCompletion).Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatal("hydration fence not applied")
	}

	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := Map(capture)
	if err != nil {
		t.Fatalf("map API-v2 snapshot: %v", err)
	}
	if !mapped.Status.BackendReady || !mapped.Status.RankingCurrent || mapped.Publication.Lifecycle != "live" || !mapped.Recovery.FenceReconciled {
		t.Fatalf("mapped product is not current: publication=%+v status=%+v recovery=%+v", mapped.Publication, mapped.Status, mapped.Recovery)
	}
	// Heap gauges and the process-wide goroutine sample are observation-time
	// diagnostics, not API product semantics. Normalize only those three fields
	// before comparing the otherwise complete encoded v2 response.
	mapped.Operations.HeapAllocBytes = ""
	mapped.Operations.HeapInUseBytes = ""
	mapped.Operations.Goroutines = 0
	body, err := json.Marshal(mapped)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
