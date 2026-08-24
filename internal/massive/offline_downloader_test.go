package massive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// TestOfflineDownloaderContract is P-C4-DL. It proves fake-provider request,
// containment, terminal accounting, and bounds; it makes no live SLA claim.
func TestOfflineDownloaderContract(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB"})
	start := binding.SessionStart()
	end := start.Add(4 * time.Second)
	t.Run("fixed query pagination sparse and successful empty", func(t *testing.T) {
		var requests atomic.Int64
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests.Add(1)
			if request.Header.Get("Authorization") != "Bearer test-token" || request.URL.Query().Get("apiKey") != "" ||
				request.URL.Query().Get("adjusted") != "false" || request.URL.Query().Get("sort") != "asc" || request.URL.Query().Get("limit") != "50000" {
				t.Errorf("request headers/query = %q %s", request.Header.Get("Authorization"), request.URL.RawQuery)
			}
			parts := strings.Split(request.URL.Path, "/")
			symbol := parts[4]
			wantSuffix := fmt.Sprintf("/range/1/second/%d/%d", start.UnixMilli(), end.Add(-time.Millisecond).UnixMilli())
			if !strings.HasSuffix(request.URL.Path, wantSuffix) {
				t.Errorf("request path = %q", request.URL.Path)
			}
			if symbol == "AAA" && request.URL.Query().Get("cursor") == "" {
				fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"count":1,"queryCount":1,"resultsCount":1,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10.5,"v":100.5,"vw":10.25,"n":3}],"next_url":%q}`,
					start.UnixMilli(), server.URL+request.URL.Path+"?cursor=two&apiKey=must-strip")
				return
			}
			fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"count":0,"results":[]}`, symbol)
		}))
		defer server.Close()
		downloader := testOfflineDownloader(t, server)
		result := downloader.Download(context.Background(), DownloadPlan{binding, start, end, 2, 8, 1 << 20})
		if !result.Complete() || requests.Load() != 3 || result.Accounting() != (DownloadAccounting{
			PlannedSymbols: 2, CompleteSymbols: 2, NonemptySymbols: 1, EmptySymbols: 1,
			Records: 1, Pages: 3, Attempts: 3, Bytes: result.Accounting().Bytes,
		}) {
			t.Fatalf("download result complete=%t requests=%d accounting=%+v outcomes=%+v", result.Complete(), requests.Load(), result.Accounting(), result.Outcomes())
		}
		records := result.Records()
		if len(records) != 1 || records[0].Symbol != "AAA" || records[0].WindowStart != start || records[0].Values.AverageTradeSize != 33 {
			t.Fatalf("normalized records = %+v", records)
		}
	})

	t.Run("bounded retries", func(t *testing.T) {
		single := component4TestBinding(t, []string{"AAA"})
		var attempts atomic.Int64
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) < 3 {
				http.Error(writer, "retry", http.StatusServiceUnavailable)
				return
			}
			io.WriteString(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`)
		}))
		defer server.Close()
		downloader := testOfflineDownloader(t, server)
		result := downloader.Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if !result.Complete() || attempts.Load() != OfflineAttemptLimit || result.Accounting().Attempts != OfflineAttemptLimit {
			t.Fatalf("retry result = %+v attempts=%d", result, attempts.Load())
		}
	})

	t.Run("error foreign and truncated evidence never becomes empty", func(t *testing.T) {
		single := component4TestBinding(t, []string{"AAA"})
		bodies := []string{
			`{"status":"ERROR","ticker":"AAA","adjusted":false,"results":[]}`,
			`{"status":"OK","ticker":"FOREIGN","adjusted":false,"results":[]}`,
			`{"status":"OK","ticker":"AAA","adjusted":false,"count":1,"results":[]}`,
			`{"status":"OK","ticker":"AAA","adjusted":false,"count":"0","results":[]}`,
			`{"status":"OK","ticker":"AAA","adjusted":false,"results":[]} {}`,
			`{"status":"OK","status":"OK","ticker":"AAA","adjusted":false,"results":[]}`,
		}
		for index, body := range bodies {
			t.Run(fmt.Sprintf("case-%d", index), func(t *testing.T) {
				server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { io.WriteString(writer, body) }))
				defer server.Close()
				result := testOfflineDownloader(t, server).Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
				if result.Complete() || result.Accounting().FailedSymbols != 1 || result.Accounting().EmptySymbols != 0 {
					t.Fatalf("false empty success: %+v", result)
				}
			})
		}
	})

	t.Run("redirect foreign continuation third page and cancellation", func(t *testing.T) {
		single := component4TestBinding(t, []string{"AAA"})
		var targetRequests atomic.Int64
		target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetRequests.Add(1) }))
		defer target.Close()
		redirect := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, target.URL, http.StatusFound)
		}))
		defer redirect.Close()
		result := testOfflineDownloader(t, redirect).Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if result.Complete() || targetRequests.Load() != 0 || result.Outcomes()[0].Reason != DownloadReasonRedirectContinuation {
			t.Fatalf("redirect containment = %+v target=%d", result, targetRequests.Load())
		}

		foreign := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[],"next_url":%q}`, target.URL+request.URL.Path)
		}))
		defer foreign.Close()
		result = testOfflineDownloader(t, foreign).Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if result.Complete() || targetRequests.Load() != 0 || result.Outcomes()[0].Reason != DownloadReasonRedirectContinuation {
			t.Fatalf("foreign continuation containment = %+v target=%d", result, targetRequests.Load())
		}

		var cyclic *httptest.Server
		cyclic = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[],"next_url":%q}`, cyclic.URL+request.URL.RequestURI())
		}))
		defer cyclic.Close()
		result = testOfflineDownloader(t, cyclic).Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if result.Complete() || result.Accounting().Pages != 1 || result.Outcomes()[0].Reason != DownloadReasonRedirectContinuation {
			t.Fatalf("cyclic continuation containment = %+v", result)
		}

		var pages atomic.Int64
		var looping *httptest.Server
		looping = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			pages.Add(1)
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[],"next_url":%q}`, looping.URL+request.URL.Path+fmt.Sprintf("?cursor=%d", pages.Load()))
		}))
		defer looping.Close()
		result = testOfflineDownloader(t, looping).Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if result.Complete() || pages.Load() != OfflinePageLimit || result.Outcomes()[0].Reason != DownloadReasonRedirectContinuation {
			t.Fatalf("page bound = %+v pages=%d", result, pages.Load())
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result = testOfflineDownloader(t, redirect).Download(ctx, DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if result.Accounting().CanceledSymbols != 1 || result.Accounting().PlannedSymbols != 1 || result.Accounting().FailedSymbols != 0 {
			t.Fatalf("cancellation accounting = %+v", result.Accounting())
		}
	})

	t.Run("in-flight cancellation stops later symbol scheduling", func(t *testing.T) {
		started := make(chan struct{})
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			close(started)
			<-request.Context().Done()
		}))
		defer server.Close()
		downloader := testOfflineDownloader(t, server)
		ctx, cancel := context.WithCancel(context.Background())
		finished := make(chan DownloadResult, 1)
		go func() {
			finished <- downloader.Download(ctx, DownloadPlan{binding, start, start.Add(time.Second), 1, 2, 1 << 20})
		}()
		<-started
		cancel()
		result := <-finished
		if result.Complete() || result.Accounting().PlannedSymbols != 2 || result.Accounting().CanceledSymbols != 2 ||
			result.Accounting().CompleteSymbols != 0 || result.Accounting().FailedSymbols != 0 {
			t.Fatalf("in-flight cancellation accounting = %+v outcomes=%+v", result.Accounting(), result.Outcomes())
		}
	})

	t.Run("attempt deadline and body bound", func(t *testing.T) {
		single := component4TestBinding(t, []string{"AAA"})
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			deadline, ok := request.Context().Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > OfflineRequestDeadline+time.Second {
				t.Fatalf("attempt deadline = %v %s", ok, time.Until(deadline))
			}
			body := `{"status":"OK","ticker":"AAA","adjusted":false,"results":[]}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
		})}
		downloader, err := NewOfflineDownloader("https://provider.invalid", func() (string, error) { return "test-token", nil }, client)
		if err != nil {
			t.Fatal(err)
		}
		result := downloader.Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, 1 << 20})
		if !result.Complete() {
			t.Fatalf("deadline test result = %+v", result)
		}

		client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(io.LimitReader(zeroReader{}, OfflinePageByteLimit+1)), Request: request}, nil
		})
		result = downloader.Download(context.Background(), DownloadPlan{single, single.SessionStart(), single.SessionStart().Add(time.Second), 1, 1, OfflinePageByteLimit + 1})
		if result.Complete() || result.Outcomes()[0].Reason != DownloadReasonResponseSizeSyntax {
			t.Fatalf("body bound = %+v", result)
		}
	})

	t.Run("global response budget saturates once across workers", func(t *testing.T) {
		var consumed atomic.Int64
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(countingZeroReader{count: &consumed}), Request: request}, nil
		})}
		downloader, err := NewOfflineDownloader("https://provider.invalid", func() (string, error) { return "test-token", nil }, client)
		if err != nil {
			t.Fatal(err)
		}
		const budget = int64(32)
		result := downloader.Download(context.Background(), DownloadPlan{binding, start, start.Add(time.Second), 2, 2, budget})
		if result.Complete() || result.Accounting().FailedSymbols != 2 || consumed.Load() > budget+1 {
			t.Fatalf("global response budget result=%+v consumed=%d", result, consumed.Load())
		}
	})

	t.Run("budget proof byte never enters terminal accounting", func(t *testing.T) {
		budget := &responseBudget{maximum: 4}
		body, err := budget.read(strings.NewReader("12345"))
		if !errors.Is(err, errResponseBudget) || string(body) != "1234" || budget.used != budget.maximum {
			t.Fatalf("budget crossing body=%q used=%d/%d err=%v", body, budget.used, budget.maximum, err)
		}
	})
}

func testOfflineDownloader(t *testing.T, server *httptest.Server) *OfflineDownloader {
	t.Helper()
	downloader, err := NewOfflineDownloader(server.URL, func() (string, error) { return "test-token", nil }, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	downloader.acquisition.sleep = func(context.Context, time.Duration) error { return nil }
	return downloader
}

func component4TestBinding(t testing.TB, symbols []string) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-06")
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			offset, _ := strconv.Atoi(request.URL.Query().Get("cursor"))
			end := min(offset+1000, len(symbols))
			records := make([]map[string]any, end-offset)
			for index, symbol := range symbols[offset:end] {
				records[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			response := map[string]any{"status": "OK", "count": len(records), "results": records}
			if end < len(symbols) {
				query := request.URL.Query()
				query.Del("apiKey")
				query.Set("cursor", strconv.Itoa(end))
				response["next_url"] = server.URL + request.URL.Path + "?" + query.Encode()
			}
			json.NewEncoder(writer).Encode(response)
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			rows := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				rows[index] = map[string]any{"T": symbol, "c": 10 + index, "t": facts.PriorRegularClose.UnixMilli()}
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(rows), "results": rows})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "reference-test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "reference-test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil || !slices.Equal(binding.UniverseSymbols(), symbols) {
		t.Fatalf("test binding: %v symbols=%v", err, binding.UniverseSymbols())
	}
	return binding
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type zeroReader struct{}

func (zeroReader) Read(destination []byte) (int, error) {
	for index := range destination {
		destination[index] = 0
	}
	return len(destination), nil
}

type countingZeroReader struct{ count *atomic.Int64 }

func (reader countingZeroReader) Read(destination []byte) (int, error) {
	reader.count.Add(int64(len(destination)))
	for index := range destination {
		destination[index] = 0
	}
	return len(destination), nil
}
