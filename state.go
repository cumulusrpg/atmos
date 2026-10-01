package atmos

import "fmt"

// State is a handle on one of an engine's states: holding it is how
// you read the state, and how you say what events do to it — no name
// to look up, no cast. A handle is bound to the engine it was made on,
// so it belongs with whatever wires that engine; rules that need it,
// or anything else that wiring knows, can be methods there (see On).
//
// The state Get returns is the engine's own: read it, don't change it.
type State[S any] struct {
	e    *Engine
	name string
}

// NewState registers a state on e, with a function that makes its
// initial value, and returns its handle. The name is what snapshots
// know it by (see SetSnapshot).
//
// An event that has a method Apply(S) S applies itself to the state:
// that's what the event does to it, and it needs no reducer. Other
// states that care about the event are updated with Reduces.
func NewState[S any](e *Engine, name string, initial func() S) State[S] {
	e.states[name] = &stateDef{
		initial: func() any { return initial() },
		apply: func(state any, event Event) (any, bool) {
			applier, ok := event.(Applier[S])
			if !ok {
				return state, false
			}
			typed, _ := state.(S)
			return applier.Apply(typed), true
		},
		reducers: make(map[string]func(any, Event) any),
	}
	e.current = nil
	return State[S]{e: e, name: name}
}

// Applier is an event that says what it does to a state of type S.
type Applier[S any] interface {
	Apply(S) S
}

// Get is the state as the event log leaves it — while a batch is being
// validated, as the batch so far leaves it. Reading doesn't replay the
// log: each event is reduced once, as it's staged.
func (s State[S]) Get() S {
	state, _ := s.e.get(s.name).(S)
	return state
}

// Name is the state's name.
func (s State[S]) Name() string { return s.name }

// Update is what events of type T do to one state: see Reduces.
type Update[T Event] struct {
	state  string
	reduce func(any, Event) any
}

// Reduces is an Update to s: reduce folds an event of type T into it.
// Like any reducer, it may change the state it's given. An event that
// applies itself to s (see NewState) can't also be reduced into it.
// Usage: On[Stocked](e).Updates(Reduces(tally, countStocked))
func Reduces[S any, T Event](s State[S], reduce func(S, T) S) Update[T] {
	var ev T
	if _, applies := any(ev).(Applier[S]); applies {
		panic(fmt.Sprintf("atmos: %T already applies itself to state %q", ev, s.name))
	}
	return Update[T]{state: s.name, reduce: func(state any, event Event) any {
		typed, _ := state.(S)
		return reduce(typed, event.(T))
	}}
}
