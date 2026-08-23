package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/belandresj/live-equities-momentum-scanner/internal/privatelauncher"
)

func main() {
	// Launcher reporting is best effort. Preserve the supervisor when its
	// foreground terminal has gone away; Run latches the failed writers.
	signal.Ignore(syscall.SIGPIPE)
	ctx, stop := privatelauncher.NotifyContext(context.Background())
	defer stop()
	root := os.Getenv("PRIVATE_SCANNER_REPO_ROOT")
	if root == "" {
		fmt.Fprintln(os.Stderr, "private scanner repository root was not supplied by scripts/run-private-scanner")
		os.Exit(1)
	}
	if err := privatelauncher.Run(ctx, root, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
