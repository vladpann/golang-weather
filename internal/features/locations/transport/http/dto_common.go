package locations_transport_http

import "github.com/vladpann/golang-weather/internal/core/domain"

type LocationResponse struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func locationResponseFromDomain(location domain.Location) LocationResponse {
	return LocationResponse{
		ID:        location.ID,
		UserID:    location.UserID,
		Name:      location.Name,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}
}
