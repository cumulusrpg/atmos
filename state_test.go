package atmos

import (
	"testing"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/stretchr/testify/assert"
)

// The shelves' events write into the map they're given (see
// batch_test.go): every one of these is about the engine never letting
// that leak into what didn't happen.

// countPuts adds a state to s that counts every put it's reduced from,
// and how many times its reducer is called.
func countPuts(s *shelves, calls *int) State[int] {
	puts := NewState(s.e, "puts", func() int { return 0 })
	On[ItemPutEvent](s.e).Updates(Reduces(puts, func(n int, _ ItemPutEvent) int { *calls++; return n + 1 }))
	return puts
}

func TestState_EachEventIsReducedOnce(t *testing.T) {
	s := shelfEngine()
	var calls int
	puts := countPuts(s, &calls)

	s.e.Emit(ItemPutEvent{"a", "cup"})
	s.e.Emit(ItemPutEvent{"b", "jar"})
	s.e.Emit(ItemTakenEvent{"a", "cup"})
	for range 5 {
		puts.Get()
	}

	assert.Equal(t, 2, calls, "reading the state doesn't replay the log")
	assert.Equal(t, Shelves{"b": "jar"}, s.state.Get())
}

func TestState_ARefusedBatchLeavesTheStateAsItWas(t *testing.T) {
	s := shelfEngine()
	s.e.Emit(ItemPutEvent{"a", "cup"})
	s.e.Emit(ItemPutEvent{"b", "jar"})

	// Taking the cup is staged, and written into the state, before
	// putting it on b is refused.
	ok := s.e.EmitAll(ItemTakenEvent{"a", "cup"}, ItemPutEvent{"b", "cup"})

	assert.False(t, ok)
	assert.Equal(t, Shelves{"a": "cup", "b": "jar"}, s.state.Get())
}

func TestState_ARefusedNestedEmitTakesBackOnlyItsOwnEvents(t *testing.T) {
	s := shelfEngine()
	s.e.Emit(ItemPutEvent{"c", "plate"})
	On[ItemPutEvent](s.e).Before(func(ev ItemPutEvent) {
		if ev.Item == "cup" {
			s.e.EmitAll(ItemPutEvent{"d", "saucer"}, ItemPutEvent{"c", "spoon"}) // c is taken
		}
	})

	ok := s.e.Emit(ItemPutEvent{"a", "cup"})

	assert.True(t, ok)
	assert.Equal(t, Shelves{"a": "cup", "c": "plate"}, s.state.Get(), "no saucer")
}

func TestState_EveryReadStartsFromAFreshInitialState(t *testing.T) {
	s := shelfEngine()
	s.e.Emit(ItemPutEvent{"a", "cup"})

	// Replacing the log rebuilds from the initial state, which the
	// events above wrote into the first time round.
	s.e.SetEvents([]Event{ItemPutEvent{"b", "jar"}})

	assert.Equal(t, Shelves{"b": "jar"}, s.state.Get())
}

func TestState_ARepositoryThatAlreadyHasEventsIsReducedOnFirstRead(t *testing.T) {
	repo := repository.NewInMemory()
	NewEngine(WithRepository(repo)).Emit(ItemPutEvent{"a", "cup"})

	s := shelfEngine(WithRepository(repo))
	var calls int
	puts := countPuts(s, &calls)

	assert.Equal(t, Shelves{"a": "cup"}, s.state.Get())
	puts.Get()
	assert.Equal(t, 1, calls)
}

func TestState_AnUpdateAddedLaterSeesTheWholeLog(t *testing.T) {
	s := shelfEngine()
	puts := NewState(s.e, "puts", func() int { return 0 })
	s.e.Emit(ItemPutEvent{"a", "cup"})
	s.e.Emit(ItemPutEvent{"b", "jar"})
	puts.Get()

	On[ItemPutEvent](s.e).Updates(Reduces(puts, func(n int, _ ItemPutEvent) int { return n + 1 }))

	assert.Equal(t, 2, puts.Get())
}
