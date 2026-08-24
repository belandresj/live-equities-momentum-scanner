package engine

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const b3Module = "github.com/belandresj/live-equities-momentum-scanner/"

var b3RemovedNames = []string{
	"activityFeatureState", "activityFeatureResult", "evaluateActivityFeatures", "applyActivityResult", "foldActivityAggregate",
	"hodDrawdown", "HODDrawdown", "rolling30", "Rolling30", "rolling60", "Rolling60",
	"cloneQualificationStateForReplay", "latestMarkBeforeReplay", "applyAggregateCandidateLocked",
	"SelectedAggregateView", "selectedAggregateView", "observeSelectedAggregate",
}

var b3ClosedStructFields = map[string]map[string]struct{}{
	"symbolAggregateState": b3NameSet("prefix", "tail", "presence", "sealedLive", "provenAbsent", "historicalConflict", "latest", "olderLatest", "committedLatest", "selectionAt", "priorSelectionAt", "priorSelectionMark", "recomputations", "canonicalRevision", "affected", "tailBoundHits", "lastConflict", "invalidStarts", "priceRange", "mvpMeasurements", "qualification", "tailCoverage", "tailCoverageBuilt", "tailCoverageUsable", "evaluationTailPresence"),
	"Config":               b3NameSet("Mode", "Clock", "Capacity", "RequiredReserve", "EvaluationDelay", "CheckpointSubmitter", "FloatLookup", "RecoveryBackoffInitial", "RecoveryBackoffMaximum"),
	"Engine":               b3NameSet("mu", "mode", "clock", "capacity", "reserve", "delay", "queue", "internalQueued", "changed", "done", "sealed", "exhausted", "lastReserved", "nextSequence", "lastSystem", "lastClock", "hasClock", "state", "counters", "transitions", "publications", "publication", "activeAggregateEvaluation", "sentinels", "lastPubID", "replayFastForwardThrough", "buildCandidate", "beforeConsume", "beforeActiveAggregateEvaluationPhase", "terminal", "publicationFault", "evaluationFault", "evaluationTimingClock", "checkpointSubmitter", "tqLimits", "tqPressurePolicy", "recoveryPolicy", "floatLookup"),
}

var b3ClosedFieldTypes = map[string]map[string]string{
	"symbolAggregateState": {
		"prefix": "aggregatePrefix", "tail": "map[int64]*canonicalAggregate", "presence": "*slotBitmap", "sealedLive": "*slotBitmap",
		"provenAbsent": "*slotBitmap", "historicalConflict": "*slotBitmap", "latest": "*latestAggregateMark", "olderLatest": "*canonicalAggregate",
		"committedLatest": "*committedAggregateMark", "priceRange": "*priceRangeFeatureState", "mvpMeasurements": "*mvpMeasurementState",
		"qualification": "*qualificationState", "tailCoverage": "evaluationTailWindow", "evaluationTailPresence": "*evaluationTailWindow",
	},
	"Config": {
		"Mode": "RunMode", "Clock": "Clock", "Capacity": "int", "RequiredReserve": "int", "EvaluationDelay": "*time.Duration",
		"CheckpointSubmitter": "checkpoint.Submitter", "FloatLookup": "reference.FloatLookup", "RecoveryBackoffInitial": "time.Duration", "RecoveryBackoffMaximum": "time.Duration",
	},
	"Engine": {
		"mode": "RunMode", "state": "*engineState", "publication": "atomic.Pointer[privatePublication]", "activeAggregateEvaluation": "atomic.Pointer[ActiveAggregateEvaluationView]",
		"buildCandidate": "func(frozenBinding) (*installedBinding, error)", "evaluationFault": "bool", "checkpointSubmitter": "checkpoint.Submitter",
	},
}

var b3EvaluationFunctions = b3NameSet(
	"runAggregateFeatureContributorLocked", "commitTrustCorrectionCycleLocked", "selectedTrustClosureCandidateLocked",
	"recordTrustCorrectionTimingLocked",
	"runAggregateEvaluatorLocked", "latchEvaluatorIntegrityLocked", "applyStagedAggregateCandidateLocked",
	"stageAggregateEvaluationLocked", "stageAggregateEvaluationAtLocked", "enrichSelectedRowsLocked", "enrichSelectedRowLocked",
	"validateAggregateEvaluation", "cloneAggregateEvaluation", "aggregateEvaluationEqual", "validTQPublication", "replayEvaluationView",
)

type b3PackageNode struct {
	productionImports  map[string]struct{}
	testImports        map[string]struct{}
	productionFindings []string
	testFindings       []string
}

// TestPLBRB3SourceExclusion parses every named live and retained-tool root,
// follows repository imports in both directions (production and test-
// inclusive), and rejects a reachable removed owner/evaluator/adapter.
func TestPLBRB3SourceExclusion(t *testing.T) {
	root := filepath.Clean("../..")
	liveRoots := []string{
		"cmd/scanner", "cmd/dashboard", "cmd/private-scanner-launcher",
		"internal/engine", "internal/operations", "internal/snapshotapi", "internal/massive",
	}
	toolRoots := []string{
		"cmd/aggregate-replay", "internal/checkpoint", "internal/replay", "internal/replayartifact",
		"internal/replayartifact/playback", "internal/replaymode",
	}
	graph := loadB3PackageGraph(t, root)
	for _, packageRoot := range append(append([]string(nil), liveRoots...), toolRoots...) {
		if graph[packageRoot] == nil {
			t.Errorf("named B3 root missing from graph: %s", packageRoot)
			continue
		}
		assertB3ReachableClean(t, graph, packageRoot, false)
		assertB3ReachableClean(t, graph, packageRoot, true)
	}

	// Retained tools may consume the current engine. The current engine cannot
	// regain a replay evaluator through a reverse dependency.
	if path := b3ImportPath(graph, "internal/replay", "internal/engine", false); len(path) == 0 {
		t.Error("retained replay-to-current-engine dependency was not inspected")
	}
	for _, forbiddenBackEdge := range []string{"internal/replay", "internal/replayartifact", "internal/replaymode"} {
		if path := b3ImportPath(graph, "internal/engine", forbiddenBackEdge, false); len(path) != 0 {
			t.Errorf("current engine imports retained replay tooling: %s", strings.Join(path, " -> "))
		}
	}

	// Checkpoint wire compatibility is the one explicit engine/tool seam. Old
	// payloads must be rejected, never restored into an engine owner.
	checkpointSource, err := os.ReadFile(filepath.Join(root, "internal/engine/checkpoint.go"))
	if err != nil {
		t.Fatal(err)
	}
	checkpointText := string(checkpointSource)
	if !strings.Contains(checkpointText, "legacy activity checkpoint unsupported") ||
		!strings.Contains(checkpointText, "legacy price range checkpoint unsupported") ||
		strings.Contains(checkpointText, "restore"+"Activity(") {
		t.Fatal("checkpoint seam can restore removed evaluator state")
	}
	for _, removed := range []string{
		"internal/engine/feature_activity.go", "internal/engine/feature_activity_test.go",
		"internal/engine/feature_price_range_test.go",
	} {
		if _, err := os.Stat(filepath.Join(root, removed)); !os.IsNotExist(err) {
			t.Errorf("removed source still exists: %s", removed)
		}
	}
}

func TestPLBRB3ExclusionDetectorCounterexamples(t *testing.T) {
	for name, source := range map[string]string{
		"derived owner":       "package p; type someState struct{}; type symbolAggregateState struct { projectionShim *someState }",
		"constructor route":   "package p; type aggregateEvaluationResult struct{}; type Config struct { Route func() aggregateEvaluationResult }",
		"alternate result":    "package p; type aggregateEvaluationResult struct{}; func projectionShim() aggregateEvaluationResult { return aggregateEvaluationResult{} }",
		"publication writer":  "package p; type cell struct{}; func (cell) Store(any){}; type Engine struct{ publication cell }; func (e *Engine) divert(v any){ e.publication.Store(v) }",
		"repurposed selector": "package p; type RunMode string; type Clock func(); type Duration int; type Submitter interface{}; type Lookup interface{}; type Config struct { Mode func(); Clock Clock; Capacity int; RequiredReserve int; EvaluationDelay *Duration; CheckpointSubmitter Submitter; FloatLookup Lookup; RecoveryBackoffInitial Duration; RecoveryBackoffMaximum Duration }",
	} {
		findings, _ := inspectB3GoSource(name+".go", []byte(source))
		structural := false
		for _, finding := range findings {
			structural = structural || strings.Contains(finding, "structural")
		}
		if !structural {
			t.Errorf("%s counterexample escaped detector", name)
		}
	}
	graph := map[string]*b3PackageNode{
		"live":    {productionImports: map[string]struct{}{"adapter": {}}},
		"adapter": {productionImports: map[string]struct{}{"removed": {}}},
		"removed": {productionImports: map[string]struct{}{}, productionFindings: []string{"removed owner"}},
	}
	if path := b3ImportPath(graph, "live", "removed", false); strings.Join(path, " -> ") != "live -> adapter -> removed" {
		t.Fatalf("transitive removed-owner path not found: %v", path)
	}
}

func loadB3PackageGraph(t *testing.T, root string) map[string]*b3PackageNode {
	t.Helper()
	graph := make(map[string]*b3PackageNode)
	for _, top := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			packagePath := filepath.ToSlash(filepath.Dir(rel))
			node := graph[packagePath]
			if node == nil {
				node = &b3PackageNode{productionImports: make(map[string]struct{}), testImports: make(map[string]struct{})}
				graph[packagePath] = node
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			findings, imports := inspectB3GoSource(rel, contents)
			if filepath.Base(path) != "lbr_b3_source_exclusion_test.go" {
				if strings.HasSuffix(path, "_test.go") {
					node.testFindings = append(node.testFindings, findings...)
				} else {
					node.productionFindings = append(node.productionFindings, findings...)
				}
			}
			destination := node.productionImports
			if strings.HasSuffix(path, "_test.go") {
				destination = node.testImports
			}
			for _, imported := range imports {
				if strings.HasPrefix(imported, b3Module) {
					destination[strings.TrimPrefix(imported, b3Module)] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return graph
}

func inspectB3GoSource(path string, contents []byte) ([]string, []string) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
	if err != nil {
		return []string{"unparseable Go source: " + err.Error()}, nil
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		if value, err := strconv.Unquote(spec.Path.Value); err == nil {
			imports = append(imports, value)
		}
	}
	text := string(contents)
	findings := make([]string, 0)
	for _, removed := range b3RemovedNames {
		if strings.Contains(text, removed) {
			findings = append(findings, path+": removed symbol "+removed)
		}
	}
	full, err := parser.ParseFile(token.NewFileSet(), path, contents, 0)
	if err != nil {
		return append(findings, path+": unparseable declarations"), imports
	}
	ast.Inspect(full, func(node ast.Node) bool {
		var value string
		switch current := node.(type) {
		case *ast.Ident:
			value = current.Name
		case *ast.BasicLit:
			if current.Kind == token.STRING {
				value, _ = strconv.Unquote(current.Value)
			}
		}
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(value))
		if strings.Contains(normalized, "evaluator") &&
			(strings.Contains(normalized, "legacy") || strings.Contains(normalized, "fallback") || strings.Contains(normalized, "shadow") || strings.Contains(normalized, "compat")) {
			findings = append(findings, path+": runtime/owner selector "+value)
		}
		return true
	})
	findings = append(findings, inspectB3Structure(path, full, strings.HasSuffix(path, "_test.go"))...)
	return findings, imports
}

func inspectB3Structure(path string, file *ast.File, testFile bool) []string {
	findings := make([]string, 0)
	for _, declaration := range file.Decls {
		switch current := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range current.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				want, closed := b3ClosedStructFields[typeSpec.Name.Name]
				if (typeSpec.Name.Name == "Config" || typeSpec.Name.Name == "Engine") &&
					strings.Contains(path, "/") && !strings.HasPrefix(filepath.ToSlash(path), "internal/engine/") {
					closed = false
				}
				if !closed {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					findings = append(findings, path+": structural owner is not a struct: "+typeSpec.Name.Name)
					continue
				}
				got := make(map[string]struct{})
				gotTypes := make(map[string]string)
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						got[name.Name] = struct{}{}
						gotTypes[name.Name] = b3TypeString(field.Type)
					}
				}
				if !b3SameNames(got, want) {
					findings = append(findings, path+": structural owner/config shape changed: "+typeSpec.Name.Name)
				}
				for field, expectedType := range b3ClosedFieldTypes[typeSpec.Name.Name] {
					if gotTypes[field] != expectedType {
						findings = append(findings, path+": structural owner/config field type changed: "+typeSpec.Name.Name+"."+field)
					}
				}
			}
		case *ast.FuncDecl:
			name := current.Name.Name
			if !testFile && b3NodeReferencesName(current.Type, "aggregateEvaluationResult") {
				if _, allowed := b3EvaluationFunctions[name]; !allowed {
					findings = append(findings, path+": structural alternate evaluation entrypoint: "+name)
				}
			}
			calls := b3FunctionCalls(current)
			if testFile {
				continue
			}
			for _, called := range calls {
				switch called {
				case "runAggregateFeatureContributorLocked", "runAggregateEvaluatorLocked":
					if name != "transition" && name != "applyHydrationChunkLocked" {
						findings = append(findings, path+": structural alternate live evaluation caller: "+name+" -> "+called)
					}
				case "storePublication":
					if name != "installInitialPublication" && name != "completePublicationDecisionLocked" {
						findings = append(findings, path+": structural alternate publication selector: "+name)
					}
				case "buildPublicationLocked":
					if name != "completePublicationDecisionLocked" && name != "buildSuppressedPublicationLocked" {
						findings = append(findings, path+": structural alternate publication builder: "+name)
					}
				case "publication.Store":
					if name != "storePublication" {
						findings = append(findings, path+": structural direct publication writer: "+name)
					}
				}
			}
			if name == "transition" {
				contributor, evaluator := b3CallIndex(calls, "runAggregateFeatureContributorLocked"), b3CallIndex(calls, "runAggregateEvaluatorLocked")
				if contributor < 0 || evaluator != contributor+1 {
					findings = append(findings, path+": structural live evaluator order changed")
				}
			}
			if name == "consume" && b3CallIndex(calls, "transition") < 0 {
				findings = append(findings, path+": structural live owner bypasses transition")
			}
		}
	}
	return findings
}

func b3FunctionCalls(function *ast.FuncDecl) []string {
	if function.Body == nil {
		return nil
	}
	calls := make([]string, 0)
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch target := call.Fun.(type) {
		case *ast.Ident:
			calls = append(calls, target.Name)
		case *ast.SelectorExpr:
			if inner, ok := target.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "publication" {
				calls = append(calls, "publication."+target.Sel.Name)
			} else {
				calls = append(calls, target.Sel.Name)
			}
		}
		return true
	})
	return calls
}

func b3NodeReferencesName(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		if identifier, ok := current.(*ast.Ident); ok && identifier.Name == name {
			found = true
			return false
		}
		return !found
	})
	return found
}

func b3NameSet(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func b3SameNames(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for name := range left {
		if _, ok := right[name]; !ok {
			return false
		}
	}
	return true
}

func b3CallIndex(calls []string, wanted string) int {
	for index, call := range calls {
		if call == wanted {
			return index
		}
	}
	return -1
}

func b3TypeString(expression ast.Expr) string {
	var output bytes.Buffer
	if err := printer.Fprint(&output, token.NewFileSet(), expression); err != nil {
		return ""
	}
	return output.String()
}

func assertB3ReachableClean(t *testing.T, graph map[string]*b3PackageNode, root string, includeTests bool) {
	t.Helper()
	visited := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		node := graph[current]
		if node == nil {
			continue
		}
		for _, finding := range node.productionFindings {
			t.Errorf("%s reachable from %s (tests=%t): %s", current, root, includeTests, finding)
		}
		if includeTests {
			for _, finding := range node.testFindings {
				t.Errorf("%s test oracle reachable from %s: %s", current, root, finding)
			}
		}
		imports := []map[string]struct{}{node.productionImports}
		if includeTests {
			imports = append(imports, node.testImports)
		}
		for _, set := range imports {
			for imported := range set {
				if graph[imported] != nil && !visited[imported] {
					visited[imported] = true
					queue = append(queue, imported)
				}
			}
		}
	}
}

func b3ImportPath(graph map[string]*b3PackageNode, from, to string, includeTests bool) []string {
	previous := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current == to {
			path := []string{current}
			for previous[current] != "" {
				current = previous[current]
				path = append(path, current)
			}
			for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
				path[left], path[right] = path[right], path[left]
			}
			return path
		}
		node := graph[current]
		if node == nil {
			continue
		}
		sets := []map[string]struct{}{node.productionImports}
		if includeTests {
			sets = append(sets, node.testImports)
		}
		for _, set := range sets {
			for imported := range set {
				if graph[imported] != nil {
					if _, seen := previous[imported]; !seen {
						previous[imported] = current
						queue = append(queue, imported)
					}
				}
			}
		}
	}
	return nil
}
