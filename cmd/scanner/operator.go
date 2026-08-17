package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

type liveOperatorSample struct {
	Status          operations.Status
	Metrics         operations.Metrics
	IngressIncident *operations.IngressIncident
	Evaluation      engine.ReplayEvaluationView
	Ranked          int
}

type operatorRenderer struct {
	stdout, stderr                        io.Writer
	location                              *time.Location
	lastSignature                         string
	lastPrinted                           time.Time
	warnedFailure                         bool
	warmupKey                             string
	hydrationKey                          string
	lastFailed, lastFenced, lastIntegrity uint64
}

func newOperatorRenderer(stdout, stderr io.Writer) *operatorRenderer {
	location, _ := time.LoadLocation("America/New_York")
	return &operatorRenderer{stdout: stdout, stderr: stderr, location: location}
}

func (r *operatorRenderer) Render(sample liveOperatorSample, force bool) error {
	status, owner := sample.Status, sample.Metrics.Engine
	signature := strings.Join([]string{status.Lifecycle, string(status.Reason), status.RankingMode, string(owner.Suppression), owner.LifecycleReason,
		fmt.Sprint(owner.Connection.Active), fmt.Sprint(owner.Connection.Acknowledged), string(owner.Hydration.Purpose), fmt.Sprint(owner.Hydration.Generation),
		fmt.Sprint(owner.Hydration.Active), fmt.Sprint(owner.Hydration.Accounting.Failed), fmt.Sprint(owner.Hydration.Accounting.Canceled),
		fmt.Sprint(owner.Hydration.Accounting.Fenced), fmt.Sprint(owner.Hydration.Rows.Integrity), fmt.Sprint(owner.Hydration.FenceReconciled)}, "|")
	changed := signature != r.lastSignature
	interval := 10 * time.Second
	warming := owner.Hydration.Active && owner.Hydration.Accounting.Planned > 0 && !owner.Hydration.FenceReconciled && (status.Lifecycle == "hydrating" || status.Lifecycle == "recovering")
	if warming {
		interval = 5 * time.Second
	}
	if incident := sample.IngressIncident; incident != nil && !r.warnedFailure {
		r.warnedFailure = true
		if _, err := fmt.Fprintf(r.stderr, "Ingress first cause · %s · %s/%s · at %s · epoch %d position %s · invariant %s · lifecycle %s/%s\n",
			incident.Owner, incident.Source, incident.Reason, incident.CapturedAt.UTC().Format(time.RFC3339Nano), incident.Epoch, ingressPosition(incident),
			firstFailedIdentityName(incident), incident.Lifecycle, incident.LifecycleReason); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(r.stderr, "Ingress evidence · generation %d active %t purpose %s · work %s/%s open %s canceled %s failed %s fenced %s · queue %d/%d high %d · bytes %d/%d high %d · max delay %s recent %s · heap alloc %s in-use %s · goroutines %d\n",
			incident.Hydration.Generation, incident.Hydration.Active, incident.Hydration.Purpose,
			comma(incident.Hydration.Accounting.CompletedValue+incident.Hydration.Accounting.CompletedEmpty), comma(incident.Hydration.Accounting.Planned),
			comma(incident.Hydration.Accounting.Open), comma(incident.Hydration.Accounting.Canceled), comma(incident.Hydration.Accounting.Failed), comma(incident.Hydration.Accounting.Fenced),
			incident.Queue.FramesQueued, incident.Queue.CapacityFrames, incident.Queue.HighFramesQueued,
			incident.Queue.QueuedBytes, incident.Queue.CapacityBytes, incident.Queue.HighQueuedBytes,
			incident.MaxProcessingDelay, incident.MaxProcessingDelayOneSecond, comma(incident.HeapAllocBytes), comma(incident.HeapInUseBytes), incident.Goroutines); err != nil {
			return err
		}
		if incident.Reason == "frame_slot_capacity" || incident.Reason == "frame_byte_capacity" {
			readRate, dispositionRate := ingressRates(incident)
			if _, err := fmt.Fprintf(r.stderr, "Capacity evidence · queued %d/%d · classifying %d · bytes %d/%d · remaining %d · incoming %d · high %d frames/%d bytes · oldest waiting %s · active %s since %s for %s · read %s · disposition %s\n",
				incident.Queue.FramesQueued, incident.Queue.CapacityFrames, incident.Queue.FramesClassifying,
				incident.Queue.QueuedBytes, incident.Queue.CapacityBytes, capacityRemainingBytes(incident), incident.IncomingFrameBytes,
				incident.Queue.HighFramesQueued, incident.Queue.HighQueuedBytes, incident.Queue.OldestWaitingFrameAge,
				capacityActiveKind(incident), capacityActiveStartedAt(incident), incident.ActiveDeliveryAgeAtCause, ingressRate(readRate), ingressRate(dispositionRate)); err != nil {
				return err
			}
		}
	}
	if status.Lifecycle == "suppressed" {
		if !r.warnedFailure {
			r.warnedFailure = true
			if failure := status.IntegrityFailure; failure != nil {
				candidate := failure.CandidateTime
				if candidate.IsZero() {
					candidate = failure.ExpectedTime
				}
				if _, err := fmt.Fprintf(r.stderr, "Integrity failure · %s · %s · engine sequence %d · candidate/fence %s · restart required\n", failure.Category, owner.LifecycleReason, failure.EngineSequence, candidate.Format(time.RFC3339)); err != nil {
					return err
				}
			} else if _, err := fmt.Fprintf(r.stderr, "Scanner suppressed · %s · %s\n", owner.LifecycleReason, owner.Suppression); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(r.stdout, "Suppressed · %s · %s\n", owner.LifecycleReason, owner.Suppression)
		if err == nil {
			evaluation := sample.Evaluation
			if sample.IngressIncident != nil && sample.IngressIncident.LastCoherentProjection != nil {
				evaluation = sample.IngressIncident.LastCoherentProjection.Evaluation
			}
			err = renderPopulationTransitionDiagnostic(r.stdout, evaluation)
		}
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	if err := r.warnHydrationFailure(sample); err != nil {
		return err
	}
	if warming {
		if !force && !changed && !r.lastPrinted.IsZero() && status.SampledAt.Sub(r.lastPrinted) < interval {
			return nil
		}
	} else if status.BackendReady || status.Reason == operations.ReasonWatermarkStale {
		// Ready and watermark-stale are state notifications, not telemetry
		// heartbeats. Emit them when the operational state changes; the
		// dashboard/API remain the source for continuously changing values.
		if !force && !changed {
			return nil
		}
	} else if !force && !changed && !r.lastPrinted.IsZero() && status.SampledAt.Sub(r.lastPrinted) < interval {
		return nil
	}
	if status.Lifecycle == "recovering" && !owner.Hydration.Active {
		phase := "reconnecting"
		if owner.Connection.Active {
			phase = "resubscribing"
		}
		if owner.Connection.Acknowledged {
			phase = "preparing recovery"
		}
		_, err := fmt.Fprintf(r.stdout, "Recovery: %s\n", phase)
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	if warming {
		if err := r.renderHydrationStart(owner.Hydration, status.Lifecycle); err != nil {
			return err
		}
		a := owner.Hydration.Accounting
		terminal := a.CompletedValue + a.CompletedEmpty + a.Failed + a.Canceled + a.Fenced
		percent := 0.0
		if a.Planned > 0 {
			percent = 100 * float64(terminal) / float64(a.Planned)
		}
		label := "Warm-up"
		if status.Lifecycle == "recovering" {
			label = "Recovery"
		}
		_, err := fmt.Fprintln(r.stdout, hydrationProgress(label, terminal, a.Planned, percent, a))
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	if status.BackendReady {
		_, err := fmt.Fprintf(r.stdout, "Ready · %s · %d ranked · watermark %s · lag %dms\n", status.RankingMode, sample.Ranked, r.formatWatermark(status.Watermark), status.WatermarkLag.Milliseconds())
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	if status.Lifecycle == "awaiting_aggregate_ack" || status.Lifecycle == "hydrating" {
		r.remember(signature, status.SampledAt)
		return nil
	}
	if status.Reason == operations.ReasonWatermarkStale {
		_, err := fmt.Fprintf(r.stdout, "Scanner degraded · watermark stale · watermark %s · lag %dms\n", r.formatWatermark(status.Watermark), status.WatermarkLag.Milliseconds())
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	_, err := fmt.Fprintf(r.stdout, "Scanner %s · %s\n", status.Lifecycle, status.Reason)
	if err == nil {
		err = renderPopulationTransitionDiagnostic(r.stdout, sample.Evaluation)
	}
	if err == nil {
		r.remember(signature, status.SampledAt)
	}
	return err
}

func (r *operatorRenderer) formatWatermark(watermark *time.Time) string {
	if watermark == nil {
		return "unavailable"
	}
	return watermark.In(r.location).Format("15:04:05 MST")
}

func (r *operatorRenderer) renderHydrationStart(h engine.OperationalHydration, lifecycle string) error {
	key := fmt.Sprintf("%s/%d", h.Purpose, h.Generation)
	if key == r.warmupKey || h.End.IsZero() {
		return nil
	}
	label := "Warm-up"
	if lifecycle == "recovering" {
		label = "Recovery"
	}
	through := h.End.Add(-time.Second).In(r.location).Format("15:04:05 MST")
	if _, err := fmt.Fprintf(r.stdout, "%s: fetching historical one-second aggregates through %s for %s stocks\n", label, through, comma(h.Accounting.Planned)); err != nil {
		return err
	}
	r.warmupKey = key
	return nil
}

func hydrationProgress(label string, terminal, planned uint64, percent float64, accounting engine.HydrationAccounting) string {
	message := fmt.Sprintf("%s: %s / %s stocks (%.1f%%)", label, comma(terminal), comma(planned), percent)
	status := make([]string, 0, 3)
	if accounting.Failed > 0 {
		status = append(status, fmt.Sprintf("failed %s", comma(accounting.Failed)))
	}
	if accounting.Canceled > 0 {
		status = append(status, fmt.Sprintf("canceled %s", comma(accounting.Canceled)))
	}
	if accounting.Fenced > 0 {
		status = append(status, fmt.Sprintf("fenced %s", comma(accounting.Fenced)))
	}
	if len(status) > 0 {
		message += " · " + strings.Join(status, " · ")
	}
	return message
}

func ingressRate(value float64) string {
	if value < 0 {
		return "unavailable"
	}
	return fmt.Sprintf("%.1f/s", value)
}

func renderPopulationTransitionDiagnostic(output io.Writer, evaluation engine.ReplayEvaluationView) error {
	d, p := evaluation.PopulationTransition, evaluation.Population
	if d.BootstrapUnknown == 0 && p.UnresolvedPopulation == 0 {
		return nil
	}
	_, err := fmt.Fprintf(output, "Population transition · bootstrap unknown %s · trusted later live %s · no later mark %s · mark not live %s · no older conflict %s · conflict at/after mark %s · invalid at/after mark %s · incomplete post-mark coverage %s · unresolved population %s · local invalid %s\n",
		comma(d.BootstrapUnknown), comma(d.TrustedByLaterLiveMark), comma(d.NoLaterEligibleMark), comma(d.LatestMarkNotLiveAuthority),
		comma(d.NoStrictlyOlderLocalizedConflict), comma(d.ConflictAtOrAfterMark), comma(d.InvalidAtOrAfterMark),
		comma(d.IncompletePostMarkCoverage), comma(p.UnresolvedPopulation), comma(evaluation.Uncertainty.LocalInvalid))
	return err
}

func capacityRemainingBytes(incident *operations.IngressIncident) int {
	if incident == nil || incident.Queue.CapacityBytes <= incident.Queue.QueuedBytes {
		return 0
	}
	return incident.Queue.CapacityBytes - incident.Queue.QueuedBytes
}

func ingressRates(incident *operations.IngressIncident) (float64, float64) {
	if incident == nil || incident.HistoryCount < 2 {
		return -1, -1
	}
	for lastIndex := incident.HistoryCount - 1; lastIndex > 0; lastIndex-- {
		last := incident.History[lastIndex]
		for firstIndex := lastIndex - 1; firstIndex >= 0; firstIndex-- {
			first := incident.History[firstIndex]
			if !incident.CapturedAt.IsZero() && (first.CapturedAt.After(incident.CapturedAt) || last.CapturedAt.After(incident.CapturedAt)) {
				continue
			}
			seconds := last.CapturedAt.Sub(first.CapturedAt).Seconds()
			if seconds <= 0 || last.FramesRead < first.FramesRead || last.FramesDispositioned < first.FramesDispositioned {
				continue
			}
			readDelta, dispositionDelta := last.FramesRead-first.FramesRead, last.FramesDispositioned-first.FramesDispositioned
			if readDelta == 0 && dispositionDelta == 0 {
				continue
			}
			return float64(readDelta) / seconds, float64(dispositionDelta) / seconds
		}
	}
	return -1, -1
}

func capacityActiveKind(incident *operations.IngressIncident) string {
	if incident == nil || incident.ActiveDeliveryKind == "" {
		return "none"
	}
	return string(incident.ActiveDeliveryKind)
}

func capacityActiveStartedAt(incident *operations.IngressIncident) string {
	if incident == nil || incident.ActiveDeliveryStartedAt.IsZero() {
		return "not_applicable"
	}
	return incident.ActiveDeliveryStartedAt.UTC().Format(time.RFC3339Nano)
}

func ingressPosition(incident *operations.IngressIncident) string {
	if incident == nil || !incident.PositionApplicable {
		return "not_applicable"
	}
	if !incident.ArrayIndexApplicable {
		return fmt.Sprintf("(%d,%d,not_applicable)", incident.Position.ConnectionEpoch, incident.Position.FrameSequence)
	}
	return fmt.Sprintf("(%d,%d,%d)", incident.Position.ConnectionEpoch, incident.Position.FrameSequence, incident.Position.ArrayIndex)
}

func firstFailedIdentityName(incident *operations.IngressIncident) string {
	if incident == nil {
		return "not_applicable"
	}
	for index := 0; index < incident.IdentityCount; index++ {
		if !incident.Identities[index].Reconciled {
			return incident.Identities[index].Name
		}
	}
	return "not_applicable"
}

func (r *operatorRenderer) warnHydrationFailure(sample liveOperatorSample) error {
	h := sample.Metrics.Engine.Hydration
	key := fmt.Sprintf("%s/%d", h.Purpose, h.Generation)
	if key != r.hydrationKey {
		r.hydrationKey, r.lastFailed, r.lastFenced, r.lastIntegrity = key, 0, 0, 0
	}
	if h.Accounting.Failed <= r.lastFailed && h.Accounting.Fenced <= r.lastFenced && h.Rows.Integrity <= r.lastIntegrity {
		return nil
	}
	status := make([]string, 0, 3)
	if h.Accounting.Failed > 0 {
		status = append(status, fmt.Sprintf("failed %s", comma(h.Accounting.Failed)))
	}
	if h.Accounting.Fenced > 0 {
		status = append(status, fmt.Sprintf("fenced %s", comma(h.Accounting.Fenced)))
	}
	if h.Rows.Integrity > 0 {
		status = append(status, fmt.Sprintf("row integrity %s", comma(h.Rows.Integrity)))
	}
	_, err := fmt.Fprintf(r.stderr, "Hydration failure: %s generation %d · %s\n", h.Purpose, h.Generation, strings.Join(status, " · "))
	if err == nil {
		r.lastFailed, r.lastFenced, r.lastIntegrity = h.Accounting.Failed, h.Accounting.Fenced, h.Rows.Integrity
	}
	return err
}

func (r *operatorRenderer) remember(signature string, at time.Time) {
	r.lastSignature, r.lastPrinted = signature, at
}
func comma(value uint64) string {
	raw := fmt.Sprint(value)
	for index := len(raw) - 3; index > 0; index -= 3 {
		raw = raw[:index] + "," + raw[index:]
	}
	return raw
}
