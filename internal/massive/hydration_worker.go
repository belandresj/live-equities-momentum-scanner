package massive

import (
	"context"
	"errors"
	"math"
	"net/http"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	MassiveProviderIdentity   = "massive"
	HydrationMaximumRows      = 16 * 60 * 60
	HydrationMaximumWorkItems = 100_000
)

// HydrationPurpose is engine intent carried as identity only. It gives the
// worker no authority over engine lifecycle or coverage consequences.
type HydrationPurpose string

const (
	HydrationFreshStart        HydrationPurpose = "fresh_start"
	HydrationCheckpointCatchUp HydrationPurpose = "checkpoint_catch_up"
	HydrationGapRecovery       HydrationPurpose = "gap_recovery"
)

// HydrationWorkItem is one immutable, exact-identity REST request. Its private
// fields prevent a caller from changing request identity after construction.
type HydrationWorkItem struct {
	bindingID       string
	generation      uint64
	requestID       uint64
	purpose         HydrationPurpose
	symbol          string
	start, end      time.Time
	connectionEpoch uint64
}

func NewHydrationWorkItem(binding reference.Binding, generation, requestID uint64, purpose HydrationPurpose, symbol string, start, end time.Time, connectionEpoch uint64) (HydrationWorkItem, error) {
	symbols := binding.UniverseSymbols()
	_, member := slices.BinarySearch(symbols, symbol)
	if binding.Identity() == "" || generation == 0 || requestID == 0 || !validHydrationPurpose(purpose) || !member ||
		start != start.UTC() || end != end.UTC() || start.Nanosecond() != 0 || end.Nanosecond() != 0 ||
		start.Before(binding.SessionStart()) || !start.Before(end) || end.After(binding.SessionEnd()) ||
		end.Sub(start) > HydrationMaximumRows*time.Second {
		return HydrationWorkItem{}, errors.New("invalid Massive hydration work item")
	}
	return HydrationWorkItem{
		bindingID: binding.Identity(), generation: generation, requestID: requestID,
		purpose: purpose, symbol: symbol, start: start, end: end, connectionEpoch: connectionEpoch,
	}, nil
}

func validHydrationPurpose(purpose HydrationPurpose) bool {
	return purpose == HydrationFreshStart || purpose == HydrationCheckpointCatchUp || purpose == HydrationGapRecovery
}

func (w HydrationWorkItem) BindingIdentity() string   { return w.bindingID }
func (w HydrationWorkItem) Generation() uint64        { return w.generation }
func (w HydrationWorkItem) RequestID() uint64         { return w.requestID }
func (w HydrationWorkItem) ResultID() uint64          { return w.requestID }
func (w HydrationWorkItem) Purpose() HydrationPurpose { return w.purpose }
func (w HydrationWorkItem) Symbol() string            { return w.symbol }
func (w HydrationWorkItem) Start() time.Time          { return w.start }
func (w HydrationWorkItem) End() time.Time            { return w.end }
func (w HydrationWorkItem) ConnectionEpoch() uint64   { return w.connectionEpoch }
func (w HydrationWorkItem) ProviderIdentity() string  { return MassiveProviderIdentity }

// HydrationWorkerPlan validates every hard bound before any worker performs I/O.
type HydrationWorkerPlan struct {
	items                    []HydrationWorkItem
	workers                  int
	rowsPerChunk             int
	maximumResponseBytes     int64
	maximumNormalizedRecords int64
	maximumResidentRecords   int64
}

func NewHydrationWorkerPlan(items []HydrationWorkItem, workers, rowsPerChunk int, maximumResponseBytes, maximumNormalizedRecords, maximumResidentRecords int64) (HydrationWorkerPlan, error) {
	if len(items) == 0 || len(items) > HydrationMaximumWorkItems || workers < 1 || workers > OfflineWorkerLimit ||
		rowsPerChunk < 1 || rowsPerChunk > HydrationMaximumRows || maximumResponseBytes <= 0 ||
		maximumNormalizedRecords <= 0 || maximumResidentRecords <= 0 {
		return HydrationWorkerPlan{}, errors.New("invalid Massive hydration worker plan")
	}
	seenRequests := make(map[uint64]struct{}, len(items))
	seenSymbols := make(map[string]struct{}, len(items))
	resultSizes := make([]int64, 0, len(items))
	var possibleRows int64
	for index, item := range items {
		seconds := int64(item.end.Sub(item.start) / time.Second)
		if item.bindingID == "" || item.generation == 0 || item.requestID == 0 || !validHydrationPurpose(item.purpose) || item.symbol == "" ||
			seconds <= 0 || seconds > HydrationMaximumRows {
			return HydrationWorkerPlan{}, errors.New("invalid Massive hydration worker plan item")
		}
		if index > 0 && (item.bindingID != items[0].bindingID || item.generation != items[0].generation ||
			item.purpose != items[0].purpose || item.connectionEpoch != items[0].connectionEpoch) {
			return HydrationWorkerPlan{}, errors.New("mixed Massive hydration plan identity")
		}
		if _, duplicate := seenRequests[item.requestID]; duplicate {
			return HydrationWorkerPlan{}, errors.New("duplicate Massive hydration request identity")
		}
		if _, duplicate := seenSymbols[item.symbol]; duplicate {
			return HydrationWorkerPlan{}, errors.New("duplicate Massive hydration symbol")
		}
		seenRequests[item.requestID] = struct{}{}
		seenSymbols[item.symbol] = struct{}{}
		if possibleRows > math.MaxInt64-seconds {
			return HydrationWorkerPlan{}, errors.New("Massive hydration row bound overflow")
		}
		possibleRows += seconds
		resultSizes = append(resultSizes, seconds)
	}
	sort.Slice(resultSizes, func(left, right int) bool { return resultSizes[left] > resultSizes[right] })
	var maximumConcurrentRows int64
	for _, size := range resultSizes[:min(workers, len(resultSizes))] {
		maximumConcurrentRows += size
	}
	if possibleRows > maximumNormalizedRecords || maximumConcurrentRows > maximumResidentRecords {
		return HydrationWorkerPlan{}, errors.New("Massive hydration plan budgets cannot contain requested intervals")
	}
	return HydrationWorkerPlan{
		items: slices.Clone(items), workers: workers, rowsPerChunk: rowsPerChunk,
		maximumResponseBytes: maximumResponseBytes, maximumNormalizedRecords: maximumNormalizedRecords,
		maximumResidentRecords: maximumResidentRecords,
	}, nil
}

type HydrationTerminalState string

const (
	HydrationCompletedValue HydrationTerminalState = "completed_value"
	HydrationCompletedEmpty HydrationTerminalState = "completed_empty"
	HydrationFailed         HydrationTerminalState = "failed"
	HydrationCanceled       HydrationTerminalState = "canceled"
)

// HydrationResultChunk contains a copied slice from a fully sealed provider
// result. Values returns another copy so admitted facts cannot be mutated.
type HydrationResultChunk struct {
	work        HydrationWorkItem
	ordinal     int
	totalChunks int
	rowOffset   int64
	totalRows   int64
	values      []RESTSecondAggregate
}

func (c HydrationResultChunk) WorkItem() HydrationWorkItem   { return c.work }
func (c HydrationResultChunk) ResultID() uint64              { return c.work.ResultID() }
func (c HydrationResultChunk) Ordinal() int                  { return c.ordinal }
func (c HydrationResultChunk) TotalChunks() int              { return c.totalChunks }
func (c HydrationResultChunk) RowOffset() int64              { return c.rowOffset }
func (c HydrationResultChunk) TotalRows() int64              { return c.totalRows }
func (c HydrationResultChunk) Values() []RESTSecondAggregate { return slices.Clone(c.values) }

type HydrationTerminal struct {
	work            HydrationWorkItem
	state           HydrationTerminalState
	reason          DownloadReason
	pages, attempts int64
	responseBytes   int64
	normalizedRows  int64
	emittedChunks   int64
	emittedRows     int64
}

func (t HydrationTerminal) WorkItem() HydrationWorkItem   { return t.work }
func (t HydrationTerminal) ResultID() uint64              { return t.work.ResultID() }
func (t HydrationTerminal) State() HydrationTerminalState { return t.state }
func (t HydrationTerminal) Reason() DownloadReason        { return t.reason }
func (t HydrationTerminal) Pages() int64                  { return t.pages }
func (t HydrationTerminal) Attempts() int64               { return t.attempts }
func (t HydrationTerminal) ResponseBytes() int64          { return t.responseBytes }
func (t HydrationTerminal) NormalizedRows() int64         { return t.normalizedRows }
func (t HydrationTerminal) EmittedChunks() int64          { return t.emittedChunks }
func (t HydrationTerminal) EmittedRows() int64            { return t.emittedRows }

// HydrationFactSink is the bounded Component 2 admission seam. Implementations
// must honor context cancellation, return ErrHydrationInputClosed only for a
// definitively closed input, and otherwise transfer terminal ownership before
// returning nil.
type HydrationFactSink interface {
	AdmitHydrationChunk(context.Context, HydrationResultChunk) error
	AdmitHydrationTerminal(context.Context, HydrationTerminal) error
}

// ErrHydrationInputClosed is the only terminal-admission error that permits
// cleanup without an engine-consumed terminal. Other errors mean the input is
// still open; after work cancellation the worker retries exactly one canceled
// terminal against the independent engine-input lifetime.
var ErrHydrationInputClosed = errors.New("hydration engine input closed")

type HydrationWorkerAccounting struct {
	ItemsStarted               int64
	ProviderCompletedValue     int64
	ProviderCompletedEmpty     int64
	ProviderFailed             int64
	ProviderCanceled           int64
	Pages                      int64
	Attempts                   int64
	ResponseBytes              int64
	NormalizedRows             int64
	EmittedChunks              int64
	EmittedRows                int64
	UnadmittedTerminals        int64
	AdmissionIntegrityFailures int64
	MaximumActiveWorkers       int64
	MaximumResidentRecords     int64
}

type HydrationWorkerResult struct {
	terminals  []HydrationTerminal
	accounting HydrationWorkerAccounting
}

func (r HydrationWorkerResult) Terminals() []HydrationTerminal        { return slices.Clone(r.terminals) }
func (r HydrationWorkerResult) Accounting() HydrationWorkerAccounting { return r.accounting }

type HydrationWorker struct {
	acquisition *aggregateRESTClient
}

func NewHydrationWorker(baseURL string, credential CredentialSource, client *http.Client) (*HydrationWorker, error) {
	acquisition, err := newAggregateRESTClient(baseURL, credential, client)
	if err != nil {
		return nil, err
	}
	return &HydrationWorker{acquisition: acquisition}, nil
}

// Run joins all workers before returning. workContext cancels provider and
// chunk work; admissionContext independently represents the lifetime of the
// engine input so a canceled work item can still admit its canceled terminal.
// Every accepted plan item has exactly one terminal in the returned cleanup
// result even if that input closes before the terminal can be admitted.
func (w *HydrationWorker) Run(workContext, admissionContext context.Context, plan HydrationWorkerPlan, sink HydrationFactSink) HydrationWorkerResult {
	result := HydrationWorkerResult{terminals: make([]HydrationTerminal, len(plan.items))}
	result.accounting.ItemsStarted = int64(len(plan.items))
	if len(plan.items) == 0 {
		return result
	}
	if workContext == nil {
		workContext = canceledContext()
	}
	if admissionContext == nil {
		admissionContext = canceledContext()
	}
	ctx, cancel := context.WithCancel(workContext)
	stopAdmissionCancellation := context.AfterFunc(admissionContext, cancel)
	defer func() {
		stopAdmissionCancellation()
		cancel()
	}()
	if ctx.Err() != nil || admissionContext.Err() != nil || w == nil || w.acquisition == nil || sink == nil {
		state, reason := HydrationFailed, DownloadReasonRequestConstruction
		if ctx.Err() != nil || admissionContext.Err() != nil {
			state, reason = HydrationCanceled, DownloadReasonCanceled
		}
		var unadmitted, admissionIntegrity int64
		for index, item := range plan.items {
			terminal := HydrationTerminal{work: item, state: state, reason: reason}
			result.terminals[index] = terminal
			terminalContext := ctx
			if state == HydrationCanceled {
				terminalContext = admissionContext
			}
			if sink == nil {
				admissionIntegrity++
				continue
			}
			if err := sink.AdmitHydrationTerminal(terminalContext, terminal); err != nil {
				if errors.Is(err, ErrHydrationInputClosed) || admissionContext.Err() != nil {
					unadmitted++
				} else {
					admissionIntegrity++
				}
			}
		}
		result.accounting.AdmissionIntegrityFailures = admissionIntegrity
		return finishHydrationWorkerResult(result, unadmitted, 0, 0)
	}

	token, credentialErr := w.acquisition.credential()
	credentialValid := credentialErr == nil && token != "" && !containsCredentialLineBreak(token)
	wireBytes := responseBudget{maximum: plan.maximumResponseBytes}
	var normalizedRecords atomic.Int64
	resident := residentRecordBudget{maximum: plan.maximumResidentRecords}
	var activeWorkers atomic.Int64
	var maximumWorkers atomic.Int64
	var unadmitted atomic.Int64
	var admissionIntegrity atomic.Int64

	finish := func(index int, terminal HydrationTerminal) {
		result.terminals[index] = terminal
		terminalContext := ctx
		if terminal.state == HydrationCanceled {
			terminalContext = admissionContext
		}
		err := sink.AdmitHydrationTerminal(terminalContext, terminal)
		if err == nil {
			return
		}
		if errors.Is(err, ErrHydrationInputClosed) || admissionContext.Err() != nil {
			unadmitted.Add(1)
			return
		}
		// A work-canceled admission is not evidence that engine input closed.
		// Retry once through the independent input lifetime with the canceled
		// fact; a conforming open sink owns bounded progress for this terminal.
		if ctx.Err() != nil && terminal.state != HydrationCanceled {
			terminal.state = HydrationCanceled
			terminal.reason = DownloadReasonCanceled
			result.terminals[index] = terminal
			err = sink.AdmitHydrationTerminal(admissionContext, terminal)
			if err == nil {
				return
			}
			if errors.Is(err, ErrHydrationInputClosed) || admissionContext.Err() != nil {
				unadmitted.Add(1)
				return
			}
		}
		// A sink that rejects while declaring input open violates the bounded
		// admission seam. Preserve that distinct diagnostic without calling it
		// closed-input cleanup.
		admissionIntegrity.Add(1)
	}
	process := func(index int) {
		item := plan.items[index]
		if !credentialValid {
			finish(index, HydrationTerminal{work: item, state: HydrationFailed, reason: DownloadReasonRequestConstruction})
			return
		}
		if ctx.Err() != nil {
			finish(index, HydrationTerminal{work: item, state: HydrationCanceled, reason: DownloadReasonCanceled})
			return
		}
		active := activeWorkers.Add(1)
		updateAtomicMaximum(&maximumWorkers, active)
		defer activeWorkers.Add(-1)
		values, outcome := w.acquisition.acquire(ctx, token, item.symbol, aggregateRESTRequest{
			start: item.start, end: item.end, maximumNormalizedRecords: plan.maximumNormalizedRecords,
		}, &wireBytes, &normalizedRecords, &resident)
		terminal := HydrationTerminal{
			work: item, reason: outcome.Reason, pages: outcome.Pages, attempts: outcome.Attempts,
			responseBytes: outcome.Bytes, normalizedRows: outcome.Records,
		}
		if outcome.State == SymbolCanceled {
			terminal.state = HydrationCanceled
			finish(index, terminal)
			return
		}
		if outcome.State != SymbolComplete {
			terminal.state = HydrationFailed
			finish(index, terminal)
			return
		}
		defer resident.release(int64(len(values)))
		if len(values) == 0 {
			terminal.state = HydrationCompletedEmpty
			finish(index, terminal)
			return
		}

		totalChunks := (len(values) + plan.rowsPerChunk - 1) / plan.rowsPerChunk
		for ordinal, offset := 0, 0; offset < len(values); ordinal, offset = ordinal+1, offset+plan.rowsPerChunk {
			if ctx.Err() != nil {
				terminal.state, terminal.reason = HydrationCanceled, DownloadReasonCanceled
				finish(index, terminal)
				return
			}
			end := min(offset+plan.rowsPerChunk, len(values))
			chunk := HydrationResultChunk{
				work: item, ordinal: ordinal, totalChunks: totalChunks, rowOffset: int64(offset),
				totalRows: int64(len(values)), values: slices.Clone(values[offset:end]),
			}
			if sink.AdmitHydrationChunk(ctx, chunk) != nil {
				terminal.state, terminal.reason = HydrationCanceled, DownloadReasonCanceled
				finish(index, terminal)
				return
			}
			terminal.emittedChunks++
			terminal.emittedRows += int64(end - offset)
		}
		if ctx.Err() != nil {
			terminal.state, terminal.reason = HydrationCanceled, DownloadReasonCanceled
		} else {
			terminal.state, terminal.reason = HydrationCompletedValue, DownloadReasonNone
		}
		finish(index, terminal)
	}

	workers := min(plan.workers, len(plan.items))
	jobs := make(chan int)
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for index := range jobs {
				process(index)
			}
		}()
	}
	next := 0
	for ; next < len(plan.items); next++ {
		select {
		case jobs <- next:
		case <-ctx.Done():
			close(jobs)
			group.Wait()
			for ; next < len(plan.items); next++ {
				finish(next, HydrationTerminal{work: plan.items[next], state: HydrationCanceled, reason: DownloadReasonCanceled})
			}
			result.accounting.AdmissionIntegrityFailures = admissionIntegrity.Load()
			return finishHydrationWorkerResult(result, unadmitted.Load(), maximumWorkers.Load(), resident.maximumObserved())
		}
	}
	close(jobs)
	group.Wait()
	result.accounting.AdmissionIntegrityFailures = admissionIntegrity.Load()
	return finishHydrationWorkerResult(result, unadmitted.Load(), maximumWorkers.Load(), resident.maximumObserved())
}

func finishHydrationWorkerResult(result HydrationWorkerResult, unadmitted, maximumWorkers, maximumResident int64) HydrationWorkerResult {
	result.accounting.UnadmittedTerminals = unadmitted
	result.accounting.MaximumActiveWorkers = maximumWorkers
	result.accounting.MaximumResidentRecords = maximumResident
	for _, terminal := range result.terminals {
		result.accounting.Pages += terminal.pages
		result.accounting.Attempts += terminal.attempts
		result.accounting.ResponseBytes += terminal.responseBytes
		result.accounting.NormalizedRows += terminal.normalizedRows
		result.accounting.EmittedChunks += terminal.emittedChunks
		result.accounting.EmittedRows += terminal.emittedRows
		switch terminal.state {
		case HydrationCompletedValue:
			result.accounting.ProviderCompletedValue++
		case HydrationCompletedEmpty:
			result.accounting.ProviderCompletedEmpty++
		case HydrationCanceled:
			result.accounting.ProviderCanceled++
		default:
			result.accounting.ProviderFailed++
		}
	}
	return result
}

type residentRecordBudget struct {
	current atomic.Int64
	peak    atomic.Int64
	maximum int64
}

func (b *residentRecordBudget) reserve() bool {
	for {
		current := b.current.Load()
		if current >= b.maximum || !b.current.CompareAndSwap(current, current+1) {
			if current >= b.maximum {
				return false
			}
			continue
		}
		updateAtomicMaximum(&b.peak, current+1)
		return true
	}
}

func (b *residentRecordBudget) release(count int64) {
	if count > 0 {
		b.current.Add(-count)
	}
}

func (b *residentRecordBudget) maximumObserved() int64 { return b.peak.Load() }

func updateAtomicMaximum(target *atomic.Int64, value int64) {
	for current := target.Load(); value > current; current = target.Load() {
		if target.CompareAndSwap(current, value) {
			return
		}
	}
}

func containsCredentialLineBreak(value string) bool {
	for _, character := range value {
		if character == '\r' || character == '\n' {
			return true
		}
	}
	return false
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
