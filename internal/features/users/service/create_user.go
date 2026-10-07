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
	// TODO: fix with Err
	if password != repeatPassword {
		return domain.User{}, fmt.Errorf("password do not match")
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.usersRepository.CreateUser(ctx, login, passwordHash)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain.User{}, core_errors.ErrLoginAlreadyExists
		}

		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
