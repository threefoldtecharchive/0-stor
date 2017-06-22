package main

import (
	"encoding/json"
	"net/http"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {
	var namespace NamespaceCreate

	namespace = r.Context().Value("namespace").(NamespaceCreate)

	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
