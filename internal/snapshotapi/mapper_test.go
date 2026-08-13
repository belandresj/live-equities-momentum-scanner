package snapshotapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func TestPC10SchemaGoldenIdentityAndSemanticMutations(t *testing.T) {
	capture := schemaCapture()
	snapshot, err := mapCaptureView(capture)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	const goldenSHA256 = "fe91d17daaacd05eadd1f8af6ccf3db4cfa4b6e49eea2580574beaec36077862"
	if got := hex.EncodeToString(hash[:]); got != goldenSHA256 {
		t.Fatalf("snapshot golden SHA-256 = %s", got)
	}
	if snapshot.Sample.ID != "9007199254740999" || snapshot.Publication.ID != "9007199254741001" || snapshot.Publication.AggregateAckPosition.ArrayIndex != 7 ||
		snapshot.Publication.HydrationFence.MarkerOrdinal != "9" || snapshot.Status.WatermarkLagMS == nil || *snapshot.Status.WatermarkLagMS != 0 || len(snapshot.Rows) != 1 {
		t.Fatalf("identity/fence/zero mapping = %+v", snapshot)
	}
	if snapshot.Accounting.PopulationTransitionDiagnostic != (PopulationTransitionDiagnostic{BootstrapUnknown: 1, TrustedByLaterLiveMark: 1}) {
		t.Fatalf("population-transition diagnostic mapping = %+v", snapshot.Accounting.PopulationTransitionDiagnostic)
	}
	latency := snapshot.Operations.DeliveryLatencyAttribution
	if !sumDecimalEquals(snapshot.Operations.Deliveries, latency.Aggregate, latency.TQ, latency.Control, latency.HydrationFence, latency.Checkpoint, latency.Timer, latency.Unknown) ||
		latency.MaximumMS != snapshot.Operations.MaxProcessingDelayOneSecondMS || latency.MaximumFamily != "aggregate" {
		t.Fatalf("delivery-latency attribution mapping = %+v operations=%+v", latency, snapshot.Operations)
	}
	badWindowState := capture
	badWindowState.Metrics.DeliveryLatencyAttribution.WindowNonempty = false
	if _, err := mapCaptureView(badWindowState); mappingInvariant(err) != "delivery_latency_attribution" {
		t.Fatalf("incoherent private latency window state = %v", err)
	}
	badLatencyCount := snapshot
	badLatencyCount.Operations.DeliveryLatencyAttribution.Unknown = "2"
	if err := validateSnapshot(badLatencyCount); err == nil {
		t.Fatal("incoherent delivery-latency count serialized")
	}
	badLatencyPair := snapshot
	badLatencyPair.Operations.DeliveryLatencyAttribution.MaximumMS++
	if err := validateSnapshot(badLatencyPair); err == nil {
		t.Fatal("split delivery-latency maximum pair serialized")
	}
	badLatencyFamily := snapshot
	badLatencyFamily.Operations.DeliveryLatencyAttribution.MaximumFamily = "symbol:AAA"
	if err := validateSnapshot(badLatencyFamily); err == nil {
		t.Fatal("unbounded delivery-latency family serialized")
	}
	badLatencyWinner := snapshot
	badLatencyWinner.Operations.DeliveryLatencyAttribution.MaximumFamily = "timer"
	badLatencyWinner.Operations.DeliveryLatencyAttribution.Timer = "0"
	badLatencyWinner.Operations.DeliveryLatencyAttribution.Unknown = "2"
	if err := validateSnapshot(badLatencyWinner); err == nil {
		t.Fatal("maximum family without a matching delivery serialized")
	}
	row := snapshot.Rows[0]
	if row.From4AMChange.ValueRatio == nil || *row.From4AMChange.ValueRatio != 0 || row.HODDrawdown.ValueRatio != nil ||
		row.TapeRate.OneSecond.TradesPerSecond == nil || *row.TapeRate.OneSecond.TradesPerSecond != 0 || row.Spread.Cents == nil || *row.Spread.Cents != 0 {
		t.Fatalf("null versus genuine zero = %+v", row)
	}
	if row.DayChangeRatio != .0025 || row.DayRangePosition.ValueRatio == nil || *row.DayRangePosition.ValueRatio != .005 ||
		row.Activity.ValueRatio == nil || *row.Activity.ValueRatio != .0125 {
		t.Fatalf("percentage-point to ratio mapping = %+v", row)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"schema_version", "sample", "publication", "status", "ranking", "rows", "accounting", "recovery", "tq", "checkpoint", "operations"} {
		if _, ok := fields[name]; !ok {
			t.Fatalf("missing normative root field %q", name)
		}
	}

	for _, mutation := range []struct {
		name string
		edit func(*operations.SnapshotCaptureView)
	}{
		{"population", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Population.ValidPriorClose++
		}},
		{"population transition diagnostic", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.PopulationTransition.NoLaterEligibleMark++
		}},
		{"hydration work planned", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.Planned++ }},
		{"hydration work open", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.Open++ }},
		{"hydration work value", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.CompletedValue++ }},
		{"hydration work empty", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.CompletedEmpty++ }},
		{"hydration work failed", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.Failed++ }},
		{"hydration work canceled", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.Canceled++ }},
		{"hydration work fenced", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Accounting.Fenced++ }},
		{"hydration rows consumed", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Consumed++ }},
		{"hydration rows inserted", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Inserted++ }},
		{"hydration rows duplicate", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Duplicate++ }},
		{"hydration rows conflict", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.ConflictOrWithdrawal++ }},
		{"hydration rows rejected", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Rejected++ }},
		{"hydration rows fenced", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Fenced++ }},
		{"hydration rows integrity", func(v *operations.SnapshotCaptureView) { v.Engine.Operational.Hydration.Rows.Integrity++ }},
		{"TQ facts", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Accounting.Rejected++ }},
		{"TQ family facts", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Accounting.AppliedTrades = 2 }},
		{"TQ commands", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Commands.Failed++ }},
		{"checkpoint", func(v *operations.SnapshotCaptureView) { v.Metrics.CheckpointEngine.Failed++ }},
		{"checkpoint projection gauge", func(v *operations.SnapshotCaptureView) {
			v.Metrics.CheckpointEngine.Eligible = 3
			v.Metrics.CheckpointEngine.ProjectionStarted = 3
			v.Metrics.CheckpointEngine.ProjectionInProgress = 2
		}},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := schemaCapture()
			mutation.edit(&changed)
			if _, err := mapCaptureView(changed); err == nil {
				t.Fatal("incoherent accounting serialized")
			}
		})
	}
	badPressureCause := schemaCapture()
	badPressureCause.Engine.TQ.PressureCause = engine.TQPressureCauseWaitingFrames
	if _, err := mapCaptureView(badPressureCause); mappingInvariant(err) != "tq_pressure_cause" {
		t.Fatalf("normal pressure accepted nonempty cause: %v", err)
	}

	changedFence := schemaCapture()
	changedFence.Engine.Operational.Connection.AckPosition.ArrayIndex = 8
	changedFence.Engine.Operational.Hydration.FenceMarkerOrdinal = 10
	changed, err := mapCaptureView(changedFence)
	if err != nil {
		t.Fatal(err)
	}
	zeroMarker := schemaCapture()
	zeroMarker.Engine.Operational.Hydration.FenceMarkerOrdinal = 0
	if _, err := mapCaptureView(zeroMarker); err == nil {
		t.Fatal("reconciled zero-marker hydration fence serialized")
	}
	if changed.Publication.AggregateAckPosition.FrameSequence != snapshot.Publication.AggregateAckPosition.FrameSequence ||
		changed.Publication.AggregateAckPosition.ArrayIndex == snapshot.Publication.AggregateAckPosition.ArrayIndex ||
		changed.Publication.HydrationFence.MarkerOrdinal == snapshot.Publication.HydrationFence.MarkerOrdinal {
		t.Fatal("unequal same-frame/fence marker collapsed")
	}

	withoutWatermark := schemaCapture()
	withoutWatermark.Engine.Publication.Watermark = nil
	withoutWatermark.Engine.Operational.Watermark = nil
	withoutWatermark.Engine.Publication.AggregateEvaluation.Mode = "unavailable"
	withoutWatermark.Engine.Publication.AggregateEvaluation.Reason = "no_committed_watermark"
	withoutWatermark.Engine.Publication.AggregateEvaluation.Rows = nil
	withoutWatermark.Engine.TQ.Desired, withoutWatermark.Engine.TQ.Rows = nil, nil
	withoutWatermark.Status.Watermark = nil
	missing, err := mapCaptureView(withoutWatermark)
	if err != nil || missing.Status.WatermarkLagMS != nil || missing.Publication.CommittedT != nil {
		t.Fatalf("missing watermark fabricated lag: snapshot=%+v err=%v", missing, err)
	}
}

func TestPC10SchemaPercentagePointBoundariesMapToRatios(t *testing.T) {
	capture := schemaCapture()
	row := &capture.Engine.Publication.AggregateEvaluation.Rows[0]
	row.DayPercent = 250
	row.From4AMPercent = engine.ReplayFieldView{Status: "current", Value: -100}
	row.HODDrawdown = engine.ReplayFieldView{Status: "current", Value: 0}
	row.SessionRange = engine.ReplayFieldView{Status: "current", Value: 100}
	row.Rolling30 = engine.ReplayFieldView{Status: "current", Value: 25}
	row.Rolling60 = engine.ReplayFieldView{Status: "current", Value: 75}
	row.Activity = engine.ReplayFieldView{Status: "current", Value: 100}

	snapshot, err := mapCaptureView(capture)
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot.Rows[0]
	if got.DayChangeRatio != 2.5 || *got.From4AMChange.ValueRatio != -1 || *got.HODDrawdown.ValueRatio != 0 ||
		*got.DayRangePosition.ValueRatio != 1 || *got.Range30MPosition.ValueRatio != .25 ||
		*got.Range60MPosition.ValueRatio != .75 || *got.Activity.ValueRatio != 1 {
		t.Fatalf("boundary ratios = %+v", got)
	}
}

func TestPC10SchemaSampleIdentityAndMutationIsolation(t *testing.T) {
	firstCapture := schemaCapture()
	first, err := mapCaptureView(firstCapture)
	if err != nil {
		t.Fatal(err)
	}
	secondCapture := schemaCapture()
	secondCapture.SampleID++
	secondCapture.SampledAt = secondCapture.SampledAt.Add(3 * time.Second)
	secondCapture.ProcessLive = false
	secondCapture.Status.ProcessLive = false
	secondCapture.Status.SampledAt = secondCapture.SampledAt
	secondCapture.Status.BackendReady = false
	secondCapture.Status.Reason = operations.ReasonRuntimeUnavailable
	secondCapture.Metrics.SampledAt = secondCapture.SampledAt
	second, err := mapCaptureView(secondCapture)
	if err != nil {
		t.Fatal(err)
	}
	firstEngine, _ := json.Marshal(struct {
		Publication Publication
		Ranking     Ranking
		Rows        []Row
		Accounting  Accounting
		Recovery    Recovery
		TQ          TQ
	}{first.Publication, first.Ranking, first.Rows, first.Accounting, first.Recovery, first.TQ})
	secondEngine, _ := json.Marshal(struct {
		Publication Publication
		Ranking     Ranking
		Rows        []Row
		Accounting  Accounting
		Recovery    Recovery
		TQ          TQ
	}{second.Publication, second.Ranking, second.Rows, second.Accounting, second.Recovery, second.TQ})
	if !bytes.Equal(firstEngine, secondEngine) || first.Sample.ID == second.Sample.ID || first.Status.ProcessLive == second.Status.ProcessLive {
		t.Fatalf("sample identity overlaid engine facts: first=%s second=%s", firstEngine, secondEngine)
	}
	secondCapture.Engine.TQ.Desired[0] = "MUTATED"
	secondCapture.Engine.TQ.Rows[0].Tape.OneSecond = 99
	if first.TQ.DesiredSymbols[0] != "AAA" || first.Rows[0].TapeRate.OneSecond.TradesPerSecond == nil || *first.Rows[0].TapeRate.OneSecond.TradesPerSecond != 0 {
		t.Fatal("mapped snapshot retained mutable aliases")
	}
}

func TestPC10SchemaPublicationStateCorpus(t *testing.T) {
	tests := []struct {
		name     string
		edit     func(*operations.SnapshotCaptureView)
		wantMode string
		wantRows int
	}{
		{"qualified current", func(*operations.SnapshotCaptureView) {}, "qualified_current", 1},
		{"checkpoint projection in progress", func(v *operations.SnapshotCaptureView) {
			v.Metrics.CheckpointEngine.Eligible++
			v.Metrics.CheckpointEngine.ProjectionStarted++
			v.Metrics.CheckpointEngine.ProjectionInProgress = 1
		}, "qualified_current", 1},
		{"exact empty", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Rows = nil
			v.Engine.Publication.AggregateEvaluation.TotalPassers = 0
			v.Engine.TQ.Desired, v.Engine.TQ.Rows = nil, nil
		}, "qualified_current", 0},
		{"degraded", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Mode = "degraded_bootstrap"
			v.Engine.Publication.AggregateEvaluation.Reason = "incomplete_population"
			v.Status.BackendReady, v.Status.RankingCurrent = false, false
			v.Status.Reason = operations.ReasonRankingNoncurrent
		}, "degraded_bootstrap", 1},
		{"degraded current", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Mode = "degraded_current"
			v.Engine.Publication.AggregateEvaluation.Reason = "qualification_incomplete"
			v.Engine.Publication.AggregateEvaluation.TotalPassers = 0
			v.Engine.Publication.AggregateEvaluation.Qualification = engine.ReplayQualificationAccountingView{Unresolved: 1}
			v.Engine.Publication.AggregateEvaluation.Uncertainty = engine.ReplayUncertaintyView{LocalInvalid: 1}
			v.Engine.Publication.AggregateEvaluation.Rows[0].TQIntentEligible = false
			v.Engine.TQ.Desired, v.Engine.TQ.Rows = nil, nil
			v.Engine.TQ.Accounting.KnownPresent = 0
			v.Engine.TQ.Commands = engine.TQCommandAccountingView{}
			v.Status.BackendReady, v.Status.RankingCurrent = true, true
			v.Status.Reason = operations.ReasonNone
		}, "degraded_current", 1},
		{"pressure shed", func(v *operations.SnapshotCaptureView) {
			v.Engine.TQ.Pressure = engine.TQPressureDegraded
			v.Engine.TQ.PressureCause = engine.TQPressureCauseWaitingFrames
			v.Engine.TQ.ShedTradesQuotes = true
			v.Engine.TQ.Rows[0].Tape = engine.TapeRateView{Status: engine.TQPressureShed, OneSecondStatus: engine.TQPressureShed, FiveSecondStatus: engine.TQPressureShed}
			v.Engine.TQ.Rows[0].Spread = engine.SpreadView{Status: engine.TQPressureShed}
		}, "qualified_current", 1},
		{"stale", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Mode = "stale"
			v.Status.BackendReady, v.Status.RankingCurrent = false, false
			v.Status.Reason = operations.ReasonRankingNoncurrent
		}, "stale", 1},
		{"suppressed", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.Lifecycle, v.Engine.Operational.Lifecycle = "suppressed", "suppressed"
			v.Engine.Publication.Suppression = engine.SuppressionRestartRequired
			v.Engine.Publication.AggregateEvaluation.Mode = "suppressed"
			v.Engine.Publication.AggregateEvaluation.Reason = "global_suppression"
			v.Status.BackendReady, v.Status.RankingCurrent = false, false
			v.Status.Reason = operations.ReasonSuppressed
		}, "suppressed", 1},
		{"ended", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.Lifecycle, v.Engine.Operational.Lifecycle = "ended", "ended"
			v.Engine.Publication.LifecycleReason = "session_end"
			v.Status.BackendReady, v.Status.RankingCurrent = false, false
			v.Status.Reason = operations.ReasonLifecycle
		}, "qualified_current", 1},
		{"unavailable", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.Watermark, v.Engine.Operational.Watermark, v.Status.Watermark = nil, nil, nil
			v.Engine.Publication.AggregateEvaluation.Mode = "unavailable"
			v.Engine.Publication.AggregateEvaluation.Reason = "no_committed_watermark"
			v.Engine.Publication.AggregateEvaluation.Rows = nil
			v.Engine.TQ.Desired, v.Engine.TQ.Rows = nil, nil
			v.Status.BackendReady, v.Status.RankingCurrent = false, false
			v.Status.Reason = operations.ReasonWatermarkMissing
		}, "unavailable", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := schemaCapture()
			test.edit(&capture)
			snapshot, err := mapCaptureView(capture)
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.Ranking.Mode != test.wantMode || len(snapshot.Rows) != test.wantRows {
				t.Fatalf("mode/rows = %q/%d", snapshot.Ranking.Mode, len(snapshot.Rows))
			}
			if test.name == "checkpoint projection in progress" && snapshot.Checkpoint.ProjectionInProgress != "1" {
				t.Fatalf("active checkpoint projection=%+v", snapshot.Checkpoint)
			}
		})
	}
}

func TestPC10SchemaRejectsRepresentableInvalidShape(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*operations.SnapshotCaptureView)
	}{
		{"unknown lifecycle", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.Lifecycle, v.Engine.Operational.Lifecycle = "invented", "invented"
		}},
		{"duplicate row symbol", func(v *operations.SnapshotCaptureView) {
			row := v.Engine.Publication.AggregateEvaluation.Rows[0]
			row.Rank = 2
			v.Engine.Publication.AggregateEvaluation.Rows = append(v.Engine.Publication.AggregateEvaluation.Rows, row)
		}},
		{"nonfinite current value", func(v *operations.SnapshotCaptureView) {
			v.Engine.Publication.AggregateEvaluation.Rows[0].From4AMPercent.Value = math.NaN()
		}},
		{"sample overlay", func(v *operations.SnapshotCaptureView) { v.Status.ProcessLive = false }},
		{"foreign TQ publication", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.PublicationID++ }},
		{"later readiness sample", func(v *operations.SnapshotCaptureView) {
			v.Status.SampledAt = v.Status.SampledAt.Add(time.Second)
			v.Status.BackendReady = false
			v.Status.Reason = operations.ReasonWatermarkStale
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			capture := schemaCapture()
			test.edit(&capture)
			if _, err := mapCaptureView(capture); err == nil {
				t.Fatal("invalid capture serialized")
			}
		})
	}
}

func TestPC10SchemaPublicMapperRequiresSealedCapture(t *testing.T) {
	binding := snapshotBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runtime, err := operations.New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelShutdown()
		if err := runtime.Shutdown(shutdown); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	admission, completion := runtime.Engine().AdmitTimer(ctx)
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("initial evaluation timer was not admitted")
	}
	select {
	case <-ctx.Done():
		t.Fatal("initial evaluation timer timed out")
	case disposition := <-completion:
		if disposition.Code != engine.DispositionTimerApplied {
			t.Fatalf("initial evaluation timer = %+v", disposition)
		}
	}
	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	inspection, ok := operations.InspectSnapshotCapture(capture)
	if !ok {
		t.Fatal("sealed capture was not inspectable")
	}
	beforeSnapshot, beforeMutation := Map(capture)
	if beforeMutation != nil || beforeSnapshot.Ranking.Mode != "unavailable" || beforeSnapshot.Ranking.Reason != "no_committed_watermark" {
		t.Fatalf("bound noncurrent runtime capture did not map: snapshot=%+v err=%v", beforeSnapshot, beforeMutation)
	}
	inspection.Engine.TQ.PublicationID++
	inspection.Status.SampledAt = inspection.Status.SampledAt.Add(time.Second)
	afterSnapshot, afterMutation := Map(capture)
	if afterMutation != nil || !reflect.DeepEqual(afterSnapshot, beforeSnapshot) {
		t.Fatalf("detached inspection changed sealed mapper behavior: before=%+v/%v after=%+v/%v", beforeSnapshot, beforeMutation, afterSnapshot, afterMutation)
	}
	if _, err := Map(operations.SnapshotCapture{}); err == nil {
		t.Fatal("nil/unconstructed capture serialized")
	}
}

func TestMappingDiagnosticsIdentifyInvariantAndRemainBounded(t *testing.T) {
	invalid := schemaCapture()
	invalid.Engine.Publication.AggregateEvaluation.Population.ValidPriorClose++
	_, err := mapCaptureView(invalid)
	if err == nil || mappingInvariant(err) != "population_prior_close_identity" {
		t.Fatalf("mapper diagnostic=%v invariant=%q", err, mappingInvariant(err))
	}

	diagnostics := NewMappingDiagnostics()
	for sequence := uint64(1); sequence <= 100; sequence++ {
		diagnostics.record(MappingFailure{Invariant: mappingInvariant(err), Route: "/api/v1/snapshot", PublicationID: "8",
			LastEngineSequence: decimal(sequence), Lifecycle: "live", RankingMode: "degraded_bootstrap"})
	}
	latest, ok := diagnostics.Latest()
	if !ok || latest.LastEngineSequence != "100" || len(diagnostics.updates) != 1 {
		t.Fatalf("bounded diagnostic latest=%+v ok=%t pending=%d", latest, ok, len(diagnostics.updates))
	}
	if pending := <-diagnostics.Updates(); pending.LastEngineSequence != "100" || pending.Invariant != "population_prior_close_identity" {
		t.Fatalf("bounded diagnostic notification=%+v", pending)
	}
}

func TestStaleSpreadRetainsNumericValuesAcrossSnapshotBoundary(t *testing.T) {
	capture := schemaCapture()
	const cents, basisPoints = 1.5, 15.0
	const quoteAge = 45 * time.Minute
	capture.Engine.TQ.Rows[0].Spread = engine.SpreadView{
		Status: engine.TQStale, Reason: "stale_quote", Cents: cents, BasisPoints: basisPoints,
		QuoteAge: quoteAge, Quality: "reviewed_ordinary",
	}

	snapshot, err := mapCaptureView(capture)
	if err != nil {
		t.Fatalf("retained stale spread rejected: %v", err)
	}
	spread := snapshot.Rows[0].Spread
	if spread.Status != "stale" || spread.Reason != "stale_quote" || spread.Cents == nil || spread.BasisPoints == nil ||
		*spread.Cents != cents || *spread.BasisPoints != basisPoints || spread.QuoteAgeMS != uint64(quoteAge.Milliseconds()) {
		t.Fatalf("retained stale spread = %+v", spread)
	}

	snapshot.Rows[0].Spread.Cents = nil
	if err := validateSnapshot(snapshot); mappingInvariant(err) != "product_row" {
		t.Fatalf("stale spread with incomplete numeric pair = %v", err)
	}
}

func schemaCapture() operations.SnapshotCaptureView {
	at := time.Date(2026, 8, 8, 16, 0, 0, 0, time.UTC)
	target := at.Add(-4 * time.Second)
	supported := target
	population := engine.ReplayPopulationView{UniverseTotal: 2, ValidPriorClose: 2, TrustedRankableMark: 1, NoPrintThroughT: 1, CoveredPopulation: 2}
	evaluation := engine.ReplayEvaluationView{At: target, Mode: "qualified_current", Population: population,
		Qualification: engine.ReplayQualificationAccountingView{Provisional: 1}, TotalPassers: 1, KnownRankableCount: 1, TQIntentAvailable: true,
		PopulationTransition: engine.ReplayPopulationTransitionDiagnosticView{BootstrapUnknown: 1, TrustedByLaterLiveMark: 1},
		Rows: []engine.ReplayRankingRowView{{Rank: 1, Symbol: "AAA", Last: 10, DayPercent: .25, MarkAge: 250 * time.Millisecond,
			From4AMPercent: engine.ReplayFieldView{Status: "current", Value: 0}, HODDrawdown: engine.ReplayFieldView{Status: "unavailable", Reason: "before_first_print"},
			SessionRange: engine.ReplayFieldView{Status: "current", Value: .5}, Rolling30: engine.ReplayFieldView{Status: "warming", Reason: "rolling_warmup"},
			Rolling60: engine.ReplayFieldView{Status: "invalid", Reason: "historical_conflict"}, Activity: engine.ReplayFieldView{Status: "current", Value: 1.25}, TQIntentEligible: true}}}
	operational := engine.OperationalView{PublicationID: 9007199254741001, LastEngineSequence: 9007199254741003, BindingIdentity: "binding", TradingDate: "2026-08-08",
		RunMode: engine.RunModeLive, Lifecycle: "live", Watermark: &target, GeneratedAt: at, RankingMode: "qualified_current", CurrentMarketClaim: true,
		Connection: engine.OperationalConnection{Epoch: 4, AckFrame: 20, AckPosition: engine.LivePosition{ConnectionEpoch: 4, FrameSequence: 20, ArrayIndex: 7}, RecoveryAttempts: 2, Active: true, Acknowledged: true},
		Hydration: engine.OperationalHydration{Purpose: engine.HydrationFreshBootstrap, Generation: 1, Start: at.Add(-time.Minute), End: at.Add(-30 * time.Second),
			Accounting: engine.HydrationAccounting{Planned: 2, CompletedValue: 1, CompletedEmpty: 1}, Rows: engine.HydrationRowAccounting{Consumed: 2, Inserted: 1, Duplicate: 1},
			FenceReconciled: true, FenceEpoch: 4, FenceThrough: 20, FenceMarkerOrdinal: 9, SupportedThrough: &supported}, InstalledCheckpoint: true}
	tq := engine.TQView{PublicationID: operational.PublicationID, Desired: []string{"AAA"}, Rows: []engine.TQSymbolView{{Symbol: "AAA", Desired: true, ProviderPresent: true, TradeCoverage: true, QuoteCoverage: true,
		Tape: engine.TapeRateView{Status: engine.TQCurrent, Reason: "qualifying_original_prints", OneSecondStatus: engine.TQCurrent, OneSecondReason: "qualifying_original_prints",
			FiveSecondStatus: engine.TQCurrent, FiveSecondReason: "qualifying_original_prints", TimestampBasis: "none"},
		Spread: engine.SpreadView{Status: engine.TQCurrent, QuoteAge: time.Second, Quality: "reviewed_ordinary"}}}, Pressure: engine.TQPressureNormal,
		Accounting: engine.TQAccountingView{Consumed: 2, Applied: 1, Duplicate: 1, KnownPresent: 1}, Commands: engine.TQCommandAccountingView{Issued: 1, Acknowledged: 1}}
	publication := engine.ReplayPublicationView{SchemaVersion: "engine-private-publication-v1", PublicationID: operational.PublicationID, BindingIdentity: operational.BindingIdentity,
		TradingDate: operational.TradingDate, RunMode: engine.RunModeLive, Lifecycle: "live", LastEngineSequence: operational.LastEngineSequence,
		Watermark: &target, GeneratedAt: at, CurrentMarketClaim: true, AggregateEvaluation: evaluation}
	metrics := operations.Metrics{SampledAt: at, Engine: operational, LiveQueue: massive.LiveQueueAccounting{CapacityFrames: 512, CapacityBytes: 64 << 20},
		Checkpoint:         checkpoint.WriterAccounting{Submitted: 1, Completed: 1},
		CheckpointEngine:   engine.CheckpointOperations{Eligible: 1, ProjectionStarted: 1, Projected: 1, Submitted: 1, Completed: 1},
		QueueCurrentFrames: 2, QueueHighFrames: 3, QueueCurrentBytes: 100, QueueHighBytes: 200,
		Deliveries: 10, ConsumerDeferred: 1, MeanProcessingDelay: 2 * time.Millisecond, MaxProcessingDelay: 3 * time.Millisecond,
		MaxProcessingDelayOneSecond: time.Millisecond,
		DeliveryLatencyAttribution: operations.DeliveryLatencyAttribution{Aggregate: 3, TQ: 2, Control: 1, HydrationFence: 1, Checkpoint: 1, Timer: 1, Unknown: 1,
			MaximumDuration: time.Millisecond, MaximumFamily: operations.DeliveryLatencyAggregate, WindowNonempty: true},
		HeapAllocBytes: 1 << 20, HeapInUseBytes: 2 << 20, Goroutines: 8, AccountingValid: true}
	status := operations.Status{ProcessLive: true, BackendReady: true, RankingCurrent: true, Lifecycle: "live", RankingMode: "qualified_current", SampledAt: at,
		PublicationID: operational.PublicationID, Watermark: &target, CausalTarget: &target, AccountingValid: true}
	return operations.SnapshotCaptureView{SampleID: 9007199254740999, SampledAt: at, ProcessLive: true,
		Engine: engine.SnapshotView{Publication: publication, Operational: operational, TQ: tq}, Status: status, Metrics: metrics}
}

func snapshotBinding(t *testing.T, requestedSymbols ...string) reference.Binding {
	t.Helper()
	if len(requestedSymbols) == 0 {
		requestedSymbols = []string{"AAA"}
	}
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			results := make([]map[string]any, len(requestedSymbols))
			for index, symbol := range requestedSymbols {
				results[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "count": len(results), "results": results})
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			results := make([]map[string]any, len(requestedSymbols))
			for index, symbol := range requestedSymbols {
				results[index] = map[string]any{"T": symbol, "c": 10.0, "t": facts.PriorRegularClose.UnixMilli()}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(results), "results": results})
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	resolverNow := facts.SessionStart.Add(time.Hour)
	dataDir := filepath.Join(t.TempDir(), "reference")
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: dataDir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return resolverNow }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: dataDir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return resolverNow }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}
