package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
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
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointModeDefaultPathStillConstructsExistingComposition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	binding := scannerTestBinding(t)
	config := operations.DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	runtime, store, err := composeLiveRuntime(ctx, ctx, binding, config, func() time.Time { return binding.SessionStart() }, "on", filepath.Join(t.TempDir(), "checkpoints"))
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("default checkpoint composition did not construct store/writer")
	}
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
}
