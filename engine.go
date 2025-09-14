package atmos

import "encoding/json"

// Engine coordinates event emission, validation, and commitment
type Engine struct {
	events         []Event
	validators     map[string][]EventValidator // event type -> validators
	listeners      map[string][]EventListener  // event type -> listeners
	eventFactories map[string]func() Event     // event type -> factory function
}

// NewEngine creates a new engine instance
func NewEngine() *Engine {
	return &Engine{
		events:         make([]Event, 0),
		validators:     make(map[string][]EventValidator),
		listeners:      make(map[string][]EventListener),
		eventFactories: make(map[string]func() Event),
	}
}

// RegisterValidator registers a validator for a specific event type
func (e *Engine) RegisterValidator(eventType string, validator EventValidator) {
	e.validators[eventType] = append(e.validators[eventType], validator)
}

// RegisterListener registers a listener for a specific event type
func (e *Engine) RegisterListener(eventType string, listener EventListener) {
	e.listeners[eventType] = append(e.listeners[eventType], listener)
}

// RegisterEventType registers a factory function for a specific event type
func (e *Engine) RegisterEventType(eventType string, factory func() Event) {
	e.eventFactories[eventType] = factory
}

// Emit attempts to emit an event through validation and commitment
func (e *Engine) Emit(event Event) bool {
	// Get validators for this event type
	validators, exists := e.validators[event.Type()]
	if exists {
		// All validators must approve
		for _, validator := range validators {
			if !validator.Validate(event, e.events, e) {
				return false // validation failed
			}
		}
	}
	// No validators or all validators passed - commit the event
	e.events = append(e.events, event)

	// Call listeners after commitment
	listeners, hasListeners := e.listeners[event.Type()]
	if hasListeners {
		for _, listener := range listeners {
			listener.Handle(event, e.events, e)
		}
	}

	return true
}

// GetEvents returns all events in the system
func (e *Engine) GetEvents() []Event {
	return append([]Event{}, e.events...)
}

// SetEvents sets the events directly (for rebuilding from event log)
func (e *Engine) SetEvents(events []Event) {
	e.events = append([]Event{}, events...)
}

// EventWrapper wraps events with their type for JSON serialization
type EventWrapper struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// MarshalEvents serializes events to JSON with type information
func (e *Engine) MarshalEvents(events []Event) ([]byte, error) {
	var wrappers []EventWrapper
	for _, event := range events {
		wrapper := EventWrapper{
			Type: event.Type(),
			Data: event,
		}
		wrappers = append(wrappers, wrapper)
	}
	return json.Marshal(wrappers)
}

// UnmarshalEvents deserializes JSON into events using registered event types
func (e *Engine) UnmarshalEvents(jsonData []byte) ([]Event, error) {
	var wrappers []EventWrapper
	if err := json.Unmarshal(jsonData, &wrappers); err != nil {
		return nil, err
	}
	
	var events []Event
	for _, wrapper := range wrappers {
		// Get factory for this event type
		factory, exists := e.eventFactories[wrapper.Type]
		if !exists {
			continue // Skip unknown event types
		}
		
		// Create new event instance and unmarshal into it
		event := factory()
		eventJSON, err := json.Marshal(wrapper.Data)
		if err != nil {
			continue // Skip events that can't be re-marshaled
		}
		
		if err := json.Unmarshal(eventJSON, event); err != nil {
			continue // Skip events that can't be unmarshaled
		}
		
		// If event is a pointer, dereference it before adding
		events = append(events, event)
	}
	
	return events, nil
}