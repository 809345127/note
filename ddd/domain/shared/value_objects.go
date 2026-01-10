package shared

import (
	"fmt"
	"math"
)

// Money Value Object - Represents monetary amount
type Money struct {
	amount   int64  // Stored in smallest currency unit (e.g., cents)
	currency string // Currency code (e.g., CNY, USD)
}

// NewMoney Create New Money Value Object
func NewMoney(amount int64, currency string) *Money {
	return &Money{
		amount:   amount,
		currency: currency,
	}
}

// Amount Get monetary amount
func (m Money) Amount() int64 {
	return m.amount
}

// Currency Get currency type
func (m Money) Currency() string {
	return m.currency
}

// Add Add amounts, return new Money value object
func (m Money) Add(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, NewValidationError("money", "currency",
			"cannot add money with different currencies: "+m.currency+" vs "+other.currency)
	}

	// Check for overflow
	result, overflow := addOverflow(m.amount, other.amount)
	if overflow {
		return nil, NewValidationError("money", "amount", "money addition overflow")
	}

	return &Money{
		amount:   result,
		currency: m.currency,
	}, nil
}

// Subtract Subtract amounts, return new Money value object
func (m Money) Subtract(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, NewValidationError("money", "currency",
			"cannot subtract money with different currencies: "+m.currency+" vs "+other.currency)
	}

	// Check for overflow
	result, overflow := subOverflow(m.amount, other.amount)
	if overflow {
		return nil, NewValidationError("money", "amount", "money subtraction overflow")
	}

	return &Money{
		amount:   result,
		currency: m.currency,
	}, nil
}

// addOverflow checks for int64 addition overflow
func addOverflow(a, b int64) (int64, bool) {
	if b > 0 {
		return a + b, a > math.MaxInt64-b
	}
	return a + b, a < math.MinInt64-b
}

// subOverflow checks for int64 subtraction overflow
func subOverflow(a, b int64) (int64, bool) {
	if b < 0 {
		return a + b, a > math.MaxInt64+b
	}
	return a - b, a < math.MinInt64+b
}

// Multiply Multiply money by a quantity with overflow check
func (m Money) Multiply(quantity int) (*Money, error) {
	if quantity == 0 {
		return &Money{amount: 0, currency: m.currency}, nil
	}

	// Check for overflow before multiplication
	if quantity > 0 {
		if m.amount > 0 && m.amount > math.MaxInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
		if m.amount < 0 && m.amount < math.MinInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
	} else {
		// quantity < 0
		if m.amount > 0 && m.amount < math.MinInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
		if m.amount < 0 && m.amount > math.MaxInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
	}

	return &Money{amount: m.amount * int64(quantity), currency: m.currency}, nil
}

// IsGreaterThan Compare if amount is greater than another amount
// Returns error if currencies don't match
func (m Money) IsGreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, NewValidationError("money", "currency",
			"cannot compare money with different currencies: "+m.currency+" vs "+other.currency)
	}
	return m.amount > other.amount, nil
}

// IsGreaterThanOrEqual Compare if amount is greater than or equal to another amount
// Returns error if currencies don't match
func (m Money) IsGreaterThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, NewValidationError("money", "currency",
			"cannot compare money with different currencies: "+m.currency+" vs "+other.currency)
	}
	return m.amount >= other.amount, nil
}

// Equals Compare if two Money value objects are equal
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

// String Returns string representation of Money
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
