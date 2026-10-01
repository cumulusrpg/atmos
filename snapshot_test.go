package atmos

import (
	"testing"

	"github.com/cumulusrpg/atmos/repository"
	"github.com/stretchr/testify/assert"
)

// TestSnapshotWithNonSnapshotRepository verifies snapshot methods handle non-snapshot repos gracefully
func TestSnapshotWithNonSnapshotRepository(t *testing.T) {
	// Default engine uses InMemory which doesn't support snapshots
	engine := NewEngine()

	// SetSnapshot should return error
	err := engine.SetSnapshot("test", map[string]int{"score": 100})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support snapshots")

	// ClearSnapshot should return error
	err = engine.ClearSnapshot("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support snapshots")

	// HasSnapshot should return false (not panic)
	hasSnapshot := engine.HasSnapshot("test")
	assert.False(t, hasSnapshot)
}

// TestSnapshotMergeWithInvalidJSON verifies mergeSnapshot handles invalid JSON gracefully
func TestSnapshotMergeWithInvalidJSON(t *testing.T) {
	engine := NewEngine()

	type SimpleState struct {
		Value int
	}

	initialState := SimpleState{Value: 42}

	// Invalid JSON should return the initial state unchanged
	result := engine.mergeSnapshot(initialState, []byte("not valid json"))
	assert.Equal(t, initialState, result)
}

// TestSnapshotMergeWithPointerState verifies mergeSnapshot handles pointer initial states
func TestSnapshotMergeWithPointerState(t *testing.T) {
	engine := NewEngine()

	type SimpleState struct {
		Value int
	}

	initialState := &SimpleState{Value: 42}

	// Should merge correctly even when initial state is a pointer
	result := engine.mergeSnapshot(initialState, []byte(`{"Value": 100}`))
	resultState := result.(SimpleState)
	assert.Equal(t, 100, resultState.Value)
}

// TestSnapshotMergeWithUnmarshalableState verifies mergeSnapshot handles unmarshalable initial state
func TestSnapshotMergeWithUnmarshalableState(t *testing.T) {
	engine := NewEngine()

	// A struct with a channel field cannot be marshaled
	type UnmarshalableState struct {
		Ch chan int
	}

	initialState := UnmarshalableState{Ch: make(chan int)}

	// Should return initial state unchanged when marshal fails
	result := engine.mergeSnapshot(initialState, []byte(`{}`))
	assert.Equal(t, initialState, result)
}

// TestSetSnapshotWithUnmarshalableData verifies SetSnapshot handles unmarshalable data
func TestSetSnapshotWithUnmarshalableData(t *testing.T) {
	engine := NewEngine(WithRepository(repository.NewInMemorySnapshot()))

	// Channels cannot be JSON marshaled
	unmarshalable := make(chan int)
	err := engine.SetSnapshot("test", unmarshalable)
	assert.Error(t, err)
}
