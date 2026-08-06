package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func TestENGRUN01BindingInstallLifecycleAtomicity(t *testing.T) {
	binding := testBinding(t)
	start, end := binding.SessionStart(), binding.SessionEnd()

	for _, test := range []struct {
		name string
		mode RunMode
		now  time.Time
		want lifecycle
	}{
		{"live before S LIFE-T01", RunModeLive, start.Add(-time.Nanosecond), lifecycleAwaitingSession},
		{"live at S LIFE-T02", RunModeLive, start, lifecycleAwaitingAggregateAck},
		{"live before E LIFE-T02", RunModeLive, end.Add(-time.Nanosecond), lifecycleAwaitingAggregateAck},
		{"live at E LIFE-T05", RunModeLive, end, lifecycleEnded},
		{"replay awaits artifact evidence", RunModeReplay, start, lifecycleInitializing},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := testEngine(t, test.mode, test.now, 4, 1)
			completion := admitValidBinding(t, e, binding)
			disposition := awaitDisposition(t, completion)
			if disposition != (Disposition{EngineSequence: 1, Code: DispositionBindingInstalled}) {
				t.Fatalf("disposition = %+v", disposition)
			}
			e.mu.Lock()
			state := e.state
			if state.lifecycle != test.want || state.binding == nil || len(state.binding.symbols) != len(binding.UniverseSymbols()) ||
				len(state.binding.index) != len(binding.UniverseSymbols()) || state.binding.identity != binding.Identity() ||
				state.binding.tradingDate != binding.TradingDate() || state.binding.sessionStart != binding.SessionStart() ||
				state.binding.sessionEnd != binding.SessionEnd() || state.binding.scheduleSchema != binding.ScheduleSchema() ||
				state.binding.scheduleVersion != binding.ScheduleVersion() || state.binding.scheduleArtifactSHA256 != binding.ScheduleArtifactSHA256() ||
				state.binding.priorSessionDate != binding.PriorSessionDate() || state.binding.priorRegularClose != binding.PriorRegularClose() ||
				state.binding.universePolicy != binding.UniversePolicy() || state.binding.universeIdentity != binding.UniverseIdentity() ||
				state.binding.priorClosePolicy != binding.PriorClosePolicy() || state.binding.priorCloseIdentity != binding.PriorCloseIdentity() ||
				state.binding.adjusted != binding.PriorCloseAdjusted() || state.binding.includeOTC != binding.PriorCloseIncludeOTC() ||
				state.binding.locale != binding.PriorCloseLocale() || state.binding.market != binding.PriorCloseMarket() ||
				state.binding.priorAccounting != binding.PriorCloseAccounting() || state.binding.universeAccounting != binding.UniverseAccounting() {
				t.Fatalf("incomplete atomic state = lifecycle %s binding %+v", state.lifecycle, state.binding)
			}
			e.mu.Unlock()
			closeAndWait(t, e)
		})
	}

	t.Run("zero partial mismatch and failed build leave no partial state", func(t *testing.T) {
		e := testEngine(t, RunModeLive, start, 4, 1)
		result, completion := e.AdmitBinding(context.Background(), BindingInstall{
			SchemaVersion: BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: reference.Binding{},
		})
		if result != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionBindingInvalid {
			t.Fatalf("zero binding result = %s", result)
		}
		assertUnboundInitializing(t, e)

		partial := freezeBinding(binding)
		partial.symbols = partial.symbols[:1]
		result, completion = admitFrozenBinding(e, partial, binding.Identity())
		if result != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionBindingInvalid {
			t.Fatalf("partial binding result = %s", result)
		}
		assertUnboundInitializing(t, e)

		for _, invalid := range []struct {
			name   string
			mutate func(*frozenBinding)
		}{
			{"over-limit population", func(value *frozenBinding) {
				value.symbols = make([]string, maximumUniverseSymbols+1)
				value.priors = make([]frozenPriorClose, maximumUniverseSymbols+1)
			}},
			{"symbol disorder", func(value *frozenBinding) {
				value.symbols[0], value.symbols[1] = value.symbols[1], value.symbols[0]
			}},
			{"duplicate symbol", func(value *frozenBinding) { value.symbols[1] = value.symbols[0] }},
			{"universe accounting mismatch", func(value *frozenBinding) { value.universeAccounting.RawReferenceRecords++ }},
			{"prior accounting mismatch", func(value *frozenBinding) { value.priorAccounting.ValidPriorClose++ }},
		} {
			t.Run(invalid.name, func(t *testing.T) {
				candidate := freezeBinding(binding)
				invalid.mutate(&candidate)
				result, completion := admitFrozenBinding(e, candidate, binding.Identity())
				if result != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionBindingInvalid {
					t.Fatalf("invalid candidate result = %s", result)
				}
				assertUnboundInitializing(t, e)
			})
		}

		wrongIdentity := binding.Identity()[:len(binding.Identity())-1] + "0"
		if wrongIdentity == binding.Identity() {
			wrongIdentity = binding.Identity()[:len(binding.Identity())-1] + "1"
		}
		result, completion = e.AdmitBinding(context.Background(), BindingInstall{
			SchemaVersion: BindingInstallSchemaV1, BindingIdentity: wrongIdentity, Binding: binding,
		})
		if result != AdmissionAdmitted || awaitDisposition(t, completion).Code != DispositionBindingInvalid {
			t.Fatalf("mismatched envelope result = %s", result)
		}
		assertUnboundInitializing(t, e)

		e.buildCandidate = func(frozenBinding) (*installedBinding, error) {
			return &installedBinding{identity: "unreachable-partial"}, errors.New("injected candidate build failure")
		}
		completion = admitValidBinding(t, e, binding)
		if awaitDisposition(t, completion).Code != DispositionBindingInvalid {
			t.Fatal("candidate-build failure escaped as success")
		}
		assertUnboundInitializing(t, e)
		e.buildCandidate = buildInstalledBinding
		completion = admitValidBinding(t, e, binding)
		if awaitDisposition(t, completion).Code != DispositionBindingInstalled {
			t.Fatal("valid install did not recover after rejected candidates")
		}
		e.mu.Lock()
		installed := e.state.binding
		e.mu.Unlock()
		completion = admitValidBinding(t, e, binding)
		if awaitDisposition(t, completion).Code != DispositionBindingAlreadyInstalled {
			t.Fatal("second binding replaced the installed binding")
		}
		e.mu.Lock()
		if e.state.binding != installed {
			t.Fatal("second installation changed the owner pointer")
		}
		e.mu.Unlock()
		closeAndWait(t, e)
	})

	t.Run("concrete provenance and construction bounds", func(t *testing.T) {
		bindingType := reflect.TypeOf(reference.Binding{})
		for index := 0; index < bindingType.NumField(); index++ {
			if bindingType.Field(index).PkgPath == "" {
				t.Fatalf("reference.Binding field %s permits external reconstruction", bindingType.Field(index).Name)
			}
		}
		for _, config := range []Config{
			{}, {Mode: RunMode("other"), Clock: time.Now, Capacity: 2, RequiredReserve: 1},
			{Mode: RunModeLive, Clock: time.Now, Capacity: 4, RequiredReserve: 1},
			{Mode: RunModeLive, Clock: time.Now, Capacity: 1, RequiredReserve: 1},
			{Mode: RunModeLive, Clock: time.Now, Capacity: 4, RequiredReserve: 0},
			{Mode: RunModeLive, Clock: time.Now, Capacity: 4, RequiredReserve: 4},
			{Mode: RunModeLive, Clock: time.Now, Capacity: 4, RequiredReserve: 1, EvaluationDelay: durationPointer(-time.Nanosecond)},
		} {
			if engine, err := New(config); err == nil || engine != nil {
				t.Fatalf("invalid config accepted: %+v", config)
			}
		}
	})

	t.Run("controlled stop seals at linkage and routes LIFE-T05 in FIFO order", func(t *testing.T) {
		e := testEngine(t, RunModeLive, start.Add(-time.Hour), 3, 1)
		bindingEntered, releaseBinding := make(chan struct{}), make(chan struct{})
		stopEntered, releaseStop := make(chan struct{}), make(chan struct{})
		consumeCount := 0
		e.beforeConsume = func(*queueNode) {
			consumeCount++
			if consumeCount == 1 {
				close(bindingEntered)
				<-releaseBinding
				return
			}
			close(stopEntered)
			<-releaseStop
		}
		bindingCompletion := admitValidBinding(t, e, binding)
		<-bindingEntered
		result, stopCompletion := e.Stop(context.Background())
		if result != AdmissionAdmitted {
			t.Fatalf("controlled-stop admission = %s", result)
		}
		result, later := e.AdmitBinding(context.Background(), validBindingInput(binding))
		if result != AdmissionNotAdmittedClosed || later != nil {
			t.Fatalf("post-stop admission = %s", result)
		}
		close(releaseBinding)
		if got := awaitDisposition(t, bindingCompletion); got != (Disposition{EngineSequence: 1, Code: DispositionBindingInstalled}) {
			t.Fatalf("binding before stop = %+v", got)
		}
		<-stopEntered
		e.mu.Lock()
		beforeStop := e.state.lifecycle
		e.mu.Unlock()
		if beforeStop != lifecycleAwaitingSession {
			t.Fatalf("later-linked stop retroactively changed binding route to %s", beforeStop)
		}
		close(releaseStop)
		if got := awaitDisposition(t, stopCompletion); got != (Disposition{EngineSequence: 2, Code: DispositionControlApplied}) {
			t.Fatalf("controlled stop = %+v", got)
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		e.mu.Lock()
		state := e.state.lifecycle
		e.mu.Unlock()
		if state != lifecycleEnded {
			t.Fatalf("controlled-stop lifecycle = %s", state)
		}
	})
}

func TestENGADMIT01BoundedOwnershipAndClosure(t *testing.T) {
	binding := testBinding(t)

	t.Run("ownership freeze and cancellation on each side of linkage", func(t *testing.T) {
		e := testEngine(t, RunModeLive, binding.SessionStart(), 3, 1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, completion := e.AdmitBinding(ctx, validBindingInput(binding))
		if result != AdmissionNotAdmittedCanceled || completion != nil {
			t.Fatalf("pre-link cancellation = %s, %v", result, completion)
		}

		entered, release := make(chan struct{}), make(chan struct{})
		e.beforeConsume = func(*queueNode) { close(entered); <-release }
		ctx, cancel = context.WithCancel(context.Background())
		input := validBindingInput(binding)
		result, completion = e.AdmitBinding(ctx, input)
		if result != AdmissionAdmitted {
			t.Fatalf("admission = %s", result)
		}
		<-entered
		returnedSymbols := input.Binding.UniverseSymbols()
		returnedPriors := input.Binding.PriorCloseFacts()
		returnedSymbols[0] = "CALLER_MUTATION"
		returnedPriors[0], returnedPriors[1] = returnedPriors[1], returnedPriors[0]
		input.Binding = reference.Binding{}
		input.BindingIdentity = "mutated"
		cancel()
		close(release)
		if disposition := awaitDisposition(t, completion); disposition.Code != DispositionBindingInstalled {
			t.Fatalf("post-link cancellation/mutation changed node = %+v", disposition)
		}
		e.mu.Lock()
		if e.state.binding.symbols[0].symbol != "AAA" || e.state.binding.symbols[0].prior.symbol != "AAA" {
			t.Fatal("caller-owned slice mutation aliased the admitted node")
		}
		e.mu.Unlock()
		assertAccounting(t, e)
		closeAndWait(t, e)
	})

	t.Run("reserve pressure blocked close drain and pause accounting", func(t *testing.T) {
		e := testEngine(t, RunModeReplay, binding.SessionStart(), 3, 1)
		entered, release := make(chan struct{}), make(chan struct{})
		var pauseOnce sync.Once
		e.beforeConsume = func(*queueNode) {
			pauseOnce.Do(func() {
				close(entered)
				<-release
			})
		}

		_, first := admitControl(t, e, false)
		<-entered
		_, optionalOne := admitControl(t, e, true)
		_, optionalTwo := admitControl(t, e, true)
		result, completion := admitControl(t, e, true)
		if result != AdmissionPressureShedOptional || completion != nil {
			t.Fatalf("optional reserve result = %s", result)
		}
		_, required := admitControl(t, e, false)
		assertAccounting(t, e)

		blockedResult := make(chan AdmissionResult, 1)
		go func() {
			result, _ := admitControl(t, e, false)
			blockedResult <- result
		}()
		waitForInProgress(t, e, 1)
		e.Close()
		if result := <-blockedResult; result != AdmissionNotAdmittedClosed {
			t.Fatalf("blocked admission after close = %s", result)
		}
		assertAccounting(t, e)
		close(release)
		for _, completion := range []<-chan Disposition{first, optionalOne, optionalTwo, required} {
			if awaitDisposition(t, completion).Code != DispositionControlApplied {
				t.Fatal("linked control did not drain")
			}
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		assertAccounting(t, e)
	})

	t.Run("checked exhaustion drains and reserves terminal sequence", func(t *testing.T) {
		e := testEngine(t, RunModeReplay, binding.SessionStart(), 2, 1)
		e.mu.Lock()
		e.lastReserved = math.MaxUint64 - 2
		e.nextSequence = math.MaxUint64 - 1
		e.mu.Unlock()
		result, completion := admitControl(t, e, false)
		if result != AdmissionAdmitted {
			t.Fatalf("last external admission = %s", result)
		}
		result, extra := admitControl(t, e, false)
		if result != AdmissionSequenceBudgetExhausted || extra != nil {
			t.Fatalf("exhaustion result = %s", result)
		}
		if disposition := awaitDisposition(t, completion); disposition.EngineSequence != math.MaxUint64-1 {
			t.Fatalf("last external sequence = %d", disposition.EngineSequence)
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		e.mu.Lock()
		terminal := e.terminal
		state := e.state.lifecycle
		e.mu.Unlock()
		if terminal == nil || *terminal != (Disposition{EngineSequence: math.MaxUint64, Code: DispositionSequenceExhausted, SuppressionDisposition: SuppressionTerminalReplayFailure}) || state != lifecycleSuppressed {
			t.Fatalf("terminal transition = %+v, lifecycle %s", terminal, state)
		}
		assertAccounting(t, e)
	})
}

func TestENGORDER01DeterministicOrderedTransitionTrace(t *testing.T) {
	binding := testBinding(t)
	e := testEngine(t, RunModeReplay, binding.SessionStart(), 5, 1)

	// Concurrent preparation deliberately completes in reverse order. The
	// coordinator then fixes FIFO linearization explicitly; preparation order
	// is not mutation order.
	prepared := make([]BindingInstall, 3)
	completionOrder := make(chan int)
	gates := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	var workers sync.WaitGroup
	for index := range prepared {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-gates[index]
			prepared[index] = validBindingInput(binding)
			completionOrder <- index
		}(index)
	}
	gotCompletion := make([]int, 0, 3)
	for _, index := range []int{2, 0, 1} {
		close(gates[index])
		gotCompletion = append(gotCompletion, <-completionOrder)
	}
	workers.Wait()
	close(completionOrder)
	if !reflect.DeepEqual(gotCompletion, []int{2, 0, 1}) {
		t.Fatalf("producer completion order = %v", gotCompletion)
	}
	prepared[0].BindingIdentity = mutateIdentity(binding.Identity())

	completions := make([]<-chan Disposition, 0, 3)
	for _, input := range prepared {
		result, completion := e.AdmitBinding(context.Background(), input)
		if result != AdmissionAdmitted {
			t.Fatalf("fixed FIFO admission = %s", result)
		}
		completions = append(completions, completion)
	}
	e.Close()
	want := []Disposition{
		{EngineSequence: 1, Code: DispositionBindingInvalid},
		{EngineSequence: 2, Code: DispositionBindingInstalled},
		{EngineSequence: 3, Code: DispositionBindingAlreadyInstalled},
	}
	for index, completion := range completions {
		if got := awaitDisposition(t, completion); got != want[index] {
			t.Fatalf("trace[%d] = %+v, want %+v", index, got, want[index])
		}
	}
	if err := e.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	assertAccounting(t, e)

	// Reversing linkage is a different trace, which is the intended and only
	// concurrency-dependent choice.
	reversed := testEngine(t, RunModeReplay, binding.SessionStart(), 3, 1)
	first := admitValidBinding(t, reversed, binding)
	invalid := validBindingInput(binding)
	invalid.BindingIdentity = mutateIdentity(binding.Identity())
	_, second := reversed.AdmitBinding(context.Background(), invalid)
	if awaitDisposition(t, first).Code != DispositionBindingInstalled ||
		awaitDisposition(t, second).Code != DispositionBindingAlreadyInstalled {
		t.Fatal("reversed linkage did not produce its deterministic alternate trace")
	}
	closeAndWait(t, reversed)

	// The reserved terminal transition is outside the external FIFO trace and
	// follows the last admitted sequence without wrap or reuse.
	exhausted := testEngine(t, RunModeReplay, binding.SessionStart(), 2, 1)
	exhausted.mu.Lock()
	exhausted.lastReserved = math.MaxUint64 - 2
	exhausted.nextSequence = math.MaxUint64 - 1
	exhausted.mu.Unlock()
	_, lastExternal := admitControl(t, exhausted, false)
	result, terminalCompletion := admitControl(t, exhausted, false)
	if result != AdmissionSequenceBudgetExhausted || terminalCompletion != nil {
		t.Fatalf("ordered exhaustion result = %s", result)
	}
	if got := awaitDisposition(t, lastExternal); got.EngineSequence != math.MaxUint64-1 || got.Code != DispositionControlApplied {
		t.Fatalf("ordered last external disposition = %+v", got)
	}
	if err := exhausted.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	exhausted.mu.Lock()
	terminal := exhausted.terminal
	exhausted.mu.Unlock()
	if terminal == nil || *terminal != (Disposition{EngineSequence: math.MaxUint64, Code: DispositionSequenceExhausted, SuppressionDisposition: SuppressionTerminalReplayFailure}) {
		t.Fatalf("ordered terminal disposition = %+v", terminal)
	}
}

func testBinding(t *testing.T) reference.Binding {
	t.Helper()
	schedule, err := session.Load()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := schedule.ForTradingDate("2026-07-29")
	if err != nil {
		t.Fatal(err)
	}
	priorTimestamp := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC).UnixMilli()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v3/reference/tickers":
			_ = json.NewEncoder(writer).Encode(map[string]any{"results": []map[string]any{
				{"ticker": "AAA", "active": true, "market": "stocks", "locale": "us", "type": "CS"},
				{"ticker": "BAD", "active": true, "market": "stocks", "locale": "us", "type": "ADRC"},
				{"ticker": "MISSING", "active": true, "market": "stocks", "locale": "us", "type": "CS"},
			}})
		case "/v2/aggs/grouped/locale/us/market/stocks/2026-07-28":
			_, _ = fmt.Fprintf(writer, `{"status":"OK","adjusted":true,"resultsCount":2,"results":[{"T":"AAA","c":10.25,"t":%d},{"T":"BAD","c":0,"t":%d}]}`, priorTimestamp, priorTimestamp)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := &reference.Resolver{
		BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(directory, "reference"), HTTPClient: server.Client(), Schedule: schedule,
		Now: func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) }, Sleep: func(context.Context, time.Duration) error { return nil },
	}
	universe, err := resolver.Resolve(context.Background(), facts)
	if err != nil {
		t.Fatal(err)
	}
	priorResolver := &reference.PriorCloseResolver{
		BaseURL: server.URL, APIKey: "test", DataDir: filepath.Join(directory, "reference"), HTTPClient: server.Client(), Schedule: schedule,
		Now: func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) }, Sleep: func(context.Context, time.Duration) error { return nil },
	}
	priors, err := priorResolver.Resolve(context.Background(), facts, universe)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func testEngine(t *testing.T, mode RunMode, now time.Time, capacity, reserve int) *Engine {
	t.Helper()
	delay := time.Duration(0)
	e, err := New(Config{Mode: mode, Clock: func() time.Time { return now }, Capacity: capacity, RequiredReserve: reserve, EvaluationDelay: &delay})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func durationPointer(value time.Duration) *time.Duration { return &value }

func validBindingInput(binding reference.Binding) BindingInstall {
	return BindingInstall{SchemaVersion: BindingInstallSchemaV1, BindingIdentity: binding.Identity(), Binding: binding}
}

func admitValidBinding(t *testing.T, e *Engine, binding reference.Binding) <-chan Disposition {
	t.Helper()
	result, completion := e.AdmitBinding(context.Background(), validBindingInput(binding))
	if result != AdmissionAdmitted || completion == nil {
		t.Fatalf("binding admission = %s", result)
	}
	return completion
}

func admitControl(t *testing.T, e *Engine, optional bool) (AdmissionResult, <-chan Disposition) {
	t.Helper()
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: inputControl}, optional)
}

func admitFrozenBinding(e *Engine, binding frozenBinding, identity string) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: inputBinding, binding: binding, bindingID: identity}, false)
}

func awaitDisposition(t *testing.T, completion <-chan Disposition) Disposition {
	t.Helper()
	select {
	case disposition, ok := <-completion:
		if !ok {
			t.Fatal("completion closed without disposition")
		}
		return disposition
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for disposition")
		return Disposition{}
	}
}

func closeAndWait(t *testing.T, e *Engine) {
	t.Helper()
	e.Close()
	if err := e.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func assertUnboundInitializing(t *testing.T, e *Engine) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.binding != nil || e.state.lifecycle != lifecycleInitializing {
		t.Fatalf("failed install exposed partial state: %+v", e.state)
	}
}

func assertAccounting(t *testing.T, e *Engine) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	counters := e.counters
	if counters.started != counters.inProgress+counters.resultsCommitted {
		t.Fatalf("call accounting does not reconcile: %+v", counters)
	}
	classified := counters.admittedExternal + counters.notAdmittedInvalid + counters.notAdmittedCanceled +
		counters.notAdmittedClosed + counters.pressureShedOptional + counters.sequenceBudgetExhausted
	if counters.resultsCommitted != classified {
		t.Fatalf("result accounting does not reconcile: %+v", counters)
	}
	if counters.admittedExternal != uint64(len(e.queue))+counters.ownerInProgress+counters.completedExternal {
		t.Fatalf("prior-close accounting does not reconcile: queue=%d counters=%+v", len(e.queue), counters)
	}
	if len(e.queue) > e.capacity {
		t.Fatalf("queue occupancy %d exceeds C=%d", len(e.queue), e.capacity)
	}
}

func waitForInProgress(t *testing.T, e *Engine, want uint64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		got := e.counters.inProgress
		e.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("admission calls in progress did not reach %d", want)
}

func mutateIdentity(identity string) string {
	last := "0"
	if identity[len(identity)-1:] == last {
		last = "1"
	}
	return identity[:len(identity)-1] + last
}
