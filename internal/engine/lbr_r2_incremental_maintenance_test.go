package engine

import (
	"runtime"
	"testing"
	"time"
)

func TestPLBRR2IncrementalTailMaintenance(t *testing.T) {
	start := time.Date(2026, 8, 25, 13, 30, 0, 0, time.UTC)
	binding := &installedBinding{sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), tailIndexBuilt: true, tailIndexValid: true}
	e := &Engine{state: &engineState{binding: binding}}

	recordAt := func(at time.Time, closeValue float64) *canonicalAggregate {
		return &canonicalAggregate{
			identity:    aggregateIdentity{symbol: "AAA", start: at.Unix()},
			windowStart: at,
			windowEnd:   at.Add(time.Second),
			values:      AggregateValues{Open: closeValue, High: closeValue, Low: closeValue, Close: closeValue, Volume: 1, VWAP: closeValue, AverageTradeSize: 1},
			authority:   aggregateEvidence{source: AggregateSourceLive},
		}
	}

	// Insert deliberately out of order. The derived order is bounded and sorted,
	// while the canonical map remains the only owner of aggregate values.
	t0 := start.Add(2 * time.Hour)
	r0, r1, r2 := recordAt(t0, 10), recordAt(t0.Add(time.Second), 11), recordAt(t0.Add(2*time.Second), 12)
	putTailAggregate(state, binding, r2)
	putTailAggregate(state, binding, r0)
	putTailAggregate(state, binding, r1)
	if got := state.tailOrder; len(got) != 3 || got[0] != r0.identity.start || got[1] != r1.identity.start || got[2] != r2.identity.start {
		t.Fatalf("sorted expiry order=%v", got)
	}
	if !exactAggregateCoverage(state, binding, t0, t0.Add(3*time.Second)) {
		t.Fatal("incremental presence did not prove the canonical tail interval")
	}

	// Equality at the correction horizon remains mutable; the first tick after
	// it expires exactly one oldest identity without scanning or rebuilding the
	// other retained identities.
	e.compactSymbolLocked(state, binding, "AAA", r0.windowEnd.Add(correctionHorizon))
	if len(state.tail) != 3 {
		t.Fatalf("strict horizon equality compacted tail len=%d", len(state.tail))
	}
	e.compactSymbolLocked(state, binding, "AAA", r0.windowEnd.Add(correctionHorizon+time.Nanosecond))
	if len(state.tail) != 2 || state.tail[r0.identity.start] != nil || state.presence == nil || !state.presence.has(sessionSlot(binding, r0.windowStart)) {
		t.Fatalf("strict expiry tail=%d oldest=%v presence=%v", len(state.tail), state.tail[r0.identity.start], state.presence)
	}
	if !exactAggregateCoverage(state, binding, t0, t0.Add(3*time.Second)) {
		t.Fatal("folded-prefix plus incremental tail coverage diverged after expiry")
	}

	// Revision preserves the identity index and updates only the canonical value.
	revised := recordAt(r1.windowStart, 21)
	putTailAggregate(state, binding, revised)
	if len(state.tailOrder) != 2 || state.tail[r1.identity.start].values.Close != 21 {
		t.Fatalf("revision duplicated identity order=%v record=%+v", state.tailOrder, state.tail[r1.identity.start])
	}
	deleteTailAggregate(state, binding, r2.identity.start)
	if len(state.tailOrder) != 1 || state.tailPresence.has(sessionSlot(binding, r2.windowStart)) {
		t.Fatalf("withdrawal left derived presence order=%v", state.tailOrder)
	}

	// An unchanged mature symbol performs no allocation. This distinguishes the
	// accepted path from the former map-copy/sort/coverage-rebuild maintenance.
	now := revised.windowEnd.Add(correctionHorizon)
	if allocations := testing.AllocsPerRun(1_000, func() {
		e.compactSymbolLocked(state, binding, "AAA", now)
	}); allocations != 0 {
		t.Fatalf("unchanged incremental maintenance allocations=%v want=0", allocations)
	}
}

func TestPLBRR2TailIndexRebuildRejectsMalformedCanonicalIdentity(t *testing.T) {
	start := time.Date(2026, 8, 25, 13, 30, 0, 0, time.UTC)
	binding := &installedBinding{sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
	record := &canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: start.Unix() + 1}, windowStart: start, windowEnd: start.Add(time.Second)}
	state := &symbolAggregateState{tail: map[int64]*canonicalAggregate{start.Unix(): record}}
	ensureTailIndex(state, binding)
	if state.tailIndexValid || exactAggregateCoverage(state, binding, start, start.Add(time.Second)) {
		t.Fatalf("malformed canonical identity index_valid=%t coverage=true", state.tailIndexValid)
	}
}

func TestPLBRR2StaleSameCardinalityIndexFailsCoverageClosed(t *testing.T) {
	start := time.Date(2026, 8, 25, 13, 30, 0, 0, time.UTC)
	binding := &installedBinding{sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), tailIndexBuilt: true, tailIndexValid: true}
	first := &canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: start.Unix()}, windowStart: start, windowEnd: start.Add(time.Second)}
	secondAt := start.Add(time.Second)
	second := &canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: secondAt.Unix()}, windowStart: secondAt, windowEnd: secondAt.Add(time.Second)}
	putTailAggregate(state, binding, first)
	if !exactAggregateCoverage(state, binding, start, start.Add(time.Second)) {
		t.Fatal("valid indexed identity did not prove coverage")
	}

	// Model the smallest future missed-index mutation: canonical cardinality is
	// unchanged, while order/presence still describe the removed identity. Every
	// canonical-tail mutation must advance tailRevision; the unmatched indexed
	// revision makes the acceleration fail closed instead of proving stale data.
	delete(state.tail, first.identity.start)
	state.tail[second.identity.start] = second
	state.tailRevision++
	if len(state.tail) != len(state.tailOrder) || state.tailPresenceCount != len(state.tail) {
		t.Fatal("counterexample must preserve cardinality")
	}
	if exactAggregateCoverage(state, binding, start, start.Add(time.Second)) {
		t.Fatal("stale same-cardinality index falsely proved removed canonical identity")
	}
}

func TestPLBRR2StaleDerivedBitmapFailsCoverageClosed(t *testing.T) {
	start := time.Date(2026, 8, 25, 13, 30, 0, 0, time.UTC)
	binding := &installedBinding{sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), tailIndexBuilt: true, tailIndexValid: true}
	record := &canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: start.Unix()}, windowStart: start, windowEnd: start.Add(time.Second)}
	putTailAggregate(state, binding, record)

	// Preserve canonical state, revisions, order, and stored cardinality while
	// moving the derived bit to a nonexistent second. The independently updated
	// complement word must detect this exact bitmap-only stale-state class.
	state.tailPresence.clear(sessionSlot(binding, start))
	state.tailPresence.set(sessionSlot(binding, start.Add(time.Second)))
	if state.tailRevision != state.tailIndexRevision || state.tailPresenceCount != len(state.tail) {
		t.Fatal("counterexample must preserve revision and cardinality metadata")
	}
	if exactAggregateCoverage(state, binding, start.Add(time.Second), start.Add(2*time.Second)) {
		t.Fatal("stale derived bitmap falsely proved nonexistent canonical identity")
	}
}

func TestPLBRR2MaturePopulationMaintenanceBound(t *testing.T) {
	if testing.Short() {
		t.Skip("allocated mature-population maintenance measurement")
	}
	const population = 5_554
	start := time.Date(2026, 8, 25, 13, 30, 0, 0, time.UTC)
	target := start.Add(5 * time.Hour)
	binding := &installedBinding{sessionStart: start, sessionEnd: start.Add(16 * time.Hour)}
	sharedOrder := make([]int64, 0, maximumTailRecords)
	records := make([]*canonicalAggregate, 0, maximumTailRecords)
	for offset := -(maximumTailRecords - 1); offset <= 0; offset++ {
		at := target.Add(time.Duration(offset) * time.Second)
		record := &canonicalAggregate{identity: aggregateIdentity{symbol: "AAA", start: at.Unix()}, windowStart: at, windowEnd: at.Add(time.Second)}
		records = append(records, record)
		sharedOrder = append(sharedOrder, record.identity.start)
	}
	states := make([]symbolAggregateState, population)
	for index := range states {
		tail := make(map[int64]*canonicalAggregate, maximumTailRecords)
		presence := &slotBitmap{}
		guard := newTailPresenceGuard()
		for _, record := range records {
			tail[record.identity.start] = record
			slot := sessionSlot(binding, record.windowStart)
			presence.set(slot)
			guard[slot/64] = ^presence[slot/64]
		}
		// Immutable record values are shared by the timing fixture, but every
		// symbol has an independent full-cardinality canonical map, expiry order,
		// and presence bitmap, matching the production memory-access boundary.
		states[index] = symbolAggregateState{
			tail: tail, tailOrder: append([]int64(nil), sharedOrder...), tailPresence: presence, tailPresenceGuard: guard,
			tailPresenceCount: maximumTailRecords, tailIndexBuilt: true, tailIndexValid: true,
		}
	}
	e := &Engine{state: &engineState{binding: binding}}
	now := target.Add(time.Second) // newest record; no identity is old enough to expire.
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	for index := range states {
		e.compactSymbolLocked(&states[index], binding, "AAA", now)
	}
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	if elapsed >= 250*time.Millisecond {
		t.Fatalf("mature %d-symbol unchanged maintenance=%s want<250ms", population, elapsed)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated != 0 {
		t.Fatalf("mature unchanged maintenance allocated_bytes=%d want=0", allocated)
	}
	t.Logf("LBR_R2_MAINTENANCE symbols=%d tail_records_per_symbol=%d elapsed=%s allocated_bytes=0", population, maximumTailRecords, elapsed)
}
