// Package store holds the billing module's GORM models (see internal/schema).
package store

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code                  string    `gorm:"not null;uniqueIndex"`
	Name                  string    `gorm:"not null"`
	FirstPeriodCents      int32     `gorm:"not null"`
	RecurringCents        int32     `gorm:"not null"`
	Currency              string    `gorm:"not null;default:EUR"`
	BillingInterval       string    `gorm:"not null;default:month"`
	UnlockChapterPosition int32     `gorm:"not null"`
}

func (Plan) TableName() string { return "plans" }

// Subscription: at most one active/past_due row per user (partial unique index).
type Subscription struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                 uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:subscriptions_one_live_per_user,where:status = 'active' OR status = 'past_due'"`
	PlanID                 uuid.UUID `gorm:"type:uuid;not null"`
	Status                 string    `gorm:"not null;check:chk_subscriptions_status,status IN ('active', 'past_due', 'canceled', 'expired')"`
	ProviderCustomerID     *string
	ProviderSubscriptionID *string `gorm:"uniqueIndex"`
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	CanceledAt             *time.Time
	ExpiredAt              *time.Time
	CreatedAt              time.Time `gorm:"not null;default:now()"`
}

func (Subscription) TableName() string { return "subscriptions" }

type Payment struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SubscriptionID    uuid.UUID `gorm:"type:uuid;not null"`
	AmountCents       int32     `gorm:"not null"`
	Currency          string    `gorm:"not null"`
	Status            string    `gorm:"not null;check:chk_payments_status,status IN ('pending', 'succeeded', 'failed', 'refunded')"`
	ProviderPaymentID *string   `gorm:"uniqueIndex"` // webhook idempotency key
	FailureReason     *string
	CreatedAt         time.Time `gorm:"not null;default:now()"`
}

func (Payment) TableName() string { return "payments" }

// Models lists every model of this module for AutoMigrate.
func Models() []any { return []any{&Plan{}, &Subscription{}, &Payment{}} }
