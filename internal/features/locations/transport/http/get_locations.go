package locations_transport_http

type GetLocationsResponse struct {
	Locations []LocationResponse `json:"locations"`
}

type LocationResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
