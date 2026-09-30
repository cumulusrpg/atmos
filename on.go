package atmos

import (
	"encoding/json"
	"reflect"
)

// Registration is what an engine does with events of type T: the rules
// they answer to, what runs before and after they're committed, and
// which states, besides their own, they update. Its methods take plain
// functions of the event — typically methods on whatever holds the
// engine's handles, reaching them through their receiver.
//
//	atmos.On[MoveMade](e).Requires(g.validMove).Then(g.checkForWinner)
type Registration[T Event] struct {
	e         *Engine
	eventType string
}

// On starts declaring what e does with events of type T. The event
// type comes from T itself, and events of type T come back from JSON
// (UnmarshalEvents) as T.
func On[T Event](e *Engine) Registration[T] {
	var ev T
	if t := reflect.TypeFor[T](); t.Kind() == reflect.Pointer {
		ev = reflect.New(t.Elem()).Interface().(T)
	}
	r := Registration[T]{e: e, eventType: ev.Type()}
	e.eventDecoders[r.eventType] = decode[T]
	return r
}

func decode[T Event](data []byte) (Event, error) {
	var ev T
	if t := reflect.TypeFor[T](); t.Kind() == reflect.Pointer {
		ev = reflect.New(t.Elem()).Interface().(T)
		return ev, json.Unmarshal(data, ev)
	}
	return ev, json.Unmarshal(data, &ev)
}

// Requires adds rules the event must pass to be committed.
func (r Registration[T]) Requires(rules ...func(T) bool) Registration[T] {
	for _, rule := range rules {
		r.e.RegisterValidator(r.eventType, Rule(rule))
	}
	return r
}

// Before adds hooks that run once the event has passed its rules, in
// its batch: whatever they emit is committed with it, or not at all.
func (r Registration[T]) Before(hooks ...func(T)) Registration[T] {
	for _, hook := range hooks {
		r.e.RegisterBeforeHook(r.eventType, listen(hook))
	}
	return r
}

// Then adds listeners that run once the event's batch is committed.
func (r Registration[T]) Then(listeners ...func(T)) Registration[T] {
	for _, listener := range listeners {
		r.e.RegisterListener(r.eventType, listen(listener))
	}
	return r
}

// Updates adds states that care about the event besides its own — the
// one it applies itself to (see NewState).
func (r Registration[T]) Updates(updates ...Update[T]) Registration[T] {
	for _, update := range updates {
		r.e.Event(r.eventType).WithReducer(update.state, update.reducer)
	}
	return r
}

func listen[T Event](handle func(T)) EventListener {
	return DoFunc(func(_ *Engine, event T) { handle(event) })
}
