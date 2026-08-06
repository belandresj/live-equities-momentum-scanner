package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	bindingIdentitySchema  = "session-binding-v1"
	universeIdentitySchema = "universe-v1"
	priorIdentitySchema    = "prior-close-v1"
	maximumUniverseSymbols = 100_000
	maximumSymbolBytes     = 64
)

type frozenPriorClose struct {
	symbol string
	status reference.PriorCloseStatus
	close  float64
	reason reference.PriorCloseInvalidReason
}

type frozenBinding struct {
	tradingDate, scheduleSchema, scheduleVersion, scheduleArtifactSHA256 string
	priorSessionDate, universePolicy, universeIdentity                   string
	priorClosePolicy, priorCloseIdentity, identity                       string
	sessionStart, sessionEnd, priorRegularClose                          time.Time
	symbols                                                              []string
	universeAccounting                                                   reference.Accounting
	priors                                                               []frozenPriorClose
	priorAccounting                                                      reference.PriorCloseAccounting
	adjusted, includeOTC                                                 bool
	locale, market                                                       string
}

type coreSymbol struct {
	symbol     string
	prior      frozenPriorClose
	aggregates *symbolAggregateState
}

type installedBinding struct {
	identity, tradingDate, scheduleSchema, scheduleVersion, scheduleArtifactSHA256 string
	priorSessionDate, universePolicy, universeIdentity                             string
	priorClosePolicy, priorCloseIdentity, locale, market                           string
	sessionStart, sessionEnd, priorRegularClose                                    time.Time
	adjusted, includeOTC                                                           bool
	symbols                                                                        []coreSymbol
	index                                                                          map[string]int
	universeAccounting                                                             reference.Accounting
	priorAccounting                                                                reference.PriorCloseAccounting
}

func freezeBinding(binding reference.Binding) frozenBinding {
	facts := binding.PriorCloseFacts()
	priors := make([]frozenPriorClose, len(facts))
	for index, fact := range facts {
		value, _ := fact.Close()
		priors[index] = frozenPriorClose{fact.Symbol(), fact.Status(), value, fact.Reason()}
	}
	return frozenBinding{
		tradingDate: binding.TradingDate(), sessionStart: binding.SessionStart(), sessionEnd: binding.SessionEnd(),
		scheduleSchema: binding.ScheduleSchema(), scheduleVersion: binding.ScheduleVersion(),
		scheduleArtifactSHA256: binding.ScheduleArtifactSHA256(), priorSessionDate: binding.PriorSessionDate(),
		priorRegularClose: binding.PriorRegularClose(), universePolicy: binding.UniversePolicy(),
		universeIdentity: binding.UniverseIdentity(), symbols: binding.UniverseSymbols(),
		universeAccounting: binding.UniverseAccounting(), priorClosePolicy: binding.PriorClosePolicy(),
		priorCloseIdentity: binding.PriorCloseIdentity(), priorAccounting: binding.PriorCloseAccounting(),
		identity: binding.Identity(), priors: priors, adjusted: binding.PriorCloseAdjusted(),
		includeOTC: binding.PriorCloseIncludeOTC(), locale: binding.PriorCloseLocale(), market: binding.PriorCloseMarket(),
	}
}

func buildInstalledBinding(binding frozenBinding) (*installedBinding, error) {
	if err := validateFrozenBinding(binding); err != nil {
		return nil, err
	}
	candidate := &installedBinding{
		identity: binding.identity, tradingDate: binding.tradingDate,
		scheduleSchema: binding.scheduleSchema, scheduleVersion: binding.scheduleVersion,
		scheduleArtifactSHA256: binding.scheduleArtifactSHA256, priorSessionDate: binding.priorSessionDate,
		universePolicy: binding.universePolicy, universeIdentity: binding.universeIdentity,
		priorClosePolicy: binding.priorClosePolicy, priorCloseIdentity: binding.priorCloseIdentity,
		locale: binding.locale, market: binding.market, sessionStart: binding.sessionStart,
		sessionEnd: binding.sessionEnd, priorRegularClose: binding.priorRegularClose,
		adjusted: binding.adjusted, includeOTC: binding.includeOTC,
		symbols: make([]coreSymbol, len(binding.symbols)), index: make(map[string]int, len(binding.symbols)),
		universeAccounting: binding.universeAccounting, priorAccounting: binding.priorAccounting,
	}
	for index, symbol := range binding.symbols {
		candidate.symbols[index] = coreSymbol{symbol: symbol, prior: binding.priors[index]}
		candidate.index[symbol] = index
	}
	return candidate, nil
}

func validateFrozenBinding(binding frozenBinding) error {
	if !validIdentityShape(binding.identity) || binding.tradingDate == "" || binding.scheduleSchema == "" ||
		binding.scheduleVersion == "" || binding.scheduleArtifactSHA256 == "" || binding.priorSessionDate == "" ||
		binding.universePolicy != reference.EligibilityPolicyVersion || binding.priorClosePolicy != reference.PriorClosePolicyVersion ||
		!binding.adjusted || binding.includeOTC || binding.locale != "us" || binding.market != "stocks" {
		return errors.New("incomplete binding metadata")
	}
	if binding.sessionStart != binding.sessionStart.UTC() || binding.sessionEnd != binding.sessionEnd.UTC() ||
		binding.priorRegularClose != binding.priorRegularClose.UTC() || !binding.sessionStart.Before(binding.sessionEnd) ||
		binding.sessionEnd.Sub(binding.sessionStart) != 16*time.Hour {
		return errors.New("invalid binding times")
	}
	if err := validateScheduleSemantics(binding); err != nil {
		return err
	}
	if len(binding.symbols) == 0 || len(binding.symbols) > maximumUniverseSymbols || len(binding.priors) != len(binding.symbols) {
		return errors.New("invalid binding population size")
	}
	previous := ""
	for index, symbol := range binding.symbols {
		if symbol == "" || len(symbol) > maximumSymbolBytes || !utf8.ValidString(symbol) || (index > 0 && symbol <= previous) {
			return errors.New("binding symbols are not canonical")
		}
		previous = symbol
		prior := binding.priors[index]
		if prior.symbol != symbol {
			return errors.New("prior population does not match universe")
		}
		switch prior.status {
		case reference.PriorCloseValid:
			if prior.close <= 0 || math.IsNaN(prior.close) || math.IsInf(prior.close, 0) || prior.reason != "" {
				return errors.New("invalid valid prior close")
			}
		case reference.PriorCloseMissing:
			if prior.close != 0 || prior.reason != "" {
				return errors.New("invalid missing prior close")
			}
		case reference.PriorCloseInvalid:
			if prior.close != 0 || !validPriorReason(prior.reason) {
				return errors.New("invalid rejected prior close")
			}
		default:
			return errors.New("unknown prior close status")
		}
	}
	accounting := binding.universeAccounting
	if accounting.RawReferenceRecords < 0 || accounting.RawReferenceRecords > maximumUniverseSymbols ||
		accounting.EligibleRecords != len(binding.symbols) || accounting.InactiveRecords < 0 ||
		accounting.WrongMarketRecords < 0 || accounting.WrongLocaleRecords < 0 || accounting.IneligibleTypeRecords < 0 ||
		accounting.RawReferenceRecords != accounting.EligibleRecords+accounting.InactiveRecords+accounting.WrongMarketRecords+
			accounting.WrongLocaleRecords+accounting.IneligibleTypeRecords {
		return errors.New("universe accounting mismatch")
	}
	priorAccounting := accountPriors(binding.priors, binding.priorAccounting.UnattributableDiagnosticRows)
	if binding.priorAccounting.UnattributableDiagnosticRows < 0 || binding.priorAccounting.UnattributableDiagnosticRows > maximumUniverseSymbols ||
		priorAccounting != binding.priorAccounting {
		return errors.New("prior accounting mismatch")
	}
	universeID, err := computeUniverseIdentity(binding.tradingDate, binding.universePolicy, binding.symbols)
	if err != nil || universeID != binding.universeIdentity {
		return errors.New("universe identity mismatch")
	}
	priorID, err := computePriorIdentity(binding.priorSessionDate, binding.priorClosePolicy, binding.priors)
	if err != nil || priorID != binding.priorCloseIdentity {
		return errors.New("prior identity mismatch")
	}
	bindingID, err := computeBindingIdentity(binding, universeID, priorID)
	if err != nil || bindingID != binding.identity {
		return errors.New("binding identity mismatch")
	}
	return nil
}

func validateScheduleSemantics(binding frozenBinding) error {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return errors.New("load America/New_York timezone")
	}
	tradingDate, err := time.ParseInLocation("2006-01-02", binding.tradingDate, location)
	if err != nil {
		return errors.New("invalid trading date")
	}
	wantStart := time.Date(tradingDate.Year(), tradingDate.Month(), tradingDate.Day(), 4, 0, 0, 0, location).UTC()
	wantEnd := time.Date(tradingDate.Year(), tradingDate.Month(), tradingDate.Day(), 20, 0, 0, 0, location).UTC()
	if binding.sessionStart != wantStart || binding.sessionEnd != wantEnd {
		return errors.New("scanner bounds do not match the bound New York date")
	}
	priorDate, err := time.ParseInLocation("2006-01-02", binding.priorSessionDate, location)
	if err != nil || !priorDate.Before(tradingDate) {
		return errors.New("invalid prior-session date")
	}
	priorClose := binding.priorRegularClose.In(location)
	if priorClose.Year() != priorDate.Year() || priorClose.Month() != priorDate.Month() || priorClose.Day() != priorDate.Day() ||
		(priorClose.Hour() != 13 && priorClose.Hour() != 16) || priorClose.Minute() != 0 || priorClose.Second() != 0 || priorClose.Nanosecond() != 0 {
		return errors.New("invalid prior regular close")
	}
	if len(binding.scheduleArtifactSHA256) != sha256.Size*2 || binding.scheduleArtifactSHA256 != strings.ToLower(binding.scheduleArtifactSHA256) {
		return errors.New("invalid schedule artifact identity")
	}
	if _, err := hex.DecodeString(binding.scheduleArtifactSHA256); err != nil {
		return errors.New("invalid schedule artifact identity")
	}
	return nil
}

func validIdentityShape(identity string) bool {
	prefix := bindingIdentitySchema + ":"
	if !strings.HasPrefix(identity, prefix) || len(identity) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(identity[len(prefix):])
	return err == nil
}

func validPriorReason(reason reference.PriorCloseInvalidReason) bool {
	switch reason {
	case reference.PriorCloseMalformed, reference.PriorCloseDuplicate, reference.PriorCloseWrongDate,
		reference.PriorCloseNonpositive, reference.PriorCloseNonfinite:
		return true
	default:
		return false
	}
}

func accountPriors(priors []frozenPriorClose, diagnostics int) reference.PriorCloseAccounting {
	result := reference.PriorCloseAccounting{UniverseTotal: len(priors), UnattributableDiagnosticRows: diagnostics}
	for _, prior := range priors {
		switch prior.status {
		case reference.PriorCloseValid:
			result.ValidPriorClose++
		case reference.PriorCloseMissing:
			result.MissingPriorClose++
		case reference.PriorCloseInvalid:
			result.InvalidPriorClose++
		}
	}
	result.InvalidOrMissingPriorClose = result.MissingPriorClose + result.InvalidPriorClose
	return result
}

func computeUniverseIdentity(date, policy string, symbols []string) (string, error) {
	payload := struct {
		Schema            string   `json:"schema"`
		TradingDate       string   `json:"trading_date"`
		EligibilityPolicy string   `json:"eligibility_policy"`
		SortedSymbols     []string `json:"sorted_symbols"`
	}{universeIdentitySchema, date, policy, symbols}
	return digestJSON(universeIdentitySchema, payload)
}

func computePriorIdentity(date, policy string, priors []frozenPriorClose) (string, error) {
	type identityFact struct {
		Symbol    string `json:"symbol"`
		Status    string `json:"status"`
		CloseBits string `json:"close_bits_or_empty"`
	}
	pairs := make([]identityFact, len(priors))
	for index, prior := range priors {
		bits := ""
		if prior.status == reference.PriorCloseValid {
			bits = fmt.Sprintf("%016x", math.Float64bits(prior.close))
		}
		pairs[index] = identityFact{prior.symbol, string(prior.status), bits}
	}
	payload := struct {
		Schema            string         `json:"schema"`
		PriorSessionDate  string         `json:"prior_session_date"`
		PriorClosePolicy  string         `json:"prior_close_policy"`
		Adjusted          bool           `json:"adjusted"`
		IncludeOTC        bool           `json:"include_otc"`
		Locale            string         `json:"locale"`
		Market            string         `json:"market"`
		SortedSymbolFacts []identityFact `json:"sorted_per_universe_symbol"`
	}{priorIdentitySchema, date, policy, true, false, "us", "stocks", pairs}
	return digestJSON(priorIdentitySchema, payload)
}

func computeBindingIdentity(binding frozenBinding, universeID, priorID string) (string, error) {
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
	}{bindingIdentitySchema, binding.tradingDate, binding.sessionStart.UTC().Format(time.RFC3339Nano),
		binding.sessionEnd.UTC().Format(time.RFC3339Nano), binding.scheduleSchema, binding.scheduleVersion,
		binding.scheduleArtifactSHA256, binding.priorSessionDate, binding.priorRegularClose.UTC().Format(time.RFC3339Nano), universeID, priorID}
	return digestJSON(bindingIdentitySchema, payload)
}

func digestJSON(schema string, payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return schema + ":" + hex.EncodeToString(sum[:]), nil
}
