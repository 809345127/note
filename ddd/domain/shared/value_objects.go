package shared

import (
	"fmt"
	"math"
)

type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) *Money {
	return &Money{
		amount:   amount,
		currency: currency,
	}
}
func (m Money) Amount() int64 {
	return m.amount
}
func (m Money) Currency() string {
	return m.currency
}
func (m Money) Add(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, NewValidationError("money", "currency",
			"cannot add money with different currencies: "+m.currency+" vs "+other.currency)
	}
	result, overflow := addOverflow(m.amount, other.amount)
	if overflow {
		return nil, NewValidationError("money", "amount", "money addition overflow")
	}

	return &Money{
		amount:   result,
		currency: m.currency,
	}, nil
}
func (m Money) Subtract(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, NewValidationError("money", "currency",
			"cannot subtract money with different currencies: "+m.currency+" vs "+other.currency)
	}
	result, overflow := subOverflow(m.amount, other.amount)
	if overflow {
		return nil, NewValidationError("money", "amount", "money subtraction overflow")
	}

	return &Money{
		amount:   result,
		currency: m.currency,
	}, nil
}
func addOverflow(a, b int64) (int64, bool) {
	if b > 0 {
		return a + b, a > math.MaxInt64-b
	}
	return a + b, a < math.MinInt64-b
}
func subOverflow(a, b int64) (int64, bool) {
	if b < 0 {
		return a + b, a > math.MaxInt64+b
	}
	return a - b, a < math.MinInt64+b
}
func (m Money) Multiply(quantity int) (*Money, error) {
	if quantity == 0 {
		return &Money{amount: 0, currency: m.currency}, nil
	}
	if quantity > 0 {
		if m.amount > 0 && m.amount > math.MaxInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
		if m.amount < 0 && m.amount < math.MinInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
	} else {
		if m.amount > 0 && m.amount < math.MinInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
		if m.amount < 0 && m.amount > math.MaxInt64/int64(quantity) {
			return nil, NewValidationError("money", "amount", "money multiplication overflow")
		}
	}

	return &Money{amount: m.amount * int64(quantity), currency: m.currency}, nil
}
func (m Money) IsGreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, NewValidationError("money", "currency",
			"cannot compare money with different currencies: "+m.currency+" vs "+other.currency)
	}
	return m.amount > other.amount, nil
}
func (m Money) IsGreaterThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, NewValidationError("money", "currency",
			"cannot compare money with different currencies: "+m.currency+" vs "+other.currency)
	}
	return m.amount >= other.amount, nil
}
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
