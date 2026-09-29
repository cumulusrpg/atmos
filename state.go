package atmos

// State is a handle on one of an engine's states: holding it is how
// you read the state, and how you say what events do to it — no name
// to look up, no cast. A handle is bound to the engine it was made on,
// so it belongs with whatever wires that engine; rules that need it,
// or anything else that wiring knows, can be methods there (see Rule).
//
// The state Get returns is the engine's own: read it, don't change it.
type State[S any] struct {
	e    *Engine
	name string
}

// NewState registers a state on e, with a function that makes its
// initial value, and returns its handle. The name is the state's key
// for snapshots and for untyped callers of GetState.
func NewState[S any](e *Engine, name string, initial func() S) State[S] {
	e.RegisterState(name, func() interface{} { return initial() })
	return State[S]{e: e, name: name}
}

// Get is the state as the event log leaves it (see GetState).
func (s State[S]) Get() S {
	state, _ := s.e.GetState(s.name).(S)
	return state
}

// Name is the state's name.
func (s State[S]) Name() string { return s.name }

// Update is what one kind of event does to one state: see Reduces.
type Update struct {
	state   string
	reducer StateReducer
}

// Reduces is an Update to s: reduce folds an event of type T into it.
// Like any reducer, it may change the state it's given.
// Usage: When("stocked").Updates(Reduces(shelf, addItem))
func Reduces[S any, T Event](s State[S], reduce func(S, T) S) Update {
	return Update{state: s.name, reducer: func(_ *Engine, state interface{}, event Event) interface{} {
		typed, _ := state.(S)
		return reduce(typed, event.(T))
	}}
}

// Rule makes a plain function of the event a validator — a method on
// whatever holds the engine's handles, typically, so it reaches them
// and the rest of that wiring through its receiver.
// Usage: Requires(Rule(p.hasRoom))
func Rule[T Event](valid func(T) bool) EventValidator {
	return ValidFunc(func(_ *Engine, event T) bool { return valid(event) })
}
