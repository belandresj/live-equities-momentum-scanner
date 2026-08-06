package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func TestBothCacheSourceCurrentnessAndCorruptionMatrix(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	symbols := []string{"AAA", "BBB"}
	universe := acceptedUniverse(t, facts.TradingDate, symbols)
	now := time.Date(2026, 7, 29, 21, 0, 0, 123, time.UTC)

	t.Run("fresh wins over valid caches", func(t *testing.T) {
		var universeRequests atomic.Int64
		universeServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			universeRequests.Add(1)
			writePage(t, writer, []tickerRecord{{Ticker: "FRESH", Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, "")
		}))
		defer universeServer.Close()
		resolver := testResolver(t, schedule, universeServer)
		resolver.Now = func() time.Time { return now }
		directory, err := prepareUniverseCache(resolver.DataDir)
		if err != nil {
			t.Fatal(err)
		}
		publishUniverseFixture(t, directory, facts.TradingDate, now.Add(-time.Hour), []string{"CACHED"})
		got, err := resolver.Resolve(context.Background(), facts)
		if err != nil || got.Source() != SourceFresh || !slices.Equal(got.Symbols(), []string{"FRESH"}) || universeRequests.Load() != 1 {
			t.Fatalf("universe source decision = %+v requests=%d err=%v", got, universeRequests.Load(), err)
		}
		universeCacheBody, err := os.ReadFile(filepath.Join(directory, facts.TradingDate+".json"))
		if err != nil || strings.Contains(string(universeCacheBody), syntheticCredential) {
			t.Fatalf("universe cache credential exclusion: err=%v body=%s", err, universeCacheBody)
		}

		timestamp := facts.PriorRegularClose.UnixMilli()
		priorServer := groupedServer(fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":1,"results":[{"T":"AAA","c":12,"t":%d}]}`, timestamp))
		defer priorServer.Close()
		priorResolver := testPriorResolver(t, schedule, priorServer)
		priorResolver.Now = func() time.Time { return now }
		priorDirectory, err := preparePriorCloseCache(priorResolver.DataDir)
		if err != nil {
			t.Fatal(err)
		}
		publishPriorFixture(t, priorDirectory, facts, now.Add(-time.Hour), symbols, 10)
		prior, err := priorResolver.Resolve(context.Background(), facts, universe)
		closeValue, valid := prior.Facts()[0].Close()
		if err != nil || prior.Source() != SourceFresh || !valid || closeValue != 12 {
			t.Fatalf("prior source decision = %+v err=%v", prior, err)
		}
	})

	t.Run("universe retrieval time and canonical encoding", func(t *testing.T) {
		resolver := &Resolver{Schedule: schedule, Now: func() time.Time { return now }}
		directory := canonicalTempDir(t)
		location, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Fatal(err)
		}
		midnight, err := time.ParseInLocation("2006-01-02", facts.TradingDate, location)
		if err != nil {
			t.Fatal(err)
		}
		publishUniverseFixture(t, directory, facts.TradingDate, midnight, symbols)
		path := filepath.Join(directory, facts.TradingDate+".json")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytesContainCanonicalTimestamp(body, canonicalCacheTime(midnight)) {
			t.Fatalf("cache timestamp is not canonical: %s", body)
		}
		if _, err := resolver.readUniverseCache(path, facts.TradingDate, facts.TradingDate, 0); err != nil {
			t.Fatalf("possible same-date retrieval time rejected: %v", err)
		}
		for name, timestamp := range map[string]string{
			"before reference date": canonicalCacheTime(midnight.Add(-time.Nanosecond)),
			"after current time":    canonicalCacheTime(now.Add(time.Nanosecond)),
			"nonzero offset":        midnight.In(time.FixedZone("offset", -4*60*60)).Format(time.RFC3339Nano),
		} {
			t.Run(name, func(t *testing.T) {
				writePrivateFile(t, path, replaceJSONTimestamp(t, body, timestamp))
				if _, err := resolver.readUniverseCache(path, facts.TradingDate, facts.TradingDate, 0); err == nil {
					t.Fatal("impossible or noncanonical retrieval time accepted")
				}
				writePrivateFile(t, path, body)
			})
		}
	})

	t.Run("configuration unavailable uses permitted cache without requests", func(t *testing.T) {
		universeResolver := &Resolver{Schedule: schedule, DataDir: filepath.Join(canonicalTempDir(t), "reference"), Now: func() time.Time { return now }}
		directory, err := prepareUniverseCache(universeResolver.DataDir)
		if err != nil {
			t.Fatal(err)
		}
		publishUniverseFixture(t, directory, facts.TradingDate, now.Add(-time.Hour), symbols)
		current, err := universeResolver.Resolve(context.Background(), facts)
		if err != nil || current.Source() != SourceCurrentCache || !current.IsCurrent() || current.Diagnostics().RequestCount != 0 {
			t.Fatalf("same-date universe cache = %+v err=%v", current, err)
		}

		priorDate := "2026-07-28"
		os.Remove(filepath.Join(directory, facts.TradingDate+".json"))
		publishUniverseFixture(t, directory, priorDate, time.Date(2026, 7, 28, 18, 0, 0, 0, time.UTC), symbols)
		observable, err := universeResolver.Resolve(context.Background(), facts)
		if err != nil || observable.Source() != SourceObservablePriorCache || observable.IsCurrent() || observable.CacheAgeDays() != 1 || observable.Diagnostics().RequestCount != 0 {
			t.Fatalf("prior-date universe cache = %+v err=%v", observable, err)
		}

		priorResolver := &PriorCloseResolver{Schedule: schedule, DataDir: filepath.Join(canonicalTempDir(t), "reference"), Now: func() time.Time { return now }}
		priorDirectory, err := preparePriorCloseCache(priorResolver.DataDir)
		if err != nil {
			t.Fatal(err)
		}
		publishPriorFixture(t, priorDirectory, facts, facts.PriorRegularClose, symbols, 10)
		prior, err := priorResolver.Resolve(context.Background(), facts, universe)
		if err != nil || prior.Source() != SourceCurrentCache || prior.Diagnostics().RequestCount != 0 || prior.Diagnostics().TerminalReason != TerminalReasonNone {
			t.Fatalf("exact prior-close cache = %+v err=%v", prior, err)
		}
	})

	t.Run("prior retrieval time and canonical encoding", func(t *testing.T) {
		resolver := &PriorCloseResolver{Schedule: schedule, Now: func() time.Time { return now }}
		directory := canonicalTempDir(t)
		publishPriorFixture(t, directory, facts, facts.PriorRegularClose, symbols, 10)
		path := filepath.Join(directory, facts.PriorSessionDate+".json")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytesContainCanonicalTimestamp(body, canonicalCacheTime(facts.PriorRegularClose)) {
			t.Fatalf("cache timestamp is not canonical: %s", body)
		}
		if _, err := resolver.readPriorCloseCache(directory, facts.PriorSessionDate, facts.PriorRegularClose, symbols); err != nil {
			t.Fatalf("retrieval exactly at prior close rejected: %v", err)
		}

		for name, timestamp := range map[string]string{
			"before prior close": canonicalCacheTime(facts.PriorRegularClose.Add(-time.Nanosecond)),
			"after current time": canonicalCacheTime(now.Add(time.Nanosecond)),
			"nonzero offset":     facts.PriorRegularClose.In(time.FixedZone("offset", -4*60*60)).Format(time.RFC3339Nano),
		} {
			t.Run(name, func(t *testing.T) {
				mutated := replaceJSONTimestamp(t, body, timestamp)
				writePrivateFile(t, path, mutated)
				if _, err := resolver.readPriorCloseCache(directory, facts.PriorSessionDate, facts.PriorRegularClose, symbols); err == nil {
					t.Fatal("impossible or noncanonical retrieval time accepted")
				}
				writePrivateFile(t, path, body)
			})
		}
	})

	t.Run("strict corruption rejection for both classes", func(t *testing.T) {
		universeDirectory := canonicalTempDir(t)
		publishUniverseFixture(t, universeDirectory, facts.TradingDate, now.Add(-time.Hour), symbols)
		universePath := filepath.Join(universeDirectory, facts.TradingDate+".json")
		universeBody, err := os.ReadFile(universePath)
		if err != nil {
			t.Fatal(err)
		}
		universeResolver := &Resolver{Schedule: schedule, Now: func() time.Time { return now }}

		priorDirectory := canonicalTempDir(t)
		publishPriorFixture(t, priorDirectory, facts, facts.PriorRegularClose, symbols, 10)
		priorPath := filepath.Join(priorDirectory, facts.PriorSessionDate+".json")
		priorBody, err := os.ReadFile(priorPath)
		if err != nil {
			t.Fatal(err)
		}
		priorResolver := &PriorCloseResolver{Schedule: schedule, Now: func() time.Time { return now }}

		mutations := []struct {
			name string
			edit func([]byte, bool) []byte
		}{
			{"duplicate", func(body []byte, _ bool) []byte { return append([]byte(`{"schema_version":"duplicate",`), body[1:]...) }},
			{"unknown", func(body []byte, _ bool) []byte { return append(body[:len(body)-2], []byte(`,"unknown":true}\n`)...) }},
			{"count", corruptCacheCount},
			{"order", corruptCacheOrder},
			{"identity", corruptCacheIdentity},
			{"malformed", func(_ []byte, _ bool) []byte { return []byte(`{"schema_version"`) }},
			{"trailing JSON", func(body []byte, _ bool) []byte { return append(body, []byte(`{}`)...) }},
		}
		for _, cacheClass := range []struct {
			name    string
			body    []byte
			path    string
			isPrior bool
			read    func() error
		}{
			{"universe", universeBody, universePath, false, func() error {
				_, err := universeResolver.readUniverseCache(universePath, facts.TradingDate, facts.TradingDate, 0)
				return err
			}},
			{"prior-close", priorBody, priorPath, true, func() error {
				_, err := priorResolver.readPriorCloseCache(priorDirectory, facts.PriorSessionDate, facts.PriorRegularClose, symbols)
				return err
			}},
		} {
			for _, mutation := range mutations {
				t.Run(cacheClass.name+"/"+mutation.name, func(t *testing.T) {
					writePrivateFile(t, cacheClass.path, mutation.edit(slices.Clone(cacheClass.body), cacheClass.isPrior))
					if err := cacheClass.read(); err == nil {
						t.Fatal("corrupt cache accepted")
					}
					writePrivateFile(t, cacheClass.path, cacheClass.body)
				})
			}
		}
	})
}

func TestRootedPrivatePersistenceAndBoundedRetentionTable(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	symbols := []string{"AAA"}
	universe := acceptedUniverse(t, facts.TradingDate, symbols)
	now := time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC)

	t.Run("fresh facts survive every persistence stage", func(t *testing.T) {
		stages := []struct {
			name   string
			reason string
			ops    *cacheFileOps
			prune  bool
			data   bool
		}{
			{"prepare", persistenceCachePrepareFailed, nil, false, false},
			{"file sync", persistenceCachePublicationFailed, &cacheFileOps{syncFile: func(*os.File) error { return errors.New("injected") }}, false, true},
			{"rename", persistenceCachePublicationFailed, &cacheFileOps{rename: func(string, string) error { return errors.New("injected") }}, false, true},
			{"directory sync", persistenceCachePublicationFailed, &cacheFileOps{syncDirectory: func(string) error { return errors.New("injected") }}, false, true},
			{"prune", persistenceCachePruneFailed, nil, true, true},
		}
		for _, stage := range stages {
			for _, cacheClass := range []string{"universe", "prior-close"} {
				t.Run(cacheClass+"/"+stage.name, func(t *testing.T) {
					dataDir := ""
					if stage.data {
						dataDir = filepath.Join(canonicalTempDir(t), "reference")
					}
					if cacheClass == "universe" {
						server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
							writePage(t, writer, []tickerRecord{{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, "")
						}))
						defer server.Close()
						resolver := testResolver(t, schedule, server)
						resolver.DataDir, resolver.Now, resolver.cacheOps = dataDir, func() time.Time { return now }, stage.ops
						if stage.prune {
							resolver.prune = func(string) error { return errors.New("injected") }
						}
						got, err := resolver.Resolve(context.Background(), facts)
						if err != nil || got.Source() != SourceFresh || !slices.Equal(got.Symbols(), symbols) || got.PersistenceReason() != stage.reason {
							t.Fatalf("fresh universe containment = %+v err=%v", got, err)
						}
						return
					}

					server := groupedServer(fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":1,"results":[{"T":"AAA","c":10,"t":%d}]}`, facts.PriorRegularClose.UnixMilli()))
					defer server.Close()
					resolver := testPriorResolver(t, schedule, server)
					resolver.DataDir, resolver.Now, resolver.cacheOps = dataDir, func() time.Time { return now }, stage.ops
					if stage.prune {
						resolver.prune = func(string) error { return errors.New("injected") }
					}
					got, err := resolver.Resolve(context.Background(), facts, universe)
					closeValue, valid := got.Facts()[0].Close()
					if err != nil || got.Source() != SourceFresh || !valid || closeValue != 10 || got.PersistenceReason() != stage.reason {
						t.Fatalf("fresh prior containment = %+v err=%v", got, err)
					}
				})
			}
		}
	})

	t.Run("ancestor and file symlinks plus nonprivate paths are rejected", func(t *testing.T) {
		for _, cacheClass := range []string{"universe", "prior-close"} {
			t.Run(cacheClass+" ancestor", func(t *testing.T) {
				base := canonicalTempDir(t)
				realParent := filepath.Join(base, "real")
				if err := os.Mkdir(realParent, 0o700); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(base, "link")
				if err := os.Symlink(realParent, link); err != nil {
					t.Fatal(err)
				}
				assertFreshWithUnsafeDataDir(t, cacheClass, filepath.Join(link, "reference"), schedule, facts, universe, now, persistenceCachePrepareFailed)
			})
			t.Run(cacheClass+" nonprivate root", func(t *testing.T) {
				root := filepath.Join(canonicalTempDir(t), "reference")
				if err := os.Mkdir(root, 0o755); err != nil {
					t.Fatal(err)
				}
				assertFreshWithUnsafeDataDir(t, cacheClass, root, schedule, facts, universe, now, persistenceCachePrepareFailed)
			})
		}

		universeDirectory := canonicalTempDir(t)
		universeTarget := filepath.Join(canonicalTempDir(t), "target.json")
		writePrivateFile(t, universeTarget, []byte("not cache evidence"))
		universePath := filepath.Join(universeDirectory, facts.TradingDate+".json")
		if err := os.Symlink(universeTarget, universePath); err != nil {
			t.Fatal(err)
		}
		resolver := &Resolver{Schedule: schedule, Now: func() time.Time { return now }}
		if _, err := resolver.readUniverseCache(universePath, facts.TradingDate, facts.TradingDate, 0); err == nil {
			t.Fatal("universe file symlink accepted")
		}

		priorDirectory := canonicalTempDir(t)
		priorPath := filepath.Join(priorDirectory, facts.PriorSessionDate+".json")
		if err := os.Symlink(universeTarget, priorPath); err != nil {
			t.Fatal(err)
		}
		priorResolver := &PriorCloseResolver{Schedule: schedule, Now: func() time.Time { return now }}
		if _, err := priorResolver.readPriorCloseCache(priorDirectory, facts.PriorSessionDate, facts.PriorRegularClose, symbols); err == nil {
			t.Fatal("prior-close file symlink accepted")
		}
	})

	t.Run("private atomic files exclude credentials and preserve last complete replacement", func(t *testing.T) {
		for _, cacheClass := range []string{"universe", "prior-close"} {
			directory := canonicalTempDir(t)
			var path string
			if cacheClass == "universe" {
				publishUniverseFixture(t, directory, facts.TradingDate, now.Add(-time.Hour), symbols)
				path = filepath.Join(directory, facts.TradingDate+".json")
			} else {
				publishPriorFixture(t, directory, facts, facts.PriorRegularClose, symbols, 10)
				path = filepath.Join(directory, facts.PriorSessionDate+".json")
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(before), syntheticCredential) {
				t.Fatal("credential persisted")
			}
			if info, err := validatePrivateRegularFile(path); err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("cache file permissions = %v err=%v", info, err)
			}
			prefix := ".universe-"
			if cacheClass == "prior-close" {
				prefix = ".prior-close-"
			}
			err = publishPrivateCacheFile(directory, filepath.Base(path), prefix, []byte("replacement"), &cacheFileOps{rename: func(string, string) error { return errors.New("interrupted") }})
			if err == nil {
				t.Fatal("injected interrupted replacement succeeded")
			}
			after, err := os.ReadFile(path)
			if err != nil || !slices.Equal(before, after) {
				t.Fatal("last complete cache changed after interrupted replacement")
			}
		}
	})

	t.Run("retention temporary ages and enumeration are independently bounded", func(t *testing.T) {
		for _, cacheClass := range []string{"universe", "prior-close"} {
			directory := canonicalTempDir(t)
			prefix := ".universe-"
			prune := func() error { return pruneUniverseCachesAt(directory, now) }
			if cacheClass == "prior-close" {
				prefix = ".prior-close-"
				prune = func() error { return prunePriorCloseCachesAt(directory, now) }
			}
			for day := 1; day <= 15; day++ {
				writePrivateFile(t, filepath.Join(directory, fmt.Sprintf("2026-06-%02d.json", day)), []byte("complete"))
			}
			oldTemporary := filepath.Join(directory, prefix+"old.tmp")
			youngTemporary := filepath.Join(directory, prefix+"young.tmp")
			unknown := filepath.Join(directory, "user-owned")
			for _, path := range []string{oldTemporary, youngTemporary, unknown} {
				writePrivateFile(t, path, []byte("partial"))
			}
			if err := os.Chtimes(oldTemporary, now.Add(-operationTimeout-time.Second), now.Add(-operationTimeout-time.Second)); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(youngTemporary, now.Add(-operationTimeout), now.Add(-operationTimeout)); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(unknown, now.Add(-24*time.Hour), now.Add(-24*time.Hour)); err != nil {
				t.Fatal(err)
			}
			if err := prune(); err != nil {
				t.Fatal(err)
			}
			assertDateFileCount(t, directory, maximumCacheDates)
			if _, err := os.Lstat(oldTemporary); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("old recognized temporary retained")
			}
			for _, path := range []string{youngTemporary, unknown} {
				if _, err := os.Lstat(path); err != nil {
					t.Fatalf("bounded retained entry removed: %s: %v", path, err)
				}
			}

			boundedDirectory := canonicalTempDir(t)
			for index := 0; index <= maximumCacheDirectoryEntries; index++ {
				writePrivateFile(t, filepath.Join(boundedDirectory, fmt.Sprintf("unexpected-%02d", index)), []byte("x"))
			}
			if cacheClass == "universe" {
				if _, err := (&Resolver{}).loadFallback(boundedDirectory, facts.TradingDate); err == nil || !strings.Contains(err.Error(), "64 entries") {
					t.Fatalf("universe enumeration bound error = %v", err)
				}
			} else if err := prunePriorCloseCachesAt(boundedDirectory, now); err == nil || !strings.Contains(err.Error(), "64 entries") {
				t.Fatalf("prior-close enumeration bound error = %v", err)
			}

			fullDirectory := canonicalTempDir(t)
			for index := 0; index < maximumCacheDirectoryEntries; index++ {
				writePrivateFile(t, filepath.Join(fullDirectory, fmt.Sprintf("unexpected-%02d", index)), []byte("x"))
			}
			newPath := filepath.Join(fullDirectory, "2026-07-29.json")
			if err := publishPrivateCacheFile(fullDirectory, filepath.Base(newPath), prefix, []byte("complete"), nil); err == nil {
				t.Fatal("publication exceeded the 64-entry retention bound")
			}
			if _, err := os.Lstat(newPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("bounded publication created an additional artifact")
			}
		}
	})
}

func publishUniverseFixture(t *testing.T, directory, date string, retrievedAt time.Time, symbols []string) {
	t.Helper()
	identity, err := universeIdentity(date, EligibilityPolicyVersion, symbols)
	if err != nil {
		t.Fatal(err)
	}
	data := normalizedUniverse{
		referenceDate: date,
		retrievedAt:   retrievedAt,
		identity:      identity,
		symbols:       slices.Clone(symbols),
		accounting:    Accounting{RawReferenceRecords: len(symbols), EligibleRecords: len(symbols)},
	}
	if err := publishUniverseCache(directory, data); err != nil {
		t.Fatal(err)
	}
}

func publishPriorFixture(t *testing.T, directory string, facts session.Facts, retrievedAt time.Time, symbols []string, closeValue float64) {
	t.Helper()
	factsBySymbol := make([]PriorCloseFact, len(symbols))
	for index, symbol := range symbols {
		factsBySymbol[index] = PriorCloseFact{symbol: symbol, status: PriorCloseValid, close: closeValue + float64(index)}
	}
	identity, err := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, factsBySymbol)
	if err != nil {
		t.Fatal(err)
	}
	data := normalizedPriorCloses{
		priorSessionDate: facts.PriorSessionDate,
		retrievedAt:      retrievedAt,
		identity:         identity,
		facts:            factsBySymbol,
		accounting:       accountPriorCloses(factsBySymbol, 0),
	}
	if err := publishPriorCloseCache(directory, data); err != nil {
		t.Fatal(err)
	}
}

func writePrivateFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

func replaceJSONTimestamp(t *testing.T, body []byte, timestamp string) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	value["retrieved_at"] = timestamp
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func bytesContainCanonicalTimestamp(body []byte, timestamp string) bool {
	return strings.Contains(string(body), `"retrieved_at": "`+timestamp+`"`)
}

func corruptCacheCount(body []byte, prior bool) []byte {
	if prior {
		var cache priorCloseCache
		_ = json.Unmarshal(body, &cache)
		cache.Accounting.UniverseTotal++
		encoded, _ := json.Marshal(cache)
		return encoded
	}
	var cache universeCache
	_ = json.Unmarshal(body, &cache)
	cache.Accounting.EligibleRecords++
	encoded, _ := json.Marshal(cache)
	return encoded
}

func corruptCacheOrder(body []byte, prior bool) []byte {
	if prior {
		var cache priorCloseCache
		_ = json.Unmarshal(body, &cache)
		cache.Facts[0], cache.Facts[1] = cache.Facts[1], cache.Facts[0]
		encoded, _ := json.Marshal(cache)
		return encoded
	}
	var cache universeCache
	_ = json.Unmarshal(body, &cache)
	cache.Symbols[0], cache.Symbols[1] = cache.Symbols[1], cache.Symbols[0]
	encoded, _ := json.Marshal(cache)
	return encoded
}

func corruptCacheIdentity(body []byte, prior bool) []byte {
	schema := universeIdentitySchema
	if prior {
		schema = priorCloseIdentitySchema
	}
	start := schema + ":"
	text := string(body)
	index := strings.Index(text, start)
	if index < 0 {
		return body
	}
	index += len(start)
	return []byte(text[:index] + strings.Repeat("0", 64) + text[index+64:])
}

func assertFreshWithUnsafeDataDir(t *testing.T, cacheClass, dataDir string, schedule *session.Schedule, facts session.Facts, universe Universe, now time.Time, reason string) {
	t.Helper()
	if cacheClass == "universe" {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writePage(t, writer, []tickerRecord{{Ticker: "AAA", Active: true, Market: "stocks", Locale: "us", Type: "CS"}}, "")
		}))
		defer server.Close()
		resolver := testResolver(t, schedule, server)
		resolver.DataDir, resolver.Now = dataDir, func() time.Time { return now }
		got, err := resolver.Resolve(context.Background(), facts)
		if err != nil || got.Source() != SourceFresh || got.PersistenceReason() != reason {
			t.Fatalf("unsafe universe path containment = %+v err=%v", got, err)
		}
		return
	}
	server := groupedServer(fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":1,"results":[{"T":"AAA","c":10,"t":%d}]}`, facts.PriorRegularClose.UnixMilli()))
	defer server.Close()
	resolver := testPriorResolver(t, schedule, server)
	resolver.DataDir, resolver.Now = dataDir, func() time.Time { return now }
	got, err := resolver.Resolve(context.Background(), facts, universe)
	if err != nil || got.Source() != SourceFresh || got.PersistenceReason() != reason {
		t.Fatalf("unsafe prior path containment = %+v err=%v", got, err)
	}
}
