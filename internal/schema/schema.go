// Package schema owns the database schema. It is the only package that knows
// every module's tables: it runs GORM AutoMigrate over all module models,
// then adds the foreign keys (kept here so modules never import each other)
// and the seed rows the app relies on.
//
// AutoMigrate only adds: it creates tables, missing columns, indexes and
// constraints. It never drops or retypes columns, so removals and renames
// need a manual SQL step.
package schema

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	billing "github.com/remojii/academy-api/internal/billing/store"
	content "github.com/remojii/academy-api/internal/content/store"
	gamification "github.com/remojii/academy-api/internal/gamification/store"
	identity "github.com/remojii/academy-api/internal/identity/store"
	introtest "github.com/remojii/academy-api/internal/introtest/store"
	learning "github.com/remojii/academy-api/internal/learning/store"
)

// lockID is an arbitrary key for the Postgres advisory lock that serialises
// migrations when several replicas start at once.
const lockID = 7_262_001

type fk struct {
	table, column, refTable, refColumn, onDelete string
}

// foreignKeys are created after all tables exist. Empty onDelete = NO ACTION.
// Learner data cascades on user delete (account deletion removes all Academy data).
// Open: this includes payments; revisit if payment records must be retained.
var foreignKeys = []fk{
	{"chapter_versions", "chapter_id", "chapters", "id", ""},
	{"chapter_versions", "created_by", "users", "id", "SET NULL"},
	{"modules", "chapter_version_id", "chapter_versions", "id", "CASCADE"},
	{"module_videos", "module_id", "modules", "id", "CASCADE"},
	{"module_exercises", "module_id", "modules", "id", "CASCADE"},
	{"quizzes", "module_id", "modules", "id", "CASCADE"},
	{"quizzes", "chapter_version_id", "chapter_versions", "id", "CASCADE"},
	{"personality_results", "user_id", "users", "id", "CASCADE"},
	{"user_chapter_progress", "user_id", "users", "id", "CASCADE"},
	{"user_chapter_progress", "chapter_id", "chapters", "id", ""},
	{"user_chapter_progress", "chapter_version_id", "chapter_versions", "id", ""},
	{"user_module_progress", "user_id", "users", "id", "CASCADE"},
	{"user_module_progress", "module_id", "modules", "id", ""},
	{"quiz_attempts", "user_id", "users", "id", "CASCADE"},
	{"quiz_attempts", "quiz_id", "quizzes", "id", ""},
	{"exercise_submissions", "user_id", "users", "id", "CASCADE"},
	{"exercise_submissions", "module_id", "modules", "id", ""},
	{"user_stats", "user_id", "users", "id", "CASCADE"},
	{"user_stats", "level", "levels", "level", ""},
	{"subscriptions", "user_id", "users", "id", "CASCADE"},
	{"subscriptions", "plan_id", "plans", "id", ""},
	{"payments", "subscription_id", "subscriptions", "id", "CASCADE"},
}

// Migrate brings the schema up to date. Safe to run repeatedly and concurrently.
func Migrate(ctx context.Context, gdb *gorm.DB) error {
	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", lockID).Error; err != nil {
			return fmt.Errorf("schema: lock: %w", err)
		}
		var models []any
		for _, m := range [][]any{
			identity.Models(), content.Models(), learning.Models(),
			gamification.Models(), billing.Models(), introtest.Models(),
		} {
			models = append(models, m...)
		}
		if err := tx.AutoMigrate(models...); err != nil {
			return fmt.Errorf("schema: automigrate: %w", err)
		}
		if err := addForeignKeys(tx); err != nil {
			return err
		}
		return seed(tx)
	})
}

func addForeignKeys(tx *gorm.DB) error {
	for _, f := range foreignKeys {
		name := fmt.Sprintf("fk_%s_%s", f.table, f.column)
		if tx.Migrator().HasConstraint(f.table, name) {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %q ADD CONSTRAINT %q FOREIGN KEY (%q) REFERENCES %q (%q)",
			f.table, name, f.column, f.refTable, f.refColumn)
		if f.onDelete != "" {
			stmt += " ON DELETE " + f.onDelete
		}
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("schema: add %s: %w", name, err)
		}
	}
	return nil
}

func seed(tx *gorm.DB) error {
	beginner := "Beginner"
	levels := []gamification.Level{{Level: 1, XPRequired: 0, Title: &beginner}}
	plans := []billing.Plan{{
		Code: "academy_plus", Name: "Academy Plus",
		FirstPeriodCents: 499, RecurringCents: 399, UnlockChapterPosition: 2,
	}}
	skip := clause.OnConflict{DoNothing: true}
	if err := tx.Clauses(skip).Create(&levels).Error; err != nil {
		return fmt.Errorf("schema: seed levels: %w", err)
	}
	if err := tx.Clauses(skip).Create(&plans).Error; err != nil {
		return fmt.Errorf("schema: seed plans: %w", err)
	}
	return nil
}
