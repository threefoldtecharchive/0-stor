package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"

	log "github.com/Sirupsen/logrus"

	"fmt"
)

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody = NamespaceStat{}

	nsid := mux.Vars(r)["nsid"]
	statsKey := fmt.Sprintf("%s_stats", nsid)

	value, err := api.db.Get(statsKey)

	// Database Error
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if value == nil{
		log.Errorln("Name space stats not found")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	stat := Stat{}

	if err := stat.fromBytes(value); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody.NrObjects = stat.NrObjects
	respBody.RequestPerHour = stat.RequestPerHour
	json.NewEncoder(w).Encode(&respBody)
}
