package rest

import (
	"encoding/json"
	"net/http"

	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/rest/models"

	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {

	var reqBody models.NamespaceCreate

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	originalLabel := reqBody.Label

	exists, err := api.db.Exists(reqBody.Key())
	if err != nil {
		// Database Error
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 409 Conflict if name space already exists
	if exists {
		http.Error(w, "Namespace already exists", http.StatusConflict)
		return
	}

	// Add new name space
	blob, err := reqBody.Encode()
	if err != nil {
		log.Errorf("Error encoding data: %v", err)
		http.Error(w, "Error encoding data", http.StatusInternalServerError)
		return
	}

	if err := api.db.Set(reqBody.Key(), blob); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Add stats
	// stats are saved prefixed with its own prefix + (non prefixed namespace)
	go func() {
		stats := models.NewNamespaceStats(originalLabel)
		b, err := stats.Encode()
		if err != nil {
			// TODO: better error handling
			log.Errorln(err.Error())
		}
		if err := api.db.Set(stats.Key(), b); err != nil {
			log.Errorln(err.Error())
		}
	}()

	respBody := &models.Namespace{
		NamespaceCreate: reqBody,
		SpaceAvailable:  0,
		SpaceUsed:       0,
	}

	respBody.Label = originalLabel

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&respBody)
}

// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	nsid := fmt.Sprintf("%s%s", models.NAMESPACE_PREFIX, mux.Vars(r)["nsid"])

	exists, err := api.db.Exists(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	err = api.db.Delete(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete objects in a namespace
	defer func() {
		resutls, err := api.db.List(fmt.Sprintf("%s:", nsid))

		for _, key := range resutls {
			if err := api.db.Delete(key); err != nil {
				log.Errorln(err.Error())
			}

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
		}
		storeStat.SizeAvailable += namespaceStats.TotalSizeReserved
		storeStat.SizeUsed -= namespaceStats.TotalSizeReserved

		// delete namespacestats
		if err = api.db.Delete(namespaceStats.Key()); err != nil {
			log.Errorln(err.Error())
		}

		b, err = storeStat.Encode()

		if err != nil {
			log.Errorln(err.Error())
		}
		// Save Updated global stats
		if err = api.db.Set(storeStat.Key(), b); err != nil {
			log.Errorln(err.Error())
		}

		// Delete reservations
		r := new(models.Reservation)
		r.Namespace = nsid

		resutls2, err := api.db.List(r.Key())

		for _, key := range resutls2 {
			if err := api.db.Delete(key); err != nil {
				log.Errorln(err.Error())

			}
		}
	}()

	// 204 has no body
	http.Error(w, "", http.StatusNoContent)
}

// Getnsid is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) Getnsid(w http.ResponseWriter, r *http.Request) {

	nsid := mux.Vars(r)["nsid"]

	namespace := models.NamespaceCreate{
		Label: nsid,
	}

	b, err := api.db.Get(namespace.Key())
	if err != nil {
		log.Errorf("error getting namespace: %v\n", err)

		if err == db.ErrNotFound {
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := namespace.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody := models.Namespace{
		NamespaceCreate:  models.NamespaceCreate{
			Label: nsid,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// Listnamespaces is the handler for GET /namespaces
// List all namespaces
func (api NamespacesAPI) Listnamespaces(w http.ResponseWriter, r *http.Request) {
	// Pagination
	pageParam := r.FormValue("page")
	perPageParam := r.FormValue("perPage")

	if pageParam == "" {
		pageParam = "1"
	}

	if perPageParam == "" {
		perPageParam = "20"
	}

	page, err := strconv.Atoi(pageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	perPage, err := strconv.Atoi(perPageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	startingIndex := (page-1)*perPage + 1
	resultsCount := perPage

	list, err := api.db.Filter(models.NAMESPACE_PREFIX, startingIndex, resultsCount)
	if err != nil {
		log.Errorln("Error listing namespace :%v", err)
		http.Error(w, "Error listing namespace", http.StatusInternalServerError)
		return
	}

	respBody := make([]models.Namespace, 0, len(list))

	for _, record := range list {
		ns := new(models.NamespaceCreate)
		if err := ns.Decode(record); err != nil {
			log.Errorln("Error decoding namespace :%v", err)
			http.Error(w, "Error decoding namespace", http.StatusInternalServerError)
			return
		}

		respBody = append(respBody, models.Namespace{
			NamespaceCreate: *ns,
		})
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []models.Namespace{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody *models.NamespaceStats

	// No need to prefix nsid here, the method ns.GetStats() does this
	nsid := mux.Vars(r)["nsid"]
	respBody.Namespace = nsid

	b, err := api.db.Get(respBody.Key())

	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if err = respBody.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// namespace shuld be returned to user without prefix
	respBody.Namespace = mux.Vars(r)["nsid"]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody models.ACL

	nsid := mux.Vars(r)["nsid"]

	namespace := models.NamespaceCreate{
		Label: nsid,
	}

	b, err := api.db.Get(namespace.Key())

	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
			return
		}
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := namespace.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reservation := r.Context().Value("reservation").(*models.Reservation)

	// decode request
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Update name space
	namespace.UpdateACL(reqBody)
	b, err = namespace.Encode()

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = api.db.Set(namespace.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser(reqBody.Id, namespace.Label, reqBody.Acl)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dataToken)
}
