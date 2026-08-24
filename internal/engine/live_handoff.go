package engine

import (
	"context"
	"errors"
	"math"
)

// LiveInput is one already-normalized mutation inside an immutable provider
// batch. Values are constructed by the typed helpers below so the ingress
// boundary cannot select an engine node kind or retain mutable aliases.
type LiveInput struct{ node *queueNode }

// LiveResult is the exact disposition of one LiveInput. Exactly one typed
// disposition is populated according to the input constructor used.
type LiveResult struct {
	Admission    AdmissionResult
	Disposition  Disposition
	Aggregate    AggregateDisposition
	Control      ConnectionControlDisposition
	Hydration    HydrationDisposition
	LiveCoverage LiveCoverageFenceDisposition
}

type liveHandoffRequest struct {
	inputs   []LiveInput
	complete chan []LiveResult
}

func NewLiveAggregate(input AggregateInput) (LiveInput, error) {
	if !boundedAggregateInput(input) {
		return LiveInput{}, errors.New("invalid live aggregate")
	}
	return LiveInput{node: &queueNode{kind: inputAggregate, aggregate: freezeAggregateInput(input)}}, nil
}

func NewLiveConnectionControl(input ConnectionControlInput) (LiveInput, error) {
	if !boundedConnectionControlInput(input) {
		return LiveInput{}, errors.New("invalid live connection control")
	}
	return LiveInput{node: &queueNode{kind: inputConnectionControl, connectionControl: frozenConnectionControlInput{ConnectionControlInput: input}}}, nil
}

func NewLiveTrade(input TradeInput) (LiveInput, error) {
	if !validTradeInput(input) {
		return LiveInput{}, errors.New("invalid live trade")
	}
	frozen := input
	frozen.Conditions = append([]int64(nil), input.Conditions...)
	return LiveInput{node: &queueNode{kind: inputTrade, trade: frozenTradeInput{frozen}}}, nil
}

func NewLiveQuote(input QuoteInput) (LiveInput, error) {
	if !validQuoteInput(input) {
		return LiveInput{}, errors.New("invalid live quote")
	}
	frozen := input
	frozen.Conditions = append([]int64(nil), input.Conditions...)
	frozen.Indicators = append([]int64(nil), input.Indicators...)
	return LiveInput{node: &queueNode{kind: inputQuote, quote: frozenQuoteInput{frozen}}}, nil
}

func NewLiveTQDrop(input TQDropInput) (LiveInput, error) {
	if input.SchemaVersion != TQSchemaV1 || !validIdentityShape(input.BindingIdentity) || (input.Family != "T" && input.Family != "Q") ||
		input.TradingDate == "" || len(input.Symbol) > maximumSymbolBytes || input.DropReason == "" || len(input.DropReason) > 64 || input.Live.ConnectionEpoch == 0 || input.Live.FrameSequence == 0 {
		return LiveInput{}, errors.New("invalid live T/Q drop")
	}
	return LiveInput{node: &queueNode{kind: inputTQDrop, tqDrop: frozenTQDropInput{input}}}, nil
}

func NewLiveTQControlError(input TQControlErrorInput) (LiveInput, error) {
	if !validTQControlErrorInput(input) {
		return LiveInput{}, errors.New("invalid live T/Q control error")
	}
	return LiveInput{node: &queueNode{kind: inputTQControlError, tqControlError: frozenTQControlErrorInput{input}}}, nil
}

func NewLiveTQCommandResult(input TQCommandResultInput) (LiveInput, error) {
	if !validTQCommand(input.command) {
		return LiveInput{}, errors.New("invalid live T/Q command result")
	}
	return LiveInput{node: &queueNode{kind: inputTQCommandResult, tqCommandResult: frozenTQCommandResultInput{input}}}, nil
}

func NewLiveAggregateIngressFence(input AggregateIngressFenceInput) (LiveInput, error) {
	if input.schemaVersion != AggregateIngressFenceSchemaV1 || !validHydrationFenceCommand(input.command) ||
		(input.state != AggregateIngressFenceComplete && input.state != AggregateIngressFenceFailed && input.state != AggregateIngressFenceCanceled) ||
		input.markerOrdinal == 0 || input.capturedAt.IsZero() || input.capturedAt != input.capturedAt.UTC() {
		return LiveInput{}, errors.New("invalid aggregate ingress fence")
	}
	return LiveInput{node: &queueNode{kind: inputAggregateIngressFence, aggregateIngressFence: frozenAggregateIngressFenceInput{input}}}, nil
}

func NewLiveCoverageFence(input LiveCoverageFenceInput) (LiveInput, error) {
	if input.schemaVersion != LiveCoverageFenceSchemaV1 || !validLiveCoverageFenceCommand(input.command) ||
		(input.state != LiveCoverageFenceComplete && input.state != LiveCoverageFenceCanceled) || input.markerOrdinal == 0 ||
		input.capturedAt.IsZero() || input.capturedAt != input.capturedAt.UTC() {
		return LiveInput{}, errors.New("invalid live coverage fence")
	}
	return LiveInput{node: &queueNode{kind: inputLiveCoverageFence, liveCoverageFence: frozenLiveCoverageFenceInput{input}}}, nil
}

// ConsumeLiveBatch transfers one complete immutable decoded batch through an
// unbuffered rendezvous to the sole engine loop. The loop consumes every input
// before accepting another batch, so live facts never enter the general FIFO
// and no per-result completion backlog exists.
func (e *Engine) ConsumeLiveBatch(ctx context.Context, inputs []LiveInput) ([]LiveResult, error) {
	if e == nil || ctx == nil || len(inputs) == 0 {
		return nil, errors.New("live batch handoff requires an engine, context, and inputs")
	}
	frozen := append([]LiveInput(nil), inputs...)
	for _, input := range frozen {
		if input.node == nil {
			return nil, errors.New("live batch contains an invalid input")
		}
	}
	request := liveHandoffRequest{inputs: frozen, complete: make(chan []LiveResult)}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.done:
		return nil, errors.New("engine is closed")
	case e.liveHandoff <- request:
	}
	// Ownership has transferred once the unbuffered send succeeds. Waiting is
	// then unconditional: returning on caller cancellation would strand the sole
	// engine loop on its equally unbuffered completion rendezvous.
	return <-request.complete, nil
}

func (e *Engine) consumeLiveHandoff(request liveHandoffRequest) {
	results := make([]LiveResult, 0, len(request.inputs))
	for _, input := range request.inputs {
		node := input.node
		e.mu.Lock()
		e.counters.started++
		e.counters.inProgress++
		if e.sealed {
			e.commitNonAdmissionLocked(AdmissionNotAdmittedClosed)
			e.mu.Unlock()
			results = append(results, LiveResult{Admission: AdmissionNotAdmittedClosed})
			continue
		}
		if e.lastReserved == math.MaxUint64-1 || systemPositionedInput(node.kind) && e.lastSystem == math.MaxUint64 {
			e.sealed, e.exhausted = true, true
			e.commitNonAdmissionLocked(AdmissionSequenceBudgetExhausted)
			e.broadcastLocked()
			e.mu.Unlock()
			results = append(results, LiveResult{Admission: AdmissionSequenceBudgetExhausted})
			continue
		}
		actual := e.clock().UTC()
		if e.hasClock && actual.Before(e.lastClock) {
			node.clockRegression, node.admissionTime = true, e.lastClock
		} else {
			e.lastClock, e.hasClock, node.admissionTime = actual, true, actual
		}
		e.lastReserved++
		node.ordinal = e.lastReserved
		if systemPositionedInput(node.kind) {
			e.lastSystem++
			node.systemSequence = e.lastSystem
		}
		node.engineSequence = e.nextSequence
		e.nextSequence++
		e.counters.inProgress--
		e.counters.resultsCommitted++
		e.counters.admittedExternal++
		e.counters.ownerInProgress++
		e.mu.Unlock()

		if e.beforeConsume != nil {
			e.beforeConsume(node)
		}
		disposition := e.transition(node)
		result := LiveResult{Admission: AdmissionAdmitted}
		switch node.kind {
		case inputAggregate:
			result.Aggregate = AggregateDisposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
		case inputConnectionControl:
			result.Control = ConnectionControlDisposition{EngineSequence: disposition.EngineSequence, SystemSequence: node.systemSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
		case inputAggregateIngressFence:
			result.Hydration = HydrationDisposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition,
				Plan: disposition.hydrationPlan, Rows: disposition.hydrationRows, Accounting: disposition.hydrationAccounting, FenceCommand: disposition.hydrationFenceCommand}
		case inputLiveCoverageFence:
			result.LiveCoverage = LiveCoverageFenceDisposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason,
				SuppressionDisposition: disposition.SuppressionDisposition, EvaluationTiming: e.ObserveEvaluationTiming()}
			e.finishLiveCoverageCommand(node, result.LiveCoverage)
		default:
			result.Disposition = Disposition{EngineSequence: disposition.EngineSequence, Code: disposition.Code, Reason: disposition.Reason, SuppressionDisposition: disposition.SuppressionDisposition}
		}
		results = append(results, result)
	}
	request.complete <- results
}
