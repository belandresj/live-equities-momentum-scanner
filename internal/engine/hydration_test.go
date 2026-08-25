package engine

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// TestC6PLAN01DeterministicModeIntervalPopulation is P-C6-PLAN. It proves the
// engine, not rank or a worker, derives each exact interval and registers the
// complete valid-prior population atomically. Dangerous counterexamples are
// R=S, T0=R, a 16-minute recovery overlap, invalid/missing priors, insufficient
// bounds, overflow, and compacted presence without retained authority. This
// proof does not establish HTTP behavior or lifecycle exit.
func TestC6PLAN01DeterministicModeIntervalPopulation(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("fresh uses clamped ceil acknowledgement and only complete valid priors", func(t *testing.T) {
		now := start.Add(3250 * time.Millisecond)
		e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
		defer closeAndWait(t, e)
		got := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if got.Code != DispositionHydrationPlanApplied || got.Plan.Start() != start || got.Plan.End() != start.Add(4*time.Second) || got.Plan.Generation() != 1 {
			t.Fatalf("fresh plan = %+v [%s,%s)", got, got.Plan.Start(), got.Plan.End())
		}
		requests := got.Plan.Requests()
		if len(requests) != 1 || requests[0].Symbol() != "AAA" || requests[0].RequestID() != 1 || requests[0].ConnectionEpoch() != 1 {
			t.Fatalf("valid-prior request population = %+v", requests)
		}
		if !slices.IsSortedFunc(requests, func(a, b HydrationRequestToken) int {
			if a.Symbol() < b.Symbol() {
				return -1
			}
			if a.Symbol() > b.Symbol() {
				return 1
			}
			return 0
		}) {
			t.Fatal("plan requests are not canonically sorted")
		}
		mutated := got.Plan.Requests()
		mutated[0] = HydrationRequestToken{}
		if got.Plan.Requests()[0].Symbol() != "AAA" {
			t.Fatal("plan request slice aliases caller memory")
		}
	})

	t.Run("empty interval advances a generation with no requests", func(t *testing.T) {
		now := start.Add(-time.Second)
		e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleAwaitingSession)
		now = start
		admission, timer := e.AdmitTimer(context.Background())
		if admission != AdmissionAdmitted || awaitTimerDisposition(t, timer).Code != DispositionTimerApplied {
			t.Fatal("session-start timer failed")
		}
		got := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if got.Code != DispositionHydrationPlanApplied || !got.Plan.Empty() || got.Plan.Generation() != 1 || got.Plan.Start() != start || got.Plan.End() != start || len(got.Plan.Requests()) != 0 || got.Accounting != (HydrationAccounting{}) {
			t.Fatalf("R=S plan = %+v plan=%+v", got, got.Plan)
		}
		closeAndWait(t, e)
	})

	for _, tc := range []struct {
		name      string
		purpose   HydrationPurpose
		lifecycle lifecycle
		setStart  func(*Engine, time.Time)
	}{
		{"gap starts at exact supported T", HydrationGapRecovery, lifecycleRecovering, func(e *Engine, at time.Time) { e.state.hydration.supportedT = &at }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := start.Add(10 * time.Second)
			e := acknowledgedHydrationEngine(t, binding, &now, now, tc.lifecycle)
			e.mu.Lock()
			intervalStart := start.Add(10 * time.Second)
			if tc.purpose == HydrationGapRecovery {
				intervalStart = start.Add(7 * time.Second)
			}
			tc.setStart(e, intervalStart)
			e.mu.Unlock()
			got := admitHydrationPlan(t, e, tc.purpose, 1, generousHydrationBudgets())
			if got.Code != DispositionHydrationPlanApplied || got.Plan.Start() != intervalStart || got.Plan.End() != start.Add(10*time.Second) {
				t.Fatalf("mode plan = %+v [%s,%s)", got, got.Plan.Start(), got.Plan.End())
			}
			closeAndWait(t, e)
		})
	}

	t.Run("invalid context rejects atomically", func(t *testing.T) {
		now := start.Add(5 * time.Second)
		e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
		bad := generousHydrationBudgets()
		bad.MaximumNormalizedRecords = 1
		got := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, bad)
		if got.Code != DispositionHydrationRejected || got.Reason != ReasonHydrationBounds || e.observeHydration().Active {
			t.Fatalf("bounded atomic rejection = %+v state=%+v", got, e.observeHydration())
		}
		e.mu.Lock()
		e.state.hydration.lastGeneration = math.MaxUint64
		e.mu.Unlock()
		got = admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if got.Code != DispositionHydrationRejected || got.Reason != ReasonHydrationBounds || e.observeHydration().Active {
			t.Fatalf("overflow atomic rejection = %+v", got)
		}
		closeAndWait(t, e)

		e = acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
		e.mu.Lock()
		state := ensureAggregateState(&e.state.binding.symbols[0])
		ensurePresence(state).set(sessionSlot(e.state.binding, start))
		e.mu.Unlock()
		got = admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		if got.Code != DispositionHydrationRejected || got.Reason != ReasonHydrationCompactedPresence || e.observeHydration().Active {
			t.Fatalf("compacted-presence rejection = %+v", got)
		}
		closeAndWait(t, e)
	})
}

// TestLBRHydrationDoesNotPinCanonicalTail preserves the old population-scan
// regression while enforcing LBR-A1's 961-record bound. Current-token old
// historical fills now fold directly, so active hydration does not need to pin
// raw live aggregates until the ingress fence.
func TestLBRHydrationDoesNotPinCanonicalTail(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(20 * time.Minute)
	e := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
	plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	requests := plan.Plan.Requests()
	if plan.Code != DispositionHydrationPlanApplied || len(requests) != 1 {
		t.Fatalf("plan = %+v requests=%d", plan, len(requests))
	}
	e.mu.Lock()
	requestIndex, indexed := e.state.hydration.generation.requestIndexBySymbol("AAA")
	e.mu.Unlock()
	if !indexed || requestIndex != 0 {
		t.Fatalf("symbol request index = %d/%t", requestIndex, indexed)
	}

	oldLive := liveAggregate(binding, "AAA", start, 1, 2)
	e.mu.Lock()
	symbol := &e.state.binding.symbols[e.state.binding.index["AAA"]]
	if code, reason := e.installAggregateLocked(symbol, freezeAggregateInput(oldLive), start.Add(time.Second), false); code != DispositionAggregateInserted {
		e.mu.Unlock()
		t.Fatalf("old live aggregate = %s/%s", code, reason)
	}
	e.state.greatestIngressPosition = oldLive.Live
	state := symbol.aggregates
	e.compactSymbolLocked(state, e.state.binding, "AAA", now)
	_, retainedWhileHydrating := state.tail[start.Unix()]
	presentWhileHydrating := state.presence != nil && state.presence.has(sessionSlot(e.state.binding, start))
	e.mu.Unlock()
	if retainedWhileHydrating || !presentWhileHydrating {
		t.Fatalf("active hydration retained old mutable identity=%t present=%t", retainedWhileHydrating, presentWhileHydrating)
	}

	terminal, err := NewHydrationTerminalInput(requests[0], requests[0].ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	terminalResult := admitHydrationTerminal(t, e, terminal)
	now = now.Add(5 * time.Second)
	fence, err := NewAggregateIngressFenceInput(terminalResult.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fence)
	if got := awaitHydrationDisposition(t, fenceCompletion); got.Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("fence = %+v", got)
	}
	e.mu.Lock()
	_, retainedAfterFence := state.tail[start.Unix()]
	presentAfterFence := state.presence != nil && state.presence.has(sessionSlot(e.state.binding, start))
	e.mu.Unlock()
	if retainedAfterFence || !presentAfterFence {
		t.Fatalf("fence compaction retained=%t present=%t", retainedAfterFence, presentAfterFence)
	}
	closeAndWait(t, e)
}

// TestC6START01FreshCheckpointLifecycleTrace is P-C6-START. It proves that
// terminal provider work alone cannot leave hydrating, while the exact current
// ingress fence does, and that epoch loss cancels/deactivates the old generation.
func TestC6START01FreshCheckpointLifecycleTrace(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(2 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	terminalResult := admitHydrationTerminal(t, e, terminal)
	if terminalResult.FenceCommand.CommandToken() == 0 || e.state.lifecycle != lifecycleHydrating {
		t.Fatalf("terminal improperly finalized startup: disposition=%+v lifecycle=%s", terminalResult, e.state.lifecycle)
	}
	if pending := e.observePublication(); pending.currentMarketClaim || len(pending.aggregateEvaluation.rows) != 0 || pending.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("successful empty appeared current before exact fence/evaluation: %+v", pending)
	}
	now = start.Add(3 * time.Second)
	staleCommand := terminalResult.FenceCommand
	staleCommand.commandToken++
	staleFact, err := NewAggregateIngressFenceInput(staleCommand, AggregateIngressFenceComplete, 1, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, staleCompletion := e.AdmitAggregateIngressFence(context.Background(), staleFact)
	if stale := awaitHydrationDisposition(t, staleCompletion); stale.Code != DispositionAggregateIngressFenceFenced || e.state.lifecycle != lifecycleHydrating {
		t.Fatalf("stale fence reached success = %+v lifecycle=%s", stale, e.state.lifecycle)
	}
	fact, err := NewAggregateIngressFenceInput(terminalResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	admission, completion := e.AdmitAggregateIngressFence(context.Background(), fact)
	got := awaitHydrationDisposition(t, completion)
	if admission != AdmissionAdmitted || got.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive || e.state.committedT == nil || *e.state.committedT != now {
		t.Fatalf("fence finalization = %s %+v lifecycle=%s committed=%v", admission, got, e.state.lifecycle, e.state.committedT)
	}
	if resolved := e.observePublication(); !resolved.currentMarketClaim || resolved.aggregateEvaluation.mode != rankingQualifiedCurrent ||
		len(resolved.aggregateEvaluation.rows) != 0 || resolved.aggregateEvaluation.enrichedRows != 0 || resolved.aggregateEvaluation.population.noPrintThroughT != 1 {
		t.Fatalf("successful empty did not resolve only at exact fence/evaluation: %+v", resolved)
	}
	closeAndWait(t, e)

	now = start.Add(2 * time.Second)
	incomplete, incompleteToken := plannedHydrationEngine(t, binding, &now)
	beforeMalformed := incomplete.observePublication()
	if _, malformedErr := NewHydrationTerminalInput(incompleteToken, incompleteToken.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, -1, 1, 2); malformedErr == nil {
		t.Fatal("malformed/incomplete success terminal was constructible")
	}
	if afterMalformed := incomplete.observePublication(); afterMalformed.publicationID != beforeMalformed.publicationID || afterMalformed.currentMarketClaim || len(afterMalformed.aggregateEvaluation.rows) != 0 || afterMalformed.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("malformed terminal changed B2 publication: before=%+v after=%+v", beforeMalformed, afterMalformed)
	}
	failedTerminal, failedTerminalErr := NewHydrationTerminalInput(incompleteToken, incompleteToken.ResultID(), HydrationFailed, HydrationReasonHTTPRetryExhausted, 1, 3, 0, 0, 0, 0)
	if failedTerminalErr != nil {
		t.Fatal(failedTerminalErr)
	}
	if failed := admitHydrationTerminal(t, incomplete, failedTerminal); failed.Code != DispositionHydrationTerminalApplied {
		t.Fatalf("incomplete terminal disposition=%+v", failed)
	}
	if incompletePublication := incomplete.observePublication(); incompletePublication.currentMarketClaim || len(incompletePublication.aggregateEvaluation.rows) != 0 || incompletePublication.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("incomplete/failed terminal appeared current or enriched: %+v", incompletePublication)
	}
	closeAndWait(t, incomplete)

	now = start.Add(2 * time.Second)
	lost, _ := plannedHydrationEngine(t, binding, &now)
	admitConnectionControl(t, lost, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
	observed := lost.observeHydration()
	if observed.Active || observed.Accounting.Canceled != 1 || lost.state.lifecycle != lifecycleAwaitingAggregateAck {
		t.Fatalf("epoch-loss cancellation = %+v lifecycle=%s", observed, lost.state.lifecycle)
	}
	if canceled := lost.observePublication(); canceled.currentMarketClaim || len(canceled.aggregateEvaluation.rows) != 0 || canceled.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("canceled generation appeared current/complete: %+v", canceled)
	}
	closeAndWait(t, lost)

	now = start.Add(2 * time.Second)
	tracing, traceToken := plannedHydrationEngine(t, binding, &now)
	traceTerminal, _ := NewHydrationTerminalInput(traceToken, traceToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	traceResult := admitHydrationTerminal(t, tracing, traceTerminal)
	now = start.Add(3 * time.Second)
	canceledFence, _ := NewAggregateIngressFenceInput(traceResult.FenceCommand, AggregateIngressFenceCanceled, 1, 1, now)
	_, canceledCompletion := tracing.AdmitAggregateIngressFence(context.Background(), canceledFence)
	if canceled := awaitHydrationDisposition(t, canceledCompletion); canceled.Code != DispositionAggregateIngressFenceRejected || tracing.state.lifecycle != lifecycleAwaitingAggregateAck || tracing.state.hydration.generation.active {
		t.Fatalf("queued-marker loss consequence = %+v lifecycle=%s active=%v", canceled, tracing.state.lifecycle, tracing.state.hydration.generation.active)
	}
	beforeReplaced := tracing.observePublication()
	lateComplete, _ := NewAggregateIngressFenceInput(traceResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
	_, lateCompletion := tracing.AdmitAggregateIngressFence(context.Background(), lateComplete)
	if late := awaitHydrationDisposition(t, lateCompletion); late.Code != DispositionAggregateIngressFenceFenced || tracing.state.lifecycle != lifecycleAwaitingAggregateAck {
		t.Fatalf("old generation completed after loss = %+v lifecycle=%s", late, tracing.state.lifecycle)
	}
	if replaced := tracing.observePublication(); replaced.currentMarketClaim || replaced.publicationID != beforeReplaced.publicationID ||
		len(replaced.aggregateEvaluation.rows) != 0 || replaced.aggregateEvaluation.enrichedRows != 0 {
		t.Fatalf("replaced generation changed publication/currentness: before=%+v after=%+v", beforeReplaced, replaced)
	}
	closeAndWait(t, tracing)

	now = start
	empty := acknowledgedHydrationEngine(t, binding, &now, now, lifecycleHydrating)
	emptyPlan := admitHydrationPlan(t, empty, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	emptyCommand, ok := emptyPlan.Plan.FenceCommand()
	if !emptyPlan.Plan.Empty() || !ok || emptyCommand.CommandToken() == 0 {
		t.Fatalf("R=S did not produce terminal-generation fence command: %+v", emptyPlan)
	}
	emptyFence, _ := NewAggregateIngressFenceInput(emptyCommand, AggregateIngressFenceComplete, 1, 1, now)
	_, emptyCompletion := empty.AdmitAggregateIngressFence(context.Background(), emptyFence)
	if got := awaitHydrationDisposition(t, emptyCompletion); got.Code != DispositionAggregateIngressFenceApplied || empty.state.lifecycle != lifecycleLive || empty.state.committedT == nil || *empty.state.committedT != start {
		t.Fatalf("R=S finalization = %+v lifecycle=%s committed=%v", got, empty.state.lifecycle, empty.state.committedT)
	}
	closeAndWait(t, empty)

	now = start.Add(2 * time.Second)
	ending, _ := plannedHydrationEngine(t, binding, &now)
	now = binding.SessionEnd()
	timerAdmission, timerCompletion := ending.AdmitTimer(context.Background())
	if timerAdmission != AdmissionAdmitted || awaitTimerDisposition(t, timerCompletion).Code != DispositionTimerApplied || ending.state.lifecycle != lifecycleEnded || ending.state.hydration.generation.active || ending.state.hydration.generation.accounting.Canceled != 1 {
		t.Fatalf("session-end hydration exit lifecycle=%s hydration=%+v", ending.state.lifecycle, ending.observeHydration())
	}
	closeAndWait(t, ending)

	now = start.Add(2 * time.Second)
	through, throughToken := plannedHydrationEngine(t, binding, &now)
	now = start.Add(4 * time.Second)
	aggregateAdmission, aggregateCompletion := through.AdmitAggregate(context.Background(), liveAggregate(binding, "AAA", start.Add(2*time.Second), 1, 2))
	if aggregateAdmission != AdmissionAdmitted || awaitAggregateDisposition(t, aggregateCompletion).Code != DispositionAggregateInserted {
		t.Fatal("pre-fence live aggregate")
	}
	throughTerminal, _ := NewHydrationTerminalInput(throughToken, throughToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	throughResult := admitHydrationTerminal(t, through, throughTerminal)
	lowFence, _ := NewAggregateIngressFenceInput(throughResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
	_, lowCompletion := through.AdmitAggregateIngressFence(context.Background(), lowFence)
	if low := awaitHydrationDisposition(t, lowCompletion); low.Code != DispositionAggregateIngressFenceFenced || through.state.lifecycle != lifecycleHydrating {
		t.Fatalf("fence behind greatest consumed raw position = %+v lifecycle=%s", low, through.state.lifecycle)
	}
	currentFence, _ := NewAggregateIngressFenceInput(throughResult.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
	_, currentCompletion := through.AdmitAggregateIngressFence(context.Background(), currentFence)
	if current := awaitHydrationDisposition(t, currentCompletion); current.Code != DispositionAggregateIngressFenceApplied {
		t.Fatalf("current through fence = %+v", current)
	}
	closeAndWait(t, through)
}

// TestC6NOPRINT01CoverageCompositionMatrix is P-C6-NOPRINT. It proves that
// exact empty REST coverage plus the reconciled live fence creates honest
// no-print state, while failure remains unknown and no aggregate is fabricated.
func TestC6NOPRINT01CoverageCompositionMatrix(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	for _, tc := range []struct {
		name     string
		terminal HydrationTerminalState
		reason   HydrationProviderReason
		want     aggregateCoverageConsequence
	}{
		{"complete empty", HydrationCompletedEmpty, HydrationReasonNone, coverageNoPrintThroughT},
		{"failed is unknown", HydrationFailed, HydrationReasonHTTPRetryExhausted, coverageUnknownFailureOrFence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := start.Add(2 * time.Second)
			e, token := plannedHydrationEngine(t, binding, &now)
			term, _ := NewHydrationTerminalInput(token, token.ResultID(), tc.terminal, tc.reason, 1, 1, 10, 0, 0, 0)
			terminalResult := admitHydrationTerminal(t, e, term)
			now = start.Add(3 * time.Second)
			fact, err := NewAggregateIngressFenceInput(terminalResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
			if err != nil {
				t.Fatal(err)
			}
			_, completion := e.AdmitAggregateIngressFence(context.Background(), fact)
			got := awaitHydrationDisposition(t, completion)
			if got.Code != DispositionAggregateIngressFenceApplied || e.state.aggregateEvaluator.coverage[0] != tc.want || aggregatePresentAt(e.state.binding.symbols[0].aggregates, start.Unix()) {
				t.Fatalf("coverage consequence = %+v coverage=%+v fabricated=%v", got, e.state.aggregateEvaluator.coverage[0], aggregatePresentAt(e.state.binding.symbols[0].aggregates, start.Unix()))
			}
			closeAndWait(t, e)
		})
	}

	t.Run("first live print clears no-print through ordinary aggregate path", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		term, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		terminalResult := admitHydrationTerminal(t, e, term)
		now = start.Add(3 * time.Second)
		fact, _ := NewAggregateIngressFenceInput(terminalResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
		_, completion := e.AdmitAggregateIngressFence(context.Background(), fact)
		fenceResult := awaitHydrationDisposition(t, completion)
		if fenceResult.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive {
			t.Fatalf("fence did not enter live: %+v lifecycle=%s", fenceResult, e.state.lifecycle)
		}
		if e.state.aggregateEvaluator.coverage[0] != coverageNoPrintThroughT {
			t.Fatal("no-print not installed")
		}
		capturedT := *e.state.committedT
		now = start.Add(10 * time.Second)
		timerAdmission, timerCompletion := e.AdmitTimer(context.Background())
		if timerAdmission != AdmissionAdmitted || awaitTimerDisposition(t, timerCompletion).Code != DispositionTimerApplied || e.state.committedT == nil || *e.state.committedT != capturedT || *e.state.hydration.supportedThrough != capturedT {
			t.Fatalf("timer manufactured support beyond fence: committed=%v supported=%v captured=%s", e.state.committedT, e.state.hydration.supportedThrough, capturedT)
		}
		if !e.state.liveEpochActive || !e.state.aggregateAcknowledged || e.state.lifecycle != lifecycleLive {
			t.Fatalf("live gate lost before later print: active=%v ack=%v lifecycle=%s", e.state.liveEpochActive, e.state.aggregateAcknowledged, e.state.lifecycle)
		}
		aggregateAdmission, aggregateCompletion := e.AdmitAggregate(context.Background(), liveAggregate(binding, "AAA", start.Add(3*time.Second), 1, 2))
		if aggregateAdmission != AdmissionAdmitted {
			t.Fatal(aggregateAdmission)
		}
		if got := awaitAggregateDisposition(t, aggregateCompletion); got.Code != DispositionAggregateInserted || got.Reason != ReasonNone {
			t.Fatalf("later aggregate = %+v", got)
		}
		if _, ok := e.state.aggregateEvaluator.coverage[0]; ok {
			t.Fatal("later accepted print did not clear no-print")
		}
		if !aggregatePresentAt(e.state.binding.symbols[0].aggregates, start.Add(3*time.Second).Unix()) {
			t.Fatal("ordinary live mark missing")
		}
		closeAndWait(t, e)
	})
}

// TestC6RECOVER01SameProcessLossReackGapRetryExhaustion is P-C6-RECOVER.
// It proves exact retained-boundary replanning, repeated-epoch supersession,
// explicit policy decisions, stale-fact fencing, and ordinary evaluator exit.
func TestC6RECOVER01SameProcessLossReackGapRetryExhaustion(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(2 * time.Second)
	e, initialToken := plannedHydrationEngine(t, binding, &now)
	initialTerminal, _ := NewHydrationTerminalInput(initialToken, initialToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	initial := admitHydrationTerminal(t, e, initialTerminal)
	now = start.Add(3 * time.Second)
	initialFence, _ := NewAggregateIngressFenceInput(initial.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
	_, initialFenceCompletion := e.AdmitAggregateIngressFence(context.Background(), initialFence)
	if got := awaitHydrationDisposition(t, initialFenceCompletion); got.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive {
		t.Fatalf("initial live = %+v lifecycle=%s", got, e.state.lifecycle)
	}
	supported := *e.state.committedT

	now = start.Add(4 * time.Second)
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
	if e.state.lifecycle != lifecycleRecovering || e.state.hydration.supportedT == nil || *e.state.hydration.supportedT != supported || e.state.committedT == nil || *e.state.committedT != supported ||
		e.state.aggregateEvaluator.current.mode != rankingStale || len(e.state.aggregateEvaluator.current.rows) != 0 || e.state.aggregateEvaluator.current.tqIntentAvailable {
		t.Fatalf("loss boundary/currentness: lifecycle=%s supported=%v committed=%v evaluation=%+v", e.state.lifecycle, e.state.hydration.supportedT, e.state.committedT, e.state.aggregateEvaluator.current)
	}
	now = start.Add(5 * time.Second)
	_, frozenTimer := e.AdmitTimer(context.Background())
	_ = awaitTimerDisposition(t, frozenTimer)
	if *e.state.committedT != supported {
		t.Fatalf("recovering timer advanced T: %s -> %s", supported, *e.state.committedT)
	}

	ackRecoveryEpoch(t, e, binding, 2, 3, 4, start.Add(6*time.Second), &now)
	firstGap := admitHydrationPlan(t, e, HydrationGapRecovery, 2, generousHydrationBudgets())
	if firstGap.Plan.Start() != supported || firstGap.Plan.End() != start.Add(6*time.Second) || firstGap.Plan.Start() == supported.Add(-16*time.Minute) || firstGap.Plan.Generation() != 2 {
		t.Fatalf("epoch-2 exact gap = [%s,%s) generation=%d supported=%s", firstGap.Plan.Start(), firstGap.Plan.End(), firstGap.Plan.Generation(), supported)
	}
	oldToken := firstGap.Plan.Requests()[0]
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 2}, now, 0, ControlFailed))
	if e.state.hydration.generation.active || e.state.hydration.generation.accounting != (HydrationAccounting{Planned: 1, Canceled: 1}) || *e.state.hydration.supportedT != supported {
		t.Fatalf("open-work second loss = %+v supported=%v", e.observeHydration(), e.state.hydration.supportedT)
	}
	lateTerminal, _ := NewHydrationTerminalInput(oldToken, oldToken.ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
	if late := admitHydrationTerminal(t, e, lateTerminal); late.Code != DispositionHydrationFenced || late.Accounting != (HydrationAccounting{Planned: 1, Canceled: 1}) {
		t.Fatalf("old terminal = %+v", late)
	}

	ackRecoveryEpoch(t, e, binding, 3, 5, 6, start.Add(8*time.Second), &now)
	retryGap := admitHydrationPlan(t, e, HydrationGapRecovery, 3, generousHydrationBudgets())
	if retryGap.Plan.Start() != supported || retryGap.Plan.End() != start.Add(8*time.Second) || retryGap.Plan.Generation() != 3 {
		t.Fatalf("epoch-3 gap = %+v", retryGap.Plan)
	}
	retryToken := retryGap.Plan.Requests()[0]
	failed, _ := NewHydrationTerminalInput(retryToken, retryToken.ResultID(), HydrationFailed, HydrationReasonHTTPRetryExhausted, 1, 3, 0, 0, 0, 0)
	failedResult := admitHydrationTerminal(t, e, failed)
	retryAction, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyRetry, 1)
	if got := admitHydrationPolicy(t, e, retryAction); got.Code != DispositionHydrationPolicyApplied || e.state.hydration.generation.active {
		t.Fatalf("retry action = %+v active=%v", got, e.state.hydration.generation.active)
	}
	secondAction, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyExhaust, 2)
	if got := admitHydrationPolicy(t, e, secondAction); got.Code != DispositionHydrationPolicyRejected || e.state.hydration.lastPolicyToken != 1 || e.state.lifecycle != lifecycleRecovering {
		t.Fatalf("second decision mutated newer recovery = %+v token=%d lifecycle=%s", got, e.state.hydration.lastPolicyToken, e.state.lifecycle)
	}
	lateFence, _ := NewAggregateIngressFenceInput(failedResult.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
	_, lateFenceCompletion := e.AdmitAggregateIngressFence(context.Background(), lateFence)
	if got := awaitHydrationDisposition(t, lateFenceCompletion); got.Code != DispositionAggregateIngressFenceFenced {
		t.Fatalf("superseded fence = %+v", got)
	}

	finalGap := admitHydrationPlan(t, e, HydrationGapRecovery, 3, generousHydrationBudgets())
	if finalGap.Plan.Generation() != 4 || finalGap.Plan.Start() != supported {
		t.Fatalf("retry plan = %+v", finalGap.Plan)
	}
	now = start.Add(9 * time.Second)
	liveAdmission, liveCompletion := e.AdmitAggregate(context.Background(), liveAggregate(binding, "AAA", start.Add(8*time.Second), 3, 2))
	if liveAdmission != AdmissionAdmitted || awaitAggregateDisposition(t, liveCompletion).Code != DispositionAggregateInserted {
		t.Fatal("current recovery live tail")
	}
	finalToken := finalGap.Plan.Requests()[0]
	finalTerminal, _ := NewHydrationTerminalInput(finalToken, finalToken.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	finalResult := admitHydrationTerminal(t, e, finalTerminal)
	invalidExhaust, _ := NewHydrationPolicyActionInput(finalResult.FenceCommand, HydrationPolicyExhaust, 2)
	if got := admitHydrationPolicy(t, e, invalidExhaust); got.Code != DispositionHydrationPolicyRejected || e.state.lifecycle != lifecycleRecovering || !e.state.hydration.generation.active {
		t.Fatalf("exhaust without unresolved gap = %+v", got)
	}
	finalFence, _ := NewAggregateIngressFenceInput(finalResult.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
	_, finalCompletion := e.AdmitAggregateIngressFence(context.Background(), finalFence)
	if got := awaitHydrationDisposition(t, finalCompletion); got.Code != DispositionAggregateIngressFenceApplied || e.state.lifecycle != lifecycleLive || e.state.committedT == nil || *e.state.committedT != now || e.state.hydration.generation.accounting != (HydrationAccounting{Planned: 1, CompletedEmpty: 1}) {
		t.Fatalf("recovery exit = %+v lifecycle=%s committed=%v accounting=%+v", got, e.state.lifecycle, e.state.committedT, e.state.hydration.generation.accounting)
	}
	closeAndWait(t, e)

	t.Run("cancel and exhaust have concrete terminal states", func(t *testing.T) {
		terminalLossEngine, terminalLossResult := failedRecoveryGeneration(t, binding, start)
		retained := *terminalLossEngine.state.hydration.supportedT
		admitConnectionControl(t, terminalLossEngine, controlFact(binding.Identity(), ConnectionLost, 2, LivePosition{ConnectionEpoch: 2, FrameSequence: 2}, start.Add(6*time.Second), 0, ControlFailed))
		if terminalLossEngine.state.lifecycle != lifecycleRecovering || terminalLossEngine.state.hydration.generation.active || terminalLossEngine.state.hydration.generation.accounting.Failed != 1 || *terminalLossEngine.state.hydration.supportedT != retained {
			t.Fatalf("terminal-before-marker loss = lifecycle=%s hydration=%+v", terminalLossEngine.state.lifecycle, terminalLossEngine.observeHydration())
		}
		lostFence, _ := NewAggregateIngressFenceInput(terminalLossResult.FenceCommand, AggregateIngressFenceComplete, 2, 1, start.Add(6*time.Second))
		_, lostFenceCompletion := terminalLossEngine.AdmitAggregateIngressFence(context.Background(), lostFence)
		if got := awaitHydrationDisposition(t, lostFenceCompletion); got.Code != DispositionAggregateIngressFenceFenced {
			t.Fatalf("terminal-before-marker old fence = %+v", got)
		}
		closeAndWait(t, terminalLossEngine)

		cancelEngine, cancelResult := failedRecoveryGeneration(t, binding, start)
		cancelAction, _ := NewHydrationPolicyActionInput(cancelResult.FenceCommand, HydrationPolicyCancel, 1)
		if got := admitHydrationPolicy(t, cancelEngine, cancelAction); got.Code != DispositionHydrationPolicyApplied || cancelEngine.state.lifecycle != lifecycleRecovering || cancelEngine.state.liveEpochActive || cancelEngine.state.hydration.generation.active || !cancelEngine.state.hydration.policyWaiting {
			t.Fatalf("cancel action = %+v lifecycle=%s", got, cancelEngine.state.lifecycle)
		}
		closeAndWait(t, cancelEngine)

		exhaustEngine, exhaustResult := failedRecoveryGeneration(t, binding, start)
		exhaustAction, _ := NewHydrationPolicyActionInput(exhaustResult.FenceCommand, HydrationPolicyExhaust, 1)
		if got := admitHydrationPolicy(t, exhaustEngine, exhaustAction); got.Code != DispositionIngressIntegrity || exhaustEngine.state.lifecycle != lifecycleSuppressed || exhaustEngine.state.suppressionDisposition != SuppressionSameBindingRecoveryAllowed {
			t.Fatalf("exhaust action = %+v lifecycle=%s suppression=%s", got, exhaustEngine.state.lifecycle, exhaustEngine.state.suppressionDisposition)
		}
		if published := exhaustEngine.observePublication(); published.lastDisposition != DispositionIngressIntegrity || published.dispositionReason != ReasonAggregateIngressFence || published.lifecycleReason != lifecycleReasonRecoveryExhausted {
			t.Fatalf("exhaust publication = %+v", published)
		}
		closeAndWait(t, exhaustEngine)

		unresolvedEngine, unresolvedResult := failedRecoveryGeneration(t, binding, start)
		unresolvedFence, _ := NewAggregateIngressFenceInput(unresolvedResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, start.Add(6*time.Second))
		_, unresolvedCompletion := unresolvedEngine.AdmitAggregateIngressFence(context.Background(), unresolvedFence)
		if got := awaitHydrationDisposition(t, unresolvedCompletion); got.Code != DispositionAggregateIngressFenceApplied || unresolvedEngine.state.lifecycle != lifecycleRecovering || !unresolvedEngine.state.hydration.generation.active || !unresolvedEngine.state.hydration.policyWaiting || unresolvedEngine.state.aggregateEvaluator.coverage[0] != coverageUnknownPostBootstrap {
			t.Fatalf("unresolved recovery fence = %+v lifecycle=%s coverage=%+v", got, unresolvedEngine.state.lifecycle, unresolvedEngine.state.aggregateEvaluator.coverage[0])
		}
		if published := unresolvedEngine.observePublication(); published.lifecycle != lifecycleRecovering || !published.hydrationPolicyWaiting || published.currentMarketClaim {
			t.Fatalf("unresolved policy-wait publication = %+v", published)
		}
		repeatedFence, _ := NewAggregateIngressFenceInput(unresolvedResult.FenceCommand, AggregateIngressFenceComplete, 1, 1, start.Add(6*time.Second))
		_, repeatedCompletion := unresolvedEngine.AdmitAggregateIngressFence(context.Background(), repeatedFence)
		if got := awaitHydrationDisposition(t, repeatedCompletion); got.Code != DispositionAggregateIngressFenceRejected || unresolvedEngine.state.lifecycle != lifecycleRecovering || !unresolvedEngine.state.hydration.policyWaiting {
			t.Fatalf("repeated unresolved fence = %+v lifecycle=%s hydration=%+v", got, unresolvedEngine.state.lifecycle, unresolvedEngine.observeHydration())
		}
		exhaust, _ := NewHydrationPolicyActionInput(unresolvedResult.FenceCommand, HydrationPolicyExhaust, 1)
		if got := admitHydrationPolicy(t, unresolvedEngine, exhaust); got.Code != DispositionIngressIntegrity || unresolvedEngine.state.lifecycle != lifecycleSuppressed {
			t.Fatalf("explicit unresolved exhaust = %+v lifecycle=%s", got, unresolvedEngine.state.lifecycle)
		}
		closeAndWait(t, unresolvedEngine)
	})

	t.Run("stop validates before sealing and ends once", func(t *testing.T) {
		stopEngine, failedResult := failedRecoveryGeneration(t, binding, start)
		wrongBinding, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyStop, 1)
		wrongBinding.bindingID = mutateIdentity(wrongBinding.bindingID)
		wrongEpoch, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyStop, 1)
		wrongEpoch.epoch++
		wrongGeneration, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyStop, 1)
		wrongGeneration.generation++
		for name, invalid := range map[string]HydrationPolicyActionInput{"binding": wrongBinding, "epoch": wrongEpoch, "generation": wrongGeneration} {
			if got := admitHydrationPolicy(t, stopEngine, invalid); got.Code != DispositionHydrationPolicyRejected || stopEngine.sealed || stopEngine.state.lifecycle != lifecycleRecovering || !stopEngine.state.hydration.generation.active || stopEngine.state.hydration.lastPolicyToken != 0 {
				t.Fatalf("%s stop mutated engine = %+v sealed=%v lifecycle=%s hydration=%+v", name, got, stopEngine.sealed, stopEngine.state.lifecycle, stopEngine.observeHydration())
			}
		}

		retry, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyRetry, 1)
		if got := admitHydrationPolicy(t, stopEngine, retry); got.Code != DispositionHydrationPolicyApplied || stopEngine.sealed {
			t.Fatalf("valid fact after rejected stop = %+v sealed=%v", got, stopEngine.sealed)
		}
		plan := admitHydrationPlan(t, stopEngine, HydrationGapRecovery, 2, generousHydrationBudgets())
		token := plan.Plan.Requests()[0]
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationFailed, HydrationReasonHTTPRetryExhausted, 1, 3, 0, 0, 0, 0)
		current := admitHydrationTerminal(t, stopEngine, terminal)

		staleToken, _ := NewHydrationPolicyActionInput(current.FenceCommand, HydrationPolicyStop, 1)
		if got := admitHydrationPolicy(t, stopEngine, staleToken); got.Code != DispositionHydrationPolicyRejected || got.Reason != ReasonHydrationToken || stopEngine.sealed || stopEngine.state.hydration.lastPolicyToken != 1 || !stopEngine.state.hydration.generation.active {
			t.Fatalf("stale-token stop mutated current generation = %+v sealed=%v hydration=%+v", got, stopEngine.sealed, stopEngine.observeHydration())
		}
		superseded, _ := NewHydrationPolicyActionInput(failedResult.FenceCommand, HydrationPolicyStop, 2)
		if got := admitHydrationPolicy(t, stopEngine, superseded); got.Code != DispositionHydrationPolicyRejected || stopEngine.sealed || stopEngine.state.hydration.lastPolicyToken != 1 || !stopEngine.state.hydration.generation.active {
			t.Fatalf("superseded stop mutated current generation = %+v sealed=%v hydration=%+v", got, stopEngine.sealed, stopEngine.observeHydration())
		}
		valid, _ := NewHydrationPolicyActionInput(current.FenceCommand, HydrationPolicyStop, 2)
		if got := admitHydrationPolicy(t, stopEngine, valid); got.Code != DispositionHydrationPolicyApplied || !stopEngine.sealed || stopEngine.state.lifecycle != lifecycleEnded || stopEngine.state.hydration.generation.active || stopEngine.state.hydration.lastPolicyToken != 2 {
			t.Fatalf("valid stop = %+v sealed=%v lifecycle=%s hydration=%+v", got, stopEngine.sealed, stopEngine.state.lifecycle, stopEngine.observeHydration())
		}
		endedSequence := stopEngine.state.latestTransition.EngineSequence
		duplicate, _ := NewHydrationPolicyActionInput(current.FenceCommand, HydrationPolicyStop, 3)
		if admission, completion := stopEngine.AdmitHydrationPolicyAction(context.Background(), duplicate); admission != AdmissionNotAdmittedClosed || completion != nil || stopEngine.state.lifecycle != lifecycleEnded || stopEngine.state.latestTransition.EngineSequence != endedSequence {
			t.Fatalf("duplicate stop = admission=%s completion=%v lifecycle=%s transition=%+v", admission, completion, stopEngine.state.lifecycle, stopEngine.state.latestTransition)
		}
		closeAndWait(t, stopEngine)
	})
}

// TestC6LEDGER01OrderedChunkTerminalSupersession is P-C6-LEDGER. It proves
// token/result/chunk order, exact terminal-bin accounting, idempotent ordered
// cancellation, and fencing of stale/late evidence. Dangerous counterexamples
// are a missing chunk, repeated terminal, stale generation, and late result
// changing accounting after supersession. It does not prove REST authenticity
// or the final no-print meaning.
func TestC6LEDGER01OrderedChunkTerminalSupersession(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	for _, tc := range []struct {
		name   string
		state  HydrationTerminalState
		reason HydrationProviderReason
		want   HydrationAccounting
	}{
		{"empty", HydrationCompletedEmpty, HydrationReasonNone, HydrationAccounting{Planned: 1, CompletedEmpty: 1}},
		{"failed", HydrationFailed, HydrationReasonHTTPRetryExhausted, HydrationAccounting{Planned: 1, Failed: 1}},
		{"canceled", HydrationCanceled, HydrationReasonCanceled, HydrationAccounting{Planned: 1, Canceled: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := start.Add(2 * time.Second)
			e, token := plannedHydrationEngine(t, binding, &now)
			terminal, err := NewHydrationTerminalInput(token, token.ResultID(), tc.state, tc.reason, 1, 1, 10, 0, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			got := admitHydrationTerminal(t, e, terminal)
			if got.Code != DispositionHydrationTerminalApplied || got.Accounting != tc.want || !got.Accounting.reconciles() {
				t.Fatalf("terminal accounting = %+v want=%+v", got, tc.want)
			}
			repeat := admitHydrationTerminal(t, e, terminal)
			if repeat.Code != DispositionHydrationRejected || repeat.Accounting != tc.want {
				t.Fatalf("repeated terminal = %+v", repeat)
			}
			closeAndWait(t, e)
		})
	}

	t.Run("ordered chunks then value terminal reconcile rows", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		rows := []HydrationRow{hydrationRow(t, "AAA", start, 10), hydrationRow(t, "AAA", start.Add(time.Second), 11)}
		first, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 2, 0, 2, rows[:1])
		second, _ := NewHydrationChunkInput(token, token.ResultID(), 1, 2, 1, 2, rows[1:])
		got := admitHydrationChunk(t, e, first)
		if got.Code != DispositionHydrationChunkApplied || got.Rows != (HydrationRowAccounting{Consumed: 1, Inserted: 1}) || got.Accounting.Open != 1 {
			t.Fatalf("first chunk = %+v", got)
		}
		got = admitHydrationChunk(t, e, second)
		if got.Code != DispositionHydrationChunkApplied || got.Rows != (HydrationRowAccounting{Consumed: 1, Inserted: 1}) {
			t.Fatalf("second chunk = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 100, 2, 2, 2)
		term := admitHydrationTerminal(t, e, terminal)
		if term.Code != DispositionHydrationTerminalApplied || term.Accounting != (HydrationAccounting{Planned: 1, CompletedValue: 1}) || !e.observeHydration().Rows.reconciles() {
			t.Fatalf("value terminal = %+v state=%+v", term, e.observeHydration())
		}
		closeAndWait(t, e)
	})

	t.Run("FIFO cancellation closes once and fences late evidence", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		staleToken := token
		staleToken.generation++
		staleToken.requestToken = hydrationRequestToken(staleToken.generation, staleToken.requestID)
		staleTerminal, _ := NewHydrationTerminalInput(staleToken, staleToken.ResultID(), HydrationFailed, HydrationReasonHTTPRetryExhausted, 0, 0, 0, 0, 0, 0)
		stale := admitHydrationTerminal(t, e, staleTerminal)
		if stale.Code != DispositionHydrationFenced || stale.Accounting != (HydrationAccounting{Planned: 1, Open: 1}) {
			t.Fatalf("stale generation terminal = %+v", stale)
		}
		result, completion := e.admitHydrationCancelForProof(context.Background(), binding.Identity(), token.Generation(), false)
		if result != AdmissionAdmitted {
			t.Fatalf("cancel admission = %s", result)
		}
		cancel := awaitHydrationDisposition(t, completion)
		if cancel.Code != DispositionHydrationTerminalApplied || cancel.Accounting != (HydrationAccounting{Planned: 1, Canceled: 1}) {
			t.Fatalf("cancel = %+v", cancel)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
		late := admitHydrationTerminal(t, e, terminal)
		if late.Code != DispositionHydrationFenced || late.Accounting != cancel.Accounting {
			t.Fatalf("late terminal = %+v", late)
		}
		result, completion = e.admitHydrationCancelForProof(context.Background(), binding.Identity(), token.Generation(), false)
		repeated := awaitHydrationDisposition(t, completion)
		if result != AdmissionAdmitted || repeated.Code != DispositionHydrationRejected || repeated.Accounting != cancel.Accounting {
			t.Fatal("repeated cancellation double counted")
		}
		closeAndWait(t, e)
	})

	t.Run("out of order chunk fails closed before canonical mutation", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		row := hydrationRow(t, "AAA", start.Add(time.Second), 11)
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 1, 2, 1, 2, []HydrationRow{row})
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		e.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		firstAdmission, firstCompletion := e.AdmitHydrationChunk(context.Background(), chunk)
		if firstAdmission != AdmissionAdmitted {
			t.Fatalf("first chunk admission = %s", firstAdmission)
		}
		<-entered
		lateAdmission, lateCompletion := e.AdmitHydrationChunk(context.Background(), chunk)
		if lateAdmission != AdmissionAdmitted {
			t.Fatalf("linked late chunk admission = %s", lateAdmission)
		}
		close(release)
		got := awaitHydrationDisposition(t, firstCompletion)
		want := HydrationAccounting{Planned: 1, Fenced: 1}
		if got.Code != DispositionHydrationIntegrity || got.Reason != ReasonHydrationChunkSequence || got.Rows.Consumed != 0 || got.Accounting != want || !got.Accounting.reconciles() {
			t.Fatalf("out-of-order chunk = %+v", got)
		}
		assertNoIdentity(t, e, "AAA", row.WindowStart())
		late := awaitHydrationDisposition(t, lateCompletion)
		if late.Code != DispositionHydrationFenced || late.Accounting != want {
			t.Fatalf("late chunk after integrity suppression = %+v", late)
		}
		closeAndWait(t, e)
	})

	t.Run("impossible producer terminal counts fence open work before suppression", func(t *testing.T) {
		cases := []struct {
			name   string
			mutate func(*HydrationTerminalInput)
		}{
			{"normalized rows without a successful page", func(input *HydrationTerminalInput) { input.normalizedRows = 1 }},
			{"normalized rows exceed request interval", func(input *HydrationTerminalInput) { input.pages, input.attempts, input.normalizedRows = 1, 1, 3 }},
			{"four attempts without a completed page", func(input *HydrationTerminalInput) { input.attempts = 4 }},
			{"wire bytes without an attempt", func(input *HydrationTerminalInput) { input.responseBytes = 1 }},
			{"emitted chunk without emitted row", func(input *HydrationTerminalInput) { input.emittedChunks = 1 }},
			{"completed empty without a page", func(input *HydrationTerminalInput) {
				input.state, input.reason = HydrationCompletedEmpty, HydrationReasonNone
			}},
			{"failed result claims normalized evidence", func(input *HydrationTerminalInput) {
				input.state, input.reason, input.pages, input.attempts, input.normalizedRows = HydrationFailed, HydrationReasonNumericCount, 1, 1, 1
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				now := start.Add(2 * time.Second)
				e, token := plannedHydrationEngine(t, binding, &now)
				terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
				if err != nil {
					t.Fatal(err)
				}
				tc.mutate(&terminal)
				entered, release := make(chan struct{}), make(chan struct{})
				var once sync.Once
				e.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
				firstAdmission, firstCompletion := e.AdmitHydrationTerminal(context.Background(), terminal)
				if firstAdmission != AdmissionAdmitted {
					t.Fatalf("first terminal admission = %s", firstAdmission)
				}
				<-entered
				lateAdmission, lateCompletion := e.AdmitHydrationTerminal(context.Background(), terminal)
				if lateAdmission != AdmissionAdmitted {
					t.Fatalf("linked late terminal admission = %s", lateAdmission)
				}
				close(release)
				got := awaitHydrationDisposition(t, firstCompletion)
				want := HydrationAccounting{Planned: 1, Fenced: 1}
				if got.Code != DispositionHydrationIntegrity || got.Reason != ReasonHydrationTerminalSequence || got.Accounting != want || !got.Accounting.reconciles() {
					t.Fatalf("impossible terminal = %+v", got)
				}
				late := awaitHydrationDisposition(t, lateCompletion)
				if late.Code != DispositionHydrationFenced || late.Accounting != want {
					t.Fatalf("late terminal after integrity = %+v", late)
				}
				closeAndWait(t, e)
			})
		}
	})

	t.Run("failed terminal may exactly consume the response budget", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		maximum := e.state.hydration.generation.budgets.MaximumResponseBytes
		terminal, err := NewHydrationTerminalInput(token, token.ResultID(), HydrationFailed, HydrationReasonResponseSizeSyntax, 0, 1, maximum, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		got := admitHydrationTerminal(t, e, terminal)
		if got.Code != DispositionHydrationTerminalApplied || got.Accounting != (HydrationAccounting{Planned: 1, Failed: 1}) {
			t.Fatalf("exact-budget failure = %+v", got)
		}
		if view := e.ObserveOperational(); view.Lifecycle == "suppressed" || view.Suppression != "" {
			t.Fatalf("bounded provider failure suppressed engine = %+v", view)
		}
		closeAndWait(t, e)
	})

	t.Run("sealed result may cancel before its first chunk", func(t *testing.T) {
		now := start.Add(2 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCanceled, HydrationReasonCanceled, 1, 1, 10, 1, 0, 0)
		got := admitHydrationTerminal(t, e, terminal)
		if got.Code != DispositionHydrationTerminalApplied || got.Accounting != (HydrationAccounting{Planned: 1, Canceled: 1}) {
			t.Fatalf("pre-chunk sealed cancellation = %+v", got)
		}
		closeAndWait(t, e)
	})

	t.Run("partial emission must retain the sealed result cardinality", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		row := hydrationRow(t, "AAA", start, 10)
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 2, 0, 2, []HydrationRow{row})
		if got := admitHydrationChunk(t, e, chunk); got.Code != DispositionHydrationChunkApplied {
			t.Fatalf("partial chunk = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCanceled, HydrationReasonCanceled, 1, 1, 10, 1, 1, 1)
		got := admitHydrationTerminal(t, e, terminal)
		if got.Code != DispositionHydrationIntegrity || got.Accounting != (HydrationAccounting{Planned: 1, Fenced: 1}) {
			t.Fatalf("cardinality-mismatched cancellation = %+v", got)
		}
		closeAndWait(t, e)
	})

	t.Run("successful terminals require possible retry counts and positive response bytes", func(t *testing.T) {
		for _, state := range []HydrationTerminalState{HydrationCompletedEmpty, HydrationCompletedValue} {
			for _, invalid := range []struct {
				name          string
				attempts      int64
				responseBytes int64
			}{
				{"six attempts for one page", 6, 10},
				{"zero-byte successful response", 1, 0},
			} {
				t.Run(string(state)+"/"+invalid.name, func(t *testing.T) {
					now := start.Add(2 * time.Second)
					e, token := plannedHydrationEngine(t, binding, &now)
					var normalized, chunks, rows int64
					if state == HydrationCompletedValue {
						row := hydrationRow(t, "AAA", start, 10)
						chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
						if got := admitHydrationChunk(t, e, chunk); got.Code != DispositionHydrationChunkApplied {
							t.Fatalf("value chunk = %+v", got)
						}
						normalized, chunks, rows = 1, 1, 1
					}
					terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), state, HydrationReasonNone, 1, invalid.attempts, invalid.responseBytes, normalized, chunks, rows)
					got := admitHydrationTerminal(t, e, terminal)
					if got.Code != DispositionHydrationIntegrity || got.Reason != ReasonHydrationTerminalSequence || got.Accounting != (HydrationAccounting{Planned: 1, Fenced: 1}) {
						t.Fatalf("impossible successful terminal = %+v", got)
					}
					closeAndWait(t, e)
				})
			}
		}
	})

	t.Run("producer permutation preserves all five terminal bins", func(t *testing.T) {
		population := hydrationPopulationBinding(t, []string{"AAA", "BBB", "CCC", "DDD", "EEE"})
		now := population.SessionStart().Add(2 * time.Second)
		e := acknowledgedHydrationEngine(t, population, &now, now, lifecycleHydrating)
		plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		tokens := plan.Plan.Requests()
		if len(tokens) != 5 {
			t.Fatalf("population plan = %+v", plan.Plan)
		}
		row := hydrationRow(t, tokens[0].Symbol(), population.SessionStart(), 10)
		chunk, _ := NewHydrationChunkInput(tokens[0], tokens[0].ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		if got := admitHydrationChunk(t, e, chunk); got.Code != DispositionHydrationChunkApplied {
			t.Fatalf("value chunk = %+v", got)
		}
		terminals := make([]HydrationTerminalInput, 4)
		terminals[0], _ = NewHydrationTerminalInput(tokens[0], tokens[0].ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		terminals[1], _ = NewHydrationTerminalInput(tokens[1], tokens[1].ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
		terminals[2], _ = NewHydrationTerminalInput(tokens[2], tokens[2].ResultID(), HydrationFailed, HydrationReasonRequestConstruction, 0, 0, 0, 0, 0, 0)
		terminals[3], _ = NewHydrationTerminalInput(tokens[3], tokens[3].ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
		completions := make([]<-chan HydrationDisposition, len(terminals))
		var group sync.WaitGroup
		group.Add(len(terminals))
		for index := range terminals {
			go func(index int) {
				defer group.Done()
				result, completion := e.AdmitHydrationTerminal(context.Background(), terminals[index])
				if result == AdmissionAdmitted {
					completions[index] = completion
				}
			}(index)
		}
		group.Wait()
		for index, completion := range completions {
			if completion == nil {
				t.Fatalf("terminal %d was not admitted", index)
			}
			if got := awaitHydrationDisposition(t, completion); got.Code != DispositionHydrationTerminalApplied || !got.Accounting.reconciles() {
				t.Fatalf("permuted terminal %d = %+v", index, got)
			}
		}
		result, completion := e.admitHydrationCancelForProof(context.Background(), population.Identity(), tokens[0].Generation(), true)
		if result != AdmissionAdmitted {
			t.Fatalf("supersession admission = %s", result)
		}
		got := awaitHydrationDisposition(t, completion)
		want := HydrationAccounting{Planned: 5, CompletedValue: 1, CompletedEmpty: 1, Failed: 1, Canceled: 1, Fenced: 1}
		if got.Code != DispositionHydrationTerminalApplied || got.Accounting != want || !got.Accounting.reconciles() {
			t.Fatalf("five-bin terminal partition = %+v want=%+v", got, want)
		}
		closeAndWait(t, e)
	})

	t.Run("conflicting repeated terminal fences every other open request", func(t *testing.T) {
		population := hydrationPopulationBinding(t, []string{"AAA", "BBB"})
		now := population.SessionStart().Add(2 * time.Second)
		e := acknowledgedHydrationEngine(t, population, &now, now, lifecycleHydrating)
		plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		tokens := plan.Plan.Requests()
		failed, _ := NewHydrationTerminalInput(tokens[0], tokens[0].ResultID(), HydrationFailed, HydrationReasonRequestConstruction, 0, 0, 0, 0, 0, 0)
		if got := admitHydrationTerminal(t, e, failed); got.Code != DispositionHydrationTerminalApplied {
			t.Fatalf("first terminal = %+v", got)
		}
		conflicting, _ := NewHydrationTerminalInput(tokens[0], tokens[0].ResultID(), HydrationCanceled, HydrationReasonCanceled, 0, 0, 0, 0, 0, 0)
		got := admitHydrationTerminal(t, e, conflicting)
		want := HydrationAccounting{Planned: 2, Failed: 1, Fenced: 1}
		if got.Code != DispositionHydrationIntegrity || got.Reason != ReasonHydrationTerminalSequence || got.Accounting != want || !got.Accounting.reconciles() {
			t.Fatalf("conflicting repeat = %+v want=%+v", got, want)
		}
		closeAndWait(t, e)
	})
}

// TestC6MERGE01ProductionHistoricalDispositionMatrix is P-C6-MERGE. It proves
// C6 rows use Component 2's fill-only canonical decisions: historical cannot
// overwrite live, equality deduplicates, resolved REST/live discrepancies stay
// diagnostic-only, and unresolved historical ambiguity remains fail-closed. It
// does not prove the later fence/lifecycle/no-print composition.
func TestC6MERGE01ProductionHistoricalDispositionMatrix(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("live equality deduplicates while provider value completes", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		live := liveAggregate(binding, "AAA", start, 1, 2)
		live.Values.ATSProvenance = ATSRESTFloorVolumeOverTrades
		live.Live.ArrayIndex = 1
		if got := admitProductionAggregate(t, e, live); got.Code != DispositionAggregateInserted {
			t.Fatalf("live = %+v", got)
		}
		row, _ := NewHydrationRow("AAA", start, start.Add(time.Second), AggregateValues{
			Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: ATSRESTFloorVolumeOverTrades,
		})
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		got := admitHydrationChunk(t, e, chunk)
		if got.Rows != (HydrationRowAccounting{Consumed: 1, Duplicate: 1}) {
			t.Fatalf("equal merge = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		if got := admitHydrationTerminal(t, e, terminal); got.Code != DispositionHydrationTerminalApplied || e.observeHydration().Requests[0].coverage != hydrationCoverageCandidateComplete {
			t.Fatalf("equal terminal/coverage = %+v state=%+v", got, e.observeHydration())
		}
		assertCanonicalClose(t, e, "AAA", start, 10)
		closeAndWait(t, e)
	})

	t.Run("unequal historical cannot overwrite live and coverage stays exact", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		live := liveAggregate(binding, "AAA", start, 1, 2)
		live.Live.ArrayIndex = 1
		admitProductionAggregate(t, e, live)
		row := hydrationRow(t, "AAA", start, 9)
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		got := admitHydrationChunk(t, e, chunk)
		if got.Rows != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) {
			t.Fatalf("conflict row = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		term := admitHydrationTerminal(t, e, terminal)
		observation := e.observeHydration()
		if term.Code != DispositionHydrationTerminalApplied || observation.Accounting.CompletedValue != 1 || observation.Requests[0].coverage != hydrationCoverageCandidateComplete {
			t.Fatalf("provider-success/canonical-coverage separation = %+v state=%+v", term, observation)
		}
		assertCanonicalClose(t, e, "AAA", start, 10)
		state := aggregateState(t, e, "AAA")
		if state.historicalConflict != nil && state.historicalConflict.has(sessionSlot(e.state.binding, start)) || !exactAggregateCoverage(state, e.state.binding, start, start.Add(time.Second)) {
			t.Fatalf("resolved REST/live discrepancy poisoned canonical coverage: %+v", state)
		}
		closeAndWait(t, e)
	})

	t.Run("source specific ATS provenance remains diagnostic only", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		live := liveAggregate(binding, "AAA", start, 1, 2)
		live.Live.ArrayIndex = 1
		admitProductionAggregate(t, e, live)
		row, _ := NewHydrationRow("AAA", start, start.Add(time.Second), AggregateValues{
			Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10,
			AverageTradeSize: 10, ATSProvenance: ATSRESTFloorVolumeOverTrades,
		})
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		if got := admitHydrationChunk(t, e, chunk); got.Rows != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) {
			t.Fatalf("ATS provenance diagnostic accounting = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		if got := admitHydrationTerminal(t, e, terminal); got.Code != DispositionHydrationTerminalApplied || e.observeHydration().Requests[0].coverage != hydrationCoverageCandidateComplete {
			t.Fatalf("ATS provenance poisoned request coverage: disposition=%+v state=%+v", got, e.observeHydration())
		}
		record := aggregateRecord(t, e, "AAA", start)
		state := aggregateState(t, e, "AAA")
		if record.values != live.Values || record.authority.source != AggregateSourceLive ||
			state.historicalConflict != nil && state.historicalConflict.has(sessionSlot(e.state.binding, start)) ||
			!exactAggregateCoverage(state, e.state.binding, start, start.Add(time.Second)) {
			t.Fatalf("ATS provenance changed canonical live authority or coverage: record=%+v state=%+v", record, state)
		}
		closeAndWait(t, e)
	})

	t.Run("unequal historical arrivals withdraw instead of choosing an arrival-order winner", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e := aggregateEngine(t, binding, &now)
		first := historicalAggregate(binding, "AAA", start, 1)
		first.Values.Close, first.Values.Low = 10, 10
		applyHistorical(t, e, first, proofFor(binding, first, start, start.Add(3*time.Second)), DispositionAggregateInserted, ReasonNone)
		admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, now, 1, ControlSucceeded))
		admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, now, 2, ControlSucceeded))
		admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, now, 2, ControlSucceeded))
		plan := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
		token := plan.Plan.Requests()[0]
		row := hydrationRow(t, "AAA", start, 9)
		chunk, _ := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
		got := admitHydrationChunk(t, e, chunk)
		if got.Rows != (HydrationRowAccounting{Consumed: 1, ConflictOrWithdrawal: 1}) {
			t.Fatalf("historical/historical conflict = %+v", got)
		}
		assertNoIdentity(t, e, "AAA", start)
		later := liveAggregate(binding, "AAA", start.Add(2*time.Second), 1, 2)
		later.Values = AggregateValues{Open: 11, High: 12, Low: 10, Close: 11, Volume: 5_000, VWAP: 11, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}
		if got := admitProductionAggregate(t, e, later); got.Code != DispositionAggregateInserted {
			t.Fatalf("later live authority = %+v", got)
		}
		terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedValue, HydrationReasonNone, 1, 1, 10, 1, 1, 1)
		terminalDisposition := admitHydrationTerminal(t, e, terminal)
		if terminalDisposition.Code != DispositionHydrationTerminalApplied || e.observeHydration().Requests[0].coverage != hydrationCoverageUnknown {
			t.Fatalf("withdrawn provider result = %+v state=%+v", terminalDisposition, e.observeHydration())
		}
		fence, _ := NewAggregateIngressFenceInput(terminalDisposition.FenceCommand, AggregateIngressFenceComplete, 2, 1, now)
		admission, completion := e.AdmitAggregateIngressFence(context.Background(), fence)
		if admission != AdmissionAdmitted || completion == nil || awaitHydrationDisposition(t, completion).Code != DispositionAggregateIngressFenceApplied {
			t.Fatalf("historical ambiguity fence admission=%s", admission)
		}
		state := aggregateState(t, e, "AAA")
		if state.historicalConflict == nil || !state.historicalConflict.has(sessionSlot(e.state.binding, start)) ||
			exactAggregateCoverage(state, e.state.binding, start, now) || state.qualification == nil || state.qualification.result.status != qualificationUnresolved {
			t.Fatalf("historical/historical ambiguity did not remain fail-closed: state=%+v", state)
		}
		closeAndWait(t, e)
	})

	t.Run("chunk constructor and FIFO contain caller aliases", func(t *testing.T) {
		now := start.Add(3 * time.Second)
		e, token := plannedHydrationEngine(t, binding, &now)
		rows := []HydrationRow{hydrationRow(t, "AAA", start, 10)}
		chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, rows)
		if err != nil {
			t.Fatal(err)
		}
		rows[0] = hydrationRow(t, "AAA", start, 99)
		admission, completion := e.AdmitHydrationChunk(context.Background(), chunk)
		if admission != AdmissionAdmitted {
			t.Fatalf("chunk admission = %s", admission)
		}
		chunk.rows[0] = hydrationRow(t, "AAA", start, 77)
		if got := awaitHydrationDisposition(t, completion); got.Code != DispositionHydrationChunkApplied {
			t.Fatalf("chunk = %+v", got)
		}
		assertCanonicalClose(t, e, "AAA", start, 10)
		closeAndWait(t, e)
	})
}

func acknowledgedHydrationEngine(t *testing.T, binding reference.Binding, now *time.Time, ackAt time.Time, target lifecycle) *Engine {
	t.Helper()
	e := aggregateEngine(t, binding, now)
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, ackAt, 1, ControlSucceeded))
	admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, ackAt, 2, ControlSucceeded))
	admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateSubscriptionResult, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt, 2, ControlSucceeded))
	if target != lifecycleHydrating && target != lifecycleAwaitingSession {
		e.mu.Lock()
		e.state.lifecycle = target
		e.mu.Unlock()
	}
	return e
}

func plannedHydrationEngine(t *testing.T, binding reference.Binding, now *time.Time) (*Engine, HydrationRequestToken) {
	t.Helper()
	e := acknowledgedHydrationEngine(t, binding, now, *now, lifecycleHydrating)
	got := admitHydrationPlan(t, e, HydrationFreshBootstrap, 1, generousHydrationBudgets())
	if got.Code != DispositionHydrationPlanApplied || len(got.Plan.Requests()) != 1 {
		t.Fatalf("plan = %+v", got)
	}
	return e, got.Plan.Requests()[0]
}

func ackRecoveryEpoch(t *testing.T, e *Engine, binding reference.Binding, epoch, attemptToken, writeToken uint64, ackAt time.Time, now *time.Time) {
	t.Helper()
	*now = ackAt
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionAttempt, epoch, LivePosition{}, ackAt, attemptToken, ControlSucceeded))
	admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateCommandWriteResult, epoch, LivePosition{}, ackAt, writeToken, ControlSucceeded))
	admitConnectionControl(t, e, controlFact(binding.Identity(), AggregateSubscriptionResult, epoch, LivePosition{ConnectionEpoch: epoch, FrameSequence: 1}, ackAt, writeToken, ControlSucceeded))
}

func admitHydrationPolicy(t *testing.T, e *Engine, input HydrationPolicyActionInput) HydrationDisposition {
	t.Helper()
	admission, completion := e.AdmitHydrationPolicyAction(context.Background(), input)
	if admission != AdmissionAdmitted || completion == nil {
		t.Fatalf("policy admission = %s", admission)
	}
	return awaitHydrationDisposition(t, completion)
}

func failedRecoveryGeneration(t *testing.T, binding reference.Binding, start time.Time) (*Engine, HydrationDisposition) {
	t.Helper()
	now := start.Add(2 * time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)
	terminal, _ := NewHydrationTerminalInput(token, token.ResultID(), HydrationCompletedEmpty, HydrationReasonNone, 1, 1, 10, 0, 0, 0)
	initial := admitHydrationTerminal(t, e, terminal)
	now = start.Add(3 * time.Second)
	fence, _ := NewAggregateIngressFenceInput(initial.FenceCommand, AggregateIngressFenceComplete, 1, 1, now)
	_, fenceCompletion := e.AdmitAggregateIngressFence(context.Background(), fence)
	_ = awaitHydrationDisposition(t, fenceCompletion)
	now = start.Add(4 * time.Second)
	admitConnectionControl(t, e, controlFact(binding.Identity(), ConnectionLost, 1, LivePosition{ConnectionEpoch: 1, FrameSequence: 2}, now, 0, ControlFailed))
	ackRecoveryEpoch(t, e, binding, 2, 3, 4, start.Add(6*time.Second), &now)
	plan := admitHydrationPlan(t, e, HydrationGapRecovery, 2, generousHydrationBudgets())
	recoveryToken := plan.Plan.Requests()[0]
	failed, _ := NewHydrationTerminalInput(recoveryToken, recoveryToken.ResultID(), HydrationFailed, HydrationReasonHTTPRetryExhausted, 1, 3, 0, 0, 0, 0)
	return e, admitHydrationTerminal(t, e, failed)
}

func generousHydrationBudgets() HydrationPlanBudgets {
	return HydrationPlanBudgets{Workers: 1, RowsPerChunk: 100, MaximumResponseBytes: 1 << 20, MaximumNormalizedRecords: 100_000, MaximumResidentRecords: 100_000}
}

func admitHydrationPlan(t *testing.T, e *Engine, purpose HydrationPurpose, epoch uint64, budgets HydrationPlanBudgets) HydrationDisposition {
	t.Helper()
	result, completion := e.AdmitHydrationPlan(context.Background(), HydrationPlanInput{SchemaVersion: HydrationPlanSchemaV1, BindingIdentity: e.state.binding.identity, Purpose: purpose, ConnectionEpoch: epoch, Budgets: budgets})
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("plan admission = %s", result)
	}
	return awaitHydrationDisposition(t, completion)
}

func admitHydrationChunk(t *testing.T, e *Engine, input HydrationChunkInput) HydrationDisposition {
	t.Helper()
	result, completion := e.AdmitHydrationChunk(context.Background(), input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("chunk admission = %s", result)
	}
	return awaitHydrationDisposition(t, completion)
}

func admitHydrationTerminal(t *testing.T, e *Engine, input HydrationTerminalInput) HydrationDisposition {
	t.Helper()
	result, completion := e.AdmitHydrationTerminal(context.Background(), input)
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("terminal admission = %s", result)
	}
	return awaitHydrationDisposition(t, completion)
}

func awaitHydrationDisposition(t *testing.T, completion <-chan HydrationDisposition) HydrationDisposition {
	t.Helper()
	select {
	case got := <-completion:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for hydration disposition")
		return HydrationDisposition{}
	}
}

func hydrationRow(t *testing.T, symbol string, at time.Time, close float64) HydrationRow {
	t.Helper()
	row, err := NewHydrationRow(symbol, at.UTC(), at.Add(time.Second).UTC(), AggregateValues{Open: close, High: close, Low: close, Close: close, Volume: 100, VWAP: close, AverageTradeSize: 10, ATSProvenance: ATSRESTFloorVolumeOverTrades})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func hydrationPopulationBinding(t *testing.T, symbols []string) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-08-06")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/reference/tickers":
			records := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				records[index] = map[string]any{"ticker": symbol, "active": true, "market": "stocks", "locale": "us", "type": "CS"}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "count": len(records), "results": records})
		case strings.HasPrefix(request.URL.Path, "/v2/aggs/grouped/locale/us/market/stocks/"):
			rows := make([]map[string]any, len(symbols))
			for index, symbol := range symbols {
				rows[index] = map[string]any{"T": symbol, "c": 10 + index, "t": facts.PriorRegularClose.UnixMilli()}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"status": "OK", "adjusted": true, "resultsCount": len(rows), "results": rows})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	now := facts.SessionStart.Add(time.Hour)
	universe, err := (&reference.Resolver{BaseURL: server.URL, APIKey: "reference-test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: server.URL, APIKey: "reference-test", DataDir: filepath.Join(t.TempDir(), "reference"), HTTPClient: server.Client(), Schedule: schedule, Now: func() time.Time { return now }, Sleep: func(context.Context, time.Duration) error { return nil }}).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil || !slices.Equal(binding.UniverseSymbols(), symbols) {
		t.Fatalf("hydration population binding: %v symbols=%v", err, binding.UniverseSymbols())
	}
	return binding
}
