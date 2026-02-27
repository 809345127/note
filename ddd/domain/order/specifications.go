package order

import (
	"context"
	"time"

	"ddd/domain/shared"
)

type ByUserIDSpecification struct {
	UserID string
}

func (spec ByUserIDSpecification) IsSatisfiedBy(ctx context.Context, entity *Order) bool {
	return entity.UserID() == spec.UserID
}

type ByStatusSpecification struct {
	Status Status
}

func (spec ByStatusSpecification) IsSatisfiedBy(ctx context.Context, entity *Order) bool {
	return entity.Status() == spec.Status
}

type ByDateRangeSpecification struct {
	Start time.Time
	End   time.Time
}

func (spec ByDateRangeSpecification) IsSatisfiedBy(ctx context.Context, entity *Order) bool {
	createdAt := entity.CreatedAt()
	if !spec.Start.IsZero() && createdAt.Before(spec.Start) {
		return false
	}
	if !spec.End.IsZero() && createdAt.After(spec.End) {
		return false
	}

	return true
}
func NewByUserIDSpecification(userID string) shared.Specification[*Order] {
	return ByUserIDSpecification{UserID: userID}
}
func NewByStatusSpecification(status Status) shared.Specification[*Order] {
	return ByStatusSpecification{Status: status}
}
func NewByDateRangeSpecification(start, end time.Time) shared.Specification[*Order] {
	return ByDateRangeSpecification{Start: start, End: end}
}
