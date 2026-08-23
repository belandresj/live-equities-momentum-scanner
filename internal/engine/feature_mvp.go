package engine

import (
	"math"
	"sort"
	"time"
)

const activityReferenceSamples = 55

type mvpMeasurementPoint struct {
	start      int64
	volume     float64
	close      float64
	cumulative float64
}

// mvpMeasurementState is derived state inside the canonical symbol owner. The
// ordered prefix is cutoff-bearing evidence for exact as-of session volume;
// the correction tail remains the sole mutable aggregate representation.
type mvpMeasurementState struct {
	folded []mvpMeasurementPoint
	tail   []mvpMeasurementPoint
	result mvpMeasurementResult
}

type mvpMeasurementResult struct {
	at                         time.Time
	sessionVolume, activity30s aggregateFeatureField
	move30s                    aggregateFeatureField
}

func ensureMVPMeasurementState(state *symbolAggregateState) *mvpMeasurementState {
	if state.mvpMeasurements == nil {
		state.mvpMeasurements = &mvpMeasurementState{result: unavailableMVPMeasurementResult(time.Time{})}
	}
	return state.mvpMeasurements
}

func foldMVPMeasurementAggregate(state *symbolAggregateState, record canonicalAggregate) {
	measurements := ensureMVPMeasurementState(state)
	point := mvpMeasurementPoint{start: record.windowStart.Unix(), volume: record.values.Volume, close: record.values.Close}
	i := sort.Search(len(measurements.folded), func(i int) bool { return measurements.folded[i].start >= point.start })
	if i < len(measurements.folded) && measurements.folded[i].start == point.start {
		measurements.folded[i] = point
	} else {
		measurements.folded = append(measurements.folded, mvpMeasurementPoint{})
		copy(measurements.folded[i+1:], measurements.folded[i:])
		measurements.folded[i] = point
	}
	recomputeMVPVolumePrefix(measurements, i)
}

func retainMutableMVPMeasurement(state *symbolAggregateState, record canonicalAggregate) {
	measurements := ensureMVPMeasurementState(state)
	point := mvpMeasurementPoint{start: record.windowStart.Unix(), volume: record.values.Volume, close: record.values.Close}
	i := sort.Search(len(measurements.tail), func(i int) bool { return measurements.tail[i].start >= point.start })
	if i < len(measurements.tail) && measurements.tail[i].start == point.start {
		measurements.tail[i] = point
	} else {
		measurements.tail = append(measurements.tail, mvpMeasurementPoint{})
		copy(measurements.tail[i+1:], measurements.tail[i:])
		measurements.tail[i] = point
	}
	recomputeMVPVolumePoints(measurements.tail, i)
}

func removeMutableMVPMeasurement(state *symbolAggregateState, start int64) {
	if state == nil || state.mvpMeasurements == nil {
		return
	}
	points := state.mvpMeasurements.tail
	i := sort.Search(len(points), func(i int) bool { return points[i].start >= start })
	if i == len(points) || points[i].start != start {
		return
	}
	copy(points[i:], points[i+1:])
	state.mvpMeasurements.tail = points[:len(points)-1]
	recomputeMVPVolumePoints(state.mvpMeasurements.tail, i)
}

func removeFoldedMVPMeasurement(state *symbolAggregateState, start int64) {
	if state == nil || state.mvpMeasurements == nil {
		return
	}
	points := state.mvpMeasurements.folded
	i := sort.Search(len(points), func(i int) bool { return points[i].start >= start })
	if i == len(points) || points[i].start != start {
		return
	}
	copy(points[i:], points[i+1:])
	state.mvpMeasurements.folded = points[:len(points)-1]
	recomputeMVPVolumePrefix(state.mvpMeasurements, i)
}

func recomputeMVPVolumePrefix(state *mvpMeasurementState, from int) {
	recomputeMVPVolumePoints(state.folded, from)
}

func recomputeMVPVolumePoints(points []mvpMeasurementPoint, from int) {
	if from < 0 {
		from = 0
	}
	running := 0.0
	if from > 0 {
		running = points[from-1].cumulative
	}
	for i := from; i < len(points); i++ {
		running += points[i].volume
		points[i].cumulative = running
	}
}

func unavailableMVPMeasurementResult(at time.Time) mvpMeasurementResult {
	unavailable := aggregateFeatureField{status: featureUnavailable, reason: featureReasonHistoryIncomplete}
	return mvpMeasurementResult{at: at, sessionVolume: unavailable, activity30s: unavailable, move30s: unavailable}
}

func evaluateMVPMeasurements(binding *installedBinding, state *symbolAggregateState, at time.Time, invalid *invalidMarkEvidence) mvpMeasurementResult {
	result := unavailableMVPMeasurementResult(at)
	if binding == nil || state == nil || at.Before(binding.sessionStart) || at.After(binding.sessionEnd) {
		return result
	}
	if invalidWithin(invalid, binding.sessionStart, at) {
		result.sessionVolume = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	} else if trust := historyTrust(state, binding, binding.sessionStart, at, false); trust.status != featureCurrent {
		result.sessionVolume = trust
	} else {
		volume := sessionVolumeBefore(state, at)
		if finiteEvaluator(volume) && volume >= 0 {
			result.sessionVolume = aggregateFeatureField{status: featureCurrent, value: volume}
		} else {
			result.sessionVolume = aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
		}
	}
	result.activity30s = evaluateActivity30s(binding, state, at, invalid)
	result.move30s = evaluateMove30s(binding, state, at, invalid)
	return result
}

func sessionVolumeBefore(state *symbolAggregateState, at time.Time) float64 {
	result := 0.0
	if state.mvpMeasurements != nil {
		points := state.mvpMeasurements.folded
		i := sort.Search(len(points), func(i int) bool { return points[i].start >= at.Unix() })
		if i > 0 {
			result = points[i-1].cumulative
		}
	}
	if state.mvpMeasurements != nil && len(state.mvpMeasurements.tail) == len(state.tail) {
		points := state.mvpMeasurements.tail
		i := sort.Search(len(points), func(i int) bool { return points[i].start >= at.Unix() })
		if i > 0 {
			return result + points[i-1].cumulative
		}
		return result
	}
	starts := make([]int64, 0, len(state.tail))
	for start := range state.tail {
		if start < at.Unix() {
			starts = append(starts, start)
		}
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	for _, start := range starts {
		result += state.tail[start].values.Volume
	}
	return result
}

func evaluateActivity30s(binding *installedBinding, state *symbolAggregateState, at time.Time, invalid *invalidMarkEvidence) aggregateFeatureField {
	floor := at.Add(-330 * time.Second)
	if floor.Before(binding.sessionStart) {
		floor = binding.sessionStart
	}
	if invalidWithin(invalid, floor, at) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	if at.Sub(binding.sessionStart) < 330*time.Second {
		return aggregateFeatureField{status: featureWarming, reason: featureReasonReferenceWarmup}
	}
	if trust := historyTrust(state, binding, floor, at, false); trust.status != featureCurrent {
		return trust
	}
	var volumes [330]float64
	for i := range volumes {
		if record, ok := aggregateAt(state, floor.Add(time.Duration(i)*time.Second).Unix()); ok {
			volumes[i] = record.volume
		}
	}
	var prefix [331]float64
	for i, value := range volumes {
		prefix[i+1] = prefix[i] + value
	}
	target := (prefix[330] - prefix[300]) / 30
	var references [activityReferenceSamples]float64
	for k := range references {
		end := 300 - 5*k
		references[k] = (prefix[end] - prefix[end-30]) / 30
	}
	lessOrEqual := 0
	for _, value := range references {
		if value <= target {
			lessOrEqual++
		}
	}
	value := 100 * float64(lessOrEqual) / activityReferenceSamples
	if !finiteEvaluator(value) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	return aggregateFeatureField{status: featureCurrent, value: value}
}

func evaluateMove30s(binding *installedBinding, state *symbolAggregateState, at time.Time, invalid *invalidMarkEvidence) aggregateFeatureField {
	if at.Sub(binding.sessionStart) < 30*time.Second {
		return aggregateFeatureField{status: featureWarming, reason: featureReasonRollingWarmup}
	}
	baseBoundary := at.Add(-30 * time.Second)
	base, baseOK := markStrictlyBefore(state, baseBoundary)
	target, targetOK := markStrictlyBefore(state, at)
	if invalidSupersedesMark(invalid, baseBoundary, base, baseOK) || invalidSupersedesMark(invalid, at, target, targetOK) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	if !baseOK || !targetOK {
		return aggregateFeatureField{status: featureWarming, reason: featureReasonBeforeFirstPrint}
	}
	if trust := historyTrust(state, binding, base.end, baseBoundary, false); trust.status != featureCurrent {
		return trust
	}
	if trust := historyTrust(state, binding, target.end, at, false); trust.status != featureCurrent {
		return trust
	}
	value := ((target.close - base.close) / base.close) * 100
	if base.close <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return aggregateFeatureField{status: featureInvalid, reason: featureReasonInvalidInput}
	}
	return aggregateFeatureField{status: featureCurrent, value: value}
}

func invalidWithin(invalid *invalidMarkEvidence, start, end time.Time) bool {
	return invalid != nil && !invalid.windowStart.Before(start) && invalid.windowStart.Before(end)
}

func invalidSupersedesMark(invalid *invalidMarkEvidence, boundary time.Time, mark mvpAggregateValue, hasMark bool) bool {
	return invalid != nil && invalid.windowStart.Before(boundary) && (!hasMark || invalid.windowStart.After(mark.start))
}

type mvpAggregateValue struct {
	start, end time.Time
	volume     float64
	close      float64
}

func aggregateAt(state *symbolAggregateState, start int64) (mvpAggregateValue, bool) {
	if state.mvpMeasurements != nil && len(state.mvpMeasurements.tail) == len(state.tail) {
		points := state.mvpMeasurements.tail
		i := sort.Search(len(points), func(i int) bool { return points[i].start >= start })
		if i < len(points) && points[i].start == start {
			point := points[i]
			return mvpAggregateValue{start: time.Unix(start, 0).UTC(), end: time.Unix(start+1, 0).UTC(), volume: point.volume, close: point.close}, true
		}
	}
	if record := state.tail[start]; record != nil {
		return mvpAggregateValue{start: record.windowStart, end: record.windowEnd, volume: record.values.Volume, close: record.values.Close}, true
	}
	if state.mvpMeasurements == nil {
		return mvpAggregateValue{}, false
	}
	points := state.mvpMeasurements.folded
	i := sort.Search(len(points), func(i int) bool { return points[i].start >= start })
	if i == len(points) || points[i].start != start {
		return mvpAggregateValue{}, false
	}
	return mvpAggregateValue{start: time.Unix(start, 0).UTC(), end: time.Unix(start+1, 0).UTC(), volume: points[i].volume, close: points[i].close}, true
}

func markStrictlyBefore(state *symbolAggregateState, boundary time.Time) (mvpAggregateValue, bool) {
	var result mvpAggregateValue
	found := false
	if state.mvpMeasurements != nil {
		points := state.mvpMeasurements.folded
		i := sort.Search(len(points), func(i int) bool { return points[i].start >= boundary.Unix() })
		if i > 0 {
			point := points[i-1]
			result = mvpAggregateValue{start: time.Unix(point.start, 0).UTC(), end: time.Unix(point.start+1, 0).UTC(), volume: point.volume, close: point.close}
			found = true
		}
		if len(state.mvpMeasurements.tail) == len(state.tail) {
			points = state.mvpMeasurements.tail
			i = sort.Search(len(points), func(i int) bool { return points[i].start >= boundary.Unix() })
			if i > 0 {
				point := points[i-1]
				candidate := mvpAggregateValue{start: time.Unix(point.start, 0).UTC(), end: time.Unix(point.start+1, 0).UTC(), volume: point.volume, close: point.close}
				if !found || result.start.Before(candidate.start) {
					result, found = candidate, true
				}
			}
			return result, found
		}
	}
	for start, record := range state.tail {
		if start < boundary.Unix() && (!found || result.start.Before(record.windowStart)) {
			result = mvpAggregateValue{start: record.windowStart, end: record.windowEnd, volume: record.values.Volume, close: record.values.Close}
			found = true
		}
	}
	return result, found
}
