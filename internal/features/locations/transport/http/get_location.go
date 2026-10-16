package locations_transport_http

import "net/http"

type GetLocationResponse struct {
	Location LocationResponse `json:"locations"`
}

func (h *LocationsHTTPHandler) GetLocation(rw http.ResponseWriter, r *http.Request) {
	// TODO: implement
	http.Error(rw, "Not implemented", http.StatusNotImplemented)
}
