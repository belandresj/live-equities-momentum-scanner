package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ui "github.com/belandresj/live-equities-momentum-scanner/internal/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	address := flags.String("address", ui.DefaultAddress, "private loopback dashboard address")
	assets := flags.String("assets", "ui", "dashboard static asset directory")
	apiOrigin := flags.String("api-origin", ui.DefaultAPIOrigin, "exact loopback scanner API origin")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("dashboard accepts no positional arguments")
	}
	server, err := ui.Listen(ui.Config{Address: *address, AssetRoot: *assets, APIOrigin: *apiOrigin})
	if err != nil {
		return err
	}
	fmt.Printf("Scanner dashboard listening on http://%s\n", server.Address())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case err := <-server.Done():
		if err != nil {
			return fmt.Errorf("dashboard server: %w", err)
		}
		return nil
	case <-signals:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("dashboard shutdown: %w", err)
	}
	if err := <-server.Done(); err != nil {
		return fmt.Errorf("dashboard join: %w", err)
	}
	return nil
}
