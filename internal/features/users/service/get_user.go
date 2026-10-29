package users_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func (s *UsersService) GetUser(
	ctx context.Context,
	login string,
	password string,
) (domain.User, error) {
	input := UserInput{
		Login:    login,
		Password: password,
	}
	if err := input.UserValidate(); err != nil {
		return domain.User{}, fmt.Errorf("get user validate input: %w", err)
	}

	user, passwordHash, err := s.usersRepository.GetUser(ctx, login)
	if err != nil {
		return domain.User{}, core_errors.ErrInvalidArgument
	}

	if err := s.passwordHasher.Compare(passwordHash, password); err != nil {
		return domain.User{}, core_errors.ErrInvalidArgument
	}

	return user, nil
}
