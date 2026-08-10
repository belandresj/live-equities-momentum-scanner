package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("aggregate-replay", flag.ContinueOnError)
	tradingDate := flags.String("trading-date", "", "exchange-local trading date YYYY-MM-DD")
	replayStart := flags.String("from", "", "optional UTC RFC3339 replay interval start")
	replayEnd := flags.String("to", "", "optional UTC RFC3339 replay interval end")
	referenceDirectory := flags.String("reference-dir", filepath.Join("var", "reference"), "local Component 1 cache directory")
	destination := flags.String("destination", filepath.Join("var", "aggregate-replay"), "local artifact directory")
	artifact := flags.String("artifact", "", "validated complete artifact to replay offline")
	providerOrigin := flags.String("provider-origin", "https://api.massive.com", "Massive HTTPS origin")
	workers := flags.Int("workers", 4, "1..8 aggregate symbol workers")
	maximumRecords := flags.Int64("maximum-records", 1_000_000_000, "maximum normalized and in-memory records")
	maximumResponseBytes := flags.Int64("maximum-response-bytes", 512<<20, "maximum aggregate response bytes")
	maximumArtifactBytes := flags.Int64("maximum-artifact-bytes", 8<<30, "maximum canonical artifact bytes")
	maximumTemporaryBytes := flags.Int64("maximum-temporary-bytes", 8<<30, "maximum temporary bytes")
	maximumTemporaryFiles := flags.Int("maximum-temporary-files", 8, "maximum Component 4 temporary files")
	queueCapacity := flags.Int("queue-capacity", 0, "explicit replay engine queue capacity")
	requiredReserve := flags.Int("required-reserve", 0, "explicit replay engine required-input reserve")
	evaluationDelay := flags.Duration("evaluation-delay", -1, "explicit nonnegative replay evaluation delay")
	paceLogicalSeconds := flags.Uint64("pace-logical-seconds", 0, "finite pace logical seconds, paired with --pace-wall-seconds")
	paceWallSeconds := flags.Uint64("pace-wall-seconds", 0, "finite pace wall seconds, paired with --pace-logical-seconds")
	if err := flags.Parse(arguments); err != nil {
		return errors.New("invalid aggregate replay compile flags")
	}
	if *tradingDate == "" || flags.NArg() != 0 {
		return errors.New("exactly one --trading-date and no positional arguments are required")
	}
	if *artifact != "" {
		return runOfflineReplay(ctx, offlineReplayConfig{
			tradingDate: *tradingDate, replayStart: *replayStart, replayEnd: *replayEnd,
			referenceDirectory: *referenceDirectory, artifact: *artifact,
			maximumArtifactBytes: *maximumArtifactBytes, maximumRecords: *maximumRecords,
			queueCapacity: *queueCapacity, requiredReserve: *requiredReserve, evaluationDelay: *evaluationDelay,
			paceLogicalSeconds: *paceLogicalSeconds, paceWallSeconds: *paceWallSeconds,
		})
	}
	if *queueCapacity != 0 || *requiredReserve != 0 || *evaluationDelay != -1 || *paceLogicalSeconds != 0 || *paceWallSeconds != 0 {
		return errors.New("replay engine and pace flags require --artifact")
	}
	if destinationOutsideIgnoredVar(*destination) {
		fmt.Fprintln(os.Stderr, "warning: aggregate artifacts may contain licensed provider data; destination is outside the repository's ignored var/ directory")
	}
	credential := os.Getenv("MASSIVE_API_KEY")
	if credential == "" {
		return errors.New("MASSIVE_API_KEY is required")
	}
	schedule, err := session.Load()
	if err != nil {
		return errors.New("load accepted exchange schedule")
	}
	facts, err := schedule.ForTradingDate(*tradingDate)
	if err != nil {
		return errors.New("trading date is not supported by the accepted schedule")
	}
	client := &http.Client{}
	universe, err := (&reference.Resolver{BaseURL: *providerOrigin, APIKey: credential, DataDir: *referenceDirectory, HTTPClient: client, Schedule: schedule}).Resolve(ctx, facts)
	if err != nil {
		return errors.New("resolve complete binding universe")
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: *providerOrigin, APIKey: credential, DataDir: *referenceDirectory, HTTPClient: client, Schedule: schedule}).Resolve(ctx, facts, universe)
	if err != nil {
		return errors.New("resolve complete binding prior closes")
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		return errors.New("assemble immutable session binding")
	}
	start, end := binding.SessionStart(), binding.SessionEnd()
	if *replayStart != "" {
		start, err = time.Parse(time.RFC3339Nano, *replayStart)
		if err != nil {
			return errors.New("invalid --from timestamp")
		}
		start = start.UTC()
	}
	if *replayEnd != "" {
		end, err = time.Parse(time.RFC3339Nano, *replayEnd)
		if err != nil {
			return errors.New("invalid --to timestamp")
		}
		end = end.UTC()
	}
	downloader, err := massive.NewOfflineDownloader(*providerOrigin, func() (string, error) { return credential, nil }, client)
	if err != nil {
		return errors.New("configure offline aggregate downloader")
	}
	plan := replayartifact.CompletePlan{Binding: binding, Start: start, End: end, Workers: *workers, DestinationDirectory: *destination,
		Limits: replayartifact.Limits{MaximumNormalizedRecords: *maximumRecords, MaximumResponseBytes: *maximumResponseBytes,
			MaximumArtifactBytes: *maximumArtifactBytes, MaximumTemporaryBytes: *maximumTemporaryBytes,
			MaximumTemporaryFiles: *maximumTemporaryFiles, MaximumInMemoryRecords: *maximumRecords}}
	result := replayartifact.Compile(ctx, plan, downloader)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return errors.New("encode compile result")
	}
	if result.State != replayartifact.CompileComplete {
		return fmt.Errorf("aggregate replay compile ended %s/%s", result.State, result.Reason)
	}
	return nil
}

type offlineReplayConfig struct {
	tradingDate, replayStart, replayEnd  string
	referenceDirectory, artifact         string
	maximumArtifactBytes, maximumRecords int64
	queueCapacity, requiredReserve       int
	evaluationDelay                      time.Duration
	paceLogicalSeconds, paceWallSeconds  uint64
}

func runOfflineReplay(ctx context.Context, config offlineReplayConfig) error {
	if ctx == nil || config.replayStart == "" || config.replayEnd == "" || config.artifact == "" ||
		config.maximumArtifactBytes <= 0 || config.maximumRecords <= 0 || config.queueCapacity <= 1 ||
		config.requiredReserve < 1 || config.requiredReserve >= config.queueCapacity || config.evaluationDelay < 0 {
		return errors.New("replay mode requires --artifact, --from, --to, --queue-capacity, --required-reserve, and nonnegative --evaluation-delay")
	}
	start, err := time.Parse(time.RFC3339Nano, config.replayStart)
	if err != nil || start != start.UTC() || start.Nanosecond() != 0 {
		return errors.New("replay --from must be a UTC whole-second RFC3339 timestamp")
	}
	end, err := time.Parse(time.RFC3339Nano, config.replayEnd)
	if err != nil || end != end.UTC() || end.Nanosecond() != 0 || !start.Before(end) {
		return errors.New("replay --to must be a later UTC whole-second RFC3339 timestamp")
	}
	pace := replay.Unpaced()
	if config.paceLogicalSeconds != 0 || config.paceWallSeconds != 0 {
		pace, err = replay.FinitePace(config.paceLogicalSeconds, config.paceWallSeconds)
		if err != nil {
			return errors.New("invalid finite replay pace")
		}
	}
	binding, err := cachedBinding(ctx, config.tradingDate, config.referenceDirectory)
	if err != nil {
		return err
	}
	handle, err := replayartifact.OpenValidated(config.artifact, replayartifact.ValidationPlan{
		Binding: binding, Start: start, End: end, ExpectedMode: replayartifact.CompleteFinalBars,
		MaximumBytes: config.maximumArtifactBytes, MaximumRecords: config.maximumRecords,
	})
	if err != nil {
		return errors.New("validate complete replay artifact")
	}
	defer handle.Close()
	clock, err := replay.NewSimulatedClock(start)
	if err != nil {
		return errors.New("initialize replay simulated clock")
	}
	owner, err := engine.New(engine.Config{Mode: engine.RunModeReplay, Clock: clock.Now, Capacity: config.queueCapacity,
		RequiredReserve: config.requiredReserve, EvaluationDelay: &config.evaluationDelay})
	if err != nil {
		return errors.New("configure replay engine")
	}
	admission, completion := owner.AdmitBinding(ctx, engine.BindingInstall{SchemaVersion: engine.BindingInstallSchemaV1,
		BindingIdentity: binding.Identity(), Binding: binding})
	if admission != engine.AdmissionAdmitted || completion == nil {
		return errors.New("install replay binding")
	}
	disposition, ok := <-completion
	if !ok || disposition.Code != engine.DispositionBindingInstalled {
		return errors.New("replay binding installation failed")
	}
	source, err := replay.NewSource(handle, owner, clock, pace)
	if err != nil {
		return errors.New("construct replay source")
	}
	result, runErr := source.Run(ctx)
	if runErr != nil && result.Outcome == "" {
		cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
		result, err = source.Cancel(cleanupContext)
		cancelCleanup()
		if err != nil {
			return fmt.Errorf("cancel replay source after run error: %w", err)
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return errors.New("encode replay result")
	}
	if result.Outcome != replay.OutcomeComplete {
		if runErr != nil {
			return fmt.Errorf("aggregate replay ended %s/%s: %w", result.Outcome, result.Reason, runErr)
		}
		return fmt.Errorf("aggregate replay ended %s/%s", result.Outcome, result.Reason)
	}
	if runErr != nil {
		return fmt.Errorf("aggregate replay completed after run error: %w", runErr)
	}
	return nil
}

func cachedBinding(ctx context.Context, tradingDate, directory string) (reference.Binding, error) {
	schedule, err := session.Load()
	if err != nil {
		return reference.Binding{}, errors.New("load accepted exchange schedule")
	}
	facts, err := schedule.ForTradingDate(tradingDate)
	if err != nil {
		return reference.Binding{}, errors.New("trading date is not supported by the accepted schedule")
	}
	// Empty credentials deliberately disable fresh acquisition. Component 1
	// then accepts only its validated exact-date private caches.
	universe, err := (&reference.Resolver{DataDir: directory, Schedule: schedule}).Resolve(ctx, facts)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("load exact-date universe from Component 1 cache: %w", err)
	}
	if !universe.IsCurrent() {
		return reference.Binding{}, errors.New("Component 1 universe cache is not exact-date current evidence")
	}
	priors, err := (&reference.PriorCloseResolver{DataDir: directory, Schedule: schedule}).Resolve(ctx, facts, universe)
	if err != nil {
		return reference.Binding{}, fmt.Errorf("load exact-date prior closes from Component 1 cache: %w", err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		return reference.Binding{}, errors.New("assemble cached immutable session binding")
	}
	return binding, nil
}

func destinationOutsideIgnoredVar(destination string) bool {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return true
	}
	ignoredRoot, ok := repositoryIgnoredVarRoot(workingDirectory)
	if !ok {
		return true
	}
	if info, statErr := os.Lstat(ignoredRoot); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return true
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return true
	}
	resolvedRoot, err := resolveExistingPath(ignoredRoot)
	if err != nil {
		return true
	}
	resolvedDestination, err := resolveExistingPath(destination)
	if err != nil {
		return true
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedDestination)
	return err != nil || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func repositoryIgnoredVarRoot(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		module, moduleErr := os.ReadFile(filepath.Join(current, "go.mod"))
		ignore, ignoreErr := os.ReadFile(filepath.Join(current, ".gitignore"))
		if moduleErr == nil && ignoreErr == nil && strings.HasPrefix(string(module), "module github.com/belandresj/live-equities-momentum-scanner\n") {
			for _, line := range strings.Split(string(ignore), "\n") {
				if strings.TrimSpace(line) == "var/" {
					return filepath.Join(current, "var"), true
				}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func resolveExistingPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	current := absolute
	var missing []string
	for {
		resolved, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr == nil {
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(resolveErr, os.ErrNotExist) {
			return "", resolveErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", resolveErr
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
