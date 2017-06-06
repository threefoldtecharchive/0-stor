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

	defer r.Body.Close()

	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// decode request
	if err := json.Unmarshal(value, &reqBody); err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

	oldValue, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if oldValue == nil{
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&Object{
		Id: id,
		Data: reqBody.Data,
		Tags: reqBody.Tags,
	})
}