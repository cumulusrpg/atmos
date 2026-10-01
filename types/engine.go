package types

// Engine is the engine a repository is handed: its log, and emitting
// to it. The concrete implementation lives in the main atmos package.
type Engine interface {
	// Emit attempts to emit an event through validation and commitment
	Emit(event Event) bool

	// EmitAll commits events together, in order, or not at all
	EmitAll(events ...Event) bool

	// GetEvents returns all events in the system
	GetEvents() []Event

	// SetEvents sets the events directly (for rebuilding from event log)
	SetEvents(events []Event)

	// MarshalEvents serializes events to JSON
	MarshalEvents(events []Event) ([]byte, error)

	// UnmarshalEvents deserializes events from JSON
	UnmarshalEvents(jsonData []byte) ([]Event, error)
}
