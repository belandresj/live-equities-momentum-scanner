package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const syntheticCredential = "synthetic-test-credential"

func TestUniverseProviderPaginationAndNormalization(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	var server *httptest.Server
	var pages atomic.Int64
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v3/reference/tickers" {
			t.Errorf("path = %q", request.URL.Path)
			http.NotFound(writer, request)
			return
		}
		if values := request.URL.Query()["apiKey"]; len(values) != 1 || values[0] != syntheticCredential {
			t.Errorf("synthetic credential missing or duplicated in request")
		}
		page := pages.Add(1)
		if page == 1 {
			assertQuery(t, request, map[string]string{
				"date": "2026-07-29", "active": "true", "market": "stocks",
				"limit": "1000", "sort": "ticker", "order": "asc",
			})
			writePage(t, writer, []tickerRecord{
				{Ticker: "aaa", Active: true, Market: "stocks", Locale: "us", Type: "ADRC"},
				{Ticker: "OLD", Active: false, Market: "otc", Locale: "ca", Type: "ETF"},
				{Ticker: "OTC", Active: true, Market: "otc", Locale: "ca", Type: "ETF"},
			}, server.URL+"/v3/reference/tickers?cursor=second")
			return
		}
		writePage(t, writer, []tickerRecord{
			{Ticker: "FOREIGN", Active: true, Market: "stocks", Locale: "ca", Type: "ETF"},
			{Ticker: "FUND", Active: true, Market: "stocks", Locale: "us", Type: "ETF"},
			{Ticker: "ZZZ", Active: true, Market: "stocks", Locale: "us", Type: "CS"},
		}, "")
	}))
	defer server.Close()

	resolver := testResolver(t, schedule, server)
	universe, err := resolver.Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	if pages.Load() != 2 || universe.Source() != SourceFresh || !universe.IsCurrent() {
		t.Fatalf("source/pages = %s/%d", universe.Source(), pages.Load())
	}
	if got := universe.Symbols(); !slices.Equal(got, []string{"ZZZ", "aaa"}) {
		t.Fatalf("symbols = %q", got)
	}
	wantAccounting := Accounting{
		RawReferenceRecords: 6, EligibleRecords: 2, InactiveRecords: 1,
		WrongMarketRecords: 1, WrongLocaleRecords: 1, IneligibleTypeRecords: 1,
	}
	if universe.Accounting() != wantAccounting || !universe.Accounting().valid() {
		t.Fatalf("accounting = %+v", universe.Accounting())
	}
	if universe.ReferenceDate() != facts.TradingDate ||
		universe.PolicyVersion() != EligibilityPolicyVersion ||
		!strings.HasPrefix(universe.Identity(), universeIdentitySchema+":") {
		t.Fatalf("universe metadata = date %s policy %s identity %s", universe.ReferenceDate(), universe.PolicyVersion(), universe.Identity())
	}
	returned := universe.Symbols()
	returned[0] = "MUTATED"
	if universe.Symbols()[0] != "ZZZ" {
		t.Fatal("returned symbol slice mutated immutable universe")
	}

	cachePath := filepath.Join(resolver.DataDir, "universe", "2026-07-29.json")
	info, err := validatePrivateRegularFile(cachePath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %v, %v", info, err)
	}
}

func TestUniverseRejectsPartialAndAmbiguousProviderResults(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*httptest.Server) http.Handler
	}{
		{
			name: "pagination cycle",
			handler: func(server *httptest.Server) http.Handler {
				var page atomic.Int64
				return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					current := page.Add(1)
					writePage(t, writer, []tickerRecord{{Ticker: fmt.Sprintf("S%d", current), Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, server.URL+"/v3/reference/tickers?cursor=same")
				})
			},
		},
		{
			name: "changed origin",
			handler: func(*httptest.Server) http.Handler {
				return onePageThen(t, "https://other.invalid/v3/reference/tickers?cursor=x")
			},
		},
		{
			name: "conflicting date",
			handler: func(server *httptest.Server) http.Handler {
				return onePageThen(t, server.URL+"/v3/reference/tickers?date=2026-07-28")
			},
		},
		{
			name: "embedded credential",
			handler: func(server *httptest.Server) http.Handler {
				return onePageThen(t, server.URL+"/v3/reference/tickers?apiKey=forbidden")
			},
		},
		{
			name: "duplicate exact identity",
			handler: func(*httptest.Server) http.Handler {
				return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					writePage(t, writer, []tickerRecord{
						{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "CS"},
						{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "ETF"},
					}, "")
				})
			},
		},
		{
			name: "incomplete page",
			handler: func(*httptest.Server) http.Handler {
				return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					_, _ = writer.Write([]byte(`{"results":null}`))
				})
			},
		},
		{
			name: "oversized page",
			handler: func(*httptest.Server) http.Handler {
				records := make([]tickerRecord, maximumPageRecords+1)
				for index := range records {
					records[index] = tickerRecord{Ticker: fmt.Sprintf("S%04d", index), Active: true, Market: "stocks", Locale: "us", Type: "CS"}
				}
				return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writePage(t, writer, records, "") })
			},
		},
		{
			name: "empty eligible population",
			handler: func(*httptest.Server) http.Handler {
				return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					writePage(t, writer, []tickerRecord{{Ticker: "ETF", Active: true, Market: "stocks", Locale: "us", Type: "ETF"}}, "")
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schedule := loadSchedule(t)
			facts := scheduleFacts(t, schedule, "2026-07-29")
			var server *httptest.Server
			var handler http.Handler
			server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				handler.ServeHTTP(writer, request)
			}))
			handler = test.handler(server)
			defer server.Close()
			resolver := testResolver(t, schedule, server)
			if _, err := resolver.Resolve(context.Background(), facts); err == nil {
				t.Fatal("ambiguous or incomplete universe was accepted")
			}
			entries, _ := os.ReadDir(filepath.Join(resolver.DataDir, "universe"))
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".json") {
					t.Fatal("failed partial result published a cache")
				}
			}
		})
	}

	t.Run("redirect is not followed", func(t *testing.T) {
		var redirected atomic.Int64
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/redirected" {
				redirected.Add(1)
				writePage(t, writer, []tickerRecord{{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, "")
				return
			}
			http.Redirect(writer, request, "/redirected", http.StatusFound)
		}))
		defer server.Close()
		schedule := loadSchedule(t)
		resolver := testResolver(t, schedule, server)
		if _, err := resolver.Resolve(context.Background(), scheduleFacts(t, schedule, "2026-07-29")); err == nil {
			t.Fatal("redirect accepted")
		}
		if redirected.Load() != 0 {
			t.Fatal("HTTP client followed redirect")
		}
	})
}

func TestUniverseProviderStructuralClassification(t *testing.T) {
	validRecord := `{"ticker":"AAA","active":true,"market":"stocks","locale":"us","type":"CS"}`
	tests := []struct {
		name string
		body string
	}{
		{"non-OK status", `{"status":"ERROR","count":1,"results":[` + validRecord + `]}`},
		{"conflicting status", `{"status":"OK","status":"ERROR","count":1,"results":[` + validRecord + `]}`},
		{"conflicting count", `{"status":"OK","count":1,"count":2,"results":[` + validRecord + `]}`},
		{"count mismatch", `{"status":"OK","count":2,"results":[` + validRecord + `]}`},
		{"duplicate results", `{"status":"OK","results":[` + validRecord + `],"results":[` + validRecord + `]}`},
		{"ambiguous ticker", `{"status":"OK","results":[{"ticker":"AAA","ticker":"BBB","active":true,"market":"stocks","locale":"us","type":"CS"}]}`},
		{"ambiguous eligibility", `{"status":"OK","results":[{"ticker":"AAA","active":true,"active":false,"market":"stocks","locale":"us","type":"CS"}]}`},
		{"missing eligibility field", `{"status":"OK","results":[{"ticker":"AAA","active":true,"market":"stocks","locale":"us"}]}`},
		{"malformed JSON", `{"status":"OK","results":[` + validRecord},
		{"trailing JSON", `{"status":"OK","results":[` + validRecord + `]} {}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schedule := loadSchedule(t)
			facts := scheduleFacts(t, schedule, "2026-07-29")
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			_, err := testResolver(t, schedule, server).Resolve(context.Background(), facts)
			diagnostics := acquisitionErrorDiagnostics(t, err)
			if diagnostics != (AcquisitionDiagnostics{Source: SourceNone, RequestCount: 1, PageCount: 1, AttemptCount: 1, TerminalReason: TerminalReasonSourceAmbiguous}) {
				t.Fatalf("diagnostics = %+v", diagnostics)
			}
		})
	}

	t.Run("documented optional status and count", func(t *testing.T) {
		schedule := loadSchedule(t)
		facts := scheduleFacts(t, schedule, "2026-07-29")
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(`{"results":[` + validRecord + `]}`))
		}))
		defer server.Close()
		got, err := testResolver(t, schedule, server).Resolve(context.Background(), facts)
		if err != nil || !slices.Equal(got.Symbols(), []string{"AAA"}) || got.Diagnostics().TerminalReason != TerminalReasonNone {
			t.Fatalf("optional envelope evidence = %+v, %v", got, err)
		}
	})
}

func TestUniverseAccountingIdentityPermutationAndCacheCurrentness(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	pagesA := [][]tickerRecord{
		{{Ticker: "aaa", Active: true, Market: "stocks", Locale: "us", Type: "ADRC"}, {Ticker: "ETF", Active: true, Market: "stocks", Locale: "us", Type: "ETF"}},
		{{Ticker: "ZZZ", Active: true, Market: "stocks", Locale: "us", Type: "CS"}, {Ticker: "OLD", Active: false, Market: "stocks", Locale: "us", Type: "CS"}},
	}
	pagesB := [][]tickerRecord{
		{{Ticker: "OLD", Active: false, Market: "stocks", Locale: "us", Type: "CS"}, {Ticker: "ZZZ", Active: true, Market: "stocks", Locale: "us", Type: "CS"}},
		{{Ticker: "ETF", Active: true, Market: "stocks", Locale: "us", Type: "ETF"}, {Ticker: "aaa", Active: true, Market: "stocks", Locale: "us", Type: "ADRC"}},
	}

	resolvePages := func(t *testing.T, pages [][]tickerRecord) (Universe, *Resolver, *httptest.Server) {
		t.Helper()
		var server *httptest.Server
		var page atomic.Int64
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			index := int(page.Add(1) - 1)
			next := ""
			if index+1 < len(pages) {
				next = fmt.Sprintf("%s/v3/reference/tickers?cursor=%d", server.URL, index+1)
			}
			writePage(t, writer, pages[index], next)
		}))
		resolver := testResolver(t, schedule, server)
		universe, err := resolver.Resolve(context.Background(), facts)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		return universe, resolver, server
	}

	first, firstResolver, firstServer := resolvePages(t, pagesA)
	second, _, secondServer := resolvePages(t, pagesB)
	defer secondServer.Close()
	if first.Identity() != second.Identity() || !slices.Equal(first.Symbols(), second.Symbols()) || first.Accounting() != second.Accounting() {
		t.Fatalf("permutation changed normalized result: first=%+v second=%+v", first, second)
	}
	var attempts atomic.Int64
	retryServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(writer, "retry", http.StatusInternalServerError)
			return
		}
		writePage(t, writer, append(slices.Clone(pagesA[1]), pagesA[0]...), "")
	}))
	retried, err := testResolver(t, schedule, retryServer).Resolve(context.Background(), facts)
	retryServer.Close()
	if err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 || retried.Identity() != first.Identity() || retried.Accounting() != first.Accounting() {
		t.Fatalf("retry changed normalized result: attempts=%d identity=%s accounting=%+v", attempts.Load(), retried.Identity(), retried.Accounting())
	}
	firstServer.Close()
	cached, err := firstResolver.Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Source() != SourceCurrentCache || !cached.IsCurrent() || cached.Identity() != first.Identity() {
		t.Fatalf("same-date cache = source %s current %t identity %s", cached.Source(), cached.IsCurrent(), cached.Identity())
	}

	dateIdentity, _ := universeIdentity("2026-07-30", EligibilityPolicyVersion, first.Symbols())
	policyIdentity, _ := universeIdentity(facts.TradingDate, EligibilityPolicyVersion+"-changed", first.Symbols())
	symbolIdentity, _ := universeIdentity(facts.TradingDate, EligibilityPolicyVersion, []string{"ZZZ", "aab"})
	for name, identity := range map[string]string{"date": dateIdentity, "policy": policyIdentity, "symbol": symbolIdentity} {
		if identity == first.Identity() {
			t.Fatalf("%s change did not change universe identity", name)
		}
	}

	priorFacts := scheduleFacts(t, schedule, "2026-07-30")
	priorOnly, err := firstResolver.Resolve(context.Background(), priorFacts)
	if err != nil {
		t.Fatal(err)
	}
	if priorOnly.Source() != SourceObservablePriorCache || priorOnly.IsCurrent() ||
		priorOnly.CacheAgeDays() != 1 || priorOnly.ReferenceDate() != facts.TradingDate {
		t.Fatalf("prior-date cache = source %s current %t age %d date %s", priorOnly.Source(), priorOnly.IsCurrent(), priorOnly.CacheAgeDays(), priorOnly.ReferenceDate())
	}
}

func loadSchedule(t *testing.T) *session.Schedule {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	return schedule
}

func scheduleFacts(t *testing.T, schedule *session.Schedule, date string) session.Facts {
	t.Helper()
	facts, err := schedule.ForTradingDate(date)
	if err != nil {
		t.Fatal(err)
	}
	return facts
}

func testResolver(t *testing.T, schedule *session.Schedule, server *httptest.Server) *Resolver {
	t.Helper()
	return &Resolver{
		BaseURL: server.URL, APIKey: syntheticCredential, DataDir: filepath.Join(canonicalTempDir(t), "reference"),
		HTTPClient: server.Client(), Schedule: schedule,
		Now:   func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) },
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func assertQuery(t *testing.T, request *http.Request, expected map[string]string) {
	t.Helper()
	for key, want := range expected {
		if got := request.URL.Query().Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
}

func writePage(t *testing.T, writer http.ResponseWriter, records []tickerRecord, next string) {
	t.Helper()
	if err := json.NewEncoder(writer).Encode(struct {
		Results []tickerRecord `json:"results"`
		NextURL string         `json:"next_url,omitempty"`
	}{records, next}); err != nil {
		t.Errorf("encode page: %v", err)
	}
}

func onePageThen(t *testing.T, next string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writePage(t, writer, []tickerRecord{{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, next)
	})
}
