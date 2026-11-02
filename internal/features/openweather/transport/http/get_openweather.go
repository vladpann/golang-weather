package openweather_transport_http

import (
	"errors"
	"net/http"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/backend/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	openweather_client "github.com/vladpann/backend/golang-weather/internal/features/openweather/client"
	"go.uber.org/zap"
)

type GetOpenWeatherRequest struct {
	Latitude  float64 `query:"latitude" validate:"required"`
	Longitude float64 `query:"longitude" validate:"required"`
}

type GetOpenWeatherResponse openweather_client.WeatherResponse

func (h *OpenWeatherHTTPHandler) GetOpenWeather(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke GetOpenWeather handler")

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

	weatherData, err := h.client.GetWeather(ctx, latitude, longitude)
	if err != nil {
		log.Warn("failed to get weather data", zap.Error(err))
		h.handleUpstreamError(responseHandler, err)
		return
	}

	responseHandler.JSONResponse(weatherData, http.StatusOK)
}

// handleUpstreamError converts upstream client errors to appropriate HTTP responses
func (h *OpenWeatherHTTPHandler) handleUpstreamError(
	responseHandler *core_http_response.HTTPResponseHandler,
	err error,
) {
	var upstreamErr *openweather_client.UpstreamError
	if errors.As(err, &upstreamErr) {
		status := upstreamErr.HTTPStatusCode()

		response := map[string]interface{}{
			"error":   http.StatusText(status),
			"message": "Unable to retrieve weather data from provider",
		}

		// Add retry-after if available
		if upstreamErr.RetryAfter > 0 {
			response["retry_after_seconds"] = upstreamErr.RetryAfter
		}

		// Add more specific messages based on error type
		switch {
		case errors.Is(upstreamErr.Err, openweather_client.ErrRateLimitExceeded):
			response["message"] = "Weather provider rate limit exceeded, please try again later"
		case errors.Is(upstreamErr.Err, openweather_client.ErrServiceUnavailable):
			response["message"] = "Weather provider is temporarily unavailable"
		case errors.Is(upstreamErr.Err, openweather_client.ErrGatewayTimeout):
			response["message"] = "Weather provider request timed out"
		case errors.Is(upstreamErr.Err, openweather_client.ErrNotFound):
			response["message"] = "Weather data not found for the specified coordinates"
			status = http.StatusNotFound
		case errors.Is(upstreamErr.Err, openweather_client.ErrBadRequest):
			response["message"] = "Invalid coordinates provided"
			status = http.StatusBadRequest
		}

		responseHandler.JSONResponse(response, status)
		return
	}

	// For non-upstream errors (context errors, etc.)
	responseHandler.ErrorResponse(err, "failed to get weather data")
}
