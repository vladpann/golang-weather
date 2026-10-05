package users_transport_http

type GetUserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type GetUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}
