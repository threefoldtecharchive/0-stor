package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"log"
	"io/ioutil"
)

// Updatensid is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) Updatensid(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	value, err := ioutil.ReadAll(r.Body)

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	key := mux.Vars(r)["nsid"]

	old_value, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}


	// NOT FOUND
	if old_value == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
	}

	if err := api.db.Set(key, value); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
	}

	json.NewEncoder(w).Encode(&respBody)
}
