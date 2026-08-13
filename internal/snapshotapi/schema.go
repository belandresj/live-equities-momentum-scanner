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
	Replay        *Replay     `json:"replay,omitempty"`
}

type Replay struct {
	Phase            string       `json:"phase"`
	ArtifactID       string       `json:"artifact_id"`
	ArtifactEnd      string       `json:"artifact_end"`
	ObservationStart string       `json:"observation_start"`
	ObservationEnd   string       `json:"observation_end"`
	LogicalTime      string       `json:"logical_time"`
	Completion       string       `json:"completion"`
	ScheduleLagMS    *uint64      `json:"schedule_lag_ms"`
	Source           ReplaySource `json:"source"`
	Window           ReplayWindow `json:"window"`
}

type ReplaySource struct {
	ArtifactRecords                     string `json:"artifact_records"`
	CompletedRecordDispositions         string `json:"completed_record_dispositions"`
	IntentionallyUnappliedSuffixRecords string `json:"intentionally_unapplied_suffix_records"`
	UnreadRecords                       string `json:"unread_records"`
	PlannedGroups                       string `json:"planned_groups"`
	CompletedGroups                     string `json:"completed_groups"`
	ActiveGroup                         string `json:"active_group"`
	RemainingGroups                     string `json:"remaining_groups"`
	CompletedRuns                       string `json:"completed_runs"`
	FailedRuns                          string `json:"failed_runs"`
	CanceledRuns                        string `json:"canceled_runs"`
}

type ReplayWindow struct {
	WarmupGroupsPlanned            string `json:"warmup_groups_planned"`
	WarmupGroupsCompleted          string `json:"warmup_groups_completed"`
	WarmupGroupActive              string `json:"warmup_group_active"`
	WarmupGroupsRemaining          string `json:"warmup_groups_remaining"`
	ObservationSecondsPlanned      string `json:"observation_seconds_planned"`
	ObservationSecondsCompleted    string `json:"observation_seconds_completed"`
	ObservationSecondActive        string `json:"observation_second_active"`
	ObservationSecondsRemaining    string `json:"observation_seconds_remaining"`
	ObservationBoundariesPublished string `json:"observation_boundaries_published"`
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

type PopulationTransitionDiagnostic struct {
	BootstrapUnknown                 uint64 `json:"bootstrap_unknown"`
	TrustedByLaterLiveMark           uint64 `json:"trusted_by_later_live_mark"`
	NoLaterEligibleMark              uint64 `json:"no_later_eligible_mark"`
	LatestMarkNotLiveAuthority       uint64 `json:"latest_mark_not_live_authority"`
	NoStrictlyOlderLocalizedConflict uint64 `json:"no_strictly_older_localized_conflict"`
	ConflictAtOrAfterMark            uint64 `json:"conflict_at_or_after_mark"`
	InvalidAtOrAfterMark             uint64 `json:"invalid_at_or_after_mark"`
	IncompletePostMarkCoverage       uint64 `json:"incomplete_post_mark_coverage"`
}

type Accounting struct {
	Population                     PopulationAccounting           `json:"population"`
	Qualification                  QualificationAccounting        `json:"qualification"`
	Uncertainty                    UncertaintyAccounting          `json:"uncertainty"`
	PopulationTransitionDiagnostic PopulationTransitionDiagnostic `json:"population_transition_diagnostic"`
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
	Installed             bool    `json:"installed"`
	Eligible              string  `json:"eligible"`
	PressureDeferred      string  `json:"pressure_deferred"`
	ProjectionStarted     string  `json:"projection_started"`
	ProjectionInProgress  string  `json:"projection_in_progress"`
	Projected             string  `json:"projected"`
	ProjectionRejected    string  `json:"projection_rejected"`
	SubmitRejected        string  `json:"submit_rejected"`
	Submitted             string  `json:"submitted"`
	Outstanding           string  `json:"outstanding"`
	InProgress            string  `json:"in_progress"`
	Pending               string  `json:"pending"`
	Completed             string  `json:"completed"`
	Failed                string  `json:"failed"`
	Canceled              string  `json:"canceled"`
	Superseded            string  `json:"superseded"`
	LastAttemptedT0       *string `json:"last_attempted_t0,omitempty"`
	LastProjectedT0       *string `json:"last_projected_t0,omitempty"`
	LastSubmittedT0       *string `json:"last_submitted_t0,omitempty"`
	LastSuccessfulT0      *string `json:"last_successful_t0,omitempty"`
	UsableAgeMS           uint64  `json:"usable_age_ms"`
	ProjectionTotalMS     uint64  `json:"projection_total_ms"`
	ProjectionLockMS      uint64  `json:"projection_lock_ms"`
	ArtifactBytes         string  `json:"artifact_bytes"`
	WriteMS               uint64  `json:"write_ms"`
	EncodeMS              uint64  `json:"encode_ms"`
	ReopenValidationMS    uint64  `json:"reopen_validation_ms"`
	LastFailureStep       string  `json:"last_failure_step,omitempty"`
	LastProjectionFailure string  `json:"last_projection_failure,omitempty"`
	LastSubmitFailure     string  `json:"last_submit_failure,omitempty"`
}

type Operations struct {
	SampleAccountingValid         bool              `json:"sample_accounting_valid"`
	QueueCapacityFrames           uint64            `json:"queue_capacity_frames"`
	QueueCurrentFrames            uint64            `json:"queue_current_frames"`
	QueueHighFrames               uint64            `json:"queue_high_frames"`
	QueueCurrentBytes             uint64            `json:"queue_current_bytes"`
	QueueHighBytes                uint64            `json:"queue_high_bytes"`
	Deliveries                    string            `json:"deliveries"`
	ConsumerDeferred              string            `json:"consumer_deferred"`
	MeanProcessingDelayMS         uint64            `json:"mean_processing_delay_ms"`
	MaxProcessingDelayMS          uint64            `json:"max_processing_delay_ms"`
	MaxProcessingDelayOneSecondMS uint64            `json:"max_processing_delay_one_second_ms"`
	HeapAllocBytes                string            `json:"heap_alloc_bytes"`
	HeapInUseBytes                string            `json:"heap_in_use_bytes"`
	Goroutines                    uint64            `json:"goroutines"`
	ConnectionRecoveryAttempts    string            `json:"connection_recovery_attempts"`
	IntegrityFailure              *IntegrityFailure `json:"integrity_failure,omitempty"`
}

type IntegrityFailure struct {
	Category       string  `json:"category"`
	EngineSequence string  `json:"engine_sequence"`
	CandidateTime  *string `json:"candidate_time"`
	ExpectedTime   *string `json:"expected_time"`
	FirstSymbol    string  `json:"first_symbol,omitempty"`
	FirstField     string  `json:"first_field,omitempty"`
	FirstReason    string  `json:"first_reason,omitempty"`
}
