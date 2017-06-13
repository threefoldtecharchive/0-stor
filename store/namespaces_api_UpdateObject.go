package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// UpdateObject is the handler for PUT /namespaces/{nsid}/objects/{id}
// Update oject
func (api NamespacesAPI) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var reqBody ObjectUpdate

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Make sure file contents are valid
	file, err := reqBody.ToFile(true)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

	oldFile, err := api.db.GetFile(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if oldFile == nil {
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
	}

	// Prepend the same value of the first byte of old data
	file.Reference = oldFile.Reference

	// Add object
	if err = api.db.Set(key, file.ToBytes()); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&Object{
		Id:   id,
		Data: reqBody.Data,
		Tags: reqBody.Tags,
	})
}
