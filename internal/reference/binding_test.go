package reference

import (
	"slices"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func TestCompleteBindingIdentityAndImmutability(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"aaa", "AAA", "MISSING"})
	priorFacts := []PriorCloseFact{
		{symbol: "AAA", status: PriorCloseValid, close: 10.25},
		{symbol: "MISSING", status: PriorCloseMissing},
		{symbol: "aaa", status: PriorCloseInvalid, reason: PriorCloseWrongDate},
	}
	priorIdentity, err := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, priorFacts)
	if err != nil {
		t.Fatal(err)
	}
	accounting := accountPriorCloses(priorFacts, 0)
	priors := PriorCloses{
		priorSessionDate: facts.PriorSessionDate, policyVersion: PriorClosePolicyVersion,
		identity: priorIdentity, facts: slices.Clone(priorFacts), accounting: accounting, source: SourceFresh,
	}
	binding, err := AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	if binding.TradingDate() != facts.TradingDate || !binding.SessionStart().Equal(facts.SessionStart) ||
		!binding.SessionEnd().Equal(facts.SessionEnd) || binding.ScheduleSchema() != facts.ScheduleSchema ||
		binding.ScheduleVersion() != facts.ScheduleVersion || binding.ScheduleArtifactSHA256() != facts.ScheduleArtifactSHA256 ||
		binding.PriorSessionDate() != facts.PriorSessionDate || !binding.PriorRegularClose().Equal(facts.PriorRegularClose) ||
		binding.UniversePolicy() != EligibilityPolicyVersion || binding.UniverseIdentity() != universe.Identity() ||
		binding.PriorClosePolicy() != PriorClosePolicyVersion || binding.PriorCloseIdentity() != priorIdentity ||
		!binding.PriorCloseAdjusted() || binding.PriorCloseIncludeOTC() || binding.PriorCloseLocale() != "us" || binding.PriorCloseMarket() != "stocks" {
		t.Fatalf("incomplete binding = %+v", binding)
	}
	if !slices.Equal(binding.UniverseSymbols(), universe.Symbols()) || !slices.Equal(binding.PriorCloseFacts(), priors.Facts()) {
		t.Fatal("binding population or prior-close facts changed")
	}
	if binding.PriorCloseAccounting() != accounting || binding.UniverseAccounting() != universe.Accounting() {
		t.Fatal("binding accounting changed")
	}
	if binding.Identity() == "" || binding.Identity()[:len(bindingIdentitySchema)+1] != bindingIdentitySchema+":" {
		t.Fatalf("binding identity = %q", binding.Identity())
	}

	cacheUniverse := universe
	cacheUniverse.source = SourceCurrentCache
	cachePriors := priors
	cachePriors.source = SourceCurrentCache
	cachedBinding, err := AssembleBinding(facts, cacheUniverse, cachePriors)
	if err != nil || cachedBinding.Identity() != binding.Identity() {
		t.Fatalf("fresh/cache equivalence = %s/%s, %v", binding.Identity(), cachedBinding.Identity(), err)
	}
	permutedFacts := slices.Clone(priorFacts)
	permutedFacts[0], permutedFacts[2] = permutedFacts[2], permutedFacts[0]
	slices.SortFunc(permutedFacts, func(left, right PriorCloseFact) int {
		if left.symbol < right.symbol {
			return -1
		}
		if left.symbol > right.symbol {
			return 1
		}
		return 0
	})
	permutedID, _ := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, permutedFacts)
	if permutedID != priorIdentity {
		t.Fatal("semantic row permutation changed prior-close identity")
	}

	returnedSymbols := binding.UniverseSymbols()
	returnedSymbols[0] = "MUTATED"
	returnedPriors := binding.PriorCloseFacts()
	returnedPriors[0].close = 999
	if binding.UniverseSymbols()[0] != "AAA" || binding.PriorCloseFacts()[0].close != 10.25 {
		t.Fatal("caller mutated immutable binding containers")
	}
	if _, ok := binding.PriorCloseFacts()[1].Close(); ok {
		t.Fatal("missing prior close exposed a numeric value")
	}
	if _, ok := binding.PriorCloseFacts()[2].Close(); ok {
		t.Fatal("invalid prior close exposed a numeric value")
	}

	if universe.Identity() != "universe-v1:c5b00c6cd4d4a620f241b6f56d1d295eb5747eb5ce96e1dabbe907ec68e9a4a5" ||
		priorIdentity != "prior-close-v1:45c4bda6ae7b265946f0a0649bd8219b0c390617885c3a0cd6043b35509d296a" ||
		binding.Identity() != "session-binding-v1:2186da57470aa8925a76b1fa12575fe96ad3c2b1584478449f8008dc7d18343d" {
		t.Fatalf("canonical identity golden changed: %s / %s / %s", universe.Identity(), priorIdentity, binding.Identity())
	}
}

func TestIdentityChangesForEveryDefiningSemanticFact(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA"})
	basePrior := []PriorCloseFact{{symbol: "AAA", status: PriorCloseValid, close: 10}}
	basePriorID, _ := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, basePrior)
	baseBindingID, _ := bindingIdentity(facts, universe.Identity(), basePriorID)

	priorChanges := map[string]struct {
		date   string
		policy string
		facts  []PriorCloseFact
	}{
		"date":   {"2026-07-27", PriorClosePolicyVersion, basePrior},
		"policy": {facts.PriorSessionDate, PriorClosePolicyVersion + "-changed", basePrior},
		"symbol": {facts.PriorSessionDate, PriorClosePolicyVersion, []PriorCloseFact{{symbol: "AAB", status: PriorCloseValid, close: 10}}},
		"status": {facts.PriorSessionDate, PriorClosePolicyVersion, []PriorCloseFact{{symbol: "AAA", status: PriorCloseMissing}}},
		"close":  {facts.PriorSessionDate, PriorClosePolicyVersion, []PriorCloseFact{{symbol: "AAA", status: PriorCloseValid, close: 10.000000000000002}}},
	}
	for name, change := range priorChanges {
		identity, err := priorCloseIdentity(change.date, change.policy, change.facts)
		if err != nil || identity == basePriorID {
			t.Fatalf("%s did not change prior-close identity: %s, %v", name, identity, err)
		}
	}

	changedFacts := map[string]session.Facts{
		"trading date":        facts,
		"session start":       facts,
		"session end":         facts,
		"schedule schema":     facts,
		"schedule version":    facts,
		"schedule artifact":   facts,
		"prior date":          facts,
		"prior regular close": facts,
	}
	value := changedFacts["trading date"]
	value.TradingDate = "2026-07-30"
	changedFacts["trading date"] = value
	value = changedFacts["session start"]
	value.SessionStart = value.SessionStart.Add(time.Second)
	changedFacts["session start"] = value
	value = changedFacts["session end"]
	value.SessionEnd = value.SessionEnd.Add(time.Second)
	changedFacts["session end"] = value
	value = changedFacts["schedule schema"]
	value.ScheduleSchema += "-changed"
	changedFacts["schedule schema"] = value
	value = changedFacts["schedule version"]
	value.ScheduleVersion += "-changed"
	changedFacts["schedule version"] = value
	value = changedFacts["schedule artifact"]
	value.ScheduleArtifactSHA256 = "changed"
	changedFacts["schedule artifact"] = value
	value = changedFacts["prior date"]
	value.PriorSessionDate = "2026-07-27"
	changedFacts["prior date"] = value
	value = changedFacts["prior regular close"]
	value.PriorRegularClose = value.PriorRegularClose.Add(time.Second)
	changedFacts["prior regular close"] = value

	for name, changed := range changedFacts {
		identity, err := bindingIdentity(changed, universe.Identity(), basePriorID)
		if err != nil || identity == baseBindingID {
			t.Fatalf("%s did not change binding identity: %s, %v", name, identity, err)
		}
	}
	for name, pair := range map[string][2]string{
		"universe identity": {"changed-universe", basePriorID},
		"prior identity":    {universe.Identity(), "changed-prior"},
	} {
		identity, err := bindingIdentity(facts, pair[0], pair[1])
		if err != nil || identity == baseBindingID {
			t.Fatalf("%s did not change binding identity: %s, %v", name, identity, err)
		}
	}
}

func TestScheduleRevalidatedCompleteBindingRejectsEveryScheduleMutation(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA"})
	priorFacts := []PriorCloseFact{{symbol: "AAA", status: PriorCloseMissing}}
	identity, _ := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, priorFacts)
	priors := PriorCloses{
		priorSessionDate: facts.PriorSessionDate,
		policyVersion:    PriorClosePolicyVersion,
		identity:         identity,
		facts:            priorFacts,
		accounting:       accountPriorCloses(priorFacts, 0),
		source:           SourceFresh,
	}

	mutations := []struct {
		name   string
		mutate func(*session.Facts)
	}{
		{"trading date", func(value *session.Facts) { value.TradingDate = "2026-07-30" }},
		{"session start", func(value *session.Facts) { value.SessionStart = value.SessionStart.Add(time.Second) }},
		{"session end", func(value *session.Facts) { value.SessionEnd = value.SessionEnd.Add(time.Second) }},
		{"prior-session date", func(value *session.Facts) { value.PriorSessionDate = "2026-07-27" }},
		{"prior regular close", func(value *session.Facts) { value.PriorRegularClose = value.PriorRegularClose.Add(time.Second) }},
		{"schedule schema", func(value *session.Facts) { value.ScheduleSchema += "-changed" }},
		{"schedule version", func(value *session.Facts) { value.ScheduleVersion += "-changed" }},
		{"schedule artifact identity", func(value *session.Facts) { value.ScheduleArtifactSHA256 = "changed" }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			changed := facts
			test.mutate(&changed)
			assertBindingRejected(t, changed, universe, priors)
		})
	}
}

func TestScheduleRevalidatedCompleteBindingRejectsUniverseAndPriorMismatches(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB"})
	priorFacts := []PriorCloseFact{
		{symbol: "AAA", status: PriorCloseValid, close: 10},
		{symbol: "BBB", status: PriorCloseMissing},
	}
	priorID, _ := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, priorFacts)
	priors := PriorCloses{
		priorSessionDate: facts.PriorSessionDate,
		policyVersion:    PriorClosePolicyVersion,
		identity:         priorID,
		facts:            slices.Clone(priorFacts),
		accounting:       accountPriorCloses(priorFacts, 0),
		source:           SourceFresh,
	}

	universeCases := []struct {
		name   string
		mutate func(*Universe)
	}{
		{"observable-only source", func(value *Universe) { value.source = SourceObservablePriorCache }},
		{"missing source", func(value *Universe) { value.source = SourceNone }},
		{"wrong current date", func(value *Universe) {
			value.referenceDate = "2026-07-28"
			value.identity, _ = universeIdentity(value.referenceDate, value.policyVersion, value.symbols)
		}},
		{"wrong policy", func(value *Universe) {
			value.policyVersion += "-changed"
			value.identity, _ = universeIdentity(value.referenceDate, value.policyVersion, value.symbols)
		}},
		{"unsorted population", func(value *Universe) {
			value.symbols = []string{"BBB", "AAA"}
			value.identity, _ = universeIdentity(value.referenceDate, value.policyVersion, value.symbols)
		}},
		{"population accounting", func(value *Universe) {
			value.accounting.EligibleRecords--
			value.accounting.InactiveRecords++
		}},
		{"closed accounting", func(value *Universe) { value.accounting.RawReferenceRecords++ }},
		{"different population", func(value *Universe) {
			value.symbols = []string{"AAA", "CCC"}
			value.identity, _ = universeIdentity(value.referenceDate, value.policyVersion, value.symbols)
		}},
		{"identity", func(value *Universe) { value.identity = "universe-v1:changed" }},
	}
	for _, test := range universeCases {
		t.Run("universe/"+test.name, func(t *testing.T) {
			changed := universe
			changed.symbols = slices.Clone(universe.symbols)
			test.mutate(&changed)
			assertBindingRejected(t, facts, changed, priors)
		})
	}

	priorCases := []struct {
		name   string
		mutate func(*PriorCloses)
	}{
		{"missing source", func(value *PriorCloses) { value.source = SourceNone }},
		{"observable-only source", func(value *PriorCloses) { value.source = SourceObservablePriorCache }},
		{"wrong date", func(value *PriorCloses) {
			value.priorSessionDate = "2026-07-27"
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
		{"wrong policy", func(value *PriorCloses) {
			value.policyVersion += "-changed"
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
		{"short population", func(value *PriorCloses) {
			value.facts = value.facts[:1]
			value.accounting = accountPriorCloses(value.facts, 0)
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
		{"different symbol population", func(value *PriorCloses) {
			value.facts[1].symbol = "CCC"
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
		{"accounting", func(value *PriorCloses) {
			value.accounting.ValidPriorClose--
			value.accounting.MissingPriorClose++
			value.accounting.InvalidOrMissingPriorClose++
		}},
		{"identity", func(value *PriorCloses) { value.identity = "prior-close-v1:changed" }},
		{"numeric missing close", func(value *PriorCloses) {
			value.facts[1].close = 99
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
		{"numeric invalid close", func(value *PriorCloses) {
			value.facts[1] = PriorCloseFact{symbol: "BBB", status: PriorCloseInvalid, close: 99, reason: PriorCloseWrongDate}
			value.accounting = accountPriorCloses(value.facts, 0)
			value.identity, _ = priorCloseIdentity(value.priorSessionDate, value.policyVersion, value.facts)
		}},
	}
	for _, test := range priorCases {
		t.Run("prior-close/"+test.name, func(t *testing.T) {
			changed := priors
			changed.facts = slices.Clone(priors.facts)
			test.mutate(&changed)
			assertBindingRejected(t, facts, universe, changed)
		})
	}
}

func TestScheduleRevalidatedCompleteBindingAllowsNoValidPriorClose(t *testing.T) {
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universe := acceptedUniverse(t, facts.TradingDate, []string{"AAA", "BBB"})
	priorFacts := []PriorCloseFact{
		{symbol: "AAA", status: PriorCloseMissing},
		{symbol: "BBB", status: PriorCloseInvalid, reason: PriorCloseWrongDate},
	}
	priorID, _ := priorCloseIdentity(facts.PriorSessionDate, PriorClosePolicyVersion, priorFacts)
	priors := PriorCloses{
		priorSessionDate: facts.PriorSessionDate,
		policyVersion:    PriorClosePolicyVersion,
		identity:         priorID,
		facts:            priorFacts,
		accounting:       accountPriorCloses(priorFacts, 0),
		source:           SourceFresh,
	}

	binding, err := AssembleBinding(facts, universe, priors)
	if err != nil {
		t.Fatal(err)
	}
	if binding.PriorCloseAccounting() != (PriorCloseAccounting{
		UniverseTotal: 2, MissingPriorClose: 1, InvalidPriorClose: 1, InvalidOrMissingPriorClose: 2,
	}) {
		t.Fatalf("no-valid-close accounting = %+v", binding.PriorCloseAccounting())
	}
	for _, fact := range binding.PriorCloseFacts() {
		if _, usable := fact.Close(); usable {
			t.Fatalf("%s received a fallback close", fact.Symbol())
		}
	}
}

func assertBindingRejected(t *testing.T, facts session.Facts, universe Universe, priors PriorCloses) {
	t.Helper()
	binding, err := AssembleBinding(facts, universe, priors)
	if err == nil {
		t.Fatalf("invalid facts escaped as binding %q", binding.Identity())
	}
	if binding.Identity() != "" || binding.UniverseSymbols() != nil || binding.PriorCloseFacts() != nil {
		t.Fatalf("failure returned a partial binding: %+v", binding)
	}
}
