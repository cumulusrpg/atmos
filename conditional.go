package atmos

// ConditionalListener wraps a listener with a condition that must pass before execution
// This allows explicit, self-documenting game rules like:
//
//	When(&IsCreatorPlayer{}).Then(&StartCardAction{})
//
// Instead of burying the condition inside the listener logic.
type ConditionalListener struct {
	Condition EventValidator
	Action    EventListener
}

// Handle executes the action only if the condition validates
func (cl ConditionalListener) Handle(engine *Engine, event Event) {
	// Check condition first
	if !cl.Condition.Validate(engine, event) {
		return
	}

	// Condition passed, execute action
	cl.Action.Handle(engine, event)
}

// When starts building a conditional listener with a validation condition
// Example:
//
//	When(&IsCreatorPlayer{}).Then(&StartCardAction{})
func When(condition EventValidator) *ConditionalBuilder {
	return &ConditionalBuilder{condition: condition}
}

// ConditionalBuilder provides a fluent API for constructing conditional listeners
type ConditionalBuilder struct {
	condition EventValidator
}

// Then completes the conditional listener by specifying the action to take
func (cb *ConditionalBuilder) Then(action EventListener) ConditionalListener {
	return ConditionalListener{
		Condition: cb.condition,
		Action:    action,
	}
}