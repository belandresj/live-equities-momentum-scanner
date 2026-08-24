package engine

import (
	"math"
	"sort"
	"time"
)

const (
	maximumSessionExtremaPoints = sessionSeconds
)

type aggregateFeatureStatus string

const (
	featureWarming     aggregateFeatureStatus = "warming"
	featureCurrent     aggregateFeatureStatus = "current"
	featureUnavailable aggregateFeatureStatus = "unavailable"
	featureInvalid     aggregateFeatureStatus = "invalid"
)

type aggregateFeatureReason string

const (
	featureReasonNone                  aggregateFeatureReason = ""
	featureReasonBeforeFirstPrint      aggregateFeatureReason = "before_first_print"
	featureReasonHistoryIncomplete     aggregateFeatureReason = "history_incomplete"
	featureReasonPriorCloseUnavailable aggregateFeatureReason = "prior_close_unavailable"
	featureReasonRollingWarmup         aggregateFeatureReason = "rolling_warmup"
	featureReasonReferenceWarmup       aggregateFeatureReason = "reference_warmup"
	featureReasonZeroWidth             aggregateFeatureReason = "zero_width"
	featureReasonHistoricalConflict    aggregateFeatureReason = "historical_conflict"
	featureReasonInvalidInput          aggregateFeatureReason = "invalid_input"
	featureReasonStateBoundExceeded    aggregateFeatureReason = "state_bound_exceeded"
)

type aggregateFeatureField struct {
	status aggregateFeatureStatus
	reason aggregateFeatureReason
	value  float64
}

type priceRangeFeatureResult struct {
	at                                    time.Time
	dayPercent, fromOpenPercent, dayRange aggregateFeatureField
}

type extremaPoint struct {
	windowStart int64
	value       float64
}

// priceRangeFeatureState contains only the bounded sufficient evidence for the
// current From Open and Day Range fields. Mutable aggregate values remain
// solely in the canonical correction tail.
type priceRangeFeatureState struct {
	firstStart, finalizedThrough       int64
	firstOpen, sessionHigh, sessionLow float64
	hasFirst, hasSessionExtrema        bool
	// sessionHighs/sessionLows are cutoff-bearing sufficient extrema evidence,
	// not raw aggregates. They retain one value per accepted second so a record
	// beyond stalled committed T cannot change an older as-of query.
	sessionHighs, sessionLows []extremaPoint
	boundExceeded             bool
	result                    priceRangeFeatureResult
}

type priceRangeTailView struct {
	firstStart              int64
	firstOpen               float64
	hasFirst                bool
	sessionLow, sessionHigh float64
	hasSession              bool
}

func ensurePriceRangeState(state *symbolAggregateState) *priceRangeFeatureState {
	if state.priceRange == nil {
		state.priceRange = &priceRangeFeatureState{result: unavailablePriceRangeResult(time.Time{})}
	}
	return state.priceRange
}

func foldPriceRangeAggregate(state *symbolAggregateState, binding *installedBinding, record canonicalAggregate) {
	features := ensurePriceRangeState(state)
	start := record.windowStart.Unix()
	if !features.hasFirst || start < features.firstStart {
		features.firstStart, features.firstOpen, features.hasFirst = start, record.values.Open, true
	}
	if !features.hasSessionExtrema {
		features.sessionHigh, features.sessionLow, features.hasSessionExtrema = record.values.High, record.values.Low, true
	} else {
		features.sessionHigh = max(features.sessionHigh, record.values.High)
		features.sessionLow = min(features.sessionLow, record.values.Low)
	}
	features.sessionHighs = insertExtremaEvidence(features.sessionHighs, extremaPoint{start, record.values.High})
	features.sessionLows = insertExtremaEvidence(features.sessionLows, extremaPoint{start, record.values.Low})
	if end := record.windowEnd.Unix(); end > features.finalizedThrough {
		features.finalizedThrough = end
	}
	if len(features.sessionHighs) > maximumSessionExtremaPoints || len(features.sessionLows) > maximumSessionExtremaPoints {
		features.boundExceeded = true
	}
}

func insertExtremaEvidence(points []extremaPoint, point extremaPoint) []extremaPoint {
	index := sort.Search(len(points), func(i int) bool { return points[i].windowStart >= point.windowStart })
	if index < len(points) && points[index].windowStart == point.windowStart {
		points[index] = point
		return points
	}
	points = append(points, extremaPoint{})
	copy(points[index+1:], points[index:])
	points[index] = point
	return points
}

// runAggregateFeatureContributorLocked is the single fixed-order Component 3
// contributor call. C3-S3 extends the accepted C3-S1/S2 state and transition;
// it does not add another owner or evaluation path.
func (e *Engine) runAggregateFeatureContributorLocked(node *queueNode, code DispositionCode, reason DispositionReason) *aggregateEvaluationResult {
	if e.state.binding == nil {
		return nil
	}
	if ((node.kind == inputTimer || node.kind == inputReplayGroup) && code == DispositionTimerApplied) ||
		(node.kind == inputAggregateIngressFence && code == DispositionAggregateIngressFenceApplied) ||
		(node.kind == inputLiveCoverageFence && code == DispositionLiveCoverageFenceApplied) {
		target, evaluate := e.aggregateEvaluationTargetLocked(node)
		if !evaluate {
			// Timer facts still own canonical compaction and other semantics-neutral
			// maintenance even when the explicit aggregate projection gate is
			// closed. Maintenance cannot consume pending projection work.
			if node.kind != inputTimer || e.state.committedT == nil {
				return nil
			}
			target = *e.state.committedT
		}
		maintenanceStarted := e.evaluationTimingStart()
		maintainSymbol := func(index int) {
			symbol := &e.state.binding.symbols[index]
			if state := symbol.aggregates; state != nil {
				e.compactSymbolLocked(state, e.state.binding, symbol.symbol, node.admissionTime)
				if evaluate && !e.hiddenReplayWarmupLocked(node) {
					advanceSelectionMark(state, e.state.binding, target)
					// Qualification is canonical owner-local state. Advance its
					// bounded proof endpoints before staging selection so the cycle
					// reads scalars and never clones proof maps. This maintenance is
					// independent of publication acceptance; it creates no mark,
					// coverage, selection membership, or watermark.
					evaluateQualificationThrough(state, e.state.binding, target, node.admissionTime)
					if e.mode == RunModeLive {
						var invalid *invalidMarkEvidence
						if evidence, ok := e.invalidMarkBeforeLocked(index, target); ok {
							copyEvidence := evidence
							invalid = &copyEvidence
						}
						maintainCurrentFieldStatuses(e.state.binding, symbol, target, invalid)
					}
				}
			}
		}
		if e.hiddenReplayWarmupLocked(node) {
			for index := range e.state.replay.aggregateTouched {
				maintainSymbol(index)
			}
			clear(e.state.replay.aggregateTouched)
		} else {
			for index := range e.state.binding.symbols {
				maintainSymbol(index)
			}
			clear(e.state.replay.aggregateTouched)
		}
		maintenanceElapsed := e.evaluationTimingElapsed(maintenanceStarted)
		if e.hiddenReplayWarmupLocked(node) {
			return nil
		}
		if !evaluate {
			return nil
		}
		if e.duplicateTrustCorrectionCycleLocked(node, target) {
			return nil
		}
		started := e.evaluationTimingStart()
		e.recordAggregateEvaluationStartLocked(node, target)
		staged := e.stageAggregateEvaluationAtLocked(target, node.admissionTime)
		stageElapsed := e.evaluationTimingElapsed(started)
		e.state.evaluationTiming.EngineSequence = node.engineSequence
		e.state.evaluationTiming.Stage = stageElapsed
		e.state.evaluationTiming.Apply = 0
		e.state.evaluationTiming.Publication = 0
		if node.kind == inputAggregateIngressFence {
			e.state.fenceTiming.Target = target
			e.state.fenceTiming.SymbolMaintenance = maintenanceElapsed
			e.state.fenceTiming.EvaluationStage = stageElapsed
		}
		return &staged
	}
	if node.kind != inputAggregate {
		return nil
	}
	changed := code == DispositionAggregateInserted || code == DispositionAggregateRevised || code == DispositionAggregateWithdrawn
	structuralTrustChange := code == DispositionAggregateRejected && reason == ReasonStructural
	if !changed && !structuralTrustChange {
		return nil
	}
	index, ok := e.state.binding.index[node.aggregate.Symbol]
	if !ok {
		return nil
	}
	state := e.state.binding.symbols[index].aggregates
	if state == nil {
		return nil
	}
	if e.state.committedT != nil {
		qualification := ensureQualificationState(state)
		qualificationBefore := qualification.result
		first := node.aggregate.WindowStart.Add(time.Second)
		last := node.aggregate.WindowStart.Add(qualificationWindow)
		if first.Before(e.state.binding.sessionStart) {
			first = e.state.binding.sessionStart
		}
		if last.After(*e.state.committedT) {
			last = *e.state.committedT
		}
		markQualificationProofsDirty(qualification, first, last)
		if e.mode == RunModeLive {
			evaluateQualificationThrough(state, e.state.binding, *e.state.committedT, node.admissionTime)
			var invalid *invalidMarkEvidence
			if evidence, ok := e.invalidMarkBeforeLocked(index, *e.state.committedT); ok {
				copyEvidence := evidence
				invalid = &copyEvidence
			}
			maintainCurrentFieldStatuses(e.state.binding, &e.state.binding.symbols[index], *e.state.committedT, invalid)
			if structuralTrustChange {
				// [S,T) is right-open. Evidence at T or later cannot change
				// current-T population, rankability, or field trust.
				if !node.aggregate.WindowStart.Before(*e.state.committedT) {
					return nil
				}
				started := e.evaluationTimingStart()
				staged := e.stageAggregateEvaluationAtLocked(*e.state.committedT, node.admissionTime)
				elapsed := e.evaluationTimingElapsed(started)
				if aggregateEvaluationEqual(e.state.aggregateEvaluator.current, staged) {
					return nil
				}
				return e.commitTrustCorrectionCycleLocked(node, staged, elapsed)
			}
			if qualification.result != qualificationBefore {
				started := e.evaluationTimingStart()
				staged := e.stageAggregateEvaluationAtLocked(*e.state.committedT, node.admissionTime)
				return e.commitTrustCorrectionCycleLocked(node, staged, e.evaluationTimingElapsed(started))
			}
			return e.selectedTrustClosureCandidateLocked(index, *e.state.committedT)
		}
		if e.deferReplayAggregateProjectionLocked(node) {
			return nil
		}
		started := e.evaluationTimingStart()
		e.recordAggregateEvaluationStartLocked(node, *e.state.committedT)
		staged := e.stageAggregateEvaluationAtLocked(*e.state.committedT, node.admissionTime)
		e.state.evaluationTiming.EngineSequence = node.engineSequence
		e.state.evaluationTiming.Stage = e.evaluationTimingElapsed(started)
		e.state.evaluationTiming.Apply = 0
		e.state.evaluationTiming.Publication = 0
		return &staged
	}
	if e.deferReplayAggregateProjectionLocked(node) {
		return nil
	}
	return nil
}

func (e *Engine) commitTrustCorrectionCycleLocked(node *queueNode, staged aggregateEvaluationResult, stage time.Duration) *aggregateEvaluationResult {
	e.state.trustCorrectionRevision++
	e.state.lastTrustCorrectionCycle = aggregateTrustCycleIdentity{
		bindingIdentity: e.state.binding.identity,
		target:          staged.at,
		trustRevision:   e.state.trustCorrectionRevision,
	}
	view := &e.state.evaluationTiming
	view.EngineSequence, view.Source, view.Target = node.engineSequence, AggregateEvaluationTrustCorrection, staged.at
	view.Stage, view.Apply, view.Publication = stage, 0, 0
	view.Starts.TrustCorrection++
	return &staged
}

func (e *Engine) duplicateTrustCorrectionCycleLocked(node *queueNode, target time.Time) bool {
	last := e.state.lastTrustCorrectionCycle
	if node == nil || node.kind != inputTimer && node.kind != inputLiveCoverageFence ||
		e.mode != RunModeLive || e.state.binding == nil || e.state.aggregateProjectionPending ||
		(e.state.lifecycle != lifecycleLive && e.state.lifecycle != lifecycleHydrating) || last.bindingIdentity == "" ||
		last.bindingIdentity != e.state.binding.identity || last.trustRevision != e.state.trustCorrectionRevision || !last.target.Equal(target) {
		return false
	}
	return e.state.aggregateEvaluationDeadline == nil || !node.admissionTime.After(*e.state.aggregateEvaluationDeadline)
}

// maintainCurrentFieldStatuses owns the fixed full-population status scalars
// read by B2 selection. It proves support/reason only; current-product values
// remain selected-row work and are never calculated here.
func maintainCurrentFieldStatuses(binding *installedBinding, symbol *coreSymbol, at time.Time, invalid *invalidMarkEvidence) {
	if binding == nil || symbol == nil || symbol.aggregates == nil {
		return
	}
	state := symbol.aggregates
	mark, hasMark := latestMarkBeforeCompact(state, at)
	priceStatuses := priceRangeFieldStatuses(binding, symbol, at, mark, hasMark)
	price := ensurePriceRangeState(state)
	price.result.at = at
	price.result.fromOpenPercent = priceStatuses.fromOpenPercent
	price.result.dayRange = priceStatuses.dayRange
	measurements := ensureMVPMeasurementState(state)
	measurements.result = mvpMeasurementFieldStatuses(binding, state, at, invalid)
}

func (e *Engine) selectedTrustClosureCandidateLocked(symbolIndex int, at time.Time) *aggregateEvaluationResult {
	current := e.state.aggregateEvaluator.current
	if e.state.binding == nil || !current.at.Equal(at) || current.mode == rankingUnavailable || current.mode == rankingStale || current.mode == rankingSuppressed {
		return nil
	}
	rowIndex := -1
	for index := range current.rows {
		if current.rows[index].symbolIndex == symbolIndex {
			rowIndex = index
			break
		}
	}
	if rowIndex < 0 {
		return nil
	}
	state := e.state.binding.symbols[symbolIndex].aggregates
	candidate := cloneAggregateEvaluation(current)
	candidate.updates = make([]aggregateSymbolEvaluationUpdate, len(e.state.binding.symbols))
	if state == nil || candidate.enrichedRows == 0 {
		candidate.invalidSupport = true
		candidate.invalidSupportSymbol = candidate.rows[rowIndex].symbol
		candidate.invalidSupportReason = "selected_trust_closure_state"
		return &candidate
	}
	rowBefore := candidate.rows[rowIndex]
	mark, hasMark := e.latestSelectionMarkLocked(state, at)
	if !hasMark || mark.values.Close != rowBefore.last || at.Sub(mark.windowEnd) != rowBefore.markAge {
		// A correction to the ranking mark needs the ordinary full-population
		// cycle; it is not a display-field trust closure and cannot preserve order.
		return nil
	}
	candidate.rows[rowIndex].canonicalRevision = state.canonicalRevision
	update := &candidate.updates[symbolIndex]
	update.present, update.qualification = true, state.qualification
	update.committedLatest = committedMark(mark)
	candidate.enrichedRows--
	e.enrichSelectedRowLocked(&candidate, rowIndex, at)
	if candidate.invalidSupport {
		return &candidate
	}
	if !selectedFieldsLessTrusted(rowBefore, candidate.rows[rowIndex]) {
		return nil
	}
	return &candidate
}

func selectedFieldsLessTrusted(before, after aggregateRankingRow) bool {
	beforeFields := [...]aggregateFeatureField{before.sessionVolume, before.fromOpenPercent, before.dayRange, before.activity30s, before.move30s}
	afterFields := [...]aggregateFeatureField{after.sessionVolume, after.fromOpenPercent, after.dayRange, after.activity30s, after.move30s}
	for index := range beforeFields {
		beforeRank, afterRank := featureTrustRank(beforeFields[index]), featureTrustRank(afterFields[index])
		if afterRank < beforeRank || afterRank == beforeRank && afterFields[index].reason != beforeFields[index].reason {
			return true
		}
	}
	return false
}

func featureTrustRank(field aggregateFeatureField) int {
	switch field.status {
	case featureCurrent:
		return 3
	case featureWarming:
		return 2
	case featureUnavailable:
		return 1
	case featureInvalid:
		return 0
	default:
		return -1
	}
}

func priceRangeFieldStatuses(binding *installedBinding, symbol *coreSymbol, at time.Time, mark canonicalAggregate, hasMark bool) priceRangeFeatureResult {
	result := unavailablePriceRangeResult(at)
	state := symbol.aggregates
	features := ensurePriceRangeState(state)
	if !hasMark {
		return result
	}
	if features.boundExceeded {
		invalid := aggregateFeatureField{status: featureInvalid, reason: featureReasonStateBoundExceeded}
		result.fromOpenPercent, result.dayRange = invalid, invalid
		return result
	}
	tail := priceRangeTailViewAt(state, features, binding, at)
	firstStart, firstOpen, hasFirst := features.firstStart, features.firstOpen, features.hasFirst
	if tail.hasFirst && (!hasFirst || tail.firstStart < firstStart) {
		firstStart, firstOpen, hasFirst = tail.firstStart, tail.firstOpen, true
	}
	if hasFirst {
		result.fromOpenPercent = historyTrust(state, binding, binding.sessionStart, time.Unix(firstStart, 0).UTC(), false)
		if result.fromOpenPercent.status == featureCurrent && (!finitePositiveFeature(mark.values.Close) || !finitePositiveFeature(firstOpen)) {
			result.fromOpenPercent = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		}
	}
	sessionLow, sessionHigh, hasSession := 0.0, 0.0, false
	if features.hasSessionExtrema && features.finalizedThrough <= at.Unix() {
		sessionLow, sessionHigh, hasSession = features.sessionLow, features.sessionHigh, true
	} else {
		sessionLow, sessionHigh, hasSession = extremaEvidenceWithin(features.sessionLows, features.sessionHighs, binding.sessionStart.Unix(), at.Unix())
	}
	if tail.hasSession {
		sessionLow, sessionHigh, hasSession = mergeRangeEvidence(sessionLow, sessionHigh, hasSession, tail.sessionLow, tail.sessionHigh)
	}
	if hasSession {
		result.dayRange = historyTrust(state, binding, binding.sessionStart, at, false)
		if result.dayRange.status == featureCurrent {
			switch {
			case !finitePositiveFeature(mark.values.Close) || !finitePositiveFeature(sessionLow) || !finitePositiveFeature(sessionHigh) || sessionLow > sessionHigh:
				result.dayRange = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
			case sessionLow == sessionHigh:
				result.dayRange = aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}
			}
		}
	}
	return result
}

// aggregateEvaluationTargetLocked chooses the sole full-population projection
// boundary. Replay retains its existing group target. Live prefers a supported
// later target, then falls back to the committed watermark when pending
// aggregate work must be exposed without claiming unsupported time.
func (e *Engine) aggregateEvaluationTargetLocked(node *queueNode) (time.Time, bool) {
	if e.mode == RunModeReplay {
		if e.state.latestTarget == nil {
			return time.Time{}, false
		}
		return *e.state.latestTarget, true
	}
	if e.mode != RunModeLive || (node.kind != inputTimer && node.kind != inputAggregateIngressFence && node.kind != inputLiveCoverageFence) {
		return time.Time{}, false
	}
	if node.kind == inputTimer && node.timerPolicy == timerMaintenanceOnly {
		terminalEvaluation := e.state.lifecycle == lifecycleEnded &&
			(e.state.aggregateProjectionPending || e.state.aggregateEvaluator.current.mode != rankingUnavailable)
		if !terminalEvaluation {
			return time.Time{}, false
		}
	}
	if node.kind != inputTimer && e.state.latestTarget != nil && e.candidateTargetSupportedLocked(*e.state.latestTarget) {
		// An accepted ingress/live fence is itself the exact supporting prefix
		// boundary and receives one evaluation opportunity even at committed T.
		return *e.state.latestTarget, true
	}
	if e.state.latestTarget != nil && (e.state.committedT == nil || e.state.latestTarget.After(*e.state.committedT)) && e.candidateTargetSupportedLocked(*e.state.latestTarget) {
		return *e.state.latestTarget, true
	}
	if e.state.aggregateProjectionPending && e.state.committedT != nil && e.candidateTargetSupportedLocked(*e.state.committedT) {
		return *e.state.committedT, true
	}
	if node.kind == inputTimer && e.state.committedT != nil && e.candidateTargetSupportedLocked(*e.state.committedT) {
		deadlineDue := e.state.aggregateEvaluationDeadline != nil && node.admissionTime.After(*e.state.aggregateEvaluationDeadline)
		projectionMissing := !e.state.aggregateEvaluator.current.at.Equal(*e.state.committedT)
		lifecycleEvaluationDue := e.state.lifecycle != lifecycleLive && e.state.lifecycle != lifecycleHydrating &&
			e.state.lifecycle != lifecycleReplaying && e.state.aggregateEvaluator.current.mode != rankingUnavailable
		finalEvaluationDue := e.state.lifecycle == lifecycleEnded &&
			(e.state.evaluationTiming.Target.IsZero() || !e.state.evaluationTiming.Target.Equal(*e.state.committedT))
		if deadlineDue || projectionMissing || lifecycleEvaluationDue || finalEvaluationDue {
			return *e.state.committedT, true
		}
	}
	return time.Time{}, false
}

func (e *Engine) recordAggregateEvaluationStartLocked(node *queueNode, target time.Time) {
	view := &e.state.evaluationTiming
	view.Target = target
	switch node.kind {
	case inputLiveCoverageFence:
		view.Source = AggregateEvaluationLiveCoverageFence
		view.Starts.LiveCoverageFence++
	case inputAggregateIngressFence:
		view.Source = AggregateEvaluationIngressFence
		view.Starts.AggregateIngressFence++
	case inputReplayGroup:
		view.Source = AggregateEvaluationReplay
		view.Starts.Replay++
	default:
		view.Source = AggregateEvaluationTimer
		view.Starts.Timer++
	}
}

func (e *Engine) evaluatePriceRangeFeaturesLocked(binding *installedBinding, symbol *coreSymbol, at time.Time) priceRangeFeatureResult {
	if symbol == nil || symbol.aggregates == nil {
		return unavailablePriceRangeResult(at)
	}
	mark, hasMark := e.latestSelectionMarkLocked(symbol.aggregates, at)
	return evaluatePriceRangeFeaturesWithMark(binding, symbol, at, mark, hasMark)
}

func evaluatePriceRangeFeaturesWithMark(binding *installedBinding, symbol *coreSymbol, at time.Time, mark canonicalAggregate, hasMark bool) priceRangeFeatureResult {
	result := unavailablePriceRangeResult(at)
	state := symbol.aggregates
	features := ensurePriceRangeState(state)
	if !hasMark {
		return result
	}
	if symbol.prior.close > 0 {
		result.dayPercent = percentChange(mark.values.Close, symbol.prior.close)
	} else {
		result.dayPercent = aggregateFeatureField{status: featureUnavailable, reason: featureReasonPriorCloseUnavailable}
	}

	if features.boundExceeded {
		invalid := aggregateFeatureField{status: featureInvalid, reason: featureReasonStateBoundExceeded}
		result.fromOpenPercent, result.dayRange = invalid, invalid
		return result
	}
	tail := priceRangeTailViewAt(state, features, binding, at)
	firstStart, firstOpen, hasFirst := features.firstStart, features.firstOpen, features.hasFirst
	if tail.hasFirst && (!hasFirst || tail.firstStart < firstStart) {
		firstStart, firstOpen, hasFirst = tail.firstStart, tail.firstOpen, true
	}
	sessionLow, sessionHigh, hasSession := 0.0, 0.0, false
	if features.hasSessionExtrema && features.finalizedThrough <= at.Unix() {
		sessionLow, sessionHigh, hasSession = features.sessionLow, features.sessionHigh, true
	} else {
		sessionLow, sessionHigh, hasSession = extremaEvidenceWithin(features.sessionLows, features.sessionHighs, binding.sessionStart.Unix(), at.Unix())
	}
	if tail.hasSession {
		sessionLow, sessionHigh, hasSession = mergeRangeEvidence(sessionLow, sessionHigh, hasSession, tail.sessionLow, tail.sessionHigh)
	}

	if hasFirst {
		trust := historyTrust(state, binding, binding.sessionStart, time.Unix(firstStart, 0).UTC(), false)
		if trust.status == featureCurrent {
			result.fromOpenPercent = percentChange(mark.values.Close, firstOpen)
		} else {
			result.fromOpenPercent = trust
		}
	}
	if hasSession {
		trust := historyTrust(state, binding, binding.sessionStart, at, false)
		if trust.status == featureCurrent {
			result.dayRange = rangePosition(mark.values.Close, sessionLow, sessionHigh)
		} else {
			result.dayRange = trust
		}
	}
	return result
}

func unavailablePriceRangeResult(at time.Time) priceRangeFeatureResult {
	result := priceRangeFeatureResult{at: at}
	missing := aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
	result.dayPercent, result.fromOpenPercent, result.dayRange = missing, missing, missing
	return result
}

func (e *Engine) latestSelectionMarkLocked(state *symbolAggregateState, at time.Time) (canonicalAggregate, bool) {
	return latestMarkBeforeCompact(state, at)
}

func installCommittedSelection(state *symbolAggregateState, at time.Time, next *committedAggregateMark) {
	if state == nil {
		return
	}
	if !state.selectionAt.IsZero() && !state.selectionAt.Equal(at) {
		state.priorSelectionAt = state.selectionAt
		state.priorSelectionMark = cloneCommittedMark(state.committedLatest)
	}
	state.selectionAt, state.committedLatest = at, cloneCommittedMark(next)
}

func cloneCommittedMark(mark *committedAggregateMark) *committedAggregateMark {
	if mark == nil {
		return nil
	}
	result := *mark
	if mark.greatestLiveSupport != nil {
		position := *mark.greatestLiveSupport
		result.greatestLiveSupport = &position
	}
	return &result
}

func advanceSelectionMark(state *symbolAggregateState, binding *installedBinding, target time.Time) {
	if state == nil || target.IsZero() || !state.selectionAt.IsZero() && target.Before(state.selectionAt) || target.Equal(state.selectionAt) {
		return
	}
	var next *committedAggregateMark
	if state.selectionAt.IsZero() || target.Sub(state.selectionAt) != time.Second {
		next = repairSelectionMarkBefore(state, binding, target)
	} else {
		next = cloneCommittedMark(state.committedLatest)
		start := target.Add(-time.Second)
		if record := state.tail[start.Unix()]; record != nil && record.windowStart.Equal(start) {
			next = committedMark(*record)
		} else if state.olderLatest != nil && state.olderLatest.windowStart.Equal(start) && !state.prefix.latestUncertain {
			next = committedMark(*state.olderLatest)
		}
	}
	installCommittedSelection(state, target, next)
}

// repairSelectionMarkBefore is correction/jump work, never routine selection
// work. It scans at most the bounded canonical tail for one affected symbol and
// consults only independently trusted folded/latest scalars.
func repairSelectionMarkBefore(state *symbolAggregateState, _ *installedBinding, at time.Time) *committedAggregateMark {
	if state == nil || at.IsZero() {
		return nil
	}
	var result *committedAggregateMark
	consider := func(record canonicalAggregate) {
		if !record.windowStart.Before(at) || result != nil && !record.windowStart.After(result.windowStart) {
			return
		}
		result = committedMark(record)
	}
	if state.committedLatest != nil && state.committedLatest.windowStart.Before(at) {
		result = cloneCommittedMark(state.committedLatest)
	}
	if state.latest != nil {
		consider(state.latest.record)
	}
	if state.olderLatest != nil && !state.prefix.latestUncertain {
		consider(*state.olderLatest)
	}
	for _, record := range state.tail {
		if record != nil {
			consider(*record)
		}
	}
	return result
}

func latestMarkBeforeCompact(state *symbolAggregateState, at time.Time) (canonicalAggregate, bool) {
	if state == nil {
		return canonicalAggregate{}, false
	}
	if state.selectionAt.Equal(at) {
		return canonicalFromCommitted(state.committedLatest)
	}
	if state.priorSelectionAt.Equal(at) {
		return canonicalFromCommitted(state.priorSelectionMark)
	}
	// latest is maintained across the folded prefix and canonical tail. When it
	// is already strictly before the as-of boundary, no other record can be a
	// later eligible mark and the session tail need not be scanned. Equality is
	// deliberately excluded because [start, at) does not include the at bar.
	if state.latest != nil && state.latest.record.windowStart.Before(at) {
		if record := state.tail[state.latest.record.identity.start]; record != nil &&
			record.identity == state.latest.record.identity && record.windowStart.Equal(state.latest.record.windowStart) {
			// Tail is canonical; latest is a maintained projection. Selecting the
			// map value also contains a malformed/direct test mutation without a
			// full scan and prevents a stale copied value from reaching ranking.
			return *record, true
		}
		return state.latest.record, true
	}
	var result canonicalAggregate
	found := false
	if state.committedLatest != nil && state.committedLatest.windowStart.Before(at) {
		result = canonicalAggregate{
			identity:    aggregateIdentity{symbol: state.committedLatest.symbol, start: state.committedLatest.start},
			windowStart: state.committedLatest.windowStart, windowEnd: state.committedLatest.windowEnd,
			values: state.committedLatest.values, authority: state.committedLatest.authority, greatestLiveSupport: state.committedLatest.greatestLiveSupport,
		}
		found = true
	}
	if state.olderLatest != nil && state.olderLatest.windowStart.Before(at) {
		if !found || !state.olderLatest.windowStart.Before(result.windowStart) {
			result, found = *state.olderLatest, true
		}
	}
	return result, found
}

func canonicalFromCommitted(mark *committedAggregateMark) (canonicalAggregate, bool) {
	if mark == nil {
		return canonicalAggregate{}, false
	}
	return canonicalAggregate{
		identity: aggregateIdentity{symbol: mark.symbol, start: mark.start}, windowStart: mark.windowStart, windowEnd: mark.windowEnd,
		values: mark.values, authority: mark.authority, greatestLiveSupport: mark.greatestLiveSupport,
	}, true
}

func priceRangeTailViewAt(state *symbolAggregateState, features *priceRangeFeatureState, binding *installedBinding, at time.Time) priceRangeTailView {
	result := priceRangeTailView{}
	for _, record := range state.tail {
		if record == nil || !record.windowStart.Before(at) {
			continue
		}
		start := record.windowStart.Unix()
		if !result.hasFirst || result.firstStart > start {
			result.firstStart, result.firstOpen, result.hasFirst = start, record.values.Open, true
		}
		result.sessionLow, result.sessionHigh, result.hasSession = mergeRangeEvidence(result.sessionLow, result.sessionHigh, result.hasSession, record.values.Low, record.values.High)
	}
	return result
}

func mergeRangeEvidence(low, high float64, found bool, candidateLow, candidateHigh float64) (float64, float64, bool) {
	if !found {
		return candidateLow, candidateHigh, true
	}
	return min(low, candidateLow), max(high, candidateHigh), true
}

func extremaEvidenceWithin(lows, highs []extremaPoint, floor, upper int64) (float64, float64, bool) {
	low, high, found := 0.0, 0.0, false
	lowStart := sort.Search(len(lows), func(i int) bool { return lows[i].windowStart >= floor })
	lowEnd := sort.Search(len(lows), func(i int) bool { return lows[i].windowStart >= upper })
	for _, point := range lows[lowStart:lowEnd] {
		if !found || point.value < low {
			low = point.value
		}
		found = true
	}
	if !found {
		return 0, 0, false
	}
	foundHigh := false
	highStart := sort.Search(len(highs), func(i int) bool { return highs[i].windowStart >= floor })
	highEnd := sort.Search(len(highs), func(i int) bool { return highs[i].windowStart >= upper })
	for _, point := range highs[highStart:highEnd] {
		if !foundHigh || point.value > high {
			high = point.value
		}
		foundHigh = true
	}
	return low, high, foundHigh
}

func historyTrust(state *symbolAggregateState, binding *installedBinding, start, end time.Time, _ bool) aggregateFeatureField {
	if state.historicalConflict != nil && bitmapHasRange(state.historicalConflict, binding, start, end) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonHistoricalConflict}
	}
	if !exactAggregateCoverage(state, binding, start, end) {
		return aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
	}
	return aggregateFeatureField{status: featureCurrent}
}

func bitmapHasRange(bitmap *slotBitmap, binding *installedBinding, start, end time.Time) bool {
	if start.Before(binding.sessionStart) {
		start = binding.sessionStart
	}
	if end.After(binding.sessionEnd) {
		end = binding.sessionEnd
	}
	for slot := sessionSlot(binding, start); slot < sessionSlot(binding, end); slot++ {
		if bitmap.has(slot) {
			return true
		}
	}
	return false
}

func percentChange(value, base float64) aggregateFeatureField {
	if !finitePositiveFeature(value) || !finitePositiveFeature(base) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	result := 100 * (value/base - 1)
	if !finiteFeature(result) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	return aggregateFeatureField{status: featureCurrent, value: result}
}

func rangePosition(last, low, high float64) aggregateFeatureField {
	if !finitePositiveFeature(last) || !finitePositiveFeature(low) || !finitePositiveFeature(high) || high < low {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	if high == low {
		return aggregateFeatureField{status: featureUnavailable, reason: featureReasonZeroWidth}
	}
	result := 100 * (last - low) / (high - low)
	if !finiteFeature(result) || result < 0 || result > 100 {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	return aggregateFeatureField{status: featureCurrent, value: result}
}

func finitePositiveFeature(value float64) bool { return value > 0 && finiteFeature(value) }
func finiteFeature(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
