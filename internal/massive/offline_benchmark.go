package massive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	OfflineSubsetBenchmarkSchema         = "rest-replay-subset-benchmark-v1"
	OfflineSubsetBenchmarkScope          = "deterministic_subset"
	OfflineSubsetSelectionMethod         = "evenly_spaced_sorted_binding_v1"
	OfflineSubsetBenchmarkMaximumTimeout = 15 * time.Minute
	OfflineSubsetBenchmarkStateComplete  = "subset_download_complete"
	OfflineSubsetBenchmarkStateFailed    = "subset_download_failed"
	OfflineSubsetBenchmarkStateCanceled  = "subset_download_canceled"
	OfflineSubsetBenchmarkStateInvalid   = "subset_download_invalid"
)

var offlineSubsetUnavailableMeasurements = []string{
	"provider_status_and_retry_reason_history",
	"acquisition_versus_local_postprocessing_duration",
	"process_cpu_peak_rss_and_host_facts",
	"artifact_and_temporary_bytes",
}

type OfflineSubsetBenchmarkConfig struct {
	FullBinding              reference.Binding
	SelectedSymbols          int
	Start, End               time.Time
	Workers                  int
	HardTimeout              time.Duration
	MaximumNormalizedRecords int64
	MaximumResponseBytes     int64
}

type OfflineSubsetSelection struct {
	FullBindingIdentity   string `json:"full_binding_identity"`
	SubsetBindingIdentity string `json:"subset_binding_identity"`
	Method                string `json:"method"`
	FullSymbols           int    `json:"full_symbols"`
	SelectedSymbols       int    `json:"selected_symbols"`
	SortedSymbolsSHA256   string `json:"sorted_symbols_sha256"`
}

// OfflineSubsetBenchmarkReport deliberately retains only measurement facts.
// It never exposes the subset binding, sealed DownloadResult, normalized rows,
// artifact identity, path, or a mutable acceptance/completeness claim.
type OfflineSubsetBenchmarkReport struct {
	selection   OfflineSubsetSelection
	start       time.Time
	end         time.Time
	workers     int
	hardTimeout time.Duration
	wall        time.Duration
	state       string
	accounting  DownloadAccounting
	outcomes    []SymbolOutcome
	reconciles  bool
}

func (r OfflineSubsetBenchmarkReport) Selection() OfflineSubsetSelection { return r.selection }
func (r OfflineSubsetBenchmarkReport) Start() time.Time                  { return r.start }
func (r OfflineSubsetBenchmarkReport) End() time.Time                    { return r.end }
func (r OfflineSubsetBenchmarkReport) Workers() int                      { return r.workers }
func (r OfflineSubsetBenchmarkReport) HardTimeout() time.Duration        { return r.hardTimeout }
func (r OfflineSubsetBenchmarkReport) WallDuration() time.Duration       { return r.wall }
func (r OfflineSubsetBenchmarkReport) State() string                     { return r.state }
func (r OfflineSubsetBenchmarkReport) Accounting() DownloadAccounting    { return r.accounting }
func (r OfflineSubsetBenchmarkReport) Outcomes() []SymbolOutcome         { return slices.Clone(r.outcomes) }
func (r OfflineSubsetBenchmarkReport) Reconciles() bool                  { return r.reconciles }
func (OfflineSubsetBenchmarkReport) EvidenceScope() string               { return OfflineSubsetBenchmarkScope }
func (OfflineSubsetBenchmarkReport) CompleteUniverse() bool              { return false }
func (OfflineSubsetBenchmarkReport) ArtifactEligible() bool              { return false }
func (OfflineSubsetBenchmarkReport) AcceptanceEligible() bool            { return false }
func (OfflineSubsetBenchmarkReport) UnavailableMeasurements() []string {
	return slices.Clone(offlineSubsetUnavailableMeasurements)
}

func (r OfflineSubsetBenchmarkReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Schema                  string                 `json:"schema"`
		EvidenceScope           string                 `json:"evidence_scope"`
		CompleteUniverse        bool                   `json:"complete_universe"`
		ArtifactEligible        bool                   `json:"artifact_eligible"`
		AcceptanceEligible      bool                   `json:"acceptance_eligible"`
		Selection               OfflineSubsetSelection `json:"selection"`
		Start                   time.Time              `json:"start"`
		End                     time.Time              `json:"end"`
		Workers                 int                    `json:"workers"`
		HardTimeoutNanoseconds  int64                  `json:"hard_timeout_nanoseconds"`
		WallDurationNanoseconds int64                  `json:"wall_duration_nanoseconds"`
		State                   string                 `json:"state"`
		Accounting              DownloadAccounting     `json:"accounting"`
		Outcomes                []SymbolOutcome        `json:"outcomes"`
		Reconciles              bool                   `json:"reconciles"`
		UnavailableMeasurements []string               `json:"unavailable_measurements"`
	}{
		Schema:                  OfflineSubsetBenchmarkSchema,
		EvidenceScope:           OfflineSubsetBenchmarkScope,
		CompleteUniverse:        false,
		ArtifactEligible:        false,
		AcceptanceEligible:      false,
		Selection:               r.selection,
		Start:                   r.start,
		End:                     r.end,
		Workers:                 r.workers,
		HardTimeoutNanoseconds:  int64(r.hardTimeout),
		WallDurationNanoseconds: int64(r.wall),
		State:                   r.state,
		Accounting:              r.accounting,
		Outcomes:                r.Outcomes(),
		Reconciles:              r.reconciles,
		UnavailableMeasurements: r.UnavailableMeasurements(),
	})
}

// RunOfflineSubsetBenchmark executes the sealed downloader over a derived
// strict subset binding. Download failure is measurement evidence and returns
// a report; only invalid harness configuration or invalid source identity is
// returned as an error.
func RunOfflineSubsetBenchmark(parent context.Context, downloader *OfflineDownloader, config OfflineSubsetBenchmarkConfig) (OfflineSubsetBenchmarkReport, error) {
	report := OfflineSubsetBenchmarkReport{
		start: config.Start, end: config.End, workers: config.Workers, hardTimeout: config.HardTimeout,
		state: OfflineSubsetBenchmarkStateInvalid,
	}
	fullSymbols := config.FullBinding.UniverseSymbols()
	if parent == nil || downloader == nil || config.SelectedSymbols < 1 || config.SelectedSymbols >= len(fullSymbols) ||
		config.Workers < 1 || config.Workers > OfflineWorkerLimit || config.HardTimeout <= 0 || config.HardTimeout > OfflineSubsetBenchmarkMaximumTimeout ||
		config.MaximumNormalizedRecords <= 0 || config.MaximumResponseBytes <= 0 {
		return report, errors.New("invalid offline subset benchmark configuration")
	}
	selected := evenlySpacedBenchmarkSymbols(fullSymbols, config.SelectedSymbols)
	report.selection = OfflineSubsetSelection{
		FullBindingIdentity: config.FullBinding.Identity(), Method: OfflineSubsetSelectionMethod,
		FullSymbols: len(fullSymbols), SelectedSymbols: len(selected), SortedSymbolsSHA256: benchmarkSymbolsDigest(selected),
	}

	var runErr error
	err := reference.WithBenchmarkSubsetBinding(config.FullBinding, selected, func(subset reference.Binding) error {
		report.selection.SubsetBindingIdentity = subset.Identity()
		operation, cancel := context.WithTimeout(parent, config.HardTimeout)
		defer cancel()
		started := time.Now()
		result := downloader.Download(operation, DownloadPlan{
			Binding: subset, Start: config.Start, End: config.End, Workers: config.Workers,
			MaximumNormalizedRecords: config.MaximumNormalizedRecords, MaximumResponseBytes: config.MaximumResponseBytes,
		})
		report.wall = time.Since(started)
		report.accounting = result.Accounting()
		report.outcomes = result.Outcomes()
		report.reconciles = benchmarkAccountingReconciles(report.accounting, report.outcomes, selected)
		switch {
		case !report.reconciles:
			report.state = OfflineSubsetBenchmarkStateInvalid
			runErr = errors.New("offline subset benchmark accounting did not reconcile")
		case result.Complete():
			report.state = OfflineSubsetBenchmarkStateComplete
		case report.accounting.FailedSymbols == 0 && report.accounting.CanceledSymbols > 0:
			report.state = OfflineSubsetBenchmarkStateCanceled
		default:
			report.state = OfflineSubsetBenchmarkStateFailed
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	return report, runErr
}

func evenlySpacedBenchmarkSymbols(symbols []string, count int) []string {
	selected := make([]string, count)
	if count == 1 {
		selected[0] = symbols[(len(symbols)-1)/2]
		return selected
	}
	last := len(symbols) - 1
	for index := range count {
		selected[index] = symbols[index*last/(count-1)]
	}
	return selected
}

func benchmarkSymbolsDigest(symbols []string) string {
	encoded, _ := json.Marshal(symbols)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func benchmarkAccountingReconciles(accounting DownloadAccounting, outcomes []SymbolOutcome, selected []string) bool {
	if accounting.PlannedSymbols != int64(len(selected)) || len(outcomes) != len(selected) ||
		accounting.PlannedSymbols != accounting.CompleteSymbols+accounting.FailedSymbols+accounting.CanceledSymbols ||
		accounting.CompleteSymbols != accounting.NonemptySymbols+accounting.EmptySymbols {
		return false
	}
	var totals DownloadAccounting
	for index, outcome := range outcomes {
		if outcome.Symbol != selected[index] || outcome.Records < 0 || outcome.Pages < 0 || outcome.Attempts < 0 || outcome.Bytes < 0 {
			return false
		}
		totals.Records += outcome.Records
		totals.Pages += outcome.Pages
		totals.Attempts += outcome.Attempts
		totals.Bytes += outcome.Bytes
		switch outcome.State {
		case SymbolComplete:
			totals.CompleteSymbols++
			if outcome.Records == 0 {
				totals.EmptySymbols++
			} else {
				totals.NonemptySymbols++
			}
		case SymbolFailed:
			totals.FailedSymbols++
		case SymbolCanceled:
			totals.CanceledSymbols++
		default:
			return false
		}
	}
	return accounting.CompleteSymbols == totals.CompleteSymbols && accounting.FailedSymbols == totals.FailedSymbols &&
		accounting.CanceledSymbols == totals.CanceledSymbols && accounting.NonemptySymbols == totals.NonemptySymbols &&
		accounting.EmptySymbols == totals.EmptySymbols && accounting.Records == totals.Records &&
		accounting.Pages == totals.Pages && accounting.Attempts == totals.Attempts && accounting.Bytes == totals.Bytes
}
