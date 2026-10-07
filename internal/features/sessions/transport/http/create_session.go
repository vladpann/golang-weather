package sessions_transport_http

import "net/http"

func (h *SessionsHTTPHandler) CreateSession(rw http.ResponseWriter, r *http.Request) {
	var userID int64 = 1 // получить авторизованного пользователя

	session, err := h.sessionsService.CreateSession(r.Context(), userID)
	if err != nil {
		// обработка
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
