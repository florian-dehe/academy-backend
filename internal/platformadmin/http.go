// Package platformadmin is a thin admin layer (users, levels, plans) that calls
// other modules. It owns no tables.
package platformadmin

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers platform admin routes (endpoints.md §7). Admin only.
func Mount(r chi.Router) {
	ni := httpx.Handle(httpx.NotImplemented)
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireRole(identity.RoleAdmin))
		r.Get("/users", ni)
		r.Patch("/users/{userId}/role", ni)
		r.Put("/levels", ni)
		r.Patch("/plans/{planId}", ni)
	})
}
