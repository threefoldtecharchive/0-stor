package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {
	var namespace NamespaceCreate

	key := mux.Vars(r)["nsid"]

	value, err := api.db.Get(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if value == nil {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	// No need to handle errors, we assume data is saved correctly
	json.Unmarshal(value, &namespace)

	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
