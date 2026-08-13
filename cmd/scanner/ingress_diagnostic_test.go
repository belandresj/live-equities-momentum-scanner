package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

func TestPersistIngressIncidentIsBoundedProtectedAndNoOverwrite(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "diagnostics")
	at := time.Date(2026, 8, 12, 17, 0, 0, 123, time.UTC)
	incident := &operations.IngressIncident{Owner: operations.IngressOwnerAdapterTerminal, Source: "protocol", Reason: string(massive.TerminalFrameSlotCapacity),
		CapturedAt: at, EngineCapturedAt: at, IncomingFrameBytes: 4096, HistoryCount: 1}
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
		envelope.Incident.Queue.FramesQueued != 1024 || envelope.Incident.HistoryCount != 1 {
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
