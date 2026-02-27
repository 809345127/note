/*
Package user 定义用户领域错误。
*/
package user

import (
	"ddd/domain/shared"
	"errors"
	"fmt"
)

var (
	ErrInvalidEmail           = errors.New("invalid email format")
	ErrInvalidName            = errors.New("name cannot be empty")
	ErrInvalidAge             = errors.New("age must be between 0 and 150")
	ErrUserNotActive          = errors.New("user is not active")
	ErrInsufficientFunds      = errors.New("insufficient funds")
	ErrConcurrentModification = errors.New("user was modified by another transaction, please retry")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrUserTooYoung           = errors.New("user is too young to make purchases")
)

func NewUserNotFoundError(userID string) error {
	return &userDomainError{
		sentinel: shared.ErrNotFound,
		entity:   "user",
		message:  "user not found: " + userID,
		stack:    shared.CaptureStack(3),
	}
}

func NewConcurrentModificationError(userID string) error {
	return &userDomainError{
		sentinel: ErrConcurrentModification,
		entity:   "user",
		message:  "user " + userID + " was modified by another transaction, please retry",
		stack:    shared.CaptureStack(3),
	}
}

func NewInvalidEmailError(email string) error {
	return &userDomainError{
		sentinel: ErrInvalidEmail,
		entity:   "user",
		field:    "email",
		message:  "invalid email format: " + email,
		stack:    shared.CaptureStack(3),
	}
}

func NewInvalidNameError() error {
	return &userDomainError{
		sentinel: ErrInvalidName,
		entity:   "user",
		field:    "name",
		message:  "name cannot be empty",
		stack:    shared.CaptureStack(3),
	}
}

func NewInvalidAgeError(age int) error {
	return &userDomainError{
		sentinel: ErrInvalidAge,
		entity:   "user",
		field:    "age",
		message:  fmt.Sprintf("age must be between 0 and 150, got: %d", age),
		stack:    shared.CaptureStack(3),
	}
}

func NewUserNotActiveError(userID string) error {
	return &userDomainError{
		sentinel: ErrUserNotActive,
		entity:   "user",
		message:  "user " + userID + " is not active",
		stack:    shared.CaptureStack(3),
	}
}

func NewEmailAlreadyExistsError(email string) error {
	return &userDomainError{
		sentinel: ErrEmailAlreadyExists,
		entity:   "user",
		field:    "email",
		message:  "email already exists: " + email,
		stack:    shared.CaptureStack(3),
	}
}

type userDomainError struct {
	sentinel error
	entity   string
	field    string
	message  string
	stack    []uintptr
}

func (e *userDomainError) Error() string   { return e.message }
func (e *userDomainError) Unwrap() error   { return e.sentinel }
func (e *userDomainError) Stack() []string { return shared.FormatStack(e.stack) }
