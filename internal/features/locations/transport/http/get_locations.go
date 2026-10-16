package locations_transport_http

import "net/http"

type GetLocationsResponse struct {
	Locations []LocationResponse `json:"locations"`
}

func (h *LocationsHTTPHandler) GetLocations(rw http.ResponseWriter, r *http.Request) {
	// TODO: implement
	http.Error(rw, "Not implemented", http.StatusNotImplemented)
}
