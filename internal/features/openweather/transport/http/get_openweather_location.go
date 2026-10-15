package openweather_transport_http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/golang-weather/internal/core/transport/http/middleware"
	core_http_request "github.com/vladpann/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

type GetOpenWeatherLocationRequest struct {
	CityName string `json:"city_name"`
}

type GetOpenWeatherLocationResponse OpenWeatherLocationDTOResponse

func (h *OpenWeatherHTTPHandler) GetOpenWeatherLocation(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(rw)

	log.Debug("invoke GetOpenWeatherLocation handler")

	var request GetOpenWeatherLocationRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")

		return
	}

	// Check rate limit before making API request
	if err := h.rateLimiter.Allow(ctx); err != nil {
		log.Warn("rate limit check failed", zap.Error(err))
		h.handleRateLimitError(responseHandler, err)
		return
	}

	url := fmt.Sprintf(
		"%s/data/2.5/weather?q=%s&appid=%s",
		h.config.APIBaseUrl,
		request.CityName,
		h.config.APIKey,
	)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create HTTP request")

		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get open weather location")

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("failed to get open weather data")
		responseHandler.ErrorResponse(fmt.Errorf("OpenWeather API returned status %d", resp.StatusCode), "failed to get weather data")

		return
	}

	var weatherData GetOpenWeatherLocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode weather response")

		return
	}

	userID, ok := core_http_middleware.GetUserIDFromContext(ctx)
	if !ok {
		responseHandler.ErrorResponse(nil, "user not authenticated")
		return
	}

	location := domainLocationFromDTO(OpenWeatherLocationDTOResponse(weatherData), userID)

	response, err := h.locationsService.CreateLocation(ctx, location)
	if err != nil {
		log.Debug("failed to create weather location")
		responseHandler.ErrorResponse(err, "failed to create weather location")

		return
	}

	responseHandler.JSONResponse(response, http.StatusOK)
}

func domainLocationFromDTO(weatherData OpenWeatherLocationDTOResponse, userID int64) domain.Location {
	return domain.Location{
		UserID:    userID,
		Name:      weatherData.Name,
		Latitude:  weatherData.Coord.Lat,
		Longitude: weatherData.Coord.Lon,
	}
}
