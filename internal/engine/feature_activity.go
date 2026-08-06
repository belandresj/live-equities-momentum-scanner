package engine

import (
	"math"
	"math/big"
	"sort"
	"time"
)

const (
	activityBlockDuration                       = 30 * time.Second
	minimumActivityReferences                   = 10
	maximumActivityReferences                   = 1920
	maximumActivityEvaluationReferences         = 119
	maximumMutableActivityBlockIDs              = 33
	maximumActivityTargetBlocks                 = 1920
	maximumActivityTargetContributions          = 57_600
	minimumActivityReferenceTx          float64 = 100
)

type activityFeatureResult struct {
	at                                     time.Time
	activity                               aggregateFeatureField
	targetTransactions, targetExpansionBPS float64
	transactionPercentile                  float64
	expansionPercentile                    float64
	referenceCount                         int
}

// activityBlockSummary is sufficient block mathematics, not a raw bar copy.
// high/low are retained so an out-of-order historical fill can be combined
// exactly after Component 2 has folded the earlier identities.
type activityBlockSummary struct {
	end            int64
	transactionSum activityExactSum
	transactions   float64
	high, low      float64
	expansionBPS   float64
	aggregateCount uint8
	invalid        bool
}

// activityExactSum is a fixed-size nonnegative binary superaccumulator. It
// makes Volume/ATS summation independent of permitted delivery order without
// retaining per-second values. Limbs use the float64 subnormal quantum
// 2^-1074; the final limb also contains the bounded carry from one 30-second
// block.
type activityExactSum [34]uint64

type activityMutableBlock struct {
	folded  activityBlockSummary
	current activityBlockSummary
}

// activityFoldedTargetBlock retains only the Activity operands needed to
// reconstruct an arbitrary whole-second [T-30s,T) after Component 2 discards
// the canonical records. Slot identity is implicit in the session-aligned
// block and bit position. It is not a second canonical aggregate record.
type activityFoldedTargetBlock struct {
	transactions [30]float64
	highs        [30]float64
	lows         [30]float64
	present      uint32
	invalid      uint32
}

// activityFeatureState is owned by the same canonical symbol graph and called
// through the same contributor transition as the accepted C3-S1 state.
// references contains at most one sufficient summary per possible reference
// position in the exact rolling interval. mutable contains only block IDs with
// a folded prefix that can still be corrected at the inclusive H boundary.
type activityFeatureState struct {
	references                map[int64]activityBlockSummary
	mutable                   map[int64]activityMutableBlock
	foldedTargets             map[int64]activityFoldedTargetBlock
	foldedTargetContributions int
	boundExceeded             bool
	result                    activityFeatureResult
}

func ensureActivityState(state *symbolAggregateState) *activityFeatureState {
	if state.activity == nil {
		state.activity = &activityFeatureState{
			references: make(map[int64]activityBlockSummary),
			mutable:    make(map[int64]activityMutableBlock),
			result:     unavailableActivityResult(time.Time{}),
		}
	}
	return state.activity
}

func activityBlockEnd(binding *installedBinding, windowStart time.Time) time.Time {
	offset := windowStart.Sub(binding.sessionStart)
	block := offset / activityBlockDuration
	return binding.sessionStart.Add((block + 1) * activityBlockDuration)
}

// foldActivityAggregate receives the exact record synchronously from
// Component 2 immediately before that owner discards it. It retains only
// sufficient block arithmetic and the aligned block identity.
func foldActivityAggregate(state *symbolAggregateState, binding *installedBinding, record canonicalAggregate, now time.Time) {
	activity := ensureActivityState(state)
	if activity.boundExceeded {
		return
	}
	retainFoldedActivityTarget(activity, binding, record)
	if activity.boundExceeded {
		return
	}
	end := activityBlockEnd(binding, record.windowStart).Unix()
	if summary, exists := activity.references[end]; exists {
		summary = finishActivitySummary(addActivityAggregate(summary, record.values))
		activity.references[end] = summary
		return
	}
	block, ok := activity.mutable[end]
	if !ok {
		blockEnd := time.Unix(end, 0).UTC()
		if now.After(blockEnd.Add(correctionHorizon)) {
			summary := activityBlockSummary{end: end, low: math.Inf(1)}
			activity.references[end] = finishActivitySummary(addActivityAggregate(summary, record.values))
			return
		}
		block.folded = activityBlockSummary{end: end, low: math.Inf(1)}
		block.current = activityBlockSummary{end: end, low: math.Inf(1)}
	}
	block.folded = addActivityAggregate(block.folded, record.values)
	if block.current.aggregateCount == 0 {
		block.current = finishActivitySummary(block.folded)
	}
	activity.mutable[end] = block
	if len(activity.mutable) > maximumMutableActivityBlockIDs {
		failActivityBound(activity)
	}
}

func retainFoldedActivityTarget(activity *activityFeatureState, binding *installedBinding, record canonicalAggregate) {
	if activity.foldedTargets == nil {
		activity.foldedTargets = make(map[int64]activityFoldedTargetBlock)
	}
	end := activityBlockEnd(binding, record.windowStart)
	blockKey := end.Unix()
	slot := int(record.windowStart.Sub(end.Add(-activityBlockDuration)) / time.Second)
	if slot < 0 || slot >= 30 {
		failActivityBound(activity)
		return
	}
	block := activity.foldedTargets[blockKey]
	mask := uint32(1) << uint(slot)
	if block.present&mask == 0 {
		activity.foldedTargetContributions++
	}
	block.present |= mask
	block.invalid &^= mask
	block.transactions[slot], block.highs[slot], block.lows[slot] = 0, record.values.High, record.values.Low
	if record.values.AverageTradeSize <= 0 {
		block.invalid |= mask
	} else {
		transactions := record.values.Volume / float64(record.values.AverageTradeSize)
		if !finiteFeature(transactions) {
			block.invalid |= mask
		} else {
			block.transactions[slot] = transactions
		}
	}
	activity.foldedTargets[blockKey] = block
	if len(activity.foldedTargets) > maximumActivityTargetBlocks ||
		activity.foldedTargetContributions > maximumActivityTargetContributions {
		failActivityBound(activity)
	}
}

func removeFoldedActivityTarget(activity *activityFeatureState, binding *installedBinding, windowStart time.Time) {
	if activity == nil || activity.foldedTargets == nil {
		return
	}
	end := activityBlockEnd(binding, windowStart)
	blockKey := end.Unix()
	block, ok := activity.foldedTargets[blockKey]
	if !ok {
		return
	}
	slot := int(windowStart.Sub(end.Add(-activityBlockDuration)) / time.Second)
	if slot < 0 || slot >= 30 {
		failActivityBound(activity)
		return
	}
	mask := uint32(1) << uint(slot)
	if block.present&mask == 0 {
		return
	}
	block.present &^= mask
	block.invalid &^= mask
	block.transactions[slot], block.highs[slot], block.lows[slot] = 0, 0, 0
	activity.foldedTargetContributions--
	if block.present == 0 {
		delete(activity.foldedTargets, blockKey)
	} else {
		activity.foldedTargets[blockKey] = block
	}
}

func addActivityAggregate(summary activityBlockSummary, values AggregateValues) activityBlockSummary {
	if summary.invalid {
		return summary
	}
	if values.AverageTradeSize <= 0 {
		summary.invalid = true
		return summary
	}
	transactions := values.Volume / float64(values.AverageTradeSize)
	if !finiteFeature(transactions) {
		summary.invalid = true
		return summary
	}
	if !summary.transactionSum.add(transactions) {
		summary.invalid = true
		return summary
	}
	if summary.aggregateCount == 0 {
		summary.high, summary.low = values.High, values.Low
	} else {
		summary.high, summary.low = max(summary.high, values.High), min(summary.low, values.Low)
	}
	if summary.aggregateCount == math.MaxUint8 {
		summary.invalid = true
		return summary
	}
	summary.aggregateCount++
	return summary
}

func finishActivitySummary(summary activityBlockSummary) activityBlockSummary {
	transactions, finite := summary.transactionSum.float64()
	summary.transactions = transactions
	if summary.invalid || summary.aggregateCount == 0 || !finite || !finiteFeature(summary.transactions) ||
		!finitePositiveFeature(summary.high) || !finitePositiveFeature(summary.low) || summary.high < summary.low {
		summary.invalid = true
		return summary
	}
	summary.expansionBPS = 10_000 * math.Log(summary.high/summary.low)
	if !finiteFeature(summary.expansionBPS) || summary.expansionBPS < 0 {
		summary.invalid = true
	}
	return summary
}

func (sum *activityExactSum) add(value float64) bool {
	if value < 0 || !finiteFeature(value) {
		return false
	}
	bits := math.Float64bits(value)
	exponent := int((bits >> 52) & 0x7ff)
	mantissa := bits & ((uint64(1) << 52) - 1)
	shift := 0
	if exponent != 0 {
		mantissa |= uint64(1) << 52
		shift = exponent - 1
	}
	if mantissa == 0 {
		return true
	}
	word, bit := shift/64, uint(shift%64)
	old := sum[word]
	sum[word] += mantissa << bit
	carry := uint64(0)
	if sum[word] < old {
		carry = 1
	}
	if bit != 0 {
		word++
		old = sum[word]
		sum[word] += mantissa >> (64 - bit)
		nextCarry := uint64(0)
		if sum[word] < old {
			nextCarry = 1
		}
		if carry != 0 {
			old = sum[word]
			sum[word]++
			if sum[word] < old {
				nextCarry = 1
			}
		}
		carry = nextCarry
	}
	for carry != 0 {
		word++
		if word >= len(sum) {
			return false
		}
		old = sum[word]
		sum[word] += carry
		if sum[word] >= old {
			carry = 0
		} else {
			carry = 1
		}
	}
	return true
}

func (sum activityExactSum) float64() (float64, bool) {
	var integer big.Int
	for index := len(sum) - 1; index >= 0; index-- {
		integer.Lsh(&integer, 64)
		if sum[index] != 0 {
			var word big.Int
			word.SetUint64(sum[index])
			integer.Add(&integer, &word)
		}
	}
	value := new(big.Float).SetPrec(53).SetMode(big.ToNearestEven).SetInt(&integer)
	value.SetMantExp(value, -1074)
	result, _ := value.Float64()
	return result, finiteFeature(result)
}

func recomputeMutableActivityBlock(state *symbolAggregateState, binding *installedBinding, blockEnd time.Time) {
	activity := ensureActivityState(state)
	if activity.boundExceeded {
		return
	}
	key := blockEnd.Unix()
	if _, immutable := activity.references[key]; immutable {
		// A deep historical insert was already combined synchronously while
		// Component 2 still supplied its exact record to foldActivityAggregate.
		return
	}
	block, ok := activity.mutable[key]
	if !ok {
		block.folded = activityBlockSummary{end: key, low: math.Inf(1)}
	}
	current := block.folded
	if current.aggregateCount == 0 {
		current.low = math.Inf(1)
	}
	for _, record := range state.tail {
		if activityBlockEnd(binding, record.windowStart).Equal(blockEnd) {
			current = addActivityAggregate(current, record.values)
		}
	}
	block.current = finishActivitySummary(current)
	activity.mutable[key] = block
	if len(activity.mutable) > maximumMutableActivityBlockIDs {
		failActivityBound(activity)
	}
}

func maintainActivityState(state *symbolAggregateState, binding *installedBinding, now, at time.Time) {
	activity := ensureActivityState(state)
	if activity.boundExceeded {
		return
	}
	finalizeActivityMutable(activity, now)
	expireActivityReferences(activity, binding, at)
	if len(activity.references) > maximumActivityReferences || len(activity.mutable) > maximumMutableActivityBlockIDs ||
		len(activity.foldedTargets) > maximumActivityTargetBlocks || activity.foldedTargetContributions > maximumActivityTargetContributions {
		failActivityBound(activity)
	}
}

func finalizeActivityMutable(activity *activityFeatureState, now time.Time) {
	keys := make([]int64, 0, len(activity.mutable))
	for end := range activity.mutable {
		keys = append(keys, end)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, end := range keys {
		blockEnd := time.Unix(end, 0).UTC()
		// Equality remains mutable; finalization is strictly beyond H.
		if !now.After(blockEnd.Add(correctionHorizon)) {
			continue
		}
		summary := activity.mutable[end].current
		activity.references[end] = summary
		delete(activity.mutable, end)
	}
}

func expireActivityReferences(activity *activityFeatureState, binding *installedBinding, at time.Time) {
	floor := at.Add(-60 * time.Minute)
	if floor.Before(binding.sessionStart) {
		floor = binding.sessionStart
	}
	for end := range activity.references {
		blockEnd := time.Unix(end, 0).UTC()
		blockStart := blockEnd.Add(-activityBlockDuration)
		// Future sufficient blocks remain available while committed T is stalled;
		// only evidence older than the committed-boundary floor is expired.
		if blockStart.Before(floor) || blockStart.Before(binding.sessionStart) || blockEnd.After(binding.sessionEnd) {
			delete(activity.references, end)
		}
	}
}

func failActivityBound(activity *activityFeatureState) {
	clear(activity.references)
	clear(activity.mutable)
	clear(activity.foldedTargets)
	activity.foldedTargetContributions = 0
	activity.boundExceeded = true
}

func evaluateActivityFeatures(binding *installedBinding, state *symbolAggregateState, at time.Time) activityFeatureResult {
	result := unavailableActivityResult(at)
	activity := ensureActivityState(state)
	if activity.boundExceeded {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonStateBoundExceeded}
		return result
	}
	targetStart := at.Add(-activityBlockDuration)
	if targetStart.Before(binding.sessionStart) {
		targetStart = binding.sessionStart
	}
	targetEnd := at
	if targetEnd.After(binding.sessionEnd) {
		targetEnd = binding.sessionEnd
	}
	if !targetStart.Before(targetEnd) {
		return result
	}
	if !activityCoverageTrustworthy(state, binding, targetStart, targetEnd) {
		result.activity = aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
		return result
	}
	target := activityBlockSummary{low: math.Inf(1)}
	for second := targetStart; second.Before(targetEnd); second = second.Add(time.Second) {
		if record := state.tail[second.Unix()]; record != nil {
			target = addActivityAggregate(target, record.values)
			continue
		}
		var ok bool
		target, ok = addFoldedActivityTarget(target, activity, binding, second)
		if !ok {
			continue
		}
	}
	if target.invalid {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		return result
	}
	if target.aggregateCount == 0 {
		result.activity = aggregateFeatureField{status: featureUnavailable, reason: featureReasonNoAggregateInTarget}
		return result
	}
	target = finishActivitySummary(target)
	if target.invalid {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		return result
	}
	result.targetTransactions = target.transactions
	result.targetExpansionBPS = target.expansionBPS

	floor := at.Add(-60 * time.Minute)
	if floor.Before(binding.sessionStart) {
		floor = binding.sessionStart
	}
	upper := at.Add(-activityBlockDuration)
	transactionReferences := make([]float64, 0, maximumActivityEvaluationReferences)
	expansionReferences := make([]float64, 0, maximumActivityEvaluationReferences)
	for blockStart := firstAlignedActivityStart(binding, floor); blockStart.Add(activityBlockDuration).Compare(upper) <= 0; blockStart = blockStart.Add(activityBlockDuration) {
		blockEnd := blockStart.Add(activityBlockDuration)
		if blockEnd.After(binding.sessionEnd) {
			break
		}
		if !activityCoverageTrustworthy(state, binding, blockStart, blockEnd) {
			result.activity = aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
			return result
		}
		summary := activityComponentsForBlock(state, binding, blockStart, blockEnd)
		if summary.invalid {
			result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
			return result
		}
		if summary.aggregateCount == 0 {
			continue
		}
		summary = finishActivitySummary(summary)
		if summary.invalid {
			result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
			return result
		}
		if summary.transactions >= minimumActivityReferenceTx {
			transactionReferences = append(transactionReferences, summary.transactions)
			expansionReferences = append(expansionReferences, summary.expansionBPS)
		}
	}
	result.referenceCount = len(transactionReferences)
	if result.referenceCount < minimumActivityReferences {
		result.activity = aggregateFeatureField{status: featureWarming, reason: featureReasonReferenceWarmup}
		return result
	}
	result.transactionPercentile = empiricalPercentile(transactionReferences, target.transactions)
	result.expansionPercentile = empiricalPercentile(expansionReferences, target.expansionBPS)
	value := math.Sqrt(result.transactionPercentile * result.expansionPercentile)
	if !finiteFeature(value) || value < 0 || value > 100 {
		result.activity = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		return result
	}
	result.activity = aggregateFeatureField{status: featureCurrent, value: value}
	return result
}

func applyActivityResult(activity *activityFeatureState, result activityFeatureResult) {
	activity.result = result
	pruneFoldedActivityTargets(activity, result.at.Add(-activityBlockDuration))
}

func addFoldedActivityTarget(summary activityBlockSummary, activity *activityFeatureState, binding *installedBinding, second time.Time) (activityBlockSummary, bool) {
	if activity.foldedTargets == nil {
		return summary, false
	}
	end := activityBlockEnd(binding, second)
	block, ok := activity.foldedTargets[end.Unix()]
	if !ok {
		return summary, false
	}
	slot := int(second.Sub(end.Add(-activityBlockDuration)) / time.Second)
	if slot < 0 || slot >= 30 {
		return summary, false
	}
	mask := uint32(1) << uint(slot)
	if block.present&mask == 0 {
		return summary, false
	}
	if block.invalid&mask != 0 {
		summary.invalid = true
		return summary, true
	}
	if !summary.transactionSum.add(block.transactions[slot]) {
		summary.invalid = true
		return summary, true
	}
	if summary.aggregateCount == 0 {
		summary.high, summary.low = block.highs[slot], block.lows[slot]
	} else {
		summary.high = max(summary.high, block.highs[slot])
		summary.low = min(summary.low, block.lows[slot])
	}
	if summary.aggregateCount == math.MaxUint8 {
		summary.invalid = true
		return summary, true
	}
	summary.aggregateCount++
	return summary, true
}

func pruneFoldedActivityTargets(activity *activityFeatureState, floor time.Time) {
	for end, block := range activity.foldedTargets {
		blockStart := time.Unix(end, 0).UTC().Add(-activityBlockDuration)
		for slot := 0; slot < 30; slot++ {
			second := blockStart.Add(time.Duration(slot) * time.Second)
			if !second.Before(floor) {
				continue
			}
			mask := uint32(1) << uint(slot)
			if block.present&mask == 0 {
				continue
			}
			block.present &^= mask
			block.invalid &^= mask
			block.transactions[slot], block.highs[slot], block.lows[slot] = 0, 0, 0
			activity.foldedTargetContributions--
		}
		if block.present == 0 {
			delete(activity.foldedTargets, end)
		} else {
			activity.foldedTargets[end] = block
		}
	}
}

func unavailableActivityResult(at time.Time) activityFeatureResult {
	return activityFeatureResult{
		at:       at,
		activity: aggregateFeatureField{status: featureUnavailable, reason: featureReasonBeforeFirstPrint},
	}
}

func activityComponentsForBlock(state *symbolAggregateState, binding *installedBinding, start, end time.Time) activityBlockSummary {
	activity := ensureActivityState(state)
	key := end.Unix()
	summary, ok := activity.references[key]
	if !ok {
		if block, exists := activity.mutable[key]; exists {
			summary, ok = block.current, true
		}
	}
	if !ok {
		summary = activityBlockSummary{end: key, low: math.Inf(1)}
		// This fallback supports direct formula evaluation before the ordinary
		// contributor has observed a test/setup state. Production transitions
		// populate exactly the touched mutable block above.
		for _, record := range state.tail {
			if record.windowStart.Before(start) || !record.windowStart.Before(end) {
				continue
			}
			summary = addActivityAggregate(summary, record.values)
		}
	}
	return summary
}

func activityCoverageTrustworthy(state *symbolAggregateState, binding *installedBinding, start, end time.Time) bool {
	if state.historicalConflict != nil && bitmapHasRange(state.historicalConflict, binding, start, end) {
		return false
	}
	return exactAggregateCoverage(state, binding, start, end)
}

func firstAlignedActivityStart(binding *installedBinding, floor time.Time) time.Time {
	if !floor.After(binding.sessionStart) {
		return binding.sessionStart
	}
	offset := floor.Sub(binding.sessionStart)
	blocks := offset / activityBlockDuration
	start := binding.sessionStart.Add(blocks * activityBlockDuration)
	if start.Before(floor) {
		start = start.Add(activityBlockDuration)
	}
	return start
}

func empiricalPercentile(values []float64, value float64) float64 {
	count := 0
	for _, reference := range values {
		if reference <= value {
			count++
		}
	}
	return 100 * float64(count) / float64(len(values))
}
