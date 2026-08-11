package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/belandresj/live-equities-momentum-scanner/internal/privatelauncher"
)

func main() {
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
