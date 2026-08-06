package reference

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const bindingIdentitySchema = "session-binding-v1"

// Binding is the complete immutable component-1 fact. It owns no engine state.
type Binding struct {
	tradingDate            string
	sessionStart           time.Time
	sessionEnd             time.Time
	scheduleSchema         string
	scheduleVersion        string
	scheduleArtifactSHA256 string
	priorSessionDate       string
	priorRegularClose      time.Time
	universePolicy         string
	universeIdentity       string
	universeSymbols        []string
	universeAccounting     Accounting
	priorClosePolicy       string
	priorCloseIdentity     string
	priorCloseFacts        []PriorCloseFact
	priorCloseAccounting   PriorCloseAccounting
	identity               string
}

func (b Binding) TradingDate() string                        { return b.tradingDate }
func (b Binding) SessionStart() time.Time                    { return b.sessionStart }
func (b Binding) SessionEnd() time.Time                      { return b.sessionEnd }
func (b Binding) ScheduleSchema() string                     { return b.scheduleSchema }
func (b Binding) ScheduleVersion() string                    { return b.scheduleVersion }
func (b Binding) ScheduleArtifactSHA256() string             { return b.scheduleArtifactSHA256 }
func (b Binding) PriorSessionDate() string                   { return b.priorSessionDate }
func (b Binding) PriorRegularClose() time.Time               { return b.priorRegularClose }
func (b Binding) UniversePolicy() string                     { return b.universePolicy }
func (b Binding) UniverseIdentity() string                   { return b.universeIdentity }
func (b Binding) UniverseSymbols() []string                  { return slices.Clone(b.universeSymbols) }
func (b Binding) UniverseAccounting() Accounting             { return b.universeAccounting }
func (b Binding) PriorClosePolicy() string                   { return b.priorClosePolicy }
func (b Binding) PriorCloseAdjusted() bool                   { return true }
func (b Binding) PriorCloseIncludeOTC() bool                 { return false }
func (b Binding) PriorCloseLocale() string                   { return "us" }
func (b Binding) PriorCloseMarket() string                   { return "stocks" }
func (b Binding) PriorCloseIdentity() string                 { return b.priorCloseIdentity }
func (b Binding) PriorCloseFacts() []PriorCloseFact          { return slices.Clone(b.priorCloseFacts) }
func (b Binding) PriorCloseAccounting() PriorCloseAccounting { return b.priorCloseAccounting }
func (b Binding) Identity() string                           { return b.identity }

// AssembleBinding joins accepted S1, S2, and S3 facts without installing them.
// It reloads the repository-accepted schedule and independently rederives the
// complete schedule fact for the requested trading date before trusting any
// caller-supplied schedule value.
func AssembleBinding(facts session.Facts, universe Universe, priors PriorCloses) (Binding, error) {
	acceptedSchedule, err := session.Load()
	if err != nil {
		return Binding{}, fmt.Errorf("load accepted schedule for binding: %w", err)
	}
	acceptedFacts, err := acceptedSchedule.ForTradingDate(facts.TradingDate)
	if err != nil || acceptedFacts != facts {
		return Binding{}, errors.New("binding schedule facts do not match the accepted schedule selection")
	}
	if !universe.IsCurrent() || universe.referenceDate != acceptedFacts.TradingDate ||
		universe.policyVersion != EligibilityPolicyVersion || len(universe.symbols) == 0 ||
		universe.accounting.EligibleRecords != len(universe.symbols) || !universe.accounting.valid() {
		return Binding{}, errors.New("binding requires a current exact-date universe")
	}
	previous := ""
	for _, symbol := range universe.symbols {
		if !validSymbol(symbol) || symbol <= previous {
			return Binding{}, errors.New("binding universe population is noncanonical")
		}
		previous = symbol
	}
	if !currentReferenceSource(priors.source) || priors.priorSessionDate != acceptedFacts.PriorSessionDate ||
		priors.policyVersion != PriorClosePolicyVersion ||
		len(priors.facts) != len(universe.symbols) || !priors.accounting.valid() {
		return Binding{}, errors.New("binding requires exact-date prior-close facts for the universe")
	}
	for index, symbol := range universe.symbols {
		if priors.facts[index].symbol != symbol {
			return Binding{}, errors.New("prior-close population does not match universe")
		}
		fact := priors.facts[index]
		switch fact.status {
		case PriorCloseValid:
			if fact.close <= 0 || !isFinite(fact.close) || fact.reason != "" {
				return Binding{}, errors.New("invalid valid prior-close binding fact")
			}
		case PriorCloseMissing:
			if fact.close != 0 || fact.reason != "" {
				return Binding{}, errors.New("invalid missing prior-close binding fact")
			}
		case PriorCloseInvalid:
			if fact.close != 0 || !validPriorInvalidReason(fact.reason) {
				return Binding{}, errors.New("invalid invalid prior-close binding fact")
			}
		default:
			return Binding{}, errors.New("unknown prior-close binding status")
		}
	}
	if accountPriorCloses(priors.facts, priors.accounting.UnattributableDiagnosticRows) != priors.accounting {
		return Binding{}, errors.New("prior-close binding accounting mismatch")
	}
	universeID, err := universeIdentity(universe.referenceDate, universe.policyVersion, universe.symbols)
	if err != nil || universeID != universe.identity {
		return Binding{}, errors.New("universe identity mismatch")
	}
	priorID, err := priorCloseIdentity(priors.priorSessionDate, priors.policyVersion, priors.facts)
	if err != nil || priorID != priors.identity {
		return Binding{}, errors.New("prior-close identity mismatch")
	}
	bindingID, err := bindingIdentity(acceptedFacts, universeID, priorID)
	if err != nil {
		return Binding{}, err
	}
	return Binding{
		tradingDate:            acceptedFacts.TradingDate,
		sessionStart:           acceptedFacts.SessionStart,
		sessionEnd:             acceptedFacts.SessionEnd,
		scheduleSchema:         acceptedFacts.ScheduleSchema,
		scheduleVersion:        acceptedFacts.ScheduleVersion,
		scheduleArtifactSHA256: acceptedFacts.ScheduleArtifactSHA256,
		priorSessionDate:       acceptedFacts.PriorSessionDate,
		priorRegularClose:      acceptedFacts.PriorRegularClose,
		universePolicy:         universe.policyVersion,
		universeIdentity:       universeID,
		universeSymbols:        slices.Clone(universe.symbols),
		universeAccounting:     universe.accounting,
		priorClosePolicy:       priors.policyVersion,
		priorCloseIdentity:     priorID,
		priorCloseFacts:        slices.Clone(priors.facts),
		priorCloseAccounting:   priors.accounting,
		identity:               bindingID,
	}, nil
}

func currentReferenceSource(source Source) bool {
	return source == SourceFresh || source == SourceCurrentCache
}

func bindingIdentity(facts session.Facts, universeID, priorID string) (string, error) {
	payload := struct {
		Schema                 string `json:"schema"`
		TradingDate            string `json:"trading_date"`
		SessionStart           string `json:"S"`
		SessionEnd             string `json:"E"`
		ScheduleSchema         string `json:"schedule_schema"`
		ScheduleVersion        string `json:"schedule_version"`
		ScheduleArtifactSHA256 string `json:"schedule_artifact_sha256"`
		PriorSessionDate       string `json:"prior_session_date"`
		PriorRegularClose      string `json:"prior_regular_close"`
		UniverseIdentity       string `json:"universe_identity"`
		PriorCloseIdentity     string `json:"prior_close_identity"`
	}{
		bindingIdentitySchema,
		facts.TradingDate,
		facts.SessionStart.UTC().Format(time.RFC3339Nano),
		facts.SessionEnd.UTC().Format(time.RFC3339Nano),
		facts.ScheduleSchema,
		facts.ScheduleVersion,
		facts.ScheduleArtifactSHA256,
		facts.PriorSessionDate,
		facts.PriorRegularClose.UTC().Format(time.RFC3339Nano),
		universeID,
		priorID,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode binding identity: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return bindingIdentitySchema + ":" + hex.EncodeToString(sum[:]), nil
}
