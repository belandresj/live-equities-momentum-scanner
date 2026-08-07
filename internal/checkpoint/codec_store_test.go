package checkpoint

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fixtureImage(sequence uint64, symbols int) Image {
	t0 := time.Date(2026, 8, 7, 14, 30, 0, 0, time.UTC)
	image := Image{SchemaVersion: SchemaV1, ProducerMode: ProducerLive, Binding: Binding{Identity: "binding-a", SessionStart: t0.Add(-30 * time.Minute), SessionEnd: t0.Add(6 * time.Hour)}, T0: t0, CreatedAt: t0.Add(time.Second), Sequence: sequence, Population: symbols, Counts: StructureCounts{EmptyStateRecords: symbols}, Symbols: make([]Symbol, symbols)}
	for i := range image.Symbols {
		image.Symbols[i].Symbol = fmt.Sprintf("S%06d", i)
	}
	return image
}

func maximalSchemaFixture() Image {
	image := fixtureImage(2, MaximumSymbols)
	image.Binding.SessionStart = image.T0.Add(-16 * time.Hour)
	image.Binding.SessionEnd = image.T0
	s := &image.Symbols[0]
	s.HasState = true
	image.Counts.RecordsWithState, image.Counts.EmptyStateRecords = 1, len(image.Symbols)-1
	for i := 0; i < 961; i++ {
		start := image.T0.Add(-time.Duration(961-i) * time.Second)
		s.Tail = append(s.Tail, Aggregate{WindowStart: start, WindowEnd: start.Add(time.Second), Values: Values{Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: "live_provider_average"}})
	}
	s.PriceRange = &PriceRange{HasFirst: true, FirstStart: image.Binding.SessionStart.Unix(), FirstOpen: 10, HasSessionExtrema: true, SessionHigh: 10, SessionLow: 10}
	for i := 0; i < 57_600; i++ {
		point := ExtremaPoint{WindowStart: image.Binding.SessionStart.Add(time.Duration(i) * time.Second).Unix(), Value: 10}
		s.PriceRange.SessionHighs = append(s.PriceRange.SessionHighs, point)
		s.PriceRange.SessionLows = append(s.PriceRange.SessionLows, point)
		if i < 3_600 {
			s.PriceRange.Highs = append(s.PriceRange.Highs, point)
			s.PriceRange.Lows = append(s.PriceRange.Lows, point)
		}
	}
	s.Activity = &Activity{}
	for i := 0; i < 1_920; i++ {
		end := image.Binding.SessionStart.Add(time.Duration(i+1) * 30 * time.Second).Unix()
		s.Activity.References = append(s.Activity.References, ActivitySummary{End: end, Invalid: true})
		target := ActivityTargetBlock{End: end, Present: (1 << 30) - 1}
		for slot := range target.Transactions {
			target.Transactions[slot], target.Highs[slot], target.Lows[slot] = 10, 10, 10
		}
		s.Activity.FoldedTargets = append(s.Activity.FoldedTargets, target)
	}
	for i := 0; i < 33; i++ {
		end := image.T0.Add(-time.Duration(32-i) * 30 * time.Second).Unix()
		s.Activity.Mutable = append(s.Activity.Mutable, ActivityMutable{End: end, Folded: ActivitySummary{End: end, Invalid: true}, Current: ActivitySummary{End: end, Invalid: true}})
	}
	s.Activity.FoldedTargetContributions = 57_600
	s.Qualification = &Qualification{AccountedThrough: image.T0}
	for i := 0; i < 57_600; i++ {
		start := image.Binding.SessionStart.Add(time.Duration(i) * time.Second).Unix()
		s.Qualification.FinalizedGateBars = append(s.Qualification.FinalizedGateBars, QualificationGateBar{Start: start, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: "live_provider_average"})
	}
	for i := 0; i < 961; i++ {
		value := image.T0.Add(-time.Duration(961-i) * time.Second).Unix()
		s.Qualification.Proofs = append(s.Qualification.Proofs, value)
		s.Qualification.Dirty = append(s.Qualification.Dirty, value)
	}
	image.Counts.TailRecords = len(s.Tail)
	image.Counts.PriceExtremaPoints = len(s.PriceRange.Highs) + len(s.PriceRange.Lows) + len(s.PriceRange.SessionHighs) + len(s.PriceRange.SessionLows)
	image.Counts.ActivityReferences, image.Counts.ActivityMutable, image.Counts.ActivityTargetBlocks, image.Counts.ActivityTargetContributions = len(s.Activity.References), len(s.Activity.Mutable), len(s.Activity.FoldedTargets), s.Activity.FoldedTargetContributions
	image.Counts.QualificationGateBars, image.Counts.QualificationProofs, image.Counts.QualificationDirty = len(s.Qualification.FinalizedGateBars), len(s.Qualification.Proofs), len(s.Qualification.Dirty)
	return image
}

func encodeFixture(t *testing.T, image Image, limit int64) []byte {
	t.Helper()
	var output bytes.Buffer
	if _, _, err := Encode(context.Background(), &output, image, limit); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func envelopeWithPayload(t *testing.T, payload []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(payload)
	data, err := json.Marshal(envelope{Format: EnvelopeV1, Payload: payload, PayloadSHA256: hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestC7CODEC01StrictDeterministicBounded is P-C7-CODEC. It proves raw
// payload identity and strict failure containment, not filesystem durability
// or engine installation.
func TestC7CODEC01StrictDeterministicBounded(t *testing.T) {
	image := fixtureImage(1, 2)
	a, b := encodeFixture(t, image, 1<<20), encodeFixture(t, image.Clone(), 1<<20)
	if !bytes.Equal(a, b) {
		t.Fatal("equal semantic images encoded differently")
	}
	candidate, n, err := Decode(context.Background(), bytes.NewReader(a), 1<<20)
	if err != nil || n != int64(len(a)) || !candidate.Integrity || !reflect.DeepEqual(candidate.Image, image) {
		t.Fatalf("round trip n=%d err=%v candidate=%+v", n, err, candidate)
	}
	fractionalCreated := image.Clone()
	fractionalCreated.CreatedAt = fractionalCreated.CreatedAt.Add(time.Nanosecond)
	fractionalArtifact := encodeFixture(t, fractionalCreated, 1<<20)
	if got, _, err := Decode(context.Background(), bytes.NewReader(fractionalArtifact), 1<<20); err != nil || !reflect.DeepEqual(got.Image, fractionalCreated) {
		t.Fatalf("fractional UTC CreatedAt control err=%v equal=%v", err, reflect.DeepEqual(got.Image, fractionalCreated))
	}

	payload, _ := json.Marshal(image)
	corruptions := map[string][]byte{
		"truncated":        a[:len(a)-1],
		"trailing value":   append(append([]byte(nil), a...), []byte(` {}`)...),
		"unknown envelope": bytes.Replace(a, []byte(`"format":`), []byte(`"unknown":1,"format":`), 1),
		"nested duplicate": envelopeWithPayload(t, bytes.Replace(payload, []byte(`"Identity":"binding-a"`), []byte(`"Identity":"binding-a","Identity":"other"`), 1)),
		"unknown payload":  envelopeWithPayload(t, bytes.Replace(payload, []byte(`"Population":2`), []byte(`"Unexpected":1,"Population":2`), 1)),
		"length overflow":  envelopeWithPayload(t, bytes.Replace(payload, []byte(`"Population":2`), []byte(`"Population":100001`), 1)),
		"numeric overflow": envelopeWithPayload(t, bytes.Replace(payload, []byte(`"Population":2`), []byte(`"Population":1e999`), 1)),
		"digest mismatch":  bytes.Replace(a, []byte(`"payload_sha256":"`), []byte(`"payload_sha256":"0`), 1),
	}
	fixed := fixtureImage(3, 1)
	fixed.Symbols[0].HasState = true
	fixed.Symbols[0].Activity = &Activity{References: []ActivitySummary{{End: fixed.T0.Unix(), Invalid: true}}, FoldedTargets: []ActivityTargetBlock{{End: fixed.T0.Unix()}}}
	fixed.Counts = StructureCounts{RecordsWithState: 1, ActivityReferences: 1, ActivityTargetBlocks: 1}
	fixedPayload, _ := json.Marshal(fixed)
	for _, key := range []string{"TransactionSum", "Transactions", "Highs", "Lows"} {
		corruptions["fixed array overflow "+key] = envelopeWithPayload(t, overflowJSONArray(t, fixedPayload, key))
	}
	semantic := fixtureImage(4, 1)
	semantic.Symbols[0].HasState = true
	semantic.Symbols[0].Tail = []Aggregate{{WindowStart: semantic.T0.Add(-time.Second), WindowEnd: semantic.T0, Values: Values{Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: "live_provider_average"}}}
	semantic.Counts = StructureCounts{RecordsWithState: 1, TailRecords: 1}
	semantic.Symbols[0].CommittedMark = pointerAggregate(semantic.Symbols[0].Tail[0])
	semantic.Symbols[0].PriceRange = &PriceRange{HasFirst: true, FirstStart: semantic.T0.Add(-time.Second).Unix(), FirstOpen: 10, HasSessionExtrema: true, SessionHigh: 10, SessionLow: 10, SessionHighs: []ExtremaPoint{{WindowStart: semantic.T0.Add(-time.Second).Unix(), Value: 10}}, SessionLows: []ExtremaPoint{{WindowStart: semantic.T0.Add(-time.Second).Unix(), Value: 10}}}
	semantic.Symbols[0].Qualification = &Qualification{AccountedThrough: semantic.T0, Proofs: []int64{semantic.T0.Unix()}, Dirty: []int64{semantic.T0.Unix()}}
	semantic.Counts = StructureCounts{RecordsWithState: 1, TailRecords: 1, PriceExtremaPoints: 2, QualificationProofs: 1, QualificationDirty: 1}
	semanticCases := map[string]func(*Image){"finite OHLC contradiction": func(i *Image) { i.Symbols[0].Tail[0].Values.High = 9 }, "ATS provenance": func(i *Image) { i.Symbols[0].Tail[0].Values.ATSProvenance = "fabricated" }, "window time": func(i *Image) { i.Symbols[0].Tail[0].WindowEnd = i.T0.Add(time.Second) }, "header time": func(i *Image) { i.CreatedAt = i.T0.Add(-time.Second) }, "T0 non-UTC": func(i *Image) {
		i.T0 = i.T0.In(time.FixedZone("non-utc", -7*60*60))
	}, "T0 fractional": func(i *Image) { i.T0 = i.T0.Add(time.Nanosecond) }, "created zero": func(i *Image) {
		i.CreatedAt = time.Time{}
	}, "created non-UTC": func(i *Image) {
		i.CreatedAt = i.CreatedAt.In(time.FixedZone("non-utc", -7*60*60))
	}, "committed latest": func(i *Image) {
		i.Symbols[0].CommittedMark.WindowStart = i.T0.Add(-2 * time.Second)
		i.Symbols[0].CommittedMark.WindowEnd = i.T0.Add(-time.Second)
	}, "committed canonical value": func(i *Image) {
		i.Symbols[0].CommittedMark.Values.Close = 11
	}, "committed older latest": func(i *Image) {
		older := i.Symbols[0].Tail[0]
		i.Symbols[0].CommittedMark.WindowStart = i.T0.Add(-2 * time.Second)
		i.Symbols[0].CommittedMark.WindowEnd = i.T0.Add(-time.Second)
		i.Symbols[0].OlderMark = &older
	}, "tail presence contradiction": func(i *Image) {
		i.Symbols[0].Presence = make([]uint64, 900)
		slot := int(i.Symbols[0].Tail[0].WindowStart.Sub(i.Binding.SessionStart) / time.Second)
		i.Symbols[0].Presence[slot/64] = uint64(1) << uint(slot%64)
		i.Counts.PresenceWords = 900
	}, "qualification accounted proof": func(i *Image) { i.Symbols[0].Qualification.AccountedThrough = i.T0.Add(-time.Second) }, "qualification dirty subset": func(i *Image) {
		i.Symbols[0].Qualification.Dirty[0]--
	}, "qualification final proof": func(i *Image) { i.Symbols[0].Qualification.FinalProofEnd = i.T0 }, "qualification finalized evidence": func(i *Image) {
		i.Symbols[0].Qualification.Finalized = true
		i.Symbols[0].Qualification.FinalProofEnd = i.T0
	}, "qualification invalid": func(i *Image) { i.Symbols[0].Qualification.Invalid = true }, "price paired extrema": func(i *Image) {
		i.Symbols[0].PriceRange.SessionLows[0].WindowStart--
	}, "activity cross-field": func(i *Image) {
		i.Symbols[0].Activity = &Activity{FoldedTargets: []ActivityTargetBlock{{End: i.T0.Unix(), Transactions: [30]float64{1}}}}
		i.Counts.ActivityTargetBlocks = 1
	}}
	for name, mutate := range semanticCases {
		invalid := semantic.Clone()
		mutate(&invalid)
		raw, _ := json.Marshal(invalid)
		corruptions["semantic "+name] = envelopeWithPayload(t, raw)
	}
	for name, data := range corruptions {
		t.Run(name, func(t *testing.T) {
			candidate, _, err := Decode(context.Background(), bytes.NewReader(data), 1<<20)
			if err == nil || !reflect.DeepEqual(candidate, Candidate{}) {
				t.Fatalf("corruption reached candidate: err=%v candidate=%+v", err, candidate)
			}
		})
	}
	arrayLimits := map[string]int{"Symbols": MaximumSymbols, "Symbols[].Tail": 961, "Symbols[].Presence": 900, "Symbols[].ProvenAbsent": 900, "Symbols[].HistoricalConflict": 900, "Symbols[].PriceRange.Highs": 3600, "Symbols[].PriceRange.Lows": 3600, "Symbols[].PriceRange.SessionHighs": 57600, "Symbols[].PriceRange.SessionLows": 57600, "Symbols[].Activity.References": 1920, "Symbols[].Activity.References[].TransactionSum": 34, "Symbols[].Activity.Mutable": 33, "Symbols[].Activity.Mutable[].Folded.TransactionSum": 34, "Symbols[].Activity.Mutable[].Current.TransactionSum": 34, "Symbols[].Activity.FoldedTargets": 1920, "Symbols[].Activity.FoldedTargets[].Transactions": 30, "Symbols[].Activity.FoldedTargets[].Highs": 30, "Symbols[].Activity.FoldedTargets[].Lows": 30, "Symbols[].Qualification.FinalizedGateBars": 57601, "Symbols[].Qualification.Proofs": 961, "Symbols[].Qualification.Dirty": 961}
	for path, limit := range arrayLimits {
		t.Run("path overflow "+path, func(t *testing.T) {
			if err := validateArrayLengthsReader(strings.NewReader(arrayOverflowDocument(path, limit))); err == nil {
				t.Fatal("path overflow accepted")
			}
		})
	}
	for path, length := range map[string]int{"Symbols[].Activity.References[].TransactionSum": 34, "Symbols[].Activity.Mutable[].Folded.TransactionSum": 34, "Symbols[].Activity.Mutable[].Current.TransactionSum": 34, "Symbols[].Activity.FoldedTargets[].Transactions": 30, "Symbols[].Activity.FoldedTargets[].Highs": 30, "Symbols[].Activity.FoldedTargets[].Lows": 30} {
		t.Run("path underlength "+path, func(t *testing.T) {
			if err := validateArrayLengthsReader(strings.NewReader(arrayLengthDocument(path, length-1))); err == nil {
				t.Fatal("fixed array underlength accepted")
			}
		})
	}
	fixedObjects := map[string][]string{
		"Symbols[].Activity.References[]":      {"TransactionSum"},
		"Symbols[].Activity.Mutable[].Folded":  {"TransactionSum"},
		"Symbols[].Activity.Mutable[].Current": {"TransactionSum"},
		"Symbols[].Activity.FoldedTargets[]":   {"Transactions", "Highs", "Lows"},
	}
	for path, fields := range fixedObjects {
		t.Run("fixed fields present "+path, func(t *testing.T) {
			if err := validateArrayLengthsReader(strings.NewReader(fixedObjectDocument(path, ""))); err != nil {
				t.Fatalf("exact-length control err=%v", err)
			}
		})
		for _, missing := range fields {
			t.Run("fixed field absent "+path+"."+missing, func(t *testing.T) {
				if err := validateArrayLengthsReader(strings.NewReader(fixedObjectDocument(path, missing))); err == nil {
					t.Fatal("absent fixed array accepted")
				}
			})
		}
	}
	if candidate, _, err := Decode(context.Background(), bytes.NewReader(a), int64(len(a)-1)); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(candidate, Candidate{}) {
		t.Fatalf("small limit err=%v candidate=%+v", err, candidate)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if candidate, _, err := Decode(canceled, bytes.NewReader(a), 1<<20); !errors.Is(err, ErrCanceled) || !reflect.DeepEqual(candidate, Candidate{}) {
		t.Fatalf("cancel err=%v candidate=%+v", err, candidate)
	}
	var canceledOutput bytes.Buffer
	if _, _, err := Encode(canceled, &canceledOutput, image, 1<<20); !errors.Is(err, ErrCanceled) || canceledOutput.Len() != 0 {
		t.Fatalf("encode cancel err=%v bytes=%d", err, canceledOutput.Len())
	}
	validationCtx, validationCancel := context.WithCancel(context.Background())
	if candidate, _, err := Decode(validationCtx, &cancelAtEOFReader{input: bytes.NewReader(a), cancel: validationCancel}, 1<<20); !errors.Is(err, ErrCanceled) || !reflect.DeepEqual(candidate, Candidate{}) {
		t.Fatalf("validation cancellation err=%v candidate=%+v", err, candidate)
	}
	var strictTarget Image
	if err := strictJSONReader(&cancelingReader{input: bytes.NewReader(payload), remaining: len(payload) / 2}, &strictTarget); !errors.Is(err, ErrCanceled) {
		t.Fatalf("mid-strict cancellation err=%v", err)
	}
	if err := rejectDuplicateKeysReader(&cancelingReader{input: bytes.NewReader(payload), remaining: len(payload) / 2}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("mid-duplicate-token cancellation err=%v", err)
	}
	if err := validateArrayLengthsReader(&cancelingReader{input: bytes.NewReader(payload), remaining: len(payload) / 2}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("mid-array-token cancellation err=%v", err)
	}
	semanticCancel := &cancelAfterChecksContext{remaining: 10}
	if err := validateSemanticImage(semanticCancel, maximalSemanticLoopFixture()); !errors.Is(err, ErrCanceled) {
		t.Fatalf("mid-semantic cancellation err=%v", err)
	}

	// The race runtime deliberately adds allocation metadata and changes this
	// process-wide TotalAlloc measurement. The ordinary focused proof owns the
	// allocation ceiling; the race pass still exercises the complete codec
	// boundary above.
	if testing.Short() || raceDetectorEnabled {
		return
	}
	maximum := maximalSchemaFixture()
	// A 57,600-second session has only 57,600 distinct valid gate-bar starts;
	// the structural 57,601 ceiling is independently safe but cannot be reached
	// by a semantically valid image. Prove that limitation on the same maximal
	// graph before restoring its largest valid shape for the allocation proof.
	q := maximum.Symbols[0].Qualification
	q.FinalizedGateBars = append(q.FinalizedGateBars, q.FinalizedGateBars[len(q.FinalizedGateBars)-1])
	maximum.Counts.QualificationGateBars++
	if err := ValidateSemanticImage(maximum); !errors.Is(err, ErrInvalid) {
		t.Fatalf("impossible 57,601 gate bars err=%v", err)
	}
	q.FinalizedGateBars = q.FinalizedGateBars[:len(q.FinalizedGateBars)-1]
	maximum.Counts.QualificationGateBars--
	encoded := encodeFixture(t, maximum, 256<<20)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	type decodeOutcome struct {
		candidate Candidate
		err       error
	}
	outcome := make(chan decodeOutcome, 1)
	go func() {
		decoded, _, err := Decode(context.Background(), bytes.NewReader(encoded), 256<<20)
		outcome <- decodeOutcome{candidate: decoded, err: err}
	}()
	peakHeapInuse := before.HeapInuse
	ticker := time.NewTicker(25 * time.Millisecond)
	var result decodeOutcome
	for waiting := true; waiting; {
		select {
		case result = <-outcome:
			waiting = false
		case <-ticker.C:
			var sample runtime.MemStats
			runtime.ReadMemStats(&sample)
			peakHeapInuse = max(peakHeapInuse, sample.HeapInuse)
		}
	}
	ticker.Stop()
	decoded, err := result.candidate, result.err
	if err != nil || len(decoded.Image.Symbols) != MaximumSymbols {
		t.Fatalf("maximum shape err=%v symbols=%d", err, len(decoded.Image.Symbols))
	}
	var during runtime.MemStats
	runtime.ReadMemStats(&during)
	peakHeapInuse = max(peakHeapInuse, during.HeapInuse)
	if transient, ceiling := during.TotalAlloc-before.TotalAlloc, uint64(40*len(encoded))+uint64(128<<20); transient > ceiling {
		t.Fatalf("transient allocation=%d ceiling=%d bytes=%d", transient, ceiling, len(encoded))
	}
	if peak, ceiling := peakHeapInuse-before.HeapInuse, uint64(16*len(encoded))+uint64(256<<20); peak > ceiling {
		t.Fatalf("peak live heap=%d ceiling=%d bytes=%d", peak, ceiling, len(encoded))
	}
	decoded, maximum, encoded = Candidate{}, Image{}, nil
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.HeapInuse > before.HeapInuse+32<<20 {
		t.Fatalf("retained heap did not plateau: before=%d after=%d", before.HeapInuse, after.HeapInuse)
	}
}

type cancelAtEOFReader struct {
	input  *bytes.Reader
	cancel context.CancelFunc
}

type cancelingReader struct {
	input     *bytes.Reader
	remaining int
}

func (r *cancelingReader) Read(value []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, ErrCanceled
	}
	if len(value) > r.remaining {
		value = value[:r.remaining]
	}
	n, err := r.input.Read(value)
	r.remaining -= n
	return n, err
}

type cancelAfterChecksContext struct{ remaining int }

func (c *cancelAfterChecksContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterChecksContext) Done() <-chan struct{}       { return nil }
func (c *cancelAfterChecksContext) Value(any) any               { return nil }
func (c *cancelAfterChecksContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func maximalSemanticLoopFixture() Image {
	return fixtureImage(99, 100)
}

func pointerAggregate(value Aggregate) *Aggregate { return &value }

func (r *cancelAtEOFReader) Read(value []byte) (int, error) {
	n, err := r.input.Read(value)
	if errors.Is(err, io.EOF) {
		r.cancel()
	}
	return n, err
}

func overflowJSONArray(t *testing.T, payload []byte, key string) []byte {
	t.Helper()
	marker := []byte(`"` + key + `":[`)
	start := bytes.Index(payload, marker)
	if start < 0 {
		t.Fatalf("missing array %s", key)
	}
	start += len(marker)
	end := bytes.IndexByte(payload[start:], ']')
	if end < 0 {
		t.Fatalf("unterminated array %s", key)
	}
	end += start
	out := append([]byte(nil), payload[:end]...)
	if end > start {
		out = append(out, ',')
	}
	out = append(out, '0')
	out = append(out, payload[end:]...)
	return out
}

func arrayOverflowDocument(path string, limit int) string {
	return arrayLengthDocument(path, limit+1)
}

func arrayLengthDocument(path string, length int) string {
	segments := strings.Split(path, ".")
	values := ""
	if length > 0 {
		values = strings.Repeat("0,", length-1) + "0"
	}
	leaf := `{"` + strings.TrimSuffix(segments[len(segments)-1], "[]") + `":[` + values + `]}`
	for index := len(segments) - 2; index >= 0; index-- {
		name := strings.TrimSuffix(segments[index], "[]")
		if strings.HasSuffix(segments[index], "[]") {
			leaf = `{"` + name + `":[` + leaf + `]}`
		} else {
			leaf = `{"` + name + `":` + leaf + `}`
		}
	}
	return leaf
}

func fixedObjectDocument(path, missing string) string {
	object := "{}"
	if path == "Symbols[].Activity.FoldedTargets[]" {
		fields := make([]string, 0, 3)
		for _, field := range []string{"Transactions", "Highs", "Lows"} {
			if field != missing {
				fields = append(fields, `"`+field+`":`+fixedArrayJSON(30))
			}
		}
		object = "{" + strings.Join(fields, ",") + "}"
	} else if missing != "TransactionSum" {
		object = `{"TransactionSum":` + fixedArrayJSON(34) + `}`
	}
	value := object
	segments := strings.Split(path, ".")
	for index := len(segments) - 1; index >= 0; index-- {
		name := strings.TrimSuffix(segments[index], "[]")
		if strings.HasSuffix(segments[index], "[]") {
			value = `{"` + name + `":[` + value + `]}`
		} else {
			value = `{"` + name + `":` + value + `}`
		}
	}
	return value
}

func fixedArrayJSON(length int) string {
	if length == 0 {
		return "[]"
	}
	return "[" + strings.Repeat("0,", length-1) + "0]"
}

// TestC7STORE01ManifestAuthorityAndExactSteps is P-C7-STORE. It proves local
// manifest authority and injected syscall-step consequences, not power-loss
// behavior beyond the exercised filesystem primitives.
func TestC7STORE01ManifestAuthorityAndExactSteps(t *testing.T) {
	newStore := func(t *testing.T) *Store {
		t.Helper()
		store, err := NewStore(StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: "binding-a", ArtifactByteLimit: 4 << 20, OperationDeadline: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	store := newStore(t)
	if got := store.Write(context.Background(), fixtureImage(1, 2)); got.Disposition != WriteCompleted {
		t.Fatalf("first write=%+v", got)
	}
	if got := store.Write(context.Background(), fixtureImage(2, 2)); got.Disposition != WriteCompleted {
		t.Fatalf("second write=%+v", got)
	}
	for _, name := range []string{"manifest.json", "checkpoint-00000000000000000002.json"} {
		info, err := os.Stat(filepath.Join(store.directory, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("private mode %s info=%v err=%v", name, info, err)
		}
	}
	if got := store.Load(context.Background()); got.Disposition != LoadedLatest || got.Candidate.Image.Sequence != 2 {
		t.Fatalf("latest=%+v", got)
	}
	if err := os.WriteFile(filepath.Join(store.directory, "checkpoint-00000000000000000002.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(context.Background()); got.Disposition != LoadedPrevious || got.Candidate.Image.Sequence != 1 {
		t.Fatalf("fallback=%+v", got)
	}
	if err := os.WriteFile(filepath.Join(store.directory, "checkpoint-99999999999999999999.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(context.Background()); got.Disposition != LoadedPrevious || got.Candidate.Image.Sequence != 1 {
		t.Fatalf("orphan became authority=%+v", got)
	}
	if err := os.Remove(filepath.Join(store.directory, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(context.Background()); got.Disposition != LoadUnavailable {
		t.Fatalf("missing manifest=%+v", got)
	}

	steps := []WriteStep{StepTempCreate, StepPayloadEncode, StepFileSync, StepFileClose, StepReopenValidation, StepGenerationRename, StepGenerationDirSync, StepManifestCreate, StepManifestEncode, StepManifestSync, StepManifestClose, StepManifestReopen, StepManifestRename, StepManifestDirSync, StepCleanup}
	for _, injected := range steps {
		t.Run(string(injected), func(t *testing.T) {
			s := newStore(t)
			if got := s.Write(context.Background(), fixtureImage(1, 2)); got.Disposition != WriteCompleted {
				t.Fatal(got)
			}
			s.SetStepHookForTest(func(step WriteStep) error {
				if step == injected {
					return errors.New("injected")
				}
				return nil
			})
			got := s.Write(context.Background(), fixtureImage(2, 2))
			if got.Step != injected {
				t.Fatalf("step=%s want=%s result=%+v", got.Step, injected, got)
			}
			loaded := s.Load(context.Background())
			postPublish := injected == StepManifestDirSync || injected == StepCleanup
			want := uint64(1)
			if postPublish {
				want = 2
			}
			if loaded.Candidate.Image.Sequence != want {
				t.Fatalf("authority after %s=%+v want sequence %d", injected, loaded, want)
			}
			if injected == StepCleanup && got.Disposition != WriteCleanupDeferred {
				t.Fatalf("cleanup disposition=%+v", got)
			}
		})
	}

	t.Run("unsafe paths and bounded cleanup", func(t *testing.T) {
		root := t.TempDir()
		real := filepath.Join(root, "real")
		if err := os.Mkdir(real, 0700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(root, "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		if _, err := NewStore(StoreConfig{Directory: link, BindingIdentity: "binding-a", ArtifactByteLimit: 1 << 20, OperationDeadline: time.Second}); err == nil {
			t.Fatal("symlink directory accepted")
		}
		s := newStore(t)
		if got := s.Write(context.Background(), fixtureImage(1, 2)); got.Disposition != WriteCompleted {
			t.Fatal(got)
		}
		for i := 2; i < 260; i++ {
			name := fmt.Sprintf("checkpoint-%020d.json", i)
			if err := os.WriteFile(filepath.Join(s.directory, name), []byte(`{}`), 0600); err != nil {
				t.Fatal(err)
			}
		}
		got := s.Write(context.Background(), fixtureImage(300, 2))
		if got.Disposition != WriteCleanupDeferred || got.Step != StepCleanup {
			t.Fatalf("cleanup bound=%+v", got)
		}
		if loaded := s.Load(context.Background()); loaded.Disposition != LoadedLatest || loaded.Candidate.Image.Sequence != 300 {
			t.Fatalf("cleanup changed authority=%+v", loaded)
		}
	})

	t.Run("manifest and generation permissions links", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			alter func(*Store) error
			want  LoadDisposition
		}{
			{"manifest mode", func(s *Store) error { return os.Chmod(filepath.Join(s.directory, "manifest.json"), 0644) }, LoadUnavailable},
			{"generation mode", func(s *Store) error {
				return os.Chmod(filepath.Join(s.directory, "checkpoint-00000000000000000002.json"), 0644)
			}, LoadedPrevious},
			{"generation symlink", func(s *Store) error {
				latest := filepath.Join(s.directory, "checkpoint-00000000000000000002.json")
				if err := os.Remove(latest); err != nil {
					return err
				}
				return os.Symlink(filepath.Join(s.directory, "checkpoint-00000000000000000001.json"), latest)
			}, LoadedPrevious},
			{"generation hardlink", func(s *Store) error {
				latest := filepath.Join(s.directory, "checkpoint-00000000000000000002.json")
				if err := os.Remove(latest); err != nil {
					return err
				}
				return os.Link(filepath.Join(s.directory, "checkpoint-00000000000000000001.json"), latest)
			}, LoadInvalid},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newStore(t)
				if s.Write(context.Background(), fixtureImage(1, 2)).Disposition != WriteCompleted || s.Write(context.Background(), fixtureImage(2, 2)).Disposition != WriteCompleted {
					t.Fatal("seed")
				}
				if err := tc.alter(s); err != nil {
					t.Fatal(err)
				}
				if got := s.Load(context.Background()); got.Disposition != tc.want {
					t.Fatalf("load=%+v want=%s", got, tc.want)
				}
			})
		}
	})

	t.Run("many unrelated entries fail bounded cleanup after publication", func(t *testing.T) {
		s := newStore(t)
		if got := s.Write(context.Background(), fixtureImage(1, 2)); got.Disposition != WriteCompleted {
			t.Fatal(got)
		}
		for i := 0; i < 257; i++ {
			if err := os.WriteFile(filepath.Join(s.directory, fmt.Sprintf("unrelated-%03d", i)), []byte("x"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		got := s.Write(context.Background(), fixtureImage(2, 2))
		if got.Disposition != WriteCleanupDeferred || got.Step != StepCleanup {
			t.Fatalf("write=%+v", got)
		}
		if loaded := s.Load(context.Background()); loaded.Disposition != LoadedLatest || loaded.Candidate.Image.Sequence != 2 {
			t.Fatalf("authority=%+v", loaded)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		s := newStore(t)
		if got := s.Write(context.Background(), fixtureImage(1, 2)); got.Disposition != WriteCompleted {
			t.Fatal(got)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := s.Load(ctx); got.Disposition != LoadCanceled {
			t.Fatalf("load cancel=%+v", got)
		}
		if got := s.Write(ctx, fixtureImage(2, 2)); got.Disposition != WriteCanceled {
			t.Fatalf("write cancel=%+v", got)
		}
	})

}
