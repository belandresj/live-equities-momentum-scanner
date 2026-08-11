package engine

import (
	"container/heap"
	"errors"
	"math"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const maximumRankingRows = 20

type rankingMode string

const (
	rankingUnavailable       rankingMode = "unavailable"
	rankingQualifiedCurrent  rankingMode = "qualified_current"
	rankingDegradedBootstrap rankingMode = "degraded_bootstrap"
	rankingStale             rankingMode = "stale"
	rankingSuppressed        rankingMode = "suppressed"
)

type rankingReason string

const (
	rankingReasonNoCommittedWatermark rankingReason = "no_committed_watermark"
	rankingReasonNoTrustedMarks       rankingReason = "no_trusted_marks"
	rankingReasonIncompletePopulation rankingReason = "incomplete_population"
	rankingReasonQualificationPending rankingReason = "qualification_incomplete"
	rankingReasonGlobalSuppression    rankingReason = "global_suppression"
)

type populationAccounting struct {
	universeTotal, validPriorClose, invalidOrMissingPriorClose uint64
	trustedRankableMark, trustedBelowPriceMark                 uint64
	noPrintThroughT, invalidMark, unknownDueFailureOrFence     uint64
	coveredPopulation, unresolvedPopulation                    uint64
}

func (p populationAccounting) reconciles() bool {
	return p.universeTotal == p.validPriorClose+p.invalidOrMissingPriorClose &&
		p.validPriorClose == p.trustedRankableMark+p.trustedBelowPriceMark+p.noPrintThroughT+p.invalidMark+p.unknownDueFailureOrFence &&
		p.coveredPopulation == p.universeTotal-p.unknownDueFailureOrFence &&
		p.unresolvedPopulation == p.unknownDueFailureOrFence
}

type qualificationAccounting struct {
	notYetPassed, provisional, finalized, unresolved uint64
}

const featureReasonBucketCount = 11

type featureDimensionAccounting struct {
	statuses [4]uint64
	reasons  [featureReasonBucketCount]uint64
	pairs    [4][featureReasonBucketCount]uint64
}

type featureAccounting struct {
	dayPercent, from4AMPercent, hodDrawdown, sessionRange, rolling30, rolling60, activity featureDimensionAccounting
}

type uncertaintyAccounting struct {
	bootstrapOrigin, postBootstrapGap, localInvalid uint64
}

type aggregateRankingRow struct {
	rank                                      uint32
	symbol                                    string
	last, dayPercent                          float64
	markAge                                   time.Duration
	from4AMPercent, hodDrawdown, sessionRange aggregateFeatureField
	rolling30, rolling60, activity            aggregateFeatureField
	tqIntentEligible                          bool
}

type aggregateEvaluationResult struct {
	at                  time.Time
	mode                rankingMode
	reason              rankingReason
	population          populationAccounting
	qualification       qualificationAccounting
	features            featureAccounting
	uncertainty         uncertaintyAccounting
	totalPassers        uint64
	knownRankableCount  uint64
	dayInvalidRankable  uint64
	qualifiedDayInvalid uint64
	rows                []aggregateRankingRow
	updates             []aggregateSymbolEvaluationUpdate
	invalidSupport      bool
	tqIntentAvailable   bool
}

type aggregateSymbolEvaluationUpdate struct {
	qualification   *qualificationState
	committedLatest *committedAggregateMark
	priceRange      priceRangeFeatureResult
	activity        activityFeatureResult
	present         bool
}

type aggregateEvaluatorState struct {
	current      aggregateEvaluationResult
	invalidMarks map[int]invalidMarkEvidence
	coverage     map[int]aggregateCoverageConsequence
}

// aggregateCoverageConsequence is engine-private consequence state only.
// Validated hydration and ordinary-live fence facts are the only production
// paths that may populate it.
type uncertaintyOrigin uint8

const (
	uncertaintyNone uncertaintyOrigin = iota
	uncertaintyBootstrapOrigin
	uncertaintyPostBootstrapGap
	uncertaintyLocalInvalid
)

type coverageOutcome uint8

const (
	coverageOutcomeNoPrint coverageOutcome = iota + 1
	coverageOutcomeUnknown
)

type aggregateCoverageConsequence struct {
	outcome coverageOutcome
	origin  uncertaintyOrigin
}

var (
	coverageNoPrintThroughT       = aggregateCoverageConsequence{outcome: coverageOutcomeNoPrint}
	coverageUnknownFailureOrFence = aggregateCoverageConsequence{outcome: coverageOutcomeUnknown, origin: uncertaintyBootstrapOrigin}
	coverageUnknownPostBootstrap  = aggregateCoverageConsequence{outcome: coverageOutcomeUnknown, origin: uncertaintyPostBootstrapGap}
)

type rankingHeap []aggregateRankingRow

func (h rankingHeap) Len() int { return len(h) }

// The root is the worst retained row.
func (h rankingHeap) Less(i, j int) bool { return rankingPrecedes(h[j], h[i]) }
func (h rankingHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *rankingHeap) Push(x any)        { *h = append(*h, x.(aggregateRankingRow)) }
func (h *rankingHeap) Pop() any {
	old := *h
	value := old[len(old)-1]
	*h = old[:len(old)-1]
	return value
}

func rankingPrecedes(a, b aggregateRankingRow) bool {
	return a.dayPercent > b.dayPercent || (a.dayPercent == b.dayPercent && a.symbol < b.symbol)
}

func retainRankingRow(h *rankingHeap, row aggregateRankingRow) {
	if h.Len() < maximumRankingRows {
		heap.Push(h, row)
		return
	}
	if rankingPrecedes(row, (*h)[0]) {
		heap.Pop(h)
		heap.Push(h, row)
	}
}

func (e *Engine) runAggregateEvaluatorLocked(node *queueNode, code DispositionCode, reason DispositionReason, candidate *aggregateEvaluationResult) bool {
	if e.state.binding == nil || e.state.globalFailure {
		return true
	}
	changed := (node.kind == inputTimer || node.kind == inputReplayGroup) && code == DispositionTimerApplied
	changed = changed || (node.kind == inputAggregateIngressFence && code == DispositionAggregateIngressFenceApplied)
	changed = changed || (e.mode == RunModeReplay && node.kind == inputAggregate && (code == DispositionAggregateInserted || code == DispositionAggregateRevised ||
		code == DispositionAggregateWithdrawn || (code == DispositionAggregateRejected && (reason == ReasonHistoricalLiveConflict || reason == ReasonStructural))))
	if !changed {
		return true
	}
	staged := aggregateEvaluationResult{}
	if candidate != nil {
		staged = *candidate
	} else {
		if e.mode == RunModeLive {
			return true
		}
		if e.state.committedT == nil {
			return true
		}
		staged = e.stageAggregateEvaluationLocked(*e.state.committedT)
	}
	expected := time.Time{}
	if e.mode == RunModeLive && candidate != nil && (node.kind == inputTimer || node.kind == inputAggregateIngressFence) {
		expected = staged.at
	} else if (node.kind == inputTimer || node.kind == inputReplayGroup || node.kind == inputAggregateIngressFence) && e.state.latestTarget != nil {
		expected = *e.state.latestTarget
	} else if e.state.committedT != nil {
		expected = *e.state.committedT
	}
	if expected.IsZero() || !staged.at.Equal(expected) {
		return false
	}
	if e.evaluationFault {
		staged.population.universeTotal++
		e.evaluationFault = false
	}
	if validateAggregateEvaluation(staged) != nil {
		return false
	}
	if !e.candidateTargetSupportedLocked(staged.at) {
		// Lack of a future Component 4-6 run-support fact is an ordinary closed
		// gate. It cannot apply candidate-T state and is not itself corruption.
		return true
	}
	e.applyStagedAggregateCandidateLocked(staged, node.admissionTime)
	consumePending := e.mode == RunModeLive && e.state.aggregateProjectionPending
	evaluationChanged := !aggregateEvaluationEqual(e.state.aggregateEvaluator.current, staged)
	if evaluationChanged {
		e.state.aggregateEvaluator.current = cloneAggregateEvaluation(staged)
		e.state.evaluationRevision++
		e.state.exposedRevision++
	}
	if consumePending {
		e.state.aggregateProjectionPending = false
		// The accepted aggregate prefix and its accounting become visible at
		// this boundary even when its recomputed market rows equal the prior
		// evaluation.
		if !evaluationChanged {
			e.state.exposedRevision++
		}
	}
	return true
}

// applyAggregateCandidateLocked is the single C3 committed-boundary apply
// point. Candidate identity, run support, contributor predicates, bounds, and
// accounting have already succeeded; this deterministic second pass applies
// no new evidence and cannot reject.
func (e *Engine) applyAggregateCandidateLocked(at, engineTime time.Time) {
	for index := range e.state.binding.symbols {
		symbol := &e.state.binding.symbols[index]
		if symbol.aggregates == nil {
			continue
		}
		if mark, ok := latestMarkBefore(symbol.aggregates, at); ok {
			symbol.aggregates.committedLatest = committedMark(mark)
		} else {
			symbol.aggregates.committedLatest = nil
		}
		evaluateQualificationThrough(symbol.aggregates, e.state.binding, at, engineTime)
		ensurePriceRangeState(symbol.aggregates).result = evaluatePriceRangeFeatures(e.state.binding, symbol, at)
		applyActivityResult(ensureActivityState(symbol.aggregates), evaluateActivityFeatures(e.state.binding, symbol.aggregates, at))
	}
	e.commitAggregateTargetLocked(at)
}

func (e *Engine) applyStagedAggregateCandidateLocked(staged aggregateEvaluationResult, engineTime time.Time) {
	if len(staged.updates) != len(e.state.binding.symbols) {
		e.applyAggregateCandidateLocked(staged.at, engineTime)
		return
	}
	for index := range e.state.binding.symbols {
		update := staged.updates[index]
		state := e.state.binding.symbols[index].aggregates
		if state == nil || !update.present {
			continue
		}
		state.committedLatest = update.committedLatest
		state.qualification = update.qualification
		ensurePriceRangeState(state).result = update.priceRange
		applyActivityResult(ensureActivityState(state), update.activity)
	}
	e.commitAggregateTargetLocked(staged.at)
}

func (e *Engine) commitAggregateTargetLocked(at time.Time) {
	if e.state.committedT == nil || at.After(*e.state.committedT) {
		e.state.committedT = immutableTime(at)
	}
}

func (e *Engine) stageAggregateEvaluationLocked(at time.Time) aggregateEvaluationResult {
	return e.stageAggregateEvaluationAtLocked(at, at)
}

func (e *Engine) stageAggregateEvaluationAtLocked(at, engineTime time.Time) aggregateEvaluationResult {
	result := aggregateEvaluationResult{at: at, mode: rankingUnavailable, reason: rankingReasonNoCommittedWatermark, tqIntentAvailable: e.mode != RunModeReplay}
	if e.state.binding == nil || at.IsZero() {
		return result
	}
	if e.state.lifecycle == lifecycleSuppressed || e.state.globalFailure {
		result.mode, result.reason = rankingSuppressed, rankingReasonGlobalSuppression
		return result
	}
	if !validAggregateEvaluatorSupport(e.state.binding, e.state.aggregateEvaluator) {
		result.invalidSupport = true
	}
	qualified, degraded := &rankingHeap{}, &rankingHeap{}
	result.updates = make([]aggregateSymbolEvaluationUpdate, len(e.state.binding.symbols))
	heap.Init(qualified)
	heap.Init(degraded)
	qualificationComplete := true
	allUnresolvedBootstrap := true
	for index := range e.state.binding.symbols {
		symbol := &e.state.binding.symbols[index]
		result.population.universeTotal++
		state := symbol.aggregates
		var mark canonicalAggregate
		hasMark := false
		if state != nil {
			mark, hasMark = latestMarkBefore(state, at)
		}
		coverage, hasCoverageConsequence := e.state.aggregateEvaluator.coverage[index]
		invalid, hasInvalidEvidence := e.state.aggregateEvaluator.invalidMarks[index]
		invalidApplicableAtT := hasInvalidEvidence && invalid.windowStart.Before(at)
		if coverage == coverageNoPrintThroughT && (hasMark || invalidApplicableAtT) {
			result.invalidSupport = true
		}

		features := unavailablePriceRangeResult(at)
		activity := unavailableActivityResult(at)
		var projectedQualification *qualificationState
		if state != nil {
			projectionState := *state
			projectionState.qualification = cloneQualificationState(state.qualification)
			projectionSymbol := *symbol
			projectionSymbol.aggregates = &projectionState
			evaluateQualificationThrough(&projectionState, e.state.binding, at, engineTime)
			features = evaluatePriceRangeFeatures(e.state.binding, &projectionSymbol, at)
			activity = evaluateActivityFeatures(e.state.binding, &projectionState, at)
			projectedQualification = projectionState.qualification
			update := aggregateSymbolEvaluationUpdate{qualification: projectedQualification, priceRange: features, activity: activity, present: true}
			if hasMark {
				update.committedLatest = committedMark(mark)
			}
			result.updates[index] = update
		}
		validPrior := symbol.prior.status == reference.PriorCloseValid && symbol.prior.close > 0 && finiteEvaluator(symbol.prior.close)
		if !validPrior {
			features.dayPercent = aggregateFeatureField{status: featureUnavailable, reason: featureReasonPriorCloseUnavailable}
		}
		if hasCoverageConsequence && coverage.outcome == coverageOutcomeUnknown {
			unknown := aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
			features.from4AMPercent, features.hodDrawdown, features.sessionRange = unknown, unknown, unknown
			features.rolling30, features.rolling60, activity.activity = unknown, unknown, unknown
		}
		if !countFeatureResult(&result.features, features, activity.activity) {
			result.invalidSupport = true
		}
		if !validPrior {
			result.population.invalidOrMissingPriorClose++
			continue
		}
		result.population.validPriorClose++

		if hasCoverageConsequence && coverage.outcome == coverageOutcomeUnknown {
			result.population.unknownDueFailureOrFence++
			countUncertaintyOrigin(&result.uncertainty, coverage.origin)
			allUnresolvedBootstrap = allUnresolvedBootstrap && coverage.origin == uncertaintyBootstrapOrigin
			continue
		}
		if state == nil || !hasMark {
			if hasCoverageConsequence && coverage.outcome == coverageOutcomeNoPrint {
				result.population.noPrintThroughT++
			} else if invalid, exists := e.state.aggregateEvaluator.invalidMarks[index]; exists && invalid.windowStart.Before(at) {
				result.population.invalidMark++
			} else {
				result.population.unknownDueFailureOrFence++
				countUncertaintyOrigin(&result.uncertainty, uncertaintyLocalInvalid)
				allUnresolvedBootstrap = false
			}
			continue
		}
		age := at.Sub(mark.windowEnd)
		if age < 0 || !finiteEvaluator(mark.values.Close) {
			result.population.unknownDueFailureOrFence++
			countUncertaintyOrigin(&result.uncertainty, uncertaintyLocalInvalid)
			allUnresolvedBootstrap = false
			continue
		}
		if mark.values.Close < minimumQualificationPrice {
			result.population.trustedBelowPriceMark++
			continue
		}
		result.population.trustedRankableMark++
		status := qualificationUnresolved
		origin := uncertaintyLocalInvalid
		if projectedQualification != nil && projectedQualification.result.at.Equal(at) {
			status = projectedQualification.result.status
			origin = projectedQualification.result.unresolvedOrigin
		}
		qualifiedPasser := false
		switch status {
		case qualificationNotYetPassed:
			result.qualification.notYetPassed++
		case qualificationProvisional:
			result.qualification.provisional++
			qualifiedPasser = true
		case qualificationFinalized:
			result.qualification.finalized++
			qualifiedPasser = true
		default:
			result.qualification.unresolved++
			qualificationComplete = false
			countUncertaintyOrigin(&result.uncertainty, origin)
			allUnresolvedBootstrap = allUnresolvedBootstrap && origin == uncertaintyBootstrapOrigin
		}
		row := aggregateRankingRow{symbol: symbol.symbol, last: mark.values.Close, markAge: age,
			from4AMPercent: features.from4AMPercent, hodDrawdown: features.hodDrawdown, sessionRange: features.sessionRange,
			rolling30: features.rolling30, rolling60: features.rolling60, activity: activity.activity}
		if features.dayPercent.status != featureCurrent || !finiteEvaluator(features.dayPercent.value) {
			result.dayInvalidRankable++
			if qualifiedPasser {
				result.qualifiedDayInvalid++
			}
			continue
		}
		row.dayPercent = features.dayPercent.value
		result.knownRankableCount++
		retainRankingRow(degraded, row)
		if qualifiedPasser {
			result.totalPassers++
			retainRankingRow(qualified, row)
		}
	}
	result.population.coveredPopulation = result.population.universeTotal - result.population.unknownDueFailureOrFence
	result.population.unresolvedPopulation = result.population.unknownDueFailureOrFence
	currentLifecycle := e.state.lifecycle == lifecycleLive || e.state.lifecycle == lifecycleHydrating || e.state.lifecycle == lifecycleReplaying
	switch {
	case !currentLifecycle:
		result.mode, result.reason = rankingUnavailable, rankingReasonNoCommittedWatermark
	case result.population.unknownDueFailureOrFence == 0 && qualificationComplete:
		result.mode, result.reason = rankingQualifiedCurrent, ""
		result.rows = sortedRankingRows(*qualified, e.mode != RunModeReplay)
	case result.knownRankableCount > 0 && allUnresolvedBootstrap:
		result.mode = rankingDegradedBootstrap
		if result.population.unknownDueFailureOrFence != 0 {
			result.reason = rankingReasonIncompletePopulation
		} else {
			result.reason = rankingReasonQualificationPending
		}
		result.rows = sortedRankingRows(*degraded, false)
	case result.knownRankableCount > 0:
		result.mode, result.reason = rankingUnavailable, rankingReasonIncompletePopulation
	default:
		result.mode, result.reason = rankingUnavailable, rankingReasonNoTrustedMarks
	}
	return result
}

func cloneQualificationState(source *qualificationState) *qualificationState {
	if source == nil {
		return nil
	}
	result := *source
	result.finalizedGateBars = make(map[int64]qualificationGateBar, len(source.finalizedGateBars))
	for key, value := range source.finalizedGateBars {
		result.finalizedGateBars[key] = value
	}
	result.proofs = make(map[int64]struct{}, len(source.proofs))
	for key := range source.proofs {
		result.proofs[key] = struct{}{}
	}
	result.dirty = make(map[int64]struct{}, len(source.dirty))
	for key := range source.dirty {
		result.dirty[key] = struct{}{}
	}
	if source.hydrationScan != nil {
		scan := *source.hydrationScan
		scan.recentPassing = make(map[int64]struct{}, len(source.hydrationScan.recentPassing))
		for key := range source.hydrationScan.recentPassing {
			scan.recentPassing[key] = struct{}{}
		}
		result.hydrationScan = &scan
	}
	return &result
}

func validAggregateEvaluatorSupport(binding *installedBinding, state aggregateEvaluatorState) bool {
	if binding == nil || len(state.invalidMarks) > len(binding.symbols) || len(state.coverage) > len(binding.symbols) {
		return false
	}
	for index, evidence := range state.invalidMarks {
		if index < 0 || index >= len(binding.symbols) || evidence.windowStart.Before(binding.sessionStart) || !evidence.windowStart.Before(binding.sessionEnd) {
			return false
		}
	}
	for index, consequence := range state.coverage {
		valid := consequence.outcome == coverageOutcomeNoPrint && consequence.origin == uncertaintyNone ||
			consequence.outcome == coverageOutcomeUnknown && (consequence.origin == uncertaintyBootstrapOrigin || consequence.origin == uncertaintyPostBootstrapGap || consequence.origin == uncertaintyLocalInvalid)
		if index < 0 || index >= len(binding.symbols) || !valid {
			return false
		}
	}
	return true
}

func sortedRankingRows(h rankingHeap, tq bool) []aggregateRankingRow {
	rows := make([]aggregateRankingRow, len(h))
	copy(rows, h)
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rankingPrecedes(rows[j], rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	for i := range rows {
		rows[i].rank = uint32(i + 1)
		rows[i].tqIntentEligible = tq
	}
	return rows
}

func validateAggregateEvaluation(r aggregateEvaluationResult) error {
	if r.mode == "" && r.at.IsZero() && len(r.rows) == 0 {
		return nil
	}
	if r.invalidSupport {
		return errors.New("contradictory evaluator support")
	}
	if r.mode != rankingUnavailable && r.mode != rankingQualifiedCurrent && r.mode != rankingDegradedBootstrap && r.mode != rankingStale && r.mode != rankingSuppressed {
		return errors.New("invalid ranking mode")
	}
	if !r.population.reconciles() || len(r.rows) > maximumRankingRows {
		return errors.New("invalid population accounting")
	}
	seen := make(map[string]struct{}, len(r.rows))
	for i, row := range r.rows {
		if row.rank != uint32(i+1) || row.symbol == "" || row.last < minimumQualificationPrice || !finiteEvaluator(row.last) || !finiteEvaluator(row.dayPercent) || row.markAge < 0 {
			return errors.New("invalid ranking row")
		}
		if i > 0 && !rankingPrecedes(r.rows[i-1], row) {
			return errors.New("unordered ranking rows")
		}
		if _, ok := seen[row.symbol]; ok {
			return errors.New("duplicate ranking row")
		}
		seen[row.symbol] = struct{}{}
		if row.tqIntentEligible != (r.mode == rankingQualifiedCurrent && r.tqIntentAvailable) {
			return errors.New("invalid TQ intent eligibility")
		}
	}
	wantQualifiedRows := minUint64(r.totalPassers, maximumRankingRows)
	wantDegradedRows := minUint64(r.knownRankableCount, maximumRankingRows)
	if r.mode == rankingQualifiedCurrent && (r.reason != "" || r.population.unknownDueFailureOrFence != 0 || r.qualification.unresolved != 0 || uint64(len(r.rows)) != wantQualifiedRows) {
		return errors.New("invalid qualified projection")
	}
	qualificationTotal := r.qualification.notYetPassed + r.qualification.provisional + r.qualification.finalized + r.qualification.unresolved
	if qualificationTotal != r.population.trustedRankableMark ||
		r.totalPassers+r.qualifiedDayInvalid != r.qualification.provisional+r.qualification.finalized ||
		r.knownRankableCount+r.dayInvalidRankable != r.population.trustedRankableMark ||
		r.qualifiedDayInvalid > r.dayInvalidRankable {
		return errors.New("invalid qualification accounting")
	}
	if r.uncertainty.bootstrapOrigin+r.uncertainty.postBootstrapGap+r.uncertainty.localInvalid !=
		r.population.unknownDueFailureOrFence+r.qualification.unresolved {
		return errors.New("invalid uncertainty-origin accounting")
	}
	for _, counts := range []featureDimensionAccounting{r.features.dayPercent, r.features.from4AMPercent, r.features.hodDrawdown, r.features.sessionRange, r.features.rolling30, r.features.rolling60, r.features.activity} {
		if !counts.reconciles(r.population.universeTotal) {
			return errors.New("invalid feature accounting")
		}
	}
	if r.mode == rankingDegradedBootstrap && (r.knownRankableCount == 0 || (r.population.unknownDueFailureOrFence == 0 && r.qualification.unresolved == 0) || uint64(len(r.rows)) != wantDegradedRows) {
		return errors.New("invalid degraded projection")
	}
	if r.mode == rankingDegradedBootstrap && (r.uncertainty.postBootstrapGap != 0 || r.uncertainty.localInvalid != 0 || r.uncertainty.bootstrapOrigin == 0) {
		return errors.New("non-bootstrap degraded projection")
	}
	if r.mode == rankingDegradedBootstrap {
		wantReason := rankingReasonQualificationPending
		if r.population.unknownDueFailureOrFence != 0 {
			wantReason = rankingReasonIncompletePopulation
		}
		if r.reason != wantReason {
			return errors.New("invalid degraded reason")
		}
	}
	if r.mode == rankingUnavailable && r.reason != rankingReasonNoCommittedWatermark && r.reason != rankingReasonNoTrustedMarks && r.reason != rankingReasonIncompletePopulation {
		return errors.New("invalid unavailable reason")
	}
	if r.mode == rankingStale && r.reason != "" {
		return errors.New("invalid stale reason")
	}
	if r.mode == rankingSuppressed && r.reason != rankingReasonGlobalSuppression {
		return errors.New("invalid suppressed reason")
	}
	if (r.mode == rankingUnavailable || r.mode == rankingStale || r.mode == rankingSuppressed) && len(r.rows) != 0 {
		return errors.New("rows in noncurrent projection")
	}
	return nil
}

func minUint64(value uint64, limit int) uint64 {
	if value < uint64(limit) {
		return value
	}
	return uint64(limit)
}

func cloneAggregateEvaluation(r aggregateEvaluationResult) aggregateEvaluationResult {
	r.rows = append([]aggregateRankingRow(nil), r.rows...)
	r.updates = nil
	return r
}
func aggregateEvaluationEqual(a, b aggregateEvaluationResult) bool {
	if a.at != b.at || a.mode != b.mode || a.reason != b.reason || a.population != b.population || a.qualification != b.qualification || a.features != b.features || a.uncertainty != b.uncertainty || a.totalPassers != b.totalPassers || a.knownRankableCount != b.knownRankableCount || a.dayInvalidRankable != b.dayInvalidRankable || a.qualifiedDayInvalid != b.qualifiedDayInvalid || a.invalidSupport != b.invalidSupport || a.tqIntentAvailable != b.tqIntentAvailable || len(a.rows) != len(b.rows) {
		return false
	}
	for i := range a.rows {
		if a.rows[i] != b.rows[i] {
			return false
		}
	}
	return true
}
func finiteEvaluator(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func featureStatusIndex(status aggregateFeatureStatus) (int, bool) {
	switch status {
	case featureWarming:
		return 0, true
	case featureCurrent:
		return 1, true
	case featureUnavailable:
		return 2, true
	case featureInvalid:
		return 3, true
	}
	return 0, false
}

func featureReasonIndex(reason aggregateFeatureReason) (int, bool) {
	switch reason {
	case featureReasonNone:
		return 0, true
	case featureReasonBeforeFirstPrint:
		return 1, true
	case featureReasonHistoryIncomplete:
		return 2, true
	case featureReasonRollingWarmup:
		return 3, true
	case featureReasonReferenceWarmup:
		return 4, true
	case featureReasonZeroWidth:
		return 5, true
	case featureReasonHistoricalConflict:
		return 6, true
	case featureReasonInvalidInput:
		return 7, true
	case featureReasonStateBoundExceeded:
		return 8, true
	case featureReasonPriorCloseUnavailable:
		return 9, true
	case featureReasonNoAggregateInTarget:
		return 10, true
	}
	return 0, false
}

func validFeatureStatusReason(status aggregateFeatureStatus, reason aggregateFeatureReason) bool {
	switch status {
	case featureCurrent:
		return reason == featureReasonNone
	case featureWarming:
		return reason == featureReasonHistoryIncomplete || reason == featureReasonRollingWarmup || reason == featureReasonReferenceWarmup
	case featureUnavailable:
		return reason == featureReasonBeforeFirstPrint || reason == featureReasonHistoryIncomplete || reason == featureReasonZeroWidth ||
			reason == featureReasonPriorCloseUnavailable || reason == featureReasonNoAggregateInTarget
	case featureInvalid:
		return reason == featureReasonHistoricalConflict || reason == featureReasonInvalidInput || reason == featureReasonStateBoundExceeded
	}
	return false
}

func (d featureDimensionAccounting) reconciles(universe uint64) bool {
	statusTotal, reasonTotal, pairTotal := uint64(0), uint64(0), uint64(0)
	for _, value := range d.statuses {
		statusTotal += value
	}
	for _, value := range d.reasons {
		reasonTotal += value
	}
	for status := range d.pairs {
		for reason, value := range d.pairs[status] {
			pairTotal += value
			if value != 0 && !validFeatureStatusReason(featureStatusFromIndex(status), featureReasonFromIndex(reason)) {
				return false
			}
		}
	}
	if statusTotal != universe || reasonTotal != universe || pairTotal != universe {
		return false
	}
	for status := range d.statuses {
		row := uint64(0)
		for _, value := range d.pairs[status] {
			row += value
		}
		if row != d.statuses[status] {
			return false
		}
	}
	for reason := range d.reasons {
		column := uint64(0)
		for status := range d.pairs {
			column += d.pairs[status][reason]
		}
		if column != d.reasons[reason] {
			return false
		}
	}
	return d.reasons[0] == d.statuses[1] && universe-d.statuses[1] == reasonTotal-d.reasons[0]
}

func featureStatusFromIndex(index int) aggregateFeatureStatus {
	return [...]aggregateFeatureStatus{featureWarming, featureCurrent, featureUnavailable, featureInvalid}[index]
}
func featureReasonFromIndex(index int) aggregateFeatureReason {
	return [...]aggregateFeatureReason{featureReasonNone, featureReasonBeforeFirstPrint, featureReasonHistoryIncomplete, featureReasonRollingWarmup,
		featureReasonReferenceWarmup, featureReasonZeroWidth, featureReasonHistoricalConflict, featureReasonInvalidInput,
		featureReasonStateBoundExceeded, featureReasonPriorCloseUnavailable, featureReasonNoAggregateInTarget}[index]
}

func countFeatureResult(f *featureAccounting, p priceRangeFeatureResult, activity aggregateFeatureField) bool {
	fields := []aggregateFeatureField{p.dayPercent, p.from4AMPercent, p.hodDrawdown, p.sessionRange, p.rolling30, p.rolling60, activity}
	counts := []*featureDimensionAccounting{&f.dayPercent, &f.from4AMPercent, &f.hodDrawdown, &f.sessionRange, &f.rolling30, &f.rolling60, &f.activity}
	for i := range fields {
		status, statusOK := featureStatusIndex(fields[i].status)
		reason, reasonOK := featureReasonIndex(fields[i].reason)
		if !statusOK || !reasonOK || !validFeatureStatusReason(fields[i].status, fields[i].reason) {
			return false
		}
		counts[i].statuses[status]++
		counts[i].reasons[reason]++
		counts[i].pairs[status][reason]++
	}
	return true
}

func countUncertaintyOrigin(accounting *uncertaintyAccounting, origin uncertaintyOrigin) {
	switch origin {
	case uncertaintyBootstrapOrigin:
		accounting.bootstrapOrigin++
	case uncertaintyPostBootstrapGap:
		accounting.postBootstrapGap++
	default:
		accounting.localInvalid++
	}
}
