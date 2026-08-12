package operations

import (
	"testing"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

func TestEvaluatorIntegrityInspectionIsDetached(t *testing.T) {
	failure := &engine.EvaluatorIntegrityView{Category: engine.EvaluatorSupportContradiction, EngineSequence: 7}
	capture := SnapshotCapture{sealed: &sealedSnapshotCapture{view: SnapshotCaptureView{
		Engine: engine.SnapshotView{Operational: engine.OperationalView{IntegrityFailure: failure}},
		Status: Status{IntegrityFailure: failure}, Metrics: Metrics{Engine: engine.OperationalView{IntegrityFailure: failure}},
	}}}
	first, ok := InspectSnapshotCapture(capture)
	if !ok {
		t.Fatal("sealed diagnostic capture unavailable")
	}
	first.Engine.Operational.IntegrityFailure.Category = engine.EvaluatorRankingRow
	first.Status.IntegrityFailure.Category = engine.EvaluatorRankingRow
	first.Metrics.Engine.IntegrityFailure.Category = engine.EvaluatorRankingRow
	second, ok := InspectSnapshotCapture(capture)
	if !ok || second.Engine.Operational.IntegrityFailure.Category != engine.EvaluatorSupportContradiction || second.Status.IntegrityFailure.Category != engine.EvaluatorSupportContradiction || second.Metrics.Engine.IntegrityFailure.Category != engine.EvaluatorSupportContradiction {
		t.Fatalf("inspection mutated sealed diagnostics: %+v", second)
	}
}
