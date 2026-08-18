package snapshotapi

import (
	"encoding/hex"
	"math"
	"strconv"
	"strings"
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
		Checkpoint: mapCheckpoint(operational.InstalledCheckpoint, capture.Metrics),
		Operations: mapOperations(capture.Metrics, operational),
	}
	if capture.Replay != nil {
		result.Replay = mapReplay(*capture.Replay)
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
		result.Rows[index] = mapped
	}
	if result.Replay != nil {
		applyReplayPresentation(&result)
	}
	if err := validateSnapshot(result); err != nil {
		return Snapshot{}, err
	}
	return result, nil
}

func mapReplay(value operations.ReplayCaptureView) *Replay {
	result := &Replay{Phase: string(value.Phase), ArtifactID: value.ArtifactID, ArtifactEnd: timestamp(value.ArtifactEnd),
		ObservationStart: timestamp(value.ObservationStart), ObservationEnd: timestamp(value.ObservationEnd), LogicalTime: timestamp(value.LogicalTime),
		Completion: string(value.Completion),
		Source: ReplaySource{ArtifactRecords: decimal(value.Source.ArtifactRecords), CompletedRecordDispositions: decimal(value.Source.CompletedRecordDispositions),
			IntentionallyUnappliedSuffixRecords: decimal(value.Source.IntentionallyUnappliedSuffixRecords), UnreadRecords: decimal(value.Source.UnreadRecords),
			PlannedGroups: decimal(value.Source.PlannedGroups), CompletedGroups: decimal(value.Source.CompletedGroups), ActiveGroup: decimal(value.Source.ActiveGroup),
			RemainingGroups: decimal(value.Source.RemainingGroups), CompletedRuns: decimal(value.Source.CompletedRuns), FailedRuns: decimal(value.Source.FailedRuns), CanceledRuns: decimal(value.Source.CanceledRuns)},
		Window: ReplayWindow{WarmupGroupsPlanned: decimal(value.Window.WarmupGroupsPlanned), WarmupGroupsCompleted: decimal(value.Window.WarmupGroupsCompleted),
			WarmupGroupActive: decimal(value.Window.WarmupGroupActive), WarmupGroupsRemaining: decimal(value.Window.WarmupGroupsRemaining),
			ObservationSecondsPlanned: decimal(value.Window.ObservationSecondsPlanned), ObservationSecondsCompleted: decimal(value.Window.ObservationSecondsCompleted),
			ObservationSecondActive: decimal(value.Window.ObservationSecondActive), ObservationSecondsRemaining: decimal(value.Window.ObservationSecondsRemaining),
			ObservationBoundariesPublished: decimal(value.Window.ObservationBoundariesPublished)}}
	if value.ScheduleLag != nil {
		lag := durationMilliseconds(*value.ScheduleLag)
		result.ScheduleLagMS = &lag
	}
	return result
}

func applyReplayPresentation(value *Snapshot) {
	value.Status.BackendReady = false
	value.Status.ReadinessReason = "not_live_mode"
	value.TQ.DesiredSymbols = []string{}
	for index := range value.Rows {
		value.Rows[index].Tape5s = Tape5s{Status: "unavailable", Reason: "replay_unavailable"}
		value.Rows[index].Spread = Spread{Status: "unavailable", Reason: "replay_unavailable"}
		value.Rows[index].TQMembership = TQMembership{}
	}
	switch value.Replay.Phase {
	case "warming":
		value.Ranking = Ranking{Mode: "unavailable", Reason: "replay_warming"}
		value.Rows = []Row{}
		value.Status.RankingCurrent = false
	case "canceling", "shutting_down":
		value.Ranking = Ranking{Mode: "unavailable", Reason: "no_committed_watermark"}
		value.Rows = []Row{}
		value.Status.RankingCurrent = false
	case "suppressed":
		value.Ranking = Ranking{Mode: "suppressed", Reason: "global_suppression"}
		value.Rows = []Row{}
		value.Status.RankingCurrent = false
	case "retained_success":
		value.Status.RankingCurrent = false
	}
}

func mapPublication(p engine.ReplayPublicationView, operational engine.OperationalView) Publication {
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

func mapRanking(value engine.ReplayEvaluationView) Ranking {
	return Ranking{Mode: value.Mode, Reason: value.Reason, TotalPassers: value.TotalPassers, KnownRankableCount: value.KnownRankableCount,
		DayInvalidRankable: value.DayInvalidRankable, QualifiedDayInvalid: value.QualifiedDayInvalid}
}

func mapAccounting(value engine.ReplayEvaluationView) Accounting {
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
	written := c.Written
	if written < c.Acknowledged { // legacy in-process fixture compatibility
		written = c.Acknowledged
	}
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
		Commands: TQCommands{Issued: decimal(c.Issued), Pending: decimal(c.Pending), Written: decimal(written), Failed: decimal(c.Failed),
			Fenced: decimal(c.Fenced), ResultFenced: decimal(c.ResultFenced)}}
}

func mapCheckpoint(installed bool, metrics operations.Metrics) Checkpoint {
	w, e := metrics.Checkpoint, metrics.CheckpointEngine
	age := time.Duration(0)
	if !e.LastSuccessfulT0.IsZero() && metrics.SampledAt.After(e.LastSuccessfulT0) {
		age = metrics.SampledAt.Sub(e.LastSuccessfulT0)
	}
	return Checkpoint{Installed: installed, Eligible: decimal(e.Eligible), PressureDeferred: decimal(e.PressureDeferred),
		ProjectionStarted: decimal(e.ProjectionStarted), ProjectionInProgress: decimal(e.ProjectionInProgress), Projected: decimal(e.Projected), ProjectionRejected: decimal(e.ProjectionRejected),
		SubmitRejected: decimal(e.SubmitRejected), Submitted: decimal(e.Submitted), Outstanding: decimal(e.Outstanding),
		InProgress: decimal(w.InProgress), Pending: decimal(w.Pending), Completed: decimal(e.Completed), Failed: decimal(e.Failed),
		Canceled: decimal(e.Canceled), Superseded: decimal(e.Superseded), LastAttemptedT0: optionalNonzeroTime(e.LastAttemptedT0),
		LastProjectedT0: optionalNonzeroTime(e.LastProjectedT0), LastSubmittedT0: optionalNonzeroTime(e.LastSubmittedT0),
		LastSuccessfulT0: optionalNonzeroTime(e.LastSuccessfulT0), UsableAgeMS: durationMilliseconds(age),
		ProjectionTotalMS: durationMilliseconds(e.LastProjectionTotal), ProjectionLockMS: durationMilliseconds(e.LastProjectionLock),
		ArtifactBytes: decimal(uint64(max(0, w.LastArtifactBytes))), WriteMS: durationMilliseconds(w.LastWriteDuration),
		EncodeMS: durationMilliseconds(w.LastEncodeDuration), ReopenValidationMS: durationMilliseconds(w.LastReopenValidationDuration),
		LastFailureStep: string(w.LastFailureStep), LastProjectionFailure: e.LastProjectionFailure, LastSubmitFailure: e.LastSubmitFailure}
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
			HydrationFence: decimal(attribution.HydrationFence), Checkpoint: decimal(attribution.Checkpoint), Timer: decimal(attribution.Timer), Unknown: decimal(attribution.Unknown),
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

func mapRow(row engine.ReplayRankingRowView, tq engine.TQSymbolView) (Row, error) {
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

func mapShares(value engine.ReplayFieldView) ShareMeasurement {
	result := ShareMeasurement{Status: value.Status, Reason: value.Reason}
	if value.Status == "current" {
		result.ValueShares = floatPointer(value.Value)
	}
	return result
}

func mapFloat(value engine.ReplayFloatFieldView) FloatMeasurement {
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

func mapRatio(value engine.ReplayFieldView) RatioMeasurement {
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
	if !oneOf(value.Publication.RunMode, "live", "replay") {
		return rejectMapping("run_mode")
	}
	if !oneOf(value.Publication.Lifecycle, "initializing", "awaiting_session", "awaiting_aggregate_ack", "hydrating", "live", "recovering", "replaying", "suppressed", "ended") {
		return rejectMapping("lifecycle")
	}
	if !oneOf(value.Publication.LifecycleReason, "", "binding_before_session", "binding_in_session", "binding_after_session", "session_start_without_aggregate_ack", "session_end", "controlled_stop", "sequence_exhaustion", "clock_regression", "canonical_integrity", "publication_integrity", "accounting_integrity", "closed", "replay_start", "replay_end", "replay_requested_end", "replay_failure", "aggregate_acknowledged", "aggregate_acknowledged_at_session_start", "aggregate_epoch_lost", "ingress_integrity", "hydration_complete", "recovery_exhausted", "scheduled_recovery") {
		return rejectMapping("lifecycle_reason")
	}
	if !oneOf(value.Publication.Suppression, "", "same_binding_recovery_allowed", "clean_reinitialization_required", "restart_required", "terminal_replay_failure") {
		return rejectMapping("suppression")
	}
	if !oneOf(value.Status.ReadinessReason, "", "runtime_unavailable", "binding_mismatch", "not_live_mode", "lifecycle_not_ready", "suppressed", "aggregate_unacknowledged", "fence_pending", "ranking_noncurrent", "watermark_missing", "watermark_stale", "accounting_invalid") {
		return rejectMapping("readiness_reason")
	}
	if !oneOf(value.Ranking.Mode, "unavailable", "qualified_current", "degraded_bootstrap", "degraded_current", "stale", "suppressed") {
		return rejectMapping("ranking_mode")
	}
	if !oneOf(value.Ranking.Reason, "", "no_committed_watermark", "no_trusted_marks", "incomplete_population", "qualification_incomplete", "global_suppression", "replay_warming") {
		return rejectMapping("ranking_reason")
	}
	if !oneOf(value.Recovery.Purpose, "", "fresh_bootstrap", "checkpoint_catchup", "gap_recovery") {
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
		!validOptionalTimestamp(value.Recovery.SupportedThrough) || !validOptionalTimestamp(value.Publication.HydrationFence.SupportedThrough) ||
		!validOptionalTimestamp(checkpoint.LastAttemptedT0) || !validOptionalTimestamp(checkpoint.LastProjectedT0) ||
		!validOptionalTimestamp(checkpoint.LastSubmittedT0) || !validOptionalTimestamp(checkpoint.LastSuccessfulT0) {
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
	if !sumDecimalEquals(checkpoint.Submitted, checkpoint.Outstanding, checkpoint.Completed, checkpoint.Failed, checkpoint.Canceled, checkpoint.Superseded) {
		return rejectMapping("checkpoint_identity")
	}
	if !sumDecimalEquals(checkpoint.Eligible, checkpoint.PressureDeferred, checkpoint.ProjectionStarted) ||
		!sumDecimalEquals(checkpoint.ProjectionStarted, checkpoint.ProjectionInProgress, checkpoint.Projected, checkpoint.ProjectionRejected) ||
		!sumDecimalEquals(checkpoint.Projected, checkpoint.Submitted, checkpoint.SubmitRejected) {
		return rejectMapping("checkpoint_projection_identity")
	}
	if checkpoint.ProjectionInProgress != "0" && checkpoint.ProjectionInProgress != "1" {
		return rejectMapping("checkpoint_projection_in_progress")
	}
	if !oneOf(checkpoint.LastProjectionFailure, "", "sequence_exhausted", "ownership_invalidated", "projection_invariant", "sealed_t0_marker_invalid", "pre_t0_mutation", "sequence_changed", "shutdown") ||
		!oneOf(checkpoint.LastSubmitFailure, "", "request_invalid", "writer_rejected") {
		return rejectMapping("checkpoint_failure_reason")
	}
	if !validReplaySnapshot(value) {
		return rejectMapping("replay_consistency")
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
		value.Operations.MeanProcessingDelayMS, value.Operations.MaxProcessingDelayMS, value.Operations.MaxProcessingDelayOneSecondMS, latency.MaximumMS, value.Operations.Goroutines,
		checkpoint.UsableAgeMS, checkpoint.ProjectionTotalMS, checkpoint.ProjectionLockMS, checkpoint.WriteMS, checkpoint.EncodeMS, checkpoint.ReopenValidationMS} {
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
		return value.TradeCoverage && oneOf(value.Reason, "coverage_warming", "five_second_warming")
	case "current":
		return value.TradeCoverage && value.Reason == "qualifying_original_prints"
	case "unavailable":
		return oneOf(value.Reason, "coverage", "channel_unconfirmed", "control_error", "replay_unavailable")
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
		return value.QuoteCoverage && value.Reason == "coverage_warming"
	case "current":
		return value.QuoteCoverage && value.Reason == ""
	case "stale":
		return value.QuoteCoverage && value.Reason == "stale_quote"
	case "unavailable":
		return value.Reason == "coverage" || value.Reason == "channel_unconfirmed" || value.Reason == "control_error" || value.Reason == "replay_unavailable" || value.QuoteCoverage && value.Reason == "one_sided_quote"
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
	return oneOf(value, "", "coverage", "channel_unconfirmed", "control_error", "coverage_warming", "five_second_warming", "qualifying_original_prints", "unequal_repeat", "one_sided_quote", "crossed_quote", "stale_quote", "insufficient_coverage", "pressure", "replay_unavailable")
}

func validReplaySnapshot(value Snapshot) bool {
	if value.Publication.RunMode == "live" {
		return value.Replay == nil
	}
	if value.Replay == nil || value.Status.BackendReady || value.Status.ReadinessReason != "not_live_mode" || !validArtifactID(value.Replay.ArtifactID) ||
		!validTimestamp(value.Replay.ArtifactEnd) || !validTimestamp(value.Replay.ObservationStart) || !validTimestamp(value.Replay.ObservationEnd) || !validTimestamp(value.Replay.LogicalTime) {
		return false
	}
	r := value.Replay
	artifactEnd, _ := time.Parse(time.RFC3339Nano, r.ArtifactEnd)
	start, _ := time.Parse(time.RFC3339Nano, r.ObservationStart)
	end, _ := time.Parse(time.RFC3339Nano, r.ObservationEnd)
	logical, _ := time.Parse(time.RFC3339Nano, r.LogicalTime)
	generated, _ := time.Parse(time.RFC3339Nano, value.Publication.GeneratedAt)
	if !start.Before(end) || end.After(artifactEnd) || generated.After(logical) || logical.Before(start) && !oneOf(r.Phase, "warming", "canceling", "suppressed", "shutting_down") || logical.After(end) ||
		!oneOf(r.Phase, "warming", "observing", "finalizing", "retained_success", "canceling", "suppressed", "shutting_down") {
		return false
	}
	switch r.Phase {
	case "warming":
		if logical.After(start) || value.Publication.Lifecycle != "replaying" || value.Status.RankingCurrent || len(value.Rows) != 0 || value.Ranking.Mode != "unavailable" || value.Ranking.Reason != "replay_warming" {
			return false
		}
	case "observing":
		if logical.Before(start) || !logical.Before(end) || value.Publication.Lifecycle != "replaying" {
			return false
		}
	case "finalizing":
		if logical != end || value.Publication.Lifecycle != "replaying" {
			return false
		}
	case "retained_success":
		artifactCompletion := r.Completion == "artifact_end"
		requestedCompletion := r.Completion == "requested_end"
		if !oneOf(r.Completion, "artifact_end", "requested_end") || artifactCompletion != end.Equal(artifactEnd) ||
			artifactCompletion != (value.Publication.LifecycleReason == "replay_end") || requestedCompletion != (value.Publication.LifecycleReason == "replay_requested_end") ||
			value.Publication.Lifecycle != "ended" || logical != end || generated != logical {
			return false
		}
	case "canceling":
		if value.Publication.Lifecycle != "ended" || value.Publication.LifecycleReason != "controlled_stop" || value.Status.RankingCurrent || len(value.Rows) != 0 {
			return false
		}
	case "suppressed":
		if value.Publication.Lifecycle != "suppressed" || value.Publication.LifecycleReason != "replay_failure" || value.Publication.Suppression != "terminal_replay_failure" || value.Status.RankingCurrent || len(value.Rows) != 0 {
			return false
		}
	case "shutting_down":
		if value.Status.RankingCurrent || len(value.Rows) != 0 {
			return false
		}
	}
	if r.Phase != "retained_success" && r.Completion != "" {
		return false
	}
	if (r.ScheduleLagMS == nil) != oneOf(r.Phase, "warming", "canceling", "suppressed", "shutting_down") {
		return false
	}
	s := r.Source
	if !sumDecimalEquals(s.ArtifactRecords, s.CompletedRecordDispositions, s.IntentionallyUnappliedSuffixRecords, s.UnreadRecords) ||
		!sumDecimalEquals(s.PlannedGroups, s.CompletedGroups, s.ActiveGroup, s.RemainingGroups) {
		return false
	}
	switch r.Phase {
	case "retained_success":
		if s.CompletedRuns != "1" || s.FailedRuns != "0" || s.CanceledRuns != "0" || s.ActiveGroup != "0" || s.RemainingGroups != "0" || s.UnreadRecords != "0" ||
			r.Completion == "artifact_end" && s.IntentionallyUnappliedSuffixRecords != "0" {
			return false
		}
	case "canceling":
		if s.CompletedRuns != "0" || s.FailedRuns != "0" || s.CanceledRuns != "1" {
			return false
		}
	case "suppressed":
		if s.CompletedRuns != "0" || s.FailedRuns != "1" || s.CanceledRuns != "0" {
			return false
		}
	}
	w := r.Window
	if !sumDecimalEquals(w.WarmupGroupsPlanned, w.WarmupGroupsCompleted, w.WarmupGroupActive, w.WarmupGroupsRemaining) ||
		!sumDecimalEquals(w.ObservationSecondsPlanned, w.ObservationSecondsCompleted, w.ObservationSecondActive, w.ObservationSecondsRemaining) {
		return false
	}
	if r.Phase == "warming" {
		planned, plannedErr := strconv.ParseUint(w.WarmupGroupsPlanned, 10, 64)
		completedWarmup, completedErr := strconv.ParseUint(w.WarmupGroupsCompleted, 10, 64)
		if plannedErr != nil || completedErr != nil || completedWarmup >= planned {
			return false
		}
	}
	if r.Phase == "retained_success" && (w.WarmupGroupActive != "0" || w.WarmupGroupsRemaining != "0" || w.ObservationSecondActive != "0" || w.ObservationSecondsRemaining != "0") {
		return false
	}
	completed, err1 := strconv.ParseUint(w.ObservationSecondsCompleted, 10, 64)
	boundaries, err2 := strconv.ParseUint(w.ObservationBoundariesPublished, 10, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	if boundaries == 0 {
		return completed == 0 && !oneOf(r.Phase, "observing", "finalizing", "retained_success")
	}
	return boundaries == completed+1 && r.Phase != "warming"
}

func validArtifactID(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digest := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(digest)
	return err == nil && hex.EncodeToString(decoded) == digest
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
	if !oneOf(attribution.MaximumFamily, "aggregate", "tq", "control", "hydration_fence", "checkpoint", "timer", "unknown") ||
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
	case "checkpoint":
		countByFamily = attribution.Checkpoint
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
