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

type TQPressureCause string

const (
	TQPressureCauseNone                TQPressureCause = ""
	TQPressureCauseWaitingFrames       TQPressureCause = "waiting_frames"
	TQPressureCauseWaitingBytes        TQPressureCause = "waiting_bytes"
	TQPressureCauseOldestWaitingFrame  TQPressureCause = "oldest_waiting_frame"
	TQPressureCauseWatermarkLag        TQPressureCause = "aggregate_watermark_lag"
	TQPressureCauseCapacityDrop        TQPressureCause = "capacity_drop"
	TQPressureCauseRetentionBound      TQPressureCause = "tq_retention_bound"
	TQPressureCauseTransportAccounting TQPressureCause = "transport_accounting_loss"
)

type TQPressureSample struct {
	WaitingFrames            uint64
	FrameCapacity            uint64
	WaitingBytes             uint64
	ByteCapacity             uint64
	OldestWaitingFrameAge    time.Duration
	SlotCapacityDrops        uint64
	ByteCapacityDrops        uint64
	AggregateWatermarkLag    time.Duration
	TQWorkPresent            bool
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
	sampleCadence, commandTimeout                                     time.Duration
	degradedQueuePercent, aggregateQueuePercent, recoveryQueuePercent uint64
	degradedOldest, aggregateOldest, recoveryOldest                   time.Duration
	degradedSamples, aggregateSamples, recoverySamples                uint8
}

func defaultTQPressurePolicy() tqPressurePolicy {
	return tqPressurePolicy{
		sampleCadence: time.Second, commandTimeout: 2 * time.Second,
		degradedQueuePercent: 10, aggregateQueuePercent: 25, recoveryQueuePercent: 1,
		degradedOldest: time.Second, aggregateOldest: 2 * time.Second, recoveryOldest: 750 * time.Millisecond,
		degradedSamples: 2, aggregateSamples: 3, recoverySamples: 5,
	}
}

type tqPressureStreaks struct {
	degradedFrames, degradedBytes, degradedOldest    uint8
	aggregateFrames, aggregateBytes, aggregateOldest uint8
	healthy                                          uint8
}

type tqPressureState struct {
	mode                         TQPressureMode
	pending                      *TQPressureCommand
	dispatched                   bool
	nextSequence                 uint64
	consecutiveMisses            uint32
	degradedAt                   time.Time
	streaks                      tqPressureStreaks
	transitions                  uint64
	fenced                       uint64
	lastTick                     time.Time
	cause                        TQPressureCause
	lastSlotDrops, lastByteDrops uint64
	capacityObserved             bool
	lastSample                   TQPressureSample
	sampleObserved               bool
	lastSampleRecoveryHealthy    bool
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
	return v.FrameCapacity > 0 && v.WaitingFrames <= v.FrameCapacity && v.ByteCapacity > 0 && v.WaitingBytes <= v.ByteCapacity &&
		v.OldestWaitingFrameAge >= 0 && v.AggregateWatermarkLag >= 0
}

func (e *Engine) advanceTQPressureTimerLocked(now time.Time) {
	if e.state.binding == nil || !e.state.liveEpochActive || e.state.lifecycle == lifecycleEnded {
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
			p.streaks = tqPressureStreaks{}
			// A missed diagnostic sample is not direct feed-consumption pressure.
		} else if p.pending != nil && elapsed >= e.tqPressurePolicy.commandTimeout+e.tqPressurePolicy.sampleCadence {
			p.consecutiveMisses++
			p.streaks = tqPressureStreaks{}
			// Keep the current pressure state; the next tick issues fresh work.
		}
	}
	p.lastTick = now
	if p.pending != nil && !now.Before(p.pending.issuedAt.Add(e.tqPressurePolicy.commandTimeout)) {
		p.pending, p.dispatched = nil, false
		p.consecutiveMisses++
		p.streaks = tqPressureStreaks{}
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
		p.fenced++
		return DispositionTQFenced, ReasonHistoricalContext
	}
	p.pending, p.dispatched, p.consecutiveMisses = nil, false, 0
	e.applyTQPressureSampleLocked(v.sample, node.admissionTime)
	return DispositionTQApplied, ReasonNone
}

func (e *Engine) applyTQPressureSampleLocked(sample TQPressureSample, now time.Time) {
	p, policy := &e.state.tq.pressure, e.tqPressurePolicy
	p.lastSample, p.sampleObserved, p.lastSampleRecoveryHealthy = sample, true, false
	capacityDrop := !p.capacityObserved && (sample.SlotCapacityDrops > 0 || sample.ByteCapacityDrops > 0) ||
		p.capacityObserved && (sample.SlotCapacityDrops > p.lastSlotDrops || sample.ByteCapacityDrops > p.lastByteDrops)
	if !p.capacityObserved || sample.SlotCapacityDrops >= p.lastSlotDrops && sample.ByteCapacityDrops >= p.lastByteDrops {
		p.lastSlotDrops, p.lastByteDrops = sample.SlotCapacityDrops, sample.ByteCapacityDrops
		p.capacityObserved = true
	} else {
		p.streaks = tqPressureStreaks{}
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseTransportAccounting)
		return
	}
	if capacityDrop {
		p.streaks = tqPressureStreaks{}
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseCapacityDrop)
		return
	}
	if !sample.TQLocalAccountingHealthy {
		p.streaks = tqPressureStreaks{}
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseTransportAccounting)
		return
	}
	s := &p.streaks
	s.degradedFrames = nextPressureStreak(s.degradedFrames, pressureAtLeast(sample.WaitingFrames, sample.FrameCapacity, policy.degradedQueuePercent))
	s.degradedBytes = nextPressureStreak(s.degradedBytes, pressureAtLeast(sample.WaitingBytes, sample.ByteCapacity, policy.degradedQueuePercent))
	s.degradedOldest = nextPressureStreak(s.degradedOldest, sample.OldestWaitingFrameAge >= policy.degradedOldest)
	s.aggregateFrames = nextPressureStreak(s.aggregateFrames, pressureAtLeast(sample.WaitingFrames, sample.FrameCapacity, policy.aggregateQueuePercent))
	s.aggregateBytes = nextPressureStreak(s.aggregateBytes, pressureAtLeast(sample.WaitingBytes, sample.ByteCapacity, policy.aggregateQueuePercent))
	s.aggregateOldest = nextPressureStreak(s.aggregateOldest, sample.OldestWaitingFrameAge >= policy.aggregateOldest)
	healthy := pressureBelow(sample.WaitingFrames, sample.FrameCapacity, policy.recoveryQueuePercent) &&
		pressureBelow(sample.WaitingBytes, sample.ByteCapacity, policy.recoveryQueuePercent) &&
		sample.OldestWaitingFrameAge < policy.recoveryOldest && !e.state.tq.globalBound
	p.lastSampleRecoveryHealthy = healthy
	if p.mode == TQPressureNormal {
		s.healthy = 0
	} else {
		s.healthy = nextPressureStreak(s.healthy, healthy)
	}
	if s.aggregateFrames >= policy.aggregateSamples {
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseWaitingFrames)
		return
	}
	if s.aggregateBytes >= policy.aggregateSamples {
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseWaitingBytes)
		return
	}
	if s.aggregateOldest >= policy.aggregateSamples {
		e.setTQPressureModeLocked(TQPressureAggregateOnly, now, TQPressureCauseOldestWaitingFrame)
		return
	}
	if p.mode == TQPressureNormal {
		if s.degradedFrames >= policy.degradedSamples {
			e.setTQPressureModeLocked(TQPressureDegraded, now, TQPressureCauseWaitingFrames)
		} else if s.degradedBytes >= policy.degradedSamples {
			e.setTQPressureModeLocked(TQPressureDegraded, now, TQPressureCauseWaitingBytes)
		} else if s.degradedOldest >= policy.degradedSamples {
			e.setTQPressureModeLocked(TQPressureDegraded, now, TQPressureCauseOldestWaitingFrame)
		}
		return
	}
	if s.healthy >= policy.recoverySamples {
		e.setTQPressureModeLocked(TQPressureNormal, now, TQPressureCauseNone)
	}
}

func nextPressureStreak(current uint8, present bool) uint8 {
	if !present {
		return 0
	}
	if current < ^uint8(0) {
		return current + 1
	}
	return current
}

func pressureAtLeast(current, capacity, percent uint64) bool {
	return current*100 >= capacity*percent
}

func pressureBelow(current, capacity, percent uint64) bool {
	return current*100 < capacity*percent
}

func (e *Engine) setTQPressureModeLocked(mode TQPressureMode, now time.Time, cause TQPressureCause) {
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
	// Pressure-mode transitions change the protection/trust state visible to a
	// trader. The ordinary sample that led to an unchanged mode remains
	// cadence-coalesced; only this transition is immediate.
	e.markTQTrustTransitionLocked()
	p.mode, p.transitions = mode, p.transitions+1
	if mode == TQPressureNormal {
		p.cause = TQPressureCauseNone
	} else if cause != TQPressureCauseNone {
		p.cause = cause
	}
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
		p.streaks, p.degradedAt = tqPressureStreaks{}, time.Time{}
		if !s.globalBound {
			s.aggregateOnly = false
			s.restoring = true
		}
	}
}
