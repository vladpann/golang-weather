package users_transport_http

type CreateUserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}
