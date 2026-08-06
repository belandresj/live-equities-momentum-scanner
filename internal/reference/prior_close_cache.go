package reference

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const priorCloseCacheSchema = "massive-prior-close-cache-v1"

type priorCloseCacheFact struct {
	Symbol string                  `json:"symbol"`
	Status PriorCloseStatus        `json:"status"`
	Close  *float64                `json:"close,omitempty"`
	Reason PriorCloseInvalidReason `json:"reason,omitempty"`
}

type priorCloseCacheSource struct {
	Adjusted   bool   `json:"adjusted"`
	IncludeOTC bool   `json:"include_otc"`
	Locale     string `json:"locale"`
	Market     string `json:"market"`
}

type priorCloseCache struct {
	SchemaVersion      string                `json:"schema_version"`
	PriorSessionDate   string                `json:"prior_session_date"`
	RetrievedAt        string                `json:"retrieved_at"`
	PriorClosePolicy   string                `json:"prior_close_policy"`
	Source             priorCloseCacheSource `json:"source_parameters"`
	PriorCloseIdentity string                `json:"prior_close_identity"`
	Accounting         PriorCloseAccounting  `json:"accounting"`
	Facts              []priorCloseCacheFact `json:"symbol_facts"`
}

func preparePriorCloseCache(root string) (string, error) {
	return prepareCacheDirectory(root, "prior-close")
}

func publishPriorCloseCache(directory string, data normalizedPriorCloses) error {
	return publishPriorCloseCacheWithOps(directory, data, nil)
}

func publishPriorCloseCacheWithOps(directory string, data normalizedPriorCloses, operations *cacheFileOps) error {
	facts := make([]priorCloseCacheFact, len(data.facts))
	for index, fact := range data.facts {
		cached := priorCloseCacheFact{Symbol: fact.symbol, Status: fact.status, Reason: fact.reason}
		if fact.status == PriorCloseValid {
			value := fact.close
			cached.Close = &value
		}
		facts[index] = cached
	}
	cache := priorCloseCache{
		SchemaVersion:      priorCloseCacheSchema,
		PriorSessionDate:   data.priorSessionDate,
		RetrievedAt:        canonicalCacheTime(data.retrievedAt),
		PriorClosePolicy:   PriorClosePolicyVersion,
		Source:             priorCloseCacheSource{Adjusted: true, IncludeOTC: false, Locale: "us", Market: "stocks"},
		PriorCloseIdentity: data.identity,
		Accounting:         data.accounting,
		Facts:              facts,
	}
	body, err := jsonMarshalPriorCache(cache)
	if err != nil {
		return err
	}
	return publishPrivateCacheFile(directory, data.priorSessionDate+".json", ".prior-close-", body, operations)
}

func jsonMarshalPriorCache(cache priorCloseCache) ([]byte, error) {
	body, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func prunePriorCloseCaches(directory string) error {
	return prunePriorCloseCachesAt(directory, time.Now())
}

func prunePriorCloseCachesAt(directory string, now time.Time) error {
	return pruneCacheDirectory(directory, ".prior-close-", now)
}

func (r *PriorCloseResolver) readPriorCloseCache(directory, date string, priorRegularClose time.Time, symbols []string) (normalizedPriorCloses, error) {
	path := filepath.Join(directory, date+".json")
	info, err := validatePrivateRegularFile(path)
	if err != nil {
		return normalizedPriorCloses{}, err
	}
	if info.Size() > maximumCacheBytes {
		return normalizedPriorCloses{}, errors.New("prior-close cache exceeds 16 MiB")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return normalizedPriorCloses{}, err
	}
	var cache priorCloseCache
	if err := decodeStrictCacheJSON(body, &cache); err != nil {
		return normalizedPriorCloses{}, err
	}
	if cache.SchemaVersion != priorCloseCacheSchema || cache.PriorSessionDate != date ||
		cache.PriorClosePolicy != PriorClosePolicyVersion || !cache.Source.Adjusted || cache.Source.IncludeOTC ||
		cache.Source.Locale != "us" || cache.Source.Market != "stocks" {
		return normalizedPriorCloses{}, errors.New("prior-close cache schema, date, policy, or source mismatch")
	}
	retrievedAt, err := parseCanonicalCacheTime(cache.RetrievedAt)
	if err != nil || retrievedAt.Before(priorRegularClose) || retrievedAt.After(r.now()) {
		return normalizedPriorCloses{}, errors.New("prior-close cache retrieval time is impossible")
	}
	if len(cache.Facts) != len(symbols) || cache.Accounting.UniverseTotal != len(symbols) || !cache.Accounting.valid() {
		return normalizedPriorCloses{}, errors.New("prior-close cache counts do not reconcile")
	}
	facts := make([]PriorCloseFact, len(cache.Facts))
	for index, cached := range cache.Facts {
		if cached.Symbol != symbols[index] {
			return normalizedPriorCloses{}, errors.New("prior-close cache population is noncanonical or mismatched")
		}
		fact := PriorCloseFact{symbol: cached.Symbol, status: cached.Status, reason: cached.Reason}
		switch cached.Status {
		case PriorCloseValid:
			if cached.Close == nil || *cached.Close <= 0 || !isFinite(*cached.Close) || cached.Reason != "" {
				return normalizedPriorCloses{}, errors.New("invalid valid prior-close cache fact")
			}
			fact.close = *cached.Close
		case PriorCloseMissing:
			if cached.Close != nil || cached.Reason != "" {
				return normalizedPriorCloses{}, errors.New("invalid missing prior-close cache fact")
			}
		case PriorCloseInvalid:
			if cached.Close != nil || !validPriorInvalidReason(cached.Reason) {
				return normalizedPriorCloses{}, errors.New("invalid invalid prior-close cache fact")
			}
		default:
			return normalizedPriorCloses{}, errors.New("unknown prior-close cache status")
		}
		facts[index] = fact
	}
	if got := accountPriorCloses(facts, cache.Accounting.UnattributableDiagnosticRows); got != cache.Accounting {
		return normalizedPriorCloses{}, errors.New("prior-close cache accounting mismatch")
	}
	identity, err := priorCloseIdentity(date, PriorClosePolicyVersion, facts)
	if err != nil || identity != cache.PriorCloseIdentity {
		return normalizedPriorCloses{}, errors.New("prior-close cache identity mismatch")
	}
	return normalizedPriorCloses{date, retrievedAt, identity, slices.Clone(facts), cache.Accounting}, nil
}

func validPriorInvalidReason(reason PriorCloseInvalidReason) bool {
	switch reason {
	case PriorCloseMalformed, PriorCloseDuplicate, PriorCloseWrongDate, PriorCloseNonpositive, PriorCloseNonfinite:
		return true
	default:
		return false
	}
}

func isFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
