package atmos

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A shelf holds at most one item. Putting needs an empty shelf; taking
// needs the item to be there.

type ItemPutEvent struct{ Shelf, Item string }

func (ItemPutEvent) Type() string { return "item_put" }

// Apply writes into the map it's given, the way Go code naturally
// does: the engine can't count on it copying first.
func (ev ItemPutEvent) Apply(s Shelves) Shelves { s[ev.Shelf] = ev.Item; return s }

type ItemTakenEvent struct{ Shelf, Item string }

func (ItemTakenEvent) Type() string { return "item_taken" }

func (ev ItemTakenEvent) Apply(s Shelves) Shelves { delete(s, ev.Shelf); return s }

type Shelves map[string]string // shelf -> item; absent means empty

type shelves struct {
	e     *Engine
	state State[Shelves]
}

func (s *shelves) isEmpty(ev ItemPutEvent) bool {
	_, taken := s.state.Get()[ev.Shelf]
	return !taken
}

func (s *shelves) holdsItem(ev ItemTakenEvent) bool { return s.state.Get()[ev.Shelf] == ev.Item }

func shelfEngine(opts ...EngineOption) *shelves {
	e := NewEngine(opts...)
	s := &shelves{e: e, state: NewState(e, "shelves", func() Shelves { return Shelves{} })}
	On[ItemPutEvent](e).Requires(s.isEmpty)
	On[ItemTakenEvent](e).Requires(s.holdsItem)
	return s
}

func TestEmitAll_CommitsEveryEventInOrder(t *testing.T) {
	s := shelfEngine()
	engine := s.e

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "jar"})

	assert.True(t, ok)
	assert.Equal(t, []Event{ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "jar"}}, engine.GetEvents())
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, s.state.Get())
}

func TestEmitAll_EachEventIsValidatedAfterTheOnesBeforeIt(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"a", "cup"})

	// Alone, putting a jar on shelf a is refused: it's full. After the
	// cup is taken, in the same batch, it's allowed.
	assert.False(t, engine.Emit(ItemPutEvent{"a", "jar"}))
	ok := engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"a", "jar"})

	assert.True(t, ok)
	assert.Equal(t, Shelves{"a": "jar"}, s.state.Get())
}

func TestEmitAll_OneRefusalAndNothingHappens(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"a", "cup"})
	var heard []Event
	On[ItemTakenEvent](engine).Then(func(ev ItemTakenEvent) { heard = append(heard, ev) })
	engine.Emit(ItemPutEvent{"b", "jar"})
	before := engine.GetEvents()

	// Moving the cup onto a shelf that isn't empty: taking it is fine,
	// putting it down isn't — so it isn't taken either.
	ok := engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, before, engine.GetEvents(), "the log is untouched")
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, s.state.Get(), "so is the state")
	assert.Empty(t, heard, "and no listener heard the refused batch")
}

func TestEmitAll_ListenersRunOnlyOnceTheWholeBatchIsIn(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"a", "cup"})
	var seen Shelves
	On[ItemTakenEvent](engine).Then(func(ItemTakenEvent) { seen = s.state.Get() })

	engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.Equal(t, Shelves{"b": "cup"}, seen, "the listener for the first event sees the second one too")
}

func TestEmitAll_ABeforeHooksEmitsBelongToTheBatch(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"b", "jar"})
	// Putting a cup on a also puts a saucer on c, as part of the same act.
	On[ItemPutEvent](engine).Before(func(ev ItemPutEvent) {
		if ev.Item == "cup" {
			engine.Emit(ItemPutEvent{"c", "saucer"})
		}
	})
	before := engine.GetEvents()

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, before, engine.GetEvents(), "the saucer goes with the refused batch")
	assert.False(t, slices.ContainsFunc(engine.GetEvents(), func(e Event) bool { return e == ItemPutEvent{"c", "saucer"} }))
}

func TestEmitAll_ARefusedNestedEmitDropsOnlyItsOwnEvents(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"c", "plate"})
	var nested bool
	On[ItemPutEvent](engine).Before(func(ev ItemPutEvent) {
		if ev.Item == "cup" {
			nested = engine.Emit(ItemPutEvent{"c", "saucer"}) // c is taken
		}
	})

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"})

	assert.True(t, ok, "the hook's refusal is the hook's business")
	assert.False(t, nested)
	assert.Equal(t, Shelves{"a": "cup", "c": "plate"}, s.state.Get())
}

func TestEmitAll_AListenersEmitIsItsOwnAct(t *testing.T) {
	s := shelfEngine()
	engine := s.e
	engine.Emit(ItemPutEvent{"b", "jar"})
	On[ItemPutEvent](engine).Then(func(ev ItemPutEvent) {
		if ev.Item == "cup" {
			engine.Emit(ItemPutEvent{"b", "lid"}) // refused: b is taken
		}
	})

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"})

	assert.True(t, ok, "the batch was already in when the listener ran")
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, s.state.Get())
}
