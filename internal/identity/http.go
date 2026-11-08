package identity

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/remojii/academy-api/internal/httpx"
)

// createUserRequest is the POST /v1/me body. The id is only a consistency
// check against the token's Laravel user, never trusted.
type createUserRequest struct {
	JobBoardUserID string `json:"user_id"`
}

// Bind implements render.Binder; render.Bind calls it after decoding.
func (b *createUserRequest) Bind(*http.Request) error {
	if b.JobBoardUserID == "" {
		return httpx.ErrBadRequest.WithMessage("user_id is required")
	}
	return nil
}

// CreateUser handles POST /v1/me (behind Authenticate only): 201 with the new
// user, or 200 with the existing one.
func CreateUser(s UserStore) http.HandlerFunc {
	return httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		lu, ok := LaravelUserFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		var body createUserRequest
		if err := render.Bind(r, &body); err != nil {
			return err
		}
		if body.JobBoardUserID != lu.ID {
			return httpx.ErrForbidden.WithMessage("user_id does not match the token")
		}
		u, created, err := s.Create(r.Context(), lu)
		if err != nil {
			return err
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		render.Status(r, status)
		render.JSON(w, r, u)
		return nil
	})
}

// GetMe handles GET /v1/me (behind RequireUser).
func GetMe() http.HandlerFunc {
	return httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		u, ok := UserFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		render.JSON(w, r, u)
		return nil
	})
}

// DeleteMe handles DELETE /v1/me (behind RequireUser): removes the Academy
// user and their data. The Laravel account is untouched, so the same person
// can create a fresh Academy user with POST /v1/me.
func DeleteMe(s UserStore) http.HandlerFunc {
	return httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		u, ok := UserFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		if err := s.Delete(r.Context(), u.ID); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}
