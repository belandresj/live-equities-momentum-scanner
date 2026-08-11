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
	deps           dependencies
	starts         []processSpec
	children       []*fakeChild
	preflights     int
	credentials    int
	opens          int
	stdout, stderr bytes.Buffer
	cancel         context.CancelFunc
	mu             sync.Mutex
}

func newLauncherFixture(t *testing.T, at time.Time) *launcherFixture {
	t.Helper()
	fixture := &launcherFixture{}
	fixture.deps = dependencies{
		now: func() time.Time { return at },
		environment: func() []string {
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
		scannerStartup: 20 * time.Millisecond, dashboardStartup: 20 * time.Millisecond, pollInterval: time.Millisecond,
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
	wantScanner := []string{"--run-mode", "live", "--trading-date", "2026-08-10", "--hydration-workers", "2", "--reference-dir", "/repo/var/reference",
		"--checkpoint-dir", "/repo/var/checkpoints", "--api-address", scannerAddress, "--allow-origin", dashboardOrigin}
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

func TestTradingDateOverrideChangesOnlyDateAndRejectsInvalidArguments(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
	if err := fixture.run(t, "--trading-date", "2026-08-11"); err != nil {
		t.Fatal(err)
	}
	if got := argumentValue(fixture.starts[0].arguments, "--trading-date"); got != "2026-08-11" {
		t.Fatalf("trading date=%q", got)
	}
	for _, test := range [][]string{{"--trading-date", "2026-8-10"}, {"--trading-date", "2026-02-30"}, {"--trading-date"},
		{"--trading-date", "2026-08-10", "--trading-date", "2026-08-11"}, {"--open", "--open"}, {"--unknown"}, {"positional"}} {
		candidate := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		if err := candidate.run(t, test...); err == nil {
			t.Fatalf("arguments %q accepted", test)
		}
		if candidate.preflights != 0 || candidate.credentials != 0 || len(candidate.starts) != 0 {
			t.Fatalf("arguments %q crossed preflight/credential/start boundary", test)
		}
	}
}

func TestLateStartWarningPrecedesCredentialAndStatesCapacityBoundary(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
	fixture.deps.credential = func(context.Context, []string) (string, string, error) {
		warning := fixture.stderr.String()
		if !strings.Contains(warning, "starting after 04:00") || !strings.Contains(warning, "Full 1x") || !strings.Contains(warning, "sustained full 2x is unsupported") || !strings.Contains(warning, "no fixed completion time") {
			return "", "", fmt.Errorf("warning was absent or inaccurate before credential acquisition: %q", warning)
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
	if help.preflights != 0 || help.credentials != 0 || len(help.starts) != 0 || !strings.Contains(help.stdout.String(), "./scripts/run-private-scanner --open") {
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
	fixture.deps.environment = func() []string { return []string{"PATH=/usr/bin", "HOME=/private/test"} }
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

	t.Run("dashboard start failure", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			if spec.name == "dashboard" {
				return nil, errors.New("dashboard start failed")
			}
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			return child, nil
		}
		if err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err == nil || fixture.children[0].signalCount() != 1 {
			t.Fatalf("dashboard start containment err=%v signals=%d", err, fixture.children[0].signalCount())
		}
	})

	t.Run("dashboard early exit", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		fixture.deps.probe = func(_ context.Context, url string) (probeResult, error) {
			if url == scannerOrigin+"/livez" {
				return probeResult{status: http.StatusOK, body: []byte(`{"process_live":true}`)}, nil
			}
			return probeResult{}, errors.New("dashboard unavailable")
		}
		fixture.deps.start = func(spec processSpec) (childProcess, error) {
			child := newFakeChild()
			fixture.starts = append(fixture.starts, spec)
			fixture.children = append(fixture.children, child)
			if spec.name == "dashboard" {
				child.exit(errors.New("dashboard failed"))
			}
			return child, nil
		}
		if err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err == nil || fixture.children[0].signalCount() != 1 {
			t.Fatalf("dashboard early-exit containment err=%v scanner_signals=%d", err, fixture.children[0].signalCount())
		}
	})

	t.Run("dashboard listener timeout", func(t *testing.T) {
		fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 3, 55))
		fixture.deps.probe = func(_ context.Context, url string) (probeResult, error) {
			if url == scannerOrigin+"/livez" {
				return probeResult{status: http.StatusOK, body: []byte(`{"process_live":true}`)}, nil
			}
			return probeResult{}, errors.New("dashboard unavailable")
		}
		if err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps); err == nil || !strings.Contains(err.Error(), "30 seconds") {
			t.Fatalf("dashboard timeout err=%v", err)
		}
		if fixture.children[0].signalCount() != 1 || fixture.children[1].signalCount() != 1 {
			t.Fatal("dashboard timeout did not reap both children")
		}
	})

	for _, failedName := range []string{"scanner", "dashboard"} {
		t.Run(failedName+" later failure", func(t *testing.T) {
			fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 12, 0))
			fixture.deps.start = func(spec processSpec) (childProcess, error) {
				child := newFakeChild()
				fixture.starts = append(fixture.starts, spec)
				fixture.children = append(fixture.children, child)
				if spec.name == "dashboard" {
					go func() {
						time.Sleep(2 * time.Millisecond)
						index := 0
						if failedName == "dashboard" {
							index = 1
						}
						fixture.children[index].exit(errors.New(failedName + " failure"))
					}()
				}
				return child, nil
			}
			err := run(context.Background(), "/repo", nil, &fixture.stdout, &fixture.stderr, fixture.deps)
			if err == nil || !strings.Contains(err.Error(), failedName+" exited unexpectedly") {
				t.Fatalf("later failure err=%v", err)
			}
			other := 1
			if failedName == "dashboard" {
				other = 0
			}
			if fixture.children[other].signalCount() != 1 {
				t.Fatalf("other child was not stopped: signals=%d", fixture.children[other].signalCount())
			}
		})
	}
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

func TestAfterSessionFailsBeforePreflightOrCredential(t *testing.T) {
	fixture := newLauncherFixture(t, nyTime(t, 2026, 8, 10, 20, 0))
	if err := fixture.run(t); err == nil || !strings.Contains(err.Error(), "session ended") {
		t.Fatalf("after-session error=%v", err)
	}
	if fixture.preflights != 0 || fixture.credentials != 0 || len(fixture.starts) != 0 {
		t.Fatal("after-session path crossed startup boundary")
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
	if err != nil || !strings.Contains(output, "./scripts/run-private-scanner --open") {
		t.Fatalf("public help output=%q err=%v", output, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("help invoked Go build: %v", err)
	}
	for _, arguments := range [][]string{{"--trading-date", "2026-02-30"}, {"--trading-date"}, {"--trading-date", "2026-08-10", "--trading-date", "2026-08-11"},
		{"--open", "--open"}, {"--unknown"}, {"positional"}, {"--help", "--open"}} {
		output, err := runPublic(arguments...)
		if err == nil || output == "" {
			t.Fatalf("public arguments %q output=%q err=%v", arguments, output, err)
		}
		if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("public arguments %q invoked Go build: %v", arguments, err)
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
	deadline := time.Now().Add(2 * time.Second)
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
