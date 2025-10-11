package tictactoe

import (
	"time"

	"github.com/cumulusrpg/atmos"
)

// CheckForWinner checks if the game is over after each move
type CheckForWinner struct{}

func (l *CheckForWinner) HandleTyped(engine *atmos.Engine, event MoveMadeEvent) {
	state := engine.GetState("game").(GameState)

	winner := state.CheckWinner()
	if winner != "" {
		// Emit game ended event
		engine.Emit(GameEndedEvent{
			Winner: winner,
			Time:   time.Now(),
		})
	}
}
