package atmos

// GameBase provides common functionality for all game types
type GameBase struct {
	Engine *Engine
}

// GetEngine returns the underlying engine
func (g *GameBase) GetEngine() *Engine {
	return g.Engine
}

// GetEvents returns all events from the engine
func (g *GameBase) GetEvents() []Event {
	return g.Engine.GetEvents()
}