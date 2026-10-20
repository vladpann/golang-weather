package locations_service

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *LocationsService) GetLocation(
	ctx context.Context,
	id int64,
	userID int64,
) (domain.Location, error) {
	location, err := s.locationsRepository.GetLocation(ctx, id, userID)
	if err != nil {
		return domain.Location{}, fmt.Errorf("get location in repository: %w", err)
	}

	return location, nil
}
