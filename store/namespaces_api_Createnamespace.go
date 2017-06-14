package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"fmt"
	"time"
)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {

	var reqBody NamespaceCreate

	value, err := ioutil.ReadAll(r.Body)

	defer r.Body.Close()

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// decode request
	if err := json.Unmarshal(value, &reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	key := reqBody.Label

	v, err := api.db.Get(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 409 Conflict if name space already exists
	if v != nil {
		http.Error(w, "Namespace already exists", http.StatusConflict)
		return
	}

	// Add new name space
	if err := api.db.Set(key, value); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Add stats
	// We assume namespaces and object names can't contain (_)
	statsKey := fmt.Sprintf("%s_%s", key, "stats")

	stats := Stat{
		creationDate: time.Now(),
		NamespaceStat: NamespaceStat{
			NrObjects: 0,
			RequestPerHour: 0,
		},
	}

	if err := api.db.Set(statsKey, stats.toBytes()); err != nil{
		api.db.Delete(key)
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
