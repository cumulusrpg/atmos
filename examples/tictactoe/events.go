package tictactoe

// MoveMadeEvent records a player's move
type MoveMadeEvent struct {
	Player   string // "X" or "O"
	Position int    // 0-8 (board position)
}

func (e MoveMadeEvent) Type() string {
	return "move_made"
}

// GameStartedEvent records the start of a game
type GameStartedEvent struct {
	PlayerX string // Name of X player
	PlayerO string // Name of O player
}

func (e GameStartedEvent) Type() string {
	return "game_started"
}

// GameEndedEvent records the end of a game
type GameEndedEvent struct {
	Winner string // "X", "O", or "draw"
}

func (e GameEndedEvent) Type() string {
	return "game_ended"
}

// Apply starts the game between the two players; X goes first.
func (e GameStartedEvent) Apply(s GameState) GameState {
	s.GameStarted = true
	s.PlayerXName = e.PlayerX
	s.PlayerOName = e.PlayerO
	s.CurrentPlayer = "X"
	return s
}

// Apply marks the board, and it's the other player's turn.
func (e MoveMadeEvent) Apply(s GameState) GameState {
	s.Board[e.Position] = e.Player
	if s.CurrentPlayer == "X" {
		s.CurrentPlayer = "O"
	} else {
		s.CurrentPlayer = "X"
	}
	return s
}

// Apply records the winner.
func (e GameEndedEvent) Apply(s GameState) GameState {
	s.Winner = e.Winner
	return s
}
