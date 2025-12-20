package user

import (
	"context"

	"ddd/domain/shared"
)

// ByEmailSpecification filters users by email address
type ByEmailSpecification struct {
	Email string
}

// IsSatisfiedBy returns true if the user has the specified email
func (spec ByEmailSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	return entity.Email().Value() == spec.Email
}

// ByStatusSpecification filters users by active/inactive status
type ByStatusSpecification struct {
	Active bool
}

// IsSatisfiedBy returns true if the user's active status matches
func (spec ByStatusSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	return entity.IsActive() == spec.Active
}

// ByAgeRangeSpecification filters users by age range
// Both Min and Max are optional - if 0, they are ignored
type ByAgeRangeSpecification struct {
	Min int
	Max int
}

// IsSatisfiedBy returns true if the user's age is within the range
func (spec ByAgeRangeSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	age := entity.Age()

	// Check minimum age (if specified)
	if spec.Min > 0 && age < spec.Min {
		return false
	}

	// Check maximum age (if specified)
	if spec.Max > 0 && age > spec.Max {
		return false
	}

	return true
}

// Helper functions for common specifications

// NewByEmailSpecification creates a specification to filter by email
func NewByEmailSpecification(email string) shared.Specification[*User] {
	return ByEmailSpecification{Email: email}
}

// NewByStatusSpecification creates a specification to filter by active status
func NewByStatusSpecification(active bool) shared.Specification[*User] {
	return ByStatusSpecification{Active: active}
}

// NewByAgeRangeSpecification creates a specification to filter by age range
func NewByAgeRangeSpecification(min, max int) shared.Specification[*User] {
	return ByAgeRangeSpecification{Min: min, Max: max}
}