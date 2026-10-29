package users_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *UsersService) GetUser(
	ctx context.Context,
	login string,
	password string,
) (domain.User, error) {
	var op = "get user"

	input := UserInput{
		Login:    login,
		Password: password,
	}
	if err := input.UserValidate(); err != nil {
		return domain.User{}, fmt.Errorf("%s validate input: %w", op, err)
	}

	user, passwordHash, err := s.usersRepository.GetUser(ctx, login)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s repository: %w", op, err)
	}

	if err := s.passwordHasher.Compare(passwordHash, password); err != nil {
		return domain.User{}, fmt.Errorf("%s password hash: %w", op, err)
	}

	return user, nil
}
