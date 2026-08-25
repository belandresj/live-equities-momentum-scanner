package engine

import "time"

type FieldView struct {
	Status, Reason string
	Value          float64
}

type PopulationView struct {
	UniverseTotal, ValidPriorClose, InvalidOrMissingPriorClose uint64
	TrustedRankableMark, TrustedBelowPriceMark                 uint64
	NoPrintThroughT, InvalidMark, UnknownDueFailureOrFence     uint64
	CoveredPopulation, UnresolvedPopulation                    uint64
}

type QualificationAccountingView struct {
	NotYetPassed, Provisional, Finalized, Unresolved uint64
}

type FeatureAccountingView struct {
	Statuses [4]uint64
	Reasons  [featureReasonBucketCount]uint64
	Pairs    [4][featureReasonBucketCount]uint64
}

type AllFeatureAccountingView struct {
	SessionVolume, FromOpenPercent, DayRange FeatureAccountingView
	Activity30s, Move30s                     FeatureAccountingView
	DayPercent                               FeatureAccountingView
}

type FloatAccountingView struct {
	Current, Stale, Unavailable, Invalid uint64
}

type UncertaintyView struct {
	BootstrapOrigin, PostBootstrapGap, LocalInvalid uint64
}

type PopulationTransitionDiagnosticView struct {
	BootstrapUnknown, TrustedByLaterLiveMark, NoLaterEligibleMark uint64
	LatestMarkNotLiveAuthority, NoStrictlyOlderLocalizedConflict  uint64
	ConflictAtOrAfterMark, InvalidAtOrAfterMark                   uint64
	IncompletePostMarkCoverage                                    uint64
}

type RankingRowView struct {
	Rank                                     uint32
	Symbol                                   string
	Last, DayPercent                         float64
	MarkAge                                  time.Duration
	Float                                    FloatFieldView
	SessionVolume, FromOpenPercent, DayRange FieldView
	Activity30s, Move30s                     FieldView
	TQIntentEligible                         bool
}

type FloatFieldView struct {
	Status, Reason          string
	Value                   float64
	Percent                 *float64
	Provider, EffectiveDate string
	RetrievedAt             time.Time
	Provenance              string
}

type EvaluationView struct {
	At                                      time.Time
	Mode, Reason                            string
	Population                              PopulationView
	Qualification                           QualificationAccountingView
	Features                                AllFeatureAccountingView
	Floats                                  FloatAccountingView
	Uncertainty                             UncertaintyView
	PopulationTransition                    PopulationTransitionDiagnosticView
	TotalPassers, KnownRankableCount        uint64
	DayInvalidRankable, QualifiedDayInvalid uint64
	TQIntentAvailable                       bool
	Rows                                    []RankingRowView
}

type PublicationView struct {
	SchemaVersion       string
	PublicationID       uint64
	BindingIdentity     string
	TradingDate         string
	RunMode             string
	Lifecycle           string
	LifecycleReason     string
	Suppression         SuppressionDisposition
	LastDisposition     DispositionCode
	DispositionReason   DispositionReason
	LastEngineSequence  uint64
	Watermark           *time.Time
	GeneratedAt         time.Time
	CurrentMarketClaim  bool
	AggregateEvaluation EvaluationView
}

// ValidateEvaluationAccounting validates only the fixed-cardinality
// accounting projection. Ranking rows are intentionally excluded so incident
// diagnostics can retain coherent market accounting without retaining symbol
// identities.
func ValidateEvaluationAccounting(r EvaluationView) bool {
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
	for _, dimension := range []FeatureAccountingView{r.Features.DayPercent, r.Features.SessionVolume, r.Features.FromOpenPercent, r.Features.DayRange, r.Features.Activity30s, r.Features.Move30s} {
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

func livePublicationView(publication *privatePublication) PublicationView {
	if publication == nil {
		return PublicationView{}
	}
	return PublicationView{
		SchemaVersion: publication.schemaVersion, PublicationID: publication.publicationID,
		BindingIdentity: publication.bindingIdentity, TradingDate: publication.tradingDate,
		RunMode: publication.mode, Lifecycle: string(publication.lifecycle), LifecycleReason: string(publication.lifecycleReason),
		Suppression: publication.suppressionDisposition, LastDisposition: publication.lastDisposition,
		DispositionReason: publication.dispositionReason, LastEngineSequence: publication.lastEngineSequence,
		Watermark: immutableTimePointer(publication.watermark), GeneratedAt: publication.generatedAt,
		CurrentMarketClaim:  publication.currentMarketClaim,
		AggregateEvaluation: evaluationView(publication.aggregateEvaluation),
	}
}

func fieldView(field aggregateFeatureField) FieldView {
	return FieldView{Status: string(field.status), Reason: string(field.reason), Value: field.value}
}

func evaluationView(value aggregateEvaluationResult) EvaluationView {
	result := EvaluationView{
		At: value.at, Mode: string(value.mode), Reason: string(value.reason),
		Population: PopulationView{
			UniverseTotal: value.population.universeTotal, ValidPriorClose: value.population.validPriorClose,
			InvalidOrMissingPriorClose: value.population.invalidOrMissingPriorClose, TrustedRankableMark: value.population.trustedRankableMark,
			TrustedBelowPriceMark: value.population.trustedBelowPriceMark, NoPrintThroughT: value.population.noPrintThroughT,
			InvalidMark: value.population.invalidMark, UnknownDueFailureOrFence: value.population.unknownDueFailureOrFence,
			CoveredPopulation: value.population.coveredPopulation, UnresolvedPopulation: value.population.unresolvedPopulation,
		},
		Qualification: QualificationAccountingView{NotYetPassed: value.qualification.notYetPassed, Provisional: value.qualification.provisional,
			Finalized: value.qualification.finalized, Unresolved: value.qualification.unresolved},
		Features: AllFeatureAccountingView{
			SessionVolume: featureAccountingView(value.features.sessionVolume), FromOpenPercent: featureAccountingView(value.features.fromOpenPercent),
			DayRange: featureAccountingView(value.features.dayRange), Activity30s: featureAccountingView(value.features.activity30s),
			Move30s:    featureAccountingView(value.features.move30s),
			DayPercent: featureAccountingView(value.features.dayPercent),
		},
		Floats:      FloatAccountingView{Current: value.floats.current, Stale: value.floats.stale, Unavailable: value.floats.unavailable, Invalid: value.floats.invalid},
		Uncertainty: UncertaintyView{BootstrapOrigin: value.uncertainty.bootstrapOrigin, PostBootstrapGap: value.uncertainty.postBootstrapGap, LocalInvalid: value.uncertainty.localInvalid},
		PopulationTransition: PopulationTransitionDiagnosticView{
			BootstrapUnknown: value.populationTransition.bootstrapUnknown, TrustedByLaterLiveMark: value.populationTransition.trustedByLaterLiveMark,
			NoLaterEligibleMark: value.populationTransition.noLaterEligibleMark, LatestMarkNotLiveAuthority: value.populationTransition.latestMarkNotLiveAuthority,
			NoStrictlyOlderLocalizedConflict: value.populationTransition.noStrictlyOlderLocalizedConflict, ConflictAtOrAfterMark: value.populationTransition.conflictAtOrAfterMark,
			InvalidAtOrAfterMark: value.populationTransition.invalidAtOrAfterMark, IncompletePostMarkCoverage: value.populationTransition.incompletePostMarkCoverage,
		},
		TotalPassers: value.totalPassers, KnownRankableCount: value.knownRankableCount,
		DayInvalidRankable: value.dayInvalidRankable, QualifiedDayInvalid: value.qualifiedDayInvalid,
		TQIntentAvailable: value.tqIntentAvailable,
		Rows:              make([]RankingRowView, len(value.rows)),
	}
	for index, row := range value.rows {
		result.Rows[index] = RankingRowView{Rank: row.rank, Symbol: row.symbol, Last: row.last, DayPercent: row.dayPercent,
			MarkAge: row.markAge, Float: floatFieldView(row.float), SessionVolume: fieldView(row.sessionVolume),
			FromOpenPercent: fieldView(row.fromOpenPercent), DayRange: fieldView(row.dayRange),
			Activity30s: fieldView(row.activity30s), Move30s: fieldView(row.move30s),
			TQIntentEligible: row.tqIntentEligible}
	}
	return result
}

func floatFieldView(field aggregateFloatField) FloatFieldView {
	var percent *float64
	if field.fact.FreeFloatPercent != nil {
		value := *field.fact.FreeFloatPercent
		percent = &value
	}
	return FloatFieldView{Status: string(field.status), Reason: field.reason, Value: field.fact.FreeFloat,
		Percent: percent, Provider: field.fact.Provider, EffectiveDate: field.fact.EffectiveDate,
		RetrievedAt: field.fact.RetrievedAt, Provenance: string(field.fact.Provenance)}
}

func featureAccountingView(value featureDimensionAccounting) FeatureAccountingView {
	return FeatureAccountingView{Statuses: value.statuses, Reasons: value.reasons, Pairs: value.pairs}
}

func uncertaintyOriginView(value uncertaintyOrigin) string {
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
