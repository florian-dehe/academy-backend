// Package gamification owns levels, XP and streaks. It has no dependency on
// learning: it listens to events (ModuleCompleted) on the bus.
package gamification

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers XP/level/streak routes (endpoints.md §4).
func Mount(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireAuth)
		r.Get("/me/stats", httpx.Handle(httpx.NotImplemented))
		r.Post("/me/streak/check-in", httpx.Handle(httpx.NotImplemented))
		r.Get("/levels", httpx.Handle(httpx.NotImplemented))
	})
}
