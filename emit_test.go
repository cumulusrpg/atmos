package atmos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test events for emit builder
type PlayerRegisteredEvent struct {
	PlayerName string
	PlayerType string
}

func (e *PlayerRegisteredEvent) Type() string { return "player_registered" }

type TokensGrantedEvent struct {
	PlayerName string
	Amount     int
}

func (e *TokensGrantedEvent) Type() string { return "tokens_granted" }

func TestEmitFluentAPI(t *testing.T) {
	engine := NewEngine()

	// Register the listener using the fluent API
	// When a player registers, emit tokens_granted if they're a creator
	engine.When("player_registered").
		Then(Emit[*PlayerRegisteredEvent, *TokensGrantedEvent]("tokens_granted").
			If(func(e *PlayerRegisteredEvent) bool {
				return e.PlayerType == "creator"
			}).
			From(func(e *PlayerRegisteredEvent) []*TokensGrantedEvent {
				return []*TokensGrantedEvent{
					{
						PlayerName: e.PlayerName,
						Amount:     100,
					},
				}
			}),
		)

	// Test 1: Creator should trigger tokens_granted
	creatorEvent := &PlayerRegisteredEvent{
		PlayerName: "Alice",
		PlayerType: "creator",
	}
	engine.Emit(creatorEvent)

	// Verify event log
	events := engine.GetEvents()
	assert.Equal(t, 2, len(events), "Should have player_registered + tokens_granted")
	assert.Equal(t, "player_registered", events[0].Type())
	assert.Equal(t, "tokens_granted", events[1].Type())

	// Verify tokens event details
	tokensEvent := events[1].(*TokensGrantedEvent)
	assert.Equal(t, "Alice", tokensEvent.PlayerName)
	assert.Equal(t, 100, tokensEvent.Amount)

	// Test 2: Regular player should NOT trigger tokens_granted
	playerEvent := &PlayerRegisteredEvent{
		PlayerName: "Bob",
		PlayerType: "player",
	}
	engine.Emit(playerEvent)

	events = engine.GetEvents()
	assert.Equal(t, 3, len(events), "Should have 2 player_registered + 1 tokens_granted")
	assert.Equal(t, "player_registered", events[2].Type())
}

func TestEmitWithoutCondition(t *testing.T) {
	engine := NewEngine()

	// Emit without condition - always triggers
	engine.When("player_registered").
		Then(Emit[*PlayerRegisteredEvent, *TokensGrantedEvent]("tokens_granted").
			From(func(e *PlayerRegisteredEvent) []*TokensGrantedEvent {
				return []*TokensGrantedEvent{
					{
						PlayerName: e.PlayerName,
						Amount:     50,
					},
				}
			}),
		)

	// Any player registration should trigger tokens
	engine.Emit(&PlayerRegisteredEvent{
		PlayerName: "Charlie",
		PlayerType: "player",
	})

	events := engine.GetEvents()
	assert.Equal(t, 2, len(events), "Should have player_registered + tokens_granted")
	assert.Equal(t, "player_registered", events[0].Type())
	assert.Equal(t, "tokens_granted", events[1].Type())

	tokensEvent := events[1].(*TokensGrantedEvent)
	assert.Equal(t, "Charlie", tokensEvent.PlayerName)
	assert.Equal(t, 50, tokensEvent.Amount)
}

type WelcomeEmailEvent struct {
	PlayerName string
}

func (e *WelcomeEmailEvent) Type() string { return "welcome_email" }

func TestEmitMultipleEvents(t *testing.T) {
	engine := NewEngine()

	// Fan-out pattern: one event triggers multiple events
	engine.When("player_registered").
		Then(Emit[*PlayerRegisteredEvent, Event]("tokens_granted_and_welcome_email").
			From(func(e *PlayerRegisteredEvent) []Event {
				return []Event{
					&TokensGrantedEvent{
						PlayerName: e.PlayerName,
						Amount:     25,
					},
					&WelcomeEmailEvent{
						PlayerName: e.PlayerName,
					},
				}
			}),
		)

	engine.Emit(&PlayerRegisteredEvent{
		PlayerName: "Dave",
		PlayerType: "player",
	})

	events := engine.GetEvents()
	assert.Equal(t, 3, len(events), "Should have player_registered + tokens_granted + welcome_email")
	assert.Equal(t, "player_registered", events[0].Type())
	assert.Equal(t, "tokens_granted", events[1].Type())
	assert.Equal(t, "welcome_email", events[2].Type())
}
