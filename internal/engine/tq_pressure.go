package engine

import (
	"context"
	"errors"
	"time"
)

type TQPressureMode string

const (
	TQPressureNormal        TQPressureMode = "normal"
	TQPressureDegraded      TQPressureMode = "taq_degraded"
	TQPressureAggregateOnly TQPressureMode = "aggregate_only"
)

type TQPressureSample struct {
	QueueCurrentFrames       uint64
	QueueCapacityFrames      uint64
	OldestFrameAge           time.Duration
	MaxDeliveryDelayOneSec   time.Duration
	HeapAllocBytes           uint64
	Goroutines               int
	TQLocalAccountingHealthy bool
}

// TQPressureCommand is an opaque, one-shot request for a fixed-cardinality
// process sample. Its private identity prevents callers from manufacturing a
// result for a command they were not given by this engine.
type TQPressureCommand struct {
	bindingIdentity string
	sequence        uint64
	issuedAt        time.Time
	done            chan Disposition
}

func (c TQPressureCommand) BindingIdentity() string { return c.bindingIdentity }
func (c TQPressureCommand) Sequence() uint64        { return c.sequence }
func (c TQPressureCommand) IssuedAt() time.Time     { return c.issuedAt }

type TQPressureResultInput struct {
	command TQPressureCommand
	sample  TQPressureSample
}

type frozenTQPressureResultInput struct{ TQPressureResultInput }

type tqPressurePolicy struct {
	sampleCadence, commandTimeout, degradedDwell, aggregateDwell      time.Duration
	recoveryDwell, minimumDegraded, restoreInterval                   time.Duration
	degradedQueuePercent, aggregateQueuePercent, recoveryQueuePercent uint64
	degradedOldest, aggregateOldest, recoveryOldest                   time.Duration
	degradedDelivery, aggregateDelivery, recoveryDelivery             time.Duration
	degradedHeap, aggregateHeap, recoveryHeap                         uint64
	degradedGoroutines, aggregateGoroutines, recoveryGoroutines       int
}

func defaultTQPressurePolicy() tqPressurePolicy {
	return tqPressurePolicy{
		sampleCadence: time.Second, commandTimeout: 2 * time.Second, degradedDwell: 0, aggregateDwell: 2 * time.Second,
		recoveryDwell: 30 * time.Second, minimumDegraded: 15 * time.Second, restoreInterval: 5 * time.Second,
		degradedQueuePercent: 25, aggregateQueuePercent: 60, recoveryQueuePercent: 20,
		degradedOldest: 250 * time.Millisecond, aggregateOldest: 1500 * time.Millisecond, recoveryOldest: 100 * time.Millisecond,
		degradedDelivery: 2 * time.Second, aggregateDelivery: 5 * time.Second, recoveryDelivery: 500 * time.Millisecond,
		degradedHeap: 512 << 20, aggregateHeap: 1280 << 20, recoveryHeap: 384 << 20,
		degradedGoroutines: 64, aggregateGoroutines: 128, recoveryGoroutines: 48,
	}
}

type tqPressureState struct {
	mode              TQPressureMode
	pending           *TQPressureCommand
	dispatched        bool
	nextSequence      uint64
	consecutiveMisses uint32
	unhealthySince    time.Time
	degradedAt        time.Time
	recoverySince     time.Time
	transitions       uint64
	fenced            uint64
	lastTick          time.Time
}

func (e *Engine) AdmitTQPressureTick(ctx context.Context) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQPressureTick}, false)
}

// AdmitOperationalIngressIntegrity routes broad queue/adapter accounting
// ambiguity through the existing global ingress-integrity lifecycle rather
// than misclassifying it as a T/Q-local pressure fact.
func (e *Engine) AdmitOperationalIngressIntegrity(ctx context.Context) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputOperationalIngressIntegrity}, false)
}

func (e *Engine) IssueTQPressureCommand() (TQPressureCommand, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := &e.state.tq.pressure
	if p.pending == nil || p.dispatched || !validTQPressureCommand(*p.pending) {
		return TQPressureCommand{}, errors.New("T/Q pressure command is not issuable")
	}
	p.dispatched = true
	return *p.pending, nil
}

func NewTQPressureResultInput(command TQPressureCommand, sample TQPressureSample) (TQPressureResultInput, error) {
	if !validTQPressureCommand(command) || !validTQPressureSample(sample) {
		return TQPressureResultInput{}, errors.New("invalid T/Q pressure result")
	}
	return TQPressureResultInput{command: command, sample: sample}, nil
}

func (e *Engine) AdmitTQPressureResult(ctx context.Context, input TQPressureResultInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validTQPressureCommand(input.command) || !validTQPressureSample(input.sample) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputTQPressureResult, tqPressureResult: frozenTQPressureResultInput{input}}, false)
}

func validTQPressureCommand(v TQPressureCommand) bool {
	return validIdentityShape(v.bindingIdentity) && v.sequence > 0 && !v.issuedAt.IsZero() && v.issuedAt == v.issuedAt.UTC() && v.done != nil
}

func validTQPressureSample(v TQPressureSample) bool {
	return v.QueueCapacityFrames > 0 && v.QueueCurrentFrames <= v.QueueCapacityFrames && v.OldestFrameAge >= 0 &&
		v.MaxDeliveryDelayOneSec >= 0 && v.Goroutines >= 0
}

func (e *Engine) advanceTQPressureTimerLocked(now time.Time) {
	if e.mode != RunModeLive || e.state.binding == nil || !e.state.liveEpochActive || e.state.lifecycle == lifecycleEnded {
		return
	}
	p := &e.state.tq.pressure
	if p.mode == "" {
		p.mode = TQPressureNormal
	}
	if !p.lastTick.IsZero() {
		elapsed := now.Sub(p.lastTick)
		if p.pending == nil && elapsed >= 2*e.tqPressurePolicy.sampleCadence {
			missed := uint32(elapsed/e.tqPressurePolicy.sampleCadence) - 1
			if missed > 2 {
				missed = 2
			}
			p.consecutiveMisses += missed
			p.recoverySince = time.Time{}
			if p.consecutiveMisses >= 2 {
				e.setTQPressureModeLocked(TQPressureAggregateOnly, now)
			} else {
				e.setTQPressureModeLocked(TQPressureDegraded, now)
			}
		} else if p.pending != nil && elapsed >= e.tqPressurePolicy.commandTimeout+e.tqPressurePolicy.sampleCadence {
			p.consecutiveMisses++
			p.recoverySince = time.Time{}
			e.setTQPressureModeLocked(TQPressureDegraded, now)
		}
	}
	p.lastTick = now
	if p.pending != nil && !now.Before(p.pending.issuedAt.Add(e.tqPressurePolicy.commandTimeout)) {
		p.pending, p.dispatched = nil, false
		p.consecutiveMisses++
		if p.consecutiveMisses >= 2 {
			e.setTQPressureModeLocked(TQPressureAggregateOnly, now)
		} else {
			e.setTQPressureModeLocked(TQPressureDegraded, now)
		}
	}
	if p.pending == nil {
		if p.nextSequence == 0 {
			p.nextSequence = 1
		}
		command := &TQPressureCommand{bindingIdentity: e.state.binding.identity, sequence: p.nextSequence, issuedAt: now, done: make(chan Disposition, 1)}
		p.nextSequence++
		p.pending, p.dispatched = command, false
	}
}

func (e *Engine) applyTQPressureResultLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, p := node.tqPressureResult.TQPressureResultInput, &e.state.tq.pressure
	if e.state.binding == nil || v.command.bindingIdentity != e.state.binding.identity || p.pending == nil || v.command != *p.pending || !p.dispatched {
		p.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	if node.admissionTime.After(v.command.issuedAt.Add(e.tqPressurePolicy.commandTimeout)) {
		p.pending, p.dispatched = nil, false
		p.consecutiveMisses++
		if p.consecutiveMisses >= 2 {
			e.setTQPressureModeLocked(TQPressureAggregateOnly, node.admissionTime)
		} else {
			e.setTQPressureModeLocked(TQPressureDegraded, node.admissionTime)
		}
		p.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	p.pending, p.dispatched, p.consecutiveMisses = nil, false, 0
	e.applyTQPressureSampleLocked(v.sample, node.admissionTime)
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyTQPressureSampleLocked(sample TQPressureSample, now time.Time) {
	p, policy := &e.state.tq.pressure, e.tqPressurePolicy
	severe := !sample.TQLocalAccountingHealthy || pressureQueueAtLeast(sample, policy.aggregateQueuePercent) || sample.OldestFrameAge >= policy.aggregateOldest ||
		sample.MaxDeliveryDelayOneSec >= policy.aggregateDelivery || sample.HeapAllocBytes >= policy.aggregateHeap || sample.Goroutines >= policy.aggregateGoroutines
	unhealthy := pressureQueueAtLeast(sample, policy.degradedQueuePercent) || sample.OldestFrameAge >= policy.degradedOldest ||
		sample.MaxDeliveryDelayOneSec >= policy.degradedDelivery || sample.HeapAllocBytes >= policy.degradedHeap || sample.Goroutines >= policy.degradedGoroutines
	healthy := sample.TQLocalAccountingHealthy && pressureQueueBelow(sample, policy.recoveryQueuePercent) && sample.OldestFrameAge < policy.recoveryOldest &&
		sample.MaxDeliveryDelayOneSec < policy.recoveryDelivery && sample.HeapAllocBytes < policy.recoveryHeap && sample.Goroutines < policy.recoveryGoroutines
	if severe {
		p.unhealthySince, p.recoverySince = now, time.Time{}
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now)
		return
	}
	if unhealthy {
		p.recoverySince = time.Time{}
		if p.unhealthySince.IsZero() {
			p.unhealthySince = now
		}
		if p.mode == TQPressureNormal && !now.Before(p.unhealthySince.Add(policy.degradedDwell)) {
			e.setTQPressureModeLocked(TQPressureDegraded, now)
		}
		if p.mode == TQPressureDegraded && !now.Before(p.unhealthySince.Add(policy.aggregateDwell)) {
			e.setTQPressureModeLocked(TQPressureAggregateOnly, now)
		}
		return
	}
	p.unhealthySince = time.Time{}
	if p.mode == TQPressureNormal {
		p.recoverySince = time.Time{}
		return
	}
	if !healthy {
		p.recoverySince = time.Time{}
		return
	}
	if p.recoverySince.IsZero() {
		p.recoverySince = now
	}
	if !now.Before(p.recoverySince.Add(policy.recoveryDwell)) && !now.Before(p.degradedAt.Add(policy.minimumDegraded)) {
		e.setTQPressureModeLocked(TQPressureNormal, now)
	}
}

func pressureQueueAtLeast(sample TQPressureSample, percent uint64) bool {
	return sample.QueueCurrentFrames*100 >= sample.QueueCapacityFrames*percent
}

func pressureQueueBelow(sample TQPressureSample, percent uint64) bool {
	return sample.QueueCurrentFrames*100 < sample.QueueCapacityFrames*percent
}

func (e *Engine) setTQPressureModeLocked(mode TQPressureMode, now time.Time) {
	p, s := &e.state.tq.pressure, &e.state.tq
	if mode == TQPressureNormal && s.globalBound {
		return
	}
	if mode == TQPressureDegraded && (p.mode == TQPressureAggregateOnly || s.aggregateOnly) {
		return
	}
	if p.mode == mode {
		return
	}
	p.mode, p.transitions = mode, p.transitions+1
	switch mode {
	case TQPressureDegraded:
		if p.degradedAt.IsZero() {
			p.degradedAt = now
		}
		e.stopTQAdditionsLocked()
		for _, member := range s.members {
			member.tradeCoverage.active, member.quoteCoverage.active = false, false
			if member.present {
				member.resetRequired = true
			}
		}
	case TQPressureAggregateOnly:
		if p.degradedAt.IsZero() {
			p.degradedAt = now
		}
		e.enterTQAggregateOnlyLocked()
	case TQPressureNormal:
		p.unhealthySince, p.recoverySince, p.degradedAt = time.Time{}, time.Time{}, time.Time{}
		if !s.globalBound {
			s.aggregateOnly, s.restoring, s.restoreNotBefore = false, true, now
		}
	}
}
