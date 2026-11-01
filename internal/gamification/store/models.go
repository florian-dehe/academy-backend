// Package store holds the gamification module's GORM models (see internal/schema).
package store

import (
	"time"

	"github.com/google/uuid"
)

// Level 1 must exist so user_stats can always reference a level (seeded by internal/schema).
type Level struct {
	Level      int32 `gorm:"primaryKey;autoIncrement:false"`
	XPRequired int32 `gorm:"column:xp_required;not null;uniqueIndex;check:chk_levels_xp_required,xp_required >= 0"`
	Title      *string
}

func (Level) TableName() string { return "levels" }

type UserStats struct {
	UserID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TotalXP          int32      `gorm:"column:total_xp;not null;default:0;check:chk_user_stats_total_xp,total_xp >= 0"`
	Level            int32      `gorm:"not null;default:1"`
	CurrentStreak    int32      `gorm:"not null;default:0"`
	LongestStreak    int32      `gorm:"not null;default:0"`
	LastActivityDate *time.Time `gorm:"type:date"`
}

func (UserStats) TableName() string { return "user_stats" }

// Models lists every model of this module for AutoMigrate.
func Models() []any { return []any{&Level{}, &UserStats{}} }
