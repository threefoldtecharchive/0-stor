package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"encoding/json"
	"strconv"
	"github.com/dgraph-io/badger"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/core/librairies/reservation"
	"time"
	"github.com/zero-os/0-stor/store/core/goraml"
)

// ListReservations is the handler for GET /namespaces/{nsid}/reservation
// Return a list of all the existing reservation for the give resource
func (api NamespacesAPI) ListReservations(w http.ResponseWriter, r *http.Request) {
	var respBody []reservation.Reservation
	// Pagination
	pageParam := r.FormValue("page")
	per_pageParam := r.FormValue("per_page")

	if pageParam == "" {
		pageParam = "1"
	}

	if per_pageParam == "" {
		per_pageParam = strconv.Itoa(api.config.DB.Pagination.PageSize)
	}

	page, err := strconv.Atoi(pageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	per_page, err := strconv.Atoi(per_pageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]
	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	exists, err := api.db.Exists(prefixedNamespaceID)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	prefix := []byte(fmt.Sprintf("%s%s", api.config.Namespace.Reservations.Prefix, nsid))

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = api.config.DB.Iterator.PreFetchSize

	it := api.db.KV.NewIterator(opt)
	defer it.Close()

	startingIndex := (page-1)*per_page + 1
	counter := 0 // Number of objects encountered
	resultsCount := per_page

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		// Found a namespace
		counter++

		// Skip this object if its index < intended startingIndex
		if counter < startingIndex {
			continue
		}

		value := item.Value()

		var res = Reservation{}

		if err := res.FromBytes(value); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		respBody = append(respBody, res.Reservation)

		if len(respBody) == resultsCount {
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []reservation.Reservation{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}


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


	// Get KV stat
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


// nsidreservationidGet is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) nsidreservationidGet(w http.ResponseWriter, r *http.Request) {
	var respBody Reservation

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)
	exists, err := api.db.Exists(prefixedNamespaceID)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s%s_%s", api.config.Namespace.Reservations.Prefix, nsid, id)

	value, err := api.db.Get(key)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if value == nil{
		http.Error(w, "Namespace or Reservation not found", http.StatusNotFound)
		return
	}

	respBody.FromBytes(value)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// CreateReservation is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody NamespacesNsidReservationPostRespBody

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	user := "admin" // we make sure to add admin user to namsepace ACL, return its data token

	namespace := NamespaceCreate{
		Label: nsid,
	}

	v, err :=  namespace.Get(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	namespaceStats, err := namespace.GetStats(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
