package engine

import (
	"context"
	"math"
	"math/bits"
	"sort"
	"time"
)

const (
	AggregateSchemaV1   = "normalized-second-aggregate-v1"
	correctionHorizon   = 16 * time.Minute
	sessionSeconds      = 16 * 60 * 60
	maximumTailRecords  = 961
	maximumContextBytes = 256
)

type AggregateSource string

const (
	AggregateSourceLive       AggregateSource = "live"
	AggregateSourceHistorical AggregateSource = "historical"
)

type ATSProvenance string

const (
	ATSLiveProviderAverage       ATSProvenance = "live_provider_average"
	ATSRESTFloorVolumeOverTrades ATSProvenance = "rest_floor_volume_over_transactions"
)

type AggregateValues struct {
	Open, High, Low, Close float64
	Volume, VWAP           float64
	AverageTradeSize       int64
	ATSProvenance          ATSProvenance
}

type LivePosition struct {
	ConnectionEpoch uint64
	FrameSequence   uint64
	ArrayIndex      uint32
}

type HistoricalPosition struct {
	Generation    uint64
	RequestToken  string
	RecordOrdinal uint64
}

// AggregateInput is one immutable-by-admission normalized aggregate fact.
// Source-specific unused positions must be zero.
type AggregateInput struct {
	SchemaVersion, BindingIdentity string
	Source                         AggregateSource
	Symbol                         string
	WindowStart, WindowEnd         time.Time
	Values                         AggregateValues
	DeliveryTime                   time.Time
	Live                           LivePosition
	Historical                     HistoricalPosition
}

type DispositionReason string

// AggregateDisposition is the immutable completion for one consumed aggregate.
// It extends, rather than changes, the accepted S1 Disposition API.
type AggregateDisposition struct {
	EngineSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
}

const (
	ReasonNone                         DispositionReason = ""
	ReasonSchema                       DispositionReason = "schema"
	ReasonBinding                      DispositionReason = "binding"
	ReasonRunSource                    DispositionReason = "run_source"
	ReasonSymbol                       DispositionReason = "symbol"
	ReasonWindow                       DispositionReason = "window"
	ReasonStructural                   DispositionReason = "structural"
	ReasonDeliveryEvidence             DispositionReason = "delivery_evidence"
	ReasonSourcePosition               DispositionReason = "source_position"
	ReasonLifecycle                    DispositionReason = "lifecycle"
	ReasonStaleLiveEpoch               DispositionReason = "stale_live_epoch"
	ReasonHistoricalContext            DispositionReason = "historical_context"
	ReasonFutureEventTime              DispositionReason = "future_event_time"
	ReasonTooLate                      DispositionReason = "too_late"
	ReasonNonprecedent                 DispositionReason = "nonprecedent"
	ReasonRepeatedPositionUnequal      DispositionReason = "repeated_position_unequal"
	ReasonHistoricalLiveConflict       DispositionReason = "historical_live_conflict"
	ReasonHistoricalHistoricalConflict DispositionReason = "historical_historical_conflict"
	ReasonClockRegression              DispositionReason = "clock_regression"
	ReasonPublication                  DispositionReason = "publication"
	ReasonAccounting                   DispositionReason = "accounting"
	ReasonPressure                     DispositionReason = "pressure"
	ReasonTerminal                     DispositionReason = "terminal"
)

type frozenAggregateInput struct {
	AggregateInput
	historicalProof *frozenHistoricalProofContext
	s2Proof         bool
	c6Historical    bool
}

func freezeAggregateInput(input AggregateInput) frozenAggregateInput {
	return frozenAggregateInput{AggregateInput: input}
}

func boundedAggregateInput(input AggregateInput) bool {
	return len(input.SchemaVersion) <= 64 && len(input.BindingIdentity) <= maximumContextBytes && len(input.Symbol) <= maximumSymbolBytes &&
		len(input.Source) <= 32 && len(input.Values.ATSProvenance) <= 64 &&
		len(input.Historical.RequestToken) <= maximumContextBytes
}

type aggregateIdentity struct {
	symbol string
	start  int64
}

type aggregateEvidence struct {
	source       AggregateSource
	restored     bool
	deliveryTime time.Time
	live         LivePosition
	historical   HistoricalPosition
}

type canonicalAggregate struct {
	identity            aggregateIdentity
	windowStart         time.Time
	windowEnd           time.Time
	values              AggregateValues
	first               aggregateEvidence
	authority           aggregateEvidence
	greatestLiveSupport *LivePosition
}

type latestAggregateMark struct {
	record canonicalAggregate
}

type committedAggregateMark struct {
	symbol                 string
	start                  int64
	windowStart, windowEnd time.Time
	values                 AggregateValues
	authority              aggregateEvidence
	greatestLiveSupport    *LivePosition
}

type slotBitmap [sessionSeconds / 64]uint64

type evaluationTailWindow struct {
	baseWord  int
	tailCount int
	words     [16]uint64
	valid     bool
}

func (b *slotBitmap) set(slot int)   { b[slot/64] |= uint64(1) << uint(slot%64) }
func (b *slotBitmap) clear(slot int) { b[slot/64] &^= uint64(1) << uint(slot%64) }
func (b *slotBitmap) has(slot int) bool {
	return b != nil && b[slot/64]&(uint64(1)<<uint(slot%64)) != 0
}

type symbolAggregateState struct {
	prefix             aggregatePrefix
	tail               map[int64]*canonicalAggregate
	presence           *slotBitmap
	sealedLive         *slotBitmap
	provenAbsent       *slotBitmap
	historicalConflict *slotBitmap
	latest             *latestAggregateMark
	olderLatest        *canonicalAggregate
	committedLatest    *committedAggregateMark
	selectionAt        time.Time
	priorSelectionAt   time.Time
	priorSelectionMark *committedAggregateMark
	recomputations     uint64
	canonicalRevision  uint64
	affected           aggregateAffectedState
	tailBoundHits      uint64
	lastConflict       *aggregateConflictEvidence
	invalidStarts      []int64
	// Current-product aggregate sufficient state remains selection-independent.
	// Canonical merge owns the facts; selected-row enrichment is its only value
	// consumer.
	priceRange      *priceRangeFeatureState
	mvpMeasurements *mvpMeasurementState
	qualification   *qualificationState
	// tailCoverage is bounded derived acceleration for the at-most-961-second
	// canonical correction tail. It is rebuilt from tail and never persisted.
	tailCoverage       evaluationTailWindow
	tailCoverageBuilt  bool
	tailCoverageUsable bool
	// evaluationTailPresence exists only on a transition-local shallow copy.
	// It is built once from canonical tail and is never applied to canonical state.
	evaluationTailPresence *evaluationTailWindow
}

type invalidMarkEvidence struct {
	windowStart time.Time
}

type aggregateConflictEvidence struct {
	identity aggregateIdentity
	current  aggregateEvidence
	incoming aggregateEvidence
}

type aggregateAccounting struct {
	consumed, inserted, revised, withdrawnConflict uint64
	exactDuplicate, rejected, fenced, integrity    uint64
}

func (a aggregateAccounting) reconciles() bool {
	return a.consumed == a.inserted+a.revised+a.withdrawnConflict+a.exactDuplicate+a.rejected+a.fenced+a.integrity
}

type historicalProofContext struct {
	bindingID, token, symbol   string
	generation                 uint64
	intervalStart, intervalEnd time.Time
	result                     *historicalProofResult
}

type frozenHistoricalProofContext struct {
	bindingID, token, symbol   string
	generation                 uint64
	intervalStart, intervalEnd time.Time
	resultCardinality          int
	resultValid                bool
	record                     *canonicalAggregate
	conflicted                 bool
}

// historicalProofResult is package-private caller-side S2 proof setup. An
// admission extracts and deep-copies only the target identity plus bounded
// cardinality; neither map nor a pointer to this result enters the FIFO or
// canonical state. Component 6's hydration generation owns the production
// active-result ledger that authorizes this path.
type historicalProofResult struct {
	records   map[aggregateIdentity]canonicalAggregate
	conflicts map[aggregateIdentity]struct{}
}

func (e *Engine) admitAggregateForProof(input AggregateInput) (AdmissionResult, <-chan AggregateDisposition) {
	e.beginAdmission()
	if !boundedAggregateInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	frozen := freezeAggregateInput(input)
	frozen.s2Proof = true
	return e.admitAggregate(context.Background(), &queueNode{kind: inputAggregate, aggregate: frozen})
}

func (e *Engine) admitHistoricalAggregateForProof(input AggregateInput, proof historicalProofContext) (AdmissionResult, <-chan AggregateDisposition) {
	e.beginAdmission()
	if !boundedAggregateInput(input) || len(proof.bindingID) > maximumContextBytes || len(proof.token) > maximumContextBytes || len(proof.symbol) > maximumSymbolBytes {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	frozen := freezeAggregateInput(input)
	frozen.historicalProof = freezeHistoricalProofContext(input, proof)
	frozen.s2Proof = true
	return e.admitAggregate(context.Background(), &queueNode{kind: inputAggregate, aggregate: frozen})
}

func freezeHistoricalProofContext(input AggregateInput, proof historicalProofContext) *frozenHistoricalProofContext {
	frozen := &frozenHistoricalProofContext{
		bindingID: proof.bindingID, token: proof.token, symbol: proof.symbol,
		generation: proof.generation, intervalStart: proof.intervalStart, intervalEnd: proof.intervalEnd,
	}
	if proof.result == nil {
		return frozen
	}
	frozen.resultValid = proof.result.records != nil && proof.result.conflicts != nil
	maximumInt := int(^uint(0) >> 1)
	if len(proof.result.records) > maximumInt-len(proof.result.conflicts) {
		frozen.resultCardinality = maximumInt
	} else {
		frozen.resultCardinality = len(proof.result.records) + len(proof.result.conflicts)
	}
	identity := aggregateIdentity{symbol: input.Symbol, start: input.WindowStart.Unix()}
	if record, ok := proof.result.records[identity]; ok {
		copyRecord := record
		if record.greatestLiveSupport != nil {
			position := *record.greatestLiveSupport
			copyRecord.greatestLiveSupport = &position
		}
		frozen.record = &copyRecord
	}
	_, frozen.conflicted = proof.result.conflicts[identity]
	return frozen
}

func (e *Engine) applyAggregateLocked(input frozenAggregateInput, now time.Time) (DispositionCode, DispositionReason) {
	e.state.aggregates.consumed++
	code, reason := e.decideAggregateLocked(input, now)
	e.updateInvalidMarkEvidenceLocked(input, now, code, reason)
	switch code {
	case DispositionAggregateInserted:
		e.state.aggregates.inserted++
	case DispositionAggregateRevised:
		e.state.aggregates.revised++
	case DispositionAggregateWithdrawn:
		e.state.aggregates.withdrawnConflict++
	case DispositionAggregateExactDuplicate:
		e.state.aggregates.exactDuplicate++
	case DispositionAggregateRejected:
		e.state.aggregates.rejected++
	case DispositionAggregateFenced:
		e.state.aggregates.fenced++
	case DispositionAggregateIntegrity:
		e.state.aggregates.integrity++
	}
	return code, reason
}

func (e *Engine) updateInvalidMarkEvidenceLocked(input frozenAggregateInput, now time.Time, code DispositionCode, reason DispositionReason) {
	if e.state.binding == nil {
		return
	}
	index, ok := e.state.binding.index[input.Symbol]
	if !ok {
		return
	}
	if code == DispositionAggregateRejected && reason == ReasonStructural &&
		input.BindingIdentity == e.state.binding.identity && e.validAggregateLifecycleLocked(input) &&
		validAggregateWindow(input.AggregateInput, e.state.binding) && !input.WindowEnd.After(now) &&
		!input.DeliveryTime.IsZero() && input.DeliveryTime == input.DeliveryTime.UTC() {
		contextReason, _ := e.validateSourceContextLocked(input)
		if contextReason != ReasonNone {
			return
		}
		if e.state.aggregateEvaluator.invalidMarks == nil {
			e.state.aggregateEvaluator.invalidMarks = make(map[int]invalidMarkEvidence)
		}
		invalid := invalidMarkEvidence{windowStart: input.WindowStart}
		state := ensureAggregateState(&e.state.binding.symbols[index])
		insertInvalidStart(state, input.WindowStart.Unix())
		currentRelevant := e.state.committedT == nil || input.WindowStart.Before(*e.state.committedT)
		hadAbsence := state.provenAbsent != nil && state.provenAbsent.has(sessionSlot(e.state.binding, invalid.windowStart))
		if state.provenAbsent != nil {
			state.provenAbsent.clear(sessionSlot(e.state.binding, invalid.windowStart))
		}
		if e.state.hydration.supportedThrough != nil && invalid.windowStart.Before(*e.state.hydration.supportedThrough) {
			if e.state.aggregateEvaluator.coverage == nil {
				e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence)
			}
			priorCoverage, exists := e.state.aggregateEvaluator.coverage[index]
			if !exists || priorCoverage.outcome != coverageOutcomeUnknown || priorCoverage.origin == uncertaintyNone {
				e.state.aggregateEvaluator.coverage[index] = coverageUnknownPostBootstrap
			}
		}
		prior, exists := e.state.aggregateEvaluator.invalidMarks[index]
		replaced := !exists || input.WindowStart.After(prior.windowStart)
		if replaced {
			e.state.aggregateEvaluator.invalidMarks[index] = invalid
		}
		if currentRelevant && (hadAbsence || replaced) {
			state.notifyAggregate(canonicalAggregate{identity: aggregateIdentity{symbol: input.Symbol, start: input.WindowStart.Unix()}, windowStart: input.WindowStart, windowEnd: input.WindowEnd}, aggregateProofAll)
		}
		return
	}
	if code == DispositionAggregateInserted || code == DispositionAggregateRevised {
		state := ensureAggregateState(&e.state.binding.symbols[index])
		removeInvalidStartsThrough(state, input.WindowStart.Unix())
		if prior, exists := e.state.aggregateEvaluator.invalidMarks[index]; exists && !input.WindowStart.Before(prior.windowStart) {
			delete(e.state.aggregateEvaluator.invalidMarks, index)
		}
	}
}

func insertInvalidStart(state *symbolAggregateState, start int64) {
	if state == nil {
		return
	}
	index := sort.Search(len(state.invalidStarts), func(index int) bool { return state.invalidStarts[index] >= start })
	if index < len(state.invalidStarts) && state.invalidStarts[index] == start {
		return
	}
	state.invalidStarts = append(state.invalidStarts, 0)
	copy(state.invalidStarts[index+1:], state.invalidStarts[index:])
	state.invalidStarts[index] = start
	if len(state.invalidStarts) > maximumTailRecords {
		state.invalidStarts = state.invalidStarts[len(state.invalidStarts)-maximumTailRecords:]
	}
}

func removeInvalidStartsThrough(state *symbolAggregateState, through int64) {
	if state == nil || len(state.invalidStarts) == 0 {
		return
	}
	index := sort.Search(len(state.invalidStarts), func(index int) bool { return state.invalidStarts[index] > through })
	state.invalidStarts = append(state.invalidStarts[:0], state.invalidStarts[index:]...)
}

func (e *Engine) invalidMarkBeforeLocked(index int, at time.Time) (invalidMarkEvidence, bool) {
	if e.state.binding == nil || index < 0 || index >= len(e.state.binding.symbols) {
		return invalidMarkEvidence{}, false
	}
	state := e.state.binding.symbols[index].aggregates
	if state != nil && len(state.invalidStarts) != 0 && !at.IsZero() {
		position := sort.Search(len(state.invalidStarts), func(position int) bool { return state.invalidStarts[position] >= at.Unix() })
		if position > 0 {
			return invalidMarkEvidence{windowStart: time.Unix(state.invalidStarts[position-1], 0).UTC()}, true
		}
		return invalidMarkEvidence{}, false
	}
	evidence, ok := e.state.aggregateEvaluator.invalidMarks[index]
	if ok && !at.IsZero() && !evidence.windowStart.Before(at) {
		return invalidMarkEvidence{}, false
	}
	return evidence, ok
}

func (e *Engine) decideAggregateLocked(input frozenAggregateInput, now time.Time) (DispositionCode, DispositionReason) {
	binding := e.state.binding
	if binding == nil {
		return DispositionAggregateRejected, ReasonLifecycle
	}
	if e.state.aggregateIntegrity {
		return DispositionAggregateIntegrity, ReasonRepeatedPositionUnequal
	}
	if !input.s2Proof && input.Source == AggregateSourceLive && e.state.liveEpoch != 0 &&
		(!e.state.liveEpochActive || input.Live.ConnectionEpoch != e.state.liveEpoch) {
		return DispositionAggregateFenced, ReasonStaleLiveEpoch
	}
	if !e.validAggregateLifecycleLocked(input) {
		return DispositionAggregateRejected, ReasonLifecycle
	}
	if input.SchemaVersion != AggregateSchemaV1 {
		return DispositionAggregateRejected, ReasonSchema
	}
	if input.BindingIdentity != binding.identity {
		return DispositionAggregateFenced, ReasonBinding
	}
	if input.Source != AggregateSourceLive && input.Source != AggregateSourceHistorical {
		return DispositionAggregateRejected, ReasonRunSource
	}
	index, ok := binding.index[input.Symbol]
	if !ok {
		return DispositionAggregateRejected, ReasonSymbol
	}
	if !validAggregateWindow(input.AggregateInput, binding) {
		return DispositionAggregateRejected, ReasonWindow
	}
	if !validAggregateValues(input.Values) {
		return DispositionAggregateRejected, ReasonStructural
	}
	if input.DeliveryTime.IsZero() || input.DeliveryTime != input.DeliveryTime.UTC() {
		return DispositionAggregateRejected, ReasonDeliveryEvidence
	}
	if reason, fenced := e.validateSourceContextLocked(input); reason != ReasonNone {
		if fenced {
			return DispositionAggregateFenced, reason
		}
		return DispositionAggregateRejected, reason
	}
	if input.WindowEnd.After(now) {
		return DispositionAggregateRejected, ReasonFutureEventTime
	}
	symbol := &binding.symbols[index]
	state := symbol.aggregates
	identityStart := input.WindowStart.Unix()
	var existing *canonicalAggregate
	if state != nil && state.tail != nil {
		existing = state.tail[identityStart]
	}
	if existing == nil && state != nil && state.latest != nil && state.latest.record.identity.start == identityStart {
		existing = &state.latest.record
	}
	if existing == nil && state != nil && state.olderLatest != nil && state.olderLatest.identity.start == identityStart {
		existing = state.olderLatest
	}
	if existing == nil && input.Source == AggregateSourceHistorical && input.historicalProof != nil {
		if input.historicalProof.record != nil {
			copyRecord := *input.historicalProof.record
			existing = &copyRecord
		}
		if input.historicalProof.conflicted {
			return DispositionAggregateRejected, ReasonHistoricalHistoricalConflict
		}
	}
	if state != nil && input.Source == AggregateSourceHistorical && state.sealedLive != nil && state.sealedLive.has(sessionSlot(binding, input.WindowStart)) {
		if sealed, ok := e.takeHydrationReconciliationLocked(input); ok && aggregateValuesEqual(sealed, input.Values) {
			return DispositionAggregateExactDuplicate, ReasonNone
		}
		state.lastConflict = &aggregateConflictEvidence{identity: aggregateIdentity{symbol: input.Symbol, start: identityStart}, current: aggregateEvidence{source: AggregateSourceLive}, incoming: evidence(input)}
		return DispositionAggregateRejected, ReasonHistoricalLiveConflict
	}
	// An old identity supported only by historical evidence must not acquire
	// live authority through the equality fast path. Existing live authority may
	// still classify a duplicate, but an unequal revision remains too late.
	if input.Source == AggregateSourceLive && now.Sub(input.WindowEnd) > correctionHorizon &&
		(existing == nil || existing.authority.source != AggregateSourceLive && existing.greatestLiveSupport == nil) {
		return DispositionAggregateRejected, ReasonTooLate
	}
	// A live identity strictly outside the correction horizon can never reopen
	// the sealed prefix. Classify it by the time boundary before consulting
	// compact coverage evidence, which does not retain the discarded raw row.
	if existing == nil && input.Source != AggregateSourceHistorical && now.Sub(input.WindowEnd) > correctionHorizon {
		return DispositionAggregateRejected, ReasonTooLate
	}
	if state != nil && existing == nil && state.presence != nil && state.presence.has(sessionSlot(binding, input.WindowStart)) {
		return DispositionAggregateFenced, ReasonHistoricalContext
	}
	if state != nil && existing == nil && input.Source == AggregateSourceHistorical && state.historicalConflict != nil && state.historicalConflict.has(sessionSlot(binding, input.WindowStart)) {
		return DispositionAggregateRejected, ReasonHistoricalHistoricalConflict
	}

	if existing != nil && aggregateValuesEqual(existing.values, input.Values) {
		if input.Source == AggregateSourceLive {
			advanceLiveAuthority(existing, input)
			e.state.liveEpoch = input.Live.ConnectionEpoch
			if state != nil && state.latest != nil && state.latest.record.identity == existing.identity {
				state.latest.record = *existing
			}
		}
		return DispositionAggregateExactDuplicate, ReasonNone
	}

	if input.Source != AggregateSourceHistorical && now.Sub(input.WindowEnd) > correctionHorizon {
		return DispositionAggregateRejected, ReasonTooLate
	}

	if existing != nil {
		switch input.Source {
		case AggregateSourceLive:
			if existing.authority.restored || existing.authority.source == AggregateSourceHistorical {
				return e.installAggregateLocked(symbol, input, now, true)
			}
			comparison := compareLive(input.Live, existing.authority.live)
			if comparison < 0 {
				return DispositionAggregateRejected, ReasonNonprecedent
			}
			if comparison == 0 {
				return e.integrityWithdrawLocked(symbol, existing), ReasonRepeatedPositionUnequal
			}
		case AggregateSourceHistorical:
			if existing.authority.source == AggregateSourceLive {
				if state != nil {
					state.lastConflict = &aggregateConflictEvidence{identity: existing.identity, current: existing.authority, incoming: evidence(input)}
				}
				return DispositionAggregateRejected, ReasonHistoricalLiveConflict
			}
			return e.historicalWithdrawLocked(symbol, existing, input), ReasonHistoricalHistoricalConflict
		}
		return e.installAggregateLocked(symbol, input, now, true)
	}
	return e.installAggregateLocked(symbol, input, now, false)
}

func (e *Engine) validAggregateLifecycleLocked(input frozenAggregateInput) bool {
	if input.c6Historical {
		return (e.state.lifecycle == lifecycleHydrating || e.state.lifecycle == lifecycleRecovering)
	}
	if input.s2Proof {
		return e.state.lifecycle == lifecycleAwaitingAggregateAck
	}
	return input.Source == AggregateSourceLive && e.state.liveEpochActive && e.state.aggregateAcknowledged &&
		(e.state.lifecycle == lifecycleHydrating || e.state.lifecycle == lifecycleLive || e.state.lifecycle == lifecycleRecovering)
}

func (e *Engine) validateSourceContextLocked(input frozenAggregateInput) (DispositionReason, bool) {
	switch input.Source {
	case AggregateSourceLive:
		if input.Live.ConnectionEpoch == 0 || input.Live.FrameSequence == 0 || input.Historical != (HistoricalPosition{}) {
			return ReasonSourcePosition, false
		}
		if e.state.liveEpoch != 0 && input.Live.ConnectionEpoch < e.state.liveEpoch {
			return ReasonStaleLiveEpoch, true
		}
		if !input.s2Proof && input.Live.ConnectionEpoch != e.state.liveEpoch {
			return ReasonStaleLiveEpoch, true
		}
		if !input.s2Proof && (!e.state.aggregateAcknowledged || compareLive(input.Live, e.state.aggregateAckPosition) <= 0) {
			return ReasonAggregateBeforeAck, false
		}
	case AggregateSourceHistorical:
		proof := input.historicalProof
		if proof == nil || !proof.resultValid || input.Historical.Generation == 0 || input.Historical.RequestToken == "" || input.Historical.RecordOrdinal == 0 || input.Live != (LivePosition{}) {
			return ReasonHistoricalContext, true
		}
		if proof.bindingID != input.BindingIdentity || proof.generation != input.Historical.Generation || proof.token != input.Historical.RequestToken || proof.symbol != input.Symbol ||
			proof.intervalStart != proof.intervalStart.UTC() || proof.intervalEnd != proof.intervalEnd.UTC() || proof.intervalStart.Nanosecond() != 0 || proof.intervalEnd.Nanosecond() != 0 ||
			proof.intervalStart.Before(e.state.binding.sessionStart) || proof.intervalEnd.After(e.state.binding.sessionEnd) || !proof.intervalStart.Before(proof.intervalEnd) ||
			proof.resultCardinality > int(proof.intervalEnd.Sub(proof.intervalStart)/time.Second) || input.WindowStart.Before(proof.intervalStart) || !input.WindowStart.Before(proof.intervalEnd) {
			return ReasonHistoricalContext, true
		}
	}
	return ReasonNone, false
}

func validAggregateWindow(input AggregateInput, binding *installedBinding) bool {
	return input.WindowStart == input.WindowStart.UTC() && input.WindowEnd == input.WindowEnd.UTC() &&
		input.WindowStart.Nanosecond() == 0 && input.WindowEnd.Nanosecond() == 0 &&
		input.WindowEnd.Sub(input.WindowStart) == time.Second &&
		!input.WindowStart.Before(binding.sessionStart) && !input.WindowEnd.After(binding.sessionEnd)
}

func validAggregateValues(v AggregateValues) bool {
	finitePositive := func(x float64) bool { return x > 0 && !math.IsNaN(x) && !math.IsInf(x, 0) }
	finiteNonnegative := func(x float64) bool { return x >= 0 && !math.IsNaN(x) && !math.IsInf(x, 0) }
	return finitePositive(v.Open) && finitePositive(v.High) && finitePositive(v.Low) && finitePositive(v.Close) &&
		finiteNonnegative(v.Volume) && finitePositive(v.VWAP) && v.AverageTradeSize >= 0 &&
		(v.ATSProvenance == ATSLiveProviderAverage || v.ATSProvenance == ATSRESTFloorVolumeOverTrades) &&
		v.Low <= v.Open && v.Open <= v.High && v.Low <= v.Close && v.Close <= v.High
}

func aggregateValuesEqual(a, b AggregateValues) bool {
	return a.Open == b.Open && a.High == b.High && a.Low == b.Low && a.Close == b.Close && a.Volume == b.Volume && a.VWAP == b.VWAP && a.AverageTradeSize == b.AverageTradeSize && a.ATSProvenance == b.ATSProvenance
}

func compareLive(a, b LivePosition) int {
	if a.ConnectionEpoch != b.ConnectionEpoch {
		if a.ConnectionEpoch < b.ConnectionEpoch {
			return -1
		}
		return 1
	}
	if a.FrameSequence != b.FrameSequence {
		if a.FrameSequence < b.FrameSequence {
			return -1
		}
		return 1
	}
	if a.ArrayIndex < b.ArrayIndex {
		return -1
	}
	if a.ArrayIndex > b.ArrayIndex {
		return 1
	}
	return 0
}

func evidence(input frozenAggregateInput) aggregateEvidence {
	return aggregateEvidence{source: input.Source, deliveryTime: input.DeliveryTime, live: input.Live, historical: input.Historical}
}

func advanceLiveAuthority(record *canonicalAggregate, input frozenAggregateInput) {
	if record.greatestLiveSupport == nil || compareLive(input.Live, *record.greatestLiveSupport) > 0 {
		position := input.Live
		record.greatestLiveSupport = &position
		record.authority = evidence(input)
	}
}

func (e *Engine) installAggregateLocked(symbol *coreSymbol, input frozenAggregateInput, now time.Time, revision bool) (DispositionCode, DispositionReason) {
	state := ensureAggregateState(symbol)
	e.compactSymbolLocked(state, e.state.binding, symbol.symbol, now)
	record := canonicalAggregate{identity: aggregateIdentity{symbol: input.Symbol, start: input.WindowStart.Unix()}, windowStart: input.WindowStart, windowEnd: input.WindowEnd, values: input.Values, first: evidence(input), authority: evidence(input)}
	if old := state.tail[record.identity.start]; old != nil {
		record.first = old.first
		record.greatestLiveSupport = old.greatestLiveSupport
	} else if state.latest != nil && state.latest.record.identity == record.identity {
		record.first = state.latest.record.first
		record.greatestLiveSupport = state.latest.record.greatestLiveSupport
	}
	if input.Source == AggregateSourceLive {
		position := input.Live
		record.greatestLiveSupport = &position
	}
	if state.provenAbsent != nil {
		state.provenAbsent.clear(sessionSlot(e.state.binding, input.WindowStart))
	}
	if input.Source == AggregateSourceLive {
		e.state.liveEpoch = input.Live.ConnectionEpoch
		if state.historicalConflict != nil {
			state.historicalConflict.clear(sessionSlot(e.state.binding, input.WindowStart))
		}
	}
	foldedDirect := input.Source == AggregateSourceHistorical && now.Sub(input.WindowEnd) > correctionHorizon
	if foldedDirect {
		slot := sessionSlot(e.state.binding, input.WindowStart)
		ensurePresence(state).set(slot)
		foldQualificationAggregate(state, e.state.binding, record, now)
		foldPriceRangeAggregate(state, e.state.binding, record)
		foldMVPMeasurementAggregate(state, record)
		state.prefix.fold(record)
		if !state.prefix.latestUncertain && (state.olderLatest == nil || record.windowStart.After(state.olderLatest.windowStart)) {
			if state.olderLatest == nil {
				state.olderLatest = &canonicalAggregate{}
			}
			*state.olderLatest = record
		}
		if e.state.committedT != nil && record.windowStart.Before(*e.state.committedT) &&
			(state.committedLatest == nil || record.windowStart.After(state.committedLatest.windowStart)) {
			state.committedLatest = committedMark(record)
		}
	} else {
		copyRecord := record
		state.tail[record.identity.start] = &copyRecord
		retainMutableMVPMeasurement(state, record)
	}
	if !foldedDirect || !state.prefix.latestUncertain {
		considerLatestMark(state, record)
	}
	updateSelectionForAcceptedMutation(state, record)
	rebuildTailCoverage(state, e.state.binding)
	state.recomputations++
	state.notifyAggregate(record, aggregateProofAll)
	if revision {
		return DispositionAggregateRevised, ReasonNone
	}
	return DispositionAggregateInserted, ReasonNone
}

func (e *Engine) integrityWithdrawLocked(symbol *coreSymbol, existing *canonicalAggregate) DispositionCode {
	state := ensureAggregateState(symbol)
	delete(state.tail, existing.identity.start)
	removeMutableMVPMeasurement(state, existing.identity.start)
	removeFoldedMVPMeasurement(state, existing.identity.start)
	if state.presence != nil {
		state.presence.clear(sessionSlot(e.state.binding, existing.windowStart))
	}
	if state.latest != nil && state.latest.record.identity == existing.identity {
		if state.olderLatest != nil && state.olderLatest.identity == existing.identity {
			state.olderLatest = nil
		}
		recomputeLatest(state)
	}
	if state.committedLatest != nil && state.committedLatest.start == existing.identity.start {
		state.committedLatest = nil
		state.committedLatest = repairSelectionMarkBefore(state, e.state.binding, state.selectionAt)
	}
	if state.priorSelectionMark != nil && state.priorSelectionMark.start == existing.identity.start {
		state.priorSelectionMark = nil
		state.priorSelectionMark = repairSelectionMarkBefore(state, e.state.binding, state.priorSelectionAt)
	}
	rebuildTailCoverage(state, e.state.binding)
	state.recomputations++
	state.notifyAggregate(*existing, aggregateProofAll)
	e.state.aggregateIntegrity = true
	return DispositionAggregateIntegrity
}

func (e *Engine) historicalWithdrawLocked(symbol *coreSymbol, existing *canonicalAggregate, input frozenAggregateInput) DispositionCode {
	state := ensureAggregateState(symbol)
	wasFolded := state.presence != nil && state.presence.has(sessionSlot(e.state.binding, existing.windowStart))
	delete(state.tail, existing.identity.start)
	removeMutableMVPMeasurement(state, existing.identity.start)
	removeFoldedMVPMeasurement(state, existing.identity.start)
	slot := sessionSlot(e.state.binding, existing.windowStart)
	if state.presence != nil {
		state.presence.clear(slot)
	}
	if wasFolded && existing.authority.source == AggregateSourceHistorical {
		state.prefix.withdrawHistorical(*existing)
	}
	ensureHistoricalConflict(state).set(slot)
	if state.provenAbsent != nil {
		state.provenAbsent.clear(slot)
	}
	state.lastConflict = &aggregateConflictEvidence{identity: existing.identity, current: existing.authority, incoming: evidence(input)}
	if state.latest != nil && state.latest.record.identity == existing.identity {
		if state.olderLatest != nil && state.olderLatest.identity == existing.identity {
			state.olderLatest = nil
		}
		recomputeLatest(state)
	}
	if state.committedLatest != nil && state.committedLatest.start == existing.identity.start {
		state.committedLatest = nil
		state.committedLatest = repairSelectionMarkBefore(state, e.state.binding, state.selectionAt)
	}
	if state.priorSelectionMark != nil && state.priorSelectionMark.start == existing.identity.start {
		state.priorSelectionMark = nil
		state.priorSelectionMark = repairSelectionMarkBefore(state, e.state.binding, state.priorSelectionAt)
	}
	rebuildTailCoverage(state, e.state.binding)
	state.recomputations++
	state.notifyAggregate(*existing, aggregateProofAll)
	return DispositionAggregateWithdrawn
}

func ensureAggregateState(symbol *coreSymbol) *symbolAggregateState {
	if symbol.aggregates == nil {
		symbol.aggregates = &symbolAggregateState{tail: make(map[int64]*canonicalAggregate)}
	}
	return symbol.aggregates
}

func ensureProvenAbsent(state *symbolAggregateState) *slotBitmap {
	if state.provenAbsent == nil {
		state.provenAbsent = &slotBitmap{}
	}
	return state.provenAbsent
}
func ensurePresence(state *symbolAggregateState) *slotBitmap {
	if state.presence == nil {
		state.presence = &slotBitmap{}
	}
	return state.presence
}

func ensureSealedLive(state *symbolAggregateState) *slotBitmap {
	if state.sealedLive == nil {
		state.sealedLive = &slotBitmap{}
	}
	return state.sealedLive
}

func ensureHistoricalConflict(state *symbolAggregateState) *slotBitmap {
	if state.historicalConflict == nil {
		state.historicalConflict = &slotBitmap{}
	}
	return state.historicalConflict
}

func (e *Engine) compactSymbolLocked(state *symbolAggregateState, binding *installedBinding, symbol string, now time.Time) {
	// Current-token historical rows can fill directly into the prefix, so an
	// active hydration request no longer pins a full-session mutable tail.
	_ = symbol
	type compactableAggregate struct {
		start  int64
		record *canonicalAggregate
	}
	compactable := make([]compactableAggregate, 0, len(state.tail))
	for start, record := range state.tail {
		if now.Sub(record.windowEnd) > correctionHorizon {
			compactable = append(compactable, compactableAggregate{start: start, record: record})
		}
	}
	// Hydration retains a full-session tail until its exact ingress fence. Map
	// iteration order is random, while price/range sufficient evidence is time
	// ordered. Folding a late-session population in map order repeatedly shifted
	// those slices and made the one-time fence pass quadratic within each symbol.
	// Sorting once makes every evidence insertion an append without changing the
	// canonical records, cutoff, or fold semantics.
	sort.Slice(compactable, func(i, j int) bool {
		if compactable[i].record.windowStart.Equal(compactable[j].record.windowStart) {
			return compactable[i].start < compactable[j].start
		}
		return compactable[i].record.windowStart.Before(compactable[j].record.windowStart)
	})
	for _, candidate := range compactable {
		record := candidate.record
		e.compactAggregateLocked(state, binding, record, now)
	}
	rebuildTailCoverage(state, binding)
	if len(state.tail) > maximumTailRecords {
		// Valid one-second identities within the inclusive horizon cannot exceed
		// 961. Preserve the state for diagnostics, but record the invariant breach
		// so it cannot masquerade as a successful bound.
		state.tailBoundHits++
	}
}

func (e *Engine) compactAggregateLocked(state *symbolAggregateState, binding *installedBinding, record *canonicalAggregate, now time.Time) {
	if state == nil || binding == nil || record == nil || now.Sub(record.windowEnd) <= correctionHorizon || state.tail[record.identity.start] != record {
		return
	}
	ensurePresence(state).set(sessionSlot(binding, record.windowStart))
	if record.authority.source == AggregateSourceLive {
		ensureSealedLive(state).set(sessionSlot(binding, record.windowStart))
		e.retainHydrationReconciliationLocked(record.identity.symbol, *record)
	}
	foldQualificationAggregate(state, binding, *record, now)
	foldPriceRangeAggregate(state, binding, *record)
	foldMVPMeasurementAggregate(state, *record)
	state.prefix.fold(*record)
	removeMutableMVPMeasurement(state, record.identity.start)
	if state.olderLatest == nil || record.windowStart.After(state.olderLatest.windowStart) {
		copyRecord := *record
		state.olderLatest = &copyRecord
	}
	if e.state.committedT != nil && record.windowStart.Before(*e.state.committedT) &&
		(state.committedLatest == nil || record.windowStart.After(state.committedLatest.windowStart)) {
		state.committedLatest = committedMark(*record)
	}
	delete(state.tail, record.identity.start)
}

func committedMark(record canonicalAggregate) *committedAggregateMark {
	mark := &committedAggregateMark{symbol: record.identity.symbol, start: record.identity.start, windowStart: record.windowStart, windowEnd: record.windowEnd,
		values: record.values, authority: record.authority}
	if record.greatestLiveSupport != nil {
		position := *record.greatestLiveSupport
		mark.greatestLiveSupport = &position
	}
	return mark
}

// considerLatestMark maintains the two compact delivery-time mark scalars used
// by strict as-of selection. Ordinary selection therefore never scans the
// correction-tail map when the newest bar begins at candidate T.
func considerLatestMark(state *symbolAggregateState, record canonicalAggregate) {
	if state.latest == nil {
		state.latest = &latestAggregateMark{record: record}
		return
	}
	latest := state.latest.record
	if record.identity == latest.identity {
		state.latest.record = record
	} else if record.windowStart.After(latest.windowStart) {
		state.latest.record = record
	}
}

func updateSelectionForAcceptedMutation(state *symbolAggregateState, record canonicalAggregate) {
	if !state.selectionAt.IsZero() && record.windowStart.Before(state.selectionAt) &&
		(state.committedLatest == nil || record.windowStart.After(state.committedLatest.windowStart) || state.committedLatest.start == record.identity.start) {
		state.committedLatest = committedMark(record)
	}
	if !state.priorSelectionAt.IsZero() && record.windowStart.Before(state.priorSelectionAt) &&
		(state.priorSelectionMark == nil || record.windowStart.After(state.priorSelectionMark.windowStart) || state.priorSelectionMark.start == record.identity.start) {
		state.priorSelectionMark = committedMark(record)
	}
}

func recomputeLatest(state *symbolAggregateState) {
	state.latest = nil
	if state.olderLatest != nil && !state.prefix.latestUncertain {
		considerLatestMark(state, *state.olderLatest)
	}
	for _, record := range state.tail {
		considerLatestMark(state, *record)
	}
}

func sessionSlot(binding *installedBinding, start time.Time) int {
	return int(start.Sub(binding.sessionStart) / time.Second)
}

// exactAggregateCoverage is the sole C3 interpretation of installed interval
// evidence. Each second must be represented by a canonical folded-presence
// bit, its canonical tail record, or an explicitly installed proven-absence
// bit, and a localized conflict overrides all three. Tail lookup is sparse:
// only required seconds not already covered by the two authoritative bitmaps
// touch the canonical map. No second mutable tail-presence truth is retained.
func exactAggregateCoverage(state *symbolAggregateState, binding *installedBinding, start, end time.Time) bool {
	if state == nil || binding == nil || start != start.UTC() || end != end.UTC() ||
		start.Nanosecond() != 0 || end.Nanosecond() != 0 || start.Before(binding.sessionStart) ||
		end.After(binding.sessionEnd) || start.After(end) {
		return false
	}
	startSlot, endSlot := sessionSlot(binding, start), sessionSlot(binding, end)
	if startSlot == endSlot {
		return true
	}
	for word := startSlot / 64; word <= (endSlot-1)/64; word++ {
		required := slotRangeWordMask(word, startSlot, endSlot)
		if state.historicalConflict != nil && state.historicalConflict[word]&required != 0 {
			return false
		}
		var covered uint64
		if state.evaluationTailPresence != nil {
			if !state.evaluationTailPresence.valid || len(state.tail) != state.evaluationTailPresence.tailCount {
				return false
			}
			index := word - state.evaluationTailPresence.baseWord
			if index >= 0 && index < len(state.evaluationTailPresence.words) {
				covered = state.evaluationTailPresence.words[index]
			}
		}
		if state.presence != nil {
			covered |= state.presence[word]
		}
		if state.provenAbsent != nil {
			covered |= state.provenAbsent[word]
		}
		missing := required &^ covered
		if state.evaluationTailPresence != nil && missing != 0 {
			return false
		}
		for ; missing != 0; missing &= missing - 1 {
			slot := word*64 + bits.TrailingZeros64(missing)
			second := binding.sessionStart.Add(time.Duration(slot) * time.Second)
			record := state.tail[second.Unix()]
			// A malformed key/record pair is not coverage. Production mutation
			// keeps these identities equal; validating here prevents a stale or
			// corrupt map entry from falsely proving another second.
			if record == nil || record.identity.start != second.Unix() || !record.windowStart.Equal(second) {
				return false
			}
		}
	}
	return true
}

func buildEvaluationTailPresence(state *symbolAggregateState, binding *installedBinding, result *evaluationTailWindow) (usable bool) {
	if state == nil || binding == nil {
		return false
	}
	*result = evaluationTailWindow{tailCount: len(state.tail), valid: true}
	if len(state.tail) == 0 {
		return true
	}
	if state.latest == nil || state.latest.record.windowStart.Before(binding.sessionStart) || !state.latest.record.windowStart.Before(binding.sessionEnd) {
		result.valid = false
		return true
	}
	latestSlot := sessionSlot(binding, state.latest.record.windowStart)
	firstSlot := max(0, latestSlot-(maximumTailRecords-1))
	result.baseWord = firstSlot / 64
	for key, record := range state.tail {
		if record == nil || record.identity.start != key || key != record.windowStart.Unix() ||
			record.windowStart.Before(binding.sessionStart) || !record.windowStart.Before(binding.sessionEnd) {
			result.valid = false
			return true
		}
		slot := sessionSlot(binding, record.windowStart)
		index := slot/64 - result.baseWord
		if slot < firstSlot || slot > latestSlot || index < 0 || index >= len(result.words) {
			// A valid but wider restored/test tail uses the allocation-free sparse
			// query rather than weakening coverage.
			return false
		}
		result.words[index] |= uint64(1) << uint(slot%64)
	}
	return true
}

func rebuildTailCoverage(state *symbolAggregateState, binding *installedBinding) {
	if state == nil {
		return
	}
	state.tailCoverageUsable = buildEvaluationTailPresence(state, binding, &state.tailCoverage)
	state.tailCoverageBuilt = true
}

func aggregatePresentAt(state *symbolAggregateState, start int64) bool {
	if state == nil {
		return false
	}
	if state.tail[start] != nil || (state.latest != nil && state.latest.record.identity.start == start) ||
		(state.olderLatest != nil && state.olderLatest.identity.start == start) {
		return true
	}
	return false
}

// installExactCoverage applies only the bounded consequence of an engine-
// validated hydration or ordinary-live fence fact. It creates absence bits,
// never synthetic aggregate records or marks.
func installExactCoverage(state *symbolAggregateState, binding *installedBinding, start, end time.Time, invalid *invalidMarkEvidence) bool {
	if state == nil || binding == nil || start != start.UTC() || end != end.UTC() ||
		start.Nanosecond() != 0 || end.Nanosecond() != 0 || start.Before(binding.sessionStart) ||
		end.After(binding.sessionEnd) || start.After(end) {
		return false
	}
	startSlot, endSlot := sessionSlot(binding, start), sessionSlot(binding, end)
	if startSlot == endSlot {
		return true
	}
	absent := ensureProvenAbsent(state)
	for word := startSlot / 64; word <= (endSlot-1)/64 && startSlot < endSlot; word++ {
		mask := slotRangeWordMask(word, startSlot, endSlot)
		if state.presence != nil {
			mask &^= state.presence[word]
		}
		if state.historicalConflict != nil {
			mask &^= state.historicalConflict[word]
		}
		if invalid != nil && !invalid.windowStart.Before(start) && invalid.windowStart.Before(end) {
			invalidSlot := sessionSlot(binding, invalid.windowStart)
			if invalidSlot/64 == word {
				mask &^= uint64(1) << uint(invalidSlot%64)
			}
		}
		absent[word] |= mask
	}
	for _, record := range state.tail {
		if record != nil && !record.windowStart.Before(start) && record.windowStart.Before(end) {
			absent.clear(sessionSlot(binding, record.windowStart))
		}
	}
	return true
}

func slotRangeWordMask(word, startSlot, endSlot int) uint64 {
	wordStart := word * 64
	from, through := max(startSlot-wordStart, 0), min(endSlot-wordStart, 64)
	if from >= through {
		return 0
	}
	upper := ^uint64(0)
	if through < 64 {
		upper = uint64(1)<<uint(through) - 1
	}
	lower := uint64(0)
	if from > 0 {
		lower = uint64(1)<<uint(from) - 1
	}
	return upper &^ lower
}

func (e *Engine) historicalRegistrationAllowedForProof(symbol string, start, end time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.binding == nil || start != start.UTC() || end != end.UTC() || start.Nanosecond() != 0 || end.Nanosecond() != 0 ||
		start.Before(e.state.binding.sessionStart) || end.After(e.state.binding.sessionEnd) || !start.Before(end) {
		return false
	}
	index, ok := e.state.binding.index[symbol]
	if !ok {
		return false
	}
	state := e.state.binding.symbols[index].aggregates
	if state == nil || state.presence == nil {
		return true
	}
	for at := start; at.Before(end); at = at.Add(time.Second) {
		if state.presence.has(sessionSlot(e.state.binding, at)) {
			return false
		}
	}
	return true
}
