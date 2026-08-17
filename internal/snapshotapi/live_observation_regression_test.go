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
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
	var snapshot Snapshot
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &snapshot) != nil {
		t.Fatalf("warm-up snapshot=%d %s", response.Code, response.Body.String())
	}
	if snapshot.Publication.BindingIdentity != binding.Identity() || snapshot.Publication.Lifecycle != "hydrating" || !snapshot.Publication.AggregateAcknowledged ||
		snapshot.Status.BackendReady || snapshot.Status.ReadinessReason != "fence_pending" || snapshot.Ranking.Mode != "unavailable" ||
		snapshot.Ranking.Reason != "no_committed_watermark" || !snapshot.Recovery.GenerationActive || snapshot.Recovery.Work.Open == "0" {
		t.Fatalf("warm-up publication=%+v status=%+v ranking=%+v recovery=%+v", snapshot.Publication, snapshot.Status, snapshot.Ranking, snapshot.Recovery)
	}
}

// Possible aggregate loss after a committed watermark routes directly to exact
// gap recovery. The API stays coherent and bound but exposes no current claim
// or ranking rows until the unsupported suffix has been rehydrated and fenced.
func TestRecoverableIngressLossSnapshotIsServableBoundAndNoncurrent(t *testing.T) {
	run, binding, now := newSnapshotRuntime(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	})
	makeSnapshotReady(t, run.Engine(), binding, *now)

	admission, completion := run.Engine().AdmitOperationalIngressIntegrity(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil || (<-completion).Code != engine.DispositionIngressIntegrity {
		t.Fatal("ingress failure was not contained")
	}
	_, timer := run.Engine().AdmitTimer(context.Background())
	if timer == nil || (<-timer).Code != engine.DispositionTimerApplied {
		t.Fatal("recovering timer was not dispositioned")
	}

	handler, err := NewHandler(run, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
	var snapshot Snapshot
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &snapshot) != nil {
		t.Fatalf("recovering snapshot=%d %s", response.Code, response.Body.String())
	}
	if snapshot.Publication.BindingIdentity != binding.Identity() || snapshot.Publication.Lifecycle != "recovering" ||
		snapshot.Publication.LifecycleReason != "aggregate_epoch_lost" || snapshot.Publication.Suppression != "" ||
		snapshot.Status.BackendReady || snapshot.Status.RankingCurrent || snapshot.Status.ReadinessReason != "lifecycle_not_ready" ||
		snapshot.Ranking.Mode != "unavailable" || snapshot.Ranking.Reason != "no_committed_watermark" || snapshot.Recovery.GenerationActive || len(snapshot.Rows) != 0 {
		t.Fatalf("recovering publication=%+v status=%+v ranking=%+v rows=%d", snapshot.Publication, snapshot.Status, snapshot.Ranking, len(snapshot.Rows))
	}
}
