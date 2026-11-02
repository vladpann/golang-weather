package users_service

import (
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

type UserInput struct {
	Login          string
	Password       string
	RepeatPassword *string
}

func (i *UserInput) UserValidate() error {
	if i.Login == "" || i.Password == "" {
		return core_errors.NewValidationError("login and password are required")
	}

	if len(i.Password) < 4 || len(i.Password) > 72 {
		return core_errors.ErrPasswordConstrains
	}

	if i.RepeatPassword != nil && i.Password != *i.RepeatPassword {
		return core_errors.ErrPasswordDoNotMatch
	}

	return nil
}
