package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func TestOfflineReplayCommandUsesCachedBindingAndArtifact(t *testing.T) {
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-06")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v3/reference/tickers":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "count": 1, "results": []map[string]any{{"ticker": "SYN", "active": true, "market": "stocks", "locale": "us", "type": "CS"}}})
		case strings.HasPrefix(r.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": 1, "results": []map[string]any{{"T": "SYN", "c": 10, "t": facts.PriorRegularClose.UnixMilli()}}})
		case strings.HasPrefix(r.URL.Path, "/v2/aggs/ticker/SYN/range/1/second/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "ticker": "SYN", "adjusted": false,
				"results": []map[string]any{{"t": facts.SessionStart.UnixMilli(), "o": 10, "h": 11, "l": 9, "c": 10.5, "v": 100, "vw": 10.25, "n": 10}}})
		default:
			http.NotFound(w, r)
		}
	}))
	privateRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	referenceDirectory := filepath.Join(privateRoot, "reference")
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: referenceDirectory, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }}).Resolve(context.Background(), facts)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: referenceDirectory, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	start, end := facts.SessionStart, facts.SessionStart.Add(time.Second)
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: 1,
		DestinationDirectory: t.TempDir(), Limits: replayartifact.Limits{MaximumNormalizedRecords: 1, MaximumResponseBytes: 1 << 20,
			MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 1}}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("compile = %+v", compiled)
	}
	t.Setenv("MASSIVE_API_KEY", "")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	runErr := run(context.Background(), []string{"--trading-date=2026-08-06", "--artifact=" + compiled.Path,
		"--from=" + start.Format(time.RFC3339Nano), "--to=" + end.Format(time.RFC3339Nano), "--reference-dir=" + referenceDirectory,
		"--queue-capacity=8", "--required-reserve=2", "--evaluation-delay=0s", "--maximum-records=1", "--maximum-artifact-bytes=1048576"})
	_ = writer.Close()
	os.Stdout = originalStdout
	defer reader.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	var result replay.Result
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Outcome != replay.OutcomeComplete || result.Status.RunMode != "replay" || result.Status.Lifecycle != "ended" || result.Status.CommittedT == nil || *result.Status.CommittedT != end {
		t.Fatalf("offline replay result = %+v", result)
	}
}

func TestDestinationProviderDataWarningBoundary(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	ignoredRoot, ok := repositoryIgnoredVarRoot(workingDirectory)
	if !ok {
		t.Fatal("repository ignored var root not found")
	}
	if destinationOutsideIgnoredVar(filepath.Join(ignoredRoot, "aggregate-replay")) {
		t.Fatal("default ignored artifact destination was classified as outside var")
	}
	if !destinationOutsideIgnoredVar(filepath.Join("build", "aggregate-replay")) {
		t.Fatal("non-var artifact destination did not require a provider-data warning")
	}
	original := workingDirectory
	t.Cleanup(func() { _ = os.Chdir(original) })
	outside := t.TempDir()
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	if !destinationOutsideIgnoredVar(filepath.Join("var", "aggregate-replay")) {
		t.Fatal("default destination outside the repository did not require a warning")
	}

	repository := filepath.Join(t.TempDir(), "repository")
	target := filepath.Join(t.TempDir(), "provider-data")
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "go.mod"), []byte("module github.com/belandresj/live-equities-momentum-scanner\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, ".gitignore"), []byte("var/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(repository, "var")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repository); err != nil {
		t.Fatal(err)
	}
	if !destinationOutsideIgnoredVar(filepath.Join("var", "aggregate-replay")) {
		t.Fatal("var symlink escaping the repository did not require a warning")
	}
}
