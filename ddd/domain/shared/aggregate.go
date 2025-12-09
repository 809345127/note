package shared

// AggregateRoot Aggregate Root Interface
// Aggregate root is a core concept in DDD, it's the entry point of an aggregate and maintains the consistency boundary of the aggregate
// Features:
// 1. Has a globally unique identifier
// 2. Maintains invariants within the aggregate
// 3. All modifications must go through the aggregate root
// 4. Responsible for publishing domain events
type AggregateRoot interface {
	// ID Returns the globally unique identifier of the aggregate root
	ID() string

	// Version Returns the current version number, used for optimistic locking concurrency control
	Version() int

	// PullEvents Gets and clears the domain events recorded by the aggregate root
	// This is the standard domain event pattern: aggregate root records events, repository publishes events after saving
	PullEvents() []DomainEvent
}

// Entity Entity Interface
// Differences between entity and value object:
// 1. Entity has a unique identifier (ID)
// 2. Entity has a longer lifecycle
// 3. Equality is determined by identifier (even with same attributes, different ID means different entity)
type Entity interface {
	ID() string
}

// ValueObject Value Object Interface
// Characteristics of value objects:
// 1. No unique identifier
// 2. Immutable
// 3. Equality is determined by attribute values
// 4. Typically used for descriptive concepts
// Note: There's no perfect way to enforce immutability in Go, it needs to be guaranteed through conventions and coding standards
type ValueObject interface {
	// Equals Compare whether two value objects are equal
	Equals(other interface{}) bool
}
