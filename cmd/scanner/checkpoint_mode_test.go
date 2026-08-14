package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

func TestCheckpointOffProductionCompositionConstructsAndSubmitsNoWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	binding := scannerTestBinding(t)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	directory := filepath.Join(t.TempDir(), "must-not-exist")
	runtime, store, err := composeLiveRuntime(ctx, ctx, binding, config, func() time.Time { return binding.SessionStart() }, "off", directory)
	if err != nil {
		t.Fatal(err)
	}
	if store != nil {
		t.Fatal("checkpoint-off constructed a store")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("checkpoint-off touched directory: %v", err)
	}
	operational := runtime.Engine().ObserveOperational()
	checkpointOps := runtime.Engine().CheckpointOperations()
	metrics := runtime.Metrics()
	if operational.InstalledCheckpoint || checkpointOps.Submitted != 0 || checkpointOps.Outstanding != 0 || metrics.Checkpoint.Submitted != 0 || metrics.Checkpoint.InProgress != 0 || metrics.Checkpoint.Pending != 0 {
		t.Fatalf("checkpoint-off status operational=%+v engine=%+v writer=%+v", operational, checkpointOps, metrics.Checkpoint)
	}
	handler, err := snapshotapi.NewHandler(runtime, snapshotapi.HandlerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	for poll := 0; poll < 10; poll++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/snapshot", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("checkpoint-off API poll %d status=%d body=%s", poll+1, response.Code, response.Body.String())
		}
	}
	checkpointOps, metrics = runtime.Engine().CheckpointOperations(), runtime.Metrics()
	if checkpointOps.Submitted != 0 || checkpointOps.Outstanding != 0 || metrics.Checkpoint.Submitted != 0 || metrics.Checkpoint.InProgress != 0 || metrics.Checkpoint.Pending != 0 {
		t.Fatalf("checkpoint-off API polls created work engine=%+v writer=%+v", checkpointOps, metrics.Checkpoint)
	}
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointModeOnFailsBeforeRestoreOrWriterConstruction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	binding := scannerTestBinding(t)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	directory := filepath.Join(t.TempDir(), "checkpoints")
	runtime, store, err := composeLiveRuntime(ctx, ctx, binding, config, func() time.Time { return binding.SessionStart() }, "on", directory)
	if err == nil || runtime != nil || store != nil || !strings.Contains(err.Error(), "incompatible with the live feature MVP") {
		t.Fatalf("checkpoint guard runtime=%v store=%v err=%v", runtime, store, err)
	}
	if _, statErr := os.Stat(directory); !os.IsNotExist(statErr) {
		t.Fatalf("checkpoint guard touched restore path: %v", statErr)
	}
}
