package engine

import (
	"sort"
	"time"
	"unsafe"
)

// aggregatePrefix is the sealed, sufficient aggregate state. It contains no
// mutable aggregate identity and can only be changed by folding a canonical
// record once or withdrawing historical-only evidence that has not yet
// reached a terminal result in the current compatibility producer.
type aggregatePrefix struct {
	foldedThrough time.Time
	volume        float64
	printCount    uint32
	firstOpen     float64
	firstOpenAt   int64
	high          float64
	low           float64
	latest        AggregateValues
	latestAt      int64
	hasFirstOpen  bool
	hasExtrema    bool
	hasLatest     bool
	// A historical-only withdrawal can remove the sole supporting record after
	// it has folded. A false value is safer than fabricating the next extrema or
	// mark after the raw identity has been permanently discarded.
	firstOpenTrusted   bool
	extremaTrusted     bool
	latestTrusted      bool
	firstOpenUncertain bool
	extremaUncertain   bool
	latestUncertain    bool
}

type sealedLiveComparison struct {
	start  int64
	values AggregateValues
}

type AggregateCoverageClass uint8

const (
	AggregateCoverageUnknown AggregateCoverageClass = iota
	AggregateCoveragePresent
	AggregateCoverageProvenAbsent
	AggregateCoverageInvalid
	AggregateCoverageConflict
)

type AggregateProof uint16

const (
	AggregateProofMark AggregateProof = 1 << iota
	AggregateProofSessionVolume
	AggregateProofFirstOpen
	AggregateProofSessionExtrema
	AggregateProofQualification
	AggregateProofActivity30s
	AggregateProofMove30s
)

const aggregateProofAll = AggregateProofMark | AggregateProofSessionVolume | AggregateProofFirstOpen |
	AggregateProofSessionExtrema | AggregateProofQualification | AggregateProofActivity30s | AggregateProofMove30s

// AggregateAffectedView is a bounded, symbol-local notification. Successive
// mutations coalesce their exact time span and proof families; no mutable
// canonical state crosses the owner boundary.
type AggregateAffectedView struct {
	CanonicalRevision uint64
	From, Through     time.Time
	Proofs            AggregateProof
}

type aggregateAffectedState struct {
	revision      uint64
	from, through time.Time
	proofs        AggregateProof
}

// SelectionStateView is the compact immutable input for Capability B's
// population selection pass.
type SelectionStateView struct {
	Symbol                 string
	PriorClose             float64
	PriorCloseValid        bool
	TrustedMark            AggregateValues
	TrustedMarkAt          time.Time
	MarkAvailable          bool
	Coverage               AggregateCoverageClass
	CanonicalRevision      uint64
	Affected               AggregateAffectedView
	PrefixVolume           float64
	PrefixPrints           uint32
	PrefixFirstOpen        float64
	PrefixHigh, PrefixLow  float64
	PrefixFirstOpenTrusted bool
	PrefixExtremaTrusted   bool
	PrefixLatestTrusted    bool
}

// CanonicalAggregateView is an immutable value copy, never a writable alias.
type CanonicalAggregateView struct {
	WindowStart, WindowEnd time.Time
	Values                 AggregateValues
	Source                 AggregateSource
}

// SelectedAggregateView is the read-only as-of seam for Capability B. Tail is
// sorted and copied so callers cannot mutate or depend on map iteration order.
type SelectedAggregateView struct {
	SelectionStateView
	PrefixFoldedThrough time.Time
	Tail                []CanonicalAggregateView
	Bounds              CanonicalBoundsView
}

type CanonicalBoundsView struct {
	TailRecords, TailRecordLimit int
	TailChargedBytes             int
	TailChargedByteLimit         int
	CoverageBytes                int
	BoundHits                    uint64
}

var canonicalTailEntryCharge = int(unsafe.Sizeof(canonicalAggregate{})) + 32 // map bucket/pointer charge, conservatively rounded

func (p *aggregatePrefix) fold(record canonicalAggregate) {
	if p.printCount == 0 {
		if !p.firstOpenUncertain {
			p.firstOpen, p.firstOpenAt, p.hasFirstOpen, p.firstOpenTrusted = record.values.Open, record.identity.start, true, true
		}
		if !p.extremaUncertain {
			p.high, p.low, p.hasExtrema, p.extremaTrusted = record.values.High, record.values.Low, true, true
		}
		if !p.latestUncertain || record.identity.start > p.latestAt {
			p.latest, p.latestAt, p.hasLatest, p.latestTrusted = record.values, record.identity.start, true, true
			p.latestUncertain = false
		}
	} else {
		if !p.firstOpenUncertain && (!p.hasFirstOpen || record.identity.start < p.firstOpenAt) {
			p.firstOpen, p.firstOpenAt, p.hasFirstOpen, p.firstOpenTrusted = record.values.Open, record.identity.start, true, true
		}
		if !p.extremaUncertain && (!p.hasExtrema || record.values.High > p.high) {
			p.high = record.values.High
		}
		if !p.extremaUncertain && (!p.hasExtrema || record.values.Low < p.low) {
			p.low = record.values.Low
		}
		if !p.extremaUncertain {
			p.hasExtrema, p.extremaTrusted = true, true
		}
		if (!p.latestUncertain || record.identity.start > p.latestAt) && (!p.hasLatest || record.identity.start > p.latestAt) {
			p.latest, p.latestAt, p.hasLatest, p.latestTrusted = record.values, record.identity.start, true, true
			p.latestUncertain = false
		}
	}
	p.volume += record.values.Volume
	p.printCount++
	if record.windowEnd.After(p.foldedThrough) {
		p.foldedThrough = record.windowEnd
	}
}

func (p *aggregatePrefix) withdrawHistorical(record canonicalAggregate) {
	if p.printCount == 0 {
		return
	}
	p.volume -= record.values.Volume
	if p.volume < 0 {
		p.volume = 0
	}
	p.printCount--
	if record.identity.start == p.firstOpenAt {
		p.hasFirstOpen, p.firstOpenTrusted, p.firstOpenUncertain = false, false, true
	}
	if p.hasExtrema && (record.values.High == p.high || record.values.Low == p.low) {
		p.hasExtrema, p.extremaTrusted, p.extremaUncertain = false, false, true
	}
	if record.identity.start == p.latestAt {
		p.hasLatest, p.latestTrusted, p.latestUncertain = false, false, true
	}
}

func (state *symbolAggregateState) notifyAggregate(record canonicalAggregate, proofs AggregateProof) {
	state.canonicalRevision++
	if state.affected.proofs == 0 || record.windowStart.Before(state.affected.from) {
		state.affected.from = record.windowStart
	}
	through := record.windowEnd.Add(correctionHorizon)
	if state.affected.proofs == 0 || through.After(state.affected.through) {
		state.affected.through = through
	}
	state.affected.revision = state.canonicalRevision
	state.affected.proofs |= proofs
}

func (e *Engine) selectionStateViewLocked(index int) SelectionStateView {
	return e.selectionStateViewAtLocked(index, time.Time{})
}

// selectionStateViewAtLocked returns the compact selection scalars at one
// candidate boundary. A zero boundary is the diagnostic latest-delivery view;
// production selection always supplies T. In particular, a bar beginning at T
// is outside [S,T) and cannot displace the retained predecessor mark.
func (e *Engine) selectionStateViewAtLocked(index int, at time.Time) SelectionStateView {
	symbol := &e.state.binding.symbols[index]
	state := symbol.aggregates
	view := SelectionStateView{Symbol: symbol.symbol, PriorClose: symbol.prior.close, PriorCloseValid: symbol.prior.status == "valid"}
	if state == nil {
		return view
	}
	view.CanonicalRevision = state.canonicalRevision
	view.Affected = AggregateAffectedView{CanonicalRevision: state.affected.revision, From: state.affected.from, Through: state.affected.through, Proofs: state.affected.proofs}
	view.PrefixVolume, view.PrefixPrints = state.prefix.volume, state.prefix.printCount
	view.PrefixFirstOpen, view.PrefixHigh, view.PrefixLow = state.prefix.firstOpen, state.prefix.high, state.prefix.low
	view.PrefixFirstOpenTrusted, view.PrefixExtremaTrusted, view.PrefixLatestTrusted = state.prefix.firstOpenTrusted, state.prefix.extremaTrusted, state.prefix.latestTrusted
	var mark canonicalAggregate
	var hasMark bool
	if at.IsZero() {
		if state.latest != nil {
			mark, hasMark = state.latest.record, true
		}
	} else {
		mark, hasMark = latestMarkBeforeCompact(state, at)
	}
	if hasMark {
		view.TrustedMark, view.TrustedMarkAt, view.MarkAvailable = mark.values, mark.windowStart, true
		view.Coverage = coverageClassAt(state, e.state.binding, mark.windowStart)
	}
	if invalid, ok := e.state.aggregateEvaluator.invalidMarks[index]; ok && (!view.MarkAvailable || !invalid.windowStart.Before(view.TrustedMarkAt)) {
		view.Coverage = AggregateCoverageInvalid
	}
	return view
}

func (e *Engine) selectedAggregateViewLocked(index int) SelectedAggregateView {
	view := SelectedAggregateView{SelectionStateView: e.selectionStateViewLocked(index)}
	state := e.state.binding.symbols[index].aggregates
	if state == nil {
		return view
	}
	view.PrefixFoldedThrough = state.prefix.foldedThrough
	coverageBytes := 0
	for _, bitmap := range []*slotBitmap{state.presence, state.sealedLive, state.provenAbsent, state.historicalConflict} {
		if bitmap != nil {
			coverageBytes += int(unsafe.Sizeof(*bitmap))
		}
	}
	view.Bounds = CanonicalBoundsView{TailRecords: len(state.tail), TailRecordLimit: maximumTailRecords,
		TailChargedBytes: len(state.tail) * canonicalTailEntryCharge, TailChargedByteLimit: maximumTailRecords * canonicalTailEntryCharge,
		CoverageBytes: coverageBytes, BoundHits: state.tailBoundHits}
	view.Tail = make([]CanonicalAggregateView, 0, len(state.tail))
	for _, record := range state.tail {
		if record == nil {
			continue
		}
		view.Tail = append(view.Tail, CanonicalAggregateView{WindowStart: record.windowStart, WindowEnd: record.windowEnd, Values: record.values, Source: record.authority.source})
	}
	sort.Slice(view.Tail, func(i, j int) bool { return view.Tail[i].WindowStart.Before(view.Tail[j].WindowStart) })
	return view
}

func coverageClassAt(state *symbolAggregateState, binding *installedBinding, at time.Time) AggregateCoverageClass {
	if state == nil || binding == nil || at.Before(binding.sessionStart) || !at.Before(binding.sessionEnd) {
		return AggregateCoverageUnknown
	}
	slot := sessionSlot(binding, at)
	if state.historicalConflict != nil && state.historicalConflict.has(slot) {
		return AggregateCoverageConflict
	}
	if aggregatePresentAt(state, at.Unix()) || state.presence != nil && state.presence.has(slot) {
		return AggregateCoveragePresent
	}
	if state.provenAbsent != nil && state.provenAbsent.has(slot) {
		return AggregateCoverageProvenAbsent
	}
	return AggregateCoverageUnknown
}

// observeSelectionState returns one immutable value per binding index.
func (e *Engine) observeSelectionState() []SelectionStateView {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.binding == nil {
		return nil
	}
	views := make([]SelectionStateView, len(e.state.binding.symbols))
	for index := range views {
		views[index] = e.selectionStateViewLocked(index)
	}
	return views
}

// observeSelectedAggregate returns a detached canonical view for one bound
// symbol. The returned tail contains value copies and is safe for any reader.
func (e *Engine) observeSelectedAggregate(symbol string) (SelectedAggregateView, bool) {
	if e == nil {
		return SelectedAggregateView{}, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.binding == nil {
		return SelectedAggregateView{}, false
	}
	index, ok := e.state.binding.index[symbol]
	if !ok {
		return SelectedAggregateView{}, false
	}
	return e.selectedAggregateViewLocked(index), true
}
