/*
Package shared 提供领域层通用错误与堆栈能力。
*/
package shared

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type DomainError struct {
	Err     error
	Entity  string
	Message string
	Field   string
	stack   []uintptr
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }
func (e *DomainError) Stack() []string {
	return FormatStack(e.stack)
}

func CaptureStack(skip int) []uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[:n]
}

func FormatStack(stack []uintptr) []string {
	if len(stack) == 0 {
		return nil
	}

	frames := runtime.CallersFrames(stack)
	result := make([]string, 0, 10)
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			result = append(result, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		}
		if !more || len(result) > 10 {
			break
		}
	}
	return result
}

func NewNotFoundError(entity string) error {
	return &DomainError{Err: ErrNotFound, Entity: entity, Message: entity + " not found", stack: CaptureStack(3)}
}

func NewConflictError(entity, message string) error {
	return &DomainError{Err: ErrConflict, Entity: entity, Message: message, stack: CaptureStack(3)}
}

func NewValidationError(entity, field, reason string) error {
	return &DomainError{Err: ErrInvalidInput, Entity: entity, Field: field, Message: reason, stack: CaptureStack(3)}
}

func NewForbiddenError(entity, reason string) error {
	return &DomainError{Err: ErrForbidden, Entity: entity, Message: reason, stack: CaptureStack(3)}
}

type Stacker interface {
	Stack() []string
}
