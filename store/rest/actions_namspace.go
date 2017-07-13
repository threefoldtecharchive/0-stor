package rest

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/db/badger"
	"github.com/zero-os/0-stor/store/rest/models"

	"fmt"
	"strconv"

	"github.com/gorilla/mux"
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
	reqBody.Label = reqBody.Key()

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

	// Update namespace stats
	defer api.UpdateNamespaceStats(mux.Vars(r)["nsid"])

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
	// defer func() {
	// 	resutls, err := api.db.List(fmt.Sprintf("%s:", nsid))
	//
	// 	for _, key := range resutls {
	// 		if err := api.db.Delete(key); err != nil {
	// 			log.Errorln(err.Error())
	//
	// 		}
	//
	// 	}
	//
	// 	storeStat := models.StoreStat{}
	// 	b, err := api.db.Get(models.STORE_STATS_PREFIX)
	// 	if err != nil {
	// 		log.Errorln(err.Error())
	// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	if err := storeStat.Decode(b); err != nil {
	// 		log.Errorln(err.Error())
	// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	ns := NamespaceCreate{
	// 		Label: nsid,
	// 	}
	//
	// 	stats, err := ns.GetStats(api.db, api.config)
	//
	// 	if err != nil {
	// 		log.Errorln(err.Error())
	// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	namespaceStats := stats
	// 	storeStat.SizeAvailable += namespaceStats.TotalSizeReserved
	// 	storeStat.SizeUsed -= namespaceStats.TotalSizeReserved
	//
	// 	// delete namespacestats
	// 	if err := namespaceStats.Delete(api.db, api.config); err != nil {
	// 		log.Errorln(err.Error())
	// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	// Save Updated global stats
	// 	if err := storeStat.Save(api.db, api.config); err != nil {
	// 		log.Println("save")
	// 		log.Errorln(err.Error())
	// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	// Delete reservations
	// 	r := Reservation{
	// 		Namespace: nsid,
	// 	}
	//
	// 	resutls, err = api.db.ListAllRecordsStartingWith(r.GetKey(api.config))
	//
	// 	for _, key := range resutls {
	// 		if err := api.db.Delete(key); err != nil {
	// 			log.Errorln(err.Error())
	//
	// 		}
	//
	// 	}
	// }()

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
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// NOT FOUND
	if err == badger.ErrNotFound {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	if err := namespace.Decode(b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	respBody := Namespace{
		NamespaceCreate: *v,
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
		perPageParam = strconv.Itoa(api.config.DB.Pagination.PageSize)
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

	list, err := api.db.List(models.NAMESPACE_PREFIX)
	if err != nil {
		log.Errorln("Error listing namespace :%v", err)
		http.Error(w, "Error listing namespace", http.StatusInternalServerError)
		return
	}

	respBody := make([]Namespace, 0, len(list))

	for i := startingIndex; i < resultsCount || i >= len(list); i++ {
		key := fmt.Sprintf("%s%s", models.NAMESPACE_PREFIX, list[i])
		b, err := api.db.Get(key)
		if err != nil {
			log.Errorln("Error loading namespace :%v", err)
			http.Error(w, "Error loading namespace", http.StatusInternalServerError)
			return
		}

		ns := models.NamespaceCreate{label: list[i]}
		if err := ns.Decode(b); err != nil {
			log.Errorln("Error decoding namespace :%v", err)
			http.Error(w, "Error decoding namespace", http.StatusInternalServerError)
			return
		}

		respBody = append(respBody, models.Namespace{
			NamespaceCreate: ns,
		})
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []Namespace{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody *NamespaceStats

	// No need to prefix nsid here, the method ns.GetStats() does this
	nonPrefixedLabel := mux.Vars(r)["nsid"]
	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nonPrefixedLabel)

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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nonPrefixedLabel)

	ns := NamespaceCreate{
		Label: nonPrefixedLabel,
	}

	respBody, err = ns.GetStats(api.db, api.config)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = respBody.Get(api.db, api.config)

	if err != nil {
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

// Updatensid is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) Updatensid(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	defer r.Body.Close()

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

	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, mux.Vars(r)["nsid"])

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	reqBody.Label = nsid

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

	if err := reqBody.Save(api.db, api.config); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// @TODO:
	if nsid != reqBody.Label {
		defer func() {
			// update namespacestats with new prefix
			// update reservations with new prefix
			// delete old namespace
			// delete old namespace stats
			// delete old namsepace reservations
		}()
	}

	respBody := &Namespace{
		NamespaceCreate: reqBody,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	namespace := NamespaceCreate{
		Label: nsid,
	}

	v, err := namespace.Get(api.db, api.config)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil {
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
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

	// Update name space
	if err := namespace.UpdateACL(reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

    namespace.Encode()
    api.db.

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
