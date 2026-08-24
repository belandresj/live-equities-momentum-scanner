package privatelauncher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type fakeChild struct {
	once      sync.Once
	done      chan error
	signals   []os.Signal
	killed    bool
	mu        sync.Mutex
	stopOnSig bool
}

func newFakeChild() *fakeChild { return &fakeChild{done: make(chan error, 1), stopOnSig: true} }

func (child *fakeChild) Signal(value os.Signal) error {
	child.mu.Lock()
	child.signals = append(child.signals, value)
	stop := child.stopOnSig
	child.mu.Unlock()
	if stop {
		child.exit(nil)
	}
	return nil
}

func (child *fakeChild) Kill() error {
	child.mu.Lock()
	child.killed = true
	child.mu.Unlock()
	child.exit(errors.New("killed"))
	return nil
}

func (child *fakeChild) Done() <-chan error { return child.done }

func (child *fakeChild) exit(err error) {
	child.once.Do(func() {
		child.done <- err
		close(child.done)
	})
}

func (child *fakeChild) signalCount() int {
	child.mu.Lock()
	defer child.mu.Unlock()
	return len(child.signals)
}

type launcherFixture struct {
	deps             dependencies
	starts           []processSpec
	children         []*fakeChild
	preflights       int
	credentials      int
	environmentReads int
	opens            int
	stdout, stderr   bytes.Buffer
	cancel           context.CancelFunc
	mu               sync.Mutex
}

func newLauncherFixture(t *testing.T, at time.Time) *launcherFixture {
	t.Helper()
	fixture := &launcherFixture{}
	var clockMu sync.Mutex
	current := at
	fixture.deps = dependencies{
		now: func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return current },
		after: func(delay time.Duration) <-chan time.Time {
			clockMu.Lock()
			current = current.Add(delay)
			firedAt := current
			clockMu.Unlock()
			ready := make(chan time.Time, 1)
			ready <- firedAt
			return ready
		},
		baseEnvironment: func() []string {
			return []string{"PATH=/usr/bin", "HOME=/private/test"}
		},
		credentialEnvironment: func() []string {
			fixture.mu.Lock()
			fixture.environmentReads++
			fixture.mu.Unlock()
			return []string{"PATH=/usr/bin", "MASSIVE_API_KEY=exported-test-key", "HOME=/private/test"}
		},
		preflight: func(_ context.Context, root string, _, _ io.Writer) (launchPaths, error) {
			fixture.mu.Lock()
			fixture.preflights++
			fixture.mu.Unlock()
			if root != "/repo" {
				return launchPaths{}, fmt.Errorf("root=%s", root)
			}
			return launchPaths{reference: "/repo/var/reference", checkpoints: "/repo/var/checkpoints", runtimeDirectory: "/repo/var/run-private-scanner",
				scannerBinary: "/repo/var/run-private-scanner/bin/scanner", dashboardBinary: "/repo/var/run-private-scanner/bin/dashboard"}, nil
		},
		credential: func(_ context.Context, environment []string) (string, string, error) {
			fixture.mu.Lock()
			fixture.credentials++
			fixture.mu.Unlock()
			value, exists := environmentValue(environment, "MASSIVE_API_KEY")
			if !exists || value == "" {
				return "simulated-keychain-secret", "simulated Keychain", nil
			}
			return value, "exported MASSIVE_API_KEY", nil
		},
		start: func(spec processSpec) (childProcess, error) {
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			if len(fixture.starts) == 2 && fixture.cancel != nil {
				go fixture.cancel()
			}
			return child, nil
		},
		probe: func(_ context.Context, url string) (probeResult, error) {
			switch url {
			case scannerOrigin + "/livez":
				return probeResult{status: http.StatusOK, body: []byte(`{"process_live":true}`)}, nil
			case dashboardOrigin + "/":
				return probeResult{status: http.StatusOK, body: []byte("ok")}, nil
			case scannerOrigin + "/readyz":
				return probeResult{status: http.StatusServiceUnavailable, body: []byte(`{"backend_ready":false,"reason":"hydration_in_progress"}`)}, nil
			default:
				return probeResult{}, errors.New("unexpected probe")
			}
		},
		open: func(context.Context, string, []string, io.Writer, io.Writer) error {
			fixture.mu.Lock()
			fixture.opens++
			fixture.mu.Unlock()
			return nil
		},
		available:      func(string) error { return nil },
		standbyRecheck: 24 * time.Hour,
		scannerStartup: 250 * time.Millisecond, dashboardStartup: 250 * time.Millisecond, pollInterval: time.Millisecond,
		scannerShutdown: 20 * time.Millisecond, dashboardShutdown: 20 * time.Millisecond,
	}
	return fixture
}

func (fixture *launcherFixture) run(t *testing.T, arguments ...string) error {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	fixture.cancel = cancel
	defer cancel()
	return run(ctx, "/repo", arguments, &fixture.stdout, &fixture.stderr, fixture.deps)
}

func TestDailyDefaultsDeriveNewYorkDateAndIsolateCredential(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	fixture := newLauncherFixture(t, time.Date(2026, 8, 10, 3, 55, 0, 0, location))
	if err := fixture.run(t); err != nil {
		t.Fatal(err)
	}
	if len(fixture.starts) != 2 || fixture.starts[0].name != "scanner" || fixture.starts[1].name != "dashboard" {
		t.Fatalf("startup order=%v", processNames(fixture.starts))
	}
	wantScanner := []string{"--run-mode", "live", "--trading-date", "2026-08-10", "--hydration-workers", "8", "--reference-dir", "/repo/var/reference",
		"--checkpoint-dir", "/repo/var/checkpoints", "--checkpoint-mode", "off", "--api-address", scannerAddress, "--allow-origin", dashboardOrigin}
	wantDashboard := []string{"--address", dashboardAddress, "--api-origin", scannerOrigin, "--assets", "/repo/ui"}
	if !reflect.DeepEqual(fixture.starts[0].arguments, wantScanner) || !reflect.DeepEqual(fixture.starts[1].arguments, wantDashboard) {
		t.Fatalf("arguments scanner=%q dashboard=%q", fixture.starts[0].arguments, fixture.starts[1].arguments)
	}
	if countEnvironment(fixture.starts[0].environment, "MASSIVE_API_KEY") != 1 || countEnvironment(fixture.starts[1].environment, "MASSIVE_API_KEY") != 0 {
		t.Fatalf("credential environment scanner=%q dashboard=%q", fixture.starts[0].environment, fixture.starts[1].environment)
	}
	if !strings.Contains(fixture.stdout.String(), "Starting before 04:00") || !strings.Contains(fixture.stdout.String(), "not ready (hydration_in_progress)") || strings.Contains(fixture.stdout.String(), "ready (authoritative") {
		t.Fatalf("startup/readiness output=%q", fixture.stdout.String())
	}
	if fixture.children[0].signalCount() != 1 || fixture.children[1].signalCount() != 1 {
		t.Fatalf("shutdown signals scanner=%d dashboard=%d", fixture.children[0].signalCount(), fixture.children[1].signalCount())
	}
}

func TestTradingDateAndHydrationWorkerOverridesRejectInvalidArguments(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	if err := fixture.run(t, "--trading-date", "2026-08-10", "--hydration-workers", "4"); err != nil {
		t.Fatal(err)
	}
	if got := argumentValue(fixture.starts[0].arguments, "--trading-date"); got != "2026-08-10" {
		t.Fatalf("trading date=%q", got)
	}
	if got := argumentValue(fixture.starts[0].arguments, "--hydration-workers"); got != "4" {
		t.Fatalf("hydration workers=%q", got)
	}
	for _, test := range [][]string{{"--trading-date", "2026-8-10"}, {"--trading-date", "2026-02-30"}, {"--trading-date"},
		{"--trading-date", "2026-08-10", "--trading-date", "2026-08-11"}, {"--hydration-workers"}, {"--hydration-workers", "0"},
		{"--hydration-workers", "3"}, {"--hydration-workers", "5"}, {"--hydration-workers", "8", "--hydration-workers", "4"},
		{"--open", "--open"}, {"--unknown"}, {"positional"}} {
		candidate := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		if err := candidate.run(t, test...); err == nil {
			t.Fatalf("arguments %q accepted", test)
		}
		if candidate.preflights != 0 || candidate.credentials != 0 || len(candidate.starts) != 0 {
			t.Fatalf("arguments %q crossed preflight/credential/start boundary", test)
		}
	}
}

func TestLateStartWarningPrecedesCredentialAndStaysConcise(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
	fixture.deps.credential = func(context.Context, []string) (string, string, error) {
		warning := fixture.stderr.String()
		if !strings.Contains(warning, "Warning: starting after 04:00 EST. Will start historical data fetches to ready scanner") || strings.Contains(strings.ToLower(warning), "checkpoint") || strings.Contains(warning, "2x") || strings.Contains(warning, "fixed completion") {
			return "", "", fmt.Errorf("warning was absent or too detailed before credential acquisition: %q", warning)
		}
		return "late-start-secret", "simulated Keychain", nil
	}
	if err := fixture.run(t); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fixture.stdout.String()+fixture.stderr.String(), "late-start-secret") {
		t.Fatal("credential reached output")
	}
}

func TestHelpAndPreflightFailuresDoNotAcquireCredentialOrStartChildren(t *testing.T) {
	help := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	if err := help.run(t, "--help"); err != nil {
		t.Fatal(err)
	}
	if help.preflights != 0 || help.credentials != 0 || len(help.starts) != 0 || !strings.Contains(help.stdout.String(), "--hydration-workers 1|2|4|8") {
		t.Fatalf("help crossed boundary: preflight=%d credential=%d starts=%d output=%q", help.preflights, help.credentials, len(help.starts), help.stdout.String())
	}

	failure := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	failure.deps.preflight = func(context.Context, string, io.Writer, io.Writer) (launchPaths, error) {
		failure.preflights++
		return launchPaths{}, errors.New("127.0.0.1:8080 occupied")
	}
	if err := failure.run(t); err == nil || !strings.Contains(err.Error(), "occupied") {
		t.Fatalf("preflight error=%v", err)
	}
	if failure.credentials != 0 || len(failure.starts) != 0 {
		t.Fatalf("preflight failure credential=%d starts=%d", failure.credentials, len(failure.starts))
	}
}

func TestSimulatedKeychainSecretIsNeverPrintedOrPassedToDashboard(t *testing.T) {
	const secret = "simulated-keychain-secret-value"
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	fixture.deps.credentialEnvironment = func() []string { return []string{"PATH=/usr/bin", "HOME=/private/test"} }
	fixture.deps.credential = func(context.Context, []string) (string, string, error) { return secret, "simulated Keychain", nil }
	if err := fixture.run(t); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fixture.stdout.String()+fixture.stderr.String(), secret) {
		t.Fatal("simulated Keychain credential reached output")
	}
	if value, ok := environmentValue(fixture.starts[0].environment, "MASSIVE_API_KEY"); !ok || value != secret {
		t.Fatal("scanner did not receive simulated credential")
	}
	if _, ok := environmentValue(fixture.starts[1].environment, "MASSIVE_API_KEY"); ok {
		t.Fatal("dashboard received simulated credential")
	}
}

func TestProductionBaseEnvironmentDoesNotReadOrCopyMassiveCredential(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "standby-secret")
	for _, entry := range productionBaseEnvironment() {
		if strings.HasPrefix(entry, "MASSIVE_API_KEY=") || strings.Contains(entry, "standby-secret") {
			t.Fatalf("base environment contains credential: %q", entry)
		}
	}
}

func TestStartupAndFailureContainmentPaths(t *testing.T) {
	t.Run("dashboard waits for scanner liveness", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		liveCalls := 0
		fixture.deps.probe = func(_ context.Context, url string) (probeResult, error) {
			switch url {
			case scannerOrigin + "/livez":
				liveCalls++
				if liveCalls < 3 {
					return probeResult{status: http.StatusServiceUnavailable, body: []byte(`{"process_live":false}`)}, nil
				}
				return probeResult{status: http.StatusOK, body: []byte(`{"process_live":true}`)}, nil
			case dashboardOrigin + "/":
				return probeResult{status: http.StatusOK}, nil
			case scannerOrigin + "/readyz":
				return probeResult{status: http.StatusServiceUnavailable, body: []byte(`{"backend_ready":false,"reason":"pre_session"}`)}, nil
			default:
				return probeResult{}, errors.New("unexpected probe")
			}
		}
		originalStart := fixture.deps.start
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			if spec.name == "dashboard" && liveCalls < 3 {
				t.Fatal("dashboard started before scanner /livez")
			}
			return originalStart(spec)
		}
		if err := fixture.run(t); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("scanner early exit", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		fixture.cancel = nil
		fixture.deps.probe = func(context.Context, string) (probeResult, error) { return probeResult{}, errors.New("not live") }
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			child.exit(errors.New("scanner failed"))
			return child, nil
		}
		if err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err == nil || len(fixture.starts) != 1 {
			t.Fatalf("early exit err=%v starts=%d", err, len(fixture.starts))
		}
	})

	t.Run("scanner live timeout", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		fixture.cancel = nil
		fixture.deps.probe = func(context.Context, string) (probeResult, error) { return probeResult{}, errors.New("not live") }
		if err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err == nil || !strings.Contains(err.Error(), "5 minutes") {
			t.Fatalf("live timeout err=%v", err)
		}
		if len(fixture.children) != 1 || fixture.children[0].signalCount() != 1 {
			t.Fatal("scanner was not contained after live timeout")
		}
	})

	t.Run("scanner later failure", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			if spec.name == "dashboard" {
				fixture.children[0].exit(errors.New("scanner failure"))
			}
			return child, nil
		}
		err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps)
		if err == nil || !strings.Contains(err.Error(), "scanner exited unexpectedly") {
			t.Fatalf("later failure err=%v", err)
		}
		if fixture.children[1].signalCount() != 1 {
			t.Fatalf("dashboard was not stopped: signals=%d", fixture.children[1].signalCount())
		}
	})
}

func TestDashboardFailuresRemainLocalToScannerSupervision(t *testing.T) {
	t.Run("start failure retries three times without signalling scanner", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
		scanner := newFakeChild()
		var dashboardSpecs []processSpec
		var delays []time.Duration
		fixture.deps.after = func(delay time.Duration) <-chan time.Time {
			delays = append(delays, delay)
			fired := make(chan time.Time, 1)
			fired <- fixture.deps.now().Add(delay)
			return fired
		}
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			if spec.name != "dashboard" {
				t.Fatalf("unexpected child %q", spec.name)
			}
			dashboardSpecs = append(dashboardSpecs, spec)
			return nil, errors.New("listener bind failed")
		}
		dashboard, err := startDashboardWithRecovery(context.Background(), scanner, "/dashboard", []string{"--address", dashboardAddress}, []string{"PATH=/usr/bin"}, "/repo", &fixture.stdout, &fixture.stderr, fixture.deps)
		if err != nil || dashboard != nil {
			t.Fatalf("dashboard=%v err=%v", dashboard, err)
		}
		if scanner.signalCount() != 0 {
			t.Fatalf("dashboard failure signalled scanner %d times", scanner.signalCount())
		}
		if !reflect.DeepEqual(delays, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}) || len(dashboardSpecs) != 4 {
			t.Fatalf("delays=%v dashboard_starts=%d", delays, len(dashboardSpecs))
		}
		for _, spec := range dashboardSpecs {
			if !reflect.DeepEqual(spec.arguments, []string{"--address", dashboardAddress}) || environmentValueMustNotExist(spec.environment, "MASSIVE_API_KEY") {
				t.Fatalf("dashboard spec=%+v", spec)
			}
		}
		if !strings.Contains(fixture.stderr.String(), "recovery exhausted") {
			t.Fatalf("missing headless warning: %q", fixture.stderr.String())
		}
	})

	t.Run("listener timeout is reaped before each replacement", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
		fixture.deps.dashboardStartup = 0
		fixture.deps.pollInterval = time.Millisecond
		fixture.deps.probe = func(_ context.Context, url string) (probeResult, error) {
			if url != dashboardOrigin+"/" {
				t.Fatalf("unexpected probe %q", url)
			}
			return probeResult{}, errors.New("not healthy")
		}
		var dashboards []*fakeChild
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			dashboards = append(dashboards, child)
			return child, nil
		}
		dashboard, err := startDashboardWithRecovery(context.Background(), newFakeChild(), "/dashboard", nil, nil, "/repo", &fixture.stdout, &fixture.stderr, fixture.deps)
		if err != nil || dashboard != nil || len(dashboards) != 4 {
			t.Fatalf("dashboard=%v err=%v starts=%d", dashboard, err, len(dashboards))
		}
		for index, child := range dashboards {
			if child.signalCount() != 1 {
				t.Fatalf("dashboard %d was not reaped before recovery: signals=%d", index, child.signalCount())
			}
		}
	})

	for _, exit := range []error{errors.New("dashboard failure"), nil} {
		name := "nonzero exit"
		if exit == nil {
			name = "unexpected zero exit"
		}
		t.Run(name, func(t *testing.T) {
			fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
			scanner, failedDashboard, replacement := newFakeChild(), newFakeChild(), newFakeChild()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fixture.deps.start = func(spec processSpec) (childProcess, error) {
				if spec.name != "dashboard" {
					t.Fatalf("unexpected child %q", spec.name)
				}
				if scanner.signalCount() != 0 {
					t.Fatalf("scanner was signalled before dashboard replacement")
				}
				cancel()
				return replacement, nil
			}
			failedDashboard.exit(exit)
			err := supervise(ctx, scanner, failedDashboard, nyTime(t, 2026, 8, 10, 20, 0), "/dashboard", nil, nil, "/repo", &fixture.stdout, &fixture.stderr, fixture.deps)
			if err != nil {
				t.Fatal(err)
			}
			if scanner.signalCount() != 1 || replacement.signalCount() != 1 {
				t.Fatalf("controlled shutdown did not reap scanner/replacement: scanner=%d replacement=%d", scanner.signalCount(), replacement.signalCount())
			}
		})
	}

	t.Run("standby exhaustion preserves the scheduled single scanner start", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		fixture.cancel = nil
		var scannerStarts int
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			switch spec.name {
			case "dashboard":
				if len(fixture.starts) == 0 {
					child := newFakeChild()
					fixture.starts = append(fixture.starts, spec)
					fixture.children = append(fixture.children, child)
					child.exit(errors.New("standby dashboard failure"))
					return child, nil
				}
				return nil, errors.New("dashboard unavailable")
			case "scanner":
				if fixture.credentials != 1 || fixture.environmentReads != 1 {
					t.Fatalf("standby acquired credential before preconnect: credentials=%d environment_reads=%d", fixture.credentials, fixture.environmentReads)
				}
				scannerStarts++
				child := newFakeChild()
				fixture.starts = append(fixture.starts, spec)
				fixture.children = append(fixture.children, child)
				child.exit(nil)
				return child, nil
			default:
				t.Fatalf("unexpected child %q", spec.name)
				return nil, nil
			}
		}
		err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps)
		if err == nil || !strings.Contains(err.Error(), "scanner exited unexpectedly") {
			t.Fatalf("run error=%v", err)
		}
		if scannerStarts != 1 || fixture.credentials != 1 || strings.Contains(fixture.stdout.String(), "Scanner dashboard:") {
			t.Fatalf("scanner_starts=%d credentials=%d stdout=%q", scannerStarts, fixture.credentials, fixture.stdout.String())
		}
	})

	t.Run("near-boundary standby defers dashboard health work until scanner starts", func(t *testing.T) {
		initial := nyTime(t, 2026, 8, 10, 3, 50)
		fixture := newLauncherFixture(t, initial)
		fixture.cancel = nil
		fixture.deps.dashboardStartup = 2 * time.Minute
		current := initial
		fixture.deps.now = func() time.Time { return current }
		fixture.deps.after = func(delay time.Duration) <-chan time.Time {
			current = current.Add(delay)
			fired := make(chan time.Time, 1)
			fired <- current
			return fired
		}
		originalPreflight := fixture.deps.preflight
		fixture.deps.preflight = func(ctx context.Context, root string, stdout, stderr io.Writer) (launchPaths, error) {
			paths, err := originalPreflight(ctx, root, stdout, stderr)
			current = nyTime(t, 2026, 8, 10, 3, 54)
			return paths, err
		}
		var started []string
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			started = append(started, spec.name)
			if spec.name == "dashboard" {
				return nil, errors.New("dashboard unavailable")
			}
			child := newFakeChild()
			child.exit(nil)
			return child, nil
		}
		err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps)
		if err == nil || !strings.Contains(err.Error(), "scanner exited unexpectedly") {
			t.Fatalf("run error=%v", err)
		}
		if len(started) == 0 || started[0] != "scanner" {
			t.Fatalf("startup order=%v", started)
		}
	})

	t.Run("operator shutdown cancels dashboard backoff and stops scanner", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
		fixture.cancel = nil
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		retryWaiting := make(chan struct{})
		fixture.deps.after = func(delay time.Duration) <-chan time.Time {
			if delay == time.Second {
				close(retryWaiting)
				return make(chan time.Time)
			}
			fired := make(chan time.Time, 1)
			fired <- fixture.deps.now().Add(delay)
			return fired
		}
		var scanner *fakeChild
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			if spec.name == "scanner" {
				scanner = child
			} else {
				child.exit(errors.New("dashboard failed"))
			}
			return child, nil
		}
		done := make(chan error, 1)
		go func() { done <- run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps) }()
		<-retryWaiting
		if scanner == nil || scanner.signalCount() != 0 {
			t.Fatalf("scanner was not live and unsignalled during backoff")
		}
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("shutdown error=%v", err)
		}
		if scanner.signalCount() != 1 {
			t.Fatalf("scanner was not stopped on controlled shutdown: signals=%d", scanner.signalCount())
		}
	})
}

func environmentValueMustNotExist(environment []string, name string) bool {
	_, exists := environmentValue(environment, name)
	return exists
}

func TestSafeOutputLatchesBrokenTerminalWithoutReportingFailure(t *testing.T) {
	broken := &failingWriter{}
	output := newSafeOutput(broken)
	if _, err := output.Write([]byte("first")); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("second")); err != nil {
		t.Fatal(err)
	}
	if broken.writes != 1 || !output.disabled {
		t.Fatalf("writes=%d disabled=%t", broken.writes, output.disabled)
	}
}

func TestBrokenLauncherOutputDoesNotEndSupervision(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
	scanner, dashboard := newFakeChild(), newFakeChild()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporting := make(chan struct{})
	fixture.deps.probe = func(_ context.Context, url string) (probeResult, error) {
		if url == scannerOrigin+"/readyz" {
			select {
			case <-reporting:
			default:
				close(reporting)
			}
		}
		return probeResult{}, errors.New("unavailable")
	}
	output := newSafeOutput(&failingWriter{})
	done := make(chan error, 1)
	go func() {
		done <- supervise(ctx, scanner, dashboard, nyTime(t, 2026, 8, 10, 20, 0), "/dashboard", nil, nil, "/repo", output, output, fixture.deps)
	}()
	<-reporting
	if scanner.signalCount() != 0 || dashboard.signalCount() != 0 {
		t.Fatalf("broken output signalled child before operator shutdown: scanner=%d dashboard=%d", scanner.signalCount(), dashboard.signalCount())
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

type failingWriter struct{ writes int }

func (writer *failingWriter) Write([]byte) (int, error) {
	writer.writes++
	return 0, errors.New("broken pipe")
}

func TestOpenFailureIsNonfatalAfterBothServicesAreHealthy(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	fixture.deps.open = func(_ context.Context, _ string, environment []string, _, _ io.Writer) error {
		if _, ok := environmentValue(environment, "MASSIVE_API_KEY"); ok {
			t.Fatal("browser opener received credential")
		}
		return errors.New("no browser")
	}
	if err := fixture.run(t, "--open"); err != nil {
		t.Fatal(err)
	}
	if len(fixture.starts) != 2 || !strings.Contains(fixture.stderr.String(), "services remain running") {
		t.Fatalf("open behavior starts=%d stderr=%q", len(fixture.starts), fixture.stderr.String())
	}
}

func TestAfterSessionEntersNextTradingSessionStandby(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 20, 0))
	if err := fixture.run(t); err != nil {
		t.Fatal(err)
	}
	if got := processNames(fixture.starts); !reflect.DeepEqual(got, []string{"dashboard", "scanner"}) {
		t.Fatalf("startup order=%v", got)
	}
	if got := argumentValue(fixture.starts[1].arguments, "--trading-date"); got != "2026-08-11" {
		t.Fatalf("scanner trading date=%q", got)
	}
	if !strings.Contains(fixture.stdout.String(), "Overnight standby for trading session 2026-08-11") {
		t.Fatalf("standby output=%q", fixture.stdout.String())
	}
}

func TestOvernightStandbyDefersCredentialAndUsesExchangeCalendar(t *testing.T) {
	for _, test := range []struct {
		name string
		at   time.Time
		wake time.Time
		want string
	}{
		{"weekend", nyTime(t, 2026, 8, 22, 10, 0), nyTime(t, 2026, 8, 24, 3, 55), "2026-08-24"},
		{"exchange holiday", nyTime(t, 2026, 9, 7, 10, 0), nyTime(t, 2026, 9, 8, 3, 55), "2026-09-08"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLauncherFixture(t, test.at)
			var clockMu sync.Mutex
			current := test.at
			fixture.deps.now = func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return current }
			release := make(chan time.Time, 1)
			fixture.deps.after = func(time.Duration) <-chan time.Time { return release }
			ctx, cancel := context.WithCancel(context.Background())
			fixture.cancel = cancel
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps) }()

			deadline := time.Now().Add(time.Second)
			for {
				fixture.mu.Lock()
				starts, credentials, environmentReads := len(fixture.starts), fixture.credentials, fixture.environmentReads
				fixture.mu.Unlock()
				if starts == 1 {
					if credentials != 0 || environmentReads != 0 || fixture.starts[0].name != "dashboard" {
						t.Fatalf("standby starts=%v credentials=%d environment_reads=%d", processNames(fixture.starts), credentials, environmentReads)
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("dashboard did not enter standby")
				}
				time.Sleep(time.Millisecond)
			}
			clockMu.Lock()
			current = test.wake
			clockMu.Unlock()
			release <- test.wake
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if got := processNames(fixture.starts); !reflect.DeepEqual(got, []string{"dashboard", "scanner"}) {
				t.Fatalf("startup order=%v", got)
			}
			if got := argumentValue(fixture.starts[1].arguments, "--trading-date"); got != test.want {
				t.Fatalf("trading date=%q, want %s", got, test.want)
			}
		})
	}
}

func TestExplicitExpiredOrUnsupportedDateDoesNotRollForward(t *testing.T) {
	for _, date := range []string{"2026-08-10", "2026-08-08"} {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 20, 0))
		err := fixture.run(t, "--trading-date", date)
		if err == nil {
			t.Fatalf("explicit date %s unexpectedly succeeded", date)
		}
		if fixture.preflights != 0 || fixture.credentials != 0 || len(fixture.starts) != 0 {
			t.Fatalf("explicit date %s crossed startup boundary", date)
		}
	}
}

func TestDelayedWakeRevalidatesSessionBeforeCredential(t *testing.T) {
	t.Run("implicit missed session rolls forward and waits again", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		var clockMu sync.Mutex
		current := nyTime(t, 2026, 8, 10, 21, 0)
		fixture.deps.now = func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return current }
		release := make(chan time.Time, 2)
		waiting := make(chan struct{}, 2)
		fixture.deps.after = func(time.Duration) <-chan time.Time { waiting <- struct{}{}; return release }
		ctx, cancel := context.WithCancel(context.Background())
		fixture.cancel = cancel
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps) }()
		waitForStarts(t, fixture, 1)
		<-waiting

		clockMu.Lock()
		current = nyTime(t, 2026, 8, 12, 21, 0)
		clockMu.Unlock()
		release <- current
		<-waiting
		fixture.mu.Lock()
		starts, credentials, environmentReads := len(fixture.starts), fixture.credentials, fixture.environmentReads
		fixture.mu.Unlock()
		if starts != 1 || credentials != 0 || environmentReads != 0 {
			t.Fatalf("starts=%d credentials=%d environment_reads=%d", starts, credentials, environmentReads)
		}

		clockMu.Lock()
		current = nyTime(t, 2026, 8, 13, 3, 55)
		clockMu.Unlock()
		release <- current
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(fixture.stdout.String(), "Standby continues for trading session 2026-08-13") {
			t.Fatalf("roll-forward output=%q", fixture.stdout.String())
		}
		if got := argumentValue(fixture.starts[1].arguments, "--trading-date"); got != "2026-08-13" {
			t.Fatalf("scanner trading date=%q", got)
		}
	})

	t.Run("explicit missed session fails exactly", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		var clockMu sync.Mutex
		current := nyTime(t, 2026, 8, 10, 21, 0)
		fixture.deps.now = func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return current }
		release := make(chan time.Time, 1)
		fixture.deps.after = func(time.Duration) <-chan time.Time { return release }
		fixture.cancel = nil
		done := make(chan error, 1)
		go func() {
			done <- run(context.Background(), "/repo", []string{"--trading-date", "2026-08-11"}, &fixture.stdout, &fixture.stderr, fixture.deps)
		}()
		waitForStarts(t, fixture, 1)
		clockMu.Lock()
		current = nyTime(t, 2026, 8, 11, 20, 1)
		clockMu.Unlock()
		release <- current
		err := <-done
		if err == nil || !strings.Contains(err.Error(), "explicit 2026-08-11 scanner session ended") {
			t.Fatalf("explicit delayed-wake error=%v", err)
		}
		if fixture.credentials != 0 || fixture.environmentReads != 0 || len(fixture.starts) != 1 || fixture.children[0].signalCount() != 1 {
			t.Fatalf("starts=%v credentials=%d environment_reads=%d dashboard_signals=%d", processNames(fixture.starts), fixture.credentials, fixture.environmentReads, fixture.children[0].signalCount())
		}
	})
}

func TestStandbyWaitCapsMonotonicTimerAndReturnsToWallClockLoop(t *testing.T) {
	now := nyTime(t, 2026, 8, 10, 21, 0)
	observed := time.Duration(0)
	deps := dependencies{
		now:            func() time.Time { return now },
		standbyRecheck: 30 * time.Second,
		after: func(delay time.Duration) <-chan time.Time {
			observed = delay
			ready := make(chan time.Time, 1)
			ready <- now.Add(delay)
			return ready
		},
	}
	dashboardDone, err := waitForPreconnect(context.Background(), newFakeChild(), now.Add(8*time.Hour), deps)
	if err != nil || dashboardDone || observed != 30*time.Second {
		t.Fatalf("dashboard_done=%t delay=%s err=%v", dashboardDone, observed, err)
	}
}

func TestStandbyShutdownAndWakePortConflictContainDashboard(t *testing.T) {
	t.Run("dashboard exit", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		fixture.deps.after = func(time.Duration) <-chan time.Time { return make(chan time.Time) }
		fixture.cancel = nil
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		failed := make(chan struct{})
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			if spec.name == "dashboard" {
				child.exit(errors.New("dashboard failed"))
				close(failed)
			}
			return child, nil
		}
		done := make(chan error, 1)
		go func() { done <- run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps) }()
		<-failed
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("dashboard-backoff shutdown error=%v", err)
		}
		if fixture.credentials != 0 || len(fixture.starts) != 1 {
			t.Fatalf("starts=%v credentials=%d", processNames(fixture.starts), fixture.credentials)
		}
	})

	t.Run("operator shutdown", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		fixture.deps.after = func(time.Duration) <-chan time.Time { return make(chan time.Time) }
		ctx, cancel := context.WithCancel(context.Background())
		fixture.cancel = nil
		done := make(chan error, 1)
		go func() { done <- run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps) }()
		waitForStarts(t, fixture, 1)
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("shutdown error=%v", err)
		}
		if fixture.credentials != 0 || fixture.children[0].signalCount() != 1 {
			t.Fatalf("credentials=%d dashboard signals=%d", fixture.credentials, fixture.children[0].signalCount())
		}
	})

	t.Run("scanner port occupied at wake", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 21, 0))
		fixture.cancel = nil
		fixture.deps.available = func(string) error { return errors.New("occupied") }
		err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps)
		if err == nil || !strings.Contains(err.Error(), "became occupied") {
			t.Fatalf("wake error=%v", err)
		}
		if fixture.credentials != 0 || len(fixture.starts) != 1 || fixture.children[0].signalCount() != 1 {
			t.Fatalf("starts=%v credentials=%d signals=%d", processNames(fixture.starts), fixture.credentials, fixture.children[0].signalCount())
		}
	})
}

func waitForStarts(t *testing.T, fixture *launcherFixture, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		fixture.mu.Lock()
		got := len(fixture.starts)
		fixture.mu.Unlock()
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("starts=%d, want at least %d", got, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestOccupiedPortCheckDoesNotKillListener(t *testing.T) {
	listener, err := net.Listen("tcp", scannerAddress)
	if err != nil {
		t.Skipf("fixed scanner port is already in use by the operator: %v", err)
	}
	defer listener.Close()
	if err := requireAvailablePort(scannerAddress); err == nil {
		t.Fatal("occupied port unexpectedly passed preflight")
	}
	if address := listener.Addr().String(); address != scannerAddress {
		t.Fatalf("occupying listener changed or stopped: %s", address)
	}
}

func TestExportedCredentialPrecedesKeychainPath(t *testing.T) {
	value, source, err := productionCredential(context.Background(), []string{"PATH=/usr/bin", "MASSIVE_API_KEY=injected-export"})
	if err != nil || value != "injected-export" || source != "exported MASSIVE_API_KEY" {
		t.Fatalf("credential=%q source=%q err=%v", value, source, err)
	}
}

func TestGracefulShutdownTimeoutForcesReapingAndReturnsFailure(t *testing.T) {
	child := newFakeChild()
	child.stopOnSig = false
	err := signalAndWait(child, syscall.SIGTERM, time.Millisecond, "scanner")
	if err == nil || !strings.Contains(err.Error(), "forcibly terminated") {
		t.Fatalf("shutdown timeout err=%v", err)
	}
	child.mu.Lock()
	killed := child.killed
	child.mu.Unlock()
	if !killed {
		t.Fatal("timed-out child was not terminated and reaped")
	}
}

func TestForegroundSignalsAreForwardedExactlyToBothChildren(t *testing.T) {
	for _, forwarded := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(forwarded.String(), func(t *testing.T) {
			fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(context.Canceled)
			fixture.deps.start = func(spec processSpec) (childProcess, error) {
				child := newFakeChild()
				fixture.starts = append(fixture.starts, spec)
				fixture.children = append(fixture.children, child)
				if spec.name == "dashboard" {
					go cancel(signalCancellation{signal: forwarded})
				}
				return child, nil
			}
			if err := run(ctx, "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err != nil {
				t.Fatal(err)
			}
			if len(fixture.children) != 2 {
				t.Fatalf("children=%d", len(fixture.children))
			}
			for index, child := range fixture.children {
				child.mu.Lock()
				signals := append([]os.Signal(nil), child.signals...)
				child.mu.Unlock()
				if len(signals) != 1 || signals[0] != forwarded {
					t.Fatalf("child %d signals=%v want=%v", index, signals, forwarded)
				}
			}
		})
	}
}

func TestCommandProcessForwardsTerminationAndReaps(t *testing.T) {
	if os.Getenv("PRIVATE_LAUNCHER_SIGNAL_HELPER") == "1" {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
		defer signal.Stop(signals)
		<-signals
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child, err := startCommand(processSpec{name: "helper", path: executable, arguments: []string{"-test.run=^TestCommandProcessForwardsTerminationAndReaps$"},
		environment: append(os.Environ(), "PRIVATE_LAUNCHER_SIGNAL_HELPER=1"), stdout: io.Discard, stderr: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := signalAndWait(child, syscall.SIGTERM, 2*time.Second, "helper"); err != nil {
		t.Fatal(err)
	}
	process := child.(*commandProcess)
	if err := process.command.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("helper was not reaped: %v", err)
	}
}

func TestPublicWrapperParsesHelpAndInvalidArgumentsBeforeBuild(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "scripts", "run-private-scanner")
	fakeDirectory := t.TempDir()
	marker := filepath.Join(fakeDirectory, "go-invoked")
	fakeGo := filepath.Join(fakeDirectory, "go")
	if err := os.WriteFile(fakeGo, []byte("#!/bin/sh\n/usr/bin/touch \"$FAKE_GO_MARKER\"\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	environment := append(removeEnvironment(removeEnvironment(os.Environ(), "PATH"), "MASSIVE_API_KEY"), "PATH="+fakeDirectory+":"+os.Getenv("PATH"), "FAKE_GO_MARKER="+marker)
	runPublic := func(arguments ...string) (string, error) {
		command := exec.Command(script, arguments...)
		command.Dir = t.TempDir()
		command.Env = environment
		output, runErr := command.CombinedOutput()
		return string(output), runErr
	}
	output, err := runPublic("--help")
	if err != nil || !strings.Contains(output, "--hydration-workers 1|2|4|8") {
		t.Fatalf("public help output=%q err=%v", output, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("help invoked Go build: %v", err)
	}
	for _, arguments := range [][]string{{"--trading-date", "2026-02-30"}, {"--trading-date"}, {"--trading-date", "2026-08-10", "--trading-date", "2026-08-11"},
		{"--hydration-workers"}, {"--hydration-workers", "0"}, {"--hydration-workers", "3"}, {"--hydration-workers", "5"},
		{"--hydration-workers", "1", "--hydration-workers", "8"}, {"--open", "--open"}, {"--unknown"}, {"positional"}, {"--help", "--open"}, {"--help", "--hydration-workers", "1"}} {
		output, err := runPublic(arguments...)
		if err == nil || output == "" {
			t.Fatalf("public arguments %q output=%q err=%v", arguments, output, err)
		}
		if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("public arguments %q invoked Go build: %v", arguments, err)
		}
	}
}

func TestPLBRA3PublicWrapperDefaultsEightHydrationWorkers(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fakeDirectory := t.TempDir()
	launcherArguments := filepath.Join(fakeDirectory, "launcher-arguments")
	fakeGo := filepath.Join(fakeDirectory, "go")
	fakeSource := "#!/bin/sh\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = -o ]; then shift; output=$1; break; fi\n" +
		"  shift\n" +
		"done\n" +
		"[ -n \"$output\" ] || exit 91\n" +
		"/usr/bin/printf '%s\\n' '#!/bin/sh' '/usr/bin/printf \"%s\\n\" \"$@\" > \"$FAKE_LAUNCHER_ARGUMENTS\"' > \"$output\"\n" +
		"/bin/chmod 700 \"$output\"\n"
	if err := os.WriteFile(fakeGo, []byte(fakeSource), 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "scripts", "run-private-scanner"))
	command.Dir = t.TempDir()
	command.Env = append(removeEnvironment(removeEnvironment(os.Environ(), "PATH"), "MASSIVE_API_KEY"),
		"PATH="+fakeDirectory+":"+os.Getenv("PATH"), "FAKE_LAUNCHER_ARGUMENTS="+launcherArguments)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("public wrapper output=%q err=%v", output, err)
	}
	rawArguments, err := os.ReadFile(launcherArguments)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Fields(string(rawArguments)), []string{"--hydration-workers", "8"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("default launcher arguments=%q want=%q", got, want)
	}
	for _, workers := range []string{"1", "2", "4", "8"} {
		command := exec.Command(filepath.Join(root, "scripts", "run-private-scanner"), "--hydration-workers", workers)
		command.Dir = t.TempDir()
		command.Env = append(removeEnvironment(removeEnvironment(os.Environ(), "PATH"), "MASSIVE_API_KEY"),
			"PATH="+fakeDirectory+":"+os.Getenv("PATH"), "FAKE_LAUNCHER_ARGUMENTS="+launcherArguments)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("workers=%s public wrapper output=%q err=%v", workers, output, err)
		}
		rawArguments, err := os.ReadFile(launcherArguments)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := strings.Fields(string(rawArguments)), []string{"--hydration-workers", workers}; !reflect.DeepEqual(got, want) {
			t.Fatalf("workers=%s launcher arguments=%q want=%q", workers, got, want)
		}
	}
}

func TestPublicWrapperForwardsAndReapsSignalDuringBootstrapBuild(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fakeDirectory := t.TempDir()
	started := filepath.Join(fakeDirectory, "started")
	terminated := filepath.Join(fakeDirectory, "terminated")
	pidFile := filepath.Join(fakeDirectory, "pid")
	fakeGo := filepath.Join(fakeDirectory, "go")
	fakeSource := "#!/bin/sh\n" +
		"trap '/usr/bin/touch \"$FAKE_GO_TERMINATED\"; exit 0' INT TERM\n" +
		"echo $$ > \"$FAKE_GO_PID\"\n" +
		"/usr/bin/touch \"$FAKE_GO_STARTED\"\n" +
		"while :; do /bin/sleep 1; done\n"
	if err := os.WriteFile(fakeGo, []byte(fakeSource), 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "scripts", "run-private-scanner"), "--trading-date", "2026-08-10")
	command.Dir = t.TempDir()
	command.Stdout, command.Stderr = io.Discard, io.Discard
	command.Env = append(removeEnvironment(removeEnvironment(os.Environ(), "PATH"), "MASSIVE_API_KEY"),
		"PATH="+fakeDirectory+":"+os.Getenv("PATH"), "FAKE_GO_STARTED="+started, "FAKE_GO_TERMINATED="+terminated, "FAKE_GO_PID="+pidFile)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stopWrapper := func() bool {
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
			return true
		case <-time.After(2 * time.Second):
			_ = command.Process.Kill()
			select {
			case <-done:
				return true
			case <-time.After(time.Second):
				return false
			}
		}
	}
	stopFakeBuild := func() {
		raw, readErr := os.ReadFile(pidFile)
		if readErr != nil {
			return
		}
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(raw)))
		if parseErr != nil || pid <= 0 || syscall.Kill(pid, syscall.Signal(0)) != nil {
			return
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
		deadline := time.Now().Add(time.Second)
		for syscall.Kill(pid, syscall.Signal(0)) == nil && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if syscall.Kill(pid, syscall.Signal(0)) == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			joined := stopWrapper()
			stopFakeBuild()
			t.Fatalf("fake bootstrap build did not start; wrapper_joined=%t", joined)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !stopWrapper() {
		stopFakeBuild()
		t.Fatal("wrapper did not reap bootstrap build")
	}
	if _, err := os.Stat(terminated); err != nil {
		t.Fatalf("bootstrap build did not receive forwarded SIGTERM: %v", err)
	}
}

func processNames(specs []processSpec) []string {
	result := make([]string, len(specs))
	for index := range specs {
		result[index] = specs[index].name
	}
	return result
}

func countEnvironment(environment []string, name string) int {
	prefix, count := name+"=", 0
	for _, value := range environment {
		if strings.HasPrefix(value, prefix) {
			count++
		}
	}
	return count
}

func argumentValue(arguments []string, name string) string {
	for index := range arguments {
		if arguments[index] == name && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

func nyTime(t *testing.T, year int, month time.Month, day, hour, minute int) time.Time {
	t.Helper()
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}
