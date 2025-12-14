package shared

import (
	"context"
)

// Specification defines the interface for domain specifications
// A specification encapsulates business rules for querying entities
// DDD principle: Specifications are domain objects that express business constraints
type Specification interface {
	// IsSatisfiedBy checks if an entity satisfies the specification
	// This method is used for in-memory filtering (e.g., in mock repositories)
	// The entity parameter should be type-asserted to the expected domain type
	IsSatisfiedBy(ctx context.Context, entity interface{}) bool
}

// ============================================================================
// Composite Specifications
// ============================================================================

// AndSpecification represents the logical AND of two specifications
type AndSpecification struct {
	Left  Specification
	Right Specification
}

// IsSatisfiedBy returns true if both left and right specifications are satisfied
func (spec AndSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) && spec.Right.IsSatisfiedBy(ctx, entity)
}

// And creates a new AndSpecification
func And(left, right Specification) Specification {
	return AndSpecification{
		Left:  left,
		Right: right,
	}
}

// OrSpecification represents the logical OR of two specifications
type OrSpecification struct {
	Left  Specification
	Right Specification
}

// IsSatisfiedBy returns true if either left or right specification is satisfied
func (spec OrSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) || spec.Right.IsSatisfiedBy(ctx, entity)
}

// Or creates a new OrSpecification
func Or(left, right Specification) Specification {
	return OrSpecification{
		Left:  left,
		Right: right,
	}
}

// NotSpecification represents the logical NOT of a specification
type NotSpecification struct {
	Spec Specification
}

// IsSatisfiedBy returns true if the inner specification is NOT satisfied
func (spec NotSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	return !spec.Spec.IsSatisfiedBy(ctx, entity)
}

// Not creates a new NotSpecification
func Not(inner Specification) Specification {
	return NotSpecification{
		Spec: inner,
	}
}