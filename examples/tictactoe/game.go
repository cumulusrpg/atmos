package tictactoe

import (
	"fmt"

	"github.com/cumulusrpg/atmos"
)

// Game represents a tic-tac-toe game using the atmos engine
type Game struct {
	engine *atmos.Engine
	state  atmos.State[GameState]
}

// NewGame creates a new tic-tac-toe game
func NewGame() *Game {
	engine := atmos.NewEngine()

	// Register game state; the handle is how the game's rules read it
	g := &Game{engine: engine, state: atmos.NewState(engine, "game", NewGameState)}

	// Register event handlers using fluent API
	engine.When("game_started", func() atmos.Event { return &GameStartedEvent{} }).
		Requires(atmos.Rule(g.notStarted)).
		Updates(atmos.Reduces(g.state, reduceGameStarted))

	engine.When("move_made", func() atmos.Event { return &MoveMadeEvent{} }).
		Requires(atmos.Rule(g.validMove)).
		Then(atmos.DoFunc(g.checkForWinner)).
		Updates(atmos.Reduces(g.state, reduceMoveMade))

	engine.When("game_ended", func() atmos.Event { return &GameEndedEvent{} }).
		Updates(atmos.Reduces(g.state, reduceGameEnded))

	return g
}

// StartGame begins a new game
func (g *Game) StartGame(playerX, playerO string) error {
	success := g.engine.Emit(GameStartedEvent{
		PlayerX: playerX,
		PlayerO: playerO,
	})

	if !success {
		return fmt.Errorf("failed to start game: game already started")
	}

	return nil
}

// MakeMove attempts to make a move
func (g *Game) MakeMove(player string, position int) error {
	success := g.engine.Emit(MoveMadeEvent{
		Player:   player,
		Position: position,
	})

	if !success {
		state := g.GetGameState()
		if !state.GameStarted {
			return fmt.Errorf("game not started")
		}
		if state.IsGameOver() {
			return fmt.Errorf("game is over")
		}
		if player != state.CurrentPlayer {
			return fmt.Errorf("not your turn (current player: %s)", state.CurrentPlayer)
		}
		if !state.IsPositionEmpty(position) {
			return fmt.Errorf("position %d is already occupied", position)
		}
		return fmt.Errorf("invalid move")
	}

	return nil
}

// GetGameState returns the current game state
func (g *Game) GetGameState() GameState {
	return g.state.Get()
}

// GetBoard returns a string representation of the board
func (g *Game) GetBoard() string {
	state := g.GetGameState()
	board := ""
	for i := 0; i < 9; i++ {
		cell := state.Board[i]
		if cell == "" {
			cell = "-"
		}
		board += cell
		if (i+1)%3 == 0 && i < 8 {
			board += "\n"
		} else if i < 8 {
			board += " "
		}
	}
	return board
}
