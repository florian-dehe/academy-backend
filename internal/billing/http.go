// Package billing owns plans, subscriptions, payments and the payment provider adapter.
package billing

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers subscription routes (endpoints.md §5).
func Mount(r chi.Router) {
	r.Get("/plans", httpx.Handle(httpx.NotImplemented)) // public

	// Provider webhook: signature-verified, not user-authenticated.
	r.Post("/webhooks/payments", httpx.Handle(httpx.NotImplemented))

	r.Group(func(r chi.Router) {
		r.Use(identity.RequireAuth)
		r.Get("/me/subscription", httpx.Handle(httpx.NotImplemented))
		r.Post("/me/subscription", httpx.Handle(httpx.NotImplemented))
		r.Post("/me/subscription/cancel", httpx.Handle(httpx.NotImplemented))
		r.Get("/me/payments", httpx.Handle(httpx.NotImplemented))
	})
}
