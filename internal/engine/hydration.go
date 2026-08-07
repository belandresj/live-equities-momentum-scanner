package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	HydrationPlanSchemaV1         = "engine-hydration-plan-v1"
	HydrationChunkSchemaV1        = "engine-hydration-chunk-v1"
	HydrationTerminalSchemaV1     = "engine-hydration-terminal-v1"
	AggregateIngressFenceSchemaV1 = "engine-aggregate-ingress-fence-v1"
	HydrationPolicyActionSchemaV1 = "engine-hydration-policy-action-v1"
	hydrationMaximumWorkers       = 8
	hydrationMaximumRows          = sessionSeconds
)

const (
	DispositionHydrationPlanApplied          DispositionCode = "hydration_plan_applied"
	DispositionHydrationChunkApplied         DispositionCode = "hydration_chunk_applied"
	DispositionHydrationTerminalApplied      DispositionCode = "hydration_terminal_applied"
	DispositionHydrationRejected             DispositionCode = "hydration_rejected"
	DispositionHydrationFenced               DispositionCode = "hydration_fenced"
	DispositionHydrationIntegrity            DispositionCode = "hydration_integrity_failure"
	DispositionAggregateIngressFenceApplied  DispositionCode = "aggregate_ingress_fence_applied"
	DispositionAggregateIngressFenceRejected DispositionCode = "aggregate_ingress_fence_rejected"
	DispositionAggregateIngressFenceFenced   DispositionCode = "aggregate_ingress_fence_fenced"
	DispositionHydrationPolicyApplied        DispositionCode = "hydration_policy_applied"
	DispositionHydrationPolicyRejected       DispositionCode = "hydration_policy_rejected"
)

const (
	ReasonHydrationGeneration        DispositionReason = "hydration_generation"
	ReasonHydrationInterval          DispositionReason = "hydration_interval"
	ReasonHydrationBounds            DispositionReason = "hydration_bounds"
	ReasonHydrationCompactedPresence DispositionReason = "hydration_compacted_presence"
	ReasonHydrationToken             DispositionReason = "hydration_token"
	ReasonHydrationChunkSequence     DispositionReason = "hydration_chunk_sequence"
	ReasonHydrationTerminalSequence  DispositionReason = "hydration_terminal_sequence"
	ReasonAggregateIngressFence      DispositionReason = "aggregate_ingress_fence"
)

type HydrationPurpose string

const (
	HydrationFreshBootstrap    HydrationPurpose = "fresh_bootstrap"
	HydrationCheckpointCatchup HydrationPurpose = "checkpoint_catchup"
	HydrationGapRecovery       HydrationPurpose = "gap_recovery"
)

type HydrationPlanBudgets struct {
	Workers                  int
	RowsPerChunk             int
	MaximumResponseBytes     int64
	MaximumNormalizedRecords int64
	MaximumResidentRecords   int64
}

// HydrationPlanInput requests one engine-owned plan. R and the interval start
// are never caller-selected: the owner derives them from its accepted
// acknowledgement and mode-specific retained state.
type HydrationPlanInput struct {
	SchemaVersion   string
	BindingIdentity string
	Purpose         HydrationPurpose
	ConnectionEpoch uint64
	Budgets         HydrationPlanBudgets
}

// HydrationRequestToken is an immutable engine allocation. Its private fields
// prevent a worker or coordinator from changing plan identity.
type HydrationRequestToken struct {
	bindingID       string
	generation      uint64
	requestID       uint64
	purpose         HydrationPurpose
	symbol          string
	start, end      time.Time
	connectionEpoch uint64
	requestToken    string
}

func (t HydrationRequestToken) BindingIdentity() string   { return t.bindingID }
func (t HydrationRequestToken) Generation() uint64        { return t.generation }
func (t HydrationRequestToken) RequestID() uint64         { return t.requestID }
func (t HydrationRequestToken) ResultID() uint64          { return t.requestID }
func (t HydrationRequestToken) Purpose() HydrationPurpose { return t.purpose }
func (t HydrationRequestToken) Symbol() string            { return t.symbol }
func (t HydrationRequestToken) Start() time.Time          { return t.start }
func (t HydrationRequestToken) End() time.Time            { return t.end }
func (t HydrationRequestToken) ConnectionEpoch() uint64   { return t.connectionEpoch }

type HydrationPlanResult struct {
	generation uint64
	purpose    HydrationPurpose
	start, end time.Time
	budgets    HydrationPlanBudgets
	requests   []HydrationRequestToken
	empty      bool
	fence      HydrationFenceCommand
}

func (r HydrationPlanResult) Generation() uint64                { return r.generation }
func (r HydrationPlanResult) Purpose() HydrationPurpose         { return r.purpose }
func (r HydrationPlanResult) Start() time.Time                  { return r.start }
func (r HydrationPlanResult) End() time.Time                    { return r.end }
func (r HydrationPlanResult) Budgets() HydrationPlanBudgets     { return r.budgets }
func (r HydrationPlanResult) Requests() []HydrationRequestToken { return slices.Clone(r.requests) }
func (r HydrationPlanResult) Empty() bool                       { return r.empty }
func (r HydrationPlanResult) FenceCommand() (HydrationFenceCommand, bool) {
	return r.fence, r.fence.commandToken != 0
}

// HydrationFenceCommand is allocated only by the engine after the generation's
// work ledger has reached terminal accounting. Its fields are intentionally
// private so the adapter can preserve, but never select, its identity.
type HydrationFenceCommand struct {
	bindingID                       string
	generation, epoch, commandToken uint64
}

func (c HydrationFenceCommand) BindingIdentity() string { return c.bindingID }
func (c HydrationFenceCommand) Generation() uint64      { return c.generation }
func (c HydrationFenceCommand) ConnectionEpoch() uint64 { return c.epoch }
func (c HydrationFenceCommand) CommandToken() uint64    { return c.commandToken }

type AggregateIngressFenceState string

const (
	AggregateIngressFenceComplete AggregateIngressFenceState = "complete"
	AggregateIngressFenceFailed   AggregateIngressFenceState = "failed"
	AggregateIngressFenceCanceled AggregateIngressFenceState = "canceled"
)

// AggregateIngressFenceInput is the closed adapter-to-engine fact. The raw
// frame boundary and marker ordinal are separate domains by construction.
type AggregateIngressFenceInput struct {
	schemaVersion                       string
	command                             HydrationFenceCommand
	state                               AggregateIngressFenceState
	throughFrameSequence, markerOrdinal uint64
	capturedAt                          time.Time
}

func NewAggregateIngressFenceInput(command HydrationFenceCommand, state AggregateIngressFenceState, throughFrameSequence, markerOrdinal uint64, capturedAt time.Time) (AggregateIngressFenceInput, error) {
	if !validHydrationFenceCommand(command) || (state != AggregateIngressFenceComplete && state != AggregateIngressFenceFailed && state != AggregateIngressFenceCanceled) ||
		markerOrdinal == 0 || capturedAt.IsZero() || capturedAt != capturedAt.UTC() {
		return AggregateIngressFenceInput{}, errors.New("invalid aggregate ingress fence")
	}
	return AggregateIngressFenceInput{schemaVersion: AggregateIngressFenceSchemaV1, command: command, state: state,
		throughFrameSequence: throughFrameSequence, markerOrdinal: markerOrdinal, capturedAt: capturedAt}, nil
}

func (f AggregateIngressFenceInput) Command() HydrationFenceCommand    { return f.command }
func (f AggregateIngressFenceInput) State() AggregateIngressFenceState { return f.state }
func (f AggregateIngressFenceInput) ThroughFrameSequence() uint64      { return f.throughFrameSequence }
func (f AggregateIngressFenceInput) MarkerOrdinal() uint64             { return f.markerOrdinal }
func (f AggregateIngressFenceInput) CapturedAt() time.Time             { return f.capturedAt }

type HydrationPolicyAction string

const (
	HydrationPolicyRetry   HydrationPolicyAction = "retry"
	HydrationPolicyExhaust HydrationPolicyAction = "exhaust"
	HydrationPolicyCancel  HydrationPolicyAction = "cancel"
	HydrationPolicyStop    HydrationPolicyAction = "stop"
)

type HydrationPolicyActionInput struct {
	schemaVersion                  string
	bindingID                      string
	generation, epoch, actionToken uint64
	action                         HydrationPolicyAction
}

func NewHydrationPolicyActionInput(command HydrationFenceCommand, action HydrationPolicyAction, actionToken uint64) (HydrationPolicyActionInput, error) {
	if !validHydrationFenceCommand(command) || !validHydrationPolicyAction(action) || actionToken == 0 {
		return HydrationPolicyActionInput{}, errors.New("invalid hydration policy action")
	}
	return HydrationPolicyActionInput{schemaVersion: HydrationPolicyActionSchemaV1, bindingID: command.bindingID,
		generation: command.generation, epoch: command.epoch, actionToken: actionToken, action: action}, nil
}

func (p HydrationPolicyActionInput) Action() HydrationPolicyAction { return p.action }
func (p HydrationPolicyActionInput) ActionToken() uint64           { return p.actionToken }

type HydrationRow struct {
	symbol                 string
	windowStart, windowEnd time.Time
	values                 AggregateValues
}

func NewHydrationRow(symbol string, start, end time.Time, values AggregateValues) (HydrationRow, error) {
	if symbol == "" || len(symbol) > maximumSymbolBytes || start != start.UTC() || end != end.UTC() ||
		start.Nanosecond() != 0 || end.Nanosecond() != 0 || end.Sub(start) != time.Second || !validAggregateValues(values) {
		return HydrationRow{}, errors.New("invalid hydration row")
	}
	return HydrationRow{symbol: symbol, windowStart: start, windowEnd: end, values: values}, nil
}

func (r HydrationRow) Symbol() string          { return r.symbol }
func (r HydrationRow) WindowStart() time.Time  { return r.windowStart }
func (r HydrationRow) WindowEnd() time.Time    { return r.windowEnd }
func (r HydrationRow) Values() AggregateValues { return r.values }

type HydrationChunkInput struct {
	schemaVersion string
	token         HydrationRequestToken
	resultID      uint64
	ordinal       int
	totalChunks   int
	rowOffset     int64
	totalRows     int64
	rows          []HydrationRow
}

func NewHydrationChunkInput(token HydrationRequestToken, resultID uint64, ordinal, totalChunks int, rowOffset, totalRows int64, rows []HydrationRow) (HydrationChunkInput, error) {
	if !validHydrationTokenShape(token) || resultID == 0 || ordinal < 0 || totalChunks <= 0 || ordinal >= totalChunks ||
		rowOffset < 0 || totalRows <= 0 || len(rows) == 0 || int64(len(rows)) > totalRows || rowOffset > totalRows-int64(len(rows)) {
		return HydrationChunkInput{}, errors.New("invalid hydration chunk")
	}
	return HydrationChunkInput{
		schemaVersion: HydrationChunkSchemaV1, token: token, resultID: resultID,
		ordinal: ordinal, totalChunks: totalChunks, rowOffset: rowOffset, totalRows: totalRows,
		rows: slices.Clone(rows),
	}, nil
}

func (c HydrationChunkInput) Token() HydrationRequestToken { return c.token }
func (c HydrationChunkInput) ResultID() uint64             { return c.resultID }
func (c HydrationChunkInput) Ordinal() int                 { return c.ordinal }
func (c HydrationChunkInput) TotalChunks() int             { return c.totalChunks }
func (c HydrationChunkInput) RowOffset() int64             { return c.rowOffset }
func (c HydrationChunkInput) TotalRows() int64             { return c.totalRows }
func (c HydrationChunkInput) Rows() []HydrationRow         { return slices.Clone(c.rows) }

type HydrationTerminalState string

const (
	HydrationCompletedValue HydrationTerminalState = "completed_value"
	HydrationCompletedEmpty HydrationTerminalState = "completed_empty"
	HydrationFailed         HydrationTerminalState = "failed"
	HydrationCanceled       HydrationTerminalState = "canceled"
)

type HydrationProviderReason string

const (
	HydrationReasonNone                   HydrationProviderReason = ""
	HydrationReasonRequestConstruction    HydrationProviderReason = "request_construction"
	HydrationReasonTransportDeadline      HydrationProviderReason = "transport_deadline"
	HydrationReasonHTTPRetryExhausted     HydrationProviderReason = "http_retry_exhausted"
	HydrationReasonRedirectContinuation   HydrationProviderReason = "redirect_continuation"
	HydrationReasonResponseSizeSyntax     HydrationProviderReason = "response_size_syntax"
	HydrationReasonEnvelopeIdentityStatus HydrationProviderReason = "envelope_identity_status"
	HydrationReasonTimestamp              HydrationProviderReason = "timestamp"
	HydrationReasonNumericCount           HydrationProviderReason = "numeric_count"
	HydrationReasonStructuralAggregate    HydrationProviderReason = "structural_aggregate"
	HydrationReasonSymbolIntervalOrder    HydrationProviderReason = "symbol_interval_order_duplicate"
	HydrationReasonPlanBudget             HydrationProviderReason = "plan_budget"
	HydrationReasonCanceled               HydrationProviderReason = "canceled"
)

type HydrationTerminalInput struct {
	schemaVersion                  string
	token                          HydrationRequestToken
	resultID                       uint64
	state                          HydrationTerminalState
	reason                         HydrationProviderReason
	pages, attempts, responseBytes int64
	normalizedRows                 int64
	emittedChunks, emittedRows     int64
}

func NewHydrationTerminalInput(token HydrationRequestToken, resultID uint64, state HydrationTerminalState, reason HydrationProviderReason,
	pages, attempts, responseBytes, normalizedRows, emittedChunks, emittedRows int64) (HydrationTerminalInput, error) {
	if !validHydrationTokenShape(token) || resultID == 0 || !validHydrationTerminalState(state) || !validHydrationProviderReason(reason) ||
		pages < 0 || attempts < 0 || responseBytes < 0 || normalizedRows < 0 || emittedChunks < 0 || emittedRows < 0 {
		return HydrationTerminalInput{}, errors.New("invalid hydration terminal")
	}
	return HydrationTerminalInput{
		schemaVersion: HydrationTerminalSchemaV1, token: token, resultID: resultID, state: state, reason: reason,
		pages: pages, attempts: attempts, responseBytes: responseBytes, normalizedRows: normalizedRows,
		emittedChunks: emittedChunks, emittedRows: emittedRows,
	}, nil
}

func (t HydrationTerminalInput) Token() HydrationRequestToken    { return t.token }
func (t HydrationTerminalInput) ResultID() uint64                { return t.resultID }
func (t HydrationTerminalInput) State() HydrationTerminalState   { return t.state }
func (t HydrationTerminalInput) Reason() HydrationProviderReason { return t.reason }

type HydrationRowAccounting struct {
	Consumed, Inserted, Duplicate, ConflictOrWithdrawal, Rejected, Fenced, Integrity uint64
}

func (a HydrationRowAccounting) reconciles() bool {
	return a.Consumed == a.Inserted+a.Duplicate+a.ConflictOrWithdrawal+a.Rejected+a.Fenced+a.Integrity
}

type HydrationAccounting struct {
	Planned, Open, CompletedValue, CompletedEmpty, Failed, Canceled, Fenced uint64
}

func (a HydrationAccounting) terminal() uint64 {
	return a.CompletedValue + a.CompletedEmpty + a.Failed + a.Canceled + a.Fenced
}

func (a HydrationAccounting) reconciles() bool {
	return a.Planned == a.Open+a.terminal()
}

type hydrationCoverage uint8

const (
	hydrationCoveragePending hydrationCoverage = iota + 1
	hydrationCoverageCandidateComplete
	hydrationCoverageUnknown
)

type hydrationLedgerEntry struct {
	token           HydrationRequestToken
	nextChunk       int
	totalChunks     int
	totalRows       int64
	consumedRows    int64
	lastWindowStart time.Time
	rowAccounting   HydrationRowAccounting
	terminal        HydrationTerminalState
	terminalReason  HydrationProviderReason
	engineClosed    bool
	fenced          bool
	coverage        hydrationCoverage
}

type hydrationGenerationState struct {
	active         bool
	generation     uint64
	purpose        HydrationPurpose
	bindingID      string
	epoch          uint64
	start, end     time.Time
	budgets        HydrationPlanBudgets
	requests       []hydrationLedgerEntry
	requestIndex   map[uint64]int
	accounting     HydrationAccounting
	rowAccounting  HydrationRowAccounting
	responseBytes  int64
	normalizedRows int64
	fenceCommand   HydrationFenceCommand
	policyDecided  bool
}

type hydrationState struct {
	lastGeneration     uint64
	lastRequestID      uint64
	lastCommandToken   uint64
	checkpointT0       *time.Time
	supportedT         *time.Time
	generation         hydrationGenerationState
	revision           uint64
	fenceReconciled    bool
	fenceEpoch         uint64
	fenceThrough       uint64
	fenceMarkerOrdinal uint64
	supportedThrough   *time.Time
	lastPolicyToken    uint64
	policyAction       HydrationPolicyAction
	policyWaiting      bool
}

type HydrationDisposition struct {
	EngineSequence         uint64
	Code                   DispositionCode
	Reason                 DispositionReason
	SuppressionDisposition SuppressionDisposition
	Plan                   HydrationPlanResult
	Rows                   HydrationRowAccounting
	Accounting             HydrationAccounting
	FenceCommand           HydrationFenceCommand
}

type frozenHydrationPlanInput struct{ HydrationPlanInput }

type frozenHydrationChunkInput struct {
	schemaVersion string
	token         HydrationRequestToken
	resultID      uint64
	ordinal       int
	totalChunks   int
	rowOffset     int64
	totalRows     int64
	rows          []HydrationRow
}

type frozenHydrationTerminalInput struct{ HydrationTerminalInput }
type frozenAggregateIngressFenceInput struct{ AggregateIngressFenceInput }
type frozenHydrationPolicyActionInput struct{ HydrationPolicyActionInput }

// frozenHydrationCancelInput is a package-private proof seam for the ordered
// cancellation/supersession consequence. Later lifecycle slices call the same
// cancelHydrationGenerationLocked primitive from their real typed transition.
type frozenHydrationCancelInput struct {
	bindingID  string
	generation uint64
	fenced     bool
}

func (e *Engine) AdmitHydrationPlan(ctx context.Context, input HydrationPlanInput) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || !boundedHydrationPlanInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputHydrationPlan, hydrationPlan: frozenHydrationPlanInput{input}}
	return e.admitHydration(ctx, node)
}

func (e *Engine) AdmitHydrationChunk(ctx context.Context, input HydrationChunkInput) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || !boundedHydrationChunkInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputHydrationChunk, hydrationChunk: freezeHydrationChunk(input)}
	return e.admitHydration(ctx, node)
}

func (e *Engine) AdmitHydrationTerminal(ctx context.Context, input HydrationTerminalInput) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || !boundedHydrationTerminalInput(input) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputHydrationTerminal, hydrationTerminal: frozenHydrationTerminalInput{input}}
	return e.admitHydration(ctx, node)
}

func (e *Engine) AdmitAggregateIngressFence(ctx context.Context, input AggregateIngressFenceInput) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || input.schemaVersion != AggregateIngressFenceSchemaV1 || !validHydrationFenceCommand(input.command) ||
		(input.state != AggregateIngressFenceComplete && input.state != AggregateIngressFenceFailed && input.state != AggregateIngressFenceCanceled) ||
		input.markerOrdinal == 0 || input.capturedAt.IsZero() || input.capturedAt != input.capturedAt.UTC() {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admitHydration(ctx, &queueNode{kind: inputAggregateIngressFence, aggregateIngressFence: frozenAggregateIngressFenceInput{input}})
}

func (e *Engine) AdmitHydrationPolicyAction(ctx context.Context, input HydrationPolicyActionInput) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || input.schemaVersion != HydrationPolicyActionSchemaV1 || !validIdentityShape(input.bindingID) ||
		input.generation == 0 || input.epoch == 0 || input.actionToken == 0 || !validHydrationPolicyAction(input.action) {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	node := &queueNode{kind: inputHydrationPolicyAction, hydrationPolicy: frozenHydrationPolicyActionInput{input}}
	return e.admitHydration(ctx, node)
}

func (e *Engine) admitHydration(ctx context.Context, node *queueNode) (AdmissionResult, <-chan HydrationDisposition) {
	result := e.admitNode(ctx, node, false)
	if result != AdmissionAdmitted {
		return result, nil
	}
	return result, node.hydrationCompletion
}

func hydrationInputKind(kind inputKind) bool {
	return kind == inputHydrationPlan || kind == inputHydrationChunk || kind == inputHydrationTerminal || kind == inputHydrationCancelProof || kind == inputAggregateIngressFence || kind == inputHydrationPolicyAction
}

func validHydrationPolicyAction(action HydrationPolicyAction) bool {
	return action == HydrationPolicyRetry || action == HydrationPolicyExhaust || action == HydrationPolicyCancel || action == HydrationPolicyStop
}

func validHydrationFenceCommand(command HydrationFenceCommand) bool {
	return validIdentityShape(command.bindingID) && command.generation > 0 && command.epoch > 0 && command.commandToken > 0
}

func (e *Engine) admitHydrationCancelForProof(ctx context.Context, bindingID string, generation uint64, fenced bool) (AdmissionResult, <-chan HydrationDisposition) {
	e.beginAdmission()
	if ctx == nil || !validIdentityShape(bindingID) || generation == 0 {
		return e.finishNonAdmission(AdmissionNotAdmittedInvalid), nil
	}
	return e.admitHydration(ctx, &queueNode{kind: inputHydrationCancelProof, hydrationCancel: frozenHydrationCancelInput{
		bindingID: bindingID, generation: generation, fenced: fenced,
	}})
}

func boundedHydrationPlanInput(input HydrationPlanInput) bool {
	return len(input.SchemaVersion) <= 64 && len(input.BindingIdentity) <= maximumContextBytes &&
		validHydrationPurpose(input.Purpose) && input.ConnectionEpoch > 0 && validHydrationBudgets(input.Budgets)
}

func boundedHydrationChunkInput(input HydrationChunkInput) bool {
	return input.schemaVersion == HydrationChunkSchemaV1 && validHydrationTokenShape(input.token) && input.resultID > 0 &&
		input.ordinal >= 0 && input.totalChunks > 0 && input.ordinal < input.totalChunks && input.rowOffset >= 0 && input.totalRows > 0 &&
		len(input.rows) > 0 && len(input.rows) <= hydrationMaximumRows && int64(len(input.rows)) <= input.totalRows &&
		input.rowOffset <= input.totalRows-int64(len(input.rows))
}

func boundedHydrationTerminalInput(input HydrationTerminalInput) bool {
	return input.schemaVersion == HydrationTerminalSchemaV1 && validHydrationTokenShape(input.token) && input.resultID > 0 &&
		validHydrationTerminalState(input.state) && validHydrationProviderReason(input.reason) &&
		input.pages >= 0 && input.attempts >= 0 && input.responseBytes >= 0 && input.normalizedRows >= 0 &&
		input.emittedChunks >= 0 && input.emittedRows >= 0
}

func freezeHydrationChunk(input HydrationChunkInput) frozenHydrationChunkInput {
	return frozenHydrationChunkInput{
		schemaVersion: input.schemaVersion, token: input.token, resultID: input.resultID,
		ordinal: input.ordinal, totalChunks: input.totalChunks, rowOffset: input.rowOffset,
		totalRows: input.totalRows, rows: slices.Clone(input.rows),
	}
}

func validHydrationPurpose(purpose HydrationPurpose) bool {
	return purpose == HydrationFreshBootstrap || purpose == HydrationCheckpointCatchup || purpose == HydrationGapRecovery
}

func validHydrationBudgets(b HydrationPlanBudgets) bool {
	return b.Workers >= 1 && b.Workers <= hydrationMaximumWorkers && b.RowsPerChunk >= 1 && b.RowsPerChunk <= hydrationMaximumRows &&
		b.MaximumResponseBytes > 0 && b.MaximumNormalizedRecords > 0 && b.MaximumResidentRecords > 0
}

func validHydrationTokenShape(token HydrationRequestToken) bool {
	return validIdentityShape(token.bindingID) && token.generation > 0 && token.requestID > 0 && validHydrationPurpose(token.purpose) &&
		token.symbol != "" && len(token.symbol) <= maximumSymbolBytes && token.start == token.start.UTC() && token.end == token.end.UTC() &&
		token.start.Nanosecond() == 0 && token.end.Nanosecond() == 0 && token.start.Before(token.end) && token.connectionEpoch > 0 &&
		token.requestToken == hydrationRequestToken(token.generation, token.requestID)
}

func validHydrationTerminalState(state HydrationTerminalState) bool {
	return state == HydrationCompletedValue || state == HydrationCompletedEmpty || state == HydrationFailed || state == HydrationCanceled
}

func validHydrationProviderReason(reason HydrationProviderReason) bool {
	switch reason {
	case HydrationReasonNone, HydrationReasonRequestConstruction, HydrationReasonTransportDeadline, HydrationReasonHTTPRetryExhausted,
		HydrationReasonRedirectContinuation, HydrationReasonResponseSizeSyntax, HydrationReasonEnvelopeIdentityStatus,
		HydrationReasonTimestamp, HydrationReasonNumericCount, HydrationReasonStructuralAggregate,
		HydrationReasonSymbolIntervalOrder, HydrationReasonPlanBudget, HydrationReasonCanceled:
		return true
	default:
		return false
	}
}

func hydrationRequestToken(generation, requestID uint64) string {
	return fmt.Sprintf("c6:%d:%d", generation, requestID)
}

func (e *Engine) applyHydrationPlanLocked(node *queueNode) (DispositionCode, DispositionReason, HydrationPlanResult) {
	input := node.hydrationPlan.HydrationPlanInput
	if input.SchemaVersion != HydrationPlanSchemaV1 {
		return DispositionHydrationRejected, ReasonSchema, HydrationPlanResult{}
	}
	if e.mode != RunModeLive || e.state.binding == nil {
		return DispositionHydrationRejected, ReasonLifecycle, HydrationPlanResult{}
	}
	if input.BindingIdentity != e.state.binding.identity {
		return DispositionHydrationFenced, ReasonBinding, HydrationPlanResult{}
	}
	if !e.state.liveEpochActive || input.ConnectionEpoch != e.state.liveEpoch {
		return DispositionHydrationFenced, ReasonStaleLiveEpoch, HydrationPlanResult{}
	}
	if !e.state.aggregateAcknowledged {
		return DispositionHydrationRejected, ReasonLifecycle, HydrationPlanResult{}
	}
	if e.state.hydration.generation.active {
		return DispositionHydrationRejected, ReasonHydrationGeneration, HydrationPlanResult{}
	}
	start, ok := e.hydrationStartLocked(input.Purpose)
	if !ok {
		return DispositionHydrationRejected, ReasonLifecycle, HydrationPlanResult{}
	}
	end := hydrationAcknowledgementBoundary(e.state.aggregateAckReceivedAt, e.state.binding.sessionStart, e.state.binding.sessionEnd)
	if start != start.UTC() || start.Nanosecond() != 0 || start.Before(e.state.binding.sessionStart) || start.After(e.state.binding.sessionEnd) || start.After(end) {
		return DispositionHydrationRejected, ReasonHydrationInterval, HydrationPlanResult{}
	}
	validPopulation := make([]int, 0, e.state.binding.priorAccounting.ValidPriorClose)
	for index := range e.state.binding.symbols {
		if e.state.binding.symbols[index].prior.status == reference.PriorCloseValid {
			validPopulation = append(validPopulation, index)
		}
	}
	seconds := int64(end.Sub(start) / time.Second)
	if seconds < 0 || (seconds > 0 && int64(len(validPopulation)) > math.MaxInt64/seconds) {
		return DispositionHydrationRejected, ReasonHydrationBounds, HydrationPlanResult{}
	}
	possibleRows := int64(len(validPopulation)) * seconds
	concurrentRows := int64(min(input.Budgets.Workers, len(validPopulation))) * seconds
	if possibleRows > input.Budgets.MaximumNormalizedRecords || concurrentRows > input.Budgets.MaximumResidentRecords {
		return DispositionHydrationRejected, ReasonHydrationBounds, HydrationPlanResult{}
	}
	if seconds > 0 {
		for _, index := range validPopulation {
			if !e.hydrationRegistrationAllowedLocked(index, start, end) {
				return DispositionHydrationRejected, ReasonHydrationCompactedPresence, HydrationPlanResult{}
			}
		}
	}
	if e.state.hydration.lastGeneration == math.MaxUint64 || uint64(len(validPopulation)) > math.MaxUint64-e.state.hydration.lastRequestID {
		return DispositionHydrationRejected, ReasonHydrationBounds, HydrationPlanResult{}
	}
	generation := e.state.hydration.lastGeneration + 1
	requestCount := len(validPopulation)
	if seconds == 0 {
		requestCount = 0
	}
	entries := make([]hydrationLedgerEntry, requestCount)
	requestIndex := make(map[uint64]int, requestCount)
	requests := make([]HydrationRequestToken, requestCount)
	for requestOffset := 0; requestOffset < requestCount; requestOffset++ {
		requestID := e.state.hydration.lastRequestID + uint64(requestOffset) + 1
		symbol := e.state.binding.symbols[validPopulation[requestOffset]].symbol
		token := HydrationRequestToken{
			bindingID: e.state.binding.identity, generation: generation, requestID: requestID,
			purpose: input.Purpose, symbol: symbol, start: start, end: end,
			connectionEpoch: e.state.liveEpoch, requestToken: hydrationRequestToken(generation, requestID),
		}
		entries[requestOffset] = hydrationLedgerEntry{token: token, coverage: hydrationCoveragePending}
		requestIndex[requestID] = requestOffset
		requests[requestOffset] = token
	}
	candidate := hydrationGenerationState{
		active: true, generation: generation, purpose: input.Purpose, bindingID: e.state.binding.identity,
		epoch: e.state.liveEpoch, start: start, end: end, budgets: input.Budgets,
		requests: entries, requestIndex: requestIndex,
		accounting: HydrationAccounting{Planned: uint64(requestCount), Open: uint64(requestCount)},
	}
	if !candidate.accounting.reconciles() {
		return DispositionHydrationIntegrity, ReasonAccounting, HydrationPlanResult{}
	}
	e.state.hydration.lastGeneration = generation
	if requestCount > 0 {
		e.state.hydration.lastRequestID = requests[len(requests)-1].requestID
	}
	e.state.hydration.generation = candidate
	e.state.hydration.policyWaiting = false
	e.state.hydration.revision++
	plan := HydrationPlanResult{
		generation: generation, purpose: input.Purpose, start: start, end: end,
		budgets: input.Budgets, requests: requests, empty: requestCount == 0,
	}
	if requestCount == 0 {
		command, ok := e.allocateHydrationFenceCommandLocked(&e.state.hydration.generation)
		if !ok {
			e.state.hydration.generation = hydrationGenerationState{}
			return DispositionHydrationRejected, ReasonHydrationBounds, HydrationPlanResult{}
		}
		plan.fence = command
	}
	return DispositionHydrationPlanApplied, ReasonNone, plan
}

func (e *Engine) allocateHydrationFenceCommandLocked(generation *hydrationGenerationState) (HydrationFenceCommand, bool) {
	if generation == nil || !generation.active || generation.accounting.Open != 0 || !generation.accounting.reconciles() {
		return HydrationFenceCommand{}, false
	}
	if generation.fenceCommand.commandToken != 0 {
		return generation.fenceCommand, true
	}
	if e.state.hydration.lastCommandToken == math.MaxUint64 {
		return HydrationFenceCommand{}, false
	}
	e.state.hydration.lastCommandToken++
	generation.fenceCommand = HydrationFenceCommand{bindingID: generation.bindingID, generation: generation.generation,
		epoch: generation.epoch, commandToken: e.state.hydration.lastCommandToken}
	return generation.fenceCommand, true
}

func (e *Engine) hydrationStartLocked(purpose HydrationPurpose) (time.Time, bool) {
	switch purpose {
	case HydrationFreshBootstrap:
		return e.state.binding.sessionStart, e.state.lifecycle == lifecycleHydrating
	case HydrationCheckpointCatchup:
		if e.state.lifecycle != lifecycleHydrating || e.state.hydration.checkpointT0 == nil {
			return time.Time{}, false
		}
		return *e.state.hydration.checkpointT0, true
	case HydrationGapRecovery:
		if e.state.lifecycle != lifecycleRecovering || e.state.hydration.supportedT == nil {
			return time.Time{}, false
		}
		return *e.state.hydration.supportedT, true
	default:
		return time.Time{}, false
	}
}

func hydrationAcknowledgementBoundary(receivedAt, start, end time.Time) time.Time {
	boundary := receivedAt.Truncate(time.Second)
	if receivedAt.After(boundary) {
		boundary = boundary.Add(time.Second)
	}
	if boundary.Before(start) {
		return start
	}
	if boundary.After(end) {
		return end
	}
	return boundary
}

func (e *Engine) hydrationRegistrationAllowedLocked(symbolIndex int, start, end time.Time) bool {
	state := e.state.binding.symbols[symbolIndex].aggregates
	if state == nil || state.presence == nil {
		return true
	}
	for at := start; at.Before(end); at = at.Add(time.Second) {
		if state.presence.has(sessionSlot(e.state.binding, at)) && !aggregatePresentAt(state, at.Unix()) {
			return false
		}
	}
	return true
}

func (e *Engine) applyHydrationChunkLocked(node *queueNode) (DispositionCode, DispositionReason, HydrationRowAccounting) {
	input := node.hydrationChunk
	entry, code, reason := e.hydrationEntryLocked(input.token, input.resultID)
	if entry == nil {
		return code, reason, HydrationRowAccounting{}
	}
	if entry.engineClosed || entry.fenced {
		return DispositionHydrationFenced, ReasonHistoricalContext, HydrationRowAccounting{}
	}
	if entry.terminal != "" {
		return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence, HydrationRowAccounting{}
	}
	if input.ordinal != entry.nextChunk || input.rowOffset != entry.consumedRows || input.totalChunks <= 0 || input.totalRows <= 0 ||
		input.ordinal >= input.totalChunks || int64(len(input.rows)) > input.totalRows-input.rowOffset ||
		(entry.totalChunks != 0 && entry.totalChunks != input.totalChunks) || (entry.totalRows != 0 && entry.totalRows != input.totalRows) ||
		len(input.rows) > e.state.hydration.generation.budgets.RowsPerChunk || input.totalRows > int64(input.token.end.Sub(input.token.start)/time.Second) {
		return DispositionHydrationIntegrity, ReasonHydrationChunkSequence, HydrationRowAccounting{}
	}
	last := entry.lastWindowStart
	for _, row := range input.rows {
		if row.symbol != input.token.symbol || row.windowStart.Before(input.token.start) || !row.windowStart.Before(input.token.end) ||
			row.windowEnd.Sub(row.windowStart) != time.Second || row.windowStart != row.windowStart.UTC() || row.windowEnd != row.windowEnd.UTC() ||
			row.windowStart.Nanosecond() != 0 || row.windowEnd.Nanosecond() != 0 || !validAggregateValues(row.values) ||
			(!last.IsZero() && !last.Before(row.windowStart)) {
			return DispositionHydrationIntegrity, ReasonHydrationChunkSequence, HydrationRowAccounting{}
		}
		last = row.windowStart
	}
	wouldConsume := input.rowOffset + int64(len(input.rows))
	if (input.ordinal+1 == input.totalChunks && wouldConsume != input.totalRows) ||
		(input.ordinal+1 < input.totalChunks && wouldConsume >= input.totalRows) {
		return DispositionHydrationIntegrity, ReasonHydrationChunkSequence, HydrationRowAccounting{}
	}
	if entry.totalChunks == 0 {
		entry.totalChunks = input.totalChunks
		entry.totalRows = input.totalRows
	}
	delta := HydrationRowAccounting{}
	for offset, row := range input.rows {
		aggregate := frozenAggregateInput{AggregateInput: AggregateInput{
			SchemaVersion: AggregateSchemaV1, BindingIdentity: input.token.bindingID,
			Source: AggregateSourceHistorical, Symbol: row.symbol, WindowStart: row.windowStart, WindowEnd: row.windowEnd,
			Values: row.values, DeliveryTime: node.admissionTime,
			Historical: HistoricalPosition{Generation: input.token.generation, RequestToken: input.token.requestToken, RecordOrdinal: uint64(input.rowOffset) + uint64(offset) + 1},
		}, historicalProof: &frozenHistoricalProofContext{
			bindingID: input.token.bindingID, generation: input.token.generation, token: input.token.requestToken,
			symbol: input.token.symbol, intervalStart: input.token.start, intervalEnd: input.token.end,
			resultCardinality: int(input.totalRows), resultValid: true,
		}, c6Historical: true}
		rowCode, rowReason := e.applyAggregateLocked(aggregate, node.admissionTime)
		delta.Consumed++
		switch rowCode {
		case DispositionAggregateInserted, DispositionAggregateRevised:
			delta.Inserted++
			e.state.exposedRevision++
		case DispositionAggregateExactDuplicate:
			delta.Duplicate++
		case DispositionAggregateWithdrawn:
			delta.ConflictOrWithdrawal++
			e.state.exposedRevision++
			entry.coverage = hydrationCoverageUnknown
		case DispositionAggregateRejected:
			if rowReason == ReasonHistoricalLiveConflict || rowReason == ReasonHistoricalHistoricalConflict {
				delta.ConflictOrWithdrawal++
			} else {
				delta.Rejected++
			}
			entry.coverage = hydrationCoverageUnknown
		case DispositionAggregateFenced:
			delta.Fenced++
			entry.coverage = hydrationCoverageUnknown
		default:
			delta.Integrity++
			entry.coverage = hydrationCoverageUnknown
		}
		synthetic := &queueNode{kind: inputAggregate, aggregate: aggregate, admissionTime: node.admissionTime, engineSequence: node.engineSequence}
		candidate := e.runAggregateFeatureContributorLocked(synthetic, rowCode, rowReason)
		if !e.runAggregateEvaluatorLocked(synthetic, rowCode, rowReason, candidate) || rowCode == DispositionAggregateIntegrity {
			addHydrationRows(&entry.rowAccounting, delta)
			addHydrationRows(&e.state.hydration.generation.rowAccounting, delta)
			e.state.hydration.revision++
			return DispositionHydrationIntegrity, ReasonAccounting, delta
		}
	}
	entry.nextChunk++
	entry.consumedRows = wouldConsume
	entry.lastWindowStart = last
	addHydrationRows(&entry.rowAccounting, delta)
	addHydrationRows(&e.state.hydration.generation.rowAccounting, delta)
	if !entry.rowAccounting.reconciles() || !e.state.hydration.generation.rowAccounting.reconciles() {
		return DispositionHydrationIntegrity, ReasonAccounting, delta
	}
	e.state.hydration.revision++
	return DispositionHydrationChunkApplied, ReasonNone, delta
}

func (e *Engine) hydrationEntryLocked(token HydrationRequestToken, resultID uint64) (*hydrationLedgerEntry, DispositionCode, DispositionReason) {
	generation := &e.state.hydration.generation
	if !generation.active || token.bindingID != generation.bindingID || token.generation != generation.generation || token.connectionEpoch != generation.epoch {
		return nil, DispositionHydrationFenced, ReasonHistoricalContext
	}
	index, ok := generation.requestIndex[token.requestID]
	if !ok {
		return nil, DispositionHydrationIntegrity, ReasonHydrationToken
	}
	entry := &generation.requests[index]
	if resultID != token.requestID || token != entry.token {
		return nil, DispositionHydrationIntegrity, ReasonHydrationToken
	}
	return entry, "", ""
}

func (e *Engine) applyHydrationTerminalLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.hydrationTerminal.HydrationTerminalInput
	entry, code, reason := e.hydrationEntryLocked(input.token, input.resultID)
	if entry == nil {
		return code, reason
	}
	if entry.engineClosed || entry.fenced {
		return DispositionHydrationFenced, ReasonHistoricalContext
	}
	if entry.terminal != "" {
		if entry.terminal == input.state && entry.terminalReason == input.reason {
			return DispositionHydrationRejected, ReasonHydrationTerminalSequence
		}
		return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
	}
	generation := &e.state.hydration.generation
	intervalRows := int64(input.token.end.Sub(input.token.start) / time.Second)
	if input.pages > 2 || input.attempts > 6 || input.attempts < input.pages || (input.pages == 0 && input.attempts > 3) ||
		(input.attempts == 0 && input.responseBytes > 0) || input.normalizedRows > intervalRows ||
		input.responseBytes > generation.budgets.MaximumResponseBytes-generation.responseBytes ||
		input.normalizedRows > e.state.hydration.generation.budgets.MaximumNormalizedRecords ||
		input.normalizedRows > generation.budgets.MaximumNormalizedRecords-generation.normalizedRows ||
		input.emittedChunks != int64(entry.nextChunk) || input.emittedRows != entry.consumedRows {
		return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
	}
	if (input.normalizedRows > 0 && (input.pages == 0 || input.attempts == 0)) ||
		(input.emittedChunks == 0) != (input.emittedRows == 0) || input.emittedRows > input.normalizedRows ||
		(entry.nextChunk > 0 && input.normalizedRows != entry.totalRows) || (input.normalizedRows == 0 && entry.nextChunk > 0) {
		return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
	}
	switch input.state {
	case HydrationCompletedValue:
		if input.reason != HydrationReasonNone || input.pages == 0 || input.attempts == 0 || input.attempts > input.pages*3 || input.responseBytes == 0 ||
			entry.totalRows <= 0 || entry.nextChunk != entry.totalChunks ||
			entry.consumedRows != entry.totalRows || input.normalizedRows != entry.totalRows {
			return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
		}
	case HydrationCompletedEmpty:
		if input.reason != HydrationReasonNone || input.pages == 0 || input.attempts == 0 || input.attempts > input.pages*3 || input.responseBytes == 0 ||
			entry.nextChunk != 0 || input.normalizedRows != 0 || input.emittedChunks != 0 || input.emittedRows != 0 {
			return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
		}
	case HydrationFailed:
		if input.reason == HydrationReasonNone || input.reason == HydrationReasonCanceled || entry.nextChunk != 0 || input.normalizedRows != 0 || input.emittedChunks != 0 || input.emittedRows != 0 {
			return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
		}
	case HydrationCanceled:
		if input.reason != HydrationReasonCanceled || input.normalizedRows < entry.consumedRows {
			return DispositionHydrationIntegrity, ReasonHydrationTerminalSequence
		}
	}
	entry.terminal = input.state
	entry.terminalReason = input.reason
	generation.responseBytes += input.responseBytes
	generation.normalizedRows += input.normalizedRows
	if entry.coverage != hydrationCoverageUnknown {
		if input.state == HydrationCompletedValue || input.state == HydrationCompletedEmpty {
			entry.coverage = hydrationCoverageCandidateComplete
		} else {
			entry.coverage = hydrationCoverageUnknown
		}
	}
	accounting := &generation.accounting
	accounting.Open--
	switch input.state {
	case HydrationCompletedValue:
		accounting.CompletedValue++
	case HydrationCompletedEmpty:
		accounting.CompletedEmpty++
	case HydrationFailed:
		accounting.Failed++
	case HydrationCanceled:
		accounting.Canceled++
	}
	if !accounting.reconciles() {
		return DispositionHydrationIntegrity, ReasonAccounting
	}
	if accounting.Open == 0 {
		if _, ok := e.allocateHydrationFenceCommandLocked(generation); !ok {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
	}
	e.state.hydration.revision++
	return DispositionHydrationTerminalApplied, ReasonNone
}

func (e *Engine) applyAggregateIngressFenceLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.aggregateIngressFence.AggregateIngressFenceInput
	generation := &e.state.hydration.generation
	if !generation.active || input.command != generation.fenceCommand || input.command.bindingID != generation.bindingID ||
		input.command.generation != generation.generation || input.command.epoch != generation.epoch {
		return DispositionAggregateIngressFenceFenced, ReasonHistoricalContext
	}
	if generation.accounting.Open != 0 || !generation.accounting.reconciles() ||
		(e.state.hydration.fenceEpoch == generation.epoch && input.markerOrdinal <= e.state.hydration.fenceMarkerOrdinal) {
		return DispositionAggregateIngressFenceRejected, ReasonAggregateIngressFence
	}
	if input.state != AggregateIngressFenceComplete {
		if !e.cancelHydrationGenerationLocked(input.state == AggregateIngressFenceFailed) {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
		generation.active = false
		e.state.hydration.fenceReconciled = false
		if input.state == AggregateIngressFenceCanceled {
			e.state.liveEpochActive = false
			e.clearAggregateAcknowledgementLocked()
			if !e.transitionLifecycleLocked(lifecycleEventAggregateLoss, node, lifecycleReasonAggregateEpochLost) {
				return DispositionAggregateIngressFenceRejected, ReasonLifecycle
			}
			return DispositionAggregateIngressFenceRejected, ReasonAggregateIngressFence
		}
		e.enterSuppressionLocked(lifecycleEventIngressIntegrity, node, lifecycleReasonIngressIntegrity)
		return DispositionIngressIntegrity, ReasonAggregateIngressFence
	}
	if !e.state.liveEpochActive || !e.state.aggregateAcknowledged || e.state.liveEpoch != generation.epoch ||
		input.throughFrameSequence < e.state.aggregateAckPosition.FrameSequence ||
		(e.state.greatestIngressPosition.ConnectionEpoch == generation.epoch && input.throughFrameSequence < e.state.greatestIngressPosition.FrameSequence) ||
		input.capturedAt.Before(e.state.aggregateAckReceivedAt) || input.capturedAt.After(node.admissionTime) {
		return DispositionAggregateIngressFenceFenced, ReasonAggregateIngressFence
	}
	target := timerTarget(input.capturedAt, e.delay, e.state.binding.sessionStart, e.state.binding.sessionEnd)
	if target.Before(generation.end) {
		return DispositionAggregateIngressFenceRejected, ReasonAggregateIngressFence
	}
	if e.state.aggregateEvaluator.coverage == nil {
		e.state.aggregateEvaluator.coverage = make(map[int]aggregateCoverageConsequence, len(e.state.binding.symbols))
	}
	recoveryUnknown := false
	for index := range e.state.binding.symbols {
		symbol := &e.state.binding.symbols[index]
		if symbol.prior.status != reference.PriorCloseValid {
			continue
		}
		coverage := hydrationCoverageCandidateComplete
		if requestIndex, ok := generation.requestIndexBySymbol(symbol.symbol); ok && requestIndex >= 0 {
			coverage = generation.requests[requestIndex].coverage
		} else if !ok {
			coverage = hydrationCoverageUnknown
		}
		state := ensureAggregateState(symbol)
		if coverage != hydrationCoverageCandidateComplete || !installExactCoverage(state, e.state.binding, generation.start, target) ||
			!exactAggregateCoverage(state, e.state.binding, e.state.binding.sessionStart, target) {
			if generation.purpose == HydrationGapRecovery {
				e.state.aggregateEvaluator.coverage[index] = coverageUnknownPostBootstrap
				recoveryUnknown = true
			} else {
				e.state.aggregateEvaluator.coverage[index] = coverageUnknownFailureOrFence
			}
			continue
		}
		if _, hasMark := latestMarkBefore(state, target); hasMark {
			delete(e.state.aggregateEvaluator.coverage, index)
		} else {
			e.state.aggregateEvaluator.coverage[index] = coverageNoPrintThroughT
		}
	}
	if generation.purpose == HydrationGapRecovery && recoveryUnknown {
		// The real ingress boundary has completed, but provider failure means the
		// retained gap is still unresolved.  Keep the generation current and
		// publish an explicit policy-wait state; only an ordered Component 8 fact
		// may choose retry, exhaustion, cancellation, or stop.
		e.state.hydration.fenceReconciled = false
		e.state.hydration.fenceEpoch = generation.epoch
		e.state.hydration.fenceThrough = input.throughFrameSequence
		e.state.hydration.fenceMarkerOrdinal = input.markerOrdinal
		e.state.hydration.policyWaiting = true
		e.state.hydration.revision++
		return DispositionAggregateIngressFenceApplied, ReasonNone
	}
	generation.active = false
	e.state.hydration.fenceReconciled = true
	e.state.hydration.fenceEpoch = generation.epoch
	e.state.hydration.fenceThrough = input.throughFrameSequence
	e.state.hydration.fenceMarkerOrdinal = input.markerOrdinal
	e.state.hydration.supportedThrough = immutableTime(target)
	e.state.latestTarget = immutableTime(target)
	if !e.transitionLifecycleLocked(lifecycleEventHydrationComplete, node, lifecycleReasonHydrationComplete) {
		return DispositionAggregateIngressFenceRejected, ReasonLifecycle
	}
	e.state.connectionControl.recoveryAttempts = 0
	e.state.hydration.revision++
	return DispositionAggregateIngressFenceApplied, ReasonNone
}

func (e *Engine) applyHydrationPolicyActionLocked(node *queueNode) (DispositionCode, DispositionReason) {
	input := node.hydrationPolicy.HydrationPolicyActionInput
	generation := &e.state.hydration.generation
	if e.mode != RunModeLive || e.state.binding == nil || e.state.lifecycle != lifecycleRecovering || !generation.active || generation.policyDecided ||
		input.bindingID != e.state.binding.identity || input.epoch != generation.epoch || input.generation != generation.generation {
		return DispositionHydrationPolicyRejected, ReasonHistoricalContext
	}
	if input.actionToken <= e.state.hydration.lastPolicyToken {
		return DispositionHydrationPolicyRejected, ReasonHydrationToken
	}
	failedAttempt := generation.accounting.Open == 0 && generation.accounting.Failed+generation.accounting.Canceled+generation.accounting.Fenced > 0
	if (input.action == HydrationPolicyRetry || input.action == HydrationPolicyExhaust) && !failedAttempt {
		return DispositionHydrationPolicyRejected, ReasonHydrationTerminalSequence
	}
	e.state.hydration.lastPolicyToken = input.actionToken
	e.state.hydration.policyAction = input.action
	generation.policyDecided = true
	e.state.hydration.policyWaiting = false
	switch input.action {
	case HydrationPolicyRetry:
		if !e.cancelHydrationGenerationLocked(false) {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
		generation.active = false
	case HydrationPolicyCancel:
		if !e.cancelHydrationGenerationLocked(false) {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
		generation.active = false
		e.state.liveEpochActive = false
		e.clearAggregateAcknowledgementLocked()
		e.state.hydration.policyWaiting = true
	case HydrationPolicyExhaust:
		if !e.cancelHydrationGenerationLocked(true) {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
		generation.active = false
		e.enterSuppressionLocked(lifecycleEventIngressIntegrity, node, lifecycleReasonRecoveryExhausted)
		e.state.hydration.revision++
		return DispositionIngressIntegrity, ReasonAggregateIngressFence
	case HydrationPolicyStop:
		if !e.cancelHydrationGenerationLocked(false) {
			return DispositionHydrationIntegrity, ReasonAccounting
		}
		generation.active = false
		if !e.transitionLifecycleLocked(lifecycleEventStop, node, lifecycleReasonControlledStop) {
			return DispositionHydrationPolicyRejected, ReasonLifecycle
		}
		e.sealed = true
		e.broadcastLocked()
	default:
		return DispositionHydrationPolicyRejected, ReasonHydrationBounds
	}
	e.state.hydration.revision++
	return DispositionHydrationPolicyApplied, ReasonNone
}

func (g *hydrationGenerationState) requestIndexBySymbol(symbol string) (int, bool) {
	for index := range g.requests {
		if g.requests[index].token.symbol == symbol {
			return index, true
		}
	}
	// Empty intervals have no provider request and are complete by construction.
	if g.start.Equal(g.end) {
		return -1, true
	}
	return -1, false
}

func (e *Engine) cancelHydrationGenerationLocked(fenced bool) bool {
	generation := &e.state.hydration.generation
	if !generation.active {
		return true
	}
	changed := false
	for index := range generation.requests {
		entry := &generation.requests[index]
		if entry.terminal != "" || entry.engineClosed || entry.fenced {
			continue
		}
		generation.accounting.Open--
		entry.coverage = hydrationCoverageUnknown
		if fenced {
			entry.fenced = true
			entry.engineClosed = true
			generation.accounting.Fenced++
		} else {
			entry.terminal = HydrationCanceled
			entry.terminalReason = HydrationReasonCanceled
			entry.engineClosed = true
			generation.accounting.Canceled++
		}
		changed = true
	}
	if changed {
		e.state.hydration.revision++
	}
	return generation.accounting.reconciles()
}

func (e *Engine) applyHydrationCancelProofLocked(node *queueNode) (DispositionCode, DispositionReason) {
	generation := &e.state.hydration.generation
	input := node.hydrationCancel
	if !generation.active || input.bindingID != generation.bindingID || input.generation != generation.generation {
		return DispositionHydrationFenced, ReasonHistoricalContext
	}
	open := generation.accounting.Open
	if !e.cancelHydrationGenerationLocked(input.fenced) {
		return DispositionHydrationIntegrity, ReasonAccounting
	}
	if open == 0 {
		return DispositionHydrationRejected, ReasonHydrationTerminalSequence
	}
	return DispositionHydrationTerminalApplied, ReasonNone
}

func addHydrationRows(target *HydrationRowAccounting, delta HydrationRowAccounting) {
	target.Consumed += delta.Consumed
	target.Inserted += delta.Inserted
	target.Duplicate += delta.Duplicate
	target.ConflictOrWithdrawal += delta.ConflictOrWithdrawal
	target.Rejected += delta.Rejected
	target.Fenced += delta.Fenced
	target.Integrity += delta.Integrity
}

func (e *Engine) hydrationPinsIdentityLocked(symbol string, start time.Time) bool {
	generation := &e.state.hydration.generation
	if !generation.active || start.Before(generation.start) || !start.Before(generation.end) {
		return false
	}
	for index := range generation.requests {
		if generation.requests[index].token.symbol == symbol {
			return true
		}
	}
	return false
}

type hydrationObservation struct {
	Active     bool
	Generation uint64
	Purpose    HydrationPurpose
	Start, End time.Time
	Accounting HydrationAccounting
	Rows       HydrationRowAccounting
	Requests   []hydrationLedgerEntry
}

func (e *Engine) observeHydration() hydrationObservation {
	e.mu.Lock()
	defer e.mu.Unlock()
	generation := e.state.hydration.generation
	return hydrationObservation{
		Active: generation.active, Generation: generation.generation, Purpose: generation.purpose,
		Start: generation.start, End: generation.end, Accounting: generation.accounting,
		Rows: generation.rowAccounting, Requests: slices.Clone(generation.requests),
	}
}
