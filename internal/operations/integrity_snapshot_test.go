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

func TestFloatPercentageInspectionIsDetached(t *testing.T) {
	percent := 25.0
	capture := SnapshotCapture{sealed: &sealedSnapshotCapture{view: SnapshotCaptureView{Engine: engine.SnapshotView{Publication: engine.PublicationView{
		AggregateEvaluation: engine.EvaluationView{Rows: []engine.RankingRowView{{Symbol: "AAA", Float: engine.FloatFieldView{Percent: &percent}}}},
	}}}}}
	first, ok := InspectSnapshotCapture(capture)
	if !ok || first.Engine.Publication.AggregateEvaluation.Rows[0].Float.Percent == nil {
		t.Fatal("sealed Float capture unavailable")
	}
	*first.Engine.Publication.AggregateEvaluation.Rows[0].Float.Percent = 99
	second, ok := InspectSnapshotCapture(capture)
	if !ok || second.Engine.Publication.AggregateEvaluation.Rows[0].Float.Percent == nil || *second.Engine.Publication.AggregateEvaluation.Rows[0].Float.Percent != 25 {
		t.Fatalf("inspection mutated sealed Float percentage: %+v", second.Engine.Publication.AggregateEvaluation.Rows)
	}
}
