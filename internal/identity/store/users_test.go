package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/remojii/academy-api/internal/identity/store"
	"github.com/remojii/academy-api/internal/platform/db"
	"github.com/remojii/academy-api/internal/schema"
)

// Integration test; see internal/schema/schema_test.go for TEST_DATABASE_URL.
func TestUserCRUD(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	gdb, err := db.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close(gdb) })
	require.NoError(t, schema.Migrate(ctx, gdb))

	s := store.New(gdb)
	name := "Ada"
	u, err := s.CreateUser(ctx, store.CreateUserParams{Email: t.Name() + "@example.com", DisplayName: &name})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.DeleteUser(ctx, u.ID) })
	require.NotZero(t, u.ID)
	require.Equal(t, "learner", u.Role, "DB default role")

	got, err := s.GetUserByEmail(ctx, u.Email)
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)

	newName := "Grace"
	upd, err := s.UpdateUserDisplayName(ctx, u.ID, &newName)
	require.NoError(t, err)
	require.Equal(t, "Grace", *upd.DisplayName)
	require.True(t, upd.UpdatedAt.After(u.UpdatedAt) || upd.UpdatedAt.Equal(u.UpdatedAt))

	// transaction rollback leaves nothing behind
	errBoom := errors.New("boom")
	err = db.WithTx(ctx, gdb, func(ctx context.Context) error {
		if _, err := s.CreateUser(ctx, store.CreateUserParams{Email: "rollback@example.com"}); err != nil {
			return err
		}
		return errBoom
	})
	require.ErrorIs(t, err, errBoom)
	_, err = s.GetUserByEmail(ctx, "rollback@example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, s.DeleteUser(ctx, u.ID))
	_, err = s.GetUserByID(ctx, u.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
