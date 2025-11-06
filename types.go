package atmos

import (
	"math/rand"
	"time"
)

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
	Validate(engine *Engine, event Event) bool
}

// EventListener responds to events after they are committed
type EventListener interface {
	Handle(engine *Engine, event Event)
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

func (w ValidatorWrapper[T]) Validate(engine *Engine, event Event) bool {
	typedEvent := event.(T) // Safe cast - engine ensures correct type
	return w.validator.ValidateTyped(engine, typedEvent)
}

// ListenerWrapper wraps a typed listener to implement the base interface
type ListenerWrapper[T Event] struct {
	listener TypedEventListener[T]
}

func (w ListenerWrapper[T]) Handle(engine *Engine, event Event) {
	typedEvent := event.(T) // Safe cast - engine ensures correct type
	w.listener.HandleTyped(engine, typedEvent)
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

// RandomContext provides controlled randomness injection
type RandomContext interface {
	Intn(n int) int
	Float64() float64
}

// DefaultRandomContext provides real randomness using math/rand
type DefaultRandomContext struct{}

func (r DefaultRandomContext) Intn(n int) int {
	return rand.Intn(n)
}

func (r DefaultRandomContext) Float64() float64 {
	return rand.Float64()
}

// SeededRandomContext provides deterministic randomness with a seed
type SeededRandomContext struct {
	rng *rand.Rand
}

func NewSeededRandomContext(seed int64) *SeededRandomContext {
	return &SeededRandomContext{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (r *SeededRandomContext) Intn(n int) int {
	return r.rng.Intn(n)
}

func (r *SeededRandomContext) Float64() float64 {
	return r.rng.Float64()
}

// EventRepository handles event storage and persistence
type EventRepository interface {
	// Add commits a new event to storage
	Add(event Event) error

	// GetAll returns all events for replay
	GetAll() []Event

	// SetAll atomically replaces all events (for rebuilding from event log)
	SetAll(events []Event) error
}
