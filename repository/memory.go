package repository

import "github.com/cumulusrpg/atmos/types"

// InMemory is a repository implementation that stores events in memory.
// Suitable for simple engines that don't need persistence.
type InMemory struct {
	events []types.Event
}

// NewInMemory creates a new in-memory repository
func NewInMemory() *InMemory {
	return &InMemory{
		events: make([]types.Event, 0),
	}
}

// Add commits a new event to the in-memory store
func (r *InMemory) Add(engine types.Engine, event types.Event) error {
	r.events = append(r.events, event)
	return nil
}

// GetAll returns all events from the in-memory store
func (r *InMemory) GetAll(engine types.Engine) []types.Event {
	return append([]types.Event{}, r.events...)
}

// SetAll atomically replaces all events in the in-memory store
func (r *InMemory) SetAll(engine types.Engine, events []types.Event) error {
	r.events = append([]types.Event{}, events...)
	return nil
}
