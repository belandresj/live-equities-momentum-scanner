package operations

import (
	"context"
	"errors"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
)

type ReplayPhase string

const (
	ReplayWarming         ReplayPhase = "warming"
	ReplayObserving       ReplayPhase = "observing"
	ReplayFinalizing      ReplayPhase = "finalizing"
	ReplayRetainedSuccess ReplayPhase = "retained_success"
	ReplayCanceling       ReplayPhase = "canceling"
	ReplaySuppressed      ReplayPhase = "suppressed"
	ReplayShuttingDown    ReplayPhase = "shutting_down"
)

type ReplayWindowAccounting struct {
	WarmupGroupsPlanned, WarmupGroupsCompleted, WarmupGroupActive, WarmupGroupsRemaining                         uint64
	ObservationSecondsPlanned, ObservationSecondsCompleted, ObservationSecondActive, ObservationSecondsRemaining uint64
	ObservationBoundariesPublished                                                                               uint64
}

type ReplayCaptureContext struct {
	Phase                                                      ReplayPhase
	ArtifactID                                                 string
	ArtifactEnd, ObservationStart, ObservationEnd, LogicalTime time.Time
	Completion                                                 replay.CompletionDisposition
	ScheduleLag                                                *time.Duration
	Source                                                     replay.Accounting
	Window                                                     ReplayWindowAccounting
}

type ReplayRuntime struct {
	engine      *engine.Engine
	binding     reference.Binding
	config      Config
	wallClock   func() time.Time
	processLive atomic.Bool
	joined      atomic.Bool
	mu          sync.Mutex
	sequence    uint64
	current     *replayCaptureView
}

type replayCaptureView struct {
	Engine  engine.SnapshotView
	Context ReplayCaptureContext
}

func NewReplay(ctx context.Context, binding reference.Binding, config Config, logicalClock, wallClock func() time.Time) (*ReplayRuntime, error) {
	if ctx == nil || binding.Identity() == "" || !config.valid() || logicalClock == nil || wallClock == nil {
		return nil, errors.New("invalid replay runtime configuration")
	}
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: logicalClock, Capacity: config.EngineCapacity, RequiredReserve: config.RequiredReserve, EvaluationDelay: &config.EvaluationDelay})
	if err != nil {
		return nil, err
	}
	admission, completion := owner.AdmitBinding(ctx, engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || completion == nil {
		owner.Close()
		_ = owner.Wait(ctx)
		return nil, errors.New("replay runtime binding was not installed")
	}
	select {
	case <-ctx.Done():
		owner.Close()
		_ = owner.Wait(ctx)
		return nil, ctx.Err()
	case disposition := <-completion:
		if disposition.Code != engine.DispositionBindingInstalled {
			owner.Close()
			_ = owner.Wait(ctx)
			return nil, errors.New("replay runtime binding was not installed")
		}
	}
	result := &ReplayRuntime{engine: owner, binding: binding, config: config, wallClock: wallClock}
	result.processLive.Store(true)
	return result, nil
}

func (r *ReplayRuntime) Engine() *engine.Engine {
	if r == nil {
		return nil
	}
	return r.engine
}

func (r *ReplayRuntime) PublishReplay(value ReplayCaptureContext) error {
	if r == nil || r.engine == nil || !validReplayCaptureContext(value, r.binding) {
		return errors.New("invalid replay capture")
	}
	view := r.engine.ObserveSnapshot()
	if view.Publication.PublicationID == 0 || view.Publication.BindingIdentity != r.binding.Identity() || view.Publication.RunMode != engine.RunModeReplay {
		return errors.New("replay capture does not match engine publication")
	}
	if view.Publication.GeneratedAt.After(value.LogicalTime) {
		return errors.New("replay capture precedes engine publication")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.joined.Load() {
		return errors.New("replay runtime already joined")
	}
	copyValue := cloneReplayCaptureContext(value)
	r.current = &replayCaptureView{Engine: cloneEngineSnapshot(view), Context: copyValue}
	return nil
}

// PublishReplayTerminal consumes Component 4's exact terminal status after its
// source-owned drain. Failure drain may make the engine observer unavailable,
// so the last coherent publication identity and operational counters are
// retained while C4's terminal lifecycle/evaluation atomically replace market
// state and C12 removes replay-authoritative presentation.
func (r *ReplayRuntime) PublishReplayTerminal(value ReplayCaptureContext, result replay.Result) (bool, error) {
	if r == nil || r.engine == nil || !validReplayCaptureContext(value, r.binding) || !validTerminalReplayResult(result, value) {
		return false, errors.New("invalid terminal replay capture")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.joined.Load() {
		return false, errors.New("replay runtime already joined")
	}
	if result.Outcome == replay.OutcomeFailed {
		// The exact engine failure publication is an unavailable sentinel with
		// no C10 identity. Withdraw the prior capture instead of rewriting its
		// immutable publication ID into a synthetic suppressed publication.
		r.current = nil
		return false, nil
	}
	if r.current == nil {
		return false, errors.New("terminal replay capture has no prior publication")
	}
	view := cloneEngineSnapshot(r.current.Engine)
	view.Publication = cloneReplayPublication(result.Status.Publication)
	view.Operational = terminalOperationalView(view.Operational, view.Publication, r.config)
	view.TQ.PublicationID = view.Publication.PublicationID
	copyValue := cloneReplayCaptureContext(value)
	r.current = &replayCaptureView{Engine: view, Context: copyValue}
	return true, nil
}

func (r *ReplayRuntime) CaptureSnapshot() (SnapshotCapture, error) {
	if r == nil || r.engine == nil || r.wallClock == nil {
		return SnapshotCapture{}, errors.New("replay runtime snapshot unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return SnapshotCapture{}, errors.New("replay runtime snapshot unavailable")
	}
	if r.sequence == math.MaxUint64 {
		return SnapshotCapture{}, errors.New("runtime snapshot sequence exhausted")
	}
	r.sequence++
	sampledAt := r.wallClock().UTC()
	if sampledAt.IsZero() {
		return SnapshotCapture{}, errors.New("invalid replay sample clock")
	}
	processLive := r.processLive.Load() && !r.joined.Load()
	engineView := cloneEngineSnapshot(r.current.Engine)
	status := deriveReplayStatus(processLive, r.binding, r.config, sampledAt, r.current.Context, engineView.Operational, engineView.Publication.CurrentMarketClaim)
	metrics := replayMetrics(sampledAt, engineView.Operational)
	replayView := ReplayCaptureView(r.current.Context)
	return SnapshotCapture{sealed: &sealedSnapshotCapture{view: SnapshotCaptureView{
		SampleID: r.sequence, SampledAt: sampledAt, ProcessLive: processLive, Engine: engineView, Status: status, Metrics: metrics, Replay: &replayView,
	}}}, nil
}

func (r *ReplayRuntime) Shutdown(ctx context.Context) error {
	if r == nil || ctx == nil {
		return errors.New("invalid replay runtime shutdown")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r.processLive.Store(false)
	r.joined.Store(true)
	return nil
}

type ReplayCaptureView ReplayCaptureContext

func deriveReplayStatus(processLive bool, binding reference.Binding, config Config, sampledAt time.Time, replayContext ReplayCaptureContext, view engine.OperationalView, currentClaim bool) Status {
	target := replayContext.LogicalTime.Add(-config.EvaluationDelay).Truncate(time.Second)
	if target.Before(binding.SessionStart()) {
		target = binding.SessionStart()
	}
	if target.After(binding.SessionEnd()) {
		target = binding.SessionEnd()
	}
	result := Status{ProcessLive: processLive, BackendReady: false, Reason: ReasonNotLiveMode, Lifecycle: view.Lifecycle, RankingMode: view.RankingMode,
		PublicationID: view.PublicationID, SampledAt: sampledAt, Watermark: cloneTime(view.Watermark), CausalTarget: &target,
		QueueCapacity: view.QueueCapacity, QueueOccupancy: view.QueueOccupancy, AccountingValid: operationalAccountingValid(view)}
	result.RankingCurrent = processLive && (replayContext.Phase == ReplayObserving || replayContext.Phase == ReplayFinalizing) && currentClaim && view.Suppression == ""
	if view.Watermark != nil && target.After(*view.Watermark) {
		result.WatermarkLag = target.Sub(*view.Watermark)
	}
	return result
}

func replayMetrics(sampledAt time.Time, view engine.OperationalView) Metrics {
	result := Metrics{SampledAt: sampledAt, Engine: view, AccountingValid: operationalAccountingValid(view),
		DeliveryLatencyAttribution: DeliveryLatencyAttribution{MaximumFamily: DeliveryLatencyUnknown}}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	result.HeapAllocBytes, result.HeapInUseBytes = memory.HeapAlloc, memory.HeapInuse
	result.Goroutines = runtime.NumGoroutine()
	return result
}

func validReplayCaptureContext(value ReplayCaptureContext, binding reference.Binding) bool {
	if value.ArtifactID == "" || value.ArtifactEnd.IsZero() || value.ObservationStart.IsZero() || value.ObservationEnd.IsZero() || value.LogicalTime.IsZero() ||
		value.ArtifactEnd != value.ArtifactEnd.UTC() || value.ObservationStart != value.ObservationStart.UTC() || value.ObservationEnd != value.ObservationEnd.UTC() || value.LogicalTime != value.LogicalTime.UTC() ||
		value.ObservationStart.Before(binding.SessionStart()) || !value.ObservationStart.Before(value.ObservationEnd) || value.ObservationEnd.After(value.ArtifactEnd) || value.ArtifactEnd.After(binding.SessionEnd()) ||
		value.LogicalTime.Before(binding.SessionStart()) || value.LogicalTime.After(value.ObservationEnd) {
		return false
	}
	switch value.Phase {
	case ReplayWarming, ReplayObserving, ReplayFinalizing, ReplayRetainedSuccess, ReplayCanceling, ReplaySuppressed, ReplayShuttingDown:
	default:
		return false
	}
	if value.Phase == ReplayRetainedSuccess {
		if value.Completion != replay.CompletionArtifactEnd && value.Completion != replay.CompletionRequestedEnd {
			return false
		}
	} else if value.Completion != "" {
		return false
	}
	return true
}

func validTerminalReplayResult(result replay.Result, value ReplayCaptureContext) bool {
	status := result.Status
	if status.RunMode != engine.RunModeReplay || status.ArtifactID != value.ArtifactID || status.LastLogicalTime.After(value.LogicalTime) || result.ArtifactID != value.ArtifactID {
		return false
	}
	switch value.Phase {
	case ReplayRetainedSuccess:
		return result.Outcome == replay.OutcomeComplete && validTerminalPublication(status.Publication, value, "ended", "")
	case ReplayCanceling:
		return result.Outcome == replay.OutcomeCanceled && validTerminalPublication(status.Publication, value, "ended", "")
	case ReplaySuppressed:
		return result.Outcome == replay.OutcomeFailed && status.Lifecycle == "suppressed" && status.Suppression == engine.SuppressionTerminalReplayFailure &&
			status.Publication.PublicationID == 0 && status.Publication.Lifecycle == "suppressed" && status.Publication.Suppression == engine.SuppressionTerminalReplayFailure
	default:
		return false
	}
}

func validTerminalPublication(value engine.ReplayPublicationView, context ReplayCaptureContext, lifecycle string, suppression engine.SuppressionDisposition) bool {
	return value.PublicationID > 0 && value.BindingIdentity != "" && value.RunMode == engine.RunModeReplay && value.Lifecycle == lifecycle &&
		value.Suppression == suppression && !value.GeneratedAt.IsZero() && !value.GeneratedAt.After(context.LogicalTime)
}

func terminalOperationalView(value engine.OperationalView, publication engine.ReplayPublicationView, config Config) engine.OperationalView {
	value.PublicationID, value.LastEngineSequence = publication.PublicationID, publication.LastEngineSequence
	value.BindingIdentity, value.TradingDate, value.RunMode = publication.BindingIdentity, publication.TradingDate, publication.RunMode
	value.Lifecycle, value.LifecycleReason, value.Suppression = publication.Lifecycle, publication.LifecycleReason, publication.Suppression
	value.Watermark, value.GeneratedAt = cloneTime(publication.Watermark), publication.GeneratedAt
	value.RankingMode, value.RankingReason, value.CurrentMarketClaim = publication.AggregateEvaluation.Mode, publication.AggregateEvaluation.Reason, publication.CurrentMarketClaim
	if value.QueueCapacity == 0 {
		value.QueueCapacity, value.RequiredReserve = config.EngineCapacity, config.RequiredReserve
	}
	return value
}

func cloneReplayPublication(value engine.ReplayPublicationView) engine.ReplayPublicationView {
	result := value
	result.Watermark = cloneTime(value.Watermark)
	result.AggregateEvaluation = cloneReplayEvaluation(value.AggregateEvaluation)
	return result
}

func cloneReplayCaptureContext(value ReplayCaptureContext) ReplayCaptureContext {
	result := value
	if value.ScheduleLag != nil {
		copyValue := *value.ScheduleLag
		result.ScheduleLag = &copyValue
	}
	return result
}

func cloneEngineSnapshot(value engine.SnapshotView) engine.SnapshotView {
	result := value
	result.Publication.AggregateEvaluation = cloneReplayEvaluation(value.Publication.AggregateEvaluation)
	result.Publication.Watermark = cloneTime(value.Publication.Watermark)
	result.Operational.Watermark = cloneTime(value.Operational.Watermark)
	result.Operational.Hydration.SupportedThrough = cloneTime(value.Operational.Hydration.SupportedThrough)
	result.TQ.Desired = append([]string(nil), value.TQ.Desired...)
	result.TQ.Rows = append([]engine.TQSymbolView(nil), value.TQ.Rows...)
	return result
}
