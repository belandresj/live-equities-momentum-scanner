package engine

import (
	"runtime"
	"testing"
	"time"
)

func TestLatestMarkBeforeMaintainedLatestFastPath(t *testing.T) {
	start := time.Date(2026, 8, 12, 13, 0, 0, 0, time.UTC)
	record := func(offset time.Duration, closeValue float64) *canonicalAggregate {
		window := start.Add(offset)
		return &canonicalAggregate{
			identity:    aggregateIdentity{symbol: "AAA", start: window.Unix()},
			windowStart: window, windowEnd: window.Add(time.Second),
			values: AggregateValues{Open: closeValue, High: closeValue, Low: closeValue, Close: closeValue},
		}
	}
	old, selected, boundary, future := record(0, 10), record(time.Second, 11), record(2*time.Second, 12), record(3*time.Second, 13)

	t.Run("maintained latest strictly before at", func(t *testing.T) {
		state := &symbolAggregateState{
			tail:            map[int64]*canonicalAggregate{old.identity.start: old, selected.identity.start: selected},
			latest:          &latestAggregateMark{record: *selected},
			committedLatest: committedMark(*old),
		}
		got, ok := latestMarkBeforeCompact(state, boundary.windowStart)
		if !ok || got != *selected {
			t.Fatalf("latest=%+v available=%t want=%+v", got, ok, *selected)
		}
	})

	t.Run("latest equal to at remains excluded", func(t *testing.T) {
		state := &symbolAggregateState{
			tail:   map[int64]*canonicalAggregate{selected.identity.start: selected, boundary.identity.start: boundary},
			latest: &latestAggregateMark{record: *boundary},
		}
		recomputeLatest(state)
		advanceSelectionMark(state, nil, boundary.windowStart)
		got, ok := latestMarkBeforeCompact(state, boundary.windowStart)
		if !ok || got != *selected {
			t.Fatalf("latest=%+v available=%t want prior=%+v", got, ok, *selected)
		}
	})

	t.Run("latest after at retains as-of scan", func(t *testing.T) {
		state := &symbolAggregateState{
			tail:   map[int64]*canonicalAggregate{selected.identity.start: selected, future.identity.start: future},
			latest: &latestAggregateMark{record: *future},
		}
		recomputeLatest(state)
		advanceSelectionMark(state, nil, boundary.windowStart)
		got, ok := latestMarkBeforeCompact(state, boundary.windowStart)
		if !ok || got != *selected {
			t.Fatalf("latest=%+v available=%t want as-of=%+v", got, ok, *selected)
		}
	})

	t.Run("folded older latest remains eligible when global latest is future", func(t *testing.T) {
		state := &symbolAggregateState{
			tail:   map[int64]*canonicalAggregate{future.identity.start: future},
			latest: &latestAggregateMark{record: *future}, olderLatest: selected,
			committedLatest: committedMark(*old),
		}
		got, ok := latestMarkBeforeCompact(state, boundary.windowStart)
		if !ok || got != *selected {
			t.Fatalf("latest=%+v available=%t want folded=%+v", got, ok, *selected)
		}
	})
}

func TestExactAggregateCoverageSparseTailEquivalence(t *testing.T) {
	binding := installedBindingForQualification(t, testBinding(t))
	start := binding.sessionStart.Add(61 * time.Second)
	end := start.Add(130 * time.Second)
	record := func(at time.Time) *canonicalAggregate {
		return qualificationRecord("AAA", at, 10, 100, 10, 10, ATSLiveProviderAverage)
	}
	fullState := func() *symbolAggregateState {
		state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), presence: &slotBitmap{}, provenAbsent: &slotBitmap{}}
		for second := start; second.Before(end); second = second.Add(time.Second) {
			switch sessionSlot(binding, second) % 3 {
			case 0:
				state.presence.set(sessionSlot(binding, second))
			case 1:
				state.provenAbsent.set(sessionSlot(binding, second))
			default:
				state.tail[second.Unix()] = record(second)
			}
		}
		return state
	}

	tests := []struct {
		name   string
		mutate func(*symbolAggregateState)
		want   bool
	}{
		{name: "mixed folded absent and tail across words", want: true},
		{name: "missing uncovered second", mutate: func(state *symbolAggregateState) {
			second := firstTailSecond(state, start, end)
			delete(state.tail, second.Unix())
		}, want: false},
		{name: "conflict overrides canonical tail", mutate: func(state *symbolAggregateState) {
			second := firstTailSecond(state, start, end)
			ensureHistoricalConflict(state).set(sessionSlot(binding, second))
		}, want: false},
		{name: "conflict overrides folded presence", mutate: func(state *symbolAggregateState) {
			second := firstBitmapSecond(state.presence, binding, start, end)
			ensureHistoricalConflict(state).set(sessionSlot(binding, second))
		}, want: false},
		{name: "conflict overrides proven absence", mutate: func(state *symbolAggregateState) {
			second := firstBitmapSecond(state.provenAbsent, binding, start, end)
			ensureHistoricalConflict(state).set(sessionSlot(binding, second))
		}, want: false},
		{name: "nil tail record cannot prove coverage", mutate: func(state *symbolAggregateState) {
			second := firstTailSecond(state, start, end)
			state.tail[second.Unix()] = nil
		}, want: false},
		{name: "wrong record identity cannot prove coverage", mutate: func(state *symbolAggregateState) {
			second := firstTailSecond(state, start, end)
			state.tail[second.Unix()].identity.start++
		}, want: false},
		{name: "wrong record window cannot prove coverage", mutate: func(state *symbolAggregateState) {
			second := firstTailSecond(state, start, end)
			state.tail[second.Unix()].windowStart = second.Add(time.Second)
		}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := fullState()
			if tc.mutate != nil {
				tc.mutate(state)
			}
			if got := exactAggregateCoverage(state, binding, start, end); got != tc.want {
				t.Fatalf("coverage=%t want=%t", got, tc.want)
			}
		})
	}
	if !exactAggregateCoverage(fullState(), binding, start, start) {
		t.Fatal("valid empty interval was not exactly covered")
	}
}

func TestExactAggregateCoverageSparseTailZeroAllocation(t *testing.T) {
	binding := installedBindingForQualification(t, testBinding(t))
	start, end := binding.sessionStart, binding.sessionStart.Add(time.Hour)
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), provenAbsent: &slotBitmap{}}
	for second := start; second.Before(end); second = second.Add(time.Second) {
		state.provenAbsent.set(sessionSlot(binding, second))
	}
	// Keep one canonical lookup in every word so the allocation proof exercises
	// both the bitmap and sparse-tail branches.
	for second := start; second.Before(end); second = second.Add(64 * time.Second) {
		state.provenAbsent.clear(sessionSlot(binding, second))
		state.tail[second.Unix()] = qualificationRecord("AAA", second, 10, 100, 10, 10, ATSLiveProviderAverage)
	}
	if allocations := testing.AllocsPerRun(1_000, func() {
		if !exactAggregateCoverage(state, binding, start, end) {
			panic("coverage changed during allocation proof")
		}
	}); allocations != 0 {
		t.Fatalf("allocations/run=%f want=0 runtime=%s", allocations, runtime.Version())
	}
}

func TestEvaluationTailCoverageRejectsCardinalityDrift(t *testing.T) {
	binding := installedBindingForQualification(t, testBinding(t))
	start, end := binding.sessionStart, binding.sessionStart.Add(2*time.Second)
	first := qualificationRecord("AAA", start, 10, 100, 10, 10, ATSLiveProviderAverage)
	second := qualificationRecord("AAA", start.Add(time.Second), 10, 100, 10, 10, ATSLiveProviderAverage)
	state := &symbolAggregateState{
		tail:   map[int64]*canonicalAggregate{first.identity.start: first, second.identity.start: second},
		latest: &latestAggregateMark{record: *second},
	}
	rebuildTailCoverage(state, binding)
	projection := *state
	projection.evaluationTailPresence = &state.tailCoverage
	if !exactAggregateCoverage(&projection, binding, start, end) {
		t.Fatal("fresh derived coverage did not match canonical tail")
	}
	delete(state.tail, first.identity.start)
	projection = *state
	projection.evaluationTailPresence = &state.tailCoverage
	if exactAggregateCoverage(&projection, binding, start, end) {
		t.Fatal("stale derived coverage falsely proved a deleted canonical identity")
	}
}

func TestHydrationDirectCompactionRefreshesTailCoverage(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(correctionHorizon + 2*time.Second)
	e, token := plannedHydrationEngine(t, binding, &now)

	row := hydrationRow(t, "AAA", start, 10)
	chunk, err := NewHydrationChunkInput(token, token.ResultID(), 0, 1, 0, 1, []HydrationRow{row})
	if err != nil {
		closeAndWait(t, e)
		t.Fatal(err)
	}
	if got := admitHydrationChunk(t, e, chunk); got.Code != DispositionHydrationChunkApplied {
		closeAndWait(t, e)
		t.Fatalf("chunk=%+v", got)
	}

	e.mu.Lock()
	state := e.state.binding.symbols[e.state.binding.index["AAA"]].aggregates
	if state == nil || len(state.tail) != 0 || state.presence == nil || !state.presence.has(sessionSlot(e.state.binding, start)) {
		e.mu.Unlock()
		closeAndWait(t, e)
		t.Fatalf("compacted state=%+v", state)
	}
	projection := *state
	if !state.tailIndexBuilt || !state.tailIndexValid || len(state.tailOrder) != 0 {
		e.mu.Unlock()
		closeAndWait(t, e)
		t.Fatalf("incremental tail index built=%t valid=%t order=%v", state.tailIndexBuilt, state.tailIndexValid, state.tailOrder)
	}
	if !exactAggregateCoverage(&projection, e.state.binding, start, start.Add(time.Second)) {
		e.mu.Unlock()
		closeAndWait(t, e)
		t.Fatal("direct hydration compaction left a stale negative tail view")
	}
	e.mu.Unlock()
	closeAndWait(t, e)
}

func firstTailSecond(state *symbolAggregateState, start, end time.Time) time.Time {
	for second := start; second.Before(end); second = second.Add(time.Second) {
		if state.tail[second.Unix()] != nil {
			return second
		}
	}
	return time.Time{}
}

func firstBitmapSecond(bitmap *slotBitmap, binding *installedBinding, start, end time.Time) time.Time {
	for second := start; second.Before(end); second = second.Add(time.Second) {
		if bitmap.has(sessionSlot(binding, second)) {
			return second
		}
	}
	return time.Time{}
}
