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
)

// AppError wraps an underlying error with an HTTP status and safe message.
type AppError struct {
	Err     error
	Status  int
	Message string
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap exposes the underlying error for errors.Is / errors.As support.
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError with the provided information.
func New(err error, status int, message string) *AppError {
	return &AppError{
		Err:     err,
		Status:  status,
		Message: message,
	}
}

// WrapRedis wraps a Redis error with a consistent status code and message.
func WrapRedis(err error) error {
	if err == nil {
		return nil
	}
	return &AppError{
		Err:     err,
		Status:  http.StatusBadGateway,
		Message: RedisErrorMessage,
	}
}

// Is reports whether the target matches the underlying error or the AppError itself.
func (e *AppError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// As allows casting to AppError or the wrapped error in a chain.
func (e *AppError) As(target any) bool {
	if errors.As(e.Err, target) {
		return true
	}
	if t, ok := target.(**AppError); ok {
		*t = e
		return true
	}
	return false
}
