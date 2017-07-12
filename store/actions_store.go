package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"
	"fmt"
)

// statsPost is the handler for POST /namespaces/stats
// Update Global Store statistics and available space
func (api NamespacesAPI) UpdateStoreStats(w http.ResponseWriter, r *http.Request) {
	var reqBody StoreStatRequest

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

	storeStat := StoreStat{}
	if err := storeStat.Get(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if reqBody.SizeAvailable < storeStat.SizeUsed{
		err := fmt.Sprintf("Store stats available size must be greater than used size (%v)",  storeStat.SizeUsed)
		http.Error(w, err, http.StatusForbidden)
		return
	}

	storeStat.SizeAvailable = reqBody.SizeAvailable

	if err := storeStat.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}

// GetStoreStats is the handler for GET /namespaces/stats
// Return usage statistics about the whole KV
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
