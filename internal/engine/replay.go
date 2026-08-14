package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact/playback"
)

type ReplayFailureReason string

const (
	ReplayFailureArtifactValidation ReplayFailureReason = "artifact_validation"
	ReplayFailureBindingCoverage    ReplayFailureReason = "binding_coverage"
	ReplayFailureSchemaCanonical    ReplayFailureReason = "schema_canonical_encoding"
	ReplayFailureOrdinalGroup       ReplayFailureReason = "ordinal_group_order"
	ReplayFailureClock              ReplayFailureReason = "clock"
	ReplayFailureAggregate          ReplayFailureReason = "aggregate_admission_disposition"
	ReplayFailureTimer              ReplayFailureReason = "timer_admission_disposition"
	ReplayFailureEngine             ReplayFailureReason = "engine_integrity_suppression"
	ReplayFailureArtifactEnd        ReplayFailureReason = "artifact_end_digest"
	ReplayFailureRequestedEnd       ReplayFailureReason = "requested_end"
	ReplayFailureDrain              ReplayFailureReason = "controlled_stop_drain"
)

type ReplayFailureInput struct {
	Reason      ReplayFailureReason
	ArtifactID  string
	LogicalTime time.Time
	Ordinal     uint64
}

type replayState struct {
	validated, complete, terminal bool
	artifactID, bindingID         string
	start, end, requestedEnd      time.Time
	observationStart              time.Time
	totalRecords, nextOrdinal     uint64
	presentSlots, absentSlots     uint64
	nextGroup, lastGroup          time.Time
	coveredThrough                *time.Time
	failureReason                 ReplayFailureReason
	failureLogical                time.Time
	failureOrdinal                uint64
	completion                    ReplayCompletionDisposition
}

type ReplayCompletionDisposition string

const (
	ReplayCompletionNone         ReplayCompletionDisposition = ""
	ReplayCompletionArtifactEnd  ReplayCompletionDisposition = "artifact_end"
	ReplayCompletionRequestedEnd ReplayCompletionDisposition = "requested_end"
)

type ReplayStatus struct {
	RunMode                 RunMode
	Lifecycle, ArtifactID   string
	CompleteEvidence        bool
	CommittedT              *time.Time
	LastLogicalTime         time.Time
	LastOrdinal             uint64
	LastEngineSequence      uint64
	LastSystemSequence      uint64
	Suppression             SuppressionDisposition
	RankingMode             string
	UniverseTotal           uint64
	TrustedRankableMarks    uint64
	NoPrintThroughT         uint64
	UnknownPopulation       uint64
	Rows                    int
	TQIntentRows            int
	PresentSlots            uint64
	ProvenAbsentSlots       uint64
	FailureReason           ReplayFailureReason
	FailureLogicalTime      time.Time
	FailureOrdinal          uint64
	AggregateInserted       uint64
	AggregateRevised        uint64
	AggregateExactDuplicate uint64
	Evaluation              ReplayEvaluationView
	Publication             ReplayPublicationView
	Completion              ReplayCompletionDisposition
}

func (e *Engine) AdmitReplayStart(ctx context.Context, evidence playback.StartEvidence) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !evidence.Valid() || evidence.ArtifactID() == "" || !validReplayArtifactID(evidence.ArtifactID()) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputReplayStart, replayStart: evidence}, false)
}

func (e *Engine) AdmitReplayRecord(ctx context.Context, evidence playback.RecordEvidence) (AdmissionResult, <-chan AggregateDisposition) {
	e.beginAdmission()
	if ctx == nil || !evidence.Valid() || !validReplayArtifactID(evidence.ArtifactID()) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	values := evidence.Values()
	input := AggregateInput{
		SchemaVersion: AggregateSchemaV1, BindingIdentity: evidence.BindingID(), Source: AggregateSourceReplay,
		Symbol: evidence.Symbol(), WindowStart: evidence.WindowStart(), WindowEnd: evidence.WindowEnd(),
		Values: AggregateValues{Open: values.Open, High: values.High, Low: values.Low, Close: values.Close, Volume: values.Volume, VWAP: values.VWAP,
			AverageTradeSize: values.AverageTradeSize, ATSProvenance: ATSProvenance(values.ATSProvenance)},
		DeliveryTime: evidence.LogicalTime(), Replay: ReplayPosition{ArtifactID: evidence.ArtifactID(), RecordOrdinal: evidence.Ordinal()},
	}
	frozen := freezeAggregateInput(input)
	frozen.replayProof = true
	return e.admitAggregate(ctx, &queueNode{kind: inputAggregate, aggregate: frozen})
}

func (e *Engine) AdmitReplayGroup(ctx context.Context, evidence playback.GroupEvidence) (AdmissionResult, <-chan TimerDisposition) {
	e.beginAdmission()
	if ctx == nil || !evidence.Valid() || !validReplayArtifactID(evidence.ArtifactID()) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputReplayGroup, replayGroup: evidence}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.timerCompletion
}

func (e *Engine) AdmitReplayEnd(ctx context.Context, evidence playback.EndEvidence) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !evidence.Valid() || !validReplayArtifactID(evidence.ArtifactID()) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputReplayEnd, replayEnd: evidence, sealOnLink: true}, false)
}

func (e *Engine) AdmitReplayRequestedEnd(ctx context.Context, evidence playback.RequestedEndEvidence) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !evidence.Valid() || !evidence.Complete() || !validReplayArtifactID(evidence.ArtifactID()) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputReplayRequestedEnd, replayRequestedEnd: evidence, sealOnLink: true}, false)
}

// AdmitReplayFailure is intentionally constructible by the runner: it can
// only remove claims and seal the failed replay instance.
func (e *Engine) AdmitReplayFailure(ctx context.Context, input ReplayFailureInput) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	if ctx == nil || !validReplayFailure(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admit(ctx, &queueNode{kind: inputReplayFailure, replayFailure: input, sealOnLink: true}, false)
}

func (e *Engine) applyReplayStartLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v := node.replayStart
	if e.mode != RunModeReplay || e.state.binding == nil || e.state.lifecycle != lifecycleInitializing || e.state.replay.validated ||
		!v.Valid() || v.BindingID() != e.state.binding.identity || v.Start().Before(e.state.binding.sessionStart) || !v.Start().Before(v.End()) || v.End().After(e.state.binding.sessionEnd) ||
		!v.Start().Before(v.RequestedEnd()) || v.RequestedEnd().After(v.End()) || (v.RequestedEnd().Before(v.End()) && !v.Complete()) ||
		v.Start() != v.Start().UTC() || v.End() != v.End().UTC() || v.Start().Nanosecond() != 0 || v.End().Nanosecond() != 0 {
		return DispositionReplayFailed, ReasonReplayEvidence
	}
	checkpointContinuation := e.state.installedCheckpoint != nil
	if checkpointContinuation {
		if v.Authority() != playback.InstalledCheckpoint || e.state.installedCheckpoint.BindingIdentity != e.state.binding.identity || e.state.installedCheckpoint.T0 != v.Start() {
			return DispositionReplayFailed, ReasonReplayEvidence
		}
	} else if v.Authority() == playback.InstalledCheckpoint || v.Authority() == playback.FreshSession && v.Start() != e.state.binding.sessionStart {
		return DispositionReplayFailed, ReasonReplayEvidence
	}
	observationStart := time.Time{}
	if v.Complete() && e.replayObservationStart != nil {
		if e.replayObservationStart.Before(v.Start()) || e.replayObservationStart.After(v.RequestedEnd()) {
			return DispositionReplayFailed, ReasonReplayEvidence
		}
		observationStart = *e.replayObservationStart
	}
	e.state.replay = replayState{validated: true, complete: v.Complete(), artifactID: v.ArtifactID(), bindingID: v.BindingID(), start: v.Start(), end: v.End(), requestedEnd: v.RequestedEnd(), observationStart: observationStart, totalRecords: v.TotalRecords(), nextOrdinal: 1, nextGroup: v.Start()}
	e.state.replayArtifact = v.ArtifactID()
	if v.Complete() && !checkpointContinuation {
		if e.state.aggregateEvaluator.coverage == nil {
			e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence, len(e.state.binding.symbols))
		}
		for index := range e.state.binding.symbols {
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
		}
	}
	if !e.transitionLifecycleLocked(lifecycleEventReplayStart, node, lifecycleReasonReplayStart) {
		return DispositionReplayFailed, ReasonLifecycle
	}
	return DispositionReplayStarted, ReasonNone
}

func wholeSecondUTC(value time.Time) bool {
	return !value.IsZero() && value == value.UTC() && value.Nanosecond() == 0
}

func (e *Engine) completeFinalReplayRecordLocked(node *queueNode) bool {
	return node != nil && node.kind == inputAggregate && node.aggregate.replayProof && e.mode == RunModeReplay && e.state.replay.complete
}

func (e *Engine) replayFastForwardGroupLocked(node *queueNode) bool {
	return node != nil && node.kind == inputReplayGroup && e.mode == RunModeReplay && e.state.replay.complete &&
		!e.state.replay.observationStart.IsZero() && node.admissionTime.Before(e.state.replay.observationStart)
}

func (e *Engine) completeFinalReplayObservationGroupLocked(node *queueNode) bool {
	return node != nil && node.kind == inputReplayGroup && e.mode == RunModeReplay && e.state.replay.complete &&
		!e.state.replay.observationStart.IsZero() && !node.admissionTime.Before(e.state.replay.observationStart)
}

func (e *Engine) completeFinalReplayQualificationPreparedLocked(state *symbolAggregateState, at time.Time) bool {
	return e.mode == RunModeReplay && e.state.replay.complete && !e.state.replay.observationStart.IsZero() &&
		!e.state.replay.lastGroup.Before(e.state.replay.observationStart) && state != nil && state.qualification != nil &&
		state.qualification.result.at.Equal(at) && !state.qualification.accountedThrough.Before(at)
}

func (e *Engine) applyReplayGroupLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, state := node.replayGroup, &e.state.replay
	if e.mode != RunModeReplay || !state.validated || state.terminal || e.state.lifecycle != lifecycleReplaying || !v.Valid() ||
		v.ArtifactID() != state.artifactID || v.BindingID() != state.bindingID || v.Complete() != state.complete || v.LogicalTime() != state.nextGroup || v.LogicalTime().After(state.requestedEnd) ||
		v.LastOrdinal()+1 != state.nextOrdinal || node.admissionTime != v.LogicalTime() {
		return DispositionReplayFailed, ReasonReplayEvidence
	}
	if state.complete && v.LogicalTime().After(state.start) {
		e.classifyReplaySlotLocked(v.LogicalTime().Add(-time.Second))
	}
	state.lastGroup = v.LogicalTime()
	state.nextGroup = v.LogicalTime().Add(time.Second)
	if state.complete {
		state.coveredThrough = immutableTime(v.LogicalTime())
	}
	return e.applyTimerLocked(node)
}

func (e *Engine) classifyReplaySlotLocked(start time.Time) {
	for index := range e.state.binding.symbols {
		symbol := &e.state.binding.symbols[index]
		state := ensureAggregateState(symbol)
		slot := sessionSlot(e.state.binding, start)
		record := state.tail[start.Unix()]
		present := record != nil && record.windowStart == start && record.authority.source == AggregateSourceReplay && record.authority.replay.ArtifactID == e.state.replay.artifactID
		if present {
			presence := ensurePresence(state)
			if !presence.has(slot) {
				presence.set(slot)
				e.state.replay.presentSlots++
			}
			if state.provenAbsent != nil {
				if state.provenAbsent.has(slot) {
					e.state.replay.absentSlots--
				}
				state.provenAbsent.clear(slot)
			}
			delete(e.state.aggregateEvaluator.coverage, index)
			continue
		}
		absence := ensureProvenAbsent(state)
		if !absence.has(slot) {
			absence.set(slot)
			e.state.replay.absentSlots++
		}
		if state.latest == nil || state.latest.record.windowEnd.After(start.Add(time.Second)) {
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
		}
	}
}

func (e *Engine) applyReplayEndLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, state := node.replayEnd, &e.state.replay
	if e.mode != RunModeReplay || !state.validated || state.terminal || e.state.lifecycle != lifecycleReplaying || !v.Valid() ||
		v.ArtifactID() != state.artifactID || v.BindingID() != state.bindingID || v.Complete() != state.complete || v.End() != state.end ||
		v.TotalRecords() != state.totalRecords || state.nextOrdinal != state.totalRecords+1 || state.lastGroup != state.end {
		return DispositionReplayFailed, ReasonReplayEvidence
	}
	state.terminal = true
	if !e.transitionLifecycleLocked(lifecycleEventReplayEnd, node, lifecycleReasonReplayEnd) {
		return DispositionReplayFailed, ReasonLifecycle
	}
	state.completion = ReplayCompletionArtifactEnd
	return DispositionReplayEnded, ReasonNone
}

func (e *Engine) applyReplayRequestedEndLocked(node *queueNode) (DispositionCode, DispositionReason) {
	v, state := node.replayRequestedEnd, &e.state.replay
	if e.mode != RunModeReplay || !state.validated || !state.complete || state.terminal || e.state.lifecycle != lifecycleReplaying || !v.Valid() ||
		v.ArtifactID() != state.artifactID || v.BindingID() != state.bindingID || v.ArtifactEnd() != state.end || v.RequestedEnd() != state.requestedEnd ||
		!v.RequestedEnd().Before(v.ArtifactEnd()) || node.admissionTime != state.requestedEnd || v.TotalRecords() != state.totalRecords ||
		v.PrefixRecords()+1 != state.nextOrdinal || state.lastGroup != state.requestedEnd {
		return DispositionReplayFailed, ReasonReplayEvidence
	}
	state.terminal = true
	if !e.transitionLifecycleLocked(lifecycleEventReplayRequestedEnd, node, lifecycleReasonReplayRequestedEnd) {
		return DispositionReplayFailed, ReasonLifecycle
	}
	state.completion = ReplayCompletionRequestedEnd
	return DispositionReplayRequestedEnd, ReasonNone
}

func (e *Engine) applyReplayFailureLocked(node *queueNode) (DispositionCode, DispositionReason) {
	if e.mode != RunModeReplay || e.state.binding == nil || e.state.lifecycle == lifecycleEnded {
		return DispositionReplayFailed, ReasonLifecycle
	}
	e.state.replay.terminal = true
	if e.state.replay.artifactID == "" {
		e.state.replay.artifactID = node.replayFailure.ArtifactID
	}
	e.state.replay.failureReason = node.replayFailure.Reason
	e.state.replay.failureLogical = node.replayFailure.LogicalTime
	e.state.replay.failureOrdinal = node.replayFailure.Ordinal
	disposition := e.enterSuppressionLocked(lifecycleEventReplayFailure, node, lifecycleReasonReplayFailure)
	_ = disposition
	return DispositionReplayFailed, ReasonReplayEvidence
}

func (e *Engine) ObserveReplay() ReplayStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := ReplayStatus{RunMode: e.mode, Lifecycle: string(e.state.lifecycle), ArtifactID: e.state.replay.artifactID, CompleteEvidence: e.state.replay.complete,
		CommittedT: immutableTimePointer(e.state.committedT), LastLogicalTime: e.state.replay.lastGroup, LastEngineSequence: e.nextSequence - 1,
		LastSystemSequence: e.lastSystem, Suppression: e.state.suppressionDisposition}
	if e.state.replay.nextOrdinal > 0 {
		result.LastOrdinal = e.state.replay.nextOrdinal - 1
	}
	evaluation := e.state.aggregateEvaluator.current
	result.RankingMode = string(evaluation.mode)
	result.UniverseTotal = evaluation.population.universeTotal
	result.TrustedRankableMarks = evaluation.population.trustedRankableMark
	result.NoPrintThroughT = evaluation.population.noPrintThroughT
	result.UnknownPopulation = evaluation.population.unknownDueFailureOrFence
	result.Rows = len(evaluation.rows)
	for _, row := range evaluation.rows {
		if row.tqIntentEligible {
			result.TQIntentRows++
		}
	}
	result.PresentSlots = e.state.replay.presentSlots
	result.ProvenAbsentSlots = e.state.replay.absentSlots
	result.FailureReason = e.state.replay.failureReason
	result.FailureLogicalTime = e.state.replay.failureLogical
	result.FailureOrdinal = e.state.replay.failureOrdinal
	result.Completion = e.state.replay.completion
	result.AggregateInserted = e.state.aggregates.inserted
	result.AggregateRevised = e.state.aggregates.revised
	result.AggregateExactDuplicate = e.state.aggregates.exactDuplicate
	result.Evaluation = replayEvaluationView(e.state.aggregateEvaluator.current)
	result.Publication = replayPublicationView(e.publication.Load())
	return result
}

func validReplayArtifactID(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || value[:len("sha256:")] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil
}

func validReplayFailure(input ReplayFailureInput) bool {
	switch input.Reason {
	case ReplayFailureArtifactValidation, ReplayFailureBindingCoverage, ReplayFailureSchemaCanonical, ReplayFailureOrdinalGroup, ReplayFailureClock,
		ReplayFailureAggregate, ReplayFailureTimer, ReplayFailureEngine, ReplayFailureArtifactEnd, ReplayFailureRequestedEnd, ReplayFailureDrain:
	default:
		return false
	}
	return (input.ArtifactID == "" || validReplayArtifactID(input.ArtifactID)) && (input.LogicalTime.IsZero() || input.LogicalTime == input.LogicalTime.UTC())
}
