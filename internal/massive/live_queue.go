package massive

import (
	"context"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	MaximumLiveFrameSlots  = 4096
	MaximumLiveQueueBytes  = 64 << 20
	liveMarkerReserveSlots = 8
	liveMarkerReserveBytes = 64 << 10
)

type LiveQueueConfig struct {
	FrameSlots      int
	MaxFrameBytes   int
	TotalFrameBytes int
}

type FrameAdmissionReason string

const (
	FrameAdmitted             FrameAdmissionReason = "admitted"
	FrameRejectedOversize     FrameAdmissionReason = "rejected_oversize"
	FrameRejectedSlotCapacity FrameAdmissionReason = "rejected_slot_capacity"
	FrameRejectedByteCapacity FrameAdmissionReason = "rejected_byte_capacity"
	FrameRejectedReceipt      FrameAdmissionReason = "rejected_receipt"
	FrameRejectedGate         FrameAdmissionReason = "rejected_gate_or_close"
	FrameShedTQCapacity       FrameAdmissionReason = "shed_tq_only_capacity"
)

type LiveQueueAccounting struct {
	FramesRead, FramesAdmitted, FramesShedTQCapacity                                                                uint64
	FramesRejectedOversize, FramesRejectedCapacity, FramesRejectedReceipt                                           uint64
	FramesRejectedSlotCapacity, FramesRejectedByteCapacity                                                          uint64
	FramesRejectedGateOrClose                                                                                       uint64
	FramesDecoding, FramesQueued, FramesClassifying, FramesDispositioned, FramesFenced                              uint64
	QueuedBytes                                                                                                     int
	HighFramesQueued                                                                                                uint64
	HighQueuedBytes                                                                                                 int
	TerminalMarkersQueued, TerminalMarkersClassifying, TerminalMarkersDispositioned                                 uint64
	IngressFencesStarted                                                                                            uint64
	IngressFencesQueued, IngressFencesClassifying, IngressFencesDispositioned                                       uint64
	TQCapacityMarkersStarted, TQCapacityMarkersQueued, TQCapacityMarkersClassifying, TQCapacityMarkersDispositioned uint64
	TQCapacityShedTrades, TQCapacityShedQuotes                                                                      uint64
	CapacityFrames, CapacityBytes                                                                                   int
	OldestWaitingFrameAge, ActiveFrameAge                                                                           time.Duration
}

func (a LiveQueueAccounting) Reconciles() bool {
	return a.FramesRead == a.FramesDecoding+a.FramesAdmitted+a.FramesShedTQCapacity+a.FramesRejectedOversize+
		a.FramesRejectedCapacity+a.FramesRejectedReceipt+a.FramesRejectedGateOrClose &&
		a.FramesRejectedCapacity == a.FramesRejectedSlotCapacity+a.FramesRejectedByteCapacity &&
		a.FramesAdmitted == a.FramesQueued+a.FramesClassifying+a.FramesDispositioned+a.FramesFenced &&
		a.TerminalMarkersQueued+a.TerminalMarkersClassifying+a.TerminalMarkersDispositioned <= 1 &&
		a.IngressFencesStarted == a.IngressFencesQueued+a.IngressFencesClassifying+a.IngressFencesDispositioned &&
		a.IngressFencesQueued+a.IngressFencesClassifying <= 1 &&
		a.TQCapacityMarkersStarted == a.TQCapacityMarkersQueued+a.TQCapacityMarkersClassifying+a.TQCapacityMarkersDispositioned &&
		a.TQCapacityMarkersQueued+a.TQCapacityMarkersClassifying <= 1
}

type queuedLiveKind uint8

const (
	queuedLiveDecodedBatch queuedLiveKind = iota + 1
	queuedLiveTerminal
	queuedLiveIngressFence
	queuedLiveCoverageFence
	queuedLiveTQCapacity
)

type socketMessageType uint8

const (
	socketMessageText socketMessageType = iota + 1
	socketMessageBinary
)

type queuedLiveFrame struct {
	epoch, sequence    uint64
	receivedAt         time.Time
	batch              DecodedBatch
	terminal           bool
	kind               queuedLiveKind
	ingressFence       AggregateIngressFenceFact
	liveCoverageFence  LiveCoverageFenceFact
	tqCapacityPosition engine.LivePosition
}

type liveFrameQueue struct {
	mu                          sync.Mutex
	changed                     chan struct{}
	recheck                     chan struct{}
	config                      LiveQueueConfig
	binding                     reference.Binding
	frames                      []queuedLiveFrame
	head, count                 int
	bytes                       int
	next                        uint64
	lastReceipt                 time.Time
	classifyingAt               time.Time
	greatestRaw                 uint64
	greatestAdmitted            uint64
	decoding                    bool
	tqCapacityMarkerOutstanding bool
	nextFenceMarker             uint64
	now                         func() time.Time
	gateOpen                    bool
	accounting                  LiveQueueAccounting
}

func validateLiveQueueConfig(config LiveQueueConfig) bool {
	return config.FrameSlots >= 1 && config.FrameSlots <= MaximumLiveFrameSlots &&
		config.MaxFrameBytes >= 1 && config.MaxFrameBytes <= MaximumLiveFrameBytes &&
		config.TotalFrameBytes >= config.MaxFrameBytes && config.TotalFrameBytes <= MaximumLiveQueueBytes
}

func newLiveFrameQueue(config LiveQueueConfig, binding ...reference.Binding) *liveFrameQueue {
	ringEntries := config.FrameSlots + liveMarkerReserveSlots
	if config.FrameSlots == MaximumLiveFrameSlots {
		ringEntries = MaximumLiveFrameSlots
	}
	q := &liveFrameQueue{config: config, frames: make([]queuedLiveFrame, ringEntries), changed: make(chan struct{}), recheck: make(chan struct{}), gateOpen: true, next: 1, nextFenceMarker: 1, now: func() time.Time { return time.Now().UTC() }}
	if len(binding) == 1 {
		q.binding = binding[0]
	}
	return q
}

func (q *liveFrameQueue) decodedSlotCapacity() int {
	if q.config.FrameSlots == MaximumLiveFrameSlots {
		return q.config.FrameSlots - liveMarkerReserveSlots
	}
	return q.config.FrameSlots
}

func (q *liveFrameQueue) decodedByteCapacity() int {
	if q.config.TotalFrameBytes == MaximumLiveQueueBytes {
		return q.config.TotalFrameBytes - liveMarkerReserveBytes
	}
	return q.config.TotalFrameBytes
}

func (q *liveFrameQueue) appendLocked(frame queuedLiveFrame) {
	if q.count >= len(q.frames) {
		panic("live frame queue internal capacity exceeded")
	}
	index := (q.head + q.count) % len(q.frames)
	q.frames[index] = frame
	q.count++
}

func (q *liveFrameQueue) frameLocked(offset int) queuedLiveFrame {
	return q.frames[(q.head+offset)%len(q.frames)]
}

func (q *liveFrameQueue) notifyLocked() {
	close(q.changed)
	q.changed = make(chan struct{})
}

// requestRecheck wakes a delivery loop without fabricating or accounting a
// provider frame. The caller uses it when command state changes while the
// loop may already be blocked on an empty queue.
func (q *liveFrameQueue) requestRecheck() {
	q.mu.Lock()
	close(q.recheck)
	q.recheck = make(chan struct{})
	q.mu.Unlock()
}

func (q *liveFrameQueue) beginDecode(epoch uint64, messageType socketMessageType, receivedAt time.Time, encodedBytes int) (uint64, FrameAdmissionReason, LiveQueueAccounting) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.accounting.FramesRead++
	if !q.gateOpen || q.decoding || epoch == 0 || q.next == 0 {
		q.accounting.FramesRejectedGateOrClose++
		return 0, FrameRejectedGate, q.admissionSnapshotLocked()
	}
	if messageType != socketMessageText && messageType != socketMessageBinary {
		q.accounting.FramesRejectedGateOrClose++
		return 0, FrameRejectedGate, q.admissionSnapshotLocked()
	}
	if encodedBytes > q.config.MaxFrameBytes || encodedBytes < 0 {
		q.accounting.FramesRejectedOversize++
		return 0, FrameRejectedOversize, q.admissionSnapshotLocked()
	}
	if receivedAt.IsZero() || receivedAt != receivedAt.UTC() || (!q.lastReceipt.IsZero() && receivedAt.Before(q.lastReceipt)) {
		q.accounting.FramesRejectedReceipt++
		return 0, FrameRejectedReceipt, q.admissionSnapshotLocked()
	}
	sequence := q.next
	q.next++
	q.greatestRaw = sequence
	q.lastReceipt = receivedAt
	q.decoding = true
	q.accounting.FramesDecoding++
	q.classifyingAt = receivedAt
	return sequence, FrameAdmitted, LiveQueueAccounting{}
}

func (q *liveFrameQueue) tryEnqueueDecoded(epoch uint64, sequence uint64, receivedAt time.Time, encodedBytes int, batch DecodedBatch) (queuedLiveFrame, FrameAdmissionReason, LiveQueueAccounting) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.decoding || sequence == 0 || sequence != q.greatestRaw || !q.gateOpen {
		if q.decoding {
			q.decoding = false
			q.classifyingAt = time.Time{}
			q.accounting.FramesDecoding--
			q.accounting.FramesRejectedGateOrClose++
		}
		q.notifyLocked()
		return queuedLiveFrame{}, FrameRejectedGate, q.admissionSnapshotLocked()
	}
	q.decoding = false
	q.accounting.FramesDecoding--
	q.classifyingAt = time.Time{}
	if batch.EncodedBytes != encodedBytes || batch.ConnectionEpoch != epoch || batch.FrameSequence != sequence || batch.ReceivedAt != receivedAt || batch.RetainedCharge < 0 || batch.RetainedCharge > MaximumDecodedBatchCharge || batch.Len() > MaximumDecodedBatchElements {
		q.accounting.FramesRejectedOversize++
		q.notifyLocked()
		return queuedLiveFrame{}, FrameRejectedOversize, q.admissionSnapshotLocked()
	}
	if q.accounting.FramesQueued >= uint64(q.decodedSlotCapacity()) {
		if q.shedTQOnlyLocked(epoch, sequence, receivedAt, batch) {
			return queuedLiveFrame{}, FrameShedTQCapacity, LiveQueueAccounting{}
		}
		q.accounting.FramesRejectedCapacity++
		q.accounting.FramesRejectedSlotCapacity++
		q.notifyLocked()
		return queuedLiveFrame{}, FrameRejectedSlotCapacity, q.admissionSnapshotLocked()
	}
	if batch.RetainedCharge > q.decodedByteCapacity()-q.bytes {
		if q.shedTQOnlyLocked(epoch, sequence, receivedAt, batch) {
			return queuedLiveFrame{}, FrameShedTQCapacity, LiveQueueAccounting{}
		}
		q.accounting.FramesRejectedCapacity++
		q.accounting.FramesRejectedByteCapacity++
		q.notifyLocked()
		return queuedLiveFrame{}, FrameRejectedByteCapacity, q.admissionSnapshotLocked()
	}
	frame := queuedLiveFrame{epoch: epoch, sequence: sequence, receivedAt: receivedAt, batch: batch, kind: queuedLiveDecodedBatch}
	q.appendLocked(frame)
	q.bytes += frame.batch.RetainedCharge
	q.accounting.FramesAdmitted++
	q.greatestAdmitted = sequence
	q.accounting.FramesQueued++
	q.accounting.QueuedBytes = q.bytes
	if q.accounting.FramesQueued > q.accounting.HighFramesQueued {
		q.accounting.HighFramesQueued = q.accounting.FramesQueued
	}
	if q.bytes > q.accounting.HighQueuedBytes {
		q.accounting.HighQueuedBytes = q.bytes
	}
	q.notifyLocked()
	// The third result is a causal rejection snapshot. Successful callers own
	// the admitted frame and do not need another clock-bearing queue sample.
	return frame, FrameAdmitted, LiveQueueAccounting{}
}

func (q *liveFrameQueue) shedTQOnlyLocked(epoch, sequence uint64, receivedAt time.Time, batch DecodedBatch) bool {
	trades, quotes := uint64(0), uint64(0)
	for _, result := range batch.results {
		switch result.Kind {
		case LiveResultTrade:
			trades++
		case LiveResultQuote:
			quotes++
		case LiveResultRejected:
			if result.Rejection.Family == LiveFamilyTrade {
				trades++
			} else if result.Rejection.Family == LiveFamilyQuote {
				quotes++
			} else {
				return false
			}
		default:
			return false
		}
	}
	if trades+quotes == 0 || q.count >= len(q.frames) {
		return false
	}
	q.accounting.FramesShedTQCapacity++
	q.accounting.TQCapacityShedTrades += trades
	q.accounting.TQCapacityShedQuotes += quotes
	if !q.tqCapacityMarkerOutstanding {
		q.appendLocked(queuedLiveFrame{epoch: epoch, sequence: sequence, receivedAt: receivedAt, kind: queuedLiveTQCapacity,
			tqCapacityPosition: engine.LivePosition{ConnectionEpoch: epoch, FrameSequence: sequence}})
		q.tqCapacityMarkerOutstanding = true
		q.accounting.TQCapacityMarkersStarted++
		q.accounting.TQCapacityMarkersQueued = 1
	}
	q.notifyLocked()
	return true
}

func (q *liveFrameQueue) admissionSnapshotLocked() LiveQueueAccounting {
	result := q.accounting
	result.CapacityFrames, result.CapacityBytes = q.config.FrameSlots, q.config.TotalFrameBytes
	for offset := 0; offset < q.count; offset++ {
		frame := q.frameLocked(offset)
		if frame.kind == queuedLiveDecodedBatch {
			// Raw receipts are monotonic and the queue is FIFO. The first raw
			// frame is therefore the oldest; only bounded fence markers can
			// precede it. Do not turn a pressure snapshot into an O(queue) lock.
			result.OldestWaitingFrameAge = q.now().Sub(frame.receivedAt)
			if result.OldestWaitingFrameAge < 0 {
				result.OldestWaitingFrameAge = 0
			}
			break
		}
	}
	if !q.classifyingAt.IsZero() {
		result.ActiveFrameAge = q.now().Sub(q.classifyingAt)
		if result.ActiveFrameAge < 0 {
			result.ActiveFrameAge = 0
		}
	}
	return result
}

func (q *liveFrameQueue) enqueueIngressFence(ctx context.Context, fact AggregateIngressFenceFact) (AggregateIngressFenceFact, bool) {
	for {
		if ctx == nil || ctx.Err() != nil {
			return AggregateIngressFenceFact{}, false
		}
		q.mu.Lock()
		if q.accounting.IngressFencesQueued+q.accounting.IngressFencesClassifying > 0 {
			q.mu.Unlock()
			return AggregateIngressFenceFact{}, false
		}
		if !q.gateOpen || q.nextFenceMarker == 0 {
			q.mu.Unlock()
			return AggregateIngressFenceFact{}, false
		}
		if q.decoding {
			changed := q.changed
			q.mu.Unlock()
			select {
			case <-ctx.Done():
				return AggregateIngressFenceFact{}, false
			case <-changed:
			}
			continue
		}
		if q.gateOpen && q.count < len(q.frames) {
			fact.ThroughFrameSequence = q.greatestRaw
			fact.MarkerOrdinal = q.nextFenceMarker
			q.nextFenceMarker++
			fact.CapturedAt = q.now()
			fact.State = engine.AggregateIngressFenceComplete
			q.appendLocked(queuedLiveFrame{epoch: fact.Command.ConnectionEpoch(), receivedAt: fact.CapturedAt,
				kind: queuedLiveIngressFence, ingressFence: fact})
			q.accounting.IngressFencesStarted++
			q.accounting.IngressFencesQueued++
			q.notifyLocked()
			q.mu.Unlock()
			return fact, true
		}
		changed := q.changed
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return AggregateIngressFenceFact{}, false
		case <-changed:
		}
	}
}

func (q *liveFrameQueue) enqueueLiveCoverageFence(ctx context.Context, fact LiveCoverageFenceFact) (LiveCoverageFenceFact, bool) {
	for {
		if ctx == nil || ctx.Err() != nil {
			return LiveCoverageFenceFact{}, false
		}
		q.mu.Lock()
		if q.accounting.IngressFencesQueued+q.accounting.IngressFencesClassifying > 0 {
			q.mu.Unlock()
			return LiveCoverageFenceFact{}, false
		}
		if !q.gateOpen || q.nextFenceMarker == 0 {
			q.mu.Unlock()
			return LiveCoverageFenceFact{}, false
		}
		if q.decoding {
			changed := q.changed
			q.mu.Unlock()
			select {
			case <-ctx.Done():
				return LiveCoverageFenceFact{}, false
			case <-changed:
			}
			continue
		}
		if q.count < len(q.frames) {
			fact.ThroughFrameSequence = q.greatestRaw
			fact.MarkerOrdinal = q.nextFenceMarker
			q.nextFenceMarker++
			fact.CapturedAt = q.now()
			fact.State = engine.LiveCoverageFenceComplete
			q.appendLocked(queuedLiveFrame{epoch: fact.Command.ConnectionEpoch(), receivedAt: fact.CapturedAt, kind: queuedLiveCoverageFence, liveCoverageFence: fact})
			q.accounting.IngressFencesStarted++
			q.accounting.IngressFencesQueued++
			q.notifyLocked()
			q.mu.Unlock()
			return fact, true
		}
		changed := q.changed
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return LiveCoverageFenceFact{}, false
		case <-changed:
		}
	}
}

func (q *liveFrameQueue) canceledIngressFence(command engine.HydrationFenceCommand) (AggregateIngressFenceFact, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.nextFenceMarker == 0 {
		return AggregateIngressFenceFact{}, false
	}
	fact := AggregateIngressFenceFact{Command: command, State: engine.AggregateIngressFenceCanceled,
		ThroughFrameSequence: q.greatestRaw, MarkerOrdinal: q.nextFenceMarker, CapturedAt: q.now()}
	q.nextFenceMarker++
	return fact, true
}

func (q *liveFrameQueue) closeGate() {
	q.mu.Lock()
	if q.gateOpen {
		q.gateOpen = false
		q.notifyLocked()
	}
	q.mu.Unlock()
}

func (q *liveFrameQueue) enqueueTerminal(ctx context.Context, epoch uint64, at time.Time, through uint64) (queuedLiveFrame, bool) {
	for {
		if ctx == nil || ctx.Err() != nil {
			return queuedLiveFrame{}, false
		}
		q.mu.Lock()
		if q.count < len(q.frames) && q.next != 0 {
			retained := make([]queuedLiveFrame, 0, q.count+1)
			for offset := 0; offset < q.count; offset++ {
				item := q.frameLocked(offset)
				if item.kind == queuedLiveIngressFence {
					item.ingressFence.State = engine.AggregateIngressFenceCanceled
					retained = append(retained, item)
					continue
				}
				if item.kind == queuedLiveCoverageFence {
					item.liveCoverageFence.State = engine.LiveCoverageFenceCanceled
					retained = append(retained, item)
					continue
				}
				if item.kind == queuedLiveTQCapacity {
					retained = append(retained, item)
					continue
				}
				if item.terminal {
					q.mu.Unlock()
					return item, true
				}
				if item.kind == queuedLiveDecodedBatch && item.sequence <= through {
					retained = append(retained, item)
					continue
				}
				q.accounting.FramesQueued--
				q.accounting.FramesFenced++
				q.bytes -= item.batch.RetainedCharge
			}
			clear(q.frames)
			q.head, q.count = 0, 0
			for _, item := range retained {
				q.appendLocked(item)
			}
			q.accounting.QueuedBytes = q.bytes
			frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true, kind: queuedLiveTerminal}
			q.next++
			q.appendLocked(frame)
			q.accounting.TerminalMarkersQueued = 1
			q.notifyLocked()
			q.mu.Unlock()
			return frame, true
		}
		changed := q.changed
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return queuedLiveFrame{}, false
		case <-changed:
		}
	}
}

func (q *liveFrameQueue) fenceQueuedAndEnqueueTerminal(epoch uint64, at time.Time) queuedLiveFrame {
	q.mu.Lock()
	defer q.mu.Unlock()
	var retained [2]queuedLiveFrame
	retainedCount := 0
	for offset := 0; offset < q.count; offset++ {
		frame := q.frameLocked(offset)
		if frame.terminal {
			return frame
		}
		if frame.kind == queuedLiveIngressFence {
			frame.ingressFence.State = engine.AggregateIngressFenceCanceled
			retained[retainedCount] = frame
			retainedCount++
			continue
		}
		if frame.kind == queuedLiveCoverageFence {
			frame.liveCoverageFence.State = engine.LiveCoverageFenceCanceled
			retained[retainedCount] = frame
			retainedCount++
			continue
		}
		if frame.kind == queuedLiveTQCapacity {
			retained[retainedCount] = frame
			retainedCount++
			continue
		}
		q.accounting.FramesQueued--
		q.accounting.FramesFenced++
	}
	clear(q.frames)
	q.head, q.count = 0, 0
	for index := 0; index < retainedCount; index++ {
		q.appendLocked(retained[index])
	}
	q.bytes = 0
	q.accounting.QueuedBytes = 0
	frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true, kind: queuedLiveTerminal}
	q.next++
	q.appendLocked(frame)
	q.accounting.TerminalMarkersQueued = 1
	q.notifyLocked()
	return frame
}

func (q *liveFrameQueue) waitForClassifier() {
	for {
		q.mu.Lock()
		if q.accounting.FramesClassifying == 0 {
			q.mu.Unlock()
			return
		}
		changed := q.changed
		q.mu.Unlock()
		<-changed
	}
}

func (q *liveFrameQueue) pop(ctx context.Context) (queuedLiveFrame, bool) {
	frame, ok, _ := q.popOrRecheck(ctx)
	return frame, ok
}

func (q *liveFrameQueue) popOrRecheck(ctx context.Context) (queuedLiveFrame, bool, bool) {
	for {
		q.mu.Lock()
		if q.count > 0 {
			frame := q.frames[q.head]
			q.frames[q.head] = queuedLiveFrame{}
			q.head = (q.head + 1) % len(q.frames)
			q.count--
			if q.count == 0 {
				q.head = 0
			}
			if frame.terminal {
				q.accounting.TerminalMarkersQueued = 0
				q.accounting.TerminalMarkersClassifying = 1
			} else if frame.kind == queuedLiveIngressFence || frame.kind == queuedLiveCoverageFence {
				q.accounting.IngressFencesQueued--
				q.accounting.IngressFencesClassifying++
			} else if frame.kind == queuedLiveTQCapacity {
				q.accounting.TQCapacityMarkersQueued = 0
				q.accounting.TQCapacityMarkersClassifying = 1
			} else {
				q.bytes -= frame.batch.RetainedCharge
				q.accounting.FramesQueued--
				q.accounting.FramesClassifying++
				q.accounting.QueuedBytes = q.bytes
				q.classifyingAt = frame.receivedAt
			}
			q.notifyLocked()
			q.mu.Unlock()
			return frame, true, false
		}
		if !q.gateOpen {
			q.mu.Unlock()
			return queuedLiveFrame{}, false, false
		}
		changed := q.changed
		recheck := q.recheck
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return queuedLiveFrame{}, false, false
		case <-recheck:
			return queuedLiveFrame{}, false, true
		case <-changed:
		}
	}
}

func (q *liveFrameQueue) complete(frame queuedLiveFrame, fenced bool) {
	q.mu.Lock()
	if frame.terminal {
		q.accounting.TerminalMarkersClassifying = 0
		q.accounting.TerminalMarkersDispositioned = 1
	} else if frame.kind == queuedLiveIngressFence || frame.kind == queuedLiveCoverageFence {
		q.accounting.IngressFencesClassifying--
		q.accounting.IngressFencesDispositioned++
	} else if frame.kind == queuedLiveTQCapacity {
		q.accounting.TQCapacityMarkersClassifying = 0
		q.accounting.TQCapacityMarkersDispositioned++
		q.tqCapacityMarkerOutstanding = false
	} else {
		q.accounting.FramesClassifying--
		q.classifyingAt = time.Time{}
		if fenced {
			q.accounting.FramesFenced++
		} else {
			q.accounting.FramesDispositioned++
		}
	}
	q.notifyLocked()
	q.mu.Unlock()
}

func (q *liveFrameQueue) snapshot() LiveQueueAccounting {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.admissionSnapshotLocked()
}

func (q *liveFrameQueue) lastRawPosition(epoch uint64) (engine.LivePosition, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if epoch == 0 || q.greatestRaw == 0 {
		return engine.LivePosition{}, false
	}
	return engine.LivePosition{ConnectionEpoch: epoch, FrameSequence: q.greatestRaw}, true
}

func (q *liveFrameQueue) lastAdmittedSequence() uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.greatestAdmitted
}
