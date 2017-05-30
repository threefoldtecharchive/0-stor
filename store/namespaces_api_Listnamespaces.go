package main

import (
	"encoding/json"
	"net/http"
)

// Listnamespaces is the handler for GET /namespaces
// List all namespaces
func (api NamespacesAPI) Listnamespaces(w http.ResponseWriter, r *http.Request) {
	var respBody []Namespace
	json.NewEncoder(w).Encode(&respBody)
}
