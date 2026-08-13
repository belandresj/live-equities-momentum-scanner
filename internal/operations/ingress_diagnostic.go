package operations

import (
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
)

type IngressIncidentOwner string

const (
	IngressOwnerAdapterTerminal   IngressIncidentOwner = "adapter_terminal"
	IngressOwnerRuntimeAccounting IngressIncidentOwner = "runtime_accounting_guard"
	IngressOwnerEngineTransition  IngressIncidentOwner = "engine_integrity_transition"
)

const (
	maximumIngressIdentityResults = 16
	maximumIngressHistorySamples  = 60
)

type IngressIdentityResult struct {
	Name        string
	Reconciled  bool
	Left, Right uint64
}

type IngressDiagnosticSample struct {
	CapturedAt                                                          time.Time
	FramesRead, FramesAdmitted, FramesDispositioned                     uint64
	FramesFenced, FramesRejectedSlot, FramesRejectedByte                uint64
	QueuedFrames, ClassifyingFrames, HighFrames                         uint64
	QueuedBytes, HighBytes, CapacityFrames, CapacityBytes               int
	OldestWaitingFrameAge, ActiveDeliveryAge, MaxDeliveryDelayOneSecond time.Duration
	ActiveDeliveryKind                                                  massive.DeliveryKind
	Deliveries                                                          uint64
	Lifecycle                                                           string
	HydrationPlanned, HydrationOpen                                     uint64
}

type LastCoherentMarketProjection struct {
	PublicationID, LastEngineSequence uint64
	BindingIdentity, TradingDate      string
	RunMode                           engine.RunMode
	Lifecycle, LifecycleReason        string
	Watermark                         *time.Time
	GeneratedAt                       time.Time
	CurrentMarketClaim                bool
	RankingRows                       uint64
	Evaluation                        engine.ReplayEvaluationView
}

// IngressIncident is one fixed-cardinality, redacted, immutable diagnostic.
// It preserves the owning component's exact first cause; it is not a second
// lifecycle or readiness decision.
type IngressIncident struct {
	Owner                         IngressIncidentOwner
	Source                        string
	Reason                        string
	Binding                       string
	Epoch                         uint64
	Position                      engine.LivePosition
	PositionApplicable            bool
	ArrayIndexApplicable          bool
	Lifecycle                     string
	LifecycleReason               string
	Suppression                   engine.SuppressionDisposition
	Hydration                     engine.OperationalHydration
	Queue                         massive.LiveQueueAccounting
	QueueHighFrames               uint64
	QueueHighBytes                int
	IncomingFrameBytes            int
	ActiveDeliveryKind            massive.DeliveryKind
	ActiveDeliveryStartedAt       time.Time
	ActiveDeliveryAgeAtCause      time.Duration
	Adapter                       massive.AdapterAccounting
	PriorEngine                   engine.OperationalView
	Engine                        engine.OperationalView
	FenceTiming                   engine.FenceTimingView
	LastCoherentProjection        *LastCoherentMarketProjection
	LastCoherentProjectionInvalid bool
	Identities                    [maximumIngressIdentityResults]IngressIdentityResult
	IdentityCount                 int
	History                       [maximumIngressHistorySamples]IngressDiagnosticSample
	HistoryCount                  int
	CapturedAt                    time.Time
	EngineCapturedAt              time.Time
}

type ingressDiagnosticHistory struct {
	mu      sync.Mutex
	samples [maximumIngressHistorySamples]IngressDiagnosticSample
	next    int
	count   int
}

func (h *ingressDiagnosticHistory) add(sample IngressDiagnosticSample) {
	h.mu.Lock()
	h.samples[h.next] = sample
	h.next = (h.next + 1) % len(h.samples)
	if h.count < len(h.samples) {
		h.count++
	}
	h.mu.Unlock()
}

func (h *ingressDiagnosticHistory) snapshot() ([maximumIngressHistorySamples]IngressDiagnosticSample, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var result [maximumIngressHistorySamples]IngressDiagnosticSample
	start := (h.next - h.count + len(h.samples)) % len(h.samples)
	for index := 0; index < h.count; index++ {
		result[index] = h.samples[(start+index)%len(h.samples)]
	}
	return result, h.count
}

type ingressIncidentLatch struct {
	mu    sync.Mutex
	value *IngressIncident
}

func (l *ingressIncidentLatch) set(value IngressIncident) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.value != nil {
		return
	}
	copyValue := value
	l.value = &copyValue
}

func (l *ingressIncidentLatch) get() *IngressIncident {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.value == nil {
		return nil
	}
	copyValue := *l.value
	copyValue.Engine.Watermark = cloneTime(l.value.Engine.Watermark)
	copyValue.Engine.Hydration.SupportedThrough = cloneTime(l.value.Engine.Hydration.SupportedThrough)
	copyValue.Engine.IntegrityFailure = cloneIntegrityFailure(l.value.Engine.IntegrityFailure)
	copyValue.PriorEngine.Watermark = cloneTime(l.value.PriorEngine.Watermark)
	copyValue.PriorEngine.Hydration.SupportedThrough = cloneTime(l.value.PriorEngine.Hydration.SupportedThrough)
	copyValue.PriorEngine.IntegrityFailure = cloneIntegrityFailure(l.value.PriorEngine.IntegrityFailure)
	copyValue.LastCoherentProjection = cloneLastCoherentProjection(l.value.LastCoherentProjection)
	return &copyValue
}

func lastCoherentProjection(publication engine.ReplayPublicationView) (*LastCoherentMarketProjection, bool) {
	if publication.PublicationID == 0 || publication.Watermark == nil || publication.AggregateEvaluation.At.IsZero() || publication.Lifecycle == "suppressed" {
		return nil, false
	}
	evaluation := publication.AggregateEvaluation
	rankingRows := uint64(len(evaluation.Rows))
	evaluation.Rows = nil
	if !engine.ValidateReplayEvaluationAccounting(evaluation) {
		return nil, true
	}
	return &LastCoherentMarketProjection{
		PublicationID: publication.PublicationID, LastEngineSequence: publication.LastEngineSequence,
		BindingIdentity: publication.BindingIdentity, TradingDate: publication.TradingDate, RunMode: publication.RunMode,
		Lifecycle: publication.Lifecycle, LifecycleReason: publication.LifecycleReason,
		Watermark: cloneTime(publication.Watermark), GeneratedAt: publication.GeneratedAt,
		CurrentMarketClaim: publication.CurrentMarketClaim, RankingRows: rankingRows, Evaluation: evaluation,
	}, false
}

func cloneLastCoherentProjection(value *LastCoherentMarketProjection) *LastCoherentMarketProjection {
	if value == nil {
		return nil
	}
	result := *value
	result.Watermark = cloneTime(value.Watermark)
	result.Evaluation.Rows = nil
	return &result
}

func ingressIdentityResults(metrics Metrics) ([maximumIngressIdentityResults]IngressIdentityResult, int) {
	var result [maximumIngressIdentityResults]IngressIdentityResult
	appendResult := func(name string, left, right uint64) {
		index := 0
		for index < len(result) && result[index].Name != "" {
			index++
		}
		if index < len(result) {
			result[index] = IngressIdentityResult{Name: name, Reconciled: left == right, Left: left, Right: right}
		}
	}
	q := metrics.LiveQueue
	appendResult("queue.frames_read", q.FramesRead, q.FramesAdmitted+q.FramesRejectedOversize+q.FramesRejectedCapacity+q.FramesRejectedReceipt+q.FramesRejectedGateOrClose)
	appendResult("queue.capacity_rejections", q.FramesRejectedCapacity, q.FramesRejectedSlotCapacity+q.FramesRejectedByteCapacity)
	appendResult("queue.frames_admitted", q.FramesAdmitted, q.FramesQueued+q.FramesClassifying+q.FramesDispositioned+q.FramesFenced)
	appendResult("queue.ingress_fences", q.IngressFencesStarted, q.IngressFencesQueued+q.IngressFencesClassifying+q.IngressFencesDispositioned)
	a := metrics.Adapter
	appendResult("adapter.attempts", a.ConnectionAttempts, a.AttemptsActive+a.AttemptsConnected+a.AttemptsFailed+a.AttemptsCanceled)
	appendResult("adapter.commands", a.CommandsStarted, a.CommandsPendingWrite+a.CommandsPendingAck+a.CommandsAcknowledged+a.CommandsFailed+a.CommandsAmbiguous+a.CommandsCanceledOrFenced)
	e := metrics.Engine
	appendResult("engine.admissions_started", e.Admissions.Started, e.Admissions.InProgress+e.Admissions.ResultsCommitted)
	appendResult("engine.admissions_results", e.Admissions.ResultsCommitted, e.Admissions.Admitted+e.Admissions.Invalid+e.Admissions.Canceled+e.Admissions.Closed+e.Admissions.PressureShedOptional+e.Admissions.SequenceExhausted)
	appendResult("engine.admissions_admitted", e.Admissions.Admitted, uint64(e.QueueOccupancy)+e.Admissions.OwnerInProgress+e.Admissions.Completed)
	appendResult("engine.transitions", e.Transitions.CompletedExternal, e.Transitions.AppliedMarket+e.Transitions.AppliedNonmarket+e.Transitions.ExactDuplicate+e.Transitions.Rejected+e.Transitions.Fenced+e.Transitions.TerminalWorkFact+e.Transitions.IntegrityFailure)
	appendResult("engine.aggregates", e.Aggregates.Consumed, e.Aggregates.Inserted+e.Aggregates.Revised+e.Aggregates.WithdrawnConflict+e.Aggregates.ExactDuplicate+e.Aggregates.Rejected+e.Aggregates.Fenced+e.Aggregates.Integrity)
	appendResult("engine.connection", e.Connection.Consumed, e.Connection.Applied+e.Connection.Deferred+e.Connection.Rejected+e.Connection.Fenced+e.Connection.Integrity)
	h := e.Hydration.Accounting
	appendResult("engine.hydration_work", h.Planned, h.Open+h.CompletedValue+h.CompletedEmpty+h.Failed+h.Canceled+h.Fenced)
	hr := e.Hydration.Rows
	appendResult("engine.hydration_rows", hr.Consumed, hr.Inserted+hr.Duplicate+hr.ConflictOrWithdrawal+hr.Rejected+hr.Fenced+hr.Integrity)
	count := 0
	for count < len(result) && result[count].Name != "" {
		count++
	}
	return result, count
}

func firstFailedIngressIdentity(metrics Metrics) (string, bool) {
	identities, count := ingressIdentityResults(metrics)
	for index := 0; index < count && index < 6; index++ {
		if !identities[index].Reconciled {
			return identities[index].Name, true
		}
	}
	return "", false
}
