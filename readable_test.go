package atmos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Jars of water, wired the readable way: events say what they do to
// the jars, the definition says only what's allowed, and pouring —
// water leaving one jar and entering another, together — is a command.

type Jars map[string]int // jar -> how much it holds

type Filled struct {
	Jar string
	N   int
}

func (Filled) Type() string { return "filled" }

func (ev Filled) Apply(j Jars) Jars { j[ev.Jar] += ev.N; return j }

type Emptied struct {
	Jar string
	N   int
}

func (Emptied) Type() string { return "emptied" }

func (ev Emptied) Apply(j Jars) Jars { j[ev.Jar] -= ev.N; return j }

type Pour struct {
	From, To string
	N        int
}

type kitchen struct {
	e        *Engine
	jars     State[Jars]
	capacity int
	heard    []Filled
}

func (k *kitchen) fits(ev Filled) bool   { return k.jars.Get()[ev.Jar]+ev.N <= k.capacity }
func (k *kitchen) holds(ev Emptied) bool { return k.jars.Get()[ev.Jar] >= ev.N }
func (k *kitchen) hear(ev Filled)        { k.heard = append(k.heard, ev) }

func (k *kitchen) pour(c Pour) []Event {
	if c.N <= 0 {
		return nil
	}
	return []Event{Emptied{c.From, c.N}, Filled{c.To, c.N}}
}

func newKitchen(capacity int) *kitchen {
	e := NewEngine()
	k := &kitchen{e: e, capacity: capacity, jars: NewState(e, "jars", func() Jars { return Jars{} })}

	Command[Pour](e).Decides(k.pour)

	On[Filled](e).Requires(k.fits).Then(k.hear)
	On[Emptied](e).Requires(k.holds)
	return k
}

func TestOn_TheEventTypeComesFromTheEvent(t *testing.T) {
	k := newKitchen(3)

	assert.True(t, k.e.Emit(Filled{"a", 2}))
	assert.False(t, k.e.Emit(Filled{"a", 2}), "a rule given as a plain method")
	assert.Equal(t, []Filled{{"a", 2}}, k.heard, "and a listener")
}

func TestOn_EventsComeBackFromJSONAsTheyWent(t *testing.T) {
	k := newKitchen(3)
	k.e.Emit(Filled{"a", 2})
	k.e.Emit(Emptied{"a", 1})

	data, err := k.e.MarshalEvents(k.e.GetEvents())
	assert.NoError(t, err)
	events, err := k.e.UnmarshalEvents(data)

	assert.NoError(t, err)
	assert.Equal(t, []Event{Filled{"a", 2}, Emptied{"a", 1}}, events)
}

func TestApply_AnEventSaysWhatItDoesToItsState(t *testing.T) {
	k := newKitchen(3)

	k.e.Emit(Filled{"a", 3})
	k.e.Emit(Emptied{"a", 1})

	assert.Equal(t, Jars{"a": 2}, k.jars.Get())
}

func TestApply_UpdatesIsForTheStatesThatAlsoCare(t *testing.T) {
	k := newKitchen(3)
	fills := NewState(k.e, "fills", func() int { return 0 })
	On[Filled](k.e).Updates(Reduces(fills, func(n int, _ Filled) int { return n + 1 }))

	k.e.Emit(Filled{"a", 1})
	k.e.Emit(Filled{"b", 1})

	assert.Equal(t, 2, fills.Get())
	assert.Equal(t, Jars{"a": 1, "b": 1}, k.jars.Get())
}

func TestApply_AnEventThatAppliesItselfCantAlsoBeReducedIntoTheSameState(t *testing.T) {
	k := newKitchen(3)

	assert.Panics(t, func() {
		Reduces(k.jars, func(j Jars, ev Filled) Jars { return j })
	})
}

func TestCommand_WhatItDecidesHappensTogether(t *testing.T) {
	k := newKitchen(3)
	k.e.Emit(Filled{"a", 2})
	k.e.Emit(Filled{"b", 2})

	assert.True(t, k.e.Do(Pour{"a", "b", 1}))
	assert.Equal(t, Jars{"a": 1, "b": 3}, k.jars.Get())

	assert.False(t, k.e.Do(Pour{"a", "b", 1}), "b is full, so a keeps its water")
	assert.Equal(t, Jars{"a": 1, "b": 3}, k.jars.Get())
}

func TestCommand_DecidingNothingIsDoingNothing(t *testing.T) {
	k := newKitchen(3)
	k.e.Emit(Filled{"a", 2})

	assert.False(t, k.e.Do(Pour{"a", "b", 0}))
	assert.Len(t, k.e.GetEvents(), 1)
}

func TestCommand_AnUndeclaredCommandIsAWiringMistake(t *testing.T) {
	k := newKitchen(3)

	assert.Panics(t, func() { k.e.Do(struct{ Stir bool }{true}) })
}
