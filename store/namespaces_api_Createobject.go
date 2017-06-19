package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/librairies/reservation"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody Object

	nsid := mux.Vars(r)["nsid"]
	statsBytes := r.Context().Value("stats").([]byte)
	stats := Stat{}
	stats.fromBytes(statsBytes)
	statsKey := r.Context().Value("statsKey").(string)

	reservationBytes := r.Context().Value("reservation").([]byte)
	res := reservation.Reservation{}
	res.FromBytes(reservationBytes)
	reservationKey := r.Context().Value("reservationKey").(string)

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

		if stats.SizeRemaining() < file.Size(){
			http.Error(w, "File Size exceeds the remaining free space in namespace", http.StatusForbidden)
			return
		}

		if err = api.db.Set(key, file.ToBytes()); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// may be sizeused, sizeresrved need to be floats
		stats.SizeUsed += int64(file.Size())
		res.SizeUsed += int64(file.Size())

		if err:= api.db.Set(statsKey, stats.toBytes()); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err:= api.db.Set(reservationKey, res.ToBytes()); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}
