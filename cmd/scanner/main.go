package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

const defaultLiveHydrationWorkers = 8

const liveHydrationResponseByteBudget = int64(4 << 30)

type ingressDiagnosticRecorder struct {
	attempts  int
	persisted bool
	lastErr   error
}

func (r *ingressDiagnosticRecorder) record(directory string, incident *operations.IngressIncident, output io.Writer, final bool, persist func(string, *operations.IngressIncident) (string, error)) error {
	if incident == nil || r.persisted {
		return nil
	}
	limit := 2
	if final {
		limit = 3
	}
	if r.attempts >= limit {
		if final {
			return r.lastErr
		}
		return nil
	}
	r.attempts++
	path, err := persist(directory, incident)
	if err != nil {
		r.lastErr = err
		fmt.Fprintf(output, "Ingress diagnostic persistence failed · attempt %d/%d · %v\n", r.attempts, limit, err)
		if final {
			return err
		}
		return nil
	}
	r.persisted = true
	r.lastErr = nil
	fmt.Fprintf(output, "Ingress diagnostic persisted · %s\n", path)
	return nil
}

func productionLiveQueueConfig() massive.LiveQueueConfig {
	return massive.LiveQueueConfig{FrameSlots: massive.MaximumLiveFrameSlots, MaxFrameBytes: 8 << 20, TotalFrameBytes: massive.MaximumLiveQueueBytes}
}

func main() {
	installScannerSignalHandling()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func installScannerSignalHandling() {
	// A closed terminal is an observability failure, never a scanner lifecycle
	// event. Ignore SIGPIPE so writes surface as ordinary failed descriptors.
	signal.Ignore(syscall.SIGPIPE)
}

func run(ctx context.Context, arguments []string) error {
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	if _, err := scalarFlags(arguments); err != nil {
		return err
	}
	flags := flag.NewFlagSet("scanner", flag.ContinueOnError)
	tradingDate := flags.String("trading-date", "", "exchange-local trading date YYYY-MM-DD")
	referenceDirectory := flags.String("reference-dir", filepath.Join("var", "reference"), "Component 1 cache directory")
	diagnosticDirectory := flags.String("diagnostic-dir", filepath.Join("var", "diagnostics"), "private local bounded incident directory")
	restOrigin := flags.String("rest-origin", "https://api.massive.com", "Massive HTTPS origin")
	websocketEndpoint := flags.String("websocket-endpoint", "wss://socket.massive.com/stocks", "Massive stocks WebSocket endpoint")
	hydrationWorkers := flags.Int("hydration-workers", defaultLiveHydrationWorkers, "live aggregate REST hydration workers: 1, 2, 4, or 8 (default 8)")
	apiAddress := flags.String("api-address", snapshotapi.DefaultAddress, "private loopback snapshot API address")
	var allowedOrigins originFlags
	flags.Var(&allowedOrigins, "allow-origin", "exact browser origin allowed to read the snapshot API; repeatable")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("scanner flags are invalid")
	}
	if *tradingDate == "" {
		return errors.New("live scanner requires one trading date")
	}
	if *diagnosticDirectory == "" {
		return errors.New("diagnostic-dir is required")
	}
	maximumNormalizedRecords, maximumResidentRecords, err := liveHydrationBounds(*hydrationWorkers, 1)
	if err != nil {
		return err
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
		return errors.New("unsupported trading date")
	}
	client := &http.Client{}
	universe, err := (&reference.Resolver{BaseURL: *restOrigin, APIKey: credential, DataDir: *referenceDirectory, HTTPClient: client, Schedule: schedule}).Resolve(runCtx, facts)
	if err != nil {
		return fmt.Errorf("resolve universe: %w", err)
	}
	priors, err := (&reference.PriorCloseResolver{BaseURL: *restOrigin, APIKey: credential, DataDir: *referenceDirectory, HTTPClient: client, Schedule: schedule}).Resolve(runCtx, facts, universe)
	if err != nil {
		return fmt.Errorf("resolve prior closes: %w", err)
	}
	binding, err := reference.AssembleBinding(facts, universe, priors)
	if err != nil {
		return errors.New("assemble immutable binding")
	}
	floatCtx, cancelFloat := context.WithTimeout(runCtx, 15*time.Second)
	floatLookup := (&reference.FloatResolver{BaseURL: *restOrigin, APIKey: credential, DataDir: *referenceDirectory, HTTPClient: client}).Resolve(floatCtx, universe)
	cancelFloat()
	runtimeConfig := operations.DefaultConfig()
	runtimeConfig.FloatLookup = floatLookup
	startup, cancelStartup := context.WithTimeout(runCtx, 30*time.Second)
	runtime, err := operations.New(startup, binding, runtimeConfig, func() time.Time { return time.Now().UTC() })
	cancelStartup()
	if err != nil {
		return err
	}
	adapter, err := massive.NewLiveAdapter(binding, massive.LiveAdapterConfig{Endpoint: *websocketEndpoint, Credential: credential, Queue: productionLiveQueueConfig()})
	if err != nil {
		_ = runtime.Shutdown(context.Background())
		return err
	}
	hydrator, err := massive.NewHydrationWorker(*restOrigin, func() (string, error) { return credential, nil }, client)
	if err != nil {
		_ = runtime.Shutdown(context.Background())
		return err
	}
	maximumNormalizedRecords, maximumResidentRecords, err = liveHydrationBounds(*hydrationWorkers, len(binding.UniverseSymbols()))
	if err != nil {
		_ = runtime.Shutdown(context.Background())
		return err
	}
	components := operations.LiveComponents{Adapter: adapter, Hydrator: hydrator, Workers: *hydrationWorkers, RowsPerChunk: 256, MaximumResponseBytes: liveHydrationResponseByteBudget,
		MaximumNormalizedRecords: maximumNormalizedRecords, MaximumResidentRecords: maximumResidentRecords,
		Durations: massive.OperationalDurations{Dial: 10 * time.Second, HandshakeStep: 5 * time.Second, HandshakeTotal: 30 * time.Second, HeartbeatInterval: 15 * time.Second, HeartbeatDeadline: 5 * time.Second, Write: 5 * time.Second, Close: 5 * time.Second}}
	if err := operations.ValidateLiveComponents(components); err != nil {
		_ = runtime.Shutdown(context.Background())
		return err
	}
	api, done, err := startLiveAfterAPI(func() (*apiSupervisor, error) {
		return newAPISupervisor(runtime, snapshotapi.ServerConfig{Address: *apiAddress, AllowedOrigins: []string(allowedOrigins)}, func(source snapshotapi.CaptureSource, config snapshotapi.ServerConfig) (snapshotServer, error) {
			return snapshotapi.Listen(source, config)
		}, nil)
	}, func() <-chan error {
		done := make(chan error, 1)
		go func() { done <- runtime.RunLive(runCtx, components) }()
		return done
	})
	if err != nil {
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = runtime.Shutdown(shutdown)
		return fmt.Errorf("start snapshot API: %w", err)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	readinessTicker := time.NewTicker(watermarkStaleReadinessCadence)
	defer readinessTicker.Stop()
	stdout, stderr := newTerminalSinks(os.Stdout, os.Stderr)
	operator := newOperatorRenderer(stdout, stderr)
	watermarkStaleRecorder := &watermarkStaleDiagnosticRecorder{}
	observeWatermarkStale := func() error {
		// Status records the scanner-local observation. A concurrent /readyz
		// capture records into the same bounded transition latch, so whichever
		// path sees the crossing first cannot be erased by timer phase.
		_ = runtime.Status()
		transition, ok := runtime.ObserveWatermarkStaleTransition()
		if !ok {
			return nil
		}
		return watermarkStaleRecorder.observeTransition(transition, runtime.ObserveWatermarkStallEvidence, *diagnosticDirectory, stderr, persistWatermarkStaleDiagnostic)
	}
	if sample, err := captureLiveOperatorSample(runtime); err == nil {
		_ = observeWatermarkStale()
		_ = operator.Render(sample, true)
	}
	diagnosticEncoder := json.NewEncoder(stderr)
	diagnosticRecorder := ingressDiagnosticRecorder{}
	return superviseLive(liveSupervisorConfig{
		context: runCtx, api: api, done: done, ticks: ticker.C, readinessTicks: readinessTicker.C,
		capture:               func() (liveOperatorSample, error) { return captureLiveOperatorSample(runtime) },
		observeWatermarkStale: observeWatermarkStale,
		render:                operator.Render,
		record: func(incident *operations.IngressIncident, final bool) error {
			if incident == nil {
				incident = runtime.FirstIngressIncident()
			}
			return diagnosticRecorder.record(*diagnosticDirectory, incident, stderr, final, persistIngressIncident)
		},
		encodeMapping: func(failure snapshotapi.MappingFailure) error {
			return encodeSnapshotMappingFailure(diagnosticEncoder, failure)
		},
		shutdown: func(liveJoined bool) error {
			return joinAndShutdownSupervisor(runtime, api, done, cancelRun, liveJoined)
		},
		onUnavailable: func(restartErr error) {
			_, _ = fmt.Fprintf(stderr, "Snapshot API unavailable after bounded restart attempts: %v\n", restartErr)
		},
	})
}

func captureLiveOperatorSample(runtime *operations.Runtime) (liveOperatorSample, error) {
	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		return liveOperatorSample{}, err
	}
	view, valid := operations.InspectSnapshotCapture(capture)
	if !valid {
		return liveOperatorSample{}, errors.New("invalid operational snapshot")
	}
	evaluation := view.Engine.Publication.AggregateEvaluation
	return liveOperatorSample{Status: view.Status, Metrics: view.Metrics, IngressIncident: view.IngressIncident, RecoveryAttempt: view.RecoveryAttempt, Evaluation: evaluation, Ranked: len(evaluation.Rows)}, nil
}

func encodeSnapshotMappingFailure(encoder *json.Encoder, failure snapshotapi.MappingFailure) error {
	return encoder.Encode(struct {
		SnapshotMappingFailure snapshotapi.MappingFailure `json:"snapshot_mapping_failure"`
	}{SnapshotMappingFailure: failure})
}

func liveHydrationBounds(workers, population int) (maximumNormalizedRecords, maximumResidentRecords int64, err error) {
	if population <= 0 {
		return 0, 0, errors.New("live hydration population is invalid")
	}
	switch workers {
	case 1, 2, 4, 8:
	default:
		return 0, 0, errors.New("hydration-workers must be one of 1, 2, 4, or 8")
	}
	if int64(population) > math.MaxInt64/57_600 {
		return 0, 0, errors.New("live hydration population exceeds normalized-record budget")
	}
	return int64(population) * 57_600, int64(workers) * 57_600, nil
}

func scalarFlags(arguments []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "--" || !strings.HasPrefix(argument, "-") {
			continue
		}
		name := strings.TrimLeft(argument, "-")
		if split := strings.IndexByte(name, '='); split >= 0 {
			name = name[:split]
		} else if index+1 < len(arguments) {
			index++
		}
		if name == "allow-origin" {
			continue
		}
		if result[name] {
			return nil, fmt.Errorf("duplicate --%s flag", name)
		}
		result[name] = true
	}
	return result, nil
}

func joinAndShutdown(runtime *operations.Runtime, api *snapshotapi.Server, liveDone, apiDone <-chan error, cancelRun context.CancelFunc, liveJoined, apiJoined bool) error {
	cancelRun()
	apiDeadline, cancelAPI := context.WithTimeout(context.Background(), 10*time.Second)
	apiShutdownErr := api.Shutdown(apiDeadline)
	cancelAPI()
	var apiJoinErr error
	if !apiJoined {
		joinDeadline, cancelJoin := context.WithTimeout(context.Background(), time.Second)
		select {
		case <-apiDone:
		case <-joinDeadline.Done():
			apiJoinErr = errors.New("snapshot API shutdown deadline exceeded")
		}
		cancelJoin()
	}
	liveDeadline, cancelLive := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLive()
	var liveJoinErr error
	if !liveJoined {
		select {
		case <-liveDone:
		case <-liveDeadline.Done():
			liveJoinErr = errors.New("live composition shutdown deadline exceeded")
		}
	}
	runtimeErr := runtime.Shutdown(liveDeadline)
	if apiShutdownErr != nil {
		return apiShutdownErr
	}
	if apiJoinErr != nil {
		return apiJoinErr
	}
	if liveJoinErr != nil {
		return liveJoinErr
	}
	return runtimeErr
}

func joinAndShutdownSupervisor(runtime *operations.Runtime, api *apiSupervisor, liveDone <-chan error, cancelRun context.CancelFunc, liveJoined bool) error {
	cancelRun()
	apiDeadline, cancelAPI := context.WithTimeout(context.Background(), 10*time.Second)
	apiErr := api.Shutdown(apiDeadline)
	cancelAPI()
	liveDeadline, cancelLive := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLive()
	var liveErr error
	if !liveJoined {
		select {
		case <-liveDone:
		case <-liveDeadline.Done():
			liveErr = errors.New("live composition shutdown deadline exceeded")
		}
	}
	runtimeErr := runtime.Shutdown(liveDeadline)
	return errors.Join(apiErr, liveErr, runtimeErr)
}

type originFlags []string

func (values *originFlags) String() string { return strings.Join(*values, ",") }

func (values *originFlags) Set(value string) error {
	*values = append(*values, value)
	return nil
}
