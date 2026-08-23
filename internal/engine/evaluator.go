package engine

import (
	"container/heap"
	"math"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const maximumRankingRows = 20

// EvaluatorIntegrityCategory is the closed classification of the first
// terminal evaluator invariant failure. It is diagnostic evidence only; the
// lifecycle reason and suppression disposition remain authoritative.
type EvaluatorIntegrityCategory string

const (
	EvaluatorCandidateTargetMismatch EvaluatorIntegrityCategory = "candidate_target_mismatch"
	EvaluatorSupportContradiction    EvaluatorIntegrityCategory = "support_contradiction"
	EvaluatorPopulationAccounting    EvaluatorIntegrityCategory = "population_accounting"
	EvaluatorQualificationAccounting EvaluatorIntegrityCategory = "qualification_accounting"
	EvaluatorUncertaintyAccounting   EvaluatorIntegrityCategory = "uncertainty_accounting"
	EvaluatorFeatureAccounting       EvaluatorIntegrityCategory = "feature_accounting"
	EvaluatorRankingProjection       EvaluatorIntegrityCategory = "ranking_projection"
	EvaluatorRankingRow              EvaluatorIntegrityCategory = "ranking_row"
	EvaluatorTQIntent                EvaluatorIntegrityCategory = "tq_intent"
	EvaluatorUnknownIntegrity        EvaluatorIntegrityCategory = "unknown_evaluator_integrity"
)

type EvaluatorIntegrityView struct {
	Category                 EvaluatorIntegrityCategory
	InputKind                string
	EngineSequence           uint64
	CandidateTime            time.Time
	ExpectedTime             time.Time
	Lifecycle                string
	HydrationPurpose         HydrationPurpose
	HydrationGeneration      uint64
	FenceEpoch               uint64
	FenceThrough             uint64
	FenceMarkerOrdinal       uint64
	UniverseTotal            uint64
	ValidPriorClose          uint64
	InvalidOrMissingPrior    uint64
	TrustedRankableMark      uint64
	TrustedBelowPriceMark    uint64
	NoPrintThroughT          uint64
	InvalidMark              uint64
	UnknownDueFailureOrFence uint64
	QualificationUnresolved  uint64
	FirstSymbol              string
	FirstField               string
	FirstReason              string
}

type evaluatorValidationResult struct {
	Category    EvaluatorIntegrityCategory
	FirstSymbol string
	FirstField  string
	FirstReason string
}

func (r *evaluatorValidationResult) valid() bool { return r == nil }
func (r *evaluatorValidationResult) Error() string {
	if r == nil {
		return ""
	}
	return string(r.Category)
}
func invalidEvaluation(category EvaluatorIntegrityCategory, field, symbol, reason string) *evaluatorValidationResult {
	return &evaluatorValidationResult{Category: category, FirstField: field, FirstSymbol: symbol, FirstReason: reason}
}

type rankingMode string

const (
	rankingUnavailable       rankingMode = "unavailable"
	rankingQualifiedCurrent  rankingMode = "qualified_current"
	rankingDegradedBootstrap rankingMode = "degraded_bootstrap"
	rankingDegradedCurrent   rankingMode = "degraded_current"
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
	sessionVolume, fromOpenPercent, dayRange, activity30s, move30s featureDimensionAccounting
	// Legacy diagnostic dimensions remain populated until snapshot API v2 owns
	// the schema cutover in MVP-S2. They are not part of the ranking row model.
	dayPercent, from4AMPercent, hodDrawdown, sessionRange, rolling30, rolling60, activity featureDimensionAccounting
}

type floatAccounting struct {
	current, stale, unavailable, invalid uint64
}

func (a floatAccounting) reconciles(universe uint64) bool {
	return a.current+a.stale+a.unavailable+a.invalid == universe
}

type aggregateFloatStatus string

const (
	aggregateFloatCurrent     aggregateFloatStatus = "current"
	aggregateFloatStale       aggregateFloatStatus = "stale"
	aggregateFloatUnavailable aggregateFloatStatus = "unavailable"
	aggregateFloatInvalid     aggregateFloatStatus = "invalid"
)

type aggregateFloatField struct {
	status aggregateFloatStatus
	reason string
	fact   reference.FloatFact
}

type uncertaintyAccounting struct {
	bootstrapOrigin, postBootstrapGap, localInvalid uint64
}

// populationTransitionDiagnostic explains the narrow bootstrap-unknown
// population decision without changing the primary population partition. Its
// reason bins are mutually exclusive and exhaustive over bootstrapUnknown.
type populationTransitionDiagnostic struct {
	bootstrapUnknown, trustedByLaterLiveMark, noLaterEligibleMark uint64
	latestMarkNotLiveAuthority, noStrictlyOlderLocalizedConflict  uint64
	conflictAtOrAfterMark, invalidAtOrAfterMark                   uint64
	incompletePostMarkCoverage                                    uint64
}

func (d populationTransitionDiagnostic) reconciles() bool {
	return d.bootstrapUnknown == d.trustedByLaterLiveMark+d.noLaterEligibleMark+d.latestMarkNotLiveAuthority+
		d.noStrictlyOlderLocalizedConflict+d.conflictAtOrAfterMark+d.invalidAtOrAfterMark+d.incompletePostMarkCoverage
}

type aggregateRankingRow struct {
	rank                                     uint32
	symbol                                   string
	last, dayPercent                         float64
	markAge                                  time.Duration
	float                                    aggregateFloatField
	sessionVolume, fromOpenPercent, dayRange aggregateFeatureField
	activity30s, move30s                     aggregateFeatureField
	tqIntentEligible                         bool
}

type aggregateEvaluationResult struct {
	at                   time.Time
	mode                 rankingMode
	reason               rankingReason
	population           populationAccounting
	qualification        qualificationAccounting
	features             featureAccounting
	floats               floatAccounting
	uncertainty          uncertaintyAccounting
	populationTransition populationTransitionDiagnostic
	totalPassers         uint64
	knownRankableCount   uint64
	dayInvalidRankable   uint64
	qualifiedDayInvalid  uint64
	rows                 []aggregateRankingRow
	updates              []aggregateSymbolEvaluationUpdate
	invalidSupport       bool
	invalidSupportSymbol string
	invalidSupportReason string
	tqIntentAvailable    bool
}

type aggregateSymbolEvaluationUpdate struct {
	qualification   *qualificationState
	committedLatest *committedAggregateMark
	priceRange      priceRangeFeatureResult
	activity        activityFeatureResult
	mvpMeasurements mvpMeasurementResult
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
	if e.hiddenReplayWarmupLocked(node) || e.deferReplayAggregateProjectionLocked(node) {
		return true
	}
	changed := (node.kind == inputTimer || node.kind == inputReplayGroup) && code == DispositionTimerApplied
	changed = changed || (node.kind == inputAggregateIngressFence && code == DispositionAggregateIngressFenceApplied)
	changed = changed || (node.kind == inputLiveCoverageFence && code == DispositionLiveCoverageFenceApplied)
	changed = changed || (e.mode == RunModeReplay && node.kind == inputAggregate && (code == DispositionAggregateInserted || code == DispositionAggregateRevised ||
		code == DispositionAggregateWithdrawn || (code == DispositionAggregateRejected && reason == ReasonStructural)))
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
	if e.mode == RunModeLive && candidate != nil && (node.kind == inputTimer || node.kind == inputAggregateIngressFence || node.kind == inputLiveCoverageFence) {
		expected = staged.at
	} else if (node.kind == inputTimer || node.kind == inputReplayGroup || node.kind == inputAggregateIngressFence) && e.state.latestTarget != nil {
		expected = *e.state.latestTarget
	} else if e.state.committedT != nil {
		expected = *e.state.committedT
	}
	if expected.IsZero() || !staged.at.Equal(expected) {
		e.latchEvaluatorIntegrityLocked(node, staged, expected, invalidEvaluation(EvaluatorCandidateTargetMismatch, "candidate_time", "", "candidate_does_not_match_expected_target"))
		return false
	}
	if e.evaluationFault {
		staged.population.universeTotal++
		e.evaluationFault = false
	}
	if validation := validateAggregateEvaluation(staged); !validation.valid() {
		e.latchEvaluatorIntegrityLocked(node, staged, expected, validation)
		return false
	}
	if !e.candidateTargetSupportedLocked(staged.at) {
		// Lack of a future Component 4-6 run-support fact is an ordinary closed
		// gate. It cannot apply candidate-T state and is not itself corruption.
		return true
	}
	applyStarted := e.evaluationTimingStart()
	e.applyStagedAggregateCandidateLocked(staged, node.admissionTime)
	e.state.evaluationAppliedSequence = node.engineSequence
	e.state.evaluationTiming.Apply = e.evaluationTimingElapsed(applyStarted)
	if node.kind == inputAggregateIngressFence {
		e.state.fenceTiming.EvaluationApply = e.state.evaluationTiming.Apply
	}
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

// ArmEvaluatorAccountingFaultForTest exposes the existing single-use
// deterministic fault seam to cross-package composition proofs. It mutates no
// market fact and is consumed by the next evaluator validation.
func (e *Engine) ArmEvaluatorAccountingFaultForTest() {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.evaluationFault = true
	// The fault targets the next evaluator validation, so expose one explicit
	// pending projection opportunity instead of relying on periodic rescans.
	e.state.aggregateProjectionPending = true
	e.mu.Unlock()
}

func (e *Engine) latchEvaluatorIntegrityLocked(node *queueNode, staged aggregateEvaluationResult, expected time.Time, validation *evaluatorValidationResult) {
	if e.state.evaluatorIntegrity != nil {
		return
	}
	category := validation.Category
	if category == "" {
		category = EvaluatorUnknownIntegrity
	}
	p := staged.population
	value := EvaluatorIntegrityView{
		Category: category, InputKind: inputKindName(node.kind), EngineSequence: node.engineSequence,
		CandidateTime: staged.at, ExpectedTime: expected, Lifecycle: string(e.state.lifecycle),
		HydrationPurpose: e.state.hydration.generation.purpose, HydrationGeneration: e.state.hydration.generation.generation,
		FenceEpoch: e.state.hydration.fenceEpoch, FenceThrough: e.state.hydration.fenceThrough, FenceMarkerOrdinal: e.state.hydration.fenceMarkerOrdinal,
		UniverseTotal: p.universeTotal, ValidPriorClose: p.validPriorClose, InvalidOrMissingPrior: p.invalidOrMissingPriorClose,
		TrustedRankableMark: p.trustedRankableMark, TrustedBelowPriceMark: p.trustedBelowPriceMark,
		NoPrintThroughT: p.noPrintThroughT, InvalidMark: p.invalidMark, UnknownDueFailureOrFence: p.unknownDueFailureOrFence,
		QualificationUnresolved: staged.qualification.unresolved,
		FirstSymbol:             validation.FirstSymbol, FirstField: validation.FirstField, FirstReason: validation.FirstReason,
	}
	e.state.evaluatorIntegrity = &value
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
		advanceSelectionMark(symbol.aggregates, e.state.binding, at)
		if mark, ok := e.latestSelectionMarkLocked(symbol.aggregates, at); ok {
			installCommittedSelection(symbol.aggregates, at, committedMark(mark))
		} else {
			installCommittedSelection(symbol.aggregates, at, nil)
		}
		evaluateQualificationThrough(symbol.aggregates, e.state.binding, at, engineTime)
		ensurePriceRangeState(symbol.aggregates).result = e.evaluatePriceRangeFeaturesLocked(e.state.binding, symbol, at)
		applyActivityResult(symbol.aggregates, e.state.binding, evaluateActivityFeatures(e.state.binding, symbol.aggregates, at))
		var invalid *invalidMarkEvidence
		if evidence, ok := e.state.aggregateEvaluator.invalidMarks[index]; ok {
			copyEvidence := evidence
			invalid = &copyEvidence
		}
		ensureMVPMeasurementState(symbol.aggregates).result = evaluateMVPMeasurements(e.state.binding, symbol.aggregates, at, invalid)
	}
	e.commitAggregateTargetLocked(at)
}

func (e *Engine) applyStagedAggregateCandidateLocked(staged aggregateEvaluationResult, engineTime time.Time) {
	if len(staged.updates) != len(e.state.binding.symbols) {
		e.applyAggregateCandidateLocked(staged.at, engineTime)
		e.refreshAggregateEvaluationDeadlineLocked()
		return
	}
	var nextDeadline *time.Time
	for index := range e.state.binding.symbols {
		update := staged.updates[index]
		state := e.state.binding.symbols[index].aggregates
		if state == nil || !update.present {
			continue
		}
		installCommittedSelection(state, staged.at, update.committedLatest)
		state.qualification = update.qualification
		ensurePriceRangeState(state).result = update.priceRange
		applyActivityResult(state, e.state.binding, update.activity)
		ensureMVPMeasurementState(state).result = update.mvpMeasurements
		if qualification := state.qualification; qualification != nil && !qualification.finalized {
			for proof := range qualification.proofs {
				deadline := time.Unix(proof, 0).UTC().Add(correctionHorizon)
				if nextDeadline == nil || deadline.Before(*nextDeadline) {
					nextDeadline = immutableTime(deadline)
				}
			}
		}
	}
	e.state.aggregateEvaluationDeadline = nextDeadline
	e.commitAggregateTargetLocked(staged.at)
}

func (e *Engine) refreshAggregateEvaluationDeadlineLocked() {
	var nextDeadline *time.Time
	for index := range e.state.binding.symbols {
		state := e.state.binding.symbols[index].aggregates
		if state == nil || state.qualification == nil || state.qualification.finalized {
			continue
		}
		for proof := range state.qualification.proofs {
			deadline := time.Unix(proof, 0).UTC().Add(correctionHorizon)
			if nextDeadline == nil || deadline.Before(*nextDeadline) {
				nextDeadline = immutableTime(deadline)
			}
		}
	}
	e.state.aggregateEvaluationDeadline = nextDeadline
}

func (e *Engine) commitAggregateTargetLocked(at time.Time) {
	if e.state.committedT == nil || at.After(*e.state.committedT) {
		e.state.committedT = immutableTime(at)
	}
}

func (e *Engine) stageAggregateEvaluationLocked(at time.Time) aggregateEvaluationResult {
	return e.stageAggregateEvaluationAtLocked(at, at)
}

type populationTransitionDecision uint8

const (
	populationTransitionNotApplicable populationTransitionDecision = iota
	populationTransitionTrustedByLaterLiveMark
	populationTransitionNoLaterEligibleMark
	populationTransitionLatestMarkNotLiveAuthority
	populationTransitionNoStrictlyOlderLocalizedConflict
	populationTransitionConflictAtOrAfterMark
	populationTransitionInvalidAtOrAfterMark
	populationTransitionIncompletePostMarkCoverage
)

// classifyPopulationTransition reports why an exact bootstrap-origin unknown
// consequence can or cannot remain field-local while a later mark supplies the
// primary population partition. It deliberately does not clear the consequence:
// historical features and qualification still consume the incomplete prefix.
func classifyPopulationTransition(binding *installedBinding, state *symbolAggregateState, mark canonicalAggregate, hasMark bool, at time.Time, coverage aggregateCoverageConsequence, invalid *invalidMarkEvidence) populationTransitionDecision {
	if coverage != coverageUnknownFailureOrFence || binding == nil {
		return populationTransitionNotApplicable
	}
	if state == nil || !hasMark || mark.windowStart.Before(binding.sessionStart) || !mark.windowStart.Before(at) || mark.windowEnd.After(at) {
		return populationTransitionNoLaterEligibleMark
	}
	if mark.authority.source != AggregateSourceLive || mark.greatestLiveSupport == nil ||
		mark.authority.live.ConnectionEpoch == 0 || mark.authority.live.FrameSequence == 0 {
		return populationTransitionLatestMarkNotLiveAuthority
	}
	if state.historicalConflict != nil && bitmapHasRange(state.historicalConflict, binding, mark.windowStart, at) {
		return populationTransitionConflictAtOrAfterMark
	}
	if invalid != nil && !invalid.windowStart.Before(mark.windowStart) && invalid.windowStart.Before(at) {
		return populationTransitionInvalidAtOrAfterMark
	}
	// Exact coverage includes the mark identity and every later second through
	// interval can hide a newer mark. The unresolved prefix itself is the
	// historical/qualification limitation; it need not also contain a conflict.
	if !exactAggregateCoverage(state, binding, mark.windowStart, at) {
		return populationTransitionIncompletePostMarkCoverage
	}
	return populationTransitionTrustedByLaterLiveMark
}

func recordPopulationTransition(diagnostic *populationTransitionDiagnostic, decision populationTransitionDecision) {
	if diagnostic == nil || decision == populationTransitionNotApplicable {
		return
	}
	diagnostic.bootstrapUnknown++
	switch decision {
	case populationTransitionTrustedByLaterLiveMark:
		diagnostic.trustedByLaterLiveMark++
	case populationTransitionNoLaterEligibleMark:
		diagnostic.noLaterEligibleMark++
	case populationTransitionLatestMarkNotLiveAuthority:
		diagnostic.latestMarkNotLiveAuthority++
	case populationTransitionNoStrictlyOlderLocalizedConflict:
		diagnostic.noStrictlyOlderLocalizedConflict++
	case populationTransitionConflictAtOrAfterMark:
		diagnostic.conflictAtOrAfterMark++
	case populationTransitionInvalidAtOrAfterMark:
		diagnostic.invalidAtOrAfterMark++
	case populationTransitionIncompletePostMarkCoverage:
		diagnostic.incompletePostMarkCoverage++
	}
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
		selectionView := e.selectionStateViewAtLocked(index, at)
		result.population.universeTotal++
		state := symbol.aggregates
		evaluationState := state
		var projectionState symbolAggregateState
		if state != nil {
			projectionState = *state
			if state.tailCoverageBuilt {
				if state.tailCoverageUsable {
					projectionState.evaluationTailPresence = &state.tailCoverage
				}
			} else {
				var tailPresence evaluationTailWindow
				if buildEvaluationTailPresence(state, e.state.binding, &tailPresence) {
					projectionState.evaluationTailPresence = &tailPresence
				}
			}
			evaluationState = &projectionState
		}
		var mark canonicalAggregate
		hasMark := false
		if state != nil {
			mark, hasMark = e.latestSelectionMarkLocked(state, at)
			if e.mode == RunModeLive && (hasMark != selectionView.MarkAvailable ||
				(hasMark && (!mark.windowStart.Equal(selectionView.TrustedMarkAt) || mark.values != selectionView.TrustedMark))) {
				result.invalidSupport = true
				if result.invalidSupportSymbol == "" {
					result.invalidSupportSymbol = symbol.symbol
					result.invalidSupportReason = "selection_view_mark_mismatch"
				}
			}
		}
		coverage, hasCoverageConsequence := e.state.aggregateEvaluator.coverage[index]
		// A complete fresh replay proves every symbol's presence or absence
		// through its covered logical boundary. Delivery may already contain a
		// mark newer than committed T, so the mutable latest-delivery consequence
		// cannot decide population coverage at T. Derive the point-in-time
		// consequence from the validated prefix and the mark eligible at T.
		if e.mode == RunModeReplay && e.state.replay.validated && e.state.replay.complete && e.state.installedCheckpoint == nil &&
			e.state.replay.coveredThrough != nil && !e.state.replay.coveredThrough.Before(at) {
			if hasMark {
				coverage, hasCoverageConsequence = aggregateCoverageConsequence{}, false
			} else {
				coverage, hasCoverageConsequence = coverageNoPrintThroughT, true
			}
		}
		invalid, hasInvalidEvidence := e.state.aggregateEvaluator.invalidMarks[index]
		invalidApplicableAtT := hasInvalidEvidence && invalid.windowStart.Before(at)
		var invalidEvidence *invalidMarkEvidence
		if hasInvalidEvidence {
			invalidEvidence = &invalid
		}
		populationTransition := populationTransitionNotApplicable
		if hasCoverageConsequence {
			populationTransition = classifyPopulationTransition(e.state.binding, evaluationState, mark, hasMark, at, coverage, invalidEvidence)
		}
		populationMarkTrusted := populationTransition == populationTransitionTrustedByLaterLiveMark
		if coverage == coverageNoPrintThroughT && (hasMark || invalidApplicableAtT) {
			result.invalidSupport = true
			if result.invalidSupportSymbol == "" {
				result.invalidSupportSymbol = symbol.symbol
				switch {
				case hasMark && invalidApplicableAtT:
					result.invalidSupportReason = "no_print_with_mark_and_invalid_evidence"
				case hasMark:
					result.invalidSupportReason = "no_print_with_mark"
				default:
					result.invalidSupportReason = "no_print_with_invalid_evidence"
				}
			}
		}

		features := unavailablePriceRangeResult(at)
		activity := unavailableActivityResult(at)
		measurements := unavailableMVPMeasurementResult(at)
		var projectedQualification *qualificationState
		if state != nil {
			// Qualification was advanced incrementally by the owner transition
			// before staging. Selection reads its fixed result scalars directly;
			// proof maps are neither cloned nor mutated by this candidate. The
			// projection copy exists only for temporary pre-B3 display adapters.
			projectionState.qualification = state.qualification
			if e.mode == RunModeReplay {
				// Replay is unsupported tooling pending E1 disposition. Preserve its
				// hidden-warmup projection compatibility without restoring a clone or
				// second qualification owner to the supported live path.
				projectionState.qualification = cloneQualificationStateForReplay(state.qualification)
				evaluateQualificationThrough(&projectionState, e.state.binding, at, engineTime)
			}
			projectionSymbol := *symbol
			projectionSymbol.aggregates = &projectionState
			features = evaluatePriceRangeFeaturesWithMark(e.state.binding, &projectionSymbol, at, mark, hasMark)
			activity = evaluateActivityFeatures(e.state.binding, &projectionState, at)
			measurements = evaluateMVPMeasurements(e.state.binding, &projectionState, at, invalidEvidence)
			projectedQualification = projectionState.qualification
			update := aggregateSymbolEvaluationUpdate{qualification: projectedQualification, priceRange: features, activity: activity, mvpMeasurements: measurements, present: true}
			if hasMark {
				update.committedLatest = committedMark(mark)
			}
			result.updates[index] = update
		}
		validPrior := symbol.prior.status == reference.PriorCloseValid && symbol.prior.close > 0 && finiteEvaluator(symbol.prior.close)
		if !validPrior {
			features.dayPercent = aggregateFeatureField{status: featureUnavailable, reason: featureReasonPriorCloseUnavailable}
		}
		if hasCoverageConsequence && coverage.outcome == coverageOutcomeUnknown && !populationMarkTrusted {
			unknown := aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
			features.from4AMPercent, features.hodDrawdown, features.sessionRange = unknown, unknown, unknown
			features.rolling30, features.rolling60, activity.activity = unknown, unknown, unknown
			measurements.sessionVolume = unknownUnlessInvalid(measurements.sessionVolume, unknown)
			measurements.activity30s = unknownUnlessInvalid(measurements.activity30s, unknown)
			measurements.move30s = unknownUnlessInvalid(measurements.move30s, unknown)
		}
		if !countFeatureResult(&result.features, features, activity.activity, measurements) {
			result.invalidSupport = true
		}
		floatField := e.floatForSymbol(symbol.symbol)
		countFloatResult(&result.floats, floatField)
		if !validPrior {
			result.population.invalidOrMissingPriorClose++
			continue
		}
		result.population.validPriorClose++
		recordPopulationTransition(&result.populationTransition, populationTransition)

		if hasCoverageConsequence && coverage.outcome == coverageOutcomeUnknown && !populationMarkTrusted {
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
		row := aggregateRankingRow{symbol: symbol.symbol, last: mark.values.Close, markAge: age, float: floatField,
			sessionVolume: measurements.sessionVolume, fromOpenPercent: features.from4AMPercent, dayRange: features.sessionRange,
			activity30s: measurements.activity30s, move30s: measurements.move30s}
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
	case e.mode == RunModeLive && e.state.lifecycle == lifecycleHydrating && result.knownRankableCount > 0 && allUnresolvedBootstrap:
		result.mode = rankingDegradedBootstrap
		if result.population.unknownDueFailureOrFence != 0 {
			result.reason = rankingReasonIncompletePopulation
		} else {
			result.reason = rankingReasonQualificationPending
		}
		result.rows = sortedRankingRows(*degraded, false)
	case e.mode == RunModeLive && result.knownRankableCount > 0:
		result.mode = rankingDegradedCurrent
		if result.population.unknownDueFailureOrFence != 0 {
			result.reason = rankingReasonIncompletePopulation
		} else {
			result.reason = rankingReasonQualificationPending
		}
		result.rows = sortedRankingRows(*degraded, false)
	case e.mode == RunModeReplay && result.knownRankableCount > 0:
		result.mode = rankingUnavailable
		if result.population.unknownDueFailureOrFence != 0 {
			result.reason = rankingReasonIncompletePopulation
		} else {
			result.reason = rankingReasonNoTrustedMarks
		}
	default:
		result.mode, result.reason = rankingUnavailable, rankingReasonNoTrustedMarks
	}
	return result
}

func cloneQualificationStateForReplay(source *qualificationState) *qualificationState {
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

func validateAggregateEvaluation(r aggregateEvaluationResult) *evaluatorValidationResult {
	if r.mode == "" && r.at.IsZero() && len(r.rows) == 0 {
		return nil
	}
	if r.invalidSupport {
		reason := r.invalidSupportReason
		if reason == "" {
			reason = "contradictory_evaluator_support"
		}
		return invalidEvaluation(EvaluatorSupportContradiction, "support", r.invalidSupportSymbol, reason)
	}
	if r.mode != rankingUnavailable && r.mode != rankingQualifiedCurrent && r.mode != rankingDegradedBootstrap && r.mode != rankingDegradedCurrent && r.mode != rankingStale && r.mode != rankingSuppressed {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.mode", "", "invalid_mode")
	}
	if !r.population.reconciles() {
		return invalidEvaluation(EvaluatorPopulationAccounting, "accounting.population", "", "identity_mismatch")
	}
	if !r.populationTransition.reconciles() ||
		r.populationTransition.trustedByLaterLiveMark > r.population.trustedRankableMark+r.population.trustedBelowPriceMark ||
		r.populationTransition.bootstrapUnknown-r.populationTransition.trustedByLaterLiveMark > r.population.unknownDueFailureOrFence {
		return invalidEvaluation(EvaluatorPopulationAccounting, "accounting.population_transition", "", "identity_mismatch")
	}
	if len(r.rows) > maximumRankingRows {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.rows", "", "row_bound_exceeded")
	}
	seen := make(map[string]struct{}, len(r.rows))
	for i, row := range r.rows {
		if row.rank != uint32(i+1) || row.symbol == "" || row.last < minimumQualificationPrice || !finiteEvaluator(row.last) || !finiteEvaluator(row.dayPercent) || row.markAge < 0 {
			return invalidEvaluation(EvaluatorRankingRow, "ranking.row", row.symbol, "invalid_row")
		}
		if i > 0 && !rankingPrecedes(r.rows[i-1], row) {
			return invalidEvaluation(EvaluatorRankingRow, "ranking.order", row.symbol, "unordered_row")
		}
		if _, ok := seen[row.symbol]; ok {
			return invalidEvaluation(EvaluatorRankingRow, "ranking.symbol", row.symbol, "duplicate_row")
		}
		seen[row.symbol] = struct{}{}
		if row.tqIntentEligible != (r.mode == rankingQualifiedCurrent && r.tqIntentAvailable) {
			return invalidEvaluation(EvaluatorTQIntent, "ranking.tq_intent", row.symbol, "eligibility_mismatch")
		}
	}
	wantQualifiedRows := minUint64(r.totalPassers, maximumRankingRows)
	wantDegradedRows := minUint64(r.knownRankableCount, maximumRankingRows)
	if r.mode == rankingQualifiedCurrent && (r.reason != "" || r.population.unknownDueFailureOrFence != 0 || r.qualification.unresolved != 0 || uint64(len(r.rows)) != wantQualifiedRows) {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.qualified", "", "projection_mismatch")
	}
	qualificationTotal := r.qualification.notYetPassed + r.qualification.provisional + r.qualification.finalized + r.qualification.unresolved
	if qualificationTotal != r.population.trustedRankableMark ||
		r.totalPassers+r.qualifiedDayInvalid != r.qualification.provisional+r.qualification.finalized ||
		r.knownRankableCount+r.dayInvalidRankable != r.population.trustedRankableMark ||
		r.qualifiedDayInvalid > r.dayInvalidRankable {
		return invalidEvaluation(EvaluatorQualificationAccounting, "accounting.qualification", "", "identity_mismatch")
	}
	if r.uncertainty.bootstrapOrigin+r.uncertainty.postBootstrapGap+r.uncertainty.localInvalid !=
		r.population.unknownDueFailureOrFence+r.qualification.unresolved {
		return invalidEvaluation(EvaluatorUncertaintyAccounting, "accounting.uncertainty", "", "identity_mismatch")
	}
	for _, counts := range []featureDimensionAccounting{r.features.dayPercent, r.features.sessionVolume, r.features.fromOpenPercent, r.features.dayRange, r.features.activity30s, r.features.move30s} {
		if !counts.reconciles(r.population.universeTotal) {
			return invalidEvaluation(EvaluatorFeatureAccounting, "accounting.feature", "", "identity_mismatch")
		}
	}
	if !r.floats.reconciles(r.population.universeTotal) {
		return invalidEvaluation(EvaluatorFeatureAccounting, "accounting.float", "", "identity_mismatch")
	}
	if r.mode == rankingDegradedBootstrap && (r.knownRankableCount == 0 || (r.population.unknownDueFailureOrFence == 0 && r.qualification.unresolved == 0) || uint64(len(r.rows)) != wantDegradedRows) {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.degraded", "", "projection_mismatch")
	}
	if r.mode == rankingDegradedBootstrap && (r.uncertainty.postBootstrapGap != 0 || r.uncertainty.localInvalid != 0 || r.uncertainty.bootstrapOrigin == 0) {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.degraded", "", "non_bootstrap_uncertainty")
	}
	if r.mode == rankingDegradedBootstrap {
		wantReason := rankingReasonQualificationPending
		if r.population.unknownDueFailureOrFence != 0 {
			wantReason = rankingReasonIncompletePopulation
		}
		if r.reason != wantReason {
			return invalidEvaluation(EvaluatorRankingProjection, "ranking.reason", "", "degraded_reason_mismatch")
		}
	}
	if r.mode == rankingDegradedCurrent && (r.knownRankableCount == 0 || (r.population.unknownDueFailureOrFence == 0 && r.qualification.unresolved == 0) || uint64(len(r.rows)) != wantDegradedRows) {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.degraded_current", "", "projection_mismatch")
	}
	if r.mode == rankingDegradedCurrent {
		wantReason := rankingReasonQualificationPending
		if r.population.unknownDueFailureOrFence != 0 {
			wantReason = rankingReasonIncompletePopulation
		}
		if r.reason != wantReason {
			return invalidEvaluation(EvaluatorRankingProjection, "ranking.reason", "", "degraded_current_reason_mismatch")
		}
	}
	if r.mode == rankingUnavailable && r.reason != rankingReasonNoCommittedWatermark && r.reason != rankingReasonNoTrustedMarks && r.reason != rankingReasonIncompletePopulation {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.reason", "", "invalid_unavailable_reason")
	}
	if r.mode == rankingStale && r.reason != "" {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.reason", "", "invalid_stale_reason")
	}
	if r.mode == rankingSuppressed && r.reason != rankingReasonGlobalSuppression {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.reason", "", "invalid_suppressed_reason")
	}
	if (r.mode == rankingUnavailable || r.mode == rankingStale || r.mode == rankingSuppressed) && len(r.rows) != 0 {
		return invalidEvaluation(EvaluatorRankingProjection, "ranking.rows", "", "rows_in_noncurrent_projection")
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
	for i := range r.rows {
		r.rows[i].float.fact = cloneRankingFloatFact(r.rows[i].float.fact)
	}
	r.updates = nil
	return r
}
func aggregateEvaluationEqual(a, b aggregateEvaluationResult) bool {
	if a.at != b.at || a.mode != b.mode || a.reason != b.reason || a.population != b.population || a.qualification != b.qualification || a.features != b.features || a.floats != b.floats || a.uncertainty != b.uncertainty || a.populationTransition != b.populationTransition || a.totalPassers != b.totalPassers || a.knownRankableCount != b.knownRankableCount || a.dayInvalidRankable != b.dayInvalidRankable || a.qualifiedDayInvalid != b.qualifiedDayInvalid || a.invalidSupport != b.invalidSupport || a.tqIntentAvailable != b.tqIntentAvailable || len(a.rows) != len(b.rows) {
		return false
	}
	for i := range a.rows {
		if !aggregateRankingRowEqual(a.rows[i], b.rows[i]) {
			return false
		}
	}
	return true
}

func aggregateRankingRowEqual(a, b aggregateRankingRow) bool {
	aPercent, bPercent := a.float.fact.FreeFloatPercent, b.float.fact.FreeFloatPercent
	a.float.fact.FreeFloatPercent, b.float.fact.FreeFloatPercent = nil, nil
	if a != b || (aPercent == nil) != (bPercent == nil) {
		return false
	}
	return aPercent == nil || *aPercent == *bPercent
}

func cloneRankingFloatFact(fact reference.FloatFact) reference.FloatFact {
	if fact.FreeFloatPercent != nil {
		value := *fact.FreeFloatPercent
		fact.FreeFloatPercent = &value
	}
	return fact
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
		return reason == featureReasonBeforeFirstPrint || reason == featureReasonHistoryIncomplete || reason == featureReasonRollingWarmup || reason == featureReasonReferenceWarmup
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

func countFeatureResult(f *featureAccounting, p priceRangeFeatureResult, activity aggregateFeatureField, measurements mvpMeasurementResult) bool {
	if !countFeatureFields(
		[]aggregateFeatureField{p.dayPercent, measurements.sessionVolume, p.from4AMPercent, p.sessionRange, measurements.activity30s, measurements.move30s},
		[]*featureDimensionAccounting{&f.dayPercent, &f.sessionVolume, &f.fromOpenPercent, &f.dayRange, &f.activity30s, &f.move30s},
	) {
		return false
	}
	// Preserve the v1 diagnostic accounting projection until MVP-S2 replaces
	// its schema. These fields do not participate in the revised row model.
	fields := []aggregateFeatureField{p.dayPercent, p.from4AMPercent, p.hodDrawdown, p.sessionRange, p.rolling30, p.rolling60, activity}
	counts := []*featureDimensionAccounting{&f.dayPercent, &f.from4AMPercent, &f.hodDrawdown, &f.sessionRange, &f.rolling30, &f.rolling60, &f.activity}
	// DayPercent was already counted in the revised set.
	fields, counts = fields[1:], counts[1:]
	return countFeatureFields(fields, counts)
}

func countFeatureFields(fields []aggregateFeatureField, counts []*featureDimensionAccounting) bool {
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

func (e *Engine) floatForSymbol(symbol string) aggregateFloatField {
	fact, ok := e.floatLookup.Lookup(symbol)
	if !ok {
		return aggregateFloatField{status: aggregateFloatUnavailable, reason: "not_available"}
	}
	switch fact.Provenance {
	case reference.FloatFresh:
		return aggregateFloatField{status: aggregateFloatCurrent, fact: fact}
	case reference.FloatCache:
		return aggregateFloatField{status: aggregateFloatStale, reason: "cached_fallback", fact: fact}
	default:
		return aggregateFloatField{status: aggregateFloatInvalid, reason: "invalid_provenance"}
	}
}

func countFloatResult(accounting *floatAccounting, field aggregateFloatField) {
	switch field.status {
	case aggregateFloatCurrent:
		accounting.current++
	case aggregateFloatStale:
		accounting.stale++
	case aggregateFloatUnavailable:
		accounting.unavailable++
	default:
		accounting.invalid++
	}
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

func unknownUnlessInvalid(field, unknown aggregateFeatureField) aggregateFeatureField {
	if field.status == featureInvalid {
		return field
	}
	return unknown
}
