package tictactoe

// GameState represents the current state of a tic-tac-toe game
type GameState struct {
	Board         [9]string // Board positions: "X", "O", or "" for empty
	CurrentPlayer string    // "X" or "O"
	Winner        string    // "X", "O", "draw", or "" if game ongoing
	GameStarted   bool
	PlayerXName   string
	PlayerOName   string
}

// NewGameState creates a fresh game state
func NewGameState() GameState {
	return GameState{
		Board:         [9]string{},
		CurrentPlayer: "X", // X always starts
		Winner:        "",
		GameStarted:   false,
	}
}

// IsPositionEmpty checks if a position is available
func (s GameState) IsPositionEmpty(position int) bool {
	if position < 0 || position > 8 {
		return false
	}
	return s.Board[position] == ""
}

// IsGameOver checks if the game has ended
func (s GameState) IsGameOver() bool {
	return s.Winner != ""
}

// CheckWinner determines if there's a winner
func (s GameState) CheckWinner() string {
	// Winning combinations
	lines := [][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
		{0, 4, 8}, {2, 4, 6}, // diagonals
	}

	for _, line := range lines {
		a, b, c := s.Board[line[0]], s.Board[line[1]], s.Board[line[2]]
		if a != "" && a == b && b == c {
			return a
		}
	}

	// Check for draw (board full)
	boardFull := true
	for _, cell := range s.Board {
		if cell == "" {
			boardFull = false
			break
		}
	}
	if boardFull {
		return "draw"
	}

	return "" // Game ongoing
}
