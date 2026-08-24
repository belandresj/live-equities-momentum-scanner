package snapshotapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	const goldenSHA256 = "5a819616fd9a5018dcad2ef057e70f0105b3ccd50a4363923dcd1bb54e301c49"
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
	if snapshot.TQ.PressureSample != (TQPressureSample{Observed: true, WaitingFrames: 2, FrameCapacity: 512, WaitingBytes: 100, ByteCapacity: 64 << 20, OldestWaitingFrameAgeMS: 500, RecoveryHealthy: true}) ||
		snapshot.TQ.PressureRecovery != (TQPressureRecovery{RequiredSamples: 5}) {
		t.Fatalf("T/Q recovery diagnostics = sample=%+v recovery=%+v", snapshot.TQ.PressureSample, snapshot.TQ.PressureRecovery)
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
	if row.FromOpenChange.ValueRatio == nil || *row.FromOpenChange.ValueRatio != 0 ||
		row.Volume.ValueShares == nil || *row.Volume.ValueShares != 0 || row.Move30s.ValueRatio == nil || *row.Move30s.ValueRatio != 0 ||
		row.Tape5s.TradesPerSecond == nil || *row.Tape5s.TradesPerSecond != 0 || row.Spread.Cents == nil || *row.Spread.Cents != 0 {
		t.Fatalf("null versus genuine zero = %+v", row)
	}
	if row.DayChangeRatio != .0025 || row.DayRangePosition.ValueRatio == nil || *row.DayRangePosition.ValueRatio != .005 ||
		row.Activity30s.ValueRatio == nil || *row.Activity30s.ValueRatio != .0125 || row.Float.ValueShares == nil || *row.Float.ValueShares != 12_000_000 {
		t.Fatalf("percentage-point to ratio mapping = %+v", row)
	}
	var rowFields map[string]json.RawMessage
	if err := json.Unmarshal(mustJSON(t, row), &rowFields); err != nil {
		t.Fatal(err)
	}
	for _, superseded := range []string{"from_4am_change", "hod_drawdown", "range_30m_position", "range_60m_position", "activity", "tape_rate"} {
		if _, exists := rowFields[superseded]; exists {
			t.Fatalf("superseded v1 row field %q serialized", superseded)
		}
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
	badPressureCapacity := snapshot
	badPressureCapacity.TQ.PressureSample.FrameCapacity = 0
	if err := validateSnapshot(badPressureCapacity); mappingInvariant(err) != "tq_pressure_recovery" {
		t.Fatalf("observed pressure sample accepted zero capacity: %v", err)
	}
	badPressureProgress := snapshot
	badPressureProgress.TQ.PressureRecovery.HealthySamples = 6
	if err := validateSnapshot(badPressureProgress); mappingInvariant(err) != "tq_pressure_recovery" {
		t.Fatalf("pressure recovery accepted excess progress: %v", err)
	}
	missResetPressure := snapshot
	missResetPressure.Rows = append([]Row(nil), snapshot.Rows...)
	missResetPressure.Status.TQPressureMode, missResetPressure.Status.TQShed = "taq_degraded", true
	missResetPressure.TQ.PressureMode, missResetPressure.TQ.PressureCause, missResetPressure.TQ.Shed = "taq_degraded", "oldest_waiting_frame", true
	missResetPressure.Rows[0].Tape5s = Tape5s{Status: "pressure_shed", Reason: "pressure"}
	missResetPressure.Rows[0].Spread = Spread{Status: "pressure_shed", Reason: "pressure"}
	missResetPressure.Rows[0].TQMembership = TQMembership{Desired: true, ProviderMembershipUnknown: true}
	if err := validateSnapshot(missResetPressure); err != nil {
		t.Fatalf("missing sample could not reset progress after a healthy accepted sample: %v", err)
	}
	badCompletedProgress := missResetPressure
	badCompletedProgress.TQ.PressureRecovery.HealthySamples = 5
	if err := validateSnapshot(badCompletedProgress); mappingInvariant(err) != "tq_pressure_recovery" {
		t.Fatalf("nonnormal pressure accepted completed recovery progress: %v", err)
	}
	badPositiveProgress := missResetPressure
	badPositiveProgress.TQ.PressureSample.RecoveryHealthy = false
	badPositiveProgress.TQ.PressureRecovery.HealthySamples = 1
	if err := validateSnapshot(badPositiveProgress); mappingInvariant(err) != "tq_pressure_recovery" {
		t.Fatalf("positive recovery progress accepted unhealthy last sample: %v", err)
	}
	badUnobservedPressure := snapshot
	badUnobservedPressure.TQ.PressureSample.Observed = false
	if err := validateSnapshot(badUnobservedPressure); mappingInvariant(err) != "tq_pressure_recovery" {
		t.Fatalf("unobserved pressure sample retained values: %v", err)
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
	row.FromOpenPercent = engine.ReplayFieldView{Status: "current", Value: -100}
	row.DayRange = engine.ReplayFieldView{Status: "current", Value: 100}
	row.Activity30s = engine.ReplayFieldView{Status: "current", Value: 100}
	row.Move30s = engine.ReplayFieldView{Status: "current", Value: -25}

	snapshot, err := mapCaptureView(capture)
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot.Rows[0]
	if got.DayChangeRatio != 2.5 || *got.FromOpenChange.ValueRatio != -1 || *got.DayRangePosition.ValueRatio != 1 ||
		*got.Activity30s.ValueRatio != 1 || *got.Move30s.ValueRatio != -.25 {
		t.Fatalf("boundary ratios = %+v", got)
	}
}

func TestPMVPAPISchemaAvailabilityProvenanceOrderingAndBounds(t *testing.T) {
	t.Run("independent states and units", func(t *testing.T) {
		capture := schemaCapture()
		row := &capture.Engine.Publication.AggregateEvaluation.Rows[0]
		percent := 25.0
		row.Float = engine.ReplayFloatFieldView{Status: "stale", Reason: "cached_fallback", Value: 8_500_000, Percent: &percent,
			Provider: "massive-stocks-float-experimental", EffectiveDate: "2026-08-01", RetrievedAt: capture.SampledAt.Add(-time.Hour), Provenance: "cache"}
		row.SessionVolume = engine.ReplayFieldView{Status: "unavailable", Reason: "history_incomplete"}
		row.FromOpenPercent = engine.ReplayFieldView{Status: "unavailable", Reason: "before_first_print"}
		row.DayRange = engine.ReplayFieldView{Status: "invalid", Reason: "historical_conflict"}
		row.Activity30s = engine.ReplayFieldView{Status: "current", Value: 0}
		row.Move30s = engine.ReplayFieldView{Status: "unavailable", Reason: "no_aggregate_in_target"}
		capture.Engine.TQ.Rows[0].Tape.Status, capture.Engine.TQ.Rows[0].Tape.Reason = engine.TQWarming, "five_second_warming"
		capture.Engine.TQ.Rows[0].Spread.Status = engine.TQStale
		capture.Engine.TQ.Rows[0].Spread.Reason = "stale_quote"
		capture.Engine.TQ.Rows[0].Spread.Cents = 1.5
		capture.Engine.TQ.Rows[0].Spread.BasisPoints = 15
		capture.Engine.TQ.Rows[0].Spread.QuoteAge = 45 * time.Minute

		snapshot, err := mapCaptureView(capture)
		if err != nil {
			t.Fatal(err)
		}
		got := snapshot.Rows[0]
		if got.Float.Status != "stale" || got.Float.ValueShares == nil || *got.Float.ValueShares != 8_500_000 ||
			got.Float.PercentRatio == nil || *got.Float.PercentRatio != .25 || got.Float.EffectiveDate == nil || *got.Float.EffectiveDate != "2026-08-01" ||
			got.Float.RetrievedAt == nil || got.Float.Provenance != "cache" || got.Volume.ValueShares != nil ||
			got.FromOpenChange.Status != "unavailable" || got.DayRangePosition.Status != "invalid" || got.Activity30s.ValueRatio == nil || *got.Activity30s.ValueRatio != 0 ||
			got.Move30s.ValueRatio != nil || got.Tape5s.TradesPerSecond != nil || got.Spread.Cents == nil || got.Spread.QuoteAgeMS != uint64((45*time.Minute)/time.Millisecond) {
			t.Fatalf("independent v2 mapping = %+v", got)
		}
	})

	t.Run("order and top twenty", func(t *testing.T) {
		capture := schemaCapture()
		base := capture.Engine.Publication.AggregateEvaluation.Rows[0]
		rows := make([]engine.ReplayRankingRowView, 20)
		for index := range rows {
			rows[index] = base
			rows[index].Rank = uint32(index + 1)
			rows[index].Symbol = fmt.Sprintf("S%02d", index+1)
			rows[index].DayPercent = float64(20 - index)
			rows[index].TQIntentEligible = index == 0
		}
		capture.Engine.Publication.AggregateEvaluation.Rows = rows
		capture.Engine.Publication.AggregateEvaluation.Population = engine.ReplayPopulationView{UniverseTotal: 20, ValidPriorClose: 20, TrustedRankableMark: 20, CoveredPopulation: 20}
		capture.Engine.Publication.AggregateEvaluation.Qualification = engine.ReplayQualificationAccountingView{Provisional: 20}
		capture.Engine.Publication.AggregateEvaluation.PopulationTransition = engine.ReplayPopulationTransitionDiagnosticView{}
		capture.Engine.Publication.AggregateEvaluation.TotalPassers = 20
		capture.Engine.Publication.AggregateEvaluation.KnownRankableCount = 20
		snapshot, err := mapCaptureView(capture)
		if err != nil || len(snapshot.Rows) != 20 {
			t.Fatalf("top twenty mapping rows=%d err=%v", len(snapshot.Rows), err)
		}
		for index, row := range snapshot.Rows {
			if row.Rank != uint64(index+1) || row.Symbol != fmt.Sprintf("S%02d", index+1) || row.DayChangeRatio != float64(20-index)/100 {
				t.Fatalf("row %d reordered: %+v", index, row)
			}
		}
		capture.Engine.Publication.AggregateEvaluation.Rows = append(capture.Engine.Publication.AggregateEvaluation.Rows, base)
		if _, err := mapCaptureView(capture); mappingInvariant(err) != "capture_coherence" {
			t.Fatalf("twenty-one rows reached schema: %v", err)
		}
	})

	t.Run("malformed source and wire values", func(t *testing.T) {
		mutations := []struct {
			name string
			edit func(*operations.SnapshotCaptureView)
		}{
			{"last below rankability bound", func(v *operations.SnapshotCaptureView) { v.Engine.Publication.AggregateEvaluation.Rows[0].Last = .249 }},
			{"negative volume", func(v *operations.SnapshotCaptureView) {
				v.Engine.Publication.AggregateEvaluation.Rows[0].SessionVolume.Value = -1
			}},
			{"nonfinite Float", func(v *operations.SnapshotCaptureView) {
				v.Engine.Publication.AggregateEvaluation.Rows[0].Float.Value = math.NaN()
			}},
			{"bad Float effective date", func(v *operations.SnapshotCaptureView) {
				v.Engine.Publication.AggregateEvaluation.Rows[0].Float.EffectiveDate = "2026-02-30"
			}},
			{"Activity above bound", func(v *operations.SnapshotCaptureView) {
				v.Engine.Publication.AggregateEvaluation.Rows[0].Activity30s.Value = 100.01
			}},
			{"Day Range below bound", func(v *operations.SnapshotCaptureView) {
				v.Engine.Publication.AggregateEvaluation.Rows[0].DayRange.Value = -.01
			}},
			{"negative Tape", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Rows[0].Tape.FiveSecond = -1 }},
			{"negative Spread", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Rows[0].Spread.Cents = -1 }},
			{"negative quote age", func(v *operations.SnapshotCaptureView) { v.Engine.TQ.Rows[0].Spread.QuoteAge = -time.Millisecond }},
		}
		for _, mutation := range mutations {
			t.Run(mutation.name, func(t *testing.T) {
				capture := schemaCapture()
				mutation.edit(&capture)
				if _, err := mapCaptureView(capture); err == nil {
					t.Fatal("malformed internal value reached v2")
				}
			})
		}
		snapshot, err := mapCaptureView(schemaCapture())
		if err != nil {
			t.Fatal(err)
		}
		zero := 0.0
		snapshot.Rows[0].Volume = ShareMeasurement{Status: "unavailable", Reason: "history_incomplete", ValueShares: &zero}
		if err := validateSnapshot(snapshot); err == nil {
			t.Fatal("unavailable wire value encoded as numeric zero")
		}
	})
}

func TestPMVPAPIRejectsContradictoryMeasurementTuples(t *testing.T) {
	base, err := mapCaptureView(schemaCapture())
	if err != nil {
		t.Fatal(err)
	}
	zero := 0.0
	mutations := []struct {
		name string
		edit func(*Row)
	}{
		{"current ratio with absence reason", func(r *Row) { r.Move30s.Reason = "history_incomplete" }},
		{"warming ratio without reason", func(r *Row) { r.Activity30s = RatioMeasurement{Status: "warming"} }},
		{"unavailable ratio without reason", func(r *Row) { r.FromOpenChange = RatioMeasurement{Status: "unavailable"} }},
		{"invalid ratio without reason", func(r *Row) { r.DayRangePosition = RatioMeasurement{Status: "invalid"} }},
		{"warming Volume", func(r *Row) { r.Volume = ShareMeasurement{Status: "warming", Reason: "history_incomplete"} }},
		{"unavailable Volume without reason", func(r *Row) { r.Volume = ShareMeasurement{Status: "unavailable"} }},
		{"invalid Volume without reason", func(r *Row) { r.Volume = ShareMeasurement{Status: "invalid"} }},
		{"unavailable Volume with zero", func(r *Row) {
			r.Volume = ShareMeasurement{Status: "unavailable", Reason: "history_incomplete", ValueShares: &zero}
		}},
		{"current Float with cache provenance", func(r *Row) { r.Float.Provenance, r.Float.Reason = "cache", "cached_fallback" }},
		{"stale Float with fresh provenance", func(r *Row) {
			r.Float.Status, r.Float.Provenance = "stale", "fresh"
			r.Float.Reason = "cached_fallback"
		}},
		{"unavailable Float without reason", func(r *Row) { r.Float = FloatMeasurement{Status: "unavailable"} }},
		{"invalid Float without reason", func(r *Row) { r.Float = FloatMeasurement{Status: "invalid"} }},
		{"warming Tape without reason", func(r *Row) { r.Tape5s = Tape5s{Status: "warming", TradeCoverage: true} }},
		{"current Tape with warmup reason", func(r *Row) { r.Tape5s.Reason = "five_second_warming" }},
		{"unavailable Tape without reason", func(r *Row) { r.Tape5s = Tape5s{Status: "unavailable"} }},
		{"invalid Tape without reason", func(r *Row) { r.Tape5s = Tape5s{Status: "invalid", TradeCoverage: true} }},
		{"current Spread with stale reason", func(r *Row) { r.Spread.Reason = "stale_quote" }},
		{"stale Spread without retained values", func(r *Row) { r.Spread = Spread{Status: "stale", Reason: "stale_quote", QuoteCoverage: true} }},
		{"unavailable Spread without reason", func(r *Row) { r.Spread = Spread{Status: "unavailable"} }},
		{"invalid Spread without reason", func(r *Row) { r.Spread = Spread{Status: "invalid", QuoteCoverage: true} }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			value := base
			value.Rows = append([]Row(nil), base.Rows...)
			mutation.edit(&value.Rows[0])
			if err := validateSnapshot(value); err == nil {
				t.Fatal("contradictory tuple serialized")
			}
		})
	}

	// Genuine numeric zero is current data for each numeric family.
	valid := base
	valid.Rows = append([]Row(nil), base.Rows...)
	valid.Rows[0].Volume = ShareMeasurement{Status: "current", ValueShares: &zero}
	valid.Rows[0].Move30s = RatioMeasurement{Status: "current", ValueRatio: &zero}
	valid.Rows[0].Tape5s = Tape5s{Status: "current", Reason: "qualifying_original_prints", TradeCoverage: true, TradesPerSecond: &zero, TimestampBasis: "none"}
	valid.Rows[0].Spread = Spread{Status: "current", QuoteCoverage: true, Cents: &zero, BasisPoints: &zero, Quality: "reviewed_ordinary"}
	if err := validateSnapshot(valid); err != nil {
		t.Fatalf("genuine zero rejected: %v", err)
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
	secondCapture.Engine.TQ.Rows[0].Tape.FiveSecond = 99
	if first.TQ.DesiredSymbols[0] != "AAA" || first.Rows[0].Tape5s.TradesPerSecond == nil || *first.Rows[0].Tape5s.TradesPerSecond != 0 {
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
			v.Engine.Publication.AggregateEvaluation.Qualification = engine.ReplayQualificationAccountingView{NotYetPassed: 1}
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
			v.Engine.TQ.PressureSample.WaitingFrames = 52
			v.Engine.TQ.PressureSample.RecoveryHealthy = false
			v.Engine.TQ.Rows[0].Tape = engine.TapeRateView{Status: engine.TQPressureShed, Reason: "pressure"}
			v.Engine.TQ.Rows[0].Spread = engine.SpreadView{Status: engine.TQPressureShed, Reason: "pressure"}
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
			v.Engine.Publication.AggregateEvaluation.Rows[0].FromOpenPercent.Value = math.NaN()
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
		diagnostics.record(MappingFailure{Invariant: mappingInvariant(err), Route: "/api/v2/snapshot", PublicationID: "8",
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
			Float:           engine.ReplayFloatFieldView{Status: "current", Value: 12_000_000, Provider: "massive-stocks-float-experimental", EffectiveDate: "2026-08-07", RetrievedAt: at, Provenance: "fresh"},
			SessionVolume:   engine.ReplayFieldView{Status: "current", Value: 0},
			FromOpenPercent: engine.ReplayFieldView{Status: "current", Value: 0}, DayRange: engine.ReplayFieldView{Status: "current", Value: .5},
			Activity30s: engine.ReplayFieldView{Status: "current", Value: 1.25}, Move30s: engine.ReplayFieldView{Status: "current", Value: 0}, TQIntentEligible: true}}}
	operational := engine.OperationalView{PublicationID: 9007199254741001, LastEngineSequence: 9007199254741003, BindingIdentity: "binding", TradingDate: "2026-08-08",
		RunMode: engine.RunModeLive, Lifecycle: "live", Watermark: &target, GeneratedAt: at, RankingMode: "qualified_current", CurrentMarketClaim: true,
		Connection: engine.OperationalConnection{Epoch: 4, AckFrame: 20, AckPosition: engine.LivePosition{ConnectionEpoch: 4, FrameSequence: 20, ArrayIndex: 7}, RecoveryAttempts: 2, Active: true, Acknowledged: true},
		Hydration: engine.OperationalHydration{Purpose: engine.HydrationFreshBootstrap, Generation: 1, Start: at.Add(-time.Minute), End: at.Add(-30 * time.Second),
			Accounting: engine.HydrationAccounting{Planned: 2, CompletedValue: 1, CompletedEmpty: 1}, Rows: engine.HydrationRowAccounting{Consumed: 2, Inserted: 1, Duplicate: 1},
			FenceReconciled: true, FenceEpoch: 4, FenceThrough: 20, FenceMarkerOrdinal: 9, SupportedThrough: &supported}, InstalledCheckpoint: true}
	tq := engine.TQView{PublicationID: operational.PublicationID, Desired: []string{"AAA"}, Rows: []engine.TQSymbolView{{Symbol: "AAA", Desired: true, ProviderPresent: true, TradeCoverage: true, QuoteCoverage: true,
		Tape:   engine.TapeRateView{Status: engine.TQCurrent, Reason: "qualifying_original_prints", TimestampBasis: "none"},
		Spread: engine.SpreadView{Status: engine.TQCurrent, QuoteAge: time.Second, Quality: "reviewed_ordinary"}}}, Pressure: engine.TQPressureNormal,
		PressureSample: engine.TQPressureSampleView{Observed: true, WaitingFrames: 2, FrameCapacity: 512, WaitingBytes: 100, ByteCapacity: 64 << 20,
			OldestWaitingFrameAge: 500 * time.Millisecond, RecoveryHealthy: true}, RecoveryRequiredSamples: 5,
		Accounting: engine.TQAccountingView{Consumed: 2, Applied: 1, Duplicate: 1, KnownPresent: 1}, Commands: engine.TQCommandAccountingView{Issued: 1, Written: 1}}
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

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
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
