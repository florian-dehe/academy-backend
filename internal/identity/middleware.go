package identity

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/remojii/academy-api/internal/httpx"
)

type ctxKey int

const (
	laravelUserKey ctxKey = iota
	userKey
)

// LaravelUserFrom returns the Laravel identity set by Authenticate.
func LaravelUserFrom(ctx context.Context) (LaravelUser, bool) {
	u, ok := ctx.Value(laravelUserKey).(LaravelUser)
	return u, ok
}

// UserFrom returns the Academy user set by RequireUser.
func UserFrom(ctx context.Context) (User, bool) {
	a, ok := ctx.Value(userKey).(User)
	return a, ok
}

// WithUser returns ctx carrying u, as RequireUser does. Other modules' tests
// use it to skip the Laravel round trip.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func bearerToken(r *http.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// Authenticate validates the bearer token against Laravel and stores the
// Laravel identity in the context. It does not require an Academy user.
func Authenticate(c UserClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			u, err := c.CurrentUser(r.Context(), token)
			switch {
			case err == nil:
			case errors.Is(err, ErrTokenRejected):
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			default:
				slog.ErrorContext(r.Context(), "laravel auth failed", "err", err)
				httpx.WriteError(w, r, httpx.ErrUnavailable)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), laravelUserKey, u)))
		})
	}
}

// RequireUser (after Authenticate) loads the Academy user, puts it in the
// context and records the actor for the request log.
func RequireUser(s UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lu, ok := LaravelUserFrom(r.Context())
			if !ok {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			acc, err := s.ByLaravelID(r.Context(), lu.ID)
			if errors.Is(err, ErrUserNotFound) {
				httpx.WriteError(w, r, httpx.ErrUserRequired)
				return
			}
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			httpx.SetActor(r.Context(), acc.ID.String(), string(acc.Role))
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, acc)))
		})
	}
}

// RequireRole (after RequireUser) rejects users below minRole with 403.
func RequireRole(minRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acc, ok := UserFrom(r.Context())
			if !ok || !acc.Role.AtLeast(minRole) {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
