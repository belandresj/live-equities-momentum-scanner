package operations

import (
	"testing"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

func TestPLBRA3BoundedParallelLiveComposition(t *testing.T) {
	base := LiveComponents{Adapter: &massive.LiveAdapter{}, Hydrator: &massive.HydrationWorker{}, Workers: 1, RowsPerChunk: 1,
		MaximumResponseBytes: 1, MaximumNormalizedRecords: 1, MaximumResidentRecords: massive.HydrationMaximumRows}
	for _, workers := range []int{1, 2, 4, 8} {
		candidate := base
		candidate.Workers = workers
		candidate.MaximumResidentRecords = int64(workers) * massive.HydrationMaximumRows
		if err := ValidateLiveComponents(candidate); err != nil {
			t.Fatalf("workers=%d fresh-start composition was rejected: %v", workers, err)
		}
	}
	for _, workers := range []int{0, 3, 5, 6, 7, 9} {
		candidate := base
		candidate.Workers = workers
		if ValidateLiveComponents(candidate) == nil {
			t.Fatalf("unsupported workers=%d reached ordinary live runtime", workers)
		}
	}
	oversized := base
	oversized.MaximumResidentRecords++
	if ValidateLiveComponents(oversized) == nil {
		t.Fatal("ordinary live composition accepted oversized resident budget")
	}
	checkpointed := base
	checkpointed.Store = &checkpoint.Store{}
	if ValidateLiveComponents(checkpointed) == nil {
		t.Fatal("ordinary live composition retained checkpoint-driven installation")
	}
}
