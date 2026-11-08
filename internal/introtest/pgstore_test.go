package introtest_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/db"
	"github.com/remojii/academy-api/internal/introtest"
)

// newTestDB creates a migrated throwaway database on the server named by
// TEST_DATABASE_URL. Skips when unset.
func newTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	require.NoError(t, err)
	name := fmt.Sprintf("academy_introtest_%d", os.Getpid())
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
	require.NoError(t, db.MigrateUp(ctx, pool))
	return ctx, pool
}

func TestPgResultStore(t *testing.T) {
	ctx, pool := newTestDB(t)
	s := introtest.NewPgResultStore(pool)

	var uid uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('a@example.com') RETURNING id`).Scan(&uid))

	_, err := s.Latest(ctx, uid)
	require.ErrorIs(t, err, introtest.ErrNoResult)

	first, err := s.Create(ctx, uid, introtest.Scores{Openness: 1, Conscientiousness: 2, Extraversion: 3, Agreeableness: 4, Neuroticism: 5})
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	second, err := s.Create(ctx, uid, introtest.Scores{Openness: 4.25, Conscientiousness: 2, Extraversion: 3, Agreeableness: 4, Neuroticism: 1})
	require.NoError(t, err)

	got, err := s.Latest(ctx, uid)
	require.NoError(t, err)
	require.Equal(t, second.ID, got.ID, "retake: newest row is current")
	require.InDelta(t, 4.25, got.Openness, 1e-9)
	require.NotEqual(t, first.ID, got.ID)

	_, err = s.Create(ctx, uid, introtest.Scores{Openness: 5.5, Conscientiousness: 2, Extraversion: 3, Agreeableness: 4, Neuroticism: 1})
	require.Error(t, err, "DB check constraint rejects out-of-range scores")

	_, err = s.Create(ctx, uuid.New(), introtest.Scores{Openness: 1, Conscientiousness: 1, Extraversion: 1, Agreeableness: 1, Neuroticism: 1})
	require.Error(t, err, "unknown user violates the foreign key")

	_, err = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
	require.NoError(t, err)
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM personality_results`).Scan(&n))
	require.Zero(t, n, "results are deleted with the user")
}
