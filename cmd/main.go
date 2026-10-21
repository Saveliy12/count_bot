package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/joho/godotenv"

	"github.com/Saveliy12/expenses/internal/app"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.NewApp(ctx)
	if err != nil {
		slog.Error("initialize application", "error", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}
