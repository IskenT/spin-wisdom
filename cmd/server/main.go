package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/IskenT/spin-wisdom/internal/app"
	"github.com/IskenT/spin-wisdom/internal/config"
)

func main() {
	cfg := config.DefaultConfig()
	config.SetupLogger(cfg.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	app, err := app.NewApplication(cfg)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	if err := app.Start(ctx); err != nil && err != context.Canceled {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()
	if err := app.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Application shutdown completed")
}
