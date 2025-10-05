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

// English-like aliases for better readability

// When is an alias for Event() to read more naturally with optional event factory
// Usage: When("player_registered").Requires(...).Then(...)
// Or: When("player_registered", func() Event { return &MyEvent{} }).Requires(...)
func (e *Engine) When(eventType string, factory ...func() Event) *EventRegistration {
	reg := e.Event(eventType)
	if len(factory) > 0 {
		reg.WithEventFactory(factory[0])
	}
	return reg
}

// Requires is an alias for WithValidator() to read like a requirement
// Accepts multiple validators for convenience
// Usage: When("player_registered").Requires(Valid(&MyValidator{}), Valid(&AnotherValidator{}))
func (r *EventRegistration) Requires(validators ...EventValidator) *EventRegistration {
	for _, validator := range validators {
		r.WithValidator(validator)
	}
	return r
}

// Then is an alias for WithListener() to read like a consequence
// Accepts multiple listeners for convenience
// Usage: When("player_registered").Then(Do(&MyListener{}), Do(&AnotherListener{}))
func (r *EventRegistration) Then(listeners ...EventListener) *EventRegistration {
	for _, listener := range listeners {
		r.WithListener(listener)
	}
	return r
}

// Updates is an alias for WithReducer() to describe state changes
// Usage: When("player_registered").Updates("players", reducer)
func (r *EventRegistration) Updates(stateName string, reducer StateReducer) *EventRegistration {
	return r.WithReducer(stateName, reducer)
}

// Except creates an exception to skip a validator under certain conditions
// This explicitly documents when and why validation rules are bypassed
// Usage: When("card_played").Requires(Valid(&RequireCardInHand{})).
//           Except(Valid(&RequireCardInHand{}), condition, "reason")
func (r *EventRegistration) Except(validator EventValidator, condition func(*Engine, Event) bool, reason string) *EventRegistration {
	exception := ValidatorException{
		Validator: validator,
		Condition: condition,
		Reason:    reason,
	}
	r.engine.RegisterException(r.eventType, exception)
	return r
}

// Helper functions for wrapping typed validators and listeners

// Valid wraps a typed validator for use with Requires()
// Usage: Requires(Valid(&MyValidator{}))
func Valid[T Event](validator TypedEventValidator[T]) EventValidator {
	return NewTypedValidator(validator)
}

// Do wraps a typed listener for use with Then()
// Usage: Then(Do(&MyListener{}))
func Do[T Event](listener TypedEventListener[T]) EventListener {
	return NewTypedListener(listener)
}