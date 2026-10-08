package users_service

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (s *UsersService) GetUser(
	ctx context.Context,
	login string,
	password string,
) (domain.User, error) {
	user, passwordHash, err := s.usersRepository.GetUser(ctx, login)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}

	if err := s.passwordHasher.Compare(passwordHash, password); err != nil {
		return domain.User{}, fmt.Errorf("invalid password: %w", err)
	}

	return user, nil
}
