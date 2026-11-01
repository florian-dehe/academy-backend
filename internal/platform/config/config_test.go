package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/platform/config"
)

func TestLoadRequiresAppEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "")
	_, err := config.Load()
	require.Error(t, err)
}

func TestLoadRejectsUnknownEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "prodd")
	_, err := config.Load()
	require.Error(t, err)
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "prod")
	c, err := config.Load()
	require.NoError(t, err)
	require.False(t, c.IsDev())
	require.False(t, c.AutoMigrate)
}

func TestNewLoggerRejectsBadLevel(t *testing.T) {
	_, err := config.Config{LogLevel: "loud"}.NewLogger()
	require.Error(t, err)
}
