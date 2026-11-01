// Worker process: async jobs and periodic work.
//
// TODO(P6): start River (riverqueue/river) here with workers for payment
// webhook processing, past_due expiry sweep, provider reconciliation and
// account deletion cascade. Until then it only holds the DB connection open.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/remojii/academy-api/internal/platform/config"
	"github.com/remojii/academy-api/internal/platform/db"
	"github.com/remojii/academy-api/internal/schema"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := cfg.NewLogger()
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gdb, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(gdb) }()

	if cfg.AutoMigrate {
		if err := schema.Migrate(ctx, gdb); err != nil {
			return err
		}
	}

	logger.Info("worker started (no jobs registered yet)")
	<-ctx.Done()
	logger.Info("worker stopping")
	return nil
}
