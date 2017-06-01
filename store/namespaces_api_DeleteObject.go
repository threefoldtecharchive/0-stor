package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"log"
)

// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the store
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprint("%s:%s", namespace, id)

	//v, err :=  api.db.Get(key)
	//
	//if err != nil{
	//	log.Println(err.Error())
	//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	//}
	//
	//// NOT FOUND
	//if v == nil{
	//	http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
	//}

	err2 := api.db.Delete(key)

	if err2 != nil{
		log.Println(err2.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	w.WriteHeader(204)
}
