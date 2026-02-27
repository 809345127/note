package user

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type Email struct {
	value string
}

func NewEmail(email string) (*Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	return &Email{value: email}, nil
}
func (e Email) Value() string {
	return e.value
}
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}
func (e Email) String() string {
	return e.value
}
