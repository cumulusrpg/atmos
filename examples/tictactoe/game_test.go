package tictactoe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTicTacToeGame(t *testing.T) {
	game := NewGame()

	// Test game creation
	state := game.GetGameState()
	assert.False(t, state.GameStarted, "Game should not be started initially")
	assert.Equal(t, "", state.Winner, "Should have no winner initially")

	// Test starting a game
	err := game.StartGame("Alice", "Bob")
	assert.NoError(t, err, "Should be able to start game")

	state = game.GetGameState()
	assert.True(t, state.GameStarted, "Game should be started")
	assert.Equal(t, "Alice", state.PlayerXName, "Player X should be Alice")
	assert.Equal(t, "Bob", state.PlayerOName, "Player O should be Bob")
	assert.Equal(t, "X", state.CurrentPlayer, "X should go first")

	// Test that we can't start a game twice
	err = game.StartGame("Charlie", "Dave")
	assert.Error(t, err, "Should not be able to start game twice")

	// Test valid moves
	err = game.MakeMove("X", 4) // X takes center
	assert.NoError(t, err, "X should be able to move")

	state = game.GetGameState()
	assert.Equal(t, "X", state.Board[4], "Center should be X")
	assert.Equal(t, "O", state.CurrentPlayer, "Should be O's turn")

	err = game.MakeMove("O", 0) // O takes top-left
	assert.NoError(t, err, "O should be able to move")

	// Test invalid move - wrong player
	err = game.MakeMove("O", 1)
	assert.Error(t, err, "Should not be able to move out of turn")

	// Test invalid move - occupied position
	err = game.MakeMove("X", 0)
	assert.Error(t, err, "Should not be able to move to occupied position")

	// Play out a winning game
	// Board state:
	// O - -
	// - X -
	// - - -
	_ = game.MakeMove("X", 2) // X takes top-right
	// O - X
	// - X -
	// - - -
	_ = game.MakeMove("O", 1) // O takes top-middle
	// O O X
	// - X -
	// - - -
	_ = game.MakeMove("X", 6) // X takes bottom-left - wins with diagonal!
	// O O X
	// - X -
	// X - -

	state = game.GetGameState()
	assert.Equal(t, "X", state.Winner, "X should win with diagonal")
	assert.True(t, state.IsGameOver(), "Game should be over")

	// Test that we can't move after game is over
	err = game.MakeMove("O", 3)
	assert.Error(t, err, "Should not be able to move after game is over")
}

func TestTicTacToeDraw(t *testing.T) {
	game := NewGame()
	_ = game.StartGame("Alice", "Bob")

	// Play out a draw
	// X O X
	// X O O
	// O X X
	moves := []struct {
		player string
		pos    int
	}{
		{"X", 0}, {"O", 1}, {"X", 2},
		{"O", 4}, {"X", 3}, {"O", 5},
		{"X", 7}, {"O", 6}, {"X", 8},
	}

	for _, move := range moves {
		err := game.MakeMove(move.player, move.pos)
		assert.NoError(t, err, "Move should be valid")
	}

	state := game.GetGameState()
	assert.Equal(t, "draw", state.Winner, "Game should be a draw")
	assert.True(t, state.IsGameOver(), "Game should be over")
}

func TestTicTacToeWinningConditions(t *testing.T) {
	// Test horizontal win
	t.Run("horizontal win", func(t *testing.T) {
		game := NewGame()
		_ = game.StartGame("Alice", "Bob")

		// X X X
		// O O -
		// - - -
		_ = game.MakeMove("X", 0)
		_ = game.MakeMove("O", 3)
		_ = game.MakeMove("X", 1)
		_ = game.MakeMove("O", 4)
		_ = game.MakeMove("X", 2) // X wins

		state := game.GetGameState()
		assert.Equal(t, "X", state.Winner, "X should win horizontally")
	})

	// Test vertical win
	t.Run("vertical win", func(t *testing.T) {
		game := NewGame()
		_ = game.StartGame("Alice", "Bob")

		// X O -
		// X O -
		// X - -
		_ = game.MakeMove("X", 0)
		_ = game.MakeMove("O", 1)
		_ = game.MakeMove("X", 3)
		_ = game.MakeMove("O", 4)
		_ = game.MakeMove("X", 6) // X wins

		state := game.GetGameState()
		assert.Equal(t, "X", state.Winner, "X should win vertically")
	})

	// Test diagonal win
	t.Run("diagonal win", func(t *testing.T) {
		game := NewGame()
		_ = game.StartGame("Alice", "Bob")

		// O - -
		// - O -
		// X X O
		_ = game.MakeMove("X", 6)
		_ = game.MakeMove("O", 0)
		_ = game.MakeMove("X", 7)
		_ = game.MakeMove("O", 4)
		_ = game.MakeMove("X", 1)
		_ = game.MakeMove("O", 8) // O wins

		state := game.GetGameState()
		assert.Equal(t, "O", state.Winner, "O should win diagonally")
	})
}

func TestEventLog(t *testing.T) {
	game := NewGame()
	_ = game.StartGame("Alice", "Bob")
	_ = game.MakeMove("X", 0)
	_ = game.MakeMove("O", 4)
	_ = game.MakeMove("X", 1)
	_ = game.MakeMove("O", 3)
	_ = game.MakeMove("X", 2) // X wins

	// Get all events
	events := game.engine.GetEvents()

	// Should have: 1 game_started, 5 move_made, 1 game_ended
	assert.Equal(t, 7, len(events), "Should have 7 events")

	// Verify event types
	assert.Equal(t, "game_started", events[0].Type())
	assert.Equal(t, "move_made", events[1].Type())
	assert.Equal(t, "move_made", events[2].Type())
	assert.Equal(t, "move_made", events[3].Type())
	assert.Equal(t, "move_made", events[4].Type())
	assert.Equal(t, "move_made", events[5].Type())
	assert.Equal(t, "game_ended", events[6].Type())

	// Verify we can rebuild state from event log
	newEngine := game.engine
	newEngine.SetEvents(events)
	rebuiltState := newEngine.GetState("game").(GameState)

	assert.True(t, rebuiltState.GameStarted, "Rebuilt state should show game started")
	assert.Equal(t, "X", rebuiltState.Winner, "Rebuilt state should show X as winner")
}
