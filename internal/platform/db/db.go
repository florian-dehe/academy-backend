// Package db wraps GORM (Postgres via pgx) and provides a transaction helper.
//
// WithTx stores the open transaction in the context. Module stores call
// Conn(ctx, gdb) so the same code runs inside or outside a transaction
// (e.g. XP granted in the same transaction as module completion).
package db

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type txKey struct{}

// New opens a GORM handle on Postgres and pings it.
func New(ctx context.Context, url string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("db: sql handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return gdb, nil
}

// Close closes the underlying connection pool.
func Close(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping checks the database is reachable.
func Ping(ctx context.Context, gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// WithTx runs fn in a transaction. If ctx already carries a transaction, fn
// joins it. fn receives a context carrying the transaction.
func WithTx(ctx context.Context, gdb *gorm.DB, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn returns the transaction in ctx if any, else gdb, bound to ctx.
func Conn(ctx context.Context, gdb *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return gdb.WithContext(ctx)
}
