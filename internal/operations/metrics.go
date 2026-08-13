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
	CheckpointEngine            engine.CheckpointOperations
	QueueCurrentFrames          uint64
	QueueHighFrames             uint64
	QueueCurrentBytes           int
	QueueHighBytes              int
	Deliveries                  uint64
	ConsumerDeferred            uint64
	MeanProcessingDelay         time.Duration
	MaxProcessingDelay          time.Duration
	MaxProcessingDelayOneSecond time.Duration
	DeliveryLatencyAttribution  DeliveryLatencyAttribution
	WatermarkLag                time.Duration
	HeapAllocBytes              uint64
	HeapInUseBytes              uint64
	Goroutines                  int
	AccountingValid             bool
	deliveryWindowVersion       uint64
}

// DeliveryLatencyFamily is the closed, symbol-free attribution vocabulary for
// one engine-delivery latency observation. Unknown is fail-closed: it includes
// absent evidence and results that claim more than one work family.
type DeliveryLatencyFamily string

const (
	DeliveryLatencyAggregate      DeliveryLatencyFamily = "aggregate"
	DeliveryLatencyTQ             DeliveryLatencyFamily = "tq"
	DeliveryLatencyControl        DeliveryLatencyFamily = "control"
	DeliveryLatencyHydrationFence DeliveryLatencyFamily = "hydration_fence"
	DeliveryLatencyCheckpoint     DeliveryLatencyFamily = "checkpoint"
	DeliveryLatencyTimer          DeliveryLatencyFamily = "timer"
	DeliveryLatencyUnknown        DeliveryLatencyFamily = "unknown"
	deliveryLatencyFamilyCount                          = 7
)

// DeliveryLatencyAttribution is fixed-cardinality cumulative accounting plus
// the family paired with the current one-second maximum. MaximumDuration is
// copied from the same locked record as MaximumFamily.
type DeliveryLatencyAttribution struct {
	Aggregate, TQ, Control, HydrationFence uint64
	Checkpoint, Timer, Unknown             uint64
	MaximumDuration                        time.Duration
	MaximumFamily                          DeliveryLatencyFamily
}

func (a DeliveryLatencyAttribution) Total() uint64 {
	return a.Aggregate + a.TQ + a.Control + a.HydrationFence + a.Checkpoint + a.Timer + a.Unknown
}

func (a DeliveryLatencyAttribution) Reconciles(deliveries uint64) bool {
	if a.Total() != deliveries || a.MaximumDuration < 0 || deliveryLatencyFamilyIndex(a.MaximumFamily) < 0 {
		return false
	}
	if deliveries == 0 {
		return a.MaximumDuration == 0 && a.MaximumFamily == DeliveryLatencyUnknown
	}
	return a.count(a.MaximumFamily) != 0
}

func (a DeliveryLatencyAttribution) count(family DeliveryLatencyFamily) uint64 {
	switch family {
	case DeliveryLatencyAggregate:
		return a.Aggregate
	case DeliveryLatencyTQ:
		return a.TQ
	case DeliveryLatencyControl:
		return a.Control
	case DeliveryLatencyHydrationFence:
		return a.HydrationFence
	case DeliveryLatencyCheckpoint:
		return a.Checkpoint
	case DeliveryLatencyTimer:
		return a.Timer
	case DeliveryLatencyUnknown:
		return a.Unknown
	default:
		return 0
	}
}

func (r *Runtime) observeDelivery(started time.Time, result massive.EngineDeliveryResult) {
	delay := time.Since(started)
	if delay < 0 {
		delay = 0
	}
	r.recordDeliveryLatency(delay, deliveryLatencyFamily(result))
	if result.ConsumerDeferred {
		r.consumerDeferred.Add(1)
	}
	r.recordAdapterTerminal(result)
	if result.Terminal != nil {
		r.metricsMu.Lock()
		attempt := r.attempt
		r.metricsMu.Unlock()
		if attempt != nil {
			attempt.AcknowledgeTerminalObservation()
		}
	}
	if result.Terminal == nil {
		r.recordEngineTransitionIncident(result)
	}
}

func (r *Runtime) recordDeliveryLatency(delay time.Duration, family DeliveryLatencyFamily) {
	if delay < 0 {
		delay = 0
	}
	if deliveryLatencyFamilyIndex(family) < 0 {
		family = DeliveryLatencyUnknown
	}
	nanos := uint64(delay)
	r.deliveryWindowMu.Lock()
	r.deliveryCount.Add(1)
	r.deliveryTotalNanos.Add(nanos)
	for old := r.deliveryMaxNanos.Load(); nanos > old && !r.deliveryMaxNanos.CompareAndSwap(old, nanos); old = r.deliveryMaxNanos.Load() {
	}
	r.deliveryFamilyCounts[deliveryLatencyFamilyIndex(family)]++
	if nanos > r.deliveryOneSecondMaxNanos || nanos == r.deliveryOneSecondMaxNanos && deliveryLatencyFamilyPriority(family) < deliveryLatencyFamilyPriority(r.deliveryOneSecondMaxFamily) {
		r.deliveryOneSecondMaxNanos, r.deliveryOneSecondMaxFamily = nanos, family
	}
	r.deliveryWindowVersion++
	r.deliveryWindowMu.Unlock()
}

func deliveryLatencyFamily(result massive.EngineDeliveryResult) DeliveryLatencyFamily {
	var family DeliveryLatencyFamily
	claimed := 0
	claim := func(candidate DeliveryLatencyFamily, present bool) {
		if !present {
			return
		}
		claimed++
		family = candidate
	}
	claim(DeliveryLatencyAggregate, result.AggregateDisposition.Code != "")
	claim(DeliveryLatencyTQ, result.TQDisposition.Code != "")
	claim(DeliveryLatencyControl, result.ControlDisposition.Code != "")
	claim(DeliveryLatencyHydrationFence, result.HydrationDisposition.Code != "" || result.LiveCoverageDisposition.Code != "")
	if claimed != 1 || result.ConsumerDeferred && family != DeliveryLatencyTQ {
		return DeliveryLatencyUnknown
	}
	return family
}

func deliveryLatencyFamilyIndex(family DeliveryLatencyFamily) int {
	switch family {
	case DeliveryLatencyAggregate:
		return 0
	case DeliveryLatencyTQ:
		return 1
	case DeliveryLatencyControl:
		return 2
	case DeliveryLatencyHydrationFence:
		return 3
	case DeliveryLatencyCheckpoint:
		return 4
	case DeliveryLatencyTimer:
		return 5
	case DeliveryLatencyUnknown:
		return 6
	default:
		return -1
	}
}

func deliveryLatencyFamilyPriority(family DeliveryLatencyFamily) int {
	if index := deliveryLatencyFamilyIndex(family); index >= 0 {
		return index
	}
	return deliveryLatencyFamilyCount
}

func deliveryLatencyAttribution(counts [deliveryLatencyFamilyCount]uint64, maximum time.Duration, family DeliveryLatencyFamily) DeliveryLatencyAttribution {
	if deliveryLatencyFamilyIndex(family) < 0 {
		family = DeliveryLatencyUnknown
	}
	return DeliveryLatencyAttribution{
		Aggregate: counts[0], TQ: counts[1], Control: counts[2], HydrationFence: counts[3],
		Checkpoint: counts[4], Timer: counts[5], Unknown: counts[6], MaximumDuration: maximum, MaximumFamily: family,
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
	if result.LiveQueue.HighFramesQueued > r.queueHighFrames {
		r.queueHighFrames = result.LiveQueue.HighFramesQueued
	}
	if result.LiveQueue.QueuedBytes > r.queueHighBytes {
		r.queueHighBytes = result.LiveQueue.QueuedBytes
	}
	if result.LiveQueue.HighQueuedBytes > r.queueHighBytes {
		r.queueHighBytes = result.LiveQueue.HighQueuedBytes
	}
	result.QueueCurrentFrames, result.QueueHighFrames = currentFrames, r.queueHighFrames
	result.QueueCurrentBytes, result.QueueHighBytes = result.LiveQueue.QueuedBytes, r.queueHighBytes
	r.metricsMu.Unlock()
	if r.writer != nil {
		result.Checkpoint = r.writer.Accounting()
	}
	result.CheckpointEngine = r.engine.CheckpointOperations()
	r.deliveryWindowMu.Lock()
	result.Deliveries = r.deliveryCount.Load()
	if result.Deliveries != 0 {
		result.MeanProcessingDelay = time.Duration(r.deliveryTotalNanos.Load() / result.Deliveries)
	}
	result.MaxProcessingDelay = time.Duration(r.deliveryMaxNanos.Load())
	result.MaxProcessingDelayOneSecond = time.Duration(r.deliveryOneSecondMaxNanos)
	result.DeliveryLatencyAttribution = deliveryLatencyAttribution(r.deliveryFamilyCounts, result.MaxProcessingDelayOneSecond, r.deliveryOneSecondMaxFamily)
	result.deliveryWindowVersion = r.deliveryWindowVersion
	r.deliveryWindowMu.Unlock()
	result.ConsumerDeferred = r.consumerDeferred.Load()
	status := deriveStatus(processLive, r.binding, r.config, result.SampledAt, result.Engine)
	result.WatermarkLag = status.WatermarkLag
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	result.HeapAllocBytes, result.HeapInUseBytes = memory.HeapAlloc, memory.HeapInuse
	result.Goroutines = runtime.NumGoroutine()
	result.AccountingValid = operationalAccountingValid(result.Engine) && result.LiveQueue.Reconciles() && result.Adapter.Reconciles() && result.TQNormalization.Reconciles() && result.CheckpointEngine.Reconciles() &&
		(r.writer == nil || result.Checkpoint.Reconciles()) && result.DeliveryLatencyAttribution.Reconciles(result.Deliveries)
	return result
}
