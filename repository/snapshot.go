package repository

import "github.com/cumulusrpg/atmos/types"

// InMemorySnapshot implements both EventRepository and SnapshotRepository.
// Stores events and snapshots in memory - suitable for testing and
// simple engines that don't need persistence.
type InMemorySnapshot struct {
	events    []types.Event
	snapshots map[string][]byte // state name -> JSON snapshot
}

// NewInMemorySnapshot creates a new in-memory repository with snapshot support
func NewInMemorySnapshot() *InMemorySnapshot {
	return &InMemorySnapshot{
		events:    make([]types.Event, 0),
		snapshots: make(map[string][]byte),
	}
}

// =============================================================================
// EventRepository implementation
// =============================================================================

// Add commits a new event to the in-memory store
func (r *InMemorySnapshot) Add(engine types.Engine, event types.Event) error {
	r.events = append(r.events, event)
	return nil
}

// GetAll returns all events from the in-memory store
func (r *InMemorySnapshot) GetAll(engine types.Engine) []types.Event {
	return append([]types.Event{}, r.events...)
}

// SetAll atomically replaces all events in the in-memory store
func (r *InMemorySnapshot) SetAll(engine types.Engine, events []types.Event) error {
	r.events = append([]types.Event{}, events...)
	return nil
}

// =============================================================================
// SnapshotRepository implementation
// =============================================================================

// GetSnapshot returns the snapshot data for a state, or false if none exists
func (r *InMemorySnapshot) GetSnapshot(stateName string) ([]byte, bool) {
	data, exists := r.snapshots[stateName]
	return data, exists
}

// SetSnapshot stores a snapshot for a state
func (r *InMemorySnapshot) SetSnapshot(stateName string, data []byte) error {
	r.snapshots[stateName] = data
	return nil
}

// ClearSnapshot removes the snapshot for a state
func (r *InMemorySnapshot) ClearSnapshot(stateName string) error {
	delete(r.snapshots, stateName)
	return nil
}
