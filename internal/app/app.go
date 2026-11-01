// Package app wires modules into one HTTP router.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/remojii/academy-api/api"
	"github.com/remojii/academy-api/internal/billing"
	"github.com/remojii/academy-api/internal/content"
	"github.com/remojii/academy-api/internal/entitlement"
	"github.com/remojii/academy-api/internal/gamification"
	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/introtest"
	"github.com/remojii/academy-api/internal/learning"
	"github.com/remojii/academy-api/internal/platform/httpx"
	"github.com/remojii/academy-api/internal/platformadmin"
)

type Deps struct {
	Logger   *slog.Logger
	Resolver identity.TokenResolver
	// Entitlement guards chapters >= 2. Defaults to entitlement.AllowAll until P6.
	Entitlement entitlement.Service
	// Ready reports dependency health for /readyz (e.g. DB ping). Optional.
	Ready func(ctx context.Context) error
}

func NewRouter(d Deps) http.Handler {
	if d.Entitlement == nil {
		d.Entitlement = entitlement.AllowAll{}
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, httpx.RequestLogger(d.Logger), httpx.Recoverer)
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if d.Ready != nil {
			ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()
			if err := d.Ready(ctx); err != nil {
				httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
				return
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Get("/openapi.yaml", httpx.SpecHandler(api.Spec))
	r.Get("/docs", httpx.SwaggerUI)

	r.Route("/v1", func(r chi.Router) {
		r.Use(identity.Authenticate(d.Resolver))
		identity.Mount(r)
		introtest.Mount(r)
		learning.Mount(r, d.Entitlement)
		gamification.Mount(r)
		billing.Mount(r)
		content.Mount(r)
		platformadmin.Mount(r)
	})
	return r
}
