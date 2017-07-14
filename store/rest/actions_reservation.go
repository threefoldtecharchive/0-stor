package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/core/librairies/reservation"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/goraml"
	"github.com/zero-os/0-stor/store/rest/models"
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
		per_pageParam = "20"
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

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	res := new(models.Reservation)
	res.Namespace = nsid

	startingIndex := (page-1)*per_page + 1
	resultsCount := per_page

	resutls2, err := api.db.Filter(res.Key(), startingIndex, resultsCount)

	for _, value := range resutls2 {
		tmp := new(models.Reservation)
		if err := tmp.Decode(value); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		respBody = append(respBody, tmp.Reservation)
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
	var respBody models.NamespacesNsidReservationPostRespBody
	user := r.Context().Value("user").(string)

	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	namespaceStats := new(models.NamespaceStats)
	namespaceStats.Namespace = nsid

	stats, err := api.db.Get(namespaceStats.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = namespaceStats.Decode(stats); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err != nil {
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

	if err := reqBody.Validate(); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Validation Error", http.StatusForbidden)
		return
	}

	reservation := models.Reservation{
		nsid,
		reservation.Reservation{
			Id: rid,
		},
	}

	b, err := api.db.Get(reservation.Key())
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		} else {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if err := reservation.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get KV stat
	storeStats := new(models.StoreStat)
	b, err = api.db.Get(storeStats.Key())
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		} else {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}

	err = storeStats.Decode(b)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	oldReservedSize := reservation.SizeReserved
	newReservedSize := float64(reqBody.Size)

	diff := newReservedSize - oldReservedSize

	if diff > 0 {
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

	resToken, err := reservation.GenerateTokenForReservation()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	adminACL := models.ACLEntry{
		Admin:  true,
		Read:   true,
		Write:  true,
		Delete: true,
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser(user, nsid, adminACL)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save Updated global stats
	b, err = storeStats.Encode()

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := api.db.Set(storeStats.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	b, err = reservation.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update reservation
	if err := api.db.Set(reservation.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	b, err = namespaceStats.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save namespacestats
	if err := api.db.Set(namespaceStats.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = models.NamespacesNsidReservationPostRespBody{
		Reservation:      reservation.Reservation,
		DataAccessToken:  resToken,
		ReservationToken: dataToken,
	}

	json.NewEncoder(w).Encode(&respBody)
}

// nsidreservationidGet is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) nsidreservationidGet(w http.ResponseWriter, r *http.Request) {
	var respBody models.Reservation

	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	rid := mux.Vars(r)["id"]

	respBody = models.Reservation{
		nsid,
		reservation.Reservation{
			Id: rid,
		},
	}

	b, err := api.db.Get(respBody.Key())
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		} else {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if err := respBody.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// CreateReservation is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody reservation.ReservationRequest
	var respBody models.NamespacesNsidReservationPostRespBody

	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	namespaceStats := new(models.NamespaceStats)
	namespaceStats.Namespace = nsid

	stats, err := api.db.Get(namespaceStats.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = namespaceStats.Decode(stats); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err != nil {
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

	storeStat := models.StoreStat{}
	b, err := api.db.Get(storeStat.Key())
	if err != nil {
		log.Errorln(err.Error())
	}

	if err := storeStat.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Validate reservation is applicable
	if err := reqBody.Validate(); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Validation Error", http.StatusForbidden)
		return
	}

	// Validate available disk space

	if err := reqBody.ValidateFreeSpace(storeStat.SizeAvailable); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Not Enough Disk Space", http.StatusForbidden)
		return
	}

	reservation, err := models.NewReservation(nsid, "admin", float64(reqBody.Size), int(reqBody.Period))

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resToken, err := reservation.GenerateTokenForReservation()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	adminACL := models.ACLEntry{
		Admin:  true,
		Read:   true,
		Write:  true,
		Delete: true,
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser("admin", nsid, adminACL)

	if err != nil {
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
	b, err = storeStat.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := api.db.Set(storeStat.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save namespacestats
	b, err = namespaceStats.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := api.db.Set(namespaceStats.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save reservation
	b, err = reservation.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := api.db.Set(reservation.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update namespace with Admin ACL
	acl := models.ACL{
		Id:  "admin",
		Acl: adminACL,
	}

	ns.UpdateACL(acl)
	b, err = ns.Encode()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := api.db.Set(ns.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = models.NamespacesNsidReservationPostRespBody{
		Reservation:      reservation.Reservation,
		DataAccessToken:  dataToken,
		ReservationToken: resToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
