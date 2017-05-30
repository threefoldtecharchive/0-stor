package main

import (
	"encoding/json"
	"net/http"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {
	var respBody Namespace
	json.NewEncoder(w).Encode(&respBody)
}
