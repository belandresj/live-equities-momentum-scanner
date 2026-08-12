package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func composeLiveRuntime(startup, lifetime context.Context, binding reference.Binding, config operations.Config, clock func() time.Time, mode, directory string) (*operations.Runtime, *checkpoint.Store, error) {
	switch mode {
	case "off":
		runtime, err := operations.New(startup, binding, config, clock)
		return runtime, nil, err
	case "on":
		checkpointPath, err := filepath.Abs(directory)
		if err != nil {
			return nil, nil, errors.New("resolve checkpoint directory")
		}
		store, err := checkpoint.NewStore(checkpoint.StoreConfig{Directory: checkpointPath, BindingIdentity: binding.Identity(), ArtifactByteLimit: 64 << 20, OperationDeadline: 30 * time.Second})
		if err != nil {
			return nil, nil, fmt.Errorf("configure checkpoint store: %w", err)
		}
		writer, err := checkpoint.NewWriter(lifetime, store)
		if err != nil {
			return nil, nil, err
		}
		runtime, err := operations.NewWithCheckpoint(startup, binding, config, clock, writer)
		if err != nil {
			writer.Close()
			return nil, nil, err
		}
		return runtime, store, nil
	default:
		return nil, nil, errors.New("checkpoint-mode must be on or off")
	}
}
