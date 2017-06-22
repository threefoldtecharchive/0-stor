package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	log "github.com/Sirupsen/logrus"
)

// nsidreservationidGet is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) nsidreservationidGet(w http.ResponseWriter, r *http.Request) {
	var respBody Reservation

	nsid := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["nsid"]

	key := fmt.Sprintf("%s%s_%s", api.config.Reservations.Namespaces.Prefix, nsid, id)

	value, err := api.db.Get(key)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if value == nil{
		http.Error(w, "Namespace or Reservation not found", http.StatusNotFound)
		return
	}

	respBody.FromBytes(value)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
