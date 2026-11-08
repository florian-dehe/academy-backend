package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/remojii/academy-api/internal/db"
	"github.com/remojii/academy-api/internal/httpx"
	"github.com/remojii/academy-api/internal/identity/store"
)

// User is the Academy user, keyed to the Laravel user by LaravelID.
type User struct {
	ID          uuid.UUID `json:"id"`
	Role        Role      `json:"role"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	LaravelID   string    `json:"-"`
}

// ErrUserNotFound: no Academy user for that Laravel user.
var ErrUserNotFound = errors.New("identity: user not found")

// UserStore persists Academy users.
type UserStore interface {
	// ByLaravelID returns ErrUserNotFound when there is no user.
	ByLaravelID(ctx context.Context, laravelID string) (User, error)
	// Create inserts a learner; created is false (and the existing
	// user returned) when one already exists. Safe under concurrency.
	Create(ctx context.Context, u LaravelUser) (acc User, created bool, err error)
	// Delete removes the user (and, via foreign keys, their Academy data).
	Delete(ctx context.Context, id uuid.UUID) error
}

// PgUserStore is the sqlc-backed UserStore.
type PgUserStore struct{ q *store.Queries }

func NewPgUserStore(d db.DBTX) *PgUserStore { return &PgUserStore{q: store.New(d)} }

func toUser(u store.User) User {
	a := User{ID: u.ID, Role: Role(u.Role), Email: u.Email, LaravelID: u.JobBoardUserID.String}
	if u.DisplayName.Valid {
		a.DisplayName = &u.DisplayName.String
	}
	return a
}

func (s *PgUserStore) ByLaravelID(ctx context.Context, id string) (User, error) {
	u, err := s.q.GetUserByJobBoardID(ctx, pgtype.Text{String: id, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return toUser(u), nil
}

func (s *PgUserStore) Create(ctx context.Context, lu LaravelUser) (User, bool, error) {
	u, err := s.q.InsertUserIfAbsent(ctx, store.InsertUserIfAbsentParams{
		Email:          lu.Email,
		JobBoardUserID: pgtype.Text{String: lu.ID, Valid: true},
		DisplayName:    pgtype.Text{String: lu.Name, Valid: lu.Name != ""},
	})
	if err == nil {
		return toUser(u), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) { // already existed
		acc, err := s.ByLaravelID(ctx, lu.ID)
		return acc, false, err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // email taken by another Laravel id
		return User{}, false, httpx.ErrConflict.WithMessage("email already used by another user")
	}
	return User{}, false, err
}

func (s *PgUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteUser(ctx, id)
}
