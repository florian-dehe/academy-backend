// Package events is a small synchronous in-process event bus.
//
// Handlers run in the publisher's goroutine with the publisher's context, so
// when the publisher is inside db.WithTx, handlers join the same transaction
// (XP on module completion). A handler error aborts Publish and thereby the
// publisher's transaction. Use the worker queue for work that may be eventual.
package events

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Event interface{ EventName() string }

type Handler func(ctx context.Context, e Event) error

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus { return &Bus{handlers: map[string][]Handler{}} }

func (b *Bus) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], h)
}

func (b *Bus) Publish(ctx context.Context, e Event) error {
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[e.EventName()]...)
	b.mu.RUnlock()
	for _, h := range hs {
		if err := h(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// Domain events shared across modules (kept here so modules do not import each other).

const (
	NameModuleCompleted     = "module.completed"
	NameChapterCompleted    = "chapter.completed"
	NameSubscriptionChanged = "subscription.changed"
)

type ModuleCompleted struct {
	UserID   uuid.UUID
	ModuleID uuid.UUID
	XP       int
}

func (ModuleCompleted) EventName() string { return NameModuleCompleted }

type ChapterCompleted struct {
	UserID    uuid.UUID
	ChapterID uuid.UUID
}

func (ChapterCompleted) EventName() string { return NameChapterCompleted }

type SubscriptionChanged struct {
	UserID uuid.UUID
	Status string // active | past_due | canceled | expired
}

func (SubscriptionChanged) EventName() string { return NameSubscriptionChanged }
