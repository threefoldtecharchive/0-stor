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

	key := fmt.Sprintf("%s:%s", namespace, id)

	v, err :=  api.db.Get(key)

	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace or object doesn't exist", http.StatusNotFound)
		return
	}

	err2 := api.db.Delete(key)

	if err2 != nil{
		log.Println(err2.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Error(w, "", http.StatusNoContent)
}
