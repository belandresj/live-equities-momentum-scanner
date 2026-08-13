package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

func TestProductionLiveQueueUsesOwnerSelectedRetryHeadroom(t *testing.T) {
	config := productionLiveQueueConfig()
	if config.FrameSlots != 32768 || config.TotalFrameBytes != 128<<20 || config.MaxFrameBytes != 8<<20 {
		t.Fatalf("production live queue=%+v", config)
	}
	if config.FrameSlots != massive.MaximumLiveFrameSlots || config.TotalFrameBytes != massive.MaximumLiveQueueBytes {
		t.Fatalf("production live queue=%+v", config)
	}
	if attempts := operations.DefaultConfig().RecoveryAttempts; attempts != 5 {
		t.Fatalf("production recovery attempts=%d", attempts)
	}
}

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

func TestSnapshotMappingFailureEncoding(t *testing.T) {
	failure := snapshotapi.MappingFailure{Invariant: "population_mark_identity", Route: "/readyz", PublicationID: "81",
		LastEngineSequence: "144", Lifecycle: "live", RankingMode: "degraded_bootstrap"}
	var output bytes.Buffer
	if err := encodeSnapshotMappingFailure(json.NewEncoder(&output), failure); err != nil {
		t.Fatal(err)
	}
	var record struct {
		SnapshotMappingFailure snapshotapi.MappingFailure `json:"snapshot_mapping_failure"`
	}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil || record.SnapshotMappingFailure != failure {
		t.Fatalf("encoded mapper failure=%s decoded=%+v err=%v", output.String(), record, err)
	}
}

func TestC12RunModeConfigurationIsMutuallyExclusive(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "")
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{"unknown mode", []string{"--run-mode=paper", "--trading-date=2026-08-07"}, "live mode"},
		{"replay missing bounds", []string{"--run-mode=replay", "--replay-artifact=/private/missing"}, "requires artifact"},
		{"replay trading date", []string{"--run-mode=replay", "--replay-artifact=/private/missing", "--observation-start=09:30:00", "--observation-end=09:35:00", "--trading-date=2026-08-07"}, "live-only"},
		{"replay checkpoint", []string{"--run-mode=replay", "--replay-artifact=/private/missing", "--observation-start=09:30:00", "--observation-end=09:35:00", "--checkpoint-dir=/tmp/checkpoints"}, "live-only"},
		{"replay checkpoint mode", []string{"--run-mode=replay", "--replay-artifact=/private/missing", "--observation-start=09:30:00", "--observation-end=09:35:00", "--checkpoint-mode=off"}, "live-only"},
		{"replay hydration workers", []string{"--run-mode=replay", "--replay-artifact=/private/missing", "--observation-start=09:30:00", "--observation-end=09:35:00", "--hydration-workers=1"}, "live-only"},
		{"live replay flag", []string{"--trading-date=2026-08-07", "--observation-start=09:30:00"}, "rejects replay"},
		{"live zero hydration workers", []string{"--trading-date=2026-08-07", "--hydration-workers=0"}, "hydration-workers"},
		{"live unsupported hydration workers", []string{"--trading-date=2026-08-07", "--hydration-workers=3"}, "hydration-workers"},
		{"duplicate scalar", []string{"--trading-date=2026-08-07", "--trading-date=2026-08-08"}, "duplicate --trading-date"},
		{"position", []string{"--trading-date=2026-08-07", "extra"}, "flags are invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := run(context.Background(), test.arguments)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("run error = %v, want %q", err, test.contains)
			}
		})
	}
	provided, err := scalarFlags([]string{"--allow-origin=http://127.0.0.1:3000", "--allow-origin", "http://127.0.0.1:4173", "--api-address=127.0.0.1:0"})
	if err != nil || !provided["api-address"] || provided["allow-origin"] {
		t.Fatalf("repeatable origin parse = %v err=%v", provided, err)
	}
}

func TestLiveHydrationWorkerBounds(t *testing.T) {
	if liveHydrationResponseByteBudget != 4<<30 {
		t.Fatalf("live cumulative response budget = %d", liveHydrationResponseByteBudget)
	}
	for _, test := range []struct {
		workers      int
		wantResident int64
	}{
		{1, 57_600},
		{2, 115_200},
		{4, 230_400},
		{8, 460_800},
	} {
		normalized, resident, err := liveHydrationBounds(test.workers, 6_000)
		if err != nil || normalized != 345_600_000 || resident != test.wantResident {
			t.Fatalf("workers=%d bounds=%d/%d err=%v", test.workers, normalized, resident, err)
		}
	}
	for _, workers := range []int{0, 3, 9} {
		if _, _, err := liveHydrationBounds(workers, 6_000); err == nil {
			t.Fatalf("workers=%d accepted", workers)
		}
	}
}

func TestC12ReplayCompositionContainsOutputAndAPIFailures(t *testing.T) {
	result := replay.Result{Outcome: replay.OutcomeComplete, Completion: replay.CompletionRequestedEnd}
	completed := &replayRunReply{result: result}
	t.Run("final output", func(t *testing.T) {
		runtime, api := &shutdownProbe{}, &shutdownProbe{}
		apiDone := make(chan error, 1)
		apiDone <- nil
		canceled := false
		err := completeReplayOutput(func(any) error { return errors.New("closed output") }, result, runtime, api, nil, apiDone, func() { canceled = true }, completed)
		if err == nil || !strings.Contains(err.Error(), "encode replay result") || !canceled || runtime.calls != 1 || api.calls != 1 {
			t.Fatalf("output containment err=%v canceled=%t runtime=%d api=%d", err, canceled, runtime.calls, api.calls)
		}
	})
	t.Run("API terminal", func(t *testing.T) {
		runtime, api := &shutdownProbe{}, &shutdownProbe{}
		canceled := false
		err := containReplayAPIFailure(errors.New("accept failed"), runtime, api, nil, nil, func() { canceled = true }, completed)
		if err == nil || !strings.Contains(err.Error(), "replay snapshot API") || !canceled || runtime.calls != 1 || api.calls != 1 {
			t.Fatalf("API containment err=%v canceled=%t runtime=%d api=%d", err, canceled, runtime.calls, api.calls)
		}
	})
	t.Run("join timeout", func(t *testing.T) {
		done := make(chan replayRunReply)
		started := time.Now()
		err := shutdownReplayWithin(nil, nil, done, nil, func() {}, nil, false, 5*time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "observation shutdown deadline") || time.Since(started) > 100*time.Millisecond {
			t.Fatalf("bounded join err=%v elapsed=%s", err, time.Since(started))
		}
	})
}

type shutdownProbe struct{ calls int }

func (p *shutdownProbe) Shutdown(context.Context) error {
	p.calls++
	return nil
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
