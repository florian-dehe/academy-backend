package app_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/app"
	"github.com/remojii/academy-api/internal/identity"
)

func newServer(t *testing.T) http.Handler {
	t.Helper()
	mk := func(role identity.Role) identity.User {
		return identity.User{ID: uuid.New(), Email: string(role) + "@example.com", Role: role}
	}
	return app.NewRouter(app.Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Resolver: identity.StaticResolver{
			"learner": mk(identity.RoleLearner),
			"editor":  mk(identity.RoleContentEditor),
			"admin":   mk(identity.RoleAdmin),
		},
	})
}

// Permission matrix (P0 seed; extend toward the full endpoints.md matrix in P8).
// Routes are 501 stubs, so "allowed" == 501 and "denied" == 401/403.
func TestPermissions(t *testing.T) {
	const ok = http.StatusNotImplemented
	cases := []struct {
		method, path, token string
		want                int
	}{
		{"GET", "/v1/me", "", 401},
		{"GET", "/v1/me", "bogus", 401},
		{"GET", "/v1/me", "learner", ok},
		{"POST", "/v1/register", "", 404},
		{"GET", "/v1/chapters/00000000-0000-0000-0000-000000000001", "learner", ok},
		{"GET", "/v1/chapters/not-a-uuid", "learner", 404},
		{"GET", "/v1/nope", "", 404},
		{"GET", "/v1/plans", "", ok},
		{"POST", "/v1/chapters", "", 401},
		{"POST", "/v1/chapters", "learner", 403},
		{"POST", "/v1/chapters", "editor", ok},
		{"GET", "/v1/chapters", "learner", ok},
		{"DELETE", "/v1/chapters/x", "editor", 403},
		{"DELETE", "/v1/chapters/x", "admin", ok},
		{"POST", "/v1/chapter-versions/x/publish", "editor", 403},
		{"POST", "/v1/chapter-versions/x/publish", "admin", ok},
		{"PUT", "/v1/levels", "editor", 403},
		{"PUT", "/v1/levels", "admin", ok},
		{"GET", "/v1/users", "editor", 403},
		{"GET", "/v1/users", "admin", ok},
		{"POST", "/v1/webhooks/payments", "", ok},
	}
	srv := newServer(t)
	for _, c := range cases {
		t.Run(c.method+" "+c.path+" as "+c.token, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), c.method, c.path, nil)
			if c.token != "" {
				req.Header.Set("Authorization", "Bearer "+c.token)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			require.Equal(t, c.want, rec.Code, rec.Body.String())
		})
	}
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	require.Equal(t, 200, rec.Code)
}
