package atmos

import (
	"fmt"
	"reflect"
)

// A command is something that can be done: asked of the engine with
// Do, it decides — from the state as it is — what happens, as events,
// and they're emitted together, or not at all. Commands aren't events:
// they're never stored, so they can carry anything, behavior included.

// CommandRegistration declares a command of type C on an engine.
type CommandRegistration[C any] struct{ e *Engine }

// Command starts declaring the command C on e.
//
//	atmos.Command[Move](e).Decides(w.move)
func Command[C any](e *Engine) CommandRegistration[C] { return CommandRegistration[C]{e} }

// Decides says what the command becomes: the events that happen when
// it's done, or none if it can't be.
func (r CommandRegistration[C]) Decides(decide func(C) []Event) {
	r.e.commands[reflect.TypeFor[C]()] = func(command any) []Event { return decide(command.(C)) }
}

// Do does a command: its decision is emitted as one batch (see
// EmitAll). It reports whether anything happened — false when nothing
// was decided, or the batch was refused. Doing a command that wasn't
// declared is a wiring mistake, and panics.
func (e *Engine) Do(command any) bool {
	decide, declared := e.commands[reflect.TypeOf(command)]
	if !declared {
		panic(fmt.Sprintf("atmos: command %T was never declared", command))
	}
	events := decide(command)
	return len(events) > 0 && e.EmitAll(events...)
}
