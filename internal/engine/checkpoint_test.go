package engine

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestC7STATE01CommittedProjectionRoundTrip is P-C7-STATE. It distinguishes
// the committed T0 graph from admitted forward state, exercises folded/dirty
// semantic fields through the actual C3 owners, proves the image is detached,
// and compares the restored graph under the ordinary evaluator before and
// after an identical continuation. It does not prove bytes, files, provider
// behavior, or capacity.
func TestC7STATE01CommittedProjectionRoundTrip(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	t0 := start.Add(2*time.Minute + 5*time.Second)
	now := t0.Add(20 * time.Second)
	source := aggregateEngine(t, binding, RunModeLive, &now)

	for i := 0; i < 125; i++ {
		input := liveAggregate(binding, "AAA", start.Add(time.Duration(i)*time.Second), 1, uint64(i+1))
		input.Values.Close, input.Values.High, input.Values.VWAP = 10+float64(i)/100, 11+float64(i)/100, 10+float64(i)/100
		applyAggregate(t, source, input, DispositionAggregateInserted, ReasonNone)
	}
	// This plausible newer mark must not cross the checkpoint cutoff.
	forward := liveAggregate(binding, "AAA", t0.Add(5*time.Second), 1, 121)
	forward.Values.Close, forward.Values.High, forward.Values.VWAP = 99, 99, 99
	applyAggregate(t, source, forward, DispositionAggregateInserted, ReasonNone)
	proveAggregateCoverage(t, source, "AAA", start, t0)

	source.mu.Lock()
	source.state.lifecycle = lifecycleLive
	source.applyAggregateCandidateLocked(t0, t0)
	source.state.aggregateEvaluator.current = source.stageAggregateEvaluationAtLocked(t0, t0)
	want := cloneAggregateEvaluation(source.state.aggregateEvaluator.current)
	source.mu.Unlock()

	projection := projectForTest(t, source)
	if _, err := json.Marshal(projection.Image); err != nil {
		t.Fatalf("semantic image is not JSON-encodable: %v", err)
	}
	if projection.Image.T0 != t0 || projection.Image.CreatedAt != now || projection.Image.Sequence != 1 || len(projection.Image.Symbols) != len(binding.UniverseSymbols()) {
		t.Fatalf("projection header/population = %+v symbols=%d", projection.Image, len(projection.Image.Symbols))
	}
	aaa := projection.Image.Symbols[0]
	for _, record := range aaa.Tail {
		if !record.WindowStart.Before(t0) || record.Values.Close == 99 {
			t.Fatalf("post-T0 record leaked: %+v", record)
		}
	}
	if aaa.CommittedMark == nil || aaa.CommittedMark.WindowStart != t0.Add(-time.Second) || aaa.CommittedMark.Values.Open == 0 || aaa.CommittedMark.Values.AverageTradeSize == 0 {
		t.Fatalf("committed real mark missing: %+v", aaa.CommittedMark)
	}

	// Caller mutation of a projected view cannot alias the source owner.
	projection.Image.Symbols[0].Tail[0].Values.Close = 777
	projection.Image.Symbols[0].Symbol = "MUTATED"
	source.mu.Lock()
	gotSource := source.state.binding.symbols[0].aggregates.tail[start.Unix()].values.Close
	sourceSymbol := source.state.binding.symbols[0].symbol
	source.mu.Unlock()
	if gotSource == 777 || sourceSymbol == "MUTATED" {
		t.Fatal("projection retained a writable engine alias")
	}

	// Restore from a pristine second projection and compare the ordinary C3
	// evaluation in the same semantic lifecycle.
	projection = projectForTest(t, source)
	restored := freshBoundEngine(t, binding, &now)
	install := installForTest(t, restored, projection.Image)
	if install.Disposition != CheckpointInstalled || install.Fact.T0 != t0 {
		t.Fatalf("install = %+v", install)
	}
	restoredLookup := restored.state.binding.symbols[0].aggregates.activity
	if restoredLookup == nil || !restoredLookup.referenceLookup.valid ||
		len(restoredLookup.referenceLookup.transactions) != len(restoredLookup.referenceLookup.expansions) {
		t.Fatalf("restored Activity lookup not rebuilt/validated: %+v", restoredLookup)
	}
	if gotSymbols := checkpointSymbolsForTest(t, restored, t0); !reflect.DeepEqual(gotSymbols, projection.Image.Symbols) {
		g, w := gotSymbols[0], projection.Image.Symbols[0]
		t.Fatalf("restored complete semantic graph differs tail=%v marks=%v/%v bitmaps=%v/%v/%v price=%v activity=%v qualification=%v support=%v/%v", reflect.DeepEqual(g.Tail, w.Tail), reflect.DeepEqual(g.OlderMark, w.OlderMark), reflect.DeepEqual(g.CommittedMark, w.CommittedMark), reflect.DeepEqual(g.Presence, w.Presence), reflect.DeepEqual(g.ProvenAbsent, w.ProvenAbsent), reflect.DeepEqual(g.HistoricalConflict, w.HistoricalConflict), reflect.DeepEqual(g.PriceRange, w.PriceRange), reflect.DeepEqual(g.Activity, w.Activity), reflect.DeepEqual(g.Qualification, w.Qualification), reflect.DeepEqual(g.InvalidMarkStart, w.InvalidMarkStart), reflect.DeepEqual(g.Coverage, w.Coverage))
	}
	restored.mu.Lock()
	restored.state.lifecycle = lifecycleLive
	got := restored.stageAggregateEvaluationAtLocked(t0, t0)
	restored.state.lifecycle = lifecycleAwaitingAggregateAck
	restored.mu.Unlock()
	if !aggregateEvaluationEqual(got, want) {
		t.Fatalf("restored T0 evaluation differs\ngot=%+v\nwant=%+v", got, want)
	}

	continuation := liveAggregate(binding, "AAA", t0.Add(-time.Second), 2, 1)
	continuation.Values.Close, continuation.Values.High, continuation.Values.VWAP = 12.5, 12.5, 12.5
	source.mu.Lock()
	source.state.lifecycle = lifecycleAwaitingAggregateAck
	source.mu.Unlock()
	applyAggregate(t, source, continuation, DispositionAggregateRevised, ReasonNone)
	applyAggregate(t, restored, continuation, DispositionAggregateRevised, ReasonNone)
	restoredForward := forward
	restoredForward.Live = LivePosition{ConnectionEpoch: 2, FrameSequence: 2}
	applyAggregate(t, restored, restoredForward, DispositionAggregateInserted, ReasonNone)
	source.mu.Lock()
	source.state.lifecycle = lifecycleLive
	afterSource := source.stageAggregateEvaluationAtLocked(t0, t0)
	source.mu.Unlock()
	restored.mu.Lock()
	restored.state.lifecycle = lifecycleLive
	afterRestored := restored.stageAggregateEvaluationAtLocked(t0, t0)
	restored.mu.Unlock()
	if !aggregateEvaluationEqual(afterRestored, afterSource) {
		t.Fatal("identical continuation diverged after restore")
	}
	later := t0.Add(25 * time.Second)
	proveAggregateCoverage(t, source, "AAA", start, later)
	proveAggregateCoverage(t, restored, "AAA", start, later)
	source.mu.Lock()
	sourceActivity := evaluateActivityFeatures(source.state.binding, source.state.binding.symbols[0].aggregates, later)
	source.mu.Unlock()
	restored.mu.Lock()
	restoredActivity := evaluateActivityFeatures(restored.state.binding, restored.state.binding.symbols[0].aggregates, later)
	restored.mu.Unlock()
	if sourceActivity != restoredActivity {
		t.Fatalf("nonaligned post-restore Activity diverged got=%+v want=%+v", restoredActivity, sourceActivity)
	}
	t.Run("strict compaction presence older mark folded activity conflict qualification and support", func(t *testing.T) {
		compactNow := start.Add(2 * time.Minute)
		compact := aggregateEngine(t, binding, RunModeLive, &compactNow)
		for i := 0; i < 90; i++ {
			input := liveAggregate(binding, "AAA", start.Add(time.Duration(i)*time.Second), 1, uint64(i+1))
			input.Values.Volume = 10_000
			input.Values.AverageTradeSize = 10
			applyAggregate(t, compact, input, DispositionAggregateInserted, ReasonNone)
		}
		ct := start.Add(90 * time.Second)
		proveAggregateCoverage(t, compact, "AAA", start, ct)
		compact.mu.Lock()
		compact.state.lifecycle = lifecycleLive
		compact.applyAggregateCandidateLocked(ct, ct)
		state := compact.state.binding.symbols[0].aggregates
		compactNow = ct.Add(correctionHorizon + time.Second)
		compact.compactSymbolLocked(state, compact.state.binding, "AAA", compactNow)
		maintainActivityState(state, compact.state.binding, compactNow, ct)
		ensureHistoricalConflict(state).set(3)
		compact.state.aggregateEvaluator.invalidMarks = map[int]invalidMarkEvidence{1: {windowStart: start.Add(time.Second)}}
		compact.state.aggregateEvaluator.coverage = map[int]aggregateCoverageConsequence{1: coverageUnknownFailureOrFence}
		compact.applyAggregateCandidateLocked(ct, ct)
		compact.state.aggregateEvaluator.current = compact.stageAggregateEvaluationAtLocked(ct, ct)
		compact.mu.Unlock()
		image := projectForTest(t, compact).Image
		s := image.Symbols[0]
		if len(s.Tail) != 0 || s.OlderMark == nil || s.CommittedMark == nil || len(s.Presence) == 0 || s.Presence[0] == 0 || len(s.HistoricalConflict) == 0 || s.HistoricalConflict[0]&(1<<3) == 0 || s.Activity == nil || s.Activity.FoldedTargetContributions == 0 || s.Qualification == nil || len(s.Qualification.FinalizedGateBars) == 0 || image.Symbols[1].InvalidMarkStart == nil || image.Symbols[1].Coverage == nil {
			t.Fatalf("compacted semantic coverage incomplete: %+v support=%+v", s, image.Symbols[1])
		}
		dst := freshBoundEngine(t, binding, &compactNow)
		if got := installForTest(t, dst, image); got.Disposition != CheckpointInstalled {
			t.Fatalf("compacted install=%+v", got)
		}
		if gotSymbols := checkpointSymbolsForTest(t, dst, ct); !reflect.DeepEqual(gotSymbols, image.Symbols) {
			t.Fatal("compacted semantic round trip differs")
		}
		compact.mu.Lock()
		q := compact.state.binding.symbols[0].aggregates.qualification
		if len(q.proofs) == 0 {
			compact.mu.Unlock()
			t.Fatal("strong compacted trace did not produce provisional qualification proof")
		}
		for proof := range q.proofs {
			q.dirty[proof] = struct{}{}
			break
		}
		other := ensureAggregateState(&compact.state.binding.symbols[1])
		other.qualification = &qualificationState{finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{}, accountedThrough: ct, finalized: true, finalProofEnd: start.Add(60 * time.Second), unresolvedOrigin: uncertaintyNone}
		other.activity = &activityFeatureState{references: map[int64]*activityBlockSummary{start.Add(30 * time.Second).Unix(): {end: start.Add(30 * time.Second).Unix(), low: math.Inf(1), invalid: true}}, mutable: map[int64]activityMutableBlock{}, foldedTargets: map[int64]activityFoldedTargetBlock{}, result: unavailableActivityResult(time.Time{})}
		compact.state.aggregateEvaluator.current = compact.stageAggregateEvaluationAtLocked(ct, ct)
		compact.mu.Unlock()
		qualificationImage := projectForTest(t, compact).Image
		if len(qualificationImage.Symbols[0].Qualification.Proofs) == 0 || len(qualificationImage.Symbols[0].Qualification.Dirty) == 0 || !qualificationImage.Symbols[1].Qualification.Finalized || qualificationImage.Symbols[1].Activity.References[0] != (checkpoint.ActivitySummary{End: start.Add(30 * time.Second).Unix(), Invalid: true}) {
			t.Fatalf("qualification/invalid Activity projection incomplete: a=%+v b=%+v", qualificationImage.Symbols[0].Qualification, qualificationImage.Symbols[1])
		}
		if _, err := json.Marshal(qualificationImage); err != nil {
			t.Fatalf("invalid/empty Activity image is not finite JSON: %v", err)
		}
		qualificationDst := freshBoundEngine(t, binding, &compactNow)
		if got := installForTest(t, qualificationDst, qualificationImage); got.Disposition != CheckpointInstalled {
			t.Fatalf("qualification install=%+v", got)
		}
		qualificationDst.mu.Lock()
		installedA := qualificationDst.state.binding.symbols[0].aggregates.qualification
		installedB := qualificationDst.state.binding.symbols[1].aggregates.qualification
		qualificationDst.mu.Unlock()
		if len(installedA.proofs) == 0 || len(installedA.dirty) != 0 || !installedB.finalized {
			t.Fatalf("qualification regeneration a=%+v b=%+v", installedA, installedB)
		}
		closeAndWait(t, compact)
		closeAndWait(t, dst)
		closeAndWait(t, qualificationDst)
	})
	t.Run("zero ATS folded target has finite canonical round trip", func(t *testing.T) {
		ct := start.Add(90 * time.Second)
		atsNow := ct
		atsSource := aggregateEngine(t, binding, RunModeLive, &atsNow)
		for i := 0; i < 90; i++ {
			input := liveAggregate(binding, "AAA", start.Add(time.Duration(i)*time.Second), 1, uint64(i+1))
			input.Values.Volume = 1_000
			input.Values.AverageTradeSize = 10
			if i == 65 {
				input.Values.AverageTradeSize = 0
			}
			applyAggregate(t, atsSource, input, DispositionAggregateInserted, ReasonNone)
		}
		proveAggregateCoverage(t, atsSource, "AAA", start, ct)
		atsSource.mu.Lock()
		atsSource.state.lifecycle = lifecycleLive
		atsSource.applyAggregateCandidateLocked(ct, ct)
		atsNow = ct.Add(correctionHorizon + time.Second)
		atsState := atsSource.state.binding.symbols[0].aggregates
		atsSource.compactSymbolLocked(atsState, atsSource.state.binding, "AAA", atsNow)
		maintainActivityState(atsState, atsSource.state.binding, atsNow, ct)
		atsSource.state.aggregateEvaluator.current = atsSource.stageAggregateEvaluationAtLocked(ct, ct)
		wantEval := cloneAggregateEvaluation(atsSource.state.aggregateEvaluator.current)
		atsSource.mu.Unlock()

		atsImage := projectForTest(t, atsSource).Image
		activity := atsImage.Symbols[0].Activity
		if activity == nil || len(activity.FoldedTargets) == 0 {
			t.Fatal("real zero-ATS aggregate did not reach a folded target")
		}
		slot := 5
		target := activity.FoldedTargets[0]
		mask := uint32(1) << uint(slot)
		if target.Present&mask == 0 || target.Invalid&mask == 0 || target.Transactions[slot] != 0 || target.Highs[slot] != 0 || target.Lows[slot] != 0 {
			t.Fatalf("zero-ATS folded slot is not canonical: %+v", target)
		}
		if _, err := json.Marshal(atsImage); err != nil {
			t.Fatalf("zero-ATS image is not standard-JSON finite: %v", err)
		}
		atsRestored := freshBoundEngine(t, binding, &atsNow)
		if got := installForTest(t, atsRestored, atsImage); got.Disposition != CheckpointInstalled {
			t.Fatalf("zero-ATS install=%+v", got)
		}
		if gotSymbols := checkpointSymbolsForTest(t, atsRestored, ct); !reflect.DeepEqual(gotSymbols, atsImage.Symbols) {
			t.Fatalf("zero-ATS semantic graph differs after install\ngot activity=%+v qualification=%+v\nwant activity=%+v qualification=%+v", gotSymbols[0].Activity, gotSymbols[0].Qualification, atsImage.Symbols[0].Activity, atsImage.Symbols[0].Qualification)
		}
		atsRestored.mu.Lock()
		atsRestored.state.lifecycle = lifecycleLive
		gotEval := atsRestored.stageAggregateEvaluationAtLocked(ct, ct)
		atsRestored.mu.Unlock()
		if !aggregateEvaluationEqual(gotEval, wantEval) {
			t.Fatalf("zero-ATS evaluator differs got=%+v want=%+v", gotEval, wantEval)
		}

		continuation := liveAggregate(binding, "AAA", ct, 9, 1)
		atsSource.mu.Lock()
		atsSource.state.lifecycle = lifecycleAwaitingAggregateAck
		atsSource.mu.Unlock()
		atsRestored.mu.Lock()
		atsRestored.state.lifecycle = lifecycleAwaitingAggregateAck
		atsRestored.mu.Unlock()
		applyAggregate(t, atsSource, continuation, DispositionAggregateInserted, ReasonNone)
		applyAggregate(t, atsRestored, continuation, DispositionAggregateInserted, ReasonNone)
		evalAt := ct.Add(time.Second)
		atsSource.mu.Lock()
		atsSource.state.lifecycle = lifecycleLive
		afterSource := atsSource.stageAggregateEvaluationAtLocked(evalAt, evalAt)
		atsSource.mu.Unlock()
		atsRestored.mu.Lock()
		atsRestored.state.lifecycle = lifecycleLive
		afterRestored := atsRestored.stageAggregateEvaluationAtLocked(evalAt, evalAt)
		atsRestored.mu.Unlock()
		if !aggregateEvaluationEqual(afterRestored, afterSource) {
			t.Fatal("zero-ATS identical continuation diverged")
		}
		closeAndWait(t, atsSource)
		closeAndWait(t, atsRestored)
	})
	// Projection completion is success-bearing only after the common accounting
	// path has completed. A late integrity failure must clear the detached image.
	source.mu.Lock()
	source.transitions.completedExternal++
	source.mu.Unlock()
	admission, completion := source.AdmitCheckpointProjection(context.Background())
	if admission != AdmissionAdmitted {
		t.Fatalf("fault projection admission=%s", admission)
	}
	faultProjection := <-completion
	if faultProjection.Disposition == CheckpointProjected || faultProjection.Code != DispositionAccountingIntegrity || !reflect.DeepEqual(faultProjection.Image, checkpoint.Image{}) || faultProjection.SuppressionDisposition == "" {
		t.Fatalf("accounting fault returned stale projection success: %+v", faultProjection)
	}
	closeAndWait(t, source)
	closeAndWait(t, restored)
}

// TestC7INSTALL01SemanticMutationAtomicity is P-C7-INSTALL. Every mutation is
// integrity-valid at the outer candidate boundary but semantically invalid;
// none may expose a partial symbol/T0/evaluation. The valid candidate installs
// once, a duplicate is stable, and restored authority accepts a new-run live
// correction without comparing dead-process positions.
func TestC7INSTALL01SemanticMutationAtomicity(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	t0 := start.Add(10 * time.Second)
	now := t0.Add(time.Second)
	source := aggregateEngine(t, binding, RunModeLive, &now)
	for i := 0; i < 10; i++ {
		applyAggregate(t, source, liveAggregate(binding, "AAA", start.Add(time.Duration(i)*time.Second), 1, uint64(i+1)), DispositionAggregateInserted, ReasonNone)
	}
	proveAggregateCoverage(t, source, "AAA", start, t0)
	source.mu.Lock()
	source.state.lifecycle = lifecycleLive
	source.applyAggregateCandidateLocked(t0, t0)
	source.state.aggregateEvaluator.current = source.stageAggregateEvaluationAtLocked(t0, t0)
	source.mu.Unlock()
	base := projectForTest(t, source).Image

	mutations := []struct {
		name   string
		mutate func(*checkpoint.Image)
	}{
		{"mixed binding", func(i *checkpoint.Image) { i.Binding.TradingDate = "2099-01-01" }},
		{"unsupported schema", func(i *checkpoint.Image) { i.SchemaVersion = "v2" }},
		{"unsupported mode", func(i *checkpoint.Image) { i.ProducerMode = "replay" }},
		{"zero sequence", func(i *checkpoint.Image) { i.Sequence = 0 }},
		{"nonwhole T0", func(i *checkpoint.Image) { i.T0 = i.T0.Add(time.Nanosecond) }},
		{"creation before T0", func(i *checkpoint.Image) { i.CreatedAt = i.T0.Add(-time.Second) }},
		{"post T0 aggregate", func(i *checkpoint.Image) {
			i.Symbols[0].Tail[0].WindowStart = t0
			i.Symbols[0].Tail[0].WindowEnd = t0.Add(time.Second)
		}},
		{"synthetic nonfinite mark", func(i *checkpoint.Image) { i.Symbols[0].CommittedMark.Values.Open = 0 }},
		{"nonfinite canonical", func(i *checkpoint.Image) { i.Symbols[0].Tail[0].Values.Close = math.NaN() }},
		{"invalid ATS provenance", func(i *checkpoint.Image) { i.Symbols[0].Tail[0].Values.ATSProvenance = "fabricated" }},
		{"missing symbol record", func(i *checkpoint.Image) { i.Symbols = i.Symbols[:len(i.Symbols)-1] }},
		{"structure count mismatch", func(i *checkpoint.Image) { i.Counts.TailRecords++ }},
		{"contradictory absence", func(i *checkpoint.Image) {
			i.Symbols[0].ProvenAbsent = make([]uint64, sessionSeconds/64)
			i.Symbols[0].ProvenAbsent[0] |= 1
		}},
		{"cached support contradiction", func(i *checkpoint.Image) { v := t0; i.Symbols[0].InvalidMarkStart = &v }},
		{"activity exact sum transactions", func(i *checkpoint.Image) { i.Symbols[0].Activity.Mutable[0].Current.Transactions++ }},
		{"activity expansion formula", func(i *checkpoint.Image) { i.Symbols[0].Activity.Mutable[0].Current.ExpansionBPS++ }},
		{"activity canonical unused fields", func(i *checkpoint.Image) {
			for n := range i.Symbols[0].Activity.Mutable {
				if i.Symbols[0].Activity.Mutable[n].Folded.AggregateCount == 0 {
					i.Symbols[0].Activity.Mutable[n].Folded.Low = 1
					return
				}
			}
			panic("no unused activity summary")
		}},
		{"qualification dirty without proof", func(i *checkpoint.Image) { i.Symbols[0].Qualification.Dirty = []int64{t0.Unix()} }},
		{"price cutoff contradiction", func(i *checkpoint.Image) { i.Symbols[0].PriceRange.RollingFloor = t0.Add(time.Second).Unix() }},
		{"coverage enum", func(i *checkpoint.Image) { i.Symbols[1].Coverage = &checkpoint.Coverage{Outcome: 99} }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			image := base.Clone()
			tc.mutate(&image)
			target := freshBoundEngine(t, binding, &now)
			result := installForTest(t, target, image)
			want, wantReason := CheckpointInvalid, CheckpointReasonNone
			switch tc.name {
			case "mixed binding":
				want, wantReason = CheckpointIncompatible, CheckpointReasonBindingMismatch
			case "unsupported schema", "unsupported mode":
				want, wantReason = CheckpointIncompatible, CheckpointReasonUnsupportedSchema
			}
			if result.Disposition != want || wantReason != CheckpointReasonNone && result.Reason != wantReason {
				t.Fatalf("mutation disposition=%+v want=%s reason=%s", result, want, wantReason)
			}
			target.mu.Lock()
			partial := target.state.committedT != nil || target.state.binding.symbols[0].aggregates != nil || target.state.installedCheckpoint != nil
			target.mu.Unlock()
			if partial {
				t.Fatal("rejected candidate partially mutated engine")
			}
			closeAndWait(t, target)
		})
	}

	t.Run("candidate mutation after admission is detached", func(t *testing.T) {
		detached := freshBoundEngine(t, binding, &now)
		entered, release := make(chan struct{}), make(chan struct{})
		detached.beforeConsume = func(node *queueNode) {
			if node.kind == inputCheckpointInstall {
				close(entered)
				<-release
			}
		}
		candidate := checkpoint.Candidate{Image: base.Clone(), Checksum: strings.Repeat("b", 64), Integrity: true}
		admission, completion := detached.AdmitCheckpointInstall(context.Background(), candidate)
		if admission != AdmissionAdmitted {
			t.Fatalf("admission=%s", admission)
		}
		<-entered
		candidate.Image.Symbols[0].Tail[0].Values.Open = 0
		candidate.Image.Symbols[0].Symbol = "ALIASED"
		close(release)
		result := <-completion
		if result.Disposition != CheckpointInstalled {
			t.Fatalf("detached candidate install=%+v", result)
		}
		closeAndWait(t, detached)
	})
	for _, fault := range []struct {
		name    string
		prepare func(*Engine)
		want    DispositionCode
	}{{"publication completion", func(e *Engine) { e.publicationFault = publicationFaultBuild }, DispositionPublicationIntegrity}, {"accounting completion", func(e *Engine) { e.transitions.completedExternal++ }, DispositionAccountingIntegrity}} {
		t.Run(fault.name+" cannot return stale install success", func(t *testing.T) {
			e := freshBoundEngine(t, binding, &now)
			e.mu.Lock()
			fault.prepare(e)
			e.mu.Unlock()
			got := installForTest(t, e, base)
			if got.Disposition == CheckpointInstalled || got.Code != fault.want || got.Fact != (InstalledCheckpointFact{}) || got.SuppressionDisposition == "" {
				t.Fatalf("fault completion=%+v", got)
			}
			closeAndWait(t, e)
		})
	}

	target := freshBoundEngine(t, binding, &now)
	first := installForTest(t, target, base)
	before := target.publication.Load()
	second := installForTest(t, target, base)
	after := target.publication.Load()
	if first.Disposition != CheckpointInstalled || first.Code != DispositionCheckpointInstalled || second.Disposition != CheckpointInstalled || second.Fact != first.Fact || second.EngineSequence != first.EngineSequence+1 || before != after {
		t.Fatalf("duplicate install unstable first=%+v second=%+v publication_changed=%v", first, second, before != after)
	}
	correction := liveAggregate(binding, "AAA", t0.Add(-time.Second), 9, 1)
	correction.Values.Close, correction.Values.High, correction.Values.VWAP = 13, 13, 13
	applyAggregate(t, target, correction, DispositionAggregateRevised, ReasonNone)
	state := aggregateState(t, target, "AAA")
	if state.tail[correction.WindowStart.Unix()].values.Close != 13 || state.tail[correction.WindowStart.Unix()].authority.live.ConnectionEpoch != 9 {
		t.Fatal("restored authority rejected new-run correction")
	}

	// Construction inspection: installation is admitted through the FIFO and
	// has exactly one whole-state owner apply; no per-symbol apply is visible.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "checkpoint.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	wholeApplies := 0
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			sel, ok := lhs.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "state" {
				continue
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "e" {
				wholeApplies++
			}
		}
		return true
	})
	if wholeApplies != 1 {
		t.Fatalf("checkpoint whole-state apply paths=%d, want 1", wholeApplies)
	}
	closeAndWait(t, source)
	closeAndWait(t, target)
}

func projectForTest(t *testing.T, e *Engine) CheckpointProjectionResult {
	t.Helper()
	admission, ch := e.AdmitCheckpointProjection(context.Background())
	if admission != AdmissionAdmitted || ch == nil {
		t.Fatalf("projection admission=%s", admission)
	}
	select {
	case r := <-ch:
		if r.Disposition != CheckpointProjected {
			t.Fatalf("projection=%+v", r)
		}
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("projection timeout")
		return CheckpointProjectionResult{}
	}
}
func installForTest(t *testing.T, e *Engine, image checkpoint.Image) CheckpointInstallResult {
	t.Helper()
	candidate := checkpoint.Candidate{Image: image, Checksum: strings.Repeat("a", 64), Integrity: true}
	admission, ch := e.AdmitCheckpointInstall(context.Background(), candidate)
	if admission != AdmissionAdmitted || ch == nil {
		t.Fatalf("install admission=%s", admission)
	}
	select {
	case r := <-ch:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("install timeout")
		return CheckpointInstallResult{}
	}
}
func freshBoundEngine(t *testing.T, binding reference.Binding, now *time.Time) *Engine {
	t.Helper()
	return aggregateEngine(t, binding, RunModeLive, now)
}

func checkpointSymbolsForTest(t *testing.T, e *Engine, t0 time.Time) []checkpoint.Symbol {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]checkpoint.Symbol, len(e.state.binding.symbols))
	for i := range out {
		var err error
		out[i], err = projectCheckpointSymbol(e.state.binding, &e.state.binding.symbols[i], e.state.aggregateEvaluator, checkpointProjectionCommittedMarkerForState(e.state.binding.symbols[i].aggregates), i, t0)
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}
