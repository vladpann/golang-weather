package locations_service

import (
	"context"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

type LocationsService struct {
	locationsRepository LocationsRepository
}

type LocationsRepository interface {
	CreateLocation(
		ctx context.Context,
		location domain.Location,
	) (domain.Location, error)

	GetLocation(
		ctx context.Context,
		id int64,
		userID int64,
	) (domain.Location, error)

	GetLocations(
		ctx context.Context,
		limit *int,
		offset *int,
	) ([]domain.Location, error)

	DeleteLocation(
		ctx context.Context,
		id int64,
	) error
}

func NewLocationsService(
	locationsRepository LocationsRepository,
) *LocationsService {
	return &LocationsService{
		locationsRepository: locationsRepository,
	}
}
