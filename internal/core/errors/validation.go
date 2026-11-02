package core_errors

import (
	"errors"
	"fmt"
)

// NewValidationError создаёт ошибку валидации, которая будет автоматически
// преобразована в HTTP 400 Bad Request в response.Handler.
//
// Все ошибки валидации должны быть обёрнуты в ErrInvalidArgument через эту функцию,
// чтобы ErrorResponse корректно определил HTTP статус-код 400.
//
// Примеры использования:
//
//	if email == "" {
//	    return NewValidationError("email is required")
//	}
//
//	if len(password) < 4 {
//	    return NewValidationError("password must be at least 4 characters")
//	}
func NewValidationError(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInvalidArgument)
}

// NewValidationErrorf создаёт ошибку валидации с форматированием.
//
// Пример использования:
//
//	if age < 0 || age > 150 {
//	    return NewValidationErrorf("age must be between 0 and 150, got: %d", age)
//	}
func NewValidationErrorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", message, ErrInvalidArgument)
}

// IsValidationError проверяет, является ли ошибка ошибкой валидации (HTTP 400).
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}
