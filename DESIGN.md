# Atmos Design Notes

## Core Model

The whole system reduces to one recursive pattern:

```
events → [validators] → log → reducers → state
                                            ↓
                                  state diffs as events
                                            ↓
                                  reducers → projection
                                            ↓
                                  reducers → client projection
                                  (vdom, TUI, generated code, etc.)
```

Same 5 components all the way down. Each layer is another application of the same pattern.

### The 5 Components

1. **Events** — things that happened, immutable facts
2. **Validators** — gates on what events can enter the log
3. **Log** — append-only record of validated events; the primary truth
4. **Reducers** — transform (state, event) → new state; pure, synchronous
5. **Listeners** — fire after commit; can emit new events into the sandbox inbox

### Hierarchy

```
log          → truth, immutable, primary
state        → privileged projection, always current, load-bearing for validation
projections  → derived, async, eventually consistent, disposable
```

State is not arbitrary — it's the validation surface. The one projection the system
guarantees is current before every write. Everything else can lag and be rebuilt.

### Log is State

```
log = state
commit(log, event) → log + [event]
```

The log IS the state. Appending an event IS the reduction. All other state objects
(design.State, board state, etc.) are derived views — folds over the log.

State is a cache of that fold, trading purity for query speed. Validators reading
current state are reading a memoized fold — not contamination, just optimization.

---

## Sandbox Processing Model

The desired processing model. Currently atmos does not implement this fully.

### Invariant

The inbox must be empty before anything is committed. Event processing always happens
in a sandbox. Everything commits atomically or nothing does.

### Algorithm

```
inbox = [incoming event]
sandbox state = copy of current state
sandbox log = []

while inbox not empty:
    event = inbox.dequeue()
    validate(sandbox state, event)
      → if invalid: discard entire sandbox, return rejection
    sandbox state = reduce(sandbox state, event)
    sandbox log += event
    listeners(event) → may emit new events → inbox.enqueue(...)

inbox empty:
    commit sandbox log + sandbox state atomically
```

### Properties

- No partial commits — observers only ever see complete sandbox batches
- Cascading events (listener emits event) are part of the same atomic unit
- Projections only see committed batches, never intermediate state
- Listeners emit into the sandbox inbox, not directly to the log

### Parallelization

Sequential processing assumed for now — one sandbox at a time, no conflict detection
needed. The upgrade path to MVCC is clear when needed:

```
txn reads state at version N
txn does work in sandbox
commit: has anything I read changed since version N?
  no  → commit, increment version
  yes → retry from new version
```

No locks held during processing. Conflict detection only at commit. Optimistic —
assumes conflicts are rare. This is what Postgres does (MVCC).

---

## Specific Additions Needed

### 1. Eager Reduction

**Current:** `GetState` replays all events through reducers on every call. O(n).

**Desired:** Keep running state in memory. Apply each event's reducer once on commit.
`GetState` returns current in-memory state. O(1).

Needed for: NATS-driven subscribers that maintain state without full replay.

### 2. State Machines

Declarative transition definitions on state fields.

```go
engine.RegisterStateMachine("book.status", transitions{
    "available": {"checkout": "on_loan"},
    "on_loan":   {"return": "available", "lose": "lost"},
    "lost":      {"find": "available"},
})
```

A validator that checks current status before allowing the event through.
A reducer that transitions the status field on success.
Invalid transitions rejected automatically.

### 3. Projections as First-Class

```go
engine.RegisterProjection("catalog", func(e *Engine) interface{} {
    return computeCatalog(e.GetState("library"))
})

engine.OnProjectionChange("catalog", func(e *Engine, result interface{}) {
    // publish to NATS, push to clients, etc.
})
```

After each commit: atmos recomputes registered projections, fires listeners where
output changed. Projections are just named folds with change listeners.

### 4. NATS Repository

Repository implementation that publishes to JetStream on Add, replays from stream
on Restore. Wraps InMemory for fast reads after initial load.

```go
type NATSRepository struct {
    memory  *repository.InMemory
    js      jetstream.JetStream
    stream  string
    subject string
}

// Add: in-memory + js.Publish
// GetAll: in-memory (fast)
// Restore: fetch all from NATS stream → populate in-memory
```

Enables: NATS as durable event log, FileManager subscribes for .hoom backup.

### 5. Multi-Event Atomic Emit

```go
engine.EmitAll([]Event{e1, e2, e3})
```

All committed or none. Natural consequence of the sandbox model — just pre-populate
the inbox with multiple events.

### 6. Validate Without Commit

```go
engine.Validate(event Event) bool
```

Runs validators against current state without appending to log. Useful for
pre-flight checks (e.g. HTTP handler validating before publishing to NATS).

---

## What Stays the Same

- Event registration + factory functions
- Typed validators and listeners (generics)
- Fluent API (When...Requires...Updates...Then)
- Repository abstraction (WithRepository)
- Snapshot support
- Service locator
- EmitBuilder (fluent cross-event emission)
- Before hooks
