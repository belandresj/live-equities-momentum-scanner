package engine

import (
	"math"
	"sort"
	"time"
)

const (
	maximumExtremaPointsPerDeque = 60 * 60
	maximumPriceRangeDequePoints = 2 * maximumExtremaPointsPerDeque
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
	featureReasonNoAggregateInTarget   aggregateFeatureReason = "no_aggregate_in_target"
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
	at                                      time.Time
	dayPercent, from4AMPercent, hodDrawdown aggregateFeatureField
	sessionRange, rolling30, rolling60      aggregateFeatureField
}

type extremaPoint struct {
	windowStart int64
	value       float64
}

// priceRangeFeatureState is the one bounded Component 3 price/range substate
// inside the canonical symbol owner. The two monotone deques are shared by the
// 30- and 60-minute queries; mutable aggregate values remain solely in the
// Component 2 tail.
type priceRangeFeatureState struct {
	firstStart, rollingFloor, finalizedThrough int64
	firstOpen, sessionHigh, sessionLow         float64
	hasFirst, hasSessionExtrema                bool
	highs, lows                                []extremaPoint
	// sessionHighs/sessionLows are cutoff-bearing sufficient extrema evidence,
	// not raw aggregates. They retain one value per accepted second so a record
	// beyond stalled committed T cannot change an older as-of query.
	sessionHighs, sessionLows []extremaPoint
	boundExceeded             bool
	result                    priceRangeFeatureResult
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
		floor := max(binding.sessionStart.Unix(), end-int64((60*time.Minute)/time.Second))
		if floor > features.rollingFloor {
			features.rollingFloor = floor
			features.highs = trimExtrema(features.highs, floor)
			features.lows = trimExtrema(features.lows, floor)
		}
	}
	if len(features.highs) > maximumExtremaPointsPerDeque {
		features.highs = features.highs[len(features.highs)-maximumExtremaPointsPerDeque:]
		features.boundExceeded = true
	}
	if len(features.lows) > maximumExtremaPointsPerDeque {
		features.lows = features.lows[len(features.lows)-maximumExtremaPointsPerDeque:]
		features.boundExceeded = true
	}
	if len(features.highs)+len(features.lows) > maximumPriceRangeDequePoints {
		features.boundExceeded = true
	}
	if len(features.sessionHighs) > sessionSeconds || len(features.sessionLows) > sessionSeconds {
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

// insertMonotonePoint produces the same canonical suffix-extrema sequence as
// chronological append, including for out-of-order finalized historical fill.
func insertMonotonePoint(points []extremaPoint, point extremaPoint, maximum bool) []extremaPoint {
	index := sort.Search(len(points), func(i int) bool { return points[i].windowStart >= point.windowStart })
	if index < len(points) && points[index].windowStart == point.windowStart {
		points = append(points[:index], points[index+1:]...)
	}
	index = sort.Search(len(points), func(i int) bool { return points[i].windowStart > point.windowStart })
	dominatedByNext := index < len(points) && ((maximum && points[index].value >= point.value) || (!maximum && points[index].value <= point.value))
	if dominatedByNext {
		return points
	}
	removeFrom := index
	for removeFrom > 0 {
		value := points[removeFrom-1].value
		if (maximum && value > point.value) || (!maximum && value < point.value) {
			break
		}
		removeFrom--
	}
	result := make([]extremaPoint, 0, len(points)-(index-removeFrom)+1)
	result = append(result, points[:removeFrom]...)
	result = append(result, point)
	result = append(result, points[index:]...)
	return result
}

func trimExtrema(points []extremaPoint, floor int64) []extremaPoint {
	index := sort.Search(len(points), func(i int) bool { return points[i].windowStart >= floor })
	if index == 0 {
		return points
	}
	copy(points, points[index:])
	return points[:len(points)-index]
}

// runAggregateFeatureContributorLocked is the single fixed-order Component 3
// contributor call. C3-S3 extends the accepted C3-S1/S2 state and transition;
// it does not add another owner or evaluation path.
func (e *Engine) runAggregateFeatureContributorLocked(node *queueNode, code DispositionCode, reason DispositionReason) *aggregateEvaluationResult {
	if e.state.binding == nil {
		return nil
	}
	if node.kind == inputTimer && code == DispositionTimerApplied {
		target := *e.state.latestTarget
		maintenanceAt := target
		if e.state.committedT != nil {
			// Until the sole central gate accepts a later target, future support
			// must not evict the retained state for the current committed T.
			maintenanceAt = *e.state.committedT
		}
		for index := range e.state.binding.symbols {
			if state := e.state.binding.symbols[index].aggregates; state != nil {
				e.compactSymbolLocked(state, e.state.binding, node.admissionTime)
				maintainActivityState(state, e.state.binding, node.admissionTime, maintenanceAt)
			}
		}
		staged := e.stageAggregateEvaluationAtLocked(target, node.admissionTime)
		return &staged
	}
	if node.kind != inputAggregate {
		return nil
	}
	changed := code == DispositionAggregateInserted || code == DispositionAggregateRevised || code == DispositionAggregateWithdrawn ||
		(code == DispositionAggregateRejected && reason == ReasonHistoricalLiveConflict)
	if !changed {
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
	ensurePriceRangeState(state)
	activity := ensureActivityState(state)
	if code == DispositionAggregateWithdrawn {
		removeFoldedActivityTarget(activity, e.state.binding, node.aggregate.WindowStart)
	}
	blockEnd := activityBlockEnd(e.state.binding, node.aggregate.WindowStart)
	recomputeMutableActivityBlock(state, e.state.binding, blockEnd)
	boundary := node.admissionTime
	if e.state.committedT != nil {
		boundary = *e.state.committedT
	} else if e.state.latestTarget != nil {
		boundary = *e.state.latestTarget
	}
	maintainActivityState(state, e.state.binding, node.admissionTime, boundary)
	if e.state.committedT != nil {
		qualification := ensureQualificationState(state)
		first := node.aggregate.WindowStart.Add(time.Second)
		last := node.aggregate.WindowStart.Add(qualificationWindow)
		if first.Before(e.state.binding.sessionStart) {
			first = e.state.binding.sessionStart
		}
		if last.After(*e.state.committedT) {
			last = *e.state.committedT
		}
		markQualificationProofsDirty(qualification, first, last)
		staged := e.stageAggregateEvaluationAtLocked(*e.state.committedT, node.admissionTime)
		return &staged
	}
	return nil
}

func evaluatePriceRangeFeatures(binding *installedBinding, symbol *coreSymbol, at time.Time) priceRangeFeatureResult {
	result := unavailablePriceRangeResult(at)
	state := symbol.aggregates
	if state == nil {
		return result
	}
	features := ensurePriceRangeState(state)
	mark, hasMark := latestMarkBefore(state, at)
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
		result.from4AMPercent, result.hodDrawdown = invalid, invalid
		result.sessionRange, result.rolling30, result.rolling60 = invalid, invalid, invalid
		return result
	}
	firstStart, firstOpen, hasFirst := features.firstStart, features.firstOpen, features.hasFirst
	sessionLow, sessionHigh, hasSession := extremaEvidenceWithin(features.sessionLows, features.sessionHighs, binding.sessionStart.Unix(), at.Unix())
	for _, record := range state.tail {
		if !record.windowStart.Before(at) {
			continue
		}
		start := record.windowStart.Unix()
		if !hasFirst || start < firstStart {
			firstStart, firstOpen, hasFirst = start, record.values.Open, true
		}
		if !hasSession {
			sessionLow, sessionHigh, hasSession = record.values.Low, record.values.High, true
		} else {
			sessionLow, sessionHigh = min(sessionLow, record.values.Low), max(sessionHigh, record.values.High)
		}
	}

	if hasFirst {
		trust := historyTrust(state, binding, binding.sessionStart, time.Unix(firstStart, 0).UTC(), false)
		if trust.status == featureCurrent {
			result.from4AMPercent = percentChange(mark.values.Close, firstOpen)
		} else {
			result.from4AMPercent = trust
		}
	}
	if hasSession {
		trust := historyTrust(state, binding, binding.sessionStart, at, false)
		if trust.status == featureCurrent {
			result.hodDrawdown = boundedPercentChange(mark.values.Close, sessionHigh, -100, 0)
			result.sessionRange = rangePosition(mark.values.Close, sessionLow, sessionHigh)
		} else {
			result.hodDrawdown, result.sessionRange = trust, trust
		}
	}
	result.rolling30 = evaluateRollingRange(state, binding, mark.values.Close, at, 30*time.Minute)
	result.rolling60 = evaluateRollingRange(state, binding, mark.values.Close, at, 60*time.Minute)
	return result
}

func unavailablePriceRangeResult(at time.Time) priceRangeFeatureResult {
	result := priceRangeFeatureResult{at: at}
	missing := aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
	result.dayPercent, result.from4AMPercent, result.hodDrawdown = missing, missing, missing
	result.sessionRange, result.rolling30, result.rolling60 = missing, missing, missing
	return result
}

func latestMarkBefore(state *symbolAggregateState, at time.Time) (canonicalAggregate, bool) {
	var result canonicalAggregate
	found := false
	if state.committedLatest != nil && state.committedLatest.windowStart.Before(at) {
		result = canonicalAggregate{
			identity:    aggregateIdentity{start: state.committedLatest.start},
			windowStart: state.committedLatest.windowStart, windowEnd: state.committedLatest.windowEnd,
			values: AggregateValues{Close: state.committedLatest.close},
		}
		found = true
	}
	if state.olderLatest != nil && state.olderLatest.windowStart.Before(at) {
		if !found || state.olderLatest.windowStart.After(result.windowStart) {
			result, found = *state.olderLatest, true
		}
	}
	for _, record := range state.tail {
		if record.windowStart.Before(at) && (!found || record.windowStart.After(result.windowStart)) {
			result, found = *record, true
		}
	}
	return result, found
}

func evaluateRollingRange(state *symbolAggregateState, binding *installedBinding, last float64, at time.Time, width time.Duration) aggregateFeatureField {
	start := at.Add(-width)
	if start.Before(binding.sessionStart) {
		start = binding.sessionStart
	}
	trust := historyTrust(state, binding, start, at, true)
	if trust.status != featureCurrent {
		return trust
	}
	features := ensurePriceRangeState(state)
	low, high, found := extremaEvidenceWithin(features.sessionLows, features.sessionHighs, start.Unix(), at.Unix())
	for _, record := range state.tail {
		if record.windowStart.Before(start) || !record.windowStart.Before(at) {
			continue
		}
		if !found {
			low, high, found = record.values.Low, record.values.High, true
		} else {
			low, high = min(low, record.values.Low), max(high, record.values.High)
		}
	}
	if !found {
		return aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint}
	}
	return rangePosition(last, low, high)
}

func extremaWithin(lows, highs []extremaPoint, floor int64) (float64, float64, bool) {
	lowIndex := sort.Search(len(lows), func(i int) bool { return lows[i].windowStart >= floor })
	highIndex := sort.Search(len(highs), func(i int) bool { return highs[i].windowStart >= floor })
	if lowIndex == len(lows) || highIndex == len(highs) {
		return 0, 0, false
	}
	return lows[lowIndex].value, highs[highIndex].value, true
}

func extremaEvidenceWithin(lows, highs []extremaPoint, floor, upper int64) (float64, float64, bool) {
	low, high, found := 0.0, 0.0, false
	for _, point := range lows {
		if point.windowStart < floor || point.windowStart >= upper {
			continue
		}
		if !found || point.value < low {
			low = point.value
		}
		found = true
	}
	if !found {
		return 0, 0, false
	}
	foundHigh := false
	for _, point := range highs {
		if point.windowStart < floor || point.windowStart >= upper {
			continue
		}
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

func boundedPercentChange(value, base, lower, upper float64) aggregateFeatureField {
	result := percentChange(value, base)
	if result.status == featureCurrent && (result.value < lower || result.value > upper) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	return result
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
