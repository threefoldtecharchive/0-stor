package main

import (
	"encoding/json"
	"net/http"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody ObjectCreate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	var respBody Object
	json.NewEncoder(w).Encode(&respBody)
}
