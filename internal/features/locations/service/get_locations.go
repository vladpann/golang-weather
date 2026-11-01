package locations_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *LocationsService) GetLocations(
	ctx context.Context,
	userID int64,
	limit int,
	offset int,
) ([]domain.Location, error) {
	users, err := s.locationsRepository.GetLocations(
		ctx,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get locations from repository: %w", err)
	}

	return users, nil
}
