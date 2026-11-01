package identity

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers account routes.
func Mount(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(RequireAuth)
		r.Get("/me", httpx.Handle(httpx.NotImplemented))
		r.Patch("/me", httpx.Handle(httpx.NotImplemented))
		r.Delete("/me", httpx.Handle(httpx.NotImplemented))
	})
}
