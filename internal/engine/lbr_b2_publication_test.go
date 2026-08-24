package engine

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestPLBRB2SelectedEnrichmentAndPublication proves the B2 boundary at the
// production evaluator/publication seam. The compacted prefix deliberately
// extends beyond candidate T; all five selected fields must still equal the
// exact [session start,T) projection, while the unselected symbol keeps its
// prior cached scalars and no canonical revision can be joined after selection.
func TestPLBRB2SelectedEnrichmentAndPublication(t *testing.T) {
	publicBinding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
	candidate := publicBinding.SessionStart().Add(330 * time.Second)
	now := candidate
	e := aggregateEngine(t, publicBinding, RunModeLive, &now)
	defer closeAndWait(t, e)
	binding := e.state.binding
	selected := buildB2FutureFoldedState(binding, candidate)
	binding.symbols[0].aggregates = selected
	installEvaluatorMarkOnSymbol(&binding.symbols[1], candidate, 11, qualificationNotYetPassed)
	installExactCoverage(binding.symbols[1].aggregates, binding, binding.sessionStart, candidate, nil)
	e.state.lifecycle, e.state.committedT, e.state.latestTarget = lifecycleLive, immutableTime(candidate), immutableTime(candidate)
	e.state.clockMonotonic, e.state.aggregateProjectionPending = true, true
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(candidate)
	maintainCurrentFieldStatuses(binding, &binding.symbols[1], candidate, nil)

	unselected := binding.symbols[1].aggregates
	unselectedPrice, unselectedMeasurements := unselected.priceRange.result, unselected.mvpMeasurements.result

	staged := e.stageAggregateEvaluationLocked(candidate)
	if invalid := validateAggregateEvaluation(staged); invalid != nil {
		t.Fatalf("B2 staged evaluation invalid: %+v", invalid)
	}
	if staged.mode != rankingQualifiedCurrent || !staged.selectedEnrichmentBound || staged.enrichedRows != 1 || len(staged.rows) != 1 {
		t.Fatalf("selected enrichment accounting = mode=%s bound=%v enriched=%d rows=%d", staged.mode, staged.selectedEnrichmentBound, staged.enrichedRows, len(staged.rows))
	}
	row := staged.rows[0]
	wantMove := 100 * (13.29/12.99 - 1)
	if row.symbol != "AAA" || row.last != 13.29 ||
		row.sessionVolume != currentField(360) || row.fromOpenPercent != currentField(32.9) ||
		row.dayRange.status != featureCurrent || absEvaluator(row.dayRange.value-100) > 1e-12 || row.activity30s != currentField(100) ||
		row.move30s.status != featureCurrent || absEvaluator(row.move30s.value-wantMove) > 1e-12 {
		t.Fatalf("exact as-of-T selected row = %+v, want volume=360 from-open=32.9 range=100 activity=100 move=%.12f", row, wantMove)
	}
	if staged.updates[1].enriched || unselected.priceRange.result != unselectedPrice || unselected.mvpMeasurements.result != unselectedMeasurements {
		t.Fatalf("unselected symbol was enriched: update=%+v", staged.updates[1])
	}

	view := e.selectedAggregateViewAtLocked(0, candidate, row.canonicalRevision)
	if !view.RevisionMatched || !view.CandidateT.Equal(candidate) || !view.PrefixFoldedThrough.After(candidate) || view.PrefixUsableAtCandidate ||
		view.PrefixVolume != 0 || view.PrefixPrints != 0 || view.PrefixFirstOpen != 0 || view.PrefixHigh != 0 || view.PrefixLow != 0 ||
		view.PrefixFirstOpenTrusted || view.PrefixExtremaTrusted || view.PrefixLatestTrusted || len(view.Tail) != 0 ||
		!view.MarkAvailable || !view.TrustedMarkAt.Equal(candidate.Add(-time.Second)) {
		t.Fatalf("bound selected projection = %+v", view)
	}

	// Drive the real owner transition, evaluator acceptance, T/Q reconciliation,
	// publication validation, and sole atomic-store decision.
	admission, completion := e.AdmitTimer(context.Background())
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("B2 timer admission=%s", admission)
	}
	if disposition := awaitTimerDisposition(t, completion); disposition.Code != DispositionTimerApplied {
		t.Fatalf("B2 production publication disposition=%+v", disposition)
	}
	snapshot := e.ObserveSnapshot()
	if snapshot.Publication.PublicationID == 0 || snapshot.Publication.Watermark == nil || !snapshot.Publication.Watermark.Equal(candidate) ||
		!snapshot.Publication.CurrentMarketClaim || len(snapshot.Publication.AggregateEvaluation.Rows) != 1 {
		t.Fatalf("atomic B2 snapshot = %+v", snapshot.Publication)
	}
	published := snapshot.Publication.AggregateEvaluation.Rows[0]
	if published.SessionVolume.Value != 360 || published.FromOpenPercent.Value != 32.9 || absEvaluator(published.DayRange.Value-100) > 1e-12 ||
		published.Activity30s.Value != 100 || absEvaluator(published.Move30s.Value-wantMove) > 1e-12 {
		t.Fatalf("published selected fields = %+v", published)
	}

	// A revision after selection cannot be silently joined into the candidate.
	selected.canonicalRevision++
	rejected := e.stageAggregateEvaluationLocked(candidate)
	if rejected.invalidSupport || len(rejected.rows) != 1 || rejected.rows[0].canonicalRevision != selected.canonicalRevision {
		t.Fatalf("fresh candidate did not bind the revised canonical state: %+v", rejected)
	}
	stale := cloneAggregateEvaluation(staged)
	e.enrichSelectedRowsLocked(&stale, candidate)
	if !stale.invalidSupport || stale.invalidSupportReason != "selected_canonical_revision" {
		t.Fatalf("stale selected revision reached enrichment: %+v", stale)
	}
}

func TestPLBRB2QualificationTrustClosure(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	candidate := binding.SessionStart().Add(60 * time.Second)
	now := candidate
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)

	e.mu.Lock()
	symbol := &e.state.binding.symbols[0]
	installEvaluatorMarkOnSymbol(symbol, candidate, 12, qualificationProvisional)
	state := symbol.aggregates
	if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, candidate, nil) {
		e.mu.Unlock()
		t.Fatal("qualification-closure coverage setup failed")
	}
	e.state.lifecycle, e.state.committedT, e.state.latestTarget = lifecycleLive, immutableTime(candidate), immutableTime(candidate)
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.state.aggregateAckPosition, e.state.aggregateAckReceivedAt = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, candidate
	e.state.clockMonotonic, e.state.aggregateProjectionPending = true, true
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(candidate)
	e.mu.Unlock()

	_, initialCompletion := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, initialCompletion); got.Code != DispositionTimerApplied {
		t.Fatalf("initial provisional publication=%+v", got)
	}
	before := e.observePublication()
	startsBefore := e.ObserveEvaluationTiming().Starts
	if before.aggregateEvaluation.mode != rankingQualifiedCurrent || len(before.aggregateEvaluation.rows) != 1 || before.aggregateEvaluation.qualification.provisional != 1 {
		t.Fatalf("initial sole-proof publication=%+v", before.aggregateEvaluation)
	}

	correction := liveAggregate(binding, "AAA", candidate.Add(-time.Second), 1, 2)
	correction.Values.Open, correction.Values.High, correction.Values.Low, correction.Values.Close, correction.Values.VWAP = 12, 12, 12, 12, 12
	correction.Values.Volume, correction.Values.AverageTradeSize = 1, 0
	if got := admitProductionAggregate(t, e, correction); got.Code != DispositionAggregateRevised {
		t.Fatalf("sole-proof correction=%+v", got)
	}
	after := e.observePublication()
	timing := e.ObserveEvaluationTiming()
	if after.publicationID <= before.publicationID || after.watermark == nil || before.watermark == nil || !after.watermark.Equal(*before.watermark) ||
		after.aggregateEvaluation.mode != rankingQualifiedCurrent || len(after.aggregateEvaluation.rows) != 0 ||
		after.aggregateEvaluation.qualification.notYetPassed != 1 || after.aggregateEvaluation.qualification.provisional != 0 ||
		after.aggregateEvaluation.enrichedRows != 0 || !after.aggregateEvaluation.population.reconciles() {
		t.Fatalf("sole mutable-proof revocation did not publish immediate same-T reclassification: before=%+v after=%+v", before, after)
	}
	if timing.Source != AggregateEvaluationTrustCorrection || !timing.Target.Equal(candidate) || timing.Starts.TrustCorrection != startsBefore.TrustCorrection+1 {
		t.Fatalf("qualification trust cycle attribution=%+v before=%+v", timing, startsBefore)
	}
	_, duplicate := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, duplicate); got.Code != DispositionTimerApplied || e.observePublication().publicationID != after.publicationID || e.ObserveEvaluationTiming().Starts != timing.Starts {
		t.Fatalf("timer duplicated same trust revision/T: disposition=%+v before=%+v after=%+v", got, timing, e.ObserveEvaluationTiming())
	}
}

func TestPLBRB2FutureStructuralBoundaryActivation(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	t0 := binding.SessionStart().Add(60 * time.Second)
	now := t0.Add(2 * time.Second)
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)
	e.mu.Lock()
	installEvaluatorMarkOnSymbol(&e.state.binding.symbols[0], t0, 12, qualificationFinalized)
	state := e.state.binding.symbols[0].aggregates
	installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, t0, nil)
	e.state.lifecycle, e.state.committedT, e.state.latestTarget = lifecycleLive, immutableTime(t0), immutableTime(t0)
	e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = 1, true, true
	e.state.aggregateAckPosition, e.state.aggregateAckReceivedAt = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, t0
	e.state.clockMonotonic, e.state.aggregateProjectionPending = true, true
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 1, 1
	e.state.hydration.supportedThrough = immutableTime(t0)
	e.mu.Unlock()
	_, initial := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, initial); got.Code != DispositionTimerApplied || len(e.observePublication().aggregateEvaluation.rows) != 1 {
		t.Fatalf("initial structural-boundary publication=%+v", got)
	}
	initialID := e.observePublication().publicationID
	initialStarts := e.ObserveEvaluationTiming().Starts
	for offset, frame := range []uint64{2, 3} {
		input := liveAggregate(binding, "AAA", t0.Add(time.Duration(offset)*time.Second), 1, frame)
		input.Values.High = 0
		input.DeliveryTime = now
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateRejected || got.Reason != ReasonStructural {
			t.Fatalf("future structural offset=%d disposition=%+v", offset, got)
		}
	}
	if e.observePublication().publicationID != initialID || e.ObserveEvaluationTiming().Starts != initialStarts || len(state.invalidStarts) != 2 || len(state.invalidStarts) > maximumTailRecords {
		t.Fatalf("future structural evidence affected T0 or exceeded bound: publication=%+v starts=%+v invalid=%v", e.observePublication(), e.ObserveEvaluationTiming().Starts, state.invalidStarts)
	}
	advance := func(target time.Time, frame, marker uint64) {
		command, err := e.IssueLiveCoverageFence()
		if err != nil {
			t.Fatal(err)
		}
		input, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, frame, marker, target)
		if err != nil {
			t.Fatal(err)
		}
		_, completion := e.AdmitLiveCoverageFence(context.Background(), input)
		if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
			t.Fatalf("advance %s=%+v", target, got)
		}
	}
	now = t0.Add(3 * time.Second)
	advance(t0.Add(time.Second), 3, 2)
	afterT1 := e.observePublication()
	if afterT1.publicationID <= initialID || afterT1.watermark == nil || !afterT1.watermark.Equal(t0.Add(time.Second)) ||
		len(afterT1.aggregateEvaluation.rows) != 0 || afterT1.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("T evidence did not close at T+1: %+v", afterT1)
	}
	startsAtT1 := e.ObserveEvaluationTiming().Starts
	now = t0.Add(4 * time.Second)
	advance(t0.Add(2*time.Second), 3, 3)
	invalidAtT2, invalidAtT2OK := e.invalidMarkBeforeLocked(0, t0.Add(2*time.Second))
	if !invalidAtT2OK || !invalidAtT2.windowStart.Equal(t0.Add(time.Second)) || e.ObserveEvaluationTiming().Starts.LiveCoverageFence != startsAtT1.LiveCoverageFence+1 || len(state.invalidStarts) != 2 {
		t.Fatalf("T+1 evidence did not become effective strictly at T+2: invalid=%+v/%t starts=%+v occupancy=%d", invalidAtT2, invalidAtT2OK, e.ObserveEvaluationTiming().Starts, len(state.invalidStarts))
	}
}

func buildB2FutureFoldedState(binding *installedBinding, candidate time.Time) *symbolAggregateState {
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), presence: &slotBitmap{}, provenAbsent: &slotBitmap{}, canonicalRevision: 41}
	var eligible canonicalAggregate
	for second := 0; second < 350; second++ {
		start := binding.sessionStart.Add(time.Duration(second) * time.Second)
		close := 10 + float64(second)/100
		volume := 1.0
		if second >= 300 && second < 330 {
			volume = 2
		}
		if second >= 330 {
			close, volume = 1_000+float64(second), 10_000
		}
		record := canonicalAggregate{
			identity: aggregateIdentity{symbol: "AAA", start: start.Unix()}, windowStart: start, windowEnd: start.Add(time.Second),
			values: AggregateValues{Open: close, High: close, Low: close, Close: close, Volume: volume, VWAP: close, AverageTradeSize: 1, ATSProvenance: ATSLiveProviderAverage},
		}
		state.prefix.fold(record)
		foldPriceRangeAggregate(state, binding, record)
		foldMVPMeasurementAggregate(state, record)
		state.presence.set(sessionSlot(binding, start))
		if start.Equal(candidate.Add(-time.Second)) {
			eligible = record
		}
		if state.latest == nil || start.After(state.latest.record.windowStart) {
			copyRecord := record
			state.latest = &latestAggregateMark{record: copyRecord}
		}
	}
	installCommittedSelection(state, candidate, committedMark(eligible))
	proofEnd := candidate.Add(-time.Minute)
	state.qualification = &qualificationState{finalized: true, finalProofEnd: proofEnd, finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{},
		accountedThrough: candidate, result: qualificationResult{at: candidate, status: qualificationFinalized, finalProofEnd: proofEnd}}
	future := candidate.Add(20 * time.Second)
	state.priceRange.result = evaluatePriceRangeFeaturesWithMark(binding, &coreSymbol{symbol: "AAA", prior: frozenPriorClose{status: reference.PriorCloseValid, close: 10}, aggregates: state}, future, state.latest.record, true)
	state.mvpMeasurements.result = evaluateMVPMeasurements(binding, state, future, nil)
	return state
}

func absEvaluator(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
