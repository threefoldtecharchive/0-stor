package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"fmt"
	"time"
	"github.com/zero-os/0-stor/store/goraml"
)

// CreateReservation is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody NamespacesNsidReservationPostRespBody

	nsid := mux.Vars(r)["nsid"]

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

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Check reservation ID does not exist
	resKey := fmt.Sprintf("%s$%s", nsid, reqBody.Id)

	exists, err = api.db.Exists(resKey)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Conflict, reservation ID exists
	if exists {
		http.Error(w, "Reservation ID already exists", http.StatusConflict)
		return
	}

	// Get available space on system
	storeStatsKey := api.config.Stats.CollectionName

	storeStatsBytes, err := api.db.Get(storeStatsKey)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	storeStat := StoreStat{}
	storeStat.fromBytes(storeStatsBytes)


	if err := reqBody.Validate(storeStat.Size); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "No enough Disk space", http.StatusForbidden)
		return
	}

	// Get namespace stats
	namespaceStats := NewStat()
	namespaceStatsKey := fmt.Sprintf("%s_%s", nsid, "stats")
	namespaceStatsBytes, err := api.db.Get(namespaceStatsKey)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create Reservation object
	creationDate := time.Now()

	expirationDate := creationDate.AddDate(0, 0, int(reqBody.Period))

	namespaceStats.fromBytes(namespaceStatsBytes)

	reservationKey := fmt.Sprintf("%s$%s", nsid, reqBody.Id)

	res := reservation.Reservation{
		AdminId: "", //@TODO: AdminId: r.Context().Value("user").(string)
		SizeReserved: reqBody.Size,
		SizeUsed: 0,
		ExpireAt: goraml.DateTime(expirationDate),
		Created: goraml.DateTime(creationDate),
		Updated: goraml.DateTime(creationDate),
		Id: reqBody.Id,
	}

	// update namespace stats
	namespaceStats.updateReservation(res)
	// global stat decrease amount
	storeStat.Size -= res.SizeReserved

	// Save Updated global stats
	if err := api.db.Set(storeStatsKey, storeStat.toBytes()); err != nil{{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}}

	// Save Updated namespace stats
	if err := api.db.Set(namespaceStatsKey, namespaceStats.toBytes()); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save reservation
	if err := api.db.Set(reservationKey, res.ToBytes()); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = NamespacesNsidReservationPostRespBody{
		Reservation: res,
		DataAccessToken: "",
		ReservationToken: "",
	}

	json.NewEncoder(w).Encode(&respBody)
}
