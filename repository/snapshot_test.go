package repository_test

import (
	"testing"

	"github.com/cumulusrpg/atmos"
	"github.com/cumulusrpg/atmos/repository"
)

// SimpleEvent for testing
type SimpleEvent struct {
	Value int
}

func (e SimpleEvent) Type() string { return "simple" }

// TestInMemorySnapshot_RestoreEventsFromLog verifies that SetAll can restore
// an engine's event log, which is useful for rebuilding state from persistence.
func TestInMemorySnapshot_RestoreEventsFromLog(t *testing.T) {
	// Given: A snapshot repository with some existing events and a snapshot
	repo := repository.NewInMemorySnapshot()
	engine := atmos.NewEngine(atmos.WithRepository(repo))

	type Counter struct {
		Count int
	}

	engine.RegisterState("counter", Counter{Count: 0})
	engine.When("simple").Updates("counter", func(e *atmos.Engine, state interface{}, event atmos.Event) interface{} {
		s := state.(Counter)
		s.Count += event.(SimpleEvent).Value
		return s
	})

	// Emit some events
	engine.Emit(SimpleEvent{Value: 1})
	engine.Emit(SimpleEvent{Value: 2})

	// Verify initial state
	state := engine.GetState("counter").(Counter)
	if state.Count != 3 {
		t.Fatalf("expected count 3, got %d", state.Count)
	}

	// When: We restore from a different event log (simulating reload from persistence)
	restoredEvents := []atmos.Event{
		SimpleEvent{Value: 10},
		SimpleEvent{Value: 20},
		SimpleEvent{Value: 30},
	}
	engine.SetEvents(restoredEvents)

	// Then: State should reflect the restored events
	state = engine.GetState("counter").(Counter)
	if state.Count != 60 {
		t.Errorf("expected count 60 after restore, got %d", state.Count)
	}

	// And: GetEvents should return the restored events
	events := engine.GetEvents()
	if len(events) != 3 {
		t.Errorf("expected 3 events after restore, got %d", len(events))
	}
}
