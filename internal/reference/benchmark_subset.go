package reference

import (
	"errors"
	"slices"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

// WithBenchmarkSubsetBinding derives one valid binding for a strict, sorted
// subset of an already accepted binding and confines that value to consume.
// It exists only for the benchmark harness: callers must not persist or treat
// the subset as complete-universe evidence.
func WithBenchmarkSubsetBinding(full Binding, selected []string, consume func(Binding) error) error {
	if consume == nil || len(selected) == 0 || len(selected) >= len(full.universeSymbols) ||
		!slices.IsSorted(selected) || len(slices.Compact(slices.Clone(selected))) != len(selected) {
		return errors.New("benchmark binding requires a strict canonical subset and consumer")
	}
	facts, err := acceptedBindingFacts(full)
	if err != nil {
		return err
	}

	selectedFacts := make([]PriorCloseFact, len(selected))
	for index, symbol := range selected {
		fullIndex, found := slices.BinarySearch(full.universeSymbols, symbol)
		if !found {
			return errors.New("benchmark symbol is outside the accepted binding")
		}
		selectedFacts[index] = full.priorCloseFacts[fullIndex]
	}
	universeID, err := universeIdentity(full.tradingDate, full.universePolicy, selected)
	if err != nil {
		return errors.New("derive benchmark universe identity")
	}
	priorID, err := priorCloseIdentity(full.priorSessionDate, full.priorClosePolicy, selectedFacts)
	if err != nil {
		return errors.New("derive benchmark prior-close identity")
	}
	universe := Universe{
		referenceDate: full.tradingDate,
		policyVersion: full.universePolicy,
		identity:      universeID,
		symbols:       slices.Clone(selected),
		accounting: Accounting{
			RawReferenceRecords: len(selected),
			EligibleRecords:     len(selected),
		},
		source: SourceCurrentCache,
	}
	priors := PriorCloses{
		priorSessionDate: full.priorSessionDate,
		policyVersion:    full.priorClosePolicy,
		identity:         priorID,
		facts:            selectedFacts,
		accounting:       accountPriorCloses(selectedFacts, 0),
		source:           SourceCurrentCache,
	}
	subset, err := AssembleBinding(facts, universe, priors)
	if err != nil {
		return errors.New("assemble valid benchmark subset binding")
	}
	return consume(subset)
}

func acceptedBindingFacts(binding Binding) (session.Facts, error) {
	if binding.identity == "" || len(binding.universeSymbols) == 0 ||
		len(binding.priorCloseFacts) != len(binding.universeSymbols) {
		return session.Facts{}, errors.New("benchmark source binding is incomplete")
	}
	schedule, err := session.Load()
	if err != nil {
		return session.Facts{}, errors.New("load accepted schedule for benchmark binding")
	}
	facts, err := schedule.ForTradingDate(binding.tradingDate)
	if err != nil || facts.SessionStart != binding.sessionStart || facts.SessionEnd != binding.sessionEnd ||
		facts.PriorSessionDate != binding.priorSessionDate || facts.PriorRegularClose != binding.priorRegularClose ||
		facts.ScheduleSchema != binding.scheduleSchema || facts.ScheduleVersion != binding.scheduleVersion ||
		facts.ScheduleArtifactSHA256 != binding.scheduleArtifactSHA256 {
		return session.Facts{}, errors.New("benchmark source binding has incompatible schedule facts")
	}
	universeID, universeErr := universeIdentity(binding.tradingDate, binding.universePolicy, binding.universeSymbols)
	priorID, priorErr := priorCloseIdentity(binding.priorSessionDate, binding.priorClosePolicy, binding.priorCloseFacts)
	bindingID, bindingErr := bindingIdentity(facts, universeID, priorID)
	if universeErr != nil || priorErr != nil || bindingErr != nil || universeID != binding.universeIdentity ||
		priorID != binding.priorCloseIdentity || bindingID != binding.identity {
		return session.Facts{}, errors.New("benchmark source binding identity is invalid")
	}
	return facts, nil
}
