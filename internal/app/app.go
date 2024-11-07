package app

import (
	"context"
	"fmt"
	"sync"

	"log/slog"

	"github.com/IskenT/spin-wisdom/internal/config"
	"github.com/IskenT/spin-wisdom/internal/service/pow"
	"github.com/IskenT/spin-wisdom/internal/service/quotes"
	"github.com/IskenT/spin-wisdom/internal/service/quotes/repo/jsonquote"
	"github.com/IskenT/spin-wisdom/internal/tcp"
	"github.com/IskenT/spin-wisdom/internal/tcp_transport"
)

// Application ...
type Application struct {
	server       *tcp.Server
	powService   *pow.Service
	quoteService *quotes.Service
}

// NewApplication ...
func NewApplication(cfg *config.Config) (*Application, error) {
	powService := pow.NewService()
	repo, err := jsonquote.NewRepo()
	if err != nil {
		return nil, fmt.Errorf("creating quote repository: %w", err)
	}
	quoteService := quotes.NewService(repo)
	handler := tcp_transport.NewHandler(powService, quoteService, cfg.Server.Difficulty)

	server := tcp.NewServer(
		cfg.Server.Port,
		handler,
		tcp.WithMaxConnections(cfg.Server.MaxConnections),
		tcp.WithTimeouts(
			cfg.Server.ReadTimeout,
			cfg.Server.WriteTimeout,
		),
	)

	return &Application{
		server:       server,
		powService:   powService,
		quoteService: quoteService,
	}, nil
}

// Start ...
func (app *Application) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := app.server.Start(ctx); err != nil {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

// Shutdown ...
func (app *Application) Shutdown(ctx context.Context) error {
	slog.Info("Starting graceful shutdown...")
	errCh := make(chan error, 3)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := app.server.Shutdown(ctx); err != nil {
			errCh <- fmt.Errorf("server shutdown error: %w", err)
			return
		}
		slog.Debug("Server shutdown completed")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.powService.Cleanup(ctx); err != nil {
			errCh <- fmt.Errorf("PoW service cleanup error: %w", err)
			return
		}
		slog.Debug("PoW service cleanup completed")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.quoteService.Cleanup(ctx); err != nil {
			errCh <- fmt.Errorf("quote service cleanup error: %w", err)
			return
		}
		slog.Debug("Quote service cleanup completed")
	}()

	go func() {
		wg.Wait()
		close(errCh)
		slog.Debug("All cleanup tasks completed")
	}()

	var shutdownErrs []error
	for err := range errCh {
		shutdownErrs = append(shutdownErrs, err)
	}

	if ctx.Err() != nil {
		shutdownErrs = append(shutdownErrs, ctx.Err())
	}

	if len(shutdownErrs) > 0 {
		return fmt.Errorf("shutdown errors: %v", shutdownErrs)
	}

	slog.Info("Graceful shutdown completed successfully")
	return nil
}
