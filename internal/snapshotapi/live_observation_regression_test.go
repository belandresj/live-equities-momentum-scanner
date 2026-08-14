package snapshotapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

// Regression for the 2026-08-11 live observation: an installed, acknowledged
// binding is a valid noncurrent product publication while its hydration ledger
// is still open. Warm-up is not an unavailable API transport.
func TestLiveWarmupSnapshotIsServableAndBound(t *testing.T) {
	run, binding, now := newSnapshotRuntime(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	})

	ackAt := now.Add(-operations.DefaultConfig().EvaluationDelay)
	applySnapshotControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, run.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, run.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := run.Engine().AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{
		SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets,
	})
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionHydrationPlanApplied {
		t.Fatal("hydration plan was not installed")
	}

	handler, err := NewHandler(run, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil))
	var snapshot Snapshot
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &snapshot) != nil {
		t.Fatalf("warm-up snapshot=%d %s", response.Code, response.Body.String())
	}
	if snapshot.Publication.BindingIdentity != binding.Identity() || snapshot.Publication.Lifecycle != "hydrating" || snapshot.Publication.AggregateAcknowledged != true ||
		snapshot.Status.BackendReady || snapshot.Status.ReadinessReason != "fence_pending" || snapshot.Ranking.Mode != "unavailable" ||
		snapshot.Ranking.Reason != "no_committed_watermark" || snapshot.Recovery.Work.Open == "0" {
		t.Fatalf("warm-up publication=%+v status=%+v ranking=%+v recovery=%+v", snapshot.Publication, snapshot.Status, snapshot.Ranking, snapshot.Recovery)
	}
}
