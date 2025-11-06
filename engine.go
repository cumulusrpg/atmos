package atmos

import (
	"encoding/json"
	"sort"
)

// StateReducer represents a function that reduces an event into a state
type StateReducer func(engine *Engine, state interface{}, event Event) interface{}

// OrderedReducer holds a reducer with its execution priority
type OrderedReducer struct {
	StateName string
	Reducer   StateReducer
	Priority  int // lower numbers execute first
}

// StateRegistry holds state and its reducers
type StateRegistry struct {
	InitialState interface{}
	Reducers     map[string]StateReducer // event type -> reducer function
}

// Engine coordinates event emission, validation, and commitment
type Engine struct {
	repository      EventRepository                 // event storage abstraction
	validators      map[string][]EventValidator     // event type -> validators
	exceptions      map[string][]ValidatorException // event type -> validator exceptions
	beforeHooks     map[string][]EventListener      // event type -> pre-commit hooks
	listeners       map[string][]EventListener      // event type -> listeners
	states          map[string]StateRegistry        // state name -> state registry
	orderedReducers map[string][]OrderedReducer     // event type -> ordered reducers
	eventFactories  map[string]func() Event         // event type -> factory function
	services        map[string]interface{}          // service name -> service instance (service locator)
}

// EngineOption configures engine construction
type EngineOption func(*Engine)

// WithRepository sets a custom event repository
func WithRepository(repository EventRepository) EngineOption {
	return func(e *Engine) {
		e.repository = repository
	}
}

// WithEvents initializes the engine with an existing event log
func WithEvents(events []Event) EngineOption {
	return func(e *Engine) {
		e.SetEvents(events)
	}
}

// NewEngine creates a new engine with optional configuration
func NewEngine(opts ...EngineOption) *Engine {
	engine := &Engine{
		repository:      NewInMemoryRepository(), // default repository
		validators:      make(map[string][]EventValidator),
		exceptions:      make(map[string][]ValidatorException),
		beforeHooks:     make(map[string][]EventListener),
		listeners:       make(map[string][]EventListener),
		states:          make(map[string]StateRegistry),
		orderedReducers: make(map[string][]OrderedReducer),
		eventFactories:  make(map[string]func() Event),
		services:        make(map[string]interface{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

// RegisterValidator registers a validator for a specific event type
func (e *Engine) RegisterValidator(eventType string, validator EventValidator) {
	e.validators[eventType] = append(e.validators[eventType], validator)
}

// RegisterException registers an exception to skip a validator under certain conditions
func (e *Engine) RegisterException(eventType string, exception ValidatorException) {
	e.exceptions[eventType] = append(e.exceptions[eventType], exception)
}

// RegisterBeforeHook registers a pre-commit hook for a specific event type
// Before hooks run after validation but before the event is committed to the event log
func (e *Engine) RegisterBeforeHook(eventType string, hook EventListener) {
	e.beforeHooks[eventType] = append(e.beforeHooks[eventType], hook)
}

// RegisterListener registers a listener for a specific event type
func (e *Engine) RegisterListener(eventType string, listener EventListener) {
	e.listeners[eventType] = append(e.listeners[eventType], listener)
}

// RegisterEventType registers a factory function for a specific event type
func (e *Engine) RegisterEventType(eventType string, factory func() Event) {
	e.eventFactories[eventType] = factory
}

// RegisterState registers a state by name with its initial value
// Reducers should be attached via the fluent API using Updates()
func (e *Engine) RegisterState(name string, initialState interface{}) {
	e.states[name] = StateRegistry{
		InitialState: initialState,
		Reducers:     make(map[string]StateReducer),
	}
}

// RegisterOrderedReducer registers a reducer with priority for cross-state coordination
func (e *Engine) RegisterOrderedReducer(eventType, stateName string, reducer StateReducer, priority int) {
	orderedReducer := OrderedReducer{
		StateName: stateName,
		Reducer:   reducer,
		Priority:  priority,
	}

	e.orderedReducers[eventType] = append(e.orderedReducers[eventType], orderedReducer)

	// Sort by priority after adding
	sort.Slice(e.orderedReducers[eventType], func(i, j int) bool {
		return e.orderedReducers[eventType][i].Priority < e.orderedReducers[eventType][j].Priority
	})
}

// RegisterService registers a service (reference data/utilities) in the service locator
func (e *Engine) RegisterService(name string, service interface{}) {
	e.services[name] = service
}

// GetService retrieves a registered service by name
func (e *Engine) GetService(name string) interface{} {
	return e.services[name]
}

// GetState runs reducers on the current event log for a state
func (e *Engine) GetState(name string) interface{} {
	registry, exists := e.states[name]
	if !exists {
		return nil
	}

	state := registry.InitialState
	for _, event := range e.repository.GetAll() {
		// Check if there are ordered reducers for this event type
		if orderedReducers, hasOrdered := e.orderedReducers[event.Type()]; hasOrdered {
			// Apply ordered reducers for this state name only
			for _, orderedReducer := range orderedReducers {
				if orderedReducer.StateName == name {
					state = orderedReducer.Reducer(e, state, event)
				}
			}
		} else {
			// Use regular reducers
			reducer, hasReducer := registry.Reducers[event.Type()]
			if hasReducer {
				state = reducer(e, state, event)
			}
		}
	}

	return state
}

// Emit attempts to emit an event through validation and commitment
func (e *Engine) Emit(event Event) bool {
	// Get validators for this event type
	validators, exists := e.validators[event.Type()]
	if exists {
		// Get exceptions for this event type
		exceptions := e.exceptions[event.Type()]

		// All validators must approve (unless exception applies)
		for _, validator := range validators {
			// Check if any exception applies to skip this validator
			shouldSkip := false
			for _, exception := range exceptions {
				if exception.Validator == validator && exception.Condition(e, event) {
					shouldSkip = true
					break
				}
			}

			// Skip validation if exception applies
			if shouldSkip {
				continue
			}

			// Run validator
			if !validator.Validate(e, event) {
				return false // validation failed
			}
		}
	}

	// Call before hooks AFTER validation but BEFORE commitment
	// This allows side effects (like fate dice) to run as part of the event's transaction
	beforeHooks, hasBeforeHooks := e.beforeHooks[event.Type()]
	if hasBeforeHooks {
		for _, hook := range beforeHooks {
			hook.Handle(e, event)
		}
	}

	// No validators or all validators passed - commit the event to repository
	if err := e.repository.Add(event); err != nil {
		return false // persistence failure
	}

	// Call listeners after commitment
	listeners, hasListeners := e.listeners[event.Type()]
	if hasListeners {
		for _, listener := range listeners {
			listener.Handle(e, event)
		}
	}

	return true
}

// GetEvents returns all events in the system
func (e *Engine) GetEvents() []Event {
	return e.repository.GetAll()
}

// SetEvents sets the events directly (for rebuilding from event log)
// Panics if the repository fails to set events
func (e *Engine) SetEvents(events []Event) {
	if err := e.repository.SetAll(events); err != nil {
		panic("failed to set events in repository: " + err.Error())
	}
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
