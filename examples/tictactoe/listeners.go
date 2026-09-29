package tictactoe

import (
	"github.com/cumulusrpg/atmos"
)

// checkForWinner checks if the game is over after each move
func (g *Game) checkForWinner(engine *atmos.Engine, event MoveMadeEvent) {
	winner := g.state.Get().CheckWinner()
	if winner != "" {
		// Emit game ended event
		engine.Emit(GameEndedEvent{
			Winner: winner,
		})
	}
}
