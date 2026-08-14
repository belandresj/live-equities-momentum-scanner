package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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
		t.Fatalf("operator output = %q stderr=%q", stdout.String(), stderr.String())
	}
	sample.Status.SampledAt = at.Add(time.Second)
	if err := renderer.Render(sample, false); err != nil || stdout.String() != want {
		t.Fatalf("one-second sample changed output: err=%v output=%q", err, stdout.String())
	}
	sample.Status.SampledAt = at.Add(5 * time.Second)
	if err := renderer.Render(sample, false); err != nil || strings.Count(stdout.String(), "Warm-up") != 2 {
		t.Fatalf("five-second warm-up output: err=%v output=%q", err, stdout.String())
	}
	sample.Status.SampledAt = at.Add(6 * time.Second)
	sample.Metrics.Engine.Hydration.FenceReconciled = true
	if err := renderer.Render(sample, false); err != nil || !strings.Contains(stdout.String(), "Scanner hydrating") {
		t.Fatalf("fence change was not immediate: err=%v output=%q", err, stdout.String())
	}
}

func TestLiveOperatorReadyAndIntegrityOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	at := time.Date(2026, 8, 11, 13, 31, 0, 0, time.UTC)
	watermark := at.Add(-4 * time.Second)
	ready := liveOperatorSample{Status: operations.Status{BackendReady: true, Lifecycle: "live", RankingMode: "qualified_current", SampledAt: at, Watermark: &watermark, WatermarkLag: 125 * time.Millisecond}, Ranked: 17}
	ready.Metrics.Engine.Connection.Active = true
	if err := renderer.Render(ready, false); err != nil || stdout.String() != "Ready · qualified_current · 17 ranked · watermark 09:30:56 EDT · lag 125ms · aggregate live connected\n" {
		t.Fatalf("ready output err=%v output=%q", err, stdout.String())
	}
	ready.Status.SampledAt = at.Add(time.Second)
	if err := renderer.Render(ready, false); err != nil || strings.Count(stdout.String(), "Ready") != 1 {
		t.Fatalf("ready output ignored ten-second rate limit: err=%v output=%q", err, stdout.String())
	}
	ready.Status.SampledAt = at.Add(10 * time.Second)
	if err := renderer.Render(ready, false); err != nil || strings.Count(stdout.String(), "Ready") != 2 {
		t.Fatalf("ready output did not print at ten seconds: err=%v output=%q", err, stdout.String())
	}
	candidate := at.Add(-time.Minute)
	failure := &engine.EvaluatorIntegrityView{Category: engine.EvaluatorSupportContradiction, EngineSequence: 81, CandidateTime: candidate}
	ready.Status = operations.Status{Lifecycle: "suppressed", Reason: operations.ReasonSuppressed, RankingMode: "suppressed", SampledAt: at.Add(11 * time.Second), IntegrityFailure: failure}
	ready.Metrics.Engine.LifecycleReason = "accounting_integrity"
	ready.Metrics.Engine.Suppression = engine.SuppressionRestartRequired
	if err := renderer.Render(ready, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Integrity failure · support_contradiction · accounting_integrity · engine sequence 81 · candidate/fence 2026-08-11T13:30:00Z · restart required") ||
		!strings.Contains(stdout.String(), "Suppressed · accounting_integrity · restart_required") {
		t.Fatalf("suppression output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLiveOperatorHydrationFailureIsImmediateAndChangeAware(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	at := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	sample := warmupOperatorSample(at)
	if err := renderer.Render(sample, false); err != nil {
		t.Fatal(err)
	}
	sample.Status.SampledAt = at.Add(time.Second)
	sample.Metrics.Engine.Hydration.Accounting.Failed = 1
	sample.Metrics.Engine.Hydration.Accounting.Canceled = 2
	sample.Metrics.Engine.Hydration.Accounting.Fenced = 3
	sample.Metrics.Engine.Hydration.Rows.Integrity = 4
	if err := renderer.Render(sample, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Hydration failure · fresh_bootstrap generation 1 · failed 1 (+1) · fenced 3 (+3) · row integrity 4 (+4)") || strings.Count(stdout.String(), "Warm-up") != 2 {
		t.Fatalf("hydration warning stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	sample.Status.SampledAt = at.Add(2 * time.Second)
	if err := renderer.Render(sample, false); err != nil || strings.Count(stderr.String(), "Hydration failure") != 1 || strings.Count(stdout.String(), "Warm-up") != 2 {
		t.Fatalf("unchanged failure repeated stdout=%q stderr=%q err=%v", stdout.String(), stderr.String(), err)
	}
}

func TestInitialOperatorFailureRunsJoinedShutdown(t *testing.T) {
	renderer := newOperatorRenderer(failingWriter{}, &bytes.Buffer{})
	shutdowns := 0
	err := renderInitialOperator(renderer, warmupOperatorSample(time.Now().UTC()), func() error { shutdowns++; return errors.New("joined shutdown") })
	if err == nil || !strings.Contains(err.Error(), "write operational status") || !strings.Contains(err.Error(), "joined shutdown") || shutdowns != 1 {
		t.Fatalf("initial failure err=%v shutdowns=%d", err, shutdowns)
	}
}

func TestPrivateDiagnosticLogSafetyAndOneSamplePerWrite(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "scanner.ndjson")
	var warnings bytes.Buffer
	log, err := openDiagnosticLog(path, &warnings)
	if err != nil {
		t.Fatal(err)
	}
	log.Write(operations.Status{Lifecycle: "suppressed", IntegrityFailure: &engine.EvaluatorIntegrityView{
		Category: engine.EvaluatorSupportContradiction, EngineSequence: 7, FirstSymbol: "AAA", FirstField: "support", FirstReason: "no_print_with_invalid_evidence",
	}}, operations.Metrics{Deliveries: 3})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 || bytes.Count(content, []byte{'\n'}) != 1 || !bytes.Contains(content, []byte(`"Deliveries":3`)) ||
		!bytes.Contains(content, []byte(`"FirstSymbol":"AAA"`)) || !bytes.Contains(content, []byte(`"FirstReason":"no_print_with_invalid_evidence"`)) || warnings.Len() != 0 {
		t.Fatalf("diagnostic mode=%o content=%q warnings=%q", info.Mode().Perm(), content, warnings.String())
	}
	if _, err := openDiagnosticLog("relative.ndjson", &warnings); err == nil {
		t.Fatal("relative diagnostic path accepted")
	}
	unsafe := filepath.Join(directory, "unsafe.ndjson")
	if err := os.WriteFile(unsafe, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := openDiagnosticLog(unsafe, &warnings); err == nil {
		t.Fatal("group/other-readable diagnostic file accepted")
	}
	symlink := filepath.Join(directory, "link.ndjson")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := openDiagnosticLog(symlink, &warnings); err == nil {
		t.Fatal("symlink diagnostic target accepted")
	}
	if _, err := openDiagnosticLog(directory, &warnings); err == nil {
		t.Fatal("non-regular diagnostic target accepted")
	}
}

func TestPrivateDiagnosticWriteFailureWarnsOnceAndDisables(t *testing.T) {
	var warnings bytes.Buffer
	writer := &failingDiagnosticWriter{}
	log := newDiagnosticLog(writer, &warnings)
	log.Write(operations.Status{}, operations.Metrics{})
	log.Write(operations.Status{}, operations.Metrics{})
	if strings.Count(warnings.String(), "diagnostic log write failed") != 1 || writer.writes != 1 || writer.closes != 1 {
		t.Fatalf("failure containment warnings=%q writes=%d closes=%d", warnings.String(), writer.writes, writer.closes)
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

type failingDiagnosticWriter struct{ writes, closes int }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("closed output") }

func (w *failingDiagnosticWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("disk full")
}

func (w *failingDiagnosticWriter) Close() error {
	w.closes++
	return nil
}
