# Atmos

```
╔═╗╔╦╗╔╦╗╔═╗╔═╗
╠═╣ ║ ║║║║ ║╚═╗
╩ ╩ ╩ ╩ ╩╚═╝╚═╝
```

[![Go Report Card](https://goreportcard.com/badge/github.com/cumulusrpg/atmos)](https://goreportcard.com/report/github.com/cumulusrpg/atmos)
[![CI](https://github.com/cumulusrpg/atmos/workflows/CI/badge.svg)](https://github.com/cumulusrpg/atmos/actions)
[![codecov](https://codecov.io/gh/cumulusrpg/atmos/branch/main/graph/badge.svg)](https://codecov.io/gh/cumulusrpg/atmos)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cumulusrpg/atmos)](https://github.com/cumulusrpg/atmos/blob/main/go.mod)
[![License: LGPL v3](https://img.shields.io/badge/License-LGPL%20v3-blue.svg)](https://www.gnu.org/licenses/lgpl-3.0)

**Event-driven architecture for Go that makes complex business logic simple, testable, and auditable.**

Atmos brings event sourcing patterns to application development with a fluent API that reads like English. Build systems where every action is recorded, every rule is explicit, and the complete history is preserved.

## The Problem

Traditional applications mix business rules, state management, and side effects together:

```go
func ProcessOrder(order Order) error {
    // Validation mixed with logic
    if order.Quantity <= 0 {
        return errors.New("invalid quantity")
    }

    // Direct state mutation
    inventory -= order.Quantity

    // Side effects scattered throughout
    sendEmail(order.CustomerEmail)
    updateDatabase(order)
    notifyWarehouse(order)

    // No audit trail
    // No way to replay what happened
    // Hard to test individual pieces
    return nil
}
```

This becomes unmaintainable as complexity grows. When something goes wrong, you can't tell what happened or why.

## The Atmos Way

Atmos separates concerns and makes every rule explicit:

```go
engine.When("order_placed", func() atmos.Event { return &OrderPlacedEvent{} }).
    Requires(
        atmos.Valid(&SufficientInventory{}),
        atmos.Valid(&ValidCustomer{}),
    ).
    Then(
        atmos.Do(&ReserveInventory{}),
        atmos.Do(&NotifyWarehouse{}),
        atmos.Do(&SendConfirmationEmail{}),
    ).
    Updates("orders", ReduceOrderPlaced).
    Updates("inventory", ReduceInventoryReserved)
```

**Every rule is visible.** No hidden logic. No surprises.

## Why Event Sourcing?

### Complete Audit Trail

Every action is recorded as an immutable event:

```go
events := engine.GetEvents()
// [OrderPlacedEvent, PaymentProcessedEvent, OrderShippedEvent, ...]
```

You can answer questions like:
- "Why did this order fail?"
- "Who changed this setting?"
- "What was the state at 3pm yesterday?"

### Time Travel

Replay state at any point in history:

```go
// Rebuild state from the first 100 events
engine.SetEvents(events[:100])
state := engine.GetState("orders")
```

Perfect for:
- Debugging production issues
- Analyzing historical trends
- Testing "what if" scenarios

### Testability

Every component is isolated and pure:

```go
func TestSufficientInventory(t *testing.T) {
    validator := SufficientInventory{}
    engine := setupTestEngine()

    event := OrderPlacedEvent{ProductID: "ABC", Quantity: 5}
    assert.True(t, validator.ValidateTyped(engine, event))
}
```

No mocks needed. No database required. Just pure functions.

### Flexibility

Add new features without changing existing code:

```go
// New requirement: send SMS on high-value orders
engine.When("order_placed").
    Then(atmos.Do(&SendSMSForHighValue{}))  // Just add it!
```

The open/closed principle in action.

## Installation

```bash
go get github.com/cumulusrpg/atmos
```

## Quick Start

Here's a simple inventory system in ~30 lines:

```go
package main

import (
    "github.com/cumulusrpg/atmos"
    "time"
)

// 1. Define events (immutable facts)
type ItemAddedEvent struct {
    ItemID   string
    Quantity int
}

func (e ItemAddedEvent) Type() string { return "item_added" }

// 2. Define state
type InventoryState struct {
    Items map[string]int
}

// 3. Create validators (business rules)
type PositiveQuantity struct{}

func (v *PositiveQuantity) ValidateTyped(engine *atmos.Engine, event ItemAddedEvent) bool {
    return event.Quantity > 0
}

// 4. Create reducers (state changes)
func ReduceItemAdded(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
    s := state.(InventoryState)
    e := event.(ItemAddedEvent)
    s.Items[e.ItemID] += e.Quantity
    return s
}

// 5. Wire it together with the fluent API
func NewInventorySystem() *atmos.Engine {
    engine := atmos.NewEngine()

    engine.RegisterState("inventory", InventoryState{Items: make(map[string]int)})

    engine.When("item_added", func() atmos.Event { return &ItemAddedEvent{} }).
        Requires(atmos.Valid(&PositiveQuantity{})).
        Updates("inventory", ReduceItemAdded)

    return engine
}

// 6. Use it
func main() {
    system := NewInventorySystem()

    system.Emit(ItemAddedEvent{
        ItemID:   "WIDGET-001",
        Quantity: 100,
        Time:     time.Now(),
    })

    state := system.GetState("inventory").(InventoryState)
    fmt.Printf("Inventory: %+v\n", state.Items)
    // Output: Inventory: map[WIDGET-001:100]
}
```

## Core Concepts

### Events

Events are **immutable facts** about what happened:

```go
type OrderPlacedEvent struct {
    OrderID    string
    CustomerID string
    Items      []OrderItem
    Total      float64
}

func (e OrderPlacedEvent) Type() string { return "order_placed" }
```

Events are:
- **Past tense** - "OrderPlaced" not "PlaceOrder"
- **Immutable** - Never changed after creation
- **Complete** - Contain all relevant data

### State

State is **derived from events** using pure reducer functions:

```go
func ReduceOrderPlaced(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
    s := state.(OrderState)
    e := event.(OrderPlacedEvent)

    s.Orders[e.OrderID] = Order{
        ID:         e.OrderID,
        CustomerID: e.CustomerID,
        Items:      e.Items,
        Total:      e.Total,
        Status:     "pending",
    }

    return s
}
```

State is never directly mutated. It's always recalculated from events.

### Validators

Validators **enforce business rules** before events commit:

```go
type SufficientInventory struct{}

func (v *SufficientInventory) ValidateTyped(engine *atmos.Engine, event OrderPlacedEvent) bool {
    inventory := engine.GetState("inventory").(InventoryState)

    for _, item := range event.Items {
        available := inventory.Items[item.ProductID]
        if available < item.Quantity {
            return false  // Not enough inventory
        }
    }

    return true
}
```

If **any** validator returns false, the event is rejected and nothing happens.

### Listeners

Listeners trigger **side effects** after events commit:

```go
type SendConfirmationEmail struct{}

func (l *SendConfirmationEmail) HandleTyped(engine *atmos.Engine, event OrderPlacedEvent) {
    customer := engine.GetState("customers").(CustomerState).Get(event.CustomerID)

    emailService.Send(EmailParams{
        To:      customer.Email,
        Subject: "Order Confirmation",
        Body:    fmt.Sprintf("Your order %s has been placed!", event.OrderID),
    })
}
```

Listeners run **after** the event is committed to the log. Use them for:
- External API calls
- Email/SMS notifications
- Database updates
- Emitting additional events

### Before Hooks

Before hooks run **after validation** but **before commitment**:

```go
engine.When("payment_processed").
    Before(atmos.Do(&GenerateInvoiceNumber{})).  // Runs as part of transaction
    Then(atmos.Do(&SendReceipt{}))               // Runs after commitment
```

Use before hooks when the side effect must be part of the same transaction (e.g., generating IDs, procedural content).

## The Fluent API

Chain methods to declare rules in one place:

```go
engine.When("order_placed", func() atmos.Event { return &OrderPlacedEvent{} }).
    Requires(
        atmos.Valid(&ValidCustomer{}),
        atmos.Valid(&SufficientInventory{}),
        atmos.Valid(&ValidPaymentMethod{}),
    ).
    Before(
        atmos.Do(&GenerateOrderNumber{}),
        atmos.Do(&CalculateTax{}),
    ).
    Then(
        atmos.Do(&ReserveInventory{}),
        atmos.Do(&ProcessPayment{}),
        atmos.Do(&SendConfirmationEmail{}),
        atmos.Do(&NotifyWarehouse{}),
    ).
    Updates("orders", ReduceOrderPlaced).
    Updates("inventory", ReduceInventoryReserved).
    Updates("payments", ReducePaymentProcessed)
```

**Everything about this event is visible in one declaration.**

### Available Methods

- `When(eventType, factory)` - Start declaring rules for an event
- `Requires(...validators)` - Add validation rules (all must pass)
- `Except(validator, condition, reason)` - Document exceptions to rules
- `Before(...hooks)` - Run before commit (transactional)
- `Then(...listeners)` - Run after commit (side effects)
- `Updates(stateName, reducer)` - Update state in response to event

## Advanced Features

### Validator Exceptions

Sometimes rules have exceptions. Document them explicitly:

```go
engine.When("order_placed").
    Requires(atmos.Valid(&RequirePaymentMethod{})).
    Except(
        atmos.Valid(&RequirePaymentMethod{}),
        func(e *atmos.Engine, event atmos.Event) bool {
            order := event.(OrderPlacedEvent)
            return order.Total == 0  // Free orders don't need payment
        },
        "Free orders don't require payment method",
    )
```

The `reason` string documents why the exception exists.

### Custom Event Repositories

By default, Atmos stores events in memory. For production use, implement a custom repository to persist events automatically:

```go
// Implement the EventRepository interface
type FileRepository struct {
    filepath string
}

func (r *FileRepository) Add(engine *atmos.Engine, event atmos.Event) error {
    // Serialize event using engine's registered event types
    jsonData, _ := engine.MarshalEvents([]atmos.Event{event})
    // Append to file atomically
    return appendToFile(r.filepath, jsonData)
}

func (r *FileRepository) GetAll(engine *atmos.Engine) []atmos.Event {
    // Load JSON from file
    jsonData := readFile(r.filepath)
    // Deserialize using engine's registered event types
    events, _ := engine.UnmarshalEvents(jsonData)
    return events
}

func (r *FileRepository) SetAll(engine *atmos.Engine, events []atmos.Event) error {
    // Serialize all events
    jsonData, _ := engine.MarshalEvents(events)
    // Replace file contents atomically
    return writeFile(r.filepath, jsonData)
}

// Use custom repository
repo := &FileRepository{filepath: "events.jsonl"}
engine := atmos.NewEngine(atmos.WithRepository(repo))

// Events are now automatically persisted on every Emit()
engine.Emit(OrderPlacedEvent{...})  // Saved to disk automatically!
```

Benefits:
- **Automatic persistence** - No manual save/load required
- **Pluggable storage** - File, database, cloud storage, etc.
- **Failure safety** - If `Add()` fails, the event is rejected
- **Simple interface** - Just three methods to implement

### Event Replay and Persistence

For manual persistence workflows, serialize events to JSON:

```go
// Get all events
events := engine.GetEvents()

// Serialize to JSON
jsonData, _ := engine.MarshalEvents(events)

// Save to database
db.Save("event_log", jsonData)

// Later: load and replay
jsonData := db.Load("event_log")
events, _ := engine.UnmarshalEvents(jsonData)

newEngine := atmos.NewEngine()
newEngine.SetEvents(events)

// State is now rebuilt from history
state := newEngine.GetState("orders")
```

Perfect for:
- Persisting application state
- Debugging production issues
- Migrating between versions
- Auditing and compliance

### Service Locator

Register reference data or utilities:

```go
// Register services
productCatalog := catalog.Load()
engine.RegisterService("catalog", productCatalog)

// Access in validators/listeners
catalog := engine.GetService("catalog").(*catalog.ProductCatalog)
product := catalog.GetProduct(productID)
```

Use for:
- Reference data (product catalogs, rate tables)
- External services (email, SMS)
- Configuration
- Shared utilities

### Multiple State Updates

One event can update multiple states:

```go
engine.When("order_placed").
    Updates("orders", ReduceOrderPlaced).
    Updates("inventory", ReduceInventoryReserved).
    Updates("customers", ReduceCustomerOrderCount).
    Updates("analytics", ReduceOrderMetrics)
```

Each reducer sees the event and updates its own state independently.

## Architecture

The event flow in Atmos:

```
User Action
    ↓
Event Created
    ↓
Validators Check ──→ [REJECT if any fail]
    ↓
Before Hooks Run (transactional)
    ↓
Event Committed to Repository ← [Point of no return]
    ↓
State Reducers Apply (pure functions)
    ↓
Listeners Run (side effects)
    ↓
Done
```

### Benefits

**Auditability**
- Every action is recorded
- Complete history of what happened
- Who, what, when, why for every change

**Time Travel**
- Replay state at any point
- Debug issues by rewinding
- Test "what if" scenarios

**Testability**
- Pure functions everywhere
- No mocks needed
- Fast, isolated unit tests

**Flexibility**
- Add features without changing code
- Rules are explicit and visible
- Easy to reason about complex logic

**Reliability**
- Events are immutable
- State is derived, never mutated
- Transactions are atomic

## Examples

See the [Tic-Tac-Toe example](examples/tictactoe) for a complete working game with tests.

## Contributing

Atmos is part of the [Cumulus RPG](https://github.com/cumulusrpg) project. Issues and PRs welcome!

## License

This project is part of the Cumulus RPG system.

---

**Built with Atmos? [Let us know!](https://github.com/cumulusrpg/atmos/discussions)**
