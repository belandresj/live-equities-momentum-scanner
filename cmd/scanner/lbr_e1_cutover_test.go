package main

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

// TestPLBRE1Cutover is P-LBR-E1-CUTOVER. It combines the ordinary scanner's
// production dependency graph and constructor source with a real live runtime
// construction. Component proofs own market semantics; this proof fails on a
// selectable fallback, replay/checkpoint import, competing constructor, second
// handoff, or unsupported hydration worker surface.
func TestPLBRE1Cutover(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, removed := range []string{"cmd/aggregate-replay", "internal/checkpoint", "internal/replay", "internal/replayartifact", "internal/replaymode"} {
		err := filepath.WalkDir(filepath.Join(root, removed), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				t.Fatalf("removed source remains at %s", path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("inspect removed source %s: %v", removed, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "list", "-deps", "./cmd/scanner")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("ordinary dependency graph: %v", err)
	}
	for _, forbidden := range []string{"/internal/checkpoint", "/internal/replay", "/internal/replayartifact", "/internal/replaymode"} {
		if strings.Contains(string(output), forbidden) {
			t.Fatalf("ordinary scanner reaches removed dependency %q", forbidden)
		}
	}
	trace := exec.CommandContext(ctx, "go", "test", "-count=1", "-run", "^TestPLBRE1ProductionCompositionTrace$", "-timeout", "45s", "./internal/operations")
	trace.Dir = root
	if output, err := trace.CombinedOutput(); err != nil {
		t.Fatalf("production RunLive composition trace: %v\n%s", err, output)
	}

	source, err := os.ReadFile(filepath.Join(root, "cmd/scanner/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{"run-mode", "replay-artifact", "checkpoint-mode", "checkpoint-dir", "runReplay", "composeLiveRuntime", "RunModeLive", "ObserveReplay", "ReplayDeterministic"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("ordinary constructor retains selector %q", forbidden)
		}
	}
	for call, want := range map[string]int{"operations.New(": 1, "massive.NewLiveAdapter(": 1, "runtime.RunLive(": 1} {
		if got := strings.Count(text, call); got != want {
			t.Fatalf("production constructor count %q=%d want %d", call, got, want)
		}
	}

	binding := scannerTestBinding(t)
	now := binding.SessionStart().Add(time.Minute)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	startup, stop := context.WithTimeout(context.Background(), 2*time.Second)
	runtime, err := operations.New(startup, binding, config, func() time.Time { return now })
	stop()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Engine() == nil || runtime.Engine() != runtime.Engine() {
		t.Fatal("runtime did not retain exactly one engine owner")
	}
	for _, workers := range []int{1, 2, 4, 8} {
		components := operations.LiveComponents{Adapter: &massive.LiveAdapter{}, Hydrator: &massive.HydrationWorker{}, Workers: workers, RowsPerChunk: 256, MaximumResponseBytes: liveHydrationResponseByteBudget, MaximumNormalizedRecords: int64(len(binding.UniverseSymbols())) * 57_600, MaximumResidentRecords: int64(workers) * 57_600}
		if err := operations.ValidateLiveComponents(components); err != nil {
			t.Fatalf("workers=%d rejected: %v", workers, err)
		}
	}
	for _, workers := range []int{0, 3, 5, 6, 7, 9} {
		components := operations.LiveComponents{Adapter: &massive.LiveAdapter{}, Hydrator: &massive.HydrationWorker{}, Workers: workers, RowsPerChunk: 256, MaximumResponseBytes: 1, MaximumNormalizedRecords: 1, MaximumResidentRecords: int64(workers) * 57_600}
		if operations.ValidateLiveComponents(components) == nil {
			t.Fatalf("unsupported workers=%d accepted", workers)
		}
	}
	shutdown, stopShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopShutdown()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}

	t.Run("API failure remains outside the live owner", TestSuperviseLiveContainsOutputAndMappingDiagnosticsWhileRuntimeAndAPIContinue)
	t.Run("API replacement retains the immutable source", TestAPISupervisorReplacementServesSameRuntimePublication)
	t.Run("initial API failure prevents live start", TestInitialAPIBindConflictFailsBeforeLiveWorkStarts)
}
