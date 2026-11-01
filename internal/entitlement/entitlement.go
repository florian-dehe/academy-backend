// Package entitlement answers "may this user access this chapter?".
// Derived from billing state; owns no tables.
package entitlement

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Service is the only thing other modules (learning) depend on.
type Service interface {
	// CanAccessChapter: chapter 1 (is_free) always; chapters >= 2 need an
	// active, or past_due-within-grace, subscription.
	CanAccessChapter(ctx context.Context, userID, chapterID uuid.UUID) (bool, error)
}

// AllowAll is the P0 stub, replaced in P6 by a billing-backed implementation.
type AllowAll struct{}

func (AllowAll) CanAccessChapter(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

// RequireChapter is the 🔒 hook: it answers 402 subscription_required when the
// user may not access the chapter that chapterID extracts from the request.
// Place it after identity.RequireAuth.
func RequireChapter(svc Service, chapterID func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := identity.MustUser(r.Context())
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			id, err := chapterID(r)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			ok, err := svc.CanAccessChapter(r.Context(), u.ID, id)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			if !ok {
				httpx.WriteError(w, r, httpx.ErrSubscriptionRequired)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ChapterIDFromPath reads the {chapterId} URL parameter. A malformed id is a 404.
func ChapterIDFromPath(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "chapterId"))
	if err != nil {
		return uuid.Nil, httpx.ErrNotFound
	}
	return id, nil
}
