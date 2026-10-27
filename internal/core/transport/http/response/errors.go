package core_http_response

type ErrorResponse struct {
	Error   string `json:"error,omitempty" example:"not found"`
	Message string `json:"message" example:"short human-readable message"`
}
