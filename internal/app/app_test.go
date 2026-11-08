package app_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	app "github.com/remojii/academy-api/internal/app"
	"github.com/remojii/academy-api/internal/identity"
)

func TestLoadRequiresAppEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LARAVEL_BASE_URL", "http://laravel.test")
	t.Setenv("APP_ENV", "")
	_, err := app.LoadConfig()
	require.Error(t, err)
}

func TestLoadRejectsUnknownEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LARAVEL_BASE_URL", "http://laravel.test")
	t.Setenv("APP_ENV", "prodd")
	_, err := app.LoadConfig()
	require.Error(t, err)
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LARAVEL_BASE_URL", "http://laravel.test")
	t.Setenv("APP_ENV", "prod")
	c, err := app.LoadConfig()
	require.NoError(t, err)
	require.False(t, c.IsDev())
}

func TestJSONLoggerRejectsBadLevel(t *testing.T) {
	_, err := app.Config{LogLevel: "loud"}.JSONLogger()
	require.Error(t, err)
}

func TestUnknownRoutesSkipAuth(t *testing.T) {
	calls := 0
	laravel := identity.UserClientFunc(func(context.Context, string) (identity.LaravelUser, error) {
		calls++
		return identity.LaravelUser{}, identity.ErrTokenRejected
	})
	h := app.NewRouter(app.Deps{Logger: slog.New(slog.DiscardHandler), LaravelUsers: laravel})

	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/v1/typo", http.StatusNotFound},
		{http.MethodPut, "/v1/me", http.StatusMethodNotAllowed},
		{http.MethodGet, "/v1/me", http.StatusUnauthorized}, // real route does authenticate
	} {
		req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, http.NoBody)
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, tc.want, rec.Code, tc.method+" "+tc.path)
	}
	require.Equal(t, 1, calls, "only the real route called Laravel")
}

func TestNonJSONBodyRejected(t *testing.T) {
	h := app.NewRouter(app.Deps{Logger: slog.New(slog.DiscardHandler), LaravelUsers: identity.UserClientFunc(
		func(context.Context, string) (identity.LaravelUser, error) { return identity.LaravelUser{}, nil })})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/me", strings.NewReader("x=1"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}
