package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"log"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody Object

	namespace := mux.Vars(r)["nsid"]

	v, err := api.db.Get(namespace)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
	}

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	key := fmt.Sprint("%s:%s", namespace, reqBody.Id)

	v, err =  api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// 409 Conflict if object already exists
	if v != nil{
		http.Error(w, "Object already exists", http.StatusConflict)
	}


	// No need to handle error. reqBody was decoded successfully earlier
	value, _ := json.Marshal(reqBody)

	// Prepend a byte with value (1) to the data
	newValue := make([]byte, len(value) + 1)
	newValue[0] = byte(1)
	copy(newValue[1:], value)

	// Add object
	if err = api.db.Set(key, newValue); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(&reqBody)
}
