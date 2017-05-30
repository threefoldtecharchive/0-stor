package main

import (
	"encoding/json"
	"net/http"
)

// Updatensid is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) Updatensid(w http.ResponseWriter, r *http.Request) {
	var respBody Namespace
	json.NewEncoder(w).Encode(&respBody)
}
