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
	startup := run.Engine().ObserveReplayDeterministic()
	startupStatus := run.Status()
	assertSlice1PopulationAndQualification(t, startup.Evaluation)
	if !startupStatus.BackendReady || !startupStatus.RankingCurrent || startup.Evaluation.Mode != "qualified_current" {
		t.Fatalf("startup projection was not current after resolved overlap: status=%+v evaluation=%+v", startupStatus, startup.Evaluation)
	}
	startupAAA := slice1CanonicalSymbol(t, startup, "AAA")
	startupOverlap := slice1CanonicalSymbol(t, startup, "OVERLAP")
	startupQuiet := slice1CanonicalSymbol(t, startup, "QUIET")
	if startupAAA.ProvenAbsentSlots != 69 || startupQuiet.ProvenAbsentSlots != 70 || startupOverlap.ProvenAbsentSlots != 69 {
		t.Fatalf("startup canonical coverage AAA/QUIET/OVERLAP=%d/%d/%d", startupAAA.ProvenAbsentSlots, startupQuiet.ProvenAbsentSlots, startupOverlap.ProvenAbsentSlots)
	}
	if len(startupOverlap.Records) != 1 || startupOverlap.Records[0].AuthoritySource != engine.AggregateSourceLive || startupOverlap.Records[0].Values.Close != 20 ||
		run.Engine().ObserveOperational().Aggregates.Rejected == 0 {
		t.Fatalf("startup overlap did not retain live authority plus discrepancy accounting: symbol=%+v aggregates=%+v", startupOverlap, run.Engine().ObserveOperational().Aggregates)
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

	final := run.Engine().ObserveReplayDeterministic()
	finalStatus := run.Status()
	assertSlice1PopulationAndQualification(t, final.Evaluation)
	if final.Publication.Watermark == nil || *final.Publication.Watermark != handoff.Add(2*time.Second) || final.Publication.PublicationID <= startup.Publication.PublicationID {
		t.Fatalf("ordinary coverage did not advance committed publication: startup=%+v final=%+v", startup.Publication, final.Publication)
	}
	if !finalStatus.BackendReady || !finalStatus.RankingCurrent || final.Evaluation.Mode != startup.Evaluation.Mode || final.Evaluation.Reason != startup.Evaluation.Reason {
		t.Fatalf("quiet seconds regressed the current projection: startup=%+v/%+v final=%+v/%+v", startupStatus, startup.Evaluation, finalStatus, final.Evaluation)
	}
	finalAAA := slice1CanonicalSymbol(t, final, "AAA")
	finalOverlap := slice1CanonicalSymbol(t, final, "OVERLAP")
	finalQuiet := slice1CanonicalSymbol(t, final, "QUIET")
	if finalAAA.ProvenAbsentSlots != 71 || finalQuiet.ProvenAbsentSlots != 72 || finalOverlap.ProvenAbsentSlots != 71 {
		t.Fatalf("ordinary canonical coverage AAA/QUIET/OVERLAP=%d/%d/%d", finalAAA.ProvenAbsentSlots, finalQuiet.ProvenAbsentSlots, finalOverlap.ProvenAbsentSlots)
	}
	if finalAAA.Qualification.Status != "not_yet_passed" || finalAAA.Qualification.UnresolvedOrigin != "" ||
		finalQuiet.Qualification.Status != "not_yet_passed" || finalQuiet.Qualification.UnresolvedOrigin != "" {
		t.Fatalf("quiet live coverage created origin-none qualification uncertainty: AAA=%+v QUIET=%+v", finalAAA.Qualification, finalQuiet.Qualification)
	}
	if len(finalOverlap.Records) != 1 || finalOverlap.Records[0] != startupOverlap.Records[0] || finalOverlap.ProvenAbsentSlots != startupOverlap.ProvenAbsentSlots+2 ||
		run.Engine().ObserveOperational().Aggregates.Rejected != startupRejected {
		t.Fatalf("ordinary coverage overwrote the accepted overlap/discrepancy: startup=%+v final=%+v aggregates=%+v", startupOverlap, finalOverlap, run.Engine().ObserveOperational().Aggregates)
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
			t.Fatalf("timed out waiting for %s: status=%+v view=%+v", description, run.Status(), run.Engine().ObserveReplayDeterministic())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func slice1CanonicalSymbol(t *testing.T, view engine.ReplayDeterministicView, symbol string) engine.ReplayCanonicalSymbol {
	t.Helper()
	for _, candidate := range view.Canonical {
		if candidate.Symbol == symbol {
			return candidate
		}
	}
	t.Fatalf("canonical symbol %s missing", symbol)
	return engine.ReplayCanonicalSymbol{}
}

func assertSlice1PopulationAndQualification(t *testing.T, evaluation engine.ReplayEvaluationView) {
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
