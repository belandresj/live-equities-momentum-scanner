package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

func TestC10ScannerCompositionJoinsAPIAndRuntime(t *testing.T) {
	binding := scannerTestBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	startup, cancelStartup := context.WithTimeout(context.Background(), 2*time.Second)
	runtime, err := operations.New(startup, binding, config, func() time.Time { return now })
	cancelStartup()
	if err != nil {
		t.Fatal(err)
	}
	api, err := snapshotapi.Listen(runtime, snapshotapi.ServerConfig{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	liveDone := make(chan error, 1)
	liveDone <- nil
	runContext, cancelRun := context.WithCancel(context.Background())
	if err := joinAndShutdown(runtime, api, liveDone, api.Done(), cancelRun, false, false); err != nil {
		t.Fatal(err)
	}
	if runContext.Err() == nil || runtime.Status().ProcessLive {
		t.Fatal("composition returned before cancellation/runtime termination")
	}
	select {
	case _, open := <-api.Done():
		if open {
			t.Fatal("snapshot API terminal channel remained open")
		}
	default:
		t.Fatal("composition returned before snapshot API joined")
	}
	client := &http.Client{Timeout: 100 * time.Millisecond}
	if response, err := client.Get("http://" + api.Address() + "/livez"); err == nil {
		_ = response.Body.Close()
		t.Fatal("snapshot listener remained reachable after joined shutdown")
	}
}

func TestC10OriginFlagsPreserveExactRepeatedValues(t *testing.T) {
	var values originFlags
	if values.String() != "" || values.Set("http://127.0.0.1:3000") != nil || values.Set("https://scanner.example") != nil || values.String() != "http://127.0.0.1:3000,https://scanner.example" {
		t.Fatalf("origin flags = %q", values.String())
	}
}

func scannerTestBinding(t *testing.T) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-07")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "count": 1, "results": []map[string]any{{"ticker": "AAA", "active": true, "market": "stocks", "locale": "us", "type": "CS"}}})
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": 1, "results": []map[string]any{{"T": "AAA", "c": 10.0, "t": facts.PriorRegularClose.UnixMilli()}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	resolverNow := facts.SessionStart.Add(time.Hour)
	dataDir := filepath.Join(t.TempDir(), "reference")
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: dataDir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return resolverNow }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: dataDir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return resolverNow }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}
