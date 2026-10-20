package locations_repository_postgres

import "github.com/vladpann/backend/golang-weather/internal/core/domain"

type LocationModel struct {
	ID        int64
	UserID    int64
	Name      string
	Latitude  float64
	Longitude float64
}

func locationDomainFromModel(location LocationModel) domain.Location {
	return domain.NewLocation(
		location.ID,
		location.UserID,
		location.Name,
		location.Latitude,
		location.Longitude,
	)
}

func locationDomainsFromModels(locationModels []LocationModel) []domain.Location {
	userDomains := make([]domain.Location, len(locationModels))

	for i, model := range locationModels {
		userDomains[i] = locationDomainFromModel(model)
	}

	return userDomains
}
