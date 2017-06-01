package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"log"
	"io/ioutil"
)

// UpdateObject is the handler for PUT /namespaces/{nsid}/objects/{id}
// Update oject
func (api NamespacesAPI) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var reqBody ObjectUpdate

	value, err := ioutil.ReadAll(r.Body)

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprint("%s:%s", namespace, id)

	oldValue, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// KEY NOT FOUND
	if oldValue == nil{
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
	}

	// Prepend the same value of the first byte of old data
	newValue := make([]byte, len(value) + 1)
	newValue[0] = oldValue[0]
	copy(newValue[1:], value)

	// Add object
	if err = api.db.Set(key, newValue); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(&reqBody)
}