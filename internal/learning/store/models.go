// Package store holds the learning module's GORM models (see internal/schema).
package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UserChapterProgress struct {
	UserID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChapterID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChapterVersionID uuid.UUID `gorm:"type:uuid;not null"`
	Status           string    `gorm:"not null;default:locked;check:chk_user_chapter_progress_status,status IN ('locked', 'unlocked', 'in_progress', 'completed')"`
	StartedAt        *time.Time
	CompletedAt      *time.Time
}

func (UserChapterProgress) TableName() string { return "user_chapter_progress" }

type UserModuleProgress struct {
	UserID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	ModuleID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Status              string    `gorm:"not null;default:locked;check:chk_user_module_progress_status,status IN ('locked', 'unlocked', 'in_progress', 'completed')"`
	VideoCompletedAt    *time.Time
	QuizPassedAt        *time.Time
	ExerciseCompletedAt *time.Time
	CompletedAt         *time.Time
}

func (UserModuleProgress) TableName() string { return "user_module_progress" }

type QuizAttempt struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID       `gorm:"type:uuid;not null;index:quiz_attempts_user_quiz,priority:1"`
	QuizID      uuid.UUID       `gorm:"type:uuid;not null;index:quiz_attempts_user_quiz,priority:2"`
	Answers     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"` // {question_id: option_id}
	Score       *int32
	Passed      *bool
	StartedAt   time.Time `gorm:"not null;default:now()"`
	SubmittedAt *time.Time
}

func (QuizAttempt) TableName() string { return "quiz_attempts" }

type ExerciseSubmission struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID       `gorm:"type:uuid;not null;index:exercise_submissions_user_module,priority:1"`
	ModuleID  uuid.UUID       `gorm:"type:uuid;not null;index:exercise_submissions_user_module,priority:2"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	Score     *int32
	Completed bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (ExerciseSubmission) TableName() string { return "exercise_submissions" }

// Models lists every model of this module for AutoMigrate.
func Models() []any {
	return []any{&UserChapterProgress{}, &UserModuleProgress{}, &QuizAttempt{}, &ExerciseSubmission{}}
}
