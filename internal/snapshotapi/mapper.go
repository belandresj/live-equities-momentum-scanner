package snapshotapi

import (
	"math"
	"strconv"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

const maximumExactJSONInteger = uint64(1<<53 - 1)

func Map(capture operations.SnapshotCapture) (Snapshot, error) {
	view, ok := operations.InspectSnapshotCapture(capture)
	if !ok {
		return Snapshot{}, rejectMapping("capture_seal")
	}
	return mapCaptureView(view)
}

func mapCaptureView(capture operations.SnapshotCaptureView) (Snapshot, error) {
	publication := capture.Engine.Publication
	operational := capture.Engine.Operational
	if capture.SampleID == 0 || !validUTC(capture.SampledAt) || publication.PublicationID == 0 || publication.BindingIdentity == "" ||
		!validUTC(publication.GeneratedAt) || capture.Status.CausalTarget == nil || !validUTC(*capture.Status.CausalTarget) ||
		capture.Status.ProcessLive != capture.ProcessLive || !capture.Status.SampledAt.Equal(capture.SampledAt) || capture.Status.PublicationID != publication.PublicationID ||
		capture.Engine.TQ.PublicationID != publication.PublicationID ||
		!capture.Metrics.SampledAt.Equal(capture.SampledAt) ||
		publication.PublicationID != operational.PublicationID || publication.BindingIdentity != operational.BindingIdentity ||
		publication.TradingDate != operational.TradingDate || publication.RunMode != operational.RunMode || publication.Lifecycle != operational.Lifecycle ||
		publication.LastEngineSequence != operational.LastEngineSequence || !sameOptionalTime(publication.Watermark, operational.Watermark) ||
		len(publication.AggregateEvaluation.Rows) > 20 ||
		capture.Metrics.LiveQueue.CapacityFrames < 0 || capture.Metrics.QueueCurrentBytes < 0 || capture.Metrics.QueueHighBytes < 0 || capture.Metrics.Goroutines < 0 {
		return Snapshot{}, rejectMapping("capture_coherence")
	}
	if capture.Metrics.DeliveryLatencyAttribution.MaximumDuration != capture.Metrics.MaxProcessingDelayOneSecond ||
		!capture.Metrics.DeliveryLatencyAttribution.Reconciles(capture.Metrics.Deliveries) {
		return Snapshot{}, rejectMapping("delivery_latency_attribution")
	}

	result := Snapshot{
		SchemaVersion: SchemaVersion,
		Sample:        Sample{ID: decimal(capture.SampleID), SampledAt: timestamp(capture.SampledAt)},
		Publication:   mapPublication(publication, operational),
		Status: Status{ProcessLive: capture.ProcessLive, BackendReady: capture.Status.BackendReady, ReadinessReason: string(capture.Status.Reason),
			RankingCurrent: capture.Status.RankingCurrent, CausalTarget: timestamp(*capture.Status.CausalTarget), AccountingValid: capture.Status.AccountingValid,
			TQPressureMode: string(capture.Engine.TQ.Pressure), TQShed: capture.Engine.TQ.ShedTradesQuotes},
		Ranking:    mapRanking(publication.AggregateEvaluation),
		Rows:       make([]Row, len(publication.AggregateEvaluation.Rows)),
		Accounting: mapAccounting(publication.AggregateEvaluation),
		Recovery:   mapRecovery(operational.Hydration),
		TQ:         mapTQ(capture.Engine.TQ, capture.Metrics.TQNormalization),
		Checkpoint: fixedDisabledCheckpoint(),
		Operations: mapOperations(capture.Metrics, operational),
	}
	if publication.Watermark != nil {
		lag := capture.Status.WatermarkLag.Milliseconds()
		if lag < 0 {
			return Snapshot{}, rejectMapping("watermark_negative_lag")
		}
		result.Status.WatermarkLagMS = &lag
	}

	tqRows := make(map[string]engine.TQSymbolView, len(capture.Engine.TQ.Rows))
	for _, row := range capture.Engine.TQ.Rows {
		if row.Symbol == "" {
			return Snapshot{}, rejectMapping("tq_source_row")
		}
		tqRows[row.Symbol] = row
	}
	for index, row := range publication.AggregateEvaluation.Rows {
		mapped, err := mapRow(row, tqRows[row.Symbol])
		if err != nil {
			return Snapshot{}, err
		}
		applyTQWatermarkAvailability(&mapped, tqRows[row.Symbol], capture.Status)
		result.Rows[index] = mapped
	}
	if err := validateSnapshot(result); err != nil {
		return Snapshot{}, err
	}
	return result, nil
}

func applyTQWatermarkAvailability(row *Row, source engine.TQSymbolView, status operations.Status) {
	if row == nil || !status.TQWatermarkVisibilityHold && status.TQWatermarkRecoveryBoundary == nil {
		return
	}
	if status.TQWatermarkVisibilityHold {
		reason := "watermark_recovery_warming"
		if status.Reason == operations.ReasonWatermarkStale {
			reason = "aggregate_watermark_stale"
		}
		if tqStatusCanWarm(row.Tape5s.Status) {
			row.Tape5s.Status, row.Tape5s.Reason, row.Tape5s.TradesPerSecond = "warming", reason, nil
		}
		if tqStatusCanWarm(row.Spread.Status) {
			row.Spread.Status, row.Spread.Reason = "warming", reason
			row.Spread.Cents, row.Spread.BasisPoints = nil, nil
		}
		return
	}
	boundary := status.TQWatermarkRecoveryBoundary
	if boundary != nil && (source.Spread.ObservedAt.IsZero() || source.Spread.ObservedAt.Before(*boundary)) && tqStatusCanWarm(row.Spread.Status) {
		row.Spread.Status, row.Spread.Reason = "warming", "watermark_recovery_quote_warming"
		row.Spread.Cents, row.Spread.BasisPoints = nil, nil
	}
}

func tqStatusCanWarm(status string) bool {
	return status == "current" || status == "stale" || status == "warming"
}

func mapPublication(p engine.PublicationView, operational engine.OperationalView) Publication {
	result := Publication{
		ID: decimal(p.PublicationID), BindingIdentity: p.BindingIdentity, TradingDate: p.TradingDate,
		RunMode: string(p.RunMode), Lifecycle: p.Lifecycle, LifecycleReason: p.LifecycleReason, Suppression: string(p.Suppression),
		GeneratedAt: timestamp(p.GeneratedAt), CommittedT: optionalTime(p.Watermark), LastEngineSequence: decimal(p.LastEngineSequence),
		ConnectionEpoch: decimal(operational.Connection.Epoch), ConnectionActive: operational.Connection.Active,
		AggregateAcknowledged: operational.Connection.Acknowledged,
		AggregateAckPosition:  Position{ConnectionEpoch: "0", FrameSequence: "0", ArrayIndex: 0},
		HydrationFence:        HydrationFence{ConnectionEpoch: "0", ThroughFrameSequence: "0", MarkerOrdinal: "0"},
	}
	if operational.Connection.Acknowledged {
		position := operational.Connection.AckPosition
		result.AggregateAckPosition = Position{ConnectionEpoch: decimal(position.ConnectionEpoch), FrameSequence: decimal(position.FrameSequence), ArrayIndex: uint64(position.ArrayIndex)}
	}
	if operational.Hydration.FenceReconciled {
		result.HydrationFence = HydrationFence{Reconciled: true, ConnectionEpoch: decimal(operational.Hydration.FenceEpoch),
			ThroughFrameSequence: decimal(operational.Hydration.FenceThrough), MarkerOrdinal: decimal(operational.Hydration.FenceMarkerOrdinal),
			SupportedThrough: optionalTime(operational.Hydration.SupportedThrough)}
	}
	return result
}

func mapRanking(value engine.EvaluationView) Ranking {
	return Ranking{Mode: value.Mode, Reason: value.Reason, TotalPassers: value.TotalPassers, KnownRankableCount: value.KnownRankableCount,
		DayInvalidRankable: value.DayInvalidRankable, QualifiedDayInvalid: value.QualifiedDayInvalid}
}

func mapAccounting(value engine.EvaluationView) Accounting {
	p, q, u, d := value.Population, value.Qualification, value.Uncertainty, value.PopulationTransition
	return Accounting{
		Population: PopulationAccounting{UniverseTotal: p.UniverseTotal, ValidPriorClose: p.ValidPriorClose, InvalidOrMissingPriorClose: p.InvalidOrMissingPriorClose,
			TrustedRankableMark: p.TrustedRankableMark, TrustedBelowPriceMark: p.TrustedBelowPriceMark, NoPrintThroughT: p.NoPrintThroughT,
			InvalidMark: p.InvalidMark, UnknownDueFailureOrFence: p.UnknownDueFailureOrFence, CoveredPopulation: p.CoveredPopulation, UnresolvedPopulation: p.UnresolvedPopulation},
		Qualification: QualificationAccounting{NotYetPassed: q.NotYetPassed, Provisional: q.Provisional, Finalized: q.Finalized, Unresolved: q.Unresolved},
		Uncertainty:   UncertaintyAccounting{BootstrapOrigin: u.BootstrapOrigin, PostBootstrapGap: u.PostBootstrapGap, LocalInvalid: u.LocalInvalid},
		PopulationTransitionDiagnostic: PopulationTransitionDiagnostic{BootstrapUnknown: d.BootstrapUnknown, TrustedByLaterLiveMark: d.TrustedByLaterLiveMark,
			NoLaterEligibleMark: d.NoLaterEligibleMark, LatestMarkNotLiveAuthority: d.LatestMarkNotLiveAuthority,
			NoStrictlyOlderLocalizedConflict: d.NoStrictlyOlderLocalizedConflict, ConflictAtOrAfterMark: d.ConflictAtOrAfterMark,
			InvalidAtOrAfterMark: d.InvalidAtOrAfterMark, IncompletePostMarkCoverage: d.IncompletePostMarkCoverage},
	}
}

func mapRecovery(value engine.OperationalHydration) Recovery {
	a, rows := value.Accounting, value.Rows
	return Recovery{GenerationActive: value.Active, Purpose: string(value.Purpose), Generation: decimal(value.Generation), Start: optionalNonzeroTime(value.Start), End: optionalNonzeroTime(value.End),
		SupportedThrough: optionalTime(value.SupportedThrough), FenceReconciled: value.FenceReconciled, PolicyWaiting: value.PolicyWaiting,
		Work: RecoveryWork{Planned: decimal(a.Planned), Open: decimal(a.Open), CompletedValue: decimal(a.CompletedValue), CompletedEmpty: decimal(a.CompletedEmpty),
			Failed: decimal(a.Failed), Canceled: decimal(a.Canceled), Fenced: decimal(a.Fenced)},
		Rows: RecoveryRows{Consumed: decimal(rows.Consumed), Inserted: decimal(rows.Inserted), Duplicate: decimal(rows.Duplicate), ConflictOrWithdrawal: decimal(rows.ConflictOrWithdrawal),
			Rejected: decimal(rows.Rejected), Fenced: decimal(rows.Fenced), Integrity: decimal(rows.Integrity)}}
}

func mapTQ(value engine.TQView, normalization massive.TQNormalizationAccounting) TQ {
	a, c := value.Accounting, value.Commands
	return TQ{DesiredSymbols: append([]string{}, value.Desired...), PressureMode: string(value.Pressure), PressureCause: string(value.PressureCause), AggregateOnly: value.AggregateOnly,
		Shed: value.ShedTradesQuotes, RetainedBoundHit: value.Bounds, PressureMisses: uint64(value.PressureMisses),
		PressureTransitions: decimal(value.PressureTransitions), PressureFenced: decimal(value.PressureFenced),
		PressureSample: TQPressureSample{Observed: value.PressureSample.Observed, WaitingFrames: value.PressureSample.WaitingFrames, FrameCapacity: value.PressureSample.FrameCapacity,
			WaitingBytes: value.PressureSample.WaitingBytes, ByteCapacity: value.PressureSample.ByteCapacity, OldestWaitingFrameAgeMS: durationMilliseconds(value.PressureSample.OldestWaitingFrameAge),
			AggregateWatermarkLagMS: durationMilliseconds(value.PressureSample.AggregateWatermarkLag), RecoveryHealthy: value.PressureSample.RecoveryHealthy},
		PressureRecovery: TQPressureRecovery{HealthySamples: uint64(value.RecoveryHealthySamples), RequiredSamples: uint64(value.RecoveryRequiredSamples)},
		KnownPresent:     uint64(a.KnownPresent), KnownAbsent: uint64(a.KnownAbsent), Unknown: uint64(a.Unknown), RetainedTrades: uint64(a.RetainedTrades),
		RetainedQuotes: uint64(a.RetainedQuotes), RetainedFingerprints: uint64(a.RetainedFingerprints),
		Facts: TQFacts{Consumed: decimal(a.Consumed), Applied: decimal(a.Applied), Duplicate: decimal(a.Duplicate), Rejected: decimal(a.Rejected),
			Fenced: decimal(a.Fenced), PressureShed: decimal(a.PressureShed), Integrity: decimal(a.Integrity), NormalizedTrades: decimal(normalization.NormalizedTrades), NormalizedQuotes: decimal(normalization.NormalizedQuotes), AppliedTrades: decimal(a.AppliedTrades),
			AppliedQuotes: decimal(a.AppliedQuotes), PressureShedTrades: decimal(a.PressureShedTrades), PressureShedQuotes: decimal(a.PressureShedQuotes)},
		Commands: TQCommands{Issued: decimal(c.Issued), Pending: decimal(c.Pending), Written: decimal(c.Written), Failed: decimal(c.Failed),
			Fenced: decimal(c.Fenced), ResultFenced: decimal(c.ResultFenced)}}
}

func fixedDisabledCheckpoint() Checkpoint {
	return Checkpoint{Eligible: "0", PressureDeferred: "0", ProjectionStarted: "0", ProjectionInProgress: "0", Projected: "0", ProjectionRejected: "0",
		SubmitRejected: "0", Submitted: "0", Outstanding: "0", InProgress: "0", Pending: "0", Completed: "0", Failed: "0", Canceled: "0", Superseded: "0", ArtifactBytes: "0"}
}

func mapOperations(metrics operations.Metrics, operational engine.OperationalView) Operations {
	attribution := metrics.DeliveryLatencyAttribution
	if attribution.MaximumFamily == "" {
		attribution.MaximumFamily = operations.DeliveryLatencyUnknown
	}
	result := Operations{SampleAccountingValid: metrics.AccountingValid, QueueCapacityFrames: nonnegativeInt(metrics.LiveQueue.CapacityFrames),
		QueueCurrentFrames: metrics.QueueCurrentFrames, QueueHighFrames: metrics.QueueHighFrames,
		QueueCurrentBytes: nonnegativeInt(metrics.QueueCurrentBytes), QueueHighBytes: nonnegativeInt(metrics.QueueHighBytes),
		Deliveries: decimal(metrics.Deliveries), ConsumerDeferred: decimal(metrics.ConsumerDeferred),
		MeanProcessingDelayMS: durationMilliseconds(metrics.MeanProcessingDelay), MaxProcessingDelayMS: durationMilliseconds(metrics.MaxProcessingDelay),
		MaxProcessingDelayOneSecondMS: durationMilliseconds(metrics.MaxProcessingDelayOneSecond),
		DeliveryLatencyAttribution: DeliveryLatencyAttribution{
			Aggregate: decimal(attribution.Aggregate), TQ: decimal(attribution.TQ), Control: decimal(attribution.Control),
			HydrationFence: decimal(attribution.HydrationFence), Checkpoint: "0", Timer: decimal(attribution.Timer), Unknown: decimal(attribution.Unknown),
			MaximumMS: durationMilliseconds(attribution.MaximumDuration), MaximumFamily: string(attribution.MaximumFamily),
		},
		HeapAllocBytes: decimal(metrics.HeapAllocBytes),
		HeapInUseBytes: decimal(metrics.HeapInUseBytes), Goroutines: nonnegativeInt(metrics.Goroutines),
		ConnectionRecoveryAttempts: decimal(operational.Connection.RecoveryAttempts)}
	if failure := operational.IntegrityFailure; failure != nil {
		result.IntegrityFailure = &IntegrityFailure{Category: string(failure.Category), EngineSequence: decimal(failure.EngineSequence),
			CandidateTime: optionalNonzeroTime(failure.CandidateTime), ExpectedTime: optionalNonzeroTime(failure.ExpectedTime),
			FirstSymbol: failure.FirstSymbol, FirstField: failure.FirstField, FirstReason: failure.FirstReason}
	}
	return result
}

func mapRow(row engine.RankingRowView, tq engine.TQSymbolView) (Row, error) {
	if row.Rank == 0 || row.Rank > 20 || row.Symbol == "" || !finite(row.Last) || row.Last < .25 || !finite(row.DayPercent) || row.MarkAge < 0 ||
		tq.Symbol != "" && tq.Spread.QuoteAge < 0 {
		return Row{}, rejectMapping("ranking_source_row")
	}
	result := Row{Rank: uint64(row.Rank), Symbol: row.Symbol, Float: mapFloat(row.Float), Volume: mapShares(row.SessionVolume),
		LastUSD: row.Last, DayChangeRatio: percentagePointsToRatio(row.DayPercent), MarkAgeMS: durationMilliseconds(row.MarkAge),
		FromOpenChange: mapRatio(row.FromOpenPercent), DayRangePosition: mapRatio(row.DayRange), Activity30s: mapRatio(row.Activity30s),
		Move30s: mapRatio(row.Move30s), Tape5s: Tape5s{Status: "unselected"},
		Spread: Spread{Status: "unselected"}}
	if tq.Symbol == "" {
		return result, nil
	}
	// The public membership fact is defined by independent coverage evidence,
	// never by a successful write or a legacy adapter hint.
	present := tq.TradeCoverage && tq.QuoteCoverage
	result.TQMembership = TQMembership{Desired: tq.Desired, ProviderPresent: present, ProviderMembershipUnknown: tq.Desired && !present}
	// Tape.Status/Reason is the engine's authoritative five-second product
	// projection. The retained nested members exist for the superseded v1
	// shape and are required by engine publication validation to agree.
	result.Tape5s = Tape5s{Status: string(tq.Tape.Status), Reason: tq.Tape.Reason, TradeCoverage: tq.TradeCoverage,
		TimestampBasis: tq.Tape.TimestampBasis, LifecycleRecordsObserved: tq.Tape.LifecycleRecordsObserved}
	if tq.Tape.Status == engine.TQCurrent {
		result.Tape5s.TradesPerSecond = floatPointer(tq.Tape.FiveSecond)
	}
	result.Spread = Spread{Status: string(tq.Spread.Status), Reason: tq.Spread.Reason, QuoteCoverage: tq.QuoteCoverage,
		QuoteAgeMS: durationMilliseconds(tq.Spread.QuoteAge), Quality: tq.Spread.Quality}
	if tq.Spread.Status == engine.TQCurrent || tq.Spread.Status == engine.TQStale {
		result.Spread.Cents, result.Spread.BasisPoints = floatPointer(tq.Spread.Cents), floatPointer(tq.Spread.BasisPoints)
	}
	return result, nil
}

func mapShares(value engine.FieldView) ShareMeasurement {
	result := ShareMeasurement{Status: value.Status, Reason: value.Reason}
	if value.Status == "current" {
		result.ValueShares = floatPointer(value.Value)
	}
	return result
}

func mapFloat(value engine.FloatFieldView) FloatMeasurement {
	result := FloatMeasurement{Status: value.Status, Reason: value.Reason}
	if value.Status != "current" && value.Status != "stale" {
		return result
	}
	result.ValueShares = floatPointer(value.Value)
	if value.Percent != nil {
		result.PercentRatio = floatPointer(percentagePointsToRatio(*value.Percent))
	}
	result.Provider, result.Provenance = value.Provider, value.Provenance
	if value.EffectiveDate != "" {
		result.EffectiveDate = stringPointer(value.EffectiveDate)
	}
	if !value.RetrievedAt.IsZero() {
		retrieved := timestamp(value.RetrievedAt)
		result.RetrievedAt = &retrieved
	}
	return result
}

func mapRatio(value engine.FieldView) RatioMeasurement {
	result := RatioMeasurement{Status: value.Status, Reason: value.Reason}
	if value.Status == "current" {
		result.ValueRatio = floatPointer(percentagePointsToRatio(value.Value))
	}
	return result
}

// Aggregate feature values are engine-owned percentage points. The public v2
// schema deliberately exposes dimensionless ratios so every percentage field
// has one stable wire unit (for example, 0.125 means 12.5%).
func percentagePointsToRatio(value float64) float64 {
	return value / 100
}

func validateSnapshot(value Snapshot) error {
	p, w, hr, tq, commands, checkpoint := value.Accounting.Population, value.Recovery.Work, value.Recovery.Rows, value.TQ.Facts, value.TQ.Commands, value.Checkpoint
	d := value.Accounting.PopulationTransitionDiagnostic
	if value.SchemaVersion != SchemaVersion {
		return rejectMapping("schema_version")
	}
	if checkpoint != fixedDisabledCheckpoint() {
		return rejectMapping("checkpoint_disabled")
	}
	if !validTimestamp(value.Sample.SampledAt) || !validTimestamp(value.Publication.GeneratedAt) || !validTimestamp(value.Status.CausalTarget) || !validTradingDate(value.Publication.TradingDate) {
		return rejectMapping("required_time")
	}
	latency := value.Operations.DeliveryLatencyAttribution
	if !validPositiveDecimal(value.Sample.ID) || !validPositiveDecimal(value.Publication.ID) || !validDecimal(value.Publication.LastEngineSequence) ||
		!validDecimal(value.Publication.ConnectionEpoch) || !validDecimal(value.Recovery.Generation) || !validDecimal(value.TQ.PressureTransitions) ||
		!validDecimal(value.TQ.PressureFenced) || !validDecimal(value.TQ.Commands.ResultFenced) || !validDecimal(value.Operations.Deliveries) ||
		!validDecimal(value.Operations.ConsumerDeferred) || !validDecimal(value.Operations.HeapAllocBytes) || !validDecimal(value.Operations.HeapInUseBytes) ||
		!validDecimal(value.Operations.ConnectionRecoveryAttempts) || !sumDecimalEquals(value.Operations.Deliveries,
		latency.Aggregate, latency.TQ, latency.Control, latency.HydrationFence, latency.Checkpoint, latency.Timer, latency.Unknown) {
		return rejectMapping("decimal_encoding")
	}
	if !deliveryLatencyAttributionValid(value.Operations) {
		return rejectMapping("delivery_latency_attribution")
	}
	if value.Publication.RunMode != "live" {
		return rejectMapping("run_mode")
	}
	if !oneOf(value.Publication.Lifecycle, "initializing", "awaiting_session", "awaiting_aggregate_ack", "hydrating", "live", "recovering", "suppressed", "ended") {
		return rejectMapping("lifecycle")
	}
	if !oneOf(value.Publication.LifecycleReason, "", "binding_before_session", "binding_in_session", "binding_after_session", "session_start_without_aggregate_ack", "session_end", "controlled_stop", "sequence_exhaustion", "clock_regression", "canonical_integrity", "publication_integrity", "accounting_integrity", "closed", "aggregate_acknowledged", "aggregate_acknowledged_at_session_start", "aggregate_epoch_lost", "ingress_integrity", "hydration_complete", "recovery_exhausted", "scheduled_recovery") {
		return rejectMapping("lifecycle_reason")
	}
	if !oneOf(value.Publication.Suppression, "", "same_binding_recovery_allowed", "clean_reinitialization_required", "restart_required") {
		return rejectMapping("suppression")
	}
	if !oneOf(value.Status.ReadinessReason, "", "runtime_unavailable", "binding_mismatch", "not_live_mode", "lifecycle_not_ready", "suppressed", "aggregate_unacknowledged", "fence_pending", "ranking_noncurrent", "watermark_missing", "watermark_stale", "accounting_invalid") {
		return rejectMapping("readiness_reason")
	}
	if !oneOf(value.Ranking.Mode, "unavailable", "qualified_current", "degraded_bootstrap", "degraded_current", "stale", "suppressed") {
		return rejectMapping("ranking_mode")
	}
	if !oneOf(value.Ranking.Reason, "", "no_committed_watermark", "no_trusted_marks", "incomplete_population", "qualification_incomplete", "global_suppression") {
		return rejectMapping("ranking_reason")
	}
	if !oneOf(value.Recovery.Purpose, "", "fresh_bootstrap", "gap_recovery") {
		return rejectMapping("recovery_purpose")
	}
	if !oneOf(value.TQ.PressureMode, "normal", "taq_degraded", "aggregate_only") || value.Status.TQPressureMode != value.TQ.PressureMode || value.Status.TQShed != value.TQ.Shed ||
		value.TQ.AggregateOnly != (value.TQ.PressureMode == "aggregate_only") || value.TQ.Shed != (value.TQ.PressureMode != "normal") {
		return rejectMapping("tq_pressure_consistency")
	}
	if value.TQ.PressureMode == "normal" && value.TQ.PressureCause != "" || value.TQ.PressureMode != "normal" && !oneOf(value.TQ.PressureCause,
		"waiting_frames", "waiting_bytes", "oldest_waiting_frame", "aggregate_watermark_lag", "capacity_drop", "tq_retention_bound", "transport_accounting_loss") {
		return rejectMapping("tq_pressure_cause")
	}
	ps, pr := value.TQ.PressureSample, value.TQ.PressureRecovery
	if pr.RequiredSamples != 5 || pr.HealthySamples > pr.RequiredSamples || ps.RecoveryHealthy && !ps.Observed ||
		value.TQ.PressureMode != "normal" && pr.HealthySamples >= pr.RequiredSamples ||
		ps.Observed && (ps.FrameCapacity == 0 || ps.WaitingFrames > ps.FrameCapacity || ps.ByteCapacity == 0 || ps.WaitingBytes > ps.ByteCapacity) ||
		!ps.Observed && (ps.WaitingFrames != 0 || ps.FrameCapacity != 0 || ps.WaitingBytes != 0 || ps.ByteCapacity != 0 || ps.OldestWaitingFrameAgeMS != 0 || ps.AggregateWatermarkLagMS != 0 || ps.RecoveryHealthy) ||
		value.TQ.PressureMode != "normal" && pr.HealthySamples > 0 && !ps.RecoveryHealthy || value.TQ.PressureMode == "normal" && pr.HealthySamples != 0 {
		return rejectMapping("tq_pressure_recovery")
	}
	if !validOptionalTimestamp(value.Publication.CommittedT) || !validOptionalTimestamp(value.Recovery.Start) || !validOptionalTimestamp(value.Recovery.End) ||
		!validOptionalTimestamp(value.Recovery.SupportedThrough) || !validOptionalTimestamp(value.Publication.HydrationFence.SupportedThrough) {
		return rejectMapping("optional_time")
	}
	if (value.Publication.CommittedT == nil) != (value.Status.WatermarkLagMS == nil) ||
		value.Status.WatermarkLagMS != nil && (*value.Status.WatermarkLagMS < 0 || uint64(*value.Status.WatermarkLagMS) > maximumExactJSONInteger) {
		return rejectMapping("watermark_lag_consistency")
	}
	if !validPosition(value.Publication.AggregateAcknowledged, value.Publication.AggregateAckPosition) ||
		value.Publication.AggregateAcknowledged && value.Publication.AggregateAckPosition.ConnectionEpoch != value.Publication.ConnectionEpoch {
		return rejectMapping("aggregate_ack_position")
	}
	if !validHydrationFence(value.Publication.HydrationFence) {
		return rejectMapping("hydration_fence")
	}
	if !sumUint64Equals(p.UniverseTotal, p.ValidPriorClose, p.InvalidOrMissingPriorClose) {
		return rejectMapping("population_prior_close_identity")
	}
	if !sumUint64Equals(p.ValidPriorClose, p.TrustedRankableMark, p.TrustedBelowPriceMark, p.NoPrintThroughT, p.InvalidMark, p.UnknownDueFailureOrFence) {
		return rejectMapping("population_mark_identity")
	}
	if p.CoveredPopulation != p.UniverseTotal-p.UnknownDueFailureOrFence || p.UnresolvedPopulation != p.UnknownDueFailureOrFence {
		return rejectMapping("population_coverage_identity")
	}
	if !sumUint64Equals(d.BootstrapUnknown, d.TrustedByLaterLiveMark, d.NoLaterEligibleMark, d.LatestMarkNotLiveAuthority,
		d.NoStrictlyOlderLocalizedConflict, d.ConflictAtOrAfterMark, d.InvalidAtOrAfterMark, d.IncompletePostMarkCoverage) ||
		d.TrustedByLaterLiveMark > p.TrustedRankableMark+p.TrustedBelowPriceMark || d.BootstrapUnknown-d.TrustedByLaterLiveMark > p.UnknownDueFailureOrFence {
		return rejectMapping("population_transition_diagnostic_identity")
	}
	if !sumDecimalEquals(w.Planned, w.Open, w.CompletedValue, w.CompletedEmpty, w.Failed, w.Canceled, w.Fenced) {
		return rejectMapping("recovery_work_identity")
	}
	if value.Recovery.GenerationActive && (value.Recovery.Generation == "0" || value.Recovery.FenceReconciled) {
		return rejectMapping("recovery_generation_activity")
	}
	if !sumDecimalEquals(hr.Consumed, hr.Inserted, hr.Duplicate, hr.ConflictOrWithdrawal, hr.Rejected, hr.Fenced, hr.Integrity) {
		return rejectMapping("recovery_row_identity")
	}
	if !sumDecimalEquals(tq.Consumed, tq.Applied, tq.Duplicate, tq.Rejected, tq.Fenced, tq.PressureShed, tq.Integrity) {
		return rejectMapping("tq_fact_identity")
	}
	if !validDecimal(tq.NormalizedTrades) || !validDecimal(tq.NormalizedQuotes) || !decimalSumAtMost(tq.Applied, tq.AppliedTrades, tq.AppliedQuotes) || !sumDecimalEquals(tq.PressureShed, tq.PressureShedTrades, tq.PressureShedQuotes) {
		return rejectMapping("tq_family_fact_identity")
	}
	if !sumDecimalEquals(commands.Issued, commands.Pending, commands.Written, commands.Failed, commands.Fenced) {
		return rejectMapping("tq_command_identity")
	}
	for _, count := range []uint64{p.UniverseTotal, p.ValidPriorClose, p.InvalidOrMissingPriorClose, p.TrustedRankableMark, p.TrustedBelowPriceMark,
		p.NoPrintThroughT, p.InvalidMark, p.UnknownDueFailureOrFence, p.CoveredPopulation, p.UnresolvedPopulation, value.Ranking.TotalPassers,
		value.Ranking.KnownRankableCount, value.Ranking.DayInvalidRankable, value.Ranking.QualifiedDayInvalid,
		value.Accounting.Qualification.NotYetPassed, value.Accounting.Qualification.Provisional, value.Accounting.Qualification.Finalized, value.Accounting.Qualification.Unresolved,
		value.Accounting.Uncertainty.BootstrapOrigin, value.Accounting.Uncertainty.PostBootstrapGap, value.Accounting.Uncertainty.LocalInvalid,
		d.BootstrapUnknown, d.TrustedByLaterLiveMark, d.NoLaterEligibleMark, d.LatestMarkNotLiveAuthority, d.NoStrictlyOlderLocalizedConflict,
		d.ConflictAtOrAfterMark, d.InvalidAtOrAfterMark, d.IncompletePostMarkCoverage,
		value.TQ.KnownPresent, value.TQ.KnownAbsent, value.TQ.Unknown, value.Operations.QueueCapacityFrames,
		value.TQ.PressureMisses, value.TQ.RetainedTrades, value.TQ.RetainedQuotes, value.TQ.RetainedFingerprints,
		value.Operations.QueueCurrentFrames, value.Operations.QueueHighFrames, value.Operations.QueueCurrentBytes, value.Operations.QueueHighBytes,
		value.Operations.MeanProcessingDelayMS, value.Operations.MaxProcessingDelayMS, value.Operations.MaxProcessingDelayOneSecondMS, latency.MaximumMS, value.Operations.Goroutines} {
		if count > maximumExactJSONInteger {
			return rejectMapping("exact_json_integer_bound")
		}
	}
	if len(value.Rows) > 20 || len(value.TQ.DesiredSymbols) > 20 {
		return rejectMapping("product_row_bound")
	}
	if failure := value.Operations.IntegrityFailure; failure != nil {
		if !oneOf(failure.Category, "candidate_target_mismatch", "support_contradiction", "population_accounting", "qualification_accounting", "uncertainty_accounting", "feature_accounting", "ranking_projection", "ranking_row", "tq_intent", "unknown_evaluator_integrity") ||
			!validPositiveDecimal(failure.EngineSequence) || !validOptionalTimestamp(failure.CandidateTime) || !validOptionalTimestamp(failure.ExpectedTime) ||
			(failure.FirstField == "") != (failure.FirstReason == "") {
			return rejectMapping("evaluator_integrity_diagnostic")
		}
	}
	seenRows := make(map[string]struct{}, len(value.Rows))
	for index, row := range value.Rows {
		if row.Rank != uint64(index+1) || row.Symbol == "" || !finite(row.LastUSD) || !finite(row.DayChangeRatio) || !measurementsValid(row) {
			return rejectMapping("product_row")
		}
		if _, exists := seenRows[row.Symbol]; exists {
			return rejectMapping("duplicate_product_row")
		}
		if row.TQMembership.ProviderPresent != (row.Tape5s.TradeCoverage && row.Spread.QuoteCoverage) ||
			row.TQMembership.ProviderMembershipUnknown != (row.TQMembership.Desired && !row.TQMembership.ProviderPresent) {
			return rejectMapping("tq_membership_confirmation")
		}
		seenRows[row.Symbol] = struct{}{}
	}
	seenDesired := make(map[string]struct{}, len(value.TQ.DesiredSymbols))
	for _, symbol := range value.TQ.DesiredSymbols {
		if symbol == "" {
			return rejectMapping("empty_tq_desired_symbol")
		}
		if _, exists := seenDesired[symbol]; exists {
			return rejectMapping("duplicate_tq_desired_symbol")
		}
		seenDesired[symbol] = struct{}{}
	}
	return nil
}

func measurementsValid(row Row) bool {
	if !ratioMeasurementValid(row.FromOpenChange, "from_open") || !ratioMeasurementValid(row.DayRangePosition, "day_range") ||
		!ratioMeasurementValid(row.Activity30s, "activity_30s") || !ratioMeasurementValid(row.Move30s, "move_30s") {
		return false
	}
	if !shareMeasurementValid(row.Volume) || !floatMeasurementValid(row.Float) ||
		!tape5sValid(row.Tape5s) ||
		(row.Tape5s.Status == "current") != (row.Tape5s.TradesPerSecond != nil) ||
		row.Tape5s.TradesPerSecond != nil && (!finite(*row.Tape5s.TradesPerSecond) || *row.Tape5s.TradesPerSecond < 0) {
		return false
	}
	if row.DayRangePosition.ValueRatio != nil && (*row.DayRangePosition.ValueRatio < 0 || *row.DayRangePosition.ValueRatio > 1) ||
		row.Activity30s.ValueRatio != nil && (*row.Activity30s.ValueRatio < 0 || *row.Activity30s.ValueRatio > 1) {
		return false
	}
	if !spreadTupleValid(row.Spread) || !oneOf(row.Spread.Quality, "", "reviewed_ordinary", "known_special", "unclassified") ||
		(row.Spread.Status == "current" || row.Spread.Status == "stale") != (row.Spread.Cents != nil && row.Spread.BasisPoints != nil) ||
		(row.Spread.Cents == nil) != (row.Spread.BasisPoints == nil) || row.Spread.Cents != nil &&
		(!finite(*row.Spread.Cents) || !finite(*row.Spread.BasisPoints) || *row.Spread.Cents < 0 || *row.Spread.BasisPoints < 0) {
		return false
	}
	return true
}

func ratioMeasurementValid(value RatioMeasurement, field string) bool {
	if value.ValueRatio != nil && !finite(*value.ValueRatio) {
		return false
	}
	if value.Status == "current" {
		return value.Reason == "" && value.ValueRatio != nil
	}
	if value.ValueRatio != nil {
		return false
	}
	switch field {
	case "from_open":
		return statusReason(value.Status, value.Reason,
			[]string{}, []string{"before_first_print", "history_incomplete"}, []string{"historical_conflict", "invalid_input", "state_bound_exceeded"})
	case "day_range":
		return statusReason(value.Status, value.Reason,
			[]string{}, []string{"before_first_print", "history_incomplete", "zero_width"}, []string{"historical_conflict", "invalid_input", "state_bound_exceeded"})
	case "activity_30s":
		return statusReason(value.Status, value.Reason,
			[]string{"reference_warmup"}, []string{"history_incomplete"}, []string{"historical_conflict", "invalid_input", "state_bound_exceeded"})
	case "move_30s":
		return statusReason(value.Status, value.Reason,
			[]string{"before_first_print", "rolling_warmup"}, []string{"history_incomplete", "no_aggregate_in_target"}, []string{"historical_conflict", "invalid_input", "state_bound_exceeded"})
	default:
		return false
	}
}

func shareMeasurementValid(value ShareMeasurement) bool {
	if value.Status == "current" {
		return value.Reason == "" && value.ValueShares != nil && finite(*value.ValueShares) && *value.ValueShares >= 0
	}
	if value.ValueShares != nil {
		return false
	}
	return statusReason(value.Status, value.Reason, nil, []string{"history_incomplete"}, []string{"historical_conflict", "invalid_input", "state_bound_exceeded"})
}

func floatMeasurementValid(value FloatMeasurement) bool {
	available := value.Status == "current" || value.Status == "stale"
	if !oneOf(value.Status, "current", "stale", "unavailable", "invalid") ||
		available != (value.ValueShares != nil) || !available && (value.PercentRatio != nil || value.Provider != "" || value.EffectiveDate != nil || value.RetrievedAt != nil || value.Provenance != "") {
		return false
	}
	if !available {
		return value.Status == "unavailable" && value.Reason == "not_available" || value.Status == "invalid" && value.Reason == "invalid_provenance"
	}
	if !finite(*value.ValueShares) || *value.ValueShares <= 0 || value.Provider != "massive-stocks-float-experimental" || value.RetrievedAt == nil || !validTimestamp(*value.RetrievedAt) ||
		value.PercentRatio != nil && (!finite(*value.PercentRatio) || *value.PercentRatio < 0 || *value.PercentRatio > 1) ||
		value.EffectiveDate != nil && !validTradingDate(*value.EffectiveDate) {
		return false
	}
	return value.Status == "current" && value.Reason == "" && value.Provenance == "fresh" ||
		value.Status == "stale" && value.Reason == "cached_fallback" && value.Provenance == "cache"
}

func tape5sValid(value Tape5s) bool {
	if !oneOf(value.TimestampBasis, "", "none", "participant", "sip_fallback", "mixed") {
		return false
	}
	switch value.Status {
	case "unselected":
		return value.Reason == "" && !value.TradeCoverage
	case "warming":
		return value.TradeCoverage && oneOf(value.Reason, "coverage_warming", "five_second_warming", "aggregate_watermark_stale", "watermark_recovery_warming")
	case "current":
		return value.TradeCoverage && value.Reason == "qualifying_original_prints"
	case "unavailable":
		return oneOf(value.Reason, "coverage", "channel_unconfirmed", "control_error")
	case "invalid":
		return value.TradeCoverage && value.Reason == "unequal_repeat"
	case "pressure_shed":
		return value.Reason == "pressure"
	default:
		return false
	}
}

func spreadTupleValid(value Spread) bool {
	switch value.Status {
	case "unselected":
		return value.Reason == "" && !value.QuoteCoverage
	case "warming":
		return value.QuoteCoverage && oneOf(value.Reason, "coverage_warming", "aggregate_watermark_stale", "watermark_recovery_warming", "watermark_recovery_quote_warming")
	case "current":
		return value.QuoteCoverage && value.Reason == ""
	case "stale":
		return value.QuoteCoverage && value.Reason == "stale_quote"
	case "unavailable":
		return value.Reason == "coverage" || value.Reason == "channel_unconfirmed" || value.Reason == "control_error" || value.QuoteCoverage && value.Reason == "one_sided_quote"
	case "invalid":
		return value.QuoteCoverage && value.Reason == "crossed_quote"
	case "pressure_shed":
		return value.Reason == "pressure"
	default:
		return false
	}
}

func statusReason(status, reason string, warming, unavailable, invalid []string) bool {
	switch status {
	case "warming":
		return oneOf(reason, warming...)
	case "unavailable":
		return oneOf(reason, unavailable...)
	case "invalid":
		return oneOf(reason, invalid...)
	default:
		return false
	}
}

func fieldReason(value string) bool {
	return oneOf(value, "", "before_first_print", "history_incomplete", "prior_close_unavailable", "no_aggregate_in_target", "rolling_warmup", "reference_warmup", "zero_width", "historical_conflict", "invalid_input", "state_bound_exceeded")
}

func tqStatus(value string) bool {
	return oneOf(value, "unselected", "warming", "current", "stale", "unavailable", "invalid", "pressure_shed")
}

func tqReason(value string) bool {
	return oneOf(value, "", "coverage", "channel_unconfirmed", "control_error", "coverage_warming", "five_second_warming", "qualifying_original_prints", "unequal_repeat", "one_sided_quote", "crossed_quote", "stale_quote", "insufficient_coverage", "pressure")
}

func validPosition(present bool, value Position) bool {
	if !validDecimal(value.ConnectionEpoch) || !validDecimal(value.FrameSequence) || value.ArrayIndex > maximumExactJSONInteger {
		return false
	}
	if !present {
		return value.ConnectionEpoch == "0" && value.FrameSequence == "0" && value.ArrayIndex == 0
	}
	return value.ConnectionEpoch != "0" && value.FrameSequence != "0"
}

func validHydrationFence(value HydrationFence) bool {
	if !validDecimal(value.ConnectionEpoch) || !validDecimal(value.ThroughFrameSequence) || !validDecimal(value.MarkerOrdinal) {
		return false
	}
	if !value.Reconciled {
		return value.ConnectionEpoch == "0" && value.ThroughFrameSequence == "0" && value.MarkerOrdinal == "0" && value.SupportedThrough == nil
	}
	return value.ConnectionEpoch != "0" && value.ThroughFrameSequence != "0" && value.MarkerOrdinal != "0" && value.SupportedThrough != nil
}

func validDecimal(value string) bool {
	parsed, err := strconv.ParseUint(value, 10, 64)
	return err == nil && decimal(parsed) == value
}

func validPositiveDecimal(value string) bool { return value != "0" && validDecimal(value) }

func deliveryLatencyAttributionValid(value Operations) bool {
	attribution := value.DeliveryLatencyAttribution
	if !oneOf(attribution.MaximumFamily, "aggregate", "tq", "control", "hydration_fence", "timer", "unknown") ||
		attribution.MaximumMS != value.MaxProcessingDelayOneSecondMS {
		return false
	}
	if attribution.MaximumMS == 0 && attribution.MaximumFamily == "unknown" {
		return true
	}
	countByFamily := attribution.Unknown
	switch attribution.MaximumFamily {
	case "aggregate":
		countByFamily = attribution.Aggregate
	case "tq":
		countByFamily = attribution.TQ
	case "control":
		countByFamily = attribution.Control
	case "hydration_fence":
		countByFamily = attribution.HydrationFence
	case "timer":
		countByFamily = attribution.Timer
	}
	count, err := strconv.ParseUint(countByFamily, 10, 64)
	return err == nil && count != 0
}

func validTimestamp(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && parsed.Location() == time.UTC && timestamp(parsed) == value
}

func validOptionalTimestamp(value *string) bool { return value == nil || validTimestamp(*value) }

func validTradingDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func decimal(value uint64) string { return strconv.FormatUint(value, 10) }

func sumDecimalEquals(total string, terms ...string) bool {
	want, err := strconv.ParseUint(total, 10, 64)
	if err != nil || decimal(want) != total {
		return false
	}
	var sum uint64
	for _, term := range terms {
		value, err := strconv.ParseUint(term, 10, 64)
		if err != nil || decimal(value) != term || math.MaxUint64-sum < value {
			return false
		}
		sum += value
	}
	return sum == want
}

func decimalSumAtMost(total string, terms ...string) bool {
	want, err := strconv.ParseUint(total, 10, 64)
	if err != nil || decimal(want) != total {
		return false
	}
	var sum uint64
	for _, term := range terms {
		value, err := strconv.ParseUint(term, 10, 64)
		if err != nil || decimal(value) != term || math.MaxUint64-sum < value {
			return false
		}
		sum += value
	}
	return sum <= want
}

func sumUint64Equals(total uint64, terms ...uint64) bool {
	var sum uint64
	for _, value := range terms {
		if math.MaxUint64-sum < value {
			return false
		}
		sum += value
	}
	return sum == total
}

func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func optionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	result := timestamp(*value)
	return &result
}

func optionalNonzeroTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	result := timestamp(value)
	return &result
}

func validUTC(value time.Time) bool { return !value.IsZero() && value == value.UTC() }
func finite(value float64) bool     { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func sameOptionalTime(left, right *time.Time) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Equal(*right)
}

func floatPointer(value float64) *float64 {
	copyValue := value
	return &copyValue
}

func stringPointer(value string) *string {
	copyValue := value
	return &copyValue
}

func durationMilliseconds(value time.Duration) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value / time.Millisecond)
}

func nonnegativeInt(value int) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}
