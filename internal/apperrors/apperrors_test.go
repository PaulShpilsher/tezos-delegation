package apperrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	t.Run("new validation error", func(t *testing.T) {
		err := NewValidationError("field", "must be positive")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation error for field 'field': must be positive")
	})

	t.Run("new validation error with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewValidationErrorWithCause("field", "must be positive", cause)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation error for field 'field': must be positive")
		assert.Equal(t, cause, errors.Unwrap(err))
	})

	t.Run("is validation error", func(t *testing.T) {
		err := NewValidationError("field", "must be positive")
		assert.True(t, IsValidationError(err))
		assert.False(t, IsValidationError(errors.New("other error")))
	})
}

func TestDatabaseError(t *testing.T) {
	t.Run("new database error", func(t *testing.T) {
		err := NewDatabaseError("query", "connection failed")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error during query: connection failed")
	})

	t.Run("new database error with cause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewDatabaseErrorWithCause("query", "connection failed", cause)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error during query: connection failed")
		assert.Equal(t, cause, errors.Unwrap(err))
	})

	t.Run("is database error", func(t *testing.T) {
		err := NewDatabaseError("query", "connection failed")
		assert.True(t, IsDatabaseError(err))
		assert.False(t, IsDatabaseError(errors.New("other error")))
	})
}

func TestExternalAPIError(t *testing.T) {
	t.Run("new external API error", func(t *testing.T) {
		err := NewExternalAPIError("tzkt", "GET", "rate limited")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "external API error (tzkt GET): rate limited")
	})

	t.Run("new external API error with cause", func(t *testing.T) {
		cause := errors.New("network timeout")
		err := NewExternalAPIErrorWithCause("tzkt", "GET", "rate limited", cause)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "external API error (tzkt GET): rate limited")
		assert.Equal(t, cause, errors.Unwrap(err))
	})

	t.Run("is external API error", func(t *testing.T) {
		err := NewExternalAPIError("tzkt", "GET", "rate limited")
		assert.True(t, IsExternalAPIError(err))
		assert.False(t, IsExternalAPIError(errors.New("other error")))
	})
}

func TestErrorConstants(t *testing.T) {
	assert.NotNil(t, ErrValidation)
	assert.NotNil(t, ErrNotFound)
	assert.NotNil(t, ErrDatabase)
	assert.NotNil(t, ErrExternalAPI)
	assert.NotNil(t, ErrConfiguration)
	assert.NotNil(t, ErrInternal)
}
