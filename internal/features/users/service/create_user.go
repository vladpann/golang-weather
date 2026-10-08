package users_service

import (
	"context"
	"errors"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/golang-weather/internal/core/errors"
	core_postgres_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool"
)

func (s *UsersService) CreateUser(
	ctx context.Context,
	login string,
	password string,
	repeatPassword string,
) (domain.User, error) {
	var op = "create user"

	input := CreateUserInput{
		Login:          login,
		Password:       password,
		RepeatPassword: repeatPassword,
	}
	if err := input.CreateUserValidate(); err != nil {
		return domain.User{}, fmt.Errorf("%s validate input: %w", op, err)
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s password hash: %w", op, err)
	}

	user, err := s.usersRepository.CreateUser(ctx, login, passwordHash)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain.User{}, core_errors.ErrLoginAlreadyExists
		}

		return domain.User{}, fmt.Errorf("%s %w:", op, err)
	}

	return user, nil
}
