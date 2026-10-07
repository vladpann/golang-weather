package users_transport_http

import (
	"net/http"

	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_request "github.com/vladpann/golang-weather/internal/core/transport/http/request"
	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
)

type CreateUserRequest struct {
	Login          string `json:"login"`
	Password       string `json:"password" validate:"required,min=4,max=72"`
	RepeatPassword string `json:"repeat_password"`
}

type CreateUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func (h *UsersHTTPHandler) CreateUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(rw)

	log.Debug("invoke CreateUser handler")

	var request CreateUserRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")

		return
	}

	login, password, repeatPassword := domainFromDTO(request)

	userDomain, err := h.usersService.CreateUser(ctx, login, password, repeatPassword)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create user")

		return
	}

	response := CreateUserResponse(userDTOFromDomain(userDomain))

	responseHandler.JSONResponse(response, http.StatusCreated)
}

func domainFromDTO(dto CreateUserRequest) (login string, password string, repeatPassword string) {
	return dto.Login, dto.Password, dto.RepeatPassword
}
