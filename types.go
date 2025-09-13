package atmos

import "time"

// Framework types that could eventually move to the atmos package

// Result represents the outcome of a game action
type Result struct {
	Success bool
	Message string
}

// Event represents something that happened in the game
type Event interface {
	Type() string
	Timestamp() time.Time
}

// EventValidator validates whether an event should be committed to the log
type EventValidator interface {
	Validate(event Event, gameState []Event, engine *Engine) bool
}

// EventListener responds to events after they are committed
type EventListener interface {
	Handle(event Event, engine *Engine)
}