package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"
	"github.com/gorilla/mux"

	log "github.com/Sirupsen/logrus"
	"time"
	"github.com/zero-os/0-stor/store/goraml"
)

// UpdateReservation is the handler for PUT /namespaces/{nsid}/reservation/{id}
// Renew an existing reservation
func (api NamespacesAPI) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody NamespacesNsidReservationPostRespBody
	user := r.Context().Value("user").(string)

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

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

	ns := NamespaceCreate{
		Label: nsid,
	}

	namespaceStats , err := ns.GetStats(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	rid := mux.Vars(r)["id"]

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Validation Error", http.StatusForbidden)
		return
	}

	reservation := Reservation{
		nsid,
		reservation.Reservation{
			Id: rid,
		},
	}

	_, err = reservation.Get(api.db, api.config)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	// Get store stat
	var storeStats StoreStat
	if err := storeStats.Get(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	oldReservedSize := reservation.SizeReserved
	newReservedSize := float64(reqBody.Size)

	diff := newReservedSize - oldReservedSize

	if diff > 0{
		if diff > storeStats.SizeAvailable {
			log.Errorln("Data size exceeds limits")
			http.Error(w, "No enough Disk space", http.StatusForbidden)
			return
		}
	}

	storeStats.SizeAvailable -= diff
	namespaceStats.TotalSizeReserved += diff

	creationDate := time.Time(reservation.Created)
	expirationDate := creationDate.AddDate(0, 0, int(reqBody.Period))

	reservation.SizeReserved += diff
	reservation.ExpireAt = goraml.DateTime(expirationDate)


	resToken, err := reservation.GenerateTokenForReservation(api.db, nsid)
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	adminACL := ACLEntry{
		Admin: true,
		Read: true,
		Write: true,
		Delete: true,
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser(user, nsid, adminACL)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save Updated global stats
	if err := storeStats.Save(api.db, api.config); err != nil{{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}}

	// Update reservation
	if err := reservation.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save namespacestats
	if err := namespaceStats.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = NamespacesNsidReservationPostRespBody{
		Reservation: reservation.Reservation,
		DataAccessToken: resToken,
		ReservationToken: dataToken,
	}

	json.NewEncoder(w).Encode(&respBody)
}
