package atmos

import (
	"testing"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/stretchr/testify/assert"
)

// These reducers write into the map they're given, the way Go code
// naturally does: the engine can't count on a reducer copying first.

type Counted struct{ calls int }

func inPlaceShelfEngine(counted *Counted, opts ...EngineOption) *Engine {
	engine := NewEngine(opts...)
	engine.RegisterState("shelves", func() interface{} { return Shelves{} })
	engine.When("item_put").
		Requires(Valid(ShelfIsEmpty{})).
		WithReducer("shelves", func(_ *Engine, state interface{}, event Event) interface{} {
			counted.calls++
			s, ev := state.(Shelves), event.(ItemPutEvent)
			s[ev.Shelf] = ev.Item
			return s
		})
	engine.When("item_taken").
		Requires(Valid(ShelfHoldsItem{})).
		WithReducer("shelves", func(_ *Engine, state interface{}, event Event) interface{} {
			counted.calls++
			s, ev := state.(Shelves), event.(ItemTakenEvent)
			delete(s, ev.Shelf)
			return s
		})
	return engine
}

func TestState_EachEventIsReducedOnce(t *testing.T) {
	var counted Counted
	engine := inPlaceShelfEngine(&counted)

	engine.Emit(ItemPutEvent{"a", "cup"})
	engine.Emit(ItemPutEvent{"b", "jar"})
	engine.Emit(ItemTakenEvent{"a", "cup"})
	for range 5 {
		engine.GetState("shelves")
	}

	assert.Equal(t, 3, counted.calls, "reading the state doesn't replay the log")
	assert.Equal(t, Shelves{"b": "jar"}, engine.GetState("shelves"))
}

func TestState_ARefusedBatchLeavesTheStateAsItWas(t *testing.T) {
	engine := inPlaceShelfEngine(&Counted{})
	engine.Emit(ItemPutEvent{"a", "cup"})
	engine.Emit(ItemPutEvent{"b", "jar"})

	// Taking the cup is staged, and written into the state, before
	// putting it on b is refused.
	ok := engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, engine.GetState("shelves"))
}

func TestState_ARefusedNestedEmitTakesBackOnlyItsOwnEvents(t *testing.T) {
	engine := inPlaceShelfEngine(&Counted{})
	engine.Emit(ItemPutEvent{"c", "plate"})
	engine.When("item_put").Before(NewTypedListener(
		TypedListenerFunc[ItemPutEvent](func(e *Engine, ev ItemPutEvent) {
			if ev.Item == "cup" {
				e.EmitAll(ItemPutEvent{"d", "saucer"}, ItemPutEvent{"c", "spoon"}) // c is taken
			}
		})))

	ok := engine.Emit(ItemPutEvent{"a", "cup"})

	assert.True(t, ok)
	assert.Equal(t, Shelves{"a": "cup", "c": "plate"}, engine.GetState("shelves"), "no saucer")
}

func TestState_EveryReadStartsFromAFreshInitialState(t *testing.T) {
	engine := inPlaceShelfEngine(&Counted{})
	engine.Emit(ItemPutEvent{"a", "cup"})

	// Replacing the log rebuilds from the initial state, which the
	// reducers above wrote into the first time round.
	engine.SetEvents([]Event{ItemPutEvent{"b", "jar"}})

	assert.Equal(t, Shelves{"b": "jar"}, engine.GetState("shelves"))
}

func TestState_ARepositoryThatAlreadyHasEventsIsReducedOnFirstRead(t *testing.T) {
	repo := repository.NewInMemory()
	first := NewEngine(WithRepository(repo))
	first.Emit(ItemPutEvent{"a", "cup"})

	var counted Counted
	engine := inPlaceShelfEngine(&counted, WithRepository(repo))

	assert.Equal(t, Shelves{"a": "cup"}, engine.GetState("shelves"))
	engine.GetState("shelves")
	assert.Equal(t, 1, counted.calls)
}

func TestState_AReducerAddedLaterSeesTheWholeLog(t *testing.T) {
	engine := NewEngine()
	engine.RegisterState("puts", func() interface{} { return 0 })
	engine.Emit(ItemPutEvent{"a", "cup"})
	engine.Emit(ItemPutEvent{"b", "jar"})
	engine.GetState("puts")

	engine.When("item_put").WithReducer("puts", func(_ *Engine, state interface{}, _ Event) interface{} {
		return state.(int) + 1
	})

	assert.Equal(t, 2, engine.GetState("puts"))
}
