package atmos

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A shelf holds at most one item. Putting needs an empty shelf; taking
// needs the item to be there.

type ItemPutEvent struct{ Shelf, Item string }

func (e ItemPutEvent) Type() string { return "item_put" }

type ItemTakenEvent struct{ Shelf, Item string }

func (e ItemTakenEvent) Type() string { return "item_taken" }

type Shelves map[string]string // shelf -> item; absent means empty

type ShelfIsEmpty struct{}

func (ShelfIsEmpty) ValidateTyped(e *Engine, ev ItemPutEvent) bool {
	_, taken := e.GetState("shelves").(Shelves)[ev.Shelf]
	return !taken
}

type ShelfHoldsItem struct{}

func (ShelfHoldsItem) ValidateTyped(e *Engine, ev ItemTakenEvent) bool {
	return e.GetState("shelves").(Shelves)[ev.Shelf] == ev.Item
}

func shelfEngine() *Engine {
	engine := NewEngine()
	engine.RegisterState("shelves", func() interface{} { return Shelves{} })
	engine.When("item_put").
		Requires(Valid(ShelfIsEmpty{})).
		WithReducer("shelves", func(_ *Engine, state interface{}, event Event) interface{} {
			s, ev := clone(state.(Shelves)), event.(ItemPutEvent)
			s[ev.Shelf] = ev.Item
			return s
		})
	engine.When("item_taken").
		Requires(Valid(ShelfHoldsItem{})).
		WithReducer("shelves", func(_ *Engine, state interface{}, event Event) interface{} {
			s, ev := clone(state.(Shelves)), event.(ItemTakenEvent)
			delete(s, ev.Shelf)
			return s
		})
	return engine
}

func clone(s Shelves) Shelves {
	out := Shelves{}
	for k, v := range s {
		out[k] = v
	}
	return out
}

func TestEmitAll_CommitsEveryEventInOrder(t *testing.T) {
	engine := shelfEngine()

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "jar"})

	assert.True(t, ok)
	assert.Equal(t, []Event{ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "jar"}}, engine.GetEvents())
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, engine.GetState("shelves"))
}

func TestEmitAll_EachEventIsValidatedAfterTheOnesBeforeIt(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"a", "cup"})

	// Alone, putting a jar on shelf a is refused: it's full. After the
	// cup is taken, in the same batch, it's allowed.
	assert.False(t, engine.Emit(ItemPutEvent{"a", "jar"}))
	ok := engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"a", "jar"})

	assert.True(t, ok)
	assert.Equal(t, Shelves{"a": "jar"}, engine.GetState("shelves"))
}

func TestEmitAll_OneRefusalAndNothingHappens(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"a", "cup"})
	var heard []Event
	engine.RegisterListener("item_taken", NewTypedListener(
		TypedListenerFunc[ItemTakenEvent](func(_ *Engine, ev ItemTakenEvent) { heard = append(heard, ev) })))
	engine.Emit(ItemPutEvent{"b", "jar"})
	before := engine.GetEvents()

	// Moving the cup onto a shelf that isn't empty: taking it is fine,
	// putting it down isn't — so it isn't taken either.
	ok := engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, before, engine.GetEvents(), "the log is untouched")
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, engine.GetState("shelves"), "so is the state")
	assert.Empty(t, heard, "and no listener heard the refused batch")
}

func TestEmitAll_ListenersRunOnlyOnceTheWholeBatchIsIn(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"a", "cup"})
	var seen Shelves
	engine.RegisterListener("item_taken", NewTypedListener(
		TypedListenerFunc[ItemTakenEvent](func(e *Engine, _ ItemTakenEvent) { seen = e.GetState("shelves").(Shelves) })))

	engine.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.Equal(t, Shelves{"b": "cup"}, seen, "the listener for the first event sees the second one too")
}

func TestEmitAll_ABeforeHooksEmitsBelongToTheBatch(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"b", "jar"})
	// Putting a cup on a also puts a saucer on c, as part of the same act.
	engine.When("item_put").Before(NewTypedListener(
		TypedListenerFunc[ItemPutEvent](func(e *Engine, ev ItemPutEvent) {
			if ev.Item == "cup" {
				e.Emit(ItemPutEvent{"c", "saucer"})
			}
		})))
	before := engine.GetEvents()

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, before, engine.GetEvents(), "the saucer goes with the refused batch")
	assert.False(t, slices.ContainsFunc(engine.GetEvents(), func(e Event) bool { return e == ItemPutEvent{"c", "saucer"} }))
}

func TestEmitAll_ARefusedNestedEmitDropsOnlyItsOwnEvents(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"c", "plate"})
	var nested bool
	engine.When("item_put").Before(NewTypedListener(
		TypedListenerFunc[ItemPutEvent](func(e *Engine, ev ItemPutEvent) {
			if ev.Item == "cup" {
				nested = e.Emit(ItemPutEvent{"c", "saucer"}) // c is taken
			}
		})))

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"})

	assert.True(t, ok, "the hook's refusal is the hook's business")
	assert.False(t, nested)
	assert.Equal(t, Shelves{"a": "cup", "c": "plate"}, engine.GetState("shelves"))
}

func TestEmitAll_AListenersEmitIsItsOwnAct(t *testing.T) {
	engine := shelfEngine()
	engine.Emit(ItemPutEvent{"b", "jar"})
	engine.RegisterListener("item_put", NewTypedListener(
		TypedListenerFunc[ItemPutEvent](func(e *Engine, ev ItemPutEvent) {
			if ev.Item == "cup" {
				e.Emit(ItemPutEvent{"b", "lid"}) // refused: b is taken
			}
		})))

	ok := engine.EmitAll(ItemPutEvent{"a", "cup"})

	assert.True(t, ok, "the batch was already in when the listener ran")
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, engine.GetState("shelves"))
}
