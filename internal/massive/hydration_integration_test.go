package massive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// TestPLBRA2HydrationWorkerAndFence is the worker/composition portion of
// P-LBR-A2-HYDRATION. It composes the real binding, fake live socket, strict
// HTTP worker, sole C4 mapper, C2 FIFO/canonical merge, C3 evaluator, fence,
// and publication path without provider access.
func TestPLBRA2HydrationWorkerAndFence(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	now := binding.SessionStart().Add(30 * time.Second)
	delay := time.Duration(0)
	state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 32, RequiredReserve: 8, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		state.Close()
		wait, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := state.Wait(wait); err != nil {
			t.Error(err)
		}
	}()
	admission, installed := state.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || (<-installed).Code != engine.DispositionBindingInstalled {
		t.Fatal("binding install")
	}

	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapterForBinding(t, socket, binding)
	attempt, started, err := adapter.Start(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := DeliverToEngine(context.Background(), state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("start = %+v %v", result, err)
	}
	handshake, err := attempt.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range handshake {
		result, err := DeliverToEngine(context.Background(), state, delivery)
		if err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake = %+v %v", result, err)
		}
	}

	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 8, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: HydrationMaximumRows, MaximumResidentRecords: HydrationMaximumRows}
	planAdmission, planCompletion := state.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1,
		BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: attempt.Epoch(), Budgets: budgets})
	if planAdmission != engine.AdmissionAdmitted {
		t.Fatal(planAdmission)
	}
	plan := <-planCompletion
	requests := plan.Plan.Requests()
	if len(requests) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	work, err := HydrationWorkItemFromEngine(binding, requests[0])
	if err != nil {
		t.Fatal(err)
	}
	workerPlan, err := NewLiveHydrationWorkerPlan([]HydrationWorkItem{work}, budgets.Workers, 8, budgets.MaximumResponseBytes, budgets.MaximumNormalizedRecords, budgets.MaximumResidentRecords)
	if err != nil {
		t.Fatal(err)
	}
	if workerPlan.workers != 1 {
		t.Fatalf("live hydration workers=%d", workerPlan.workers)
	}
	checkpointWork, err := NewHydrationWorkItem(binding, work.Generation(), work.RequestID()+1, HydrationCheckpointCatchUp, "AAA", work.Start(), work.End(), work.ConnectionEpoch())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLiveHydrationWorkerPlan([]HydrationWorkItem{checkpointWork}, budgets.Workers, 8, budgets.MaximumResponseBytes, budgets.MaximumNormalizedRecords, budgets.MaximumResidentRecords); err == nil {
		t.Fatal("supported live worker accepted checkpoint catch-up work")
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer integration-token" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":100,"vw":10.25,"n":4}]}`, work.Start().UnixMilli())
	}))
	defer server.Close()
	worker, err := NewHydrationWorker(server.URL, func() (string, error) { return "integration-token", nil }, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	sink := &integrationHydrationEngineSink{state: state, tokens: map[uint64]engine.HydrationRequestToken{requests[0].RequestID(): requests[0]}}
	workerResult := worker.Run(context.Background(), context.Background(), workerPlan, sink)
	if workerResult.Accounting().ProviderCompletedValue != 1 || workerResult.Accounting().MaximumActiveWorkers != 1 || sink.err != nil || sink.command.CommandToken() == 0 || sink.chunkDisposition.Rows.Inserted != 1 {
		t.Fatalf("worker/sink = %+v err=%v command=%+v", workerResult.Accounting(), sink.err, sink.command)
	}
	// The pre-capture live frame overlaps the REST identity and replaces its
	// historical authority before the real causal fence reaches Component 2.
	liveAt := work.Start()
	socket.send(socketMessageText, "["+aggregateLiveJSON("AAA", liveAt, `"dv":"1000.5"`)+"]")
	waitForQueuedFrames(t, attempt, 1)
	revision, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
	if err != nil || !ok || revision.AggregateDisposition.Code != engine.DispositionAggregateRevised {
		t.Fatalf("live authority revision = %+v ok=%v err=%v", revision, ok, err)
	}

	command, err := CaptureAggregateIngressFenceCommandFromEngine(sink.command)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.CaptureAggregateIngressFence(context.Background(), state, command); err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC()
	fenceDelivery, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
	if err != nil || !ok || fenceDelivery.HydrationDisposition.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence = %+v ok=%v err=%v", fenceDelivery, ok, err)
	}

	view := state.ObserveReplayDeterministic()
	if view.Publication.Lifecycle != "live" || view.Publication.Watermark == nil || view.Publication.LastDisposition != engine.DispositionAggregateIngressFenceApplied ||
		len(view.Canonical) != 1 || view.Canonical[0].Symbol != "AAA" || view.Canonical[0].PresentSlots != 1 || view.Canonical[0].LatestWindowStart != liveAt {
		t.Fatalf("one canonical/evaluator/publication path = %+v", view)
	}
	if view.Canonical[0].LatestAuthoritySource != engine.AggregateSourceLive ||
		view.Canonical[0].LatestValues.Close != 10.5 || view.Canonical[0].LatestValues.Volume != 1000.5 {
		t.Fatalf("historical overwrote live authority = %+v", view.Canonical[0])
	}
	if !reflect.DeepEqual(view.Evaluation, view.Publication.AggregateEvaluation) || !view.Publication.CurrentMarketClaim ||
		view.Evaluation.Mode != "qualified_current" || view.Evaluation.Reason != "" ||
		view.Evaluation.Population.UniverseTotal != 1 || view.Evaluation.Population.ValidPriorClose != 1 ||
		view.Evaluation.Population.TrustedRankableMark != 1 || view.Evaluation.Population.CoveredPopulation != 1 ||
		view.Evaluation.Population.UnresolvedPopulation != 0 || view.Evaluation.Qualification.NotYetPassed != 1 ||
		view.Evaluation.TotalPassers != 0 || view.Evaluation.KnownRankableCount != 1 || len(view.Evaluation.Rows) != 0 {
		t.Fatalf("ordinary C3 evaluation/publication = evaluation=%+v publication=%+v", view.Evaluation, view.Publication)
	}
	if err := attempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: 2, Cause: CloseControlledStop}); err != nil {
		t.Fatal(err)
	}
	if terminal, ok := attempt.nextForProof(context.Background()); !ok || terminal.Kind != DeliveryTerminal {
		t.Fatalf("terminal = %+v ok=%v", terminal, ok)
	}
}

type integrationHydrationEngineSink struct {
	mu                  sync.Mutex
	state               *engine.Engine
	tokens              map[uint64]engine.HydrationRequestToken
	command             engine.HydrationFenceCommand
	chunkDisposition    engine.HydrationDisposition
	terminalDisposition engine.HydrationDisposition
	err                 error
}

func (s *integrationHydrationEngineSink) AdmitHydrationChunk(ctx context.Context, chunk HydrationResultChunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	input, err := EngineHydrationChunk(s.tokens[chunk.ResultID()], chunk)
	if err != nil {
		s.err = err
		return err
	}
	admission, completion := s.state.AdmitHydrationChunk(ctx, input)
	if admission != engine.AdmissionAdmitted {
		s.err = fmt.Errorf("chunk admission: %s", admission)
		return s.err
	}
	got := <-completion
	if got.Code != engine.DispositionHydrationChunkApplied {
		s.err = fmt.Errorf("chunk disposition: %s/%s", got.Code, got.Reason)
		return s.err
	}
	s.chunkDisposition = got
	return nil
}

func (s *integrationHydrationEngineSink) AdmitHydrationTerminal(ctx context.Context, terminal HydrationTerminal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	input, err := EngineHydrationTerminal(s.tokens[terminal.ResultID()], terminal)
	if err != nil {
		s.err = err
		return err
	}
	admission, completion := s.state.AdmitHydrationTerminal(ctx, input)
	if admission != engine.AdmissionAdmitted {
		if admission == engine.AdmissionNotAdmittedClosed {
			return ErrHydrationInputClosed
		}
		if admission == engine.AdmissionNotAdmittedCanceled && ctx.Err() != nil {
			return ctx.Err()
		}
		s.err = fmt.Errorf("terminal admission: %s", admission)
		return s.err
	}
	got := <-completion
	if got.Code != engine.DispositionHydrationTerminalApplied {
		s.err = fmt.Errorf("terminal disposition: %s/%s", got.Code, got.Reason)
		return s.err
	}
	s.terminalDisposition = got
	s.command = got.FenceCommand
	return nil
}
