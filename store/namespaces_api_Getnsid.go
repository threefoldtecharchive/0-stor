package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {


	nsid := mux.Vars(r)["nsid"]
	namespace := NamespaceCreate{
		Label: nsid,
	}

	v, err :=  namespace.Get(api.db, api.config)


	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
