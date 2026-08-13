package checkpoint

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Request struct {
	BindingIdentity  string
	RequestID        uint64
	ArtifactSequence uint64
	T0               time.Time
	image            Image
}

// ProjectionBuilder isolates each supplied symbol into package-private
// storage. Callers can retain or mutate their source values after AddSymbol;
// neither the completed Request nor writer-observed bytes can alias them.
type ProjectionBuilder struct {
	image Image
	next  int
}

func NewProjectionBuilder(schemaVersion, producerMode string, binding Binding, t0, createdAt time.Time, sequence, population uint64) (*ProjectionBuilder, bool) {
	if schemaVersion == "" || producerMode == "" || binding.Identity == "" || t0.IsZero() || createdAt.Before(t0) || sequence == 0 || population == 0 || population > MaximumSymbols {
		return nil, false
	}
	return &ProjectionBuilder{image: Image{SchemaVersion: schemaVersion, ProducerMode: producerMode, Binding: binding, T0: t0, CreatedAt: createdAt,
		Sequence: sequence, Population: int(population), Symbols: make([]Symbol, int(population))}}, true
}

func (b *ProjectionBuilder) AddSymbol(index int, symbol Symbol) bool {
	if b == nil || index != b.next || index < 0 || index >= len(b.image.Symbols) {
		return false
	}
	b.image.Symbols[index] = cloneSymbol(symbol)
	b.next++
	return true
}

func (b *ProjectionBuilder) Finish(bindingIdentity string, requestID uint64) (Request, bool) {
	if b == nil || b.next != len(b.image.Symbols) || bindingIdentity != b.image.Binding.Identity || requestID == 0 {
		return Request{}, false
	}
	b.image.Counts = structureCounts(b.image.Symbols)
	if b.image.Counts.RecordsWithState+b.image.Counts.EmptyStateRecords != b.image.Population {
		return Request{}, false
	}
	request := Request{BindingIdentity: bindingIdentity, RequestID: requestID, ArtifactSequence: b.image.Sequence, T0: b.image.T0, image: b.image}
	b.image = Image{}
	b.next = 0
	return request, true
}

func structureCounts(symbols []Symbol) StructureCounts {
	var c StructureCounts
	for _, s := range symbols {
		if s.HasState {
			c.RecordsWithState++
		} else {
			c.EmptyStateRecords++
		}
		c.TailRecords += len(s.Tail)
		c.PresenceWords += len(s.Presence)
		c.ProvenAbsentWords += len(s.ProvenAbsent)
		c.ConflictWords += len(s.HistoricalConflict)
		if s.PriceRange != nil {
			c.PriceExtremaPoints += len(s.PriceRange.Highs) + len(s.PriceRange.Lows) + len(s.PriceRange.SessionHighs) + len(s.PriceRange.SessionLows)
		}
		if s.Activity != nil {
			c.ActivityReferences += len(s.Activity.References)
			c.ActivityMutable += len(s.Activity.Mutable)
			c.ActivityTargetBlocks += len(s.Activity.FoldedTargets)
			c.ActivityTargetContributions += s.Activity.FoldedTargetContributions
		}
		if s.Qualification != nil {
			c.QualificationGateBars += len(s.Qualification.FinalizedGateBars)
			c.QualificationProofs += len(s.Qualification.Proofs)
			c.QualificationDirty += len(s.Qualification.Dirty)
		}
		if s.InvalidMarkStart != nil {
			c.InvalidMarks++
		}
		if s.Coverage != nil {
			c.CoverageConsequences++
		}
	}
	return c
}

// NewRequest is the defensive external/test constructor. It isolates the
// writer from every shallow alias a caller may have retained before transfer.
// Production projection uses ProjectionBuilder to spread this same isolation
// across bounded FIFO continuations rather than cloning one complete image.
func NewRequest(bindingIdentity string, requestID uint64, image *Image) (Request, bool) {
	if image == nil || bindingIdentity == "" || requestID == 0 || image.Sequence == 0 || image.T0.IsZero() ||
		image.Binding.Identity != bindingIdentity {
		return Request{}, false
	}
	owned := image.Clone()
	request := Request{BindingIdentity: bindingIdentity, RequestID: requestID, ArtifactSequence: owned.Sequence, T0: owned.T0, image: owned}
	*image = Image{}
	return request, true
}

type TerminalDisposition string

const (
	TerminalCompleted  TerminalDisposition = "completed"
	TerminalFailed     TerminalDisposition = "failed"
	TerminalCanceled   TerminalDisposition = "canceled"
	TerminalSuperseded TerminalDisposition = "superseded"
)

type TerminalResult struct {
	BindingIdentity             string
	RequestID, ArtifactSequence uint64
	T0                          time.Time
	Disposition                 TerminalDisposition
	Step                        WriteStep
	Reason                      string
}

type SubmitDisposition string

const (
	SubmitAccepted SubmitDisposition = "accepted"
	SubmitRejected SubmitDisposition = "rejected"
)

type SubmitResult struct {
	Disposition SubmitDisposition
	Superseded  *TerminalResult
}

type Submitter interface{ Submit(Request) SubmitResult }

type WriterAccounting struct {
	Submitted, InProgress, Pending                                      uint64
	Completed, Failed, Canceled, Superseded                             uint64
	LastSubmittedT0, LastSuccessfulT0                                   time.Time
	LastArtifactBytes                                                   int64
	LastWriteDuration, LastEncodeDuration, LastReopenValidationDuration time.Duration
	LastFailureStep                                                     WriteStep
}

func (a WriterAccounting) Reconciles() bool {
	return a.Submitted == a.InProgress+a.Pending+a.Completed+a.Failed+a.Canceled+a.Superseded && a.InProgress <= 1 && a.Pending <= 1
}

// Writer owns one sequential filesystem operation and one replaceable pending
// detached request. It never receives an engine reference.
type Writer struct {
	store          *Store
	mu             sync.Mutex
	pending        *Request
	inProgress     *Request
	accounting     WriterAccounting
	wake           chan struct{}
	terminalWake   chan struct{}
	terminals      []TerminalResult
	deliveryBudget uint64
	ctx            context.Context
	cancel         context.CancelFunc
	done           chan struct{}
	closed         bool
}

func NewWriter(parent context.Context, store *Store) (*Writer, error) {
	if parent == nil || store == nil {
		return nil, errors.New("writer requires context and store")
	}
	ctx, cancel := context.WithCancel(parent)
	w := &Writer{store: store, wake: make(chan struct{}, 1), terminalWake: make(chan struct{}, 1), terminals: make([]TerminalResult, 0, 256), ctx: ctx, cancel: cancel, done: make(chan struct{})}
	go w.run()
	return w, nil
}

func (w *Writer) Submit(request Request) SubmitResult {
	if w == nil {
		return SubmitResult{Disposition: SubmitRejected}
	}
	if request.BindingIdentity == "" || request.BindingIdentity != w.store.binding || request.RequestID == 0 || request.ArtifactSequence == 0 || request.T0.IsZero() || request.image.Binding.Identity != request.BindingIdentity || request.image.Sequence != request.ArtifactSequence || request.image.T0 != request.T0 {
		return SubmitResult{Disposition: SubmitRejected}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return SubmitResult{Disposition: SubmitRejected}
	}
	if w.deliveryBudget >= 256 {
		return SubmitResult{Disposition: SubmitRejected}
	}
	w.accounting.Submitted++
	w.accounting.LastSubmittedT0 = request.T0
	w.deliveryBudget++
	var result SubmitResult
	result.Disposition = SubmitAccepted
	if w.pending != nil {
		old := *w.pending
		terminal := terminalFor(old, TerminalSuperseded, "", "newer_pending_request")
		w.accounting.Superseded++
		w.deliveryBudget--
		result.Superseded = &terminal
	}
	w.pending = &request
	w.accounting.Pending = 1
	select {
	case w.wake <- struct{}{}:
	default:
	}
	return result
}

func (w *Writer) NextResult(ctx context.Context) (TerminalResult, error) {
	if ctx == nil {
		return TerminalResult{}, errors.New("result context required")
	}
	for {
		w.mu.Lock()
		if len(w.terminals) != 0 {
			result := w.terminals[0]
			copy(w.terminals, w.terminals[1:])
			w.terminals[len(w.terminals)-1] = TerminalResult{}
			w.terminals = w.terminals[:len(w.terminals)-1]
			w.deliveryBudget--
			w.mu.Unlock()
			return result, nil
		}
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return TerminalResult{}, errors.New("writer closed")
		}
		select {
		case <-w.terminalWake:
		case <-w.done:
		case <-ctx.Done():
			return TerminalResult{}, ctx.Err()
		}
	}
}

func (w *Writer) Accounting() WriterAccounting {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.accounting
}

func (w *Writer) Close() { w.cancel() }

func (w *Writer) Wait(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Writer) run() {
	defer close(w.done)
	defer func() {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()
		select {
		case w.terminalWake <- struct{}{}:
		default:
		}
	}()
	for {
		select {
		case <-w.ctx.Done():
			w.cancelPending()
			return
		case <-w.wake:
			for {
				request, ok := w.take()
				if !ok {
					break
				}
				write := w.store.Write(w.ctx, request.image)
				disposition, reason := TerminalCompleted, ""
				switch write.Disposition {
				case WriteCompleted, WriteCleanupDeferred:
					disposition = TerminalCompleted
					if write.Disposition == WriteCleanupDeferred {
						reason = "cleanup_deferred"
					}
				case WriteCanceled:
					disposition, reason = TerminalCanceled, "canceled"
				default:
					disposition, reason = TerminalFailed, "write_failed"
				}
				terminal := terminalFor(request, disposition, write.Step, reason)
				w.finish(disposition, write, request.T0)
				w.publishTerminal(terminal)
			}
		}
	}
}

func (w *Writer) take() (Request, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pending == nil {
		return Request{}, false
	}
	request := *w.pending
	w.pending = nil
	w.accounting.Pending = 0
	w.inProgress = &request
	w.accounting.InProgress = 1
	return request, true
}

func (w *Writer) finish(disposition TerminalDisposition, write WriteResult, t0 time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inProgress = nil
	w.accounting.InProgress = 0
	w.accounting.LastArtifactBytes = write.Entry.FileBytes
	w.accounting.LastWriteDuration = write.TotalDuration
	w.accounting.LastEncodeDuration = write.EncodeDuration
	w.accounting.LastReopenValidationDuration = write.ReopenValidationDuration
	w.accounting.LastFailureStep = ""
	switch disposition {
	case TerminalCompleted:
		w.accounting.Completed++
		w.accounting.LastSuccessfulT0 = t0
	case TerminalFailed:
		w.accounting.Failed++
		w.accounting.LastFailureStep = write.Step
	case TerminalCanceled:
		w.accounting.Canceled++
		w.accounting.LastFailureStep = write.Step
	}
}

func (w *Writer) cancelPending() {
	w.mu.Lock()
	w.closed = true
	if w.pending == nil {
		w.mu.Unlock()
		return
	}
	request := *w.pending
	w.pending = nil
	w.accounting.Pending = 0
	w.accounting.Canceled++
	w.mu.Unlock()
	w.publishTerminal(terminalFor(request, TerminalCanceled, "", "shutdown"))
}

func (w *Writer) publishTerminal(result TerminalResult) {
	w.mu.Lock()
	w.terminals = append(w.terminals, result)
	w.mu.Unlock()
	select {
	case w.terminalWake <- struct{}{}:
	default:
	}
}

func terminalFor(request Request, disposition TerminalDisposition, step WriteStep, reason string) TerminalResult {
	return TerminalResult{BindingIdentity: request.BindingIdentity, RequestID: request.RequestID, ArtifactSequence: request.ArtifactSequence, T0: request.T0, Disposition: disposition, Step: step, Reason: reason}
}

func ValidTerminalResult(result TerminalResult) bool {
	switch result.Disposition {
	case TerminalCompleted:
		return result.Step == "" && result.Reason == "" || result.Step == StepCleanup && result.Reason == "cleanup_deferred"
	case TerminalFailed:
		return validTerminalStep(result.Step) && result.Step != StepCleanup && result.Reason == "write_failed"
	case TerminalCanceled:
		return (result.Step == "" || validTerminalStep(result.Step) && result.Step != StepCleanup) && (result.Reason == "canceled" || result.Reason == "shutdown")
	case TerminalSuperseded:
		return result.Step == "" && result.Reason == "newer_pending_request"
	default:
		return false
	}
}

func validTerminalStep(step WriteStep) bool {
	switch step {
	case StepTempCreate, StepPayloadEncode, StepFileSync, StepFileClose, StepReopenValidation, StepGenerationRename, StepGenerationDirSync, StepManifestCreate, StepManifestEncode, StepManifestSync, StepManifestClose, StepManifestReopen, StepManifestRename, StepManifestDirSync, StepCleanup:
		return true
	default:
		return false
	}
}
