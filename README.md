# Atmos

Atmos is a lightweight, event-driven game engine for Go. It provides a flexible architecture for building turn-based games with event sourcing patterns.

## Features

- **Event-Driven Architecture**: Built on an event emission and handling system
- **Fluent API**: English-like syntax for declaring game rules and event handlers
- **Type-Safe**: Leverages Go generics for type-safe event handling
- **Conditional Logic**: Support for complex game rules with before/after hooks
- **Game State Management**: Clean separation of game state and game logic

## Installation

```bash
go get github.com/cumulusrpg/atmos
```

## Usage

```go
import "github.com/cumulusrpg/atmos"

// Create a new game engine
engine := atmos.NewEngine()

// Define event handlers using the fluent API
engine.When(MyEvent{}).Then(func(ctx *atmos.Context[MyEvent]) {
    // Handle event
})

// Emit events
engine.Emit(ctx, MyEvent{})
```

## Architecture

Atmos provides the following core components:

- **Engine**: Central event processing system
- **Context**: Execution context for event handlers
- **Validators**: Pre-execution validation hooks
- **Reducers**: State transformation functions
- **Listeners**: Post-execution side effects

## License

This project is part of the Cumulus RPG system.
