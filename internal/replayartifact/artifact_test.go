package replayartifact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// TestCompilerConsumesRESTNormalizer is P-C4-COMP-NORM. The complete compiler
// can receive provider data only through the sealed DownloadResult produced by
// OfflineDownloader, and an S1 rejection prevents artifact construction.
func TestCompilerConsumesRESTNormalizer(t *testing.T) {
	binding, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
	defer closeServer()
	result := Compile(context.Background(), plan, downloader)
	if result.State != CompileComplete || result.Download.Records != 1 || result.Download.CompleteSymbols != 2 {
		t.Fatalf("valid normalized compile = %+v", result)
	}
	handle, err := OpenValidated(result.Path, ValidationPlan{binding, plan.Start, plan.End, CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	_, rejectedDownloader, rejectedPlan, closeRejected := compileFixture(t, t.TempDir(), true)
	defer closeRejected()
	rejected := Compile(context.Background(), rejectedPlan, rejectedDownloader)
	if rejected.State != CompileFailed || rejected.Reason != CompileReasonDownload || rejected.Download.FailedSymbols != 1 {
		t.Fatalf("invalid raw row escaped mapper boundary: %+v", rejected)
	}
	entries, err := os.ReadDir(rejectedPlan.DestinationDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			t.Fatalf("rejected row published artifact %q", entry.Name())
		}
	}
}

// TestCanonicalArtifactBytes is P-C4-ART-BYTES. It proves the normative golden
// bytes and identity plus order/signed-zero invariance and partial separation.
func TestCanonicalArtifactBytes(t *testing.T) {
	context := artifactContext{mode: CompleteFinalBars, bindingID: "session-binding-v1:example", universeID: "universe-v1:example", date: "2026-08-06",
		sessionStart: time.Date(2026, 8, 6, 8, 0, 0, 0, time.UTC), sessionEnd: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
		replayStart: time.Date(2026, 8, 6, 8, 0, 0, 0, time.UTC), replayEnd: time.Date(2026, 8, 6, 8, 0, 1, 0, time.UTC)}
	record := canonicalRecord{context.replayEnd, "SYN", context.replayStart, context.replayEnd,
		engine.AggregateValues{Open: 10, High: 11, Low: 9, Close: 10.5, Volume: 2.5, VWAP: 10.25, AverageTradeSize: 1, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades}}
	got, metadata, err := buildCanonical(context, []string{"SYN"}, []canonicalRecord{record}, 4096, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "" +
		`{"kind":"header","schema":"aggregate-replay-jsonl-v1","artifact_mode":"complete_final_bars","binding_id":"session-binding-v1:example","universe_id":"universe-v1:example","trading_date":"2026-08-06","session_start":"2026-08-06T08:00:00Z","session_end":"2026-08-07T00:00:00Z","replay_start":"2026-08-06T08:00:00Z","replay_end":"2026-08-06T08:00:01Z","provider":"massive","endpoint":"/v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}","normalization_policy":"massive-rest-second-aggregate-v1","compile_format":"aggregate-replay-jsonl-v1"}` + "\n" +
		`{"kind":"aggregate","ordinal":1,"logical_delivery_time":"2026-08-06T08:00:01Z","symbol":"SYN","window_start":"2026-08-06T08:00:00Z","window_end":"2026-08-06T08:00:01Z","open":10,"high":11,"low":9,"close":10.5,"volume":2.5,"vwap":10.25,"average_trade_size":1,"ats_provenance":"rest_floor_volume_over_transactions"}` + "\n" +
		`{"kind":"coverage","symbol":"SYN","start":"2026-08-06T08:00:00Z","end":"2026-08-06T08:00:01Z","class":"complete_interval","record_count":1}` + "\n" +
		`{"kind":"summary","aggregate_records":1,"coverage_entries":1,"empty_symbols":0,"body_bytes":1008}` + "\n" +
		`{"kind":"seal","artifact_id":"sha256:31ef4b0cbaa85a0b60b3bd70acbbfa4162b9386c539ded580b4ed4897d18d901","sealed_bytes":1106}` + "\n"
	if string(got) != want || metadata.ArtifactID != "sha256:31ef4b0cbaa85a0b60b3bd70acbbfa4162b9386c539ded580b4ed4897d18d901" || metadata.SealedBytes != 1106 {
		t.Fatalf("golden changed:\n%s\nmetadata=%+v", got, metadata)
	}
	exponentRecord := record
	exponentRecord.values.Volume = 1e-7
	exponentBytes, _, err := buildCanonical(context, []string{"SYN"}, []canonicalRecord{exponentRecord}, 4096, 1)
	if err != nil {
		t.Fatal(err)
	}
	expectedFloat := `"volume":` + strconv.FormatFloat(exponentRecord.values.Volume, 'g', -1, 64)
	if !bytes.Contains(exponentBytes, []byte(expectedFloat)) {
		t.Fatalf("aggregate float is not canonical strconv form %q:\n%s", expectedFloat, exponentBytes)
	}

	permutedContext := context
	permutedContext.replayEnd = context.replayEnd.Add(time.Second)
	first := record
	second := canonicalRecord{permutedContext.replayEnd, "EMPTY", context.replayEnd, permutedContext.replayEnd,
		engine.AggregateValues{Open: 20, High: 20, Low: 20, Close: 20, Volume: math.Copysign(0, -1), VWAP: 20, AverageTradeSize: 0, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades}}
	first.logicalDeliveryTime = first.windowEnd
	left, leftMeta, err := buildCanonical(permutedContext, []string{"SYN", "EMPTY", "ZERO"}, []canonicalRecord{second, first}, 8192, 2)
	if err != nil {
		t.Fatal(err)
	}
	second.values.Volume = 0
	right, rightMeta, err := buildCanonical(permutedContext, []string{"ZERO", "EMPTY", "SYN"}, []canonicalRecord{first, second}, 8192, 2)
	if err != nil || !bytes.Equal(left, right) || leftMeta.ArtifactID != rightMeta.ArtifactID || bytes.Contains(left, []byte(`"volume":-0`)) || !bytes.Contains(left, []byte(`"symbol":"ZERO"`)) {
		t.Fatalf("permutation/signed-zero/empty coverage changed identity: %v %s %s", err, leftMeta.ArtifactID, rightMeta.ArtifactID)
	}

	binding := replayArtifactTestBinding(t, []string{"SYN"})
	partial, _, err := BuildPartial(PartialInput{Binding: binding, Start: binding.SessionStart(), End: binding.SessionStart().Add(time.Second), DeclaredSymbols: []string{"SYN"},
		Records: []SyntheticRecord{{LogicalDeliveryTime: binding.SessionStart().Add(time.Second + time.Nanosecond), Symbol: "SYN", WindowStart: binding.SessionStart(), WindowEnd: binding.SessionStart().Add(time.Second),
			Values: engine.AggregateValues{Open: 1, High: 1, Low: 1, Close: 1, Volume: 1, VWAP: 1, AverageTradeSize: 1, ATSProvenance: engine.ATSLiveProviderAverage}}}, MaximumBytes: 4096, MaximumRecords: 1})
	if err != nil || !bytes.Contains(partial, []byte(`"artifact_mode":"partial_synthetic"`)) || bytes.Contains(partial, []byte(`"class":"complete_interval"`)) {
		t.Fatalf("partial artifact separation = %v\n%s", err, partial)
	}
}

// TestArtifactTrustAndPublication is P-C4-ART-TRUST. It exercises whole-file
// trust, atomic no-replace, cancellation, uncertainty, lease, and remnant bounds.
func TestArtifactTrustAndPublication(t *testing.T) {
	t.Run("temporary enumeration limit overflow fails before IO", func(t *testing.T) {
		_, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		completeDestination := filepath.Join(t.TempDir(), "complete-must-not-exist")
		plan.DestinationDirectory = completeDestination
		plan.Limits.MaximumTemporaryFiles = int(^uint(0) >> 1)
		result := Compile(context.Background(), plan, downloader)
		if result.State != CompileFailed || result.Reason != CompileReasonPlanBudget {
			t.Fatalf("oversized complete temporary-file limit = %+v", result)
		}
		if _, err := os.Stat(completeDestination); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid complete plan touched destination: %v", err)
		}

		partialDestination := filepath.Join(t.TempDir(), "partial-must-not-exist")
		partial := CompilePartialArtifact(context.Background(), PartialPlan{
			Input:                PartialInput{Binding: plan.Binding, Start: plan.Start, End: plan.End, DeclaredSymbols: []string{"AAA"}, MaximumBytes: 4096, MaximumRecords: 1},
			DestinationDirectory: partialDestination, MaximumTemporaryBytes: 4096, MaximumTemporaryFiles: int(^uint(0) >> 1),
		})
		if partial.State != CompileFailed || partial.Reason != CompileReasonPlanBudget {
			t.Fatalf("oversized partial temporary-file limit = %+v", partial)
		}
		if _, err := os.Stat(partialDestination); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid partial plan touched destination: %v", err)
		}
	})

	t.Run("complete validate twice idempotent existing", func(t *testing.T) {
		binding, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		first := Compile(context.Background(), plan, downloader)
		if first.State != CompileComplete || first.Accounting.Complete != 1 {
			t.Fatalf("first compile = %+v", first)
		}
		handle, err := OpenValidated(first.Path, ValidationPlan{binding, plan.Start, plan.End, CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords})
		if err != nil || handle.ValidateAgain() != nil || handle.Metadata().ArtifactID != first.ArtifactID {
			t.Fatalf("same-open validation = %v metadata=%+v", err, handle.Metadata())
		}
		handle.Close()
		before, _ := os.ReadFile(first.Path)
		second := Compile(context.Background(), plan, downloader)
		after, _ := os.ReadFile(first.Path)
		if second.State != CompileComplete || second.ArtifactID != first.ArtifactID || !bytes.Equal(before, after) {
			t.Fatalf("idempotent existing = %+v bytes_equal=%t", second, bytes.Equal(before, after))
		}
	})

	t.Run("corruption truncation wrong binding and partial fail closed", func(t *testing.T) {
		binding, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		result := Compile(context.Background(), plan, downloader)
		body, err := os.ReadFile(result.Path)
		if err != nil {
			t.Fatal(err)
		}
		validation := ValidationPlan{binding, plan.Start, plan.End, CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords}
		for name, mutated := range map[string][]byte{"truncated": body[:len(body)-1], "corrupt": bytes.Replace(slices.Clone(body), []byte(`"record_count":1`), []byte(`"record_count":0`), 1)} {
			path := filepath.Join(t.TempDir(), name+".jsonl")
			if err := os.WriteFile(path, mutated, 0o600); err != nil {
				t.Fatal(err)
			}
			if handle, err := OpenValidated(path, validation); err == nil || handle != nil {
				t.Fatalf("%s artifact validated", name)
			}
		}
		other := replayArtifactTestBinding(t, []string{"OTHER"})
		if handle, err := OpenValidated(result.Path, ValidationPlan{other, other.SessionStart(), other.SessionStart().Add(time.Second), CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords}); err == nil || handle != nil {
			t.Fatal("wrong binding artifact validated")
		}
		incomplete, _, err := buildCanonical(contextForBinding(CompleteFinalBars, binding, plan.Start, plan.End), []string{"AAA"}, nil,
			plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords)
		if err != nil {
			t.Fatal(err)
		}
		incompletePath := filepath.Join(t.TempDir(), "missing-binding-symbol.jsonl")
		if err := os.WriteFile(incompletePath, incomplete, 0o600); err != nil {
			t.Fatal(err)
		}
		if handle, err := OpenValidated(incompletePath, validation); err == nil || handle != nil {
			t.Fatal("canonically sealed artifact with incomplete binding coverage validated")
		}
		partialPlan := PartialPlan{Input: PartialInput{Binding: binding, Start: plan.Start, End: plan.End, DeclaredSymbols: []string{"AAA"}, MaximumBytes: 4096, MaximumRecords: 1},
			DestinationDirectory: t.TempDir(), MaximumTemporaryBytes: 4096, MaximumTemporaryFiles: 2}
		partial := CompilePartialArtifact(context.Background(), partialPlan)
		if partial.State != CompileComplete {
			t.Fatalf("partial publication = %+v", partial)
		}
		partialHandle, err := OpenValidated(partial.Path, ValidationPlan{binding, plan.Start, plan.End, PartialSynthetic, 4096, 1})
		if err != nil || partialHandle.Close() != nil {
			t.Fatalf("partial validation = %v", err)
		}
		if handle, err := OpenValidated(partial.Path, validation); err == nil || handle != nil {
			t.Fatal("partial artifact validated as complete")
		}
	})

	t.Run("pre and post publication cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		_, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		ops := ordinaryPersistence
		ops.beforeLink = cancel
		result := compileWithOps(ctx, plan, downloader, ops)
		if result.State != CompileCanceled || artifactFiles(t, plan.DestinationDirectory) != 0 {
			t.Fatalf("pre-link cancellation = %+v files=%d", result, artifactFiles(t, plan.DestinationDirectory))
		}

		ctx, cancel = context.WithCancel(context.Background())
		_, downloader, plan, closeServer = compileFixture(t, t.TempDir(), false)
		defer closeServer()
		ops = ordinaryPersistence
		ops.link = func(old, new string) error {
			err := os.Link(old, new)
			cancel()
			return err
		}
		result = compileWithOps(ctx, plan, downloader, ops)
		if result.State != CompileComplete || artifactFiles(t, plan.DestinationDirectory) != 1 {
			t.Fatalf("post-link cancellation = %+v files=%d", result, artifactFiles(t, plan.DestinationDirectory))
		}
	})

	t.Run("sync uncertainty later adoption and conflicting existing", func(t *testing.T) {
		binding, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		ops := ordinaryPersistence
		ops.syncDirectory = func(string) error { return errors.New("injected sync failure") }
		uncertain := compileWithOps(context.Background(), plan, downloader, ops)
		if uncertain.State != CompilePersistenceUncertain || uncertain.Accounting.PersistenceUncertain != 1 || artifactFiles(t, plan.DestinationDirectory) != 1 {
			t.Fatalf("sync uncertainty = %+v files=%d", uncertain, artifactFiles(t, plan.DestinationDirectory))
		}
		adopted := Compile(context.Background(), plan, downloader)
		if adopted.State != CompileComplete || adopted.ArtifactID != uncertain.ArtifactID {
			t.Fatalf("later adoption = %+v", adopted)
		}
		if handle, err := OpenValidated(adopted.Path, ValidationPlan{binding, plan.Start, plan.End, CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords}); err != nil || handle.Close() != nil {
			t.Fatalf("adopted validation = %v", err)
		}

		original, _ := os.ReadFile(adopted.Path)
		if err := os.WriteFile(adopted.Path, []byte("conflict\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		conflict := Compile(context.Background(), plan, downloader)
		preserved, _ := os.ReadFile(adopted.Path)
		if conflict.State != CompileFailed || conflict.Reason != CompileReasonExistingDestination || string(preserved) != "conflict\n" || bytes.Equal(original, preserved) {
			t.Fatalf("conflicting existing = %+v preserved=%q", conflict, preserved)
		}
	})

	t.Run("nonregular existing destination fails promptly", func(t *testing.T) {
		_, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		initial := Compile(context.Background(), plan, downloader)
		if initial.State != CompileComplete {
			t.Fatalf("initial compile = %+v", initial)
		}
		validBytes, err := os.ReadFile(initial.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(initial.Path); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(initial.Path, 0o600); err != nil {
			t.Fatal(err)
		}
		finished := make(chan CompileResult, 1)
		go func() { finished <- Compile(context.Background(), plan, downloader) }()
		select {
		case result := <-finished:
			if result.State != CompileFailed || result.Reason != CompileReasonExistingDestination {
				t.Fatalf("FIFO existing destination = %+v", result)
			}
		case <-time.After(time.Second):
			t.Fatal("FIFO existing destination blocked validation")
		}

		if err := os.Remove(initial.Path); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "valid-target.jsonl")
		if err := os.WriteFile(target, validBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, initial.Path); err != nil {
			t.Fatal(err)
		}
		result := Compile(context.Background(), plan, downloader)
		if result.State != CompileFailed || result.Reason != CompileReasonExistingDestination {
			t.Fatalf("symlink existing destination = %+v", result)
		}
	})

	t.Run("retained temporary and lease stop accumulation", func(t *testing.T) {
		_, downloader, plan, closeServer := compileFixture(t, t.TempDir(), false)
		defer closeServer()
		ops := ordinaryPersistence
		ops.remove = func(path string) error {
			if strings.HasPrefix(filepath.Base(path), temporaryPrefix) {
				return errors.New("injected temporary removal failure")
			}
			return os.Remove(path)
		}
		uncertain := compileWithOps(context.Background(), plan, downloader, ops)
		if uncertain.State != CompilePersistenceUncertain {
			t.Fatalf("temporary uncertainty = %+v", uncertain)
		}
		entriesBefore, _ := os.ReadDir(plan.DestinationDirectory)
		blocked := Compile(context.Background(), plan, downloader)
		entriesAfter, _ := os.ReadDir(plan.DestinationDirectory)
		if blocked.State != CompileFailed || len(entriesAfter) != len(entriesBefore) {
			t.Fatalf("remnant accumulation = %+v before=%d after=%d", blocked, len(entriesBefore), len(entriesAfter))
		}

		otherDirectory := t.TempDir()
		_, otherDownloader, otherPlan, closeOther := compileFixture(t, otherDirectory, false)
		defer closeOther()
		if err := os.WriteFile(filepath.Join(otherPlan.DestinationDirectory, leaseName), []byte("held"), 0o600); err != nil {
			t.Fatal(err)
		}
		leaseBlocked := Compile(context.Background(), otherPlan, otherDownloader)
		if leaseBlocked.State != CompileFailed || artifactFiles(t, otherPlan.DestinationDirectory) != 0 {
			t.Fatalf("lease conflict = %+v", leaseBlocked)
		}
	})
}

func compileFixture(t *testing.T, destination string, invalid bool) (reference.Binding, *massive.OfflineDownloader, CompletePlan, func()) {
	t.Helper()
	binding := replayArtifactTestBinding(t, []string{"AAA", "BBB"})
	start := binding.SessionStart()
	end := start.Add(time.Second)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		parts := strings.Split(request.URL.Path, "/")
		symbol := parts[4]
		if symbol == "AAA" {
			if invalid {
				fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10}]}`, start.UnixMilli())
				return
			}
			fmt.Fprintf(writer, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"n":1}]}`, start.UnixMilli())
			return
		}
		fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, symbol)
	}))
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "compile-test-token", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	plan := CompletePlan{Binding: binding, Start: start, End: end, Workers: 2, DestinationDirectory: destination,
		Limits: Limits{MaximumNormalizedRecords: 2, MaximumResponseBytes: 1 << 20, MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 2}}
	return binding, downloader, plan, server.Close
}

func replayArtifactTestBinding(t *testing.T, symbols []string) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-06")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			records := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				records[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "count": len(records), "results": records})
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

func artifactFiles(t *testing.T, directory string) int {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			count++
		}
	}
	return count
}
