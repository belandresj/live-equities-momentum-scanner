package massive

import (
	"context"
	"sync"
	"time"
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
}

func (a LiveQueueAccounting) Reconciles() bool {
	return a.FramesRead == a.FramesAdmitted+a.FramesRejectedOversize+
		a.FramesRejectedCapacity+a.FramesRejectedReceipt+a.FramesRejectedGateOrClose &&
		a.FramesAdmitted == a.FramesQueued+a.FramesClassifying+a.FramesDispositioned+a.FramesFenced &&
		a.TerminalMarkersQueued+a.TerminalMarkersClassifying+a.TerminalMarkersDispositioned <= 1
}

type socketMessageType uint8

const (
	socketMessageText socketMessageType = iota + 1
	socketMessageBinary
)

type queuedLiveFrame struct {
	epoch, sequence uint64
	receivedAt      time.Time
	messageType     socketMessageType
	data            []byte
	terminal        bool
}

type liveFrameQueue struct {
	mu          sync.Mutex
	changed     chan struct{}
	config      LiveQueueConfig
	frames      []queuedLiveFrame
	bytes       int
	next        uint64
	lastReceipt time.Time
	gateOpen    bool
	accounting  LiveQueueAccounting
}

func validateLiveQueueConfig(config LiveQueueConfig) bool {
	return config.FrameSlots >= 1 && config.FrameSlots <= MaximumLiveFrameSlots &&
		config.MaxFrameBytes >= 1 && config.MaxFrameBytes <= MaximumLiveFrameBytes &&
		config.TotalFrameBytes >= config.MaxFrameBytes && config.TotalFrameBytes <= MaximumLiveQueueBytes
}

func newLiveFrameQueue(config LiveQueueConfig) *liveFrameQueue {
	return &liveFrameQueue{config: config, frames: make([]queuedLiveFrame, 0, config.FrameSlots), changed: make(chan struct{}), gateOpen: true, next: 1}
}

func (q *liveFrameQueue) notifyLocked() {
	close(q.changed)
	q.changed = make(chan struct{})
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
	if len(q.frames) >= q.config.FrameSlots || len(data) > q.config.TotalFrameBytes-q.bytes {
		q.accounting.FramesRejectedCapacity++
		return queuedLiveFrame{}, FrameRejectedCapacity
	}
	frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: receivedAt, messageType: messageType, data: append([]byte(nil), data...)}
	q.next++
	q.lastReceipt = receivedAt
	q.frames = append(q.frames, frame)
	q.bytes += len(frame.data)
	q.accounting.FramesAdmitted++
	q.accounting.FramesQueued++
	q.accounting.QueuedBytes = q.bytes
	q.notifyLocked()
	return frame, FrameAdmitted
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
		if len(q.frames) < q.config.FrameSlots && q.next != 0 {
			for range q.frames {
				q.accounting.FramesQueued--
				q.accounting.FramesFenced++
			}
			q.frames = q.frames[:0]
			q.bytes = 0
			q.accounting.QueuedBytes = 0
			frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true}
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
	for _, frame := range q.frames {
		if frame.terminal {
			return frame
		}
		q.accounting.FramesQueued--
		q.accounting.FramesFenced++
	}
	q.frames = q.frames[:0]
	q.bytes = 0
	q.accounting.QueuedBytes = 0
	frame := queuedLiveFrame{epoch: epoch, sequence: q.next, receivedAt: at.UTC(), terminal: true}
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
			} else {
				q.bytes -= len(frame.data)
				q.accounting.FramesQueued--
				q.accounting.FramesClassifying++
				q.accounting.QueuedBytes = q.bytes
			}
			q.notifyLocked()
			q.mu.Unlock()
			return frame, true
		}
		if !q.gateOpen {
			q.mu.Unlock()
			return queuedLiveFrame{}, false
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

func (q *liveFrameQueue) complete(frame queuedLiveFrame, fenced bool) {
	q.mu.Lock()
	if frame.terminal {
		q.accounting.TerminalMarkersClassifying = 0
		q.accounting.TerminalMarkersDispositioned = 1
	} else {
		q.accounting.FramesClassifying--
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
	return q.accounting
}
