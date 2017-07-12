package main


import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
	"strings"
	"strconv"
	"fmt"
)

// Createnamespace is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Createnamespace(w http.ResponseWriter, r *http.Request) {

	var reqBody NamespaceCreate

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

	originalLabel := reqBody.Label
	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, reqBody.Label)
	reqBody.Label = nsid

	exists, err := reqBody.Exists(api.db, api.config)

	// Database Error
	if err != nil {
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
	if err := reqBody.Save(api.db, api.config); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Add stats
	// stats are saved prefixed with its own prefix + (non prefixed namespace)
	defer func(){
		stats := NewNamespaceStats(originalLabel)
		if err := stats.Save(api.db, api.config); err != nil{
			log.Errorln(err.Error())
		}
	}()

	respBody:= &Namespace{
		NamespaceCreate: reqBody,
		SpaceAvailable: 0,
		SpaceUsed: 0,
	}

	respBody.Label = originalLabel

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&respBody)
}


// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, mux.Vars(r)["nsid"])

	// Update namespace stats
	defer api.UpdateNamespaceStats(mux.Vars(r)["nsid"])

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

	err = api.db.Delete(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete objects in a namespace
	defer func() {
		resutls, err := api.db.ListAllRecordsStartingWith(fmt.Sprintf("%s:", nsid))

		for _, key := range(resutls){
			if err := api.db.Delete(key); err != nil {
				log.Errorln(err.Error())

			}

		}

		storeStat := StoreStat{}
		if err := storeStat.Get(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ns := NamespaceCreate{
			Label: nsid,
		}

		stats, err := ns.GetStats(api.db, api.config)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		namespaceStats := stats
		storeStat.SizeAvailable += namespaceStats.TotalSizeReserved
		storeStat.SizeUsed -= namespaceStats.TotalSizeReserved

		// delete namespacestats
		if err := namespaceStats.Delete(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Save Updated global stats
		if err := storeStat.Save(api.db, api.config); err != nil{
			log.Println("save")
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Delete reservations
		r := Reservation{
			Namespace: nsid,
		}

		resutls, err = api.db.ListAllRecordsStartingWith(r.GetKey(api.config))

		for _, key := range(resutls){
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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	v.Label = strings.Replace(namespace.Label, api.config.Namespace.Prefix, "", 1)
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
	var respBody []Namespace


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

	results, err := api.db.GetAllStartingWith(api.config.Namespace.Prefix, startingIndex, resultsCount)

	for _, v := range(results){
		var namespace NamespaceCreate
		namespace.FromBytes(v)
		namespace.Label = strings.Replace(namespace.Label, api.config.Namespace.Prefix, "", 1)
		respBody = append(respBody, Namespace{
			NamespaceCreate: namespace,
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

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	// Update namespace stats
	defer api.UpdateNamespaceStats(nonPrefixedLabel)

	ns := NamespaceCreate{
		Label: nonPrefixedLabel,
	}

	respBody, err = ns.GetStats(api.db, api.config)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = respBody.Get(api.db, api.config)

	if err != nil{
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

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, mux.Vars(r)["nsid"])

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	reqBody.Label = nsid

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

	if err := reqBody.Save(api.db, api.config); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// @TODO:
	if nsid != reqBody.Label{
		defer func(){
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

	reservation :=r.Context().Value("reservation").(*Reservation)

	// decode request
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Update name space
	if err := namespace.UpdateACL(api.db, api.config, reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	dataToken, err := reservation.GenerateDataAccessTokenForUser(reqBody.Id, namespace.Label, reqBody.Acl)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(dataToken)
}
