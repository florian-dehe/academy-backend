// Package introtest stores the learner's intro test (personality) result.
// The frontend asks the 30 questions and scores them; this module only
// validates and stores the five axis scores, and serves the latest one.
package introtest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/remojii/academy-api/internal/db"
	"github.com/remojii/academy-api/internal/httpx"
	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/introtest/store"
)

const (
	minScore = 1.0
	maxScore = 5.0
)

// Scores is one score per Big Five axis, each within 1.0..5.0.
type Scores struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
}

// Result is a stored intro test result. Retakes add rows; the newest is current.
type Result struct {
	ID uuid.UUID `json:"id"`
	Scores
	CreatedAt time.Time `json:"created_at"`
}

// ErrNoResult: the user has not completed the intro test.
var ErrNoResult = errors.New("introtest: no result")

// ResultStore persists personality results.
type ResultStore interface {
	Create(ctx context.Context, userID uuid.UUID, s Scores) (Result, error)
	// Latest returns ErrNoResult when the user has none.
	Latest(ctx context.Context, userID uuid.UUID) (Result, error)
}

// PgResultStore is the sqlc-backed ResultStore.
type PgResultStore struct{ q *store.Queries }

func NewPgResultStore(d db.DBTX) *PgResultStore { return &PgResultStore{q: store.New(d)} }

func toResult(r store.PersonalityResult) Result {
	return Result{
		ID: r.ID,
		Scores: Scores{
			Openness:          r.Openness,
			Conscientiousness: r.Conscientiousness,
			Extraversion:      r.Extraversion,
			Agreeableness:     r.Agreeableness,
			Neuroticism:       r.Neuroticism,
		},
		CreatedAt: r.CreatedAt,
	}
}

func (s *PgResultStore) Create(ctx context.Context, userID uuid.UUID, sc Scores) (Result, error) {
	r, err := s.q.InsertPersonalityResult(ctx, store.InsertPersonalityResultParams{
		UserID:            userID,
		Openness:          sc.Openness,
		Conscientiousness: sc.Conscientiousness,
		Extraversion:      sc.Extraversion,
		Agreeableness:     sc.Agreeableness,
		Neuroticism:       sc.Neuroticism,
	})
	if err != nil {
		return Result{}, err
	}
	return toResult(r), nil
}

func (s *PgResultStore) Latest(ctx context.Context, userID uuid.UUID) (Result, error) {
	r, err := s.q.GetLatestPersonalityResult(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrNoResult
	}
	if err != nil {
		return Result{}, err
	}
	return toResult(r), nil
}

// submitRequest is the POST /v1/intro-test/result body. Pointers tell a
// missing or null score from a valid one.
type submitRequest struct {
	Openness          *float64 `json:"openness"`
	Conscientiousness *float64 `json:"conscientiousness"`
	Extraversion      *float64 `json:"extraversion"`
	Agreeableness     *float64 `json:"agreeableness"`
	Neuroticism       *float64 `json:"neuroticism"`
}

// Bind implements render.Binder; render.Bind calls it after decoding.
func (b *submitRequest) Bind(*http.Request) error {
	for name, v := range map[string]*float64{
		"openness":          b.Openness,
		"conscientiousness": b.Conscientiousness,
		"extraversion":      b.Extraversion,
		"agreeableness":     b.Agreeableness,
		"neuroticism":       b.Neuroticism,
	} {
		if v == nil {
			return httpx.ErrBadRequest.WithMessage(name + " is required")
		}
		if *v < minScore || *v > maxScore { // NaN cannot come from JSON
			return httpx.ErrBadRequest.WithMessage(name + " must be between 1.0 and 5.0")
		}
	}
	return nil
}

func (b *submitRequest) scores() Scores {
	return Scores{
		Openness:          *b.Openness,
		Conscientiousness: *b.Conscientiousness,
		Extraversion:      *b.Extraversion,
		Agreeableness:     *b.Agreeableness,
		Neuroticism:       *b.Neuroticism,
	}
}

// Submit handles POST /v1/intro-test/result (behind RequireUser): stores the
// caller's scores as a new result.
func Submit(s ResultStore) http.HandlerFunc {
	return httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		u, ok := identity.UserFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		var body submitRequest
		if err := render.Bind(r, &body); err != nil {
			return err
		}
		res, err := s.Create(r.Context(), u.ID, body.scores())
		if err != nil {
			return err
		}
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, res)
		return nil
	})
}

// GetLatest handles GET /v1/me/personality-result (behind RequireUser).
func GetLatest(s ResultStore) http.HandlerFunc {
	return httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		u, ok := identity.UserFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		res, err := s.Latest(r.Context(), u.ID)
		if errors.Is(err, ErrNoResult) {
			return httpx.ErrNotFound.WithMessage("no personality result yet")
		}
		if err != nil {
			return err
		}
		render.JSON(w, r, res)
		return nil
	})
}
