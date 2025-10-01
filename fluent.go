package atmos

// EventRegistration provides a fluent API for configuring event handlers
type EventRegistration struct {
	engine    *Engine
	eventType string
}

// Event starts a fluent event registration chain
// This allows self-documenting engine configuration:
//
//	engine.Event("player_registered").
//		WithValidator(&RequireValidPlayerName{}).
//		WithListener(&PlacePlayerOnBoard{}).
//		WithReducer("turns", UpdateCurrentPlayer)
//
// This is equivalent to the traditional style but more readable at a glance.
func (e *Engine) Event(eventType string) *EventRegistration {
	return &EventRegistration{
		engine:    e,
		eventType: eventType,
	}
}

// WithValidator adds a validator to this event (chainable)
func (r *EventRegistration) WithValidator(validator EventValidator) *EventRegistration {
	r.engine.RegisterValidator(r.eventType, validator)
	return r
}

// WithListener adds a listener to this event (chainable)
func (r *EventRegistration) WithListener(listener EventListener) *EventRegistration {
	r.engine.RegisterListener(r.eventType, listener)
	return r
}

// WithReducer adds a state reducer for this event (chainable)
// stateName is the state key (e.g., "turns", "tokens")
// reducer is the function that updates that state
func (r *EventRegistration) WithReducer(stateName string, reducer StateReducer) *EventRegistration {
	// Get existing state registry
	if registry, exists := r.engine.states[stateName]; exists {
		// Add reducer to existing registry
		registry.Reducers[r.eventType] = reducer
		r.engine.states[stateName] = registry
	}
	// If state doesn't exist, this is a no-op (state must be registered first)
	return r
}

// WithEventFactory registers a factory function for JSON deserialization (chainable)
func (r *EventRegistration) WithEventFactory(factory func() Event) *EventRegistration {
	r.engine.RegisterEventType(r.eventType, factory)
	return r
}