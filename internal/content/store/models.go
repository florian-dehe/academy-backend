// Package store holds the content module's GORM models. The schema is created
// by internal/schema (AutoMigrate); keep constraints in the struct tags.
package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Chapter struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Position  int32     `gorm:"not null"`
	IsFree    bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	DeletedAt *time.Time
}

func (Chapter) TableName() string { return "chapters" }

type ChapterVersion struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ChapterID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_chapter_versions_chapter_version,priority:1;uniqueIndex:chapter_versions_one_published,where:status = 'published'"`
	Version       int32     `gorm:"not null;uniqueIndex:uq_chapter_versions_chapter_version,priority:2"`
	Status        string    `gorm:"not null;default:draft;check:chk_chapter_versions_status,status IN ('draft', 'published', 'archived')"`
	Title         string    `gorm:"not null"`
	Description   *string
	MilestoneText *string
	CreatedBy     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt     time.Time  `gorm:"not null;default:now()"`
	PublishedAt   *time.Time
}

func (ChapterVersion) TableName() string { return "chapter_versions" }

type Module struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ChapterVersionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_modules_version_key,priority:1"`
	ModuleKey        uuid.UUID `gorm:"type:uuid;not null;default:gen_random_uuid();uniqueIndex:uq_modules_version_key,priority:2"` // stable across versions
	Position         int32     `gorm:"not null"`
	Title            string    `gorm:"not null"`
	XPReward         int32     `gorm:"column:xp_reward;not null;default:0;check:chk_modules_xp_reward,xp_reward >= 0"`
}

func (Module) TableName() string { return "modules" }

type ModuleVideo struct {
	ModuleID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	VideoURL        string    `gorm:"column:video_url;not null"`
	DurationSeconds *int32
	Transcript      *string
}

func (ModuleVideo) TableName() string { return "module_videos" }

type ModuleExercise struct {
	ModuleID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	ExerciseType string    `gorm:"not null"`
	Title        *string
	Instructions *string
	Config       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
}

func (ModuleExercise) TableName() string { return "module_exercises" }

// Quiz is a recap quiz (module_id set) or a final quiz (chapter_version_id set).
type Quiz struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Kind             string          `gorm:"not null;check:chk_quizzes_kind,kind IN ('recap', 'final')"`
	ModuleID         *uuid.UUID      `gorm:"type:uuid;uniqueIndex"`
	ChapterVersionID *uuid.UUID      `gorm:"type:uuid;uniqueIndex;check:chk_quizzes_one_target,(kind = 'recap' AND module_id IS NOT NULL AND chapter_version_id IS NULL) OR (kind = 'final' AND chapter_version_id IS NOT NULL AND module_id IS NULL)"`
	PassScore        int32           `gorm:"not null;check:chk_quizzes_pass_score,pass_score BETWEEN 0 AND 100"`
	Questions        json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
}

func (Quiz) TableName() string { return "quizzes" }

// Models lists every model of this module for AutoMigrate.
func Models() []any {
	return []any{&Chapter{}, &ChapterVersion{}, &Module{}, &ModuleVideo{}, &ModuleExercise{}, &Quiz{}}
}
