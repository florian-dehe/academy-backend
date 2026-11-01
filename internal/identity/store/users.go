// Package store is the identity module's persistence layer (GORM).
// The schema is created by internal/schema (GORM AutoMigrate); constraints live in the struct tags.
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/remojii/academy-api/internal/platform/db"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email          string    `gorm:"not null;uniqueIndex"`
	JobBoardUserID *string   `gorm:"uniqueIndex"`
	DisplayName    *string
	Role           string    `gorm:"not null;default:learner;check:chk_users_role,role IN ('learner', 'content_editor', 'admin')"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (User) TableName() string { return "users" }

// Store runs identity queries on the pool, or on the ctx transaction if any.
type Store struct {
	gdb *gorm.DB
}

// Models lists every model of this module for AutoMigrate.
func Models() []any { return []any{&User{}} }

func New(gdb *gorm.DB) *Store { return &Store{gdb: gdb} }

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := db.Conn(ctx, s.gdb).Take(&u, "id = ?", id).Error
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := db.Conn(ctx, s.gdb).Take(&u, "email = ?", email).Error
	return u, err
}

type CreateUserParams struct {
	Email          string
	JobBoardUserID *string
	DisplayName    *string
}

func (s *Store) CreateUser(ctx context.Context, p CreateUserParams) (User, error) {
	u := User{Email: p.Email, JobBoardUserID: p.JobBoardUserID, DisplayName: p.DisplayName}
	// Empty Role falls back to the column default ('learner'), read back via RETURNING.
	err := db.Conn(ctx, s.gdb).Create(&u).Error
	return u, err
}

func (s *Store) UpdateUserDisplayName(ctx context.Context, id uuid.UUID, name *string) (User, error) {
	var u User
	res := db.Conn(ctx, s.gdb).Model(&u).
		Where("id = ?", id).
		Updates(map[string]any{"display_name": name, "updated_at": time.Now()})
	if res.Error != nil {
		return u, res.Error
	}
	if res.RowsAffected == 0 {
		return u, gorm.ErrRecordNotFound
	}
	err := db.Conn(ctx, s.gdb).Take(&u, "id = ?", id).Error
	return u, err
}

func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return db.Conn(ctx, s.gdb).Delete(&User{}, "id = ?", id).Error
}

// UpsertByJobBoardID returns the user linked to jobBoardID, creating it (role
// learner) on first sight. Safe under concurrent first requests. An email that
// already belongs to a different row surfaces the unique-violation error.
func (s *Store) UpsertByJobBoardID(ctx context.Context, p CreateUserParams) (User, error) {
	conn := db.Conn(ctx, s.gdb)
	u := User{Email: p.Email, JobBoardUserID: p.JobBoardUserID, DisplayName: p.DisplayName}
	err := conn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "job_board_user_id"}},
		DoNothing: true,
	}).Create(&u).Error
	if err != nil {
		return User{}, err
	}
	var out User
	err = conn.Take(&out, "job_board_user_id = ?", *p.JobBoardUserID).Error
	return out, err
}
