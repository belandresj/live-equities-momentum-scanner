package massive

import (
	"context"
	"encoding/json"
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

type countingZeroReader struct{ count *atomic.Int64 }

func (reader countingZeroReader) Read(destination []byte) (int, error) {
	reader.count.Add(int64(len(destination)))
	for index := range destination {
		destination[index] = 0
	}
	return len(destination), nil
}
