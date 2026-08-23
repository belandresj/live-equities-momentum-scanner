package operations

import (
	"testing"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

func TestPLBRA2OneWorkerLiveComposition(t *testing.T) {
	base := LiveComponents{Adapter: &massive.LiveAdapter{}, Hydrator: &massive.HydrationWorker{}, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1, MaximumNormalizedRecords: 1, MaximumResidentRecords: 1}
	if err := ValidateLiveComponents(base); err != nil {
		t.Fatalf("one-worker fresh-start composition was rejected: %v", err)
	}
	multi := base
	multi.Workers = 2
	if ValidateLiveComponents(multi) == nil {
		t.Fatal("ordinary live composition retained multiworker topology")
	}
	checkpointed := base
	checkpointed.Store = &checkpoint.Store{}
	if ValidateLiveComponents(checkpointed) == nil {
		t.Fatal("ordinary live composition retained checkpoint-driven installation")
	}
}
