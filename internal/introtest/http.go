// Package introtest owns the 30-question personality test and personality_results.
package introtest

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers intro test routes (endpoints.md §2).
func Mount(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireAuth)
		r.Get("/intro-test/questions", httpx.Handle(httpx.NotImplemented))
		r.Post("/intro-test/result", httpx.Handle(httpx.NotImplemented))
		r.Get("/me/personality-result", httpx.Handle(httpx.NotImplemented))
	})
}
