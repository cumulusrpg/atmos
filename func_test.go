package atmos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidFunc_APlainFunctionIsARule(t *testing.T) {
	engine := NewEngine()
	engine.When("order_placed").Requires(ValidFunc(func(_ *Engine, ev OrderPlacedEvent) bool {
		return ev.Amount > 0
	}))

	assert.False(t, engine.Emit(OrderPlacedEvent{OrderID: "free", Amount: 0}))
	assert.True(t, engine.Emit(OrderPlacedEvent{OrderID: "paid", Amount: 5}))
	assert.Len(t, engine.GetEvents(), 1)
}

func TestDoFunc_APlainFunctionIsAListener(t *testing.T) {
	engine := NewEngine()
	var heard []string
	engine.When("order_placed").Then(DoFunc(func(_ *Engine, ev OrderPlacedEvent) {
		heard = append(heard, ev.OrderID)
	}))

	engine.Emit(OrderPlacedEvent{OrderID: "ORD-1"})

	assert.Equal(t, []string{"ORD-1"}, heard)
}
