package atmos

import (
	"errors"
	"testing"

	"github.com/cumulusrpg/atmos/types"
)

// Test event for repository tests
type TestEvent struct {
	Name string
}

func (e TestEvent) Type() string {
	return "test_event"
}

// CustomRepository that tracks all adds and can fail on demand
type CustomRepository struct {
	events     []Event
	addCalls   int
	shouldFail bool
}

func (r *CustomRepository) Add(engine types.Engine, event types.Event) error {
	r.addCalls++
	if r.shouldFail {
		return errors.New("simulated repository failure")
	}
	r.events = append(r.events, event)
	return nil
}

func (r *CustomRepository) GetAll(engine types.Engine) []types.Event {
	return append([]types.Event{}, r.events...)
}

func (r *CustomRepository) SetAll(engine types.Engine, events []types.Event) error {
	r.events = append([]types.Event{}, events...)
	return nil
}

// TestCustomRepositoryInterceptsEvents verifies custom repositories can intercept events
func TestCustomRepositoryInterceptsEvents(t *testing.T) {
	customRepo := &CustomRepository{}
	engine := NewEngine(WithRepository(customRepo))

	// Emit an event
	event := TestEvent{Name: "test1"}
	success := engine.Emit(event)

	if !success {
		t.Fatal("Event should have been emitted successfully")
	}

	if customRepo.addCalls != 1 {
		t.Errorf("Expected 1 Add() call, got %d", customRepo.addCalls)
	}

	events := engine.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

// TestRepositoryFailureRejectsEvent verifies that Add() failures reject events
func TestRepositoryFailureRejectsEvent(t *testing.T) {
	customRepo := &CustomRepository{shouldFail: true}
	engine := NewEngine(WithRepository(customRepo))

	// Try to emit an event - should fail due to repository failure
	event := TestEvent{Name: "test1"}
	success := engine.Emit(event)

	if success {
		t.Fatal("Event should have been rejected due to repository failure")
	}

	if customRepo.addCalls != 1 {
		t.Errorf("Expected 1 Add() call attempt, got %d", customRepo.addCalls)
	}

	events := engine.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events (add failed), got %d", len(events))
	}
}

// TestRepositoryUsedForStateReplay verifies repository is used for state replay
func TestRepositoryUsedForStateReplay(t *testing.T) {
	customRepo := &CustomRepository{}
	engine := NewEngine(WithRepository(customRepo))

	// Register a simple state
	type CountState struct {
		Count int
	}

	engine.RegisterState("counter", CountState{Count: 0})
	engine.RegisterEventType("test_event", func() Event { return &TestEvent{} })

	// Register a reducer
	engine.When("test_event", func() Event { return &TestEvent{} }).
		Updates("counter", func(e *Engine, state interface{}, event Event) interface{} {
			s := state.(CountState)
			s.Count++
			return s
		})

	// Emit some events
	engine.Emit(TestEvent{Name: "event1"})
	engine.Emit(TestEvent{Name: "event2"})
	engine.Emit(TestEvent{Name: "event3"})

	// Verify state is built from repository
	state := engine.GetState("counter").(CountState)
	if state.Count != 3 {
		t.Errorf("Expected count 3, got %d", state.Count)
	}

	// Verify repository has all events
	if len(customRepo.events) != 3 {
		t.Errorf("Expected 3 events in repository, got %d", len(customRepo.events))
	}
}

// TestSetEventsWithCustomRepository verifies SetEvents works with custom repositories
func TestSetEventsWithCustomRepository(t *testing.T) {
	customRepo := &CustomRepository{}
	engine := NewEngine(WithRepository(customRepo))

	// Create some events
	events := []Event{
		TestEvent{Name: "event1"},
		TestEvent{Name: "event2"},
	}

	// Set events
	engine.SetEvents(events)

	// Verify repository has the events
	repoEvents := customRepo.GetAll(engine)
	if len(repoEvents) != 2 {
		t.Errorf("Expected 2 events in repository, got %d", len(repoEvents))
	}

	// Verify GetEvents returns the same events
	engineEvents := engine.GetEvents()
	if len(engineEvents) != 2 {
		t.Errorf("Expected 2 events from engine, got %d", len(engineEvents))
	}
}

// TestInMemoryRepositoryDefault verifies InMemoryRepository is used by default
func TestInMemoryRepositoryDefault(t *testing.T) {
	engine := NewEngine()

	event := TestEvent{Name: "test1"}
	success := engine.Emit(event)

	if !success {
		t.Fatal("Event should have been emitted successfully")
	}

	events := engine.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
