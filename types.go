package atmos

import "github.com/cumulusrpg/atmos/types"

// =============================================================================
// Re-exported types from types package for convenience
// =============================================================================

// Event represents something that happened in the system
type Event = types.Event

// EventEmitter provides minimal interface for emitting events
type EventEmitter = types.EventEmitter

// EventValidator validates whether an event should be committed to the log
type EventValidator = types.EventValidator

// EventListener responds to events after they are committed
type EventListener = types.EventListener

// EventRepository handles event storage and persistence
type EventRepository = types.EventRepository

// SnapshotRepository handles snapshot storage for state seeding
type SnapshotRepository = types.SnapshotRepository

// =============================================================================
// Types that remain in main atmos package
// =============================================================================

// Result represents the outcome of a game action
type Result struct {
	Success bool
	Message string
}

// ValidatorException defines when a validator should be skipped
// This allows explicitly documenting exceptions to validation rules
type ValidatorException struct {
	Validator EventValidator            // The validator to skip
	Condition func(*Engine, Event) bool // When to skip it (returns true to skip)
	Reason    string                    // Documentation of why this exception exists
}

// TypedEventValidator validates a specific event type with type safety
type TypedEventValidator[T Event] interface {
	ValidateTyped(engine *Engine, event T) bool
}

// TypedEventListener handles a specific event type with type safety
type TypedEventListener[T Event] interface {
	HandleTyped(engine *Engine, event T)
}

// ValidatorWrapper wraps a typed validator to implement the base interface
type ValidatorWrapper[T Event] struct {
	validator TypedEventValidator[T]
}

func (w ValidatorWrapper[T]) Validate(engine types.Engine, event Event) bool {
	concreteEngine := engine.(*Engine)
	typedEvent := event.(T)
	return w.validator.ValidateTyped(concreteEngine, typedEvent)
}

// ListenerWrapper wraps a typed listener to implement the base interface
type ListenerWrapper[T Event] struct {
	listener TypedEventListener[T]
}

func (w ListenerWrapper[T]) Handle(engine types.Engine, event Event) {
	concreteEngine := engine.(*Engine)
	typedEvent := event.(T)
	w.listener.HandleTyped(concreteEngine, typedEvent)
}

// NewTypedValidator creates a wrapper for a typed validator
func NewTypedValidator[T Event](validator TypedEventValidator[T]) EventValidator {
	return ValidatorWrapper[T]{validator: validator}
}

// NewTypedListener creates a wrapper for a typed listener
func NewTypedListener[T Event](listener TypedEventListener[T]) EventListener {
	return ListenerWrapper[T]{listener: listener}
}

// Context interfaces for explicit dependency injection

// EventLogContext provides access to the event log for validation/projection
type EventLogContext interface {
	GetEvents() []Event
}

// EmitterContext provides controlled event emission capability
type EmitterContext interface {
	Emit(event Event) bool
}
