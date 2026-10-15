package locations_transport_http

import "net/http"

func (h *LocationsHTTPHandler) DeleteLocation(rw http.ResponseWriter, r *http.Request) {
	// TODO: implement
	http.Error(rw, "Not implemented", http.StatusNotImplemented)
}

