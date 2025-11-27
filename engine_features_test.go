package atmos

import (
	"testing"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/stretchr/testify/assert"
)

// Test events for engine features
type OrderPlacedEvent struct {
	OrderID string
	Amount  float64
}

func (e OrderPlacedEvent) Type() string { return "order_placed" }

type InvoiceGeneratedEvent struct {
	OrderID   string
	InvoiceID string
}

func (e InvoiceGeneratedEvent) Type() string { return "invoice_generated" }

type PaymentValidatedEvent struct {
	OrderID string
}

func (e PaymentValidatedEvent) Type() string { return "payment_validated" }

// TestBeforeHooks demonstrates before hooks running after validation but before commit
func TestBeforeHooks(t *testing.T) {
	engine := NewEngine()

	// Track emitted events
	var emittedEvents []Event

	// Register a before hook that generates an invoice number
	engine.When("order_placed").
		Before(NewTypedListener(
			TypedListenerFunc[OrderPlacedEvent](func(e *Engine, event OrderPlacedEvent) {
				// Generate invoice as part of the transaction
				e.Emit(InvoiceGeneratedEvent{
					OrderID:   event.OrderID,
					InvoiceID: "INV-" + event.OrderID,
				})
			}),
		))

	// Register a listener to track all events
	engine.RegisterListener("order_placed", NewTypedListener(
		TypedListenerFunc[OrderPlacedEvent](func(e *Engine, event OrderPlacedEvent) {
			emittedEvents = append(emittedEvents, event)
		}),
	))
	engine.RegisterListener("invoice_generated", NewTypedListener(
		TypedListenerFunc[InvoiceGeneratedEvent](func(e *Engine, event InvoiceGeneratedEvent) {
			emittedEvents = append(emittedEvents, event)
		}),
	))

	// Emit order
	engine.Emit(OrderPlacedEvent{
		OrderID: "ORD-123",
		Amount:  99.99,
	})

	// Verify both events were emitted
	// Before hook emits invoice BEFORE the original event is committed
	events := engine.GetEvents()
	assert.Equal(t, 2, len(events), "Should have invoice_generated + order_placed")
	assert.Equal(t, "invoice_generated", events[0].Type(), "Before hook event comes first")
	assert.Equal(t, "order_placed", events[1].Type(), "Original event comes second")

	// Verify invoice has correct order ID
	invoice := events[0].(InvoiceGeneratedEvent)
	assert.Equal(t, "ORD-123", invoice.OrderID)
	assert.Equal(t, "INV-ORD-123", invoice.InvoiceID)
}

// RequirePaymentValidator rejects all orders (for testing exceptions)
type RequirePaymentValidator struct{}

func (v RequirePaymentValidator) ValidateTyped(e *Engine, event OrderPlacedEvent) bool {
	// Normally require payment validation - always fail
	return false
}

// TestValidatorExceptions demonstrates conditional validator skipping
func TestValidatorExceptions(t *testing.T) {
	engine := NewEngine()

	// Create validator instance
	requirePayment := NewTypedValidator(RequirePaymentValidator{})

	// Register validator with exception for free orders
	engine.When("order_placed").
		Requires(requirePayment).
		Except(requirePayment, func(e *Engine, event Event) bool {
			// Skip payment validation for free orders
			order := event.(OrderPlacedEvent)
			return order.Amount == 0.0
		}, "Free orders don't require payment validation")

	// Test 1: Paid order should fail validation
	success := engine.Emit(OrderPlacedEvent{
		OrderID: "ORD-001",
		Amount:  99.99,
	})
	assert.False(t, success, "Paid order should fail payment validation")

	// Test 2: Free order should skip validation and succeed
	success = engine.Emit(OrderPlacedEvent{
		OrderID: "ORD-002",
		Amount:  0.0,
	})
	assert.True(t, success, "Free order should skip payment validation")

	// Verify only free order was committed
	events := engine.GetEvents()
	assert.Equal(t, 1, len(events))
	assert.Equal(t, "ORD-002", events[0].(OrderPlacedEvent).OrderID)
}

// TestServiceLocator demonstrates registering and retrieving services
func TestServiceLocator(t *testing.T) {
	engine := NewEngine()

	// Create a product catalog service
	type ProductCatalog struct {
		Products map[string]float64
	}

	catalog := &ProductCatalog{
		Products: map[string]float64{
			"WIDGET-1": 19.99,
			"WIDGET-2": 29.99,
		},
	}

	// Register service
	engine.RegisterService("catalog", catalog)

	// Retrieve service
	retrievedCatalog := engine.GetService("catalog").(*ProductCatalog)

	// Verify it's the same instance
	assert.NotNil(t, retrievedCatalog)
	assert.Equal(t, 19.99, retrievedCatalog.Products["WIDGET-1"])
	assert.Equal(t, 29.99, retrievedCatalog.Products["WIDGET-2"])

	// Verify nil for non-existent service
	nonExistent := engine.GetService("does-not-exist")
	assert.Nil(t, nonExistent)
}

// TestMarshalUnmarshalEvents demonstrates JSON serialization of events
func TestMarshalUnmarshalEvents(t *testing.T) {
	engine := NewEngine()

	// Register event factories for deserialization (must return pointers)
	engine.When("order_placed", func() Event {
		return &OrderPlacedEvent{}
	})
	engine.When("invoice_generated", func() Event {
		return &InvoiceGeneratedEvent{}
	})

	// Create some events (use pointers to match factory return type)
	originalEvents := []Event{
		&OrderPlacedEvent{
			OrderID: "ORD-123",
			Amount:  99.99,
		},
		&InvoiceGeneratedEvent{
			OrderID:   "ORD-123",
			InvoiceID: "INV-123",
		},
	}

	// Marshal to JSON
	jsonData, err := engine.MarshalEvents(originalEvents)
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)

	// Unmarshal back to events
	unmarshaled, err := engine.UnmarshalEvents(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(unmarshaled))

	// Verify first event
	order := unmarshaled[0].(*OrderPlacedEvent)
	assert.Equal(t, "ORD-123", order.OrderID)
	assert.Equal(t, 99.99, order.Amount)

	// Verify second event
	invoice := unmarshaled[1].(*InvoiceGeneratedEvent)
	assert.Equal(t, "ORD-123", invoice.OrderID)
	assert.Equal(t, "INV-123", invoice.InvoiceID)
}

// TestMarshalUnmarshalRoundTrip demonstrates full persistence cycle
func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	// Create engine and emit some events
	engine1 := NewEngine()
	engine1.When("order_placed", func() Event { return &OrderPlacedEvent{} })

	engine1.Emit(&OrderPlacedEvent{OrderID: "ORD-1", Amount: 10.0})
	engine1.Emit(&OrderPlacedEvent{OrderID: "ORD-2", Amount: 20.0})
	engine1.Emit(&OrderPlacedEvent{OrderID: "ORD-3", Amount: 30.0})

	// Save events to JSON
	events := engine1.GetEvents()
	jsonData, err := engine1.MarshalEvents(events)
	assert.NoError(t, err)

	// Create new engine and restore from JSON
	engine2 := NewEngine()
	engine2.When("order_placed", func() Event { return &OrderPlacedEvent{} })

	restoredEvents, err := engine2.UnmarshalEvents(jsonData)
	assert.NoError(t, err)

	engine2.SetEvents(restoredEvents)

	// Verify same events
	finalEvents := engine2.GetEvents()
	assert.Equal(t, 3, len(finalEvents))
	assert.Equal(t, "ORD-1", finalEvents[0].(*OrderPlacedEvent).OrderID)
	assert.Equal(t, "ORD-2", finalEvents[1].(*OrderPlacedEvent).OrderID)
	assert.Equal(t, "ORD-3", finalEvents[2].(*OrderPlacedEvent).OrderID)
}

// TypedValidatorFunc is a helper for creating validators from functions
type TypedValidatorFunc[T Event] func(*Engine, T) bool

func (f TypedValidatorFunc[T]) ValidateTyped(engine *Engine, event T) bool {
	return f(engine, event)
}

// TypedListenerFunc is a helper for creating listeners from functions
type TypedListenerFunc[T Event] func(*Engine, T)

func (f TypedListenerFunc[T]) HandleTyped(engine *Engine, event T) {
	f(engine, event)
}

// TestSnapshotWithNonSnapshotRepository verifies snapshot methods handle non-snapshot repos gracefully
func TestSnapshotWithNonSnapshotRepository(t *testing.T) {
	// Default engine uses InMemory which doesn't support snapshots
	engine := NewEngine()

	// SetSnapshot should return error
	err := engine.SetSnapshot("test", map[string]int{"score": 100})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support snapshots")

	// ClearSnapshot should return error
	err = engine.ClearSnapshot("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support snapshots")

	// HasSnapshot should return false (not panic)
	hasSnapshot := engine.HasSnapshot("test")
	assert.False(t, hasSnapshot)
}

// TestSnapshotMergeWithInvalidJSON verifies mergeSnapshot handles invalid JSON gracefully
func TestSnapshotMergeWithInvalidJSON(t *testing.T) {
	engine := NewEngine()

	type SimpleState struct {
		Value int
	}

	initialState := SimpleState{Value: 42}

	// Invalid JSON should return the initial state unchanged
	result := engine.mergeSnapshot(initialState, []byte("not valid json"))
	assert.Equal(t, initialState, result)
}

// TestSnapshotMergeWithPointerState verifies mergeSnapshot handles pointer initial states
func TestSnapshotMergeWithPointerState(t *testing.T) {
	engine := NewEngine()

	type SimpleState struct {
		Value int
	}

	initialState := &SimpleState{Value: 42}

	// Should merge correctly even when initial state is a pointer
	result := engine.mergeSnapshot(initialState, []byte(`{"Value": 100}`))
	resultState := result.(SimpleState)
	assert.Equal(t, 100, resultState.Value)
}

// TestSnapshotMergeWithUnmarshalableState verifies mergeSnapshot handles unmarshalable initial state
func TestSnapshotMergeWithUnmarshalableState(t *testing.T) {
	engine := NewEngine()

	// A struct with a channel field cannot be marshaled
	type UnmarshalableState struct {
		Ch chan int
	}

	initialState := UnmarshalableState{Ch: make(chan int)}

	// Should return initial state unchanged when marshal fails
	result := engine.mergeSnapshot(initialState, []byte(`{}`))
	assert.Equal(t, initialState, result)
}

// TestSetSnapshotWithUnmarshalableData verifies SetSnapshot handles unmarshalable data
func TestSetSnapshotWithUnmarshalableData(t *testing.T) {
	engine := NewEngine(WithRepository(repository.NewInMemorySnapshot()))

	// Channels cannot be JSON marshaled
	unmarshalable := make(chan int)
	err := engine.SetSnapshot("test", unmarshalable)
	assert.Error(t, err)
}
