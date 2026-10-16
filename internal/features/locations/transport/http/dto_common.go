package locations_transport_http

import "github.com/vladpann/golang-weather/internal/core/domain"

type LocationDTOResponse struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func locationDTOFromDomain(location domain.Location) LocationDTOResponse {
	return LocationDTOResponse{
		ID:        location.ID,
		UserID:    location.UserID,
		Name:      location.Name,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}
}

func locationDTOFromDomains(locations []domain.Location) []LocationDTOResponse {
	locationsDTO := make([]LocationDTOResponse, len(locations))

	for i, location := range locations {
		locationsDTO[i] = LocationDTOResponse(location)
	}

	return locationsDTO
}
