package db_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/db"
)

// newTestDB creates a throwaway database on the server named by
// TEST_DATABASE_URL and returns a pool on it. Skips when unset.
func newTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	require.NoError(t, err)
	name := fmt.Sprintf("academy_test_%d", os.Getpid())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)

	u, err := url.Parse(base)
	require.NoError(t, err)
	u.Path = "/" + name
	pool, err := db.New(ctx, u.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	return ctx, pool
}

func TestMigrations(t *testing.T) {
	ctx, pool := newTestDB(t)

	require.Error(t, db.CheckMigrated(ctx, pool), "fresh database has pending migrations")
	require.NoError(t, db.MigrateUp(ctx, pool))
	require.NoError(t, db.CheckMigrated(ctx, pool))
	require.NoError(t, db.MigrateUp(ctx, pool), "idempotent")

	require.NoError(t, db.MigrateDown(ctx, pool))
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users'").Scan(&n))
	require.Zero(t, n)
}

func TestWithTxRollsBackAndJoins(t *testing.T) {
	ctx, pool := newTestDB(t)
	require.NoError(t, db.MigrateUp(ctx, pool))

	insert := func(ctx context.Context, email string) error {
		_, err := db.Conn(ctx, pool).Exec(ctx, "INSERT INTO users (email) VALUES ($1)", email)
		return err
	}
	count := func() (n int) {
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&n))
		return
	}

	err := db.WithTx(ctx, pool, func(ctx context.Context) error {
		require.NoError(t, insert(ctx, "a@example.com"))
		// nested call joins the outer transaction
		require.NoError(t, db.WithTx(ctx, pool, func(ctx context.Context) error { return insert(ctx, "b@example.com") }))
		return errors.New("boom")
	})
	require.Error(t, err)
	require.Zero(t, count(), "rollback undoes both inserts")

	require.NoError(t, db.WithTx(ctx, pool, func(ctx context.Context) error { return insert(ctx, "c@example.com") }))
	require.Equal(t, 1, count())
}
