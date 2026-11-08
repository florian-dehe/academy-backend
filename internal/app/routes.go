package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/remojii/academy-api/internal/httpx"
)

// routes registers every /v1 route (see endpoints.md). All handlers are
// 501 stubs for now; the role guards are real.
func routes(r chi.Router) {
	ni := httpx.Handle(httpx.NotImplemented)

	// Account (§1)
	r.Group(func(r chi.Router) {
		r.Get("/me", ni)
		r.Patch("/me", ni)
		r.Delete("/me", ni)
	})

	// Intro test (§2)
	r.Group(func(r chi.Router) {
		r.Get("/intro-test/questions", ni)
		r.Post("/intro-test/result", ni)
		r.Get("/me/personality-result", ni)
	})

	// Learner training path (§3).
	// The 🔒 hook guards chapter reads now (stub allows all until P6).
	// TODO(P3): module and quiz routes must resolve their chapter and apply
	// identity.RequireChapter too (module reads, video/quiz/exercise writes).
	r.Group(func(r chi.Router) {
		r.Get("/chapters", ni)
		r.Get("/chapters/{chapterId}", ni)
		r.Get("/modules/{moduleId}", ni)
		r.Post("/modules/{moduleId}/video/complete", ni)
		r.Post("/quizzes/{quizId}/attempts", ni)
		r.Post("/quiz-attempts/{attemptId}/submit", ni)
		r.Post("/modules/{moduleId}/exercise-submissions", ni)
		r.Get("/me/progress", ni)
	})

	// XP, levels, streaks (§4)
	r.Group(func(r chi.Router) {
		r.Get("/me/stats", ni)
		r.Post("/me/streak/check-in", ni)
		r.Get("/levels", ni)
	})

	// Content editing (§6).
	// Editors and admins may use all; admin-only routes add adminOnly.
	// Module endpoints must return 409 version_not_draft for non-draft versions.
	r.Group(func(r chi.Router) {
		// Chapters and versions
		r.Post("/chapters", ni)
		r.Patch("/chapters/{chapterId}", ni)
		r.Put("/chapters/order", ni)
		r.Delete("/chapters/{chapterId}", ni)
		r.Get("/chapters/{chapterId}/versions", ni)
		r.Post("/chapters/{chapterId}/versions", ni)
		r.Patch("/chapter-versions/{versionId}", ni)
		r.Post("/chapter-versions/{versionId}/publish", ni)
		r.Post("/chapter-versions/{versionId}/rollback", ni)
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

	// Platform admin (§7). Admin only.
	r.Group(func(r chi.Router) {
		r.Get("/users", ni)
		r.Patch("/users/{userId}/role", ni)
		r.Put("/levels", ni)
	})
}
