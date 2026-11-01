package schema_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/platform/db"
	"github.com/remojii/academy-api/internal/schema"
)

// Integration test: needs a throwaway Postgres, e.g.
//
//	TEST_DATABASE_URL=postgres://academy:academy@localhost:5432/academy_test?sslmode=disable
func TestMigrateIsIdempotentAndSeeds(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	gdb, err := db.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close(gdb) })

	require.NoError(t, schema.Migrate(ctx, gdb))
	require.NoError(t, schema.Migrate(ctx, gdb), "second run must be a no-op")

	var levels, plans int64
	require.NoError(t, gdb.Table("levels").Count(&levels).Error)
	require.NoError(t, gdb.Table("plans").Where("code = ?", "academy_plus").Count(&plans).Error)
	require.EqualValues(t, 1, levels)
	require.EqualValues(t, 1, plans)
}
