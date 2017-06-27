package main

import (
	"encoding/json"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// CreateReservation is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody NamespacesNsidReservationPostRespBody

	nsid := mux.Vars(r)["nsid"]
	user := "admin" // we make sure to add admin user to namsepace ACL, return its data token
	namespace := r.Context().Value("namespace").(NamespaceCreate)
	namespaceStats := r.Context().Value("namespaceStats").(*NamespaceStats)

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	storeStat := StoreStat{}
	if err := storeStat.Get(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Validate reservation is applicable
	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Validation Error", http.StatusForbidden)
		return
	}

	// Validate available disk space

	if err:= reqBody.ValidateFreeSpace(storeStat.SizeAvailable); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Not Enough Disk Space", http.StatusForbidden)
		return
	}

	reservation, err := NewReservation(nsid, user, float64(reqBody.Size), int(reqBody.Period))

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


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


	// global stat decrease amount
	storeStat.SizeAvailable -= reservation.SizeReserved
	storeStat.SizeUsed += reservation.SizeReserved

	// Ad reserved size to namespace stats
	namespaceStats.TotalSizeReserved += reservation.SizeReserved

	// Save Updated global stats
	if err := storeStat.Save(api.db, api.config); err != nil{
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

	// Save reservation
	if err := reservation.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update namespace with Admin ACL
	acl := ACL{
		Id: user,
		Acl: adminACL,
	}

	if err := namespace.UpdateACL(api.db, api.config, acl); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = NamespacesNsidReservationPostRespBody{
		Reservation: reservation.Reservation,
		DataAccessToken: dataToken,
		ReservationToken: resToken,
	}

	json.NewEncoder(w).Encode(&respBody)
}
