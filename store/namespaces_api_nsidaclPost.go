package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL
	var namespace NamespaceCreate

	nsid := mux.Vars(r)["nsid"]
	namespaceObj :=r.Context().Value("namespace").([]byte)

	// decode request
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// If data was not saved correctly for any reason fail
	if err := json.Unmarshal(namespaceObj, &namespace); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	aclIndex := -1 // -1 means ACL for that user does not exist

	// Find if ACL for that user already exists
	for i, item := range namespace.Acl {
		if item.Id == reqBody.Id {
			aclIndex = i
			break
		}
	}

	// Update User ACL
	if aclIndex != -1 {
		namespace.Acl[aclIndex] = reqBody
	} else { // Insert new ACL
		namespace.Acl = append(namespace.Acl, reqBody)
	}

	newACL, err := json.Marshal(namespace)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update name space
	if err := api.db.Set(nsid, newACL); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//@TODO: return proper Access token
	json.NewEncoder(w).Encode("Access-Token")
}
