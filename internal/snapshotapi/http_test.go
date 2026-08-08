package snapshotapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type countingCaptureSource struct {
	runtime *operations.Runtime
	calls   atomic.Uint64
}

func (source *countingCaptureSource) CaptureSnapshot() (operations.SnapshotCapture, error) {
	source.calls.Add(1)
	return source.runtime.CaptureSnapshot()
}

func TestPC10HTTPRoutesCORSAndLifecycle(t *testing.T) {
	runtime, binding, now := newSnapshotRuntime(t)
	source := &countingCaptureSource{runtime: runtime}
	handler, err := NewHandler(source, HandlerConfig{AllowedOrigins: []string{"http://127.0.0.1:3000"}})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/livez", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || source.calls.Load() != 1 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Length") != strconvLen(response.Body.Bytes()) {
		t.Fatalf("initial liveness = code %d calls %d headers=%v body=%s", response.Code, source.calls.Load(), response.Header(), response.Body.String())
	}
	var live livenessResponse
	if err := json.Unmarshal(response.Body.Bytes(), &live); err != nil || !live.ProcessLive || live.Reason != "" || live.SampleID == "0" {
		t.Fatalf("initial liveness body = %+v/%v", live, err)
	}

	beforeReady := source.calls.Load()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var readiness readinessResponse
	if response.Code != http.StatusServiceUnavailable || source.calls.Load() != beforeReady+1 || json.Unmarshal(response.Body.Bytes(), &readiness) != nil || readiness.PublicationID != nil || readiness.BindingIdentity != nil || readiness.Reason != "publication_unavailable" {
		t.Fatalf("initial readiness = code %d calls %d body=%s", response.Code, source.calls.Load(), response.Body.String())
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/api/v1/snapshot", http.StatusServiceUnavailable)
	for _, path := range []string{"/readyz", "/api/v1/snapshot"} {
		before := source.calls.Load()
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, path, nil))
		if response.Code != http.StatusServiceUnavailable || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || source.calls.Load() != before+1 {
			t.Fatalf("unavailable HEAD %s = code %d calls %d headers=%v body=%q", path, response.Code, source.calls.Load(), response.Header(), response.Body.String())
		}
	}

	before := source.calls.Load()
	for _, test := range []struct {
		request *http.Request
		status  int
		allow   string
	}{
		{httptest.NewRequest(http.MethodPost, "/livez", nil), http.StatusMethodNotAllowed, "GET, HEAD"},
		{httptest.NewRequest(http.MethodGet, "/missing", nil), http.StatusNotFound, ""},
		{httptest.NewRequest(http.MethodGet, "/livez?schema=2", nil), http.StatusBadRequest, ""},
		{httptest.NewRequest(http.MethodHead, "/missing", nil), http.StatusNotFound, ""},
	} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, test.request)
		if response.Code != test.status || response.Header().Get("Allow") != test.allow || source.calls.Load() != before ||
			response.Header().Get("Content-Length") == "" || test.request.Method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatalf("invalid route/method = %s %s code=%d allow=%q calls=%d headers=%v body=%q", test.request.Method, test.request.URL, response.Code, response.Header().Get("Allow"), source.calls.Load(), response.Header(), response.Body.String())
		}
	}
	for _, test := range []struct {
		name   string
		origin []string
	}{
		{"unlisted", []string{"https://evil.example"}},
		{"wildcard", []string{"*"}},
		{"null", []string{"null"}},
		{"malformed", []string{"://bad"}},
		{"multiple", []string{"http://127.0.0.1:3000", "http://127.0.0.1:3000"}},
	} {
		request = httptest.NewRequest(http.MethodGet, "/livez", nil)
		for _, origin := range test.origin {
			request.Header.Add("Origin", origin)
		}
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "" || response.Header().Get("Vary") != "" {
			t.Fatalf("%s origin reached capture or received CORS data: code=%d headers=%v", test.name, response.Code, response.Header())
		}
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v1/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" || response.Header().Get("Vary") != "Origin" {
		t.Fatalf("preflight = code %d calls %d headers=%v", response.Code, source.calls.Load(), response.Header())
	}
	request = httptest.NewRequest(http.MethodOptions, "/readyz", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodHead)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Methods") != "GET, HEAD" || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" {
		t.Fatalf("HEAD preflight = code %d calls %d headers=%v", response.Code, source.calls.Load(), response.Header())
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v1/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before {
		t.Fatal("invalid preflight reached capture")
	}
	request = httptest.NewRequest(http.MethodOptions, "/api/v1/snapshot", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "Authorization")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("custom-header preflight received access")
	}
	request = httptest.NewRequest(http.MethodOptions, "/missing", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || source.calls.Load() != before {
		t.Fatal("unknown-route preflight was exposed")
	}

	request = httptest.NewRequest(http.MethodHead, "/livez", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" || response.Header().Get("Vary") != "Origin" {
		t.Fatalf("HEAD parity = code %d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}

	makeSnapshotReady(t, runtime.Engine(), binding, *now)
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/readyz", http.StatusOK)
	for _, path := range []string{"/readyz", "/api/v1/snapshot"} {
		before := source.calls.Load()
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, path, nil))
		if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Length") == "0" || source.calls.Load() != before+1 {
			t.Fatalf("ready HEAD %s = code %d calls %d headers=%v body=%q", path, response.Code, source.calls.Load(), response.Header(), response.Body.String())
		}
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil)
	response = httptest.NewRecorder()
	before = source.calls.Load()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || source.calls.Load() != before+1 {
		t.Fatalf("snapshot = code %d calls %d body=%s", response.Code, source.calls.Load(), response.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil || snapshot.SchemaVersion != SchemaVersion || snapshot.Publication.ID == "0" || snapshot.Rows == nil {
		t.Fatalf("snapshot body = %+v/%v", snapshot, err)
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := runtime.Shutdown(shutdown); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	request = httptest.NewRequest(http.MethodGet, "/livez", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || json.Unmarshal(response.Body.Bytes(), &live) != nil || live.ProcessLive || live.Reason != "runtime_unavailable" {
		t.Fatalf("terminal liveness = code %d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || json.Unmarshal(response.Body.Bytes(), &readiness) != nil || readiness.PublicationID == nil || readiness.BackendReady || readiness.Reason != "runtime_unavailable" {
		t.Fatalf("terminal readiness = code %d body=%s", response.Code, response.Body.String())
	}
	assertStatusAndOneCapture(t, handler, source, http.MethodGet, "/api/v1/snapshot", http.StatusOK)
}

func TestPC10HTTPBoundsLoopbackCancellationAndProgress(t *testing.T) {
	var nilSource *countingCaptureSource
	if _, err := NewHandler(nilSource, HandlerConfig{}); err == nil {
		t.Fatal("typed-nil source configured")
	}
	if _, err := NewHandler(&countingCaptureSource{}, HandlerConfig{AllowedOrigins: []string{"null"}}); err == nil {
		t.Fatal("null origin configured")
	}
	if _, err := NewHandler(&countingCaptureSource{}, HandlerConfig{AllowedOrigins: []string{"http://127.0.0.1:99999"}}); err == nil {
		t.Fatal("invalid origin port configured")
	}
	for _, address := range []string{"0.0.0.0:0", "localhost:0", "192.0.2.1:8080", ":8080"} {
		if loopbackAddress(address) {
			t.Fatalf("non-explicit-loopback address accepted: %s", address)
		}
	}
	if !loopbackAddress("127.0.0.1:0") || !loopbackAddress("[::1]:8080") {
		t.Fatal("loopback address rejected")
	}
	if _, err := marshalBounded(strings.Repeat("x", maximumResponseBytes)); err == nil {
		t.Fatal("over-bound encoded response accepted")
	}

	runtime, binding, now := newSnapshotRuntime(t)
	makeSnapshotReady(t, runtime.Engine(), binding, *now)
	t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
	source := &countingCaptureSource{runtime: runtime}
	server, err := Listen(source, ServerConfig{Address: "127.0.0.1:0", AllowedOrigins: []string{"http://127.0.0.1:3000"}})
	if err != nil {
		t.Fatal(err)
	}
	if server.http.ReadHeaderTimeout != 5*time.Second || server.http.ReadTimeout != 5*time.Second || server.http.WriteTimeout != 10*time.Second || server.http.IdleTimeout != 30*time.Second || server.http.MaxHeaderBytes != 16<<10 {
		t.Fatalf("server bounds = %+v", server.http)
	}
	response, err := http.Get("http://" + server.Address() + "/api/v1/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || len(body) == 0 || len(body) > maximumResponseBytes {
		t.Fatalf("real loopback response = status %d bytes %d err %v", response.StatusCode, len(body), readErr)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	failures := make(chan string, 33)
	var concurrent sync.WaitGroup
	concurrent.Add(1)
	go func() {
		defer concurrent.Done()
		for i := 0; i < 16; i++ {
			admission, completion := runtime.Engine().AdmitTimer(context.Background())
			if admission != engine.AdmissionAdmitted || completion == nil {
				failures <- "concurrent timer not admitted"
				return
			}
			if disposition := <-completion; disposition.Code != engine.DispositionTimerApplied {
				failures <- "concurrent timer not applied"
				return
			}
		}
	}()
	for i := 0; i < 16; i++ {
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			response, err := client.Get("http://" + server.Address() + "/api/v1/snapshot")
			if err != nil {
				failures <- "concurrent snapshot request failed"
				return
			}
			defer response.Body.Close()
			var snapshot Snapshot
			if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&snapshot) != nil || snapshot.Publication.ID == "0" || snapshot.Status.TQPressureMode != snapshot.TQ.PressureMode {
				failures <- "concurrent snapshot was incoherent"
			}
		}()
	}
	concurrentDone := make(chan struct{})
	go func() { concurrent.Wait(); close(concurrentDone) }()
	select {
	case <-concurrentDone:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent publication reads timed out")
	}
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
	client.CloseIdleConnections()
	http.DefaultClient.CloseIdleConnections()
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := server.Shutdown(shutdown); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-server.Done():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not join")
	}

	handler, err := NewHandler(source, HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()
	before := source.calls.Load()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/livez", nil).WithContext(canceled))
	if source.calls.Load() != before {
		t.Fatal("canceled client reached capture")
	}

	blocked := newBlockingWriter()
	handlerDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil))
		close(handlerDone)
	}()
	select {
	case <-blocked.started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not reach blocked client")
	}
	admission, completion := runtime.Engine().AdmitTimer(context.Background())
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("engine progress was blocked by HTTP writer")
	}
	select {
	case <-completion:
	case <-time.After(2 * time.Second):
		t.Fatal("engine transition blocked behind HTTP writer")
	}
	close(blocked.release)
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("blocked handler did not finish")
	}
}

func assertStatusAndOneCapture(t *testing.T, handler http.Handler, source *countingCaptureSource, method, path string, status int) {
	t.Helper()
	before := source.calls.Load()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != status || source.calls.Load() != before+1 {
		t.Fatalf("%s %s = code %d calls %d body=%s", method, path, response.Code, source.calls.Load(), response.Body.String())
	}
}

func newSnapshotRuntime(t *testing.T) (*operations.Runtime, reference.Binding, *time.Time) {
	t.Helper()
	binding := snapshotBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runtime, err := operations.New(ctx, binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return runtime, binding, &now
}

func makeSnapshotReady(t *testing.T, owner *engine.Engine, binding reference.Binding, at time.Time) {
	t.Helper()
	ackAt := at.Add(-operations.DefaultConfig().EvaluationDelay)
	applySnapshotControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applySnapshotControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 57_600, MaximumResidentRecords: 57_600}
	admission, completion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatal("hydration plan not admitted")
	}
	plan := <-completion
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		terminal, err := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, terminalCompletion := owner.AdmitHydrationTerminal(context.Background(), terminal)
		result := <-terminalCompletion
		if result.Code != engine.DispositionHydrationTerminalApplied {
			t.Fatalf("hydration terminal = %+v", result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("hydration produced no fence")
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 1, 1, at)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := owner.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if result := <-fenceCompletion; result.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("hydration fence = %+v", result)
	}
}

func applySnapshotControl(t *testing.T, owner *engine.Engine, binding reference.Binding, kind engine.ConnectionControlKind, epoch, token uint64, position engine.LivePosition, at time.Time) {
	t.Helper()
	admission, completion := owner.AdmitConnectionControl(context.Background(), engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind, ConnectionEpoch: epoch, CommandToken: token, Position: position, ReceiptTime: at, Outcome: engine.ControlSucceeded})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("control %s not admitted", kind)
	}
	if result := <-completion; result.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("control %s = %+v", kind, result)
	}
}

func shutdownSnapshotRuntime(t *testing.T, runtime *operations.Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Errorf("runtime shutdown: %v", err)
	}
}

func strconvLen(value []byte) string { return strconv.Itoa(len(value)) }

type blockingWriter struct {
	header  http.Header
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{header: make(http.Header), started: make(chan struct{}), release: make(chan struct{})}
}

func (writer *blockingWriter) Header() http.Header { return writer.header }
func (writer *blockingWriter) WriteHeader(int)     {}
func (writer *blockingWriter) Write(value []byte) (int, error) {
	writer.once.Do(func() { close(writer.started) })
	<-writer.release
	return len(value), nil
}
