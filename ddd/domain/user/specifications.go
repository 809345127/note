package user

import (
	"context"

	"ddd/domain/shared"
)

type ByEmailSpecification struct {
	Email string
}

func (spec ByEmailSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	return entity.Email().Value() == spec.Email
}

type ByStatusSpecification struct {
	Active bool
}

func (spec ByStatusSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	return entity.IsActive() == spec.Active
}

type ByAgeRangeSpecification struct {
	Min int
	Max int
}

func (spec ByAgeRangeSpecification) IsSatisfiedBy(ctx context.Context, entity *User) bool {
	age := entity.Age()
	if spec.Min > 0 && age < spec.Min {
		return false
	}
	if spec.Max > 0 && age > spec.Max {
		return false
	}

	return true
}
func NewByEmailSpecification(email string) shared.Specification[*User] {
	return ByEmailSpecification{Email: email}
}
func NewByStatusSpecification(active bool) shared.Specification[*User] {
	return ByStatusSpecification{Active: active}
}
func NewByAgeRangeSpecification(min, max int) shared.Specification[*User] {
	return ByAgeRangeSpecification{Min: min, Max: max}
}
