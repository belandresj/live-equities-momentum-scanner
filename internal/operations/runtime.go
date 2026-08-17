package operations

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type Config struct {
	EngineCapacity, RequiredReserve             int
	RecoveryAttempts                            int
	EvaluationDelay, ReadinessTolerance         time.Duration
	SampleCadence                               time.Duration
	RecoveryBackoffInitial, RecoveryBackoffMax  time.Duration
	ConnectionAttemptDeadline, ShutdownDeadline time.Duration
	FloatLookup                                 reference.FloatLookup
}

func DefaultConfig() Config {
	return Config{EngineCapacity: 8192, RequiredReserve: 128, RecoveryAttempts: 5, EvaluationDelay: 4 * time.Second, ReadinessTolerance: 2 * time.Second,
		SampleCadence: time.Second, RecoveryBackoffInitial: time.Second, RecoveryBackoffMax: 30 * time.Second,
		ConnectionAttemptDeadline: 60 * time.Second, ShutdownDeadline: 10 * time.Second}
}

func (c Config) valid() bool {
	return c.EngineCapacity > 1 && c.RequiredReserve > 0 && c.RequiredReserve < c.EngineCapacity && c.RecoveryAttempts > 0 && c.RecoveryAttempts <= 10 &&
		c.EvaluationDelay >= 0 && c.ReadinessTolerance >= 0 && c.ReadinessTolerance <= 10*time.Second &&
		c.SampleCadence > 0 && c.SampleCadence <= 10*time.Minute &&
		c.RecoveryBackoffInitial > 0 && c.RecoveryBackoffMax >= c.RecoveryBackoffInitial && c.RecoveryBackoffMax <= 5*time.Minute &&
		c.ConnectionAttemptDeadline > 0 && c.ConnectionAttemptDeadline <= 2*time.Minute &&
		c.ShutdownDeadline > 0 && c.ShutdownDeadline <= time.Minute
}

type Runtime struct {
	engine                     *engine.Engine
	binding                    reference.Binding
	config                     Config
	clock                      func() time.Time
	processLive                atomic.Bool
	joined                     atomic.Bool
	retirementFailed           atomic.Bool
	writer                     *checkpoint.Writer
	checkpointResultDone       chan struct{}
	metricsMu                  sync.Mutex
	liveMu                     sync.Mutex
	shutdownMu                 sync.Mutex
	captureMu                  sync.Mutex
	captureSequence            uint64
	attempt                    *massive.LiveAttempt
	adapter                    *massive.LiveAdapter
	liveCancel                 context.CancelFunc
	liveDone                   chan struct{}
	liveRunning                bool
	queueHighFrames            uint64
	queueHighBytes             int
	deliveryCount              atomic.Uint64
	deliveryTotalNanos         atomic.Uint64
	deliveryMaxNanos           atomic.Uint64
	deliveryWindowMu           sync.Mutex
	deliveryOneSecondMaxNanos  uint64
	deliveryOneSecondMaxFamily DeliveryLatencyFamily
	deliveryWindowNonempty     bool
	deliveryFamilyCounts       [deliveryLatencyFamilyCount]uint64
	deliveryWindowVersion      uint64
	consumerDeferred           atomic.Uint64
	pressureSampler            func(Metrics) engine.TQPressureSample
	metricsSnapshot            func() Metrics
	timerCancel                context.CancelFunc
	timerDone                  chan struct{}
	ingressSamplerCancel       context.CancelFunc
	ingressSamplerDone         chan struct{}
	automaticTimerObserverMu   sync.RWMutex
	automaticTimerObserver     func(automaticTimerObservation)
	ingressIncident            ingressIncidentLatch
	recoveryAttempt            recoveryAttemptLatch
	ingressHistory             ingressDiagnosticHistory
	// beforeHydrationPump is a package-private diagnostic-test seam. A nil
	// hook is the complete production behavior; tests use it only to hold the
	// consumer while exercising the fixed production queue ceiling.
	beforeHydrationPump func(*massive.LiveAttempt)
}

// automaticTimerObservation is a package-private, read-only test seam for an
// engine-owned timer completion. It is captured before runTimer performs any
// follow-on T/Q synchronization or selects another ready ticker branch.
type automaticTimerObservation struct {
	disposition engine.TimerDisposition
	timing      engine.EvaluationTimingView
	capture     SnapshotCapture
	captureErr  error
}

func New(ctx context.Context, binding reference.Binding, config Config, clock func() time.Time) (*Runtime, error) {
	return NewWithCheckpoint(ctx, binding, config, clock, nil)
}

func NewWithCheckpoint(ctx context.Context, binding reference.Binding, config Config, clock func() time.Time, writer *checkpoint.Writer) (*Runtime, error) {
	if ctx == nil || binding.Identity() == "" || !config.valid() || clock == nil {
		return nil, errors.New("invalid scanner runtime configuration")
	}
	var submitter checkpoint.Submitter
	if writer != nil {
		submitter = writer
	}
	owner, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: clock, Capacity: config.EngineCapacity, RequiredReserve: config.RequiredReserve, EvaluationDelay: &config.EvaluationDelay,
		CheckpointSubmitter: submitter, FloatLookup: config.FloatLookup, RecoveryBackoffInitial: config.RecoveryBackoffInitial, RecoveryBackoffMaximum: config.RecoveryBackoffMax})
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{engine: owner, binding: binding, config: config, clock: clock, writer: writer, pressureSampler: defaultTQPressureSample, deliveryOneSecondMaxFamily: DeliveryLatencyUnknown}
	runtime.metricsSnapshot = runtime.Metrics
	admission, completion := owner.AdmitBinding(ctx, engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || completion == nil {
		owner.Close()
		_ = owner.Wait(ctx)
		return nil, errors.New("scanner runtime binding was not installed")
	}
	select {
	case <-ctx.Done():
		owner.Close()
		_ = owner.Wait(ctx)
		return nil, errors.New("scanner runtime binding installation deadline exceeded")
	case result := <-completion:
		if result.Code != engine.DispositionBindingInstalled {
			owner.Close()
			_ = owner.Wait(ctx)
			return nil, errors.New("scanner runtime binding was not installed")
		}
	}
	runtime.processLive.Store(true)
	if writer != nil {
		runtime.checkpointResultDone = make(chan struct{})
		go runtime.runCheckpointResults()
	}
	timerCtx, cancelTimer := context.WithCancel(context.Background())
	runtime.timerCancel, runtime.timerDone = cancelTimer, make(chan struct{})
	go runtime.runTimer(timerCtx)
	ingressSamplerCtx, cancelIngressSampler := context.WithCancel(context.Background())
	runtime.ingressSamplerCancel, runtime.ingressSamplerDone = cancelIngressSampler, make(chan struct{})
	go runtime.runIngressDiagnosticSampler(ingressSamplerCtx)
	return runtime, nil
}

func (r *Runtime) runCheckpointResults() {
	defer close(r.checkpointResultDone)
	for {
		result, err := r.writer.NextResult(context.Background())
		if err != nil {
			return
		}
		started := time.Now()
		admission, completion := r.engine.AdmitCheckpointTerminal(context.Background(), result)
		if admission != engine.AdmissionAdmitted || completion == nil {
			continue
		}
		<-completion
		r.recordDeliveryLatency(time.Since(started), DeliveryLatencyCheckpoint)
	}
}

func (r *Runtime) runTimer(ctx context.Context) {
	defer close(r.timerDone)
	evaluationTicker := time.NewTicker(r.config.SampleCadence)
	pressureTicker := time.NewTicker(time.Second)
	defer evaluationTicker.Stop()
	defer pressureTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-evaluationTicker.C:
			r.captureLiveCoverage(ctx)
			started := time.Now()
			admission, completion := r.engine.AdmitTimer(ctx)
			if admission != engine.AdmissionAdmitted || completion == nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case disposition := <-completion:
				r.recordDeliveryLatency(time.Since(started), DeliveryLatencyTimer)
				r.captureAutomaticTimerObservation(disposition)
				r.recordEngineDispositionIncident(disposition.Code, disposition.Reason)
				r.syncTQCommand(ctx)
			}
		case <-pressureTicker.C:
			admission, completion := r.engine.AdmitTQPressureTick(ctx)
			if admission != engine.AdmissionAdmitted || completion == nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-completion:
				r.syncTQPressure(ctx)
				r.syncTQCommand(ctx)
			}
		}
	}
}

func (r *Runtime) recordEngineTransitionIncident(result massive.EngineDeliveryResult) {
	switch {
	case result.AggregateDisposition.Code != "":
		r.recordEngineDispositionIncident(result.AggregateDisposition.Code, result.AggregateDisposition.Reason)
	case result.ControlDisposition.Code != "":
		r.recordEngineDispositionIncident(result.ControlDisposition.Code, result.ControlDisposition.Reason)
	case result.HydrationDisposition.Code != "":
		r.recordEngineDispositionIncident(result.HydrationDisposition.Code, result.HydrationDisposition.Reason)
	case result.LiveCoverageDisposition.Code != "":
		r.recordEngineDispositionIncident(result.LiveCoverageDisposition.Code, result.LiveCoverageDisposition.Reason)
	case result.TQDisposition.Code != "":
		r.recordEngineDispositionIncident(result.TQDisposition.Code, result.TQDisposition.Reason)
	}
}

func (r *Runtime) recordEngineDispositionIncident(code engine.DispositionCode, reason engine.DispositionReason) {
	if r == nil || !engineTerminalDisposition(code) {
		return
	}
	view := r.engine.ObserveOperational()
	metrics := r.Metrics()
	identities, count := ingressIdentityResults(metrics)
	history, historyCount := r.ingressHistory.snapshot()
	projection, projectionInvalid := lastCoherentProjection(r.engine.ObserveLastCoherentPublication())
	r.ingressIncident.set(IngressIncident{
		Owner: IngressOwnerEngineTransition, Source: "engine_transition", Reason: string(code), Binding: r.binding.Identity(),
		Epoch: view.Connection.Epoch, Lifecycle: view.Lifecycle, LifecycleReason: view.LifecycleReason, Suppression: view.Suppression,
		Hydration: view.Hydration, Queue: metrics.LiveQueue, QueueHighFrames: metrics.QueueHighFrames, QueueHighBytes: metrics.QueueHighBytes,
		MaxProcessingDelay: metrics.MaxProcessingDelay, MaxProcessingDelayOneSecond: metrics.MaxProcessingDelayOneSecond,
		HeapAllocBytes: metrics.HeapAllocBytes, HeapInUseBytes: metrics.HeapInUseBytes, Goroutines: metrics.Goroutines,
		Adapter: metrics.Adapter, PriorEngine: metrics.Engine, Engine: view, Identities: identities, IdentityCount: count,
		History: history, HistoryCount: historyCount, FenceTiming: r.engine.ObserveFenceTiming(),
		LastCoherentProjection: projection, LastCoherentProjectionInvalid: projectionInvalid,
		CapturedAt: metrics.SampledAt, EngineCapturedAt: r.clock().UTC(),
	})
	_ = reason
}

func engineTerminalDisposition(code engine.DispositionCode) bool {
	switch code {
	case engine.DispositionClockRegression, engine.DispositionAggregateIntegrity, engine.DispositionPublicationIntegrity,
		engine.DispositionAccountingIntegrity, engine.DispositionReplayFailed, engine.DispositionIngressIntegrity,
		engine.DispositionHydrationIntegrity, engine.DispositionRecoveryExhausted, engine.DispositionSequenceExhausted:
		return true
	default:
		return false
	}
}

func (r *Runtime) runIngressDiagnosticSampler(ctx context.Context) {
	defer close(r.ingressSamplerDone)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.captureIngressDiagnosticSample()
		}
	}
}

func (r *Runtime) setAutomaticTimerObserver(observer func(automaticTimerObservation)) {
	if r == nil {
		return
	}
	r.automaticTimerObserverMu.Lock()
	r.automaticTimerObserver = observer
	r.automaticTimerObserverMu.Unlock()
}

func (r *Runtime) captureAutomaticTimerObservation(disposition engine.TimerDisposition) {
	r.automaticTimerObserverMu.RLock()
	observer := r.automaticTimerObserver
	r.automaticTimerObserverMu.RUnlock()
	if observer == nil {
		return
	}
	capture, err := r.CaptureSnapshot()
	observer(automaticTimerObservation{
		disposition: disposition,
		timing:      r.engine.ObserveEvaluationTiming(),
		capture:     capture,
		captureErr:  err,
	})
}

func (r *Runtime) syncTQPressure(ctx context.Context) {
	metrics := r.metricsSnapshot()
	if !metrics.LiveQueue.Reconciles() || !metrics.Adapter.TransportReconciles() {
		failedIdentity := "adapter.transport"
		if !metrics.LiveQueue.Reconciles() {
			failedIdentity, _ = firstFailedIngressIdentity(metrics)
		}
		prior := r.engine.ObserveSnapshot().Publication
		admission, completion := r.engine.AdmitOperationalIngressIntegrity(ctx)
		if admission == engine.AdmissionAdmitted && completion != nil {
			select {
			case <-ctx.Done():
				return
			case <-completion:
			}
		}
		r.recordRuntimeAccountingIncident(failedIdentity, metrics, prior)
		return
	}
	if !metrics.Adapter.TQReconciles() {
		view := r.engine.ObserveOperational()
		input := engine.TQControlQuarantineInput{SchemaVersion: engine.TQSchemaV1, BindingIdentity: r.binding.Identity(), ConnectionEpoch: view.Connection.Epoch,
			ReceiptTime: r.clock().UTC(), Failure: engine.TQControlAccounting}
		admission, completion := r.engine.AdmitTQControlQuarantine(ctx, input)
		if admission == engine.AdmissionAdmitted && completion != nil {
			select {
			case <-ctx.Done():
			case <-completion:
			}
		}
		return
	}
	command, err := r.engine.IssueTQPressureCommand()
	if err != nil {
		return
	}
	sample := r.pressureSampler(metrics)
	tq := r.engine.ObserveTQ()
	sample.TQWorkPresent = sample.TQWorkPresent || len(tq.Desired) > 0 || tq.Accounting.KnownPresent > 0 || tq.Accounting.Unknown > 0
	input, err := engine.NewTQPressureResultInput(command, sample)
	if err != nil {
		return
	}
	admission, completion := r.engine.AdmitTQPressureResult(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	case disposition := <-completion:
		if disposition.Code == engine.DispositionTQApplied {
			r.deliveryWindowMu.Lock()
			if r.deliveryWindowVersion == metrics.deliveryWindowVersion {
				r.deliveryOneSecondMaxNanos = 0
				r.deliveryOneSecondMaxFamily = DeliveryLatencyUnknown
				r.deliveryWindowNonempty = false
			}
			r.deliveryWindowMu.Unlock()
		}
	}
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	if attempt != nil {
		attempt.SetTQShedding(r.engine.ObserveTQ().ShedTradesQuotes)
	}
}

func (r *Runtime) recordRuntimeAccountingIncident(reason string, metrics Metrics, priorPublications ...engine.ReplayPublicationView) {
	priorPublication := engine.ReplayPublicationView{}
	if len(priorPublications) > 0 {
		priorPublication = priorPublications[0]
	}
	view := r.engine.ObserveOperational()
	identities, count := ingressIdentityResults(metrics)
	history, historyCount := r.ingressHistory.snapshot()
	projection, projectionInvalid := lastCoherentProjection(priorPublication)
	r.ingressIncident.set(IngressIncident{
		Owner: IngressOwnerRuntimeAccounting, Source: "sync_tq_pressure_accounting_guard", Reason: reason,
		Binding: r.binding.Identity(), Epoch: view.Connection.Epoch, Lifecycle: view.Lifecycle,
		LifecycleReason: view.LifecycleReason, Suppression: view.Suppression, Hydration: metrics.Engine.Hydration,
		Queue: metrics.LiveQueue, QueueHighFrames: metrics.QueueHighFrames, QueueHighBytes: metrics.QueueHighBytes,
		MaxProcessingDelay: metrics.MaxProcessingDelay, MaxProcessingDelayOneSecond: metrics.MaxProcessingDelayOneSecond,
		HeapAllocBytes: metrics.HeapAllocBytes, HeapInUseBytes: metrics.HeapInUseBytes, Goroutines: metrics.Goroutines,
		Adapter: metrics.Adapter, PriorEngine: metrics.Engine, Engine: view, Identities: identities,
		IdentityCount: count, History: history, HistoryCount: historyCount,
		FenceTiming: r.engine.ObserveFenceTiming(), LastCoherentProjection: projection, LastCoherentProjectionInvalid: projectionInvalid,
		CapturedAt: metrics.SampledAt, EngineCapturedAt: r.clock().UTC(),
	})
}

func (r *Runtime) recordAdapterTerminal(result massive.EngineDeliveryResult) {
	if result.Terminal == nil {
		return
	}
	terminal := *result.Terminal
	if terminal.Reason == massive.TerminalCloseRequested || terminal.Reason == massive.TerminalContextCanceled {
		return
	}
	view := r.engine.ObserveOperational()
	processMetrics := r.Metrics()
	metrics := Metrics{SampledAt: terminal.CauseAccountingCapturedAt, LiveQueue: terminal.QueueAtCause, Adapter: terminal.AdapterAtCause, Engine: result.PriorEngine}
	metrics.Deliveries = r.deliveryCount.Load()
	r.deliveryWindowMu.Lock()
	metrics.MaxProcessingDelayOneSecond = time.Duration(r.deliveryOneSecondMaxNanos)
	r.deliveryWindowMu.Unlock()
	identities, count := ingressIdentityResults(metrics)
	history, historyCount := r.ingressHistory.snapshot()
	projection, projectionInvalid := lastCoherentProjection(result.PriorPublication)
	r.ingressIncident.set(IngressIncident{
		Owner: IngressOwnerAdapterTerminal, Source: string(terminal.Source), Reason: string(terminal.Reason),
		Binding: terminal.BindingIdentity, Epoch: terminal.ConnectionEpoch, Position: terminal.CausalPosition,
		PositionApplicable: terminal.PositionApplicable, ArrayIndexApplicable: terminal.ArrayIndexApplicable,
		Lifecycle: view.Lifecycle, LifecycleReason: view.LifecycleReason, Suppression: view.Suppression,
		Hydration: result.PriorEngine.Hydration, Queue: terminal.QueueAtCause, QueueHighFrames: terminal.QueueAtCause.HighFramesQueued, QueueHighBytes: terminal.QueueAtCause.HighQueuedBytes,
		IncomingFrameBytes: terminal.IncomingFrameBytes, ActiveDeliveryKind: terminal.ActiveDeliveryKind,
		ActiveDeliveryStartedAt: terminal.ActiveDeliveryStartedAt, ActiveDeliveryAgeAtCause: terminal.ActiveDeliveryAgeAtCause,
		MaxProcessingDelay: processMetrics.MaxProcessingDelay, MaxProcessingDelayOneSecond: processMetrics.MaxProcessingDelayOneSecond,
		HeapAllocBytes: processMetrics.HeapAllocBytes, HeapInUseBytes: processMetrics.HeapInUseBytes, Goroutines: processMetrics.Goroutines,
		Adapter: terminal.AdapterAtCause, PriorEngine: result.PriorEngine, Engine: view,
		Identities: identities, IdentityCount: count, History: history, HistoryCount: historyCount,
		FenceTiming: r.engine.ObserveFenceTiming(), LastCoherentProjection: projection, LastCoherentProjectionInvalid: projectionInvalid,
		CapturedAt: terminal.CauseAccountingCapturedAt, EngineCapturedAt: r.clock().UTC(),
	})
}

func (r *Runtime) captureIngressDiagnosticSample() {
	if r == nil {
		return
	}
	metrics := r.Metrics()
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	active := massive.ActiveDeliveryDiagnostic{}
	if attempt != nil {
		active = attempt.ActiveDeliveryDiagnostic()
	}
	r.ingressHistory.add(ingressDiagnosticSampleFromMetrics(metrics, active))
}

func ingressDiagnosticSampleFromMetrics(metrics Metrics, active massive.ActiveDeliveryDiagnostic) IngressDiagnosticSample {
	q, h := metrics.LiveQueue, metrics.Engine.Hydration.Accounting
	return IngressDiagnosticSample{
		CapturedAt: metrics.SampledAt, FramesRead: q.FramesRead, FramesAdmitted: q.FramesAdmitted,
		FramesDispositioned: q.FramesDispositioned, FramesFenced: q.FramesFenced,
		FramesRejectedSlot: q.FramesRejectedSlotCapacity, FramesRejectedByte: q.FramesRejectedByteCapacity,
		QueuedFrames: q.FramesQueued, ClassifyingFrames: q.FramesClassifying, HighFrames: q.HighFramesQueued,
		QueuedBytes: q.QueuedBytes, HighBytes: q.HighQueuedBytes, CapacityFrames: q.CapacityFrames, CapacityBytes: q.CapacityBytes,
		OldestWaitingFrameAge: q.OldestWaitingFrameAge, ActiveDeliveryKind: active.Kind, ActiveDeliveryAge: active.Age,
		MaxDeliveryDelayOneSecond: metrics.MaxProcessingDelayOneSecond, Deliveries: metrics.Deliveries,
		Lifecycle: metrics.Engine.Lifecycle, HydrationPlanned: h.Planned, HydrationOpen: h.Open,
	}
}

func (r *Runtime) FirstIngressIncident() *IngressIncident {
	if r == nil {
		return nil
	}
	if current := r.ingressIncident.get(); current != nil {
		return current
	}
	// DeliverToEngine suppresses before observeDelivery can latch the adapter's
	// exact immutable terminal. Do not let a concurrent API snapshot replace
	// that transport first cause with the engine consequence during this narrow
	// interval; a later call supplies the direct-engine fallback if needed.
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	if attempt != nil && attempt.ActiveDeliveryDiagnostic().Kind != "" {
		return nil
	}
	view := r.engine.ObserveOperational()
	if view.Lifecycle == "suppressed" {
		publication := r.engine.ObserveSnapshot().Publication
		r.recordEngineDispositionIncident(publication.LastDisposition, publication.DispositionReason)
	}
	return r.ingressIncident.get()
}

func defaultTQPressureSample(metrics Metrics) engine.TQPressureSample {
	frameCapacity, byteCapacity := uint64(0), uint64(0)
	if metrics.LiveQueue.CapacityFrames > 0 {
		frameCapacity = uint64(metrics.LiveQueue.CapacityFrames)
	}
	if metrics.LiveQueue.CapacityBytes > 0 {
		byteCapacity = uint64(metrics.LiveQueue.CapacityBytes)
	}
	attribution := metrics.DeliveryLatencyAttribution
	attributed := attribution.MaximumFamily != DeliveryLatencyUnknown &&
		attribution.MaximumDuration == metrics.MaxProcessingDelayOneSecond &&
		attribution.Reconciles(metrics.Deliveries)
	return engine.TQPressureSample{
		WaitingFrames: uint64(metrics.LiveQueue.FramesQueued), FrameCapacity: frameCapacity,
		WaitingBytes: uint64(metrics.LiveQueue.QueuedBytes), ByteCapacity: byteCapacity,
		OldestWaitingFrameAge: metrics.LiveQueue.OldestWaitingFrameAge, ActiveFrameAge: metrics.LiveQueue.ActiveFrameAge,
		SlotCapacityDrops: metrics.LiveQueue.FramesRejectedSlotCapacity, ByteCapacityDrops: metrics.LiveQueue.FramesRejectedByteCapacity,
		AggregateWatermarkLag:  metrics.WatermarkLag,
		MaxDeliveryDelayOneSec: metrics.MaxProcessingDelayOneSecond, DeliveryLatencyAttributed: attributed, HeapAllocBytes: metrics.HeapAllocBytes,
		Goroutines: metrics.Goroutines, TQLocalAccountingHealthy: metrics.LiveQueue.Reconciles() && metrics.Adapter.TransportReconciles() && metrics.Adapter.TQReconciles() && metrics.TQNormalization.Reconciles(),
	}
}

func (r *Runtime) syncTQCommand(ctx context.Context) {
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	if attempt == nil {
		return
	}
	command, err := r.engine.IssueTQCommand()
	if err != nil {
		return
	}
	adapterCommand, err := massive.ChangeTQCommandFromEngine(command)
	if err != nil {
		return
	}
	delivery, writeErr := attempt.ChangeTQ(ctx, adapterCommand)
	if delivery.Kind != "" {
		started := time.Now()
		result, _ := massive.DeliverToEngine(ctx, r.engine, delivery)
		r.observeDelivery(started, result)
	} else if writeErr != nil {
		input, inputErr := engine.NewTQCommandResultInput(command, engine.LivePosition{}, r.clock().UTC(), engine.ControlFailed)
		if inputErr == nil {
			admission, completion := r.engine.AdmitTQCommandResult(ctx, input)
			if admission == engine.AdmissionAdmitted && completion != nil {
				select {
				case <-ctx.Done():
				case <-completion:
				}
			}
		}
	}
}

func (r *Runtime) captureLiveCoverage(ctx context.Context) {
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	if attempt == nil {
		return
	}
	command, err := r.engine.IssueLiveCoverageFence()
	if err != nil {
		return
	}
	if err := attempt.CaptureLiveCoverageFence(ctx, r.engine, command); err != nil {
		admission, completion := r.engine.AdmitLiveCoverageFenceCancellation(ctx, command)
		if admission == engine.AdmissionAdmitted && completion != nil {
			select {
			case <-ctx.Done():
			case <-completion:
			}
		}
		return
	}
	_, _ = command.Wait(ctx)
}

func (r *Runtime) Engine() *engine.Engine { return r.engine }

func (r *Runtime) Status() Status {
	if r == nil || r.engine == nil {
		return Status{Reason: ReasonRuntimeUnavailable}
	}
	return deriveStatus(r.processLive.Load() && !r.joined.Load(), r.binding, r.config, r.clock().UTC(), r.engine.ObserveOperational())
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.engine == nil || ctx == nil {
		return errors.New("invalid scanner runtime shutdown")
	}
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	if r.joined.Load() {
		return nil
	}
	deadline, cancel := context.WithTimeout(ctx, r.config.ShutdownDeadline)
	defer cancel()
	r.liveMu.Lock()
	r.processLive.Store(false)
	liveCancel, liveDone := r.liveCancel, r.liveDone
	if liveCancel != nil {
		liveCancel()
	}
	r.liveMu.Unlock()
	if liveDone != nil {
		select {
		case <-liveDone:
		case <-deadline.Done():
			return errors.New("live composition shutdown deadline exceeded")
		}
	}
	r.metricsMu.Lock()
	attempt := r.attempt
	r.metricsMu.Unlock()
	if attempt != nil {
		if err := attempt.Wait(deadline); err != nil {
			return errors.New("live adapter cleanup deadline exceeded")
		}
	}
	if r.ingressSamplerCancel != nil {
		r.ingressSamplerCancel()
	}
	if r.ingressSamplerDone != nil {
		select {
		case <-r.ingressSamplerDone:
		case <-deadline.Done():
			return errors.New("ingress diagnostic sampler shutdown deadline exceeded")
		}
	}
	if r.timerCancel != nil {
		r.timerCancel()
	}
	if r.timerDone != nil {
		select {
		case <-r.timerDone:
		case <-deadline.Done():
			return errors.New("scanner timer shutdown deadline exceeded")
		}
	}
	if r.writer != nil {
		r.writer.Close()
		if err := r.writer.Wait(deadline); err != nil {
			return errors.New("checkpoint writer shutdown deadline exceeded")
		}
		select {
		case <-r.checkpointResultDone:
		case <-deadline.Done():
			return errors.New("checkpoint result join deadline exceeded")
		}
	}
	r.engine.Close()
	if err := r.engine.Wait(deadline); err != nil {
		return errors.New("scanner runtime shutdown deadline exceeded")
	}
	r.joined.Store(true)
	return nil
}

func (r *Runtime) beginLive(ctx context.Context) (context.Context, func(), error) {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	if !r.processLive.Load() || r.liveDone != nil {
		return nil, nil, errors.New("live runtime composition already started or stopped")
	}
	lifetime, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	r.liveCancel, r.liveDone, r.liveRunning = cancel, done, true
	finish := func() {
		cancel()
		r.liveMu.Lock()
		if r.liveRunning {
			r.liveRunning = false
			close(done)
		}
		r.liveMu.Unlock()
	}
	return lifetime, finish, nil
}
