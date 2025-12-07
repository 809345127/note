package user

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Email Value object - immutable, represents email address
type Email struct {
	value string
}

// NewEmail Create new Email value object
func NewEmail(email string) (*Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	return &Email{value: email}, nil
}

// Value Get email value
func (e Email) Value() string {
	return e.value
}

// Equals Compare if two Email value objects are equal
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// String Implement Stringer interface
func (e Email) String() string {
	return e.value
}
