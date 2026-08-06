package engine

import (
	"time"
)

const (
	maximumQualificationProofs        = 961
	maximumFinalizedGateBars          = sessionSeconds + 1
	qualificationWindow               = 60 * time.Second
	qualificationShortWindow          = 5 * time.Second
	minimumQualificationPrice         = 0.25
	minimumQualificationSeconds       = 45
	maximumQualificationGap           = 3
	minimumQualificationA60           = 1000
	minimumQualificationA5            = 100
	minimumQualificationDV60          = 250000
	maximumQualificationConcentration = 0.50
)

type qualificationStatus string

const (
	qualificationUnresolved   qualificationStatus = "unresolved"
	qualificationNotYetPassed qualificationStatus = "not_yet_passed"
	qualificationProvisional  qualificationStatus = "provisional"
	qualificationFinalized    qualificationStatus = "finalized"
)

type qualificationResult struct {
	at                time.Time
	status            qualificationStatus
	unresolvedOrigin  uncertaintyOrigin
	currentProofCount int
	finalProofEnd     time.Time
}

type qualificationGateFacts struct {
	latestPrice                float64
	presentSeconds, longestGap int
	a60, a5, dollarVolume60    float64
	concentration              float64
	a60Available, a5Available  bool
	valid                      bool
	liveATS, historicalATS     bool
}

// qualificationGateBar is the compact arithmetic needed only where a
// still-mutable 60-second gate overlaps Component 2's strictly finalized tail.
// It is not a second canonical aggregate or session-long history.
type qualificationGateBar struct {
	close, volume, vwap float64
	averageTradeSize    int64
	provenance          ATSProvenance
}

// qualificationState is the one bounded C3-S3 substate in canonical symbol
// ownership. Proof and dirty identities are whole-second window ends.
type qualificationState struct {
	finalizedGateBars                               map[int64]qualificationGateBar
	proofs                                          map[int64]struct{}
	dirty                                           map[int64]struct{}
	accountedThrough                                time.Time
	finalized                                       bool
	finalProofEnd                                   time.Time
	boundExceeded                                   bool
	invalid                                         bool
	unresolvedOrigin                                uncertaintyOrigin
	installed, revalidated, revoked, finalizedCount uint64
	maximumProofOccupancy, maximumDirtyOccupancy    int
	result                                          qualificationResult
}

func ensureQualificationState(state *symbolAggregateState) *qualificationState {
	if state.qualification == nil {
		state.qualification = &qualificationState{
			finalizedGateBars: make(map[int64]qualificationGateBar),
			proofs:            make(map[int64]struct{}),
			dirty:             make(map[int64]struct{}),
			unresolvedOrigin:  uncertaintyBootstrapOrigin,
			result:            qualificationResult{status: qualificationUnresolved, unresolvedOrigin: uncertaintyBootstrapOrigin},
		}
	}
	return state.qualification
}

func foldQualificationAggregate(state *symbolAggregateState, _ *installedBinding, record canonicalAggregate, _ time.Time) {
	qualification := ensureQualificationState(state)
	if qualification.boundExceeded || qualification.invalid {
		return
	}
	qualification.finalizedGateBars[record.identity.start] = qualificationGateBar{
		close: record.values.Close, volume: record.values.Volume, vwap: record.values.VWAP,
		averageTradeSize: record.values.AverageTradeSize, provenance: record.values.ATSProvenance,
	}
	if len(qualification.finalizedGateBars) > maximumFinalizedGateBars {
		failQualificationState(qualification, true)
	}
}

func pruneFinalizedGateBars(qualification *qualificationState, now time.Time) {
	if !qualification.finalized {
		return
	}
	cutoff := now.Add(-correctionHorizon - qualificationWindow).Unix()
	for start := range qualification.finalizedGateBars {
		if start < cutoff {
			delete(qualification.finalizedGateBars, start)
		}
	}
	if len(qualification.finalizedGateBars) > maximumFinalizedGateBars {
		failQualificationState(qualification, true)
	}
}

func maintainQualificationState(state *symbolAggregateState, binding *installedBinding, engineTime time.Time) {
	qualification := ensureQualificationState(state)
	if !validQualificationState(qualification, binding) {
		failQualificationState(qualification, false)
		updateQualificationResult(state, binding, qualification.result.at)
		return
	}
	pruneFinalizedGateBars(qualification, engineTime)
	revalidateDirtyQualificationProofs(state, binding)
	finalizeQualificationProof(state, engineTime)
	updateQualificationResult(state, binding, qualification.result.at)
}

func recomputeQualificationForAggregate(state *symbolAggregateState, binding *installedBinding, windowStart, at, engineTime time.Time) {
	qualification := ensureQualificationState(state)
	if !validQualificationState(qualification, binding) {
		failQualificationState(qualification, false)
	}
	if qualification.finalized || qualification.boundExceeded || qualification.invalid {
		updateQualificationResult(state, binding, at)
		return
	}
	first := windowStart.Add(time.Second)
	last := windowStart.Add(qualificationWindow)
	if first.Before(binding.sessionStart) {
		first = binding.sessionStart
	}
	if last.After(at) {
		last = at
	}
	markQualificationProofsDirty(qualification, first, last)
	revalidateDirtyQualificationProofs(state, binding)
	for proofEnd := first; !proofEnd.After(last) && !qualification.finalized; proofEnd = proofEnd.Add(time.Second) {
		evaluateQualificationProof(state, binding, proofEnd, engineTime)
	}
	if qualification.accountedThrough.IsZero() || qualification.accountedThrough.Before(at) {
		evaluateQualificationThrough(state, binding, at, engineTime)
	} else {
		finalizeQualificationProof(state, engineTime)
		updateQualificationResult(state, binding, at)
	}
}

func evaluateQualificationThrough(state *symbolAggregateState, binding *installedBinding, at, engineTime time.Time) {
	qualification := ensureQualificationState(state)
	if !validQualificationState(qualification, binding) {
		failQualificationState(qualification, false)
	}
	if qualification.finalized || qualification.boundExceeded || qualification.invalid || at.Before(binding.sessionStart) || at.After(binding.sessionEnd) || at.After(engineTime) {
		updateQualificationResult(state, binding, at)
		return
	}
	revalidateDirtyQualificationProofs(state, binding)
	first := binding.sessionStart
	if !qualification.accountedThrough.IsZero() {
		first = qualification.accountedThrough.Add(time.Second)
	}
	for proofEnd := first; !proofEnd.After(at) && !qualification.finalized; proofEnd = proofEnd.Add(time.Second) {
		evaluateQualificationProof(state, binding, proofEnd, engineTime)
		qualification.accountedThrough = proofEnd
	}
	finalizeQualificationProof(state, engineTime)
	updateQualificationResult(state, binding, at)
}

func evaluateQualificationProof(state *symbolAggregateState, binding *installedBinding, proofEnd, engineTime time.Time) {
	qualification := ensureQualificationState(state)
	if !qualificationCoverageTrustworthy(state, binding, proofEnd.Add(-qualificationWindow), proofEnd) ||
		!passesQualificationGate(calculateQualificationGateFacts(state, binding, proofEnd)) {
		return
	}
	if engineTime.After(proofEnd.Add(correctionHorizon)) {
		qualification.finalized = true
		qualification.finalProofEnd = proofEnd
		qualification.finalizedCount++
		clear(qualification.proofs)
		clear(qualification.dirty)
		clear(qualification.finalizedGateBars)
		return
	}
	installQualificationProof(qualification, proofEnd.Unix())
}

func installQualificationProof(qualification *qualificationState, proof int64) {
	if qualification.finalized {
		return
	}
	if _, exists := qualification.proofs[proof]; exists {
		return
	}
	if len(qualification.proofs) >= maximumQualificationProofs {
		failQualificationState(qualification, true)
		return
	}
	qualification.proofs[proof] = struct{}{}
	qualification.installed++
	qualification.maximumProofOccupancy = max(qualification.maximumProofOccupancy, len(qualification.proofs))
}

func markQualificationProofsDirty(qualification *qualificationState, first, last time.Time) {
	if qualification.finalized || first.After(last) {
		return
	}
	for proof := range qualification.proofs {
		if proof < first.Unix() || proof > last.Unix() {
			continue
		}
		if len(qualification.dirty) >= maximumQualificationProofs {
			failQualificationState(qualification, true)
			return
		}
		qualification.dirty[proof] = struct{}{}
	}
	qualification.maximumDirtyOccupancy = max(qualification.maximumDirtyOccupancy, len(qualification.dirty))
}

func revalidateDirtyQualificationProofs(state *symbolAggregateState, binding *installedBinding) {
	qualification := ensureQualificationState(state)
	if qualification.finalized || qualification.boundExceeded || len(qualification.dirty) == 0 {
		return
	}
	for proof := range qualification.dirty {
		if _, exists := qualification.proofs[proof]; !exists {
			continue
		}
		qualification.revalidated++
		proofEnd := time.Unix(proof, 0).UTC()
		if !qualificationCoverageTrustworthy(state, binding, proofEnd.Add(-qualificationWindow), proofEnd) ||
			!passesQualificationGate(calculateQualificationGateFacts(state, binding, proofEnd)) {
			delete(qualification.proofs, proof)
			qualification.revoked++
		}
	}
	clear(qualification.dirty)
}

func finalizeQualificationProof(state *symbolAggregateState, engineTime time.Time) {
	qualification := ensureQualificationState(state)
	if qualification.finalized || qualification.boundExceeded {
		return
	}
	var earliest int64
	for proof := range qualification.proofs {
		proofEnd := time.Unix(proof, 0).UTC()
		if engineTime.After(proofEnd.Add(correctionHorizon)) && (earliest == 0 || proof < earliest) {
			earliest = proof
		}
	}
	if earliest == 0 {
		return
	}
	qualification.finalized = true
	qualification.finalProofEnd = time.Unix(earliest, 0).UTC()
	qualification.finalizedCount++
	clear(qualification.proofs)
	clear(qualification.dirty)
	clear(qualification.finalizedGateBars)
}

func updateQualificationResult(state *symbolAggregateState, binding *installedBinding, at time.Time) {
	qualification := ensureQualificationState(state)
	result := qualificationResult{at: at, status: qualificationUnresolved, unresolvedOrigin: qualification.unresolvedOrigin, currentProofCount: len(qualification.proofs)}
	switch {
	case qualification.boundExceeded || qualification.invalid:
		result.status = qualificationUnresolved
	case qualification.finalized:
		result.status = qualificationFinalized
		result.unresolvedOrigin = uncertaintyNone
		result.finalProofEnd = qualification.finalProofEnd
	case len(qualification.proofs) > 0:
		result.status = qualificationProvisional
		result.unresolvedOrigin = uncertaintyNone
	case !at.IsZero() && !qualification.accountedThrough.Before(at):
		if qualificationCoverageTrustworthy(state, binding, binding.sessionStart, at) {
			result.status = qualificationNotYetPassed
			result.unresolvedOrigin = uncertaintyNone
		}
	}
	qualification.result = result
}

func validQualificationState(qualification *qualificationState, binding *installedBinding) bool {
	if qualification == nil || binding == nil || qualification.invalid || len(qualification.proofs) > maximumQualificationProofs ||
		len(qualification.dirty) > maximumQualificationProofs || len(qualification.finalizedGateBars) > maximumFinalizedGateBars {
		return false
	}
	if !qualification.accountedThrough.IsZero() && (qualification.accountedThrough.Before(binding.sessionStart) ||
		qualification.accountedThrough.After(binding.sessionEnd) || qualification.accountedThrough.Nanosecond() != 0) {
		return false
	}
	if qualification.finalized {
		if qualification.finalProofEnd.Before(binding.sessionStart) || qualification.finalProofEnd.After(binding.sessionEnd) ||
			qualification.finalProofEnd.Nanosecond() != 0 || len(qualification.proofs) != 0 || len(qualification.dirty) != 0 {
			return false
		}
	} else if !qualification.finalProofEnd.IsZero() {
		return false
	}
	for proof := range qualification.proofs {
		if proof < binding.sessionStart.Unix() || proof > binding.sessionEnd.Unix() ||
			(!qualification.accountedThrough.IsZero() && proof > qualification.accountedThrough.Unix()) {
			return false
		}
	}
	for proof := range qualification.dirty {
		if _, exists := qualification.proofs[proof]; !exists {
			return false
		}
	}
	for start := range qualification.finalizedGateBars {
		if start < binding.sessionStart.Unix() || start >= binding.sessionEnd.Unix() {
			return false
		}
	}
	return true
}

func failQualificationState(qualification *qualificationState, bound bool) {
	clear(qualification.proofs)
	clear(qualification.dirty)
	clear(qualification.finalizedGateBars)
	if bound {
		qualification.boundExceeded = true
	} else {
		qualification.invalid = true
	}
	qualification.unresolvedOrigin = uncertaintyLocalInvalid
}

func calculateQualificationGateFacts(state *symbolAggregateState, binding *installedBinding, proofEnd time.Time) qualificationGateFacts {
	facts := qualificationGateFacts{a60Available: true, a5Available: true, valid: true}
	windowStart := proofEnd.Add(-qualificationWindow)
	shortStart := proofEnd.Add(-qualificationShortWindow)
	present := [60]bool{}
	totalVolume, maximumVolume := 0.0, 0.0
	for slot := 0; slot < len(present); slot++ {
		start := windowStart.Add(time.Duration(slot) * time.Second)
		if start.Before(binding.sessionStart) || !start.Before(binding.sessionEnd) {
			continue
		}
		bar, ok := qualificationBarAt(state, start.Unix())
		if !ok {
			continue
		}
		present[slot] = true
		facts.presentSeconds++
		facts.latestPrice = bar.close
		facts.liveATS = facts.liveATS || bar.provenance == ATSLiveProviderAverage
		facts.historicalATS = facts.historicalATS || bar.provenance == ATSRESTFloorVolumeOverTrades
		if !qualificationFiniteAdd(&totalVolume, bar.volume) || !qualificationFiniteAdd(&facts.dollarVolume60, bar.volume*bar.vwap) {
			facts.valid = false
		}
		maximumVolume = max(maximumVolume, bar.volume)
		if bar.averageTradeSize <= 0 {
			facts.a60Available = false
			if !start.Before(shortStart) {
				facts.a5Available = false
			}
			continue
		}
		activity := bar.volume / float64(bar.averageTradeSize)
		if !qualificationFiniteAdd(&facts.a60, activity) {
			facts.valid = false
		}
		if !start.Before(shortStart) && !qualificationFiniteAdd(&facts.a5, activity) {
			facts.valid = false
		}
	}
	facts.longestGap = longestQualificationGap(present)
	if !finiteFeature(totalVolume) || totalVolume <= 0 {
		facts.valid = false
		return facts
	}
	facts.concentration = maximumVolume / totalVolume
	facts.valid = facts.valid && finiteFeature(facts.latestPrice) && finiteFeature(facts.a60) && finiteFeature(facts.a5) &&
		finiteFeature(facts.dollarVolume60) && finiteFeature(facts.concentration)
	return facts
}

func qualificationBarAt(state *symbolAggregateState, start int64) (qualificationGateBar, bool) {
	if state.tail != nil {
		if record := state.tail[start]; record != nil {
			return qualificationGateBar{close: record.values.Close, volume: record.values.Volume, vwap: record.values.VWAP,
				averageTradeSize: record.values.AverageTradeSize, provenance: record.values.ATSProvenance}, true
		}
	}
	if state.qualification == nil {
		return qualificationGateBar{}, false
	}
	bar, ok := state.qualification.finalizedGateBars[start]
	return bar, ok
}

func passesQualificationGate(facts qualificationGateFacts) bool {
	return facts.valid && facts.latestPrice >= minimumQualificationPrice && facts.presentSeconds >= minimumQualificationSeconds &&
		facts.longestGap <= maximumQualificationGap && facts.a60Available && facts.a60 >= minimumQualificationA60 &&
		facts.dollarVolume60 >= minimumQualificationDV60 && facts.a5Available && facts.a5 >= minimumQualificationA5 &&
		facts.concentration <= maximumQualificationConcentration
}

func qualificationCoverageTrustworthy(state *symbolAggregateState, binding *installedBinding, start, end time.Time) bool {
	if start.Before(binding.sessionStart) {
		start = binding.sessionStart
	}
	if end.After(binding.sessionEnd) {
		end = binding.sessionEnd
	}
	if state.historicalConflict != nil && bitmapHasRange(state.historicalConflict, binding, start, end) {
		return false
	}
	return exactAggregateCoverage(state, binding, start, end)
}

func qualificationFiniteAdd(sum *float64, value float64) bool {
	if !finiteFeature(value) {
		return false
	}
	*sum += value
	return finiteFeature(*sum)
}

func longestQualificationGap(present [60]bool) int {
	longest, current := 0, 0
	for _, exists := range present {
		if exists {
			current = 0
			continue
		}
		current++
		longest = max(longest, current)
	}
	return longest
}
