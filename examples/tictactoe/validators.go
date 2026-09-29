package tictactoe

// validMove is the rule that a move is legal
func (g *Game) validMove(event MoveMadeEvent) bool {
	state := g.state.Get()

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

// notStarted is the rule that the game hasn't started yet
func (g *Game) notStarted(GameStartedEvent) bool {
	return !g.state.Get().GameStarted
}
