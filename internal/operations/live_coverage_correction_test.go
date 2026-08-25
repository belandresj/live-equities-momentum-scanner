package operations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

// TestSlice1RuntimeRunLiveInstallsOrdinaryLiveCoverage is the primary Slice 1
// proof. It crosses the production hydration worker and fake live adapter,
// then requires two ordinary live fences to extend the same canonical
// presence/absence truth installed by the startup ingress fence.
func TestSlice1RuntimeRunLiveInstallsOrdinaryLiveCoverage(t *testing.T) {
	runSlice1RuntimeRunLiveComposition(t)
}

// TestPLBRE1ProductionCompositionTrace enters exclusively through RunLive;
// the fixture's fake providers drive the normal socket, hydration, fence,
// timer, publication, terminal, and joined-shutdown orchestration.
func TestPLBRE1ProductionCompositionTrace(t *testing.T) {
	runSlice1RuntimeRunLiveComposition(t)
}

func runSlice1RuntimeRunLiveComposition(t *testing.T) {
	binding := capacityBinding(t, []string{"AAA", "OVERLAP", "QUIET"})
	handoff := binding.SessionStart().Add(70 * time.Second)
	var clockNanos atomic.Int64
	clockNanos.Store(handoff.UnixNano())
	clock := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }

	liveWritten := make(chan struct{})
	websocketServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		ctx := request.Context()
		write := func(value string) bool { return connection.Write(ctx, websocket.MessageText, []byte(value)) == nil }
		if !write(`[{"ev":"status","status":"connected"}]`) {
			return
		}
		if _, _, err := connection.Read(ctx); err != nil || !write(`[{"ev":"status","status":"auth_success"}]`) {
			return
		}
		if _, raw, err := connection.Read(ctx); err != nil || !strings.Contains(string(raw), "A.*") || !write(`[{"ev":"status","status":"success"}]`) {
			return
		}
		frame := "[" + strings.Join([]string{
			capacityAggregateJSON("AAA", handoff.Add(-time.Second), 12, 12),
			capacityAggregateJSON("OVERLAP", handoff.Add(-2*time.Second), 20, 20),
		}, ",") + "]"
		if !write(frame) {
			return
		}
		close(liveWritten)
		<-ctx.Done()
	}))
	t.Cleanup(websocketServer.Close)

	releaseREST := make(chan struct{})
	var releaseOnce sync.Once
	hydrationServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
			return
		case <-releaseREST:
		}
		parts := strings.Split(request.URL.Path, "/")
		if len(parts) != 10 {
			http.Error(writer, "unexpected hydration path", http.StatusBadRequest)
			return
		}
		symbol := parts[4]
		writer.Header().Set("Content-Type", "application/json")
		if symbol == "OVERLAP" {
			// This deliberately disagrees with the already accepted live identity.
			// The live value must remain canonical while the discrepancy stays a
			// diagnostic across later ordinary no-print coverage.
			_, _ = fmt.Fprintf(writer, `{"status":"OK","ticker":"OVERLAP","adjusted":false,"count":1,"results":[{"t":%d,"o":19,"h":21,"l":18,"c":19,"v":100,"vw":19,"n":10}]}`,
				handoff.Add(-2*time.Second).UnixMilli())
			return
		}
		_, _ = fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, symbol)
	}))
	t.Cleanup(hydrationServer.Close)

	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{
		Endpoint: "ws" + strings.TrimPrefix(websocketServer.URL, "http"), Credential: "fixture",
		Queue: massive.LiveQueueConfig{FrameSlots: 8, MaxFrameBytes: 4096, TotalFrameBytes: 16384}, Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	hydrator, err := massive.NewHydrationWorker(hydrationServer.URL, func() (string, error) { return "fixture", nil }, hydrationServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.EvaluationDelay = 0
	config.ReadinessTolerance = time.Second
	config.SampleCadence = 20 * time.Millisecond
	config.ConnectionAttemptDeadline = 2 * time.Second
	config.ShutdownDeadline = 2 * time.Second
	run, err := New(context.Background(), binding, config, clock)
	if err != nil {
		t.Fatal(err)
	}
	components := LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: 1, RowsPerChunk: 4,
		MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 3 * 57_600, MaximumResidentRecords: 57_600,
		Durations: capacityDurations()}
	joined := make(chan error, 1)
	go func() { joined <- run.RunLive(context.Background(), components) }()
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseREST) })
		if run.joined.Load() {
			return
		}
		shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownDeadline)
		defer cancel()
		_ = run.Shutdown(shutdown)
	})

	select {
	case <-liveWritten:
	case err := <-joined:
		t.Fatalf("RunLive ended before live overlap: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("fake live adapter did not emit aggregate frame")
	}
	waitForSlice1Coverage(t, run, joined, func() bool {
		return run.Engine().ObserveOperational().Aggregates.Inserted == 2
	}, "live aggregates to precede REST")
	releaseOnce.Do(func() { close(releaseREST) })

	waitForSlice1Coverage(t, run, joined, func() bool {
		view := run.Engine().ObserveOperational()
		return view.Lifecycle == "live" && view.Watermark != nil && *view.Watermark == handoff
	}, "startup ingress fence")
	startup := run.Engine().ObserveSnapshot()
	startupStatus := run.Status()
	assertSlice1PopulationAndQualification(t, startup.Publication.AggregateEvaluation)
	if !startupStatus.BackendReady || !startupStatus.RankingCurrent || startup.Publication.AggregateEvaluation.Mode != "qualified_current" {
		t.Fatalf("startup projection was not current after resolved overlap: status=%+v evaluation=%+v", startupStatus, startup.Publication.AggregateEvaluation)
	}
	if run.Engine().ObserveOperational().Aggregates.Rejected == 0 {
		t.Fatalf("startup overlap did not retain discrepancy accounting: aggregates=%+v", run.Engine().ObserveOperational().Aggregates)
	}
	startupRejected := run.Engine().ObserveOperational().Aggregates.Rejected
	startupFences := run.Metrics().LiveQueue.IngressFencesDispositioned

	for step := 1; step <= 2; step++ {
		target := handoff.Add(time.Duration(step) * time.Second)
		clockNanos.Store(target.UnixNano())
		waitForSlice1Coverage(t, run, joined, func() bool {
			view := run.Engine().ObserveOperational()
			return view.Watermark != nil && *view.Watermark == target
		}, fmt.Sprintf("ordinary live fence %d", step))
	}

	final := run.Engine().ObserveSnapshot()
	finalStatus := run.Status()
	assertSlice1PopulationAndQualification(t, final.Publication.AggregateEvaluation)
	if final.Publication.Watermark == nil || *final.Publication.Watermark != handoff.Add(2*time.Second) || final.Publication.PublicationID <= startup.Publication.PublicationID {
		t.Fatalf("ordinary coverage did not advance committed publication: startup=%+v final=%+v", startup.Publication, final.Publication)
	}
	if !finalStatus.BackendReady || !finalStatus.RankingCurrent || final.Publication.AggregateEvaluation.Mode != startup.Publication.AggregateEvaluation.Mode || final.Publication.AggregateEvaluation.Reason != startup.Publication.AggregateEvaluation.Reason {
		t.Fatalf("quiet seconds regressed the current projection: startup=%+v/%+v final=%+v/%+v", startupStatus, startup.Publication.AggregateEvaluation, finalStatus, final.Publication.AggregateEvaluation)
	}
	if final.TQ.PublicationID != final.Publication.PublicationID || len(final.TQ.Desired) != 0 || len(final.TQ.Rows) != 0 || final.TQ.Bounds || final.TQ.AggregateOnly || final.TQ.Pressure != engine.TQPressureNormal {
		t.Fatalf("fixture-derived T/Q/publication identity mismatch: publication=%d tq=%+v", final.Publication.PublicationID, final.TQ)
	}
	if run.Engine().ObserveOperational().Aggregates.Rejected != startupRejected {
		t.Fatalf("ordinary coverage changed accepted overlap discrepancy accounting: aggregates=%+v", run.Engine().ObserveOperational().Aggregates)
	}
	metrics := run.Metrics()
	if metrics.LiveQueue.IngressFencesDispositioned < startupFences+2 || !metrics.AccountingValid || !metrics.LiveQueue.Reconciles() || !metrics.Adapter.Reconciles() {
		t.Fatalf("ordinary fence/accounting proof failed: startup_fences=%d metrics=%+v", startupFences, metrics)
	}

	shutdown, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownDeadline)
	defer cancelShutdown()
	if err := run.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-joined:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunLive shutdown result=%v", err)
		}
	case <-time.After(config.ShutdownDeadline):
		t.Fatal("Runtime.Shutdown returned before RunLive joined")
	}
	if !run.joined.Load() {
		t.Fatal("runtime did not record joined shutdown")
	}
}

func waitForSlice1Coverage(t *testing.T, run *Runtime, joined <-chan error, ready func() bool, description string) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for !ready() {
		select {
		case err := <-joined:
			t.Fatalf("RunLive ended while waiting for %s: %v", description, err)
		case <-deadline.C:
			t.Fatalf("timed out waiting for %s: status=%+v view=%+v", description, run.Status(), run.Engine().ObserveSnapshot())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func assertSlice1PopulationAndQualification(t *testing.T, evaluation engine.EvaluationView) {
	t.Helper()
	population := evaluation.Population
	qualification := evaluation.Qualification
	if population.UniverseTotal != 3 || population.ValidPriorClose != 3 || population.InvalidOrMissingPriorClose != 0 ||
		population.TrustedRankableMark != 2 || population.TrustedBelowPriceMark != 0 || population.NoPrintThroughT != 1 ||
		population.InvalidMark != 0 || population.UnknownDueFailureOrFence != 0 || population.CoveredPopulation != 3 || population.UnresolvedPopulation != 0 {
		t.Fatalf("population accounting=%+v", population)
	}
	if population.UniverseTotal != population.ValidPriorClose+population.InvalidOrMissingPriorClose ||
		population.ValidPriorClose != population.TrustedRankableMark+population.TrustedBelowPriceMark+population.NoPrintThroughT+population.InvalidMark+population.UnknownDueFailureOrFence {
		t.Fatalf("population identities do not reconcile: %+v", population)
	}
	if qualification.NotYetPassed != 2 || qualification.Provisional != 0 || qualification.Finalized != 0 || qualification.Unresolved != 0 ||
		qualification.NotYetPassed+qualification.Provisional+qualification.Finalized+qualification.Unresolved != population.TrustedRankableMark {
		t.Fatalf("qualification accounting=%+v population=%+v", qualification, population)
	}
	if evaluation.Uncertainty.BootstrapOrigin != 0 || evaluation.Uncertainty.PostBootstrapGap != 0 || evaluation.Uncertainty.LocalInvalid != 0 ||
		evaluation.Uncertainty.BootstrapOrigin+evaluation.Uncertainty.PostBootstrapGap+evaluation.Uncertainty.LocalInvalid != population.UnknownDueFailureOrFence+qualification.Unresolved {
		t.Fatalf("uncertainty-origin accounting=%+v population=%+v qualification=%+v", evaluation.Uncertainty, population, qualification)
	}
}
