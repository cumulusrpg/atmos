# Tic-Tac-Toe Example

A simple tic-tac-toe game demonstrating how to use the Atmos event-driven game engine.

## Running the Tests

```bash
go test -v
```

## How It Works

This example demonstrates the core features of Atmos:

### 1. Events (events.go)

Events represent things that happen in the game:
- `GameStartedEvent` - Records when the game begins
- `MoveMadeEvent` - Records each player's move
- `GameEndedEvent` - Records the game result

### 2. State (state.go)

The `GameState` struct holds the current game state:
- Board positions (9 cells)
- Current player
- Winner status
- Player names

### 3. Validators (validators.go)

Validators ensure events are valid before they're committed:
- `ValidMove` - Checks if a move is legal (correct player, valid position, game ongoing)
- `GameNotStarted` - Ensures game can only start once

### 4. Reducers (reducers.go)

Reducers update state in response to events:
- `ReduceGameStarted` - Initializes game state
- `ReduceMoveMade` - Updates board and switches players
- `ReduceGameEnded` - Records the winner

### 5. Listeners (listeners.go)

Listeners trigger side effects after events:
- `CheckForWinner` - Automatically checks for a winner after each move and emits `GameEndedEvent`

### 6. Game Setup (game.go)

The game uses Atmos's fluent API to wire everything together:

```go
engine.When("game_started", func() atmos.Event { return &GameStartedEvent{} }).
    Requires(atmos.Valid(&GameNotStarted{}))

engine.When("move_made", func() atmos.Event { return &MoveMadeEvent{} }).
    Requires(atmos.Valid(&ValidMove{})).
    Then(atmos.Do(&CheckForWinner{}))

engine.When("game_ended", func() atmos.Event { return &GameEndedEvent{} })
```

## Key Features Demonstrated

### Event Sourcing
All game state is derived from events. You can replay the event log to rebuild state at any point.

### Validation
Invalid moves are rejected before being added to the event log, ensuring game rules are enforced.

### Automatic Side Effects
The `CheckForWinner` listener automatically checks for game-ending conditions after each move, demonstrating how complex game logic can be composed from simple handlers.

### Type Safety
Using Go generics, validators and listeners are type-safe:
```go
func (v *ValidMove) ValidateTyped(engine *atmos.Engine, event MoveMadeEvent) bool
```

## Example Usage

```go
game := NewGame()

// Start the game
game.StartGame("Alice", "Bob")

// Make moves
game.MakeMove("X", 4) // X takes center
game.MakeMove("O", 0) // O takes top-left
game.MakeMove("X", 2) // X takes top-right
// ... game continues until win or draw

// Get current state
state := game.GetGameState()
fmt.Println("Winner:", state.Winner)

// Get event history
events := game.engine.GetEvents()
```
