package main

import (
	"encoding/json"
	"net/http"
	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
	"fmt"
)

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody *NamespaceStats

	// No need to prefix nsid here, the method ns.GetStats() does this
	nonPrefixedLabel := mux.Vars(r)["nsid"]
	nsid := fmt.Sprintf("%s%s", api.config.Namespace.prefix, nonPrefixedLabel)

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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nonPrefixedLabel)

	ns := NamespaceCreate{
		Label: nonPrefixedLabel,
	}

	respBody, err = ns.GetStats(api.db, api.config)

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

	// namespace shuld be returned to user without prefix
	respBody.Namespace = mux.Vars(r)["nsid"]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
