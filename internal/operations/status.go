package operations

import (
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type ReadinessReason string

const (
	ReasonNone                    ReadinessReason = ""
	ReasonRuntimeUnavailable      ReadinessReason = "runtime_unavailable"
	ReasonBindingMismatch         ReadinessReason = "binding_mismatch"
	ReasonNotLiveMode             ReadinessReason = "not_live_mode"
	ReasonLifecycle               ReadinessReason = "lifecycle_not_ready"
	ReasonSuppressed              ReadinessReason = "suppressed"
	ReasonAggregateUnacknowledged ReadinessReason = "aggregate_unacknowledged"
	ReasonFencePending            ReadinessReason = "fence_pending"
	ReasonRankingNoncurrent       ReadinessReason = "ranking_noncurrent"
	ReasonWatermarkMissing        ReadinessReason = "watermark_missing"
	ReasonWatermarkStale          ReadinessReason = "watermark_stale"
	ReasonAccounting              ReadinessReason = "accounting_invalid"
)

type Status struct {
	ProcessLive, BackendReady, RankingCurrent bool
	Reason                                    ReadinessReason
	Lifecycle, RankingMode                    string
	PublicationID                             uint64
	SampledAt                                 time.Time
	Watermark, CausalTarget                   *time.Time
	WatermarkLag                              time.Duration
	QueueCapacity, QueueOccupancy             int
	TQAvailable                               bool
	AccountingValid                           bool
	IntegrityFailure                          *engine.EvaluatorIntegrityView
}

func deriveStatus(processLive bool, binding reference.Binding, config Config, now time.Time, view engine.OperationalView) Status {
	result := Status{ProcessLive: processLive, Lifecycle: view.Lifecycle, RankingMode: view.RankingMode, PublicationID: view.PublicationID, SampledAt: now,
		Watermark: cloneTime(view.Watermark), QueueCapacity: view.QueueCapacity, QueueOccupancy: view.QueueOccupancy,
		RankingCurrent: view.CurrentMarketClaim, TQAvailable: false}
	result.IntegrityFailure = cloneIntegrityFailure(view.IntegrityFailure)
	target := readinessCausalTarget(binding, config, now)
	result.CausalTarget = &target
	result.AccountingValid = operationalAccountingValid(view)
	switch {
	case !processLive:
		result.Reason = ReasonRuntimeUnavailable
	case view.Suppression != "":
		result.Reason = ReasonSuppressed
	case view.BindingIdentity != binding.Identity():
		result.Reason = ReasonBindingMismatch
	case view.RunMode != "live":
		result.Reason = ReasonNotLiveMode
	case view.Lifecycle != "live" && view.Lifecycle != "hydrating":
		result.Reason = ReasonLifecycle
	case !view.Connection.Active || !view.Connection.Acknowledged:
		result.Reason = ReasonAggregateUnacknowledged
	case !view.Hydration.FenceReconciled:
		result.Reason = ReasonFencePending
	case !view.CurrentMarketClaim:
		result.Reason = ReasonRankingNoncurrent
	case view.Watermark == nil:
		result.Reason = ReasonWatermarkMissing
	case target.Sub(*view.Watermark) > config.ReadinessTolerance:
		result.Reason = ReasonWatermarkStale
	case !result.AccountingValid:
		result.Reason = ReasonAccounting
	default:
		result.BackendReady = true
		result.Reason = ReasonNone
	}
	if view.Watermark != nil && target.After(*view.Watermark) {
		result.WatermarkLag = target.Sub(*view.Watermark)
	}
	return result
}

func readinessCausalTarget(binding reference.Binding, config Config, now time.Time) time.Time {
	target := now.UTC().Truncate(time.Second).Add(-config.EvaluationDelay)
	if target.Before(binding.SessionStart()) {
		target = binding.SessionStart()
	}
	if target.After(binding.SessionEnd()) {
		target = binding.SessionEnd()
	}
	return target
}

func operationalAccountingValid(view engine.OperationalView) bool {
	h := view.Hydration.Accounting
	hr := view.Hydration.Rows
	return view.Admissions.Reconciles(view.QueueOccupancy) && view.Transitions.Reconciles() && view.Aggregates.Reconciles() && view.Connection.Reconciles() &&
		h.Planned == h.Open+h.CompletedValue+h.CompletedEmpty+h.Failed+h.Canceled+h.Fenced &&
		hr.Consumed == hr.Inserted+hr.Duplicate+hr.ConflictOrWithdrawal+hr.Rejected+hr.Fenced+hr.Integrity
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
