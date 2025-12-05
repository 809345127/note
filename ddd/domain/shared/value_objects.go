package shared

import "errors"

// Money 值对象 - 表示金额
type Money struct {
	amount   int64  // 以最小货币单位存储（如分）
	currency string // 货币代码（如CNY, USD）
}

// NewMoney 创建新的Money值对象
func NewMoney(amount int64, currency string) *Money {
	return &Money{
		amount:   amount,
		currency: currency,
	}
}

// Amount 获取金额数量
func (m Money) Amount() int64 {
	return m.amount
}

// Currency 获取货币类型
func (m Money) Currency() string {
	return m.currency
}

// Add 金额相加，返回新的Money值对象
func (m Money) Add(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot add money with different currencies")
	}

	return &Money{
		amount:   m.amount + other.amount,
		currency: m.currency,
	}, nil
}

// Subtract 金额相减，返回新的Money值对象
func (m Money) Subtract(other Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot subtract money with different currencies")
	}

	return &Money{
		amount:   m.amount - other.amount,
		currency: m.currency,
	}, nil
}

// IsGreaterThan 比较金额是否大于另一个金额
func (m Money) IsGreaterThan(other Money) bool {
	return m.amount > other.amount
}

// IsGreaterThanOrEqual 比较金额是否大于或等于另一个金额
func (m Money) IsGreaterThanOrEqual(other Money) bool {
	return m.amount >= other.amount
}

// Equals 比较两个Money值对象是否相等
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}
