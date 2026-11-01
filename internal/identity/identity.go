// Package identity owns users, roles and request authentication.
//
// Authentication belongs to Job Board (Laravel Sanctum): a TokenResolver maps a
// bearer access token to a User (see JobBoardResolver). Academy has no
// register/login endpoint; the local user is created on first request.
package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/remojii/academy-api/internal/platform/httpx"
)

type Role string

const (
	RoleLearner       Role = "learner"
	RoleContentEditor Role = "content_editor"
	RoleAdmin         Role = "admin"
)

func (r Role) rank() int {
	switch r {
	case RoleLearner:
		return 1
	case RoleContentEditor:
		return 2
	case RoleAdmin:
		return 3
	}
	return 0
}

// AtLeast reports whether r is minRole or higher in admin ⊃ content_editor ⊃ learner.
func (r Role) AtLeast(minRole Role) bool { return r.rank() >= minRole.rank() && r.rank() > 0 }

type User struct {
	ID             uuid.UUID
	Email          string
	JobBoardUserID *string
	DisplayName    *string
	Role           Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

var ErrInvalidToken = errors.New("identity: invalid token")

// TokenResolver maps a bearer token to a User. Return ErrInvalidToken when unknown.
type TokenResolver interface {
	Resolve(ctx context.Context, token string) (User, error)
}

type ctxKey struct{}

func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// FromContext returns the authenticated user, if any.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}

// MustUser returns the authenticated user or ErrUnauthorized. Ownership (👤):
// handlers take the user id from here, never from path or body.
func MustUser(ctx context.Context) (User, error) {
	if u, ok := FromContext(ctx); ok {
		return u, nil
	}
	return User{}, httpx.ErrUnauthorized
}

// Authenticate resolves "Authorization: Bearer <token>" into the context.
// No header: request continues anonymous. Bad token: 401.
func Authenticate(res TokenResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := strings.CutPrefix(h, "Bearer ")
			if !ok || token == "" {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			u, err := res.Resolve(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					httpx.WriteError(w, r, httpx.ErrUnauthorized)
				} else {
					httpx.WriteError(w, r, err)
				}
				return
			}
			httpx.SetActor(r.Context(), u.ID.String(), string(u.Role))
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
		})
	}
}

// RequireAuth rejects anonymous requests with 401.
func RequireAuth(next http.Handler) http.Handler { return RequireRole(RoleLearner)(next) }

// RequireRole rejects anonymous (401) and users below minRole (403).
func RequireRole(minRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			if !u.Role.AtLeast(minRole) {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
