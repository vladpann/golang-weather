package users_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
)

type SignInUserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password" validate:"required,min=4,max=72"`
}

func (h *UsersHTTPHandler) SignInUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(rw)

	log.Debug("invoke SignInUser handler")

	var request SignInUserRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")

		return
	}

	login, password := domainFromSignInUserDTO(request)

	user, err := h.usersService.GetUser(ctx, login, password)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to sign-in user")

		return
	}

	session, err := h.sessionsService.CreateSession(ctx, user.ID)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create user session")

		return
	}

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
