package atmos

// EmitBuilder provides a fluent API for constructing event emission listeners
// This allows explicit, self-documenting event flows like:
//
//	Emit("tokens_granted").
//	    If(func(e *PlayerRegisteredEvent) bool { return e.PlayerType == Creator }).
//	    From(func(e *PlayerRegisteredEvent) []Event {
//	        return []Event{&TokensGrantedEvent{PlayerName: e.PlayerName, Amount: 3, ...}}
//	    })
type EmitBuilder[TIn Event, TOut Event] struct {
	eventType string
	condition func(TIn) bool
	transform func(TIn) []TOut
}

// Emit starts building an event emission listener
// The eventType parameter documents what event will be emitted
func Emit[TIn Event, TOut Event](eventType string) *EmitBuilder[TIn, TOut] {
	return &EmitBuilder[TIn, TOut]{
		eventType: eventType,
	}
}

// If adds an optional condition - the events are only emitted if the condition returns true
func (eb *EmitBuilder[TIn, TOut]) If(condition func(TIn) bool) *EmitBuilder[TIn, TOut] {
	eb.condition = condition
	return eb
}

// From specifies the transformation function that creates new events from the incoming event
// The function returns a slice to support emitting multiple events (fan-out pattern)
func (eb *EmitBuilder[TIn, TOut]) From(transform func(TIn) []TOut) EventListener {
	eb.transform = transform

	// Return a typed listener wrapper
	return NewTypedListener(&emitListener[TIn, TOut]{
		condition: eb.condition,
		transform: eb.transform,
	})
}

// emitListener is the actual listener implementation
type emitListener[TIn Event, TOut Event] struct {
	condition func(TIn) bool
	transform func(TIn) []TOut
}

// HandleTyped implements the TypedEventListener interface
func (el *emitListener[TIn, TOut]) HandleTyped(engine *Engine, event TIn) {
	// Check condition if present
	if el.condition != nil && !el.condition(event) {
		return
	}

	// Transform incoming event to new events
	newEvents := el.transform(event)

	// Emit each new event (will go through engine's validation)
	for _, newEvent := range newEvents {
		engine.Emit(newEvent)
	}
}
