package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

func TestLiveOperatorWarmupArithmeticAndRateLimit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	at := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	sample := warmupOperatorSample(at)
	if err := renderer.Render(sample, false); err != nil {
		t.Fatal(err)
	}
	want := "Warm-up 3,481 / 5,540 · 62.8%\nvalues 1,064 · empty 2,417 · open 2,059 · failed 0 · canceled 0 · fenced 0\naggregate live connected/acknowledged · final fence pending\n"
	if stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("operator output=%q stderr=%q", stdout.String(), stderr.String())
	}
	sample.Status.SampledAt = at.Add(time.Second)
	if err := renderer.Render(sample, false); err != nil || stdout.String() != want {
		t.Fatalf("rate limit err=%v output=%q", err, stdout.String())
	}
	sample.Status.SampledAt = at.Add(5 * time.Second)
	if err := renderer.Render(sample, false); err != nil || strings.Count(stdout.String(), "Warm-up") != 2 {
		t.Fatalf("five-second output err=%v output=%q", err, stdout.String())
	}
}

func TestLiveOperatorReadyAndIntegrityOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	at := time.Date(2026, 8, 11, 13, 31, 0, 0, time.UTC)
	watermark := at.Add(-4 * time.Second)
	sample := liveOperatorSample{Status: operations.Status{BackendReady: true, Lifecycle: "live", RankingMode: "qualified_current", SampledAt: at, Watermark: &watermark, WatermarkLag: 125 * time.Millisecond}, Ranked: 17}
	sample.Metrics.Engine.Connection.Active = true
	if err := renderer.Render(sample, false); err != nil || stdout.String() != "Ready · qualified_current · 17 ranked · watermark 09:30:56 EDT · lag 125ms · aggregate live connected\n" {
		t.Fatalf("ready err=%v output=%q", err, stdout.String())
	}
	failure := &engine.EvaluatorIntegrityView{Category: engine.EvaluatorSupportContradiction, EngineSequence: 81, CandidateTime: at.Add(-time.Minute)}
	sample.Status = operations.Status{Lifecycle: "suppressed", Reason: operations.ReasonSuppressed, RankingMode: "suppressed", SampledAt: at.Add(time.Second), IntegrityFailure: failure}
	sample.Metrics.Engine.LifecycleReason = "accounting_integrity"
	sample.Metrics.Engine.Suppression = engine.SuppressionRestartRequired
	if err := renderer.Render(sample, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Integrity failure · support_contradiction · accounting_integrity · engine sequence 81 · candidate/fence 2026-08-11T13:30:00Z · restart required") || !strings.Contains(stdout.String(), "Suppressed · accounting_integrity · restart_required") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestInitialOperatorFailureRunsJoinedShutdown(t *testing.T) {
	renderer := newOperatorRenderer(failingWriter{}, &bytes.Buffer{})
	shutdowns := 0
	err := renderInitialOperator(renderer, warmupOperatorSample(time.Now().UTC()), func() error { shutdowns++; return errors.New("joined shutdown") })
	if err == nil || !strings.Contains(err.Error(), "write operational status") || !strings.Contains(err.Error(), "joined shutdown") || shutdowns != 1 {
		t.Fatalf("err=%v shutdowns=%d", err, shutdowns)
	}
}

func warmupOperatorSample(at time.Time) liveOperatorSample {
	result := liveOperatorSample{Status: operations.Status{Lifecycle: "hydrating", Reason: operations.ReasonFencePending, SampledAt: at}}
	result.Metrics.Engine.Connection.Active = true
	result.Metrics.Engine.Connection.Acknowledged = true
	result.Metrics.Engine.Hydration.Purpose = engine.HydrationFreshBootstrap
	result.Metrics.Engine.Hydration.Generation = 1
	result.Metrics.Engine.Hydration.Accounting = engine.HydrationAccounting{Planned: 5540, CompletedValue: 1064, CompletedEmpty: 2417}
	return result
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("closed output") }
