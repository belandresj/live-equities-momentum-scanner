package massive

import (
	"context"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

const (
	MaximumLiveFrameSlots = 512
	MaximumLiveQueueBytes = 64 << 20
)

type LiveQueueConfig struct {
	FrameSlots      int
	MaxFrameBytes   int
	TotalFrameBytes int
}

type FrameAdmissionReason string

const (
	FrameAdmitted         FrameAdmissionReason = "admitted"
	FrameRejectedOversize FrameAdmissionReason = "rejected_oversize"
	FrameRejectedCapacity FrameAdmissionReason = "rejected_capacity"
	FrameRejectedReceipt  FrameAdmissionReason = "rejected_receipt"
	FrameRejectedGate     FrameAdmissionReason = "rejected_gate_or_close"
)

type LiveQueueAccounting struct {
	FramesRead, FramesAdmitted                                                      uint64
	FramesRejectedOversize, FramesRejectedCapacity, FramesRejectedReceipt           uint64
	FramesRejectedGateOrClose                                                       uint64
	FramesQueued, FramesClassifying, FramesDispositioned, FramesFenced              uint64
	QueuedBytes                                                                     int
	TerminalMarkersQueued, TerminalMarkersClassifying, TerminalMarkersDispositioned uint64
	IngressFencesStarted                                                            uint64
	IngressFencesQueued, IngressFencesClassifying, IngressFencesDispositioned       uint64
	CapacityFrames, CapacityBytes                                                   int
	OldestFrameAge                                                                  time.Duration
}

func (a LiveQueueAccounting) Reconciles() bool {
	return a.FramesRead == a.FramesAdmitted+a.FramesRejectedOversize+
		a.FramesRejectedCapacity+a.FramesRejectedReceipt+a.FramesRejectedGateOrClose &&
		a.FramesAdmitted == a.FramesQueued+a.FramesClassifying+a.FramesDispositioned+a.FramesFenced &&
		a.TerminalMarkersQueued+a.TerminalMarkersClassifying+a.TerminalMarkersDispositioned <= 1 &&
		a.IngressFencesStarted == a.IngressFencesQueued+a.IngressFencesClassifying+a.IngressFencesDispositioned &&
		a.IngressFencesQueued+a.IngressFencesClassifying <= 1
}

type queuedLiveKind uint8

const (
	queuedLiveRaw queuedLiveKind = iota + 1
	queuedLiveTerminal
	queuedLiveIngressFence
	queuedLiveCoverageFence
)

type socketMessageType uint8

const (
	socketMessageText socketMessageType = iota + 1
	socketMessageBinary
)

type queuedLiveFrame struct {
	epoch, sequence   uint64
	receivedAt        time.Time
	messageType       socketMessageType
	data              []byte
	terminal          bool
	kind              queuedLiveKind
	ingressFence      AggregateIngressFenceFact
	liveCoverageFence LiveCoverageFenceFact
}

type liveFrameQueue struct {
	mu              sync.Mutex
	changed         chan struct{}
	recheck         chan struct{}
	config          LiveQueueConfig
	frames          []queuedLiveFrame
	bytes           int
	next            uint64
	lastReceipt     time.Time
	classifyingAt   time.Time
	greatestRaw     uint64
	nextFenceMarker uint64
	now             func() time.Time
	gateOpen        bool
	accounting      LiveQueueAccounting
}

func validateLiveQueueConfig(config LiveQueueConfig) bool {
	return config.FrameSlots >= 1 && config.FrameSlots <= MaximumLiveFrameSlots &&
		config.MaxFrameBytes >= 1 && config.MaxFrameBytes <= MaximumLiveFrameBytes &&
		config.TotalFrameBytes >= config.MaxFrameBytes && config.TotalFrameBytes <= MaximumLiveQueueBytes
}

func newLiveFrameQueue(config LiveQueueConfig) *liveFrameQueue {
	return &liveFrameQueue{config: config, frames: make([]queuedLiveFrame, 0, config.FrameSlots+2), changed: make(chan struct{}), recheck: make(chan struct{}), gateOpen: true, next: 1, nextFenceMarker: 1, now: func() time.Time { return time.Now().UTC() }}
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

func (q *liveFrameQueue) tryEnqueue(epoch uint64, messageType socketMessageType, receivedAt time.Time, data []byte) (queuedLiveFrame, FrameAdmissionReason) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.accounting.FramesRead++
	if !q.gateOpen || epoch == 0 || q.next == 0 {
		q.accounting.FramesRejectedGateOrClose++
		return queuedLiveFrame{}, FrameRejectedGate
	}
	if messageType != socketMessageText && messageType != socketMessageBinary {
		q.accounting.FramesRejectedGateOrClose++
		return queuedLiveFrame{}, FrameRejectedGate
	}
	if len(data) > q.config.MaxFrameBytes {
		q.accounting.FramesRejectedOversize++
		return queuedLiveFrame{}, FrameRejectedOversize
	}
	if receivedAt.IsZero() || receivedAt != receivedAt.UTC() || (!q.lastReceipt.IsZero() && receivedAt.Before(q.lastReceipt)) {
		q.accounting.FramesRejectedReceipt++
		return queuedLiveFrame{}, FrameRejectedReceipt
	}
	if q.accounting.FramesQueued >= uint64(q.config.FrameSlots) || len(data) > q.config.TotalFrameBytes-q.bytes {
		q.accounting.FramesRejectedCapacity++
		return queuedLiveFrame{}, FrameRejectedCapacity
	}
	frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: receivedAt, messageType: messageType, data: append([]byte(nil), data...), kind: queuedLiveRaw}
	q.next++
	q.greatestRaw = frame.sequence
	q.lastReceipt = receivedAt
	q.frames = append(q.frames, frame)
	q.bytes += len(frame.data)
	q.accounting.FramesAdmitted++
	q.accounting.FramesQueued++
	q.accounting.QueuedBytes = q.bytes
	q.notifyLocked()
	return frame, FrameAdmitted
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
		if q.gateOpen && q.accounting.FramesQueued < uint64(q.config.FrameSlots) {
			fact.ThroughFrameSequence = q.greatestRaw
			fact.MarkerOrdinal = q.nextFenceMarker
			q.nextFenceMarker++
			fact.CapturedAt = q.now()
			fact.State = engine.AggregateIngressFenceComplete
			q.frames = append(q.frames, queuedLiveFrame{epoch: fact.Command.ConnectionEpoch(), receivedAt: fact.CapturedAt,
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
		if q.accounting.FramesQueued < uint64(q.config.FrameSlots) {
			fact.ThroughFrameSequence = q.greatestRaw
			fact.MarkerOrdinal = q.nextFenceMarker
			q.nextFenceMarker++
			fact.CapturedAt = q.now()
			fact.State = engine.LiveCoverageFenceComplete
			q.frames = append(q.frames, queuedLiveFrame{epoch: fact.Command.ConnectionEpoch(), receivedAt: fact.CapturedAt, kind: queuedLiveCoverageFence, liveCoverageFence: fact})
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

func (q *liveFrameQueue) enqueueTerminal(ctx context.Context, epoch uint64, at time.Time) (queuedLiveFrame, bool) {
	for {
		q.mu.Lock()
		if q.accounting.FramesQueued < uint64(q.config.FrameSlots) && q.next != 0 {
			retained := q.frames[:0]
			for _, item := range q.frames {
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
				if item.terminal {
					q.mu.Unlock()
					return item, true
				}
				q.accounting.FramesQueued--
				q.accounting.FramesFenced++
			}
			q.frames = retained
			q.bytes = 0
			q.accounting.QueuedBytes = 0
			frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true, kind: queuedLiveTerminal}
			q.next++
			q.frames = append(q.frames, frame)
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
	retained := q.frames[:0]
	for _, frame := range q.frames {
		if frame.terminal {
			return frame
		}
		if frame.kind == queuedLiveIngressFence {
			frame.ingressFence.State = engine.AggregateIngressFenceCanceled
			retained = append(retained, frame)
			continue
		}
		if frame.kind == queuedLiveCoverageFence {
			frame.liveCoverageFence.State = engine.LiveCoverageFenceCanceled
			retained = append(retained, frame)
			continue
		}
		q.accounting.FramesQueued--
		q.accounting.FramesFenced++
	}
	q.frames = retained
	q.bytes = 0
	q.accounting.QueuedBytes = 0
	frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true, kind: queuedLiveTerminal}
	q.next++
	q.frames = append(q.frames, frame)
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
		if len(q.frames) > 0 {
			frame := q.frames[0]
			copy(q.frames, q.frames[1:])
			q.frames[len(q.frames)-1] = queuedLiveFrame{}
			q.frames = q.frames[:len(q.frames)-1]
			if frame.terminal {
				q.accounting.TerminalMarkersQueued = 0
				q.accounting.TerminalMarkersClassifying = 1
			} else if frame.kind == queuedLiveIngressFence || frame.kind == queuedLiveCoverageFence {
				q.accounting.IngressFencesQueued--
				q.accounting.IngressFencesClassifying++
			} else {
				q.bytes -= len(frame.data)
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
	result := q.accounting
	result.CapacityFrames, result.CapacityBytes = q.config.FrameSlots, q.config.TotalFrameBytes
	oldest := q.classifyingAt
	for _, frame := range q.frames {
		if frame.kind == queuedLiveRaw && (oldest.IsZero() || frame.receivedAt.Before(oldest)) {
			oldest = frame.receivedAt
		}
	}
	if !oldest.IsZero() {
		result.OldestFrameAge = q.now().Sub(oldest)
		if result.OldestFrameAge < 0 {
			result.OldestFrameAge = 0
		}
	}
	return result
}
