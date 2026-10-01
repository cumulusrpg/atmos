package types

// Event represents something that happened in the system
type Event interface {
	Type() string
}
