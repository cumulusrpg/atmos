package tictactoe

import "github.com/cumulusrpg/atmos"

// ReduceGameStarted updates state when game starts
func ReduceGameStarted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(GameState)
	e := event.(GameStartedEvent)

	s.GameStarted = true
	s.PlayerXName = e.PlayerX
	s.PlayerOName = e.PlayerO
	s.CurrentPlayer = "X" // X always goes first

	return s
}

// ReduceMoveMade updates state when a move is made
func ReduceMoveMade(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(GameState)
	e := event.(MoveMadeEvent)

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

// ReduceGameEnded updates state when game ends
func ReduceGameEnded(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(GameState)
	e := event.(GameEndedEvent)

	s.Winner = e.Winner

	return s
}
