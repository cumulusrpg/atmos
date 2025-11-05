package atmos

import (
	"fmt"
	"time"
)

// EmitWithTimestamp creates and emits an event with current timestamp
// Returns error if emission fails with descriptive message
func EmitWithTimestamp(engine *Engine, event Event, errorContext string) error {
	// Set timestamp if event has a Time field
	if timedEvent, ok := event.(interface{ SetTime(time.Time) }); ok {
		timedEvent.SetTime(time.Now())
	}

	success := engine.Emit(event)
	if !success {
		return fmt.Errorf("failed to %s", errorContext)
	}
	return nil
}

// GetTypedState returns typed state from engine with error handling
func GetTypedState[T any](engine *Engine, stateName string) (T, error) {
	var zero T
	state := engine.GetState(stateName)
	if state == nil {
		return zero, fmt.Errorf("state %s not found", stateName)
	}

	typedState, ok := state.(T)
	if !ok {
		return zero, fmt.Errorf("state %s has wrong type", stateName)
	}

	return typedState, nil
}
