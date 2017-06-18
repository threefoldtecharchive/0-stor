package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"time"
	"github.com/zero-os/0-stor/store/goraml"
)

// UpdateReservation is the handler for PUT /namespaces/{nsid}/reservation/{id}
// Renew an existing reservation
func (api NamespacesAPI) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody NamespacesNsidReservationPostRespBody

	nsid := mux.Vars(r)["nsid"]
	rid := mux.Vars(r)["id"]

	key :=  fmt.Sprintf("%s$%s", nsid, rid)

	v, err := api.db.Get(key)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	res := reservation.Reservation{}
	if err := res.FromBytes(v); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
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

	namespaceStats.fromBytes(namespaceStatsBytes)

	if namespaceStats.Id != rid{
		http.Error(w, "Trying to update old reservation", http.StatusForbidden)
		return
	}

	// Get store stat
	storeStatsKey := api.config.Stats.CollectionName

	storeStatsBytes, err := api.db.Get(storeStatsKey)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	storeStat := StoreStat{}
	storeStat.fromBytes(storeStatsBytes)

	oldReservedSize := namespaceStats.SizeReserved
	newReservedSize := reqBody.Size

	diff := newReservedSize - oldReservedSize

	if diff > 0{
		diff = newReservedSize - oldReservedSize
		if diff > storeStat.Size{
			log.Errorln("Data size exceeds limits")
			http.Error(w, "No enough Disk space", http.StatusForbidden)
			return
		}
	}

	storeStat.Size -= diff

	creationDate := time.Time(namespaceStats.Created)
	expirationDate := creationDate.AddDate(0, 0, int(reqBody.Period))

	namespaceStats.ExpireAt = goraml.DateTime(expirationDate)
	namespaceStats.SizeReserved += diff

	res.SizeReserved += diff
	res.ExpireAt = namespaceStats.ExpireAt

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
	if err := api.db.Set(key, res.ToBytes()); err != nil{
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
