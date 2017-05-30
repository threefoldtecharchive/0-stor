package main

import (
	"encoding/json"
	"net/http"
)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	var respBody Namespace
	json.NewEncoder(w).Encode(&respBody)
}
