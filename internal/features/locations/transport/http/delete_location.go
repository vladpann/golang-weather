package locations_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
)

func (h *LocationsHTTPHandler) DeleteLocation(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke DeleteLocation handler")

	id, err := core_http_request.GetInt64PathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get location id path value",
		)

		return
	}

	err = h.locationsService.DeleteLocation(ctx, id)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to delete location")

		return
	}

	responseHandler.NoContentResponse()
}
