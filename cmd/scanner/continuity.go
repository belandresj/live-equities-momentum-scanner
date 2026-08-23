package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

// terminalSink makes terminal output observational. Once a stream rejects a
// write, subsequent records are discarded rather than repeatedly attempting a
// known-broken descriptor. The peer notification deliberately uses a direct
// write, so two failed descriptors cannot recursively report each other.
type terminalSink struct {
	mu       sync.Mutex
	name     string
	output   io.Writer
	failed   bool
	reported bool
	peer     *terminalSink
}

func newTerminalSinks(stdout, stderr io.Writer) (*terminalSink, *terminalSink) {
	standard := &terminalSink{name: "stdout", output: stdout}
	errorOutput := &terminalSink{name: "stderr", output: stderr}
	standard.peer, errorOutput.peer = errorOutput, standard
	return standard, errorOutput
}

func (sink *terminalSink) Write(record []byte) (int, error) {
	if sink == nil {
		return len(record), nil
	}
	sink.mu.Lock()
	if sink.failed {
		sink.mu.Unlock()
		return len(record), nil
	}
	written, err := sink.output.Write(record)
	if err == nil && written != len(record) {
		err = io.ErrShortWrite
	}
	if err == nil {
		sink.mu.Unlock()
		return len(record), nil
	}
	sink.failed = true
	report := !sink.reported
	sink.reported = true
	peer, name := sink.peer, sink.name
	sink.mu.Unlock()
	if report && peer != nil {
		peer.reportPeerFailure(name)
	}
	return len(record), nil
}

func (sink *terminalSink) reportPeerFailure(name string) {
	sink.mu.Lock()
	if sink.failed {
		sink.mu.Unlock()
		return
	}
	_, err := fmt.Fprintf(sink.output, "Scanner terminal %s became unavailable; suppressing further output to that sink\n", name)
	if err != nil {
		sink.failed = true
	}
	sink.mu.Unlock()
}

func (sink *terminalSink) Failed() bool {
	if sink == nil {
		return true
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.failed
}

type snapshotServer interface {
	Done() <-chan error
	MappingFailures() <-chan snapshotapi.MappingFailure
	Address() string
	Shutdown(context.Context) error
}

type snapshotServerFactory func(snapshotapi.CaptureSource, snapshotapi.ServerConfig) (snapshotServer, error)

type apiRestartTimer interface {
	Chan() <-chan time.Time
	Stop() bool
}

type standardAPIRestartTimer struct{ timer *time.Timer }

func (timer standardAPIRestartTimer) Chan() <-chan time.Time { return timer.timer.C }
func (timer standardAPIRestartTimer) Stop() bool             { return timer.timer.Stop() }

type apiRestartTimerFactory func(time.Duration) apiRestartTimer

var apiRestartBackoff = [...]time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}

// apiSupervisor owns only the replaceable HTTP listener. It never owns or
// cancels the runtime that supplies immutable captures.
type apiSupervisor struct {
	source  snapshotapi.CaptureSource
	config  snapshotapi.ServerConfig
	factory snapshotServerFactory
	timers  apiRestartTimerFactory

	server          snapshotServer
	done            <-chan error
	mappingFailures <-chan snapshotapi.MappingFailure
	restartTimer    apiRestartTimer
	attempts        int
	unavailable     bool
}

type liveSupervisorConfig struct {
	context               context.Context
	api                   *apiSupervisor
	done                  <-chan error
	ticks                 <-chan time.Time
	readinessTicks        <-chan time.Time
	capture               func() (liveOperatorSample, error)
	observeWatermarkStale func() error
	render                func(liveOperatorSample, bool) error
	record                func(*operations.IngressIncident, bool) error
	encodeMapping         func(snapshotapi.MappingFailure) error
	shutdown              func(bool) error
	onUnavailable         func(error)
}

// startLiveAfterAPI preserves the fail-fast startup boundary: market intake is
// not launched until the configured snapshot endpoint has bound successfully.
func startLiveAfterAPI(startAPI func() (*apiSupervisor, error), startLive func() <-chan error) (*apiSupervisor, <-chan error, error) {
	if startAPI == nil || startLive == nil {
		return nil, nil, errors.New("invalid live startup configuration")
	}
	api, err := startAPI()
	if err != nil {
		return nil, nil, err
	}
	return api, startLive(), nil
}

// superviseLive is the scanner-local lifecycle join point. Output and API
// events are deliberately absent from its runtime-terminal path; only context,
// capture integrity, and the live composition result can invoke shutdown.
func superviseLive(config liveSupervisorConfig) error {
	if config.context == nil || config.api == nil || config.done == nil || config.capture == nil || config.render == nil || config.record == nil || config.encodeMapping == nil || config.shutdown == nil {
		return errors.New("invalid live supervisor configuration")
	}
	for {
		select {
		case <-config.context.Done():
			if config.observeWatermarkStale != nil {
				_ = config.observeWatermarkStale()
			}
			diagnosticErr := config.record(nil, true)
			return errors.Join(config.context.Err(), diagnosticErr, config.shutdown(false))
		case err := <-config.done:
			var diagnosticErr error
			incident := (*operations.IngressIncident)(nil)
			if sample, captureErr := config.capture(); captureErr == nil {
				_ = config.render(sample, true)
				incident = sample.IngressIncident
			}
			if config.observeWatermarkStale != nil {
				_ = config.observeWatermarkStale()
			}
			diagnosticErr = config.record(incident, true)
			if shutdownErr := config.shutdown(true); shutdownErr != nil {
				return shutdownErr
			}
			return errors.Join(err, diagnosticErr)
		case <-config.api.Done():
			config.api.UnexpectedTermination()
		case <-config.api.RestartTimer():
			if restartErr := config.api.RestartDue(); restartErr != nil && config.api.Unavailable() && config.onUnavailable != nil {
				config.onUnavailable(restartErr)
			}
		case failure := <-config.api.MappingFailures():
			_ = config.encodeMapping(failure)
		case <-config.readinessTicks:
			if config.observeWatermarkStale != nil {
				_ = config.observeWatermarkStale()
			}
		case <-config.ticks:
			sample, err := config.capture()
			if err != nil {
				if shutdownErr := config.shutdown(false); shutdownErr != nil {
					return shutdownErr
				}
				return errors.New("capture operational status")
			}
			_ = config.record(sample.IngressIncident, false)
			_ = config.render(sample, false)
		}
	}
}

func newAPISupervisor(source snapshotapi.CaptureSource, config snapshotapi.ServerConfig, factory snapshotServerFactory, timers apiRestartTimerFactory) (*apiSupervisor, error) {
	if source == nil || factory == nil {
		return nil, errors.New("invalid snapshot API supervisor")
	}
	if timers == nil {
		timers = func(delay time.Duration) apiRestartTimer { return standardAPIRestartTimer{timer: time.NewTimer(delay)} }
	}
	supervisor := &apiSupervisor{source: source, config: config, factory: factory, timers: timers}
	if err := supervisor.install(); err != nil {
		return nil, err
	}
	return supervisor, nil
}

func (supervisor *apiSupervisor) install() error {
	server, err := supervisor.factory(supervisor.source, supervisor.config)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.New("snapshot API factory returned no server")
	}
	supervisor.server = server
	supervisor.done = server.Done()
	supervisor.mappingFailures = server.MappingFailures()
	if address := server.Address(); address != "" {
		supervisor.config.Address = address
	}
	supervisor.unavailable = false
	return nil
}

func (supervisor *apiSupervisor) Done() <-chan error {
	if supervisor == nil {
		return nil
	}
	return supervisor.done
}

func (supervisor *apiSupervisor) MappingFailures() <-chan snapshotapi.MappingFailure {
	if supervisor == nil {
		return nil
	}
	return supervisor.mappingFailures
}

func (supervisor *apiSupervisor) RestartTimer() <-chan time.Time {
	if supervisor == nil || supervisor.restartTimer == nil {
		return nil
	}
	return supervisor.restartTimer.Chan()
}

// UnexpectedTermination retires the exact active generation before scheduling
// one serial replacement attempt. A closed or retired Done channel is never
// retained in the select set.
func (supervisor *apiSupervisor) UnexpectedTermination() {
	if supervisor == nil {
		return
	}
	supervisor.server, supervisor.done, supervisor.mappingFailures = nil, nil, nil
	supervisor.scheduleNext()
}

func (supervisor *apiSupervisor) scheduleNext() {
	if supervisor.attempts >= len(apiRestartBackoff) {
		supervisor.restartTimer = nil
		supervisor.unavailable = true
		return
	}
	delay := apiRestartBackoff[supervisor.attempts]
	supervisor.restartTimer = supervisor.timers(delay)
}

// RestartDue makes one replacement attempt. The returned error is deliberately
// operational-only: failed listeners leave the scanner runtime untouched.
func (supervisor *apiSupervisor) RestartDue() error {
	if supervisor == nil || supervisor.restartTimer == nil {
		return nil
	}
	supervisor.restartTimer = nil
	supervisor.attempts++
	if err := supervisor.install(); err != nil {
		if supervisor.attempts >= len(apiRestartBackoff) {
			supervisor.unavailable = true
			return err
		}
		supervisor.scheduleNext()
		return err
	}
	// A successful replacement begins a new, distinct server lifetime.
	supervisor.attempts = 0
	return nil
}

func (supervisor *apiSupervisor) Unavailable() bool {
	return supervisor != nil && supervisor.unavailable
}

// Shutdown cancels pending retry work and joins only the currently installed
// listener. Earlier generations have already delivered their Done result.
func (supervisor *apiSupervisor) Shutdown(ctx context.Context) error {
	if supervisor == nil {
		return nil
	}
	if supervisor.restartTimer != nil {
		supervisor.restartTimer.Stop()
		supervisor.restartTimer = nil
	}
	if supervisor.server == nil {
		return nil
	}
	server, done := supervisor.server, supervisor.done
	supervisor.server, supervisor.done, supervisor.mappingFailures = nil, nil, nil
	shutdownErr := server.Shutdown(ctx)
	select {
	case <-done:
	case <-ctx.Done():
		return errors.Join(shutdownErr, errors.New("snapshot API shutdown deadline exceeded"))
	}
	return shutdownErr
}
