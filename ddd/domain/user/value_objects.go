package user

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Email 值对象 - 不可变，表示电子邮件地址
type Email struct {
	value string
}

// NewEmail 创建新的Email值对象
func NewEmail(email string) (*Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	return &Email{value: email}, nil
}

// Value 获取邮箱值
func (e Email) Value() string {
	return e.value
}

// Equals 比较两个Email值对象是否相等
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// String 实现Stringer接口
func (e Email) String() string {
	return e.value
}
