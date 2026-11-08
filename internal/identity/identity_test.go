package identity_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/identity"
)

// fakeLaravel answers GET /api/v1/user like the Laravel identity.
func fakeLaravel(t *testing.T, handler http.HandlerFunc) identity.UserClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := identity.NewHTTPUserClient(srv.URL, "/api/v1/user", time.Second)
	require.NoError(t, err)
	return c
}

func laravelOK(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer good" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, _ = w.Write([]byte(`{"data":{"id":42,"email":"a@example.com","first_name":"Ada","last_name":"Lovelace"}}`))
}

type memStore struct {
	mu   sync.Mutex
	rows map[string]identity.User
}

func (m *memStore) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, a := range m.rows {
		if a.ID == id {
			delete(m.rows, k)
		}
	}
	return nil
}

func newMemStore() *memStore { return &memStore{rows: map[string]identity.User{}} }

func (m *memStore) ByLaravelID(_ context.Context, id string) (identity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.rows[id]; ok {
		return a, nil
	}
	return identity.User{}, identity.ErrUserNotFound
}

func (m *memStore) Create(_ context.Context, u identity.LaravelUser) (identity.User, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.rows[u.ID]; ok {
		return a, false, nil
	}
	name := u.Name
	a := identity.User{ID: uuid.New(), Role: identity.RoleLearner, Email: u.Email, DisplayName: &name, LaravelID: u.ID}
	m.rows[u.ID] = a
	return a, true, nil
}

func router(c identity.UserClient, s identity.UserStore) http.Handler {
	r := chi.NewRouter()
	r.Use(identity.Authenticate(c))
	r.Post("/me", identity.CreateUser(s))
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireUser(s))
		r.Get("/me", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		r.Get("/profile", identity.GetMe())
		r.Delete("/me", identity.DeleteMe(s))
		r.Group(func(r chi.Router) {
			r.Use(identity.RequireRole(identity.RoleAdmin))
			r.Get("/admin", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		})
	})
	return r
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthenticate(t *testing.T) {
	h := router(fakeLaravel(t, laravelOK), newMemStore())

	require.Equal(t, http.StatusUnauthorized, do(h, "GET", "/me", "", "").Code, "no token")
	require.Equal(t, http.StatusUnauthorized, do(h, "GET", "/me", "bad", "").Code, "rejected token")

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/me", http.NoBody)
	req.Header.Set("Authorization", "Basic good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code, "not a bearer scheme")
}

func TestAuthenticateLaravelDown(t *testing.T) {
	for name, status := range map[string]int{"5xx": 500, "429": 429, "404 (wrong path)": 404} {
		h := router(fakeLaravel(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }), newMemStore())
		rec := do(h, "GET", "/me", "good", "")
		require.Equal(t, http.StatusServiceUnavailable, rec.Code, name)
		require.Contains(t, rec.Body.String(), "upstream_unavailable")
	}

	srv := httptest.NewServer(http.NotFoundHandler())
	c, err := identity.NewHTTPUserClient(srv.URL, "/x", time.Second)
	require.NoError(t, err)
	srv.Close() // connection refused
	require.Equal(t, http.StatusServiceUnavailable, do(router(c, newMemStore()), "GET", "/me", "good", "").Code)
}

func TestLaravelIDParsing(t *testing.T) {
	for body, want := range map[string]int{
		`{"data":{"id":42,"email":"a@b.c"}}`:      http.StatusNoContent,
		`{"data":{"id":"42","email":"a@b.c"}}`:    http.StatusNoContent,
		`{"data":{"id":{"a":1},"email":"a@b.c"}}`: http.StatusServiceUnavailable,
		`{"data":{"id":1.5,"email":"a@b.c"}}`:     http.StatusServiceUnavailable,
		`{"data":{"id":"","email":"a@b.c"}}`:      http.StatusServiceUnavailable,
		`{"data":{"id":null,"email":"a@b.c"}}`:    http.StatusServiceUnavailable,
	} {
		s := newMemStore()
		s.rows["42"] = identity.User{ID: uuid.New(), Role: identity.RoleLearner}
		h := router(fakeLaravel(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }), s)
		require.Equal(t, want, do(h, "GET", "/me", "t", "").Code, body)
	}
}

func TestAuthenticateMalformedBody(t *testing.T) {
	h := router(fakeLaravel(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":{}}`)) }), newMemStore())
	require.Equal(t, http.StatusServiceUnavailable, do(h, "GET", "/me", "good", "").Code)
}

func TestClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	c, err := identity.NewHTTPUserClient(srv.URL, "/x", 50*time.Millisecond)
	require.NoError(t, err)
	_, err = c.CurrentUser(context.Background(), "t")
	require.ErrorIs(t, err, identity.ErrUpstream)
}

func TestRequireUser(t *testing.T) {
	s := newMemStore()
	h := router(fakeLaravel(t, laravelOK), s)

	rec := do(h, "GET", "/me", "good", "")
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "user_required")

	require.Equal(t, http.StatusCreated, do(h, "POST", "/me", "good", `{"user_id":"42"}`).Code)
	require.Equal(t, http.StatusNoContent, do(h, "GET", "/me", "good", "").Code)
	require.Equal(t, http.StatusForbidden, do(h, "GET", "/admin", "good", "").Code, "learner is not admin")
}

func TestCreateUser(t *testing.T) {
	s := newMemStore()
	h := router(fakeLaravel(t, laravelOK), s)

	rec := do(h, "POST", "/me", "good", `{"user_id":"42"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"role":"learner"`)
	require.Contains(t, rec.Body.String(), `"email":"a@example.com"`)
	require.Contains(t, rec.Body.String(), `"display_name":"Ada Lovelace"`)

	rec = do(h, "POST", "/me", "good", `{"user_id":"42"}`)
	require.Equal(t, http.StatusOK, rec.Code, "repeat call is idempotent")

	require.Equal(t, http.StatusForbidden, do(h, "POST", "/me", "good", `{"user_id":"43"}`).Code, "id mismatch")
	require.Equal(t, http.StatusBadRequest, do(h, "POST", "/me", "good", `{}`).Code)
	require.Equal(t, http.StatusBadRequest, do(h, "POST", "/me", "good", `{"user_id":"42","role":"admin"}`).Code, "unknown field")
	require.Equal(t, http.StatusUnauthorized, do(h, "POST", "/me", "bad", `{"user_id":"42"}`).Code)
	require.Len(t, s.rows, 1)
}

func TestGetAndDeleteMe(t *testing.T) {
	s := newMemStore()
	h := router(fakeLaravel(t, laravelOK), s)

	require.Equal(t, http.StatusForbidden, do(h, "GET", "/profile", "good", "").Code, "no user yet")
	require.Equal(t, http.StatusCreated, do(h, "POST", "/me", "good", `{"user_id":"42"}`).Code)

	rec := do(h, "GET", "/profile", "good", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"email":"a@example.com"`)
	require.Equal(t, http.StatusUnauthorized, do(h, "GET", "/profile", "bad", "").Code)

	require.Equal(t, http.StatusNoContent, do(h, "DELETE", "/me", "good", "").Code)
	require.Empty(t, s.rows)
	require.Equal(t, http.StatusForbidden, do(h, "GET", "/profile", "good", "").Code, "deleted")
	require.Equal(t, http.StatusCreated, do(h, "POST", "/me", "good", `{"user_id":"42"}`).Code, "can sign up again")
}
