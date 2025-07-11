package apperrors

import (
	"errors"
	"fmt"
)

// Common error types for the application
var (
	ErrValidation    = errors.New("validation error")
	ErrNotFound      = errors.New("not found")
	ErrDatabase      = errors.New("database error")
	ErrExternalAPI   = errors.New("external API error")
	ErrConfiguration = errors.New("configuration error")
	ErrInternal      = errors.New("internal error")
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorWithCause creates a new validation error with a cause
func NewValidationErrorWithCause(field, message string, cause error) error {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     cause,
	}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// DatabaseError represents a database-specific error
type DatabaseError struct {
	Operation string
	Message   string
	Err       error
}

func (e *DatabaseError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("database error during %s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("database error: %s", e.Message)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NewDatabaseError creates a new database error
func NewDatabaseError(operation, message string) error {
	return &DatabaseError{
		Operation: operation,
		Message:   message,
	}
}

// NewDatabaseErrorWithCause creates a new database error with a cause
func NewDatabaseErrorWithCause(operation, message string, cause error) error {
	return &DatabaseError{
		Operation: operation,
		Message:   message,
		Err:       cause,
	}
}

// IsDatabaseError checks if an error is a database error
func IsDatabaseError(err error) bool {
	var dbErr *DatabaseError
	return errors.As(err, &dbErr)
}

// ExternalAPIError represents an external API error
type ExternalAPIError struct {
	Service   string
	Operation string
	Message   string
	Err       error
}

func (e *ExternalAPIError) Error() string {
	if e.Service != "" && e.Operation != "" {
		return fmt.Sprintf("external API error (%s %s): %s", e.Service, e.Operation, e.Message)
	}
	return fmt.Sprintf("external API error: %s", e.Message)
}

func (e *ExternalAPIError) Unwrap() error {
	return e.Err
}

// NewExternalAPIError creates a new external API error
func NewExternalAPIError(service, operation, message string) error {
	return &ExternalAPIError{
		Service:   service,
		Operation: operation,
		Message:   message,
	}
}

// NewExternalAPIErrorWithCause creates a new external API error with a cause
func NewExternalAPIErrorWithCause(service, operation, message string, cause error) error {
	return &ExternalAPIError{
		Service:   service,
		Operation: operation,
		Message:   message,
		Err:       cause,
	}
}

// IsExternalAPIError checks if an error is an external API error
func IsExternalAPIError(err error) bool {
	var apiErr *ExternalAPIError
	return errors.As(err, &apiErr)
}
