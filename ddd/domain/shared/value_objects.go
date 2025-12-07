package shared

import "errors"

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
		return nil, errors.New("cannot add money with different currencies")
	}

	return &Money{
		amount:   m.amount + other.amount,
		currency: m.currency,
	}, nil
}

// Subtract Subtract amounts, return new Money value object
func (m Money) Subtract(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot subtract money with different currencies")
	}

	return &Money{
		amount:   m.amount - other.amount,
		currency: m.currency,
	}, nil
}

// IsGreaterThan Compare if amount is greater than another amount
func (m Money) IsGreaterThan(other Money) bool {
	return m.amount > other.amount
}

// IsGreaterThanOrEqual Compare if amount is greater than or equal to another amount
func (m Money) IsGreaterThanOrEqual(other Money) bool {
	return m.amount >= other.amount
}

// Equals Compare if two Money value objects are equal
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}
