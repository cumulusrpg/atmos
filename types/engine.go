package types

// Engine defines the interface for the event engine.
// The concrete implementation lives in the main atmos package.
type Engine interface {
	// Emit attempts to emit an event through validation and commitment
	Emit(event Event) bool

	// GetState runs reducers on the current event log for a state
	GetState(name string) interface{}

	// GetEvents returns all events in the system
	GetEvents() []Event

	// SetEvents sets the events directly (for rebuilding from event log)
	SetEvents(events []Event)

	// GetService retrieves a registered service by name
	GetService(name string) interface{}
}
