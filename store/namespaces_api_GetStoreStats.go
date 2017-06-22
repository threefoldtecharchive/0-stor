package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"
)

// GetStoreStats is the handler for GET /namespaces/stats
// Return usage statistics about the whole store
func (api NamespacesAPI) GetStoreStats(w http.ResponseWriter, r *http.Request) {
	var respBody StoreStat

	if err := respBody.Get(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&respBody)
}
