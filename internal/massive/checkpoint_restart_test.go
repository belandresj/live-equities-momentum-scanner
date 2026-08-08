package massive

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestC7LIVE01CheckpointCatchupEquivalence is P-C7-LIVE. It composes the
// accepted fake C5 transport, real C6 HTTP acquisition/normalization/worker
// ledger, causal live tail and fence, and ordinary C2/C3 owner. It does not
// claim provider latency, T/Q readiness, or deployed capacity.
func TestC7LIVE01CheckpointCatchupEquivalence(t *testing.T) {
	binding := component4TestBinding(t, []string{"AAA", "BBB"})
	s, t0, r := binding.SessionStart(), binding.SessionStart().Add(10*time.Second), binding.SessionStart().Add(20*time.Second)
	server := checkpointHydrationServer(t, map[string]map[int64]float64{"AAA": {t0.Add(-time.Second).UnixMilli(): 10, t0.UnixMilli(): 11}})
	defer server.Close()

	// Produce a real committed T0 checkpoint through fresh C5/C6 startup.
	prefixNow := t0
	prefix := newBoundLiveEngine(t, binding, &prefixNow)
	prefixAttempt, prefixSocket := startAggregateEpochAt(t, prefix, binding, t0)
	prefixFence := runCheckpointHydration(t, prefix, binding, prefixAttempt.Epoch(), engine.HydrationFreshBootstrap, server)
	finishCheckpointFence(t, prefix, prefixAttempt, prefixFence)
	projection := projectCheckpointLive(t, prefix)
	if projection.Image.T0 != t0 {
		t.Fatalf("prefix T0=%s", projection.Image.T0)
	}

	// Manifest latest is structurally valid but binding-incompatible; bounded
	// authority-order selection must install the compatible previous candidate.
	store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: t.TempDir(), BindingIdentity: binding.Identity(), ArtifactByteLimit: 32 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Write(context.Background(), projection.Image); got.Disposition != checkpoint.WriteCompleted {
		t.Fatalf("write previous=%+v", got)
	}
	bad := projection.Image.Clone()
	bad.Sequence++
	bad.Binding.TradingDate = "2099-01-01"
	if got := store.Write(context.Background(), bad); got.Disposition != checkpoint.WriteCompleted {
		t.Fatalf("write incompatible latest=%+v", got)
	}

	restartNow := r
	restarted := newBoundLiveEngine(t, binding, &restartNow)
	loaded := [2]checkpoint.LoadResult{store.LoadCandidate(context.Background(), checkpoint.CandidateLatest)}
	var installed engine.CheckpointInstallResult
	if loaded[0].Candidate.Integrity {
		installed = installCheckpointLive(t, restarted, loaded[0].Candidate)
	}
	if installed.Disposition != engine.CheckpointInstalled {
		loaded[1] = store.LoadCandidate(context.Background(), checkpoint.CandidatePrevious)
		if loaded[1].Candidate.Integrity {
			installed = installCheckpointLive(t, restarted, loaded[1].Candidate)
		}
	}
	if loaded[0].Disposition != checkpoint.LoadedLatest || loaded[1].Disposition != checkpoint.LoadedPrevious || installed.Disposition != engine.CheckpointInstalled || installed.Fact.ArtifactSequence != projection.Image.Sequence {
		t.Fatalf("candidate selection latest=%+v previous=%+v install=%+v", loaded[0], loaded[1], installed)
	}

	// Restarted and uninterrupted engines consume the same [T0,R) historical
	// facts, overlapping new-epoch live correction, and causal fence.
	restartAttempt, restartSocket := startAggregateEpochAt(t, restarted, binding, r)
	restartFence := runCheckpointHydration(t, restarted, binding, restartAttempt.Epoch(), engine.HydrationCheckpointCatchup, server)
	if restartFence.start != t0 || restartFence.end != r || restartFence.planned != 2 {
		t.Fatalf("checkpoint plan=%+v", restartFence)
	}
	deliverCheckpointLiveCorrection(t, restarted, restartAttempt, restartSocket, "AAA", t0, 12)
	finishCheckpointFence(t, restarted, restartAttempt, restartFence.command)

	oracleNow := r
	oracle := newBoundLiveEngine(t, binding, &oracleNow)
	oracleAttempt, oracleSocket := startAggregateEpochAt(t, oracle, binding, r)
	oracleFence := runCheckpointHydration(t, oracle, binding, oracleAttempt.Epoch(), engine.HydrationFreshBootstrap, server)
	if oracleFence.start != s || oracleFence.end != r {
		t.Fatalf("fresh plan=%+v", oracleFence)
	}
	deliverCheckpointLiveCorrection(t, oracle, oracleAttempt, oracleSocket, "AAA", t0, 12)
	finishCheckpointFence(t, oracle, oracleAttempt, oracleFence.command)

	got, want := restarted.ObserveReplayDeterministic(), oracle.ObserveReplayDeterministic()
	gotSemantic, wantSemantic := projectCheckpointLive(t, restarted).Image.Symbols, projectCheckpointLive(t, oracle).Image.Symbols
	if !reflect.DeepEqual(gotSemantic, wantSemantic) || !reflect.DeepEqual(got.Evaluation, want.Evaluation) ||
		got.Publication.Lifecycle != "live" || want.Publication.Lifecycle != "live" ||
		!reflect.DeepEqual(got.Publication.AggregateEvaluation, want.Publication.AggregateEvaluation) || restarted.ObserveReplay().TQIntentRows != 0 || oracle.ObserveReplay().TQIntentRows != 0 {
		t.Fatalf("restart/oracle differ\ngot=%+v\nwant=%+v", got, want)
	}
	if got.Canonical[0].LatestAuthoritySource != engine.AggregateSourceLive || got.Canonical[0].LatestValues.Close != 12 || got.Canonical[1].PresentSlots != 0 {
		t.Fatalf("live authority/empty evidence=%+v", got.Canonical)
	}

	closeCheckpointAttempt(t, prefixAttempt, prefixSocket, binding, 2)
	closeCheckpointAttempt(t, restartAttempt, restartSocket, binding, 2)
	closeCheckpointAttempt(t, oracleAttempt, oracleSocket, binding, 2)
	closeCheckpointEngine(t, prefix)
	closeCheckpointEngine(t, restarted)
	closeCheckpointEngine(t, oracle)
}

// TestC7OBJECTIVE01CurrentHostRestart is P-C7-OBJECTIVE. It uses the exact
// 6,000-symbol population, a 30-second checkpoint gap, every success-bearing
// checkpoint family, and the accepted fake C5 plus real C6 acquisition,
// normalization, ledger, and ingress-fence path. Timings are current-host
// evidence only, not a portable capacity claim.
func TestC7OBJECTIVE01CurrentHostRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("C7 capacity/acceptance proof")
	}
	const (
		population   = 6_000
		trials       = 3
		trialTimeout = 2 * time.Minute
	)
	symbols := make([]string, population)
	for index := range symbols {
		symbols[index] = fmt.Sprintf("S%05d", index)
	}
	binding := component4TestBinding(t, symbols)
	t0 := binding.SessionStart().Add(20 * time.Minute)
	r := t0.Add(30 * time.Second)
	image := objectiveCheckpointImage(binding, t0)
	if err := checkpoint.ValidateSemanticImage(image); err != nil {
		t.Fatalf("objective fixture semantic validation: %v", err)
	}
	if image.Population != population || len(image.Symbols) != population || image.T0 != t0 || image.Counts != objectiveStructureCounts(image) ||
		image.Counts.TailRecords != 961 || image.Counts.PriceExtremaPoints == 0 || image.Counts.ActivityTargetBlocks == 0 ||
		image.Counts.QualificationProofs == 0 || image.Counts.QualificationDirty == 0 || image.Counts.CoverageConsequences == 0 || image.Counts.InvalidMarks == 0 ||
		image.Counts.PresenceWords != 900 || image.Counts.ProvenAbsentWords != 900 || image.Counts.ConflictWords != 900 {
		t.Fatalf("objective fixture manifest population=%d symbols=%d t0=%s counts=%+v", image.Population, len(image.Symbols), image.T0, image.Counts)
	}
	var preflight bytes.Buffer
	preflightBytes, preflightChecksum, err := checkpoint.Encode(context.Background(), &preflight, image, 64<<20)
	if err != nil || preflightBytes != int64(preflight.Len()) || preflightBytes <= 0 || len(preflightChecksum) != 64 {
		t.Fatalf("objective artifact preflight bytes=%d buffered=%d checksum=%q err=%v", preflightBytes, preflight.Len(), preflightChecksum, err)
	}
	const checkpointRecords, freshRecords = int64(population / 2), int64(population/2) * 60
	t.Logf("preflight host=%s/%s go=%s cpus=%d workers=%d population=%d gap=%s schema=%s artifact_bytes=%d payload_sha256=%s corrections=%d counts=%+v checkpoint_requests=%d checkpoint_records=%d fresh_requests=%d fresh_records=%d",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU(), min(32, max(2, runtime.NumCPU())), population, r.Sub(t0), image.SchemaVersion, preflightBytes, preflightChecksum, image.Counts.TailRecords, image.Counts, population, checkpointRecords, population, freshRecords)
	server := objectiveHydrationServer(t)
	defer server.Close()

	type segments struct {
		projection, encode, write, load, install, acknowledgement, catchup, endToEnd time.Duration
		artifactBytes                                                                int64
	}
	runs, fresh := make([]segments, trials), make([]time.Duration, trials)
	workers := min(32, max(2, runtime.NumCPU()))
	for run := range runs {
		checkpointContext, cancelCheckpoint := context.WithTimeout(context.Background(), trialTimeout)
		sourceNow := t0
		source := newBoundLiveEngineContext(t, checkpointContext, binding, &sourceNow)
		if installed := installCheckpointLiveContext(t, checkpointContext, source, checkpoint.Candidate{Image: image, Checksum: strings.Repeat("0", 64), Integrity: true}); installed.Disposition != engine.CheckpointInstalled {
			t.Fatalf("fixture install=%+v", installed)
		}
		sourceAttempt, sourceSocket := startAggregateEpochAtContext(t, checkpointContext, source, binding, t0)
		finishEmptyCheckpointHydrationContext(t, checkpointContext, source, binding, sourceAttempt)
		started := time.Now()
		_ = projectCheckpointLiveContext(t, checkpointContext, source)
		runs[run].projection = time.Since(started)

		started = time.Now()
		var encoded bytes.Buffer
		encodedBytes, encodedChecksum, err := checkpoint.Encode(checkpointContext, &encoded, image, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		if encodedBytes != preflightBytes || encodedChecksum != preflightChecksum || !bytes.Equal(encoded.Bytes(), preflight.Bytes()) {
			t.Fatalf("trial %d artifact differs from preflight bytes=%d/%d checksum=%s/%s", run+1, encodedBytes, preflightBytes, encodedChecksum, preflightChecksum)
		}
		runs[run].encode = time.Since(started)
		runs[run].artifactBytes = int64(encoded.Len())

		store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: t.TempDir(), BindingIdentity: binding.Identity(), ArtifactByteLimit: 64 << 20, OperationDeadline: 30 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		started = time.Now()
		if result := store.Write(checkpointContext, image); result.Disposition != checkpoint.WriteCompleted {
			t.Fatalf("write=%+v", result)
		}
		runs[run].write = time.Since(started)

		restartNow := r
		restart := newBoundLiveEngineContext(t, checkpointContext, binding, &restartNow)
		e2e := time.Now()
		started = time.Now()
		loaded := store.Load(checkpointContext)
		runs[run].load = time.Since(started)
		if loaded.Disposition != checkpoint.LoadedLatest {
			t.Fatalf("load=%+v", loaded)
		}
		started = time.Now()
		if installed := installCheckpointLiveContext(t, checkpointContext, restart, loaded.Candidate); installed.Disposition != engine.CheckpointInstalled {
			t.Fatalf("restart install=%+v", installed)
		}
		runs[run].install = time.Since(started)
		started = time.Now()
		restartAttempt, restartSocket := startAggregateEpochAtContext(t, checkpointContext, restart, binding, r)
		runs[run].acknowledgement = time.Since(started)
		started = time.Now()
		checkpointRun := runCheckpointHydrationWorkersContext(t, checkpointContext, restart, binding, restartAttempt.Epoch(), engine.HydrationCheckpointCatchup, server, workers)
		finishCheckpointFenceContext(t, checkpointContext, restart, restartAttempt, checkpointRun.command)
		runs[run].catchup = time.Since(started)
		runs[run].endToEnd = time.Since(e2e)
		if checkpointRun.start != t0 || checkpointRun.end != r || checkpointRun.planned != population || checkpointRun.normalized != checkpointRecords || restart.ObserveReplay().TQIntentRows != 0 {
			t.Fatalf("checkpoint run=%+v TQ=%d", checkpointRun, restart.ObserveReplay().TQIntentRows)
		}
		closeCheckpointAttemptContext(t, checkpointContext, sourceAttempt, sourceSocket, binding, 2)
		closeCheckpointAttemptContext(t, checkpointContext, restartAttempt, restartSocket, binding, 2)
		closeCheckpointEngineContext(t, checkpointContext, source)
		closeCheckpointEngineContext(t, checkpointContext, restart)
		if err := checkpointContext.Err(); err != nil {
			t.Fatalf("checkpoint trial %d exceeded %s: %v", run+1, trialTimeout, err)
		}
		cancelCheckpoint()
		t.Logf("checkpoint trial=%d segments=%+v", run+1, runs[run])

		freshContext, cancelFresh := context.WithTimeout(context.Background(), trialTimeout)
		freshNow := r
		freshEngine := newBoundLiveEngineContext(t, freshContext, binding, &freshNow)
		freshAttempt, freshSocket := startAggregateEpochAtContext(t, freshContext, freshEngine, binding, r)
		started = time.Now()
		freshRun := runCheckpointHydrationWorkersContext(t, freshContext, freshEngine, binding, freshAttempt.Epoch(), engine.HydrationFreshBootstrap, server, workers)
		finishCheckpointFenceContext(t, freshContext, freshEngine, freshAttempt, freshRun.command)
		fresh[run] = time.Since(started)
		if freshRun.start != binding.SessionStart() || freshRun.end != r || freshRun.planned != population || freshRun.normalized != freshRecords {
			t.Fatalf("fresh run=%+v", freshRun)
		}
		closeCheckpointAttemptContext(t, freshContext, freshAttempt, freshSocket, binding, 2)
		closeCheckpointEngineContext(t, freshContext, freshEngine)
		if err := freshContext.Err(); err != nil {
			t.Fatalf("fresh trial %d exceeded %s: %v", run+1, trialTimeout, err)
		}
		cancelFresh()
		t.Logf("fresh trial=%d elapsed=%s", run+1, fresh[run])
	}
	checkpointTotals := make([]time.Duration, len(runs))
	for index := range runs {
		checkpointTotals[index] = runs[index].endToEnd
		if runs[index].endToEnd > 60*time.Second {
			t.Fatalf("objective target run %d: %+v", index+1, runs[index])
		}
	}
	checkpointMedian, freshMedian := durationMedian(checkpointTotals), durationMedian(fresh)
	if checkpointMedian*5 > freshMedian*4 {
		t.Fatalf("checkpoint median %s is not at least 20%% faster than fresh median %s; checkpoint=%v fresh=%v", checkpointMedian, freshMedian, checkpointTotals, fresh)
	}
	t.Logf("host=%s/%s go=%s cpus=%d workers=%d population=%d gap=%s artifact_bytes=%d checkpoint=%+v fresh=%v checkpoint_median=%s fresh_median=%s",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU(), workers, population, r.Sub(t0), runs[0].artifactBytes, runs, fresh, checkpointMedian, freshMedian)
}

func objectiveCheckpointImage(binding reference.Binding, t0 time.Time) checkpoint.Image {
	image := checkpoint.Image{SchemaVersion: checkpoint.SchemaV1, ProducerMode: checkpoint.ProducerLive, Binding: checkpointBinding(binding), T0: t0, CreatedAt: t0.Add(500 * time.Millisecond), Sequence: 1, Population: len(binding.UniverseSymbols()), Symbols: make([]checkpoint.Symbol, len(binding.UniverseSymbols()))}
	for index, symbol := range binding.UniverseSymbols() {
		image.Symbols[index].Symbol = symbol
	}
	values := checkpoint.Values{Open: 10, High: 11, Low: 9, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: "live_provider_average"}
	aggregate := func(at time.Time) checkpoint.Aggregate {
		return checkpoint.Aggregate{WindowStart: at, WindowEnd: at.Add(time.Second), Values: values}
	}
	primary := &image.Symbols[0]
	primary.HasState = true
	primary.Tail = make([]checkpoint.Aggregate, 961)
	for index := range primary.Tail {
		primary.Tail[index] = aggregate(t0.Add(-time.Duration(961-index) * time.Second))
	}
	older, committed := aggregate(t0.Add(-962*time.Second)), primary.Tail[len(primary.Tail)-1]
	primary.OlderMark, primary.CommittedMark = &older, &committed
	primary.PriceRange = &checkpoint.PriceRange{FirstStart: binding.SessionStart().Add(9 * time.Second).Unix(), RollingFloor: binding.SessionStart().Unix(), FinalizedThrough: t0.Unix(), FirstOpen: 10, SessionHigh: 11, SessionLow: 9, HasFirst: true, HasSessionExtrema: true,
		Highs: []checkpoint.ExtremaPoint{{WindowStart: binding.SessionStart().Add(10 * time.Second).Unix(), Value: 11}}, Lows: []checkpoint.ExtremaPoint{{WindowStart: binding.SessionStart().Add(10 * time.Second).Unix(), Value: 9}},
		SessionHighs: []checkpoint.ExtremaPoint{{WindowStart: binding.SessionStart().Add(10 * time.Second).Unix(), Value: 11}}, SessionLows: []checkpoint.ExtremaPoint{{WindowStart: binding.SessionStart().Add(10 * time.Second).Unix(), Value: 9}}}
	end := binding.SessionStart().Add(90 * time.Second).Unix()
	target := checkpoint.ActivityTargetBlock{End: end, Present: 1}
	target.Highs[0], target.Lows[0] = 10, 9
	primary.Activity = &checkpoint.Activity{References: []checkpoint.ActivitySummary{{End: binding.SessionStart().Add(30 * time.Second).Unix(), Invalid: true}}, Mutable: []checkpoint.ActivityMutable{{End: binding.SessionStart().Add(60 * time.Second).Unix(), Folded: checkpoint.ActivitySummary{End: binding.SessionStart().Add(60 * time.Second).Unix(), Invalid: true}, Current: checkpoint.ActivitySummary{End: binding.SessionStart().Add(60 * time.Second).Unix()}}}, FoldedTargets: []checkpoint.ActivityTargetBlock{target}, FoldedTargetContributions: 1}
	proof := binding.SessionStart().Add(100 * time.Second).Unix()
	primary.Qualification = &checkpoint.Qualification{Proofs: []int64{proof}, Dirty: []int64{proof}, AccountedThrough: t0}

	presence := make([]uint64, 900)
	presence[0] = uint64(1) << 20
	image.Symbols[1].HasState, image.Symbols[1].Presence = true, presence
	absent := make([]uint64, 900)
	absent[0] = uint64(1) << 21
	image.Symbols[2].HasState, image.Symbols[2].ProvenAbsent = true, absent
	conflict := make([]uint64, 900)
	conflict[0] = uint64(1) << 22
	image.Symbols[3].HasState, image.Symbols[3].HistoricalConflict = true, conflict
	image.Symbols[3].Qualification = &checkpoint.Qualification{FinalizedGateBars: []checkpoint.QualificationGateBar{{Start: binding.SessionStart().Add(22 * time.Second).Unix(), Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: "rest_floor_volume_over_transactions"}}, AccountedThrough: t0, Finalized: true, FinalProofEnd: binding.SessionStart().Add(60 * time.Second)}
	image.Symbols[4].Coverage = &checkpoint.Coverage{Outcome: 1}
	invalid := binding.SessionStart().Add(23 * time.Second)
	image.Symbols[5].InvalidMarkStart = &invalid
	image.Counts = objectiveStructureCounts(image)
	return image
}

func checkpointBinding(binding reference.Binding) checkpoint.Binding {
	return checkpoint.Binding{Identity: binding.Identity(), TradingDate: binding.TradingDate(), ScheduleSchema: binding.ScheduleSchema(), ScheduleVersion: binding.ScheduleVersion(), ScheduleArtifactSHA256: binding.ScheduleArtifactSHA256(), PriorSessionDate: binding.PriorSessionDate(), UniversePolicy: binding.UniversePolicy(), UniverseIdentity: binding.UniverseIdentity(), PriorClosePolicy: binding.PriorClosePolicy(), PriorCloseIdentity: binding.PriorCloseIdentity(), Locale: binding.PriorCloseLocale(), Market: binding.PriorCloseMarket(), SessionStart: binding.SessionStart(), SessionEnd: binding.SessionEnd(), PriorRegularClose: binding.PriorRegularClose(), Adjusted: binding.PriorCloseAdjusted(), IncludeOTC: binding.PriorCloseIncludeOTC()}
}

func objectiveStructureCounts(image checkpoint.Image) checkpoint.StructureCounts {
	var result checkpoint.StructureCounts
	for _, symbol := range image.Symbols {
		if symbol.HasState {
			result.RecordsWithState++
		} else {
			result.EmptyStateRecords++
		}
		result.TailRecords += len(symbol.Tail)
		result.PresenceWords += len(symbol.Presence)
		result.ProvenAbsentWords += len(symbol.ProvenAbsent)
		result.ConflictWords += len(symbol.HistoricalConflict)
		if symbol.PriceRange != nil {
			result.PriceExtremaPoints += len(symbol.PriceRange.Highs) + len(symbol.PriceRange.Lows) + len(symbol.PriceRange.SessionHighs) + len(symbol.PriceRange.SessionLows)
		}
		if symbol.Activity != nil {
			result.ActivityReferences += len(symbol.Activity.References)
			result.ActivityMutable += len(symbol.Activity.Mutable)
			result.ActivityTargetBlocks += len(symbol.Activity.FoldedTargets)
			result.ActivityTargetContributions += symbol.Activity.FoldedTargetContributions
		}
		if symbol.Qualification != nil {
			result.QualificationGateBars += len(symbol.Qualification.FinalizedGateBars)
			result.QualificationProofs += len(symbol.Qualification.Proofs)
			result.QualificationDirty += len(symbol.Qualification.Dirty)
		}
		if symbol.InvalidMarkStart != nil {
			result.InvalidMarks++
		}
		if symbol.Coverage != nil {
			result.CoverageConsequences++
		}
	}
	return result
}

func finishEmptyCheckpointHydration(t *testing.T, state *engine.Engine, binding reference.Binding, attempt *LiveAttempt) {
	finishEmptyCheckpointHydrationContext(t, context.Background(), state, binding, attempt)
}

func finishEmptyCheckpointHydrationContext(t *testing.T, ctx context.Context, state *engine.Engine, binding reference.Binding, attempt *LiveAttempt) {
	t.Helper()
	budgets := engine.HydrationPlanBudgets{Workers: 1, RowsPerChunk: 1, MaximumResponseBytes: 1, MaximumNormalizedRecords: 1, MaximumResidentRecords: 1}
	admission, completion := state.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationCheckpointCatchup, ConnectionEpoch: attempt.Epoch(), Budgets: budgets})
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	result := <-completion
	command, ok := result.Plan.FenceCommand()
	if result.Code != engine.DispositionHydrationPlanApplied || !result.Plan.Empty() || !ok {
		t.Fatalf("empty checkpoint plan=%+v", result)
	}
	finishCheckpointFenceContext(t, ctx, state, attempt, command)
}

func objectiveHydrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		parts := strings.Split(request.URL.Path, "/")
		if len(parts) != 10 {
			http.NotFound(writer, request)
			return
		}
		symbol := parts[4]
		index, _ := strconv.Atoi(strings.TrimPrefix(symbol, "S"))
		from, _ := strconv.ParseInt(parts[8], 10, 64)
		through, _ := strconv.ParseInt(parts[9], 10, 64)
		rows := 0
		if index%2 == 0 {
			rows = 1
			if through-from > 30_000 {
				rows = 60
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[`, symbol)
		for row := 0; row < rows; row++ {
			if row > 0 {
				fmt.Fprint(writer, ",")
			}
			at := from + int64(row)*1000
			fmt.Fprintf(writer, `{"t":%d,"o":10,"h":11,"l":9,"c":10,"v":100,"vw":10,"n":10}`, at)
		}
		fmt.Fprint(writer, `]}`)
	}))
}

func durationMedian(values []time.Duration) time.Duration {
	copyValues := append([]time.Duration(nil), values...)
	for i := 1; i < len(copyValues); i++ {
		for j := i; j > 0 && copyValues[j] < copyValues[j-1]; j-- {
			copyValues[j], copyValues[j-1] = copyValues[j-1], copyValues[j]
		}
	}
	return copyValues[len(copyValues)/2]
}

type checkpointHydrationRun struct {
	command    engine.HydrationFenceCommand
	start      time.Time
	end        time.Time
	planned    int
	normalized int64
}

func newBoundLiveEngine(t *testing.T, binding reference.Binding, now *time.Time) *engine.Engine {
	return newBoundLiveEngineContext(t, context.Background(), binding, now)
}

func newBoundLiveEngineContext(t *testing.T, ctx context.Context, binding reference.Binding, now *time.Time) *engine.Engine {
	t.Helper()
	delay := time.Duration(0)
	state, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: func() time.Time { return *now }, Capacity: 256, RequiredReserve: 16, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := state.AdmitBinding(ctx, engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionBindingInstalled {
		t.Fatal("binding install")
	}
	return state
}

func startAggregateEpochAt(t *testing.T, state *engine.Engine, binding reference.Binding, acknowledgement time.Time) (*LiveAttempt, *fakeLiveSocket) {
	return startAggregateEpochAtContext(t, context.Background(), state, binding, acknowledgement)
}

func startAggregateEpochAtContext(t *testing.T, ctx context.Context, state *engine.Engine, binding reference.Binding, acknowledgement time.Time) (*LiveAttempt, *fakeLiveSocket) {
	t.Helper()
	socket := newFakeLiveSocket()
	enqueueHandshake(socket)
	adapter, open := testLiveAdapterForBinding(t, socket, binding)
	attempt, started, err := adapter.Start(ctx, open)
	if err != nil {
		t.Fatal(err)
	}
	attempt.queue.now = func() time.Time { return acknowledgement }
	if result, err := DeliverToEngine(ctx, state, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		t.Fatalf("attempt=%+v err=%v", result, err)
	}
	deliveries, err := attempt.Handshake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range deliveries {
		if delivery.Control.Kind == engine.AggregateSubscriptionResult {
			delivery.Control.ReceiptTime = acknowledgement
		}
		if result, err := DeliverToEngine(ctx, state, delivery); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("handshake=%+v err=%v delivery=%+v", result, err, delivery)
		}
	}
	return attempt, socket
}

func runCheckpointHydration(t *testing.T, state *engine.Engine, binding reference.Binding, epoch uint64, purpose engine.HydrationPurpose, server *httptest.Server) checkpointHydrationRun {
	// P-C7-LIVE proves semantic equivalence, not worker concurrency. One worker
	// keeps the two-request loopback fixture deterministic under package load;
	// the separately selected objective proof retains its explicit worker shape.
	return runCheckpointHydrationWorkers(t, state, binding, epoch, purpose, server, 1)
}

func runCheckpointHydrationWorkers(t *testing.T, state *engine.Engine, binding reference.Binding, epoch uint64, purpose engine.HydrationPurpose, server *httptest.Server, workers int) checkpointHydrationRun {
	return runCheckpointHydrationWorkersContext(t, context.Background(), state, binding, epoch, purpose, server, workers)
}

func runCheckpointHydrationWorkersContext(t *testing.T, ctx context.Context, state *engine.Engine, binding reference.Binding, epoch uint64, purpose engine.HydrationPurpose, server *httptest.Server, workers int) checkpointHydrationRun {
	t.Helper()
	seconds := int64(binding.SessionEnd().Sub(binding.SessionStart()) / time.Second)
	budgets := engine.HydrationPlanBudgets{Workers: workers, RowsPerChunk: 64, MaximumResponseBytes: 64 << 20, MaximumNormalizedRecords: int64(len(binding.UniverseSymbols())) * seconds, MaximumResidentRecords: int64(workers) * seconds}
	admission, completion := state.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	planDisposition := <-completion
	if planDisposition.Code != engine.DispositionHydrationPlanApplied {
		t.Fatalf("plan=%+v", planDisposition)
	}
	requests := planDisposition.Plan.Requests()
	work := make([]HydrationWorkItem, len(requests))
	tokens := make(map[uint64]engine.HydrationRequestToken, len(requests))
	for index, token := range requests {
		var err error
		work[index], err = HydrationWorkItemFromEngine(binding, token)
		if err != nil {
			t.Fatal(err)
		}
		tokens[token.RequestID()] = token
	}
	workerPlan, err := NewHydrationWorkerPlan(work, workers, 64, budgets.MaximumResponseBytes, budgets.MaximumNormalizedRecords, budgets.MaximumResidentRecords)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewHydrationWorker(server.URL, func() (string, error) { return "offline-c7", nil }, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	sink := &integrationHydrationEngineSink{state: state, tokens: tokens}
	result := worker.Run(ctx, ctx, workerPlan, sink)
	accounting := result.Accounting()
	terminals := accounting.ProviderCompletedValue + accounting.ProviderCompletedEmpty + accounting.ProviderFailed + accounting.ProviderCanceled
	if sink.err != nil || accounting.ItemsStarted != terminals || accounting.ProviderCompletedValue+accounting.ProviderCompletedEmpty != int64(len(requests)) || accounting.UnadmittedTerminals != 0 || accounting.AdmissionIntegrityFailures != 0 || sink.command.CommandToken() == 0 {
		t.Fatalf("worker=%+v sink=%v command=%+v", result.Accounting(), sink.err, sink.command)
	}
	return checkpointHydrationRun{command: sink.command, start: planDisposition.Plan.Start(), end: planDisposition.Plan.End(), planned: len(requests), normalized: accounting.NormalizedRows}
}

func finishCheckpointFence(t *testing.T, state *engine.Engine, attempt *LiveAttempt, command any) {
	finishCheckpointFenceContext(t, context.Background(), state, attempt, command)
}

func finishCheckpointFenceContext(t *testing.T, ctx context.Context, state *engine.Engine, attempt *LiveAttempt, command any) {
	t.Helper()
	var value engine.HydrationFenceCommand
	switch typed := command.(type) {
	case checkpointHydrationRun:
		value = typed.command
	case engine.HydrationFenceCommand:
		value = typed
	default:
		t.Fatalf("invalid fence command %T", command)
	}
	capture, err := CaptureAggregateIngressFenceCommandFromEngine(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.CaptureAggregateIngressFence(ctx, state, capture); err != nil {
		t.Fatal(err)
	}
	delivery, ok, err := attempt.DeliverNextToEngine(ctx, state)
	if err != nil || !ok || delivery.HydrationDisposition.Code != engine.DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence=%+v ok=%v err=%v", delivery, ok, err)
	}
}

func deliverCheckpointLiveCorrection(t *testing.T, state *engine.Engine, attempt *LiveAttempt, socket *fakeLiveSocket, symbol string, at time.Time, close float64) {
	t.Helper()
	raw := fmt.Sprintf(`{"ev":"A","sym":%q,"s":%d,"e":%d,"o":10,"h":%g,"l":9,"c":%g,"v":100,"z":10,"vw":%g}`, symbol, at.UnixMilli(), at.Add(time.Second).UnixMilli(), close+1, close, close)
	socket.send(socketMessageText, "["+raw+"]")
	waitForQueuedFrames(t, attempt, 1)
	delivery, ok, err := attempt.DeliverNextToEngine(context.Background(), state)
	if err != nil || !ok || (delivery.AggregateDisposition.Code != engine.DispositionAggregateRevised && delivery.AggregateDisposition.Code != engine.DispositionAggregateInserted) {
		t.Fatalf("live correction=%+v ok=%v err=%v", delivery, ok, err)
	}
}

func projectCheckpointLive(t *testing.T, state *engine.Engine) engine.CheckpointProjectionResult {
	return projectCheckpointLiveContext(t, context.Background(), state)
}

func projectCheckpointLiveContext(t *testing.T, ctx context.Context, state *engine.Engine) engine.CheckpointProjectionResult {
	t.Helper()
	admission, completion := state.AdmitCheckpointProjection(ctx)
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	got := <-completion
	if got.Disposition != engine.CheckpointProjected {
		t.Fatalf("projection=%+v", got)
	}
	return got
}

func installCheckpointLive(t *testing.T, state *engine.Engine, candidate checkpoint.Candidate) engine.CheckpointInstallResult {
	return installCheckpointLiveContext(t, context.Background(), state, candidate)
}

func installCheckpointLiveContext(t *testing.T, ctx context.Context, state *engine.Engine, candidate checkpoint.Candidate) engine.CheckpointInstallResult {
	t.Helper()
	admission, completion := state.AdmitCheckpointInstall(ctx, candidate)
	if admission != engine.AdmissionAdmitted {
		t.Fatal(admission)
	}
	return <-completion
}

func checkpointHydrationServer(t *testing.T, records map[string]map[int64]float64) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		parts := strings.Split(request.URL.Path, "/")
		if len(parts) != 10 {
			http.NotFound(writer, request)
			return
		}
		symbol := parts[4]
		from, _ := strconv.ParseInt(parts[8], 10, 64)
		through, _ := strconv.ParseInt(parts[9], 10, 64)
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(writer, `{"status":"OK","ticker":%q,"adjusted":false,"results":[`, symbol)
		first := true
		for at, close := range records[symbol] {
			if at < from || at > through {
				continue
			}
			if !first {
				fmt.Fprint(writer, ",")
			}
			first = false
			fmt.Fprintf(writer, `{"t":%d,"o":%g,"h":%g,"l":9,"c":%g,"v":100,"vw":%g,"n":10}`, at, close, close+1, close, close)
		}
		fmt.Fprint(writer, `]}`)
	}))
}

func closeCheckpointAttempt(t *testing.T, attempt *LiveAttempt, _ *fakeLiveSocket, binding reference.Binding, token uint64) {
	closeCheckpointAttemptContext(t, context.Background(), attempt, nil, binding, token)
}

func closeCheckpointAttemptContext(t *testing.T, ctx context.Context, attempt *LiveAttempt, _ *fakeLiveSocket, binding reference.Binding, token uint64) {
	t.Helper()
	if err := attempt.Close(CloseEpochCommand{BindingIdentity: binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: token, Cause: CloseControlledStop}); err != nil {
		t.Fatal(err)
	}
	_, _ = attempt.nextForProof(ctx)
}

func closeCheckpointEngine(t *testing.T, state *engine.Engine) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	closeCheckpointEngineContext(t, ctx, state)
}

func closeCheckpointEngineContext(t *testing.T, ctx context.Context, state *engine.Engine) {
	t.Helper()
	state.Close()
	if err := state.Wait(ctx); err != nil {
		t.Fatal(err)
	}
}
