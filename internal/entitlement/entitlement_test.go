package entitlement_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/entitlement"
	"github.com/remojii/academy-api/internal/identity"
)

type denyAll struct{}

func (denyAll) CanAccessChapter(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func serve(t *testing.T, svc entitlement.Service, path string) int {
	t.Helper()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(identity.WithUser(req.Context(), identity.User{ID: uuid.New(), Role: identity.RoleLearner})))
		})
	})
	r.With(entitlement.RequireChapter(svc, entitlement.ChapterIDFromPath)).
		Get("/chapters/{chapterId}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec.Code
}

func TestRequireChapter(t *testing.T) {
	id := "/chapters/" + uuid.NewString()
	require.Equal(t, http.StatusOK, serve(t, entitlement.AllowAll{}, id))
	require.Equal(t, http.StatusPaymentRequired, serve(t, denyAll{}, id))
	require.Equal(t, http.StatusNotFound, serve(t, entitlement.AllowAll{}, "/chapters/x"))
}
