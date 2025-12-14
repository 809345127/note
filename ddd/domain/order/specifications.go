package order

import (
	"context"
	"time"

	"ddd/domain/shared"
)

// ByUserIDSpecification filters orders by user ID
type ByUserIDSpecification struct {
	UserID string
}

// IsSatisfiedBy returns true if the order belongs to the specified user
func (spec ByUserIDSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	order, ok := entity.(*Order)
	if !ok {
		return false
	}
	return order.UserID() == spec.UserID
}

// ByStatusSpecification filters orders by status
type ByStatusSpecification struct {
	Status Status
}

// IsSatisfiedBy returns true if the order has the specified status
func (spec ByStatusSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	order, ok := entity.(*Order)
	if !ok {
		return false
	}
	return order.Status() == spec.Status
}

// ByDateRangeSpecification filters orders by creation date range
// Both Start and End are optional - if zero, they are ignored
type ByDateRangeSpecification struct {
	Start time.Time
	End   time.Time
}

// IsSatisfiedBy returns true if the order was created within the date range
func (spec ByDateRangeSpecification) IsSatisfiedBy(ctx context.Context, entity interface{}) bool {
	order, ok := entity.(*Order)
	if !ok {
		return false
	}
	createdAt := order.CreatedAt()

	// Check start date (if specified)
	if !spec.Start.IsZero() && createdAt.Before(spec.Start) {
		return false
	}

	// Check end date (if specified)
	if !spec.End.IsZero() && createdAt.After(spec.End) {
		return false
	}

	return true
}

// Helper functions for common specifications

// NewByUserIDSpecification creates a specification to filter by user ID
func NewByUserIDSpecification(userID string) shared.Specification {
	return ByUserIDSpecification{UserID: userID}
}

// NewByStatusSpecification creates a specification to filter by status
func NewByStatusSpecification(status Status) shared.Specification {
	return ByStatusSpecification{Status: status}
}

// NewByDateRangeSpecification creates a specification to filter by date range
func NewByDateRangeSpecification(start, end time.Time) shared.Specification {
	return ByDateRangeSpecification{Start: start, End: end}
}