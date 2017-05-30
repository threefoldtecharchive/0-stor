package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"
)

// CreateReservation is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	var respBody NamespacesNsidReservationPostRespBody
	json.NewEncoder(w).Encode(&respBody)
}
