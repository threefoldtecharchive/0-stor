package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"
	"strings"
)

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {

	nsid := api.GetNamespaceID(r)

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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	namespace.Label = strings.Replace(namespace.Label, api.config.Namespace.prefix, "", 1)
	respBody := &Namespace{
		NamespaceCreate: namespace,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
