package errors

import (
	"errors"
	"fmt"
)

// Common application errors
var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrRateLimit     = errors.New("rate limit exceeded")
	ErrExternalAPI   = errors.New("external API error")
	ErrDatabaseError = errors.New("database error")
	ErrInternalError = errors.New("internal error")
)

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// APIError represents errors from external APIs
type APIError struct {
	Service    string
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s API error (status %d): %s", e.Service, e.StatusCode, e.Message)
}

// CharityNotFoundError represents a specific charity not found error
type CharityNotFoundError struct {
	Number int
}

func (e CharityNotFoundError) Error() string {
	return fmt.Sprintf("charity with number %d not found", e.Number)
}

// Is allows error comparison using errors.Is
func (e CharityNotFoundError) Is(target error) bool {
	return errors.Is(target, ErrNotFound)
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
