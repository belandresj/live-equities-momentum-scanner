package engine

import (
	"context"
	"testing"
	"time"
)

type d3PopulationTransitionTrace struct {
	rowAccounting         HydrationRowAccounting
	requestCoverage       hydrationCoverage
	bootstrapCoverage     aggregateCoverageConsequence
	bootstrapCoverageSet  bool
	laterLiveDisposition  AggregateDisposition
	ordinaryCoverage      aggregateCoverageConsequence
	ordinaryCoverageSet   bool
	conflictAtOldIdentity bool
	latestMarkWindow      time.Time
	latestMarkAuthority   AggregateSource
	features              ReplayFeatureView
	qualification         ReplayQualificationView
	evaluation            ReplayEvaluationView
}

// TestD3OneSymbolHydrationConflictPopulationTransition is
// P-D3-POPULATION-TRANSITION. The matched cases differ solely in whether one
// REST row is equal to or differs from an already accepted live aggregate.
// Valid live precedence must make that discrepancy diagnostic-only through the
// same bootstrap and ordinary-live coverage fences.
func TestD3OneSymbolHydrationConflictPopulationTransition(t *testing.T) {
	control := runD3OneSymbolPopulationTransition(t, false)
	candidate := runD3OneSymbolPopulationTransition(t, true)

	if control.rowAccounting != (HydrationRowAccounting{Consumed: 1, Duplicate: 1}) ||
		control.requestCoverage != hydrationCoverageCandidateComplete {
		t.Fatalf("matched equal control did not establish complete request coverage: %+v", control)
	}
	if candidate.rowAccounting != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) ||
		candidate.requestCoverage != hydrationCoverageCandidateComplete {
		t.Fatalf("unequal overlap did not retain diagnostic accounting plus exact request coverage: %+v", candidate)
	}
	if control.bootstrapCoverageSet || candidate.bootstrapCoverageSet {
		t.Fatalf("resolved discrepancy installed bootstrap uncertainty: control=%+v candidate=%+v", control, candidate)
	}
	if control.laterLiveDisposition.Code != DispositionAggregateInserted || candidate.laterLiveDisposition.Code != DispositionAggregateInserted ||
		control.latestMarkWindow.IsZero() || candidate.latestMarkWindow != control.latestMarkWindow {
		t.Fatalf("later independent live mark was not matched and trustworthy: control=%+v candidate=%+v", control, candidate)
	}
	if control.ordinaryCoverageSet || candidate.ordinaryCoverageSet || candidate.conflictAtOldIdentity || control.conflictAtOldIdentity {
		t.Fatalf("resolved discrepancy poisoned ordinary coverage or installed conflict state: control=%+v candidate=%+v", control, candidate)
	}

	controlPopulation, candidatePopulation := control.evaluation.Population, candidate.evaluation.Population
	if controlPopulation.UniverseTotal != 1 || controlPopulation.TrustedRankableMark != 1 || controlPopulation.UnknownDueFailureOrFence != 0 ||
		controlPopulation.CoveredPopulation != 1 || controlPopulation.UnresolvedPopulation != 0 {
		t.Fatalf("matched control population did not resolve: %+v evaluation=%+v", controlPopulation, control.evaluation)
	}
	if candidatePopulation.UniverseTotal != 1 || candidatePopulation.TrustedRankableMark != 1 || candidatePopulation.UnknownDueFailureOrFence != 0 ||
		candidatePopulation.CoveredPopulation != 1 || candidatePopulation.UnresolvedPopulation != 0 || candidate.latestMarkAuthority != AggregateSourceLive {
		t.Fatalf("candidate did not trust the later independently authoritative live mark: %+v evaluation=%+v", candidatePopulation, candidate.evaluation)
	}
	if control.evaluation.PopulationTransition != (ReplayPopulationTransitionDiagnosticView{}) ||
		candidate.evaluation.PopulationTransition != (ReplayPopulationTransitionDiagnosticView{}) {
		t.Fatalf("diagnostic-only discrepancy entered the bootstrap-repair path: control=%+v candidate=%+v", control.evaluation.PopulationTransition, candidate.evaluation.PopulationTransition)
	}
	if candidate.features != control.features || candidate.qualification != control.qualification ||
		candidate.evaluation.Qualification != control.evaluation.Qualification || candidate.evaluation.Features != control.evaluation.Features ||
		candidate.evaluation.Mode != "qualified_current" {
		t.Fatalf("diagnostic-only discrepancy changed features or qualification: control=%+v candidate=%+v", control, candidate)
	}

	t.Logf("resolved_discrepancy=hydration_row conflict_or_withdrawal with exact request coverage; bootstrap_fence=%+v; later_live=%s at=%s; ordinary_fence=%+v; final_trusted=%d final_unresolved=%d qualification_unresolved=%d",
		candidate.bootstrapCoverage, candidate.laterLiveDisposition.Code, candidate.latestMarkWindow.Format(time.RFC3339), candidate.ordinaryCoverage,
		candidatePopulation.TrustedRankableMark, candidatePopulation.UnresolvedPopulation, candidate.evaluation.Qualification.Unresolved)
}

// TestResolvedRESTLiveDiscrepancyPreservesQualificationAndFeatures proves the
// trader-facing consequence of DTE-MERGE-04 live precedence. A fresh request
// contains one unequal REST/live overlap, then ordinary live coverage supplies
// the minimum complete session needed for qualification and Activity. The
// discrepancy remains counted but cannot suppress any aggregate-derived field
// or ranking when no other uncertainty exists.
func TestResolvedRESTLiveDiscrepancyPreservesQualificationAndFeatures(t *testing.T) {
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	now := start.Add(3 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	defer closeAndWait(t, e)

	overlap := liveAggregate(binding, "AAA", start, 1, 2)
	overlap.Live.ArrayIndex = 1
	overlap.Values = AggregateValues{Open: 10, High: 10.10, Low: 9.90, Close: 10, Volume: 5_000, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}
	if got := admitProductionAggregate(t, e, overlap); got.Code != DispositionAggregateInserted {
		t.Fatalf("initial live authority=%+v", got)
	}
	row, err := NewHydrationRow("AAA", start, start.Add(time.Second), AggregateValues{
		Open: 9, High: 9.25, Low: 8.75, Close: 9, Volume: 7_500, VWAP: 9.1,
		AverageTradeSize: 25, ATSProvenance: ATSRESTFloorVolumeOverTrades,
	})
	if err != nil {
		t.Fatal(err)
	}
	chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
	if err != nil {
		t.Fatal(err)
	}
	chunkDisposition := admitHydrationChunk(t, e, chunk)
	if chunkDisposition.Rows != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) {
		t.Fatalf("resolved discrepancy accounting=%+v", chunkDisposition)
	}
	terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	terminalDisposition := admitHydrationTerminal(t, e, terminal)
	if terminalDisposition.Code != DispositionHydrationTerminalApplied || e.observeHydration().Requests[0].coverage != hydrationCoverageCandidateComplete {
		t.Fatalf("resolved discrepancy terminal=%+v state=%+v", terminalDisposition, e.observeHydration())
	}
	now = token.end.Add(time.Second)
	initialFence, err := NewAggregateIngressFenceInput(terminalDisposition.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitAggregateIngressFence(context.Background(), initialFence)
	if admission != AdmissionAdmitted || completion == nil || awaitHydrationDisposition(t, completion).Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("initial fence admission=%s", admission)
	}

	frame := uint64(3)
	for second := 3; second < 330; second++ {
		window := start.Add(time.Duration(second) * time.Second)
		now = window.Add(time.Second)
		price := 10 + float64(second%17)/100
		input := liveAggregate(binding, "AAA", window, 1, frame)
		input.Values = AggregateValues{Open: price, High: price + 0.10, Low: price - 0.10, Close: price,
			Volume: 5_000, VWAP: price, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}
		if got := admitProductionAggregate(t, e, input); got.Code != DispositionAggregateInserted {
			t.Fatalf("live continuation second=%d disposition=%+v", second, got)
		}
		frame++
	}
	applySlice1CoverageFence(t, e, frame-1, 2, now)
	if admission, completion := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatalf("evaluation timer admission=%s", admission)
	}

	view := e.ObserveReplayDeterministic()
	canonical := view.Canonical[0]
	if len(canonical.Records) == 0 || canonical.LatestAuthoritySource != AggregateSourceLive ||
		canonical.Qualification.Status != "provisional" || view.Evaluation.Qualification.Provisional != 1 ||
		view.Evaluation.Mode != "qualified_current" || view.Evaluation.TotalPassers != 1 || len(view.Evaluation.Rows) != 1 {
		t.Fatalf("resolved discrepancy did not preserve qualification/ranking: canonical=%+v evaluation=%+v", canonical, view.Evaluation)
	}
	for name, field := range map[string]ReplayFieldView{
		"hod": canonical.Features.HODDrawdown, "session": canonical.Features.SessionRange,
		"rolling30": canonical.Features.Rolling30, "rolling60": canonical.Features.Rolling60,
		"activity": canonical.Features.Activity,
	} {
		if field.Status != "current" {
			t.Fatalf("%s unavailable after resolved discrepancy: %+v features=%+v", name, field, canonical.Features)
		}
	}
	state := aggregateState(t, e, "AAA")
	if state.historicalConflict != nil && state.historicalConflict.has(sessionSlot(e.state.binding, start)) ||
		!exactAggregateCoverage(state, e.state.binding, start, now) {
		t.Fatalf("resolved discrepancy did not retain exact session coverage: state=%+v", state)
	}
}

func TestD3PopulationTrustDangerousCounterexamples(t *testing.T) {
	binding := installedBindingForQualification(t, testBinding(t))
	conflictAt := binding.sessionStart
	markAt := conflictAt.Add(2 * time.Second)
	at := markAt.Add(3 * time.Second)

	tests := []struct {
		name        string
		coverage    aggregateCoverageConsequence
		authority   AggregateSource
		markAt      time.Time
		conflictAt  time.Time
		hasMark     bool
		setConflict bool
		incomplete  bool
		invalidAt   *time.Time
		want        populationTransitionDecision
	}{
		{name: "later live mark with exact post-mark coverage", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, conflictAt: conflictAt, hasMark: true, setConflict: true, want: populationTransitionTrustedByLaterLiveMark},
		{name: "later live mark after isolated missing prefix", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, hasMark: true, want: populationTransitionTrustedByLaterLiveMark},
		{name: "no later eligible mark", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, conflictAt: conflictAt, setConflict: true, want: populationTransitionNoLaterEligibleMark},
		{name: "latest mark is not live authority", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceHistorical, markAt: markAt, conflictAt: conflictAt, hasMark: true, setConflict: true, want: populationTransitionLatestMarkNotLiveAuthority},
		{name: "conflict at latest mark", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, conflictAt: markAt, hasMark: true, setConflict: true, want: populationTransitionConflictAtOrAfterMark},
		{name: "later structurally invalid evidence", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, conflictAt: conflictAt, hasMark: true, setConflict: true, invalidAt: timePointer(markAt.Add(time.Second)), want: populationTransitionInvalidAtOrAfterMark},
		{name: "post-mark coverage incomplete", coverage: coverageUnknownFailureOrFence, authority: AggregateSourceLive, markAt: markAt, conflictAt: conflictAt, hasMark: true, setConflict: true, incomplete: true, want: populationTransitionIncompletePostMarkCoverage},
		{name: "later provider failure or fence is outside diagnostic", coverage: coverageUnknownPostBootstrap, authority: AggregateSourceLive, markAt: markAt, conflictAt: conflictAt, hasMark: true, setConflict: true, want: populationTransitionNotApplicable},
	}

	diagnostic := populationTransitionDiagnostic{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			position := LivePosition{ConnectionEpoch: 1, FrameSequence: 2, ArrayIndex: 0}
			mark := canonicalAggregate{
				identity: aggregateIdentity{symbol: "AAA", start: tc.markAt.Unix()}, windowStart: tc.markAt, windowEnd: tc.markAt.Add(time.Second),
				values: AggregateValues{Close: 10}, authority: aggregateEvidence{source: tc.authority, live: position}, greatestLiveSupport: &position,
			}
			state := &symbolAggregateState{tail: map[int64]*canonicalAggregate{mark.identity.start: &mark}}
			if tc.setConflict {
				ensureHistoricalConflict(state).set(sessionSlot(binding, tc.conflictAt))
			}
			if !installExactCoverage(state, binding, tc.markAt, at, nil) {
				t.Fatal("post-mark coverage setup failed")
			}
			if tc.incomplete {
				state.provenAbsent.clear(sessionSlot(binding, tc.markAt.Add(time.Second)))
			}
			var invalid *invalidMarkEvidence
			if tc.invalidAt != nil {
				invalid = &invalidMarkEvidence{windowStart: *tc.invalidAt}
			}
			got := classifyPopulationTransition(binding, state, mark, tc.hasMark, at, tc.coverage, invalid)
			if got != tc.want {
				t.Fatalf("decision=%d want=%d coverage=%+v mark=%s conflict=%s invalid=%v exact=%t", got, tc.want, tc.coverage, tc.markAt, tc.conflictAt, tc.invalidAt, exactAggregateCoverage(state, binding, tc.markAt, at))
			}
			recordPopulationTransition(&diagnostic, got)
		})
	}
	if diagnostic != (populationTransitionDiagnostic{bootstrapUnknown: 7, trustedByLaterLiveMark: 2, noLaterEligibleMark: 1,
		latestMarkNotLiveAuthority: 1, conflictAtOrAfterMark: 1,
		invalidAtOrAfterMark: 1, incompletePostMarkCoverage: 1}) || !diagnostic.reconciles() {
		t.Fatalf("population-transition reason ledger=%+v", diagnostic)
	}
}

func timePointer(value time.Time) *time.Time { return &value }

func runD3OneSymbolPopulationTransition(t *testing.T, unequalOverlap bool) d3PopulationTransitionTrace {
	t.Helper()
	binding := hydrationPopulationBinding(t, []string{"AAA"})
	start := binding.SessionStart()
	now := start.Add(3 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	defer closeAndWait(t, e)

	// Use REST ATS provenance so equality is exact in the control. The
	// candidate changes only the hydration row values at this same identity.
	overlap := liveAggregate(binding, "AAA", start, 1, 2)
	overlap.Live.ArrayIndex = 1
	overlap.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
	if got := admitProductionAggregate(t, e, overlap); got.Code != DispositionAggregateInserted {
		t.Fatalf("initial live overlap=%+v", got)
	}
	rowClose := 10.0
	if unequalOverlap {
		rowClose = 9
	}
	row := hydrationRow(t, "AAA", start, rowClose)
	chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
	if err != nil {
		t.Fatal(err)
	}
	chunkDisposition := admitHydrationChunk(t, e, chunk)
	terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	terminalDisposition := admitHydrationTerminal(t, e, terminal)
	if terminalDisposition.Code != DispositionHydrationTerminalApplied {
		t.Fatalf("hydration terminal=%+v", terminalDisposition)
	}
	requestCoverage := e.observeHydration().Requests[0].coverage

	initialFenceAt := token.end.Add(time.Second)
	now = initialFenceAt
	initialFence, err := NewAggregateIngressFenceInput(terminalDisposition.FenceCommand, AggregateIngressFenceComplete, 2, 1, initialFenceAt)
	if err != nil {
		t.Fatal(err)
	}
	initialAdmission, initialCompletion := e.AdmitAggregateIngressFence(context.Background(), initialFence)
	if initialAdmission != AdmissionAdmitted || initialCompletion == nil {
		t.Fatalf("bootstrap fence admission=%s", initialAdmission)
	}
	if got := awaitHydrationDisposition(t, initialCompletion); got.Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("bootstrap fence=%+v", got)
	}
	index := e.state.binding.index["AAA"]
	e.mu.Lock()
	bootstrapCoverage, bootstrapCoverageSet := e.state.aggregateEvaluator.coverage[index]
	e.mu.Unlock()

	// A strictly later live identity is independently trustworthy and becomes
	// the canonical latest mark in both cases.
	laterWindow := initialFenceAt
	now = laterWindow.Add(time.Second)
	later := liveAggregate(binding, "AAA", laterWindow, 1, 3)
	laterDisposition := admitProductionAggregate(t, e, later)

	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	liveFence, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, 3, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	liveAdmission, liveCompletion := e.AdmitLiveCoverageFence(context.Background(), liveFence)
	if liveAdmission != AdmissionAdmitted || liveCompletion == nil {
		t.Fatalf("ordinary coverage fence admission=%s", liveAdmission)
	}
	waitContext, cancelWait := context.WithTimeout(context.Background(), time.Second)
	defer cancelWait()
	var liveDisposition LiveCoverageFenceDisposition
	select {
	case liveDisposition = <-liveCompletion:
	case <-waitContext.Done():
		t.Fatalf("ordinary coverage fence completion: %v", waitContext.Err())
	}
	if liveDisposition.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("ordinary coverage fence=%+v", liveDisposition)
	}
	if got, err := command.Wait(waitContext); err != nil || got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("ordinary coverage command=%+v err=%v", got, err)
	}
	if admission, completion := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || completion == nil || awaitTimerDisposition(t, completion).Code != DispositionTimerApplied {
		t.Fatalf("evaluation timer admission=%s", admission)
	}

	e.mu.Lock()
	ordinaryCoverage, ordinaryCoverageSet := e.state.aggregateEvaluator.coverage[index]
	state := e.state.binding.symbols[index].aggregates
	conflictAtOldIdentity := state.historicalConflict != nil && state.historicalConflict.has(sessionSlot(e.state.binding, start))
	latest, latestSet := latestMarkBefore(state, now)
	e.mu.Unlock()
	latestWindow := time.Time{}
	if latestSet {
		latestWindow = latest.windowStart
	}

	view := e.ObserveReplayDeterministic()
	canonical := view.Canonical[0]
	return d3PopulationTransitionTrace{
		rowAccounting: chunkDisposition.Rows, requestCoverage: requestCoverage,
		bootstrapCoverage: bootstrapCoverage, bootstrapCoverageSet: bootstrapCoverageSet,
		laterLiveDisposition: laterDisposition, ordinaryCoverage: ordinaryCoverage, ordinaryCoverageSet: ordinaryCoverageSet,
		conflictAtOldIdentity: conflictAtOldIdentity, latestMarkWindow: latestWindow, latestMarkAuthority: canonical.LatestAuthoritySource,
		features: canonical.Features, qualification: canonical.Qualification, evaluation: view.Evaluation,
	}
}
