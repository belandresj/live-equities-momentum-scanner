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
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

type scanCancelContext struct {
	context.Context
	mu       sync.Mutex
	done     chan struct{}
	checks   int
	cancelOn int
	canceled bool
}

func newScanCancelContext(cancelOn int) *scanCancelContext {
	return &scanCancelContext{Context: context.Background(), done: make(chan struct{}), cancelOn: cancelOn}
}
func (c *scanCancelContext) Done() <-chan struct{} { return c.done }
func (c *scanCancelContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks++
	if c.cancelOn > 0 && c.checks >= c.cancelOn && !c.canceled {
		c.canceled = true
		close(c.done)
	}
	if c.canceled {
		return context.Canceled
	}
	return nil
}
func (c *scanCancelContext) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.checks
}

// TestC4CORE01DeterministicAggregateCore is P-C4-CORE. A representative
// complete artifact built through the accepted downloader/normalizer/compiler
// path drives the same Component 1-3 engine at unpaced and finite accelerated
// wall pace. Every logical group and final immutable proof/output projection
// must be identical.
func TestC4CORE01DeterministicAggregateCore(t *testing.T) {
	t.Skip("unsupported replay full-history feature projection no longer constrains the live engine after LBR-B3 removal")
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

// TestReplayFastForwardBoundaryEquivalence is RW-EQUIV. The ordinary replay
// schedule and the hidden warm-up schedule consume the same complete artifact.
// They must be identical at O0 and for sixty subsequent logical boundaries;
// only publication IDs may differ because hidden intermediate publications are
// deliberately omitted.
func TestReplayFastForwardBoundaryEquivalence(t *testing.T) {
	t.Skip("unsupported replay full-history projection no longer constrains the live engine after LBR-B3 removal")
	binding := replayBinding(t, []string{"CONTINUOUS", "EARLY", "EMPTY", "LATE", "RESUMES", "SPARSE"})
	start := binding.SessionStart()
	observationStart := start.Add(400 * time.Second)
	end := observationStart.Add(60 * time.Second)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		symbol := parts[4]
		rows := make([]map[string]any, 0, 460)
		for second := 0; second < 460; second++ {
			include := symbol == "CONTINUOUS" || symbol == "EARLY" && second < 60 || symbol == "LATE" && second >= 340 ||
				symbol == "RESUMES" && second >= 420 || symbol == "SPARSE" && (second == 0 || second%97 == 0)
			if !include {
				continue
			}
			closeValue, volume, transactions := 13.2, 1000.0, 100
			switch symbol {
			case "LATE":
				closeValue = 22 + float64(second-340)/100
			case "RESUMES":
				closeValue = 18 + float64(second-420)/100
			case "SPARSE":
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
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: 4,
		DestinationDirectory: directory, Limits: replayartifact.Limits{MaximumNormalizedRecords: 5000, MaximumResponseBytes: 1 << 20,
			MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: 5000}}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("equivalence compile = %+v", compiled)
	}
	open := func() *replayartifact.Handle {
		handle, openErr := replayartifact.OpenValidated(compiled.Path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end,
			ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: 5000})
		if openErr != nil {
			t.Fatal(openErr)
		}
		return handle
	}
	ordinaryHandle, acceleratedHandle := open(), open()
	defer ordinaryHandle.Close()
	defer acceleratedHandle.Close()
	ordinary := sourceFromHandleWithDelay(t, ordinaryHandle, binding, start, Unpaced(), 4*time.Second)
	accelerated := sourceFromHandleWithDelay(t, acceleratedHandle, binding, start, Unpaced(), 4*time.Second)
	if err := accelerated.engine.ConfigureReplayFastForwardThrough(observationStart); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ordinary.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := accelerated.Start(ctx); err != nil {
		t.Fatal(err)
	}
	var observationView engine.ReplayDeterministicView
	for group := start; !group.After(end); group = group.Add(time.Second) {
		ordinaryGroup, ordinaryErr := ordinary.Step(ctx)
		acceleratedGroup, acceleratedErr := accelerated.Step(ctx)
		if ordinaryErr != nil || acceleratedErr != nil || ordinaryGroup.LogicalTime != group || acceleratedGroup.LogicalTime != group ||
			ordinaryGroup.Records != acceleratedGroup.Records || ordinaryGroup.Timer.EngineSequence != acceleratedGroup.Timer.EngineSequence {
			t.Fatalf("group %s ordinary=%+v/%v accelerated=%+v/%v", group, ordinaryGroup, ordinaryErr, acceleratedGroup, acceleratedErr)
		}
		if group.Before(observationStart) {
			continue
		}
		assertReplaySourceProgressEqual(t, ordinary, accelerated, group)
		left, right := normalizedFastForwardView(ordinary.engine.ObserveReplayDeterministic()), normalizedFastForwardView(accelerated.engine.ObserveReplayDeterministic())
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("fast-forward diverged at %s:\nordinary=%+v\naccelerated=%+v", group, left, right)
		}
		if right.Evaluation.Population.UnknownDueFailureOrFence != 0 || right.Evaluation.Population.UnresolvedPopulation != 0 ||
			right.Evaluation.Mode != "qualified_current" {
			t.Fatalf("complete replay lost committed-T coverage at %s: %+v", group, right.Evaluation)
		}
		if group.Equal(observationStart) {
			observationView = accelerated.engine.ObserveReplayDeterministic()
		}
	}
	ordinaryResult, ordinaryErr := ordinary.Finish(ctx)
	acceleratedResult, acceleratedErr := accelerated.Finish(ctx)
	if ordinaryErr != nil || acceleratedErr != nil || ordinaryResult.Outcome != OutcomeComplete || acceleratedResult.Outcome != OutcomeComplete ||
		ordinaryResult.Accounting != acceleratedResult.Accounting {
		t.Fatalf("terminal equivalence ordinary=%+v/%v accelerated=%+v/%v", ordinaryResult, ordinaryErr, acceleratedResult, acceleratedErr)
	}
	view := accelerated.engine.ObserveReplayDeterministic()
	canonical := make(map[string]engine.ReplayCanonicalSymbol, len(view.Canonical))
	for _, symbol := range view.Canonical {
		canonical[symbol.Symbol] = symbol
	}
	if canonical["EARLY"].Qualification.Status != "provisional" || canonical["EARLY"].Qualification.CurrentProofCount == 0 ||
		canonical["LATE"].Qualification.Status != "provisional" || canonical["EMPTY"].ProvenAbsentSlots != 460 {
		t.Fatalf("equivalence fixture breadth: early=%+v late=%+v empty_absent=%d", canonical["EARLY"].Qualification,
			canonical["LATE"].Qualification, canonical["EMPTY"].ProvenAbsentSlots)
	}
	assertCurrentReplayMVPRow(t, observationView, "CONTINUOUS")
	assertCurrentReplayMVPRow(t, view, "CONTINUOUS")
}

func assertCurrentReplayMVPRow(t *testing.T, view engine.ReplayDeterministicView, symbol string) {
	t.Helper()
	for _, row := range view.Evaluation.Rows {
		if row.Symbol == symbol {
			if row.Activity30s.Status != "current" || row.Move30s.Status != "current" {
				t.Fatalf("%s MVP fields are not current at %s: activity=%+v move=%+v", symbol, view.Evaluation.At, row.Activity30s, row.Move30s)
			}
			return
		}
	}
	t.Fatalf("%s absent from ranked rows at %s", symbol, view.Evaluation.At)
}

func TestReplayFastForwardReplacementContinuationEquivalence(t *testing.T) {
	binding := replayBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	observationStart := start.Add(400 * time.Second)
	end := observationStart.Add(60 * time.Second)
	records := make([]replayartifact.SyntheticRecord, 0, 462)
	for second := 0; second < 460; second++ {
		values := replayValues()
		values.Open, values.High, values.Low, values.Close = 12.9, 13.4, 12.8, 13.2
		values.Volume, values.AverageTradeSize = 1000, 10
		windowStart := start.Add(time.Duration(second) * time.Second)
		records = append(records, replayartifact.SyntheticRecord{LogicalDeliveryTime: windowStart.Add(time.Second), Symbol: "AAA",
			WindowStart: windowStart, WindowEnd: windowStart.Add(time.Second), Values: values})
		if second == 399 {
			revised := records[390]
			revised.LogicalDeliveryTime = observationStart
			revised.Values.Open, revised.Values.High, revised.Values.Low, revised.Values.Close = 14, 14.2, 13.9, 14.1
			revised.Values.Volume, revised.Values.VWAP = 1500, 14.1
			records = append(records, revised, revised)
		}
	}
	bytes, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"},
		Records: records, MaximumBytes: 1 << 20, MaximumRecords: 1000})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fast-forward-replacement.jsonl")
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	open := func() *replayartifact.Handle {
		handle, openErr := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end,
			ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 1000})
		if openErr != nil {
			t.Fatal(openErr)
		}
		return handle
	}
	ordinaryHandle, acceleratedHandle := open(), open()
	defer ordinaryHandle.Close()
	defer acceleratedHandle.Close()
	ordinary := sourceFromHandleWithDelay(t, ordinaryHandle, binding, start, Unpaced(), 4*time.Second)
	accelerated := sourceFromHandleWithDelay(t, acceleratedHandle, binding, start, Unpaced(), 4*time.Second)
	if err := accelerated.engine.ConfigureReplayFastForwardThrough(observationStart); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ordinary.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := accelerated.Start(ctx); err != nil {
		t.Fatal(err)
	}
	for group := start; !group.After(end); group = group.Add(time.Second) {
		leftGroup, leftErr := ordinary.Step(ctx)
		rightGroup, rightErr := accelerated.Step(ctx)
		if leftErr != nil || rightErr != nil || leftGroup.LogicalTime != group || rightGroup.LogicalTime != group || leftGroup.Records != rightGroup.Records {
			t.Fatalf("replacement group %s ordinary=%+v/%v accelerated=%+v/%v", group, leftGroup, leftErr, rightGroup, rightErr)
		}
		if group.Before(observationStart) {
			continue
		}
		assertReplaySourceProgressEqual(t, ordinary, accelerated, group)
		left, right := normalizedFastForwardView(ordinary.engine.ObserveReplayDeterministic()), normalizedFastForwardView(accelerated.engine.ObserveReplayDeterministic())
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("replacement ordinary-path state diverged at %s:\nordinary=%+v\nconfigured=%+v", group, left, right)
		}
		if group.Equal(observationStart) && len(right.Canonical) != 1 {
			t.Fatalf("replacement fixture omits canonical state at O0: %+v", right.Canonical)
		}
	}
	status := accelerated.engine.ObserveReplay()
	if status.AggregateInserted != 460 || status.AggregateRevised != 1 || status.AggregateExactDuplicate != 1 {
		t.Fatalf("replacement dispositions = %+v", status)
	}
}

func assertReplaySourceProgressEqual(t *testing.T, ordinary, accelerated *Source, at time.Time) {
	t.Helper()
	if ordinary.accounting != accelerated.accounting || ordinary.nextGroup != accelerated.nextGroup || ordinary.terminal != accelerated.terminal ||
		ordinary.clock.Now() != accelerated.clock.Now() {
		t.Fatalf("source progress diverged at %s: ordinary=%+v next=%s terminal=%v clock=%s accelerated=%+v next=%s terminal=%v clock=%s",
			at, ordinary.accounting, ordinary.nextGroup, ordinary.terminal, ordinary.clock.Now(), accelerated.accounting, accelerated.nextGroup,
			accelerated.terminal, accelerated.clock.Now())
	}
}

func TestReplayFastForwardPolicyIsReplayOnlyAndImmutable(t *testing.T) {
	clock := time.Unix(1_800_000_000, 0).UTC()
	delay := 4 * time.Second
	live, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return clock }, Capacity: 16, RequiredReserve: 2, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		live.Close()
		_ = live.Wait(context.Background())
	}()
	if err := live.ConfigureReplayFastForwardThrough(clock); err == nil {
		t.Fatal("live engine accepted replay fast-forward policy")
	}

	source, cleanup := completeSource(t, Unpaced())
	defer cleanup()
	boundary := source.handle.Metadata().ReplayStart.Add(time.Second)
	if err := source.engine.ConfigureReplayFastForwardThrough(boundary); err != nil {
		t.Fatal(err)
	}
	if err := source.engine.ConfigureReplayFastForwardThrough(boundary.Add(time.Second)); err == nil {
		t.Fatal("replay engine accepted a second fast-forward boundary")
	}
	if err := source.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := source.engine.ConfigureReplayFastForwardThrough(boundary); err == nil {
		t.Fatal("active replay accepted fast-forward reconfiguration")
	}
}

func normalizedFastForwardView(value engine.ReplayDeterministicView) engine.ReplayDeterministicView {
	value.Publication.PublicationID = 0
	return value
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
	for !source.nextGroup.After(source.start.RequestedEnd()) {
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
		result, err := source.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
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
	t.Run("direct finish cancellation requires explicit bounded cancel", func(t *testing.T) {
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
		if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(result, Result{}) || source.terminal || source.engine.ObserveReplay().Lifecycle != "replaying" {
			t.Fatalf("direct finish cancellation = err=%v result=%+v terminal=%v", err, result, source.terminal)
		}
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		result, err = source.Cancel(cancelContext)
		if err != nil || result.Outcome != OutcomeCanceled || result.Reason != ReasonCanceled || result.Status.Lifecycle != "ended" ||
			result.Accounting.CompletedRuns != 0 || result.Accounting.FailedRuns != 0 || result.Accounting.CanceledRuns != 1 {
			t.Fatalf("explicit finish cancel = err=%v result=%+v", err, result)
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
		result, _ := source.Run(context.Background())
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
		result, _ := source.Run(context.Background())
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
		result, runErr := source.Run(ctx)
		if !errors.Is(runErr, context.Canceled) || !reflect.DeepEqual(result, Result{}) || source.engine.ObserveReplay().Lifecycle != "initializing" {
			t.Fatalf("canceled run fabricated terminal result: result=%+v err=%v", result, runErr)
		}
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		result, err := source.Cancel(cancelContext)
		if err != nil || result.Outcome != OutcomeCanceled || result.Accounting.CanceledRuns != 1 || result.Accounting.CompletedRuns != 0 || result.Status.CommittedT != nil {
			t.Fatalf("explicit pre-start cancel: result=%+v err=%v", result, err)
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
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		result, err := source.Cancel(cancelContext)
		a := result.Accounting
		if err != nil || result.Outcome != OutcomeCanceled || a.ArtifactRecords != a.CompletedRecordDispositions+a.UnreadRecords ||
			a.PlannedGroups != a.CompletedGroups+a.ActiveGroup+a.RemainingGroups || a.ActiveGroup != 0 || a.CanceledRuns != 1 {
			t.Fatalf("paced cancellation accounting: %+v err=%v", result, err)
		}
	})
}

// TestC4BOUNDEDCANCEL01ManualSource is the lifecycle/playback portion of
// P-C4-BOUNDED-CANCEL. It distinguishes pre-link cancellation from sealed
// terminal outcomes and proves the sole Cancel operation is bounded and
// idempotent.
func TestC4BOUNDEDCANCEL01ManualSource(t *testing.T) {
	t.Run("cancel before start is one controlled stop", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		first, err := source.Cancel(ctx)
		if err != nil || first.Outcome != OutcomeCanceled || first.Reason != ReasonCanceled || first.Status.Lifecycle != "ended" ||
			first.Accounting.CanceledRuns != 1 || first.Accounting.CompletedRuns != 0 || first.Accounting.FailedRuns != 0 {
			t.Fatalf("first cancel = %+v err=%v", first, err)
		}
		second, err := source.Cancel(ctx)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatalf("repeated cancel changed result: first=%+v second=%+v err=%v", first, second, err)
		}
		if err := source.Start(context.Background()); !errors.Is(err, context.Canceled) {
			t.Fatalf("post-cancel source admitted new work: %v", err)
		}
	})

	t.Run("cancel before step closes started source", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result, err := source.Cancel(ctx)
		if err != nil || result.Outcome != OutcomeCanceled || result.Status.Lifecycle != "ended" ||
			result.Accounting.ArtifactRecords != result.Accounting.CompletedRecordDispositions+result.Accounting.IntentionallyUnappliedSuffixRecords+result.Accounting.UnreadRecords {
			t.Fatalf("pre-step cancel = %+v err=%v", result, err)
		}
	})

	t.Run("already linked nonterminal work drains before cancel", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		stepContext, cancelStep := context.WithCancel(context.Background())
		var once sync.Once
		source.afterNonterminalAdmission = func() { once.Do(cancelStep) }
		if _, err := source.Step(stepContext); !errors.Is(err, context.Canceled) {
			t.Fatalf("linked step did not observe cancellation after disposition: %v", err)
		}
		source.afterNonterminalAdmission = nil
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result, err := source.Cancel(ctx)
		a := result.Accounting
		if err != nil || result.Outcome != OutcomeCanceled || result.Status.Lifecycle != "ended" ||
			a.PlannedGroups != a.CompletedGroups+a.ActiveGroup+a.RemainingGroups || a.CompletedGroups != 1 || a.ActiveGroup != 0 {
			t.Fatalf("linked disposition cancel = %+v err=%v", result, err)
		}
	})

	t.Run("expired operation deadline is joined by cancel", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		stepContext, cancelStep := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancelStep()
		var once sync.Once
		source.afterNonterminalAdmission = func() { once.Do(func() { <-stepContext.Done() }) }
		if _, err := source.Step(stepContext); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("step deadline = %v", err)
		}
		source.afterNonterminalAdmission = nil
		join, cancelJoin := context.WithTimeout(context.Background(), time.Second)
		defer cancelJoin()
		result, err := source.Cancel(join)
		a := result.Accounting
		if err != nil || result.Outcome != OutcomeCanceled || a.CompletedGroups != 1 || a.ActiveGroup != 0 ||
			a.PlannedGroups != a.CompletedGroups+a.ActiveGroup+a.RemainingGroups {
			t.Fatalf("deadline join = %+v err=%v", result, err)
		}
	})

	t.Run("suffix scan cancellation returns no end evidence", func(t *testing.T) {
		countSource, _, countCleanup := requestedCompleteSource(t, time.Second)
		defer countCleanup()
		if err := countSource.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !countSource.nextGroup.After(countSource.start.RequestedEnd()) {
			if _, err := countSource.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		counter := newScanCancelContext(0)
		if evidence, err := countSource.cursor.RequestedEndContext(counter); err != nil || !evidence.Valid() {
			t.Fatalf("count suffix scan = valid=%v err=%v", evidence.Valid(), err)
		}
		checks := counter.count()
		if checks < 6 {
			t.Fatalf("suffix scan lacked per-line checks: %d", checks)
		}

		for _, cancelOn := range []int{checks / 2, checks} {
			source, _, cleanup := requestedCompleteSource(t, time.Second)
			if err := source.Start(context.Background()); err != nil {
				cleanup()
				t.Fatal(err)
			}
			for !source.nextGroup.After(source.start.RequestedEnd()) {
				if _, err := source.Step(context.Background()); err != nil {
					cleanup()
					t.Fatal(err)
				}
			}
			cancelScan := newScanCancelContext(cancelOn)
			evidence, err := source.cursor.RequestedEndContext(cancelScan)
			if !errors.Is(err, context.Canceled) || evidence.Valid() {
				cleanup()
				t.Fatalf("canceled suffix check %d returned end evidence: valid=%v err=%v", cancelOn, evidence.Valid(), err)
			}
			cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			result, cancelErr := source.Cancel(cancelCtx)
			cancel()
			cleanup()
			if cancelErr != nil || result.Outcome != OutcomeCanceled || result.Completion != "" || result.Accounting.CompletedRuns != 0 {
				t.Fatalf("suffix cancellation check %d = %+v err=%v", cancelOn, result, cancelErr)
			}
		}
	})

	t.Run("first playback pass cancellation returns no cursor", func(t *testing.T) {
		countSource, countCleanup := completeSource(t, Unpaced())
		defer countCleanup()
		counter := newScanCancelContext(0)
		cursor, err := countSource.handle.BeginPlaybackContext(counter)
		if err != nil || cursor == nil {
			t.Fatalf("count first pass = %v err=%v", cursor, err)
		}
		checks := counter.count()
		if checks < 10 {
			t.Fatalf("first playback pass lacked scan checks: %d", checks)
		}

		for _, cancelOn := range []int{checks / 2, checks} {
			source, cleanup := completeSource(t, Unpaced())
			cancelScan := newScanCancelContext(cancelOn)
			cursor, err := source.handle.BeginPlaybackContext(cancelScan)
			if !errors.Is(err, context.Canceled) || cursor != nil {
				cleanup()
				t.Fatalf("canceled first pass check %d returned cursor: %v err=%v", cancelOn, cursor, err)
			}
			cancelContext, cancel := context.WithTimeout(context.Background(), time.Second)
			result, cancelErr := source.Cancel(cancelContext)
			cancel()
			cleanup()
			if cancelErr != nil || result.Outcome != OutcomeCanceled || result.Accounting.CanceledRuns != 1 {
				t.Fatalf("first-pass cancellation check %d = %+v err=%v", cancelOn, result, cancelErr)
			}
		}
	})

	t.Run("cancel timeout fabricates no terminal and retry joins", func(t *testing.T) {
		pace, _ := FinitePace(1, 1)
		source, cleanup := completeSource(t, pace)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := source.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		entered, release := make(chan struct{}), make(chan struct{})
		source.pacer.wait = func(ctx context.Context, _ time.Duration) error {
			close(entered)
			<-release
			return ctx.Err()
		}
		stepDone := make(chan error, 1)
		go func() {
			_, err := source.Step(context.Background())
			stepDone <- err
		}()
		<-entered
		timeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Millisecond)
		result, err := source.Cancel(timeout)
		cancelTimeout()
		if !errors.Is(err, context.DeadlineExceeded) || result.Outcome != "" || result.Accounting != (Accounting{}) {
			t.Fatalf("timed-out cancel fabricated result: %+v err=%v", result, err)
		}
		close(release)
		if err := <-stepDone; !errors.Is(err, context.Canceled) {
			t.Fatalf("active step did not stop: %v", err)
		}
		join, cancelJoin := context.WithTimeout(context.Background(), time.Second)
		defer cancelJoin()
		result, err = source.Cancel(join)
		if err != nil || result.Outcome != OutcomeCanceled || result.Status.Lifecycle != "ended" || result.Accounting.CanceledRuns != 1 ||
			result.Accounting.ActiveGroup != 0 || result.Accounting.PlannedGroups != result.Accounting.CompletedGroups+result.Accounting.RemainingGroups {
			t.Fatalf("cancel retry = %+v err=%v", result, err)
		}
	})

	t.Run("cancellation before failure linkage remains canceled", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := source.clock.advance(source.start.Start().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		stepContext, cancelStep := context.WithCancel(context.Background())
		source.beforeFailureAdmission = cancelStep
		if _, err := source.Step(stepContext); err == nil {
			t.Fatal("clock contradiction succeeded")
		}
		source.beforeFailureAdmission = nil
		if source.resultSealed || source.engine.ObserveReplay().Lifecycle != "replaying" {
			t.Fatalf("unlinked failure sealed: result=%+v status=%+v", source.sealedResult, source.engine.ObserveReplay())
		}
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		result, err := source.Cancel(cancelContext)
		if err != nil || result.Outcome != OutcomeCanceled || result.Accounting.FailedRuns != 0 || result.Accounting.CanceledRuns != 1 ||
			result.Accounting.ActiveGroup != 0 || result.Status.Lifecycle != "ended" {
			t.Fatalf("pre-link failure cancellation = %+v err=%v", result, err)
		}
	})

	t.Run("linked failure deadline is joined as failed", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := source.clock.advance(source.start.Start().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		stepContext, cancelStep := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancelStep()
		source.afterFailureAdmission = func() { <-stepContext.Done() }
		if _, err := source.Step(stepContext); err == nil {
			t.Fatal("clock contradiction succeeded")
		}
		source.afterFailureAdmission = nil
		if source.resultSealed {
			t.Fatalf("deadline fabricated failure: %+v", source.sealedResult)
		}
		join, cancelJoin := context.WithTimeout(context.Background(), time.Second)
		defer cancelJoin()
		result, err := source.Cancel(join)
		if err != nil || result.Outcome != OutcomeFailed || result.Reason != ReasonClock || result.Accounting.FailedRuns != 1 ||
			result.Accounting.CanceledRuns != 0 || result.Accounting.ActiveGroup != 0 || result.Status.Lifecycle != "suppressed" {
			t.Fatalf("linked failure join = %+v err=%v", result, err)
		}
	})

	t.Run("run and concurrent cancel return one sealed result", func(t *testing.T) {
		pace, _ := FinitePace(1, 1)
		source, cleanup := completeSource(t, pace)
		defer cleanup()
		entered := make(chan struct{})
		var once sync.Once
		source.pacer.wait = func(ctx context.Context, _ time.Duration) error {
			once.Do(func() { close(entered) })
			<-ctx.Done()
			return ctx.Err()
		}
		type runReply struct {
			result Result
			err    error
		}
		runDone := make(chan runReply, 1)
		go func() {
			result, err := source.Run(context.Background())
			runDone <- runReply{result: result, err: err}
		}()
		<-entered
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		canceled, err := source.Cancel(cancelContext)
		runResult := <-runDone
		if err != nil || !errors.Is(runResult.err, context.Canceled) || canceled.Outcome != OutcomeCanceled ||
			!reflect.DeepEqual(runResult.result, canceled) || canceled.Accounting.ActiveGroup != 0 {
			t.Fatalf("run/cancel disagreement: run=%+v runErr=%v cancel=%+v cancelErr=%v", runResult.result, runResult.err, canceled, err)
		}
	})

	t.Run("post-link terminal result wins", func(t *testing.T) {
		source, cleanup := completeSource(t, Unpaced())
		defer cleanup()
		_, completed := stepAll(t, source)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result, err := source.Cancel(ctx)
		if err != nil || result.Outcome != OutcomeComplete || !reflect.DeepEqual(result, completed) || result.Accounting.CanceledRuns != 0 {
			t.Fatalf("post-terminal cancel displaced success: completed=%+v cancel=%+v err=%v", completed, result, err)
		}
	})

	t.Run("post-link terminal deadline retry preserves success", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		finishContext, cancelFinish := context.WithTimeout(context.Background(), 5*time.Millisecond)
		source.afterRequestedEndAdmission = func() { <-finishContext.Done() }
		result, err := source.Finish(finishContext)
		cancelFinish()
		if !errors.Is(err, context.DeadlineExceeded) || result.Outcome != "" {
			t.Fatalf("terminal deadline fabricated result: %+v err=%v", result, err)
		}
		join, cancelJoin := context.WithTimeout(context.Background(), time.Second)
		defer cancelJoin()
		result, err = source.Cancel(join)
		if err != nil || result.Outcome != OutcomeComplete || result.Completion != CompletionRequestedEnd || result.Accounting.CanceledRuns != 0 {
			t.Fatalf("terminal deadline retry displaced result: %+v err=%v", result, err)
		}
	})
}

// TestC4PREFIXEND01RequestedEnd is P-C4-PREFIX-END. It proves that a complete
// source remains fully trusted while only its exact requested prefix reaches
// the engine, and that cancellation or any prefix/suffix/end contradiction
// cannot retain a successful completion claim.
func TestC4PREFIXEND01RequestedEnd(t *testing.T) {
	t.Run("exact prefix success never applies suffix", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSourceWithDelay(t, time.Second, time.Second)
		defer cleanup()
		trace, result := stepAll(t, source)
		requested := source.start.RequestedEnd()
		a := result.Accounting
		if result.Outcome != OutcomeComplete || result.Completion != CompletionRequestedEnd || result.Reason != ReasonNone ||
			result.Status.Completion != engine.ReplayCompletionRequestedEnd || result.Status.Lifecycle != "ended" ||
			result.Status.LastLogicalTime != requested || result.Status.CommittedT == nil || *result.Status.CommittedT != requested.Add(-time.Second) ||
			result.Status.LastOrdinal != 1 || result.Status.AggregateInserted != 1 || result.Status.PresentSlots != 1 || result.Status.ProvenAbsentSlots != 1 ||
			len(trace) != 2 || trace[0].LogicalTime != source.start.Start() || trace[1].LogicalTime != requested ||
			a.ArtifactRecords != 2 || a.CompletedRecordDispositions != 1 || a.IntentionallyUnappliedSuffixRecords != 1 || a.UnreadRecords != 0 ||
			a.ArtifactRecords != a.CompletedRecordDispositions+a.IntentionallyUnappliedSuffixRecords+a.UnreadRecords ||
			a.PlannedGroups != 2 || a.CompletedGroups != 2 || a.ActiveGroup != 0 || a.RemainingGroups != 0 || a.CompletedRuns != 1 {
			t.Fatalf("requested prefix success = result=%+v trace=%+v", result, trace)
		}
		view := source.engine.ObserveReplayDeterministic()
		for _, symbol := range view.Canonical {
			if symbol.Symbol == "AAA" && len(symbol.Records) != 1 {
				t.Fatalf("suffix aggregate reached canonical state: %+v", symbol)
			}
		}
	})

	t.Run("artifact end parity", func(t *testing.T) {
		ordinary, ordinaryCleanup := completeSource(t, Unpaced())
		defer ordinaryCleanup()
		explicitBase, _, explicitCleanup := completeSourceWithPath(t, Unpaced())
		defer explicitCleanup()
		explicit, err := NewSourceThrough(explicitBase.handle, explicitBase.engine, explicitBase.clock, Unpaced(), explicitBase.handle.Metadata().ReplayEnd)
		if err != nil {
			t.Fatal(err)
		}
		ordinaryTrace, ordinaryResult := stepAll(t, ordinary)
		explicitTrace, explicitResult := stepAll(t, explicit)
		if ordinaryResult.Completion != CompletionArtifactEnd || explicitResult.Completion != CompletionArtifactEnd ||
			ordinaryResult.Status.Completion != engine.ReplayCompletionArtifactEnd || explicitResult.Status.Completion != engine.ReplayCompletionArtifactEnd ||
			!reflect.DeepEqual(ordinaryTrace, explicitTrace) || !reflect.DeepEqual(ordinaryResult, explicitResult) {
			t.Fatalf("O1=R changed artifact end:\nordinary=%+v\nexplicit=%+v", ordinaryResult, explicitResult)
		}
	})

	t.Run("invalid ends fail before replay mutation", func(t *testing.T) {
		base, _, cleanup := completeSourceWithPath(t, Unpaced())
		defer cleanup()
		start, end := base.handle.Metadata().ReplayStart, base.handle.Metadata().ReplayEnd
		ambiguous := start.Add(time.Second).In(time.FixedZone("not-utc", 3600))
		for name, requested := range map[string]time.Time{
			"zero": {}, "at start": start, "subsecond": start.Add(time.Second + time.Nanosecond),
			"after artifact": end.Add(time.Second), "non-UTC": ambiguous,
		} {
			t.Run(name, func(t *testing.T) {
				if source, err := NewSourceThrough(base.handle, base.engine, base.clock, Unpaced(), requested); err == nil || source != nil {
					t.Fatalf("invalid requested end accepted: %s", requested)
				}
				status := base.engine.ObserveReplay()
				if status.Lifecycle != "initializing" || status.LastOrdinal != 0 || status.AggregateInserted != 0 || status.Completion != engine.ReplayCompletionNone {
					t.Fatalf("invalid end mutated replay state: %+v", status)
				}
			})
		}
		partial, partialCleanup := partialSource(t)
		defer partialCleanup()
		if source, err := NewSourceThrough(partial.handle, partial.engine, partial.clock, Unpaced(), partial.handle.Metadata().ReplayStart.Add(time.Second)); err == nil || source != nil {
			t.Fatal("partial artifact accepted a prefix requested end")
		}
	})

	t.Run("missing prefix evidence is unavailable", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if result, err := source.Finish(context.Background()); err == nil || result.Outcome != "" || result.Completion != "" {
			t.Fatalf("missing prefix evidence reached terminal result: err=%v result=%+v", err, result)
		}
		status := source.engine.ObserveReplay()
		if status.Lifecycle != "replaying" || status.Completion != engine.ReplayCompletionNone || status.AggregateInserted != 0 {
			t.Fatalf("missing prefix evidence mutated completion: %+v", status)
		}
	})

	t.Run("mutated suffix cannot yield success", func(t *testing.T) {
		source, path, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		value, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := []byte(strings.Replace(string(value), `"ordinal":2`, `"ordinal":3`, 1))
		if slices.Equal(value, mutated) || len(value) != len(mutated) {
			t.Fatal("suffix ordinal mutation was not exact")
		}
		if err := os.WriteFile(path, mutated, 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := source.Finish(context.Background())
		if err == nil || result.Outcome != OutcomeFailed || result.Completion != "" || result.Reason != ReasonArtifactEnd ||
			result.Status.Lifecycle != "suppressed" || result.Status.Completion != engine.ReplayCompletionNone ||
			result.Status.FailureReason != engine.ReplayFailureArtifactEnd ||
			result.Status.AggregateInserted != 1 || result.Status.LastOrdinal != 1 || result.Accounting.UnreadRecords != 1 ||
			result.Accounting.IntentionallyUnappliedSuffixRecords != 0 {
			t.Fatalf("mutated suffix retained success: err=%v result=%+v", err, result)
		}
	})

	t.Run("requested-end contradiction suppresses", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		foreign, _, foreignCleanup := requestedCompleteSource(t, 2*time.Second)
		defer foreignCleanup()
		if source.artifactID != foreign.artifactID {
			t.Fatalf("contradiction fixtures differ: %s %s", source.artifactID, foreign.artifactID)
		}
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		cursor, err := foreign.handle.BeginPlaybackThrough(foreign.requestedEnd)
		if err != nil {
			t.Fatal(err)
		}
		startEvidence, err := cursor.Start()
		if err != nil {
			t.Fatal(err)
		}
		for group := startEvidence.Start(); !group.After(startEvidence.RequestedEnd()); group = group.Add(time.Second) {
			for {
				if _, ok, err := cursor.NextRecord(group); err != nil {
					t.Fatal(err)
				} else if !ok {
					break
				}
			}
			if _, err := cursor.FinishGroup(group); err != nil {
				t.Fatal(err)
			}
		}
		contradiction, err := cursor.RequestedEnd()
		if err != nil {
			t.Fatal(err)
		}
		admission, completion := source.engine.AdmitReplayRequestedEnd(context.Background(), contradiction)
		if admission != engine.AdmissionAdmitted || completion == nil {
			t.Fatalf("contradictory evidence admission = %s", admission)
		}
		disposition := <-completion
		status := source.engine.ObserveReplay()
		if disposition.Code != engine.DispositionReplayFailed || status.Lifecycle != "suppressed" ||
			status.Suppression != engine.SuppressionTerminalReplayFailure || status.Completion != engine.ReplayCompletionNone {
			t.Fatalf("requested-end contradiction retained completion: disposition=%+v status=%+v", disposition, status)
		}
	})

	t.Run("cancellation before terminal linkage remains canceled", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		source.beforeRequestedEndAdmission = cancel
		result, err := source.Finish(ctx)
		if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(result, Result{}) || source.resultSealed {
			t.Fatalf("requested-end cancellation = err=%v result=%+v", err, result)
		}
		cancelContext, cancelSource := context.WithTimeout(context.Background(), time.Second)
		defer cancelSource()
		result, err = source.Cancel(cancelContext)
		if err != nil || result.Outcome != OutcomeCanceled || result.Completion != "" || result.Accounting.CanceledRuns != 1 ||
			result.Accounting.CompletedRuns != 0 || result.Status.Completion != engine.ReplayCompletionNone {
			t.Fatalf("requested-end explicit cancel = err=%v result=%+v", err, result)
		}
	})

	t.Run("cancellation after terminal linkage preserves requested-end success", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		source.afterRequestedEndAdmission = cancel
		result, err := source.Finish(ctx)
		if err != nil || ctx.Err() != context.Canceled || result.Outcome != OutcomeComplete || result.Completion != CompletionRequestedEnd ||
			result.Accounting.CompletedRuns != 1 || result.Accounting.CanceledRuns != 0 || result.Status.Completion != engine.ReplayCompletionRequestedEnd {
			t.Fatalf("post-link cancellation displaced terminal result: err=%v ctx=%v result=%+v", err, ctx.Err(), result)
		}
	})

	t.Run("terminal admission at a clock other than requested end fails", func(t *testing.T) {
		source, _, cleanup := requestedCompleteSource(t, time.Second)
		defer cleanup()
		if err := source.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		for !source.nextGroup.After(source.start.RequestedEnd()) {
			if _, err := source.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		if err := source.clock.advance(source.start.RequestedEnd().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		result, err := source.Finish(context.Background())
		if err == nil || result.Outcome != OutcomeFailed || result.Completion != "" || result.Reason != ReasonEngine ||
			result.Accounting.CompletedRuns != 0 || result.Status.Completion != engine.ReplayCompletionNone || result.Status.Lifecycle != "suppressed" {
			t.Fatalf("wrong-clock terminal admission retained completion: err=%v result=%+v", err, result)
		}
	})

	t.Run("clock and engine rejection retain no completion", func(t *testing.T) {
		clockSource, _, clockCleanup := requestedCompleteSource(t, time.Second)
		defer clockCleanup()
		if err := clockSource.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := clockSource.clock.advance(clockSource.start.Start().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := clockSource.Step(context.Background()); err == nil {
			t.Fatal("requested-end clock contradiction succeeded")
		}
		clockResult := clockSource.result(OutcomeFailed, clockSource.currentReason())
		if clockResult.Completion != "" || clockResult.Status.Completion != engine.ReplayCompletionNone || clockResult.Status.Lifecycle != "suppressed" {
			t.Fatalf("clock contradiction retained completion: %+v", clockResult)
		}

		engineSource, _, engineCleanup := requestedCompleteSource(t, time.Second)
		defer engineCleanup()
		if err := engineSource.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		engineSource.engine.Close()
		if _, err := engineSource.Step(context.Background()); err == nil {
			t.Fatal("closed engine accepted requested prefix group")
		}
		engineResult := engineSource.result(OutcomeFailed, engineSource.currentReason())
		if engineResult.Completion != "" || engineResult.Status.Completion != engine.ReplayCompletionNone || engineResult.Accounting.CompletedRuns != 0 {
			t.Fatalf("engine rejection retained completion: %+v", engineResult)
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
	for !source.nextGroup.After(source.start.RequestedEnd()) {
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
	return completeSourceWithPathAndDelay(t, pace, 0)
}

func completeSourceWithPathAndDelay(t *testing.T, pace Pace, delay time.Duration) (*Source, string, func()) {
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

func requestedCompleteSource(t *testing.T, offset time.Duration) (*Source, string, func()) {
	return requestedCompleteSourceWithDelay(t, offset, 0)
}

func requestedCompleteSourceWithDelay(t *testing.T, offset, delay time.Duration) (*Source, string, func()) {
	t.Helper()
	base, path, cleanup := completeSourceWithPathAndDelay(t, Unpaced(), delay)
	requested := base.handle.Metadata().ReplayStart.Add(offset)
	source, err := NewSourceThrough(base.handle, base.engine, base.clock, Unpaced(), requested)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	return source, path, func() {
		base.engine.Close()
		_ = base.engine.Wait(context.Background())
		cleanup()
	}
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
	return sourceFromHandleWithDelay(t, handle, binding, start, pace, 0)
}

func sourceFromHandleWithDelay(t *testing.T, handle *replayartifact.Handle, binding reference.Binding, start time.Time, pace Pace, delay time.Duration) *Source {
	t.Helper()
	clock, err := NewSimulatedClock(start)
	if err != nil {
		t.Fatal(err)
	}
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
