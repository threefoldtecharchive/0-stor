package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"log"
)

// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["nsid"]

	//v, err :=  api.db.Get(key)
	//
	//if err != nil{
	//	log.Println(err.Error())
	//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	//	return
	//}
	//
	//// NOT FOUND
	//if v == nil{
	//	http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
	//	return
	//}

	err2 := api.db.Delete(key)

	if err2 != nil{
		log.Println(err2.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Error(w, "", http.StatusNoContent)
}
