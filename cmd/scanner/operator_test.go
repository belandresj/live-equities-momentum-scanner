package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
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

func TestLiveOperatorRendersPopulationTransitionDiagnostic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	sample := liveOperatorSample{
		Status: operations.Status{Lifecycle: "live", Reason: operations.ReasonRankingNoncurrent, RankingMode: "degraded_bootstrap", SampledAt: time.Now().UTC()},
		Evaluation: engine.ReplayEvaluationView{
			Population:  engine.ReplayPopulationView{UnresolvedPopulation: 4},
			Uncertainty: engine.ReplayUncertaintyView{LocalInvalid: 1},
			PopulationTransition: engine.ReplayPopulationTransitionDiagnosticView{BootstrapUnknown: 5, TrustedByLaterLiveMark: 2, NoLaterEligibleMark: 1,
				LatestMarkNotLiveAuthority: 1, IncompletePostMarkCoverage: 1},
		},
	}
	if err := renderer.Render(sample, true); err != nil {
		t.Fatal(err)
	}
	want := "Population transition · bootstrap unknown 5 · trusted later live 2 · no later mark 1 · mark not live 1 · no older conflict 0 · conflict at/after mark 0 · invalid at/after mark 0 · incomplete post-mark coverage 1 · unresolved population 4 · local invalid 1"
	if !strings.Contains(stdout.String(), want) || stderr.Len() != 0 {
		t.Fatalf("operator diagnostic output=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLiveOperatorRendersPopulationTransitionDiagnosticOnSuppression(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	sample := liveOperatorSample{
		Status: operations.Status{Lifecycle: "suppressed", Reason: operations.ReasonSuppressed, RankingMode: "suppressed", SampledAt: time.Now().UTC()},
		Evaluation: engine.ReplayEvaluationView{
			Population:           engine.ReplayPopulationView{UnresolvedPopulation: 1},
			Uncertainty:          engine.ReplayUncertaintyView{LocalInvalid: 1},
			PopulationTransition: engine.ReplayPopulationTransitionDiagnosticView{BootstrapUnknown: 1, NoLaterEligibleMark: 1},
		},
	}
	sample.Metrics.Engine.LifecycleReason = "ingress_integrity"
	sample.Metrics.Engine.Suppression = engine.SuppressionSameBindingRecoveryAllowed
	if err := renderer.Render(sample, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Suppressed · ingress_integrity · same_binding_recovery_allowed") ||
		!strings.Contains(stdout.String(), "Population transition · bootstrap unknown 1 · trusted later live 0 · no later mark 1") {
		t.Fatalf("suppressed final sample lost population diagnostic: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLiveOperatorRendersTypedIngressFirstCause(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	sample := liveOperatorSample{
		Status: operations.Status{Lifecycle: "suppressed", Reason: operations.ReasonSuppressed, RankingMode: "suppressed", SampledAt: time.Now().UTC()},
		IngressIncident: &operations.IngressIncident{Owner: operations.IngressOwnerAdapterTerminal, Source: "protocol", Reason: "ingress_ambiguity", Epoch: 3,
			Position: engine.LivePosition{ConnectionEpoch: 3, FrameSequence: 41, ArrayIndex: 2}, PositionApplicable: true, ArrayIndexApplicable: true,
			Lifecycle: "suppressed", LifecycleReason: "ingress_integrity"},
	}
	sample.Metrics.Engine.LifecycleReason = "ingress_integrity"
	sample.Metrics.Engine.Suppression = engine.SuppressionSameBindingRecoveryAllowed
	if err := renderer.Render(sample, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Ingress first cause · adapter_terminal · protocol/ingress_ambiguity · epoch 3 position (3,41,2) · invariant not_applicable · lifecycle suppressed/ingress_integrity") ||
		!strings.Contains(stdout.String(), "Suppressed · ingress_integrity · same_binding_recovery_allowed") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	renderer = newOperatorRenderer(&stdout, &stderr)
	sample.Status.Lifecycle = "awaiting_aggregate_ack"
	sample.IngressIncident.Source = "reader"
	sample.IngressIncident.Reason = "read_failed"
	sample.IngressIncident.Lifecycle = "awaiting_aggregate_ack"
	sample.IngressIncident.LifecycleReason = "aggregate_epoch_lost"
	if err := renderer.Render(sample, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "reader/read_failed") {
		t.Fatalf("forced terminal output lost non-suppression cause: %q", stderr.String())
	}
}

func TestLiveOperatorRendersDecisiveCapacityOperands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	renderer := newOperatorRenderer(&stdout, &stderr)
	at := time.Now().UTC()
	incident := &operations.IngressIncident{Owner: operations.IngressOwnerAdapterTerminal, Source: "protocol", Reason: "frame_slot_capacity", Epoch: 1,
		Position: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 42220}, PositionApplicable: true,
		Lifecycle: "suppressed", LifecycleReason: "ingress_integrity", IncomingFrameBytes: 16384,
		ActiveDeliveryKind: massive.DeliveryAggregateIngressFence, ActiveDeliveryStartedAt: at.Add(-1250 * time.Millisecond), ActiveDeliveryAgeAtCause: 1250 * time.Millisecond, HistoryCount: 3, CapturedAt: at}
	incident.Queue = massive.LiveQueueAccounting{FramesQueued: 1024, FramesClassifying: 1, QueuedBytes: 12 << 20, CapacityFrames: 1024, CapacityBytes: 128 << 20,
		HighFramesQueued: 1024, HighQueuedBytes: 12 << 20, OldestWaitingFrameAge: 2 * time.Second}
	incident.History[0] = operations.IngressDiagnosticSample{CapturedAt: at.Add(-time.Second), FramesRead: 100, FramesDispositioned: 90}
	incident.History[1] = operations.IngressDiagnosticSample{CapturedAt: at, FramesRead: 300, FramesDispositioned: 110}
	incident.History[2] = operations.IngressDiagnosticSample{CapturedAt: at.Add(time.Second), FramesRead: 300, FramesDispositioned: 110}
	sample := liveOperatorSample{Status: operations.Status{Lifecycle: "suppressed", SampledAt: at}, IngressIncident: incident}
	sample.Metrics.Engine.LifecycleReason = "ingress_integrity"
	sample.Metrics.Engine.Suppression = engine.SuppressionSameBindingRecoveryAllowed
	if err := renderer.Render(sample, true); err != nil {
		t.Fatal(err)
	}
	want := "Capacity evidence · queued 1024/1024 · classifying 1 · bytes 12582912/134217728 · remaining 121634816 · incoming 16384 · high 1024 frames/12582912 bytes · oldest waiting 2s · active aggregate_ingress_fence since " + at.Add(-1250*time.Millisecond).Format(time.RFC3339Nano) + " for 1.25s · read 200.0/s · disposition 20.0/s"
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("capacity output=%q", stderr.String())
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
