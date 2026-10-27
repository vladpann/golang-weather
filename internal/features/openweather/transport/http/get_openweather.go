package openweather_transport_http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/backend/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

type GetOpenWeatherRequest struct {
	Latitude  float64 `query:"latitude" validate:"required"`
	Longitude float64 `query:"longitude" validate:"required"`
}

type GetOpenWeatherResponse OpenWeatherDTOResponse

func (h *OpenWeatherHTTPHandler) GetOpenWeather(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke GetOpenWeatherLocation handler")

	latitude, err := core_http_request.GetFloat64QueryParam(r, "latitude")
	if err != nil {
		responseHandler.ErrorResponse(err, "invalid or missing latitude parameter")
		return
	}

	longitude, err := core_http_request.GetFloat64QueryParam(r, "longitude")
	if err != nil {
		responseHandler.ErrorResponse(err, "invalid or missing longitude parameter")
		return
	}

	if err := h.rateLimiter.Allow(ctx); err != nil {
		log.Warn("rate limit check failed", zap.Error(err))
		h.handleRateLimitError(responseHandler, err)
		return
	}

	query := url.Values{}
	query.Set("lat", strconv.FormatFloat(latitude, 'f', -1, 64))
	query.Set("lon", strconv.FormatFloat(longitude, 'f', -1, 64))
	query.Set("appid", h.config.APIKey)

	reqURL := fmt.Sprintf("%s/data/2.5/weather?%s", h.config.APIBaseUrl, query.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		responseHandler.ErrorResponse(sanitizeRequestError(err), "failed to create HTTP request")
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		responseHandler.ErrorResponse(sanitizeRequestError(err), "failed to get open weather location")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("failed to get open weather data", zap.Int("status", resp.StatusCode))
		responseHandler.ErrorResponse(
			fmt.Errorf("OpenWeather API returned status %d", resp.StatusCode),
			"failed to get weather data",
		)
		return
	}

	var weatherData GetOpenWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode weather response")
		return
	}

	responseHandler.JSONResponse(weatherData, http.StatusOK)
}
