package main

import (
	"context"
	"errors"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/checkpoint"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

func composeLiveRuntime(startup, _ context.Context, binding reference.Binding, config operations.Config, clock func() time.Time, mode, _ string) (*operations.Runtime, *checkpoint.Store, error) {
	switch mode {
	case "off":
		runtime, err := operations.New(startup, binding, config, clock)
		return runtime, nil, err
	case "on":
		return nil, nil, errors.New("checkpoint-mode on is incompatible with the live feature MVP; use --checkpoint-mode=off for fresh hydration")
	default:
		return nil, nil, errors.New("checkpoint-mode must be on or off")
	}
}
