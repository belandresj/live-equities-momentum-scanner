package replaymode

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const (
	MaximumArtifactBytes = int64(8 << 30)
	MaximumRecords       = int64(1_000_000_000)
)

type StartupConfig struct {
	ArtifactPath, ReferenceDirectory string
	ObservationStart, ObservationEnd string
}

type StatusRecord struct {
	Phase            operations.ReplayPhase `json:"phase"`
	TradingDate      string                 `json:"trading_date"`
	ObservationStart string                 `json:"observation_start"`
	ObservationEnd   string                 `json:"observation_end"`
	LogicalTime      string                 `json:"logical_time"`
	ScheduleLagMS    *uint64                `json:"schedule_lag_ms"`
	CompletedGroups  string                 `json:"completed_groups"`
	RemainingGroups  string                 `json:"remaining_groups"`
	CompletedRuns    string                 `json:"completed_runs"`
	FailedRuns       string                 `json:"failed_runs"`
	CanceledRuns     string                 `json:"canceled_runs"`
}

type schedule struct {
	now      func() time.Time
	wait     func(context.Context, time.Time) error
	wallStep time.Duration
}

type Runtime struct {
	handle           *replayartifact.Handle
	source           *replay.Source
	operations       *operations.ReplayRuntime
	clock            *replay.SimulatedClock
	metadata         replayartifact.Metadata
	observationStart time.Time
	observationEnd   time.Time
	schedule         schedule
	shutdownLimit    time.Duration
	mu               sync.Mutex
	accounting       replay.Accounting
	window           operations.ReplayWindowAccounting
	lastLogical      time.Time
	lastLag          *time.Duration
	terminal         replay.Result
	runStarted       bool
	reporter         func(StatusRecord) error
	reportedPhase    operations.ReplayPhase
	hasCapture       bool
	afterStart       func()
	beforeStep       func(time.Time)
	afterStep        func(time.Time)
	afterPublish     func(operations.ReplayPhase)
}

func Prepare(ctx context.Context, config StartupConfig) (*Runtime, error) {
	if ctx == nil || config.ArtifactPath == "" || config.ReferenceDirectory == "" || config.ObservationStart == "" || config.ObservationEnd == "" {
		return nil, errors.New("invalid replay startup configuration")
	}
	candidate, err := replayartifact.ProbeCandidateHeader(ctx, config.ArtifactPath, MaximumArtifactBytes)
	if err != nil {
		return nil, fmt.Errorf("probe replay artifact candidate: %w", err)
	}
	if candidate.Mode != replayartifact.CompleteFinalBars || candidate.ReplayStart != candidate.SessionStart {
		return nil, errors.New("replay artifact candidate is not a complete session source")
	}
	binding, err := cachedBinding(ctx, candidate.TradingDate, config.ReferenceDirectory)
	if err != nil {
		return nil, err
	}
	if candidate.BindingIdentity != binding.Identity() || candidate.UniverseIdentity != binding.UniverseIdentity() || candidate.TradingDate != binding.TradingDate() ||
		candidate.SessionStart != binding.SessionStart() || candidate.SessionEnd != binding.SessionEnd() {
		return nil, errors.New("replay artifact candidate does not match exact-date binding")
	}
	observationStart, err := observationTime(binding.TradingDate(), config.ObservationStart)
	if err != nil {
		return nil, errors.New("invalid observation start")
	}
	observationEnd, err := observationTime(binding.TradingDate(), config.ObservationEnd)
	if err != nil || observationStart.Before(binding.SessionStart()) || !observationStart.Before(observationEnd) || observationEnd.After(binding.SessionEnd()) || observationEnd.After(candidate.ReplayEnd) {
		return nil, errors.New("invalid observation window")
	}
	handle, err := replayartifact.OpenValidatedContext(ctx, config.ArtifactPath, replayartifact.ValidationPlan{Binding: binding, Start: candidate.ReplayStart, End: candidate.ReplayEnd,
		ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: MaximumArtifactBytes, MaximumRecords: MaximumRecords})
	if err != nil {
		return nil, fmt.Errorf("validate complete replay artifact: %w", err)
	}
	runtime, err := newRuntime(ctx, binding, handle, observationStart, observationEnd, productionSchedule())
	if err != nil {
		handle.Close()
		return nil, err
	}
	return runtime, nil
}

func newRuntime(ctx context.Context, binding reference.Binding, handle *replayartifact.Handle, observationStart, observationEnd time.Time, pacing schedule) (*Runtime, error) {
	if handle == nil || pacing.now == nil || pacing.wait == nil || pacing.wallStep <= 0 {
		return nil, errors.New("invalid replay runtime")
	}
	metadata := handle.Metadata()
	if metadata.Mode != replayartifact.CompleteFinalBars || metadata.BindingIdentity != binding.Identity() || metadata.ReplayStart != binding.SessionStart() ||
		observationStart.Before(metadata.ReplayStart) || !observationStart.Before(observationEnd) || observationEnd.After(metadata.ReplayEnd) {
		return nil, errors.New("replay runtime bounds do not match validated artifact")
	}
	logicalClock, err := replay.NewSimulatedClock(metadata.ReplayStart)
	if err != nil {
		return nil, err
	}
	config := operations.DefaultConfig()
	owner, err := operations.NewReplay(ctx, binding, config, logicalClock.Now, func() time.Time { return pacing.now().UTC() })
	if err != nil {
		return nil, err
	}
	if err := owner.Engine().ConfigureReplayFastForwardThrough(observationStart); err != nil {
		shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownDeadline)
		defer cancel()
		_ = owner.Shutdown(shutdown)
		return nil, err
	}
	source, err := replay.NewSourceThrough(handle, owner.Engine(), logicalClock, replay.Unpaced(), observationEnd)
	if err != nil {
		shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownDeadline)
		defer cancel()
		_ = owner.Shutdown(shutdown)
		return nil, err
	}
	totalRecords := uint64(metadata.AggregateRecords)
	plannedGroups := uint64(observationEnd.Sub(metadata.ReplayStart)/time.Second) + 1
	warmupGroups := uint64(observationStart.Sub(metadata.ReplayStart)/time.Second) + 1
	return &Runtime{handle: handle, source: source, operations: owner, clock: logicalClock, metadata: metadata, observationStart: observationStart,
		observationEnd: observationEnd, schedule: pacing, shutdownLimit: config.ShutdownDeadline, lastLogical: metadata.ReplayStart,
		accounting: replay.Accounting{ArtifactRecords: totalRecords, UnreadRecords: totalRecords, PlannedGroups: plannedGroups, RemainingGroups: plannedGroups},
		window: operations.ReplayWindowAccounting{WarmupGroupsPlanned: warmupGroups, WarmupGroupsRemaining: warmupGroups,
			ObservationSecondsPlanned: uint64(observationEnd.Sub(observationStart) / time.Second), ObservationSecondsRemaining: uint64(observationEnd.Sub(observationStart) / time.Second)}}, nil
}

func productionSchedule() schedule {
	return schedule{now: time.Now, wallStep: time.Second, wait: func(ctx context.Context, deadline time.Time) error {
		delay := time.Until(deadline)
		if delay <= 0 {
			return ctx.Err()
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}}
}

func (r *Runtime) CaptureSnapshot() (operations.SnapshotCapture, error) {
	return r.operations.CaptureSnapshot()
}

func (r *Runtime) Run(ctx context.Context) (replay.Result, error) {
	return r.run(ctx, nil)
}

func (r *Runtime) RunReporting(ctx context.Context, reporter func(StatusRecord) error) (replay.Result, error) {
	if reporter == nil {
		return replay.Result{}, errors.New("replay status reporter is required")
	}
	return r.run(ctx, reporter)
}

func (r *Runtime) run(ctx context.Context, reporter func(StatusRecord) error) (replay.Result, error) {
	if r == nil || ctx == nil {
		return replay.Result{}, errors.New("invalid replay observation run")
	}
	r.mu.Lock()
	if r.runStarted {
		r.mu.Unlock()
		return replay.Result{}, errors.New("replay observation already started")
	}
	r.runStarted = true
	r.reporter = reporter
	r.mu.Unlock()
	if err := r.source.Start(ctx); err != nil {
		return r.contain(err)
	}
	if r.afterStart != nil {
		r.afterStart()
	}
	if err := r.publish(operations.ReplayWarming, "", nil, r.accounting); err != nil {
		return r.contain(err)
	}
	for group := r.metadata.ReplayStart; !group.After(r.observationStart); group = group.Add(time.Second) {
		if r.beforeStep != nil {
			r.beforeStep(group)
		}
		step, err := r.source.Step(ctx)
		if err != nil {
			return r.contain(err)
		}
		r.completeGroup(step.Records, group, true)
		if r.afterStep != nil {
			r.afterStep(group)
		}
	}
	anchor := r.schedule.now()
	zero := time.Duration(0)
	r.lastLag = &zero
	r.window.ObservationBoundariesPublished = 1
	if err := r.publish(operations.ReplayObserving, "", r.lastLag, r.accounting); err != nil {
		return r.contain(err)
	}
	for group := r.observationStart.Add(time.Second); !group.After(r.observationEnd); group = group.Add(time.Second) {
		deadline := anchor.Add(time.Duration(group.Sub(r.observationStart)/time.Second) * r.schedule.wallStep)
		if err := r.schedule.wait(ctx, deadline); err != nil {
			return r.contain(err)
		}
		if r.beforeStep != nil {
			r.beforeStep(group)
		}
		step, err := r.source.Step(ctx)
		if err != nil {
			return r.contain(err)
		}
		r.completeGroup(step.Records, group, false)
		if r.afterStep != nil {
			r.afterStep(group)
		}
		lag := r.schedule.now().Sub(deadline)
		if lag < 0 {
			lag = 0
		}
		r.lastLag = &lag
		r.window.ObservationBoundariesPublished = 1 + r.window.ObservationSecondsCompleted
		phase := operations.ReplayObserving
		if group.Equal(r.observationEnd) {
			phase = operations.ReplayFinalizing
		}
		if err := r.publish(phase, "", r.lastLag, r.accounting); err != nil {
			return r.contain(err)
		}
	}
	result, err := r.source.Finish(ctx)
	if err != nil {
		return r.contain(err)
	}
	if result.Outcome != replay.OutcomeComplete {
		return r.contain(errors.New("replay source did not complete"))
	}
	r.accounting = result.Accounting
	r.terminal = result
	if err := r.publish(operations.ReplayRetainedSuccess, result.Completion, r.lastLag, result.Accounting); err != nil {
		return r.contain(err)
	}
	return result, nil
}

func (r *Runtime) completeGroup(records uint64, logical time.Time, warmup bool) {
	r.lastLogical = logical
	r.accounting.CompletedRecordDispositions += records
	if records <= r.accounting.UnreadRecords {
		r.accounting.UnreadRecords -= records
	}
	r.accounting.CompletedGroups++
	r.accounting.ActiveGroup = 0
	r.accounting.RemainingGroups = r.accounting.PlannedGroups - r.accounting.CompletedGroups
	if warmup {
		r.window.WarmupGroupsCompleted++
		r.window.WarmupGroupsRemaining = r.window.WarmupGroupsPlanned - r.window.WarmupGroupsCompleted
	} else {
		r.window.ObservationSecondsCompleted++
		r.window.ObservationSecondsRemaining = r.window.ObservationSecondsPlanned - r.window.ObservationSecondsCompleted
	}
}

func (r *Runtime) publish(phase operations.ReplayPhase, completion replay.CompletionDisposition, lag *time.Duration, accounting replay.Accounting) error {
	return r.publishContext(operations.ReplayCaptureContext{Phase: phase, ArtifactID: r.metadata.ArtifactID, ArtifactEnd: r.metadata.ReplayEnd,
		ObservationStart: r.observationStart, ObservationEnd: r.observationEnd, LogicalTime: r.lastLogical, Completion: completion,
		ScheduleLag: lag, Source: accounting, Window: r.window})
}

func (r *Runtime) publishContext(value operations.ReplayCaptureContext) error {
	err := r.operations.PublishReplay(value)
	if err == nil {
		r.hasCapture = true
		err = r.reportTransition(value.Phase, value.ScheduleLag, value.Source)
	}
	if err == nil && r.afterPublish != nil {
		r.afterPublish(value.Phase)
	}
	return err
}

func (r *Runtime) publishTerminal(phase operations.ReplayPhase, completion replay.CompletionDisposition, lag *time.Duration, result replay.Result) error {
	value := operations.ReplayCaptureContext{Phase: phase, ArtifactID: r.metadata.ArtifactID, ArtifactEnd: r.metadata.ReplayEnd,
		ObservationStart: r.observationStart, ObservationEnd: r.observationEnd, LogicalTime: r.lastLogical, Completion: completion,
		ScheduleLag: lag, Source: result.Accounting, Window: r.window}
	retained, err := r.operations.PublishReplayTerminal(value, result)
	if err == nil {
		r.hasCapture = retained
		err = r.reportTransition(phase, lag, result.Accounting)
	}
	if err == nil && retained && r.afterPublish != nil {
		r.afterPublish(phase)
	}
	return err
}

func (r *Runtime) reportTransition(phase operations.ReplayPhase, lag *time.Duration, accounting replay.Accounting) error {
	if r.reporter == nil || r.reportedPhase == phase {
		return nil
	}
	record := StatusRecord{Phase: phase, TradingDate: r.metadata.TradingDate, ObservationStart: r.observationStart.Format(time.RFC3339Nano),
		ObservationEnd: r.observationEnd.Format(time.RFC3339Nano), LogicalTime: r.lastLogical.Format(time.RFC3339Nano),
		CompletedGroups: strconv.FormatUint(accounting.CompletedGroups, 10), RemainingGroups: strconv.FormatUint(accounting.RemainingGroups, 10),
		CompletedRuns: strconv.FormatUint(accounting.CompletedRuns, 10), FailedRuns: strconv.FormatUint(accounting.FailedRuns, 10), CanceledRuns: strconv.FormatUint(accounting.CanceledRuns, 10)}
	if lag != nil {
		value := uint64(0)
		if *lag > 0 {
			value = uint64(*lag / time.Millisecond)
		}
		record.ScheduleLagMS = &value
	}
	if err := r.reporter(record); err != nil {
		r.reporter = nil
		return err
	}
	r.reportedPhase = phase
	return nil
}

func (r *Runtime) contain(cause error) (replay.Result, error) {
	shutdown, cancel := context.WithTimeout(context.Background(), r.shutdownLimit)
	defer cancel()
	result, cancelErr := r.source.Cancel(shutdown)
	if result.Outcome != "" {
		r.accounting, r.terminal = result.Accounting, result
		r.lastLogical = r.clock.Now()
		phase := operations.ReplayCanceling
		if result.Outcome == replay.OutcomeComplete {
			phase = operations.ReplayRetainedSuccess
			if err := r.publishTerminal(phase, result.Completion, r.lastLag, result); err != nil {
				return result, errors.Join(cause, cancelErr, err)
			}
			return result, nil
		}
		if result.Outcome == replay.OutcomeFailed {
			phase = operations.ReplaySuppressed
		}
		if err := r.publishTerminal(phase, "", nil, result); err != nil {
			return result, errors.Join(cause, cancelErr, err)
		}
	}
	return result, errors.Join(cause, cancelErr)
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || ctx == nil {
		return errors.New("invalid replay observation shutdown")
	}
	if r.terminal.Outcome == "" {
		result, err := r.source.Cancel(ctx)
		if err != nil {
			return err
		}
		r.terminal, r.accounting = result, result.Accounting
		r.lastLogical = r.clock.Now()
	}
	if r.hasCapture {
		if err := r.publish(operations.ReplayShuttingDown, "", nil, r.accounting); err != nil {
			return err
		}
	}
	return r.operations.Shutdown(ctx)
}

func (r *Runtime) Close() error {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.Close()
}

func cachedBinding(ctx context.Context, tradingDate, directory string) (reference.Binding, error) {
	schedule, err := session.Load()
	if err != nil {
		return reference.Binding{}, errors.New("load accepted exchange schedule")
	}
	facts, err := schedule.ForTradingDate(tradingDate)
	if err != nil {
		return reference.Binding{}, errors.New("unsupported artifact trading date")
	}
	universe, err := (&reference.Resolver{DataDir: directory, Schedule: schedule}).Resolve(ctx, facts)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("load exact-date universe cache: %w", err)
	}
	if !universe.IsCurrent() {
		return reference.Binding{}, errors.New("load exact-date universe cache: cache is not current for the artifact date")
	}
	priors, err := (&reference.PriorCloseResolver{DataDir: directory, Schedule: schedule}).Resolve(ctx, facts, universe)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("load exact-date prior-close cache: %w", err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		return reference.Binding{}, errors.New("assemble cached immutable binding")
	}
	return binding, nil
}

func observationTime(tradingDate, value string) (time.Time, error) {
	if len(value) != len("15:04:05") || strings.TrimSpace(value) != value {
		return time.Time{}, errors.New("observation time must be HH:MM:SS")
	}
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", tradingDate+" "+value, location)
	if err != nil || parsed.Format("15:04:05") != value {
		return time.Time{}, errors.New("invalid observation time")
	}
	return parsed.UTC(), nil
}
