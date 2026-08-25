package engine

import (
	"context"
	"testing"
	"time"
)

func TestC8LiveCoverageFenceAuthorityAndAdvance(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleLive)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = 1
	e.state.hydration.fenceThrough = 1
	e.state.hydration.fenceMarkerOrdinal = 1
	e.state.hydration.supportedThrough = immutableTime(now)
	for index := range e.state.binding.symbols {
		state := ensureAggregateState(&e.state.binding.symbols[index])
		if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, now, nil) {
			e.mu.Unlock()
			t.Fatal("startup exact coverage setup failed")
		}
	}
	e.mu.Unlock()
	priorSupported := now
	now = now.Add(2 * time.Second)
	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, 1, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("coverage admission=%s", admission)
	}
	if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("coverage disposition=%+v", got)
	}
	if got, err := command.Wait(context.Background()); err != nil || got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("command completion=%+v err=%v", got, err)
	}
	if admission, timer := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || timer == nil || (<-timer).Code != DispositionTimerApplied {
		t.Fatal("coverage timer was not applied")
	}
	if view := e.ObserveOperational(); view.Watermark == nil || *view.Watermark != now {
		t.Fatalf("coverage did not support exact T: %+v", view)
	}
	e.mu.Lock()
	failure := ""
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		if !exactAggregateCoverage(state, e.state.binding, priorSupported, now) || e.state.aggregateEvaluator.coverage[index] != coverageNoPrintThroughT {
			failure = e.state.binding.symbols[index].symbol
			break
		}
	}
	e.mu.Unlock()
	if failure != "" {
		t.Fatalf("symbol %s ordinary interval was not exact no-print", failure)
	}

	// A retained command cannot be replayed, and a forged command cannot claim
	// completion or replace the current engine-issued authority.
	_, replayCompletion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if got := <-replayCompletion; got.Code != DispositionLiveCoverageFenceFenced {
		t.Fatalf("replayed command=%+v", got)
	}
	foreign := command
	foreign.token++
	foreign.done = make(chan LiveCoverageFenceDisposition, 1)
	foreignFact, _ := NewLiveCoverageFenceInput(foreign, LiveCoverageFenceComplete, 1, 3, now)
	_, foreignCompletion := e.AdmitLiveCoverageFence(context.Background(), foreignFact)
	if got := <-foreignCompletion; got.Code != DispositionLiveCoverageFenceFenced {
		t.Fatalf("foreign command=%+v", got)
	}
	select {
	case got := <-foreign.done:
		t.Fatalf("foreign command received engine authority: %+v", got)
	case <-time.After(10 * time.Millisecond):
	}

	cancelCommand, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	_, canceled := e.AdmitLiveCoverageFenceCancellation(context.Background(), cancelCommand)
	if got := <-canceled; got.Code != DispositionLiveCoverageFenceRejected {
		t.Fatalf("cancellation=%+v", got)
	}
	if _, err := cancelCommand.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	finalCommand, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatalf("cancellation left command stuck: %v", err)
	}
	_, finalCancellation := e.AdmitLiveCoverageFenceCancellation(context.Background(), finalCommand)
	<-finalCancellation
}

func TestSlice1LiveCoverageFenceClosesNewUncertaintyOrigin(t *testing.T) {
	binding := testBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleLive)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = 1
	e.state.hydration.fenceThrough = 1
	e.state.hydration.fenceMarkerOrdinal = 1
	e.state.hydration.supportedThrough = immutableTime(now)
	state := ensureAggregateState(&e.state.binding.symbols[e.state.binding.index["AAA"]])
	if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, now, nil) {
		e.mu.Unlock()
		t.Fatal("startup exact coverage setup failed")
	}
	conflictAt := now.Add(time.Second)
	ensureHistoricalConflict(state).set(sessionSlot(e.state.binding, conflictAt))
	e.mu.Unlock()

	now = now.Add(2 * time.Second)
	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, 1, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	_, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("coverage disposition=%+v", got)
	}
	e.mu.Lock()
	got := e.state.aggregateEvaluator.coverage[e.state.binding.index["AAA"]]
	conflictPreserved := state.historicalConflict.has(sessionSlot(e.state.binding, conflictAt))
	e.mu.Unlock()
	if got != coverageUnknownPostBootstrap || !conflictPreserved {
		t.Fatalf("ordinary unresolved coverage=%+v conflict_preserved=%t", got, conflictPreserved)
	}
}

func TestSlice1LiveCoverageFenceRetainsInvalidMarkEvidence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		olderMark bool
	}{
		{name: "no older mark"},
		{name: "older valid mark", olderMark: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := testBinding(t)
			t0 := binding.SessionStart().Add(10 * time.Second)
			now := t0
			e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleLive)
			defer closeAndWait(t, e)
			index := e.state.binding.index["AAA"]

			e.mu.Lock()
			e.state.hydration.fenceReconciled = true
			e.state.hydration.fenceEpoch = 1
			e.state.hydration.fenceThrough = 1
			e.state.hydration.fenceMarkerOrdinal = 1
			e.state.hydration.supportedThrough = immutableTime(t0)
			state := ensureAggregateState(&e.state.binding.symbols[index])
			if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, t0, nil) {
				e.mu.Unlock()
				t.Fatal("startup exact coverage setup failed")
			}
			if e.state.aggregateEvaluator.coverage == nil {
				e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
			}
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
			e.mu.Unlock()

			frame := uint64(2)
			var olderWindow time.Time
			if tc.olderMark {
				olderWindow = t0.Add(-2 * time.Second)
				if got := admitProductionAggregate(t, e, liveAggregate(binding, "AAA", olderWindow, 1, frame)); got.Code != DispositionAggregateInserted {
					t.Fatalf("older aggregate=%+v", got)
				}
				e.mu.Lock()
				e.state.hydration.fenceThrough = frame
				e.mu.Unlock()
				frame++
			}
			admission, initialTimer := e.AdmitTimer(context.Background())
			if admission != AdmissionAdmitted || awaitTimerDisposition(t, initialTimer).Code != DispositionTimerApplied {
				t.Fatal("initial T0 timer was not applied")
			}
			initial := e.ObserveSnapshot()
			if initial.Publication.Watermark == nil || *initial.Publication.Watermark != t0 {
				t.Fatalf("initial watermark=%+v", initial.Publication)
			}

			invalidWindow := t0
			now = invalidWindow.Add(time.Second)
			invalid := liveAggregate(binding, "AAA", invalidWindow, 1, frame)
			invalid.Values.High = 0
			got := admitProductionAggregate(t, e, invalid)
			if got.Code != DispositionAggregateRejected || got.Reason != ReasonStructural {
				t.Fatalf("structurally invalid aggregate=%+v", got)
			}
			e.mu.Lock()
			retainedBefore, retainedBeforeOK := e.state.aggregateEvaluator.invalidMarks[index]
			absentBefore := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, invalidWindow))
			e.mu.Unlock()
			if !retainedBeforeOK || retainedBefore.windowStart != invalidWindow || absentBefore {
				t.Fatalf("invalid evidence before fence=%+v present=%t absent=%t", retainedBefore, retainedBeforeOK, absentBefore)
			}

			now = t0.Add(2 * time.Second)
			command, err := e.IssueLiveCoverageFence()
			if err != nil {
				t.Fatal(err)
			}
			fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, frame, 2, now)
			if err != nil {
				t.Fatal(err)
			}
			admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
			if admission != AdmissionAdmitted || completion == nil {
				t.Fatalf("coverage admission=%s", admission)
			}
			if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
				t.Fatalf("coverage disposition=%+v", got)
			}

			e.mu.Lock()
			retainedAfterFence, retainedAfterFenceOK := e.state.aggregateEvaluator.invalidMarks[index]
			absentAfterFence := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, invalidWindow))
			exactAfterFence := exactAggregateCoverage(state, e.state.binding, t0, now)
			coverageAfterFence := e.state.aggregateEvaluator.coverage[index]
			e.mu.Unlock()
			if !retainedAfterFenceOK || retainedAfterFence.windowStart != invalidWindow || absentAfterFence || exactAfterFence || coverageAfterFence != coverageUnknownPostBootstrap {
				t.Fatalf("post-fence invalid=%+v present=%t absent=%t exact=%t coverage=%+v", retainedAfterFence, retainedAfterFenceOK, absentAfterFence, exactAfterFence, coverageAfterFence)
			}

			admission, timer := e.AdmitMaintenanceTimer(context.Background())
			if admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
				t.Fatal("post-fence timer was not applied")
			}
			final := e.ObserveSnapshot()
			operational := e.ObserveOperational()
			evaluation := final.Publication.AggregateEvaluation
			if final.Publication.Watermark == nil || *final.Publication.Watermark != now || final.Publication.LastDisposition != DispositionLiveCoverageFenceApplied ||
				final.Publication.Suppression != "" || final.Publication.Lifecycle != "live" {
				t.Fatalf("incoherent post-fence publication=%+v", final.Publication)
			}
			if evaluation.Population.UniverseTotal != 3 || evaluation.Population.ValidPriorClose != 1 || evaluation.Population.InvalidOrMissingPriorClose != 2 ||
				evaluation.Population.TrustedRankableMark != 0 || evaluation.Population.NoPrintThroughT != 0 || evaluation.Population.InvalidMark != 0 ||
				evaluation.Population.UnknownDueFailureOrFence != 1 || evaluation.Population.CoveredPopulation != 2 || evaluation.Population.UnresolvedPopulation != 1 ||
				evaluation.Uncertainty.BootstrapOrigin != 0 || evaluation.Uncertainty.PostBootstrapGap != 1 || evaluation.Uncertainty.LocalInvalid != 0 ||
				evaluation.Qualification.NotYetPassed+evaluation.Qualification.Provisional+evaluation.Qualification.Finalized+evaluation.Qualification.Unresolved != 0 ||
				evaluation.Mode != "unavailable" || len(evaluation.Rows) != 0 {
				t.Fatalf("incoherent post-fence evaluation=%+v", evaluation)
			}
			if !operational.Admissions.Reconciles(operational.QueueOccupancy) || !operational.Transitions.Reconciles() || !operational.Aggregates.Reconciles() ||
				operational.Aggregates.Rejected != 1 || operational.Suppression != "" {
				t.Fatalf("post-fence operational accounting=%+v", operational)
			}
			e.mu.Lock()
			retainedAfterTimer, retainedAfterTimerOK := e.state.aggregateEvaluator.invalidMarks[index]
			e.mu.Unlock()
			if !retainedAfterTimerOK || retainedAfterTimer.windowStart != invalidWindow {
				t.Fatalf("timer lost invalid evidence=%+v present=%t", retainedAfterTimer, retainedAfterTimerOK)
			}
			if tc.olderMark {
				state := aggregateState(t, e, "AAA")
				if state.latest == nil || state.latest.record.windowStart != olderWindow || evaluation.Population.TrustedRankableMark != 0 {
					t.Fatalf("older mark crossed newer invalid identity: state=%+v evaluation=%+v", state, evaluation)
				}
			}
		})
	}
}

func TestSlice1LiveCoverageHalfOpenBoundaryMatrix(t *testing.T) {
	for _, olderMark := range []bool{false, true} {
		for _, offset := range []time.Duration{0, time.Second, 2 * time.Second} {
			name := "no_older_mark"
			if olderMark {
				name = "older_accepted_mark"
			}
			t.Run(name+"/invalid_offset_"+offset.String(), func(t *testing.T) {
				binding := testBinding(t)
				t0 := binding.SessionStart().Add(10 * time.Second)
				t1 := t0.Add(2 * time.Second)
				delay := time.Second
				now := t0.Add(delay)
				e := acknowledgedHydrationEngine(t, binding, &now, t0, lifecycleLive)
				defer closeAndWait(t, e)
				e.mu.Lock()
				e.delay = delay
				e.state.hydration.fenceReconciled = true
				e.state.hydration.fenceEpoch = 1
				e.state.hydration.fenceThrough = 1
				e.state.hydration.fenceMarkerOrdinal = 1
				e.state.hydration.supportedThrough = immutableTime(t0)
				index := e.state.binding.index["AAA"]
				state := ensureAggregateState(&e.state.binding.symbols[index])
				if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, t0, nil) {
					e.mu.Unlock()
					t.Fatal("startup exact coverage setup failed")
				}
				if e.state.aggregateEvaluator.coverage == nil {
					e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
				}
				e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
				e.mu.Unlock()

				frame := uint64(2)
				olderWindow := t0.Add(-2 * time.Second)
				if olderMark {
					if got := admitProductionAggregate(t, e, liveAggregate(binding, "AAA", olderWindow, 1, frame)); got.Code != DispositionAggregateInserted {
						t.Fatalf("older aggregate=%+v", got)
					}
					e.mu.Lock()
					delete(e.state.aggregateEvaluator.coverage, index)
					e.state.hydration.fenceThrough = frame
					e.mu.Unlock()
					frame++
				}

				invalidWindow := t0.Add(offset)
				now = invalidWindow.Add(time.Second)
				invalid := liveAggregate(binding, "AAA", invalidWindow, 1, frame)
				invalid.Values.High = 0
				if got := admitProductionAggregate(t, e, invalid); got.Code != DispositionAggregateRejected || got.Reason != ReasonStructural {
					t.Fatalf("structurally invalid aggregate=%+v", got)
				}

				// The nonzero evaluation delay makes the completed identity at T1
				// causally precede a fence captured at T1+delay whose target is T1.
				now = t1.Add(delay)
				applySlice1CoverageFence(t, e, frame, 2, now)
				if admission, timer := e.AdmitMaintenanceTimer(context.Background()); admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
					t.Fatal("T1 timer was not applied")
				}

				view := e.ObserveSnapshot()
				operational := e.ObserveOperational()
				if view.Publication.Watermark == nil || *view.Publication.Watermark != t1 || view.Publication.Suppression != "" ||
					view.Publication.LastDisposition != DispositionLiveCoverageFenceApplied || operational.Suppression != "" {
					t.Fatalf("T1 publication was not coherent: publication=%+v operational=%+v", view.Publication, operational)
				}
				if !operational.Admissions.Reconciles(operational.QueueOccupancy) || !operational.Transitions.Reconciles() || !operational.Aggregates.Reconciles() {
					t.Fatalf("T1 operational accounting=%+v", operational)
				}
				e.mu.Lock()
				retained, retainedOK := e.state.aggregateEvaluator.invalidMarks[index]
				absentAtInvalid := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, invalidWindow))
				exactAtT1 := exactAggregateCoverage(state, e.state.binding, t0, t1)
				consequence, hasConsequence := e.state.aggregateEvaluator.coverage[index]
				e.mu.Unlock()
				if !retainedOK || retained.windowStart != invalidWindow || absentAtInvalid {
					t.Fatalf("invalid identity invariant retained=%+v present=%t absent=%t", retained, retainedOK, absentAtInvalid)
				}

				if offset == 2*time.Second {
					if !exactAtT1 {
						t.Fatal("right-open endpoint incorrectly made [T0,T1) non-exact")
					}
					assertSlice1EndpointEvaluation(t, view.Publication.AggregateEvaluation, olderMark)
					if olderMark && hasConsequence {
						t.Fatalf("older accepted mark retained coverage consequence=%+v", consequence)
					}
					if !olderMark && (!hasConsequence || consequence != coverageNoPrintThroughT) {
						t.Fatalf("endpoint no-print consequence=%+v present=%t", consequence, hasConsequence)
					}

					nextTarget := t1.Add(time.Second)
					now = nextTarget.Add(delay)
					applySlice1CoverageFence(t, e, frame, 3, now)
					if admission, timer := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
						t.Fatal("post-T1 timer was not applied")
					}
					next := e.ObserveSnapshot()
					if next.Publication.Watermark == nil || *next.Publication.Watermark != nextTarget || next.Publication.Suppression != "" {
						t.Fatalf("post-T1 publication=%+v", next.Publication)
					}
					assertSlice1ApplicableInvalidEvaluation(t, next.Publication.AggregateEvaluation)
					e.mu.Lock()
					nextAbsent := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, invalidWindow))
					nextExact := exactAggregateCoverage(state, e.state.binding, t1, nextTarget)
					nextConsequence := e.state.aggregateEvaluator.coverage[index]
					e.mu.Unlock()
					if nextAbsent || nextExact || nextConsequence != coverageUnknownPostBootstrap {
						t.Fatalf("post-T1 invariant absent=%t exact=%t consequence=%+v", nextAbsent, nextExact, nextConsequence)
					}
					return
				}

				if exactAtT1 || !hasConsequence || consequence != coverageUnknownPostBootstrap {
					t.Fatalf("applicable invalid coverage exact=%t consequence=%+v present=%t", exactAtT1, consequence, hasConsequence)
				}
				assertSlice1ApplicableInvalidEvaluation(t, view.Publication.AggregateEvaluation)
				if olderMark {
					state := aggregateState(t, e, "AAA")
					if state.latest == nil || state.latest.record.windowStart != olderWindow {
						t.Fatalf("older canonical mark was not preserved: %+v", state)
					}
				}
			})
		}
	}
}

func TestSlice1LateInvalidIdentityImmediatelyRevokesAbsence(t *testing.T) {
	binding := testBinding(t)
	t0 := binding.SessionStart().Add(10 * time.Second)
	t1 := t0.Add(2 * time.Second)
	delay := time.Second
	now := t1.Add(delay)
	e := acknowledgedHydrationEngine(t, binding, &now, t0, lifecycleLive)
	defer closeAndWait(t, e)
	e.mu.Lock()
	e.delay = delay
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = 1
	e.state.hydration.fenceThrough = 1
	e.state.hydration.fenceMarkerOrdinal = 1
	e.state.hydration.supportedThrough = immutableTime(t1)
	index := e.state.binding.index["AAA"]
	state := ensureAggregateState(&e.state.binding.symbols[index])
	if !installExactCoverage(state, e.state.binding, e.state.binding.sessionStart, t1, nil) {
		e.mu.Unlock()
		t.Fatal("exact coverage setup failed")
	}
	if e.state.aggregateEvaluator.coverage == nil {
		e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
	}
	e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
	e.mu.Unlock()

	endpointInvalid := liveAggregate(binding, "AAA", t1, 1, 2)
	endpointInvalid.Values.High = 0
	if got := admitProductionAggregate(t, e, endpointInvalid); got.Code != DispositionAggregateRejected || got.Reason != ReasonStructural {
		t.Fatalf("endpoint invalid aggregate=%+v", got)
	}
	e.mu.Lock()
	endpointCoverage := e.state.aggregateEvaluator.coverage[index]
	retainedEndpoint := e.state.aggregateEvaluator.invalidMarks[index]
	e.mu.Unlock()
	if endpointCoverage != coverageNoPrintThroughT || retainedEndpoint.windowStart != t1 {
		t.Fatalf("right-open endpoint affected T1 early: coverage=%+v retained=%+v", endpointCoverage, retainedEndpoint)
	}

	lateInvalid := liveAggregate(binding, "AAA", t0, 1, 3)
	lateInvalid.Values.High = 0
	if got := admitProductionAggregate(t, e, lateInvalid); got.Code != DispositionAggregateRejected || got.Reason != ReasonStructural {
		t.Fatalf("late invalid aggregate=%+v", got)
	}
	e.mu.Lock()
	absent := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, t0))
	consequence := e.state.aggregateEvaluator.coverage[index]
	retained := e.state.aggregateEvaluator.invalidMarks[index]
	e.mu.Unlock()
	if absent || consequence != coverageUnknownPostBootstrap || retained.windowStart != t1 {
		t.Fatalf("late invalid did not revoke absence while preserving bounded newest evidence: absent=%t consequence=%+v retained=%+v", absent, consequence, retained)
	}

	applySlice1CoverageFence(t, e, 3, 2, now)
	if admission, timer := e.AdmitTimer(context.Background()); admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
		t.Fatal("late-invalid timer was not applied")
	}
	view := e.ObserveSnapshot()
	if view.Publication.Watermark == nil || *view.Publication.Watermark != t1 || view.Publication.Suppression != "" {
		t.Fatalf("late-invalid publication=%+v", view.Publication)
	}
	assertSlice1ApplicableInvalidEvaluation(t, view.Publication.AggregateEvaluation)
}

func applySlice1CoverageFence(t *testing.T, e *Engine, throughFrame, marker uint64, capturedAt time.Time) {
	t.Helper()
	command, err := e.IssueLiveCoverageFence()
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewLiveCoverageFenceInput(command, LiveCoverageFenceComplete, throughFrame, marker, capturedAt)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitLiveCoverageFence(context.Background(), fact)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("coverage admission=%s", admission)
	}
	if got := <-completion; got.Code != DispositionLiveCoverageFenceApplied {
		t.Fatalf("coverage disposition=%+v", got)
	}
}

func assertSlice1EndpointEvaluation(t *testing.T, evaluation EvaluationView, olderMark bool) {
	t.Helper()
	p := evaluation.Population
	if p.UniverseTotal != 3 || p.ValidPriorClose != 1 || p.InvalidOrMissingPriorClose != 2 || p.UnknownDueFailureOrFence != 0 ||
		p.CoveredPopulation != 3 || p.UnresolvedPopulation != 0 || evaluation.Uncertainty != (UncertaintyView{}) || evaluation.Mode != "qualified_current" {
		t.Fatalf("endpoint evaluation=%+v", evaluation)
	}
	if olderMark {
		if p.TrustedRankableMark != 1 || p.NoPrintThroughT != 0 || evaluation.Qualification.NotYetPassed != 1 {
			t.Fatalf("endpoint older-mark accounting=%+v", evaluation)
		}
	} else if p.TrustedRankableMark != 0 || p.NoPrintThroughT != 1 || evaluation.Qualification != (QualificationAccountingView{}) {
		t.Fatalf("endpoint no-mark accounting=%+v", evaluation)
	}
	if p.ValidPriorClose != p.TrustedRankableMark+p.TrustedBelowPriceMark+p.NoPrintThroughT+p.InvalidMark+p.UnknownDueFailureOrFence {
		t.Fatalf("endpoint population does not reconcile: %+v", p)
	}
}

func assertSlice1ApplicableInvalidEvaluation(t *testing.T, evaluation EvaluationView) {
	t.Helper()
	p := evaluation.Population
	if p.UniverseTotal != 3 || p.ValidPriorClose != 1 || p.InvalidOrMissingPriorClose != 2 || p.TrustedRankableMark != 0 ||
		p.NoPrintThroughT != 0 || p.InvalidMark != 0 || p.UnknownDueFailureOrFence != 1 || p.CoveredPopulation != 2 || p.UnresolvedPopulation != 1 ||
		evaluation.Uncertainty.BootstrapOrigin != 0 || evaluation.Uncertainty.PostBootstrapGap != 1 || evaluation.Uncertainty.LocalInvalid != 0 ||
		evaluation.Qualification != (QualificationAccountingView{}) || evaluation.Mode != "unavailable" || len(evaluation.Rows) != 0 {
		t.Fatalf("applicable-invalid evaluation=%+v", evaluation)
	}
	if p.ValidPriorClose != p.TrustedRankableMark+p.TrustedBelowPriceMark+p.NoPrintThroughT+p.InvalidMark+p.UnknownDueFailureOrFence {
		t.Fatalf("applicable-invalid population does not reconcile: %+v", p)
	}
}
