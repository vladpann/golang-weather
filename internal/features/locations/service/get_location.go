package locations_service

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (s *LocationsService) GetLocation(
	ctx context.Context,
	location domain.Location,
) (domain.Location, error) {
	location, err := s.locationsRepository.GetLocation(ctx, location)
	if err != nil {
		return domain.Location{}, fmt.Errorf("get location in repository: %w", err)
	}

	return location, nil
}
