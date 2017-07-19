package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-units"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/goraml"
	"github.com/zero-os/0-stor/store/jwt"
	"github.com/zero-os/0-stor/store/rest/models"
)

// ListReservations is the handler for GET /namespaces/{nsid}/reservation
// Return a list of all the existing reservation for the give resource
func (api NamespacesAPI) ListReservations(w http.ResponseWriter, r *http.Request) {
	var respBody []models.Reservation
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

	namespaceID := mux.Vars(r)["nsid"]
	_, err = api.namespaceStat(mux.Vars(r)["nsid"])
	if err != nil {
		log.Errorln(err.Error())
		if err == db.ErrNotFound {
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	res := new(models.Reservation)
	res.Namespace = namespaceID

	startingIndex := (page-1)*per_page + 1
	resultsCount := per_page

	resutls2, err := api.db.Filter(res.Key(), startingIndex, resultsCount)

	for _, value := range resutls2 {
		tmp := models.Reservation{}
		if err := tmp.Decode(value); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		respBody = append(respBody, tmp)
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []models.Reservation{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// UpdateReservation is the handler for PUT /namespaces/{nsid}/reservation/{id}
// Renew an existing reservation
func (api NamespacesAPI) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	var reqBody models.ReservationRequest
	var respBody models.NamespacesNsidReservationPostRespBody
	user := r.Context().Value("user").(string)

	namespaceID := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: namespaceID,
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

	namespaceStats, err := api.namespaceStat(namespaceID)
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

	reservation, err := api.reservation(rid, namespaceID)
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

	// Get KV stat
	storeStats, err := api.storeStatsMgr.GetStats()
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

	oldReservedSize := reservation.SizeReserved
	newReservedSize := reqBody.Size
	diff := uint64(newReservedSize - int64(oldReservedSize))

	if diff > storeStats.SizeAvailable {
		log.Errorln("Data size exceeds limits")
		http.Error(w, "No enough Disk space", http.StatusForbidden)
		return
	}

	storeStats.SizeAvailable -= diff
	storeStats.SizeUsed += diff
	namespaceStats.TotalSizeReserved += diff

	creationDate := time.Time(reservation.Created)
	expirationDate := creationDate.AddDate(0, 0, int(reqBody.Period))

	reservation.SizeReserved += diff
	reservation.ExpireAt = goraml.DateTime(expirationDate)

	resToken, err := jwt.GenerateReservationToken(*reservation, api.JWTKey())
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

	dataToken, err := jwt.GenerateDataAccessToken(user, *reservation, adminACL, api.jwtKey)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save Updated global stats
	err = api.storeStatsMgr.SetStats(storeStats.SizeAvailable, storeStats.SizeUsed)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = api.setReservation(reservation)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = api.setNamespaceStat(namespaceStats)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = models.NamespacesNsidReservationPostRespBody{
		Reservation:      *reservation,
		DataAccessToken:  resToken,
		ReservationToken: dataToken,
	}

	json.NewEncoder(w).Encode(&respBody)
}

// GetReservation is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) GetReservation(w http.ResponseWriter, r *http.Request) {
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
		Namespace: nsid,
		Id:        rid,
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
	var reqBody models.ReservationRequest
	var respBody models.NamespacesNsidReservationPostRespBody

	namespaceID := mux.Vars(r)["nsid"]
	ns := models.NamespaceCreate{
		Label: namespaceID,
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

	_, err = api.namespaceStat(mux.Vars(r)["nsid"])
	if err != nil {
		log.Errorln(err.Error())
		if err == db.ErrNotFound {
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	namespaceStats, err := api.namespaceStat(namespaceID)
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

	// Validate reservation is applicable
	if err := reqBody.Validate(); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Validation Error", http.StatusForbidden)
		return
	}

	// Validate available disk space
	storeStat, err := api.storeStatsMgr.GetStats()
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// reqBody.Size is in Mib, storeStat.SizeAvailable in bytes
	if storeStat.SizeAvailable <= uint64(reqBody.Size*units.MiB) {
		http.Error(w, "Not Enough Disk Space", http.StatusForbidden)
		return
	}

	// TODO replace admin
	reservation, err := models.NewReservation(namespaceID, "admin", uint64(reqBody.Size*units.MiB), int(reqBody.Period))
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resToken, err := jwt.GenerateReservationToken(*reservation, api.JWTKey())
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

	dataToken, err := jwt.GenerateDataAccessToken(reservation.AdminId, *reservation, adminACL, api.jwtKey)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// global stat decrease amount
	if err := api.storeStatsMgr.SetStats(
		storeStat.SizeAvailable-reservation.SizeReserved,
		storeStat.SizeUsed+reservation.SizeReserved); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ad reserved size to namespace stats
	namespaceStats.TotalSizeReserved += reservation.SizeReserved
	// Save namespacestats
	err = api.setNamespaceStat(namespaceStats)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Save reservation
	err = api.setReservation(reservation)
	if err != nil {
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
	b, err := ns.Encode()
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
		Reservation:      *reservation,
		DataAccessToken:  dataToken,
		ReservationToken: resToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
