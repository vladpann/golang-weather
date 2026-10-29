package locations_transport_http

import (
	"net/http"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_request "github.com/vladpann/backend/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

type CreateLocationRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=255"`
	Latitude  float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
}

func (h *LocationsHTTPHandler) CreateLocation(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke CreateLocation handler")

	userID, ok := core_http_middleware.GetUserIDFromContext(ctx)
	if !ok {
		responseHandler.ErrorResponse(nil, "user not authenticated")

		return
	}

	var request CreateLocationRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")

		return
	}

	location := locationDomainFromDTO(request, userID)

	response, err := h.locationsService.CreateLocation(ctx, location)
	if err != nil {
		log.Debug("failed to create location")
		responseHandler.ErrorResponse(err, "failed to create location")

		return
	}

	responseHandler.JSONResponse(response, http.StatusCreated)
}

func locationDomainFromDTO(dto CreateLocationRequest, userID int64) domain.Location {
	return domain.Location{
		UserID:    userID,
		Name:      dto.Name,
		Latitude:  dto.Latitude,
		Longitude: dto.Longitude,
	}
}
