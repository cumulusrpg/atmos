package tictactoe

import "time"

// MoveMadeEvent records a player's move
type MoveMadeEvent struct {
	Player   string // "X" or "O"
	Position int    // 0-8 (board position)
	Time     time.Time
}

func (e MoveMadeEvent) Type() string {
	return "move_made"
}

func (e MoveMadeEvent) Timestamp() time.Time {
	return e.Time
}

// GameStartedEvent records the start of a game
type GameStartedEvent struct {
	PlayerX string // Name of X player
	PlayerO string // Name of O player
	Time    time.Time
}

func (e GameStartedEvent) Type() string {
	return "game_started"
}

func (e GameStartedEvent) Timestamp() time.Time {
	return e.Time
}

// GameEndedEvent records the end of a game
type GameEndedEvent struct {
	Winner string // "X", "O", or "draw"
	Time   time.Time
}

func (e GameEndedEvent) Type() string {
	return "game_ended"
}

func (e GameEndedEvent) Timestamp() time.Time {
	return e.Time
}
