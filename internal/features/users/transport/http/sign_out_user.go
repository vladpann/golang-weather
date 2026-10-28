package users_transport_http

import (
	"net/http"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

func (h *UsersHTTPHandler) SignOutUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke SignOutUser handler")

	sessionID, ok := core_http_middleware.GetSessionIDFromContext(ctx)
	if !ok {
		log.Debug("no session ID in context")
	}

	err := h.sessionsService.DeleteSession(ctx, sessionID)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to delete session")

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	responseHandler.ErrorResponse(core_errors.ErrInvalidCredentials, "user is not authorised")
}
