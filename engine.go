package atmos

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/cumulusrpg/atmos/types"
)

// StateReducer represents a function that reduces an event into a state
type StateReducer func(engine *Engine, state interface{}, event Event) interface{}

// StateRegistry holds state and its reducers
type StateRegistry struct {
	Initial  func() interface{}      // a new initial state, each time it's called
	Reducers map[string]StateReducer // event type -> reducer function

	// apply folds an event that applies itself to this state (see
	// NewState); ok is false if it doesn't.
	apply func(state interface{}, event Event) (next interface{}, ok bool)
}

// Engine coordinates event emission, validation, and commitment
type Engine struct {
	repository     types.EventRepository                  // event storage abstraction
	validators     map[string][]EventValidator            // event type -> validators
	exceptions     map[string][]ValidatorException        // event type -> validator exceptions
	beforeHooks    map[string][]EventListener             // event type -> pre-commit hooks
	listeners      map[string][]EventListener             // event type -> listeners
	states         map[string]StateRegistry               // state name -> state registry
	eventFactories map[string]func() Event                // event type -> factory function
	eventDecoders  map[string]func([]byte) (Event, error) // event type -> decoder (see On)
	commands       map[reflect.Type]func(any) []Event     // command type -> decision (see Command)
	services       map[string]interface{}                 // service name -> service instance (service locator)

	batch   []Event // events validated but not yet committed (see EmitAll)
	inBatch bool

	// current is every state as the log, and the batch so far, leave
	// it: each event is reduced once, as it's staged. nil means it has
	// to be rebuilt from the log — at first, and whenever something
	// has been reduced that didn't happen.
	current map[string]interface{}
}

// EngineOption configures engine construction
type EngineOption func(*Engine)

// WithRepository sets a custom event repository
func WithRepository(repo types.EventRepository) EngineOption {
	return func(e *Engine) {
		e.repository = repo
	}
}

// NewEngine creates a new engine with optional configuration
func NewEngine(opts ...EngineOption) *Engine {
	engine := &Engine{
		repository:     repository.NewInMemory(), // default repository
		validators:     make(map[string][]EventValidator),
		exceptions:     make(map[string][]ValidatorException),
		beforeHooks:    make(map[string][]EventListener),
		listeners:      make(map[string][]EventListener),
		states:         make(map[string]StateRegistry),
		eventFactories: make(map[string]func() Event),
		eventDecoders:  make(map[string]func([]byte) (Event, error)),
		commands:       make(map[reflect.Type]func(any) []Event),
		services:       make(map[string]interface{}),
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

// RegisterState registers a state by name, with a function that makes
// its initial value. It's called again whenever the state is rebuilt
// from the log, so reducers may change the state they're given: no
// state is ever reused once something it was reduced from is undone.
// Reducers should be attached via the fluent API using Updates()
func (e *Engine) RegisterState(name string, initial func() interface{}) {
	e.states[name] = StateRegistry{
		Initial:  initial,
		Reducers: make(map[string]StateReducer),
	}
	e.current = nil
}

// RegisterService registers a service (reference data/utilities) in the service locator
func (e *Engine) RegisterService(name string, service interface{}) {
	e.services[name] = service
}

// GetService retrieves a registered service by name
func (e *Engine) GetService(name string) interface{} {
	return e.services[name]
}

// GetState is a state as the event log leaves it — while a batch is
// being validated, as the batch so far leaves it. Events are reduced as
// they're staged, so reading a state doesn't replay the log; it's
// rebuilt from the log only the first time, and after anything that
// was reduced is undone (a refused batch, a replaced log, a changed
// snapshot or reducer). The engine is taken to be the only writer to
// its repository while it's in use.
//
// The state returned is the engine's own: callers mustn't change it.
func (e *Engine) GetState(name string) interface{} {
	if _, exists := e.states[name]; !exists {
		return nil
	}
	if e.current == nil {
		e.rebuild()
	}
	return e.current[name]
}

// rebuild reduces every state from a new initial value — or a snapshot
// merged over one (partial snapshots are supported) — through the log
// and the batch so far.
func (e *Engine) rebuild() {
	current := make(map[string]interface{}, len(e.states))
	for name, registry := range e.states {
		state := registry.Initial()
		if snapshotRepo, ok := e.repository.(types.SnapshotRepository); ok {
			if snapshotData, hasSnapshot := snapshotRepo.GetSnapshot(name); hasSnapshot {
				state = e.mergeSnapshot(state, snapshotData)
			}
		}
		current[name] = state
	}
	e.current = current
	for _, event := range e.GetEvents() {
		e.reduce(event)
	}
}

// reduce folds one event into every state that has a reducer for it.
func (e *Engine) reduce(event Event) {
	for name, registry := range e.states {
		if reducer, hasReducer := registry.Reducers[event.Type()]; hasReducer {
			e.current[name] = reducer(e, e.current[name], event)
		} else if registry.apply != nil {
			if next, ok := registry.apply(e.current[name], event); ok {
				e.current[name] = next
			}
		}
	}
}

// Emit attempts to emit an event through validation and commitment.
// It's EmitAll with one event.
func (e *Engine) Emit(event Event) bool {
	return e.EmitAll(event)
}

// EmitAll commits events together, in order, or not at all. Each is
// validated against the state the ones before it leave — GetState,
// while a batch is being validated, answers with the batch so far — and
// if any is refused, nothing is committed and no listener hears of any
// of them. Before hooks run inside the batch, and whatever they emit
// joins it; listeners run once the whole batch is in, and whatever
// they emit is a batch of its own.
//
// Called while a batch is in progress (from a before hook or a
// validator), EmitAll adds to that batch; if it's refused, only its own
// events are dropped.
func (e *Engine) EmitAll(events ...Event) bool {
	outer := !e.inBatch
	e.inBatch = true
	mark := len(e.batch)
	for _, event := range events {
		if !e.stage(event) {
			if len(e.batch) > mark {
				e.current = nil // what was reduced from the dropped events didn't happen
			}
			e.batch = e.batch[:mark]
			if outer {
				e.inBatch = false
			}
			return false
		}
	}
	if !outer {
		return true
	}
	batch := e.batch
	e.batch, e.inBatch = nil, false

	for _, event := range batch {
		if err := e.repository.Add(e, event); err != nil {
			e.current = nil
			return false // persistence failure
		}
	}

	// Call listeners after the whole batch is committed
	for _, event := range batch {
		for _, listener := range e.listeners[event.Type()] {
			listener.Handle(e, event)
		}
	}
	return true
}

// stage validates event against the batch so far, runs its before
// hooks, and adds it to the batch.
func (e *Engine) stage(event Event) bool {
	// Get exceptions for this event type
	exceptions := e.exceptions[event.Type()]

	// All validators must approve (unless exception applies)
	for _, validator := range e.validators[event.Type()] {
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

	// Call before hooks AFTER validation but BEFORE the event joins the
	// batch: whatever they emit lands ahead of it, in the same batch
	for _, hook := range e.beforeHooks[event.Type()] {
		hook.Handle(e, event)
	}

	e.batch = append(e.batch, event)
	if e.current != nil {
		e.reduce(event)
	}
	return true
}

// GetEvents returns all events in the system — while a batch is being
// validated, including the batch so far
func (e *Engine) GetEvents() []Event {
	return slices.Concat(e.repository.GetAll(e), e.batch)
}

// SetEvents sets the events directly (for rebuilding from event log)
// Panics if the repository fails to set events
func (e *Engine) SetEvents(events []Event) {
	e.current = nil
	if err := e.repository.SetAll(e, events); err != nil {
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
		// Events declared with On decode as their own type
		if decode, declared := e.eventDecoders[wrapper.Type]; declared {
			eventJSON, err := json.Marshal(wrapper.Data)
			if err != nil {
				continue
			}
			if event, err := decode(eventJSON); err == nil {
				events = append(events, event)
			}
			continue
		}

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

// =============================================================================
// Snapshot API
// =============================================================================

// SetSnapshot stores a snapshot for a state. The snapshot can be a struct or a map.
// Partial snapshots are supported - only provided fields will override defaults.
// Returns an error if the repository doesn't support snapshots.
func (e *Engine) SetSnapshot(stateName string, snapshot interface{}) error {
	snapshotRepo, ok := e.repository.(types.SnapshotRepository)
	if !ok {
		return errors.New("repository does not support snapshots")
	}

	// Serialize the snapshot to JSON
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	e.current = nil
	return snapshotRepo.SetSnapshot(stateName, data)
}

// ClearSnapshot removes the snapshot for a state.
// Returns an error if the repository doesn't support snapshots.
func (e *Engine) ClearSnapshot(stateName string) error {
	snapshotRepo, ok := e.repository.(types.SnapshotRepository)
	if !ok {
		return errors.New("repository does not support snapshots")
	}

	e.current = nil
	return snapshotRepo.ClearSnapshot(stateName)
}

// HasSnapshot returns true if a snapshot exists for the given state.
// Returns false if the repository doesn't support snapshots.
func (e *Engine) HasSnapshot(stateName string) bool {
	snapshotRepo, ok := e.repository.(types.SnapshotRepository)
	if !ok {
		return false
	}

	_, exists := snapshotRepo.GetSnapshot(stateName)
	return exists
}

// mergeSnapshot merges snapshot JSON data over an initial state value.
// This supports partial snapshots where only some fields are provided.
func (e *Engine) mergeSnapshot(initialState interface{}, snapshotData []byte) interface{} {
	// Get the type of the initial state
	initialType := reflect.TypeOf(initialState)
	if initialType.Kind() == reflect.Pointer {
		initialType = initialType.Elem()
	}

	// Create a new instance of the initial state type
	newState := reflect.New(initialType).Interface()

	// First, marshal the initial state to JSON and unmarshal into the new instance
	// This creates a deep copy
	initialJSON, err := json.Marshal(initialState)
	if err != nil {
		return initialState
	}
	if err := json.Unmarshal(initialJSON, newState); err != nil {
		return initialState
	}

	// Now unmarshal the snapshot data over it (partial merge)
	if err := json.Unmarshal(snapshotData, newState); err != nil {
		return initialState
	}

	// Return the dereferenced value to match the original type
	return reflect.ValueOf(newState).Elem().Interface()
}
