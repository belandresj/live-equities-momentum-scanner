package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	lbrB3ManifestSHA = "3a70dda5bc040d1a77fbe3e114d788d3fc04c93fbb51ff2e5eec646f4c10e6e3"
	lbrB3Population  = 5_694
	lbrB3ValueRows   = 5_470_012
)

// TestPLBRB3Removal is the semantic half of P-LBR-B3-REMOVAL. Source and
// dependency exclusions are checked by the companion bounded shell proof.
func TestPLBRB3Removal(t *testing.T) {
	e, target, rows := generatedLBRB3MatureEngine(t)
	if rows != lbrB3ValueRows || len(e.state.binding.symbols) != lbrB3Population {
		t.Fatalf("manifest population=%d rows=%d", len(e.state.binding.symbols), rows)
	}
	staged := e.stageAggregateEvaluationAtLocked(target, target)
	if validation := validateAggregateEvaluation(staged); validation != nil {
		t.Fatalf("current-product evaluation: %v", validation)
	}
	if staged.mode != rankingQualifiedCurrent || staged.population.universeTotal != lbrB3Population ||
		staged.population.trustedRankableMark != 5_692 || staged.population.noPrintThroughT != 2 ||
		len(staged.rows) != maximumRankingRows || staged.enrichedRows != maximumRankingRows {
		t.Fatalf("mature projection=%+v rows=%d enriched=%d", staged.population, len(staged.rows), staged.enrichedRows)
	}
	for _, row := range staged.rows {
		for name, field := range map[string]aggregateFeatureField{
			"volume": row.sessionVolume, "from_open": row.fromOpenPercent, "day_range": row.dayRange,
			"activity_30s": row.activity30s, "move_30s": row.move30s,
		} {
			if field.status != featureCurrent {
				t.Fatalf("%s %s=%+v", row.symbol, name, field)
			}
		}
	}
}

// TestLiveRetainedTailCostAttribution is the one allocated B3 mature-cycle
// measurement. It validates the frozen manifest file before timing and reports
// replacement latency/allocation beside the frozen historical evaluator fact.
func TestLiveRetainedTailCostAttribution(t *testing.T) {
	if testing.Short() {
		t.Skip("allocated 5,694-symbol mature-cycle measurement")
	}
	manifest, err := os.ReadFile("../../docs/live-backend-replacement/baseline-characterization.md")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(manifest)) != lbrB3ManifestSHA {
		t.Fatalf("frozen manifest mismatch err=%v sha=%x", err, sha256.Sum256(manifest))
	}
	e, target, rows := generatedLBRB3MatureEngine(t)
	if rows != lbrB3ValueRows {
		t.Fatalf("manifest rows=%d", rows)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	var staged aggregateEvaluationResult
	pprof.Do(context.Background(), pprof.Labels("phase", "lbr_b3_mature_cycle"), func(context.Context) {
		staged = e.stageAggregateEvaluationAtLocked(target, target)
		if validation := validateAggregateEvaluation(staged); validation != nil {
			t.Fatalf("evaluation: %v", validation)
		}
		e.applyStagedAggregateCandidateLocked(staged, target)
	})
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	t.Logf("LBR_B3_MATURE manifest_sha=%s symbols=%d hydration_rows=%d cycle=%s allocated_bytes=%d baseline_mean=399.871ms baseline_max=601.020ms baseline_allocated_bytes=unknown",
		lbrB3ManifestSHA, lbrB3Population, rows, elapsed, after.TotalAlloc-before.TotalAlloc)
}

func generatedLBRB3MatureEngine(t *testing.T) (*Engine, time.Time, int) {
	t.Helper()
	start := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	target := time.Date(2026, 8, 12, 21, 15, 0, 0, time.UTC)
	binding := &installedBinding{identity: "lbr-mature-v1:2026-08-12", tradingDate: "2026-08-12", sessionStart: start,
		sessionEnd: time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), symbols: make([]coreSymbol, lbrB3Population), index: make(map[string]int, lbrB3Population)}
	rows := 0
	for i := range binding.symbols {
		symbol := fmt.Sprintf("S%04d", i)
		binding.index[symbol] = i
		binding.symbols[i] = coreSymbol{symbol: symbol, prior: frozenPriorClose{symbol: symbol, status: reference.PriorCloseValid, close: 10}}
		if i >= 2 {
			state, count := lbrB3MatureValueState(binding, symbol, i, target)
			binding.symbols[i].aggregates, rows = state, rows+count
		}
	}
	e := &Engine{mode: RunModeLive, state: &engineState{binding: binding, lifecycle: lifecycleLive, committedT: immutableTime(target), latestTarget: immutableTime(target), clockMonotonic: true}}
	e.state.aggregateEvaluator.coverage = map[int]aggregateCoverageConsequence{0: coverageNoPrintThroughT, 1: coverageNoPrintThroughT}
	return e, target, rows
}

func lbrB3MatureValueState(binding *installedBinding, symbol string, index int, target time.Time) (*symbolAggregateState, int) {
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate), presence: &slotBitmap{}, provenAbsent: &slotBitmap{}}
	for slot := 0; slot < sessionSlot(binding, target); slot++ {
		state.provenAbsent.set(slot)
	}
	for q := -961; q < 0; q++ {
		start := target.Add(time.Duration(q) * time.Second)
		remainder := (index + q) % lbrB3Population
		if remainder < 0 {
			remainder += lbrB3Population
		}
		closeValue := 10 + float64(remainder)/8192
		record := canonicalAggregate{identity: aggregateIdentity{symbol: symbol, start: start.Unix()}, windowStart: start, windowEnd: start.Add(time.Second),
			values: AggregateValues{Open: closeValue - 1.0/8192, High: closeValue + 2.0/8192, Low: closeValue - 2.0/8192, Close: closeValue, Volume: 1500 + float64(index%17), VWAP: closeValue, AverageTradeSize: 10, ATSProvenance: ATSLiveProviderAverage}}
		state.provenAbsent.clear(sessionSlot(binding, start))
		state.presence.set(sessionSlot(binding, start))
		foldPriceRangeAggregate(state, binding, record)
		foldMVPMeasurementAggregate(state, record)
		state.prefix.fold(record)
		copyRecord := record
		state.olderLatest = &copyRecord
	}
	state.latest = &latestAggregateMark{record: *state.olderLatest}
	state.selectionAt, state.committedLatest = target, committedMark(*state.olderLatest)
	state.qualification = &qualificationState{finalizedGateBars: map[int64]qualificationGateBar{}, proofs: map[int64]struct{}{}, dirty: map[int64]struct{}{},
		accountedThrough: target, finalized: true, finalProofEnd: target.Add(-time.Minute), result: qualificationResult{at: target, status: qualificationFinalized, finalProofEnd: target.Add(-time.Minute)}}
	state.priceRange.result = priceRangeFeatureResult{at: target, fromOpenPercent: aggregateFeatureField{status: featureCurrent}, dayRange: aggregateFeatureField{status: featureCurrent}}
	state.mvpMeasurements.result = mvpMeasurementResult{at: target, sessionVolume: aggregateFeatureField{status: featureCurrent}, activity30s: aggregateFeatureField{status: featureCurrent}, move30s: aggregateFeatureField{status: featureCurrent}}
	rebuildTailCoverage(state, binding)
	return state, 961
}

func currentField(value float64) aggregateFeatureField {
	return aggregateFeatureField{status: featureCurrent, value: value}
}

func proveAggregateCoverage(t *testing.T, e *Engine, symbol string, start, end time.Time) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	index := e.state.binding.index[symbol]
	state := ensureAggregateState(&e.state.binding.symbols[index])
	if !installExactCoverage(state, e.state.binding, start, end, nil) {
		t.Fatalf("could not install exact coverage %s [%s,%s)", symbol, start, end)
	}
}
