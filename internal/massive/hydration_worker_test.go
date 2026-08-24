package massive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestC6S1FactsConvertToEngineLedger proves the concrete S1/S2 seam preserves
// the engine allocation and copies sealed provider values. It does not repeat
// worker HTTP or engine canonical-merge proofs.
func TestC6S1FactsConvertToEngineLedger(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	now := binding.SessionStart().Add(2 * time.Second)
	delay := time.Duration(0)
	e, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		e.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	result, bindingCompletion := e.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if result != engine.AdmissionAdmitted || (<-bindingCompletion).Code != engine.DispositionBindingInstalled {
		t.Fatalf("binding admission = %s", result)
	}
	controls := []engine.ConnectionControlInput{
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.ConnectionAttempt, ConnectionEpoch: 1, ReceiptTime: now, CommandToken: 1, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateCommandWriteResult, ConnectionEpoch: 1, ReceiptTime: now, CommandToken: 2, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateSubscriptionResult, ConnectionEpoch: 1, Position: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ReceiptTime: now, CommandToken: 2, Outcome: engine.ControlSucceeded},
	}
	for _, input := range controls {
		admission, completion := e.AdmitConnectionControl(context.Background(), input)
		if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("control admission = %s", admission)
		}
	}
	planAdmission, planCompletion := e.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{
		SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1,
		Budgets: engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 2, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 10, MaximumResidentRecords: 10},
	})
	if planAdmission != engine.AdmissionAdmitted {
		t.Fatalf("plan admission = %s", planAdmission)
	}
	plan := <-planCompletion
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	token := requests[0]
	work, err := HydrationWorkItemFromEngine(binding, token)
	if err != nil {
		t.Fatal(err)
	}
	value := RESTSecondAggregate{Symbol: "AAA", WindowStart: binding.SessionStart(), WindowEnd: binding.SessionStart().Add(time.Second), Values: engine.AggregateValues{
		Open: 10, High: 10, Low: 10, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades,
	}}
	chunk := HydrationResultChunk{work: work, ordinal: 0, totalChunks: 1, totalRows: 1, values: []RESTSecondAggregate{value}}
	engineChunk, err := EngineHydrationChunk(token, chunk)
	if err != nil {
		t.Fatal(err)
	}
	chunk.values[0].Symbol = "MUTATED"
	chunkAdmission, chunkCompletion := e.AdmitHydrationChunk(context.Background(), engineChunk)
	if chunkAdmission != engine.AdmissionAdmitted || (<-chunkCompletion).Code != engine.DispositionHydrationChunkApplied {
		t.Fatalf("chunk admission = %s", chunkAdmission)
	}
	terminal := HydrationTerminal{work: work, state: HydrationCompletedValue, reason: DownloadReasonNone, pages: 1, attempts: 1, responseBytes: 10, normalizedRows: 1, emittedChunks: 1, emittedRows: 1}
	engineTerminal, err := EngineHydrationTerminal(token, terminal)
	if err != nil {
		t.Fatal(err)
	}
	terminalAdmission, terminalCompletion := e.AdmitHydrationTerminal(context.Background(), engineTerminal)
	got := <-terminalCompletion
	if terminalAdmission != engine.AdmissionAdmitted || got.Code != engine.DispositionHydrationTerminalApplied || got.Accounting.CompletedValue != 1 || got.Accounting.Open != 0 {
		t.Fatalf("terminal admission=%s disposition=%+v", terminalAdmission, got)
	}
	other := token
	// The immutable token has no setters; a work item from another binding or
	// request cannot be converted under this allocation.
	bad := work
	bad.requestID++
	if _, err := EngineHydrationChunk(other, HydrationResultChunk{work: bad, ordinal: 0, totalChunks: 1, totalRows: 1, values: []RESTSecondAggregate{value}}); err == nil {
		t.Fatal("mismatched provider request converted to an engine chunk")
	}
}

// TestHydrationSharedRESTContract is P-C6-REST. It proves the production
// consumer uses the exact strict C4 acquisition path and sole row mapper; it
// makes no provider-availability or engine-consumption claim.
func TestHydrationSharedRESTContract(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	end := start.Add(3 * time.Second)
	item := mustHydrationWork(t, binding, 1, 1, "AAA", start, end)
	plan := mustHydrationPlan(t, []HydrationWorkItem{item}, 1, 2, 1<<20, 3, 3)

	t.Run("fixed request strict continuation and sole mapper", func(t *testing.T) {
		var server *httptest.Server
		var requests atomic.Int64
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests.Add(1)
			if request.Header.Get("Authorization") != "Bearer test-token" || request.URL.Query().Get("apiKey") != "" ||
				request.URL.Query().Get("adjusted") != "false" || request.URL.Query().Get("sort") != "asc" || request.URL.Query().Get("limit") != "50000" {
				t.Errorf("request credential/query = %q %q", request.Header.Get("Authorization"), request.URL.RawQuery)
			}
			wantPath := fmt.Sprintf("/v2/aggs/ticker/AAA/range/1/second/%d/%d", start.UnixMilli(), end.Add(-time.Millisecond).UnixMilli())
			if request.URL.Path != wantPath {
				t.Errorf("request path = %q want %q", request.URL.Path, wantPath)
			}
			if request.URL.Query().Get("cursor") == "" {
				fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":100.5,"vw":10.25,"n":3}],"next_url":%q}`,
					start.UnixMilli(), server.URL+wantPath+"?cursor=two&apiKey=must-strip")
				return
			}
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"resultsCount":1,"results":[{"t":%d,"o":11,"h":12,"l":10,"c":11.5,"v":10,"vw":11.25,"n":4}]}`,
				start.Add(2*time.Second).UnixMilli())
		}))
		defer server.Close()
		worker := testHydrationWorker(t, server)
		sink := &recordingHydrationSink{}
		result := worker.Run(context.Background(), context.Background(), plan, sink)
		accounting := result.Accounting()
		terminals := result.Terminals()
		if requests.Load() != 2 || len(terminals) != 1 || terminals[0].State() != HydrationCompletedValue ||
			terminals[0].Pages() != 2 || terminals[0].NormalizedRows() != 2 || accounting.EmittedRows != 2 || accounting.UnadmittedTerminals != 0 {
			t.Fatalf("shared acquisition result requests=%d accounting=%+v terminals=%+v", requests.Load(), accounting, terminals)
		}
		chunks, _ := sink.snapshot()
		if len(chunks) != 1 || len(chunks[0].Values()) != 2 || chunks[0].Values()[0].Values.AverageTradeSize != 33 {
			t.Fatalf("normalized chunks = %+v", chunks)
		}
	})

	t.Run("late invalid row discards complete result", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":2,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1},{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10}]}`,
				start.UnixMilli(), start.Add(time.Second).UnixMilli())
		}))
		defer server.Close()
		sink := &recordingHydrationSink{}
		result := testHydrationWorker(t, server).Run(context.Background(), context.Background(), plan, sink)
		chunks, terminals := sink.snapshot()
		if len(chunks) != 0 || len(terminals) != 1 || terminals[0].State() != HydrationFailed || terminals[0].Reason() != DownloadReasonNumericCount || result.Accounting().ProviderFailed != 1 {
			t.Fatalf("late invalid row exposed success chunks=%+v terminals=%+v accounting=%+v", chunks, terminals, result.Accounting())
		}
	})

	t.Run("descending rows fail instead of becoming value or empty", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":2,"results":[{"t":%d,"o":11,"h":12,"l":10,"c":11.5,"v":1,"vw":11,"n":1},{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}]}`,
				start.Add(time.Second).UnixMilli(), start.UnixMilli())
		}))
		defer server.Close()
		sink := &recordingHydrationSink{}
		result := testHydrationWorker(t, server).Run(context.Background(), context.Background(), plan, sink)
		chunks, terminals := sink.snapshot()
		if len(chunks) != 0 || len(terminals) != 1 || terminals[0].State() != HydrationFailed ||
			terminals[0].Reason() != DownloadReasonSymbolIntervalOrder || result.Accounting().ProviderFailed != 1 ||
			result.Accounting().ProviderCompletedValue != 0 || result.Accounting().ProviderCompletedEmpty != 0 {
			t.Fatalf("descending rows exposed false success chunks=%+v terminals=%+v accounting=%+v", chunks, terminals, result.Accounting())
		}
	})

	t.Run("invalid second page discards buffered first page", func(t *testing.T) {
		var requests atomic.Int64
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests.Add(1)
			if request.URL.Query().Get("cursor") == "" {
				fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":1,"vw":10,"n":1}],"next_url":%q}`,
					start.UnixMilli(), server.URL+request.URL.Path+"?cursor=two")
				return
			}
			fmt.Fprint(writer, `{"status":"OK","ticker":"FOREIGN","adjusted":false,"results":[]}`)
		}))
		defer server.Close()
		sink := &recordingHydrationSink{}
		result := testHydrationWorker(t, server).Run(context.Background(), context.Background(), plan, sink)
		chunks, admittedTerminals := sink.snapshot()
		terminals := result.Terminals()
		if requests.Load() != 2 || len(chunks) != 0 || len(admittedTerminals) != 1 || len(terminals) != 1 ||
			terminals[0].State() != HydrationFailed || terminals[0].Reason() != DownloadReasonEnvelopeIdentityStatus ||
			result.Accounting().ProviderFailed != 1 || result.Accounting().ProviderCompletedEmpty != 0 || result.Accounting().ProviderCompletedValue != 0 {
			t.Fatalf("second-page false success requests=%d chunks=%+v admitted=%+v terminals=%+v accounting=%+v",
				requests.Load(), chunks, admittedTerminals, terminals, result.Accounting())
		}
	})

	t.Run("construction has one decoder and mapper call", func(t *testing.T) {
		files, err := filepath.Glob("*.go")
		if err != nil {
			t.Fatal(err)
		}
		var production strings.Builder
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			production.Write(body)
		}
		source := production.String()
		if strings.Count(source, "func decodeRESTPage(") != 1 || strings.Count(source, "decodeRESTPage(body") != 2 ||
			strings.Count(source, "func NormalizeRESTSecondAggregate(") != 1 || strings.Count(source, "NormalizeRESTSecondAggregate(symbol, raw)") != 1 ||
			strings.Count(source, ".acquire(ctx, token") != 2 {
			t.Fatalf("shared construction counts decoder-def=%d decoder-use+def=%d mapper-def=%d mapper-use=%d core-consumers=%d",
				strings.Count(source, "func decodeRESTPage("), strings.Count(source, "decodeRESTPage(body"),
				strings.Count(source, "func NormalizeRESTSecondAggregate("), strings.Count(source, "NormalizeRESTSecondAggregate(symbol, raw)"),
				strings.Count(source, ".acquire(ctx, token"))
		}
	})
}

// TestHydrationWorkerSealedChunksAndTerminal is P-C6-WORKER. It proves full
// sealing, copied contiguous chunks, strict empty/value distinction, and one
// producer terminal per item; engine applicability remains unproved.
func TestHydrationWorkerSealedChunksAndTerminal(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB"})
	start := binding.SessionStart()
	end := start.Add(3 * time.Second)
	items := []HydrationWorkItem{
		mustHydrationWork(t, binding, 7, 11, "AAA", start, end),
		mustHydrationWork(t, binding, 7, 12, "BBB", start, end),
	}
	plan := mustHydrationPlan(t, items, 2, 1, 1<<20, 6, 6)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		symbol := strings.Split(request.URL.Path, "/")[4]
		if symbol == "BBB" {
			fmt.Fprint(writer, `{"status":"OK","ticker":"BBB","adjusted":false,"results":[]}`)
			return
		}
		fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":3,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"n":1},{"t":%d,"o":11,"h":12,"l":10,"c":11,"v":2,"vw":11,"n":1},{"t":%d,"o":12,"h":13,"l":11,"c":12,"v":3,"vw":12,"n":1}]}`,
			start.UnixMilli(), start.Add(time.Second).UnixMilli(), start.Add(2*time.Second).UnixMilli())
	}))
	defer server.Close()
	sink := &recordingHydrationSink{}
	result := testHydrationWorker(t, server).Run(context.Background(), context.Background(), plan, sink)
	chunks, admittedTerminals := sink.snapshot()
	if len(chunks) != 3 || len(admittedTerminals) != 2 || result.Accounting().ProviderCompletedValue != 1 || result.Accounting().ProviderCompletedEmpty != 1 ||
		result.Accounting().ItemsStarted != result.Accounting().ProviderCompletedValue+result.Accounting().ProviderCompletedEmpty+result.Accounting().ProviderFailed+result.Accounting().ProviderCanceled {
		t.Fatalf("worker partition chunks=%d terminals=%+v accounting=%+v", len(chunks), admittedTerminals, result.Accounting())
	}
	for ordinal, chunk := range chunks {
		if chunk.WorkItem().Symbol() != "AAA" || chunk.Ordinal() != ordinal || chunk.TotalChunks() != 3 || chunk.RowOffset() != int64(ordinal) || chunk.TotalRows() != 3 || len(chunk.Values()) != 1 {
			t.Fatalf("chunk %d = %+v values=%+v", ordinal, chunk, chunk.Values())
		}
	}
	probe := chunks[0].Values()
	probe[0].Symbol = "MUTATED"
	if chunks[0].Values()[0].Symbol != "AAA" || result.Terminals()[0].WorkItem().Symbol() != "AAA" {
		t.Fatal("caller mutation changed sealed chunk or terminal identity")
	}

	t.Run("cancellation after last chunk cannot report completed value", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancelingSink := &recordingHydrationSink{afterChunk: func(chunk HydrationResultChunk) {
			if chunk.Ordinal()+1 == chunk.TotalChunks() {
				cancel()
			}
		}}
		single := mustHydrationPlan(t, items[:1], 1, 2, 1<<20, 3, 3)
		got := testHydrationWorker(t, server).Run(ctx, context.Background(), single, cancelingSink)
		_, terminals := cancelingSink.snapshot()
		if len(terminals) != 1 || terminals[0].State() != HydrationCanceled || terminals[0].EmittedChunks() != 2 || got.Accounting().ProviderCanceled != 1 {
			t.Fatalf("last-chunk cancellation terminals=%+v accounting=%+v", terminals, got.Accounting())
		}
	})
}

// TestHydrationWorkerBoundsAndCancellation is P-C6-BOUND. It proves worker,
// page/body/row/resident/chunk bounds, cancellation-aware blocked admission,
// and joined workers; it does not choose deployed capacity or retry policy.
func TestHydrationWorkerBoundsAndCancellation(t *testing.T) {
	symbols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	binding := component4TestBinding(t, symbols)
	start := binding.SessionStart()
	items := make([]HydrationWorkItem, len(symbols))
	for index, symbol := range symbols {
		items[index] = mustHydrationWork(t, binding, 1, uint64(index+1), symbol, start, start.Add(time.Second))
	}
	if _, err := NewHydrationWorkerPlan(items, 9, 1, 1<<20, 8, 8); err == nil {
		t.Fatal("ninth worker was accepted")
	}
	if _, err := NewHydrationWorkerPlan(items, 8, 0, 1<<20, 8, 8); err == nil {
		t.Fatal("zero chunk size was accepted")
	}

	var active atomic.Int64
	var maximum atomic.Int64
	allActive := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		updateAtomicMaximum(&maximum, current)
		if current == int64(len(symbols)) {
			close(allActive)
		}
		<-release
		active.Add(-1)
		symbol := strings.Split(request.URL.Path, "/")[4]
		fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"n":1}]}`, symbol, start.UnixMilli())
	}))
	plan := mustHydrationPlan(t, items, 8, 1, 1<<20, 8, 8)
	worker := testHydrationWorker(t, server)
	done := make(chan HydrationWorkerResult, 1)
	go func() {
		done <- worker.Run(context.Background(), context.Background(), plan, &recordingHydrationSink{})
	}()
	select {
	case <-allActive:
	case <-time.After(3 * time.Second):
		t.Fatal("eight workers did not become concurrently active")
	}
	close(release)
	result := <-done
	server.Close()
	if maximum.Load() != 8 || result.Accounting().MaximumActiveWorkers != 8 || result.Accounting().MaximumResidentRecords > 8 || result.Accounting().ProviderCompletedValue != 8 {
		t.Fatalf("worker/resident bounds server-max=%d accounting=%+v", maximum.Load(), result.Accounting())
	}

	t.Run("response budget is atomic across workers", func(t *testing.T) {
		var consumed atomic.Int64
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(countingZeroReader{count: &consumed}), Request: request}, nil
		})}
		bounded, err := NewHydrationWorker("https://provider.invalid", func() (string, error) { return "test-token", nil }, client)
		if err != nil {
			t.Fatal(err)
		}
		const byteBudget = int64(32)
		boundedPlan := mustHydrationPlan(t, items[:2], 2, 1, byteBudget, 2, 2)
		got := bounded.Run(context.Background(), context.Background(), boundedPlan, &recordingHydrationSink{})
		if consumed.Load() > byteBudget+1 || got.Accounting().ProviderFailed != 2 || got.Accounting().UnadmittedTerminals != 0 {
			t.Fatalf("response budget consumed=%d accounting=%+v", consumed.Load(), got.Accounting())
		}
	})

	t.Run("cancel releases blocked chunk admission and joins", func(t *testing.T) {
		oneServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(writer, `{"status":"OK","ticker":"A","adjusted":false,"count":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"n":1}]}`, start.UnixMilli())
		}))
		defer oneServer.Close()
		entered := make(chan struct{})
		blocked := &blockingHydrationSink{entered: entered}
		ctx, cancel := context.WithCancel(context.Background())
		joined := make(chan HydrationWorkerResult, 1)
		oneWorker := testHydrationWorker(t, oneServer)
		onePlan := mustHydrationPlan(t, items[:1], 1, 1, 1<<20, 1, 1)
		go func() {
			joined <- oneWorker.Run(ctx, context.Background(), onePlan, blocked)
		}()
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("chunk admission did not block")
		}
		cancel()
		select {
		case got := <-joined:
			if got.Accounting().ProviderCanceled != 1 || got.Accounting().MaximumActiveWorkers != 1 || got.Accounting().UnadmittedTerminals != 0 {
				t.Fatalf("blocked-admission cancellation accounting=%+v", got.Accounting())
			}
		case <-time.After(3 * time.Second):
			t.Fatal("worker goroutine outlived canceled operation")
		}
	})

	t.Run("work canceled with admission open emits canceled terminal", func(t *testing.T) {
		workContext, cancelWork := context.WithCancel(context.Background())
		cancelWork()
		sink := &contextRespectingHydrationSink{}
		onePlan := mustHydrationPlan(t, items[:1], 1, 1, 1<<20, 1, 1)
		got := worker.Run(workContext, context.Background(), onePlan, sink)
		_, terminals := sink.snapshot()
		if len(terminals) != 1 || terminals[0].State() != HydrationCanceled || got.Accounting().ProviderCanceled != 1 ||
			got.Accounting().UnadmittedTerminals != 0 || got.Accounting().AdmissionIntegrityFailures != 0 {
			t.Fatalf("open-input cancellation terminals=%+v accounting=%+v", terminals, got.Accounting())
		}
	})

	t.Run("closed admission reports explicit cleanup", func(t *testing.T) {
		admissionContext, closeAdmission := context.WithCancel(context.Background())
		closeAdmission()
		sink := &contextRespectingHydrationSink{closed: true}
		onePlan := mustHydrationPlan(t, items[:1], 1, 1, 1<<20, 1, 1)
		got := worker.Run(context.Background(), admissionContext, onePlan, sink)
		_, terminals := sink.snapshot()
		if len(terminals) != 0 || got.Accounting().ProviderCanceled != 1 || got.Accounting().UnadmittedTerminals != 1 || got.Accounting().AdmissionIntegrityFailures != 0 {
			t.Fatalf("closed-input cleanup terminals=%+v accounting=%+v", terminals, got.Accounting())
		}
	})

	t.Run("cancel releases blocked terminal admission and retries canceled terminal", func(t *testing.T) {
		emptyServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(writer, `{"status":"OK","ticker":"A","adjusted":false,"results":[]}`)
		}))
		defer emptyServer.Close()
		entered := make(chan struct{})
		blocked := &blockingTerminalHydrationSink{entered: entered}
		workContext, cancelWork := context.WithCancel(context.Background())
		admissionContext, closeAdmission := context.WithCancel(context.Background())
		defer closeAdmission()
		oneWorker := testHydrationWorker(t, emptyServer)
		onePlan := mustHydrationPlan(t, items[:1], 1, 1, 1<<20, 1, 1)
		joined := make(chan HydrationWorkerResult, 1)
		go func() {
			joined <- oneWorker.Run(workContext, admissionContext, onePlan, blocked)
		}()
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("terminal admission did not block")
		}
		cancelWork()
		select {
		case got := <-joined:
			terminals := got.Terminals()
			if admissionContext.Err() != nil || len(terminals) != 1 || terminals[0].State() != HydrationCanceled ||
				got.Accounting().ProviderCanceled != 1 || got.Accounting().UnadmittedTerminals != 0 || got.Accounting().AdmissionIntegrityFailures != 0 ||
				got.Accounting().MaximumActiveWorkers != 1 || blocked.admittedTerminal().State() != HydrationCanceled {
				t.Fatalf("blocked-terminal cancellation terminals=%+v accounting=%+v admission-err=%v", terminals, got.Accounting(), admissionContext.Err())
			}
		case <-time.After(3 * time.Second):
			t.Fatal("terminal admission outlived canceled work")
		}
	})
}

// TestPLBRA3LivePlanBoundsAndJoinedCancellation is the direct worker half of
// P-LBR-A3-PARALLEL-HYDRATION. The operations proof composes the same plans
// with the live pump, engine ledger, fence, evaluation, and publication.
func TestPLBRA3LivePlanBoundsAndJoinedCancellation(t *testing.T) {
	symbols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	binding := component4TestBinding(t, symbols)
	start := binding.SessionStart()
	end := start.Add(16 * time.Hour)
	items := make([]HydrationWorkItem, len(symbols))
	for index, symbol := range symbols {
		items[index] = mustHydrationWork(t, binding, 1, uint64(index+1), symbol, start, end)
	}
	const rowsPerRequest = int64(57_600)
	for _, workers := range []int{1, 2, 4, 8} {
		t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
			resident := int64(workers) * rowsPerRequest
			plan, err := NewLiveHydrationWorkerPlan(items, workers, 256, 1<<20, int64(len(items))*rowsPerRequest, resident)
			if err != nil || plan.workers != workers || plan.maximumResidentRecords != resident {
				t.Fatalf("live plan workers/resident=%d/%d err=%v", plan.workers, plan.maximumResidentRecords, err)
			}
			if _, err := NewLiveHydrationWorkerPlan(items, workers, 256, 1<<20, int64(len(items))*rowsPerRequest, resident-1); err == nil {
				t.Fatal("live plan accepted resident capacity below workers * 57,600")
			}

			var active atomic.Int64
			var maximum atomic.Int64
			allActive := make(chan struct{})
			var once sync.Once
			server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				current := active.Add(1)
				updateAtomicMaximum(&maximum, current)
				if current == int64(workers) {
					once.Do(func() { close(allActive) })
				}
				<-request.Context().Done()
				active.Add(-1)
			}))
			worker := testHydrationWorker(t, server)
			workCtx, cancelWork := context.WithCancel(context.Background())
			joined := make(chan HydrationWorkerResult, 1)
			sink := &recordingHydrationSink{}
			go func() { joined <- worker.Run(workCtx, context.Background(), plan, sink) }()
			select {
			case <-allActive:
			case <-time.After(3 * time.Second):
				t.Fatal("configured REST worker ceiling was not reached")
			}
			cancelWork()
			var result HydrationWorkerResult
			select {
			case result = <-joined:
			case <-time.After(3 * time.Second):
				t.Fatal("canceled live worker pool did not join")
			}
			server.Close()
			_, terminals := sink.snapshot()
			if maximum.Load() != int64(workers) || result.Accounting().MaximumActiveWorkers != int64(workers) ||
				result.Accounting().ProviderCanceled != int64(len(items)) || len(terminals) != len(items) {
				t.Fatalf("joined cancellation maximum=%d accounting=%+v terminals=%d", maximum.Load(), result.Accounting(), len(terminals))
			}
			seen := make(map[uint64]struct{}, len(terminals))
			for _, terminal := range terminals {
				if terminal.State() != HydrationCanceled {
					t.Fatalf("request=%d terminal=%s", terminal.WorkItem().RequestID(), terminal.State())
				}
				seen[terminal.WorkItem().RequestID()] = struct{}{}
			}
			if len(seen) != len(items) {
				t.Fatalf("unique terminal identities=%d want=%d", len(seen), len(items))
			}
		})
	}
	for _, workers := range []int{0, 3, 5, 6, 7, 9} {
		if _, err := NewLiveHydrationWorkerPlan(items, workers, 256, 1<<20, int64(len(items))*rowsPerRequest, 8*rowsPerRequest); err == nil {
			t.Fatalf("unsupported live workers=%d accepted", workers)
		}
	}
	if _, err := NewLiveHydrationWorkerPlan(items, 8, 256, 1<<20, int64(len(items))*rowsPerRequest, 8*rowsPerRequest+1); err == nil {
		t.Fatal("live plan accepted resident capacity above workers * 57,600")
	}

	t.Run("response bodies transfer concurrently", func(t *testing.T) {
		var active atomic.Int64
		var maximum atomic.Int64
		allReading := make(chan struct{})
		release := make(chan struct{})
		var once sync.Once
		payload := `{"status":"OK","ticker":"A","adjusted":false,"results":[]}`
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			symbol := strings.Split(request.URL.Path, "/")[4]
			body := strings.Replace(payload, `"A"`, fmt.Sprintf("%q", symbol), 1)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &barrierReadCloser{
				reader: strings.NewReader(body), entered: func() {
					current := active.Add(1)
					updateAtomicMaximum(&maximum, current)
					if current == int64(len(items)) {
						once.Do(func() { close(allReading) })
					}
				}, release: release, exited: func() { active.Add(-1) }}, Request: request}, nil
		})}
		worker, err := NewHydrationWorker("https://provider.invalid", func() (string, error) { return "test-token", nil }, client)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := NewLiveHydrationWorkerPlan(items, 8, 256, 1<<20, int64(len(items))*rowsPerRequest, 8*rowsPerRequest)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan HydrationWorkerResult, 1)
		go func() {
			done <- worker.Run(context.Background(), context.Background(), plan, &recordingHydrationSink{})
		}()
		select {
		case <-allReading:
		case <-time.After(3 * time.Second):
			t.Fatalf("response-body reads serialized; maximum concurrent reads=%d", maximum.Load())
		}
		close(release)
		result := <-done
		if maximum.Load() != 8 || result.Accounting().ProviderCompletedEmpty != int64(len(items)) {
			t.Fatalf("body concurrency maximum=%d accounting=%+v", maximum.Load(), result.Accounting())
		}
	})
}

type barrierReadCloser struct {
	reader          io.Reader
	entered, exited func()
	release         <-chan struct{}
	once            sync.Once
}

func (r *barrierReadCloser) Read(destination []byte) (int, error) {
	r.once.Do(func() {
		r.entered()
		<-r.release
		r.exited()
	})
	return r.reader.Read(destination)
}

func (*barrierReadCloser) Close() error { return nil }

type recordingHydrationSink struct {
	mu         sync.Mutex
	chunks     []HydrationResultChunk
	terminals  []HydrationTerminal
	afterChunk func(HydrationResultChunk)
}

func (s *recordingHydrationSink) AdmitHydrationChunk(_ context.Context, chunk HydrationResultChunk) error {
	s.mu.Lock()
	s.chunks = append(s.chunks, chunk)
	s.mu.Unlock()
	if s.afterChunk != nil {
		s.afterChunk(chunk)
	}
	return nil
}

func (s *recordingHydrationSink) AdmitHydrationTerminal(_ context.Context, terminal HydrationTerminal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.terminals = append(s.terminals, terminal)
	return nil
}

func (s *recordingHydrationSink) snapshot() ([]HydrationResultChunk, []HydrationTerminal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.chunks), slices.Clone(s.terminals)
}

type blockingHydrationSink struct {
	entered chan struct{}
	once    sync.Once
}

func (s *blockingHydrationSink) AdmitHydrationChunk(ctx context.Context, _ HydrationResultChunk) error {
	s.once.Do(func() { close(s.entered) })
	<-ctx.Done()
	return ctx.Err()
}

func (*blockingHydrationSink) AdmitHydrationTerminal(context.Context, HydrationTerminal) error {
	return nil
}

type blockingTerminalHydrationSink struct {
	entered  chan struct{}
	mu       sync.Mutex
	attempts int
	terminal HydrationTerminal
}

func (*blockingTerminalHydrationSink) AdmitHydrationChunk(context.Context, HydrationResultChunk) error {
	return nil
}

func (s *blockingTerminalHydrationSink) AdmitHydrationTerminal(ctx context.Context, terminal HydrationTerminal) error {
	s.mu.Lock()
	s.attempts++
	attempt := s.attempts
	s.mu.Unlock()
	if attempt == 1 {
		close(s.entered)
		<-ctx.Done()
		return ctx.Err()
	}
	s.mu.Lock()
	s.terminal = terminal
	s.mu.Unlock()
	return nil
}

func (s *blockingTerminalHydrationSink) admittedTerminal() HydrationTerminal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

type contextRespectingHydrationSink struct {
	recordingHydrationSink
	closed bool
}

func (s *contextRespectingHydrationSink) AdmitHydrationTerminal(ctx context.Context, terminal HydrationTerminal) error {
	if s.closed {
		return ErrHydrationInputClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.recordingHydrationSink.AdmitHydrationTerminal(ctx, terminal)
	}
}

func testHydrationWorker(t *testing.T, server *httptest.Server) *HydrationWorker {
	t.Helper()
	worker, err := NewHydrationWorker(server.URL, func() (string, error) { return "test-token", nil }, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	worker.acquisition.sleep = func(context.Context, time.Duration) error { return nil }
	return worker
}

func mustHydrationWork(t *testing.T, binding reference.Binding, generation, requestID uint64, symbol string, start, end time.Time) HydrationWorkItem {
	t.Helper()
	item, err := NewHydrationWorkItem(binding, generation, requestID, HydrationFreshStart, symbol, start, end, 0)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func mustHydrationPlan(t *testing.T, items []HydrationWorkItem, workers, rowsPerChunk int, maximumResponseBytes, maximumNormalizedRecords, maximumResidentRecords int64) HydrationWorkerPlan {
	t.Helper()
	plan, err := NewHydrationWorkerPlan(items, workers, rowsPerChunk, maximumResponseBytes, maximumNormalizedRecords, maximumResidentRecords)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
