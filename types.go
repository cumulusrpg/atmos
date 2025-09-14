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

// EventEmitter provides minimal interface for emitting events
type EventEmitter interface {
	Emit(event Event) bool
}

// EventValidator validates whether an event should be committed to the log
type EventValidator interface {
	Validate(event Event, eventLog []Event, emitter EventEmitter) bool
}

// EventListener responds to events after they are committed
type EventListener interface {
	Handle(event Event, eventLog []Event, emitter EventEmitter)
}

// TypedEventValidator validates a specific event type with type safety
type TypedEventValidator[T Event] interface {
	ValidateTyped(event T, eventLog []Event, emitter EventEmitter) bool
}

// TypedEventListener handles a specific event type with type safety
type TypedEventListener[T Event] interface {
	HandleTyped(event T, eventLog []Event, emitter EventEmitter)
}

// ValidatorWrapper wraps a typed validator to implement the base interface
type ValidatorWrapper[T Event] struct {
	validator TypedEventValidator[T]
}

func (w ValidatorWrapper[T]) Validate(event Event, eventLog []Event, emitter EventEmitter) bool {
	typedEvent := event.(T) // Safe cast - engine ensures correct type
	return w.validator.ValidateTyped(typedEvent, eventLog, emitter)
}

// ListenerWrapper wraps a typed listener to implement the base interface
type ListenerWrapper[T Event] struct {
	listener TypedEventListener[T]
}

func (w ListenerWrapper[T]) Handle(event Event, eventLog []Event, emitter EventEmitter) {
	typedEvent := event.(T) // Safe cast - engine ensures correct type
	w.listener.HandleTyped(typedEvent, eventLog, emitter)
}

// NewTypedValidator creates a wrapper for a typed validator
func NewTypedValidator[T Event](validator TypedEventValidator[T]) EventValidator {
	return ValidatorWrapper[T]{validator: validator}
}

// NewTypedListener creates a wrapper for a typed listener
func NewTypedListener[T Event](listener TypedEventListener[T]) EventListener {
	return ListenerWrapper[T]{listener: listener}
}

// EventProjection transforms events into state through reduction
type EventProjection interface {
	InitialState() interface{}
	Reduce(state interface{}, event Event) interface{}
}