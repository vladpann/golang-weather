package locations_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func (s *LocationsService) GetLocations(
	ctx context.Context,
	userID int64,
	limit *int,
	offset *int,
) ([]domain.Location, error) {
	if limit != nil && *limit < 0 {
		return nil, fmt.Errorf(
			"limit must be non-negative: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if offset != nil && *offset < 0 {
		return nil, fmt.Errorf(
			"offset must be non-negative: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	users, err := s.locationsRepository.GetLocations(
		ctx,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get users from repository: %w", err)
	}

	return users, nil
}
