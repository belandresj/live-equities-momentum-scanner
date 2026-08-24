package engine

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func TestPLBRB3LegacyCheckpointPayloadsRejected(t *testing.T) {
	at := time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC)
	e := evaluatorProofEngine(at, []evaluatorSymbol{{"AAA", reference.PriorCloseValid, 10, 12, qualificationFinalized}})
	e.state.aggregateEvaluator.current = e.stageAggregateEvaluationAtLocked(at, at)
	if validation := validateAggregateEvaluation(e.state.aggregateEvaluator.current); validation != nil {
		t.Fatalf("base evaluation: %v", validation)
	}
	e.installInitialPublication()
	projected := e.projectCheckpointLocked(at)
	if projected.Disposition != CheckpointProjected || len(projected.Image.Symbols) != 1 {
		t.Fatalf("base checkpoint projection=%+v", projected)
	}

	tests := map[string]func(*checkpoint.Image){
		"non-nil empty legacy activity": func(image *checkpoint.Image) {
			image.Symbols[0].Activity = &checkpoint.Activity{}
		},
		"nonempty legacy activity": func(image *checkpoint.Image) {
			image.Symbols[0].Activity = &checkpoint.Activity{References: []checkpoint.ActivitySummary{{End: at.Unix()}}}
		},
		"legacy highs only": func(image *checkpoint.Image) {
			ensureCheckpointPriceRange(image)
			start := at.Add(-2 * time.Second).Unix()
			image.Symbols[0].PriceRange.Highs = []checkpoint.ExtremaPoint{{WindowStart: start, Value: 12.5}}
		},
		"legacy lows only": func(image *checkpoint.Image) {
			ensureCheckpointPriceRange(image)
			start := at.Add(-2 * time.Second).Unix()
			image.Symbols[0].PriceRange.Lows = []checkpoint.ExtremaPoint{{WindowStart: start, Value: 11.5}}
		},
		"legacy rolling floor only": func(image *checkpoint.Image) {
			ensureCheckpointPriceRange(image)
			image.Symbols[0].PriceRange.RollingFloor = at.Add(-time.Minute).Unix()
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			image := projected.Image.Clone()
			mutate(&image)
			image.Counts = checkpointStructureCounts(image)
			candidate := checkpoint.Candidate{Image: image, Checksum: stringsOfZeroes(64), Integrity: true}
			if boundedCheckpointCandidate(image) {
				t.Fatal("legacy candidate passed bounded validation")
			}
			bindingBefore := e.state.binding
			evaluationBefore := cloneAggregateEvaluation(e.state.aggregateEvaluator.current)
			publicationBefore := e.publication.Load()
			installedBefore := e.state.installedCheckpoint
			result := e.installCheckpointLocked(candidate)
			if result.Disposition != CheckpointInvalid {
				t.Fatalf("direct install disposition=%+v", result)
			}
			if admission, completion := e.AdmitCheckpointInstall(context.Background(), candidate); admission != AdmissionNotAdmittedInvalid || completion != nil {
				t.Fatalf("admission=%s completion=%v", admission, completion)
			}
			if e.state.binding != bindingBefore || e.state.installedCheckpoint != installedBefore ||
				!aggregateEvaluationEqual(e.state.aggregateEvaluator.current, evaluationBefore) || e.publication.Load() != publicationBefore {
				t.Fatal("legacy checkpoint mutated canonical state or current publication")
			}
		})
	}
}

func ensureCheckpointPriceRange(image *checkpoint.Image) {
	if image.Symbols[0].PriceRange == nil {
		image.Symbols[0].PriceRange = &checkpoint.PriceRange{}
	}
}

func stringsOfZeroes(count int) string {
	value := make([]byte, count)
	for index := range value {
		value[index] = '0'
	}
	return string(value)
}
