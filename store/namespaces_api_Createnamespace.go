package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"

)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {

	var reqBody NamespaceCreate

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := reqBody.Label

	exists, err := reqBody.Exists(api.db, api.config)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 409 Conflict if name space already exists
	if exists {
		http.Error(w, "Namespace already exists", http.StatusConflict)
		return
	}


	// Add new name space
	if err := reqBody.Save(api.db, api.config); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Add stats
	defer func(){
		stats := NewNamespaceStats(nsid)
		if err := stats.Save(api.db, api.config); err != nil{
			log.Errorln(err.Error())
		}
	}()

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
		SpaceAvailable: 0,
		SpaceUsed: 0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&respBody)
}
