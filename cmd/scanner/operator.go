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
		fmt.Sprint(owner.Hydration.Accounting.Failed), fmt.Sprint(owner.Hydration.Accounting.Fenced), fmt.Sprint(owner.Hydration.Rows.Integrity), fmt.Sprint(owner.Hydration.FenceReconciled)}, "|")
	changed := signature != r.lastSignature
	interval := 10 * time.Second
	warming := owner.Hydration.Accounting.Planned > 0 && !owner.Hydration.FenceReconciled && (status.Lifecycle == "hydrating" || status.Lifecycle == "recovering")
	if warming {
		interval = 5 * time.Second
	}
	if incident := sample.IngressIncident; incident != nil && !r.warnedFailure && (force || status.Lifecycle == "suppressed") {
		r.warnedFailure = true
		if _, err := fmt.Fprintf(r.stderr, "Ingress first cause · %s · %s/%s · epoch %d position %s · invariant %s · lifecycle %s/%s\n",
			incident.Owner, incident.Source, incident.Reason, incident.Epoch, ingressPosition(incident),
			firstFailedIdentityName(incident), incident.Lifecycle, incident.LifecycleReason); err != nil {
			return err
		}
		if incident.Reason == "frame_slot_capacity" || incident.Reason == "frame_byte_capacity" {
			readRate, dispositionRate := ingressRates(incident)
			if _, err := fmt.Fprintf(r.stderr, "Capacity evidence · queued %d/%d · classifying %d · bytes %d/%d · remaining %d · incoming %d · high %d frames/%d bytes · oldest %s · active %s since %s for %s · read %s · disposition %s\n",
				incident.Queue.FramesQueued, incident.Queue.CapacityFrames, incident.Queue.FramesClassifying,
				incident.Queue.QueuedBytes, incident.Queue.CapacityBytes, capacityRemainingBytes(incident), incident.IncomingFrameBytes,
				incident.Queue.HighFramesQueued, incident.Queue.HighQueuedBytes, incident.Queue.OldestFrameAge,
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
	if !force && !changed && !r.lastPrinted.IsZero() && status.SampledAt.Sub(r.lastPrinted) < interval {
		return nil
	}
	if warming {
		a := owner.Hydration.Accounting
		terminal := a.CompletedValue + a.CompletedEmpty + a.Failed + a.Canceled + a.Fenced
		open := uint64(0)
		if a.Planned >= terminal {
			open = a.Planned - terminal
		}
		percent := 0.0
		if a.Planned > 0 {
			percent = 100 * float64(terminal) / float64(a.Planned)
		}
		connection := "aggregate live disconnected"
		if owner.Connection.Active {
			connection = "aggregate live connected"
		}
		if owner.Connection.Acknowledged {
			connection += "/acknowledged"
		}
		fence := "final fence pending"
		if owner.Hydration.FenceReconciled {
			fence = "final fence reconciled"
		}
		_, err := fmt.Fprintf(r.stdout, "Warm-up %s / %s · %.1f%%\nvalues %s · empty %s · open %s · failed %s · canceled %s · fenced %s\n%s · %s\n",
			comma(terminal), comma(a.Planned), percent, comma(a.CompletedValue), comma(a.CompletedEmpty), comma(open), comma(a.Failed), comma(a.Canceled), comma(a.Fenced), connection, fence)
		if err == nil {
			r.remember(signature, status.SampledAt)
		}
		return err
	}
	if status.BackendReady {
		watermark := "unavailable"
		if status.Watermark != nil {
			watermark = status.Watermark.In(r.location).Format("15:04:05 MST")
		}
		connection := "aggregate live disconnected"
		if owner.Connection.Active {
			connection = "aggregate live connected"
		}
		_, err := fmt.Fprintf(r.stdout, "Ready · %s · %d ranked · watermark %s · lag %dms · %s\n", status.RankingMode, sample.Ranked, watermark, status.WatermarkLag.Milliseconds(), connection)
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
	_, err := fmt.Fprintf(r.stderr, "Hydration failure · %s generation %d · failed %s · fenced %s · row integrity %s\n", h.Purpose, h.Generation, comma(h.Accounting.Failed), comma(h.Accounting.Fenced), comma(h.Rows.Integrity))
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
