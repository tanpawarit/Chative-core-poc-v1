package errx

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	// SystemErrorMessage is a user-facing fallback when internal errors occur.
	SystemErrorMessage = "internal server error"
	// RedisErrorMessage describes Redis related failures.
	RedisErrorMessage = "redis operation failed"
	// RedisNotFoundMessage is returned when a Redis key is missing.
	RedisNotFoundMessage = "record not found"
)

// Error wraps an underlying error with an HTTP status code and safe message.
type Error struct {
	Err     error
	Status  int
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err == nil {
		if e.Message == "" {
			return SystemErrorMessage
		}
		return e.Message
	}
	if e.Message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap exposes the wrapped error for errors.Is / errors.As support.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// StatusCode reports the HTTP status code, defaulting to 500.
func (e *Error) StatusCode() int {
	if e == nil || e.Status == 0 {
		return http.StatusInternalServerError
	}
	return e.Status
}

// PublicMessage returns a safe message that can be surfaced to external clients.
func (e *Error) PublicMessage() string {
	if e == nil || e.Message == "" {
		return SystemErrorMessage
	}
	return e.Message
}

// New constructs a new Error from the provided components.
func New(err error, status int, message string) *Error {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if message == "" {
		message = SystemErrorMessage
	}
	return &Error{Err: err, Status: status, Message: message}
}

// AsError attempts to coerce err into an *Error instance.
func AsError(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// Is compares err against a template Error value using status/message fields.
func Is(err error, target *Error) bool {
	if target == nil {
		return errors.Is(err, nil)
	}
	if actual, ok := AsError(err); ok {
		if target.Status != 0 && actual.StatusCode() != target.Status {
			return false
		}
		if target.Message != "" && actual.PublicMessage() != target.Message {
			return false
		}
		return true
	}
	return false
}

var _ error = (*Error)(nil)
