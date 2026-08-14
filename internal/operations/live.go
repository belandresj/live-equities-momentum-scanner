package operations

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

type LiveComponents struct {
	Adapter                                                                *massive.LiveAdapter
	Hydrator                                                               *massive.HydrationWorker
	Store                                                                  *checkpoint.Store
	Workers, RowsPerChunk                                                  int
	MaximumResponseBytes, MaximumNormalizedRecords, MaximumResidentRecords int64
	Durations                                                              massive.OperationalDurations
}

func (c LiveComponents) valid() bool {
	return c.Adapter != nil && c.Hydrator != nil && c.Workers > 0 && c.RowsPerChunk > 0 &&
		c.MaximumResponseBytes > 0 && c.MaximumNormalizedRecords > 0 && c.MaximumResidentRecords > 0
}

// RunLive is the concrete Components 2/5/6/7 composition. It supplies facts to
// the sole engine owner and never derives readiness itself.
func (r *Runtime) RunLive(ctx context.Context, components LiveComponents) error {
	if r == nil || r.engine == nil || ctx == nil || !components.valid() {
		return errors.New("invalid live runtime composition")
	}
	lifetime, finish, err := r.beginLive(ctx)
	if err != nil {
		return err
	}
	defer finish()
	return r.runLive(lifetime, components)
}

func (r *Runtime) runLive(ctx context.Context, components LiveComponents) error {
	if components.Store != nil {
		r.installCheckpoint(ctx, components.Store)
	}
	var lastErr error
	for attemptOrdinal := uint64(1); ; attemptOrdinal++ {
		view := r.engine.ObserveOperational()
		if view.Lifecycle == "ended" {
			return nil
		}
		if view.Lifecycle == "suppressed" {
			if view.Suppression != engine.SuppressionSameBindingRecoveryAllowed {
				return errors.Join(errors.New("aggregate runtime requires restart"), lastErr)
			}
			if err := r.awaitScheduledRecovery(ctx); err != nil {
				return errors.Join(err, lastErr)
			}
		}
		establishmentCtx, cancelEstablishment := context.WithTimeout(ctx, r.config.ConnectionAttemptDeadline)
		attempt, err := r.openAttempt(ctx, establishmentCtx, components, attemptOrdinal*100)
		cancelEstablishment()
		if err != nil {
			if attempt != nil {
				r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			if r.recoveryBudgetExhausted() {
				if err := r.finishRecoveryExhaustion(ctx, lastErr); err != nil {
					return err
				}
			}
			continue
		}
		r.setLiveSources(attempt, components.Adapter)
		view = r.engine.ObserveOperational()
		var purpose engine.HydrationPurpose
		preSession := view.Lifecycle == "awaiting_session"
		if preSession {
			purpose, err = r.awaitHydrationAuthorization(ctx, attempt)
		} else {
			purpose, err = hydrationPurpose(view)
		}
		if err != nil {
			r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
			if !preSession {
				return err
			}
			lastErr = err
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if terminalErr := terminalLiveStateError(r.engine.ObserveOperational()); terminalErr != nil {
				return errors.Join(terminalErr, lastErr)
			}
			if r.recoveryBudgetExhausted() {
				if err := r.finishRecoveryExhaustion(ctx, lastErr); err != nil {
					return err
				}
			}
			continue
		}
		fenceCommand, err := r.hydrate(ctx, components, attempt, purpose)
		if err != nil {
			lastErr = err
			r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if r.recoveryBudgetExhausted() {
				if err := r.finishRecoveryExhaustion(ctx, lastErr); err != nil {
					return err
				}
			}
			continue
		}
		if r.engine.ObserveOperational().Hydration.PolicyWaiting {
			action := engine.HydrationPolicyRetry
			if r.recoveryBudgetExhausted() {
				action = engine.HydrationPolicyExhaust
			}
			if err := r.applyHydrationPolicy(ctx, fenceCommand, action, attemptOrdinal); err != nil {
				r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
				return err
			}
			r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseSuperseded)
			if action == engine.HydrationPolicyExhaust {
				lastErr = errors.New("aggregate recovery attempts exhausted")
			}
			continue
		}
		attemptClosed := false
	deliveryLoop:
		for {
			started := time.Now()
			deliveryCtx, cancelDelivery := context.WithTimeout(ctx, liveLifecyclePollInterval)
			result, ok, err := attempt.DeliverNextToEngine(deliveryCtx, r.engine)
			pollExpired := errors.Is(deliveryCtx.Err(), context.DeadlineExceeded)
			cancelDelivery()
			if ok {
				r.observeDelivery(started, result)
			}
			view = r.engine.ObserveOperational()
			switch {
			case view.Lifecycle == "ended":
				r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseSessionEnd)
				return nil
			case view.Lifecycle == "suppressed" && view.Suppression != engine.SuppressionSameBindingRecoveryAllowed:
				r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
				return errors.Join(errors.New("aggregate runtime requires restart"), lastErr)
			case view.Lifecycle == "suppressed" || (view.Lifecycle == "recovering" && !view.Connection.Active):
				r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
				attemptClosed = true
				break deliveryLoop
			}
			if pollExpired && ctx.Err() == nil && err == nil && !ok {
				continue
			}
			if err != nil || !ok {
				lastErr = err
				break
			}
		}
		if ctx.Err() != nil {
			r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseControlledStop)
			return ctx.Err()
		}
		// A deadline can stop the caller before the adapter has emitted its
		// terminal. Close and deliver that terminal so the engine, rather than
		// the supervisor, owns the transition to recovery.
		if !attemptClosed {
			r.closeAndDrain(attempt, attemptOrdinal*100+90, massive.CloseIntegrityLoss)
		}
	}
}

const liveLifecyclePollInterval = 100 * time.Millisecond

// awaitHydrationAuthorization keeps the established aggregate epoch consumed
// while the engine is legitimately waiting for the bound 04:00 session. The
// engine-owned timer is the sole authority that advances awaiting_session to
// hydrating; operations neither chooses that time nor admits a manual timer.
func (r *Runtime) awaitHydrationAuthorization(ctx context.Context, attempt *massive.LiveAttempt) (engine.HydrationPurpose, error) {
	for {
		view := r.engine.ObserveOperational()
		if err := terminalLiveStateError(view); err != nil {
			return "", err
		}
		if purpose, err := hydrationPurpose(view); err == nil {
			return purpose, nil
		} else if view.Lifecycle != "awaiting_session" {
			return "", err
		}

		wait := r.config.SampleCadence
		if wait > 100*time.Millisecond {
			wait = 100 * time.Millisecond
		}
		deliveryCtx, cancel := context.WithTimeout(ctx, wait)
		started := time.Now()
		result, ok, err := attempt.DeliverNextToEngine(deliveryCtx, r.engine)
		cancel()
		if ok {
			r.observeDelivery(started, result)
		}
		if err != nil {
			return "", errors.Join(errors.New("aggregate connection failed before hydration authorization"), err)
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !ok && !r.engine.ObserveOperational().Connection.Active {
			return "", errors.New("aggregate connection ended before hydration authorization")
		}
	}
}

func terminalLiveStateError(view engine.OperationalView) error {
	if view.Lifecycle != "suppressed" && view.Lifecycle != "ended" && view.Suppression == "" {
		return nil
	}
	return fmt.Errorf("live engine is terminal: lifecycle=%s reason=%s suppression=%s", view.Lifecycle, view.LifecycleReason, view.Suppression)
}

func hydrationPurpose(view engine.OperationalView) (engine.HydrationPurpose, error) {
	switch {
	case view.Lifecycle == "recovering":
		return engine.HydrationGapRecovery, nil
	case view.InstalledCheckpoint:
		return engine.HydrationCheckpointCatchup, nil
	case view.Lifecycle == "hydrating":
		return engine.HydrationFreshBootstrap, nil
	default:
		return "", fmt.Errorf("engine lifecycle %q does not authorize hydration", view.Lifecycle)
	}
}

func (r *Runtime) exhaustRecovery(ctx context.Context) error {
	view := r.engine.ObserveOperational()
	input := engine.RecoveryExhaustionInput{SchemaVersion: engine.RecoveryExhaustionSchemaV1, BindingIdentity: r.binding.Identity(), Attempts: view.Connection.RecoveryAttempts}
	admission, completion := r.engine.AdmitRecoveryExhaustion(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return errors.New("recovery exhaustion was not admitted")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-completion:
		if result.Code != engine.DispositionRecoveryExhausted || result.SuppressionDisposition != engine.SuppressionSameBindingRecoveryAllowed {
			return fmt.Errorf("recovery exhaustion was rejected: %s/%s", result.Code, result.Reason)
		}
		return nil
	}
}

func (r *Runtime) recoveryBudgetExhausted() bool {
	return r.engine.ObserveOperational().Connection.RecoveryAttempts >= uint64(r.config.RecoveryAttempts)
}

func (r *Runtime) finishRecoveryExhaustion(ctx context.Context, cause error) error {
	if err := r.exhaustRecovery(ctx); err != nil {
		return errors.Join(errors.New("aggregate recovery attempts exhausted without engine suppression"), err, cause)
	}
	return nil
}

func (r *Runtime) awaitScheduledRecovery(ctx context.Context) error {
	command, err := r.engine.IssueScheduledRecoveryCommand()
	if err != nil {
		return errors.New("engine did not issue scheduled recovery")
	}
	delay := command.EarliestAt().Sub(command.IssuedAt())
	if delay <= 0 || delay > r.config.RecoveryBackoffMax {
		return errors.New("engine issued invalid recovery deadline")
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	input, err := engine.NewScheduledRecoveryInput(command)
	if err != nil {
		return err
	}
	admission, completion := r.engine.AdmitScheduledRecovery(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return errors.New("scheduled recovery was not admitted")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-completion:
		if result.Code != engine.DispositionRecoveryScheduled {
			return fmt.Errorf("scheduled recovery was rejected: %s/%s", result.Code, result.Reason)
		}
		return nil
	}
}

func (r *Runtime) closeAndDrain(attempt *massive.LiveAttempt, token uint64, cause massive.CloseCause) {
	_ = attempt.Close(massive.CloseEpochCommand{BindingIdentity: r.binding.Identity(), ConnectionEpoch: attempt.Epoch(), CommandToken: token, Cause: cause})
	drainCtx, cancel := context.WithTimeout(context.Background(), r.config.ShutdownDeadline)
	defer cancel()
	for {
		started := time.Now()
		result, ok, err := attempt.DeliverNextToEngine(drainCtx, r.engine)
		if ok {
			r.observeDelivery(started, result)
		}
		if err != nil || !ok {
			_ = attempt.Wait(drainCtx)
			return
		}
	}
}

func (r *Runtime) installCheckpoint(ctx context.Context, store *checkpoint.Store) bool {
	for _, authority := range []checkpoint.CandidateAuthority{checkpoint.CandidateLatest, checkpoint.CandidatePrevious} {
		loaded := store.LoadCandidate(ctx, authority)
		if !loaded.Candidate.Integrity {
			continue
		}
		admission, completion := r.engine.AdmitCheckpointInstall(ctx, loaded.Candidate)
		if admission != engine.AdmissionAdmitted || completion == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return false
		case result := <-completion:
			if result.Disposition == engine.CheckpointInstalled {
				return true
			}
		}
	}
	return false
}

func (r *Runtime) openAttempt(lifetime, operation context.Context, components LiveComponents, token uint64) (*massive.LiveAttempt, error) {
	attempt, started, err := components.Adapter.Start(lifetime, massive.OpenAggregateEpoch{BindingIdentity: r.binding.Identity(), CommandToken: token, Durations: components.Durations})
	if err != nil {
		return nil, err
	}
	if result, err := massive.DeliverToEngine(operation, r.engine, started); err != nil || result.ControlDisposition.Code != engine.DispositionConnectionControlApplied {
		return attempt, errors.New("aggregate connection attempt was not installed")
	}
	deliveries, err := attempt.Handshake(operation)
	if err != nil {
		return attempt, err
	}
	for _, delivery := range deliveries {
		result, err := massive.DeliverToEngine(operation, r.engine, delivery)
		if err != nil || (result.ControlDisposition.Code != engine.DispositionConnectionControlApplied && result.ControlDisposition.Code != engine.DispositionConnectionControlDeferred) {
			return attempt, errors.New("aggregate handshake was not accepted")
		}
	}
	return attempt, nil
}

func (r *Runtime) hydrate(ctx context.Context, components LiveComponents, attempt *massive.LiveAttempt, purpose engine.HydrationPurpose) (engine.HydrationFenceCommand, error) {
	budgets := engine.HydrationPlanBudgets{Workers: components.Workers, RowsPerChunk: components.RowsPerChunk, MaximumResponseBytes: components.MaximumResponseBytes, MaximumNormalizedRecords: components.MaximumNormalizedRecords, MaximumResidentRecords: components.MaximumResidentRecords}
	admission, completion := r.engine.AdmitHydrationPlan(ctx, engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: r.binding.Identity(), Purpose: purpose, ConnectionEpoch: attempt.Epoch(), Budgets: budgets})
	if admission != engine.AdmissionAdmitted || completion == nil {
		return engine.HydrationFenceCommand{}, errors.New("hydration plan was not admitted")
	}
	var planResult engine.HydrationDisposition
	select {
	case <-ctx.Done():
		return engine.HydrationFenceCommand{}, ctx.Err()
	case planResult = <-completion:
	}
	if planResult.Code != engine.DispositionHydrationPlanApplied {
		return engine.HydrationFenceCommand{}, errors.New("hydration plan was rejected")
	}
	requests := planResult.Plan.Requests()
	if len(requests) == 0 {
		command, ok := planResult.Plan.FenceCommand()
		if !ok {
			return engine.HydrationFenceCommand{}, errors.New("empty hydration omitted fence")
		}
		return command, r.finishFence(ctx, attempt, command, planResult.Plan.End())
	}
	work := make([]massive.HydrationWorkItem, len(requests))
	tokens := make(map[uint64]engine.HydrationRequestToken, len(requests))
	for index, token := range requests {
		item, err := massive.HydrationWorkItemFromEngine(r.binding, token)
		if err != nil {
			return engine.HydrationFenceCommand{}, err
		}
		work[index], tokens[token.RequestID()] = item, token
	}
	plan, err := massive.NewHydrationWorkerPlan(work, components.Workers, components.RowsPerChunk, components.MaximumResponseBytes, components.MaximumNormalizedRecords, components.MaximumResidentRecords)
	if err != nil {
		return engine.HydrationFenceCommand{}, err
	}
	sink := &engineHydrationSink{owner: r.engine, tokens: tokens}
	workCtx, cancelWork := context.WithCancel(ctx)
	pumpCtx, cancelPump := context.WithCancel(ctx)
	defer cancelPump()
	pumpDone := make(chan error, 1)
	go func() {
		if r.beforeHydrationPump != nil {
			r.beforeHydrationPump(attempt)
		}
		for {
			started := time.Now()
			result, ok, err := attempt.DeliverNextToEngine(pumpCtx, r.engine)
			if ok {
				r.observeDelivery(started, result)
			}
			if err != nil {
				cancelWork()
				pumpDone <- err
				return
			}
			if !ok {
				if pumpCtx.Err() != nil {
					pumpDone <- pumpCtx.Err()
				} else {
					cancelWork()
					pumpDone <- errors.New("aggregate connection ended during hydration")
				}
				return
			}
		}
	}()
	result := components.Hydrator.Run(workCtx, ctx, plan, sink)
	cancelWork()
	accounting := result.Accounting()
	fence, sinkErr := sink.result()
	if sinkErr != nil || accounting.ItemsStarted != accounting.ProviderCompletedValue+accounting.ProviderCompletedEmpty+accounting.ProviderFailed+accounting.ProviderCanceled || fence.CommandToken() == 0 {
		cancelPump()
		<-pumpDone
		return engine.HydrationFenceCommand{}, errors.New("hydration worker did not terminally reconcile")
	}
	if ctx.Err() != nil {
		cancelPump()
		<-pumpDone
		return engine.HydrationFenceCommand{}, ctx.Err()
	}
	// REST may finish before its supported boundary is eligible at T. Keep the
	// sole live consumer running during that wait; otherwise the socket reader
	// can fill C5's bounded queue while no goroutine dispositions the live tail.
	eligibleAt := planResult.Plan.End().Add(r.config.EvaluationDelay)
	if wait := eligibleAt.Sub(r.clock().UTC()); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			cancelPump()
			<-pumpDone
			return engine.HydrationFenceCommand{}, ctx.Err()
		case pumpErr := <-pumpDone:
			if !timer.Stop() {
				<-timer.C
			}
			if pumpErr == nil {
				pumpErr = errors.New("aggregate connection ended during hydration fence wait")
			}
			return engine.HydrationFenceCommand{}, errors.Join(errors.New("live delivery failed during hydration"), pumpErr)
		case <-timer.C:
		}
	}
	cancelPump()
	pumpErr := <-pumpDone
	if pumpErr != nil && !errors.Is(pumpErr, context.Canceled) {
		return engine.HydrationFenceCommand{}, errors.Join(errors.New("live delivery failed during hydration"), pumpErr)
	}
	return fence, r.finishEligibleFence(ctx, attempt, fence)
}

func (r *Runtime) applyHydrationPolicy(ctx context.Context, command engine.HydrationFenceCommand, action engine.HydrationPolicyAction, token uint64) error {
	input, err := engine.NewHydrationPolicyActionInput(command, action, token)
	if err != nil {
		return err
	}
	admission, completion := r.engine.AdmitHydrationPolicyAction(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return errors.New("hydration recovery policy was not admitted")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-completion:
		if action == engine.HydrationPolicyExhaust {
			if result.Code != engine.DispositionIngressIntegrity {
				return fmt.Errorf("hydration exhaustion was rejected: %s/%s", result.Code, result.Reason)
			}
			return nil
		}
		if result.Code != engine.DispositionHydrationPolicyApplied {
			return fmt.Errorf("hydration retry was rejected: %s/%s", result.Code, result.Reason)
		}
		return nil
	}
}

func (r *Runtime) finishFence(ctx context.Context, attempt *massive.LiveAttempt, command engine.HydrationFenceCommand, supportedEnd time.Time) error {
	eligibleAt := supportedEnd.Add(r.config.EvaluationDelay)
	if wait := eligibleAt.Sub(r.clock().UTC()); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return r.finishEligibleFence(ctx, attempt, command)
}

func (r *Runtime) finishEligibleFence(ctx context.Context, attempt *massive.LiveAttempt, command engine.HydrationFenceCommand) error {
	capture, err := massive.CaptureAggregateIngressFenceCommandFromEngine(command)
	if err != nil {
		return err
	}
	if err := attempt.CaptureAggregateIngressFence(ctx, r.engine, capture); err != nil {
		return err
	}
	for {
		started := time.Now()
		result, ok, err := attempt.DeliverNextToEngine(ctx, r.engine)
		if ok {
			r.observeDelivery(started, result)
		}
		if err != nil {
			return errors.Join(errors.New("hydration fence delivery failed"), err)
		}
		if !ok {
			return errors.New("aggregate connection ended before hydration fence reconciliation")
		}
		if result.HydrationDisposition.Code == engine.DispositionAggregateIngressFenceApplied {
			return nil
		}
	}
}

type engineHydrationSink struct {
	owner  *engine.Engine
	tokens map[uint64]engine.HydrationRequestToken
	mu     sync.Mutex
	fence  engine.HydrationFenceCommand
	err    error
}

func (s *engineHydrationSink) result() (engine.HydrationFenceCommand, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fence, s.err
}

func (s *engineHydrationSink) AdmitHydrationChunk(ctx context.Context, chunk massive.HydrationResultChunk) error {
	token, ok := s.tokens[chunk.ResultID()]
	if !ok {
		return errors.New("unknown hydration chunk")
	}
	input, err := massive.EngineHydrationChunk(token, chunk)
	if err != nil {
		return err
	}
	admission, completion := s.owner.AdmitHydrationChunk(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return massive.ErrHydrationInputClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-completion:
		if result.Code != engine.DispositionHydrationChunkApplied {
			return errors.New("hydration chunk rejected")
		}
	}
	return nil
}

func (s *engineHydrationSink) AdmitHydrationTerminal(ctx context.Context, terminal massive.HydrationTerminal) error {
	token, ok := s.tokens[terminal.ResultID()]
	if !ok {
		return errors.New("unknown hydration terminal")
	}
	input, err := massive.EngineHydrationTerminal(token, terminal)
	if err != nil {
		return err
	}
	admission, completion := s.owner.AdmitHydrationTerminal(ctx, input)
	if admission != engine.AdmissionAdmitted || completion == nil {
		return massive.ErrHydrationInputClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-completion:
		if result.Code != engine.DispositionHydrationTerminalApplied {
			s.mu.Lock()
			s.err = errors.Join(s.err, errors.New("hydration terminal rejected"))
			s.mu.Unlock()
			return nil
		}
		if result.FenceCommand.CommandToken() != 0 {
			s.mu.Lock()
			s.fence = result.FenceCommand
			s.mu.Unlock()
		}
	}
	return nil
}

func defaultDurations() massive.OperationalDurations {
	return massive.OperationalDurations{Dial: 10 * time.Second, HandshakeStep: 5 * time.Second, HandshakeTotal: 30 * time.Second, HeartbeatInterval: 15 * time.Second, HeartbeatDeadline: 5 * time.Second, Write: 5 * time.Second, Close: 5 * time.Second}
}
