package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// TestC4CORE01DeterministicAggregateCore is P-C4-CORE. A representative
// complete artifact built through the accepted downloader/normalizer/compiler
// path drives the same Component 1-3 engine at unpaced and finite accelerated
// wall pace. Every logical group and final immutable proof/output projection
// must be identical.
func TestC4CORE01DeterministicAggregateCore(t *testing.T) {
	binding := replayBinding(t, []string{"EMPTY", "QUAL", "SPARSE"})
	start, end := binding.SessionStart(), binding.SessionStart().Add(60*time.Second)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		symbol := parts[4]
		rows := make([]map[string]any, 0, 60)
		for second := 0; second < 60; second++ {
			if symbol == "EMPTY" || symbol == "SPARSE" && second != 0 && second != 30 && second != 59 {
				continue
			}
			closeValue, volume, transactions := 13.2, 1000.0, 100
			if symbol == "SPARSE" {
				closeValue, volume, transactions = 15, 100, 10
			}
			rows = append(rows, map[string]any{"t": start.Add(time.Duration(second) * time.Second).UnixMilli(),
				"o": closeValue - 0.1, "h": closeValue + 0.2, "l": closeValue - 0.2, "c": closeValue,
				"v": volume, "vw": closeValue, "n": transactions})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK", "ticker": symbol, "adjusted": false, "results": rows})
	}))
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test-token", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	directory := t.TempDir()
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{
		Binding: binding, Start: start, End: end, Workers: 3, DestinationDirectory: directory,
		Limits: replayartifact.Limits{MaximumNormalizedRecords: 200, MaximumResponseBytes: 1 << 20,
			MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 200},
	}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("representative compile = %+v", compiled)
	}
	open := func() *replayartifact.Handle {
		handle, openErr := replayartifact.OpenValidated(compiled.Path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end,
			ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: 200})
		if openErr != nil {
			t.Fatal(openErr)
		}
		return handle
	}
	leftHandle, rightHandle := open(), open()
	defer leftHandle.Close()
	defer rightHandle.Close()
	left := sourceFromHandle(t, leftHandle, binding, start, Unpaced())
	finite, err := FinitePace(1000, 1)
	if err != nil {
		t.Fatal(err)
	}
	right := sourceFromHandle(t, rightHandle, binding, start, finite)
	wall := time.Unix(0, 0).UTC()
	right.pacer.now = func() time.Time { return wall }
	right.pacer.wait = func(_ context.Context, duration time.Duration) error { wall = wall.Add(duration); return nil }

	leftTrace, leftResult, leftView := stepCore(t, left)
	rightTrace, rightResult, rightView := stepCore(t, right)
	if len(leftTrace) != 61 || len(rightTrace) != len(leftTrace) {
		t.Fatalf("representative group counts left=%d right=%d", len(leftTrace), len(rightTrace))
	}
	for index := range leftTrace {
		if !reflect.DeepEqual(leftTrace[index], rightTrace[index]) {
			t.Fatalf("pace changed deterministic group %d at %s:\nleft=%+v\nright=%+v", index, leftTrace[index].Group.LogicalTime, leftTrace[index], rightTrace[index])
		}
	}
	if !reflect.DeepEqual(leftResult, rightResult) {
		t.Fatalf("pace changed final deterministic result:\nleft=%+v\nright=%+v", leftResult, rightResult)
	}
	if !reflect.DeepEqual(leftView, rightView) {
		t.Fatalf("pace changed final Component 1-3 proof view:\nleft=%+v\nright=%+v", leftView, rightView)
	}
	status := leftResult.Status
	if leftResult.Outcome != OutcomeComplete || status.Lifecycle != "ended" || status.CommittedT == nil || *status.CommittedT != end ||
		status.RankingMode != "qualified_current" || status.UniverseTotal != 3 || status.TrustedRankableMarks != 2 ||
		status.NoPrintThroughT != 1 || status.UnknownPopulation != 0 || status.Rows != 1 || status.TQIntentRows != 0 {
		t.Fatalf("representative final status is not nontrivial: %+v", status)
	}
	view := leftView
	if view.Evaluation.TotalPassers != 1 || view.Evaluation.TQIntentAvailable || len(view.Evaluation.Rows) != 1 || view.Evaluation.Rows[0].Symbol != "QUAL" ||
		view.Evaluation.Rows[0].Rank != 1 || view.Evaluation.Rows[0].TQIntentEligible ||
		view.Publication.PublicationID == 0 || view.Publication.GeneratedAt != end || view.Publication.Watermark == nil || *view.Publication.Watermark != end ||
		view.Publication.RunMode != engine.RunModeReplay || view.Publication.AggregateEvaluation.Mode != "qualified_current" {
		t.Fatalf("representative evaluation/publication = %+v", view)
	}
	canonical := make(map[string]engine.ReplayCanonicalSymbol, len(view.Canonical))
	for _, symbol := range view.Canonical {
		canonical[symbol.Symbol] = symbol
	}
	if len(canonical["QUAL"].Records) != 60 || canonical["QUAL"].Qualification.Status != "provisional" ||
		canonical["QUAL"].Features.DayPercent.Status != "current" || len(canonical["SPARSE"].Records) != 3 ||
		len(canonical["EMPTY"].Records) != 0 || canonical["EMPTY"].ProvenAbsentSlots != 60 {
		t.Fatalf("representative canonical/feature/qualification breadth = %+v", canonical)
	}
}

type coreGroupTrace struct {
	Group GroupResult
	View  engine.ReplayDeterministicView
}

func stepCore(t *testing.T, source *Source) ([]coreGroupTrace, Result, engine.ReplayDeterministicView) {
	t.Helper()
	ctx := context.Background()
	if err := source.Start(ctx); err != nil {
		t.Fatal(err)
	}
	trace := make([]coreGroupTrace, 0, source.accounting.PlannedGroups)
	for !source.nextGroup.After(source.start.End()) {
		logical := source.nextGroup
		group, err := source.Step(ctx)
		if err != nil {
			t.Fatalf("core group %s: %v status=%+v", logical, err, source.engine.ObserveReplay())
		}
		trace = append(trace, coreGroupTrace{Group: group, View: source.engine.ObserveReplayDeterministic()})
	}
	result, err := source.Finish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return trace, result, source.engine.ObserveReplayDeterministic()
}

// TestC4SCHED01SparseClockTrace is P-C4-SCHED. It compares every logical
// group at unpaced and finite accelerated pace, including quiet seconds.
func TestC4SCHED01SparseClockTrace(t *testing.T) {
	left, closeLeft := completeSource(t, Unpaced())
	defer closeLeft()
	finite, err := FinitePace(1000, 1)
	if err != nil {
		t.Fatal(err)
	}
	right, closeRight := completeSource(t, finite)
	defer closeRight()
	wall := time.Unix(0, 0)
	right.pacer.now = func() time.Time { return wall }
	right.pacer.wait = func(ctx context.Context, duration time.Duration) error { wall = wall.Add(duration); return nil }

	leftTrace, leftResult := stepAll(t, left)
	rightTrace, rightResult := stepAll(t, right)
	if len(leftTrace) != 4 || len(rightTrace) != 4 {
		t.Fatalf("group count left=%d right=%d", len(leftTrace), len(rightTrace))
	}
	for index := range leftTrace {
		l, r := leftTrace[index], rightTrace[index]
		if l.LogicalTime != r.LogicalTime || l.Records != r.Records || l.Timer.EngineSequence != r.Timer.EngineSequence ||
			l.Timer.SystemSequence != r.Timer.SystemSequence || l.Timer.AdmissionTime != r.Timer.AdmissionTime || l.Status.LastOrdinal != r.Status.LastOrdinal ||
			l.Status.CommittedT == nil || r.Status.CommittedT == nil || *l.Status.CommittedT != *r.Status.CommittedT {
			t.Fatalf("pace changed logical group %d:\nleft=%+v\nright=%+v", index, l, r)
		}
	}
	if leftTrace[0].Records != 0 || leftTrace[2].Records != 0 || leftResult.Outcome != OutcomeComplete || rightResult.Outcome != OutcomeComplete ||
		leftResult.Accounting != rightResult.Accounting || leftResult.Accounting.CompletedGroups != 4 || leftResult.Accounting.CompletedRecordDispositions != 2 {
		t.Fatalf("sparse trace/result mismatch: %+v %+v trace=%+v", leftResult, rightResult, leftTrace)
	}

	t.Run("maximum reducible pace remains valid", func(t *testing.T) {
		pace, err := FinitePace(1_000_000_000, 1_000_000_000)
		if err != nil {
			t.Fatal(err)
		}
		wall := time.Unix(0, 0)
		pacer := wallPacer{pace: pace, started: wall, now: func() time.Time { return wall }}
		pacer.wait = func(_ context.Context, duration time.Duration) error {
			if duration != 10*time.Second {
				t.Fatalf("reduced pace wait = %s", duration)
			}
			wall = wall.Add(duration)
			return nil
		}
		if err := pacer.waitUntil(context.Background(), 10*time.Second); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fractional duration cannot overflow after whole seconds", func(t *testing.T) {
		pace, err := FinitePace(1736, 277982185)
		if err != nil {
			t.Fatal(err)
		}
		wall := time.Unix(0, 0).UTC()
		pacer := wallPacer{pace: pace, started: wall, now: func() time.Time { return wall }, wait: func(context.Context, time.Duration) error {
			t.Fatal("overflowed duration reached wall wait")
			return nil
		}}
		if err := pacer.waitUntil(context.Background(), 16*time.Hour); err == nil || err.Error() != "pace duration overflow" {
			t.Fatalf("fractional duration overflow = %v", err)
		}
	})
}

// TestC4RUN01LifecycleCoverageCommit is P-C4-RUN. Complete evidence commits
// every group and proves slot absence; partial corrections remain commit-ineligible.
func TestC4RUN01LifecycleCoverageCommit(t *testing.T) {
	complete, closeComplete := completeSource(t, Unpaced())
	defer closeComplete()
	_, result := stepAll(t, complete)
	if result.Outcome != OutcomeComplete || result.Status.Lifecycle != "ended" || result.Status.CommittedT == nil || *result.Status.CommittedT != complete.start.End() ||
		result.Status.LastOrdinal != 2 || result.Status.PresentSlots != 2 || result.Status.ProvenAbsentSlots != 4 || result.Status.UniverseTotal != 2 ||
		result.Status.TrustedRankableMarks != 1 || result.Status.NoPrintThroughT != 1 || result.Status.UnknownPopulation != 0 || result.Status.TQIntentRows != 0 ||
		result.Status.RunMode != engine.RunModeReplay || result.Status.AggregateInserted != 2 || result.Status.AggregateRevised != 0 {
		t.Fatalf("complete replay lifecycle/coverage/commit = %+v", result)
	}

	partial, closePartial := partialSource(t)
	defer closePartial()
	trace, partialResult := stepAll(t, partial)
	if partialResult.Outcome != OutcomeComplete || partialResult.Status.Lifecycle != "ended" || partialResult.Status.CommittedT != nil || partialResult.Status.CompleteEvidence ||
		partialResult.Status.LastOrdinal != 2 || partialResult.Status.AggregateInserted != 1 || partialResult.Status.AggregateRevised != 1 || len(trace) != 3 || trace[1].Records != 1 || trace[2].Records != 1 {
		t.Fatalf("partial replay gained complete support: result=%+v trace=%+v", partialResult, trace)
	}

	t.Run("short tail R equals E ends only after replay evidence", func(t *testing.T) {
		source, cleanup := sessionEndEmptySource(t)
		defer cleanup()
		result := source.Run(context.Background())
		if result.Outcome != OutcomeComplete || result.Status.Lifecycle != "ended" || result.Status.CommittedT == nil || *result.Status.CommittedT != source.start.End() ||
			result.Status.LastLogicalTime != source.start.End() || result.Status.UniverseTotal != 1 ||
			result.Accounting.PlannedGroups != 3 || result.Accounting.CompletedGroups != 3 || result.Accounting.CompletedRecordDispositions != 0 {
			t.Fatalf("session-end replay boundary = %+v", result)
		}
	})
}

// TestC4FAIL01Containment is P-C4-FAIL. It covers changed second-pass bytes,
// clock contradiction, and cancellation without artifact-end success.
func TestC4FAIL01Containment(t *testing.T) {
	t.Run("direct finish cancellation closes and accounts", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		ctx := context.Background()
		if err := source.Start(ctx); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.End()) {
			if _, err := source.Step(ctx); err != nil {
				t.Fatal(err)
			}
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		result, err := source.Finish(canceled)
		if !errors.Is(err, context.Canceled) || result.Outcome != OutcomeCanceled || result.Reason != ReasonCanceled ||
			result.Accounting.CompletedRuns != 0 || result.Accounting.FailedRuns != 0 || result.Accounting.CanceledRuns != 1 ||
			!source.terminal || result.Status.Lifecycle != "ended" {
			t.Fatalf("direct finish cancellation = err=%v result=%+v terminal=%v", err, result, source.terminal)
		}
	})

	t.Run("same-open file changed", func(t *testing.T) {
		source, path, cleanup := completeSourceWithPath(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.End()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bytes = append(bytes, ' ')
		if err := os.WriteFile(path, bytes, 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := source.Finish(context.Background())
		status := source.engine.ObserveReplay()
		if err == nil || result.Outcome != OutcomeFailed || result.Reason != ReasonArtifactEnd || status.Lifecycle != "suppressed" || status.Suppression != engine.SuppressionTerminalReplayFailure ||
			status.FailureReason != engine.ReplayFailureArtifactEnd || status.FailureLogicalTime != source.start.End() || status.FailureOrdinal != 2 {
			t.Fatalf("changed artifact false success: err=%v result=%+v status=%+v", err, result, source.engine.ObserveReplay())
		}
	})

	t.Run("validated handle identity changed before playback", func(t *testing.T) {
		binding := replayBinding(t, []string{"AAA"})
		start, end := binding.SessionStart(), binding.SessionStart().Add(2*time.Second)
		values := replayValues()
		first, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, Records: []replayartifact.SyntheticRecord{{LogicalDeliveryTime: end, Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: values}}, MaximumBytes: 1 << 20, MaximumRecords: 1})
		if err != nil {
			t.Fatal(err)
		}
		values.Close = 10.5
		second, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, Records: []replayartifact.SyntheticRecord{{LogicalDeliveryTime: end, Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: values}}, MaximumBytes: 1 << 20, MaximumRecords: 1})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "identity.jsonl")
		if err := os.WriteFile(path, first, 0o600); err != nil {
			t.Fatal(err)
		}
		handle, err := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 1})
		if err != nil {
			t.Fatal(err)
		}
		defer handle.Close()
		validatedArtifactID := handle.Metadata().ArtifactID
		if err := os.WriteFile(path, second, 0o600); err != nil {
			t.Fatal(err)
		}
		source := sourceFromHandle(t, handle, binding, start, Unpaced())
		result := source.Run(context.Background())
		if result.Outcome != OutcomeFailed || result.Reason != ReasonArtifactValidation || result.ArtifactID != validatedArtifactID || result.Status.ArtifactID != validatedArtifactID ||
			result.Status.FailureReason != engine.ReplayFailureArtifactValidation ||
			!result.Status.FailureLogicalTime.IsZero() || result.Status.FailureOrdinal != 0 || result.Status.Lifecycle != "suppressed" {
			t.Fatalf("rewritten validated identity was adopted: %+v", result)
		}
	})

	t.Run("wrong ordinal reports exact supported boundary", func(t *testing.T) {
		source, path, expectedLogical, cleanup := longPartialSource(t)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		value, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := []byte(strings.Replace(string(value), `"ordinal":20`, `"ordinal":21`, 1))
		if slices.Equal(value, mutated) || len(value) != len(mutated) {
			t.Fatal("ordinal mutation did not preserve file size")
		}
		if err := os.WriteFile(path, mutated, 0o600); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.End()) {
			if _, err := source.Step(context.Background()); err != nil {
				break
			}
		}
		status := source.engine.ObserveReplay()
		if status.FailureReason != engine.ReplayFailureOrdinalGroup || status.FailureOrdinal != 19 || status.FailureLogicalTime != expectedLogical ||
			status.Lifecycle != "suppressed" || status.Suppression != engine.SuppressionTerminalReplayFailure {
			t.Fatalf("wrong ordinal boundary = %+v", status)
		}
	})

	for _, mutation := range []struct {
		name         string
		from, to     string
		resultReason Reason
		engineReason engine.ReplayFailureReason
	}{
		{"coverage classification", `"class":"declared_partial"`, `"class":"declared_partiaX"`, ReasonBindingCoverage, engine.ReplayFailureBindingCoverage},
		{"schema canonical classification", `"kind":"summary"`, `"kind":"Summary"`, ReasonSchemaCanonical, engine.ReplayFailureSchemaCanonical},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			source, path, _, cleanup := longPartialSource(t)
			defer cleanup()
			if err := source.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			value, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := []byte(strings.Replace(string(value), mutation.from, mutation.to, 1))
			if slices.Equal(value, changed) {
				t.Fatalf("%s mutation did not apply", mutation.name)
			}
			if err := os.WriteFile(path, changed, 0o600); err != nil {
				t.Fatal(err)
			}
			for !source.nextGroup.After(source.start.End()) {
				if _, err := source.Step(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			result, err := source.Finish(context.Background())
			if err == nil || result.Outcome != OutcomeFailed || result.Reason != mutation.resultReason || result.Status.FailureReason != mutation.engineReason ||
				result.Status.FailureLogicalTime != source.start.End() || result.Status.FailureOrdinal != 30 || result.Status.Lifecycle != "suppressed" {
				t.Fatalf("%s boundary: err=%v result=%+v", mutation.name, err, result)
			}
		})
	}

	for _, mutation := range []struct {
		name  string
		apply func([]byte) []byte
	}{
		{"truncated EOF", func(value []byte) []byte { return value[:len(value)-32] }},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			source, path, cleanup := completeSourceWithPath(t, Unpaced())
			defer cleanup()
			if err := source.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			value, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, mutation.apply(value), 0o600); err != nil {
				t.Fatal(err)
			}
			for !source.nextGroup.After(source.start.End()) {
				if _, stepErr := source.Step(context.Background()); stepErr != nil {
					break
				}
			}
			if !source.terminal {
				_, _ = source.Finish(context.Background())
			}
			status := source.engine.ObserveReplay()
			if status.Lifecycle != "suppressed" || status.Suppression != engine.SuppressionTerminalReplayFailure ||
				status.FailureReason != engine.ReplayFailureArtifactEnd || status.FailureLogicalTime != source.start.End() || status.FailureOrdinal != 2 {
				t.Fatalf("%s escaped containment: %+v", mutation.name, status)
			}
			if admission, completion := source.engine.AdmitTimer(context.Background()); admission != engine.AdmissionNotAdmittedClosed || completion != nil {
				t.Fatalf("%s restored failed engine: %s", mutation.name, admission)
			}
		})
	}

	t.Run("valid artifact record rejected by engine", func(t *testing.T) {
		source, cleanup := latePartialSource(t)
		defer cleanup()
		result := source.Run(context.Background())
		status := source.engine.ObserveReplay()
		if result.Outcome != OutcomeFailed || result.Accounting.ArtifactRecords != 1 || result.Accounting.CompletedRecordDispositions != 1 || result.Accounting.UnreadRecords != 0 ||
			status.Lifecycle != "suppressed" || status.Suppression != engine.SuppressionTerminalReplayFailure || status.FailureReason != engine.ReplayFailureAggregate {
			t.Fatalf("engine rejection containment: result=%+v status=%+v", result, status)
		}
	})

	t.Run("clock regression", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := source.clock.advance(source.start.Start().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := source.Step(context.Background()); err == nil {
			t.Fatal("clock regression succeeded")
		}
		status := source.engine.ObserveReplay()
		if status.Lifecycle != "suppressed" || status.Suppression != engine.SuppressionTerminalReplayFailure {
			t.Fatalf("clock failure containment = %+v", status)
		}
	})

	t.Run("canceled before start", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := source.Run(ctx)
		if result.Outcome != OutcomeCanceled || result.Accounting.CanceledRuns != 1 || result.Accounting.CompletedRuns != 0 || result.Status.CommittedT != nil {
			t.Fatalf("cancellation claimed completion: %+v", result)
		}
	})

	t.Run("canceled during group pacing", func(t *testing.T) {
		pace, _ := FinitePace(1, 1)
		source, cleanup := completeSource(t, pace)
		defer cleanup()
		ctx, cancel := context.WithCancel(context.Background())
		if err := source.Start(ctx); err != nil {
			t.Fatal(err)
		}
		source.pacer.wait = func(context.Context, time.Duration) error { cancel(); return ctx.Err() }
		if _, err := source.Step(ctx); err != nil {
			t.Fatal(err)
		} // S has no wall wait.
		if _, err := source.Step(ctx); err == nil {
			t.Fatal("paced cancellation succeeded")
		}
		result := source.canceled()
		a := result.Accounting
		if result.Outcome != OutcomeCanceled || a.ArtifactRecords != a.CompletedRecordDispositions+a.UnreadRecords || a.PlannedGroups != a.CompletedGroups+a.ActiveGroup+a.RemainingGroups || a.CanceledRuns != 1 {
			t.Fatalf("paced cancellation accounting: %+v", result)
		}
	})
}

func stepAll(t *testing.T, source *Source) ([]GroupResult, Result) {
	t.Helper()
	ctx := context.Background()
	if err := source.Start(ctx); err != nil {
		t.Fatal(err)
	}
	var trace []GroupResult
	for !source.nextGroup.After(source.start.End()) {
		logical := source.nextGroup
		group, err := source.Step(ctx)
		if err != nil {
			t.Fatalf("group %s: %v status=%+v", logical, err, source.engine.ObserveReplay())
		}
		trace = append(trace, group)
	}
	result, err := source.Finish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return trace, result
}

func completeSource(t *testing.T, pace Pace) (*Source, func()) {
	source, _, cleanup := completeSourceWithPath(t, pace)
	return source, cleanup
}
func completeSourceWithPath(t *testing.T, pace Pace) (*Source, string, func()) {
	t.Helper()
	binding := replayBinding(t, []string{"AAA", "BBB"})
	start, end := binding.SessionStart(), binding.SessionStart().Add(3*time.Second)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		symbol := parts[4]
		if symbol == "AAA" {
			fmt.Fprintf(w, `{"status":"OK","ticker":"AAA","adjusted":false,"results":[{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":1,"vw":10,"n":1},{"t":%d,"o":11,"h":12,"l":10,"c":11,"v":2,"vw":11,"n":2}]}`, start.UnixMilli(), start.Add(2*time.Second).UnixMilli())
			return
		}
		fmt.Fprintf(w, `{"status":"OK","ticker":%q,"adjusted":false,"results":[]}`, symbol)
	}))
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test-token", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	dir := t.TempDir()
	plan := replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: 2, DestinationDirectory: dir, Limits: replayartifact.Limits{MaximumNormalizedRecords: 6, MaximumResponseBytes: 1 << 20, MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 6}}
	compiled := replayartifact.Compile(context.Background(), plan, downloader)
	if compiled.State != replayartifact.CompileComplete {
		server.Close()
		t.Fatalf("compile=%+v", compiled)
	}
	handle, err := replayartifact.OpenValidated(compiled.Path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: 6})
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	clock, _ := NewSimulatedClock(start)
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	if disposition := <-completion; disposition.Code != engine.DispositionBindingInstalled {
		t.Fatal(disposition)
	}
	source, err := NewSource(handle, owner, clock, pace)
	if err != nil {
		t.Fatal(err)
	}
	return source, compiled.Path, func() { handle.Close(); server.Close() }
}

func sessionEndEmptySource(t *testing.T) (*Source, func()) {
	t.Helper()
	binding := replayBinding(t, []string{"EMPTY"})
	start, end := binding.SessionEnd().Add(-2*time.Second), binding.SessionEnd()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"OK","ticker":"EMPTY","adjusted":false,"results":[]}`)
	}))
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "test-token", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	directory := t.TempDir()
	maximumRecords := int64(2)
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: 1,
		DestinationDirectory: directory, Limits: replayartifact.Limits{MaximumNormalizedRecords: maximumRecords, MaximumResponseBytes: 1 << 20,
			MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: maximumRecords}}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("session-end empty compile = %+v", compiled)
	}
	handle, err := replayartifact.OpenValidated(compiled.Path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end,
		ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: maximumRecords})
	if err != nil {
		t.Fatal(err)
	}
	source := sourceFromHandle(t, handle, binding, start, Unpaced())
	return source, func() { _ = handle.Close() }
}

func partialSource(t *testing.T) (*Source, func()) {
	t.Helper()
	binding := replayBinding(t, []string{"AAA", "BBB"})
	start, end := binding.SessionStart(), binding.SessionStart().Add(2*time.Second)
	values := engine.AggregateValues{Open: 10, High: 11, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1, ATSProvenance: engine.ATSLiveProviderAverage}
	corrected := values
	corrected.Close = 10.5
	bytes, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, Records: []replayartifact.SyntheticRecord{{LogicalDeliveryTime: start.Add(time.Second), Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: values}, {LogicalDeliveryTime: end, Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: corrected}}, MaximumBytes: 1 << 20, MaximumRecords: 2})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	handle, err := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 2})
	if err != nil {
		t.Fatal(err)
	}
	clock, _ := NewSimulatedClock(start)
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	_, completion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if disposition := <-completion; disposition.Code != engine.DispositionBindingInstalled {
		t.Fatal(disposition)
	}
	source, err := NewSource(handle, owner, clock, Unpaced())
	if err != nil {
		t.Fatal(err)
	}
	return source, func() { handle.Close() }
}

func replayValues() engine.AggregateValues {
	return engine.AggregateValues{Open: 10, High: 12, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1, ATSProvenance: engine.ATSLiveProviderAverage}
}

func sourceFromHandle(t *testing.T, handle *replayartifact.Handle, binding reference.Binding, start time.Time, pace Pace) *Source {
	t.Helper()
	clock, err := NewSimulatedClock(start)
	if err != nil {
		t.Fatal(err)
	}
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 64, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || completion == nil {
		t.Fatalf("binding admission = %s", admission)
	}
	if disposition := <-completion; disposition.Code != engine.DispositionBindingInstalled {
		t.Fatalf("binding disposition = %+v", disposition)
	}
	source, err := NewSource(handle, owner, clock, pace)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func longPartialSource(t *testing.T) (*Source, string, time.Time, func()) {
	t.Helper()
	binding := replayBinding(t, []string{"AAA"})
	start, end := binding.SessionStart(), binding.SessionStart().Add(2*time.Second)
	records := make([]replayartifact.SyntheticRecord, 30)
	for index := range records {
		values := replayValues()
		values.Close = 10 + float64(index)/100
		records[index] = replayartifact.SyntheticRecord{LogicalDeliveryTime: end, Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: values}
	}
	value, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, Records: records, MaximumBytes: 1 << 20, MaximumRecords: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(value) <= 4096 {
		t.Fatalf("ordinal mutation fixture is too small: %d bytes", len(value))
	}
	path := filepath.Join(t.TempDir(), "long-partial.jsonl")
	if err := os.WriteFile(path, value, 0o600); err != nil {
		t.Fatal(err)
	}
	handle, err := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 30})
	if err != nil {
		t.Fatal(err)
	}
	source := sourceFromHandle(t, handle, binding, start, Unpaced())
	return source, path, start.Add(time.Second), func() { handle.Close() }
}

func latePartialSource(t *testing.T) (*Source, func()) {
	t.Helper()
	binding := replayBinding(t, []string{"AAA"})
	start, end := binding.SessionStart(), binding.SessionStart().Add(16*time.Minute+2*time.Second)
	values := engine.AggregateValues{Open: 10, High: 11, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1, ATSProvenance: engine.ATSLiveProviderAverage}
	bytes, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, Records: []replayartifact.SyntheticRecord{{LogicalDeliveryTime: end, Symbol: "AAA", WindowStart: start, WindowEnd: start.Add(time.Second), Values: values}}, MaximumBytes: 1 << 20, MaximumRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "late-partial.jsonl")
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	handle, err := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	clock, _ := NewSimulatedClock(start)
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	_, completion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	<-completion
	source, err := NewSource(handle, owner, clock, Unpaced())
	if err != nil {
		t.Fatal(err)
	}
	return source, func() { handle.Close() }
}

func replayBinding(t *testing.T, symbols []string) reference.Binding {
	t.Helper()
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
			records := make([]map[string]any, len(symbols))
			for i, s := range symbols {
				records[i] = map[string]any{"ticker": s, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			json.NewEncoder(w).Encode(map[string]any{"status": "OK", "count": len(records), "results": records})
		case strings.HasPrefix(r.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			rows := make([]map[string]any, len(symbols))
			for i, s := range symbols {
				rows[i] = map[string]any{"T": s, "c": 10 + i, "t": facts.PriorRegularClose.UnixMilli()}
			}
			json.NewEncoder(w).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(rows), "results": rows})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	dir := filepath.Join(t.TempDir(), "reference")
	resolver := &reference.Resolver{BaseURL: server.URL, APIKey: "test", DataDir: dir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}
	universe, err := resolver.Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priorResolver := &reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "test", DataDir: dir, HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}
	priors, err := priorResolver.Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil || !slices.Equal(binding.UniverseSymbols(), symbols) {
		t.Fatalf("binding %v %v", err, binding.UniverseSymbols())
	}
	return binding
}
