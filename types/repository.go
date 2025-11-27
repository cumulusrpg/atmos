package types

// EventRepository handles event storage and persistence
type EventRepository interface {
	// Add commits a new event to storage
	Add(engine Engine, event Event) error

	// GetAll returns all events for replay
	GetAll(engine Engine) []Event

	// SetAll atomically replaces all events (for rebuilding from event log)
	SetAll(engine Engine, events []Event) error
}

// SnapshotRepository handles snapshot storage for state seeding (opt-in interface)
// Repositories that implement this interface enable snapshot-based state projection.
// This is useful for E2E testing where you want to seed specific states without
// replaying many events.
type SnapshotRepository interface {
	// GetSnapshot returns the snapshot data for a state, or false if none exists
	GetSnapshot(stateName string) ([]byte, bool)

	// SetSnapshot stores a snapshot for a state
	SetSnapshot(stateName string, data []byte) error

	// ClearSnapshot removes the snapshot for a state
	ClearSnapshot(stateName string) error
}
