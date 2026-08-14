package replayartifact

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"
	"time"

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
	file      *os.File
	plan      ValidationPlan
	metadata  Metadata
	validated playback.ValidatedArtifact
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
	validated, err := playback.ValidateContext(ctx, file, playbackPlan(plan, "", plan.End))
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
	metadata := metadataFromValidated(validated)
	return &Handle{file: file, plan: plan, metadata: metadata, validated: validated}, nil
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

// BeginPlayback verifies the same-open validation evidence and rewinds the file
// description without scanning it, then returns the only producer of opaque
// replay success evidence.
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
	return playback.NewContext(ctx, h.validated, playbackPlan(h.plan, h.metadata.ArtifactID, requestedEnd))
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
	validated, err := playback.ValidateContext(ctx, h.file, playbackPlan(h.plan, "", h.plan.End))
	if err != nil {
		return err
	}
	metadata := metadataFromValidated(validated)
	if metadata != h.metadata {
		return errors.New("artifact metadata changed between validations")
	}
	h.validated = validated
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

func playbackPlan(plan ValidationPlan, artifactID string, requestedEnd time.Time) playback.Plan {
	return playback.Plan{ArtifactID: artifactID,
		BindingID: plan.Binding.Identity(), UniverseID: plan.Binding.UniverseIdentity(), TradingDate: plan.Binding.TradingDate(),
		SessionStart: plan.Binding.SessionStart(), SessionEnd: plan.Binding.SessionEnd(), Start: plan.Start, End: plan.End, RequestedEnd: requestedEnd,
		Mode: string(plan.ExpectedMode), Symbols: plan.Binding.UniverseSymbols(), MaximumBytes: plan.MaximumBytes, MaximumRecords: plan.MaximumRecords}
}

func metadataFromValidated(value playback.ValidatedArtifact) Metadata {
	return Metadata{Mode: ArtifactMode(value.Mode()), ArtifactID: value.ArtifactID(), BindingIdentity: value.BindingID(), UniverseIdentity: value.UniverseID(),
		TradingDate: value.TradingDate(), SessionStart: value.SessionStart(), SessionEnd: value.SessionEnd(), ReplayStart: value.Start(), ReplayEnd: value.End(),
		AggregateRecords: int64(value.TotalRecords()), CoverageEntries: int64(value.CoverageEntries()), EmptySymbols: int64(value.EmptySymbols()), SealedBytes: value.SealedBytes()}
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
