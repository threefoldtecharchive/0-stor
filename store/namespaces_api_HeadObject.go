package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the store
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", nsid, id)

	exists, err := api.db.Exists(key)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if exists {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
