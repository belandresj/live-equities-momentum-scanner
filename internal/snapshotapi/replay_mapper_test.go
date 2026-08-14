package snapshotapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replay"
)

func TestReplaySchemaRejectsMixedPhasePublicationAndAccounting(t *testing.T) {
	capture := replaySchemaCapture()
	snapshot, err := mapCaptureView(capture)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Replay == nil || snapshot.Replay.Phase != "observing" || snapshot.Status.BackendReady || snapshot.Status.ReadinessReason != "not_live_mode" ||
		len(snapshot.Rows) != 1 || snapshot.Rows[0].Tape5s.Reason != "replay_unavailable" || snapshot.Rows[0].Spread.Reason != "replay_unavailable" {
		t.Fatalf("replay schema = %+v", snapshot)
	}
	mutations := []struct {
		name string
		edit func(*Snapshot)
	}{
		{"missing replay root", func(value *Snapshot) { value.Replay = nil }},
		{"noncanonical artifact identity", func(value *Snapshot) { value.Replay.ArtifactID = "sha256:xyz" }},
		{"source record contradiction", func(value *Snapshot) { value.Replay.Source.UnreadRecords = "2" }},
		{"source group contradiction", func(value *Snapshot) { value.Replay.Source.RemainingGroups = "61" }},
		{"window contradiction", func(value *Snapshot) { value.Replay.Window.ObservationSecondsRemaining = "61" }},
		{"boundary contradiction", func(value *Snapshot) { value.Replay.Window.ObservationBoundariesPublished = "60" }},
		{"mixed retained lifecycle", func(value *Snapshot) {
			value.Replay.Phase, value.Replay.Completion = "retained_success", "requested_end"
		}},
		{"mixed suppressed lifecycle", func(value *Snapshot) { value.Replay.Phase = "suppressed"; value.Replay.ScheduleLagMS = nil }},
		{"completion outside retained success", func(value *Snapshot) { value.Replay.Completion = "artifact_end" }},
		{"logical time before observation", func(value *Snapshot) { value.Replay.LogicalTime = "2026-08-08T15:58:00Z" }},
		{"engine publication after logical time", func(value *Snapshot) { value.Publication.GeneratedAt = "2026-08-08T16:00:01Z" }},
		{"live mode with replay root", func(value *Snapshot) { value.Publication.RunMode = "live" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := cloneReplaySnapshot(t, snapshot)
			mutation.edit(&changed)
			if err := validateSnapshot(changed); err == nil {
				t.Fatal("invalid replay snapshot reached success")
			}
		})
	}
}

func replaySchemaCapture() operations.SnapshotCaptureView {
	value := schemaCapture()
	logical := time.Date(2026, 8, 8, 16, 0, 0, 0, time.UTC)
	observationStart, observationEnd := logical.Add(-time.Minute), logical.Add(time.Minute)
	artifactEnd := observationEnd.Add(time.Minute)
	value.Engine.Publication.RunMode = engine.RunModeReplay
	value.Engine.Publication.Lifecycle = "replaying"
	value.Engine.Publication.LifecycleReason = "replay_start"
	value.Engine.Operational.RunMode = engine.RunModeReplay
	value.Engine.Operational.Lifecycle = "replaying"
	value.Engine.Operational.LifecycleReason = "replay_start"
	value.Status.BackendReady = false
	value.Status.Reason = operations.ReasonNotLiveMode
	value.Status.Lifecycle = "replaying"
	value.Replay = &operations.ReplayCaptureView{Phase: operations.ReplayObserving,
		ArtifactID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ArtifactEnd: artifactEnd,
		ObservationStart: observationStart, ObservationEnd: observationEnd, LogicalTime: logical, ScheduleLag: durationPointer(250 * time.Millisecond),
		Source: replay.Accounting{ArtifactRecords: 2, CompletedRecordDispositions: 1, UnreadRecords: 1, PlannedGroups: 121, CompletedGroups: 61, RemainingGroups: 60},
		Window: operations.ReplayWindowAccounting{WarmupGroupsPlanned: 1, WarmupGroupsCompleted: 1, ObservationSecondsPlanned: 120,
			ObservationSecondsCompleted: 60, ObservationSecondsRemaining: 60, ObservationBoundariesPublished: 61}}
	return value
}

func cloneReplaySnapshot(t *testing.T, value Snapshot) Snapshot {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result Snapshot
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func durationPointer(value time.Duration) *time.Duration { return &value }
