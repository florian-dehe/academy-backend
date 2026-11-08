package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/remojii/academy-api/migrations"
)

func provider(sqlDB *sql.DB) (*goose.Provider, error) {
	// goose takes a Postgres session lock, so concurrent runners are safe.
	return goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
}

// MigrateUp applies all pending migrations. Used by tests and `make migrate-up`
// style tooling; the API itself only calls CheckMigrated.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	p, err := provider(sqlDB)
	if err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back every migration.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	p, err := provider(sqlDB)
	if err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		return fmt.Errorf("db: migrate down: %w", err)
	}
	return nil
}

// CheckMigrated fails if the database has pending migrations. The API calls it
// at startup instead of migrating: schema changes are an explicit deploy step.
func CheckMigrated(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	p, err := provider(sqlDB)
	if err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	has, err := p.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("db: check migrations: %w", err)
	}
	if has {
		return errors.New("db: pending migrations; run `make migrate-up`")
	}
	return nil
}
