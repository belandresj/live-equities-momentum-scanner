package engine

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
)

type CheckpointReason string

const (
	CheckpointReasonNone                 CheckpointReason = ""
	CheckpointReasonProjectionIneligible CheckpointReason = "projection_ineligible"
	CheckpointReasonProjectionInvariant  CheckpointReason = "projection_invariant"
	CheckpointReasonUnsupportedSchema    CheckpointReason = "unsupported_schema"
	CheckpointReasonBindingMismatch      CheckpointReason = "binding_mismatch"
	CheckpointReasonIntegrity            CheckpointReason = "integrity"
	CheckpointReasonStructure            CheckpointReason = "structure"
	CheckpointReasonSemanticInvariant    CheckpointReason = "semantic_invariant"
	CheckpointReasonEvaluation           CheckpointReason = "regenerated_evaluation"
)

const (
	DispositionCheckpointProjected    DispositionCode = "checkpoint_projected"
	DispositionCheckpointRejected     DispositionCode = "checkpoint_projection_rejected"
	DispositionCheckpointInstalled    DispositionCode = "checkpoint_installed"
	DispositionCheckpointIncompatible DispositionCode = "checkpoint_incompatible"
	DispositionCheckpointInvalid      DispositionCode = "checkpoint_invalid"
)

type CheckpointProjectionDisposition string

const (
	CheckpointProjected          CheckpointProjectionDisposition = "projected"
	CheckpointProjectionRejected CheckpointProjectionDisposition = "rejected"
)

type CheckpointProjectionResult struct {
	EngineSequence         uint64
	Code                   DispositionCode
	EngineReason           DispositionReason
	SuppressionDisposition SuppressionDisposition
	Disposition            CheckpointProjectionDisposition
	Reason                 CheckpointReason
	Image                  checkpoint.Image
}

type CheckpointInstallDisposition string

const (
	CheckpointInstalled    CheckpointInstallDisposition = "installed"
	CheckpointIncompatible CheckpointInstallDisposition = "incompatible"
	CheckpointInvalid      CheckpointInstallDisposition = "invalid"
)

// InstalledCheckpointFact is the bounded engine-owned restart baseline fact.
// It proves neither current coverage nor readiness.
type InstalledCheckpointFact struct {
	BindingIdentity  string
	SchemaVersion    string
	T0               time.Time
	ArtifactSequence uint64
	Checksum         string
}

type CheckpointInstallResult struct {
	EngineSequence         uint64
	Code                   DispositionCode
	EngineReason           DispositionReason
	SuppressionDisposition SuppressionDisposition
	Disposition            CheckpointInstallDisposition
	Reason                 CheckpointReason
	Fact                   InstalledCheckpointFact
}

func finalizeCheckpointProjection(result CheckpointProjectionResult, disposition transitionDisposition) CheckpointProjectionResult {
	result.EngineSequence, result.Code, result.EngineReason, result.SuppressionDisposition = disposition.EngineSequence, disposition.Code, disposition.Reason, disposition.SuppressionDisposition
	if disposition.Code != DispositionCheckpointProjected {
		result.Disposition = CheckpointProjectionRejected
		if result.Reason == CheckpointReasonNone {
			result.Reason = CheckpointReasonProjectionInvariant
		}
		result.Image = checkpoint.Image{}
	}
	return result
}

func finalizeCheckpointInstall(result CheckpointInstallResult, disposition transitionDisposition) CheckpointInstallResult {
	result.EngineSequence, result.Code, result.EngineReason, result.SuppressionDisposition = disposition.EngineSequence, disposition.Code, disposition.Reason, disposition.SuppressionDisposition
	matchedOutcome := disposition.Code == DispositionCheckpointInstalled && result.Disposition == CheckpointInstalled ||
		disposition.Code == DispositionCheckpointIncompatible && result.Disposition == CheckpointIncompatible ||
		disposition.Code == DispositionCheckpointInvalid && result.Disposition == CheckpointInvalid
	if !matchedOutcome {
		result.Disposition = CheckpointInvalid
		if result.Reason == CheckpointReasonNone {
			result.Reason = CheckpointReasonSemanticInvariant
		}
		result.Fact = InstalledCheckpointFact{}
	}
	return result
}

func setCheckpointTerminalResult(node *queueNode, disposition *transitionDisposition, reason CheckpointReason) {
	if node.kind == inputCheckpointProjection {
		disposition.checkpointProjection = CheckpointProjectionResult{Disposition: CheckpointProjectionRejected, Reason: reason}
	}
	if node.kind == inputCheckpointInstall {
		disposition.checkpointInstall = CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: reason}
	}
}

// AdmitCheckpointProjection orders one deep semantic projection through the
// sole FIFO owner. Later persistence receives only its detached result.
func (e *Engine) AdmitCheckpointProjection(ctx context.Context) (AdmissionResult, <-chan CheckpointProjectionResult) {
	e.beginAdmission()
	if ctx == nil {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputCheckpointProjection}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.checkpointProjection
}

func (e *Engine) projectCheckpointLocked(created time.Time) CheckpointProjectionResult {
	if e.mode != RunModeLive || e.state.binding == nil || e.state.lifecycle != lifecycleLive || e.state.globalFailure ||
		e.state.committedT == nil || e.state.committedT.IsZero() ||
		!e.state.aggregateEvaluator.current.at.Equal(*e.state.committedT) {
		return CheckpointProjectionResult{Disposition: CheckpointProjectionRejected, Reason: CheckpointReasonProjectionIneligible}
	}
	t0 := *e.state.committedT
	if created.Before(t0) || e.state.checkpointSequence == math.MaxUint64 {
		return CheckpointProjectionResult{Disposition: CheckpointProjectionRejected, Reason: CheckpointReasonProjectionInvariant}
	}
	image := checkpoint.Image{
		SchemaVersion: checkpoint.SchemaV1, ProducerMode: checkpoint.ProducerLive,
		Binding: projectCheckpointBinding(e.state.binding), T0: t0, CreatedAt: created,
		Sequence: e.state.checkpointSequence + 1, Population: len(e.state.binding.symbols), Symbols: make([]checkpoint.Symbol, len(e.state.binding.symbols)),
	}
	for index := range e.state.binding.symbols {
		projected, err := projectCheckpointSymbol(e.state.binding, &e.state.binding.symbols[index], e.state.aggregateEvaluator, checkpointProjectionCommittedMarkerForState(e.state.binding.symbols[index].aggregates), index, t0)
		if err != nil {
			return CheckpointProjectionResult{Disposition: CheckpointProjectionRejected, Reason: CheckpointReasonProjectionInvariant}
		}
		image.Symbols[index] = projected
	}
	image.Counts = checkpointStructureCounts(image)
	if !boundedCheckpointCandidate(image) || image.Population != len(image.Symbols) ||
		image.Counts != checkpointStructureCounts(image) || image.Counts.RecordsWithState+image.Counts.EmptyStateRecords != image.Population {
		return CheckpointProjectionResult{Disposition: CheckpointProjectionRejected, Reason: CheckpointReasonProjectionInvariant}
	}
	e.state.checkpointSequence++
	return CheckpointProjectionResult{Disposition: CheckpointProjected, Image: image}
}

func projectCheckpointBinding(b *installedBinding) checkpoint.Binding {
	return checkpoint.Binding{
		Identity: b.identity, TradingDate: b.tradingDate, ScheduleSchema: b.scheduleSchema,
		ScheduleVersion: b.scheduleVersion, ScheduleArtifactSHA256: b.scheduleArtifactSHA256,
		PriorSessionDate: b.priorSessionDate, UniversePolicy: b.universePolicy, UniverseIdentity: b.universeIdentity,
		PriorClosePolicy: b.priorClosePolicy, PriorCloseIdentity: b.priorCloseIdentity, Locale: b.locale, Market: b.market,
		SessionStart: b.sessionStart, SessionEnd: b.sessionEnd, PriorRegularClose: b.priorRegularClose,
		Adjusted: b.adjusted, IncludeOTC: b.includeOTC,
	}
}

var errCheckpointSealedCommittedMarker = errors.New("sealed T0 committed marker invalid")

func checkpointProjectionCommittedMarkerForState(state *symbolAggregateState) checkpointProjectionCommittedMarker {
	if state == nil || state.committedLatest == nil {
		return checkpointProjectionCommittedMarker{}
	}
	return checkpointProjectionCommittedMarker{mark: *state.committedLatest, present: true}
}

func projectCheckpointSymbol(binding *installedBinding, symbol *coreSymbol, evaluator aggregateEvaluatorState, committed checkpointProjectionCommittedMarker, index int, t0 time.Time) (checkpoint.Symbol, error) {
	r := checkpoint.Symbol{Symbol: symbol.symbol}
	state := symbol.aggregates
	if evidence, ok := evaluator.invalidMarks[index]; ok {
		v := evidence.windowStart
		r.InvalidMarkStart = &v
	}
	if consequence, ok := evaluator.coverage[index]; ok {
		r.Coverage = &checkpoint.Coverage{Outcome: uint8(consequence.outcome), Origin: uint8(consequence.origin)}
	}
	if state == nil {
		if committed.present {
			return checkpoint.Symbol{}, errCheckpointSealedCommittedMarker
		}
		return r, nil
	}
	r.HasState = true
	starts := make([]int64, 0, len(state.tail))
	for start, record := range state.tail {
		if record != nil && record.windowStart.Before(t0) {
			starts = append(starts, start)
		}
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	r.Tail = make([]checkpoint.Aggregate, 0, len(starts))
	for _, start := range starts {
		r.Tail = append(r.Tail, projectAggregate(*state.tail[start]))
	}
	if state.olderLatest != nil && state.olderLatest.windowStart.Before(t0) {
		v := projectAggregate(*state.olderLatest)
		r.OlderMark = &v
	}
	if committed.present {
		var full *canonicalAggregate
		retainedAsCanonical := false
		if record := state.tail[committed.mark.start]; record != nil {
			full = record
			retainedAsCanonical = true
		}
		if full == nil && state.olderLatest != nil && state.olderLatest.identity.start == committed.mark.start {
			full = state.olderLatest
			retainedAsCanonical = true
		}
		if full == nil {
			full = &canonicalAggregate{identity: aggregateIdentity{symbol: symbol.symbol, start: committed.mark.start},
				windowStart: committed.mark.windowStart, windowEnd: committed.mark.windowEnd, values: committed.mark.values}
		}
		if !full.windowStart.Before(t0) {
			return checkpoint.Symbol{}, errCheckpointSealedCommittedMarker
		}
		if !retainedAsCanonical {
			// The checkpoint schema restores canonical marks from Tail or OlderMark;
			// CommittedMark is evaluator support, not a second canonical source.
			// When folded forward state displaced olderLatest beyond T0, carry the
			// retained full committed aggregate as the one clipped older mark.
			v := projectAggregate(*full)
			r.OlderMark = &v
		}
		v := projectAggregate(*full)
		r.CommittedMark = &v
	}
	cutoff := sessionSlot(binding, t0)
	r.Presence = projectBitmap(state.presence, cutoff)
	r.ProvenAbsent = projectBitmap(state.provenAbsent, cutoff)
	r.HistoricalConflict = projectBitmap(state.historicalConflict, cutoff)
	r.PriceRange = projectPriceRange(state.priceRange, t0)
	r.Activity = projectActivity(state, binding, t0)
	r.Qualification = projectQualification(state.qualification, t0)
	return r, nil
}

func projectAggregate(a canonicalAggregate) checkpoint.Aggregate {
	return checkpoint.Aggregate{WindowStart: a.windowStart, WindowEnd: a.windowEnd, Values: checkpoint.Values{
		Open: a.values.Open, High: a.values.High, Low: a.values.Low, Close: a.values.Close, Volume: a.values.Volume,
		VWAP: a.values.VWAP, AverageTradeSize: a.values.AverageTradeSize, ATSProvenance: string(a.values.ATSProvenance),
	}}
}

func projectBitmap(source *slotBitmap, cutoff int) []uint64 {
	if source == nil {
		return nil
	}
	r := append([]uint64(nil), source[:]...)
	for slot := max(0, cutoff); slot < sessionSeconds; slot++ {
		r[slot/64] &^= uint64(1) << uint(slot%64)
	}
	return r
}

func projectPriceRange(source *priceRangeFeatureState, t0 time.Time) *checkpoint.PriceRange {
	if source == nil {
		return nil
	}
	r := &checkpoint.PriceRange{FirstStart: source.firstStart, RollingFloor: min(source.rollingFloor, t0.Unix()),
		FinalizedThrough: min(source.finalizedThrough, t0.Unix()), FirstOpen: source.firstOpen,
		HasFirst: source.hasFirst && source.firstStart < t0.Unix(), BoundExceeded: source.boundExceeded}
	if !r.HasFirst {
		r.FirstStart, r.FirstOpen = 0, 0
	}
	r.Highs = projectExtrema(source.highs, t0)
	r.Lows = projectExtrema(source.lows, t0)
	r.SessionHighs = projectExtrema(source.sessionHighs, t0)
	r.SessionLows = projectExtrema(source.sessionLows, t0)
	for _, p := range r.SessionHighs {
		if !r.HasSessionExtrema || p.Value > r.SessionHigh {
			r.SessionHigh = p.Value
		}
		r.HasSessionExtrema = true
	}
	for _, p := range r.SessionLows {
		if r.SessionLow == 0 || p.Value < r.SessionLow {
			r.SessionLow = p.Value
		}
	}
	return r
}

func projectExtrema(source []extremaPoint, t0 time.Time) []checkpoint.ExtremaPoint {
	r := make([]checkpoint.ExtremaPoint, 0, len(source))
	for _, p := range source {
		if p.windowStart < t0.Unix() {
			r = append(r, checkpoint.ExtremaPoint{WindowStart: p.windowStart, Value: p.value})
		}
	}
	if len(r) == 0 {
		return nil
	}
	return r
}

func projectActivity(state *symbolAggregateState, binding *installedBinding, t0 time.Time) *checkpoint.Activity {
	source := state.activity
	if source == nil {
		return nil
	}
	r := &checkpoint.Activity{BoundExceeded: source.boundExceeded,
		References:    make([]checkpoint.ActivitySummary, 0, len(source.references)),
		Mutable:       make([]checkpoint.ActivityMutable, 0, len(source.mutable)),
		FoldedTargets: make([]checkpoint.ActivityTargetBlock, 0, len(source.foldedTargets))}
	keys := sortedInt64Keys(source.references)
	for _, end := range keys {
		if end <= t0.Unix() {
			r.References = append(r.References, projectActivitySummary(*source.references[end]))
		}
	}
	keys = sortedInt64Keys(source.mutable)
	for _, end := range keys {
		if end <= t0.Unix() {
			b := source.mutable[end]
			r.Mutable = append(r.Mutable, checkpoint.ActivityMutable{End: end, Folded: projectActivitySummary(b.folded), Current: projectActivitySummary(b.current)})
		}
	}
	keys = sortedInt64Keys(source.foldedTargets)
	for _, end := range keys {
		block := source.foldedTargets[end]
		out := checkpoint.ActivityTargetBlock{End: end, Transactions: block.transactions, Highs: block.highs, Lows: block.lows, Present: block.present, Invalid: block.invalid}
		start := time.Unix(end, 0).UTC().Add(-activityBlockDuration)
		floor := t0.Add(-activityBlockDuration)
		for slot := 0; slot < 30; slot++ {
			mask := uint32(1) << uint(slot)
			second := start.Add(time.Duration(slot) * time.Second)
			if second.Before(floor) || !second.Before(t0) {
				out.Present &^= mask
				out.Invalid &^= mask
			}
			if out.Present&mask == 0 || out.Invalid&mask != 0 {
				out.Transactions[slot], out.Highs[slot], out.Lows[slot] = 0, 0, 0
			}
		}
		if out.Present != 0 {
			r.FoldedTargets = append(r.FoldedTargets, out)
			r.FoldedTargetContributions += bitsSet32(out.Present)
		}
	}
	// A stalled T0 may split one aligned block after compacted forward state has
	// already entered the live Activity graph. Rebuild that partial block from
	// identity-positioned pre-T0 folded contributions plus the retained tail;
	// copying a combined post-T0 mutable/reference summary would contaminate it.
	partialEnds := make(map[int64]struct{})
	for end, block := range source.foldedTargets {
		if end > t0.Unix() && block.present != 0 {
			partialEnds[end] = struct{}{}
		}
	}
	for _, record := range state.tail {
		end := activityBlockEnd(binding, record.windowStart).Unix()
		if end > t0.Unix() && record.windowStart.Before(t0) {
			partialEnds[end] = struct{}{}
		}
	}
	for _, end := range sortedSetKeys(partialEnds) {
		folded := activityBlockSummary{end: end, low: math.Inf(1)}
		if block, ok := source.foldedTargets[end]; ok {
			blockStart := time.Unix(end, 0).UTC().Add(-activityBlockDuration)
			for slot := 0; slot < 30; slot++ {
				mask := uint32(1) << uint(slot)
				if block.present&mask == 0 || !blockStart.Add(time.Duration(slot)*time.Second).Before(t0) {
					continue
				}
				folded = addCheckpointActivityContribution(folded, block.transactions[slot], block.highs[slot], block.lows[slot], block.invalid&mask != 0)
			}
		}
		current := folded
		for _, record := range state.tail {
			if record.windowStart.Before(t0) && activityBlockEnd(binding, record.windowStart).Unix() == end {
				current = addActivityAggregate(current, record.values)
			}
		}
		if folded.aggregateCount != 0 || current.aggregateCount != 0 || folded.invalid || current.invalid {
			r.Mutable = append(r.Mutable, checkpoint.ActivityMutable{End: end, Folded: projectActivitySummary(folded), Current: projectActivitySummary(finishActivitySummary(current))})
		}
	}
	sort.Slice(r.Mutable, func(i, j int) bool { return r.Mutable[i].End < r.Mutable[j].End })
	return r
}

func addCheckpointActivityContribution(summary activityBlockSummary, transactions, high, low float64, invalid bool) activityBlockSummary {
	if summary.invalid || invalid || !summary.transactionSum.add(transactions) || !finitePositiveFeature(high) || !finitePositiveFeature(low) || high < low || summary.aggregateCount == math.MaxUint8 {
		summary.invalid = true
		return summary
	}
	if summary.aggregateCount == 0 {
		summary.high, summary.low = high, low
	} else {
		summary.high, summary.low = max(summary.high, high), min(summary.low, low)
	}
	summary.aggregateCount++
	return summary
}

func projectActivitySummary(s activityBlockSummary) checkpoint.ActivitySummary {
	if s.invalid {
		return checkpoint.ActivitySummary{End: s.end, Invalid: true}
	}
	if s.aggregateCount == 0 {
		return checkpoint.ActivitySummary{End: s.end}
	}
	s = finishActivitySummary(s)
	if s.invalid {
		return checkpoint.ActivitySummary{End: s.end, Invalid: true}
	}
	return checkpoint.ActivitySummary{End: s.end, TransactionSum: checkpoint.ExactSum(s.transactionSum), Transactions: s.transactions,
		High: s.high, Low: s.low, ExpansionBPS: s.expansionBPS, AggregateCount: s.aggregateCount, Invalid: s.invalid}
}

func projectQualification(source *qualificationState, t0 time.Time) *checkpoint.Qualification {
	if source == nil {
		return nil
	}
	r := &checkpoint.Qualification{AccountedThrough: source.accountedThrough, Finalized: source.finalized, FinalProofEnd: source.finalProofEnd,
		BoundExceeded: source.boundExceeded, Invalid: source.invalid, UnresolvedOrigin: uint8(source.unresolvedOrigin),
		FinalizedGateBars: make([]checkpoint.QualificationGateBar, 0, len(source.finalizedGateBars)),
		Proofs:            make([]int64, 0, len(source.proofs)), Dirty: make([]int64, 0, len(source.dirty))}
	for _, start := range sortedInt64Keys(source.finalizedGateBars) {
		if start < t0.Unix() {
			b := source.finalizedGateBars[start]
			r.FinalizedGateBars = append(r.FinalizedGateBars, checkpoint.QualificationGateBar{Start: start, Close: b.close, Volume: b.volume, VWAP: b.vwap, AverageTradeSize: b.averageTradeSize, ATSProvenance: string(b.provenance)})
		}
	}
	for _, proof := range sortedSetKeys(source.proofs) {
		if proof <= t0.Unix() {
			r.Proofs = append(r.Proofs, proof)
		}
	}
	for _, proof := range sortedSetKeys(source.dirty) {
		if proof <= t0.Unix() {
			r.Dirty = append(r.Dirty, proof)
		}
	}
	if r.AccountedThrough.After(t0) {
		r.AccountedThrough = t0
	}
	return r
}

func sortedInt64Keys[V any](m map[int64]V) []int64 {
	r := make([]int64, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Slice(r, func(i, j int) bool { return r[i] < r[j] })
	return r
}
func sortedSetKeys(m map[int64]struct{}) []int64 { return sortedInt64Keys(m) }
func bitsSet32(v uint32) int {
	n := 0
	for v != 0 {
		v &= v - 1
		n++
	}
	return n
}

// AdmitCheckpointInstall orders validation and installation through the sole
// FIFO owner. The owner rebuilds a complete scratch graph before the single
// assignment below; no candidate slice/map is retained.
func (e *Engine) AdmitCheckpointInstall(ctx context.Context, candidate checkpoint.Candidate) (AdmissionResult, <-chan CheckpointInstallResult) {
	e.beginAdmission()
	if ctx == nil || len(candidate.Checksum) > 64 || !boundedCheckpointCandidate(candidate.Image) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputCheckpointInstall, checkpointCandidate: checkpoint.Candidate{Image: candidate.Image.Clone(), Checksum: candidate.Checksum, Integrity: candidate.Integrity}}
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.checkpointInstall
}

func boundedCheckpointCandidate(image checkpoint.Image) bool {
	if len(image.Symbols) > maximumUniverseSymbols {
		return false
	}
	for _, s := range image.Symbols {
		if len(s.Symbol) > maximumSymbolBytes || len(s.Tail) > maximumTailRecords ||
			(len(s.Presence) != 0 && len(s.Presence) != len(slotBitmap{})) ||
			(len(s.ProvenAbsent) != 0 && len(s.ProvenAbsent) != len(slotBitmap{})) ||
			(len(s.HistoricalConflict) != 0 && len(s.HistoricalConflict) != len(slotBitmap{})) {
			return false
		}
		if p := s.PriceRange; p != nil && (len(p.Highs) > maximumExtremaPointsPerDeque || len(p.Lows) > maximumExtremaPointsPerDeque ||
			len(p.SessionHighs) > sessionSeconds || len(p.SessionLows) > sessionSeconds) {
			return false
		}
		if a := s.Activity; a != nil && (len(a.References) > maximumActivityReferences || len(a.Mutable) > maximumMutableActivityBlockIDs ||
			len(a.FoldedTargets) > maximumActivityTargetBlocks || a.FoldedTargetContributions < 0 || a.FoldedTargetContributions > maximumActivityTargetContributions) {
			return false
		}
		if q := s.Qualification; q != nil && (len(q.FinalizedGateBars) > maximumFinalizedGateBars || len(q.Proofs) > maximumQualificationProofs || len(q.Dirty) > maximumQualificationProofs) {
			return false
		}
	}
	return true
}

func checkpointStructureCounts(image checkpoint.Image) checkpoint.StructureCounts {
	var c checkpoint.StructureCounts
	for _, s := range image.Symbols {
		if s.HasState {
			c.RecordsWithState++
		} else {
			c.EmptyStateRecords++
		}
		c.TailRecords += len(s.Tail)
		c.PresenceWords += len(s.Presence)
		c.ProvenAbsentWords += len(s.ProvenAbsent)
		c.ConflictWords += len(s.HistoricalConflict)
		if s.PriceRange != nil {
			c.PriceExtremaPoints += len(s.PriceRange.Highs) + len(s.PriceRange.Lows) + len(s.PriceRange.SessionHighs) + len(s.PriceRange.SessionLows)
		}
		if s.Activity != nil {
			c.ActivityReferences += len(s.Activity.References)
			c.ActivityMutable += len(s.Activity.Mutable)
			c.ActivityTargetBlocks += len(s.Activity.FoldedTargets)
			c.ActivityTargetContributions += s.Activity.FoldedTargetContributions
		}
		if s.Qualification != nil {
			c.QualificationGateBars += len(s.Qualification.FinalizedGateBars)
			c.QualificationProofs += len(s.Qualification.Proofs)
			c.QualificationDirty += len(s.Qualification.Dirty)
		}
		if s.InvalidMarkStart != nil {
			c.InvalidMarks++
		}
		if s.Coverage != nil {
			c.CoverageConsequences++
		}
	}
	return c
}

func (e *Engine) installCheckpointLocked(candidate checkpoint.Candidate) CheckpointInstallResult {
	image := candidate.Image.Clone()
	if !candidate.Integrity || len(candidate.Checksum) != 64 || strings.Trim(candidate.Checksum, "0123456789abcdef") != "" {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonIntegrity}
	}
	if image.SchemaVersion != checkpoint.SchemaV1 || image.ProducerMode != checkpoint.ProducerLive {
		return CheckpointInstallResult{Disposition: CheckpointIncompatible, Reason: CheckpointReasonUnsupportedSchema}
	}
	if e.state.binding == nil || projectCheckpointBinding(e.state.binding) != image.Binding {
		return CheckpointInstallResult{Disposition: CheckpointIncompatible, Reason: CheckpointReasonBindingMismatch}
	}
	if image.Population != len(image.Symbols) || image.Counts != checkpointStructureCounts(image) || image.Counts.RecordsWithState+image.Counts.EmptyStateRecords != image.Population {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonStructure}
	}
	fact := InstalledCheckpointFact{BindingIdentity: image.Binding.Identity, SchemaVersion: image.SchemaVersion, T0: image.T0, ArtifactSequence: image.Sequence, Checksum: candidate.Checksum}
	if e.state.installedCheckpoint != nil {
		if *e.state.installedCheckpoint == fact {
			return CheckpointInstallResult{Disposition: CheckpointInstalled, Fact: fact}
		}
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonSemanticInvariant}
	}
	if !checkpointInstallLifecycle(e) || !validCheckpointTimes(image, e.state.binding) {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonStructure}
	}
	binding, evaluator, err := buildCheckpointBinding(e.state.binding, image)
	if err != nil {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonSemanticInvariant}
	}
	scratchState := &engineState{lifecycle: e.state.lifecycle, binding: binding, aggregateEvaluator: evaluator, committedT: immutableTime(image.T0), clockMonotonic: true}
	scratch := &Engine{mode: e.mode, state: scratchState}
	staged := scratch.stageAggregateEvaluationAtLocked(image.T0, image.T0)
	if err := validateAggregateEvaluation(staged); err != nil {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonEvaluation}
	}
	scratch.applyAggregateCandidateLocked(image.T0, image.T0)
	staged = scratch.stageAggregateEvaluationAtLocked(image.T0, image.T0)
	if err := validateAggregateEvaluation(staged); err != nil {
		return CheckpointInstallResult{Disposition: CheckpointInvalid, Reason: CheckpointReasonEvaluation}
	}
	scratch.state.aggregateEvaluator.current = cloneAggregateEvaluation(staged)
	scratch.state.installedCheckpoint = &fact
	scratch.state.checkpointSequence = image.Sequence
	if e.mode == RunModeLive {
		scratch.state.hydration.checkpointT0 = immutableTime(image.T0)
	}
	// Sole atomic apply point: the prior bound fresh graph remains untouched on
	// every return above.
	scratch.state.latestTransition = e.state.latestTransition
	scratch.state.exposedRevision = e.state.exposedRevision + 1
	scratch.state.evaluationRevision = e.state.evaluationRevision + 1
	e.state = scratch.state
	return CheckpointInstallResult{Disposition: CheckpointInstalled, Fact: fact}
}

func checkpointInstallLifecycle(e *Engine) bool {
	if e.state.committedT != nil || e.state.globalFailure || e.state.aggregateIntegrity {
		return false
	}
	if e.mode == RunModeLive {
		return e.state.lifecycle == lifecycleAwaitingAggregateAck && !e.state.aggregateAcknowledged && e.state.liveEpoch == 0
	}
	return e.mode == RunModeReplay && e.state.lifecycle == lifecycleInitializing
}

func validCheckpointTimes(i checkpoint.Image, b *installedBinding) bool {
	return i.Sequence > 0 && !i.T0.IsZero() && i.T0 == i.T0.UTC() && i.T0.Nanosecond() == 0 && !i.T0.Before(b.sessionStart) && !i.T0.After(b.sessionEnd) &&
		i.CreatedAt == i.CreatedAt.UTC() && !i.CreatedAt.IsZero() && !i.CreatedAt.Before(i.T0)
}

func buildCheckpointBinding(current *installedBinding, image checkpoint.Image) (*installedBinding, aggregateEvaluatorState, error) {
	b := *current
	b.symbols = make([]coreSymbol, len(current.symbols))
	b.index = make(map[string]int, len(current.index))
	evaluator := aggregateEvaluatorState{coverage: make(map[int]aggregateCoverageConsequence), invalidMarks: make(map[int]invalidMarkEvidence)}
	for i, source := range image.Symbols {
		if source.Symbol != current.symbols[i].symbol {
			return nil, evaluator, errors.New("symbol order")
		}
		b.symbols[i] = coreSymbol{symbol: current.symbols[i].symbol, prior: current.symbols[i].prior}
		b.index[source.Symbol] = i
		state, err := buildCheckpointSymbol(&b, source, image.T0)
		if err != nil {
			return nil, evaluator, err
		}
		b.symbols[i].aggregates = state
		if source.InvalidMarkStart != nil {
			if source.InvalidMarkStart.Before(b.sessionStart) || !source.InvalidMarkStart.Before(image.T0) {
				return nil, evaluator, errors.New("invalid mark")
			}
			evaluator.invalidMarks[i] = invalidMarkEvidence{windowStart: *source.InvalidMarkStart}
		}
		if source.Coverage != nil {
			c := aggregateCoverageConsequence{outcome: coverageOutcome(source.Coverage.Outcome), origin: uncertaintyOrigin(source.Coverage.Origin)}
			evaluator.coverage[i] = c
		}
	}
	if !validAggregateEvaluatorSupport(&b, evaluator) {
		return nil, evaluator, errors.New("evaluator support")
	}
	return &b, evaluator, nil
}

func buildCheckpointSymbol(binding *installedBinding, source checkpoint.Symbol, t0 time.Time) (*symbolAggregateState, error) {
	if !source.HasState {
		if len(source.Tail) > 0 || source.OlderMark != nil || source.CommittedMark != nil || len(source.Presence) > 0 || len(source.ProvenAbsent) > 0 || len(source.HistoricalConflict) > 0 || source.PriceRange != nil || source.Activity != nil || source.Qualification != nil {
			return nil, errors.New("state omission")
		}
		return nil, nil
	}
	state := &symbolAggregateState{tail: make(map[int64]*canonicalAggregate)}
	for _, a := range source.Tail {
		r, err := restoreAggregate(source.Symbol, a, binding, t0)
		if err != nil {
			return nil, err
		}
		if len(state.tail) >= maximumTailRecords || state.tail[r.identity.start] != nil {
			return nil, errors.New("tail")
		}
		state.tail[r.identity.start] = r
	}
	if source.OlderMark != nil {
		r, err := restoreAggregate(source.Symbol, *source.OlderMark, binding, t0)
		if err != nil {
			return nil, err
		}
		state.olderLatest = r
	}
	if source.CommittedMark != nil {
		r, err := restoreAggregate(source.Symbol, *source.CommittedMark, binding, t0)
		if err != nil {
			return nil, err
		}
		state.committedLatest = committedMark(*r)
	}
	var err error
	if state.presence, err = restoreBitmap(source.Presence, binding, t0); err != nil {
		return nil, err
	}
	if state.provenAbsent, err = restoreBitmap(source.ProvenAbsent, binding, t0); err != nil {
		return nil, err
	}
	if state.historicalConflict, err = restoreBitmap(source.HistoricalConflict, binding, t0); err != nil {
		return nil, err
	}
	for slot := 0; slot < sessionSlot(binding, t0); slot++ {
		present := state.presence.has(slot)
		absent := state.provenAbsent.has(slot)
		conflict := state.historicalConflict.has(slot)
		if absent && (present || conflict) {
			return nil, errors.New("bitmap contradiction")
		}
	}
	for _, record := range state.tail {
		slot := sessionSlot(binding, record.windowStart)
		if state.provenAbsent.has(slot) || state.presence.has(slot) {
			return nil, errors.New("canonical/bitmap contradiction")
		}
	}
	state.priceRange, err = restorePriceRange(source.PriceRange, binding, t0)
	if err != nil {
		return nil, err
	}
	if state.priceRange != nil {
		for _, record := range state.tail {
			retainMutablePriceRangeEvidence(state.priceRange, *record)
		}
	}
	for _, record := range state.tail {
		retainMutableMVPMeasurement(state, *record)
	}
	state.activity, err = restoreActivity(source.Activity, binding, t0)
	if err != nil {
		return nil, err
	}
	state.qualification, err = restoreQualification(source.Qualification, binding, t0)
	if err != nil {
		return nil, err
	}
	recomputeLatest(state)
	rebuildTailCoverage(state, binding)
	if source.CommittedMark != nil {
		wanted := source.CommittedMark.WindowStart
		mark, ok := latestMarkBefore(state, t0)
		if !ok || mark.windowStart != wanted {
			return nil, errors.New("committed mark mismatch")
		}
	}
	return state, nil
}

func restoreAggregate(symbol string, a checkpoint.Aggregate, b *installedBinding, t0 time.Time) (*canonicalAggregate, error) {
	v := AggregateValues{Open: a.Values.Open, High: a.Values.High, Low: a.Values.Low, Close: a.Values.Close, Volume: a.Values.Volume, VWAP: a.Values.VWAP, AverageTradeSize: a.Values.AverageTradeSize, ATSProvenance: ATSProvenance(a.Values.ATSProvenance)}
	in := AggregateInput{Symbol: symbol, WindowStart: a.WindowStart, WindowEnd: a.WindowEnd, Values: v}
	if !validAggregateWindow(in, b) || !validAggregateValues(v) || !a.WindowStart.Before(t0) {
		return nil, errors.New("aggregate")
	}
	r := &canonicalAggregate{identity: aggregateIdentity{symbol: symbol, start: a.WindowStart.Unix()}, windowStart: a.WindowStart, windowEnd: a.WindowEnd, values: v, first: aggregateEvidence{restored: true}, authority: aggregateEvidence{restored: true}}
	return r, nil
}

func restoreBitmap(v []uint64, b *installedBinding, t0 time.Time) (*slotBitmap, error) {
	if len(v) == 0 {
		return nil, nil
	}
	if len(v) != len(slotBitmap{}) {
		return nil, errors.New("bitmap length")
	}
	r := &slotBitmap{}
	copy(r[:], v)
	for slot := sessionSlot(b, t0); slot < sessionSeconds; slot++ {
		if r.has(slot) {
			return nil, errors.New("post cutoff bitmap")
		}
	}
	return r, nil
}

func restorePriceRange(v *checkpoint.PriceRange, b *installedBinding, t0 time.Time) (*priceRangeFeatureState, error) {
	if v == nil {
		return nil, nil
	}
	r := &priceRangeFeatureState{firstStart: v.FirstStart, rollingFloor: v.RollingFloor, finalizedThrough: v.FinalizedThrough, firstOpen: v.FirstOpen, sessionHigh: v.SessionHigh, sessionLow: v.SessionLow, hasFirst: v.HasFirst, hasSessionExtrema: v.HasSessionExtrema, boundExceeded: v.BoundExceeded, result: unavailablePriceRangeResult(time.Time{})}
	var err error
	r.highs, err = restoreExtrema(v.Highs, b, t0)
	if err != nil {
		return nil, err
	}
	r.lows, err = restoreExtrema(v.Lows, b, t0)
	if err != nil {
		return nil, err
	}
	r.sessionHighs, err = restoreExtrema(v.SessionHighs, b, t0)
	if err != nil {
		return nil, err
	}
	r.sessionLows, err = restoreExtrema(v.SessionLows, b, t0)
	if err != nil {
		return nil, err
	}
	if len(r.highs) > maximumExtremaPointsPerDeque || len(r.lows) > maximumExtremaPointsPerDeque || len(r.sessionHighs) > sessionSeconds || len(r.sessionLows) > sessionSeconds || r.finalizedThrough > t0.Unix() || r.rollingFloor > t0.Unix() {
		return nil, errors.New("price range")
	}
	if (r.hasFirst && (r.firstStart < b.sessionStart.Unix() || r.firstStart >= t0.Unix() || !finitePositiveFeature(r.firstOpen))) ||
		(!r.hasFirst && (r.firstStart != 0 || r.firstOpen != 0)) ||
		(r.hasSessionExtrema && (!finitePositiveFeature(r.sessionHigh) || !finitePositiveFeature(r.sessionLow) || r.sessionHigh < r.sessionLow)) ||
		(!r.hasSessionExtrema && (r.sessionHigh != 0 || r.sessionLow != 0)) || len(r.sessionHighs) != len(r.sessionLows) {
		return nil, errors.New("price range scalars")
	}
	for i := range r.sessionHighs {
		if r.sessionHighs[i].windowStart != r.sessionLows[i].windowStart || r.sessionHighs[i].value < r.sessionLows[i].value {
			return nil, errors.New("price range evidence")
		}
	}
	return r, nil
}
func restoreExtrema(v []checkpoint.ExtremaPoint, b *installedBinding, t0 time.Time) ([]extremaPoint, error) {
	r := make([]extremaPoint, len(v))
	last := int64(0)
	for i, p := range v {
		if p.WindowStart < b.sessionStart.Unix() || p.WindowStart >= t0.Unix() || (i > 0 && p.WindowStart <= last) || !finiteFeature(p.Value) || p.Value <= 0 {
			return nil, errors.New("extrema")
		}
		r[i] = extremaPoint{windowStart: p.WindowStart, value: p.Value}
		last = p.WindowStart
	}
	return r, nil
}

func restoreActivity(v *checkpoint.Activity, b *installedBinding, t0 time.Time) (*activityFeatureState, error) {
	if v == nil {
		return nil, nil
	}
	r := &activityFeatureState{references: make(map[int64]*activityBlockSummary), mutable: make(map[int64]activityMutableBlock), foldedTargets: make(map[int64]activityFoldedTargetBlock), boundExceeded: v.BoundExceeded, result: unavailableActivityResult(time.Time{})}
	for _, s := range v.References {
		x := restoreActivitySummary(s)
		if s.End > b.sessionEnd.Unix() || s.End > t0.Unix() || r.references[s.End] != nil || !validCheckpointActivitySummary(x, s.End) {
			return nil, errors.New("activity reference")
		}
		normalizeCheckpointActivitySummary(&x)
		r.references[s.End] = &x
	}
	for _, m := range v.Mutable {
		folded, current := restoreActivitySummary(m.Folded), restoreActivitySummary(m.Current)
		blockStart := time.Unix(m.End, 0).UTC().Add(-activityBlockDuration)
		if m.End > b.sessionEnd.Unix() || !blockStart.Before(t0) || r.mutable[m.End].current.end != 0 || !validCheckpointActivitySummary(folded, m.End) || !validCheckpointActivitySummary(current, m.End) {
			return nil, errors.New("activity mutable")
		}
		normalizeCheckpointActivitySummary(&folded)
		normalizeCheckpointActivitySummary(&current)
		r.mutable[m.End] = activityMutableBlock{folded: folded, current: current}
	}
	count := 0
	for _, x := range v.FoldedTargets {
		if x.End > b.sessionEnd.Unix() || r.foldedTargets[x.End].present != 0 || x.Invalid&^x.Present != 0 {
			return nil, errors.New("activity target")
		}
		block := activityFoldedTargetBlock{transactions: x.Transactions, highs: x.Highs, lows: x.Lows, present: x.Present, invalid: x.Invalid}
		start := time.Unix(x.End, 0).UTC().Add(-activityBlockDuration)
		for slot := 0; slot < 30; slot++ {
			mask := uint32(1) << uint(slot)
			if block.present&mask == 0 {
				if block.transactions[slot] != 0 || block.highs[slot] != 0 || block.lows[slot] != 0 {
					return nil, errors.New("activity unused slot")
				}
				continue
			}
			if block.invalid&mask != 0 {
				if block.transactions[slot] != 0 || block.highs[slot] != 0 || block.lows[slot] != 0 {
					return nil, errors.New("activity invalid slot")
				}
			} else {
				if !start.Add(time.Duration(slot)*time.Second).Before(t0) || !finiteFeature(block.transactions[slot]) || block.transactions[slot] < 0 || !finitePositiveFeature(block.highs[slot]) || !finitePositiveFeature(block.lows[slot]) || block.highs[slot] < block.lows[slot] {
					return nil, errors.New("activity slot")
				}
			}
			count++
		}
		r.foldedTargets[x.End] = block
	}
	r.foldedTargetContributions = count
	if count != v.FoldedTargetContributions || len(r.references) > maximumActivityReferences || len(r.mutable) > maximumMutableActivityBlockIDs || len(r.foldedTargets) > maximumActivityTargetBlocks || count > maximumActivityTargetContributions {
		return nil, errors.New("activity bounds")
	}
	return r, nil
}

func validCheckpointActivitySummary(s activityBlockSummary, end int64) bool {
	if s.end != end || s.aggregateCount > 30 {
		return false
	}
	if s.invalid {
		return s.aggregateCount == 0 && s.transactionSum == (activityExactSum{}) && s.transactions == 0 && s.high == 0 && s.low == 0 && s.expansionBPS == 0
	}
	if s.aggregateCount == 0 {
		return s.transactionSum == (activityExactSum{}) && s.transactions == 0 && s.high == 0 && (s.low == 0 || math.IsInf(s.low, 1)) && s.expansionBPS == 0
	}
	wantTransactions, finite := s.transactionSum.float64()
	wantExpansion := 10_000 * math.Log(s.high/s.low)
	return finite && s.transactions == wantTransactions && finiteFeature(s.transactions) && s.transactions >= 0 && finitePositiveFeature(s.high) && finitePositiveFeature(s.low) &&
		s.high >= s.low && finiteFeature(s.expansionBPS) && s.expansionBPS == wantExpansion
}
func restoreActivitySummary(s checkpoint.ActivitySummary) activityBlockSummary {
	return activityBlockSummary{end: s.End, transactionSum: activityExactSum(s.TransactionSum), transactions: s.Transactions, high: s.High, low: s.Low, expansionBPS: s.ExpansionBPS, aggregateCount: s.AggregateCount, invalid: s.Invalid}
}

func normalizeCheckpointActivitySummary(s *activityBlockSummary) {
	if !s.invalid && s.aggregateCount == 0 {
		s.low = math.Inf(1)
	}
}

func restoreQualification(v *checkpoint.Qualification, b *installedBinding, t0 time.Time) (*qualificationState, error) {
	if v == nil {
		return nil, nil
	}
	r := &qualificationState{finalizedGateBars: make(map[int64]qualificationGateBar), proofs: make(map[int64]struct{}), dirty: make(map[int64]struct{}), accountedThrough: v.AccountedThrough, finalized: v.Finalized, finalProofEnd: v.FinalProofEnd, boundExceeded: v.BoundExceeded, invalid: v.Invalid, unresolvedOrigin: uncertaintyOrigin(v.UnresolvedOrigin), result: qualificationResult{status: qualificationUnresolved}}
	for _, x := range v.FinalizedGateBars {
		if x.Start < b.sessionStart.Unix() || x.Start >= t0.Unix() || r.finalizedGateBars[x.Start] != (qualificationGateBar{}) ||
			!finitePositiveFeature(x.Close) || !finiteFeature(x.Volume) || x.Volume < 0 || !finitePositiveFeature(x.VWAP) || x.AverageTradeSize < 0 ||
			(ATSProvenance(x.ATSProvenance) != ATSLiveProviderAverage && ATSProvenance(x.ATSProvenance) != ATSRESTFloorVolumeOverTrades) {
			return nil, errors.New("qualification bars")
		}
		r.finalizedGateBars[x.Start] = qualificationGateBar{close: x.Close, volume: x.Volume, vwap: x.VWAP, averageTradeSize: x.AverageTradeSize, provenance: ATSProvenance(x.ATSProvenance)}
	}
	for _, x := range v.Proofs {
		if x > b.sessionEnd.Unix() || x > t0.Unix() {
			return nil, errors.New("proof")
		}
		if _, ok := r.proofs[x]; ok {
			return nil, errors.New("proof duplicate")
		}
		r.proofs[x] = struct{}{}
	}
	for _, x := range v.Dirty {
		if _, ok := r.proofs[x]; !ok {
			return nil, errors.New("dirty")
		}
		r.dirty[x] = struct{}{}
	}
	if !validQualificationState(r, b) {
		return nil, errors.New("qualification")
	}
	if r.accountedThrough.After(t0) || r.unresolvedOrigin > uncertaintyLocalInvalid {
		return nil, errors.New("qualification cutoff")
	}
	return r, nil
}
