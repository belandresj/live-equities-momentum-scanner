package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

func TestPMVPRecoveryObsCoordinatesTerminalEvidenceAndBoundedDurableRetry(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "diagnostics")
	at := time.Date(2026, 8, 17, 15, 35, 12, 0, time.UTC)
	sample := warmupOperatorSample(at)
	incident := &operations.IngressIncident{
		Owner: operations.IngressOwnerAdapterTerminal, Source: "heartbeat", Reason: "heartbeat_deadline", CapturedAt: at,
		Epoch: 2, Lifecycle: "recovering", LifecycleReason: "aggregate_epoch_lost", Hydration: sample.Metrics.Engine.Hydration,
		MaxProcessingDelay: 9 * time.Second, MaxProcessingDelayOneSecond: 857 * time.Millisecond,
		HeapAllocBytes: 1_500_000_000, HeapInUseBytes: 1_800_000_000, Goroutines: 32,
	}
	incident.Queue = massive.LiveQueueAccounting{FramesQueued: 17, QueuedBytes: 64 << 10, CapacityFrames: 512, CapacityBytes: 128 << 20}
	sample.IngressIncident = incident

	var stdout, stderr bytes.Buffer
	if err := newOperatorRenderer(&stdout, &stderr).Render(sample, false); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	persist := func(directory string, incident *operations.IngressIncident) (string, error) {
		attempts++
		if attempts == 1 {
			return "", errors.New("transient diagnostic failure")
		}
		return persistIngressIncident(directory, incident)
	}
	recorder := ingressDiagnosticRecorder{}
	if err := recorder.record(directory, incident, &stderr, false, persist); err != nil {
		t.Fatal(err)
	}
	if err := recorder.record(directory, incident, &stderr, false, persist); err != nil {
		t.Fatal(err)
	}
	if !recorder.persisted || attempts != 2 {
		t.Fatalf("recorder=%+v attempts=%d", recorder, attempts)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("durable diagnostic entries=%v err=%v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
	if err != nil || !bytes.Contains(data, []byte(`"Reason":"heartbeat_deadline"`)) {
		t.Fatalf("durable diagnostic=%s err=%v", data, err)
	}
	log := stderr.String()
	for _, evidence := range []string{"Ingress first cause", "heartbeat/heartbeat_deadline", "Ingress evidence", "max delay 9s", "persistence failed · attempt 1/2", "Ingress diagnostic persisted"} {
		if !strings.Contains(log, evidence) {
			t.Fatalf("missing %q in stderr=%q", evidence, log)
		}
	}

	failedAttempts := 0
	failed := ingressDiagnosticRecorder{}
	alwaysFail := func(string, *operations.IngressIncident) (string, error) {
		failedAttempts++
		return "", errors.New("still unavailable")
	}
	for range 4 {
		_ = failed.record(directory, incident, &bytes.Buffer{}, false, alwaysFail)
	}
	if err := failed.record(directory, incident, &bytes.Buffer{}, true, alwaysFail); err == nil || failedAttempts != 3 {
		t.Fatalf("bounded final retry err=%v attempts=%d", err, failedAttempts)
	}
}

func TestPersistIngressIncidentIsBoundedProtectedAndNoOverwrite(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "diagnostics")
	at := time.Date(2026, 8, 12, 17, 0, 0, 123, time.UTC)
	incident := &operations.IngressIncident{Owner: operations.IngressOwnerAdapterTerminal, Source: "protocol", Reason: string(massive.TerminalFrameSlotCapacity),
		CapturedAt: at, EngineCapturedAt: at, IncomingFrameBytes: 4096, HistoryCount: 1,
		MaxProcessingDelay: 9 * time.Second, MaxProcessingDelayOneSecond: 857 * time.Millisecond,
		HeapAllocBytes: 1_500_000_000, HeapInUseBytes: 1_800_000_000, Goroutines: 32}
	incident.Queue = massive.LiveQueueAccounting{FramesQueued: 1024, QueuedBytes: 8 << 20, CapacityFrames: 1024, CapacityBytes: 128 << 20}
	incident.History[0] = operations.IngressDiagnosticSample{CapturedAt: at.Add(-time.Second), QueuedFrames: 900, QueuedBytes: 7 << 20}
	path, err := persistIngressIncident(directory, incident)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("diagnostic mode: info=%v err=%v", info, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Schema   int                        `json:"schema"`
		Incident operations.IngressIncident `json:"incident"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Schema != ingressDiagnosticSchema || envelope.Incident.Reason != string(massive.TerminalFrameSlotCapacity) ||
		envelope.Incident.Queue.FramesQueued != 1024 || envelope.Incident.HistoryCount != 1 || envelope.Incident.HeapInUseBytes != 1_800_000_000 ||
		envelope.Incident.MaxProcessingDelay != 9*time.Second || envelope.Incident.Goroutines != 32 {
		t.Fatalf("persisted diagnostic=%s", data)
	}
	if _, err := persistIngressIncident(directory, incident); err == nil {
		t.Fatal("diagnostic overwrote existing incident")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("atomic publication left unexpected files: %v", entries)
	}
}
