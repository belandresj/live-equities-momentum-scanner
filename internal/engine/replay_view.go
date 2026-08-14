package engine

import (
	"math/bits"
	"sort"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

// ReplayDeterministicView is an internal, defensive proof/output projection of
// the Component 1-3 state owned by the engine. It is not the Component 10 API
// schema and carries no writable alias into canonical or publication state.
type ReplayDeterministicView struct {
	Canonical   []ReplayCanonicalSymbol
	Evaluation  ReplayEvaluationView
	Publication ReplayPublicationView
}

type ReplayCanonicalSymbol struct {
	Symbol                string
	PriorStatus           reference.PriorCloseStatus
	PriorClose            float64
	Records               []ReplayCanonicalRecord
	LatestWindowStart     time.Time
	LatestValues          AggregateValues
	LatestAuthoritySource AggregateSource
	CommittedWindowStart  time.Time
	PresentSlots          uint64
	ProvenAbsentSlots     uint64
	Features              ReplayFeatureView
	Qualification         ReplayQualificationView
}

type ReplayCanonicalRecord struct {
	WindowStart, WindowEnd time.Time
	Values                 AggregateValues
	FirstSource            AggregateSource
	FirstDeliveryTime      time.Time
	FirstReplayOrdinal     uint64
	AuthoritySource        AggregateSource
	AuthorityDeliveryTime  time.Time
	AuthorityReplayOrdinal uint64
}

type ReplayFieldView struct {
	Status, Reason string
	Value          float64
}

type ReplayFeatureView struct {
	At                                      time.Time
	DayPercent, From4AMPercent, HODDrawdown ReplayFieldView
	SessionRange, Rolling30, Rolling60      ReplayFieldView
	Activity                                ReplayFieldView
}

type ReplayQualificationView struct {
	At                time.Time
	Status            string
	UnresolvedOrigin  string
	CurrentProofCount int
	FinalProofEnd     time.Time
}

type ReplayPopulationView struct {
	UniverseTotal, ValidPriorClose, InvalidOrMissingPriorClose uint64
	TrustedRankableMark, TrustedBelowPriceMark                 uint64
	NoPrintThroughT, InvalidMark, UnknownDueFailureOrFence     uint64
	CoveredPopulation, UnresolvedPopulation                    uint64
}

type ReplayQualificationAccountingView struct {
	NotYetPassed, Provisional, Finalized, Unresolved uint64
}

type ReplayFeatureAccountingView struct {
	Statuses [4]uint64
	Reasons  [featureReasonBucketCount]uint64
	Pairs    [4][featureReasonBucketCount]uint64
}

type ReplayAllFeatureAccountingView struct {
	DayPercent, From4AMPercent, HODDrawdown, SessionRange ReplayFeatureAccountingView
	Rolling30, Rolling60, Activity                        ReplayFeatureAccountingView
}

type ReplayUncertaintyView struct {
	BootstrapOrigin, PostBootstrapGap, LocalInvalid uint64
}

type ReplayRankingRowView struct {
	Rank                                      uint32
	Symbol                                    string
	Last, DayPercent                          float64
	MarkAge                                   time.Duration
	From4AMPercent, HODDrawdown, SessionRange ReplayFieldView
	Rolling30, Rolling60, Activity            ReplayFieldView
	TQIntentEligible                          bool
}

type ReplayEvaluationView struct {
	At                                      time.Time
	Mode, Reason                            string
	Population                              ReplayPopulationView
	Qualification                           ReplayQualificationAccountingView
	Features                                ReplayAllFeatureAccountingView
	Uncertainty                             ReplayUncertaintyView
	TotalPassers, KnownRankableCount        uint64
	DayInvalidRankable, QualifiedDayInvalid uint64
	TQIntentAvailable                       bool
	Rows                                    []ReplayRankingRowView
}

type ReplayPublicationView struct {
	SchemaVersion       string
	PublicationID       uint64
	BindingIdentity     string
	TradingDate         string
	RunMode             RunMode
	Lifecycle           string
	LifecycleReason     string
	Suppression         SuppressionDisposition
	LastDisposition     DispositionCode
	DispositionReason   DispositionReason
	LastEngineSequence  uint64
	Watermark           *time.Time
	GeneratedAt         time.Time
	CurrentMarketClaim  bool
	AggregateEvaluation ReplayEvaluationView
}

func (e *Engine) replayDeterministicViewLocked() ReplayDeterministicView {
	result := ReplayDeterministicView{Evaluation: replayEvaluationView(e.state.aggregateEvaluator.current)}
	if e.state.binding != nil {
		result.Canonical = make([]ReplayCanonicalSymbol, len(e.state.binding.symbols))
		for index := range e.state.binding.symbols {
			result.Canonical[index] = replayCanonicalSymbolView(e.state.binding.symbols[index])
		}
	}
	result.Publication = replayPublicationView(e.publication.Load())
	return result
}

// ObserveReplayDeterministic returns the defensive Component 1-3 proof view
// used by P-C4-CORE. Ordinary replay status excludes the per-symbol copy.
func (e *Engine) ObserveReplayDeterministic() ReplayDeterministicView {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.replayDeterministicViewLocked()
}

func replayPublicationView(publication *privatePublication) ReplayPublicationView {
	if publication == nil {
		return ReplayPublicationView{}
	}
	return ReplayPublicationView{
		SchemaVersion: publication.schemaVersion, PublicationID: publication.publicationID,
		BindingIdentity: publication.bindingIdentity, TradingDate: publication.tradingDate,
		RunMode: publication.mode, Lifecycle: string(publication.lifecycle), LifecycleReason: string(publication.lifecycleReason),
		Suppression: publication.suppressionDisposition, LastDisposition: publication.lastDisposition,
		DispositionReason: publication.dispositionReason, LastEngineSequence: publication.lastEngineSequence,
		Watermark: immutableTimePointer(publication.watermark), GeneratedAt: publication.generatedAt,
		CurrentMarketClaim:  publication.currentMarketClaim,
		AggregateEvaluation: replayEvaluationView(publication.aggregateEvaluation),
	}
}

func replayCanonicalSymbolView(symbol coreSymbol) ReplayCanonicalSymbol {
	result := ReplayCanonicalSymbol{Symbol: symbol.symbol, PriorStatus: symbol.prior.status, PriorClose: symbol.prior.close}
	state := symbol.aggregates
	if state == nil {
		return result
	}
	starts := make([]int64, 0, len(state.tail))
	for start := range state.tail {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	result.Records = make([]ReplayCanonicalRecord, 0, len(starts))
	for _, start := range starts {
		record := state.tail[start]
		if record == nil {
			continue
		}
		result.Records = append(result.Records, ReplayCanonicalRecord{
			WindowStart: record.windowStart, WindowEnd: record.windowEnd, Values: record.values,
			FirstSource: record.first.source, FirstDeliveryTime: record.first.deliveryTime, FirstReplayOrdinal: record.first.replay.RecordOrdinal,
			AuthoritySource: record.authority.source, AuthorityDeliveryTime: record.authority.deliveryTime, AuthorityReplayOrdinal: record.authority.replay.RecordOrdinal,
		})
	}
	if state.latest != nil {
		result.LatestWindowStart = state.latest.record.windowStart
		result.LatestValues = state.latest.record.values
		result.LatestAuthoritySource = state.latest.record.authority.source
	}
	if state.committedLatest != nil {
		result.CommittedWindowStart = state.committedLatest.windowStart
	}
	result.PresentSlots = bitmapPopulation(state.presence)
	result.ProvenAbsentSlots = bitmapPopulation(state.provenAbsent)
	features := unavailablePriceRangeResult(time.Time{})
	activity := unavailableActivityResult(time.Time{})
	if state.priceRange != nil {
		features = state.priceRange.result
	}
	if state.activity != nil {
		activity = state.activity.result
	}
	result.Features = ReplayFeatureView{
		At: features.at, DayPercent: replayFieldView(features.dayPercent), From4AMPercent: replayFieldView(features.from4AMPercent),
		HODDrawdown: replayFieldView(features.hodDrawdown), SessionRange: replayFieldView(features.sessionRange),
		Rolling30: replayFieldView(features.rolling30), Rolling60: replayFieldView(features.rolling60), Activity: replayFieldView(activity.activity),
	}
	if state.qualification != nil {
		qualification := state.qualification.result
		result.Qualification = ReplayQualificationView{At: qualification.at, Status: string(qualification.status),
			UnresolvedOrigin: replayUncertaintyOrigin(qualification.unresolvedOrigin), CurrentProofCount: qualification.currentProofCount,
			FinalProofEnd: qualification.finalProofEnd}
	}
	return result
}

func bitmapPopulation(bitmap *slotBitmap) uint64 {
	if bitmap == nil {
		return 0
	}
	var result uint64
	for _, word := range bitmap {
		result += uint64(bits.OnesCount64(word))
	}
	return result
}

func replayFieldView(field aggregateFeatureField) ReplayFieldView {
	return ReplayFieldView{Status: string(field.status), Reason: string(field.reason), Value: field.value}
}

func replayEvaluationView(value aggregateEvaluationResult) ReplayEvaluationView {
	result := ReplayEvaluationView{
		At: value.at, Mode: string(value.mode), Reason: string(value.reason),
		Population: ReplayPopulationView{
			UniverseTotal: value.population.universeTotal, ValidPriorClose: value.population.validPriorClose,
			InvalidOrMissingPriorClose: value.population.invalidOrMissingPriorClose, TrustedRankableMark: value.population.trustedRankableMark,
			TrustedBelowPriceMark: value.population.trustedBelowPriceMark, NoPrintThroughT: value.population.noPrintThroughT,
			InvalidMark: value.population.invalidMark, UnknownDueFailureOrFence: value.population.unknownDueFailureOrFence,
			CoveredPopulation: value.population.coveredPopulation, UnresolvedPopulation: value.population.unresolvedPopulation,
		},
		Qualification: ReplayQualificationAccountingView{NotYetPassed: value.qualification.notYetPassed, Provisional: value.qualification.provisional,
			Finalized: value.qualification.finalized, Unresolved: value.qualification.unresolved},
		Features: ReplayAllFeatureAccountingView{
			DayPercent: replayFeatureAccountingView(value.features.dayPercent), From4AMPercent: replayFeatureAccountingView(value.features.from4AMPercent),
			HODDrawdown: replayFeatureAccountingView(value.features.hodDrawdown), SessionRange: replayFeatureAccountingView(value.features.sessionRange),
			Rolling30: replayFeatureAccountingView(value.features.rolling30), Rolling60: replayFeatureAccountingView(value.features.rolling60),
			Activity: replayFeatureAccountingView(value.features.activity),
		},
		Uncertainty:  ReplayUncertaintyView{BootstrapOrigin: value.uncertainty.bootstrapOrigin, PostBootstrapGap: value.uncertainty.postBootstrapGap, LocalInvalid: value.uncertainty.localInvalid},
		TotalPassers: value.totalPassers, KnownRankableCount: value.knownRankableCount,
		DayInvalidRankable: value.dayInvalidRankable, QualifiedDayInvalid: value.qualifiedDayInvalid,
		TQIntentAvailable: value.tqIntentAvailable,
		Rows:              make([]ReplayRankingRowView, len(value.rows)),
	}
	for index, row := range value.rows {
		result.Rows[index] = ReplayRankingRowView{Rank: row.rank, Symbol: row.symbol, Last: row.last, DayPercent: row.dayPercent,
			MarkAge: row.markAge, From4AMPercent: replayFieldView(row.from4AMPercent), HODDrawdown: replayFieldView(row.hodDrawdown),
			SessionRange: replayFieldView(row.sessionRange), Rolling30: replayFieldView(row.rolling30), Rolling60: replayFieldView(row.rolling60),
			Activity: replayFieldView(row.activity), TQIntentEligible: row.tqIntentEligible}
	}
	return result
}


func replayFeatureAccountingView(value featureDimensionAccounting) ReplayFeatureAccountingView {
	return ReplayFeatureAccountingView{Statuses: value.statuses, Reasons: value.reasons, Pairs: value.pairs}
}

func replayUncertaintyOrigin(value uncertaintyOrigin) string {
	switch value {
	case uncertaintyBootstrapOrigin:
		return "bootstrap_origin"
	case uncertaintyPostBootstrapGap:
		return "post_bootstrap_gap"
	case uncertaintyLocalInvalid:
		return "local_invalid"
	default:
		return ""
	}
}
