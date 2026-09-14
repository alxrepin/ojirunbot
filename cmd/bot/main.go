package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ojirun/internal/bootstrap"
)

func main() {
	if err := run(); err != nil {
		slog.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
	slog.Info("bot stopped")
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(ctx)
	if err != nil {
		return fmt.Errorf("startup: %w", err)
	}
	defer app.Stop()

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
