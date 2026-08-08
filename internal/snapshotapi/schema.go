package snapshotapi

const SchemaVersion = "scanner.snapshot.v1"

type Snapshot struct {
	SchemaVersion string      `json:"schema_version"`
	Sample        Sample      `json:"sample"`
	Publication   Publication `json:"publication"`
	Status        Status      `json:"status"`
	Ranking       Ranking     `json:"ranking"`
	Rows          []Row       `json:"rows"`
	Accounting    Accounting  `json:"accounting"`
	Recovery      Recovery    `json:"recovery"`
	TQ            TQ          `json:"tq"`
	Checkpoint    Checkpoint  `json:"checkpoint"`
	Operations    Operations  `json:"operations"`
}

type Sample struct {
	ID        string `json:"id"`
	SampledAt string `json:"sampled_at"`
}

type Position struct {
	ConnectionEpoch string `json:"connection_epoch"`
	FrameSequence   string `json:"frame_sequence"`
	ArrayIndex      uint64 `json:"array_index"`
}

type HydrationFence struct {
	Reconciled           bool    `json:"reconciled"`
	ConnectionEpoch      string  `json:"connection_epoch"`
	ThroughFrameSequence string  `json:"through_frame_sequence"`
	MarkerOrdinal        string  `json:"marker_ordinal"`
	SupportedThrough     *string `json:"supported_through"`
}

type Publication struct {
	ID                    string         `json:"id"`
	BindingIdentity       string         `json:"binding_identity"`
	TradingDate           string         `json:"trading_date"`
	RunMode               string         `json:"run_mode"`
	Lifecycle             string         `json:"lifecycle"`
	LifecycleReason       string         `json:"lifecycle_reason"`
	Suppression           string         `json:"suppression"`
	GeneratedAt           string         `json:"generated_at"`
	CommittedT            *string        `json:"committed_t"`
	LastEngineSequence    string         `json:"last_engine_sequence"`
	ConnectionEpoch       string         `json:"connection_epoch"`
	ConnectionActive      bool           `json:"connection_active"`
	AggregateAcknowledged bool           `json:"aggregate_acknowledged"`
	AggregateAckPosition  Position       `json:"aggregate_ack_position"`
	HydrationFence        HydrationFence `json:"hydration_fence"`
}

type Status struct {
	ProcessLive     bool   `json:"process_live"`
	BackendReady    bool   `json:"backend_ready"`
	ReadinessReason string `json:"readiness_reason"`
	RankingCurrent  bool   `json:"ranking_current"`
	CausalTarget    string `json:"causal_target"`
	WatermarkLagMS  *int64 `json:"watermark_lag_ms"`
	AccountingValid bool   `json:"accounting_valid"`
	TQPressureMode  string `json:"tq_pressure_mode"`
	TQShed          bool   `json:"tq_shed"`
}

type Ranking struct {
	Mode                string `json:"mode"`
	Reason              string `json:"reason"`
	TotalPassers        uint64 `json:"total_passers"`
	KnownRankableCount  uint64 `json:"known_rankable_count"`
	DayInvalidRankable  uint64 `json:"day_invalid_rankable"`
	QualifiedDayInvalid uint64 `json:"qualified_day_invalid"`
}

type RatioMeasurement struct {
	Status     string   `json:"status"`
	Reason     string   `json:"reason"`
	ValueRatio *float64 `json:"value_ratio"`
}

type RateMeasurement struct {
	Status          string   `json:"status"`
	Reason          string   `json:"reason"`
	TradesPerSecond *float64 `json:"trades_per_second"`
}

type TapeRate struct {
	Status                   string          `json:"status"`
	Reason                   string          `json:"reason"`
	TradeCoverage            bool            `json:"trade_coverage"`
	OneSecond                RateMeasurement `json:"one_second"`
	FiveSecond               RateMeasurement `json:"five_second"`
	TimestampBasis           string          `json:"timestamp_basis"`
	LifecycleRecordsObserved bool            `json:"lifecycle_records_observed"`
}

type Spread struct {
	Status          string   `json:"status"`
	Reason          string   `json:"reason"`
	QuoteCoverage   bool     `json:"quote_coverage"`
	Cents           *float64 `json:"cents"`
	BasisPoints     *float64 `json:"basis_points"`
	ValidDurationMS uint64   `json:"valid_duration_ms"`
	Quality         string   `json:"quality"`
}

type TQMembership struct {
	Desired                   bool `json:"desired"`
	ProviderPresent           bool `json:"provider_present"`
	ProviderMembershipUnknown bool `json:"provider_membership_unknown"`
}

type Row struct {
	Rank             uint64           `json:"rank"`
	Symbol           string           `json:"symbol"`
	LastUSD          float64          `json:"last_usd"`
	DayChangeRatio   float64          `json:"day_change_ratio"`
	MarkAgeMS        uint64           `json:"mark_age_ms"`
	From4AMChange    RatioMeasurement `json:"from_4am_change"`
	HODDrawdown      RatioMeasurement `json:"hod_drawdown"`
	DayRangePosition RatioMeasurement `json:"day_range_position"`
	Range30MPosition RatioMeasurement `json:"range_30m_position"`
	Range60MPosition RatioMeasurement `json:"range_60m_position"`
	Activity         RatioMeasurement `json:"activity"`
	TapeRate         TapeRate         `json:"tape_rate"`
	Spread           Spread           `json:"spread"`
	TQMembership     TQMembership     `json:"tq_membership"`
}

type PopulationAccounting struct {
	UniverseTotal              uint64 `json:"universe_total"`
	ValidPriorClose            uint64 `json:"valid_prior_close"`
	InvalidOrMissingPriorClose uint64 `json:"invalid_or_missing_prior_close"`
	TrustedRankableMark        uint64 `json:"trusted_rankable_mark"`
	TrustedBelowPriceMark      uint64 `json:"trusted_below_price_mark"`
	NoPrintThroughT            uint64 `json:"no_print_through_t"`
	InvalidMark                uint64 `json:"invalid_mark"`
	UnknownDueFailureOrFence   uint64 `json:"unknown_due_failure_or_fence"`
	CoveredPopulation          uint64 `json:"covered_population"`
	UnresolvedPopulation       uint64 `json:"unresolved_population"`
}

type QualificationAccounting struct {
	NotYetPassed uint64 `json:"not_yet_passed"`
	Provisional  uint64 `json:"provisional"`
	Finalized    uint64 `json:"finalized"`
	Unresolved   uint64 `json:"unresolved"`
}

type UncertaintyAccounting struct {
	BootstrapOrigin  uint64 `json:"bootstrap_origin"`
	PostBootstrapGap uint64 `json:"post_bootstrap_gap"`
	LocalInvalid     uint64 `json:"local_invalid"`
}

type Accounting struct {
	Population    PopulationAccounting    `json:"population"`
	Qualification QualificationAccounting `json:"qualification"`
	Uncertainty   UncertaintyAccounting   `json:"uncertainty"`
}

type RecoveryWork struct {
	Planned        string `json:"planned"`
	Open           string `json:"open"`
	CompletedValue string `json:"completed_value"`
	CompletedEmpty string `json:"completed_empty"`
	Failed         string `json:"failed"`
	Canceled       string `json:"canceled"`
	Fenced         string `json:"fenced"`
}

type RecoveryRows struct {
	Consumed             string `json:"consumed"`
	Inserted             string `json:"inserted"`
	Duplicate            string `json:"duplicate"`
	ConflictOrWithdrawal string `json:"conflict_or_withdrawal"`
	Rejected             string `json:"rejected"`
	Fenced               string `json:"fenced"`
	Integrity            string `json:"integrity"`
}

type Recovery struct {
	Purpose          string       `json:"purpose"`
	Generation       string       `json:"generation"`
	Start            *string      `json:"start"`
	End              *string      `json:"end"`
	SupportedThrough *string      `json:"supported_through"`
	FenceReconciled  bool         `json:"fence_reconciled"`
	PolicyWaiting    bool         `json:"policy_waiting"`
	Work             RecoveryWork `json:"work"`
	Rows             RecoveryRows `json:"rows"`
}

type TQFacts struct {
	Consumed     string `json:"consumed"`
	Applied      string `json:"applied"`
	Duplicate    string `json:"duplicate"`
	Rejected     string `json:"rejected"`
	Fenced       string `json:"fenced"`
	PressureShed string `json:"pressure_shed"`
	Integrity    string `json:"integrity"`
}

type TQCommands struct {
	Issued       string `json:"issued"`
	Pending      string `json:"pending"`
	Acknowledged string `json:"acknowledged"`
	Failed       string `json:"failed"`
	Fenced       string `json:"fenced"`
	ResultFenced string `json:"result_fenced"`
}

type TQ struct {
	DesiredSymbols       []string   `json:"desired_symbols"`
	PressureMode         string     `json:"pressure_mode"`
	AggregateOnly        bool       `json:"aggregate_only"`
	Shed                 bool       `json:"shed"`
	RetainedBoundHit     bool       `json:"retained_bound_hit"`
	PressureMisses       uint64     `json:"pressure_misses"`
	PressureTransitions  string     `json:"pressure_transitions"`
	PressureFenced       string     `json:"pressure_fenced"`
	KnownPresent         uint64     `json:"known_present"`
	KnownAbsent          uint64     `json:"known_absent"`
	Unknown              uint64     `json:"unknown"`
	RetainedTrades       uint64     `json:"retained_trades"`
	RetainedQuotes       uint64     `json:"retained_quotes"`
	RetainedFingerprints uint64     `json:"retained_fingerprints"`
	Facts                TQFacts    `json:"facts"`
	Commands             TQCommands `json:"commands"`
}

type Checkpoint struct {
	Installed  bool   `json:"installed"`
	Submitted  string `json:"submitted"`
	InProgress string `json:"in_progress"`
	Pending    string `json:"pending"`
	Completed  string `json:"completed"`
	Failed     string `json:"failed"`
	Canceled   string `json:"canceled"`
	Superseded string `json:"superseded"`
}

type Operations struct {
	SampleAccountingValid         bool   `json:"sample_accounting_valid"`
	QueueCapacityFrames           uint64 `json:"queue_capacity_frames"`
	QueueCurrentFrames            uint64 `json:"queue_current_frames"`
	QueueHighFrames               uint64 `json:"queue_high_frames"`
	QueueCurrentBytes             uint64 `json:"queue_current_bytes"`
	QueueHighBytes                uint64 `json:"queue_high_bytes"`
	Deliveries                    string `json:"deliveries"`
	ConsumerDeferred              string `json:"consumer_deferred"`
	MeanProcessingDelayMS         uint64 `json:"mean_processing_delay_ms"`
	MaxProcessingDelayMS          uint64 `json:"max_processing_delay_ms"`
	MaxProcessingDelayOneSecondMS uint64 `json:"max_processing_delay_one_second_ms"`
	HeapAllocBytes                string `json:"heap_alloc_bytes"`
	HeapInUseBytes                string `json:"heap_in_use_bytes"`
	Goroutines                    uint64 `json:"goroutines"`
	ConnectionRecoveryAttempts    string `json:"connection_recovery_attempts"`
}
