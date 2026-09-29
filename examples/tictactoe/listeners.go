package tictactoe

// checkForWinner checks if the game is over after each move
func (g *Game) checkForWinner(MoveMadeEvent) {
	winner := g.state.Get().CheckWinner()
	if winner != "" {
		// Emit game ended event
		g.engine.Emit(GameEndedEvent{
			Winner: winner,
		})
	}
}
