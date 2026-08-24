package engine

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// TestPMVPVolumeProductionPath closes P-MVP-VOLUME at the real ordered engine
// boundary. Canonical inserts, a live revision, and a historical ambiguity
// withdrawal each reach a newly sealed same-T publication; proven no-print
// seconds remain zero contribution and unknown/conflict evidence never becomes
// a partial current total. Checkpoint projection is deliberately outside this
// proof.
func TestPMVPVolumeProductionPath(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
	target := binding.SessionStart().Add(330 * time.Second)
	now := target
	e := aggregateEngine(t, binding, RunModeLive, &now)
	defer closeAndWait(t, e)
	historical := historicalAggregate(binding, "AAA", target.Add(-3*time.Second), 1)
	historical.Values.Open, historical.Values.High, historical.Values.Low, historical.Values.Close, historical.Values.VWAP = 12, 12, 12, 12, 12
	historical.Values.Volume = 7
	applyHistorical(t, e, historical, proofFor(binding, historical, e.state.binding.sessionStart, target), DispositionAggregateInserted, ReasonNone)

	e.mu.Lock()
	e.state.lifecycle, e.state.liveEpoch, e.state.liveEpochActive, e.state.aggregateAcknowledged = lifecycleLive, 1, true, true
	e.state.aggregateAckPosition = LivePosition{ConnectionEpoch: 1, FrameSequence: 1}
	e.state.aggregateAckReceivedAt = target
	e.state.hydration.fenceReconciled, e.state.hydration.fenceEpoch = true, 1
	e.state.hydration.fenceThrough, e.state.hydration.fenceMarkerOrdinal = 100, 1
	e.state.hydration.supportedThrough = immutableTime(target)
	for index := range e.state.binding.symbols {
		state := ensureAggregateState(&e.state.binding.symbols[index])
		if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, target, nil) {
			e.mu.Unlock()
			t.Fatal("exact no-print coverage setup failed")
		}
	}
	e.mu.Unlock()

	frame := uint64(1)
	insert := func(symbol string, at time.Time, volume, close float64) AggregateInput {
		frame++
		input := liveAggregate(binding, symbol, at, 1, frame)
		input.Values.Open, input.Values.High, input.Values.Low, input.Values.Close, input.Values.VWAP = close, close, close, close, close
		input.Values.Volume = volume
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
			t.Fatalf("insert %s/%s = %+v", symbol, at, got)
		}
		return input
	}
	aaaMutable := insert("AAA", target.Add(-2*time.Second), 10, 12)
	insert("AAA", target.Add(-time.Second), 20, 12)
	insert("BBB", target.Add(-time.Second), 5, 11)
	e.mu.Lock()
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		proofEnd := target.Add(-time.Minute)
		state.qualification = &qualificationState{finalized: true, finalProofEnd: proofEnd, finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{},
			accountedThrough: target, result: qualificationResult{at: target, status: qualificationFinalized, finalProofEnd: proofEnd}}
	}
	e.mu.Unlock()

	publish := func() publicationView {
		admission, completion := e.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
			t.Fatal("volume proof timer did not publish")
		}
		return e.observePublication()
	}
	initial := publish()
	assertVolumePublication := func(publication publicationView, status aggregateFeatureStatus, reason aggregateFeatureReason, value float64) aggregateRankingRow {
		t.Helper()
		if publication.watermark == nil || !publication.watermark.Equal(target) || publication.aggregateEvaluation.mode != rankingQualifiedCurrent || len(publication.aggregateEvaluation.rows) != 2 {
			t.Fatalf("coherent volume publication = %+v", publication.aggregateEvaluation)
		}
		row := publication.aggregateEvaluation.rows[0]
		if row.symbol != "AAA" || row.sessionVolume.status != status || row.sessionVolume.reason != reason || status == featureCurrent && row.sessionVolume.value != value {
			t.Fatalf("AAA Volume row = %+v", row)
		}
		return row
	}
	initialRow := assertVolumePublication(initial, featureCurrent, featureReasonNone, 37)

	revision := aaaMutable
	frame++
	revision.Live.FrameSequence, revision.Values.Volume = frame, 15
	if got := admitProductionAggregate(t, e, revision); got.Code != DispositionAggregateRevised {
		t.Fatalf("accepted Volume revision = %+v", got)
	}
	if afterValueOnly := e.observePublication(); afterValueOnly.publicationID != initial.publicationID {
		t.Fatalf("same-trust value revision bypassed one-second coalescing: before=%d after=%d", initial.publicationID, afterValueOnly.publicationID)
	}
	revised := publish()
	revisedRow := assertVolumePublication(revised, featureCurrent, featureReasonNone, 42)
	if revised.publicationID <= initial.publicationID || !revised.watermark.Equal(*initial.watermark) {
		t.Fatalf("same-T revision publication initial=%d revised=%d", initial.publicationID, revised.publicationID)
	}
	e.mu.Lock()
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	unknownSlot := sessionSlot(e.state.binding, target.Add(-4*time.Second))
	state.provenAbsent.clear(unknownSlot)
	e.state.aggregateProjectionPending = true
	e.mu.Unlock()
	unknown := publish()
	assertVolumePublication(unknown, featureUnavailable, featureReasonHistoryIncomplete, 0)
	e.mu.Lock()
	state.provenAbsent.set(unknownSlot)
	e.state.aggregateProjectionPending = true
	e.mu.Unlock()
	restored := publish()
	assertVolumePublication(restored, featureCurrent, featureReasonNone, 42)

	e.mu.Lock()
	e.state.lifecycle = lifecycleHydrating
	e.mu.Unlock()
	plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	var token HydrationRequestToken
	for _, request := range plan.Plan.Requests() {
		if request.Symbol() == "AAA" {
			token = request
			break
		}
	}
	if token.Symbol() != "AAA" {
		t.Fatalf("AAA hydration request unavailable: %+v", plan.Plan.Requests())
	}
	values := AggregateValues{Open: 12, High: 12, Low: 12, Close: 12, Volume: 9, VWAP: 12, AverageTradeSize: 1, ATSProvenance: ATSRESTFloorVolumeOverTrades}
	conflictRow, _ := NewHydrationRow("AAA", target.Add(-3*time.Second), target.Add(-2*time.Second), values)
	conflictChunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{conflictRow})
	beforeTrustClosure := e.observePublication()
	if got := admitHydrationChunk(t, e, conflictChunk); got.Code != DispositionHydrationChunkApplied || got.Rows != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) {
		t.Fatalf("production historical withdrawal = %+v", got)
	}
	immediateTrustClosure := e.observePublication()
	if immediateTrustClosure.publicationID <= beforeTrustClosure.publicationID || immediateTrustClosure.watermark == nil || !immediateTrustClosure.watermark.Equal(target) ||
		immediateTrustClosure.aggregateEvaluation.mode != rankingQualifiedCurrent || len(immediateTrustClosure.aggregateEvaluation.rows) != 2 ||
		immediateTrustClosure.aggregateEvaluation.rows[0].symbol != "AAA" || immediateTrustClosure.aggregateEvaluation.rows[0].rank != initialRow.rank ||
		immediateTrustClosure.aggregateEvaluation.rows[0].sessionVolume.status != featureInvalid || immediateTrustClosure.aggregateEvaluation.rows[0].sessionVolume.reason != featureReasonHistoricalConflict ||
		immediateTrustClosure.aggregateEvaluation.enrichedRows != uint32(len(immediateTrustClosure.aggregateEvaluation.rows)) {
		t.Fatalf("same-T selected trust closure did not publish coherently: before=%+v after=%+v", beforeTrustClosure, immediateTrustClosure)
	}
	var fence HydrationFenceCommand
	for _, request := range plan.Plan.Requests() {
		state, pages, rows := HydrationCompletedEmpty, int64(1), int64(0)
		if request.Symbol() == "AAA" {
			state, pages, rows = HydrationCompletedValue, 1, 1
		}
		terminal, err := NewHydrationTerminalInput(request, request.ResultID(), state, HydrationReasonNone, pages, pages, 10, rows, rows, rows)
		if err != nil {
			t.Fatal(err)
		}
		result := admitHydrationTerminal(t, e, terminal)
		if result.Code != DispositionHydrationTerminalApplied {
			t.Fatalf("Volume proof terminal %s = %+v", request.Symbol(), result)
		}
		if result.FenceCommand.CommandToken() != 0 {
			fence = result.FenceCommand
		}
	}
	if fence.CommandToken() == 0 {
		t.Fatal("Volume proof hydration produced no fence")
	}
	fenceInput, err := NewAggregateIngressFenceInput(fence, AggregateIngressFenceComplete, frame, 2, target)
	if err != nil {
		t.Fatal(err)
	}
	admission, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fenceInput)
	if admission != AdmissionAdmitted || fenceCompletion == nil {
		t.Fatalf("Volume proof fence admission=%s", admission)
	}
	fenceResult := awaitHydrationDisposition(t, fenceCompletion)
	if fenceResult.Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("Volume proof fence=%+v", fenceResult)
	}
	e.mu.Lock()
	withdrawnState := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	withdrawnExactSum := sessionVolumeBefore(withdrawnState, target)
	e.mu.Unlock()
	if withdrawnExactSum != 35 {
		t.Fatalf("accepted withdrawal retained its prior contribution: exact sum=%v want=35", withdrawnExactSum)
	}
	withdrawn := publish()
	withdrawnRow := assertVolumePublication(withdrawn, featureInvalid, featureReasonHistoricalConflict, 0)
	operational := e.ObserveOperational()
	if operational.Lifecycle != "live" || !operational.CurrentMarketClaim || operational.RankingMode != "qualified_current" || !operational.Hydration.FenceReconciled {
		t.Fatalf("Volume withdrawal changed engine readiness inputs: %+v", operational)
	}
	if withdrawn.publicationID <= restored.publicationID || initialRow.symbol != revisedRow.symbol || revisedRow.symbol != withdrawnRow.symbol ||
		initialRow.rank != revisedRow.rank || revisedRow.rank != withdrawnRow.rank || initialRow.dayPercent != revisedRow.dayPercent || revisedRow.dayPercent != withdrawnRow.dayPercent ||
		initialRow.last != revisedRow.last || revisedRow.last != withdrawnRow.last || initialRow.float != revisedRow.float || revisedRow.float != withdrawnRow.float ||
		initialRow.fromOpenPercent != revisedRow.fromOpenPercent || initialRow.dayRange != revisedRow.dayRange ||
		initialRow.move30s != revisedRow.move30s ||
		!initialRow.tqIntentEligible || !revisedRow.tqIntentEligible || !withdrawnRow.tqIntentEligible {
		t.Fatalf("Volume mutation changed rank/readiness-independent fields initial=%+v revised=%+v withdrawn=%+v", initialRow, revisedRow, withdrawnRow)
	}

}

// TestPMVPVolumeActivityMoveExactCorrectionAndPermutation is the primary
// P-MVP-VOLUME / P-MVP-ACTIVITY-MOVE proof. It exercises the canonical
// present/absent coverage distinction, inclusive percentile ties, a revision,
// and order-independent folded reconstruction at the same committed boundary.
func TestPMVPVolumeActivityMoveExactCorrectionAndPermutation(t *testing.T) {
	binding := mvpTestBinding()
	at := binding.sessionStart.Add(330 * time.Second)
	records := make([]canonicalAggregate, 330)
	for i := range records {
		records[i] = mvpTestRecord(binding.sessionStart.Add(time.Duration(i)*time.Second), 1, 100+float64(i))
	}

	chronological := mvpFoldedState(binding, records, false)
	reversed := mvpFoldedState(binding, records, true)
	wantMove := 100 * (records[329].values.Close/records[299].values.Close - 1)
	for name, state := range map[string]*symbolAggregateState{"chronological": chronological, "reversed": reversed} {
		got := evaluateMVPMeasurements(binding, state, at, nil)
		if got.sessionVolume != currentField(330) || got.activity30s != currentField(100) || got.move30s.status != featureCurrent || math.Abs(got.move30s.value-wantMove) > 1e-12 {
			t.Fatalf("%s measurements=%+v want move %.12f", name, got, wantMove)
		}
	}
	if chronological.mvpMeasurements.result != reversed.mvpMeasurements.result {
		t.Fatalf("arrival permutation changed committed result: chronological=%+v reversed=%+v", chronological.mvpMeasurements.result, reversed.mvpMeasurements.result)
	}

	// Revise one second in the target window. Re-folding the same canonical
	// identity replaces its contribution; no prior accumulation survives.
	revised := records[329]
	revised.values.Volume = 0
	revised.values.Close = 500
	foldMVPMeasurementAggregate(chronological, revised)
	corrected := evaluateMVPMeasurements(binding, chronological, at, nil)
	wantCorrectedMove := 100 * (500/records[299].values.Close - 1)
	if corrected.sessionVolume != currentField(329) || corrected.activity30s != currentField(0) || math.Abs(corrected.move30s.value-wantCorrectedMove) > 1e-12 {
		t.Fatalf("corrected measurements=%+v want move %.12f", corrected, wantCorrectedMove)
	}

	unknown := mvpFoldedState(binding, records, false)
	unknown.provenAbsent.clear(sessionSlot(binding, binding.sessionStart.Add(10*time.Second)))
	unknown.presence.clear(sessionSlot(binding, binding.sessionStart.Add(10*time.Second)))
	if got := evaluateMVPMeasurements(binding, unknown, at, nil); got.sessionVolume.reason != featureReasonHistoryIncomplete || got.activity30s.reason != featureReasonHistoryIncomplete {
		t.Fatalf("unknown second became zero: %+v", got)
	}
	conflict := mvpFoldedState(binding, records, false)
	ensureHistoricalConflict(conflict).set(sessionSlot(binding, binding.sessionStart.Add(10*time.Second)))
	if got := evaluateMVPMeasurements(binding, conflict, at, nil); got.sessionVolume.status != featureInvalid || got.activity30s.status != featureInvalid {
		t.Fatalf("conflict became usable: %+v", got)
	}
}

func TestPMVPSparseProvenNoPrintAndRankInvariance(t *testing.T) {
	binding := mvpTestBinding()
	at := binding.sessionStart.Add(330 * time.Second)
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), provenAbsent: &slotBitmap{}}
	for slot := 0; slot < 330; slot++ {
		state.provenAbsent.set(slot)
	}
	for _, record := range []canonicalAggregate{
		mvpTestRecord(binding.sessionStart.Add(299*time.Second), 3, 10),
		mvpTestRecord(binding.sessionStart.Add(329*time.Second), 7, 11),
	} {
		copyRecord := record
		state.tail[record.identity.start] = &copyRecord
		state.provenAbsent.clear(sessionSlot(binding, record.windowStart))
	}
	got := evaluateMVPMeasurements(binding, state, at, nil)
	if got.sessionVolume != currentField(10) || got.activity30s != currentField(100) || got.move30s != currentField(10) {
		t.Fatalf("sparse proven-no-print measurements=%+v", got)
	}

	// P-MVP-RANK: Float and all five display measurements are absent from the
	// comparator. Changing them cannot change identity, membership, or order.
	a := aggregateRankingRow{symbol: "AAA", dayPercent: 5, float: aggregateFloatField{status: aggregateFloatUnavailable}}
	b := aggregateRankingRow{symbol: "BBB", dayPercent: 4, float: aggregateFloatField{status: aggregateFloatCurrent}, sessionVolume: currentField(1e9), activity30s: currentField(100), move30s: currentField(50)}
	if !rankingPrecedes(a, b) || rankingPrecedes(b, a) {
		t.Fatalf("display enrichment changed Day %% ordering a=%+v b=%+v", a, b)
	}
	a.float, a.sessionVolume, a.activity30s, a.move30s = b.float, b.sessionVolume, b.activity30s, b.move30s
	if !rankingPrecedes(a, b) || rankingPrecedes(b, a) {
		t.Fatal("rank comparator depends on Float or revised measurements")
	}
}

func TestPMVPFloatFieldProvenanceAndAbsence(t *testing.T) {
	retrieved := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	percent := 25.0
	lookup, err := reference.NewFloatLookup([]reference.FloatFact{{Ticker: "AAA", FreeFloat: 12_345_678, FreeFloatPercent: &percent, Provider: "massive-stocks-float-experimental", RetrievedAt: retrieved, Provenance: reference.FloatFresh}}, reference.FloatFresh, retrieved)
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{floatLookup: lookup}
	current := e.floatForSymbol("AAA")
	if current.status != aggregateFloatCurrent || current.fact.FreeFloat != 12_345_678 || current.fact.RetrievedAt != retrieved {
		t.Fatalf("fresh Float field=%+v", current)
	}
	rowA := aggregateRankingRow{symbol: "AAA", dayPercent: 1, float: current}
	rowB := aggregateRankingRow{symbol: "AAA", dayPercent: 1, float: e.floatForSymbol("AAA")}
	if !aggregateRankingRowEqual(rowA, rowB) {
		t.Fatal("equal Float percentages compared by pointer identity")
	}
	view := replayFloatFieldView(current)
	*view.Percent = 99
	if *current.fact.FreeFloatPercent != 25 {
		t.Fatal("Float diagnostic view aliases immutable engine publication")
	}
	missing := e.floatForSymbol("BBB")
	if missing.status != aggregateFloatUnavailable || missing.fact.FreeFloat != 0 {
		t.Fatalf("missing Float was not explicit unavailable: %+v", missing)
	}

	staleLookup, err := reference.NewFloatLookup([]reference.FloatFact{{Ticker: "AAA", FreeFloat: 12_345_678, Provider: "massive-stocks-float-experimental", RetrievedAt: retrieved, Provenance: reference.FloatCache}}, reference.FloatCache, retrieved)
	if err != nil {
		t.Fatal(err)
	}
	e.floatLookup = staleLookup
	if stale := e.floatForSymbol("AAA"); stale.status != aggregateFloatStale || stale.reason != "cached_fallback" {
		t.Fatalf("cached Float lost stale provenance: %+v", stale)
	}
}

// TestPMVPRankSelectionAndPreselectionHistory drives the full evaluator heap,
// qualification, and top-20 fill. A 21st symbol retains exact 330-second
// history plus a Move predecessor while unselected, then enters solely because
// another symbol falls below it. Float population varies independently across
// an otherwise identical engine and cannot alter membership or order.
func TestPMVPRankSelectionAndPreselectionHistory(t *testing.T) {
	at := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	specs := make([]evaluatorSymbol, 21)
	for i := range specs {
		specs[i] = evaluatorSymbol{symbol: fmt.Sprintf("S%02d", i), priorStatus: reference.PriorCloseValid, prior: 10, mark: 40 - float64(i), qualification: qualificationProvisional}
	}
	unselectedAt := at.Add(-331 * time.Second)
	base := evaluatorProofEngine(unselectedAt, specs)
	earlier := base.stageAggregateEvaluationLocked(unselectedAt)
	if at.Sub(unselectedAt) <= 330*time.Second || len(earlier.rows) != 20 || earlier.rows[19].symbol != "S19" {
		t.Fatalf("strict >330s unselected baseline T=%s rows=%v", unselectedAt, rankingSymbols(earlier.rows))
	}
	for _, row := range earlier.rows {
		if row.symbol == "S20" {
			t.Fatal("future entrant was displayed at the >330s unselected baseline")
		}
	}
	// Advance the same canonical owner by 331 seconds. No symbol-local state is
	// replaced: the old unselected mark remains session evidence while one new
	// mark and exact intervening no-print coverage reach the later candidate.
	for index := range base.state.binding.symbols {
		symbol := &base.state.binding.symbols[index]
		state := symbol.aggregates
		record := mvpTestRecord(at.Add(-time.Second), 1, specs[index].mark)
		record.identity.symbol = symbol.symbol
		state.tail[record.identity.start] = &record
		state.latest = &latestAggregateMark{record: record}
		state.canonicalRevision++
		ensurePresence(state).set(sessionSlot(base.state.binding, record.windowStart))
		retainMutableMVPMeasurement(state, record)
		retainMutablePriceRangeEvidence(ensurePriceRangeState(state), record)
		installExactCoverage(state, base.state.binding, base.state.binding.sessionStart, at, nil)
		state.qualification.accountedThrough = at
		state.qualification.result.at = at
		ensurePriceRangeState(state).result = evaluatePriceRangeFeatures(base.state.binding, symbol, at)
		ensureMVPMeasurementState(state).result = evaluateMVPMeasurements(base.state.binding, state, at, nil)
	}
	base.state.committedT = immutableTime(at)
	outsider := &base.state.binding.symbols[20]
	baseRecord := mvpTestRecord(at.Add(-31*time.Second), 5, 19)
	baseRecord.identity.symbol = outsider.symbol
	outsider.aggregates.tail[baseRecord.identity.start] = &baseRecord
	retainMutableMVPMeasurement(outsider.aggregates, baseRecord)
	retainMutablePriceRangeEvidence(ensurePriceRangeState(outsider.aggregates), baseRecord)
	ensurePresence(outsider.aggregates).set(sessionSlot(base.state.binding, baseRecord.windowStart))
	outsider.aggregates.provenAbsent.clear(sessionSlot(base.state.binding, baseRecord.windowStart))
	ensurePresence(outsider.aggregates).set(sessionSlot(base.state.binding, at.Add(-time.Second)))

	preselection := evaluateMVPMeasurements(base.state.binding, outsider.aggregates, at, nil)
	floor := at.Add(-330 * time.Second)
	present, absent := 0, 0
	for second := floor; second.Before(at); second = second.Add(time.Second) {
		slot := sessionSlot(base.state.binding, second)
		switch {
		case outsider.aggregates.presence.has(slot):
			present++
		case outsider.aggregates.provenAbsent.has(slot):
			absent++
		default:
			t.Fatalf("outsider lost exact retained evidence at %s", second)
		}
		if outsider.aggregates.historicalConflict != nil && outsider.aggregates.historicalConflict.has(slot) {
			t.Fatalf("outsider retained conflicting evidence at %s", second)
		}
	}
	wantActivity := 100 * float64(54) / 55
	wantMove := 100 * (20.0/19.0 - 1)
	// Independently enumerate the exact target and every non-overlapping
	// reference endpoint. This oracle deliberately does not call the prefix-sum
	// evaluator, so an endpoint/indexing defect cannot hide behind the final
	// 54/55 scalar.
	windowRate := func(end time.Time) float64 {
		total := 0.0
		for second := end.Add(-30 * time.Second); second.Before(end); second = second.Add(time.Second) {
			if aggregate, ok := aggregateAt(outsider.aggregates, second.Unix()); ok {
				total += aggregate.volume
			}
		}
		return total / 30
	}
	targetRate := windowRate(at)
	if targetRate != 1.0/30 {
		t.Fatalf("Activity target rate=%v want=%v", targetRate, 1.0/30)
	}
	referenceLessOrEqual := 0
	for k := 0; k < 55; k++ {
		endpoint := at.Add(-30*time.Second - time.Duration(k)*5*time.Second)
		wantRate := 0.0
		if k == 0 {
			wantRate = 5.0 / 30
		}
		if gotRate := windowRate(endpoint); gotRate != wantRate {
			t.Fatalf("Activity reference k=%d endpoint=%s rate=%v want=%v", k, endpoint, gotRate, wantRate)
		}
		if wantRate <= targetRate {
			referenceLessOrEqual++
		}
	}
	if referenceLessOrEqual != 54 {
		t.Fatalf("Activity inclusive reference count=%d want=54", referenceLessOrEqual)
	}
	predecessor, predecessorOK := markStrictlyBefore(outsider.aggregates, at.Add(-30*time.Second))
	if present != 2 || absent != 328 || !exactAggregateCoverage(outsider.aggregates, base.state.binding, floor, at) ||
		!predecessorOK || predecessor.start != baseRecord.windowStart || preselection.sessionVolume != currentField(7) ||
		preselection.activity30s.status != featureCurrent || math.Abs(preselection.activity30s.value-wantActivity) > 1e-12 ||
		preselection.move30s.status != featureCurrent || math.Abs(preselection.move30s.value-wantMove) > 1e-12 {
		t.Fatalf("retained outsider evidence present=%d absent=%d predecessor=%+v/%t measurements=%+v", present, absent, predecessor, predecessorOK, preselection)
	}
	first := base.stageAggregateEvaluationLocked(at)
	if len(first.rows) != 20 || first.rows[19].symbol != "S19" {
		t.Fatalf("initial top-20 fill=%v", rankingSymbols(first.rows))
	}
	for _, row := range first.rows {
		if row.symbol == outsider.symbol {
			t.Fatal("21st symbol entered before rank change")
		}
	}

	// Change only S19. S20's canonical history and revised measurements remain
	// byte-for-byte identical while it crosses the selection boundary.
	competitor := base.state.binding.symbols[19].aggregates.tail[at.Add(-time.Second).Unix()]
	competitor.values.Open, competitor.values.High, competitor.values.Low, competitor.values.Close, competitor.values.VWAP = 10.1, 10.1, 10.1, 10.1, 10.1
	base.state.binding.symbols[19].aggregates.latest.record = *competitor
	second := base.stageAggregateEvaluationLocked(at)
	if len(second.rows) != 20 || second.rows[19].symbol != outsider.symbol {
		t.Fatalf("post-change top-20 fill=%v", rankingSymbols(second.rows))
	}
	entered := second.rows[19]
	if entered.sessionVolume != preselection.sessionVolume || entered.activity30s != preselection.activity30s || entered.move30s != preselection.move30s {
		t.Fatalf("selection changed retained measurements row=%+v preselection=%+v", entered, preselection)
	}
	freshSelected := &tqSymbolState{tradeCoverage: tqCoverage{active: true, start: at}, quoteCoverage: tqCoverage{active: true, start: at}}
	tape, spread := tapeView(freshSelected, &at), spreadView(freshSelected, &at)
	if !entered.tqIntentEligible || tape.Status != TQWarming || tape.Reason != "coverage_warming" || spread.Status != TQWarming || spread.Reason != "coverage_warming" {
		t.Fatalf("new selection did not isolate warm-up to T/Q: row=%+v tape=%+v spread=%+v", entered, tape, spread)
	}

	withFloat := evaluatorProofEngine(at, specs)
	retrieved := time.Date(2026, 8, 14, 17, 59, 0, 0, time.UTC)
	facts := make([]reference.FloatFact, 0, 10)
	for i := 0; i < 21; i += 2 {
		facts = append(facts, reference.FloatFact{Ticker: specs[i].symbol, FreeFloat: 1_000_000 + float64(i), Provider: "massive-stocks-float-experimental", RetrievedAt: retrieved, Provenance: reference.FloatFresh})
	}
	lookup, err := reference.NewFloatLookup(facts, reference.FloatFresh, retrieved)
	if err != nil {
		t.Fatal(err)
	}
	withFloat.floatLookup = lookup
	withoutRows, withRows := evaluatorProofEngine(at, specs).stageAggregateEvaluationLocked(at).rows, withFloat.stageAggregateEvaluationLocked(at).rows
	if fmt.Sprint(rankingSymbols(withoutRows)) != fmt.Sprint(rankingSymbols(withRows)) {
		t.Fatalf("Float availability changed membership/order without=%v with=%v", rankingSymbols(withoutRows), rankingSymbols(withRows))
	}
	for i := range withoutRows {
		if withoutRows[i].rank != withRows[i].rank || withoutRows[i].tqIntentEligible != withRows[i].tqIntentEligible {
			t.Fatalf("Float changed rank/qualification at %d: without=%+v with=%+v", i, withoutRows[i], withRows[i])
		}
	}
	cachedFacts := make([]reference.FloatFact, len(facts))
	for i := range facts {
		cachedFacts[i] = facts[i]
		cachedFacts[i].Provenance = reference.FloatCache
	}
	cachedLookup, err := reference.NewFloatLookup(cachedFacts, reference.FloatCache, retrieved)
	if err != nil {
		t.Fatal(err)
	}
	baseline := evaluatorProofEngine(at, specs).stageAggregateEvaluationLocked(at)
	for name, candidateLookup := range map[string]reference.FloatLookup{
		"fresh with missing symbols": lookup,
		"validated cached":           cachedLookup,
		"unavailable":                {},
	} {
		candidateEngine := evaluatorProofEngine(at, specs)
		candidateEngine.floatLookup = candidateLookup
		candidate := candidateEngine.stageAggregateEvaluationLocked(at)
		if candidate.mode != baseline.mode || candidate.reason != baseline.reason || candidate.qualification != baseline.qualification ||
			fmt.Sprint(rankingSymbols(candidate.rows)) != fmt.Sprint(rankingSymbols(baseline.rows)) {
			t.Fatalf("%s Float changed readiness/ranking: baseline=%+v candidate=%+v", name, baseline, candidate)
		}
		for i := range baseline.rows {
			if candidate.rows[i].rank != baseline.rows[i].rank || candidate.rows[i].dayPercent != baseline.rows[i].dayPercent ||
				candidate.rows[i].tqIntentEligible != baseline.rows[i].tqIntentEligible {
				t.Fatalf("%s Float changed row %d: baseline=%+v candidate=%+v", name, i, baseline.rows[i], candidate.rows[i])
			}
		}
	}
}

func rankingSymbols(rows []aggregateRankingRow) []string {
	result := make([]string, len(rows))
	for i := range rows {
		result[i] = rows[i].symbol
	}
	return result
}

func mvpTestBinding() *installedBinding {
	start := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	return &installedBinding{identity: "mvp-measurements", tradingDate: "2026-08-14", sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
}

func mvpTestRecord(start time.Time, volume, close float64) canonicalAggregate {
	return canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: start.Unix()}, windowStart: start, windowEnd: start.Add(time.Second),
		values: AggregateValues{Open: close, High: close, Low: close, Close: close, Volume: volume, VWAP: close, ATSProvenance: ATSLiveProviderAverage}}
}

func mvpFoldedState(binding *installedBinding, records []canonicalAggregate, reverse bool) *symbolAggregateState {
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), presence: &slotBitmap{}, provenAbsent: &slotBitmap{}}
	for i := range records {
		record := records[i]
		if reverse {
			record = records[len(records)-1-i]
		}
		state.presence.set(sessionSlot(binding, record.windowStart))
		foldMVPMeasurementAggregate(state, record)
	}
	state.mvpMeasurements.result = evaluateMVPMeasurements(binding, state, binding.sessionStart.Add(330*time.Second), nil)
	return state
}
