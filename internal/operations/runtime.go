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
	ConnectionAttemptDeadline, ShutdownDeadline time.Duration
}

func DefaultConfig() Config {
	return Config{EngineCapacity: 8192, RequiredReserve: 128, RecoveryAttempts: 3, EvaluationDelay: 4 * time.Second, ReadinessTolerance: 2 * time.Second, SampleCadence: time.Second, ConnectionAttemptDeadline: 60 * time.Second, ShutdownDeadline: 10 * time.Second}
}

func (c Config) valid() bool {
	return c.EngineCapacity > 1 && c.RequiredReserve > 0 && c.RequiredReserve < c.EngineCapacity && c.RecoveryAttempts > 0 && c.RecoveryAttempts <= 10 &&
		c.EvaluationDelay >= 0 && c.ReadinessTolerance >= 0 && c.ReadinessTolerance <= 10*time.Second &&
		c.SampleCadence > 0 && c.SampleCadence <= 10*time.Minute &&
		c.ConnectionAttemptDeadline > 0 && c.ConnectionAttemptDeadline <= 2*time.Minute &&
		c.ShutdownDeadline > 0 && c.ShutdownDeadline <= time.Minute
}

type Runtime struct {
	engine                    *engine.Engine
	binding                   reference.Binding
	config                    Config
	clock                     func() time.Time
	processLive               atomic.Bool
	joined                    atomic.Bool
	writer                    *checkpoint.Writer
	metricsMu                 sync.Mutex
	liveMu                    sync.Mutex
	shutdownMu                sync.Mutex
	captureMu                 sync.Mutex
	captureSequence           uint64
	attempt                   *massive.LiveAttempt
	adapter                   *massive.LiveAdapter
	liveCancel                context.CancelFunc
	liveDone                  chan struct{}
	liveRunning               bool
	queueHighFrames           uint64
	queueHighBytes            int
	deliveryCount             atomic.Uint64
	deliveryTotalNanos        atomic.Uint64
	deliveryMaxNanos          atomic.Uint64
	deliveryWindowMu          sync.Mutex
	deliveryOneSecondMaxNanos uint64
	deliveryWindowVersion     uint64
	consumerDeferred          atomic.Uint64
	pressureSampler           func(Metrics) engine.TQPressureSample
	metricsSnapshot           func() Metrics
	timerCancel               context.CancelFunc
	timerDone                 chan struct{}
	automaticTimerObserverMu  sync.RWMutex
	automaticTimerObserver    func(automaticTimerObservation)
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
	owner, err := engine.New(engine.Config{Mode: engine.RunModeLive, Clock: clock, Capacity: config.EngineCapacity, RequiredReserve: config.RequiredReserve, EvaluationDelay: &config.EvaluationDelay, CheckpointSubmitter: writer})
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{engine: owner, binding: binding, config: config, clock: clock, writer: writer, pressureSampler: defaultTQPressureSample}
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
	timerCtx, cancelTimer := context.WithCancel(context.Background())
	runtime.timerCancel, runtime.timerDone = cancelTimer, make(chan struct{})
	go runtime.runTimer(timerCtx)
	return runtime, nil
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
				r.captureAutomaticTimerObservation(disposition)
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
	command, err := r.engine.IssueTQPressureCommand()
	if err != nil {
		return
	}
	metrics := r.metricsSnapshot()
	if !metrics.LiveQueue.Reconciles() || !metrics.Adapter.Reconciles() {
		admission, completion := r.engine.AdmitOperationalIngressIntegrity(ctx)
		if admission == engine.AdmissionAdmitted && completion != nil {
			select {
			case <-ctx.Done():
			case <-completion:
			}
		}
		return
	}
	sample := r.pressureSampler(metrics)
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

func defaultTQPressureSample(metrics Metrics) engine.TQPressureSample {
	capacity := uint64(0)
	if metrics.LiveQueue.CapacityFrames > 0 {
		capacity = uint64(metrics.LiveQueue.CapacityFrames)
	}
	return engine.TQPressureSample{
		QueueCurrentFrames:  metrics.QueueCurrentFrames,
		QueueCapacityFrames: capacity, OldestFrameAge: metrics.LiveQueue.OldestFrameAge,
		MaxDeliveryDelayOneSec: metrics.MaxProcessingDelayOneSecond, HeapAllocBytes: metrics.HeapAllocBytes,
		Goroutines: metrics.Goroutines, TQLocalAccountingHealthy: metrics.TQNormalization.Reconciles(),
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
	r.engine.Close()
	if r.writer != nil {
		r.writer.Close()
	}
	if err := r.engine.Wait(deadline); err != nil {
		return errors.New("scanner runtime shutdown deadline exceeded")
	}
	if r.writer != nil {
		if err := r.writer.Wait(deadline); err != nil {
			return errors.New("checkpoint writer shutdown deadline exceeded")
		}
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
