package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL

	nsid := mux.Vars(r)["nsid"]
	namespace := NamespaceCreate{
		Label: nsid,
	}

	v, err :=  namespace.Get(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	reservation :=r.Context().Value("reservation").(*Reservation)

	// decode request
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Update name space
	if err := namespace.UpdateACL(api.db, api.config, reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser(reqBody.Id, namespace.Label, reqBody.Acl)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(dataToken)
}
