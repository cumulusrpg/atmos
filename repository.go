package atmos

// InMemoryRepository is the default repository implementation that stores events in memory
type InMemoryRepository struct {
	events []Event
}

// NewInMemoryRepository creates a new in-memory repository
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		events: make([]Event, 0),
	}
}

// Add commits a new event to the in-memory store
func (r *InMemoryRepository) Add(event Event) error {
	r.events = append(r.events, event)
	return nil
}

// GetAll returns all events from the in-memory store
func (r *InMemoryRepository) GetAll() []Event {
	return append([]Event{}, r.events...)
}

// SetAll atomically replaces all events in the in-memory store
func (r *InMemoryRepository) SetAll(events []Event) error {
	r.events = append([]Event{}, events...)
	return nil
}
