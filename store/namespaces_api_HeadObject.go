package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
)

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the store
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

	if api.db.Exists(key){
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
	}

	// head has no body
	http.Error(w, "", http.StatusNotFound)
}
