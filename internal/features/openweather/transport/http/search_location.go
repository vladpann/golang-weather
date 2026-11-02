package openweather_transport_http

import (
	"errors"
	"net/http"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	openweather_client "github.com/vladpann/backend/golang-weather/internal/features/openweather/client"
	"go.uber.org/zap"
)

type GeocodingResponse openweather_client.GeocodingResponse

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

	log.Debug("searching locations", zap.String("city", cityName))

	locations, err := h.client.SearchLocation(ctx, cityName, 5)
	if err != nil {
		log.Warn("failed to search locations", zap.Error(err))

		var upstreamErr *openweather_client.UpstreamError
		if errors.As(err, &upstreamErr) {
			status := upstreamErr.HTTPStatusCode()

			response := map[string]interface{}{
				"error":   http.StatusText(status),
				"message": "Unable to search locations from provider",
			}

			if upstreamErr.RetryAfter > 0 {
				response["retry_after_seconds"] = upstreamErr.RetryAfter
			}

			switch {
			case errors.Is(upstreamErr.Err, openweather_client.ErrRateLimitExceeded):
				response["message"] = "Weather provider rate limit exceeded, please try again later"
			case errors.Is(upstreamErr.Err, openweather_client.ErrServiceUnavailable):
				response["message"] = "Weather provider is temporarily unavailable"
			case errors.Is(upstreamErr.Err, openweather_client.ErrNotFound):
				response["message"] = "No locations found"
				status = http.StatusNotFound
			}

			responseHandler.JSONResponse(response, status)
			return
		}

		responseHandler.ErrorResponse(err, "failed to search locations")
		return
	}

	responseHandler.JSONResponse(locations, http.StatusOK)
}
