package replay

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
)

// TestC7REPLAY01CheckpointContinuationDifferential is P-C7-REPLAY. It proves
// exact-cutoff continuation through the ordinary C4 runner at two paces,
// rejects repeated/gapped artifacts before engine delivery, and permits only
// complete-from-session-start fallback. It makes no live-epoch or latency claim.
func TestC7REPLAY01CheckpointContinuationDifferential(t *testing.T) {
	t.Skip("unsupported replay/checkpoint differential no longer constrains the live engine after LBR-B3 removal")
	binding := replayBinding(t, []string{"AAA", "BBB"})
	s, t0, end := binding.SessionStart(), binding.SessionStart().Add(2*time.Second), binding.SessionStart().Add(4*time.Second)
	full := compileC7ReplayArtifact(t, binding, s, end)
	defer full.Close()
	continuation := compileC7ReplayArtifact(t, binding, t0, end)
	defer continuation.Close()
	gapped := compileC7ReplayArtifact(t, binding, t0.Add(time.Second), end)
	defer gapped.Close()
	image := livePrefixCheckpointForReplay(t, binding, t0)

	pace, err := FinitePace(1_000_000, 1)
	if err != nil {
		t.Fatal(err)
	}
	for name, selected := range map[string]Pace{"unpaced": Unpaced(), "finite": pace} {
		t.Run(name, func(t *testing.T) {
			fullSource := replaySourceWithOptionalCheckpoint(t, full, binding, s, selected, nil)
			fullTrace, fullResult := stepAll(t, fullSource)
			if fullResult.Outcome != OutcomeComplete {
				t.Fatalf("full=%+v", fullResult)
			}

			checkpointSource := replaySourceWithOptionalCheckpoint(t, continuation, binding, t0, selected, &image)
			checkpointTrace, checkpointResult := stepAll(t, checkpointSource)
			if checkpointResult.Outcome != OutcomeComplete {
				t.Fatalf("checkpoint=%+v", checkpointResult)
			}
			for index := range checkpointTrace {
				fullIndex := index + int(t0.Sub(s)/time.Second)
				if checkpointTrace[index].LogicalTime != fullTrace[fullIndex].LogicalTime || !reflect.DeepEqual(checkpointTrace[index].Status.Evaluation, fullTrace[fullIndex].Status.Evaluation) {
					t.Fatalf("logical checkpoint %d differs checkpoint=%+v full=%+v", index, checkpointTrace[index], fullTrace[fullIndex])
				}
			}
			got, want := checkpointSource.engine.ObserveReplayDeterministic(), fullSource.engine.ObserveReplayDeterministic()
			if !reflect.DeepEqual(replaySemanticCanonical(got.Canonical), replaySemanticCanonical(want.Canonical)) || !reflect.DeepEqual(got.Evaluation, want.Evaluation) || !reflect.DeepEqual(got.Publication.AggregateEvaluation, want.Publication.AggregateEvaluation) {
				t.Fatalf("checkpoint/full end differ\ngot=%+v\nwant=%+v", got, want)
			}
		})
	}

	// Exact interval identity—not first nonempty record time—owns the cutoff.
	badOwner, badClock := boundReplayEngine(t, binding, t0)
	installed := installReplayCheckpoint(t, badOwner, image)
	if _, err := NewCheckpointSource(full, badOwner, badClock, Unpaced(), installed); err == nil {
		t.Fatal("pre-T0 repeated continuation accepted")
	}
	if _, err := NewCheckpointSource(gapped, badOwner, badClock, Unpaced(), installed); err == nil {
		t.Fatal("post-T0 gapped continuation accepted")
	}
	badOwner.Close()
	_ = badOwner.Wait(context.Background())

	// The engine, not a caller-supplied fact, owns prefix authority. A forged
	// fact cannot make a partial artifact complete on a fresh engine, and an
	// installed checkpoint cannot be paired with a full-from-session artifact.
	foreignOwner, foreignClock := boundReplayEngine(t, binding, t0)
	forged := installed
	foreignSource, err := NewCheckpointSource(continuation, foreignOwner, foreignClock, Unpaced(), forged)
	if err != nil {
		t.Fatal(err)
	}
	if result, _ := foreignSource.Run(context.Background()); result.Outcome == OutcomeComplete {
		t.Fatalf("foreign checkpoint fact authorized partial replay: %+v", result)
	}
	installedOwner, installedClock := boundReplayEngine(t, binding, t0)
	installedFact := installReplayCheckpoint(t, installedOwner, image)
	if installedFact.T0 != t0 {
		t.Fatal("installed cutoff")
	}
	fullOnCheckpoint, err := NewSource(full, installedOwner, installedClock, Unpaced())
	if err != nil {
		t.Fatal(err)
	}
	if result, _ := fullOnCheckpoint.Run(context.Background()); result.Outcome == OutcomeComplete {
		t.Fatalf("installed checkpoint accepted mismatched full replay: %+v", result)
	}

	fallbackOwner, fallbackClock := boundReplayEngine(t, binding, s)
	fallback, err := NewCompleteFallbackSource(full, fallbackOwner, fallbackClock, Unpaced())
	result, runErr := fallback.Run(context.Background())
	if err != nil || runErr != nil || result.Outcome != OutcomeComplete {
		t.Fatalf("complete fallback err=%v", err)
	}
	partial := partialC7ReplayArtifact(t, binding, s, end)
	defer partial.Close()
	partialOwner, partialClock := boundReplayEngine(t, binding, s)
	if _, err := NewCompleteFallbackSource(partial, partialOwner, partialClock, Unpaced()); err == nil {
		t.Fatal("partial fallback accepted")
	}
	partialOwner.Close()
	_ = partialOwner.Wait(context.Background())
}

type replayCanonicalSemantic struct {
	Symbol               string
	PriorStatus          string
	PriorClose           float64
	Records              []engine.ReplayCanonicalRecord
	LatestWindowStart    time.Time
	LatestValues         engine.AggregateValues
	CommittedWindowStart time.Time
	ProvenAbsent         uint64
	Features             engine.ReplayFeatureView
	Qualification        engine.ReplayQualificationView
}

func replaySemanticCanonical(values []engine.ReplayCanonicalSymbol) []replayCanonicalSemantic {
	result := make([]replayCanonicalSemantic, len(values))
	for index, value := range values {
		records := value.Records
		for recordIndex := range records {
			records[recordIndex].FirstSource = ""
			records[recordIndex].FirstDeliveryTime = time.Time{}
			records[recordIndex].FirstReplayOrdinal = 0
			records[recordIndex].AuthoritySource = ""
			records[recordIndex].AuthorityDeliveryTime = time.Time{}
			records[recordIndex].AuthorityReplayOrdinal = 0
		}
		result[index] = replayCanonicalSemantic{value.Symbol, string(value.PriorStatus), value.PriorClose, records, value.LatestWindowStart, value.LatestValues, value.CommittedWindowStart, value.ProvenAbsentSlots, value.Features, value.Qualification}
	}
	return result
}

func replaySourceWithOptionalCheckpoint(t *testing.T, handle *replayartifact.Handle, binding reference.Binding, start time.Time, pace Pace, image *checkpoint.Image) *Source {
	t.Helper()
	owner, clock := boundReplayEngine(t, binding, start)
	if image == nil {
		source, err := NewSource(handle, owner, clock, pace)
		if err != nil {
			t.Fatal(err)
		}
		return source
	}
	installed := installReplayCheckpoint(t, owner, *image)
	source, err := NewCheckpointSource(handle, owner, clock, pace, installed)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func boundReplayEngine(t *testing.T, binding reference.Binding, start time.Time) (*engine.Engine, *SimulatedClock) {
	t.Helper()
	clock, err := NewSimulatedClock(start)
	if err != nil {
		t.Fatal(err)
	}
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: 128, RequiredReserve: 4, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionBindingInstalled {
		t.Fatal("replay binding")
	}
	return owner, clock
}

func installReplayCheckpoint(t *testing.T, owner *engine.Engine, image checkpoint.Image) engine.InstalledCheckpointFact {
	t.Helper()
	admission, completion := owner.AdmitCheckpointInstall(context.Background(), checkpoint.Candidate{Image: image, Integrity: true, Checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	result := <-completion
	if result.Disposition != engine.CheckpointInstalled {
		t.Fatalf("replay install=%+v", result)
	}
	return result.Fact
}

func livePrefixCheckpointForReplay(t *testing.T, binding reference.Binding, t0 time.Time) checkpoint.Image {
	t.Helper()
	now := t0
	delay := time.Duration(0)
	owner, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return now }, Capacity: 128, RequiredReserve: 4, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Close(); _ = owner.Wait(context.Background()) }()
	_, bindingCompletion := owner.AdmitBinding(context.Background(), engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if (<-bindingCompletion).Code != engine.DispositionBindingInstalled {
		t.Fatal("live prefix binding")
	}
	for _, input := range []engine.ConnectionControlInput{
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.ConnectionAttempt, ConnectionEpoch: 1, ReceiptTime: t0, CommandToken: 1, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateCommandWriteResult, ConnectionEpoch: 1, ReceiptTime: t0, CommandToken: 2, Outcome: engine.ControlSucceeded},
		{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: engine.AggregateSubscriptionResult, ConnectionEpoch: 1, Position: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ReceiptTime: t0, CommandToken: 2, Outcome: engine.ControlSucceeded},
	} {
		admission, completion := owner.AdmitConnectionControl(context.Background(), input)
		if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionConnectionControlApplied {
			t.Fatal("live prefix control")
		}
	}
	seconds := int64(t0.Sub(binding.SessionStart()) / time.Second)
	budgets := engine.HydrationPlanBudgets{Workers: 2, RowsPerChunk: 64, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: int64(len(binding.UniverseSymbols())) * seconds, MaximumResidentRecords: 2 * seconds}
	_, planCompletion := owner.AdmitHydrationPlan(context.Background(), engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1, Budgets: budgets})
	plan := <-planCompletion
	var fence engine.HydrationFenceCommand
	for _, token := range plan.Plan.Requests() {
		if token.Symbol() == "AAA" {
			rows := make([]engine.HydrationRow, 0, seconds)
			for at := token.Start(); at.Before(token.End()); at = at.Add(time.Second) {
				row, err := engine.NewHydrationRow("AAA", at, at.Add(time.Second), replayAggregateValues(at))
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, row)
			}
			chunk, _ := engine.NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, int64(len(rows)), rows)
			_, completion := owner.AdmitHydrationChunk(context.Background(), chunk)
			if (<-completion).Code != engine.DispositionHydrationChunkApplied {
				t.Fatal("prefix chunk")
			}
			terminal, _ := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedValue, engine.HydrationReasonNone, 1, 1, 100, int64(len(rows)), 1, int64(len(rows)))
			_, completion = owner.AdmitHydrationTerminal(context.Background(), terminal)
			fence = (<-completion).FenceCommand
		} else {
			terminal, _ := engine.NewHydrationTerminalInput(token, token.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
			_, completion := owner.AdmitHydrationTerminal(context.Background(), terminal)
			fence = (<-completion).FenceCommand
		}
	}
	fenceInput, _ := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, 1, 1, t0)
	_, fenceCompletion := owner.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if (<-fenceCompletion).Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatal("prefix fence")
	}
	_, projectionCompletion := owner.AdmitCheckpointProjection(context.Background())
	projection := <-projectionCompletion
	if projection.Disposition != engine.CheckpointProjected {
		t.Fatalf("prefix projection=%+v", projection)
	}
	return projection.Image
}

func replayAggregateValues(at time.Time) engine.AggregateValues {
	price := 10 + float64(at.Unix()%10)/10
	return engine.AggregateValues{Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 100, VWAP: price, AverageTradeSize: 10, ATSProvenance: engine.ATSRESTFloorVolumeOverTrades}
}

func compileC7ReplayArtifact(t *testing.T, binding reference.Binding, start, end time.Time) *replayartifact.Handle {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		parts := splitReplayPath(request.URL.Path)
		symbol := parts[4]
		fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[`, symbol)
		if symbol == "AAA" {
			for at, first := start, true; at.Before(end); at = at.Add(time.Second) {
				if !first {
					fmt.Fprint(writer, ",")
				}
				first = false
				v := replayAggregateValues(at)
				fmt.Fprintf(writer, `{"t":%d,"o":%g,"h":%g,"l":%g,"c":%g,"v":%g,"vw":%g,"n":10}`, at.UnixMilli(), v.Open, v.High, v.Low, v.Close, v.Volume, v.VWAP)
			}
		}
		fmt.Fprint(writer, `]}`)
	}))
	downloader, err := massive.NewOfflineDownloader(server.URL, func() (string, error) { return "c7-replay", nil }, server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	directory := t.TempDir()
	maximumRecords := int64(len(binding.UniverseSymbols())) * int64(end.Sub(start)/time.Second)
	limits := replayartifact.Limits{MaximumNormalizedRecords: maximumRecords, MaximumResponseBytes: 1 << 20, MaximumArtifactBytes: 1 << 20, MaximumTemporaryBytes: 1 << 20, MaximumTemporaryFiles: 8, MaximumInMemoryRecords: maximumRecords}
	compiled := replayartifact.Compile(context.Background(), replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: 2, DestinationDirectory: directory, Limits: limits}, downloader)
	server.Close()
	if compiled.State != replayartifact.CompileComplete {
		t.Fatalf("compile=%+v", compiled)
	}
	handle, err := replayartifact.OpenValidated(compiled.Path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: limits.MaximumNormalizedRecords})
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

func splitReplayPath(path string) []string {
	parts := make([]string, 0, 10)
	current := ""
	for _, value := range path {
		if value == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(value)
		}
	}
	return append(parts, current)
}

func partialC7ReplayArtifact(t *testing.T, binding reference.Binding, start, end time.Time) *replayartifact.Handle {
	t.Helper()
	value, _, err := replayartifact.BuildPartial(replayartifact.PartialInput{Binding: binding, Start: start, End: end, DeclaredSymbols: []string{"AAA"}, MaximumBytes: 1 << 20, MaximumRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	if err := os.WriteFile(path, value, 0600); err != nil {
		t.Fatal(err)
	}
	handle, err := replayartifact.OpenValidated(path, replayartifact.ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.PartialSynthetic, MaximumBytes: 1 << 20, MaximumRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	return handle
}
