package atmos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A pantry, wired the way an app would: the struct that wires the
// engine holds its state handle and its configuration, and its rules
// are methods on it.

type Stocked struct{ Item string }

func (Stocked) Type() string { return "stocked" }

type Used struct{ Item string }

func (Used) Type() string { return "used" }

type pantry struct {
	shelf    State[[]string]
	capacity int
}

func (p *pantry) hasRoom(Stocked) bool { return len(p.shelf.Get()) < p.capacity }

func newPantry(capacity int) *pantry {
	e := NewEngine()
	p := &pantry{shelf: NewState(e, "shelf", func() []string { return nil }), capacity: capacity}
	e.When("stocked").
		Requires(Rule(p.hasRoom)).
		Updates(Reduces(p.shelf, func(s []string, ev Stocked) []string { return append(s, ev.Item) }))
	e.When("used").
		Updates(Reduces(p.shelf, func(s []string, ev Used) []string {
			for i, item := range s {
				if item == ev.Item {
					return append(s[:i], s[i+1:]...)
				}
			}
			return s
		}))
	return p
}

func (p *pantry) emit(events ...Event) bool { return p.shelf.e.EmitAll(events...) }

func TestState_AHandleReadsItsStateTyped(t *testing.T) {
	p := newPantry(3)

	p.emit(Stocked{"flour"}, Stocked{"salt"}, Used{"flour"})

	assert.Equal(t, []string{"salt"}, p.shelf.Get())
}

func TestState_AHandlesNameIsItsStatesName(t *testing.T) {
	p := newPantry(3)
	p.emit(Stocked{"flour"})

	assert.Equal(t, "shelf", p.shelf.Name())
	assert.Equal(t, []string{"flour"}, p.shelf.e.GetState("shelf"))
}

func TestRule_AMethodIsARule(t *testing.T) {
	p := newPantry(1)

	assert.True(t, p.emit(Stocked{"flour"}))
	assert.False(t, p.emit(Stocked{"salt"}), "the capacity reaches the rule through its receiver")
	assert.Equal(t, []string{"flour"}, p.shelf.Get())
}

func TestState_HandlesAreBoundToTheirEngine(t *testing.T) {
	small, big := newPantry(1), newPantry(5)

	small.emit(Stocked{"flour"})
	big.emit(Stocked{"salt"}, Stocked{"sugar"})

	assert.Equal(t, []string{"flour"}, small.shelf.Get())
	assert.Equal(t, []string{"salt", "sugar"}, big.shelf.Get())
}
