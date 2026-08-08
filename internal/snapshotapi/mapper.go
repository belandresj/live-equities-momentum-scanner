package snapshotapi

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

const maximumExactJSONInteger = uint64(1<<53 - 1)

func Map(capture operations.SnapshotCapture) (Snapshot, error) {
	view, ok := operations.InspectSnapshotCapture(capture)
	if !ok {
		return Snapshot{}, errors.New("snapshot capture unavailable")
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
		return Snapshot{}, errors.New("invalid snapshot capture")
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
		TQ:         mapTQ(capture.Engine.TQ),
		Checkpoint: mapCheckpoint(operational.InstalledCheckpoint, capture.Metrics),
		Operations: mapOperations(capture.Metrics, operational),
	}
	if publication.Watermark != nil {
		lag := capture.Status.WatermarkLag.Milliseconds()
		if lag < 0 {
			return Snapshot{}, errors.New("negative watermark lag")
		}
		result.Status.WatermarkLagMS = &lag
	}

	tqRows := make(map[string]engine.TQSymbolView, len(capture.Engine.TQ.Rows))
	for _, row := range capture.Engine.TQ.Rows {
		if row.Symbol == "" {
			return Snapshot{}, errors.New("invalid T/Q row")
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
	if err := validateSnapshot(result); err != nil {
		return Snapshot{}, err
	}
	return result, nil
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
	p, q, u := value.Population, value.Qualification, value.Uncertainty
	return Accounting{
		Population: PopulationAccounting{UniverseTotal: p.UniverseTotal, ValidPriorClose: p.ValidPriorClose, InvalidOrMissingPriorClose: p.InvalidOrMissingPriorClose,
			TrustedRankableMark: p.TrustedRankableMark, TrustedBelowPriceMark: p.TrustedBelowPriceMark, NoPrintThroughT: p.NoPrintThroughT,
			InvalidMark: p.InvalidMark, UnknownDueFailureOrFence: p.UnknownDueFailureOrFence, CoveredPopulation: p.CoveredPopulation, UnresolvedPopulation: p.UnresolvedPopulation},
		Qualification: QualificationAccounting{NotYetPassed: q.NotYetPassed, Provisional: q.Provisional, Finalized: q.Finalized, Unresolved: q.Unresolved},
		Uncertainty:   UncertaintyAccounting{BootstrapOrigin: u.BootstrapOrigin, PostBootstrapGap: u.PostBootstrapGap, LocalInvalid: u.LocalInvalid},
	}
}

func mapRecovery(value engine.OperationalHydration) Recovery {
	a, rows := value.Accounting, value.Rows
	return Recovery{Purpose: string(value.Purpose), Generation: decimal(value.Generation), Start: optionalNonzeroTime(value.Start), End: optionalNonzeroTime(value.End),
		SupportedThrough: optionalTime(value.SupportedThrough), FenceReconciled: value.FenceReconciled, PolicyWaiting: value.PolicyWaiting,
		Work: RecoveryWork{Planned: decimal(a.Planned), Open: decimal(a.Open), CompletedValue: decimal(a.CompletedValue), CompletedEmpty: decimal(a.CompletedEmpty),
			Failed: decimal(a.Failed), Canceled: decimal(a.Canceled), Fenced: decimal(a.Fenced)},
		Rows: RecoveryRows{Consumed: decimal(rows.Consumed), Inserted: decimal(rows.Inserted), Duplicate: decimal(rows.Duplicate), ConflictOrWithdrawal: decimal(rows.ConflictOrWithdrawal),
			Rejected: decimal(rows.Rejected), Fenced: decimal(rows.Fenced), Integrity: decimal(rows.Integrity)}}
}

func mapTQ(value engine.TQView) TQ {
	a, c := value.Accounting, value.Commands
	return TQ{DesiredSymbols: append([]string{}, value.Desired...), PressureMode: string(value.Pressure), AggregateOnly: value.AggregateOnly,
		Shed: value.ShedTradesQuotes, RetainedBoundHit: value.Bounds, PressureMisses: uint64(value.PressureMisses),
		PressureTransitions: decimal(value.PressureTransitions), PressureFenced: decimal(value.PressureFenced),
		KnownPresent: uint64(a.KnownPresent), KnownAbsent: uint64(a.KnownAbsent), Unknown: uint64(a.Unknown), RetainedTrades: uint64(a.RetainedTrades),
		RetainedQuotes: uint64(a.RetainedQuotes), RetainedFingerprints: uint64(a.RetainedFingerprints),
		Facts: TQFacts{Consumed: decimal(a.Consumed), Applied: decimal(a.Applied), Duplicate: decimal(a.Duplicate), Rejected: decimal(a.Rejected),
			Fenced: decimal(a.Fenced), PressureShed: decimal(a.PressureShed), Integrity: decimal(a.Integrity)},
		Commands: TQCommands{Issued: decimal(c.Issued), Pending: decimal(c.Pending), Acknowledged: decimal(c.Acknowledged), Failed: decimal(c.Failed),
			Fenced: decimal(c.Fenced), ResultFenced: decimal(c.ResultFenced)}}
}

func mapCheckpoint(installed bool, metrics operations.Metrics) Checkpoint {
	a := metrics.Checkpoint
	return Checkpoint{Installed: installed, Submitted: decimal(a.Submitted), InProgress: decimal(a.InProgress), Pending: decimal(a.Pending),
		Completed: decimal(a.Completed), Failed: decimal(a.Failed), Canceled: decimal(a.Canceled), Superseded: decimal(a.Superseded)}
}

func mapOperations(metrics operations.Metrics, operational engine.OperationalView) Operations {
	return Operations{SampleAccountingValid: metrics.AccountingValid, QueueCapacityFrames: nonnegativeInt(metrics.LiveQueue.CapacityFrames),
		QueueCurrentFrames: metrics.QueueCurrentFrames, QueueHighFrames: metrics.QueueHighFrames,
		QueueCurrentBytes: nonnegativeInt(metrics.QueueCurrentBytes), QueueHighBytes: nonnegativeInt(metrics.QueueHighBytes),
		Deliveries: decimal(metrics.Deliveries), ConsumerDeferred: decimal(metrics.ConsumerDeferred),
		MeanProcessingDelayMS: durationMilliseconds(metrics.MeanProcessingDelay), MaxProcessingDelayMS: durationMilliseconds(metrics.MaxProcessingDelay),
		MaxProcessingDelayOneSecondMS: durationMilliseconds(metrics.MaxProcessingDelayOneSecond), HeapAllocBytes: decimal(metrics.HeapAllocBytes),
		HeapInUseBytes: decimal(metrics.HeapInUseBytes), Goroutines: nonnegativeInt(metrics.Goroutines),
		ConnectionRecoveryAttempts: decimal(operational.Connection.RecoveryAttempts)}
}

func mapRow(row engine.ReplayRankingRowView, tq engine.TQSymbolView) (Row, error) {
	if row.Rank == 0 || row.Rank > 20 || row.Symbol == "" || !finite(row.Last) || !finite(row.DayPercent) || row.MarkAge < 0 {
		return Row{}, errors.New("invalid ranking row")
	}
	result := Row{Rank: uint64(row.Rank), Symbol: row.Symbol, LastUSD: row.Last, DayChangeRatio: row.DayPercent,
		MarkAgeMS: durationMilliseconds(row.MarkAge), From4AMChange: mapRatio(row.From4AMPercent), HODDrawdown: mapRatio(row.HODDrawdown),
		DayRangePosition: mapRatio(row.SessionRange), Range30MPosition: mapRatio(row.Rolling30), Range60MPosition: mapRatio(row.Rolling60), Activity: mapRatio(row.Activity),
		TapeRate: TapeRate{Status: "unselected", OneSecond: RateMeasurement{Status: "unselected"}, FiveSecond: RateMeasurement{Status: "unselected"}},
		Spread:   Spread{Status: "unselected"}}
	if tq.Symbol == "" {
		return result, nil
	}
	result.TQMembership = TQMembership{Desired: tq.Desired, ProviderPresent: tq.ProviderPresent, ProviderMembershipUnknown: tq.ProviderMembershipUnknown}
	result.TapeRate = TapeRate{Status: string(tq.Tape.Status), Reason: tq.Tape.Reason, TradeCoverage: tq.TradeCoverage,
		OneSecond:      mapRate(tq.Tape.OneSecondStatus, tq.Tape.OneSecondReason, tq.Tape.OneSecond),
		FiveSecond:     mapRate(tq.Tape.FiveSecondStatus, tq.Tape.FiveSecondReason, tq.Tape.FiveSecond),
		TimestampBasis: tq.Tape.TimestampBasis, LifecycleRecordsObserved: tq.Tape.LifecycleRecordsObserved}
	result.Spread = Spread{Status: string(tq.Spread.Status), Reason: tq.Spread.Reason, QuoteCoverage: tq.QuoteCoverage,
		ValidDurationMS: durationMilliseconds(tq.Spread.ValidDuration), Quality: tq.Spread.Quality}
	if tq.Spread.Status == engine.TQCurrent {
		result.Spread.Cents, result.Spread.BasisPoints = floatPointer(tq.Spread.Cents), floatPointer(tq.Spread.BasisPoints)
	}
	return result, nil
}

func mapRatio(value engine.ReplayFieldView) RatioMeasurement {
	result := RatioMeasurement{Status: value.Status, Reason: value.Reason}
	if value.Status == "current" {
		result.ValueRatio = floatPointer(value.Value)
	}
	return result
}

func mapRate(status engine.TQFieldStatus, reason string, value float64) RateMeasurement {
	result := RateMeasurement{Status: string(status), Reason: reason}
	if status == engine.TQCurrent {
		result.TradesPerSecond = floatPointer(value)
	}
	return result
}

func validateSnapshot(value Snapshot) error {
	p, w, hr, tq, commands, checkpoint := value.Accounting.Population, value.Recovery.Work, value.Recovery.Rows, value.TQ.Facts, value.TQ.Commands, value.Checkpoint
	if value.SchemaVersion != SchemaVersion || !validTimestamp(value.Sample.SampledAt) || !validTimestamp(value.Publication.GeneratedAt) ||
		!validTimestamp(value.Status.CausalTarget) || !validTradingDate(value.Publication.TradingDate) ||
		!validPositiveDecimal(value.Sample.ID) || !validPositiveDecimal(value.Publication.ID) || !validDecimal(value.Publication.LastEngineSequence) ||
		!validDecimal(value.Publication.ConnectionEpoch) || !validDecimal(value.Recovery.Generation) || !validDecimal(value.TQ.PressureTransitions) ||
		!validDecimal(value.TQ.PressureFenced) || !validDecimal(value.TQ.Commands.ResultFenced) || !validDecimal(value.Operations.Deliveries) ||
		!validDecimal(value.Operations.ConsumerDeferred) || !validDecimal(value.Operations.HeapAllocBytes) || !validDecimal(value.Operations.HeapInUseBytes) ||
		!validDecimal(value.Operations.ConnectionRecoveryAttempts) ||
		!oneOf(value.Publication.RunMode, "live", "replay") ||
		!oneOf(value.Publication.Lifecycle, "initializing", "awaiting_session", "awaiting_aggregate_ack", "hydrating", "live", "recovering", "replaying", "suppressed", "ended") ||
		!oneOf(value.Publication.LifecycleReason, "", "binding_before_session", "binding_in_session", "binding_after_session", "session_start_without_aggregate_ack", "session_end", "controlled_stop", "sequence_exhaustion", "clock_regression", "canonical_integrity", "publication_integrity", "accounting_integrity", "closed", "replay_start", "replay_end", "replay_failure", "aggregate_acknowledged", "aggregate_acknowledged_at_session_start", "aggregate_epoch_lost", "ingress_integrity", "hydration_complete", "recovery_exhausted") ||
		!oneOf(value.Publication.Suppression, "", "same_binding_recovery_allowed", "clean_reinitialization_required", "restart_required", "terminal_replay_failure") ||
		!oneOf(value.Status.ReadinessReason, "", "runtime_unavailable", "binding_mismatch", "not_live_mode", "lifecycle_not_ready", "suppressed", "aggregate_unacknowledged", "fence_pending", "ranking_noncurrent", "watermark_missing", "watermark_stale", "accounting_invalid") ||
		!oneOf(value.Ranking.Mode, "unavailable", "qualified_current", "degraded_bootstrap", "stale", "suppressed") ||
		!oneOf(value.Ranking.Reason, "", "no_committed_watermark", "no_trusted_marks", "incomplete_population", "qualification_incomplete", "global_suppression") ||
		!oneOf(value.Recovery.Purpose, "", "fresh_bootstrap", "checkpoint_catchup", "gap_recovery") ||
		!oneOf(value.TQ.PressureMode, "normal", "taq_degraded", "aggregate_only") || value.Status.TQPressureMode != value.TQ.PressureMode || value.Status.TQShed != value.TQ.Shed ||
		value.TQ.AggregateOnly != (value.TQ.PressureMode == "aggregate_only") || value.TQ.Shed != (value.TQ.PressureMode != "normal") ||
		!validOptionalTimestamp(value.Publication.CommittedT) || !validOptionalTimestamp(value.Recovery.Start) || !validOptionalTimestamp(value.Recovery.End) ||
		!validOptionalTimestamp(value.Recovery.SupportedThrough) || !validOptionalTimestamp(value.Publication.HydrationFence.SupportedThrough) ||
		(value.Publication.CommittedT == nil) != (value.Status.WatermarkLagMS == nil) ||
		value.Status.WatermarkLagMS != nil && (*value.Status.WatermarkLagMS < 0 || uint64(*value.Status.WatermarkLagMS) > maximumExactJSONInteger) ||
		!validPosition(value.Publication.AggregateAcknowledged, value.Publication.AggregateAckPosition) ||
		value.Publication.AggregateAcknowledged && value.Publication.AggregateAckPosition.ConnectionEpoch != value.Publication.ConnectionEpoch ||
		!validHydrationFence(value.Publication.HydrationFence) ||
		!sumUint64Equals(p.UniverseTotal, p.ValidPriorClose, p.InvalidOrMissingPriorClose) ||
		!sumUint64Equals(p.ValidPriorClose, p.TrustedRankableMark, p.TrustedBelowPriceMark, p.NoPrintThroughT, p.InvalidMark, p.UnknownDueFailureOrFence) ||
		!sumDecimalEquals(w.Planned, w.Open, w.CompletedValue, w.CompletedEmpty, w.Failed, w.Canceled, w.Fenced) ||
		!sumDecimalEquals(hr.Consumed, hr.Inserted, hr.Duplicate, hr.ConflictOrWithdrawal, hr.Rejected, hr.Fenced, hr.Integrity) ||
		!sumDecimalEquals(tq.Consumed, tq.Applied, tq.Duplicate, tq.Rejected, tq.Fenced, tq.PressureShed, tq.Integrity) ||
		!sumDecimalEquals(commands.Issued, commands.Pending, commands.Acknowledged, commands.Failed, commands.Fenced) ||
		!sumDecimalEquals(checkpoint.Submitted, checkpoint.InProgress, checkpoint.Pending, checkpoint.Completed, checkpoint.Failed, checkpoint.Canceled, checkpoint.Superseded) {
		return errors.New("snapshot accounting identity failed")
	}
	for _, count := range []uint64{p.UniverseTotal, p.ValidPriorClose, p.InvalidOrMissingPriorClose, p.TrustedRankableMark, p.TrustedBelowPriceMark,
		p.NoPrintThroughT, p.InvalidMark, p.UnknownDueFailureOrFence, p.CoveredPopulation, p.UnresolvedPopulation, value.Ranking.TotalPassers,
		value.Ranking.KnownRankableCount, value.Ranking.DayInvalidRankable, value.Ranking.QualifiedDayInvalid,
		value.Accounting.Qualification.NotYetPassed, value.Accounting.Qualification.Provisional, value.Accounting.Qualification.Finalized, value.Accounting.Qualification.Unresolved,
		value.Accounting.Uncertainty.BootstrapOrigin, value.Accounting.Uncertainty.PostBootstrapGap, value.Accounting.Uncertainty.LocalInvalid,
		value.TQ.KnownPresent, value.TQ.KnownAbsent, value.TQ.Unknown, value.Operations.QueueCapacityFrames,
		value.TQ.PressureMisses, value.TQ.RetainedTrades, value.TQ.RetainedQuotes, value.TQ.RetainedFingerprints,
		value.Operations.QueueCurrentFrames, value.Operations.QueueHighFrames, value.Operations.QueueCurrentBytes, value.Operations.QueueHighBytes,
		value.Operations.MeanProcessingDelayMS, value.Operations.MaxProcessingDelayMS, value.Operations.MaxProcessingDelayOneSecondMS, value.Operations.Goroutines} {
		if count > maximumExactJSONInteger {
			return errors.New("snapshot integer exceeds exact JSON bound")
		}
	}
	if len(value.Rows) > 20 || len(value.TQ.DesiredSymbols) > 20 {
		return errors.New("snapshot row bound exceeded")
	}
	seenRows := make(map[string]struct{}, len(value.Rows))
	for index, row := range value.Rows {
		if row.Rank != uint64(index+1) || row.Symbol == "" || !finite(row.LastUSD) || !finite(row.DayChangeRatio) || !measurementsValid(row) {
			return errors.New("invalid snapshot row")
		}
		if _, exists := seenRows[row.Symbol]; exists {
			return errors.New("duplicate snapshot row")
		}
		seenRows[row.Symbol] = struct{}{}
	}
	seenDesired := make(map[string]struct{}, len(value.TQ.DesiredSymbols))
	for _, symbol := range value.TQ.DesiredSymbols {
		if symbol == "" {
			return errors.New("empty desired symbol")
		}
		if _, exists := seenDesired[symbol]; exists {
			return errors.New("duplicate desired symbol")
		}
		seenDesired[symbol] = struct{}{}
	}
	return nil
}

func measurementsValid(row Row) bool {
	for _, field := range []RatioMeasurement{row.From4AMChange, row.HODDrawdown, row.DayRangePosition, row.Range30MPosition, row.Range60MPosition, row.Activity} {
		if !oneOf(field.Status, "warming", "current", "unavailable", "invalid") || !fieldReason(field.Reason) ||
			(field.Status == "current") != (field.ValueRatio != nil) || field.ValueRatio != nil && !finite(*field.ValueRatio) {
			return false
		}
	}
	if !tqStatus(row.TapeRate.Status) || !tqReason(row.TapeRate.Reason) || !tqStatus(row.TapeRate.OneSecond.Status) ||
		!tqReason(row.TapeRate.OneSecond.Reason) || !tqStatus(row.TapeRate.FiveSecond.Status) || !tqReason(row.TapeRate.FiveSecond.Reason) ||
		!oneOf(row.TapeRate.TimestampBasis, "", "none", "participant", "sip_fallback", "mixed") ||
		(row.TapeRate.OneSecond.Status == "current") != (row.TapeRate.OneSecond.TradesPerSecond != nil) ||
		(row.TapeRate.FiveSecond.Status == "current") != (row.TapeRate.FiveSecond.TradesPerSecond != nil) ||
		row.TapeRate.OneSecond.TradesPerSecond != nil && !finite(*row.TapeRate.OneSecond.TradesPerSecond) ||
		row.TapeRate.FiveSecond.TradesPerSecond != nil && !finite(*row.TapeRate.FiveSecond.TradesPerSecond) {
		return false
	}
	if !tqStatus(row.Spread.Status) || !tqReason(row.Spread.Reason) || !oneOf(row.Spread.Quality, "", "reviewed_ordinary", "known_special", "unclassified") ||
		(row.Spread.Status == "current") != (row.Spread.Cents != nil && row.Spread.BasisPoints != nil) ||
		(row.Spread.Cents == nil) != (row.Spread.BasisPoints == nil) || row.Spread.Cents != nil && (!finite(*row.Spread.Cents) || !finite(*row.Spread.BasisPoints)) {
		return false
	}
	return true
}

func fieldReason(value string) bool {
	return oneOf(value, "", "before_first_print", "history_incomplete", "prior_close_unavailable", "no_aggregate_in_target", "rolling_warmup", "reference_warmup", "zero_width", "historical_conflict", "invalid_input", "state_bound_exceeded")
}

func tqStatus(value string) bool {
	return oneOf(value, "unselected", "warming", "current", "stale", "unavailable", "invalid", "pressure_shed")
}

func tqReason(value string) bool {
	return oneOf(value, "", "coverage", "coverage_warming", "five_second_warming", "qualifying_original_prints", "unequal_repeat", "one_sided_quote", "crossed_quote", "stale_quote", "insufficient_coverage", "pressure")
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
