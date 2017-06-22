package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"strings"
	"encoding/json"
	"strconv"
	"github.com/dgraph-io/badger/badger"

	log "github.com/Sirupsen/logrus"
)

// ListReservations is the handler for GET /namespaces/{nsid}/reservation
// Return a list of all the existing reservation for the give resource
func (api NamespacesAPI) ListReservations(w http.ResponseWriter, r *http.Request) {
	var respBody []Reservation
	// Pagination
	pageParam := r.FormValue("page")
	per_pageParam := r.FormValue("per_page")

	if pageParam == "" {
		pageParam = "1"
	}

	if per_pageParam == "" {
		per_pageParam = strconv.Itoa(api.config.Pagination.PageSize)
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

	prefix := fmt.Sprintf("%s%s", api.config.Reservations.Namespaces.Prefix, nsid)

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = api.config.Iterator.PreFetchSize

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := (page-1)*per_page + 1
	counter := 0 // Number of objects encountered
	resultsCount := per_page

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		key := string(item.Key()[:])
		/* Skip non intended records
		*/
		if !strings.Contains(key, prefix) {
			continue
		}

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

		respBody = append(respBody, res)

		if len(respBody) == resultsCount {
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []Reservation{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
