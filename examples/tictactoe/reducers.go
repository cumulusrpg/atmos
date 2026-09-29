package tictactoe

// reduceGameStarted updates state when game starts
func reduceGameStarted(s GameState, e GameStartedEvent) GameState {
	s.GameStarted = true
	s.PlayerXName = e.PlayerX
	s.PlayerOName = e.PlayerO
	s.CurrentPlayer = "X" // X always goes first

	return s
}

// reduceMoveMade updates state when a move is made
func reduceMoveMade(s GameState, e MoveMadeEvent) GameState {
	// Make the move
	s.Board[e.Position] = e.Player

	// Switch players
	if s.CurrentPlayer == "X" {
		s.CurrentPlayer = "O"
	} else {
		s.CurrentPlayer = "X"
	}

	return s
}

// reduceGameEnded updates state when game ends
func reduceGameEnded(s GameState, e GameEndedEvent) GameState {
	s.Winner = e.Winner

	return s
}
