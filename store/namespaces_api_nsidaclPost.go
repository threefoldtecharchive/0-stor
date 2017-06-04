package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"log"
)

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL
	var namespace NamespaceCreate

	key := mux.Vars(r)["nsid"]
	value, err := api.db.Get(key)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if value == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	// decode request
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// If data was not saved correctly for any reason fail
	if err := json.Unmarshal(value, &namespace); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	aclIndex := -1 // -1 means ACL for that user does not exist

	// Find if ACL for that user already exists
	for i, item := range namespace.Acl{
		if item.Id == reqBody.Id{
			aclIndex = i
			break
		}
	}

	// Update User ACL
	if aclIndex != -1 {
		namespace.Acl[aclIndex] = reqBody
	}else { // Insert new ACL
		namespace.Acl = append(namespace.Acl, reqBody)
	}

	newACL, err := json.Marshal(namespace)

	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update name space
	if err := api.db.Set(key, newACL); err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//@TODO: return proper Access token
	json.NewEncoder(w).Encode("Access-Token")
}
