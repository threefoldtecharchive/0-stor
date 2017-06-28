package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody Object

	nsid := mux.Vars(r)["nsid"]

	storeStat := StoreStat{}
	if err := storeStat.Get(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reservation := r.Context().Value("reservation").(*Reservation)

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Make sure file contents are valid
	file, err := reqBody.ToFile(true)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s:%s", nsid, reqBody.Id)

	oldFile, err := api.db.GetFile(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var addObject bool = true

	// object already exists
	if oldFile != nil {
		if oldFile.Reference < 255 {
			file.Reference = oldFile.Reference + 1
			log.Debugln(file.Reference)
		} else {
			addObject = false
		}
	}

	// Add or update object
	if addObject {

		if reservation.SizeRemaining() < file.Size(){
			http.Error(w, "File SizeAvailable exceeds the remaining free space in namespace", http.StatusForbidden)
			return
		}

		if err = api.db.Set(key, file.ToBytes()); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		reservation.SizeUsed += file.Size()


		if err:= reservation.Save(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}
