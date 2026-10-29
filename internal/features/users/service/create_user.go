package users_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *UsersService) CreateUser(
	ctx context.Context,
	login string,
	password string,
	repeatPassword string,
) (domain.User, error) {
	var op = "create user"

	input := UserInput{
		Login:          login,
		Password:       password,
		RepeatPassword: &repeatPassword,
	}
	if err := input.UserValidate(); err != nil {
		return domain.User{}, fmt.Errorf("%s validate input: %w", op, err)
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s password hash: %w", op, err)
	}

	user, err := s.usersRepository.CreateUser(ctx, login, passwordHash)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
