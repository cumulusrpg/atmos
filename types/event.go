package types

// Event represents something that happened in the system
type Event interface {
	Type() string
}

// EventEmitter provides minimal interface for emitting events
type EventEmitter interface {
	Emit(event Event) bool
}

// EventValidator validates whether an event should be committed to the log
type EventValidator interface {
	Validate(engine Engine, event Event) bool
}

// EventListener responds to events after they are committed
type EventListener interface {
	Handle(engine Engine, event Event)
}
