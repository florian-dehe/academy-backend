// Package content owns chapters, versions, modules and quizzes: draft editing,
// publish/rollback, and reads of published content.
package content

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers the content editing routes (endpoints.md §6).
// Editors and admins may use all; admin-only routes add RequireRole(admin).
// Module endpoints must return 409 version_not_draft for non-draft versions.
func Mount(r chi.Router) {
	adminOnly := identity.RequireRole(identity.RoleAdmin)
	ni := httpx.Handle(httpx.NotImplemented)

	r.Group(func(r chi.Router) {
		r.Use(identity.RequireRole(identity.RoleContentEditor))

		// Chapters and versions
		r.Post("/chapters", ni)
		r.Patch("/chapters/{chapterId}", ni)
		r.Put("/chapters/order", ni)
		r.With(adminOnly).Delete("/chapters/{chapterId}", ni)
		r.Get("/chapters/{chapterId}/versions", ni)
		r.Post("/chapters/{chapterId}/versions", ni)
		r.Patch("/chapter-versions/{versionId}", ni)
		r.With(adminOnly).Post("/chapter-versions/{versionId}/publish", ni)
		r.With(adminOnly).Post("/chapter-versions/{versionId}/rollback", ni)
		r.Delete("/chapter-versions/{versionId}", ni)

		// Modules (inside a draft version)
		r.Post("/chapter-versions/{versionId}/modules", ni)
		r.Patch("/modules/{moduleId}", ni)
		r.Put("/modules/{moduleId}/video", ni)
		r.Put("/modules/{moduleId}/quiz", ni)
		r.Put("/modules/{moduleId}/exercise", ni)
		r.Put("/chapter-versions/{versionId}/final-quiz", ni)
		r.Put("/chapter-versions/{versionId}/modules/order", ni)
		r.Delete("/modules/{moduleId}", ni)
	})
}
