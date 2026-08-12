package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

type liveOperatorSample struct {
	Status  operations.Status
	Metrics operations.Metrics
	Ranked  int
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
		r.remember(signature, status.SampledAt)
	}
	return err
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
