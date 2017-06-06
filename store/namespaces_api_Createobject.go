package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"log"
	"io/ioutil"
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
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}


	value, err := ioutil.ReadAll(r.Body)

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

	key := fmt.Sprintf("%s:%s", namespace, reqBody.Id)

	v, err =  api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var newValue []byte
	var addObject bool = false

	// object already exists
	if v != nil{
		newValue = v
		counter := int(newValue[0])
		if counter < 255 {
			counter++
			newValue[0] = byte(counter)
			addObject = true
		}
	}else{
		// Prepend a byte with value (1) to the data
		newValue = make([]byte, len(value) + 1)
		newValue[0] = byte(1)
		copy(newValue[1:], value)
		addObject = true
	}

	// Add object
	if addObject{
		if err = api.db.Set(key, newValue); err != nil{
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(&reqBody)
}
