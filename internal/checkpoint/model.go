// Package checkpoint defines the detached semantic image exchanged between the
// scanner-state owner and Component 7's later codec/store boundary.  It owns no
// engine state and deliberately contains no provider or run-local positions.
package checkpoint

import "time"

const (
	SchemaV1     = "scanner-checkpoint-semantic-v1"
	ProducerLive = "live_aggregate"
)

// Binding is the complete compatibility-bearing session identity.
type Binding struct {
	Identity, TradingDate, ScheduleSchema, ScheduleVersion, ScheduleArtifactSHA256 string
	PriorSessionDate, UniversePolicy, UniverseIdentity                             string
	PriorClosePolicy, PriorCloseIdentity, Locale, Market                           string
	SessionStart, SessionEnd, PriorRegularClose                                    time.Time
	Adjusted, IncludeOTC                                                           bool
}

// Image is a detached, semantic as-of-T0 projection. Slice and map values are
// intentionally representation-neutral. Live persistence transfers one image
// into a Request; untrusted startup candidates are still defensively copied
// before admission and installation.
type Image struct {
	SchemaVersion, ProducerMode string
	Binding                     Binding
	T0, CreatedAt               time.Time
	Sequence                    uint64
	Population                  int
	Counts                      StructureCounts
	Symbols                     []Symbol
}

// StructureCounts make sparse allocation and every bounded variable-length
// structure explicit before codec allocation or engine installation.
type StructureCounts struct {
	RecordsWithState, EmptyStateRecords                            int
	TailRecords, PresenceWords, ProvenAbsentWords, ConflictWords   int
	PriceExtremaPoints                                             int
	ActivityReferences, ActivityMutable, ActivityTargetBlocks      int
	ActivityTargetContributions                                    int
	QualificationGateBars, QualificationProofs, QualificationDirty int
	InvalidMarks, CoverageConsequences                             int
}

// Candidate is the bounded semantic value supplied to the engine after the
// external codec has established its integrity identity. S1 uses the same
// shape in memory; S2 will supply the checksum from verified bytes.
type Candidate struct {
	Image     Image
	Checksum  string
	Integrity bool
}

type Values struct {
	Open, High, Low, Close float64
	Volume, VWAP           float64
	AverageTradeSize       int64
	ATSProvenance          string
}

type Aggregate struct {
	WindowStart, WindowEnd time.Time
	Values                 Values
}

type ExtremaPoint struct {
	WindowStart int64
	Value       float64
}

type PriceRange struct {
	FirstStart, RollingFloor, FinalizedThrough int64
	FirstOpen, SessionHigh, SessionLow         float64
	HasFirst, HasSessionExtrema                bool
	Highs, Lows                                []ExtremaPoint
	SessionHighs, SessionLows                  []ExtremaPoint
	BoundExceeded                              bool
}

type ExactSum [34]uint64

type ActivitySummary struct {
	End            int64
	TransactionSum ExactSum
	Transactions   float64
	High, Low      float64
	ExpansionBPS   float64
	AggregateCount uint8
	Invalid        bool
}

type ActivityMutable struct {
	End             int64
	Folded, Current ActivitySummary
}

type ActivityTargetBlock struct {
	End                       int64
	Transactions, Highs, Lows [30]float64
	Present, Invalid          uint32
}

type Activity struct {
	References                []ActivitySummary
	Mutable                   []ActivityMutable
	FoldedTargets             []ActivityTargetBlock
	FoldedTargetContributions int
	BoundExceeded             bool
}

type QualificationGateBar struct {
	Start               int64
	Close, Volume, VWAP float64
	AverageTradeSize    int64
	ATSProvenance       string
}

type Qualification struct {
	FinalizedGateBars []QualificationGateBar
	Proofs, Dirty     []int64
	AccountedThrough  time.Time
	Finalized         bool
	FinalProofEnd     time.Time
	BoundExceeded     bool
	Invalid           bool
	UnresolvedOrigin  uint8
}

type Coverage struct {
	Outcome, Origin uint8
}

// Symbol is explicit for every binding symbol. HasState distinguishes a real
// empty canonical record from an omitted record.
type Symbol struct {
	Symbol                   string
	HasState                 bool
	Tail                     []Aggregate
	OlderMark, CommittedMark *Aggregate
	Presence, ProvenAbsent   []uint64
	HistoricalConflict       []uint64
	PriceRange               *PriceRange
	Activity                 *Activity
	Qualification            *Qualification
	InvalidMarkStart         *time.Time
	Coverage                 *Coverage
}

// Clone returns a fully detached candidate/image graph.
func (i Image) Clone() Image {
	r := i
	r.Symbols = make([]Symbol, len(i.Symbols))
	for n := range i.Symbols {
		r.Symbols[n] = cloneSymbol(i.Symbols[n])
	}
	return r
}

func cloneSymbol(s Symbol) Symbol {
	r := s
	r.Tail = append([]Aggregate(nil), s.Tail...)
	r.Presence = append([]uint64(nil), s.Presence...)
	r.ProvenAbsent = append([]uint64(nil), s.ProvenAbsent...)
	r.HistoricalConflict = append([]uint64(nil), s.HistoricalConflict...)
	if s.OlderMark != nil {
		v := *s.OlderMark
		r.OlderMark = &v
	}
	if s.CommittedMark != nil {
		v := *s.CommittedMark
		r.CommittedMark = &v
	}
	if s.InvalidMarkStart != nil {
		v := *s.InvalidMarkStart
		r.InvalidMarkStart = &v
	}
	if s.Coverage != nil {
		v := *s.Coverage
		r.Coverage = &v
	}
	if s.PriceRange != nil {
		v := *s.PriceRange
		v.Highs = append([]ExtremaPoint(nil), s.PriceRange.Highs...)
		v.Lows = append([]ExtremaPoint(nil), s.PriceRange.Lows...)
		v.SessionHighs = append([]ExtremaPoint(nil), s.PriceRange.SessionHighs...)
		v.SessionLows = append([]ExtremaPoint(nil), s.PriceRange.SessionLows...)
		r.PriceRange = &v
	}
	if s.Activity != nil {
		v := *s.Activity
		v.References = append([]ActivitySummary(nil), s.Activity.References...)
		v.Mutable = append([]ActivityMutable(nil), s.Activity.Mutable...)
		v.FoldedTargets = append([]ActivityTargetBlock(nil), s.Activity.FoldedTargets...)
		r.Activity = &v
	}
	if s.Qualification != nil {
		v := *s.Qualification
		v.FinalizedGateBars = append([]QualificationGateBar(nil), s.Qualification.FinalizedGateBars...)
		v.Proofs = append([]int64(nil), s.Qualification.Proofs...)
		v.Dirty = append([]int64(nil), s.Qualification.Dirty...)
		r.Qualification = &v
	}
	return r
}
