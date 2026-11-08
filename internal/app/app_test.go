package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	app "github.com/remojii/academy-api/internal/app"
)

func TestLoadRequiresAppEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "")
	_, err := app.LoadConfig()
	require.Error(t, err)
}

func TestLoadRejectsUnknownEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "prodd")
	_, err := app.LoadConfig()
	require.Error(t, err)
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "prod")
	c, err := app.LoadConfig()
	require.NoError(t, err)
	require.False(t, c.IsDev())
}

func TestJsonLoggerRejectsBadLevel(t *testing.T) {
	_, err := app.Config{LogLevel: "loud"}.JsonLogger()
	require.Error(t, err)
}
