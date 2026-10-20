package locations_transport_http

import (
	"fmt"
	"net/http"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_request "github.com/vladpann/backend/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

type GetLocationsResponse []LocationDTOResponse

func (h *LocationsHTTPHandler) GetLocations(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke GetLocations handler")

	userID, ok := core_http_middleware.GetUserIDFromContext(ctx)
	if !ok {
		responseHandler.ErrorResponse(nil, "user not authenticated")
		return
	}

	limit, offset, err := getLimitOffsetQueryParams(r)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get 'limit'/'offset' query param",
		)
	}

	locationDomains, err := h.locationsService.GetLocations(ctx, userID, limit, offset)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get locations",
		)

		return
	}

	response := GetLocationsResponse(locationDTOFromDomains(locationDomains))

	responseHandler.JSONResponse(response, http.StatusOK)
}

func getLimitOffsetQueryParams(r *http.Request) (*int, *int, error) {
	const (
		limitQueryParamKey  = "limit"
		offsetQueryParamKey = "offset"
	)

	limit, err := core_http_request.GetIntQueryParam(r, limitQueryParamKey)
	if err != nil {
		return nil, nil, fmt.Errorf("get 'limit' query param: %w", err)
	}

	offset, err := core_http_request.GetIntQueryParam(r, offsetQueryParamKey)
	if err != nil {
		return nil, nil, fmt.Errorf("get 'offset' query param: %w", err)
	}

	return limit, offset, nil
}
