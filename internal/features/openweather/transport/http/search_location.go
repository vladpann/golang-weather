package openweather_transport_http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

type GeocodingResponse struct {
	Name       string            `json:"name"`
	Lat        float64           `json:"lat"`
	Lon        float64           `json:"lon"`
	Country    string            `json:"country"`
	State      string            `json:"state,omitempty"`
	LocalNames map[string]string `json:"local_names,omitempty"`
}

func (h *OpenWeatherHTTPHandler) SearchLocation(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke SearchLocation handler")

	cityName := r.URL.Query().Get("city_name")
	if cityName == "" {
		responseHandler.ErrorResponse(core_errors.ErrCityNameRequired, "city_name parameter is required")
		return
	}

	if err := h.rateLimiter.Allow(ctx); err != nil {
		log.Warn("rate limit check failed", zap.Error(err))
		h.handleRateLimitError(responseHandler, err)
		return
	}

	query := url.Values{}
	query.Set("q", cityName)
	query.Set("limit", "5")
	query.Set("appid", h.config.APIKey)

	apiURL := fmt.Sprintf("%s/geo/1.0/direct?%s", h.config.APIBaseUrl, query.Encode())

	log.Debug("calling OpenWeather Geocoding API", zap.String("city", cityName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		responseHandler.ErrorResponse(sanitizeRequestError(err), "failed to create HTTP request")
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		responseHandler.ErrorResponse(sanitizeRequestError(err), "failed to search locations")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("OpenWeather API error",
			zap.Int("status", resp.StatusCode),
			zap.String("city", cityName))
		responseHandler.ErrorResponse(
			fmt.Errorf("OpenWeather API returned status %d", resp.StatusCode),
			"failed to search locations",
		)
		return
	}

	var locations []GeocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode geocoding response")
		return
	}

	responseHandler.JSONResponse(locations, http.StatusOK)
}
