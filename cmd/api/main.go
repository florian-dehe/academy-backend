// Command api runs the Academy HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/remojii/academy-api/internal/app"
	"github.com/remojii/academy-api/internal/identity"
	idstore "github.com/remojii/academy-api/internal/identity/store"
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

	var resolver identity.TokenResolver = identity.UnconfiguredResolver{}
	switch {
	case cfg.JobBoardBaseURL != "":
		resolver, err = identity.NewJobBoardResolver(identity.JobBoardConfig{
			BaseURL:  cfg.JobBoardBaseURL,
			MePath:   cfg.JobBoardMePath,
			CacheTTL: time.Duration(cfg.JobBoardCacheTTLSec) * time.Second,
		}, idstore.Provisioner{S: idstore.New(gdb)})
		if err != nil {
			return err
		}
		logger.Info("auth: job board token validation", "base_url", cfg.JobBoardBaseURL)
	case cfg.IsDev():
		resolver = devResolver()
		if cfg.AutoMigrate {
			if err := seedDevUsers(ctx, gdb); err != nil {
				return err
			}
		}
		logger.Warn("dev auth enabled: tokens learner-token, editor-token, admin-token")
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: app.NewRouter(app.Deps{
			Logger:   logger,
			Resolver: resolver,
			Ready:    func(ctx context.Context) error { return db.Ping(ctx, gdb) },
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}

// Fixed ids so dev tokens survive restarts and match rows in users.
var devUsers = []struct {
	token string
	user  identity.User
}{
	{"learner-token", identity.User{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Email: "learner@example.com", Role: identity.RoleLearner}},
	{"editor-token", identity.User{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), Email: "editor@example.com", Role: identity.RoleContentEditor}},
	{"admin-token", identity.User{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), Email: "admin@example.com", Role: identity.RoleAdmin}},
}

// devResolver returns fixed users for local manual testing. Never used outside APP_ENV=dev.
func devResolver() identity.StaticResolver {
	m := identity.StaticResolver{}
	for _, d := range devUsers {
		m[d.token] = d.user
	}
	return m
}

// seedDevUsers inserts the dev users so FKs from progress tables resolve.
func seedDevUsers(ctx context.Context, gdb *gorm.DB) error {
	for _, d := range devUsers {
		row := idstore.User{ID: d.user.ID, Email: d.user.Email, Role: string(d.user.Role)}
		if err := gdb.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return fmt.Errorf("seed dev user %s: %w", d.user.Email, err)
		}
	}
	return nil
}
