package users_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

func (h *UsersHTTPHandler) SignOutUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke SignOutUser handler")

	// Clear the session cookie
	http.SetCookie(rw, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	responseHandler.NoContentResponse()
}
