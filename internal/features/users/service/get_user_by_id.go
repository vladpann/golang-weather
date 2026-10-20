package users_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *UsersService) GetUserByID(
	ctx context.Context,
	userID int64,
) (domain.User, error) {
	user, err := s.usersRepository.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}
