package engine

import "time"

func evaluateTestPriceRangeFeatures(binding *installedBinding, symbol *coreSymbol, at time.Time) priceRangeFeatureResult {
	if symbol == nil || symbol.aggregates == nil {
		return unavailablePriceRangeResult(at)
	}
	mark, hasMark := latestMarkBeforeCompact(symbol.aggregates, at)
	return evaluatePriceRangeFeaturesWithMark(binding, symbol, at, mark, hasMark)
}
