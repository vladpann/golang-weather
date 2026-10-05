package openweather_transport_http

type GetOpenWeatherRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GetOpenWeatherResponse OpenWeatherDTOResponse
