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

	// No need to handle error. JSON is assumed to be correct at this point
	value, _ := json.Marshal(reqBody)

	key := reqBody.Label

	// 409 Conflict if name space already exists
	if v := api.db.Get(key); v != nil{
		w.WriteHeader(409)
		return
	}

	// Add new name space
	api.db.Set(key, value)

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
	}

	json.NewEncoder(w).Encode(&respBody)
}
