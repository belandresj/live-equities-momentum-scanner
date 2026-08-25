package operations

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

func TestFenceIncidentProjectionRetainsAccountingWithoutSymbols(t *testing.T) {
	evaluation := validProjectedEvaluation()
	watermark := evaluation.At
	publication := engine.PublicationView{
		PublicationID: 9, LastEngineSequence: 81, BindingIdentity: "binding", TradingDate: "2026-08-12", RunMode: "live",
		Lifecycle: "live", Watermark: &watermark, GeneratedAt: watermark, CurrentMarketClaim: true, AggregateEvaluation: evaluation,
	}
	projection, invalid := lastCoherentProjection(publication)
	if invalid || projection == nil || projection.PublicationID != 9 || projection.RankingRows != 1 || len(projection.Evaluation.Rows) != 0 ||
		projection.Evaluation.PopulationTransition.TrustedByLaterLiveMark != 1 || !engine.ValidateEvaluationAccounting(projection.Evaluation) {
		t.Fatalf("projection=%+v invalid=%t", projection, invalid)
	}
	encoded, err := json.Marshal(IngressIncident{LastCoherentProjection: projection})
	if err != nil || strings.Contains(string(encoded), "AAA") {
		t.Fatalf("bounded projection json=%s err=%v", encoded, err)
	}
	if absent, absentInvalid := lastCoherentProjection(engine.PublicationView{}); absent != nil || absentInvalid {
		t.Fatalf("pre-publication control=%+v invalid=%t", absent, absentInvalid)
	}

	broken := publication
	broken.AggregateEvaluation.Population.UniverseTotal++
	if got, gotInvalid := lastCoherentProjection(broken); got != nil || !gotInvalid {
		t.Fatalf("incoherent projection serialized=%+v invalid=%t", got, gotInvalid)
	}
}

func TestEngineTerminalIncidentRetainsPriorCoherentProjection(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	run, err := New(context.Background(), binding, DefaultConfig(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = run.Shutdown(context.Background()) }()
	ackAt := now.Add(-DefaultConfig().EvaluationDelay)
	applyControl(t, run.Engine(), binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, run.Engine(), binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	completeHydration(t, run.Engine(), binding, engine.HydrationFreshBootstrap, 1, now)
	prior := run.Engine().ObserveLastCoherentPublication()
	if prior.PublicationID == 0 {
		t.Fatal("missing coherent publication")
	}
	run.Engine().ArmEvaluatorAccountingFaultForTest()
	_, completion := run.Engine().AdmitTimer(context.Background())
	disposition := <-completion
	if disposition.Code != engine.DispositionAccountingIntegrity {
		t.Fatalf("terminal=%+v", disposition)
	}
	incident := run.FirstIngressIncident()
	if incident == nil || incident.Owner != IngressOwnerEngineTransition || incident.LastCoherentProjection == nil ||
		incident.LastCoherentProjection.PublicationID != prior.PublicationID || incident.LastCoherentProjectionInvalid {
		t.Fatalf("engine incident=%+v", incident)
	}
}

func validProjectedEvaluation() engine.EvaluationView {
	feature := engine.FeatureAccountingView{}
	feature.Statuses[0], feature.Reasons[0], feature.Pairs[0][0] = 1, 1, 1
	return engine.EvaluationView{
		At: time.Date(2026, 8, 12, 18, 44, 48, 0, time.UTC), Mode: "qualified_current",
		Population:           engine.PopulationView{UniverseTotal: 1, ValidPriorClose: 1, TrustedRankableMark: 1, CoveredPopulation: 1},
		Qualification:        engine.QualificationAccountingView{NotYetPassed: 1},
		Features:             engine.AllFeatureAccountingView{DayPercent: feature, SessionVolume: feature, FromOpenPercent: feature, DayRange: feature, Activity30s: feature, Move30s: feature},
		Floats:               engine.FloatAccountingView{Unavailable: 1},
		PopulationTransition: engine.PopulationTransitionDiagnosticView{BootstrapUnknown: 1, TrustedByLaterLiveMark: 1},
		KnownRankableCount:   1, Rows: []engine.RankingRowView{{Rank: 1, Symbol: "AAA"}},
	}
}
