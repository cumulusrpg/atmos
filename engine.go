package atmos

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/cumulusrpg/atmos/types"
)

// Event is something that happened.
type Event = types.Event

// EventRepository is where an engine keeps its log.
type EventRepository = types.EventRepository

// SnapshotRepository is a repository that can also seed states (see
// SetSnapshot).
type SnapshotRepository = types.SnapshotRepository

// Engine keeps an event log, and the states it leaves. What it does
// with each kind of event is declared with On, its states with
// NewState, and what can be done with Command.
type Engine struct {
	repository types.EventRepository
	kinds      map[string]*kind                   // event type -> what's done with it (see On)
	states     map[string]*stateDef               // state name -> how it's made (see NewState)
	commands   map[reflect.Type]func(any) []Event // command type -> decision (see Command)

	batch   []Event // events validated but not yet committed (see EmitAll)
	inBatch bool

	// current is every state as the log, and the batch so far, leave
	// it: each event is reduced once, as it's staged. nil means it has
	// to be rebuilt from the log — at first, and whenever something
	// has been reduced that didn't happen.
	current map[string]any
}

// kind is what an engine does with one type of event.
type kind struct {
	rules  []func(Event) bool // all must pass for it to be committed
	before []func(Event)      // run in its batch, once it's passed
	then   []func(Event)      // run once its batch is committed
	decode func([]byte) (Event, error)
}

// stateDef is one of an engine's states: how it starts, and what
// events do to it.
type stateDef struct {
	initial  func() any                      // a new initial value, each time it's called
	apply    func(any, Event) (any, bool)    // an event that applies itself; ok is false if it doesn't
	reducers map[string]func(any, Event) any // event type -> update (see Reduces)
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
		repository: repository.NewInMemory(), // default repository
		kinds:      make(map[string]*kind),
		states:     make(map[string]*stateDef),
		commands:   make(map[reflect.Type]func(any) []Event),
	}
	for _, opt := range opts {
		opt(engine)
	}
	return engine
}

// kind is what e does with events of eventType, declared or not.
func (e *Engine) kind(eventType string) *kind {
	k := e.kinds[eventType]
	if k == nil {
		k = &kind{}
		e.kinds[eventType] = k
	}
	return k
}

// get is the state name as the event log leaves it — while a batch is
// being validated, as the batch so far leaves it. Events are reduced as
// they're staged, so reading a state doesn't replay the log; it's
// rebuilt from the log only the first time, and after anything that
// was reduced is undone (a refused batch, a replaced log, a changed
// snapshot or update). The engine is taken to be the only writer to
// its repository while it's in use.
func (e *Engine) get(name string) any {
	if e.current == nil {
		e.rebuild()
	}
	return e.current[name]
}

// rebuild reduces every state from a new initial value — or a snapshot
// merged over one (partial snapshots are supported) — through the log
// and the batch so far.
func (e *Engine) rebuild() {
	current := make(map[string]any, len(e.states))
	for name, def := range e.states {
		state := def.initial()
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

// reduce folds one event into every state it does something to.
func (e *Engine) reduce(event Event) {
	for name, def := range e.states {
		if reduce, updates := def.reducers[event.Type()]; updates {
			e.current[name] = reduce(e.current[name], event)
		} else if next, applies := def.apply(e.current[name], event); applies {
			e.current[name] = next
		}
	}
}

// Emit attempts to emit an event through validation and commitment.
// It's EmitAll with one event.
func (e *Engine) Emit(event Event) bool {
	return e.EmitAll(event)
}

// EmitAll commits events together, in order, or not at all. Each is
// validated against the state the ones before it leave — a state read
// while a batch is being validated answers with the batch so far — and
// if any is refused, nothing is committed and no listener hears of any
// of them. Before hooks run inside the batch, and whatever they emit
// joins it; listeners run once the whole batch is in, and whatever
// they emit is a batch of its own.
//
// Called while a batch is in progress (from a before hook or a
// rule), EmitAll adds to that batch; if it's refused, only its own
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

	// Listeners run once the whole batch is committed
	for _, event := range batch {
		if k := e.kinds[event.Type()]; k != nil {
			for _, then := range k.then {
				then(event)
			}
		}
	}
	return true
}

// stage validates event against the batch so far, runs its before
// hooks, and adds it to the batch.
func (e *Engine) stage(event Event) bool {
	if k := e.kinds[event.Type()]; k != nil {
		for _, rule := range k.rules {
			if !rule(event) {
				return false
			}
		}
		// Before hooks run once the event has passed, but before it
		// joins the batch: whatever they emit lands ahead of it
		for _, before := range k.before {
			before(event)
		}
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

// written is an event as it's written to JSON, with its type; read
// is one being read back, before its type says what to decode it as.
type (
	written struct {
		Type string `json:"type"`
		Data Event  `json:"data"`
	}
	read struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
)

// MarshalEvents serializes events to JSON with type information
func (e *Engine) MarshalEvents(events []Event) ([]byte, error) {
	out := make([]written, 0, len(events))
	for _, event := range events {
		out = append(out, written{Type: event.Type(), Data: event})
	}
	return json.Marshal(out)
}

// UnmarshalEvents deserializes JSON into events, each as the type it
// was declared with (see On). Events of a type that wasn't declared,
// or that don't decode as it, are skipped.
func (e *Engine) UnmarshalEvents(jsonData []byte) ([]Event, error) {
	var in []read
	if err := json.Unmarshal(jsonData, &in); err != nil {
		return nil, err
	}
	var events []Event
	for _, w := range in {
		k := e.kinds[w.Type]
		if k == nil || k.decode == nil {
			continue
		}
		if event, err := k.decode(w.Data); err == nil {
			events = append(events, event)
		}
	}
	return events, nil
}

// =============================================================================
// Snapshot API
// =============================================================================

// SetSnapshot stores a snapshot for the state named stateName (see
// State.Name). The snapshot can be a struct or a map.
// Partial snapshots are supported - only provided fields will override defaults.
// Returns an error if the repository doesn't support snapshots.
func (e *Engine) SetSnapshot(stateName string, snapshot any) error {
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
func (e *Engine) mergeSnapshot(initialState any, snapshotData []byte) any {
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
