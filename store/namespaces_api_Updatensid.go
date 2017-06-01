package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
)

// Updatensid is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) Updatensid(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// No need to handle error. JSON is assumed to be correct at this point
	value, _ := json.Marshal(reqBody)

	key := mux.Vars(r)["nsid"]

	old_value := api.db.Get(key)

	// NOT FOUND
	if old_value == nil{
		w.WriteHeader(404)
		return
	}

	api.db.Set(key, value)


	respBody:= &Namespace{
		NamespaceCreate: reqBody,
	}

	json.NewEncoder(w).Encode(&respBody)
}
