package reference

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	universeCacheSchema = "massive-eligible-universe-cache-v1"
	maximumPriorAgeDays = 7
)

type universeCache struct {
	SchemaVersion            string     `json:"schema_version"`
	ReferenceDate            string     `json:"reference_date"`
	RetrievedAt              string     `json:"retrieved_at"`
	EligibilityPolicyVersion string     `json:"eligibility_policy_version"`
	UniverseIdentity         string     `json:"universe_identity"`
	Accounting               Accounting `json:"accounting"`
	Symbols                  []string   `json:"symbols"`
}

func prepareUniverseCache(root string) (string, error) {
	return prepareCacheDirectory(root, "universe")
}

func publishUniverseCache(directory string, data normalizedUniverse) error {
	return publishUniverseCacheWithOps(directory, data, nil)
}

func publishUniverseCacheWithOps(directory string, data normalizedUniverse, operations *cacheFileOps) error {
	cache := universeCache{
		SchemaVersion:            universeCacheSchema,
		ReferenceDate:            data.referenceDate,
		RetrievedAt:              canonicalCacheTime(data.retrievedAt),
		EligibilityPolicyVersion: EligibilityPolicyVersion,
		UniverseIdentity:         data.identity,
		Accounting:               data.accounting,
		Symbols:                  slices.Clone(data.symbols),
	}
	body, err := jsonMarshalCache(cache)
	if err != nil {
		return err
	}
	return publishPrivateCacheFile(directory, data.referenceDate+".json", ".universe-", body, operations)
}

func jsonMarshalCache(cache universeCache) ([]byte, error) {
	body, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func pruneUniverseCaches(directory string) error {
	return pruneUniverseCachesAt(directory, time.Now())
}

func pruneUniverseCachesAt(directory string, now time.Time) error {
	return pruneCacheDirectory(directory, ".universe-", now)
}

func (r *Resolver) loadFallback(directory, tradingDate string) (Universe, error) {
	entries, err := readCacheDirectoryBounded(directory)
	if err != nil {
		return Universe{}, err
	}
	type candidate struct {
		data normalizedUniverse
		age  int
	}
	candidates := make([]candidate, 0, maximumPriorAgeDays+1)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() || !validCacheFilename(entry.Name()) {
			continue
		}
		date := strings.TrimSuffix(entry.Name(), ".json")
		age, err := calendarAge(date, tradingDate)
		if err != nil || age < 0 || age > maximumPriorAgeDays {
			continue
		}
		data, err := r.readUniverseCache(filepath.Join(directory, entry.Name()), date, tradingDate, age)
		if err == nil {
			candidates = append(candidates, candidate{data: data, age: age})
		}
	}
	if len(candidates) == 0 {
		return Universe{}, errors.New("no valid universe cache at most seven calendar days old")
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].data.referenceDate > candidates[j].data.referenceDate
	})
	selected := candidates[0]
	source := SourceObservablePriorCache
	if selected.age == 0 {
		source = SourceCurrentCache
	}
	return universeFromNormalized(selected.data, source, selected.age), nil
}

func (r *Resolver) readUniverseCache(path, filenameDate, tradingDate string, age int) (normalizedUniverse, error) {
	info, err := validatePrivateRegularFile(path)
	if err != nil {
		return normalizedUniverse{}, err
	}
	if info.Size() > maximumCacheBytes {
		return normalizedUniverse{}, errors.New("universe cache exceeds 16 MiB")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return normalizedUniverse{}, err
	}
	var cache universeCache
	if err := decodeStrictCacheJSON(body, &cache); err != nil {
		return normalizedUniverse{}, err
	}
	if cache.SchemaVersion != universeCacheSchema ||
		cache.EligibilityPolicyVersion != EligibilityPolicyVersion || cache.ReferenceDate != filenameDate {
		return normalizedUniverse{}, errors.New("universe cache schema, policy, or date mismatch")
	}
	expectedAge, err := calendarAge(cache.ReferenceDate, tradingDate)
	if err != nil || expectedAge != age || age < 0 || age > maximumPriorAgeDays {
		return normalizedUniverse{}, errors.New("universe cache age mismatch")
	}
	if _, err := r.Schedule.ForTradingDate(cache.ReferenceDate); err != nil {
		return normalizedUniverse{}, errors.New("universe cache date is not supported by the exchange schedule")
	}
	retrievedAt, err := parseCanonicalCacheTime(cache.RetrievedAt)
	if err != nil {
		return normalizedUniverse{}, err
	}
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return normalizedUniverse{}, errors.New("load America/New_York timezone")
	}
	midnight, err := time.ParseInLocation("2006-01-02", cache.ReferenceDate, location)
	if err != nil || retrievedAt.Before(midnight) || retrievedAt.After(r.now()) {
		return normalizedUniverse{}, errors.New("universe cache retrieval time is impossible")
	}
	if len(cache.Symbols) == 0 || cache.Accounting.EligibleRecords != len(cache.Symbols) || !cache.Accounting.valid() {
		return normalizedUniverse{}, errors.New("universe cache counts do not reconcile")
	}
	previous := ""
	for _, symbol := range cache.Symbols {
		if !validSymbol(symbol) || symbol <= previous {
			return normalizedUniverse{}, errors.New("universe cache symbols are not strict sorted unique identities")
		}
		previous = symbol
	}
	identity, err := universeIdentity(cache.ReferenceDate, EligibilityPolicyVersion, cache.Symbols)
	if err != nil || identity != cache.UniverseIdentity {
		return normalizedUniverse{}, errors.New("universe cache identity mismatch")
	}
	return normalizedUniverse{
		referenceDate: cache.ReferenceDate,
		retrievedAt:   retrievedAt,
		identity:      cache.UniverseIdentity,
		symbols:       slices.Clone(cache.Symbols),
		accounting:    cache.Accounting,
	}, nil
}

func validCacheFilename(name string) bool {
	if len(name) != len("2006-01-02.json") || !strings.HasSuffix(name, ".json") {
		return false
	}
	date := strings.TrimSuffix(name, ".json")
	parsed, err := time.Parse("2006-01-02", date)
	return err == nil && parsed.Format("2006-01-02") == date
}

func calendarAge(referenceDate, tradingDate string) (int, error) {
	reference, err := time.Parse("2006-01-02", referenceDate)
	if err != nil {
		return 0, err
	}
	requested, err := time.Parse("2006-01-02", tradingDate)
	if err != nil {
		return 0, err
	}
	return int(requested.Sub(reference) / (24 * time.Hour)), nil
}
