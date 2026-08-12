package replayartifact

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const maximumDiagnosticScaleRecords = int64(2_000_000)

type resourceFixture struct {
	path       string
	records    int64
	bytes      int64
	fileSHA256 string
	metadata   Metadata
}

type readerResourceResult struct {
	Records              int64   `json:"records"`
	Bytes                int64   `json:"bytes"`
	Phase                string  `json:"phase"`
	LastOrdinal          int64   `json:"last_ordinal"`
	ArtifactID           string  `json:"artifact_id"`
	WallSeconds          float64 `json:"wall_seconds"`
	UserCPUSeconds       float64 `json:"user_cpu_seconds"`
	SystemCPUSeconds     float64 `json:"system_cpu_seconds"`
	HeapAllocBefore      uint64  `json:"heap_alloc_before"`
	HeapAllocCurrent     uint64  `json:"heap_alloc_current"`
	HeapAllocRetained    uint64  `json:"heap_alloc_retained"`
	PeakHeapAlloc        uint64  `json:"peak_heap_alloc"`
	PeakHeapInuse        uint64  `json:"peak_heap_inuse"`
	CurrentRSSBefore     uint64  `json:"current_rss_before"`
	CurrentRSSAfter      uint64  `json:"current_rss_after"`
	PeakRSS              uint64  `json:"peak_rss"`
	TotalAllocatedBytes  uint64  `json:"total_allocated_bytes"`
	GarbageCollections   uint32  `json:"garbage_collections"`
	ProgressObservations int64   `json:"progress_observations"`
}

// TestOpenValidatedContextResourceScale is the explicit, non-short
// P-NARROW-READER-RESOURCE scale rung. Each invocation generates one valid,
// fixed-cardinality artifact through the production canonical encoder, then
// validates it in a fresh child process through OpenValidatedContext so fixture
// construction does not contaminate reader heap/RSS measurements.
func TestOpenValidatedContextResourceScale(t *testing.T) {
	if testing.Short() {
		t.Skip("resource scale proof is an explicit acceptance test")
	}
	recordText := os.Getenv("REPLAYARTIFACT_RESOURCE_RECORDS")
	if recordText == "" {
		t.Skip("set REPLAYARTIFACT_RESOURCE_RECORDS to one bounded scale rung")
	}
	records, err := strconv.ParseInt(recordText, 10, 64)
	if err != nil || records <= 0 || records > maximumDiagnosticScaleRecords {
		t.Fatalf("invalid diagnostic record cardinality %q (maximum %d)", recordText, maximumDiagnosticScaleRecords)
	}

	binding := replayArtifactTestBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	end := start.Add(2 * time.Second)
	fixture := writeResourceFixture(t, binding, start, end, records)
	t.Logf("fixture records=%d bytes=%d file_sha256=%s artifact_id=%s mode=%s binding=%s interval=[%s,%s)",
		fixture.records, fixture.bytes, fixture.fileSHA256, fixture.metadata.ArtifactID, fixture.metadata.Mode,
		binding.Identity(), start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano))

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	profilePath := ""
	if records == 100_000 {
		profilePath = filepath.Join(t.TempDir(), "reader-allocs.pprof")
	}
	commandContext, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(commandContext, executable, "-test.run=^TestOpenValidatedContextResourceHelper$", "-test.v")
	command.Env = append(os.Environ(),
		"REPLAYARTIFACT_RESOURCE_HELPER=1",
		"REPLAYARTIFACT_RESOURCE_PATH="+fixture.path,
		"REPLAYARTIFACT_RESOURCE_EXPECTED_RECORDS="+strconv.FormatInt(records, 10),
		"REPLAYARTIFACT_RESOURCE_EXPECTED_BYTES="+strconv.FormatInt(fixture.bytes, 10),
		"REPLAYARTIFACT_RESOURCE_ALLOCS_PROFILE="+profilePath,
	)
	output, commandErr := command.CombinedOutput()
	if commandContext.Err() != nil {
		t.Fatalf("reader child exceeded 90s bound: %v\n%s", commandContext.Err(), output)
	}
	if commandErr != nil {
		t.Fatalf("reader child failed: %v\n%s", commandErr, output)
	}
	t.Logf("reader child:\n%s", output)

	if profilePath != "" {
		profileContext, profileCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer profileCancel()
		profile := exec.CommandContext(profileContext, "go", "tool", "pprof", "-top", "-alloc_space", "-nodecount=15", executable, profilePath)
		profileOutput, profileErr := profile.CombinedOutput()
		if profileErr != nil {
			t.Fatalf("allocation attribution failed: %v\n%s", profileErr, profileOutput)
		}
		t.Logf("reader allocation attribution:\n%s", profileOutput)
	}
}

func TestOpenValidatedContextResourceHelper(t *testing.T) {
	if os.Getenv("REPLAYARTIFACT_RESOURCE_HELPER") != "1" {
		t.Skip("resource child only")
	}
	path := os.Getenv("REPLAYARTIFACT_RESOURCE_PATH")
	expectedRecords, err1 := strconv.ParseInt(os.Getenv("REPLAYARTIFACT_RESOURCE_EXPECTED_RECORDS"), 10, 64)
	expectedBytes, err2 := strconv.ParseInt(os.Getenv("REPLAYARTIFACT_RESOURCE_EXPECTED_BYTES"), 10, 64)
	if path == "" || err1 != nil || err2 != nil || expectedRecords <= 0 || expectedBytes <= 0 {
		t.Fatal("invalid resource child configuration")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != expectedBytes {
		t.Fatalf("fixture identity changed: stat=%v size=%d expected=%d", err, info.Size(), expectedBytes)
	}
	binding := replayArtifactTestBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	end := start.Add(2 * time.Second)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	usageBefore := readProcessUsage(t)
	currentRSSBefore := currentRSSBytes(t)

	type sampledPeak struct {
		sync.Mutex
		heapAlloc uint64
		heapInuse uint64
	}
	peak := sampledPeak{heapAlloc: before.HeapAlloc, heapInuse: before.HeapInuse}
	stopSampling := make(chan struct{})
	samplingDone := make(chan struct{})
	go func() {
		defer close(samplingDone)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var sample runtime.MemStats
				runtime.ReadMemStats(&sample)
				peak.Lock()
				if sample.HeapAlloc > peak.heapAlloc {
					peak.heapAlloc = sample.HeapAlloc
				}
				if sample.HeapInuse > peak.heapInuse {
					peak.heapInuse = sample.HeapInuse
				}
				peak.Unlock()
			case <-stopSampling:
				return
			}
		}
	}()

	var progressMu sync.Mutex
	progress := validationProgress{}
	var observations int64
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	ctx = withValidationProgressObserver(ctx, 10_000, func(value validationProgress) {
		progressMu.Lock()
		progress = value
		observations++
		progressMu.Unlock()
	})
	wallStart := time.Now()
	handle, openErr := OpenValidatedContext(ctx, path, ValidationPlan{
		Binding: binding, Start: start, End: end, ExpectedMode: PartialSynthetic,
		MaximumBytes: expectedBytes, MaximumRecords: expectedRecords,
	})
	wall := time.Since(wallStart)
	close(stopSampling)
	<-samplingDone
	if openErr != nil {
		t.Fatalf("OpenValidatedContext: %v", openErr)
	}
	metadata := handle.Metadata()
	if closeErr := handle.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if metadata.AggregateRecords != expectedRecords {
		t.Fatalf("validated records=%d expected=%d", metadata.AggregateRecords, expectedRecords)
	}

	var current runtime.MemStats
	runtime.ReadMemStats(&current)
	currentRSSAfter := currentRSSBytes(t)
	usageAfter := readProcessUsage(t)
	runtime.GC()
	var retained runtime.MemStats
	runtime.ReadMemStats(&retained)
	peak.Lock()
	peakHeapAlloc, peakHeapInuse := peak.heapAlloc, peak.heapInuse
	peak.Unlock()
	progressMu.Lock()
	finalProgress, progressCount := progress, observations
	progressMu.Unlock()
	if finalProgress.phase != "validated" || finalProgress.records != expectedRecords || finalProgress.lastOrdinal != expectedRecords || finalProgress.bytes != expectedBytes {
		t.Fatalf("final progress=%+v expected records=%d bytes=%d", finalProgress, expectedRecords, expectedBytes)
	}

	result := readerResourceResult{
		Records: expectedRecords, Bytes: expectedBytes, Phase: finalProgress.phase, LastOrdinal: finalProgress.lastOrdinal,
		ArtifactID: metadata.ArtifactID, WallSeconds: wall.Seconds(),
		UserCPUSeconds: usageAfter.userSeconds - usageBefore.userSeconds, SystemCPUSeconds: usageAfter.systemSeconds - usageBefore.systemSeconds,
		HeapAllocBefore: before.HeapAlloc, HeapAllocCurrent: current.HeapAlloc, HeapAllocRetained: retained.HeapAlloc,
		PeakHeapAlloc: peakHeapAlloc, PeakHeapInuse: peakHeapInuse, CurrentRSSBefore: currentRSSBefore,
		CurrentRSSAfter: currentRSSAfter, PeakRSS: usageAfter.peakRSS,
		TotalAllocatedBytes: current.TotalAlloc - before.TotalAlloc, GarbageCollections: current.NumGC - before.NumGC,
		ProgressObservations: progressCount,
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("READER_RESOURCE %s\n", encoded)

	if profilePath := os.Getenv("REPLAYARTIFACT_RESOURCE_ALLOCS_PROFILE"); profilePath != "" {
		profileFile, err := os.Create(profilePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.Lookup("allocs").WriteTo(profileFile, 0); err != nil {
			_ = profileFile.Close()
			t.Fatal(err)
		}
		if err := profileFile.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

type processUsage struct {
	userSeconds   float64
	systemSeconds float64
	peakRSS       uint64
}

func readProcessUsage(t *testing.T) processUsage {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	peakRSS := uint64(usage.Maxrss)
	if runtime.GOOS != "darwin" {
		peakRSS *= 1024
	}
	return processUsage{
		userSeconds:   float64(usage.Utime.Sec) + float64(usage.Utime.Usec)/1e6,
		systemSeconds: float64(usage.Stime.Sec) + float64(usage.Stime.Usec)/1e6,
		peakRSS:       peakRSS,
	}
}

func currentRSSBytes(t *testing.T) uint64 {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/bin/ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Logf("current RSS unavailable: %v", err)
		return 0
	}
	kib, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		t.Logf("current RSS unavailable: %v", err)
		return 0
	}
	return kib * 1024
}

func writeResourceFixture(t *testing.T, binding reference.Binding, start, end time.Time, records int64) resourceFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), fmt.Sprintf("resource-%d.jsonl", records))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := bufio.NewWriterSize(file, 1<<20)
	digest := sha256.New()
	written := int64(0)
	writeLine := func(value any, includeInDigest bool) {
		t.Helper()
		line, encodeErr := encodeCanonicalLine(value)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if _, writeErr := writer.Write(line); writeErr != nil {
			t.Fatal(writeErr)
		}
		written += int64(len(line))
		if includeInDigest {
			_, _ = digest.Write(line)
		}
	}

	artifactContext := contextForBinding(PartialSynthetic, binding, start, end)
	writeLine(headerLine{
		Kind: "header", Schema: SchemaV1, ArtifactMode: string(PartialSynthetic), BindingID: artifactContext.bindingID,
		UniverseID: artifactContext.universeID, TradingDate: artifactContext.date,
		SessionStart: canonicalTime(artifactContext.sessionStart), SessionEnd: canonicalTime(artifactContext.sessionEnd),
		ReplayStart: canonicalTime(start), ReplayEnd: canonicalTime(end), Provider: SyntheticProvider,
		Endpoint: "", NormalizationPolicy: SyntheticPolicy, CompileFormat: SchemaV1,
	}, true)
	windowEnd := start.Add(time.Second)
	for ordinal := int64(1); ordinal <= records; ordinal++ {
		writeLine(aggregateLine{
			Kind: "aggregate", Ordinal: ordinal, LogicalDeliveryTime: canonicalTime(windowEnd), Symbol: "AAA",
			WindowStart: canonicalTime(start), WindowEnd: canonicalTime(windowEnd),
			Open: 10, High: 11, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1,
			ATSProvenance: string(engine.ATSLiveProviderAverage),
		}, true)
	}
	writeLine(coverageLine{
		Kind: "coverage", Symbol: "AAA", Start: canonicalTime(start), End: canonicalTime(end),
		Class: PartialCoverage, RecordCount: records,
	}, true)
	bodyBytes := written
	writeLine(summaryLine{
		Kind: "summary", AggregateRecords: records, CoverageEntries: 1, EmptySymbols: 0, BodyBytes: bodyBytes,
	}, true)
	sealedBytes := written
	artifactID := "sha256:" + hex.EncodeToString(digest.Sum(nil))
	writeLine(sealLine{Kind: "seal", ArtifactID: artifactID, SealedBytes: sealedBytes}, false)
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	opened, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	fileDigest := sha256.New()
	if _, err := io.Copy(fileDigest, opened); err != nil {
		_ = opened.Close()
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	return resourceFixture{
		path: path, records: records, bytes: written, fileSHA256: hex.EncodeToString(fileDigest.Sum(nil)),
		metadata: Metadata{
			Mode: PartialSynthetic, ArtifactID: artifactID, BindingIdentity: binding.Identity(), UniverseIdentity: binding.UniverseIdentity(),
			TradingDate: binding.TradingDate(), SessionStart: binding.SessionStart(), SessionEnd: binding.SessionEnd(),
			ReplayStart: start, ReplayEnd: end, AggregateRecords: records, CoverageEntries: 1, SealedBytes: sealedBytes,
		},
	}
}
