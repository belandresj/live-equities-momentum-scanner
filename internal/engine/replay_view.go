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
	PresentBitmap         []uint64
	ProvenAbsentBitmap    []uint64
	HistoricalConflict    []uint64
	OlderLatest           *ReplayCanonicalRecord
	Recomputations        uint64
	TailCoverage          ReplayTailCoverageView
	Features              ReplayFeatureView
	Qualification         ReplayQualificationView
	PriceRangeState       ReplayPriceRangeStateView
	ActivityState         ReplayActivityStateView
	QualificationState    ReplayQualificationStateView
	MVPState              ReplayMVPStateView
}

type ReplayTailCoverageView struct {
	BaseWord, TailCount  int
	Words                [16]uint64
	Valid, Built, Usable bool
}

type ReplayExtremaPointView struct {
	WindowStart int64
	Value       float64
}

type ReplayPriceRangeStateView struct {
	Present                                    bool
	FirstStart, RollingFloor, FinalizedThrough int64
	FirstOpen, SessionHigh, SessionLow         float64
	HasFirst, HasSessionExtrema, BoundExceeded bool
	Highs, Lows, SessionHighs, SessionLows     []ReplayExtremaPointView
}

type ReplayActivitySummaryView struct {
	End                                   int64
	TransactionSum                        [34]uint64
	Transactions, High, Low, ExpansionBPS float64
	AggregateCount                        uint8
	Invalid                               bool
}

type ReplayActivityMutableView struct {
	End             int64
	Folded, Current ReplayActivitySummaryView
}
type ReplayActivityTargetView struct {
	End                       int64
	Transactions, Highs, Lows [30]float64
	Present, Invalid          uint32
}
type ReplayActivityStateView struct {
	Present                                                                            bool
	References                                                                         []ReplayActivitySummaryView
	Mutable                                                                            []ReplayActivityMutableView
	FoldedTargets                                                                      []ReplayActivityTargetView
	FoldedTargetContributions                                                          int
	BoundExceeded                                                                      bool
	ResultAt                                                                           time.Time
	ResultActivity                                                                     ReplayFieldView
	TargetTransactions, TargetExpansionBPS, TransactionPercentile, ExpansionPercentile float64
	ReferenceCount                                                                     int
	LookupAt                                                                           time.Time
	LookupTransactions, LookupExpansions                                               []float64
	LookupValid                                                                        bool
}

type ReplayQualificationGateBarView struct {
	Start               int64
	Close, Volume, VWAP float64
	AverageTradeSize    int64
	ATSProvenance       string
}
type ReplayQualificationStateView struct {
	Present                                         bool
	FinalizedGateBars                               []ReplayQualificationGateBarView
	Proofs, Dirty                                   []int64
	AccountedThrough                                time.Time
	Finalized                                       bool
	FinalProofEnd                                   time.Time
	BoundExceeded, Invalid                          bool
	UnresolvedOrigin                                string
	Installed, Revalidated, Revoked, FinalizedCount uint64
	MaximumProofOccupancy, MaximumDirtyOccupancy    int
	HydrationScanPresent                            bool
}

type ReplayMVPPointView struct {
	Start                     int64
	Volume, Close, Cumulative float64
}
type ReplayMVPStateView struct {
	Present                             bool
	Folded                              []ReplayMVPPointView
	At                                  time.Time
	SessionVolume, Activity30s, Move30s ReplayFieldView
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
	SessionVolume, FromOpenPercent, DayRange              ReplayFeatureAccountingView
	Activity30s, Move30s                                  ReplayFeatureAccountingView
	DayPercent, From4AMPercent, HODDrawdown, SessionRange ReplayFeatureAccountingView
	Rolling30, Rolling60, Activity                        ReplayFeatureAccountingView
}

type ReplayFloatAccountingView struct {
	Current, Stale, Unavailable, Invalid uint64
}

type ReplayUncertaintyView struct {
	BootstrapOrigin, PostBootstrapGap, LocalInvalid uint64
}

type ReplayPopulationTransitionDiagnosticView struct {
	BootstrapUnknown, TrustedByLaterLiveMark, NoLaterEligibleMark uint64
	LatestMarkNotLiveAuthority, NoStrictlyOlderLocalizedConflict  uint64
	ConflictAtOrAfterMark, InvalidAtOrAfterMark                   uint64
	IncompletePostMarkCoverage                                    uint64
}

type ReplayRankingRowView struct {
	Rank                                     uint32
	Symbol                                   string
	Last, DayPercent                         float64
	MarkAge                                  time.Duration
	Float                                    ReplayFloatFieldView
	SessionVolume, FromOpenPercent, DayRange ReplayFieldView
	Activity30s, Move30s                     ReplayFieldView
	TQIntentEligible                         bool
}

type ReplayFloatFieldView struct {
	Status, Reason          string
	Value                   float64
	Percent                 *float64
	Provider, EffectiveDate string
	RetrievedAt             time.Time
	Provenance              string
}

type ReplayEvaluationView struct {
	At                                      time.Time
	Mode, Reason                            string
	Population                              ReplayPopulationView
	Qualification                           ReplayQualificationAccountingView
	Features                                ReplayAllFeatureAccountingView
	Floats                                  ReplayFloatAccountingView
	Uncertainty                             ReplayUncertaintyView
	PopulationTransition                    ReplayPopulationTransitionDiagnosticView
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

// ValidateReplayEvaluationAccounting validates only the fixed-cardinality
// accounting projection. Ranking rows are intentionally excluded so incident
// diagnostics can retain coherent market accounting without retaining symbol
// identities.
func ValidateReplayEvaluationAccounting(r ReplayEvaluationView) bool {
	p := r.Population
	if p.UniverseTotal != p.ValidPriorClose+p.InvalidOrMissingPriorClose ||
		p.ValidPriorClose != p.TrustedRankableMark+p.TrustedBelowPriceMark+p.NoPrintThroughT+p.InvalidMark+p.UnknownDueFailureOrFence ||
		p.CoveredPopulation != p.UniverseTotal-p.UnknownDueFailureOrFence || p.UnresolvedPopulation != p.UnknownDueFailureOrFence {
		return false
	}
	d := r.PopulationTransition
	if d.BootstrapUnknown != d.TrustedByLaterLiveMark+d.NoLaterEligibleMark+d.LatestMarkNotLiveAuthority+d.NoStrictlyOlderLocalizedConflict+d.ConflictAtOrAfterMark+d.InvalidAtOrAfterMark+d.IncompletePostMarkCoverage ||
		d.TrustedByLaterLiveMark > p.TrustedRankableMark+p.TrustedBelowPriceMark || d.BootstrapUnknown-d.TrustedByLaterLiveMark > p.UnknownDueFailureOrFence {
		return false
	}
	q := r.Qualification
	if q.NotYetPassed+q.Provisional+q.Finalized+q.Unresolved != p.TrustedRankableMark ||
		r.TotalPassers+r.QualifiedDayInvalid != q.Provisional+q.Finalized ||
		r.KnownRankableCount+r.DayInvalidRankable != p.TrustedRankableMark || r.QualifiedDayInvalid > r.DayInvalidRankable {
		return false
	}
	if r.Uncertainty.BootstrapOrigin+r.Uncertainty.PostBootstrapGap+r.Uncertainty.LocalInvalid != p.UnknownDueFailureOrFence+q.Unresolved {
		return false
	}
	for _, dimension := range []ReplayFeatureAccountingView{r.Features.DayPercent, r.Features.SessionVolume, r.Features.FromOpenPercent, r.Features.DayRange, r.Features.Activity30s, r.Features.Move30s} {
		var statuses, reasons, pairs uint64
		var pairStatuses [4]uint64
		var pairReasons [featureReasonBucketCount]uint64
		for status := range dimension.Statuses {
			statuses += dimension.Statuses[status]
			for reason := range dimension.Reasons {
				pairStatuses[status] += dimension.Pairs[status][reason]
				pairReasons[reason] += dimension.Pairs[status][reason]
				pairs += dimension.Pairs[status][reason]
			}
		}
		for _, count := range dimension.Reasons {
			reasons += count
		}
		if statuses != p.UniverseTotal || reasons != p.UniverseTotal || pairs != p.UniverseTotal || pairStatuses != dimension.Statuses || pairReasons != dimension.Reasons {
			return false
		}
	}
	if r.Floats.Current+r.Floats.Stale+r.Floats.Unavailable+r.Floats.Invalid != p.UniverseTotal {
		return false
	}
	return true
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
	result.PresentBitmap = replayBitmapView(state.presence)
	result.ProvenAbsentBitmap = replayBitmapView(state.provenAbsent)
	result.HistoricalConflict = replayBitmapView(state.historicalConflict)
	result.Recomputations = state.recomputations
	result.TailCoverage = ReplayTailCoverageView{BaseWord: state.tailCoverage.baseWord, TailCount: state.tailCoverage.tailCount,
		Words: state.tailCoverage.words, Valid: state.tailCoverage.valid, Built: state.tailCoverageBuilt, Usable: state.tailCoverageUsable}
	if state.olderLatest != nil {
		value := replayCanonicalRecordView(*state.olderLatest)
		result.OlderLatest = &value
	}
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
	result.PriceRangeState = replayPriceRangeStateView(state.priceRange)
	result.ActivityState = replayActivityStateView(state.activity)
	result.QualificationState = replayQualificationStateView(state.qualification)
	result.MVPState = replayMVPStateView(state.mvpMeasurements)
	return result
}

func replayBitmapView(bitmap *slotBitmap) []uint64 {
	if bitmap == nil {
		return nil
	}
	return append([]uint64(nil), bitmap[:]...)
}

func replayCanonicalRecordView(record canonicalAggregate) ReplayCanonicalRecord {
	return ReplayCanonicalRecord{WindowStart: record.windowStart, WindowEnd: record.windowEnd, Values: record.values,
		FirstSource: record.first.source, FirstDeliveryTime: record.first.deliveryTime, FirstReplayOrdinal: record.first.replay.RecordOrdinal,
		AuthoritySource: record.authority.source, AuthorityDeliveryTime: record.authority.deliveryTime, AuthorityReplayOrdinal: record.authority.replay.RecordOrdinal}
}

func replayExtremaPointsView(points []extremaPoint) []ReplayExtremaPointView {
	result := make([]ReplayExtremaPointView, len(points))
	for index, point := range points {
		result[index] = ReplayExtremaPointView{WindowStart: point.windowStart, Value: point.value}
	}
	return result
}

func replayPriceRangeStateView(state *priceRangeFeatureState) ReplayPriceRangeStateView {
	if state == nil {
		return ReplayPriceRangeStateView{}
	}
	return ReplayPriceRangeStateView{Present: true, FirstStart: state.firstStart, RollingFloor: state.rollingFloor,
		FinalizedThrough: state.finalizedThrough, FirstOpen: state.firstOpen, SessionHigh: state.sessionHigh, SessionLow: state.sessionLow,
		HasFirst: state.hasFirst, HasSessionExtrema: state.hasSessionExtrema, BoundExceeded: state.boundExceeded,
		Highs: replayExtremaPointsView(state.highs), Lows: replayExtremaPointsView(state.lows),
		SessionHighs: replayExtremaPointsView(state.sessionHighs), SessionLows: replayExtremaPointsView(state.sessionLows)}
}

func replayActivitySummaryView(value activityBlockSummary) ReplayActivitySummaryView {
	return ReplayActivitySummaryView{End: value.end, TransactionSum: value.transactionSum, Transactions: value.transactions,
		High: value.high, Low: value.low, ExpansionBPS: value.expansionBPS, AggregateCount: value.aggregateCount, Invalid: value.invalid}
}

func replayActivityStateView(state *activityFeatureState) ReplayActivityStateView {
	if state == nil {
		return ReplayActivityStateView{}
	}
	result := ReplayActivityStateView{Present: true, FoldedTargetContributions: state.foldedTargetContributions,
		BoundExceeded: state.boundExceeded, ResultAt: state.result.at, ResultActivity: replayFieldView(state.result.activity),
		TargetTransactions: state.result.targetTransactions, TargetExpansionBPS: state.result.targetExpansionBPS,
		TransactionPercentile: state.result.transactionPercentile, ExpansionPercentile: state.result.expansionPercentile,
		ReferenceCount: state.result.referenceCount, LookupAt: state.referenceLookup.at,
		LookupTransactions: append([]float64(nil), state.referenceLookup.transactions...),
		LookupExpansions:   append([]float64(nil), state.referenceLookup.expansions...), LookupValid: state.referenceLookup.valid}
	for _, end := range sortedInt64Keys(state.references) {
		result.References = append(result.References, replayActivitySummaryView(*state.references[end]))
	}
	for _, end := range sortedInt64Keys(state.mutable) {
		value := state.mutable[end]
		result.Mutable = append(result.Mutable, ReplayActivityMutableView{End: end, Folded: replayActivitySummaryView(value.folded), Current: replayActivitySummaryView(value.current)})
	}
	for _, end := range sortedInt64Keys(state.foldedTargets) {
		value := state.foldedTargets[end]
		result.FoldedTargets = append(result.FoldedTargets, ReplayActivityTargetView{End: end, Transactions: value.transactions, Highs: value.highs, Lows: value.lows, Present: value.present, Invalid: value.invalid})
	}
	return result
}

func replayQualificationStateView(state *qualificationState) ReplayQualificationStateView {
	if state == nil {
		return ReplayQualificationStateView{}
	}
	result := ReplayQualificationStateView{Present: true, Proofs: sortedSetKeys(state.proofs), Dirty: sortedSetKeys(state.dirty),
		AccountedThrough: state.accountedThrough, Finalized: state.finalized, FinalProofEnd: state.finalProofEnd,
		BoundExceeded: state.boundExceeded, Invalid: state.invalid, UnresolvedOrigin: replayUncertaintyOrigin(state.unresolvedOrigin),
		Installed: state.installed, Revalidated: state.revalidated, Revoked: state.revoked, FinalizedCount: state.finalizedCount,
		MaximumProofOccupancy: state.maximumProofOccupancy, MaximumDirtyOccupancy: state.maximumDirtyOccupancy,
		HydrationScanPresent: state.hydrationScan != nil}
	for _, start := range sortedInt64Keys(state.finalizedGateBars) {
		value := state.finalizedGateBars[start]
		result.FinalizedGateBars = append(result.FinalizedGateBars, ReplayQualificationGateBarView{Start: start, Close: value.close, Volume: value.volume, VWAP: value.vwap, AverageTradeSize: value.averageTradeSize, ATSProvenance: string(value.provenance)})
	}
	return result
}

func replayMVPStateView(state *mvpMeasurementState) ReplayMVPStateView {
	if state == nil {
		return ReplayMVPStateView{}
	}
	result := ReplayMVPStateView{Present: true, At: state.result.at, SessionVolume: replayFieldView(state.result.sessionVolume),
		Activity30s: replayFieldView(state.result.activity30s), Move30s: replayFieldView(state.result.move30s), Folded: make([]ReplayMVPPointView, len(state.folded))}
	for index, value := range state.folded {
		result.Folded[index] = ReplayMVPPointView{Start: value.start, Volume: value.volume, Close: value.close, Cumulative: value.cumulative}
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
			SessionVolume: replayFeatureAccountingView(value.features.sessionVolume), FromOpenPercent: replayFeatureAccountingView(value.features.fromOpenPercent),
			DayRange: replayFeatureAccountingView(value.features.dayRange), Activity30s: replayFeatureAccountingView(value.features.activity30s),
			Move30s:    replayFeatureAccountingView(value.features.move30s),
			DayPercent: replayFeatureAccountingView(value.features.dayPercent), From4AMPercent: replayFeatureAccountingView(value.features.from4AMPercent),
			HODDrawdown: replayFeatureAccountingView(value.features.hodDrawdown), SessionRange: replayFeatureAccountingView(value.features.sessionRange),
			Rolling30: replayFeatureAccountingView(value.features.rolling30), Rolling60: replayFeatureAccountingView(value.features.rolling60),
			Activity: replayFeatureAccountingView(value.features.activity),
		},
		Floats:      ReplayFloatAccountingView{Current: value.floats.current, Stale: value.floats.stale, Unavailable: value.floats.unavailable, Invalid: value.floats.invalid},
		Uncertainty: ReplayUncertaintyView{BootstrapOrigin: value.uncertainty.bootstrapOrigin, PostBootstrapGap: value.uncertainty.postBootstrapGap, LocalInvalid: value.uncertainty.localInvalid},
		PopulationTransition: ReplayPopulationTransitionDiagnosticView{
			BootstrapUnknown: value.populationTransition.bootstrapUnknown, TrustedByLaterLiveMark: value.populationTransition.trustedByLaterLiveMark,
			NoLaterEligibleMark: value.populationTransition.noLaterEligibleMark, LatestMarkNotLiveAuthority: value.populationTransition.latestMarkNotLiveAuthority,
			NoStrictlyOlderLocalizedConflict: value.populationTransition.noStrictlyOlderLocalizedConflict, ConflictAtOrAfterMark: value.populationTransition.conflictAtOrAfterMark,
			InvalidAtOrAfterMark: value.populationTransition.invalidAtOrAfterMark, IncompletePostMarkCoverage: value.populationTransition.incompletePostMarkCoverage,
		},
		TotalPassers: value.totalPassers, KnownRankableCount: value.knownRankableCount,
		DayInvalidRankable: value.dayInvalidRankable, QualifiedDayInvalid: value.qualifiedDayInvalid,
		TQIntentAvailable: value.tqIntentAvailable,
		Rows:              make([]ReplayRankingRowView, len(value.rows)),
	}
	for index, row := range value.rows {
		result.Rows[index] = ReplayRankingRowView{Rank: row.rank, Symbol: row.symbol, Last: row.last, DayPercent: row.dayPercent,
			MarkAge: row.markAge, Float: replayFloatFieldView(row.float), SessionVolume: replayFieldView(row.sessionVolume),
			FromOpenPercent: replayFieldView(row.fromOpenPercent), DayRange: replayFieldView(row.dayRange),
			Activity30s: replayFieldView(row.activity30s), Move30s: replayFieldView(row.move30s),
			TQIntentEligible: row.tqIntentEligible}
	}
	return result
}

func replayFloatFieldView(field aggregateFloatField) ReplayFloatFieldView {
	var percent *float64
	if field.fact.FreeFloatPercent != nil {
		value := *field.fact.FreeFloatPercent
		percent = &value
	}
	return ReplayFloatFieldView{Status: string(field.status), Reason: field.reason, Value: field.fact.FreeFloat,
		Percent: percent, Provider: field.fact.Provider, EffectiveDate: field.fact.EffectiveDate,
		RetrievedAt: field.fact.RetrievedAt, Provenance: string(field.fact.Provenance)}
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
