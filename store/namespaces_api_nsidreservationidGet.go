package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"
)

// nsidreservationidGet is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) nsidreservationidGet(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.Reservation

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

}
