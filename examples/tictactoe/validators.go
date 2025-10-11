package tictactoe

import "github.com/cumulusrpg/atmos"

// ValidMove validates that a move is legal
type ValidMove struct{}

func (v *ValidMove) ValidateTyped(engine *atmos.Engine, event MoveMadeEvent) bool {
	state := engine.GetState("game").(GameState)

	// Game must be started
	if !state.GameStarted {
		return false
	}

	// Game must not be over
	if state.IsGameOver() {
		return false
	}

	// Must be the correct player's turn
	if event.Player != state.CurrentPlayer {
		return false
	}

	// Position must be valid and empty
	return state.IsPositionEmpty(event.Position)
}

// GameNotStarted validates that the game hasn't started yet
type GameNotStarted struct{}

func (v *GameNotStarted) ValidateTyped(engine *atmos.Engine, event GameStartedEvent) bool {
	state := engine.GetState("game").(GameState)
	return !state.GameStarted
}
