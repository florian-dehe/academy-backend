package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type actor struct{ id, role string }

type actorKey struct{}

// SetActor records who made the request so RequestLogger can log it. Called by
// the auth middleware once the token resolves; no-op outside RequestLogger.
func SetActor(ctx context.Context, userID, role string) {
	if a, ok := ctx.Value(actorKey{}).(*actor); ok {
		a.id, a.role = userID, role
	}
}

// RequestLogger logs one structured line per request, including the
// authenticated user. Admin writes have no audit table, so these logs are the
// only record.
func RequestLogger(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()
			a := &actor{}
			req := r.WithContext(context.WithValue(ctx, actorKey{}, a))
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(ctx),
			}
			if a.id != "" {
				attrs = append(attrs, "user_id", a.id, "role", a.role)
			}
			l.InfoContext(ctx, "request", attrs...)
		})
	}
}
