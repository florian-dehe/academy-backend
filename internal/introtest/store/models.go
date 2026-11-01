// Package store holds the introtest module's GORM models (see internal/schema).
package store

import (
	"time"

	"github.com/google/uuid"
)

// PersonalityResult holds the five axes, each 1.0..5.0.
type PersonalityResult struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;index:personality_results_user_latest,priority:1"`
	Openness          float64   `gorm:"type:numeric(2,1);not null;check:chk_personality_openness,openness BETWEEN 1.0 AND 5.0"`
	Conscientiousness float64   `gorm:"type:numeric(2,1);not null;check:chk_personality_conscientiousness,conscientiousness BETWEEN 1.0 AND 5.0"`
	Extraversion      float64   `gorm:"type:numeric(2,1);not null;check:chk_personality_extraversion,extraversion BETWEEN 1.0 AND 5.0"`
	Agreeableness     float64   `gorm:"type:numeric(2,1);not null;check:chk_personality_agreeableness,agreeableness BETWEEN 1.0 AND 5.0"`
	Neuroticism       float64   `gorm:"type:numeric(2,1);not null;check:chk_personality_neuroticism,neuroticism BETWEEN 1.0 AND 5.0"`
	CreatedAt         time.Time `gorm:"not null;default:now();index:personality_results_user_latest,priority:2,sort:desc"`
}

func (PersonalityResult) TableName() string { return "personality_results" }

// Models lists every model of this module for AutoMigrate.
func Models() []any { return []any{&PersonalityResult{}} }
