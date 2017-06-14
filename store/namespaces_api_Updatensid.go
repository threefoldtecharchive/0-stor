package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Updatensid is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) Updatensid(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	value, err := ioutil.ReadAll(r.Body)

	defer r.Body.Close()

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// decode request
	if err := json.Unmarshal(value, &reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]

	if err := api.db.Set(nsid, value); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody := &Namespace{
		NamespaceCreate: reqBody,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
