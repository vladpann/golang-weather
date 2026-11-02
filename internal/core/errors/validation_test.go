package core_errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func TestNewValidationError(t *testing.T) {
	t.Run("создаёт ошибку, обёрнутую в ErrInvalidArgument", func(t *testing.T) {
		err := core_errors.NewValidationError("email is required")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, core_errors.ErrInvalidArgument))
		assert.Contains(t, err.Error(), "email is required")
	})

	t.Run("работает с IsValidationError", func(t *testing.T) {
		err := core_errors.NewValidationError("invalid input")

		assert.True(t, core_errors.IsValidationError(err))
	})
}

func TestNewValidationErrorf(t *testing.T) {
	t.Run("форматирует сообщение и оборачивает в ErrInvalidArgument", func(t *testing.T) {
		age := 200
		err := core_errors.NewValidationErrorf("age must be between 0 and 150, got: %d", age)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, core_errors.ErrInvalidArgument))
		assert.Contains(t, err.Error(), "age must be between 0 and 150, got: 200")
	})
}

func TestIsValidationError(t *testing.T) {
	t.Run("возвращает true для ошибок валидации", func(t *testing.T) {
		err := core_errors.NewValidationError("test")
		assert.True(t, core_errors.IsValidationError(err))
	})

	t.Run("возвращает true для ErrInvalidArgument", func(t *testing.T) {
		assert.True(t, core_errors.IsValidationError(core_errors.ErrInvalidArgument))
	})

	t.Run("возвращает false для других sentinel errors", func(t *testing.T) {
		assert.False(t, core_errors.IsValidationError(core_errors.ErrNotFound))
		assert.False(t, core_errors.IsValidationError(core_errors.ErrConflict))
	})

	t.Run("возвращает false для произвольных ошибок", func(t *testing.T) {
		err := errors.New("random error")
		assert.False(t, core_errors.IsValidationError(err))
	})

	t.Run("возвращает false для nil", func(t *testing.T) {
		assert.False(t, core_errors.IsValidationError(nil))
	})
}
