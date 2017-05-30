package main

import (
	"encoding/json"
	"net/http"
)

// Listobjects is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) Listobjects(w http.ResponseWriter, r *http.Request) {
	var respBody []Object
	json.NewEncoder(w).Encode(&respBody)
}
