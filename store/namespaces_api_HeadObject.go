package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
)

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the store
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	exists, err := api.db.Exists(nsid)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", nsid, id)

	exists, err = api.db.Exists(key)

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
