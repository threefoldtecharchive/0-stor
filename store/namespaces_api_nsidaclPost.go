package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL
	namespace :=r.Context().Value("namespace").(NamespaceCreate)

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//@TODO: return proper Access token
	json.NewEncoder(w).Encode("Access-Token")
}
