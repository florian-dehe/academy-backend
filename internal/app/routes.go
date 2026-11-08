package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/httpx"
	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/introtest"
)

// routes registers every /v1 route in one place; handlers and middleware live
// in their module packages. The /me routes are real; /users is still a 501
// stub behind auth.
func routes(r chi.Router, d Deps) {
	ni := httpx.Handle(httpx.NotImplemented)

	// Authenticate is applied per group, not with r.Use, so unknown paths and
	// wrong methods still get 404/405 without a Laravel round trip.
	r.Group(func(r chi.Router) {
		r.Use(identity.Authenticate(d.LaravelUsers))

		user := identity.RequireUser(d.Users)
		admin := identity.RequireRole(identity.RoleAdmin)

		// Own profile (§1). POST creates the Academy user, so it skips RequireUser.
		// Routes are flat (no r.Route/Mount): a mounted sub-router would sit
		// behind Authenticate and turn 404/405 into 401.
		r.Post("/me", identity.CreateUser(d.Users))
		r.With(user).Get("/me", identity.GetMe())
		r.With(user).Delete("/me", identity.DeleteMe(d.Users))

		// Intro test (§2). The frontend asks and scores the questions; the API
		// only stores the five axis scores and serves the latest.
		r.With(user).Post("/intro-test/result", introtest.Submit(d.Results))
		r.With(user).Get("/me/personality-result", introtest.GetLatest(d.Results))

		// User administration (§7). Admin only.
		r.With(user, admin).Get("/users", ni)
		r.With(user, admin).Patch("/users/{userId}/role", ni)
	})
}
