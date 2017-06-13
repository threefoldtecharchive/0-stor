package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// GetObject is the handler for GET /namespaces/{nsid}/objects/{id}
// Retrieve object from the store
func (api NamespacesAPI) GetObject(w http.ResponseWriter, r *http.Request) {

	var file = &File{}

	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

	value, err := api.db.Get(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if value == nil {
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(file.ToObject(value, id))
}
