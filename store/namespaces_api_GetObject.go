package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"encoding/json"
	"log"
)

// GetObject is the handler for GET /namespaces/{nsid}/objects/{id}
// Retrieve object from the store
func (api NamespacesAPI) GetObject(w http.ResponseWriter, r *http.Request) {

	var object Object

	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprint("%s:%s", namespace, id)

	value, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if value == nil{
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
	}

	json.Unmarshal(value[1:], &object)
	json.NewEncoder(w).Encode(&object)
}
