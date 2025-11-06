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
