package main

import (
	"encoding/json"
	"net/http"
)

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody NamespaceStat
	json.NewEncoder(w).Encode(&respBody)
}
