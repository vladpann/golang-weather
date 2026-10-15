package locations_service

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (s *LocationsService) CreateLocation(
	ctx context.Context,
	location domain.Location,
) (domain.Location, error) {
	createdLocation, err := s.locationsRepository.CreateLocation(ctx, location)
	if err != nil {
		return domain.Location{}, fmt.Errorf("create location in repository: %w", err)
	}

	return createdLocation, nil
}
