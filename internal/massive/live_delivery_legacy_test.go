package massive

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// liveBatchCursor preserves historical per-result decoder assertions only in
// tests. Production transfers the complete DecodedBatch through the bounded
// ring and unbuffered engine handoff.
type liveBatchCursor struct {
	batch DecodedBatch
	index int
}

func newLiveFrameCursor(frame LiveFrame, statusContext *StatusContext, options LiveNormalizationOptions) *liveBatchCursor {
	return &liveBatchCursor{batch: decodeLiveFrame(frame, statusContext, options)}
}

func (c *liveBatchCursor) Next() (LiveResult, bool) {
	if c == nil || c.index >= len(c.batch.results) {
		return LiveResult{}, false
	}
	result := c.batch.results[c.index]
	c.index++
	return result, true
}

func (c *liveBatchCursor) Accounting() LiveFrameAccounting { return c.batch.accounting }

// Handshake is retained only for historical adapter-fact tests. Supported
// production calls HandshakeAndDeliver, which never builds this per-result
// view and transfers each decoded handshake batch directly to the engine.
func (a *LiveAttempt) Handshake(ctx context.Context) ([]AdapterDelivery, error) {
	if ctx == nil {
		return nil, errAttemptState
	}
	a.mu.Lock()
	if a.handshakeClaimed {
		a.mu.Unlock()
		return nil, errAttemptState
	}
	a.handshakeClaimed = true
	close(a.handshakeStart)
	a.mu.Unlock()
	<-a.handshakeOwnerDone
	stop := context.AfterFunc(ctx, a.cancel)
	defer stop()
	return a.performHandshakeForProof()
}

func (a *LiveAttempt) performHandshakeForProof() ([]AdapterDelivery, error) {
	totalCtx, totalCancel := context.WithTimeout(a.ctx, a.durations.HandshakeTotal)
	defer totalCancel()
	dialCtx, dialCancel := context.WithTimeout(totalCtx, a.durations.Dial)
	connection, err := a.connector.Dial(dialCtx, a.endpoint, int64(a.queue.config.MaxFrameBytes))
	dialCancel()
	if err != nil {
		a.triggerTerminal(TerminalReader, TerminalDialFailed, 0, false)
		return nil, errTransportFailed
	}
	a.mu.Lock()
	if a.terminal != nil || a.finished {
		a.mu.Unlock()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), a.durations.Close)
		_ = connection.Close(closeCtx)
		closeCancel()
		return nil, errTransportFailed
	}
	a.connection = connection
	a.startWorkers()
	a.mu.Unlock()

	deliveries := make([]AdapterDelivery, 0, 4)
	connected, err := a.awaitHandshakeStatusForProof(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseConnected, CommandKind: CommandConnection, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.ConnectionEstablished)
	deliveries = append(deliveries, connected...)
	if err != nil {
		return deliveries, err
	}
	a.accountConnected()
	authPayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "auth", Params: a.credential})
	if err := a.write(totalCtx, authPayload); err != nil {
		a.triggerTerminal(TerminalWriter, TerminalAuthenticationFailed, 0, false)
		return deliveries, errCommandWrite
	}
	authenticated, err := a.awaitHandshakeStatusForProof(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseAuthSuccess, CommandKind: CommandAuthentication, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.AuthenticationResult)
	deliveries = append(deliveries, authenticated...)
	if err != nil {
		return deliveries, err
	}
	aggregatePayload, _ := json.Marshal(struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}{Action: "subscribe", Params: "A.*"})
	if err := a.write(totalCtx, aggregatePayload); err != nil {
		deliveries = append(deliveries, controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlFailed, engine.LivePosition{}, a.adapter.now()))
		a.accountOpenCommand(engine.ControlFailed, false)
		a.triggerTerminal(TerminalWriter, TerminalAggregateSubscribeFailed, 0, false)
		return deliveries, errCommandWrite
	}
	a.accountOpenWriteSucceeded()
	deliveries = append(deliveries, controlDelivery(a.binding.Identity(), a.epoch, engine.AggregateCommandWriteResult, a.openToken, engine.ControlSucceeded, engine.LivePosition{}, a.adapter.now()))
	acknowledged, err := a.awaitHandshakeStatusForProof(totalCtx, StatusContext{ConnectionEpoch: a.epoch, ExpectedPhase: StatusPhaseSuccess, CommandKind: CommandAggregateSubscribe, CommandToken: strconv.FormatUint(a.openToken, 10), ExpectedCount: 1}, engine.AggregateSubscriptionResult)
	deliveries = append(deliveries, acknowledged...)
	if err != nil {
		return deliveries, err
	}
	a.accountOpenCommand(engine.ControlSucceeded, true)
	a.mu.Lock()
	a.handshaken = true
	a.mu.Unlock()
	return deliveries, nil
}

func (a *LiveAttempt) awaitHandshakeStatusForProof(parent context.Context, status StatusContext, kind engine.ConnectionControlKind) ([]AdapterDelivery, error) {
	stepCtx, cancel := context.WithTimeout(parent, a.durations.HandshakeStep)
	defer cancel()
	deliveries := make([]AdapterDelivery, 0, 4)
	for {
		frame, ok := a.queue.pop(stepCtx)
		if !ok {
			a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, engine.ControlAmbiguous, true), 0, false)
			return deliveries, errTransportFailed
		}
		if frame.terminal {
			return append(deliveries, a.finishTerminal(frame)), errTransportFailed
		}
		cursor := &liveBatchCursor{batch: frame.batch}
		found, failed, failureOutcome := false, false, engine.ControlAmbiguous
		for {
			result, ok := cursor.Next()
			if !ok {
				break
			}
			switch result.Kind {
			case LiveResultStatus:
				result.Status.CommandKind, result.Status.CommandToken = status.CommandKind, status.CommandToken
				if validStatusContext(&status, a.epoch) && result.Status.Phase == status.ExpectedPhase && result.Status.ObservedCount <= status.ExpectedCount {
					result.Status.Disposition, result.Status.Reason = StatusAcknowledged, ""
				}
				a.adapter.mu.Lock()
				a.adapter.accounting.HandshakeStatuses++
				a.adapter.mu.Unlock()
				outcome := engine.ControlSucceeded
				if result.Status.Disposition == StatusFailed {
					outcome = engine.ControlFailed
				} else if result.Status.Disposition != StatusAcknowledged || found {
					outcome = engine.ControlAmbiguous
				}
				deliveries = append(deliveries, controlDelivery(a.binding.Identity(), a.epoch, kind, a.openToken, outcome, result.Position, result.Status.ReceiptTime))
				found = outcome == engine.ControlSucceeded
				if outcome != engine.ControlSucceeded {
					failed, failureOutcome = true, outcome
				}
			case LiveResultAggregate, LiveResultTrade, LiveResultQuote, LiveResultRejected, LiveResultAmbiguous:
				if delivery, emit := a.mapResult(result, frame); emit {
					deliveries = append(deliveries, delivery)
				}
				if result.Kind == LiveResultAmbiguous {
					failed = true
				}
			}
		}
		a.queue.complete(frame, false)
		if !cursor.Accounting().Reconciles() {
			a.triggerTerminalAt(TerminalProtocol, TerminalIngressAmbiguity, frame.sequence, true, engine.LivePosition{ConnectionEpoch: a.epoch, FrameSequence: frame.sequence}, true, false)
			return deliveries, errTransportFailed
		}
		if failed {
			a.triggerTerminal(TerminalProtocol, handshakeTerminalReason(status.ExpectedPhase, failureOutcome, false), frame.sequence, false)
			return deliveries, errTransportFailed
		}
		if found {
			return deliveries, nil
		}
	}
}

// Historical adapter-boundary tests inspect individual decoded facts. This
// test-only cursor is deliberately absent from production, whose only consumer
// transfers and applies the complete DecodedBatch synchronously.
var legacyDeliveryProofState = struct {
	sync.Mutex
	cursors map[*LiveAttempt]*legacyDeliveryCursor
}{cursors: make(map[*LiveAttempt]*legacyDeliveryCursor)}

type legacyDeliveryCursor struct {
	frame  queuedLiveFrame
	cursor liveBatchCursor
}

func (a *LiveAttempt) nextForProof(ctx context.Context) (AdapterDelivery, bool) {
	a.deliveryMu.Lock()
	defer a.deliveryMu.Unlock()
	if ctx == nil || ctx.Err() != nil {
		return AdapterDelivery{}, false
	}
	for {
		legacyDeliveryProofState.Lock()
		active := legacyDeliveryProofState.cursors[a]
		if active != nil {
			result, ok := active.cursor.Next()
			if !ok {
				delete(legacyDeliveryProofState.cursors, a)
				a.queue.complete(active.frame, false)
				legacyDeliveryProofState.Unlock()
				continue
			}
			legacyDeliveryProofState.Unlock()
			if delivery, emit := a.mapResult(result, active.frame); emit {
				return delivery, true
			}
			continue
		}
		legacyDeliveryProofState.Unlock()

		a.mu.Lock()
		if a.finished && a.terminalReturned {
			a.mu.Unlock()
			return AdapterDelivery{}, false
		}
		terminal := a.terminal
		handshaken := a.handshaken
		a.mu.Unlock()
		if !handshaken && terminal == nil {
			return AdapterDelivery{}, false
		}
		frame, ok, recheck := a.queue.popOrRecheck(ctx)
		if recheck {
			continue
		}
		if !ok {
			a.mu.Lock()
			terminal = a.terminal
			a.mu.Unlock()
			if terminal == nil {
				return AdapterDelivery{}, false
			}
			select {
			case <-ctx.Done():
				return AdapterDelivery{}, false
			case <-a.cleanupDone:
			}
			frame, ok = a.queue.pop(ctx)
			if !ok {
				return AdapterDelivery{}, false
			}
		}
		if frame.terminal {
			return a.finishTerminal(frame), true
		}
		if frame.kind == queuedLiveIngressFence {
			a.queue.complete(frame, false)
			return AdapterDelivery{Kind: DeliveryAggregateIngressFence, AggregateIngressFence: frame.ingressFence}, true
		}
		if frame.kind == queuedLiveCoverageFence {
			a.queue.complete(frame, false)
			return AdapterDelivery{Kind: DeliveryLiveCoverageFence, LiveCoverageFence: frame.liveCoverageFence}, true
		}
		if terminal != nil && terminal.fenceAfter > 0 && frame.sequence > terminal.fenceAfter {
			a.queue.complete(frame, true)
			continue
		}
		legacyDeliveryProofState.Lock()
		legacyDeliveryProofState.cursors[a] = &legacyDeliveryCursor{frame: frame, cursor: liveBatchCursor{batch: frame.batch}}
		legacyDeliveryProofState.Unlock()
	}
}
