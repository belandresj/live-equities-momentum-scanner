package checkpoint

import (
	"context"
	"math"
	"math/big"
	"time"
)

func ValidateSemanticImage(image Image) error {
	return validateSemanticImage(context.Background(), image)
}

func validateSemanticImage(ctx context.Context, image Image) error {
	if ctx == nil {
		return ErrCanceled
	}
	if image.Binding.SessionStart.IsZero() || !image.Binding.SessionStart.Before(image.Binding.SessionEnd) || !validWholeSecondUTC(image.T0) || !validUTCInstant(image.CreatedAt) || image.CreatedAt.Before(image.T0) || image.T0.Before(image.Binding.SessionStart) || image.T0.After(image.Binding.SessionEnd) {
		return ErrInvalid
	}
	for _, symbol := range image.Symbols {
		if err := semanticContext(ctx); err != nil {
			return err
		}
		if !symbol.HasState {
			if len(symbol.Tail) != 0 || symbol.OlderMark != nil || symbol.CommittedMark != nil || len(symbol.Presence) != 0 || len(symbol.ProvenAbsent) != 0 || len(symbol.HistoricalConflict) != 0 || symbol.PriceRange != nil || symbol.Activity != nil || symbol.Qualification != nil {
				return ErrInvalid
			}
		}
		last := time.Time{}
		for _, aggregate := range symbol.Tail {
			if err := semanticContext(ctx); err != nil {
				return err
			}
			if !last.IsZero() && !aggregate.WindowStart.After(last) || !validSemanticAggregate(aggregate, image) {
				return ErrInvalid
			}
			last = aggregate.WindowStart
		}
		if symbol.OlderMark != nil && !validSemanticAggregate(*symbol.OlderMark, image) {
			return ErrInvalid
		}
		if symbol.CommittedMark != nil && !validSemanticAggregate(*symbol.CommittedMark, image) {
			return ErrInvalid
		}
		if symbol.CommittedMark != nil {
			latest := symbol.CommittedMark.WindowStart
			matchedCanonical := symbol.OlderMark != nil && *symbol.OlderMark == *symbol.CommittedMark
			if symbol.OlderMark != nil && symbol.OlderMark.WindowStart.After(latest) {
				return ErrInvalid
			}
			for _, aggregate := range symbol.Tail {
				if aggregate.WindowStart.After(latest) {
					return ErrInvalid
				}
				matchedCanonical = matchedCanonical || aggregate == *symbol.CommittedMark
			}
			if !matchedCanonical {
				return ErrInvalid
			}
		}
		for _, bitmap := range [][]uint64{symbol.Presence, symbol.ProvenAbsent, symbol.HistoricalConflict} {
			if len(bitmap) != 0 && len(bitmap) != 900 {
				return ErrInvalid
			}
		}
		if len(symbol.Presence) != 0 || len(symbol.ProvenAbsent) != 0 || len(symbol.HistoricalConflict) != 0 {
			cutoff := int(image.T0.Sub(image.Binding.SessionStart) / time.Second)
			for slot := 0; slot < 57_600; slot++ {
				present, absent, conflict := bitmapHas(symbol.Presence, slot), bitmapHas(symbol.ProvenAbsent, slot), bitmapHas(symbol.HistoricalConflict, slot)
				if slot >= cutoff && (present || absent || conflict) || absent && (present || conflict) {
					return ErrInvalid
				}
			}
		}
		for _, aggregate := range symbol.Tail {
			slot := int(aggregate.WindowStart.Sub(image.Binding.SessionStart) / time.Second)
			if bitmapHas(symbol.Presence, slot) || bitmapHas(symbol.ProvenAbsent, slot) {
				return ErrInvalid
			}
		}
		if !validPriceRange(ctx, symbol.PriceRange, image) || !validActivity(ctx, symbol.Activity, image) || !validQualification(ctx, symbol.Qualification, image) {
			if err := semanticContext(ctx); err != nil {
				return err
			}
			return ErrInvalid
		}
		if symbol.InvalidMarkStart != nil && (!symbol.InvalidMarkStart.Before(image.T0) || symbol.InvalidMarkStart.Before(image.Binding.SessionStart)) {
			return ErrInvalid
		}
		if symbol.Coverage != nil && !((symbol.Coverage.Outcome == 1 && symbol.Coverage.Origin == 0) || (symbol.Coverage.Outcome == 2 && symbol.Coverage.Origin >= 1 && symbol.Coverage.Origin <= 3)) {
			return ErrInvalid
		}
	}
	if err := semanticContext(ctx); err != nil {
		return err
	}
	return nil
}

func semanticContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return ErrCanceled
	}
	return nil
}

func bitmapHas(bitmap []uint64, slot int) bool {
	return len(bitmap) != 0 && slot >= 0 && slot < 57_600 && bitmap[slot/64]&(uint64(1)<<uint(slot%64)) != 0
}

func validSemanticAggregate(a Aggregate, image Image) bool {
	v := a.Values
	return a.WindowStart == a.WindowStart.UTC() && a.WindowEnd == a.WindowEnd.UTC() && a.WindowStart.Nanosecond() == 0 && a.WindowEnd.Nanosecond() == 0 &&
		!a.WindowStart.Before(image.Binding.SessionStart) && a.WindowStart.Before(image.T0) && !a.WindowEnd.After(image.Binding.SessionEnd) && a.WindowEnd == a.WindowStart.Add(time.Second) &&
		finitePositive(v.Open) && finitePositive(v.High) && finitePositive(v.Low) && finitePositive(v.Close) && finite(v.Volume) && v.Volume >= 0 && finitePositive(v.VWAP) &&
		v.High >= max(v.Open, max(v.Close, v.Low)) && v.Low <= min(v.Open, v.Close) && v.AverageTradeSize >= 0 && validATS(v.ATSProvenance)
}

func validPriceRange(ctx context.Context, v *PriceRange, image Image) bool {
	if v == nil {
		return true
	}
	if v.FinalizedThrough > image.T0.Unix() || v.RollingFloor > image.T0.Unix() {
		return false
	}
	if v.HasFirst != (v.FirstStart != 0) || v.HasFirst && (v.FirstStart < image.Binding.SessionStart.Unix() || v.FirstStart >= image.T0.Unix() || !finitePositive(v.FirstOpen)) || !v.HasFirst && v.FirstOpen != 0 {
		return false
	}
	if v.HasSessionExtrema && (!finitePositive(v.SessionHigh) || !finitePositive(v.SessionLow) || v.SessionHigh < v.SessionLow) || !v.HasSessionExtrema && (v.SessionHigh != 0 || v.SessionLow != 0) || len(v.SessionHighs) != len(v.SessionLows) {
		return false
	}
	for _, points := range [][]ExtremaPoint{v.Highs, v.Lows, v.SessionHighs, v.SessionLows} {
		last := int64(0)
		for _, point := range points {
			if semanticContext(ctx) != nil {
				return false
			}
			if point.WindowStart < image.Binding.SessionStart.Unix() || point.WindowStart >= image.T0.Unix() || point.WindowStart <= last || !finitePositive(point.Value) {
				return false
			}
			last = point.WindowStart
		}
	}
	for index := range v.SessionHighs {
		if v.SessionHighs[index].WindowStart != v.SessionLows[index].WindowStart || v.SessionHighs[index].Value < v.SessionLows[index].Value {
			return false
		}
	}
	return true
}

func validActivity(ctx context.Context, v *Activity, image Image) bool {
	if v == nil {
		return true
	}
	last := int64(0)
	for _, summary := range v.References {
		if semanticContext(ctx) != nil {
			return false
		}
		if summary.End <= last || summary.End > image.T0.Unix() || !validActivitySummary(summary, summary.End) {
			return false
		}
		last = summary.End
	}
	last = 0
	for _, mutable := range v.Mutable {
		if semanticContext(ctx) != nil || mutable.End <= last || mutable.End > image.Binding.SessionEnd.Unix() || !time.Unix(mutable.End, 0).UTC().Add(-30*time.Second).Before(image.T0) || !validActivitySummary(mutable.Folded, mutable.End) || !validActivitySummary(mutable.Current, mutable.End) {
			return false
		}
		last = mutable.End
	}
	count := 0
	last = 0
	for _, block := range v.FoldedTargets {
		if semanticContext(ctx) != nil || block.End <= last || block.End > image.Binding.SessionEnd.Unix() || block.Invalid&^block.Present != 0 {
			return false
		}
		last = block.End
		start := time.Unix(block.End, 0).UTC().Add(-30 * time.Second)
		for slot := 0; slot < 30; slot++ {
			if semanticContext(ctx) != nil {
				return false
			}
			mask := uint32(1) << uint(slot)
			present, invalid := block.Present&mask != 0, block.Invalid&mask != 0
			if !present || invalid {
				if block.Transactions[slot] != 0 || block.Highs[slot] != 0 || block.Lows[slot] != 0 {
					return false
				}
			} else if !start.Add(time.Duration(slot)*time.Second).Before(image.T0) || !finite(block.Transactions[slot]) || block.Transactions[slot] < 0 || !finitePositive(block.Highs[slot]) || !finitePositive(block.Lows[slot]) || block.Highs[slot] < block.Lows[slot] {
				return false
			}
			if present {
				count++
			}
		}
	}
	return count == v.FoldedTargetContributions
}

func validActivitySummary(s ActivitySummary, end int64) bool {
	zero := ExactSum{}
	if s.End != end || s.AggregateCount > 30 {
		return false
	}
	if s.Invalid {
		return s.AggregateCount == 0 && s.TransactionSum == zero && s.Transactions == 0 && s.High == 0 && s.Low == 0 && s.ExpansionBPS == 0
	}
	if s.AggregateCount == 0 {
		return s.TransactionSum == zero && s.Transactions == 0 && s.High == 0 && s.Low == 0 && s.ExpansionBPS == 0
	}
	want, ok := exactSumFloat(s.TransactionSum)
	return ok && s.Transactions == want && finite(s.Transactions) && s.Transactions >= 0 && finitePositive(s.High) && finitePositive(s.Low) && s.High >= s.Low && s.ExpansionBPS == 10_000*math.Log(s.High/s.Low) && finite(s.ExpansionBPS)
}

func exactSumFloat(sum ExactSum) (float64, bool) {
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
	return result, finite(result)
}

func validQualification(ctx context.Context, v *Qualification, image Image) bool {
	if v == nil {
		return true
	}
	last := int64(0)
	for _, bar := range v.FinalizedGateBars {
		if semanticContext(ctx) != nil || bar.Start <= last || bar.Start < image.Binding.SessionStart.Unix() || bar.Start >= image.T0.Unix() || !finitePositive(bar.Close) || !finite(bar.Volume) || bar.Volume < 0 || !finitePositive(bar.VWAP) || bar.AverageTradeSize < 0 || !validATS(bar.ATSProvenance) {
			return false
		}
		last = bar.Start
	}
	proofs := make(map[int64]bool, len(v.Proofs))
	last = 0
	for _, proof := range v.Proofs {
		if semanticContext(ctx) != nil || proof <= last || proof < image.Binding.SessionStart.Unix() || proof > image.Binding.SessionEnd.Unix() || proof > image.T0.Unix() || !v.AccountedThrough.IsZero() && proof > v.AccountedThrough.Unix() {
			return false
		}
		proofs[proof] = true
		last = proof
	}
	last = 0
	for _, dirty := range v.Dirty {
		if semanticContext(ctx) != nil || dirty <= last || !proofs[dirty] {
			return false
		}
		last = dirty
	}
	if v.Invalid || v.UnresolvedOrigin > 3 || !v.AccountedThrough.IsZero() && (v.AccountedThrough.Before(image.Binding.SessionStart) || v.AccountedThrough.After(image.Binding.SessionEnd) || v.AccountedThrough.After(image.T0) || v.AccountedThrough.Nanosecond() != 0) {
		return false
	}
	if v.Finalized {
		return !v.FinalProofEnd.Before(image.Binding.SessionStart) && !v.FinalProofEnd.After(image.Binding.SessionEnd) && v.FinalProofEnd.Nanosecond() == 0 && len(v.Proofs) == 0 && len(v.Dirty) == 0
	}
	return v.FinalProofEnd.IsZero()
}

func validATS(value string) bool {
	return value == "live_provider_average" || value == "rest_floor_volume_over_transactions"
}
func finite(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finitePositive(value float64) bool { return finite(value) && value > 0 }
