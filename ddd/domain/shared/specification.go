package shared

import (
	"context"
)

// Specification defines the interface for domain specifications
// A specification encapsulates business rules for querying entities
// DDD principle: Specifications are domain objects that express business constraints
// T is the domain entity type (e.g., *order.Order, *user.User)
type Specification[T any] interface {
	// IsSatisfiedBy checks if an entity satisfies the specification
	// This method is used for in-memory filtering (e.g., in mock repositories)
	IsSatisfiedBy(ctx context.Context, entity T) bool
}

// ============================================================================
// Composite Specifications
// ============================================================================

// AndSpecification represents the logical AND of two specifications
// T is the domain entity type
type AndSpecification[T any] struct {
	Left  Specification[T]
	Right Specification[T]
}

// IsSatisfiedBy returns true if both left and right specifications are satisfied
func (spec AndSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) && spec.Right.IsSatisfiedBy(ctx, entity)
}

// And creates a new AndSpecification
func And[T any](left, right Specification[T]) Specification[T] {
	return AndSpecification[T]{
		Left:  left,
		Right: right,
	}
}

// OrSpecification represents the logical OR of two specifications
type OrSpecification[T any] struct {
	Left  Specification[T]
	Right Specification[T]
}

// IsSatisfiedBy returns true if either left or right specification is satisfied
func (spec OrSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) || spec.Right.IsSatisfiedBy(ctx, entity)
}

// Or creates a new OrSpecification
func Or[T any](left, right Specification[T]) Specification[T] {
	return OrSpecification[T]{
		Left:  left,
		Right: right,
	}
}

// NotSpecification represents the logical NOT of a specification
type NotSpecification[T any] struct {
	Spec Specification[T]
}

// IsSatisfiedBy returns true if the inner specification is NOT satisfied
func (spec NotSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return !spec.Spec.IsSatisfiedBy(ctx, entity)
}

// Not creates a new NotSpecification
func Not[T any](inner Specification[T]) Specification[T] {
	return NotSpecification[T]{
		Spec: inner,
	}
}