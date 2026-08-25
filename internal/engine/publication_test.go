package engine

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestENGINPUT01ClosedCommonCrossFamilyValidation(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	now := start.Add(time.Minute)
	e := testEngine(t, now, 8, 2)

	initial := e.observePublication()
	if initial.kind != publicationInitial || initial.bindingIdentity != "" || initial.watermark != nil || initial.publicationID != 0 {
		t.Fatalf("initial publication = %+v", initial)
	}
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}
	assertPublication(t, e, publicationNormal, 1, DispositionBindingInstalled)

	before := e.observePublication()
	if got := awaitDisposition(t, admitValidBinding(t, e, binding)); got.Code != DispositionBindingAlreadyInstalled {
		t.Fatalf("repeated binding = %+v", got)
	}
	assertSamePublication(t, before, e.observePublication())

	badSchema := liveAggregate(binding, "AAA", start, 1, 1)
	badSchema.SchemaVersion = "normalized-second-aggregate-v2"
	applyAggregate(t, e, badSchema, DispositionAggregateRejected, ReasonSchema)
	assertSamePublication(t, before, e.observePublication())

	insert := liveAggregate(binding, "AAA", start, 1, 1)
	applyAggregate(t, e, insert, DispositionAggregateInserted, ReasonNone)
	inserted := e.observePublication()
	assertSamePublication(t, before, inserted)
	applyAggregate(t, e, insert, DispositionAggregateExactDuplicate, ReasonNone)
	assertSamePublication(t, inserted, e.observePublication())

	stale := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2)
	stale.BindingIdentity = mutateIdentity(binding.Identity())
	applyAggregate(t, e, stale, DispositionAggregateFenced, ReasonBinding)
	assertSamePublication(t, inserted, e.observePublication())

	badWindow := liveAggregate(binding, "AAA", start.Add(2*time.Second), 1, 3)
	badWindow.WindowEnd = badWindow.WindowEnd.Add(time.Second)
	applyAggregate(t, e, badWindow, DispositionAggregateRejected, ReasonWindow)
	assertSamePublication(t, inserted, e.observePublication())

	_, unsupported := admitUnsupportedSchemaForProof(e)
	if got := awaitDisposition(t, unsupported); got.Code != DispositionUnsupportedSchema || got.Reason != ReasonSchema {
		t.Fatalf("unsupported schema = %+v", got)
	}
	_, illegal := admitIllegalForProof(e)
	if got := awaitDisposition(t, illegal); got.Code != DispositionIllegalLifecycle || got.Reason != ReasonLifecycle {
		t.Fatalf("illegal kind = %+v", got)
	}
	for _, kind := range []inputKind{0, 255} {
		_, unknown := admitKindForProof(e, kind)
		if got := awaitDisposition(t, unknown); got.Code != DispositionIllegalLifecycle || got.Reason != ReasonLifecycle {
			t.Fatalf("unknown kind %d became opaque success: %+v", kind, got)
		}
	}

	_, timer := e.AdmitTimer(context.Background())
	if got := awaitTimerDisposition(t, timer); got.Code != DispositionTimerApplied {
		t.Fatalf("timer = %+v", got)
	}
	if e.observeTimeLifecycle().CommittedT != nil {
		t.Fatal("Component 2 timer advanced T")
	}

	result, stopped := e.Stop(context.Background())
	if result != AdmissionAdmitted || awaitDisposition(t, stopped).Code != DispositionControlApplied {
		t.Fatalf("stop = %s", result)
	}
	if final := e.observePublication(); final.kind != publicationNormal || final.lifecycle != lifecycleEnded || final.watermark != nil {
		t.Fatalf("terminal publication = %+v", final)
	}
	if err := e.Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	assertCompletedAccounting(t, e)
}

func TestENGPUBLISH01AtomicImmutablePublicationClock(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()
	clock := &controlledClock{now: start.Add(time.Minute)}
	e := newS3Engine(t, clock.read, 8, 2, 0)

	if initial := e.observePublication(); initial.kind != publicationInitial || initial.publicationID != 0 || initial.currentMarketClaim {
		t.Fatalf("initial = %+v", initial)
	}
	reads := clock.count()
	completion := admitValidBinding(t, e, binding)
	if got := awaitDisposition(t, completion); got.Code != DispositionBindingInstalled {
		t.Fatalf("binding = %+v", got)
	}
	first := e.observePublication()
	if first.publicationID != 1 || first.generatedAt != start.Add(time.Minute) || clock.count() != reads+2 {
		t.Fatalf("first publication/clock = %+v reads=%d/%d", first, clock.count(), reads+2)
	}
	if first.lastEngineSequence != 1 || first.watermark != nil || first.currentMarketClaim ||
		first.aggregateEvaluation.mode != rankingUnavailable || first.aggregateEvaluation.reason != rankingReasonNoCommittedWatermark ||
		!first.aggregateEvaluation.at.IsZero() {
		t.Fatalf("first publication claims = %+v", first)
	}

	t.Run("reader sees old cell and no completion during publication clock stage", func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		var mu sync.Mutex
		reads := 0
		blockingClock := func() time.Time {
			mu.Lock()
			reads++
			read := reads
			mu.Unlock()
			if read == 2 {
				close(entered)
				<-release
			}
			return start.Add(time.Minute)
		}
		staged := newS3Engine(t, blockingClock, 3, 1, 0)
		result, completion := staged.AdmitBinding(context.Background(), validBindingInput(binding))
		if result != AdmissionAdmitted {
			t.Fatalf("staged admission = %s", result)
		}
		<-entered
		if view := staged.observePublication(); view.kind != publicationInitial {
			t.Fatalf("reader saw partial candidate: %+v", view)
		}
		select {
		case got := <-completion:
			t.Fatalf("completion preceded publication decision: %+v", got)
		default:
		}
		close(release)
		if got := awaitDisposition(t, completion); got.Code != DispositionBindingInstalled || staged.observePublication().kind != publicationNormal {
			t.Fatalf("staged publication = %+v view=%+v", got, staged.observePublication())
		}
		closeAndWait(t, staged)
	})

	aggregate := liveAggregate(binding, "AAA", start, 1, 1)
	applyAggregate(t, e, aggregate, DispositionAggregateInserted, ReasonNone)
	second := e.observePublication()
	assertSamePublication(t, first, second)
	reads = clock.count()
	applyAggregate(t, e, aggregate, DispositionAggregateExactDuplicate, ReasonNone)
	assertSamePublication(t, second, e.observePublication())
	if clock.count() != reads+1 { // admission only; no publication clock/ID for no change
		t.Fatalf("duplicate clock reads = %d, want %d", clock.count(), reads+1)
	}

	correction := changedClose(aggregate, 10.5)
	correction.Live.FrameSequence = 2
	applyAggregate(t, e, correction, DispositionAggregateRevised, ReasonNone)
	third := e.observePublication()
	assertSamePublication(t, first, third)

	// Readers race only with complete atomic replacements. Value-only views
	// cannot alias canonical maps; mutating a returned copy changes no cell.
	mutated := e.observePublication()
	mutated.bindingIdentity = "reader mutation"
	if current := e.observePublication(); current.bindingIdentity != binding.Identity() {
		t.Fatalf("reader mutation reached cell: %+v", current)
	}
	var wg sync.WaitGroup
	seenBad := make(chan publicationView, 1)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 500 {
				view := e.observePublication()
				if view.kind != publicationNormal || view.publicationID != 1 || view.schemaVersion != privatePublicationSchemaV1 {
					select {
					case seenBad <- view:
					default:
					}
					return
				}
			}
		}()
	}
	clock.set(start.Add(2 * time.Minute))
	next := changedClose(correction, 10.75)
	next.Live.FrameSequence = 3
	applyAggregate(t, e, next, DispositionAggregateRevised, ReasonNone)
	wg.Wait()
	select {
	case bad := <-seenBad:
		t.Fatalf("partial reader view = %+v", bad)
	default:
	}

	t.Run("builder and semantic validation fail closed", func(t *testing.T) {
		for _, fault := range []publicationFault{publicationFaultBuild, publicationFaultValidation} {
			clock := &controlledClock{now: start.Add(time.Minute)}
			failed := newS3Engine(t, clock.read, 4, 1, 0)
			failed.publicationFault = fault
			result, completion := failed.AdmitBinding(context.Background(), validBindingInput(binding))
			if result != AdmissionAdmitted {
				t.Fatalf("fault admission = %s", result)
			}
			got := awaitDisposition(t, completion)
			if got.Code != DispositionPublicationIntegrity || got.Reason != ReasonPublication {
				t.Fatalf("fault %d disposition = %+v", fault, got)
			}
			view := failed.observePublication()
			if view.kind != publicationUnavailableSentinel || view.publicationID != 0 || view.bindingIdentity != "" || view.watermark != nil || view.currentMarketClaim ||
				view.lifecycleReason != lifecycleReasonPublicationIntegrity || view.suppressionDisposition != SuppressionRestartRequired ||
				view.lastDisposition != DispositionPublicationIntegrity || view.dispositionReason != ReasonPublication {
				t.Fatalf("fault %d sentinel = %+v", fault, view)
			}
			if err := failed.Wait(testContext(t)); err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("publication ID exhaustion does not wrap", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		exhausted := newS3Engine(t, clock.read, 4, 1, 0)
		awaitDisposition(t, admitValidBinding(t, exhausted, binding))
		exhausted.mu.Lock()
		exhausted.lastPubID = math.MaxUint64
		exhausted.mu.Unlock()
		fact := controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, start.Add(time.Minute), 1, ControlSucceeded)
		_, completion := exhausted.AdmitConnectionControl(context.Background(), fact)
		got := <-completion
		if got.Code != DispositionPublicationIntegrity || got.SuppressionDisposition != SuppressionRestartRequired || exhausted.observePublication().kind != publicationUnavailableSentinel ||
			exhausted.observePublication().lifecycleReason != lifecycleReasonPublicationIntegrity {
			t.Fatalf("publication exhaustion = %+v view=%+v", got, exhausted.observePublication())
		}
		if exhausted.lastPubID != math.MaxUint64 {
			t.Fatal("publication ID wrapped or was consumed")
		}
		if err := exhausted.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("publication clock does not retroactively invalidate linked admissions", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		backlog := newS3Engine(t, clock.read, 5, 1, 0)
		awaitDisposition(t, admitValidBinding(t, backlog, binding))
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		backlog.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		clock.set(start.Add(2 * time.Minute))
		firstFact := controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, start.Add(2*time.Minute), 1, ControlSucceeded)
		_, firstCompletion := backlog.AdmitConnectionControl(context.Background(), firstFact)
		<-entered
		clock.set(start.Add(3 * time.Minute))
		secondFact := controlFact(binding.Identity(), AggregateCommandWriteResult, 1, LivePosition{}, start.Add(3*time.Minute), 2, ControlSucceeded)
		_, secondCompletion := backlog.AdmitConnectionControl(context.Background(), secondFact)
		clock.set(start.Add(4 * time.Minute))
		close(release)
		firstResult := <-firstCompletion
		secondResult := <-secondCompletion
		if firstResult.Code != DispositionConnectionControlApplied || secondResult.Code != DispositionConnectionControlApplied {
			t.Fatalf("linked backlog invalidated by later publication read: %+v %+v", firstResult, secondResult)
		}
		if view := backlog.observePublication(); view.generatedAt != start.Add(4*time.Minute) || view.publicationID != 3 {
			t.Fatalf("backlog publication = %+v", view)
		}
		closeAndWait(t, backlog)
	})

	t.Run("publication-time clock regression fails closed", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		regressed := newS3Engine(t, clock.read, 4, 1, 0)
		awaitDisposition(t, admitValidBinding(t, regressed, binding))
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		regressed.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		clock.set(start.Add(3 * time.Minute))
		fact := controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, start.Add(3*time.Minute), 1, ControlSucceeded)
		_, completion := regressed.AdmitConnectionControl(context.Background(), fact)
		<-entered
		clock.set(start.Add(2 * time.Minute))
		close(release)
		got := <-completion
		if got.Code != DispositionClockRegression || got.SuppressionDisposition != SuppressionRestartRequired || regressed.observePublication().kind != publicationUnavailableSentinel ||
			regressed.observePublication().lifecycleReason != lifecycleReasonClockRegression || regressed.observePublication().suppressionDisposition != SuppressionRestartRequired {
			t.Fatalf("publication clock regression = %+v view=%+v", got, regressed.observePublication())
		}
		closeAndWait(t, regressed)
	})

	closeAndWait(t, e)
}

func TestENGMODULE01CompiledExtensionConstruction(t *testing.T) {
	directory := engineSourceDirectory(t)
	files, err := filepath.Glob(filepath.Join(directory, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"RegisterContributor", "ContributorRegistry", "GenericReducer", "EventBus", "PublishSnapshot", "time.Now("}
	var foundEngine, foundPublicationCell, foundDecisionOwner bool
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, phrase := range forbidden {
			if strings.Contains(string(contents), phrase) {
				t.Fatalf("forbidden construction %q in %s", phrase, filepath.Base(file))
			}
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, contents, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			switch value := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range value.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					if typeSpec.Name.Name == "Engine" {
						foundEngine = true
					}
					for _, field := range structure.Fields.List {
						for _, name := range field.Names {
							if typeSpec.Name.Name == "Engine" && name.Name == "publication" {
								foundPublicationCell = true
							}
							if typeSpec.Name.Name == "canonicalAggregate" && name.Name == "proofResult" {
								t.Fatal("canonical state retains the mutable historical proof result")
							}
						}
					}
				}
			case *ast.FuncDecl:
				if value.Name.Name == "completePublicationDecisionLocked" {
					foundDecisionOwner = true
				}
				ast.Inspect(value.Body, func(node ast.Node) bool {
					switch expression := node.(type) {
					case *ast.GoStmt:
						call, ok := expression.Call.Fun.(*ast.SelectorExpr)
						if !ok || call.Sel.Name != "consume" {
							t.Fatalf("independent goroutine owner in %s.%s", filepath.Base(file), value.Name.Name)
						}
					case *ast.CallExpr:
						if selector, ok := expression.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Store" && value.Name.Name != "storePublication" {
							t.Fatalf("atomic publication bypass in %s.%s", filepath.Base(file), value.Name.Name)
						}
					}
					return true
				})
			}
		}
	}
	if !foundEngine || !foundPublicationCell || !foundDecisionOwner {
		t.Fatalf("missing engine ownership boundary: engine=%v publication=%v decision=%v", foundEngine, foundPublicationCell, foundDecisionOwner)
	}
	engineType := reflect.TypeOf((*Engine)(nil))
	for index := 0; index < engineType.NumMethod(); index++ {
		name := engineType.Method(index).Name
		if strings.Contains(name, "Publish") || strings.Contains(name, "Register") || strings.Contains(name, "Apply") || strings.Contains(name, "Validate") || strings.Contains(name, "State") {
			t.Fatalf("exported mutation/registration seam %s", name)
		}
	}
	for _, owner := range []reflect.Type{reflect.TypeOf(canonicalAggregate{}), reflect.TypeOf(frozenHistoricalProofContext{})} {
		for index := 0; index < owner.NumField(); index++ {
			field := owner.Field(index)
			switch field.Type.Kind() {
			case reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
				t.Fatalf("mutable retained alias %s.%s has type %s", owner.Name(), field.Name, field.Type)
			}
			if field.Type.Kind() == reflect.Pointer && field.Type.Elem() == reflect.TypeOf(historicalProofResult{}) {
				t.Fatalf("historical proof result retained by %s.%s", owner.Name(), field.Name)
			}
		}
	}
}

func TestENGFAIL01CrossPathContainmentTerminalDrain(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("publication failure drains captured FIFO and removes stale success", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		e := newS3Engine(t, clock.read, 4, 1, 0)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		if e.observePublication().kind != publicationNormal {
			t.Fatal("binding did not establish prior successful publication")
		}
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		e.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		e.publicationFault = publicationFaultBuild
		fact := controlFact(binding.Identity(), ConnectionAttempt, 1, LivePosition{}, start.Add(time.Minute), 1, ControlSucceeded)
		_, trigger := e.AdmitConnectionControl(context.Background(), fact)
		<-entered
		_, behind := e.AdmitTimer(context.Background())
		close(release)
		if got := <-trigger; got.Code != DispositionPublicationIntegrity {
			t.Fatalf("trigger = %+v", got)
		}
		if got := awaitTimerDisposition(t, behind); got.Code != DispositionTerminal {
			t.Fatalf("drained node = %+v", got)
		}
		if view := e.observePublication(); view.kind != publicationUnavailableSentinel || view.bindingIdentity != "" || view.currentMarketClaim {
			t.Fatalf("stale success remained visible: %+v", view)
		}
		if result, completion := e.AdmitTimer(context.Background()); result != AdmissionNotAdmittedClosed || completion != nil {
			t.Fatalf("post-failure admission = %s/%v", result, completion)
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		assertCompletedAccounting(t, e)
	})

	t.Run("local rejection preserves canonical state", func(t *testing.T) {
		e := testEngine(t, start.Add(time.Minute), 5, 1)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		valid := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, e, valid, DispositionAggregateInserted, ReasonNone)
		before := aggregateRecord(t, e, "AAA", start)
		invalid := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2)
		invalid.Values.High = 0
		applyAggregate(t, e, invalid, DispositionAggregateRejected, ReasonStructural)
		after := aggregateRecord(t, e, "AAA", start)
		if !aggregateValuesEqual(before.values, after.values) {
			t.Fatal("local rejection changed unrelated canonical state")
		}
		closeAndWait(t, e)
	})

	t.Run("canonical contradiction closes admission and drains the captured FIFO", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		e := newS3Engine(t, clock.read, 5, 1, 0)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		base := liveAggregate(binding, "AAA", start, 1, 1)
		applyAggregate(t, e, base, DispositionAggregateInserted, ReasonNone)
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		e.beforeConsume = func(*queueNode) { once.Do(func() { close(entered); <-release }) }
		contradiction := changedClose(base, base.Values.Close+1)
		_, trigger := e.admitAggregateForProof(contradiction)
		<-entered
		_, captured := e.AdmitTimer(context.Background())
		close(release)
		got := awaitAggregateDisposition(t, trigger)
		if got.Code != DispositionAggregateIntegrity || got.SuppressionDisposition != SuppressionCleanReinitializationRequired {
			t.Fatalf("canonical trigger = %+v", got)
		}
		if got := awaitTimerDisposition(t, captured); got.Code != DispositionTerminal || got.SuppressionDisposition != SuppressionCleanReinitializationRequired {
			t.Fatalf("captured FIFO disposition = %+v", got)
		}
		view := e.observePublication()
		if view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonCanonicalIntegrity ||
			view.suppressionDisposition != SuppressionCleanReinitializationRequired || view.lastDisposition != DispositionAggregateIntegrity ||
			view.dispositionReason != ReasonRepeatedPositionUnequal {
			t.Fatalf("canonical sentinel = %+v", view)
		}
		if result, completion := e.AdmitTimer(context.Background()); result != AdmissionNotAdmittedClosed || completion != nil {
			t.Fatalf("post-canonical admission = %s/%v", result, completion)
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		assertCompletedAccounting(t, e)
	})

	t.Run("clock and canonical integrity install unavailable sentinel without restoration", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		e := newS3Engine(t, clock.read, 5, 1, 0)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		clock.set(start)
		_, regressed := e.AdmitTimer(context.Background())
		if got := awaitTimerDisposition(t, regressed); got.Code != DispositionClockRegression || got.SuppressionDisposition != SuppressionRestartRequired {
			t.Fatalf("clock regression = %+v", got)
		}
		if view := e.observePublication(); view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonClockRegression || view.suppressionDisposition != SuppressionRestartRequired || view.lastDisposition != DispositionClockRegression || view.dispositionReason != ReasonClockRegression {
			t.Fatal("clock failure left a current publication")
		}
		clock.set(start.Add(2 * time.Minute))
		if result, later := e.AdmitTimer(context.Background()); result != AdmissionNotAdmittedClosed || later != nil {
			t.Fatalf("ordinary timer admitted after restart-required suppression: %s/%v", result, later)
		}
		closeAndWait(t, e)
	})

	t.Run("accounting contradiction fails closed without counter repair", func(t *testing.T) {
		e := testEngine(t, start.Add(time.Minute), 4, 1)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		e.mu.Lock()
		e.transitions.rejected++ // model detectable same-package memory corruption
		e.mu.Unlock()
		_, timer := e.AdmitTimer(context.Background())
		got := awaitTimerDisposition(t, timer)
		if got.Code != DispositionAccountingIntegrity || got.Reason != ReasonAccounting || got.SuppressionDisposition != SuppressionRestartRequired ||
			e.observePublication().kind != publicationUnavailableSentinel || e.observePublication().lifecycleReason != lifecycleReasonAccountingIntegrity || e.observePublication().suppressionDisposition != SuppressionRestartRequired {
			t.Fatalf("accounting containment = %+v view=%+v", got, e.observePublication())
		}
		e.mu.Lock()
		stillContradictory := !e.transitions.reconciles()
		e.mu.Unlock()
		if !stillContradictory {
			t.Fatal("accounting counters were adjusted to manufacture reconciliation")
		}
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("internal close uses common accounting containment", func(t *testing.T) {
		e := testEngine(t, start.Add(time.Minute), 4, 1)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		e.mu.Lock()
		e.transitions.rejected++
		e.mu.Unlock()
		e.Close()
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		e.mu.Lock()
		globalFailure := e.state.globalFailure
		e.mu.Unlock()
		if view := e.observePublication(); !globalFailure || view.kind != publicationUnavailableSentinel || view.lifecycleReason != lifecycleReasonAccountingIntegrity || view.suppressionDisposition != SuppressionRestartRequired {
			t.Fatalf("internal accounting false success: failure=%v view=%+v", globalFailure, e.observePublication())
		}
	})

	t.Run("internal publication clock regression uses clock containment", func(t *testing.T) {
		clock := &controlledClock{now: start.Add(time.Minute)}
		e := newS3Engine(t, clock.read, 4, 1, 0)
		awaitDisposition(t, admitValidBinding(t, e, binding))
		clock.set(start)
		e.Close()
		if err := e.Wait(testContext(t)); err != nil {
			t.Fatal(err)
		}
		if obs := e.observeTimeLifecycle(); obs.ClockMonotonic || obs.SuppressionDisposition != SuppressionRestartRequired || e.observePublication().kind != publicationUnavailableSentinel || e.observePublication().lifecycleReason != lifecycleReasonClockRegression {
			t.Fatalf("internal clock false success: %+v view=%+v", obs, e.observePublication())
		}
	})
}

func TestENGOBS01CompletedAccountingCardinality(t *testing.T) {
	binding := testBinding(t)
	start := binding.SessionStart()

	t.Run("dequeue is not completion", func(t *testing.T) {
		paused := testEngine(t, start.Add(time.Minute), 3, 1)
		entered, release := make(chan struct{}), make(chan struct{})
		paused.beforeConsume = func(*queueNode) { close(entered); <-release }
		_, completion := paused.AdmitBinding(context.Background(), validBindingInput(binding))
		<-entered
		paused.mu.Lock()
		admission := paused.counters
		transitions := paused.transitions
		publications := paused.publications
		paused.mu.Unlock()
		if admission.ownerInProgress != 1 || admission.completedExternal != 0 || transitions.completedExternal != 0 || publications.completedDecisions != 0 || paused.observePublication().kind != publicationInitial {
			t.Fatalf("dequeue advanced completion: admission=%+v transitions=%+v publications=%+v view=%+v", admission, transitions, publications, paused.observePublication())
		}
		close(release)
		if got := awaitDisposition(t, completion); got.Code != DispositionBindingInstalled {
			t.Fatalf("paused completion = %+v", got)
		}
		assertCompletedAccounting(t, paused)
		closeAndWait(t, paused)
	})

	e := testEngine(t, start.Add(time.Minute), 6, 2)
	awaitDisposition(t, admitValidBinding(t, e, binding))
	base := liveAggregate(binding, "AAA", start, 1, 1)
	applyAggregate(t, e, base, DispositionAggregateInserted, ReasonNone)
	for range 250 {
		applyAggregate(t, e, base, DispositionAggregateExactDuplicate, ReasonNone)
	}
	rejected := liveAggregate(binding, "AAA", start.Add(time.Second), 1, 2)
	rejected.Values.Low = rejected.Values.High + 1
	applyAggregate(t, e, rejected, DispositionAggregateRejected, ReasonStructural)
	fenced := liveAggregate(binding, "AAA", start.Add(2*time.Second), 1, 3)
	fenced.BindingIdentity = mutateIdentity(binding.Identity())
	applyAggregate(t, e, fenced, DispositionAggregateFenced, ReasonBinding)
	_, illegal := admitIllegalForProof(e)
	awaitDisposition(t, illegal)
	assertCompletedAccounting(t, e)

	e.mu.Lock()
	if len(e.queue) > e.capacity || e.state.aggregates.consumed != 253 || e.lastPubID != 1 {
		t.Fatalf("bounded trace queue=%d aggregates=%+v publicationID=%d", len(e.queue), e.state.aggregates, e.lastPubID)
	}
	if len(e.state.binding.symbols) != binding.UniverseAccounting().EligibleRecords {
		t.Fatal("canonical symbol cardinality changed")
	}
	e.mu.Unlock()
	closeAndWait(t, e)
	assertCompletedAccounting(t, e)
}

func admitUnsupportedSchemaForProof(e *Engine) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: inputUnsupportedSchema}, false)
}

func admitKindForProof(e *Engine, kind inputKind) (AdmissionResult, <-chan Disposition) {
	e.beginAdmission()
	return e.admit(context.Background(), &queueNode{kind: kind}, false)
}

func assertPublication(t *testing.T, e *Engine, kind publicationKind, id uint64, disposition DispositionCode) {
	t.Helper()
	view := e.observePublication()
	if view.kind != kind || view.publicationID != id || view.lastDisposition != disposition || view.schemaVersion != privatePublicationSchemaV1 {
		t.Fatalf("publication = %+v", view)
	}
}

func assertSamePublication(t *testing.T, before, after publicationView) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("publication changed: before=%+v after=%+v", before, after)
	}
}

func assertCompletedAccounting(t *testing.T, e *Engine) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.transitions.reconciles() || !e.publications.reconciles(e.transitions.completedExternal+e.transitions.completedInternal) ||
		e.transitions.completedExternal != e.counters.completedExternal || !prospectiveAccountingCoherent(e, e.counters, e.transitions, e.publications) {
		t.Fatalf("completed accounting admission=%+v transitions=%+v publications=%+v aggregates=%+v", e.counters, e.transitions, e.publications, e.state.aggregates)
	}
}

func engineSourceDirectory(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate engine source")
	}
	return filepath.Dir(file)
}
