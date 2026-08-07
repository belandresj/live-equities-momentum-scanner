// Package replay owns sequential validated-artifact playback and its one-writer
// simulated clock. It owns no canonical scanner or evaluator state.
package replay

import (
	"context"
	"errors"
	"math"
	"math/bits"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact/playback"
)

type SimulatedClock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewSimulatedClock(start time.Time) (*SimulatedClock, error) {
	if !wholeSecond(start) {
		return nil, errors.New("simulated clock requires a UTC whole second")
	}
	return &SimulatedClock{now: start}, nil
}

func (c *SimulatedClock) Now() time.Time { c.mu.RLock(); defer c.mu.RUnlock(); return c.now }
func (c *SimulatedClock) advance(value time.Time) error {
	if c == nil || !wholeSecond(value) {
		return errors.New("invalid simulated clock value")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if value.Before(c.now) {
		return errors.New("simulated clock regression")
	}
	c.now = value
	return nil
}

type Pace struct{ numerator, denominator uint64 }

func Unpaced() Pace { return Pace{} }
func FinitePace(logicalSeconds, wallSeconds uint64) (Pace, error) {
	if logicalSeconds == 0 || wallSeconds == 0 || logicalSeconds > 1_000_000_000 || wallSeconds > 1_000_000_000 {
		return Pace{}, errors.New("pace requires a bounded positive rational")
	}
	divisor := gcd(logicalSeconds, wallSeconds)
	logicalSeconds, wallSeconds = logicalSeconds/divisor, wallSeconds/divisor
	return Pace{logicalSeconds, wallSeconds}, nil
}
func (p Pace) valid() bool { return p == (Pace{}) || p.numerator > 0 && p.denominator > 0 }

type Outcome string

const (
	OutcomeComplete Outcome = "complete"
	OutcomeFailed   Outcome = "failed"
	OutcomeCanceled Outcome = "canceled"
)

type Reason string

const (
	ReasonNone               Reason = ""
	ReasonArtifactValidation Reason = "artifact_validation"
	ReasonBindingCoverage    Reason = "binding_coverage"
	ReasonSchemaCanonical    Reason = "schema_canonical_encoding"
	ReasonOrdinalGroup       Reason = "ordinal_group_order"
	ReasonClock              Reason = "clock"
	ReasonAggregate          Reason = "aggregate_admission_disposition"
	ReasonTimer              Reason = "timer_admission_disposition"
	ReasonArtifactEnd        Reason = "artifact_end_digest"
	ReasonEngine             Reason = "engine_integrity_suppression"
	ReasonCanceled           Reason = "canceled"
	ReasonDrain              Reason = "controlled_stop_drain"
)

type Accounting struct {
	ArtifactRecords, CompletedRecordDispositions, UnreadRecords  uint64
	PlannedGroups, CompletedGroups, ActiveGroup, RemainingGroups uint64
	CompletedRuns, FailedRuns, CanceledRuns                      uint64
}

type GroupResult struct {
	LogicalTime time.Time
	Records     uint64
	Timer       engine.TimerDisposition
	Status      engine.ReplayStatus
}

type Result struct {
	Outcome         Outcome
	Reason          Reason
	ArtifactID      string
	LastLogicalTime time.Time
	LastOrdinal     uint64
	Accounting      Accounting
	Status          engine.ReplayStatus
}

type Source struct {
	handle         *replayartifact.Handle
	artifactID     string
	engine         *engine.Engine
	clock          *SimulatedClock
	pace           Pace
	startAuthority playback.StartAuthority
	cursor         *playback.Cursor
	start          playback.StartEvidence
	nextGroup      time.Time
	accounting     Accounting
	terminal       bool
	failureReason  Reason
	pacer          wallPacer
}

type wallPacer struct {
	pace    Pace
	started time.Time
	now     func() time.Time
	wait    func(context.Context, time.Duration) error
}

func NewSource(handle *replayartifact.Handle, owner *engine.Engine, clock *SimulatedClock, pace Pace) (*Source, error) {
	if handle == nil || owner == nil || clock == nil || !pace.valid() {
		return nil, errors.New("replay source requires handle, engine, clock, and valid pace")
	}
	p := wallPacer{pace: pace, now: time.Now, wait: waitDuration}
	return &Source{handle: handle, artifactID: handle.Metadata().ArtifactID, engine: owner, clock: clock, pace: pace, pacer: p}, nil
}

// NewCheckpointSource binds an already-installed semantic baseline to one
// ordinary Component 4 continuation. Artifact record delivery may begin later
// because an interval can be empty, but the artifact interval identity itself
// must begin exactly at the installed cutoff; earlier repeats and later gaps
// never reach the replay engine.
func NewCheckpointSource(handle *replayartifact.Handle, owner *engine.Engine, clock *SimulatedClock, pace Pace, installed engine.InstalledCheckpointFact) (*Source, error) {
	if installed.BindingIdentity == "" || installed.T0.IsZero() || handle == nil {
		return nil, errors.New("checkpoint replay requires an installed cutoff")
	}
	metadata := handle.Metadata()
	if metadata.BindingIdentity != installed.BindingIdentity || metadata.ReplayStart != installed.T0 {
		return nil, errors.New("checkpoint replay cutoff mismatch")
	}
	source, err := NewSource(handle, owner, clock, pace)
	if err == nil {
		source.startAuthority = playback.InstalledCheckpoint
	}
	return source, err
}

// NewCompleteFallbackSource is the only C7 fallback after checkpoint rejection:
// the ordinary C4 artifact must itself prove complete input from session start.
func NewCompleteFallbackSource(handle *replayartifact.Handle, owner *engine.Engine, clock *SimulatedClock, pace Pace) (*Source, error) {
	if handle == nil {
		return nil, errors.New("fresh replay fallback requires an artifact")
	}
	metadata := handle.Metadata()
	if metadata.Mode != replayartifact.CompleteFinalBars || metadata.ReplayStart != metadata.SessionStart {
		return nil, errors.New("fresh replay fallback requires complete input from session start")
	}
	source, err := NewSource(handle, owner, clock, pace)
	if err == nil {
		source.startAuthority = playback.FreshSession
	}
	return source, err
}

func (s *Source) Start(ctx context.Context) error {
	if s == nil || ctx == nil || s.cursor != nil || s.terminal {
		return errors.New("replay start is unavailable")
	}
	cursor, err := s.handle.BeginPlayback()
	if err != nil {
		s.failPlayback(err)
		return err
	}
	start, err := cursor.Start()
	if err != nil {
		s.failPlayback(err)
		return err
	}
	start = start.WithAuthority(s.startAuthority)
	s.cursor, s.start, s.nextGroup = cursor, start, start.Start()
	s.accounting.ArtifactRecords = start.TotalRecords()
	s.accounting.UnreadRecords = start.TotalRecords()
	s.accounting.PlannedGroups = uint64(start.End().Sub(start.Start())/time.Second) + 1
	s.accounting.RemainingGroups = s.accounting.PlannedGroups
	admission, completion := s.engine.AdmitReplayStart(ctx, start)
	if admission != engine.AdmissionAdmitted || completion == nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.fail(ReasonEngine, engine.ReplayFailureEngine)
		return errors.New("replay start was not admitted")
	}
	disposition, ok := awaitDisposition(context.Background(), completion)
	if !ok || disposition.Code != engine.DispositionReplayStarted {
		s.fail(ReasonEngine, engine.ReplayFailureEngine)
		return errors.New("replay start failed")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.pacer.started = s.pacer.now()
	return nil
}

func (s *Source) Step(ctx context.Context) (GroupResult, error) {
	if s == nil || ctx == nil || s.cursor == nil || s.terminal || s.nextGroup.After(s.start.End()) {
		return GroupResult{}, errors.New("replay group is unavailable")
	}
	group := s.nextGroup
	s.accounting.ActiveGroup, s.accounting.RemainingGroups = 1, s.accounting.PlannedGroups-s.accounting.CompletedGroups-1
	if err := s.pacer.waitUntil(ctx, group.Sub(s.start.Start())); err != nil {
		return GroupResult{}, s.cancelOrFail(err)
	}
	if err := s.clock.advance(group); err != nil {
		s.fail(ReasonClock, engine.ReplayFailureClock)
		return GroupResult{}, err
	}
	var records uint64
	for {
		record, ok, err := s.cursor.NextRecord(group)
		if err != nil {
			s.failPlayback(err)
			return GroupResult{}, err
		}
		if !ok {
			break
		}
		admission, completion := s.engine.AdmitReplayRecord(ctx, record)
		if admission != engine.AdmissionAdmitted || completion == nil {
			return GroupResult{}, s.cancelOrFail(errors.New("replay aggregate was not admitted"))
		}
		disposition, completed := awaitAggregate(context.Background(), completion)
		if !completed {
			s.fail(ReasonAggregate, engine.ReplayFailureAggregate)
			return GroupResult{}, errors.New("replay aggregate disposition failed")
		}
		s.accounting.CompletedRecordDispositions++
		s.accounting.UnreadRecords--
		if !acceptedAggregate(s.start.Complete(), disposition.Code) {
			s.fail(ReasonAggregate, engine.ReplayFailureAggregate)
			return GroupResult{}, errors.New("replay aggregate disposition failed")
		}
		records++
		if ctx.Err() != nil {
			return GroupResult{}, ctx.Err()
		}
	}
	proof, err := s.cursor.FinishGroup(group)
	if err != nil {
		s.failPlayback(err)
		return GroupResult{}, err
	}
	admission, completion := s.engine.AdmitReplayGroup(ctx, proof)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return GroupResult{}, s.cancelOrFail(errors.New("replay group timer was not admitted"))
	}
	timer, completed := awaitTimer(context.Background(), completion)
	if !completed || timer.Code != engine.DispositionTimerApplied || timer.AdmissionTime != group {
		s.fail(ReasonTimer, engine.ReplayFailureTimer)
		return GroupResult{}, errors.New("replay group timer disposition failed")
	}
	s.accounting.CompletedGroups++
	s.accounting.ActiveGroup = 0
	s.nextGroup = group.Add(time.Second)
	if ctx.Err() != nil {
		return GroupResult{}, ctx.Err()
	}
	return GroupResult{LogicalTime: group, Records: records, Timer: timer, Status: s.engine.ObserveReplay()}, nil
}

func (s *Source) Finish(ctx context.Context) (Result, error) {
	if s == nil || ctx == nil || s.cursor == nil || s.terminal || !s.nextGroup.After(s.start.End()) {
		return Result{}, errors.New("replay finish is unavailable")
	}
	end, err := s.cursor.End()
	if err != nil {
		s.failPlayback(err)
		return s.result(OutcomeFailed, s.currentReason()), err
	}
	admission, completion := s.engine.AdmitReplayEnd(ctx, end)
	if admission != engine.AdmissionAdmitted || completion == nil {
		if ctx.Err() != nil {
			return s.canceled(), ctx.Err()
		}
		s.fail(ReasonArtifactEnd, engine.ReplayFailureArtifactEnd)
		return s.result(OutcomeFailed, ReasonArtifactEnd), errors.New("replay end was not admitted")
	}
	disposition, completed := awaitDisposition(context.Background(), completion)
	if !completed || disposition.Code != engine.DispositionReplayEnded {
		s.fail(ReasonEngine, engine.ReplayFailureEngine)
		return s.result(OutcomeFailed, ReasonEngine), errors.New("replay end failed")
	}
	if ctx.Err() != nil {
		return s.canceled(), ctx.Err()
	}
	if err := s.engine.Wait(ctx); err != nil {
		s.terminal = true
		return s.result(OutcomeFailed, ReasonDrain), err
	}
	s.terminal = true
	s.accounting.CompletedRuns = 1
	return s.result(OutcomeComplete, ReasonNone), nil
}

func (s *Source) Run(ctx context.Context) Result {
	if err := s.Start(ctx); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return s.canceled()
		}
		return s.result(OutcomeFailed, s.currentReason())
	}
	for !s.nextGroup.After(s.start.End()) {
		if _, err := s.Step(ctx); err != nil {
			if ctx.Err() != nil {
				return s.canceled()
			}
			return s.result(OutcomeFailed, s.currentReason())
		}
	}
	result, err := s.Finish(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return s.canceled()
		}
		return result
	}
	return result
}

func (s *Source) fail(reason Reason, engineReason engine.ReplayFailureReason) {
	if s == nil || s.terminal {
		return
	}
	s.failureReason = reason
	status := s.engine.ObserveReplay()
	artifactID := status.ArtifactID
	if artifactID == "" {
		artifactID = s.artifactID
	}
	input := engine.ReplayFailureInput{Reason: engineReason, ArtifactID: artifactID, LogicalTime: status.LastLogicalTime, Ordinal: status.LastOrdinal}
	admission, completion := s.engine.AdmitReplayFailure(context.Background(), input)
	if admission == engine.AdmissionAdmitted && completion != nil {
		_, _ = awaitDisposition(context.Background(), completion)
	}
	s.terminal = true
	s.accounting.FailedRuns = 1
}

func (s *Source) failPlayback(err error) {
	switch playback.Classify(err) {
	case playback.ErrorBindingCoverage:
		s.fail(ReasonBindingCoverage, engine.ReplayFailureBindingCoverage)
	case playback.ErrorSchemaCanonical:
		s.fail(ReasonSchemaCanonical, engine.ReplayFailureSchemaCanonical)
	case playback.ErrorOrdinalGroup:
		s.fail(ReasonOrdinalGroup, engine.ReplayFailureOrdinalGroup)
	case playback.ErrorArtifactEnd:
		s.fail(ReasonArtifactEnd, engine.ReplayFailureArtifactEnd)
	default:
		s.fail(ReasonArtifactValidation, engine.ReplayFailureArtifactValidation)
	}
}

func (s *Source) cancelOrFail(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	s.fail(ReasonEngine, engine.ReplayFailureEngine)
	return err
}
func (s *Source) canceled() Result {
	if s == nil {
		return Result{Outcome: OutcomeCanceled, Reason: ReasonCanceled, Accounting: Accounting{CanceledRuns: 1}}
	}
	if s.engine.ObserveReplay().Suppression == engine.SuppressionTerminalReplayFailure {
		s.accounting.FailedRuns = 1
		return s.result(OutcomeFailed, ReasonEngine)
	}
	if !s.terminal {
		admission, completion := s.engine.Stop(context.Background())
		if admission == engine.AdmissionAdmitted && completion != nil {
			_, _ = awaitDisposition(context.Background(), completion)
		}
		s.engine.Close()
		_ = s.engine.Wait(context.Background())
	}
	s.terminal = true
	s.accounting.CanceledRuns = 1
	return s.result(OutcomeCanceled, ReasonCanceled)
}
func (s *Source) currentReason() Reason {
	if s.failureReason != ReasonNone {
		return s.failureReason
	}
	if s.engine.ObserveReplay().Suppression != "" {
		return ReasonEngine
	}
	return ReasonArtifactValidation
}
func (s *Source) result(outcome Outcome, reason Reason) Result {
	status := s.engine.ObserveReplay()
	artifactID := status.ArtifactID
	if artifactID == "" {
		artifactID = s.artifactID
	}
	return Result{Outcome: outcome, Reason: reason, ArtifactID: artifactID, LastLogicalTime: status.LastLogicalTime, LastOrdinal: status.LastOrdinal, Accounting: s.accounting, Status: status}
}

func (p *wallPacer) waitUntil(ctx context.Context, logical time.Duration) error {
	if p.pace == (Pace{}) {
		return nil
	}
	seconds := uint64(logical / time.Second)
	hi, lo := bits.Mul64(seconds, p.pace.denominator)
	if hi != 0 {
		return errors.New("pace duration overflow")
	}
	whole, remainder := lo/p.pace.numerator, lo%p.pace.numerator
	if whole > math.MaxInt64/uint64(time.Second) {
		return errors.New("pace duration overflow")
	}
	fractionalNanoseconds := remainder * uint64(time.Second) / p.pace.numerator
	wholeNanoseconds := whole * uint64(time.Second)
	if fractionalNanoseconds > math.MaxInt64-wholeNanoseconds {
		return errors.New("pace duration overflow")
	}
	nanoseconds := wholeNanoseconds + fractionalNanoseconds
	wait := p.started.Add(time.Duration(nanoseconds)).Sub(p.now())
	if wait <= 0 {
		return nil
	}
	return p.wait(ctx, wait)
}
func waitDuration(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func awaitDisposition(ctx context.Context, ch <-chan engine.Disposition) (engine.Disposition, bool) {
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return engine.Disposition{}, false
	}
}
func awaitAggregate(ctx context.Context, ch <-chan engine.AggregateDisposition) (engine.AggregateDisposition, bool) {
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return engine.AggregateDisposition{}, false
	}
}
func awaitTimer(ctx context.Context, ch <-chan engine.TimerDisposition) (engine.TimerDisposition, bool) {
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return engine.TimerDisposition{}, false
	}
}
func acceptedAggregate(complete bool, code engine.DispositionCode) bool {
	if complete {
		return code == engine.DispositionAggregateInserted
	}
	return code == engine.DispositionAggregateInserted || code == engine.DispositionAggregateRevised || code == engine.DispositionAggregateExactDuplicate
}
func wholeSecond(value time.Time) bool {
	return !value.IsZero() && value == value.UTC() && value.Nanosecond() == 0
}
func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
