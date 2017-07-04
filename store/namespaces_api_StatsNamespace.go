package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
)

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody *NamespaceStats

	nsid := mux.Vars(r)["nsid"]
	ns := NamespaceCreate{
		Label: nsid,
	}

	respBody, err := ns.GetStats(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = respBody.Get(api.db, api.config)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(&respBody)
}
