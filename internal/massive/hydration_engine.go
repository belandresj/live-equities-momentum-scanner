package massive

import (
	"context"
	"errors"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

type CaptureAggregateIngressFenceCommand struct{ engine.HydrationFenceCommand }

const CaptureAggregateIngressFenceAction = "capture_aggregate_ingress_fence"

func (CaptureAggregateIngressFenceCommand) Action() string { return CaptureAggregateIngressFenceAction }

func CaptureAggregateIngressFenceCommandFromEngine(command engine.HydrationFenceCommand) (CaptureAggregateIngressFenceCommand, error) {
	if command.BindingIdentity() == "" || command.Generation() == 0 || command.ConnectionEpoch() == 0 || command.CommandToken() == 0 {
		return CaptureAggregateIngressFenceCommand{}, errors.New("invalid engine ingress fence command")
	}
	return CaptureAggregateIngressFenceCommand{HydrationFenceCommand: command}, nil
}

type AggregateIngressFenceFact struct {
	Command              engine.HydrationFenceCommand
	State                engine.AggregateIngressFenceState
	ThroughFrameSequence uint64
	MarkerOrdinal        uint64
	CapturedAt           time.Time
	deliveryStartedAt    time.Time
}

type LiveCoverageFenceFact struct {
	Command              engine.LiveCoverageFenceCommand
	State                engine.LiveCoverageFenceState
	ThroughFrameSequence uint64
	MarkerOrdinal        uint64
	CapturedAt           time.Time
}

func EngineLiveCoverageFence(fact LiveCoverageFenceFact) (engine.LiveCoverageFenceInput, error) {
	return engine.NewLiveCoverageFenceInput(fact.Command, fact.State, fact.ThroughFrameSequence, fact.MarkerOrdinal, fact.CapturedAt)
}

func (a *LiveAttempt) CaptureLiveCoverageFence(ctx context.Context, state *engine.Engine, command engine.LiveCoverageFenceCommand) error {
	if ctx == nil || state == nil || command.BindingIdentity() != a.binding.Identity() || command.ConnectionEpoch() != a.epoch || command.CommandToken() == 0 {
		return errCommand
	}
	a.mu.Lock()
	valid := a.started && a.handshaken && !a.finished && a.terminal == nil
	a.mu.Unlock()
	if !valid {
		return errCommand
	}
	if _, admitted := a.queue.enqueueLiveCoverageFence(ctx, LiveCoverageFenceFact{Command: command}); !admitted {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errCommand
	}
	return nil
}

func EngineAggregateIngressFence(fact AggregateIngressFenceFact) (engine.AggregateIngressFenceInput, error) {
	if !fact.deliveryStartedAt.IsZero() {
		return engine.NewAggregateIngressFenceInputAtDelivery(fact.Command, fact.State, fact.ThroughFrameSequence, fact.MarkerOrdinal, fact.CapturedAt, fact.deliveryStartedAt)
	}
	return engine.NewAggregateIngressFenceInput(fact.Command, fact.State, fact.ThroughFrameSequence, fact.MarkerOrdinal, fact.CapturedAt)
}

func (a *LiveAttempt) CaptureAggregateIngressFence(ctx context.Context, state *engine.Engine, command CaptureAggregateIngressFenceCommand) error {
	if ctx == nil || state == nil || command.BindingIdentity() != a.binding.Identity() || command.ConnectionEpoch() != a.epoch || command.CommandToken() == 0 {
		return errCommand
	}
	a.mu.Lock()
	valid := a.started && a.handshaken && !a.finished && a.terminal == nil && command.CommandToken() > a.captureToken
	if valid {
		a.captureToken = command.CommandToken()
	}
	a.mu.Unlock()
	if !valid {
		return errCommand
	}
	_, admitted := a.queue.enqueueIngressFence(ctx, AggregateIngressFenceFact{Command: command.HydrationFenceCommand})
	if !admitted {
		if err := a.cancelEngineHydrationFence(state, command.HydrationFenceCommand); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errCommand
	}
	return nil
}

func (a *LiveAttempt) cancelEngineHydrationFence(state *engine.Engine, command engine.HydrationFenceCommand) error {
	fact, ok := a.queue.canceledIngressFence(command)
	if !ok {
		return errCommand
	}
	input, err := EngineAggregateIngressFence(fact)
	if err != nil {
		return err
	}
	admission, completion := state.AdmitAggregateIngressFence(context.Background(), input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return errCommand
	}
	result := <-completion
	if result.Code != engine.DispositionAggregateIngressFenceRejected && result.Code != engine.DispositionAggregateIngressFenceFenced {
		return errCommand
	}
	return nil
}

// HydrationWorkItemFromEngine converts one engine allocation without giving
// Massive authority to select or change any plan identity.
func HydrationWorkItemFromEngine(binding reference.Binding, token engine.HydrationRequestToken) (HydrationWorkItem, error) {
	if binding.Identity() != token.BindingIdentity() {
		return HydrationWorkItem{}, errors.New("hydration token binding mismatch")
	}
	purpose, ok := massiveHydrationPurpose(token.Purpose())
	if !ok {
		return HydrationWorkItem{}, errors.New("unsupported engine hydration purpose")
	}
	return NewHydrationWorkItem(binding, token.Generation(), token.RequestID(), purpose, token.Symbol(), token.Start(), token.End(), token.ConnectionEpoch())
}

// EngineHydrationChunk copies one sealed Massive chunk into the engine's
// closed historical-input family and validates it against the allocation.
func EngineHydrationChunk(token engine.HydrationRequestToken, chunk HydrationResultChunk) (engine.HydrationChunkInput, error) {
	if !workMatchesToken(chunk.work, token) {
		return engine.HydrationChunkInput{}, errors.New("hydration chunk token mismatch")
	}
	values := chunk.Values()
	rows := make([]engine.HydrationRow, len(values))
	for index, value := range values {
		row, err := engine.NewHydrationRow(value.Symbol, value.WindowStart, value.WindowEnd, value.Values)
		if err != nil {
			return engine.HydrationChunkInput{}, err
		}
		rows[index] = row
	}
	return engine.NewHydrationChunkInput(token, chunk.ResultID(), chunk.Ordinal(), chunk.TotalChunks(), chunk.RowOffset(), chunk.TotalRows(), rows)
}

// EngineHydrationTerminal converts exactly one producer terminal fact. The
// engine constructor closes the state/reason/count vocabulary again.
func EngineHydrationTerminal(token engine.HydrationRequestToken, terminal HydrationTerminal) (engine.HydrationTerminalInput, error) {
	if !workMatchesToken(terminal.work, token) {
		return engine.HydrationTerminalInput{}, errors.New("hydration terminal token mismatch")
	}
	return engine.NewHydrationTerminalInput(token, terminal.ResultID(), engine.HydrationTerminalState(terminal.State()),
		engine.HydrationProviderReason(terminal.Reason()), terminal.Pages(), terminal.Attempts(), terminal.ResponseBytes(),
		terminal.NormalizedRows(), terminal.EmittedChunks(), terminal.EmittedRows())
}

func massiveHydrationPurpose(purpose engine.HydrationPurpose) (HydrationPurpose, bool) {
	switch purpose {
	case engine.HydrationFreshBootstrap:
		return HydrationFreshStart, true
	case engine.HydrationCheckpointCatchup:
		return HydrationCheckpointCatchUp, true
	case engine.HydrationGapRecovery:
		return HydrationGapRecovery, true
	default:
		return "", false
	}
}

func workMatchesToken(work HydrationWorkItem, token engine.HydrationRequestToken) bool {
	purpose, ok := massiveHydrationPurpose(token.Purpose())
	return ok && work.BindingIdentity() == token.BindingIdentity() && work.Generation() == token.Generation() &&
		work.RequestID() == token.RequestID() && work.Purpose() == purpose && work.Symbol() == token.Symbol() &&
		work.Start() == token.Start() && work.End() == token.End() && work.ConnectionEpoch() == token.ConnectionEpoch()
}
