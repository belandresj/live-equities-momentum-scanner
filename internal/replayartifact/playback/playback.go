// Package playback performs the streaming validation pass that produces the
// opaque evidence consumed by the replay engine. It owns no engine state.
package playback

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaV1          = "aggregate-replay-jsonl-v1"
	CompleteFinalBars = "complete_final_bars"
	PartialSynthetic  = "partial_synthetic"
	completeCoverage  = "complete_interval"
	partialCoverage   = "declared_partial"
	massiveProvider   = "massive"
	syntheticProvider = "synthetic"
	massiveEndpoint   = "/v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}"
	massivePolicy     = "massive-rest-second-aggregate-v1"
	syntheticPolicy   = "synthetic-declared-v1"
)

type Plan struct {
	ArtifactID                         string
	BindingID, UniverseID, TradingDate string
	SessionStart, SessionEnd           time.Time
	Start, End                         time.Time
	Mode                               string
	Symbols                            []string
	MaximumBytes, MaximumRecords       int64
}

// ErrorClass is the bounded trust boundary at which playback rejected an
// artifact. It deliberately excludes paths, symbols, values, and wall time.
type ErrorClass string

const (
	ErrorArtifactValidation ErrorClass = "artifact_validation"
	ErrorBindingCoverage    ErrorClass = "binding_coverage"
	ErrorSchemaCanonical    ErrorClass = "schema_canonical_encoding"
	ErrorOrdinalGroup       ErrorClass = "ordinal_group_order"
	ErrorArtifactEnd        ErrorClass = "artifact_end_digest"
)

type playbackError struct {
	class   ErrorClass
	message string
}

func (e *playbackError) Error() string { return e.message }

func classified(class ErrorClass, message string) error {
	return &playbackError{class: class, message: message}
}

// Classify returns the bounded rejection reason carried by playback errors.
func Classify(err error) ErrorClass {
	var value *playbackError
	if errors.As(err, &value) {
		return value.class
	}
	return ErrorArtifactValidation
}

type Values struct {
	Open, High, Low, Close float64
	Volume, VWAP           float64
	AverageTradeSize       int64
	ATSProvenance          string
}

// Evidence values have private validity fields so ordinary callers cannot
// construct successful replay facts. Zero values are always rejected.
type StartAuthority uint8

const (
	ArtifactInterval StartAuthority = iota
	FreshSession
	InstalledCheckpoint
)

type StartEvidence struct {
	evidence  evidence
	authority StartAuthority
}
type RecordEvidence struct {
	evidence evidence
	record   record
}
type GroupEvidence struct {
	evidence    evidence
	logicalTime time.Time
	lastOrdinal uint64
}
type EndEvidence struct {
	evidence evidence
	ordinal  uint64
}

type evidence struct {
	valid, complete bool
	artifactID      string
	bindingID       string
	start, end      time.Time
	totalRecords    uint64
}

func (e StartEvidence) Valid() bool               { return e.evidence.valid }
func (e StartEvidence) Complete() bool            { return e.evidence.complete }
func (e StartEvidence) ArtifactID() string        { return e.evidence.artifactID }
func (e StartEvidence) BindingID() string         { return e.evidence.bindingID }
func (e StartEvidence) Start() time.Time          { return e.evidence.start }
func (e StartEvidence) End() time.Time            { return e.evidence.end }
func (e StartEvidence) TotalRecords() uint64      { return e.evidence.totalRecords }
func (e StartEvidence) Authority() StartAuthority { return e.authority }
func (e StartEvidence) WithAuthority(authority StartAuthority) StartEvidence {
	if authority <= InstalledCheckpoint {
		e.authority = authority
	}
	return e
}
func (e RecordEvidence) Valid() bool            { return e.evidence.valid }
func (e RecordEvidence) Complete() bool         { return e.evidence.complete }
func (e RecordEvidence) ArtifactID() string     { return e.evidence.artifactID }
func (e RecordEvidence) BindingID() string      { return e.evidence.bindingID }
func (e RecordEvidence) Ordinal() uint64        { return e.record.ordinal }
func (e RecordEvidence) LogicalTime() time.Time { return e.record.logicalTime }
func (e RecordEvidence) Symbol() string         { return e.record.symbol }
func (e RecordEvidence) WindowStart() time.Time { return e.record.windowStart }
func (e RecordEvidence) WindowEnd() time.Time   { return e.record.windowEnd }
func (e RecordEvidence) Values() Values         { return e.record.values }
func (e GroupEvidence) Valid() bool             { return e.evidence.valid }
func (e GroupEvidence) Complete() bool          { return e.evidence.complete }
func (e GroupEvidence) ArtifactID() string      { return e.evidence.artifactID }
func (e GroupEvidence) BindingID() string       { return e.evidence.bindingID }
func (e GroupEvidence) LogicalTime() time.Time  { return e.logicalTime }
func (e GroupEvidence) LastOrdinal() uint64     { return e.lastOrdinal }
func (e EndEvidence) Valid() bool               { return e.evidence.valid }
func (e EndEvidence) Complete() bool            { return e.evidence.complete }
func (e EndEvidence) ArtifactID() string        { return e.evidence.artifactID }
func (e EndEvidence) BindingID() string         { return e.evidence.bindingID }
func (e EndEvidence) End() time.Time            { return e.evidence.end }
func (e EndEvidence) TotalRecords() uint64      { return e.ordinal }

type Cursor struct {
	file                *os.File
	plan                Plan
	reader              *bufio.Reader
	digest              hashWriter
	evidence            evidence
	nextLine            []byte
	ordinal             uint64
	lastGroup           time.Time
	started             bool
	sealed              bool
	validatedArtifactID string
	expectedRecords     uint64
	recordCounts        map[string]int64
	priorRecord         *record
	validatedSize       int64
	validatedModTime    time.Time
}

type hashWriter struct {
	value hash.Hash
	count int64
}

func (h *hashWriter) Write(value []byte) { _, _ = h.value.Write(value); h.count += int64(len(value)) }
func (h *hashWriter) Sum() string {
	return "sha256:" + hex.EncodeToString(h.value.Sum(nil))
}

// New validates the complete same-open file once without retaining records,
// rewinds it, and returns a streaming second-pass cursor.
func New(file *os.File, plan Plan) (*Cursor, error) {
	if file == nil || !validPlan(plan) {
		return nil, classified(ErrorArtifactValidation, "invalid playback plan")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.New("rewind playback artifact")
	}
	first := newCursor(file, plan)
	if _, err := first.Start(); err != nil {
		return nil, err
	}
	for group := plan.Start; !group.After(plan.End); group = group.Add(time.Second) {
		for {
			record, ok, err := first.NextRecord(group)
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
			_ = record
		}
		if _, err := first.FinishGroup(group); err != nil {
			return nil, err
		}
	}
	endEvidence, err := first.End()
	if err != nil {
		return nil, err
	}
	if endEvidence.ArtifactID() != plan.ArtifactID {
		return nil, classified(ErrorArtifactValidation, "validated artifact identity changed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.New("rewind validated playback artifact")
	}
	second := newCursor(file, plan)
	second.validatedArtifactID = plan.ArtifactID
	second.expectedRecords = endEvidence.TotalRecords()
	info, err := file.Stat()
	if err != nil {
		return nil, errors.New("stat validated playback artifact")
	}
	second.validatedSize, second.validatedModTime = info.Size(), info.ModTime()
	return second, nil
}

func newCursor(file *os.File, plan Plan) *Cursor {
	return &Cursor{file: file, plan: plan, reader: bufio.NewReader(file), digest: hashWriter{value: sha256.New()}, recordCounts: make(map[string]int64)}
}

func validPlan(plan Plan) bool {
	if !validArtifactID(plan.ArtifactID) || plan.BindingID == "" || plan.UniverseID == "" || plan.TradingDate == "" ||
		(plan.Mode != CompleteFinalBars && plan.Mode != PartialSynthetic) || plan.MaximumBytes <= 0 || plan.MaximumRecords <= 0 ||
		!whole(plan.SessionStart) || !whole(plan.SessionEnd) || !whole(plan.Start) || !whole(plan.End) ||
		!plan.SessionStart.Before(plan.SessionEnd) || plan.Start.Before(plan.SessionStart) || !plan.Start.Before(plan.End) || plan.End.After(plan.SessionEnd) || len(plan.Symbols) == 0 {
		return false
	}
	prior := ""
	for _, symbol := range plan.Symbols {
		if symbol == "" || symbol <= prior {
			return false
		}
		prior = symbol
	}
	return true
}

func validArtifactID(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func (c *Cursor) Start() (StartEvidence, error) {
	if c == nil || c.started {
		return StartEvidence{}, classified(ErrorOrdinalGroup, "playback start is unavailable")
	}
	info, err := c.file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > c.plan.MaximumBytes {
		return StartEvidence{}, classified(ErrorArtifactValidation, "artifact size or type changed")
	}
	line, err := c.readLine()
	if err != nil {
		return StartEvidence{}, err
	}
	var header headerLine
	if strictLine(line, &header) != nil {
		return StartEvidence{}, classified(ErrorSchemaCanonical, "playback header is noncanonical")
	}
	if !validHeader(header, c.plan) {
		return StartEvidence{}, classified(ErrorBindingCoverage, "playback header is incompatible with binding")
	}
	c.digest.Write(line)
	c.started = true
	c.evidence = evidence{valid: true, complete: c.plan.Mode == CompleteFinalBars, artifactID: c.validatedArtifactID, bindingID: c.plan.BindingID, start: c.plan.Start, end: c.plan.End, totalRecords: c.expectedRecords}
	return StartEvidence{evidence: c.evidence}, nil
}

func (c *Cursor) NextRecord(group time.Time) (RecordEvidence, bool, error) {
	if c == nil || !c.started || c.sealed || !whole(group) || group.Before(c.plan.Start) || group.After(c.plan.End) || (!c.lastGroup.IsZero() && !group.Equal(c.lastGroup.Add(time.Second))) {
		return RecordEvidence{}, false, classified(ErrorOrdinalGroup, "invalid playback group")
	}
	if c.nextLine == nil {
		line, err := c.readLine()
		if err != nil {
			return RecordEvidence{}, false, err
		}
		c.nextLine = line
	}
	kind, err := kindOf(c.nextLine)
	if err != nil {
		return RecordEvidence{}, false, classified(ErrorSchemaCanonical, "playback line kind is invalid")
	}
	if kind != "aggregate" {
		return RecordEvidence{}, false, nil
	}
	var value aggregateLine
	if strictAggregate(c.nextLine, &value) != nil {
		return RecordEvidence{}, false, classified(ErrorSchemaCanonical, "aggregate line changed or is noncanonical")
	}
	record, err := recordFrom(value)
	if err != nil || !validRecord(c.plan, record) {
		return RecordEvidence{}, false, classified(ErrorSchemaCanonical, "playback aggregate is invalid")
	}
	if record.ordinal != c.ordinal+1 {
		return RecordEvidence{}, false, classified(ErrorOrdinalGroup, "playback aggregate ordinal is invalid")
	}
	if c.priorRecord != nil {
		comparison := compareRecord(*c.priorRecord, record)
		if comparison > 0 || c.plan.Mode == CompleteFinalBars && c.priorRecord.symbol == record.symbol && c.priorRecord.windowStart == record.windowStart {
			return RecordEvidence{}, false, classified(ErrorOrdinalGroup, "playback aggregate order or identity is invalid")
		}
	}
	if record.logicalTime.Before(group) {
		return RecordEvidence{}, false, classified(ErrorOrdinalGroup, "playback aggregate group regressed")
	}
	if record.logicalTime.After(group) {
		return RecordEvidence{}, false, nil
	}
	c.ordinal++
	copyRecord := record
	c.priorRecord = &copyRecord
	c.recordCounts[record.symbol]++
	c.digest.Write(c.nextLine)
	c.nextLine = nil
	e := c.evidence
	e.totalRecords = c.ordinal
	return RecordEvidence{evidence: e, record: record}, true, nil
}

func (c *Cursor) FinishGroup(group time.Time) (GroupEvidence, error) {
	if c == nil || !c.started || c.sealed || !whole(group) || group.Before(c.plan.Start) || group.After(c.plan.End) || (!c.lastGroup.IsZero() && !group.Equal(c.lastGroup.Add(time.Second))) {
		return GroupEvidence{}, classified(ErrorOrdinalGroup, "invalid completed playback group")
	}
	if c.nextLine == nil {
		line, err := c.readLine()
		if err != nil {
			return GroupEvidence{}, err
		}
		c.nextLine = line
	}
	if kind, err := kindOf(c.nextLine); err != nil {
		return GroupEvidence{}, classified(ErrorSchemaCanonical, "playback line kind is invalid")
	} else if kind == "aggregate" {
		var value aggregateLine
		if strictAggregate(c.nextLine, &value) != nil {
			return GroupEvidence{}, classified(ErrorSchemaCanonical, "aggregate line changed or is noncanonical")
		}
		record, parseErr := recordFrom(value)
		if parseErr != nil || !record.logicalTime.After(group) {
			return GroupEvidence{}, classified(ErrorOrdinalGroup, "playback group was not fully consumed")
		}
	}
	c.lastGroup = group
	e := c.evidence
	e.totalRecords = c.ordinal
	return GroupEvidence{evidence: e, logicalTime: group, lastOrdinal: c.ordinal}, nil
}

func (c *Cursor) End() (EndEvidence, error) {
	if c == nil || !c.started || c.sealed || c.lastGroup != c.plan.End {
		return EndEvidence{}, classified(ErrorArtifactEnd, "playback end is unavailable")
	}
	counts := make(map[string]int64)
	covered := make(map[string]struct{})
	var coverageCount, emptyCount, bodyBytes int64
	// Recompute body bytes from the digest input accumulated through records.
	bodyBytes = c.digest.count
	prior := ""
	for {
		line := c.nextLine
		if line == nil {
			var err error
			line, err = c.readLine()
			if err != nil {
				return EndEvidence{}, err
			}
		}
		c.nextLine = nil
		kind, err := kindOf(line)
		if err != nil {
			return EndEvidence{}, classified(ErrorSchemaCanonical, "playback line kind is invalid")
		}
		switch kind {
		case "coverage":
			var value coverageLine
			if strictLine(line, &value) != nil || value.Symbol <= prior || value.Start != canonicalTime(c.plan.Start) || value.End != canonicalTime(c.plan.End) || value.RecordCount < 0 || value.RecordCount != c.recordCounts[value.Symbol] {
				return EndEvidence{}, classified(ErrorBindingCoverage, "playback coverage is invalid")
			}
			if c.plan.Mode == CompleteFinalBars && value.Class != completeCoverage || c.plan.Mode == PartialSynthetic && value.Class != partialCoverage {
				return EndEvidence{}, classified(ErrorBindingCoverage, "playback coverage class is invalid")
			}
			if _, ok := symbolIndex(c.plan.Symbols, value.Symbol); !ok {
				return EndEvidence{}, classified(ErrorBindingCoverage, "playback coverage symbol is outside binding")
			}
			prior = value.Symbol
			covered[value.Symbol] = struct{}{}
			counts[value.Symbol] = value.RecordCount
			coverageCount++
			if c.plan.Mode == CompleteFinalBars && value.RecordCount == 0 {
				emptyCount++
			}
			c.digest.Write(line)
			bodyBytes += int64(len(line))
		case "summary":
			var value summaryLine
			if strictLine(line, &value) != nil || value.AggregateRecords != int64(c.ordinal) || value.CoverageEntries != coverageCount || value.EmptySymbols != emptyCount || value.BodyBytes != bodyBytes {
				return EndEvidence{}, classified(ErrorArtifactEnd, "playback summary does not reconcile")
			}
			if c.plan.Mode == CompleteFinalBars && len(covered) != len(c.plan.Symbols) || c.plan.Mode == PartialSynthetic && (coverageCount == 0 || coverageCount > int64(len(c.plan.Symbols)) || emptyCount != 0) {
				return EndEvidence{}, classified(ErrorBindingCoverage, "playback coverage population is invalid")
			}
			for _, symbol := range c.plan.Symbols {
				if c.plan.Mode == CompleteFinalBars {
					if _, ok := covered[symbol]; !ok {
						return EndEvidence{}, classified(ErrorBindingCoverage, "complete playback coverage is incomplete")
					}
				}
			}
			c.digest.Write(line)
			seal, readErr := c.readLine()
			if readErr != nil {
				return EndEvidence{}, readErr
			}
			var sealValue sealLine
			if strictLine(seal, &sealValue) != nil || sealValue.SealedBytes != c.digest.count || sealValue.ArtifactID != c.digest.Sum() {
				return EndEvidence{}, classified(ErrorArtifactEnd, "playback seal or digest is invalid")
			}
			if extra, readErr := c.reader.ReadByte(); readErr != io.EOF || extra != 0 {
				return EndEvidence{}, classified(ErrorArtifactEnd, "playback seal is not final")
			}
			if c.validatedSize != 0 {
				info, statErr := c.file.Stat()
				if statErr != nil || info.Size() != c.validatedSize || !info.ModTime().Equal(c.validatedModTime) {
					return EndEvidence{}, classified(ErrorArtifactEnd, "playback artifact changed during second pass")
				}
			}
			c.sealed = true
			c.evidence.artifactID = sealValue.ArtifactID
			c.evidence.totalRecords = c.ordinal
			return EndEvidence{evidence: c.evidence, ordinal: c.ordinal}, nil
		default:
			return EndEvidence{}, classified(ErrorSchemaCanonical, "playback artifact phase is invalid")
		}
	}
}

func (c *Cursor) readLine() ([]byte, error) {
	line, err := c.reader.ReadBytes('\n')
	if err != nil || len(line) == 0 || line[len(line)-1] != '\n' {
		return nil, classified(ErrorArtifactEnd, "playback artifact is truncated")
	}
	return line, nil
}

type headerLine struct{ Kind, Schema, ArtifactMode, BindingID, UniverseID, TradingDate, SessionStart, SessionEnd, ReplayStart, ReplayEnd, Provider, Endpoint, NormalizationPolicy, CompileFormat string }

func (h *headerLine) UnmarshalJSON(data []byte) error {
	type wire struct {
		Kind                string `json:"kind"`
		Schema              string `json:"schema"`
		ArtifactMode        string `json:"artifact_mode"`
		BindingID           string `json:"binding_id"`
		UniverseID          string `json:"universe_id"`
		TradingDate         string `json:"trading_date"`
		SessionStart        string `json:"session_start"`
		SessionEnd          string `json:"session_end"`
		ReplayStart         string `json:"replay_start"`
		ReplayEnd           string `json:"replay_end"`
		Provider            string `json:"provider"`
		Endpoint            string `json:"endpoint"`
		NormalizationPolicy string `json:"normalization_policy"`
		CompileFormat       string `json:"compile_format"`
	}
	var v wire
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*h = headerLine{v.Kind, v.Schema, v.ArtifactMode, v.BindingID, v.UniverseID, v.TradingDate, v.SessionStart, v.SessionEnd, v.ReplayStart, v.ReplayEnd, v.Provider, v.Endpoint, v.NormalizationPolicy, v.CompileFormat}
	return nil
}
func (h headerLine) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind                string `json:"kind"`
		Schema              string `json:"schema"`
		ArtifactMode        string `json:"artifact_mode"`
		BindingID           string `json:"binding_id"`
		UniverseID          string `json:"universe_id"`
		TradingDate         string `json:"trading_date"`
		SessionStart        string `json:"session_start"`
		SessionEnd          string `json:"session_end"`
		ReplayStart         string `json:"replay_start"`
		ReplayEnd           string `json:"replay_end"`
		Provider            string `json:"provider"`
		Endpoint            string `json:"endpoint"`
		NormalizationPolicy string `json:"normalization_policy"`
		CompileFormat       string `json:"compile_format"`
	}{h.Kind, h.Schema, h.ArtifactMode, h.BindingID, h.UniverseID, h.TradingDate, h.SessionStart, h.SessionEnd, h.ReplayStart, h.ReplayEnd, h.Provider, h.Endpoint, h.NormalizationPolicy, h.CompileFormat})
}

type aggregateLine struct {
	Kind                string  `json:"kind"`
	Ordinal             int64   `json:"ordinal"`
	LogicalDeliveryTime string  `json:"logical_delivery_time"`
	Symbol              string  `json:"symbol"`
	WindowStart         string  `json:"window_start"`
	WindowEnd           string  `json:"window_end"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	VWAP                float64 `json:"vwap"`
	AverageTradeSize    int64   `json:"average_trade_size"`
	ATSProvenance       string  `json:"ats_provenance"`
}
type coverageLine struct {
	Kind        string `json:"kind"`
	Symbol      string `json:"symbol"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Class       string `json:"class"`
	RecordCount int64  `json:"record_count"`
}
type summaryLine struct {
	Kind             string `json:"kind"`
	AggregateRecords int64  `json:"aggregate_records"`
	CoverageEntries  int64  `json:"coverage_entries"`
	EmptySymbols     int64  `json:"empty_symbols"`
	BodyBytes        int64  `json:"body_bytes"`
}
type sealLine struct {
	Kind        string `json:"kind"`
	ArtifactID  string `json:"artifact_id"`
	SealedBytes int64  `json:"sealed_bytes"`
}
type record struct {
	ordinal                uint64
	logicalTime            time.Time
	symbol                 string
	windowStart, windowEnd time.Time
	values                 Values
}

func validHeader(h headerLine, p Plan) bool {
	ss, e1 := parseTime(h.SessionStart)
	se, e2 := parseTime(h.SessionEnd)
	rs, e3 := parseTime(h.ReplayStart)
	re, e4 := parseTime(h.ReplayEnd)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || h.Kind != "header" || h.Schema != SchemaV1 || h.CompileFormat != SchemaV1 || h.ArtifactMode != p.Mode || h.BindingID != p.BindingID || h.UniverseID != p.UniverseID || h.TradingDate != p.TradingDate || ss != p.SessionStart || se != p.SessionEnd || rs != p.Start || re != p.End {
		return false
	}
	return p.Mode == CompleteFinalBars && h.Provider == massiveProvider && h.Endpoint == massiveEndpoint && h.NormalizationPolicy == massivePolicy || p.Mode == PartialSynthetic && h.Provider == syntheticProvider && h.Endpoint == "" && h.NormalizationPolicy == syntheticPolicy
}
func recordFrom(v aggregateLine) (record, error) {
	lt, e1 := parseTime(v.LogicalDeliveryTime)
	ws, e2 := parseTime(v.WindowStart)
	we, e3 := parseTime(v.WindowEnd)
	if e1 != nil || e2 != nil || e3 != nil || v.Ordinal <= 0 {
		return record{}, errors.New("invalid record")
	}
	return record{uint64(v.Ordinal), lt, v.Symbol, ws, we, Values{v.Open, v.High, v.Low, v.Close, v.Volume, v.VWAP, v.AverageTradeSize, v.ATSProvenance}}, nil
}
func validRecord(p Plan, r record) bool {
	v := r.values
	finiteP := func(x float64) bool { return x > 0 && !math.IsNaN(x) && !math.IsInf(x, 0) }
	finiteN := func(x float64) bool { return x >= 0 && !math.IsNaN(x) && !math.IsInf(x, 0) }
	_, member := symbolIndex(p.Symbols, r.symbol)
	base := whole(r.logicalTime) && whole(r.windowStart) && whole(r.windowEnd) && r.windowEnd.Sub(r.windowStart) == time.Second && !r.windowStart.Before(p.Start) && r.windowStart.Before(p.End) && !r.windowEnd.After(p.End) && member && finiteP(v.Open) && finiteP(v.High) && finiteP(v.Low) && finiteP(v.Close) && finiteN(v.Volume) && finiteP(v.VWAP) && v.AverageTradeSize >= 0 && v.Low <= v.Open && v.Open <= v.High && v.Low <= v.Close && v.Close <= v.High
	if !base {
		return false
	}
	if p.Mode == CompleteFinalBars {
		return r.logicalTime == r.windowEnd && v.ATSProvenance == "rest_floor_volume_over_transactions"
	}
	return !r.logicalTime.Before(r.windowEnd) && (v.ATSProvenance == "rest_floor_volume_over_transactions" || v.ATSProvenance == "live_provider_average")
}
func compareRecord(a, b record) int {
	if a.logicalTime.Before(b.logicalTime) {
		return -1
	}
	if a.logicalTime.After(b.logicalTime) {
		return 1
	}
	if a.windowStart.Before(b.windowStart) {
		return -1
	}
	if a.windowStart.After(b.windowStart) {
		return 1
	}
	return strings.Compare(a.symbol, b.symbol)
}
func symbolIndex(symbols []string, s string) (int, bool) {
	lo, hi := 0, len(symbols)
	for lo < hi {
		m := (lo + hi) / 2
		if symbols[m] < s {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo, lo < len(symbols) && symbols[lo] == s
}
func whole(t time.Time) bool           { return !t.IsZero() && t == t.UTC() && t.Nanosecond() == 0 }
func canonicalTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func parseTime(s string) (time.Time, error) {
	if !strings.HasSuffix(s, "Z") {
		return time.Time{}, errors.New("not UTC")
	}
	t, e := time.Parse(time.RFC3339Nano, s)
	if e != nil || canonicalTime(t) != s {
		return time.Time{}, errors.New("noncanonical time")
	}
	return t, nil
}
func kindOf(line []byte) (string, error) {
	var v struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(line, &v) != nil || v.Kind == "" {
		return "", errors.New("line kind")
	}
	return v.Kind, nil
}
func strictLine(line []byte, dst any) error {
	if len(line) == 0 || line[len(line)-1] != '\n' {
		return errors.New("line")
	}
	dec := json.NewDecoder(bytes.NewReader(line[:len(line)-1]))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var x json.RawMessage
	if dec.Decode(&x) != io.EOF {
		return errors.New("trailing")
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(dst); err != nil {
		return err
	}
	if !bytes.Equal(b.Bytes(), line) {
		return errors.New("noncanonical")
	}
	return nil
}
func strictAggregate(line []byte, dst *aggregateLine) error {
	if len(line) == 0 || line[len(line)-1] != '\n' {
		return errors.New("line")
	}
	dec := json.NewDecoder(bytes.NewReader(line[:len(line)-1]))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var x json.RawMessage
	if dec.Decode(&x) != io.EOF {
		return errors.New("trailing")
	}
	var b []byte
	b = append(b, `{"kind":"aggregate","ordinal":`...)
	b = strconv.AppendInt(b, dst.Ordinal, 10)
	for _, f := range []struct{ name, value string }{{"logical_delivery_time", dst.LogicalDeliveryTime}, {"symbol", dst.Symbol}, {"window_start", dst.WindowStart}, {"window_end", dst.WindowEnd}} {
		b = append(b, ',')
		b = appendJSONString(b, f.name)
		b = append(b, ':')
		b = appendJSONString(b, f.value)
	}
	for _, f := range []struct {
		name  string
		value float64
	}{{"open", dst.Open}, {"high", dst.High}, {"low", dst.Low}, {"close", dst.Close}, {"volume", dst.Volume}, {"vwap", dst.VWAP}} {
		b = append(b, ',')
		b = appendJSONString(b, f.name)
		b = append(b, ':')
		if f.value == 0 {
			f.value = 0
		}
		b = strconv.AppendFloat(b, f.value, 'g', -1, 64)
	}
	b = append(b, `,"average_trade_size":`...)
	b = strconv.AppendInt(b, dst.AverageTradeSize, 10)
	b = append(b, `,"ats_provenance":`...)
	b = appendJSONString(b, dst.ATSProvenance)
	b = append(b, '}', '\n')
	if !bytes.Equal(b, line) {
		return errors.New("noncanonical aggregate")
	}
	return nil
}
func appendJSONString(dst []byte, value string) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
	encoded := b.Bytes()
	return append(dst, encoded[:len(encoded)-1]...)
}
