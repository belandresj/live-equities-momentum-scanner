package privatelauncher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/session"
)

const (
	scannerAddress   = "127.0.0.1:8080"
	dashboardAddress = "127.0.0.1:4173"
	scannerOrigin    = "http://" + scannerAddress
	dashboardOrigin  = "http://" + dashboardAddress
	keychainAccount  = "joshuabelandres"
	keychainService  = "momentum-scanner-massive-api"
	preconnectLead   = 5 * time.Minute
	dashboardRetries = 3
)

var dashboardRetryDelays = [...]time.Duration{time.Second, 2 * time.Second, 4 * time.Second}

// Run starts the private daily scanner workflow rooted at repoRoot. The
// repository-owned wrapper resolves repoRoot from its own location so this
// function never depends on the caller's working directory.
func Run(ctx context.Context, repoRoot string, arguments []string, stdout, stderr io.Writer) error {
	return run(ctx, repoRoot, arguments, stdout, stderr, productionDependencies())
}

type signalCancellation struct{ signal os.Signal }

func (cause signalCancellation) Error() string { return "received " + cause.signal.String() }

// NotifyContext preserves whether the foreground launcher received Ctrl-C or
// SIGTERM so the supervisor can forward that exact signal to both children.
func NotifyContext(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(parent)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case received := <-signals:
			cancel(signalCancellation{signal: received})
		case <-done:
		}
	}()
	var once sync.Once
	return ctx, func() {
		once.Do(func() {
			signal.Stop(signals)
			close(done)
			cancel(context.Canceled)
		})
	}
}

type options struct {
	tradingDate      string
	hydrationWorkers int
	open             bool
	help             bool
}

func parseOptions(arguments []string) (options, error) {
	result := options{hydrationWorkers: 1}
	seenDate, seenWorkers, seenOpen, seenHelp := false, false, false, false
	for index := 0; index < len(arguments); index++ {
		switch arguments[index] {
		case "--trading-date":
			if seenDate {
				return options{}, errors.New("--trading-date may be supplied only once")
			}
			if index+1 >= len(arguments) {
				return options{}, errors.New("--trading-date requires YYYY-MM-DD")
			}
			seenDate = true
			index++
			result.tradingDate = arguments[index]
		case "--hydration-workers":
			if seenWorkers {
				return options{}, errors.New("--hydration-workers may be supplied only once")
			}
			if index+1 >= len(arguments) {
				return options{}, errors.New("--hydration-workers requires 1")
			}
			seenWorkers = true
			index++
			if arguments[index] != "1" {
				return options{}, errors.New("--hydration-workers must be exactly 1")
			}
			result.hydrationWorkers = 1
		case "--open":
			if seenOpen {
				return options{}, errors.New("--open may be supplied only once")
			}
			seenOpen, result.open = true, true
		case "--help":
			if seenHelp {
				return options{}, errors.New("--help may be supplied only once")
			}
			seenHelp, result.help = true, true
		default:
			if strings.HasPrefix(arguments[index], "-") {
				return options{}, fmt.Errorf("unknown argument %q", arguments[index])
			}
			return options{}, fmt.Errorf("positional argument %q is not supported", arguments[index])
		}
	}
	if result.help && (seenDate || seenWorkers || seenOpen) {
		return options{}, errors.New("--help cannot be combined with startup arguments")
	}
	if seenDate {
		parsed, err := time.Parse("2006-01-02", result.tradingDate)
		if err != nil || parsed.Format("2006-01-02") != result.tradingDate {
			return options{}, errors.New("--trading-date must be a valid date in YYYY-MM-DD form")
		}
	}
	return result, nil
}

func printHelp(writer io.Writer) {
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  ./scripts/run-private-scanner [--trading-date YYYY-MM-DD] [--hydration-workers 1] [--open]")
}

type launchPaths struct {
	reference, checkpoints, runtimeDirectory string
	scannerBinary, dashboardBinary           string
}

type processSpec struct {
	name, path       string
	arguments        []string
	environment      []string
	workingDirectory string
	stdout, stderr   io.Writer
}

type childProcess interface {
	Signal(os.Signal) error
	Kill() error
	Done() <-chan error
}

type probeResult struct {
	status int
	body   []byte
}

type dependencies struct {
	now                   func() time.Time
	after                 func(time.Duration) <-chan time.Time
	baseEnvironment       func() []string
	credentialEnvironment func() []string
	preflight             func(context.Context, string, io.Writer, io.Writer) (launchPaths, error)
	credential            func(context.Context, []string) (string, string, error)
	start                 func(processSpec) (childProcess, error)
	probe                 func(context.Context, string) (probeResult, error)
	open                  func(context.Context, string, []string, io.Writer, io.Writer) error
	available             func(string) error
	standbyRecheck        time.Duration
	scannerStartup        time.Duration
	dashboardStartup      time.Duration
	pollInterval          time.Duration
	scannerShutdown       time.Duration
	dashboardShutdown     time.Duration
}

// safeOutput makes launcher progress reporting best effort. In particular, a
// closed foreground pipe must not turn a reporting failure into child
// containment. Returning a successful full write also keeps os/exec's output
// copier draining child pipes after the terminal has gone away.
type safeOutput struct {
	mu       sync.Mutex
	writer   io.Writer
	disabled bool
}

func newSafeOutput(writer io.Writer) *safeOutput { return &safeOutput{writer: writer} }

func (output *safeOutput) Write(data []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	if output.disabled {
		return len(data), nil
	}
	written, err := output.writer.Write(data)
	if err != nil || written != len(data) {
		output.disabled = true
	}
	return len(data), nil
}

func run(ctx context.Context, repoRoot string, arguments []string, stdout, stderr io.Writer, deps dependencies) error {
	if ctx == nil || stdout == nil || stderr == nil {
		return errors.New("private scanner launcher requires context and output streams")
	}
	stdout, stderr = newSafeOutput(stdout), newSafeOutput(stderr)
	parsed, err := parseOptions(arguments)
	if err != nil {
		return err
	}
	if parsed.help {
		printHelp(stdout)
		return nil
	}

	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return errors.New("load America/New_York timezone")
	}
	now := deps.now().In(location)
	schedule, err := session.Load()
	if err != nil {
		return errors.New("load accepted exchange schedule")
	}
	facts, err := resolveLaunchSession(schedule, now, parsed.tradingDate)
	if err != nil {
		return err
	}
	tradingDate := facts.TradingDate
	sessionStart, sessionEnd := facts.SessionStart.In(location), facts.SessionEnd.In(location)
	preconnectAt := sessionStart.Add(-preconnectLead)
	standby := now.Before(preconnectAt)

	paths, err := deps.preflight(ctx, repoRoot, stdout, stderr)
	if err != nil {
		return fmt.Errorf("preflight: %w", err)
	}
	// Preflight can compile both children and may cross the selected session's
	// preconnect boundary. Re-resolve from the current wall clock before any
	// optional dashboard work so it cannot delay the ordinary scanner start.
	now = deps.now().In(location)
	facts, err = resolveLaunchSession(schedule, now, parsed.tradingDate)
	if err != nil {
		return err
	}
	tradingDate = facts.TradingDate
	sessionStart, sessionEnd = facts.SessionStart.In(location), facts.SessionEnd.In(location)
	preconnectAt = sessionStart.Add(-preconnectLead)
	standby = now.Before(preconnectAt)
	baseEnvironment := deps.baseEnvironment()
	dashboardArguments := []string{
		"--address", dashboardAddress,
		"--api-origin", scannerOrigin,
		"--assets", filepath.Join(repoRoot, "ui"),
	}
	var scanner, dashboard childProcess
	dashboardRecovered := false
	if standby {
		fmt.Fprintf(stdout, "Overnight standby for trading session %s. Dashboard starts now; scanner credential, reference, and provider work begin at %s.\n", tradingDate, preconnectAt.Format("2006-01-02 15:04 MST"))
		// Do not let a listener-health deadline cross the scanner's fixed
		// preconnect boundary. A dashboard is optional during standby; once its
		// full health budget no longer fits, scanner startup takes precedence.
		if now.Add(dashboardStandbyBudget(deps)).Before(preconnectAt) {
			dashboard, err = startDashboard(ctx, paths.dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
			if err != nil {
				dashboardRecovered = true
				fmt.Fprintf(stderr, "WARNING: dashboard unavailable during standby: %v; standby and the scheduled scanner start continue without a dashboard.\n", err)
				if dashboard != nil {
					if stopErr := signalAndWait(dashboard, containmentSignal(ctx), deps.dashboardShutdown, "dashboard"); stopErr == nil {
						dashboard = nil
					} else {
						fmt.Fprintf(stderr, "WARNING: standby dashboard could not be reaped: %v; it remains supervised without a health claim.\n", stopErr)
					}
				}
			}
		} else {
			dashboardRecovered = true
			fmt.Fprintln(stderr, "WARNING: dashboard standby start deferred to preserve the scheduled scanner preconnect boundary.")
		}
		if dashboard != nil {
			fmt.Fprintf(stdout, "Scanner dashboard: %s (scanner disconnected during standby)\n", dashboardOrigin)
		}
		if parsed.open && dashboard != nil && !dashboardRecovered {
			if err := deps.open(ctx, dashboardOrigin, baseEnvironment, stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "WARNING: could not open the dashboard browser: %v; standby remains active.\n", err)
			}
		}
		for {
			priorTradingDate, priorPreconnectAt := tradingDate, preconnectAt
			dashboardDone, waitErr := waitForPreconnect(ctx, dashboard, preconnectAt, deps)
			if waitErr != nil {
				stopErr := stopChildren(nil, dashboard, false, dashboardDone || dashboard == nil, containmentSignal(ctx), deps)
				if errors.Is(waitErr, context.Canceled) {
					return stopErr
				}
				return errors.Join(waitErr, stopErr)
			}
			if dashboardDone {
				dashboard = nil
				dashboardRecovered = true
				fmt.Fprintln(stderr, "WARNING: dashboard exited during standby; standby and the scheduled scanner start continue without a dashboard.")
				continue
			}
			wakeNow := deps.now().In(location)
			refreshed, refreshErr := resolveLaunchSession(schedule, wakeNow, parsed.tradingDate)
			if refreshErr != nil {
				return errors.Join(refreshErr, stopChildren(nil, dashboard, false, false, syscall.SIGTERM, deps))
			}
			facts = refreshed
			tradingDate = facts.TradingDate
			sessionStart, sessionEnd = facts.SessionStart.In(location), facts.SessionEnd.In(location)
			preconnectAt = sessionStart.Add(-preconnectLead)
			if wakeNow.Before(preconnectAt) {
				if tradingDate != priorTradingDate || !preconnectAt.Equal(priorPreconnectAt) {
					fmt.Fprintf(stdout, "Standby continues for trading session %s; scanner work is scheduled for %s.\n", tradingDate, preconnectAt.Format("2006-01-02 15:04 MST"))
				}
				continue
			}
			break
		}
		if err := deps.available(scannerAddress); err != nil {
			return errors.Join(fmt.Errorf("local port %s became occupied during standby", scannerAddress), stopChildren(nil, dashboard, false, false, syscall.SIGTERM, deps))
		}
		fmt.Fprintf(stdout, "Standby complete; starting scanner for %s.\n", tradingDate)
	} else if !now.Before(sessionStart) {
		fmt.Fprintln(stderr, "Warning: starting after 04:00 EST. Will start historical data fetches to ready scanner")
	} else {
		fmt.Fprintln(stdout, "Starting before 04:00 America/New_York; authoritative readiness is expected only after the session begins and required hydration/fencing completes.")
	}

	environment := deps.credentialEnvironment()
	credential, source, err := deps.credential(ctx, environment)
	if err != nil {
		credentialErr := errors.New("Massive credential unavailable; export MASSIVE_API_KEY or create the macOS Keychain generic-password item for account joshuabelandres and service momentum-scanner-massive-api")
		if dashboard != nil {
			return errors.Join(credentialErr, stopChildren(nil, dashboard, false, false, syscall.SIGTERM, deps))
		}
		return credentialErr
	}
	defer zeroString(&credential)
	fmt.Fprintf(stdout, "Credential loaded from %s; it will be passed only in the scanner child environment.\n", source)
	scannerArguments := []string{
		"--run-mode", "live",
		"--trading-date", tradingDate,
		"--hydration-workers", fmt.Sprint(parsed.hydrationWorkers),
		"--reference-dir", paths.reference,
		"--checkpoint-dir", paths.checkpoints,
		"--checkpoint-mode", "off",
		"--api-address", scannerAddress,
		"--allow-origin", dashboardOrigin,
	}
	scanner, err = deps.start(processSpec{name: "scanner", path: paths.scannerBinary, arguments: scannerArguments,
		environment: append(append([]string(nil), baseEnvironment...), "MASSIVE_API_KEY="+credential), workingDirectory: repoRoot, stdout: stdout, stderr: stderr})
	credential = ""
	if err != nil {
		return errors.Join(fmt.Errorf("start scanner: %w", err), stopChildren(nil, dashboard, false, false, syscall.SIGTERM, deps))
	}
	if err := waitForLive(ctx, scanner, scannerOrigin+"/livez", deps); err != nil {
		return errors.Join(err, stopChildren(scanner, dashboard, false, false, containmentSignal(ctx), deps))
	}
	if dashboard == nil {
		dashboard, err = startDashboard(ctx, paths.dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
		if err != nil {
			dashboardRecovered = true
			dashboard, err = recoverDashboardAfterFailure(ctx, scanner, dashboard, err, paths.dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
		}
		if err != nil {
			stopErr := stopChildren(scanner, dashboard, false, dashboard == nil, containmentSignal(ctx), deps)
			if errors.Is(err, context.Canceled) {
				return stopErr
			}
			return errors.Join(err, stopErr)
		}
	}

	fmt.Fprintf(stdout, "Scanner snapshot: %s/api/v2/snapshot\n", scannerOrigin)
	fmt.Fprintf(stdout, "Scanner liveness: %s/livez\n", scannerOrigin)
	fmt.Fprintf(stdout, "Scanner readiness: %s/readyz\n", scannerOrigin)
	if parsed.open && !standby && !dashboardRecovered && dashboard != nil {
		if err := deps.open(ctx, dashboardOrigin, baseEnvironment, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "WARNING: could not open the dashboard browser: %v; services remain running.\n", err)
		}
	}
	return supervise(ctx, scanner, dashboard, sessionEnd, paths.dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
}

func resolveLaunchSession(schedule *session.Schedule, now time.Time, explicit string) (session.Facts, error) {
	if schedule == nil {
		return session.Facts{}, errors.New("accepted exchange schedule is required")
	}
	if explicit != "" {
		facts, err := schedule.ForTradingDate(explicit)
		if err != nil {
			return session.Facts{}, fmt.Errorf("unsupported trading date %s", explicit)
		}
		if !now.Before(facts.SessionEnd) {
			return session.Facts{}, fmt.Errorf("the explicit %s scanner session ended at 20:00 America/New_York; historical and replay workflows are separate", explicit)
		}
		return facts, nil
	}
	today := now.Format("2006-01-02")
	if facts, err := schedule.ForTradingDate(today); err == nil && now.Before(facts.SessionEnd) {
		return facts, nil
	}
	candidate := today
	if facts, err := schedule.ForTradingDate(today); err == nil && !now.Before(facts.SessionEnd) {
		candidate = now.AddDate(0, 0, 1).Format("2006-01-02")
	}
	tradingDate, err := schedule.TradingDateOnOrAfter(candidate)
	if err != nil {
		return session.Facts{}, fmt.Errorf("resolve next trading date: %w", err)
	}
	facts, err := schedule.ForTradingDate(tradingDate)
	if err != nil {
		return session.Facts{}, errors.New("resolve declared trading session")
	}
	return facts, nil
}

func waitForPreconnect(ctx context.Context, dashboard childProcess, target time.Time, deps dependencies) (bool, error) {
	delay := target.Sub(deps.now())
	if delay <= 0 {
		return false, nil
	}
	if deps.standbyRecheck > 0 && delay > deps.standbyRecheck {
		delay = deps.standbyRecheck
	}
	var dashboardDone <-chan error
	if dashboard != nil {
		dashboardDone = dashboard.Done()
	}
	select {
	case <-dashboardDone:
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-deps.after(delay):
		select {
		case <-dashboardDone:
			return true, nil
		default:
			return false, nil
		}
	}
}

func waitForLive(ctx context.Context, child childProcess, url string, deps dependencies) error {
	deadline := time.NewTimer(deps.scannerStartup)
	defer deadline.Stop()
	ticker := time.NewTicker(deps.pollInterval)
	defer ticker.Stop()
	for {
		probeCtx, cancel := context.WithTimeout(ctx, minDuration(2*time.Second, deps.pollInterval*4))
		result, err := deps.probe(probeCtx, url)
		cancel()
		if err == nil && result.status == http.StatusOK {
			var live struct {
				ProcessLive bool `json:"process_live"`
			}
			if json.Unmarshal(result.body, &live) == nil && live.ProcessLive {
				return nil
			}
		}
		select {
		case childErr := <-child.Done():
			return fmt.Errorf("scanner exited before /livez became successful: %w", normalizeExit(childErr))
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("scanner /livez did not become successful within 5 minutes")
		case <-ticker.C:
		}
	}
}

func waitForDashboard(ctx context.Context, child childProcess, url string, deps dependencies) error {
	deadline := time.NewTimer(deps.dashboardStartup)
	defer deadline.Stop()
	ticker := time.NewTicker(deps.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case childErr := <-child.Done():
			return fmt.Errorf("dashboard exited before its listener became healthy: %w", normalizeExit(childErr))
		default:
		}
		probeCtx, cancel := context.WithTimeout(ctx, minDuration(2*time.Second, deps.pollInterval*4))
		result, err := deps.probe(probeCtx, url)
		cancel()
		if err == nil && result.status == http.StatusOK {
			select {
			case childErr := <-child.Done():
				return fmt.Errorf("dashboard exited before its listener became healthy: %w", normalizeExit(childErr))
			default:
				return nil
			}
		}
		select {
		case childErr := <-child.Done():
			return fmt.Errorf("dashboard exited before its listener became healthy: %w", normalizeExit(childErr))
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("dashboard listener did not become healthy within 30 seconds")
		case <-ticker.C:
		}
	}
}

func startDashboard(ctx context.Context, dashboardBinary string, dashboardArguments, baseEnvironment []string, repoRoot string, stdout, stderr io.Writer, deps dependencies) (childProcess, error) {
	dashboard, err := deps.start(processSpec{name: "dashboard", path: dashboardBinary, arguments: append([]string(nil), dashboardArguments...),
		environment: append([]string(nil), baseEnvironment...), workingDirectory: repoRoot, stdout: stdout, stderr: stderr})
	if err != nil {
		return nil, fmt.Errorf("start dashboard: %w", err)
	}
	if err := waitForDashboard(ctx, dashboard, dashboardOrigin+"/", deps); err != nil {
		return dashboard, err
	}
	return dashboard, nil
}

func startDashboardWithRecovery(ctx context.Context, scanner childProcess, dashboardBinary string, dashboardArguments, baseEnvironment []string, repoRoot string, stdout, stderr io.Writer, deps dependencies) (childProcess, error) {
	dashboard, err := startDashboard(ctx, dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
	if err == nil {
		return dashboard, nil
	}
	return recoverDashboardAfterFailure(ctx, scanner, dashboard, err, dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
}

// recoverDashboard starts at most dashboardRetries replacements after one
// dashboard-local failure. It intentionally never contains the scanner: a
// failed dashboard leaves the scanner running headless when recovery cannot
// establish a healthy listener.
func recoverDashboard(ctx context.Context, scanner, dashboard childProcess, dashboardBinary string, dashboardArguments, baseEnvironment []string, repoRoot string, stdout, stderr io.Writer, deps dependencies) (childProcess, error) {
	return recoverDashboardAfterFailure(ctx, scanner, dashboard, errors.New("dashboard unavailable"), dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
}

func recoverDashboardAfterFailure(ctx context.Context, scanner, dashboard childProcess, cause error, dashboardBinary string, dashboardArguments, baseEnvironment []string, repoRoot string, stdout, stderr io.Writer, deps dependencies) (childProcess, error) {
	if dashboard != nil {
		if err := ctx.Err(); err != nil {
			return dashboard, err
		}
		if err := signalAndWait(dashboard, syscall.SIGTERM, deps.dashboardShutdown, "dashboard"); err != nil {
			fmt.Fprintf(stderr, "WARNING: dashboard recovery could not reap the failed dashboard: %v; scanner remains supervised without a dashboard health claim.\n", err)
			return dashboard, nil
		}
	}
	fmt.Fprintf(stderr, "WARNING: dashboard unavailable: %v; retrying without interrupting the scanner.\n", cause)
	for attempt, delay := range dashboardRetryDelays {
		if err := waitForDashboardRetry(ctx, scanner, delay, deps); err != nil {
			return nil, err
		}
		fmt.Fprintf(stderr, "WARNING: dashboard recovery attempt %d of %d.\n", attempt+1, dashboardRetries)
		replacement, err := startDashboard(ctx, dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
		if err == nil {
			fmt.Fprintln(stdout, "Scanner dashboard: available after local recovery.")
			return replacement, nil
		}
		if replacement != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return replacement, contextErr
			}
			if stopErr := signalAndWait(replacement, syscall.SIGTERM, deps.dashboardShutdown, "dashboard"); stopErr != nil {
				fmt.Fprintf(stderr, "WARNING: dashboard recovery could not reap attempt %d: %v; scanner remains supervised without a dashboard health claim.\n", attempt+1, stopErr)
				return replacement, nil
			}
		}
		cause = err
	}
	fmt.Fprintf(stderr, "WARNING: dashboard recovery exhausted after %d attempts (%v); scanner remains running headless. Start cmd/dashboard independently after correcting the local dashboard failure.\n", dashboardRetries, cause)
	return nil, nil
}

func waitForDashboardRetry(ctx context.Context, scanner childProcess, delay time.Duration, deps dependencies) error {
	var scannerDone <-chan error
	if scanner != nil {
		scannerDone = scanner.Done()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case scannerErr := <-scannerDone:
		return fmt.Errorf("scanner exited unexpectedly: %w", normalizeExit(scannerErr))
	case <-deps.after(delay):
		return nil
	}
}

func supervise(ctx context.Context, scanner, dashboard childProcess, sessionEnd time.Time, dashboardBinary string, dashboardArguments, baseEnvironment []string, repoRoot string, stdout, stderr io.Writer, deps dependencies) error {
	ticker := time.NewTicker(deps.pollInterval)
	defer ticker.Stop()
	lastReadiness := ""
	reportReadiness := func() {
		probeCtx, cancel := context.WithTimeout(ctx, minDuration(2*time.Second, deps.pollInterval*4))
		defer cancel()
		result, err := deps.probe(probeCtx, scannerOrigin+"/readyz")
		message := "Scanner readiness: unavailable"
		if err == nil {
			var ready struct {
				BackendReady bool   `json:"backend_ready"`
				Reason       string `json:"reason"`
			}
			if json.Unmarshal(result.body, &ready) == nil {
				if result.status == http.StatusOK && ready.BackendReady {
					message = "Scanner readiness: ready"
				} else {
					reason := ready.Reason
					if reason == "" {
						reason = "runtime has not declared readiness"
					}
					message = "Scanner readiness: not ready (" + reason + ")"
				}
			}
		}
		if message != lastReadiness {
			fmt.Fprintln(stdout, message)
			lastReadiness = message
		}
	}
	reportReadiness()
	for {
		var dashboardDone <-chan error
		if dashboard != nil {
			dashboardDone = dashboard.Done()
		}
		select {
		case <-ctx.Done():
			return stopChildren(scanner, dashboard, false, false, containmentSignal(ctx), deps)
		case err := <-scanner.Done():
			stopErr := stopChildren(scanner, dashboard, true, false, syscall.SIGTERM, deps)
			if err == nil && !deps.now().Before(sessionEnd) {
				return stopErr
			}
			return errors.Join(fmt.Errorf("scanner exited unexpectedly: %w", normalizeExit(err)), stopErr)
		case err := <-dashboardDone:
			fmt.Fprintf(stderr, "WARNING: dashboard exited unexpectedly: %v; scanner supervision continues.\n", normalizeExit(err))
			dashboard = nil
			var recoveryErr error
			dashboard, recoveryErr = recoverDashboard(ctx, scanner, nil, dashboardBinary, dashboardArguments, baseEnvironment, repoRoot, stdout, stderr, deps)
			if recoveryErr != nil {
				if errors.Is(recoveryErr, context.Canceled) {
					return stopChildren(scanner, dashboard, false, dashboard == nil, containmentSignal(ctx), deps)
				}
				return errors.Join(recoveryErr, stopChildren(scanner, dashboard, false, dashboard == nil, syscall.SIGTERM, deps))
			}
		case <-ticker.C:
			reportReadiness()
		}
	}
}

func containmentSignal(ctx context.Context) os.Signal {
	var cause signalCancellation
	if ctx != nil && errors.As(context.Cause(ctx), &cause) && cause.signal != nil {
		return cause.signal
	}
	return syscall.SIGTERM
}

func stopChildren(scanner, dashboard childProcess, scannerDone, dashboardDone bool, shutdownSignal os.Signal, deps dependencies) error {
	var errs []error
	if scanner != nil && !scannerDone {
		if err := signalAndWait(scanner, shutdownSignal, deps.scannerShutdown, "scanner"); err != nil {
			errs = append(errs, err)
		}
	}
	if dashboard != nil && !dashboardDone {
		if err := signalAndWait(dashboard, shutdownSignal, deps.dashboardShutdown, "dashboard"); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func signalAndWait(child childProcess, signal os.Signal, limit time.Duration, name string) error {
	if err := child.Signal(signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
		// A failed graceful signal must not discard the only handle to a live
		// dashboard. Attempt bounded forced termination and reap it before
		// reporting the original signal failure.
		if killErr := child.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return fmt.Errorf("signal %s: %w; forced termination failed: %v", name, err, killErr)
		}
		select {
		case <-child.Done():
			return fmt.Errorf("signal %s: %w; forcibly terminated and reaped", name, err)
		case <-time.After(time.Second):
			return fmt.Errorf("signal %s: %w; could not be reaped after forced termination", name, err)
		}
	}
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case <-child.Done():
		return nil
	case <-timer.C:
		if err := child.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("%s exceeded graceful shutdown and could not be terminated: %w", name, err)
		}
		select {
		case <-child.Done():
			return fmt.Errorf("%s exceeded its graceful shutdown deadline and was forcibly terminated", name)
		case <-time.After(time.Second):
			return fmt.Errorf("%s could not be reaped after forced termination", name)
		}
	}
}

func normalizeExit(err error) error {
	if err == nil {
		return errors.New("exit status 0")
	}
	return err
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}

func dashboardStandbyBudget(deps dependencies) time.Duration {
	probeBudget := minDuration(2*time.Second, deps.pollInterval*4)
	if probeBudget <= 0 {
		probeBudget = 2 * time.Second
	}
	return deps.dashboardStartup + probeBudget + deps.dashboardShutdown
}

func zeroString(value *string) {
	if value != nil {
		*value = ""
	}
}

func removeEnvironment(environment []string, name string) []string {
	prefix := name + "="
	result := make([]string, 0, len(environment))
	for _, value := range environment {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func environmentValue(environment []string, name string) (string, bool) {
	prefix := name + "="
	for index := len(environment) - 1; index >= 0; index-- {
		if strings.HasPrefix(environment[index], prefix) {
			return strings.TrimPrefix(environment[index], prefix), true
		}
	}
	return "", false
}

type commandProcess struct {
	command *exec.Cmd
	done    chan error
}

func (process *commandProcess) Signal(signal os.Signal) error {
	return process.command.Process.Signal(signal)
}
func (process *commandProcess) Kill() error        { return process.command.Process.Kill() }
func (process *commandProcess) Done() <-chan error { return process.done }

func productionDependencies() dependencies {
	return dependencies{
		now:                   time.Now,
		after:                 time.After,
		baseEnvironment:       productionBaseEnvironment,
		credentialEnvironment: os.Environ,
		preflight:             productionPreflight,
		credential:            productionCredential,
		start:                 startCommand,
		probe:                 probeHTTP,
		open:                  openBrowser,
		available:             requireAvailablePort,
		standbyRecheck:        30 * time.Second,
		scannerStartup:        5 * time.Minute,
		dashboardStartup:      30 * time.Second,
		pollInterval:          time.Second,
		scannerShutdown:       15 * time.Second,
		dashboardShutdown:     5 * time.Second,
	}
}

func productionPreflight(ctx context.Context, repoRoot string, _ io.Writer, stderr io.Writer) (launchPaths, error) {
	if runtime.GOOS != "darwin" {
		return launchPaths{}, errors.New("the private daily launcher currently supports macOS only")
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil || repoRoot == "" {
		return launchPaths{}, errors.New("resolve repository root")
	}
	for _, required := range []string{"go.mod", filepath.Join("cmd", "scanner"), filepath.Join("cmd", "dashboard"), "ui"} {
		if _, err := os.Stat(filepath.Join(root, required)); err != nil {
			return launchPaths{}, fmt.Errorf("repository root is missing %s", required)
		}
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		return launchPaths{}, errors.New("Go 1.26 toolchain is required")
	}
	for _, address := range []string{scannerAddress, dashboardAddress} {
		if err := requireAvailablePort(address); err != nil {
			return launchPaths{}, fmt.Errorf("local port %s is already occupied", address)
		}
	}
	paths := launchPaths{
		reference:        filepath.Join(root, "var", "reference"),
		checkpoints:      filepath.Join(root, "var", "checkpoints"),
		runtimeDirectory: filepath.Join(root, "var", "run-private-scanner"),
	}
	for _, directory := range []string{paths.reference, paths.checkpoints, paths.runtimeDirectory, filepath.Join(paths.runtimeDirectory, "bin")} {
		if err := ensureWritableDirectory(directory); err != nil {
			return launchPaths{}, err
		}
	}
	paths.scannerBinary = filepath.Join(paths.runtimeDirectory, "bin", "scanner")
	paths.dashboardBinary = filepath.Join(paths.runtimeDirectory, "bin", "dashboard")
	buildEnvironment := productionBaseEnvironment()
	for _, build := range []struct{ output, target string }{{paths.scannerBinary, "./cmd/scanner"}, {paths.dashboardBinary, "./cmd/dashboard"}} {
		command := exec.CommandContext(ctx, goBinary, "build", "-trimpath", "-o", build.output, build.target)
		command.Dir, command.Env, command.Stdout, command.Stderr = root, buildEnvironment, stderr, stderr
		if err := command.Run(); err != nil {
			return launchPaths{}, fmt.Errorf("build %s: %w", build.target, err)
		}
	}
	return paths, nil
}

// productionBaseEnvironment deliberately reads only non-secret process
// settings needed by local builds, the dashboard, and the browser opener. In
// particular it never reads MASSIVE_API_KEY during overnight standby.
func productionBaseEnvironment() []string {
	names := []string{
		"HOME", "PATH", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "LC_CTYPE", "TZ",
		"GOROOT", "GOPATH", "GOCACHE", "GOMODCACHE", "GOENV", "GOFLAGS", "GOPROXY",
		"GONOPROXY", "GONOSUMDB", "GOPRIVATE", "GOSUMDB", "GOTOOLCHAIN", "CGO_ENABLED",
		"CC", "CXX", "SDKROOT", "DEVELOPER_DIR",
	}
	result := make([]string, 0, len(names))
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			result = append(result, name+"="+value)
		}
	}
	return result
}

func requireAvailablePort(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	return listener.Close()
}

func ensureWritableDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create runtime directory %s: %w", path, err)
	}
	test, err := os.CreateTemp(path, ".private-scanner-write-test-")
	if err != nil {
		return fmt.Errorf("directory is not writable: %s", path)
	}
	name := test.Name()
	if closeErr := test.Close(); closeErr != nil {
		_ = os.Remove(name)
		return fmt.Errorf("close directory write test: %w", closeErr)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove directory write test: %w", err)
	}
	return nil
}

func productionCredential(ctx context.Context, environment []string) (string, string, error) {
	if value, exists := environmentValue(environment, "MASSIVE_API_KEY"); exists && value != "" {
		return value, "exported MASSIVE_API_KEY", nil
	}
	if runtime.GOOS != "darwin" {
		return "", "", errors.New("macOS Keychain is unavailable")
	}
	credentialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(credentialCtx, "/usr/bin/security", "find-generic-password", "-a", keychainAccount, "-s", keychainService, "-w")
	command.Env = removeEnvironment(environment, "MASSIVE_API_KEY")
	output, err := command.Output()
	if err != nil {
		return "", "", errors.New("Keychain item unavailable")
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r")
	for index := range output {
		output[index] = 0
	}
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return "", "", errors.New("Keychain item is invalid")
	}
	return value, "macOS Keychain", nil
}

func startCommand(spec processSpec) (childProcess, error) {
	command := exec.Command(spec.path, spec.arguments...)
	// Children run in separate process groups so a terminal Ctrl-C reaches the
	// foreground launcher first. The supervisor can then forward that exact
	// signal in the required scanner-then-dashboard shutdown order.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Dir, command.Env, command.Stdout, command.Stderr = spec.workingDirectory, spec.environment, spec.stdout, spec.stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	process := &commandProcess{command: command, done: make(chan error, 1)}
	go func() {
		process.done <- command.Wait()
		close(process.done)
	}()
	return process, nil
}

func probeHTTP(ctx context.Context, url string) (probeResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return probeResult{}, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return probeResult{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return probeResult{status: response.StatusCode, body: body}, err
}

func openBrowser(ctx context.Context, url string, environment []string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, "/usr/bin/open", url)
	command.Env, command.Stdout, command.Stderr = removeEnvironment(environment, "MASSIVE_API_KEY"), stdout, stderr
	return command.Run()
}
