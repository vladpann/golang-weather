package users_transport_http

import "github.com/vladpann/golang-weather/internal/core/domain"

type UserDTOResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func userDTOFromDomain(user domain.User) UserDTOResponse {
	return UserDTOResponse{
		ID:    user.ID,
		Login: user.Login,
	}
}
