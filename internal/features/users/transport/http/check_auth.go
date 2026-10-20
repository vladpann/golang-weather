package users_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

type CheckAuthResponse struct {
	Authenticated bool    `json:"authenticated"`
	UserID        *int64  `json:"user_id,omitempty"`
	Login         *string `json:"login,omitempty"`
}

func (h *UsersHTTPHandler) CheckAuth(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke CheckAuth handler")

	response := CheckAuthResponse{
		Authenticated: true,
	}

	userID, ok := core_http_middleware.GetUserIDFromContext(ctx)
	if !ok {
		log.Debug("no user ID in context")
	}

	log.Debug("got user ID from context", zap.Int64("user_id", userID))
	user, err := h.usersService.GetUserByID(ctx, userID)
	if err != nil {
		log.Error("failed to get user by id", zap.Int64("user_id", userID), zap.Error(err))
	} else {
		log.Debug("got user from service", zap.String("login", user.Login))
		response.UserID = &user.ID
		response.Login = &user.Login
	}

	responseHandler.JSONResponse(response, http.StatusOK)
}
