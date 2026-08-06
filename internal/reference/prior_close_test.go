package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func TestPriorCloseResolutionAndSymbolLocalContainment(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	symbols := []string{"AAA", "BADDATE", "DUP", "INF", "MISSING", "NEG", "aaa"}
	universe := acceptedUniverse(t, facts.TradingDate, symbols)
	validTimestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	wrongTimestamp := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC).UnixMilli()
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/v2/aggs/grouped/locale/us/market/stocks/2026-07-28" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("adjusted") != "true" || request.URL.Query().Get("include_otc") != "false" || len(request.URL.Query()) != 2 {
			t.Errorf("query = %v", request.URL.Query())
		}
		if request.Header.Get("Authorization") != "Bearer "+syntheticCredential {
			t.Error("missing synthetic bearer credential")
		}
		fmt.Fprintf(writer, `{"status":"OK","adjusted":true,"resultsCount":9,"results":[
			{"T":"aaa","c":5.5,"t":%d},
			{"T":"DUP","c":2,"t":%d},{"T":"DUP","c":3,"t":%d},
			{"T":"BADDATE","c":4,"t":%d},{"T":"NEG","c":0,"t":%d},
			{"T":"INF","c":"NaN","t":%d},{"T":"AAA","c":10.25,"t":%d},
			{"T":"OTHER","c":7,"t":%d},{"c":1,"t":%d}]}`,
			validTimestamp, validTimestamp, validTimestamp, wrongTimestamp, validTimestamp,
			validTimestamp, validTimestamp, validTimestamp, validTimestamp)
	}))
	defer server.Close()

	resolver := testPriorResolver(t, schedule, server)
	got, err := resolver.Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || got.Source() != SourceFresh || !strings.HasPrefix(got.Identity(), priorCloseIdentitySchema+":") {
		t.Fatalf("requests/source/identity = %d/%s/%s", requests.Load(), got.Source(), got.Identity())
	}
	want := []PriorCloseFact{
		{symbol: "AAA", status: PriorCloseValid, close: 10.25},
		{symbol: "BADDATE", status: PriorCloseInvalid, reason: PriorCloseWrongDate},
		{symbol: "DUP", status: PriorCloseInvalid, reason: PriorCloseDuplicate},
		{symbol: "INF", status: PriorCloseInvalid, reason: PriorCloseNonfinite},
		{symbol: "MISSING", status: PriorCloseMissing},
		{symbol: "NEG", status: PriorCloseInvalid, reason: PriorCloseNonpositive},
		{symbol: "aaa", status: PriorCloseValid, close: 5.5},
	}
	if !slices.Equal(got.Facts(), want) {
		t.Fatalf("facts = %#v, want %#v", got.Facts(), want)
	}
	wantAccounting := PriorCloseAccounting{
		UniverseTotal: 7, ValidPriorClose: 2, MissingPriorClose: 1, InvalidPriorClose: 4,
		InvalidOrMissingPriorClose: 5, UnattributableDiagnosticRows: 2,
	}
	if got.Accounting() != wantAccounting || !got.Accounting().valid() {
		t.Fatalf("accounting = %+v", got.Accounting())
	}
	returned := got.Facts()
	returned[0].close = 999
	if got.Facts()[0].close != 10.25 {
		t.Fatal("caller mutated immutable prior-close facts")
	}
}

func TestPriorCloseValidEmptyAndGlobalEnvelopeValidation(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB"})

	t.Run("valid empty maps every symbol to missing", func(t *testing.T) {
		server := groupedServer(`{"status":"OK","adjusted":true,"resultsCount":0,"results":[]}`)
		defer server.Close()
		got, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe)
		if err != nil {
			t.Fatal(err)
		}
		if got.Accounting() != (PriorCloseAccounting{UniverseTotal: 2, MissingPriorClose: 2, InvalidOrMissingPriorClose: 2}) {
			t.Fatalf("empty accounting = %+v", got.Accounting())
		}
	})

	for name, body := range map[string]string{
		"missing status":       `{"adjusted":true,"resultsCount":0,"results":[]}`,
		"non-OK status":        `{"status":"ERROR","adjusted":true,"resultsCount":0,"results":[]}`,
		"duplicate status":     `{"status":"OK","status":"ERROR","adjusted":true,"resultsCount":0,"results":[]}`,
		"duplicate adjustment": `{"status":"OK","adjusted":true,"adjusted":false,"resultsCount":0,"results":[]}`,
		"duplicate count":      `{"status":"OK","adjusted":true,"resultsCount":0,"resultsCount":1,"results":[]}`,
		"duplicate results":    `{"status":"OK","adjusted":true,"resultsCount":0,"results":[],"results":[]}`,
		"adjustment ambiguity": `{"status":"OK","adjusted":false,"resultsCount":0,"results":[]}`,
		"count mismatch":       `{"status":"OK","adjusted":true,"resultsCount":1,"results":[]}`,
		"missing results":      `{"status":"OK","adjusted":true,"resultsCount":0}`,
		"null results":         `{"status":"OK","adjusted":true,"resultsCount":0,"results":null}`,
		"trailing value":       `{"status":"OK","adjusted":true,"resultsCount":0,"results":[]} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := groupedServer(body)
			defer server.Close()
			if _, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe); err == nil {
				t.Fatal("globally ambiguous envelope was accepted")
			}
		})
	}
}

func TestPriorCloseRowStructuralContainment(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB", "MISSING"})
	timestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	body := fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":5,"results":[
		{"T":"AAA","c":10,"c":11,"t":%d},
		{"T":"BBB","c":20,"t":%d,"t":%d},
		{"T":"AAA","T":"MISSING","c":30,"t":%d},
		{"T":7,"c":40,"t":%d},
		{"T":"OTHER","c":50,"t":%d,"unknown":true}]}`,
		timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)
	server := groupedServer(body)
	defer server.Close()
	got, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	want := []PriorCloseFact{
		{symbol: "AAA", status: PriorCloseInvalid, reason: PriorCloseMalformed},
		{symbol: "BBB", status: PriorCloseInvalid, reason: PriorCloseMalformed},
		{symbol: "MISSING", status: PriorCloseMissing},
	}
	if !slices.Equal(got.Facts(), want) {
		t.Fatalf("facts = %#v, want %#v", got.Facts(), want)
	}
	if got.Accounting() != (PriorCloseAccounting{
		UniverseTotal: 3, InvalidPriorClose: 2, MissingPriorClose: 1,
		InvalidOrMissingPriorClose: 3, UnattributableDiagnosticRows: 3,
	}) {
		t.Fatalf("accounting = %+v", got.Accounting())
	}
	if got.Diagnostics() != (AcquisitionDiagnostics{Source: SourceFresh, RequestCount: 1, AttemptCount: 1, UnattributablePriorRows: 3, TerminalReason: TerminalReasonNone}) {
		t.Fatalf("diagnostics = %+v", got.Diagnostics())
	}
}

func TestPriorCloseIdentityIgnoresProviderRowOrder(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB"})
	timestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	rows := [2]string{
		fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":2,"results":[{"T":"AAA","c":10,"t":%d},{"T":"BBB","c":20,"t":%d}]}`, timestamp, timestamp),
		fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":2,"results":[{"T":"BBB","c":20,"t":%d},{"T":"AAA","c":10,"t":%d}]}`, timestamp, timestamp),
	}
	identities := [2]string{}
	for index, body := range rows {
		server := groupedServer(body)
		got, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe)
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		identities[index] = got.Identity()
	}
	if identities[0] != identities[1] {
		t.Fatalf("provider row order changed prior-close identity: %s != %s", identities[0], identities[1])
	}
}

func TestPriorCloseCachePolicyPersistenceAndPruning(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB"})
	timestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	server := groupedServer(fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":1,"results":[{"T":"AAA","c":10,"t":%d}]}`, timestamp))
	resolver := testPriorResolver(t, schedule, server)
	fresh, err := resolver.Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(resolver.DataDir, "prior-close", facts.PriorSessionDate+".json")
	if info, err := validatePrivateRegularFile(cachePath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private cache = %v, %v", info, err)
	}
	body, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), syntheticCredential) {
		t.Fatal("credential persisted in prior-close cache")
	}
	server.Close()
	cached, err := resolver.Resolve(context.Background(), facts, universe)
	if err != nil || cached.Source() != SourceCurrentCache || cached.Identity() != fresh.Identity() || !slices.Equal(cached.Facts(), fresh.Facts()) {
		t.Fatalf("exact-date cache = %+v, %v", cached, err)
	}

	mutations := map[string]func([]byte) []byte{
		"unknown field": func(body []byte) []byte { return append(body[:len(body)-2], []byte(`,"unknown":true}\n`)...) },
		"trailing JSON": func(body []byte) []byte { return append(body, []byte(`{}`)...) },
		"wrong policy": func(body []byte) []byte {
			return []byte(strings.Replace(string(body), PriorClosePolicyVersion, PriorClosePolicyVersion+"-changed", 1))
		},
		"noncanonical order": func(body []byte) []byte {
			var cache priorCloseCache
			if err := json.Unmarshal(body, &cache); err != nil {
				t.Fatal(err)
			}
			cache.Facts[0], cache.Facts[1] = cache.Facts[1], cache.Facts[0]
			encoded, err := json.Marshal(cache)
			if err != nil {
				t.Fatal(err)
			}
			return encoded
		},
		"inconsistent count": func(body []byte) []byte {
			var cache priorCloseCache
			if err := json.Unmarshal(body, &cache); err != nil {
				t.Fatal(err)
			}
			cache.Accounting.UniverseTotal++
			encoded, err := json.Marshal(cache)
			if err != nil {
				t.Fatal(err)
			}
			return encoded
		},
		"identity mismatch": func(body []byte) []byte {
			return []byte(strings.Replace(string(body), fresh.Identity(), priorCloseIdentitySchema+":"+strings.Repeat("0", 64), 1))
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			copyPath := filepath.Join(t.TempDir(), facts.PriorSessionDate+".json")
			if err := os.WriteFile(copyPath, mutate(slices.Clone(body)), 0o600); err != nil {
				t.Fatal(err)
			}
			directory := filepath.Dir(copyPath)
			if _, err := resolver.readPriorCloseCache(directory, facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols); err == nil {
				t.Fatal("corrupt cache accepted")
			}
		})
	}
	oversizedDirectory := t.TempDir()
	oversizedPath := filepath.Join(oversizedDirectory, facts.PriorSessionDate+".json")
	oversized, err := os.OpenFile(oversizedPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := oversized.Truncate(maximumCacheBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := oversized.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.readPriorCloseCache(oversizedDirectory, facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols); err == nil {
		t.Fatal("oversized cache accepted")
	}

	if err := os.Chmod(cachePath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.readPriorCloseCache(filepath.Dir(cachePath), facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols); err == nil {
		t.Fatal("nonprivate cache accepted")
	}
	if err := os.Chmod(cachePath, 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkDirectory := t.TempDir()
	if err := os.Symlink(cachePath, filepath.Join(symlinkDirectory, facts.PriorSessionDate+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.readPriorCloseCache(symlinkDirectory, facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols); err == nil {
		t.Fatal("symlink cache accepted")
	}

	priorDirectory := filepath.Join(resolver.DataDir, "prior-close")
	for day := 1; day <= 15; day++ {
		name := fmt.Sprintf("2026-06-%02d.json", day)
		if err := os.WriteFile(filepath.Join(priorDirectory, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := prunePriorCloseCaches(priorDirectory); err != nil {
		t.Fatal(err)
	}
	assertDateFileCount(t, priorDirectory, maximumCacheDates)

	universeDirectory := filepath.Join(resolver.DataDir, "universe")
	if err := os.Mkdir(universeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	for day := 1; day <= 15; day++ {
		if err := os.WriteFile(filepath.Join(universeDirectory, fmt.Sprintf("2026-05-%02d.json", day)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := pruneUniverseCaches(universeDirectory); err != nil {
		t.Fatal(err)
	}
	assertDateFileCount(t, universeDirectory, maximumCacheDates)

	// An interrupted publication leaves only a private temporary file beside
	// the last complete cache; readers continue to select the dated file.
	if err := os.WriteFile(filepath.Join(priorDirectory, ".prior-close-interrupted.tmp"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolver.readPriorCloseCache(priorDirectory, facts.PriorSessionDate, facts.PriorRegularClose, universe.symbols); err != nil || got.identity != fresh.Identity() {
		t.Fatalf("last complete cache after interrupted publication = %s, %v", got.identity, err)
	}
}

func TestFreshPriorCloseFactsSurvivePersistenceFailures(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA"})
	timestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	for name, test := range map[string]struct {
		configure func(*PriorCloseResolver)
		reason    string
	}{
		"publication": {func(resolver *PriorCloseResolver) {
			resolver.publish = func(string, normalizedPriorCloses) error { return errors.New("injected") }
		}, "cache_publication_failed"},
		"pruning": {func(resolver *PriorCloseResolver) {
			resolver.prune = func(string) error { return errors.New("injected") }
		}, "cache_prune_failed"},
	} {
		t.Run(name, func(t *testing.T) {
			server := groupedServer(fmt.Sprintf(`{"status":"OK","adjusted":true,"resultsCount":1,"results":[{"T":"AAA","c":10,"t":%d}]}`, timestamp))
			defer server.Close()
			resolver := testPriorResolver(t, schedule, server)
			test.configure(resolver)
			got, err := resolver.Resolve(context.Background(), facts, universe)
			if err != nil || got.Source() != SourceFresh || got.Facts()[0].close != 10 || got.PersistenceReason() != test.reason {
				t.Fatalf("fresh persistence degradation = %+v, %v", got, err)
			}
		})
	}
}

func TestPriorCloseBoundedFailurePolicy(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA"})
	validBody := `{"status":"OK","adjusted":true,"resultsCount":0,"results":[]}`

	t.Run("transient retry then success", func(t *testing.T) {
		var attempts atomic.Int64
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) == 1 {
				http.Error(writer, "retry", http.StatusInternalServerError)
				return
			}
			_, _ = io.WriteString(writer, validBody)
		}))
		defer server.Close()
		got, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe)
		if err != nil || attempts.Load() != 2 || got.Accounting().MissingPriorClose != 1 {
			t.Fatalf("retry = attempts %d, result %+v, err %v", attempts.Load(), got, err)
		}
	})

	t.Run("permanent status is not retried", func(t *testing.T) {
		var attempts atomic.Int64
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			attempts.Add(1)
			http.Error(writer, "bad", http.StatusBadRequest)
		}))
		defer server.Close()
		_, err := testPriorResolver(t, schedule, server).Resolve(context.Background(), facts, universe)
		if err == nil || attempts.Load() != 1 {
			t.Fatalf("permanent failure attempts = %d, err %v", attempts.Load(), err)
		}
	})

	t.Run("transport exhaustion after three attempts", func(t *testing.T) {
		var attempts atomic.Int64
		resolver := testPriorResolver(t, schedule, groupedServer(validBody))
		resolver.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("transport")
		})}
		_, err := resolver.Resolve(context.Background(), facts, universe)
		if err == nil || attempts.Load() != 3 {
			t.Fatalf("transport attempts = %d, err %v", attempts.Load(), err)
		}
	})

	t.Run("response read exhaustion after three attempts", func(t *testing.T) {
		var attempts atomic.Int64
		resolver := testPriorResolver(t, schedule, groupedServer(validBody))
		resolver.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(failingReader{})}, nil
		})}
		_, err := resolver.Resolve(context.Background(), facts, universe)
		if err == nil || attempts.Load() != 3 {
			t.Fatalf("read attempts = %d, err %v", attempts.Load(), err)
		}
	})

	t.Run("cancellation terminates retry", func(t *testing.T) {
		server := groupedServer("retry")
		defer server.Close()
		resolver := testPriorResolver(t, schedule, server)
		resolver.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("transport")
		})}
		resolver.Sleep = func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := resolver.Resolve(ctx, facts, universe); !errors.Is(err, context.Canceled) && !strings.Contains(fmt.Sprint(err), "context canceled") {
			t.Fatalf("cancellation error = %v", err)
		}
	})

	t.Run("deadline terminates before transport", func(t *testing.T) {
		var attempts atomic.Int64
		server := groupedServer(validBody)
		defer server.Close()
		resolver := testPriorResolver(t, schedule, server)
		resolver.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("transport")
		})}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		_, err := resolver.Resolve(ctx, facts, universe)
		if !errors.Is(err, context.DeadlineExceeded) || attempts.Load() != 0 {
			t.Fatalf("deadline attempts = %d, err %v", attempts.Load(), err)
		}
	})

	t.Run("empty credential is permanent configuration failure", func(t *testing.T) {
		server := groupedServer(validBody)
		defer server.Close()
		resolver := testPriorResolver(t, schedule, server)
		resolver.APIKey = ""
		if _, err := resolver.Resolve(context.Background(), facts, universe); err == nil {
			t.Fatal("empty credential accepted")
		}
	})
}

func acceptedUniverse(t *testing.T, date string, symbols []string) Universe {
	t.Helper()
	symbols = slices.Clone(symbols)
	slices.Sort(symbols)
	identity, err := universeIdentity(date, EligibilityPolicyVersion, symbols)
	if err != nil {
		t.Fatal(err)
	}
	return Universe{
		referenceDate: date, policyVersion: EligibilityPolicyVersion, identity: identity,
		symbols: symbols, accounting: Accounting{RawReferenceRecords: len(symbols), EligibleRecords: len(symbols)}, source: SourceFresh,
	}
}

func testPriorResolver(t *testing.T, schedule *session.Schedule, server *httptest.Server) *PriorCloseResolver {
	t.Helper()
	return &PriorCloseResolver{
		BaseURL: server.URL, APIKey: syntheticCredential, DataDir: filepath.Join(canonicalTempDir(t), "reference"),
		HTTPClient: server.Client(), Schedule: schedule,
		Now:   func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) },
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
}

func groupedServer(body string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(writer, body) }))
}

func assertDateFileCount(t *testing.T, directory string, want int) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if validCacheFilename(entry.Name()) {
			count++
		}
	}
	if count != want {
		t.Fatalf("date files in %s = %d, want %d", directory, count, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read") }

func acquisitionErrorDiagnostics(t *testing.T, err error) AcquisitionDiagnostics {
	t.Helper()
	var acquisitionErr *AcquisitionError
	if !errors.As(err, &acquisitionErr) {
		t.Fatalf("error %v is not a bounded acquisition error", err)
	}
	return acquisitionErr.Diagnostics()
}
