package engine

import (
	"context"
	"math"
	"testing"
	"time"
)

// TestC3QUAL01ExactGateBoundaryMatrix is the sole C3-QUAL-01 primary proof.
// It makes the inclusive thresholds, half-open slots, edge gaps, ATS-local
// availability, finite arithmetic, canonical identity, and provenance
// observable. Provider ATS parity and production latency are intentionally not
// proved.
func TestC3QUAL01ExactGateBoundaryMatrix(t *testing.T) {
	baseline := qualificationGateFacts{
		latestPrice: 0.25, presentSeconds: 45, longestGap: 3,
		a60: 1000, a5: 100, dollarVolume60: 250000, concentration: 0.50,
		a60Available: true, a5Available: true, valid: true,
	}
	if !passesQualificationGate(baseline) {
		t.Fatal("inclusive qualification boundary did not pass")
	}
	for _, test := range []struct {
		name   string
		change func(*qualificationGateFacts)
	}{
		{"price", func(f *qualificationGateFacts) { f.latestPrice = math.Nextafter(.25, 0) }},
		{"present seconds", func(f *qualificationGateFacts) { f.presentSeconds = 44 }},
		{"longest gap", func(f *qualificationGateFacts) { f.longestGap = 4 }},
		{"A60 unavailable", func(f *qualificationGateFacts) { f.a60Available = false }},
		{"A60", func(f *qualificationGateFacts) { f.a60 = math.Nextafter(1000, 0) }},
		{"dollar volume", func(f *qualificationGateFacts) { f.dollarVolume60 = math.Nextafter(250000, 0) }},
		{"A5 unavailable", func(f *qualificationGateFacts) { f.a5Available = false }},
		{"A5", func(f *qualificationGateFacts) { f.a5 = math.Nextafter(100, 0) }},
		{"concentration", func(f *qualificationGateFacts) { f.concentration = math.Nextafter(.5, 1) }},
		{"invalid arithmetic", func(f *qualificationGateFacts) { f.valid = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := baseline
			test.change(&facts)
			if passesQualificationGate(facts) {
				t.Fatal("adjacent failing boundary passed")
			}
		})
	}

	binding := testBinding(t)
	start := binding.SessionStart()
	proofEnd := start.Add(60 * time.Second)
	bindingState := installedBindingForQualification(t, binding)
	newState := func(present func(int) bool) *symbolAggregateState {
		state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate)}
		for second := 0; second < 60; second++ {
			if !present(second) {
				continue
			}
			window := start.Add(time.Duration(second) * time.Second)
			provenance := ATSLiveProviderAverage
			if second%2 != 0 {
				provenance = ATSRESTFloorVolumeOverTrades
			}
			state.tail[window.Unix()] = qualificationRecord("AAA", window, 10, 500, 10, 25, provenance)
		}
		installExactCoverage(state, bindingState, start, proofEnd)
		return state
	}
	for _, test := range []struct {
		name    string
		present func(int) bool
		wantGap int
	}{
		{"leading", func(second int) bool { return second >= 3 }, 3},
		{"interior", func(second int) bool { return second < 20 || second > 22 }, 3},
		{"trailing", func(second int) bool { return second < 57 }, 3},
		{"adjacent failure", func(second int) bool { return second >= 4 }, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := calculateQualificationGateFacts(newState(test.present), installedBindingForQualification(t, binding), proofEnd)
			if facts.longestGap != test.wantGap {
				t.Fatalf("longest gap = %d, want %d", facts.longestGap, test.wantGap)
			}
		})
	}

	state := newState(func(int) bool { return true })
	facts := calculateQualificationGateFacts(state, bindingState, proofEnd)
	if !passesQualificationGate(facts) || !facts.liveATS || !facts.historicalATS || facts.presentSeconds != 60 {
		t.Fatalf("mixed-provenance canonical gate = %+v", facts)
	}
	if facts.latestPrice != 10 || facts.a60 != 1200 || facts.a5 != 100 || facts.dollarVolume60 != 300000 || facts.concentration != 1.0/60.0 {
		t.Fatalf("exact gate arithmetic = %+v", facts)
	}
	state.tail[proofEnd.Unix()] = qualificationRecord("AAA", proofEnd, 100, 50000, 100, 1, ATSLiveProviderAverage)
	if got := calculateQualificationGateFacts(state, bindingState, proofEnd); got != facts {
		t.Fatalf("bar beginning at proof end entered half-open gate: before=%+v after=%+v", facts, got)
	}
	// One identity remains one second after its canonical value is replaced.
	replaced := qualificationRecord("AAA", start.Add(30*time.Second), 10, 500, 10, 25, ATSLiveProviderAverage)
	replaced.values.Volume = 1000
	state.tail[replaced.identity.start] = replaced
	if got := calculateQualificationGateFacts(state, bindingState, proofEnd).presentSeconds; got != 60 {
		t.Fatalf("canonical revision counted as another second: %d", got)
	}

	outsideFive := state.tail[start.Add(54*time.Second).Unix()]
	outsideFive.values.AverageTradeSize = 0
	facts = calculateQualificationGateFacts(state, bindingState, proofEnd)
	if facts.a60Available || !facts.a5Available {
		t.Fatalf("ATS outside A5 contaminated wrong windows: %+v", facts)
	}
	insideFive := state.tail[start.Add(58*time.Second).Unix()]
	insideFive.values.AverageTradeSize = 0
	facts = calculateQualificationGateFacts(state, bindingState, proofEnd)
	if facts.a60Available || facts.a5Available {
		t.Fatalf("ATS inside A5 availability = %+v", facts)
	}

	state = newState(func(int) bool { return true })
	overflow := state.tail[start.Unix()]
	overflow.values.Volume, overflow.values.VWAP = math.MaxFloat64, math.MaxFloat64
	facts = calculateQualificationGateFacts(state, bindingState, proofEnd)
	if facts.valid || passesQualificationGate(facts) {
		t.Fatalf("nonfinite intermediate reached success: %+v", facts)
	}
}

// TestC3QUAL02CorrectionAndStrictFinalizationTrace is the sole C3-QUAL-02
// primary proof. It observes provisional multiproofs, targeted revocation,
// equality remaining mutable, strict finalization, quiet-tape permanence,
// too-late containment, and the 961 proof/dirty construction bound. It does
// not prove a Component 7 checkpoint codec or live correction distribution.
func TestC3QUAL02CorrectionAndStrictFinalizationTrace(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	proofEnd := start.Add(60 * time.Second)

	t.Run("multiproof correction revokes only affected proofs", func(t *testing.T) {
		now := proofEnd.Add(time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		for second := 0; second < 60; second++ {
			input := qualificationInput(binding, start.Add(time.Duration(second)*time.Second), uint64(second+1), 500)
			applyAggregate(t, e, input, DispositionAggregateInserted, ReasonNone)
		}
		proveAggregateCoverage(t, e, "AAA", start, proofEnd)
		setCommittedQualificationTime(e, proofEnd)
		applyQualificationTimer(t, e)
		qualification := qualificationForSymbol(t, e, "AAA")
		if qualification.result.status != qualificationProvisional || len(qualification.proofs) != 4 {
			t.Fatalf("initial multiproof state = %+v proofs=%v", qualification.result, qualification.proofs)
		}

		for correction := 0; correction < 4; correction++ {
			window := proofEnd.Add(-time.Duration(correction+1) * time.Second)
			input := qualificationInput(binding, window, uint64(61+correction), 500)
			input.Values.AverageTradeSize = 0
			applyAggregate(t, e, input, DispositionAggregateRevised, ReasonNone)
			qualification = qualificationForSymbol(t, e, "AAA")
			want := 3 - correction
			if len(qualification.proofs) != want || qualification.currentDirtyCount() != 0 {
				t.Fatalf("correction %d proofs=%v dirty=%d, want %d/0", correction, qualification.proofs, qualification.currentDirtyCount(), want)
			}
		}
		if qualification.result.status != qualificationNotYetPassed || qualification.revoked != 4 || qualification.revalidated < 4 {
			t.Fatalf("revoked result = %+v state=%+v", qualification.result, qualification)
		}
		closeAndWait(t, e)
	})

	t.Run("equality mutable strict old permanent and quiet", func(t *testing.T) {
		now := proofEnd.Add(time.Second)
		e := aggregateEngine(t, binding, RunModeLive, &now)
		for second := 0; second < 60; second++ {
			volume := 497.5 // 19.9 transactions: prior proof ends fail exact A5.
			if second == 59 {
				volume = 510 // 20.4 transactions makes A5 exactly 100 only at P.
			}
			applyAggregate(t, e, qualificationInput(binding, start.Add(time.Duration(second)*time.Second), uint64(second+1), volume), DispositionAggregateInserted, ReasonNone)
		}
		proveAggregateCoverage(t, e, "AAA", start, proofEnd)
		setCommittedQualificationTime(e, proofEnd)
		applyQualificationTimer(t, e)
		qualification := qualificationForSymbol(t, e, "AAA")
		if qualification.result.status != qualificationProvisional || len(qualification.proofs) != 1 {
			t.Fatalf("sole proof = %+v proofs=%v", qualification.result, qualification.proofs)
		}

		now = proofEnd.Add(correctionHorizon)
		applyQualificationTimer(t, e)
		qualification = qualificationForSymbol(t, e, "AAA")
		if qualification.finalized || len(qualification.proofs) != 1 {
			t.Fatalf("proof finalized at equality: %+v", qualification)
		}

		now = now.Add(time.Nanosecond)
		applyQualificationTimer(t, e)
		qualification = qualificationForSymbol(t, e, "AAA")
		if qualification.result.status != qualificationFinalized || !qualification.finalized || len(qualification.proofs) != 0 || !qualification.finalProofEnd.Equal(proofEnd) {
			t.Fatalf("strict finalization = %+v", qualification)
		}
		if len(qualification.finalizedGateBars) > maximumFinalizedGateBars {
			t.Fatalf("finalized gate overlap retained %d bars", len(qualification.finalizedGateBars))
		}

		now = now.Add(time.Minute)
		applyQualificationTimer(t, e)
		late := qualificationInput(binding, proofEnd.Add(-time.Second), 100, 50000)
		if got := applyAggregate(t, e, late, DispositionAggregateRejected, ReasonTooLate); got.Code != DispositionAggregateRejected {
			t.Fatalf("late correction = %+v", got)
		}
		qualification = qualificationForSymbol(t, e, "AAA")
		if qualification.result.status != qualificationFinalized || !qualification.finalProofEnd.Equal(proofEnd) {
			t.Fatalf("quiet/late input revoked final latch: %+v", qualification)
		}
		closeAndWait(t, e)
	})

	t.Run("proof and dirty bounds", func(t *testing.T) {
		qualification := ensureQualificationState(&symbolAggregateState{})
		for index := 0; index < maximumQualificationProofs; index++ {
			installQualificationProof(qualification, start.Add(time.Duration(index)*time.Second).Unix())
		}
		markQualificationProofsDirty(qualification, start, start.Add(time.Duration(maximumQualificationProofs-1)*time.Second))
		if len(qualification.proofs) != maximumQualificationProofs || len(qualification.dirty) != maximumQualificationProofs ||
			qualification.maximumProofOccupancy != maximumQualificationProofs || qualification.maximumDirtyOccupancy != maximumQualificationProofs {
			t.Fatalf("961 bounds = proofs:%d dirty:%d maxima:%d/%d", len(qualification.proofs), len(qualification.dirty), qualification.maximumProofOccupancy, qualification.maximumDirtyOccupancy)
		}
		installQualificationProof(qualification, start.Add(maximumQualificationProofs*time.Second).Unix())
		if !qualification.boundExceeded || len(qualification.proofs) != 0 || len(qualification.dirty) != 0 {
			t.Fatalf("overflow did not fail closed: %+v", qualification)
		}
	})
}

func qualificationInput(binding interface {
	Identity() string
}, window time.Time, frame uint64, volume float64) AggregateInput {
	// Tests use the concrete binding only through its stable identity; the
	// remaining normalized fields are independent of Component 1 internals.
	return AggregateInput{
		SchemaVersion: AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: AggregateSourceLive,
		Symbol: "AAA", WindowStart: window, WindowEnd: window.Add(time.Second), DeliveryTime: window.Add(time.Second),
		Values: AggregateValues{Open: 10, High: 10, Low: 10, Close: 10, Volume: volume, VWAP: 10, AverageTradeSize: 25, ATSProvenance: ATSLiveProviderAverage},
		Live:   LivePosition{ConnectionEpoch: 1, FrameSequence: frame},
	}
}

func qualificationRecord(symbol string, window time.Time, close, volume, vwap float64, ats int64, provenance ATSProvenance) *canonicalAggregate {
	return &canonicalAggregate{
		identity: aggregateIdentity{symbol: symbol, start: window.Unix()}, windowStart: window, windowEnd: window.Add(time.Second),
		values: AggregateValues{Open: close, High: close, Low: close, Close: close, Volume: volume, VWAP: vwap, AverageTradeSize: ats, ATSProvenance: provenance},
	}
}

func installedBindingForQualification(t *testing.T, binding interface{ Identity() string }) *installedBinding {
	t.Helper()
	// Reuse the ordinary engine constructor so this proof does not construct a
	// competing binding representation.
	concrete := testBinding(t)
	if concrete.Identity() != binding.Identity() {
		t.Fatal("test binding identity changed")
	}
	now := concrete.SessionStart()
	e := aggregateEngine(t, concrete, RunModeLive, &now)
	result := e.state.binding
	closeAndWait(t, e)
	return result
}

func setCommittedQualificationTime(e *Engine, at time.Time) {
	e.mu.Lock()
	e.state.committedT = immutableTime(at)
	e.mu.Unlock()
}

func applyQualificationTimer(t *testing.T, e *Engine) {
	t.Helper()
	e.mu.Lock()
	if e.state.committedT != nil {
		now := e.clock().UTC()
		e.delay = now.Sub(*e.state.committedT)
	}
	e.mu.Unlock()
	result, completion := e.AdmitTimer(context.Background())
	if result != AdmissionAdmitted {
		t.Fatalf("timer admission = %s", result)
	}
	if got := awaitTimerDisposition(t, completion); got.Code != DispositionTimerApplied {
		t.Fatalf("timer disposition = %+v", got)
	}
}

func qualificationForSymbol(t *testing.T, e *Engine, symbol string) *qualificationState {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.state.binding.symbols[e.state.binding.index[symbol]].aggregates
	if state == nil || state.qualification == nil {
		t.Fatalf("qualification state missing for %s", symbol)
	}
	return state.qualification
}

func (qualification *qualificationState) currentDirtyCount() int { return len(qualification.dirty) }
