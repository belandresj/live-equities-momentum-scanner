package reference

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFloatResolverFreshPaginationAndContainment is P-MVP-FLOAT's fresh
// boundary: only complete paginated, exact-symbol, unambiguous facts become
// current; bad rows remain symbol-local and cannot invent a proxy value.
func TestFloatResolverFreshPaginationAndContainment(t *testing.T) {
	now := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" || r.URL.Query().Get("limit") != "5000" || r.URL.Query().Get("sort") != "ticker.asc" {
			t.Fatalf("unexpected request auth=%q query=%q", r.Header.Get("Authorization"), r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "two" {
			fmt.Fprint(w, `{"status":"OK","results":[`+
				`{"ticker":"BBB","free_float":200,"free_float_percent":25},`+
				`{"ticker":"CCC","free_float":300},`+
				`{"ticker":"CCC","free_float":301},`+
				`{"ticker":"DDD","free_float":400,"effective_date":"not-a-date"},`+
				`{"ticker":"ZZZ","free_float":999}`+
				`]}`)
			return
		}
		fmt.Fprintf(w, `{"status":"OK","request_id":"one","results":[`+
			`{"ticker":"AAA","free_float":100,"free_float_percent":12.5,"effective_date":"2026-08-01"},`+
			`{"ticker":"BAD","free_float":0}`+
			`],"next_url":%q}`, server.URL+"/stocks/vX/float?cursor=two&limit=5000&sort=ticker.asc")
	}))
	defer server.Close()

	lookup := (&FloatResolver{BaseURL: server.URL, APIKey: "secret", DataDir: privateTempDir(t), HTTPClient: server.Client(), Now: func() time.Time { return now }}).
		Resolve(context.Background(), testFloatUniverse("AAA", "BBB", "CCC", "DDD"))
	if lookup.Provenance() != FloatFresh || lookup.Len() != 2 {
		t.Fatalf("fresh lookup=%+v", lookup)
	}
	aaa, ok := lookup.Lookup("AAA")
	if !ok || aaa.FreeFloat != 100 || aaa.FreeFloatPercent == nil || *aaa.FreeFloatPercent != 12.5 || aaa.EffectiveDate != "2026-08-01" || aaa.Provider != floatSourceIdentity || !aaa.RetrievedAt.Equal(now) {
		t.Fatalf("AAA fact=%+v ok=%v", aaa, ok)
	}
	if bbb, ok := lookup.Lookup("BBB"); !ok || bbb.FreeFloat != 200 || bbb.EffectiveDate != "" {
		t.Fatalf("BBB fact=%+v ok=%v", bbb, ok)
	}
	for _, ticker := range []string{"CCC", "DDD", "ZZZ", "BAD"} {
		if fact, ok := lookup.Lookup(ticker); ok {
			t.Fatalf("ambiguous/malformed/wrong symbol %s became usable: %+v", ticker, fact)
		}
	}
	// Returned values cannot mutate the immutable lookup.
	copyFact, _ := lookup.Lookup("AAA")
	*copyFact.FreeFloatPercent = 99
	again, _ := lookup.Lookup("AAA")
	if *again.FreeFloatPercent != 12.5 {
		t.Fatalf("lookup aliases caller mutation: %+v", again)
	}
}

// TestFloatResolverFailedPartialFallsBackToValidatedCache proves that a later
// page failure cannot publish a partial refresh and that fallback provenance is
// explicitly cache, never fresh.
func TestFloatResolverFailedPartialFallsBackToValidatedCache(t *testing.T) {
	now := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	directory := privateTempDir(t)
	good := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"OK","results":[{"ticker":"AAA","free_float":123,"effective_date":"2026-07-31"}]}`)
	}))
	resolver := &FloatResolver{BaseURL: good.URL, APIKey: "secret", DataDir: directory, HTTPClient: good.Client(), Now: func() time.Time { return now }}
	if got := resolver.Resolve(context.Background(), testFloatUniverse("AAA", "BBB")); got.Provenance() != FloatFresh || got.Len() != 1 {
		t.Fatalf("seed fresh=%+v", got)
	}
	if got, err := resolver.readCache(filepath.Join(directory, "float", "latest.json"), []string{"AAA", "BBB"}); err != nil || got.Len() != 1 {
		t.Fatalf("seed cache got=%+v err=%v", got, err)
	}
	good.Close()

	var failing *httptest.Server
	failing = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "broken" {
			http.Error(w, "failed", http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, `{"status":"OK","results":[{"ticker":"BBB","free_float":999}],"next_url":%q}`, failing.URL+"/stocks/vX/float?cursor=broken")
	}))
	defer failing.Close()
	resolver.BaseURL, resolver.HTTPClient, resolver.Now = failing.URL, failing.Client(), func() time.Time { return now.Add(time.Hour) }
	cached := resolver.Resolve(context.Background(), testFloatUniverse("AAA", "BBB"))
	if cached.Provenance() != FloatCache || cached.Len() != 1 {
		t.Fatalf("fallback=%+v", cached)
	}
	if _, ok := cached.Lookup("BBB"); ok {
		t.Fatal("partial refresh became visible")
	}
	fact, ok := cached.Lookup("AAA")
	if !ok || fact.FreeFloat != 123 || fact.Provenance != FloatCache || fact.EffectiveDate != "2026-07-31" || !fact.RetrievedAt.Equal(now) {
		t.Fatalf("cached fact=%+v ok=%v", fact, ok)
	}

	if err := os.WriteFile(filepath.Join(directory, "float", "latest.json"), []byte(`{"schema_version":"wrong"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	unavailable := resolver.Resolve(context.Background(), testFloatUniverse("AAA"))
	if unavailable.Provenance() != FloatUnavailable || unavailable.Len() != 0 {
		t.Fatalf("corrupt cache became usable: %+v", unavailable)
	}
}

func TestFloatResolverPaginationAndAcquisitionBounds(t *testing.T) {
	base, err := url.Parse("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{}
	first, err := validateFloatPageURL(base, "/stocks/vX/float?cursor=one", seen)
	if err != nil {
		t.Fatal(err)
	}
	seen[first.String()] = struct{}{}
	if _, err := validateFloatPageURL(base, first.String(), seen); err == nil {
		t.Fatal("pagination cycle accepted")
	}
	if _, err := validateFloatPageURL(base, "https://other.test/stocks/vX/float?cursor=two", map[string]struct{}{}); err == nil {
		t.Fatal("changed-origin continuation accepted")
	}

	cases := []struct {
		name                         string
		page, bytes, pageRows, prior int
	}{
		{"page bound", maximumFloatPages, 0, 0, 0},
		{"response byte bound", 0, maximumFloatPageBytes + 1, 0, 0},
		{"page result bound", 0, 0, maximumFloatPageResult + 1, 0},
		{"record bound", maximumFloatPages - 1, 0, 1, maximumFloatRecords},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateFloatFetchBounds(tc.page, tc.bytes, tc.pageRows, tc.prior); err == nil {
				t.Fatal("Float acquisition bound accepted")
			}
		})
	}
	if err := validateFloatFetchBounds(maximumFloatPages-1, maximumFloatPageBytes, maximumFloatPageResult, maximumFloatRecords-maximumFloatPageResult); err != nil {
		t.Fatalf("exact Float acquisition bounds rejected: %v", err)
	}
}

func TestFloatCacheRejectsFutureAndMalformedMetadata(t *testing.T) {
	now := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	directory := privateTempDir(t)
	floatDirectory := filepath.Join(directory, "float")
	if err := os.MkdirAll(floatDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(floatDirectory, "latest.json")
	resolver := &FloatResolver{Now: func() time.Time { return now }}
	cases := []struct {
		name, retrievedAt string
	}{
		{"future", now.Add(time.Second).Format(time.RFC3339Nano)},
		{"malformed", "2026-08-14 18:00:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"schema_version":%q,"source":%q,"retrieved_at":%q,"facts":[{"ticker":"AAA","free_float":123}]}`,
				floatCacheSchema, floatSourceIdentity, tc.retrievedAt)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := resolver.readCache(path, []string{"AAA"}); err == nil {
				t.Fatal("invalid Float cache metadata accepted")
			}
		})
	}
}

func testFloatUniverse(symbols ...string) Universe {
	return Universe{symbols: symbols, source: SourceFresh}
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		t.Fatal(err)
	}
	if parsed, err := url.Parse(absolute); err != nil || parsed.Path == "" {
		t.Fatal("invalid temp path")
	}
	return absolute
}
