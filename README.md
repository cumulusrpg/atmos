# Atmos

[![CI](https://github.com/cumulusrpg/atmos/workflows/CI/badge.svg)](https://github.com/cumulusrpg/atmos/actions)
[![License: LGPL v3](https://img.shields.io/badge/License-LGPL%20v3-blue.svg)](https://www.gnu.org/licenses/lgpl-3.0)

**Go** &nbsp;
[![Go Report Card](https://goreportcard.com/badge/github.com/cumulusrpg/atmos)](https://goreportcard.com/report/github.com/cumulusrpg/atmos)
[![codecov (Go)](https://codecov.io/gh/cumulusrpg/atmos/branch/main/graph/badge.svg?flag=go)](https://codecov.io/gh/cumulusrpg/atmos)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cumulusrpg/atmos)](https://github.com/cumulusrpg/atmos/blob/main/go.mod)

**JS** &nbsp;
[![codecov (JS)](https://codecov.io/gh/cumulusrpg/atmos/branch/main/graph/badge.svg?flag=js)](https://codecov.io/gh/cumulusrpg/atmos)

**An event-sourcing primitive that makes complex business logic simple, testable, and auditable.**

Atmos is a small, opinionated engine for event-sourced applications: events are immutable facts, reducers fold them into state, validators gate what can enter the log. Every action is recorded, every rule is explicit, and the complete history is preserved. The design lives in [DESIGN.md](./DESIGN.md).

## Implementations

Two runtimes of the same primitive. Shared semantics, independent test suites, and (eventually) a shared fixture-driven parity check.

| Runtime | Status | Location |
|---|---|---|
| **atmos (Go)** — full-featured, stable | in use | this repo, root |
| **atmos-js** — nascent, targets the model in [DESIGN.md](./DESIGN.md) | WIP | [`js/`](./js) |

Go gets the deep dive below. For the JS engine, see [`js/README.md`](./js/README.md).

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

Atmos separates concerns and makes every rule explicit. An engine's definition
reads as what the system does:

```go
// What can be done
atmos.Command[PlaceOrder](e).Decides(shop.placeOrder)

// What can happen, and when it's allowed
atmos.On[OrderPlaced](e).
    Requires(shop.validCustomer, shop.sufficientInventory).
    Then(shop.notifyWarehouse, shop.sendConfirmation)
```

**Every rule is visible.** No hidden logic. No surprises.

## Why Event Sourcing?

### Complete Audit Trail

Every action is recorded as an immutable event:

```go
events := engine.GetEvents()
// [OrderPlaced, PaymentProcessed, OrderShipped, ...]
```

You can answer questions like:
- "Why did this order fail?"
- "Who changed this setting?"
- "What was the state at 3pm yesterday?"

### Time Travel

Replay state at any point in history:

```go
// Rebuild state from the first 100 events
shop.engine.SetEvents(events[:100])
state := shop.orders.Get()
```

Perfect for:
- Debugging production issues
- Analyzing historical trends
- Testing "what if" scenarios

### Testability

Every component is isolated and pure:

```go
func TestSufficientInventory(t *testing.T) {
    shop := newTestShop()

    event := OrderPlaced{ProductID: "ABC", Quantity: 5}
    assert.True(t, shop.sufficientInventory(event))
}
```

No mocks needed. No database required. Just pure functions.

### Flexibility

Add new features without changing existing code:

```go
// New requirement: send SMS on high-value orders
atmos.On[OrderPlaced](e).
    Then(shop.smsIfHighValue)  // Just add it!
```

The open/closed principle in action.

## Installation

```bash
go get github.com/cumulusrpg/atmos
```

## Quick Start

Here's a simple inventory system:

```go
package main

import (
    "fmt"

    "github.com/cumulusrpg/atmos"
)

// 1. Define state
type Stock map[string]int // item -> how many

// 2. Define events (immutable facts), and what each does to the state
type ItemAdded struct {
    ItemID   string
    Quantity int
}

func (ItemAdded) Type() string { return "item_added" }

func (e ItemAdded) Apply(s Stock) Stock {
    s[e.ItemID] += e.Quantity
    return s
}

// 3. Hold the engine and its state handles; rules are methods
type Inventory struct {
    engine *atmos.Engine
    stock  atmos.State[Stock]
}

func (i *Inventory) positiveQuantity(e ItemAdded) bool {
    return e.Quantity > 0
}

// 4. Declare what can happen, and when it's allowed
func NewInventory() *Inventory {
    engine := atmos.NewEngine()
    i := &Inventory{
        engine: engine,
        stock:  atmos.NewState(engine, "stock", func() Stock { return Stock{} }),
    }

    atmos.On[ItemAdded](engine).Requires(i.positiveQuantity)

    return i
}

// 5. Use it
func main() {
    inventory := NewInventory()

    inventory.engine.Emit(ItemAdded{ItemID: "WIDGET-001", Quantity: 100})

    fmt.Printf("Inventory: %+v\n", inventory.stock.Get())
    // Output: Inventory: map[WIDGET-001:100]
}
```

## Core Concepts

### Events

Events are **immutable facts** about what happened:

```go
type OrderPlaced struct {
    OrderID    string
    CustomerID string
    Items      []OrderItem
    Total      float64
}

func (OrderPlaced) Type() string { return "order_placed" }
```

`atmos.On[OrderPlaced](e)` declares an event type on an engine; the type's name
comes from the event, and events come back from JSON as the same type.

Events are:
- **Past tense** - "OrderPlaced" not "PlaceOrder"
- **Immutable** - Never changed after creation
- **Complete** - Contain all relevant data

### State

State is **derived from events**. `atmos.NewState` registers a state, with a
function that makes its initial value, and returns its handle: `orders.Get()`
reads it — typed, with no name to look up and no cast. A handle is bound to its
engine, so it lives with whatever wires the engine.

An event says what it does to its state, with an `Apply` method:

```go
func (e OrderPlaced) Apply(s Orders) Orders {
    s[e.OrderID] = Order{
        ID:         e.OrderID,
        CustomerID: e.CustomerID,
        Items:      e.Items,
        Total:      e.Total,
        Status:     "pending",
    }
    return s
}
```

Any state of type `Orders` folds in `OrderPlaced` events through `Apply`, with no
wiring. Other states that care about the event say so with `Updates` (see
[Multiple State Updates](#multiple-state-updates)).

The engine keeps each state current, applying every event once, as it's
emitted, so reading a state doesn't replay the log. `Apply` may change the
state it's given, as above: whenever an event is undone (a refused batch, say),
the engine throws the state away and rebuilds it from a new initial value and
the log. The state `Get` returns is the engine's own: read it, don't change it.

### Validators

Validators **enforce business rules** before events commit. A rule is a
method on whatever holds the engine's handles, so it reaches them — and any
configuration — through its receiver:

```go
// atmos.On[OrderPlaced](e).Requires(shop.sufficientInventory)
func (shop *Shop) sufficientInventory(event OrderPlaced) bool {
    inventory := shop.inventory.Get()

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
// atmos.On[OrderPlaced](e).Then(shop.sendConfirmation)
func (shop *Shop) sendConfirmation(event OrderPlaced) {
    customer := shop.customers.Get()[event.CustomerID]

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
atmos.On[PaymentProcessed](e).
    Before(shop.generateInvoiceNumber).  // Runs as part of transaction
    Then(shop.sendReceipt)               // Runs after commitment
```

Use before hooks when the side effect must be part of the same transaction (e.g., generating IDs, procedural content).

### Commands

A command is something that can be done. Asked of the engine with `Do`, it
decides — from the state as it is — what happens, as events, and they're
emitted together, or not at all:

```go
type Transfer struct {
    From, To string
    Amount   int
}

atmos.Command[Transfer](e).Decides(bank.transfer)

func (bank *Bank) transfer(c Transfer) []atmos.Event {
    if c.From == c.To {
        return nil // nothing to do
    }
    return []atmos.Event{Withdrawn{c.From, c.Amount}, Deposited{c.To, c.Amount}}
}

ok := e.Do(Transfer{"alice", "bob", 50}) // both happen, or neither
```

Each event still answers to its own rules: if bob's account can't take the
deposit, alice keeps her money. `Do` reports whether anything happened.
Commands aren't events — they're never stored — so they can carry anything,
behavior included. An action that's exactly one event doesn't need a command;
emit the event.

## One Declaration

Declare everything in one place — what can be done, and what can happen:

```go
// What can be done
atmos.Command[PlaceOrder](e).Decides(shop.placeOrder)

// What can happen, and when it's allowed
atmos.On[OrderPlaced](e).
    Requires(shop.validCustomer, shop.sufficientInventory, shop.validPaymentMethod).
    Before(shop.generateOrderNumber, shop.calculateTax).
    Then(shop.reserveInventory, shop.processPayment, shop.sendConfirmation).
    Updates(atmos.Reduces(customers, countOrder))
```

**Everything about this event is visible in one declaration.**

### Available Methods

- `atmos.Command[C](e).Decides(decide)` - Declare what a command becomes, as events
- `atmos.On[T](e)` - Start declaring rules for an event type
- `Requires(...rules)` - Add validation rules (all must pass)
- `Before(...hooks)` - Run before commit (transactional)
- `Then(...listeners)` - Run after commit (side effects)
- `Updates(atmos.Reduces(state, reducer))` - Update a state other than the event's own

This is the only way to define an engine: there's no untyped or by-name
registration beside it.

## Advanced Features

### Custom Event Repositories

By default, Atmos stores events in memory. For production use, implement a custom repository to persist events automatically:

```go
// Implement the EventRepository interface (types is
// github.com/cumulusrpg/atmos/types)
type FileRepository struct {
    filepath string
}

func (r *FileRepository) Add(engine types.Engine, event atmos.Event) error {
    // Serialize the event, with its type
    jsonData, _ := engine.MarshalEvents([]atmos.Event{event})
    // Append to file atomically
    return appendToFile(r.filepath, jsonData)
}

func (r *FileRepository) GetAll(engine types.Engine) []atmos.Event {
    // Load JSON from file
    jsonData := readFile(r.filepath)
    // Deserialize, as the event types the engine declared with On
    events, _ := engine.UnmarshalEvents(jsonData)
    return events
}

func (r *FileRepository) SetAll(engine types.Engine, events []atmos.Event) error {
    // Serialize all events
    jsonData, _ := engine.MarshalEvents(events)
    // Replace file contents atomically
    return writeFile(r.filepath, jsonData)
}

// Use custom repository
repo := &FileRepository{filepath: "events.jsonl"}
engine := atmos.NewEngine(atmos.WithRepository(repo))

// Events are now automatically persisted on every Emit()
engine.Emit(OrderPlaced{...})  // Saved to disk automatically!
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

// Later: load and replay, into an engine defined the same way, so it
// knows what each event's type decodes as
shop := NewShop()
events, _ := shop.engine.UnmarshalEvents(db.Load("event_log"))
shop.engine.SetEvents(events)

// State is now rebuilt from history
state := shop.orders.Get()
```

Perfect for:
- Persisting application state
- Debugging production issues
- Migrating between versions
- Auditing and compliance

### Dependencies

Reference data and services belong to whatever wires the engine, next to its
state handles. Rules and listeners are methods there, so they reach them
through the receiver — no lookup by name, no cast:

```go
type Shop struct {
    engine  *atmos.Engine
    orders  atmos.State[Orders]
    catalog *catalog.ProductCatalog
    email   EmailService
}

func (shop *Shop) knownProduct(e OrderPlaced) bool {
    return shop.catalog.GetProduct(e.ProductID) != nil
}
```

### Multiple State Updates

An event applies itself to its own state. Other states that care about it say
so — the definition is where that decision is visible:

```go
atmos.On[OrderPlaced](e).
    Updates(atmos.Reduces(customers, countOrder)).  // orders per customer
    Updates(atmos.Reduces(analytics, recordSale))   // sales metrics

func countOrder(s CustomerOrders, e OrderPlaced) CustomerOrders {
    s[e.CustomerID]++
    return s
}
```

Each reducer sees the event and updates its own state independently. An event
that applies itself to a state can't also be reduced into it.

## Architecture

The event flow in Atmos:

```
User Action
    ↓
Command Decides (optional) ──→ the events that happen, together
    ↓
Validators Check ──→ [REJECT if any fail]
    ↓
Before Hooks Run (transactional)
    ↓
Events Apply to State
    ↓
Event Committed to Repository ← [Point of no return]
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
- Rules, events and decisions are plain functions
- No mocks needed
- Fast, isolated unit tests

**Flexibility**
- Add features without changing code
- Rules are explicit and visible
- Easy to reason about complex logic

**Reliability**
- Events are immutable
- State is derived from events, never set directly
- Transactions are atomic

## Examples

See the [Tic-Tac-Toe example](examples/tictactoe) for a complete working game with tests.

## Contributing

Atmos is part of the [Cumulus RPG](https://github.com/cumulusrpg) project. Issues and PRs welcome!

## License

This project is part of the Cumulus RPG system.

---

**Built with Atmos? [Let us know!](https://github.com/cumulusrpg/atmos/discussions)**
