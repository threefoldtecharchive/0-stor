package main

import (
	"encoding/json"
	"net/http"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {
	var namespace NamespaceCreate

	namespaceObj := r.Context().Value("namespace").([]byte)

	// No need to handle errors, we assume data is saved correctly
	json.Unmarshal(namespaceObj, &namespace)

	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
