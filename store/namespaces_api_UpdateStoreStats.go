package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"
)

// statsPost is the handler for POST /namespaces/stats
// Update Global Store statistics and available space
func (api NamespacesAPI) UpdateStoreStats(w http.ResponseWriter, r *http.Request) {
	var reqBody StoreStat

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err:= reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	key := api.config.Stats.CollectionName

	if err := api.db.Set(key, reqBody.toBytes()); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}
