package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"

)

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody *NamespaceStats

	respBody = r.Context().Value("namespaceStats").(*NamespaceStats)

	// Database Error
	_, err := respBody.Get(api.db, api.config)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(&respBody)
}
