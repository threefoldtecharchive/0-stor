package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {
	var namespace NamespaceCreate

	key := mux.Vars(r)["nsid"]
	value := api.db.Get(key)

	// NOT FOUND
	if value == nil{
		w.WriteHeader(404)
		return
	}

	// No need to handle errors, we assume data is saved correctly
	json.Unmarshal(value, &namespace)

	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	json.NewEncoder(w).Encode(&respBody)
}
