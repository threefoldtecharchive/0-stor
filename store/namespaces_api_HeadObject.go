package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"log"
)

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the store
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

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
}
