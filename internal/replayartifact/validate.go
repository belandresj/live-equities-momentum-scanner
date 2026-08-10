package replayartifact

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact/playback"
)

type ValidationPlan struct {
	Binding        reference.Binding
	Start, End     time.Time
	ExpectedMode   ArtifactMode
	MaximumBytes   int64
	MaximumRecords int64
}

type Handle struct {
	file     *os.File
	plan     ValidationPlan
	metadata Metadata
}

func OpenValidated(path string, plan ValidationPlan) (*Handle, error) {
	return OpenValidatedContext(context.Background(), path, plan)
}

func OpenValidatedContext(ctx context.Context, path string, plan ValidationPlan) (*Handle, error) {
	if ctx == nil {
		return nil, errors.New("artifact validation requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validValidationPlan(plan) {
		return nil, errors.New("invalid artifact validation plan")
	}
	file, err := openRegularReadOnly(path)
	if err != nil {
		return nil, errors.New("open artifact")
	}
	metadata, err := validateOpenFile(ctx, file, plan)
	if err != nil {
		file.Close()
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, errors.New("rewind validated artifact")
	}
	if err := ctx.Err(); err != nil {
		file.Close()
		return nil, err
	}
	return &Handle{file: file, plan: plan, metadata: metadata}, nil
}

// CandidateHeader is deliberately untrusted. It contains only the canonical
// first-line fields needed to select candidate reference data; it proves no
// artifact identity, coverage, seal, or playback success.
type CandidateHeader struct {
	Schema, CompileFormat             string
	Mode                              ArtifactMode
	BindingIdentity, UniverseIdentity string
	TradingDate                       string
	SessionStart, SessionEnd          time.Time
	ReplayStart, ReplayEnd            time.Time
}

func ProbeCandidateHeader(ctx context.Context, path string, maximumBytes int64) (CandidateHeader, error) {
	if ctx == nil || maximumBytes <= 0 {
		return CandidateHeader{}, errors.New("invalid artifact candidate probe")
	}
	if err := ctx.Err(); err != nil {
		return CandidateHeader{}, err
	}
	file, err := openRegularReadOnly(path)
	if err != nil {
		return CandidateHeader{}, errors.New("open artifact candidate")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > maximumBytes {
		return CandidateHeader{}, errors.New("artifact candidate size is invalid")
	}
	reader := bufio.NewReader(io.LimitReader(file, maximumBytes))
	if err := ctx.Err(); err != nil {
		return CandidateHeader{}, err
	}
	line, err := reader.ReadBytes('\n')
	if err != nil || len(line) == 0 || line[len(line)-1] != '\n' {
		return CandidateHeader{}, errors.New("artifact candidate header is truncated")
	}
	if err := ctx.Err(); err != nil {
		return CandidateHeader{}, err
	}
	var header headerLine
	if strictCanonicalLine(line, &header) != nil || header.Kind != "header" {
		return CandidateHeader{}, errors.New("invalid artifact candidate header")
	}
	sessionStart, err1 := parseCanonicalTime(header.SessionStart)
	sessionEnd, err2 := parseCanonicalTime(header.SessionEnd)
	replayStart, err3 := parseCanonicalTime(header.ReplayStart)
	replayEnd, err4 := parseCanonicalTime(header.ReplayEnd)
	mode := ArtifactMode(header.ArtifactMode)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || header.Schema == "" || header.CompileFormat == "" ||
		(mode != CompleteFinalBars && mode != PartialSynthetic) || header.BindingID == "" || header.UniverseID == "" || header.TradingDate == "" ||
		!sessionStart.Before(sessionEnd) || replayStart.Before(sessionStart) || !replayStart.Before(replayEnd) || replayEnd.After(sessionEnd) {
		return CandidateHeader{}, errors.New("artifact candidate header fields are invalid")
	}
	if err := ctx.Err(); err != nil {
		return CandidateHeader{}, err
	}
	return CandidateHeader{Schema: header.Schema, CompileFormat: header.CompileFormat, Mode: mode, BindingIdentity: header.BindingID,
		UniverseIdentity: header.UniverseID, TradingDate: header.TradingDate, SessionStart: sessionStart, SessionEnd: sessionEnd,
		ReplayStart: replayStart, ReplayEnd: replayEnd}, nil
}

func openRegularReadOnly(path string) (*os.File, error) {
	descriptor, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		_ = syscall.Close(descriptor)
		return nil, errors.New("create artifact file handle")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("artifact is not a regular file")
	}
	if err := syscall.SetNonblock(descriptor, false); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (h *Handle) Metadata() Metadata { return h.metadata }

// BeginPlayback validates and rewinds the same already-open file description,
// then returns the only producer of opaque replay success evidence.
func (h *Handle) BeginPlayback() (*playback.Cursor, error) {
	return h.BeginPlaybackContext(context.Background())
}

func (h *Handle) BeginPlaybackContext(ctx context.Context) (*playback.Cursor, error) {
	if h == nil {
		return nil, os.ErrClosed
	}
	return h.BeginPlaybackThroughContext(ctx, h.metadata.ReplayEnd)
}

// BeginPlaybackThrough keeps the artifact's validated [S,R) identity intact
// while selecting an exact engine application boundary in (S,R].
func (h *Handle) BeginPlaybackThrough(requestedEnd time.Time) (*playback.Cursor, error) {
	return h.BeginPlaybackThroughContext(context.Background(), requestedEnd)
}

func (h *Handle) BeginPlaybackThroughContext(ctx context.Context, requestedEnd time.Time) (*playback.Cursor, error) {
	if h == nil || h.file == nil {
		return nil, os.ErrClosed
	}
	if requestedEnd.IsZero() || requestedEnd != requestedEnd.UTC() || requestedEnd.Nanosecond() != 0 ||
		!h.metadata.ReplayStart.Before(requestedEnd) || requestedEnd.After(h.metadata.ReplayEnd) ||
		(requestedEnd.Before(h.metadata.ReplayEnd) && h.metadata.Mode != CompleteFinalBars) {
		return nil, errors.New("invalid requested replay end")
	}
	mode := string(h.plan.ExpectedMode)
	return playback.NewContext(ctx, h.file, playback.Plan{
		ArtifactID: h.metadata.ArtifactID,
		BindingID:  h.plan.Binding.Identity(), UniverseID: h.plan.Binding.UniverseIdentity(), TradingDate: h.plan.Binding.TradingDate(),
		SessionStart: h.plan.Binding.SessionStart(), SessionEnd: h.plan.Binding.SessionEnd(), Start: h.plan.Start, End: h.plan.End, RequestedEnd: requestedEnd,
		Mode: mode, Symbols: h.plan.Binding.UniverseSymbols(), MaximumBytes: h.plan.MaximumBytes, MaximumRecords: h.plan.MaximumRecords,
	})
}
func (h *Handle) Read(destination []byte) (int, error) {
	if h == nil || h.file == nil {
		return 0, os.ErrClosed
	}
	return h.file.Read(destination)
}
func (h *Handle) ValidateAgain() error {
	return h.ValidateAgainContext(context.Background())
}
func (h *Handle) ValidateAgainContext(ctx context.Context) error {
	if h == nil || h.file == nil {
		return os.ErrClosed
	}
	if _, err := h.file.Seek(0, io.SeekStart); err != nil {
		return errors.New("rewind artifact for validation")
	}
	metadata, err := validateOpenFile(ctx, h.file, h.plan)
	if err != nil {
		return err
	}
	if metadata != h.metadata {
		return errors.New("artifact metadata changed between validations")
	}
	_, err = h.file.Seek(0, io.SeekStart)
	if err == nil {
		err = ctx.Err()
	}
	return err
}
func (h *Handle) Close() error {
	if h == nil || h.file == nil {
		return nil
	}
	err := h.file.Close()
	h.file = nil
	return err
}

func validValidationPlan(plan ValidationPlan) bool {
	return plan.Binding.Identity() != "" && validReplayInterval(plan.Binding, plan.Start, plan.End) &&
		(plan.ExpectedMode == CompleteFinalBars || plan.ExpectedMode == PartialSynthetic) && plan.MaximumBytes > 0 && plan.MaximumRecords > 0
}

func validateOpenFile(ctx context.Context, file *os.File, plan ValidationPlan) (Metadata, error) {
	if ctx == nil {
		return Metadata{}, errors.New("artifact validation requires a context")
	}
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > plan.MaximumBytes {
		return Metadata{}, errors.New("artifact size or file type is invalid")
	}
	reader := bufio.NewReader(file)
	readLine := func() ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, readErr := reader.ReadBytes('\n')
		if readErr != nil || len(line) == 0 || line[len(line)-1] != '\n' {
			return nil, errors.New("artifact is truncated")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return line, nil
	}
	headerBytes, err := readLine()
	if err != nil {
		return Metadata{}, err
	}
	var header headerLine
	if err := strictCanonicalLine(headerBytes, &header); err != nil || header.Kind != "header" {
		return Metadata{}, errors.New("invalid artifact header")
	}
	context, err := validateHeader(header, plan)
	if err != nil {
		return Metadata{}, err
	}
	digest := sha256.New()
	_, _ = digest.Write(headerBytes)
	sealedBytes := int64(len(headerBytes))
	bodyBytes := sealedBytes

	expectedSymbols := plan.Binding.UniverseSymbols()
	member := make(map[string]struct{}, len(expectedSymbols))
	for _, symbol := range expectedSymbols {
		member[symbol] = struct{}{}
	}
	counts := make(map[string]int64)
	covered := make(map[string]struct{})
	var aggregateCount int64
	var coverageCount int64
	var emptyCount int64
	var priorRecord *canonicalRecord
	var priorCoverage string
	phase := "aggregate"
	var summary summaryLine
	for {
		line, readErr := readLine()
		if readErr != nil {
			return Metadata{}, readErr
		}
		kind, kindErr := lineKind(line)
		if kindErr != nil {
			return Metadata{}, kindErr
		}
		switch kind {
		case "aggregate":
			if phase != "aggregate" || aggregateCount >= plan.MaximumRecords {
				return Metadata{}, errors.New("aggregate line is out of order or over budget")
			}
			var value aggregateLine
			if err := strictCanonicalAggregateLine(line, &value); err != nil || value.Ordinal != aggregateCount+1 {
				return Metadata{}, errors.New("invalid canonical aggregate line")
			}
			record, err := recordFromLine(value)
			if err != nil || !validRecord(context, record) {
				return Metadata{}, errors.New("invalid aggregate record")
			}
			if _, ok := member[record.symbol]; !ok {
				return Metadata{}, errors.New("aggregate symbol is outside binding")
			}
			if priorRecord != nil {
				comparison := compareRecords(*priorRecord, record)
				if comparison > 0 || plan.ExpectedMode == CompleteFinalBars && priorRecord.symbol == record.symbol && priorRecord.windowStart == record.windowStart {
					return Metadata{}, errors.New("aggregate order or identity is invalid")
				}
			}
			copy := record
			priorRecord = &copy
			counts[record.symbol]++
			aggregateCount++
			_, _ = digest.Write(line)
			sealedBytes += int64(len(line))
			bodyBytes += int64(len(line))
		case "coverage":
			if phase == "summary" || phase == "seal" {
				return Metadata{}, errors.New("coverage line is out of order")
			}
			phase = "coverage"
			var value coverageLine
			if err := strictCanonicalLine(line, &value); err != nil || value.Symbol <= priorCoverage || value.RecordCount < 0 ||
				value.Start != canonicalTime(plan.Start) || value.End != canonicalTime(plan.End) || value.RecordCount != counts[value.Symbol] {
				return Metadata{}, errors.New("invalid coverage line")
			}
			if _, ok := member[value.Symbol]; !ok {
				return Metadata{}, errors.New("coverage symbol is outside binding")
			}
			if plan.ExpectedMode == CompleteFinalBars && value.Class != CompleteCoverage || plan.ExpectedMode == PartialSynthetic && value.Class != PartialCoverage {
				return Metadata{}, errors.New("coverage class does not match artifact mode")
			}
			priorCoverage = value.Symbol
			covered[value.Symbol] = struct{}{}
			coverageCount++
			if plan.ExpectedMode == CompleteFinalBars && value.RecordCount == 0 {
				emptyCount++
			}
			_, _ = digest.Write(line)
			sealedBytes += int64(len(line))
			bodyBytes += int64(len(line))
		case "summary":
			if phase != "coverage" {
				return Metadata{}, errors.New("summary is out of order")
			}
			phase = "summary"
			if err := strictCanonicalLine(line, &summary); err != nil || summary.AggregateRecords != aggregateCount ||
				summary.CoverageEntries != coverageCount || summary.EmptySymbols != emptyCount || summary.BodyBytes != bodyBytes {
				return Metadata{}, errors.New("artifact summary does not reconcile")
			}
			if plan.ExpectedMode == PartialSynthetic && summary.EmptySymbols != 0 {
				return Metadata{}, errors.New("partial artifact cannot claim empty symbols")
			}
			if err := validateCoveragePopulation(plan, expectedSymbols, counts, covered, coverageCount, priorCoverage); err != nil {
				return Metadata{}, err
			}
			_, _ = digest.Write(line)
			sealedBytes += int64(len(line))
		case "seal":
			if phase != "summary" {
				return Metadata{}, errors.New("seal is out of order")
			}
			phase = "seal"
			var seal sealLine
			if err := strictCanonicalLine(line, &seal); err != nil || seal.SealedBytes != sealedBytes || seal.ArtifactID != "sha256:"+hex.EncodeToString(digest.Sum(nil)) {
				return Metadata{}, errors.New("artifact seal or digest is invalid")
			}
			if extra, readErr := reader.ReadByte(); readErr != io.EOF || extra != 0 {
				return Metadata{}, errors.New("seal is not the final artifact line")
			}
			if err := ctx.Err(); err != nil {
				return Metadata{}, err
			}
			return Metadata{plan.ExpectedMode, seal.ArtifactID, header.BindingID, header.UniverseID, header.TradingDate,
				context.sessionStart, context.sessionEnd, context.replayStart, context.replayEnd,
				aggregateCount, coverageCount, emptyCount, sealedBytes}, nil
		default:
			return Metadata{}, errors.New("unknown artifact line kind")
		}
	}
}

func strictCanonicalLine(line []byte, destination any) error {
	if len(line) == 0 || line[len(line)-1] != '\n' {
		return errors.New("canonical line lacks LF")
	}
	decoder := json.NewDecoder(bytes.NewReader(line[:len(line)-1]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("canonical line has trailing JSON")
	}
	reencoded, err := appendCanonicalLine(nil, destination, int64(len(line)))
	if err != nil || !bytes.Equal(reencoded, line) {
		return errors.New("line is not canonical")
	}
	return nil
}

func strictCanonicalAggregateLine(line []byte, destination *aggregateLine) error {
	if err := strictCanonicalLine(line, destination); err != nil {
		return err
	}
	normalized := *destination
	normalized.Open = normalizeZero(normalized.Open)
	normalized.High = normalizeZero(normalized.High)
	normalized.Low = normalizeZero(normalized.Low)
	normalized.Close = normalizeZero(normalized.Close)
	normalized.Volume = normalizeZero(normalized.Volume)
	normalized.VWAP = normalizeZero(normalized.VWAP)
	reencoded, err := appendCanonicalLine(nil, normalized, int64(len(line)))
	if err != nil || !bytes.Equal(reencoded, line) {
		return errors.New("aggregate numeric encoding is not canonical")
	}
	return nil
}

func lineKind(line []byte) (string, error) {
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(line, &probe); err != nil || probe.Kind == "" {
		return "", errors.New("artifact line has no kind")
	}
	return probe.Kind, nil
}

func validateHeader(header headerLine, plan ValidationPlan) (artifactContext, error) {
	sessionStart, err1 := parseCanonicalTime(header.SessionStart)
	sessionEnd, err2 := parseCanonicalTime(header.SessionEnd)
	replayStart, err3 := parseCanonicalTime(header.ReplayStart)
	replayEnd, err4 := parseCanonicalTime(header.ReplayEnd)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || header.Schema != SchemaV1 || header.CompileFormat != SchemaV1 ||
		header.ArtifactMode != string(plan.ExpectedMode) || header.BindingID != plan.Binding.Identity() || header.UniverseID != plan.Binding.UniverseIdentity() ||
		header.TradingDate != plan.Binding.TradingDate() || sessionStart != plan.Binding.SessionStart() || sessionEnd != plan.Binding.SessionEnd() ||
		replayStart != plan.Start || replayEnd != plan.End {
		return artifactContext{}, errors.New("artifact header is incompatible with binding")
	}
	if plan.ExpectedMode == CompleteFinalBars {
		if header.Provider != MassiveProvider || header.Endpoint != MassiveEndpoint || header.NormalizationPolicy != MassivePolicy {
			return artifactContext{}, errors.New("complete artifact provenance is invalid")
		}
	} else if header.Provider != SyntheticProvider || header.Endpoint != "" || header.NormalizationPolicy != SyntheticPolicy {
		return artifactContext{}, errors.New("partial artifact provenance is invalid")
	}
	return artifactContext{plan.ExpectedMode, header.BindingID, header.UniverseID, header.TradingDate, sessionStart, sessionEnd, replayStart, replayEnd}, nil
}

func recordFromLine(value aggregateLine) (canonicalRecord, error) {
	logical, err1 := parseCanonicalTime(value.LogicalDeliveryTime)
	start, err2 := parseCanonicalTime(value.WindowStart)
	end, err3 := parseCanonicalTime(value.WindowEnd)
	if err1 != nil || err2 != nil || err3 != nil || value.Ordinal <= 0 {
		return canonicalRecord{}, errors.New("invalid aggregate time or ordinal")
	}
	values := engine.AggregateValues{Open: value.Open, High: value.High, Low: value.Low, Close: value.Close, Volume: value.Volume, VWAP: value.VWAP,
		AverageTradeSize: value.AverageTradeSize, ATSProvenance: engine.ATSProvenance(value.ATSProvenance)}
	return canonicalRecord{logical, value.Symbol, start, end, values}, nil
}

func compareRecords(left, right canonicalRecord) int {
	if left.logicalDeliveryTime.Before(right.logicalDeliveryTime) {
		return -1
	}
	if left.logicalDeliveryTime.After(right.logicalDeliveryTime) {
		return 1
	}
	if left.windowStart.Before(right.windowStart) {
		return -1
	}
	if left.windowStart.After(right.windowStart) {
		return 1
	}
	return strings.Compare(left.symbol, right.symbol)
}

func validateCoveragePopulation(plan ValidationPlan, bindingSymbols []string, counts map[string]int64, covered map[string]struct{}, coverageCount int64, last string) error {
	for symbol := range counts {
		if _, exists := covered[symbol]; !exists {
			return errors.New("aggregate symbol lacks coverage")
		}
	}
	if plan.ExpectedMode == CompleteFinalBars {
		if coverageCount != int64(len(bindingSymbols)) || len(covered) != len(bindingSymbols) || len(counts) > len(bindingSymbols) || last != bindingSymbols[len(bindingSymbols)-1] {
			return errors.New("complete artifact coverage is incomplete")
		}
	} else if coverageCount <= 0 || coverageCount > int64(len(bindingSymbols)) {
		return errors.New("partial artifact coverage is invalid")
	}
	return nil
}
