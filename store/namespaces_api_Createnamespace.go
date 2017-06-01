package main

import (
	"encoding/json"
	"net/http"
	"log"
)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {

	var reqBody NamespaceCreate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	// No need to handle error. reqBody was decoded successfully earlier
	value, _ := json.Marshal(reqBody)

	key := reqBody.Label


	v, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// 409 Conflict if name space already exists
	if v != nil{
		http.Error(w, "Namespace already exists", http.StatusConflict)
	}

	// Add new name space
	if err := api.db.Set(key, value); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
	}

	json.NewEncoder(w).Encode(&respBody)
}
