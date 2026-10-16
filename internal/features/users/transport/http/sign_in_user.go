package users_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

type SignInUserRequest struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required,min=4,max=72"`
}

func (h *UsersHTTPHandler) SignInUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	log.Debug("invoke SignInUser handler")

	var request SignInUserRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		log.Error("failed to decode and validate HTTP request", zap.Error(err))
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")

		return
	}

	log.Debug("attempting to sign in user", zap.String("login", request.Login))

	login, password := domainFromSignInUserDTO(request)

	user, err := h.usersService.GetUser(ctx, login, password)
	if err != nil {
		log.Error("failed to sign-in user", zap.String("login", login), zap.Error(err))
		responseHandler.ErrorResponse(err, "failed to sign-in user")

		return
	}

	log.Debug("user authenticated successfully", zap.Int64("user_id", user.ID))

	session, err := h.sessionsService.CreateSession(ctx, user.ID)
	if err != nil {
		log.Error("failed to create user session", zap.Int64("user_id", user.ID), zap.Error(err))
		responseHandler.ErrorResponse(err, "failed to create user session")

		return
	}

	log.Debug("session created successfully", zap.String("session_id", session.ID.String()))

	http.SetCookie(rw, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID.String(),
		Path:     "/",
		HttpOnly: true,
		Expires:  session.ExpiresAt,
	})

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func domainFromSignInUserDTO(dto SignInUserRequest) (login string, password string) {
	return dto.Login, dto.Password
}
