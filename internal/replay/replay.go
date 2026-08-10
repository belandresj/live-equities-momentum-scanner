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

type CompletionDisposition string

const (
	CompletionArtifactEnd  CompletionDisposition = "artifact_end"
	CompletionRequestedEnd CompletionDisposition = "requested_end"
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
	ReasonRequestedEnd       Reason = "requested_end"
	ReasonEngine             Reason = "engine_integrity_suppression"
	ReasonCanceled           Reason = "canceled"
	ReasonDrain              Reason = "controlled_stop_drain"
)

type Accounting struct {
	ArtifactRecords, CompletedRecordDispositions, IntentionallyUnappliedSuffixRecords, UnreadRecords uint64
	PlannedGroups, CompletedGroups, ActiveGroup, RemainingGroups                                     uint64
	CompletedRuns, FailedRuns, CanceledRuns                                                          uint64
}

type GroupResult struct {
	LogicalTime time.Time
	Records     uint64
	Timer       engine.TimerDisposition
	Status      engine.ReplayStatus
}

type Result struct {
	Outcome         Outcome
	Completion      CompletionDisposition
	Reason          Reason
	ArtifactID      string
	LastLogicalTime time.Time
	LastOrdinal     uint64
	Accounting      Accounting
	Status          engine.ReplayStatus
}

type Source struct {
	handle                  *replayartifact.Handle
	artifactID              string
	engine                  *engine.Engine
	clock                   *SimulatedClock
	pace                    Pace
	startAuthority          playback.StartAuthority
	cursor                  *playback.Cursor
	start                   playback.StartEvidence
	requestedEnd            time.Time
	nextGroup               time.Time
	accounting              Accounting
	terminal                bool
	failureReason           Reason
	pacer                   wallPacer
	operation               chan struct{}
	lifetime                context.Context
	cancelLifetime          context.CancelFunc
	cancelOnce              sync.Once
	terminalLinked          bool
	terminalCompletion      <-chan engine.Disposition
	terminalWant            engine.DispositionCode
	terminalCompletionKind  CompletionDisposition
	terminalOutcome         Outcome
	terminalDispositionDone bool
	terminalDispositionFail bool
	pendingLinked           linkedOperation
	pendingDisposition      <-chan engine.Disposition
	pendingAggregate        <-chan engine.AggregateDisposition
	pendingTimer            <-chan engine.TimerDisposition
	pendingGroup            time.Time
	stopCompletion          <-chan engine.Disposition
	stopCompleted           bool
	engineClosed            bool
	sealedResult            Result
	resultSealed            bool
	// These package-private hooks expose only the requested-end terminal-fact
	// and already-linked nonterminal boundaries to deterministic lifecycle tests.
	beforeRequestedEndAdmission func()
	afterRequestedEndAdmission  func()
	afterNonterminalAdmission   func()
	beforeFailureAdmission      func()
	afterFailureAdmission       func()
}

type linkedOperation uint8

const (
	linkedNone linkedOperation = iota
	linkedReplayStart
	linkedReplayAggregate
	linkedReplayTimer
)

type wallPacer struct {
	pace    Pace
	started time.Time
	now     func() time.Time
	wait    func(context.Context, time.Duration) error
}

func NewSource(handle *replayartifact.Handle, owner *engine.Engine, clock *SimulatedClock, pace Pace) (*Source, error) {
	if handle == nil {
		return nil, errors.New("replay source requires an artifact")
	}
	return NewSourceThrough(handle, owner, clock, pace, handle.Metadata().ReplayEnd)
}

// NewSourceThrough selects an exact application boundary without changing the
// complete artifact's trusted header interval or allowing partial evidence.
func NewSourceThrough(handle *replayartifact.Handle, owner *engine.Engine, clock *SimulatedClock, pace Pace, requestedEnd time.Time) (*Source, error) {
	if handle == nil || owner == nil || clock == nil || !pace.valid() {
		return nil, errors.New("replay source requires handle, engine, clock, and valid pace")
	}
	metadata := handle.Metadata()
	if !wholeSecond(requestedEnd) || !metadata.ReplayStart.Before(requestedEnd) || requestedEnd.After(metadata.ReplayEnd) ||
		(requestedEnd.Before(metadata.ReplayEnd) && metadata.Mode != replayartifact.CompleteFinalBars) {
		return nil, errors.New("invalid requested replay end")
	}
	p := wallPacer{pace: pace, now: time.Now, wait: waitDuration}
	lifetime, cancelLifetime := context.WithCancel(context.Background())
	operation := make(chan struct{}, 1)
	operation <- struct{}{}
	return &Source{handle: handle, artifactID: metadata.ArtifactID, engine: owner, clock: clock, pace: pace, requestedEnd: requestedEnd, pacer: p,
		operation: operation, lifetime: lifetime, cancelLifetime: cancelLifetime}, nil
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

const linkedDispositionLimit = 2 * time.Minute

func (s *Source) beginOperation(ctx context.Context) (context.Context, func(), error) {
	if s == nil || ctx == nil {
		return nil, nil, errors.New("replay operation requires a source and context")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if err := s.lifetime.Err(); err != nil {
		return nil, nil, err
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-s.lifetime.Done():
		return nil, nil, context.Canceled
	case <-s.operation:
	}
	if err := ctx.Err(); err != nil {
		s.operation <- struct{}{}
		return nil, nil, err
	}
	if err := s.lifetime.Err(); err != nil {
		s.operation <- struct{}{}
		return nil, nil, err
	}
	operationContext, cancel := context.WithCancel(ctx)
	stopLifetime := context.AfterFunc(s.lifetime, cancel)
	release := func() {
		stopLifetime()
		cancel()
		s.operation <- struct{}{}
	}
	return operationContext, release, nil
}

func linkedContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithTimeout(context.Background(), linkedDispositionLimit)
}

func cancellationError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (s *Source) Start(ctx context.Context) error {
	operationContext, release, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer release()
	if s.cursor != nil || s.terminal {
		return errors.New("replay start is unavailable")
	}
	cursor, err := s.handle.BeginPlaybackThroughContext(operationContext, s.requestedEnd)
	if err != nil {
		if !cancellationError(err) {
			s.failPlayback(operationContext, err)
		}
		return err
	}
	start, err := cursor.StartContext(operationContext)
	if err != nil {
		if !cancellationError(err) {
			s.failPlayback(operationContext, err)
		}
		return err
	}
	start = start.WithAuthority(s.startAuthority)
	s.cursor, s.start, s.nextGroup = cursor, start, start.Start()
	s.accounting.ArtifactRecords = start.TotalRecords()
	s.accounting.UnreadRecords = start.TotalRecords()
	s.accounting.PlannedGroups = uint64(start.RequestedEnd().Sub(start.Start())/time.Second) + 1
	s.accounting.RemainingGroups = s.accounting.PlannedGroups
	admission, completion := s.engine.AdmitReplayStart(operationContext, start)
	if admission != engine.AdmissionAdmitted || completion == nil {
		if operationContext.Err() != nil {
			return operationContext.Err()
		}
		s.fail(operationContext, ReasonEngine, engine.ReplayFailureEngine)
		return errors.New("replay start was not admitted")
	}
	if s.afterNonterminalAdmission != nil {
		s.afterNonterminalAdmission()
	}
	waitContext, cancelWait := linkedContext(ctx)
	disposition, ok := awaitDisposition(waitContext, completion)
	cancelWait()
	if !ok || disposition.Code != engine.DispositionReplayStarted {
		if ctx.Err() != nil {
			s.pendingLinked = linkedReplayStart
			s.pendingDisposition = completion
			s.cancelOnce.Do(s.cancelLifetime)
			return ctx.Err()
		}
		s.fail(operationContext, ReasonEngine, engine.ReplayFailureEngine)
		return errors.New("replay start failed")
	}
	if operationContext.Err() != nil {
		return operationContext.Err()
	}
	s.pacer.started = s.pacer.now()
	return nil
}

func (s *Source) Step(ctx context.Context) (GroupResult, error) {
	operationContext, release, err := s.beginOperation(ctx)
	if err != nil {
		return GroupResult{}, err
	}
	defer release()
	if s.cursor == nil || s.terminal || s.nextGroup.After(s.start.RequestedEnd()) {
		return GroupResult{}, errors.New("replay group is unavailable")
	}
	group := s.nextGroup
	s.accounting.ActiveGroup, s.accounting.RemainingGroups = 1, s.accounting.PlannedGroups-s.accounting.CompletedGroups-1
	if err := s.pacer.waitUntil(operationContext, group.Sub(s.start.Start())); err != nil {
		return GroupResult{}, s.cancelOrFail(operationContext, err)
	}
	if err := s.clock.advance(group); err != nil {
		s.fail(operationContext, ReasonClock, engine.ReplayFailureClock)
		return GroupResult{}, err
	}
	var records uint64
	for {
		record, ok, err := s.cursor.NextRecordContext(operationContext, group)
		if err != nil {
			if !cancellationError(err) {
				s.failPlayback(operationContext, err)
			}
			return GroupResult{}, err
		}
		if !ok {
			break
		}
		admission, completion := s.engine.AdmitReplayRecord(operationContext, record)
		if admission != engine.AdmissionAdmitted || completion == nil {
			return GroupResult{}, s.cancelOrFail(operationContext, errors.New("replay aggregate was not admitted"))
		}
		if s.afterNonterminalAdmission != nil {
			s.afterNonterminalAdmission()
		}
		waitContext, cancelWait := linkedContext(ctx)
		disposition, completed := awaitAggregate(waitContext, completion)
		cancelWait()
		if !completed {
			if ctx.Err() != nil {
				s.pendingLinked = linkedReplayAggregate
				s.pendingAggregate = completion
				s.cancelOnce.Do(s.cancelLifetime)
				return GroupResult{}, ctx.Err()
			}
			s.fail(operationContext, ReasonAggregate, engine.ReplayFailureAggregate)
			return GroupResult{}, errors.New("replay aggregate disposition failed")
		}
		s.accounting.CompletedRecordDispositions++
		s.accounting.UnreadRecords--
		if !acceptedAggregate(s.start.Complete(), disposition.Code) {
			s.fail(operationContext, ReasonAggregate, engine.ReplayFailureAggregate)
			return GroupResult{}, errors.New("replay aggregate disposition failed")
		}
		records++
		if operationContext.Err() != nil {
			return GroupResult{}, operationContext.Err()
		}
	}
	proof, err := s.cursor.FinishGroupContext(operationContext, group)
	if err != nil {
		if !cancellationError(err) {
			s.failPlayback(operationContext, err)
		}
		return GroupResult{}, err
	}
	admission, completion := s.engine.AdmitReplayGroup(operationContext, proof)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return GroupResult{}, s.cancelOrFail(operationContext, errors.New("replay group timer was not admitted"))
	}
	if s.afterNonterminalAdmission != nil {
		s.afterNonterminalAdmission()
	}
	waitContext, cancelWait := linkedContext(ctx)
	timer, completed := awaitTimer(waitContext, completion)
	cancelWait()
	if !completed || timer.Code != engine.DispositionTimerApplied || timer.AdmissionTime != group {
		if !completed && ctx.Err() != nil {
			s.pendingLinked = linkedReplayTimer
			s.pendingTimer = completion
			s.pendingGroup = group
			s.cancelOnce.Do(s.cancelLifetime)
			return GroupResult{}, ctx.Err()
		}
		s.fail(operationContext, ReasonTimer, engine.ReplayFailureTimer)
		return GroupResult{}, errors.New("replay group timer disposition failed")
	}
	s.accounting.CompletedGroups++
	s.accounting.ActiveGroup = 0
	s.nextGroup = group.Add(time.Second)
	if operationContext.Err() != nil {
		return GroupResult{}, operationContext.Err()
	}
	return GroupResult{LogicalTime: group, Records: records, Timer: timer, Status: s.engine.ObserveReplay()}, nil
}

func (s *Source) Finish(ctx context.Context) (Result, error) {
	operationContext, release, err := s.beginOperation(ctx)
	if err != nil {
		return Result{}, err
	}
	defer release()
	if s.cursor == nil || s.terminal || !s.nextGroup.After(s.start.RequestedEnd()) {
		return Result{}, errors.New("replay finish is unavailable")
	}
	var admission engine.AdmissionResult
	var completion <-chan engine.Disposition
	completionDisposition := CompletionArtifactEnd
	failureReason, engineFailureReason := ReasonArtifactEnd, engine.ReplayFailureArtifactEnd
	if s.start.RequestedEnd() == s.start.End() {
		end, err := s.cursor.EndContext(operationContext)
		if err != nil {
			if !cancellationError(err) {
				s.failPlayback(operationContext, err)
			}
			return s.terminalResult(), err
		}
		admission, completion = s.engine.AdmitReplayEnd(operationContext, end)
	} else {
		end, err := s.cursor.RequestedEndContext(operationContext)
		if err != nil {
			if !cancellationError(err) {
				s.failPlayback(operationContext, err)
			}
			return s.terminalResult(), err
		}
		if end.PrefixRecords() != s.accounting.CompletedRecordDispositions || end.TotalRecords() < end.PrefixRecords() {
			s.fail(operationContext, ReasonRequestedEnd, engine.ReplayFailureRequestedEnd)
			return s.terminalResult(), errors.New("requested replay end accounting contradicted")
		}
		s.accounting.IntentionallyUnappliedSuffixRecords = end.TotalRecords() - end.PrefixRecords()
		s.accounting.UnreadRecords = 0
		completionDisposition = CompletionRequestedEnd
		failureReason, engineFailureReason = ReasonRequestedEnd, engine.ReplayFailureRequestedEnd
		if s.beforeRequestedEndAdmission != nil {
			s.beforeRequestedEndAdmission()
		}
		admission, completion = s.engine.AdmitReplayRequestedEnd(operationContext, end)
		if s.afterRequestedEndAdmission != nil {
			s.afterRequestedEndAdmission()
		}
	}
	if admission != engine.AdmissionAdmitted || completion == nil {
		if operationContext.Err() != nil {
			return Result{}, operationContext.Err()
		}
		s.fail(operationContext, failureReason, engineFailureReason)
		return s.terminalResult(), errors.New("replay end was not admitted")
	}
	s.terminal = true
	s.terminalLinked = true
	s.terminalCompletion = completion
	s.terminalWant = engine.DispositionReplayEnded
	if completionDisposition == CompletionRequestedEnd {
		s.terminalWant = engine.DispositionReplayRequestedEnd
	}
	s.terminalCompletionKind = completionDisposition
	s.terminalOutcome = OutcomeComplete
	waitContext, cancelWait := linkedContext(ctx)
	result, terminalErr := s.completeLinkedTerminal(waitContext)
	cancelWait()
	return result, terminalErr
}

func (s *Source) Run(ctx context.Context) (Result, error) {
	if err := s.Start(ctx); err != nil {
		return s.sealedResultWithin(ctx), err
	}
	for !s.nextGroup.After(s.start.RequestedEnd()) {
		if _, err := s.Step(ctx); err != nil {
			return s.sealedResultWithin(ctx), err
		}
	}
	result, err := s.Finish(ctx)
	if err != nil {
		if result.Outcome != "" {
			return result, err
		}
		return s.sealedResultWithin(ctx), err
	}
	return result, nil
}

func (s *Source) fail(ctx context.Context, reason Reason, engineReason engine.ReplayFailureReason) error {
	if s == nil || s.terminal {
		return nil
	}
	s.failureReason = reason
	status := s.engine.ObserveReplay()
	artifactID := status.ArtifactID
	if artifactID == "" {
		artifactID = s.artifactID
	}
	input := engine.ReplayFailureInput{Reason: engineReason, ArtifactID: artifactID, LogicalTime: status.LastLogicalTime, Ordinal: status.LastOrdinal}
	if s.beforeFailureAdmission != nil {
		s.beforeFailureAdmission()
	}
	admission, completion := s.engine.AdmitReplayFailure(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.engine.ObserveReplay().Suppression == engine.SuppressionTerminalReplayFailure {
			return s.completeSuppressedFailure(ctx)
		}
		return errors.New("replay failure was not admitted")
	}
	s.terminal = true
	s.terminalLinked = true
	s.terminalCompletion = completion
	s.terminalWant = engine.DispositionReplayFailed
	s.terminalCompletionKind = ""
	s.terminalOutcome = OutcomeFailed
	if s.afterFailureAdmission != nil {
		s.afterFailureAdmission()
	}
	waitContext, cancelWait := linkedContext(ctx)
	_, err := s.completeLinkedTerminal(waitContext)
	cancelWait()
	return err
}

func (s *Source) failPlayback(ctx context.Context, err error) {
	switch playback.Classify(err) {
	case playback.ErrorBindingCoverage:
		s.fail(ctx, ReasonBindingCoverage, engine.ReplayFailureBindingCoverage)
	case playback.ErrorSchemaCanonical:
		s.fail(ctx, ReasonSchemaCanonical, engine.ReplayFailureSchemaCanonical)
	case playback.ErrorOrdinalGroup:
		s.fail(ctx, ReasonOrdinalGroup, engine.ReplayFailureOrdinalGroup)
	case playback.ErrorArtifactEnd:
		s.fail(ctx, ReasonArtifactEnd, engine.ReplayFailureArtifactEnd)
	default:
		s.fail(ctx, ReasonArtifactValidation, engine.ReplayFailureArtifactValidation)
	}
}

func (s *Source) cancelOrFail(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if s.lifetime.Err() != nil {
		return s.lifetime.Err()
	}
	s.fail(ctx, ReasonEngine, engine.ReplayFailureEngine)
	return err
}

// Cancel is the sole manual-source shutdown owner. It is idempotent: repeated
// calls wait for or return the same sealed outcome and never admit a second
// controlled stop. A caller deadline bounds acquisition, disposition, and
// engine drain without fabricating a terminal result on timeout.
func (s *Source) Cancel(ctx context.Context) (Result, error) {
	if s == nil || ctx == nil {
		return Result{}, errors.New("replay cancellation requires a source and context")
	}
	s.cancelOnce.Do(s.cancelLifetime)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-s.operation:
	}
	if err := ctx.Err(); err != nil {
		s.operation <- struct{}{}
		return Result{}, err
	}
	defer func() { s.operation <- struct{}{} }()
	return s.cancelLocked(ctx)
}

func (s *Source) cancelLocked(ctx context.Context) (Result, error) {
	if s.resultSealed {
		return s.sealedResult, nil
	}
	if err := s.drainPendingNonterminal(ctx); err != nil {
		return Result{}, err
	}
	if s.resultSealed {
		return s.sealedResult, nil
	}
	if s.terminalLinked && s.terminalOutcome != "" {
		return s.completeLinkedTerminal(ctx)
	}
	if s.engine.ObserveReplay().Suppression == engine.SuppressionTerminalReplayFailure {
		if err := s.completeSuppressedFailure(ctx); err != nil {
			return Result{}, err
		}
		return s.sealedResult, nil
	}
	if s.terminal {
		result := s.result(OutcomeFailed, ReasonDrain)
		s.sealResult(result)
		return result, nil
	}
	if s.stopCompletion == nil && !s.stopCompleted {
		admission, completion := s.engine.Stop(ctx)
		if admission != engine.AdmissionAdmitted || completion == nil {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			return Result{}, errors.New("replay controlled stop was not admitted")
		}
		s.terminalLinked = true
		s.stopCompletion = completion
	}
	if !s.stopCompleted {
		disposition, ok := awaitDisposition(ctx, s.stopCompletion)
		if !ok {
			return Result{}, ctx.Err()
		}
		if disposition.Code != engine.DispositionControlApplied {
			return Result{}, errors.New("replay controlled stop disposition failed")
		}
		s.stopCompletion = nil
		s.stopCompleted = true
	}
	if !s.engineClosed {
		s.engine.Close()
		s.engineClosed = true
	}
	if err := s.engine.Wait(ctx); err != nil {
		return Result{}, err
	}
	s.terminal = true
	s.clearActiveGroup()
	s.accounting.CanceledRuns = 1
	result := s.result(OutcomeCanceled, ReasonCanceled)
	s.sealResult(result)
	return result, nil
}

func (s *Source) drainPendingNonterminal(ctx context.Context) error {
	switch s.pendingLinked {
	case linkedNone:
		return nil
	case linkedReplayStart:
		disposition, ok := awaitDisposition(ctx, s.pendingDisposition)
		if !ok {
			return ctx.Err()
		}
		s.pendingDisposition = nil
		s.pendingLinked = linkedNone
		if disposition.Code != engine.DispositionReplayStarted {
			s.fail(ctx, ReasonEngine, engine.ReplayFailureEngine)
		}
	case linkedReplayAggregate:
		disposition, ok := awaitAggregate(ctx, s.pendingAggregate)
		if !ok {
			return ctx.Err()
		}
		s.pendingAggregate = nil
		s.pendingLinked = linkedNone
		s.accounting.CompletedRecordDispositions++
		if s.accounting.UnreadRecords > 0 {
			s.accounting.UnreadRecords--
		}
		if !acceptedAggregate(s.start.Complete(), disposition.Code) {
			s.fail(ctx, ReasonAggregate, engine.ReplayFailureAggregate)
		}
	case linkedReplayTimer:
		disposition, ok := awaitTimer(ctx, s.pendingTimer)
		if !ok {
			return ctx.Err()
		}
		group := s.pendingGroup
		s.pendingTimer = nil
		s.pendingGroup = time.Time{}
		s.pendingLinked = linkedNone
		if disposition.Code != engine.DispositionTimerApplied || disposition.AdmissionTime != group {
			s.fail(ctx, ReasonTimer, engine.ReplayFailureTimer)
			return nil
		}
		s.accounting.CompletedGroups++
		s.accounting.ActiveGroup = 0
		s.nextGroup = group.Add(time.Second)
	default:
		return errors.New("unknown linked replay operation")
	}
	return nil
}

func (s *Source) completeLinkedTerminal(ctx context.Context) (Result, error) {
	if s.resultSealed {
		return s.sealedResult, nil
	}
	if !s.terminalLinked || s.terminalCompletion == nil && !s.terminalDispositionDone {
		return Result{}, errors.New("replay terminal completion is unavailable")
	}
	if !s.terminalDispositionDone {
		disposition, ok := awaitDisposition(ctx, s.terminalCompletion)
		if !ok {
			return Result{}, ctx.Err()
		}
		s.terminalCompletion = nil
		s.terminalDispositionDone = true
		if disposition.Code != s.terminalWant {
			s.failureReason = ReasonEngine
			s.terminalOutcome = OutcomeFailed
			s.terminalDispositionFail = true
		}
	}
	if err := s.engine.Wait(ctx); err != nil {
		return Result{}, err
	}
	s.terminal = true
	s.clearActiveGroup()
	if s.terminalOutcome == OutcomeFailed {
		s.accounting.FailedRuns = 1
		result := s.result(OutcomeFailed, s.currentReason())
		s.sealResult(result)
		if s.terminalDispositionFail {
			return result, errors.New("replay terminal disposition failed")
		}
		return result, nil
	}
	s.accounting.CompletedRuns = 1
	result := s.result(OutcomeComplete, ReasonNone)
	result.Completion = s.terminalCompletionKind
	s.sealResult(result)
	return result, nil
}

func (s *Source) completeSuppressedFailure(ctx context.Context) error {
	if !s.engineClosed {
		s.engine.Close()
		s.engineClosed = true
	}
	if err := s.engine.Wait(ctx); err != nil {
		return err
	}
	s.terminal = true
	s.clearActiveGroup()
	s.accounting.FailedRuns = 1
	s.sealResult(s.result(OutcomeFailed, s.currentReason()))
	return nil
}

func (s *Source) clearActiveGroup() {
	s.accounting.ActiveGroup = 0
	if s.accounting.PlannedGroups >= s.accounting.CompletedGroups {
		s.accounting.RemainingGroups = s.accounting.PlannedGroups - s.accounting.CompletedGroups
	}
}

func (s *Source) sealResult(result Result) {
	if s.resultSealed {
		return
	}
	s.sealedResult = result
	s.resultSealed = true
}

func (s *Source) terminalResult() Result {
	if s != nil && s.resultSealed {
		return s.sealedResult
	}
	return Result{}
}

func (s *Source) sealedResultWithin(ctx context.Context) Result {
	if s == nil || ctx == nil {
		return Result{}
	}
	select {
	case <-s.operation:
		defer func() { s.operation <- struct{}{} }()
		return s.terminalResult()
	default:
	}
	select {
	case <-ctx.Done():
		return Result{}
	case <-s.operation:
		defer func() { s.operation <- struct{}{} }()
		return s.terminalResult()
	}
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
	if ctx == nil || ctx.Err() != nil {
		return engine.Disposition{}, false
	}
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return engine.Disposition{}, false
	}
}
func awaitAggregate(ctx context.Context, ch <-chan engine.AggregateDisposition) (engine.AggregateDisposition, bool) {
	if ctx == nil || ctx.Err() != nil {
		return engine.AggregateDisposition{}, false
	}
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return engine.AggregateDisposition{}, false
	}
}
func awaitTimer(ctx context.Context, ch <-chan engine.TimerDisposition) (engine.TimerDisposition, bool) {
	if ctx == nil || ctx.Err() != nil {
		return engine.TimerDisposition{}, false
	}
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
