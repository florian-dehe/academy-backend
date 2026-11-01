// Package learning owns learner progress: step state machine, quiz scoring,
// module and chapter completion. Depends on content and entitlement only.
package learning

import (
	"github.com/go-chi/chi/v5"

	"github.com/remojii/academy-api/internal/entitlement"
	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/platform/httpx"
)

// Mount registers learner training path routes (endpoints.md §3).
//
// The 🔒 hook guards chapter reads now (stub allows all until P6).
// TODO(P3): module and quiz routes must resolve their chapter and apply
// entitlement.RequireChapter too (module reads, video/quiz/exercise writes).
func Mount(r chi.Router, ent entitlement.Service) {
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireAuth)
		r.Get("/chapters", httpx.Handle(httpx.NotImplemented))
		r.With(entitlement.RequireChapter(ent, entitlement.ChapterIDFromPath)).
			Get("/chapters/{chapterId}", httpx.Handle(httpx.NotImplemented))
		r.Get("/modules/{moduleId}", httpx.Handle(httpx.NotImplemented))
		r.Post("/modules/{moduleId}/video/complete", httpx.Handle(httpx.NotImplemented))
		r.Post("/quizzes/{quizId}/attempts", httpx.Handle(httpx.NotImplemented))
		r.Post("/quiz-attempts/{attemptId}/submit", httpx.Handle(httpx.NotImplemented))
		r.Post("/modules/{moduleId}/exercise-submissions", httpx.Handle(httpx.NotImplemented))
		r.Get("/me/progress", httpx.Handle(httpx.NotImplemented))
	})
}
