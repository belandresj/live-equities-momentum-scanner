package operations

import (
	"runtime"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

// Metrics is a fixed-cardinality process sample. Every field is either an
// engine-owned monotonic counter, a bounded component accounting family, or a
// scalar runtime observation; it contains no symbol/provider labels.
type Metrics struct {
	SampledAt                   time.Time
	Engine                      engine.OperationalView
	Adapter                     massive.AdapterAccounting
	LiveQueue                   massive.LiveQueueAccounting
	TQNormalization             massive.TQNormalizationAccounting
	Checkpoint                  checkpoint.WriterAccounting
	QueueCurrentFrames          uint64
	QueueHighFrames             uint64
	QueueCurrentBytes           int
	QueueHighBytes              int
	Deliveries                  uint64
	ConsumerDeferred            uint64
	MeanProcessingDelay         time.Duration
	MaxProcessingDelay          time.Duration
	MaxProcessingDelayOneSecond time.Duration
	WatermarkLag                time.Duration
	HeapAllocBytes              uint64
	HeapInUseBytes              uint64
	Goroutines                  int
	AccountingValid             bool
	deliveryWindowVersion       uint64
}

func (r *Runtime) observeDelivery(started time.Time, result massive.EngineDeliveryResult) {
	delay := time.Since(started)
	if delay < 0 {
		delay = 0
	}
	nanos := uint64(delay)
	r.deliveryCount.Add(1)
	r.deliveryTotalNanos.Add(nanos)
	for old := r.deliveryMaxNanos.Load(); nanos > old && !r.deliveryMaxNanos.CompareAndSwap(old, nanos); old = r.deliveryMaxNanos.Load() {
	}
	r.deliveryWindowMu.Lock()
	if nanos > r.deliveryOneSecondMaxNanos {
		r.deliveryOneSecondMaxNanos = nanos
	}
	r.deliveryWindowVersion++
	r.deliveryWindowMu.Unlock()
	if result.ConsumerDeferred {
		r.consumerDeferred.Add(1)
	}
}

func (r *Runtime) setLiveSources(attempt *massive.LiveAttempt, adapter *massive.LiveAdapter) {
	r.metricsMu.Lock()
	r.attempt, r.adapter = attempt, adapter
	r.metricsMu.Unlock()
}

func (r *Runtime) Metrics() Metrics {
	if r == nil || r.engine == nil {
		return Metrics{}
	}
	sampledAt := r.clock().UTC()
	processLive := r.processLive.Load() && !r.joined.Load()
	return r.metricsFromPublication(sampledAt, processLive, r.engine.ObserveOperational())
}

func (r *Runtime) metricsFromPublication(sampledAt time.Time, processLive bool, view engine.OperationalView) Metrics {
	result := Metrics{SampledAt: sampledAt, Engine: view}
	r.metricsMu.Lock()
	if r.attempt != nil {
		result.LiveQueue = r.attempt.QueueAccounting()
		result.TQNormalization = r.attempt.TQNormalizationAccounting()
	}
	if r.adapter != nil {
		result.Adapter = r.adapter.Accounting()
	}
	currentFrames := result.LiveQueue.FramesQueued + result.LiveQueue.FramesClassifying + result.LiveQueue.IngressFencesQueued + result.LiveQueue.IngressFencesClassifying + result.LiveQueue.TerminalMarkersQueued + result.LiveQueue.TerminalMarkersClassifying
	if currentFrames > r.queueHighFrames {
		r.queueHighFrames = currentFrames
	}
	if result.LiveQueue.QueuedBytes > r.queueHighBytes {
		r.queueHighBytes = result.LiveQueue.QueuedBytes
	}
	result.QueueCurrentFrames, result.QueueHighFrames = currentFrames, r.queueHighFrames
	result.QueueCurrentBytes, result.QueueHighBytes = result.LiveQueue.QueuedBytes, r.queueHighBytes
	r.metricsMu.Unlock()
	if r.writer != nil {
		result.Checkpoint = r.writer.Accounting()
	}
	result.Deliveries = r.deliveryCount.Load()
	result.ConsumerDeferred = r.consumerDeferred.Load()
	if result.Deliveries != 0 {
		result.MeanProcessingDelay = time.Duration(r.deliveryTotalNanos.Load() / result.Deliveries)
	}
	result.MaxProcessingDelay = time.Duration(r.deliveryMaxNanos.Load())
	r.deliveryWindowMu.Lock()
	result.MaxProcessingDelayOneSecond = time.Duration(r.deliveryOneSecondMaxNanos)
	result.deliveryWindowVersion = r.deliveryWindowVersion
	r.deliveryWindowMu.Unlock()
	status := deriveStatus(processLive, r.binding, r.config, result.SampledAt, result.Engine)
	result.WatermarkLag = status.WatermarkLag
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	result.HeapAllocBytes, result.HeapInUseBytes = memory.HeapAlloc, memory.HeapInuse
	result.Goroutines = runtime.NumGoroutine()
	result.AccountingValid = operationalAccountingValid(result.Engine) && result.LiveQueue.Reconciles() && result.Adapter.Reconciles() && result.TQNormalization.Reconciles() && (r.writer == nil || result.Checkpoint.Reconciles())
	return result
}
