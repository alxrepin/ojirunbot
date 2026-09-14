package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ojirun/internal/bootstrap"
	"ojirun/internal/context/infrastructure/db/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}

func run() error {
	_ = bootstrap.LoadDotEnv(".env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://nutrition:nutrition@localhost:5432/nutrition?sslmode=disable"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer store.Close()

	return store.Migrate(ctx, "migrations")
}
