package massive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOfflineSubsetBenchmarkSelectionMeasurementsAndClaimBoundary(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB", "CCC", "DDD", "EEE"})
	start := binding.SessionStart()
	end := start.Add(2 * time.Second)
	var servedBytes atomic.Int64
	var eeeAttempts atomic.Int64
	requested := make(map[string]int)
	var requestedMu sync.Mutex
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		symbol := strings.Split(request.URL.Path, "/")[4]
		requestedMu.Lock()
		requested[symbol]++
		requestedMu.Unlock()
		write := func(status int, body string) {
			writer.WriteHeader(status)
			_, _ = io.WriteString(writer, body)
			servedBytes.Add(int64(len(body)))
		}
		switch {
		case symbol == "AAA" && request.URL.Query().Get("cursor") == "":
			write(http.StatusOK, fmt.Sprintf(`{"status":"OK","ticker":"AAA","adjusted":false,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":10,"vw":10,"n":2}],"next_url":%q}`,
				start.UnixMilli(), server.URL+request.URL.Path+"?cursor=two"))
		case symbol == "AAA":
			write(http.StatusOK, fmt.Sprintf(`{"status":"OK","ticker":"AAA","adjusted":false,"results":[{"t":%d,"o":11,"h":12,"l":10,"c":11,"v":12,"vw":11,"n":3}]}`,
				start.Add(time.Second).UnixMilli()))
		case symbol == "CCC":
			write(http.StatusOK, `{"status":"OK","ticker":"CCC","adjusted":false,"results":[]}`)
		case symbol == "EEE" && eeeAttempts.Add(1) == 1:
			write(http.StatusServiceUnavailable, "retry")
		case symbol == "EEE":
			write(http.StatusOK, fmt.Sprintf(`{"status":"OK","ticker":"EEE","adjusted":false,"results":[{"t":%d,"o":20,"h":21,"l":19,"c":20,"v":20,"vw":20,"n":4}]}`,
				start.UnixMilli()))
		default:
			write(http.StatusBadRequest, "unexpected symbol")
		}
	}))
	defer server.Close()

	report, err := RunOfflineSubsetBenchmark(context.Background(), testOfflineDownloader(t, server), OfflineSubsetBenchmarkConfig{
		FullBinding: binding, SelectedSymbols: 3, Start: start, End: end, Workers: 2, HardTimeout: time.Second,
		MaximumNormalizedRecords: 6, MaximumResponseBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	selection := report.Selection()
	if selection.Method != OfflineSubsetSelectionMethod || selection.FullBindingIdentity != binding.Identity() ||
		selection.SubsetBindingIdentity == "" || selection.FullSymbols != 5 || selection.SelectedSymbols != 3 ||
		selection.SortedSymbolsSHA256 != benchmarkSymbolsDigest([]string{"AAA", "CCC", "EEE"}) {
		t.Fatalf("selection identity = %+v", selection)
	}
	requestedMu.Lock()
	if len(requested) != 3 || requested["AAA"] != 2 || requested["CCC"] != 1 || requested["EEE"] != 2 {
		t.Fatalf("requested symbols = %+v", requested)
	}
	requestedMu.Unlock()
	want := DownloadAccounting{PlannedSymbols: 3, CompleteSymbols: 3, NonemptySymbols: 2, EmptySymbols: 1,
		Records: 3, Pages: 4, Attempts: 5, Bytes: servedBytes.Load()}
	if report.State() != OfflineSubsetBenchmarkStateComplete || !report.Reconciles() || report.Accounting() != want {
		t.Fatalf("report state=%s reconciles=%t accounting=%+v want=%+v", report.State(), report.Reconciles(), report.Accounting(), want)
	}
	outcomes := report.Outcomes()
	if len(outcomes) != 3 || outcomes[0].Symbol != "AAA" || outcomes[0].Records != 2 || outcomes[0].Pages != 2 || outcomes[0].Attempts != 2 ||
		outcomes[1].Symbol != "CCC" || outcomes[1].Records != 0 || outcomes[1].Pages != 1 || outcomes[1].Attempts != 1 ||
		outcomes[2].Symbol != "EEE" || outcomes[2].Records != 1 || outcomes[2].Pages != 1 || outcomes[2].Attempts != 2 {
		t.Fatalf("per-symbol measurements = %+v", outcomes)
	}
	assertSubsetOnlyBenchmarkReport(t, report)
}

func TestOfflineSubsetBenchmarkTerminalFailureAndCancellationContainment(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB", "CCC", "DDD", "EEE"})
	start := binding.SessionStart()

	t.Run("terminal failure remains subset failure evidence", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			symbol := strings.Split(request.URL.Path, "/")[4]
			if symbol == "CCC" {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(writer, "terminal")
				return
			}
			fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, symbol)
		}))
		defer server.Close()
		report, err := RunOfflineSubsetBenchmark(context.Background(), testOfflineDownloader(t, server), OfflineSubsetBenchmarkConfig{
			FullBinding: binding, SelectedSymbols: 3, Start: start, End: start.Add(time.Second), Workers: 2, HardTimeout: time.Second,
			MaximumNormalizedRecords: 3, MaximumResponseBytes: 1 << 20,
		})
		if err != nil {
			t.Fatal(err)
		}
		accounting := report.Accounting()
		if report.State() != OfflineSubsetBenchmarkStateFailed || !report.Reconciles() || accounting.PlannedSymbols != 3 ||
			accounting.CompleteSymbols != 2 || accounting.FailedSymbols != 1 || accounting.CanceledSymbols != 0 ||
			accounting.NonemptySymbols != 0 || accounting.EmptySymbols != 2 || report.Outcomes()[1].Reason != DownloadReasonEnvelopeIdentityStatus {
			t.Fatalf("terminal failure report state=%s accounting=%+v outcomes=%+v", report.State(), accounting, report.Outcomes())
		}
		assertSubsetOnlyBenchmarkReport(t, report)
	})

	t.Run("hard timeout cancels in-flight and unscheduled symbols", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()
		const hardTimeout = 40 * time.Millisecond
		wallStart := time.Now()
		report, err := RunOfflineSubsetBenchmark(context.Background(), testOfflineDownloader(t, server), OfflineSubsetBenchmarkConfig{
			FullBinding: binding, SelectedSymbols: 3, Start: start, End: start.Add(time.Second), Workers: 1, HardTimeout: hardTimeout,
			MaximumNormalizedRecords: 3, MaximumResponseBytes: 1 << 20,
		})
		elapsed := time.Since(wallStart)
		if err != nil {
			t.Fatal(err)
		}
		accounting := report.Accounting()
		if report.State() != OfflineSubsetBenchmarkStateCanceled || !report.Reconciles() || accounting.PlannedSymbols != 3 ||
			accounting.CanceledSymbols != 3 || accounting.CompleteSymbols != 0 || accounting.FailedSymbols != 0 ||
			report.HardTimeout() != hardTimeout || report.WallDuration() < hardTimeout/2 || elapsed > time.Second {
			t.Fatalf("timeout report state=%s accounting=%+v wall=%s elapsed=%s", report.State(), accounting, report.WallDuration(), elapsed)
		}
		assertSubsetOnlyBenchmarkReport(t, report)
	})
}

func assertSubsetOnlyBenchmarkReport(t *testing.T, report OfflineSubsetBenchmarkReport) {
	t.Helper()
	if report.EvidenceScope() != OfflineSubsetBenchmarkScope || report.CompleteUniverse() || report.ArtifactEligible() || report.AcceptanceEligible() {
		t.Fatalf("subset claim boundary scope=%q complete=%t artifact=%t acceptance=%t", report.EvidenceScope(), report.CompleteUniverse(), report.ArtifactEligible(), report.AcceptanceEligible())
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatal(err)
	}
	if value["schema"] != OfflineSubsetBenchmarkSchema || value["evidence_scope"] != OfflineSubsetBenchmarkScope ||
		value["complete_universe"] != false || value["artifact_eligible"] != false || value["acceptance_eligible"] != false {
		t.Fatalf("serialized subset boundary = %s", encoded)
	}
	if _, ok := value["artifact_id"]; ok {
		t.Fatalf("subset report exposes artifact identity: %s", encoded)
	}
	if _, ok := value["path"]; ok {
		t.Fatalf("subset report exposes artifact path: %s", encoded)
	}
	if len(report.UnavailableMeasurements()) != 4 {
		t.Fatalf("unavailable measurements = %v", report.UnavailableMeasurements())
	}
}

func TestOfflineSubsetBenchmarkRejectsCompleteBindingScope(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB"})
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid complete-binding benchmark reached HTTPS")
	}))
	defer server.Close()
	start := binding.SessionStart()
	report, err := RunOfflineSubsetBenchmark(context.Background(), testOfflineDownloader(t, server), OfflineSubsetBenchmarkConfig{
		FullBinding: binding, SelectedSymbols: 2, Start: start, End: start.Add(time.Second), Workers: 1, HardTimeout: time.Second,
		MaximumNormalizedRecords: 2, MaximumResponseBytes: 1 << 20,
	})
	if err == nil || report.State() != OfflineSubsetBenchmarkStateInvalid || report.CompleteUniverse() || report.ArtifactEligible() || report.AcceptanceEligible() {
		t.Fatalf("complete-binding scope was not rejected: err=%v report=%+v", err, report)
	}
}
