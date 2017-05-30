package main

import (
	"encoding/json"
	"net/http"
)

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

}
