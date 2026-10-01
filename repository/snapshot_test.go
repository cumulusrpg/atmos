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

	counter := atmos.NewState(engine, "counter", func() int { return 0 })
	atmos.On[SimpleEvent](engine).Updates(atmos.Reduces(counter, func(n int, ev SimpleEvent) int { return n + ev.Value }))

	// Emit some events
	engine.Emit(SimpleEvent{Value: 1})
	engine.Emit(SimpleEvent{Value: 2})

	// Verify initial state
	if count := counter.Get(); count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// When: We restore from a different event log (simulating reload from persistence)
	restoredEvents := []atmos.Event{
		SimpleEvent{Value: 10},
		SimpleEvent{Value: 20},
		SimpleEvent{Value: 30},
	}
	engine.SetEvents(restoredEvents)

	// Then: State should reflect the restored events
	if count := counter.Get(); count != 60 {
		t.Errorf("expected count 60 after restore, got %d", count)
	}

	// And: GetEvents should return the restored events
	events := engine.GetEvents()
	if len(events) != 3 {
		t.Errorf("expected 3 events after restore, got %d", len(events))
	}
}
