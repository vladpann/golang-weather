package openweather_transport_http

type GetOpenWeatherLocationRequest struct {
	CityName string
}

type GetOpenWeatherLocationResponse OpenWeatherDTOResponse
