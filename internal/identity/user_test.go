package identity_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/db"
	"github.com/remojii/academy-api/internal/identity"
)

// --- PgUserStore against a real database (needs TEST_DATABASE_URL) ---

func TestPgUserStore(t *testing.T) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	require.NoError(t, err)
	name := fmt.Sprintf("academy_identity_test_%d", os.Getpid())
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

	s := identity.NewPgUserStore(pool)
	lu := identity.LaravelUser{ID: "7", Email: "x@example.com", Name: "X Y"}

	_, err = s.ByLaravelID(ctx, "7")
	require.ErrorIs(t, err, identity.ErrUserNotFound)

	// concurrent first calls: exactly one creates, nobody errors
	var created int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	ids := map[uuid.UUID]bool{}
	var errs []error
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acc, c, err := s.Create(ctx, lu)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			if c {
				created++
			}
			ids[acc.ID] = true
		}()
	}
	wg.Wait()
	require.Empty(t, errs)
	require.EqualValues(t, 1, created)
	require.Len(t, ids, 1)

	acc, err := s.ByLaravelID(ctx, "7")
	require.NoError(t, err)
	require.Equal(t, identity.RoleLearner, acc.Role)
	require.Equal(t, "X Y", *acc.DisplayName)

	require.NoError(t, s.Delete(ctx, acc.ID))
	_, err = s.ByLaravelID(ctx, "7")
	require.ErrorIs(t, err, identity.ErrUserNotFound)
	_, c, err := s.Create(ctx, lu)
	require.NoError(t, err)
	require.True(t, c, "recreated after delete")

	// same email, different Laravel id -> conflict, not a 500
	_, _, err = s.Create(ctx, identity.LaravelUser{ID: "8", Email: "x@example.com"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflict")
}
