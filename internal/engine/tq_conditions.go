package engine

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
)

//go:embed trade_conditions_fixture.json
var reviewedTradeConditionFixture []byte

const reviewedTradeConditionFileSHA256 = "90e93fdc8cb4d4e661c3e3f5ba55df15d4a5ea799c2497fd7586fdd8740a4c16"

type tradeConditionFixture struct {
	SchemaVersion          int                  `json:"schema_version"`
	ReviewedAt             string               `json:"reviewed_at"`
	AssetClass             string               `json:"asset_class"`
	SourceNormalizedSHA256 string               `json:"source_normalized_sha256"`
	Rules                  []tradeConditionRule `json:"rules"`
}

type tradeConditionRule struct {
	ID            int64 `json:"id"`
	UpdatesVolume bool  `json:"updates_volume"`
	Reviewed      bool  `json:"reviewed"`
}

var reviewedTradeConditions map[int64]tradeConditionRule

func init() {
	digest := sha256.Sum256(reviewedTradeConditionFixture)
	if hex.EncodeToString(digest[:]) != reviewedTradeConditionFileSHA256 {
		panic(errors.New("trade-condition fixture hash mismatch"))
	}
	var fixture tradeConditionFixture
	decoder := json.NewDecoder(bytes.NewReader(reviewedTradeConditionFixture))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&fixture) != nil || decoder.Decode(new(any)) != io.EOF || fixture.SchemaVersion != 1 || fixture.ReviewedAt != "2026-08-03" || fixture.AssetClass != "stocks" ||
		fixture.SourceNormalizedSHA256 != "fba0b47b559168d8b55589118db84469f805c6567873ef86db0100997787ea42" || len(fixture.Rules) != 55 {
		panic(errors.New("invalid reviewed trade-condition fixture"))
	}
	reviewedTradeConditions = make(map[int64]tradeConditionRule, len(fixture.Rules))
	var prior int64
	for index, rule := range fixture.Rules {
		if rule.ID <= 0 || (index > 0 && rule.ID <= prior) {
			panic(errors.New("unsorted trade-condition fixture"))
		}
		prior = rule.ID
		reviewedTradeConditions[rule.ID] = rule
	}
}

func classifyTradeConditionEvidence(shapeClassified bool, conditions []int64) (classified, eligible bool) {
	if !shapeClassified {
		return false, false
	}
	for _, id := range conditions {
		rule, ok := reviewedTradeConditions[id]
		if !ok || !rule.Reviewed {
			return false, false
		}
		if !rule.UpdatesVolume {
			return true, false
		}
	}
	return true, true
}
