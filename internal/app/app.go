package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	serviceProvider *serviceProvider
}

func NewApp(ctx context.Context) (*App, error) {
	a := &App{serviceProvider: newServiceProvider()}

	if _, err := a.serviceProvider.Config(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.serviceProvider.Close(); err != nil {
			slog.Error("close database", "error", err)
		}
	}()

	server, err := a.serviceProvider.HTTPServer(ctx)
	if err != nil {
		return err
	}

	botRunner, err := a.serviceProvider.TelegramBot(ctx)
	if err != nil {
		return err
	}

	if err := a.serviceProvider.SeedDemoUser(ctx); err != nil {
		return err
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	var background sync.WaitGroup
	if botRunner != nil {
		background.Add(1)
		go func() {
			defer background.Done()
			botRunner.Start(runCtx)
		}()
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("serve HTTP: %w", err)
		}
	}()

	var runErr error
	select {
	case runErr = <-serverErrors:
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	cancelRun()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	shutdownErr := server.Shutdown(shutdownCtx)
	background.Wait()

	if runErr != nil {
		return runErr
	}
	if shutdownErr != nil {
		return fmt.Errorf("shutdown HTTP server: %w", shutdownErr)
	}
	slog.Info("http server stopped")

	return nil
}
