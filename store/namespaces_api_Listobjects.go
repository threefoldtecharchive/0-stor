package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
	"strings"
)

// Listobjects is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) Listobjects(w http.ResponseWriter, r *http.Request) {
	var respBody []Object

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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNsid := fmt.Sprintf("%s%s", api.config.Namespace.prefix, nsid)
	exists, err := api.db.Exists(prefixedNsid)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	prefixStr := fmt.Sprintf("%s:", nsid)
	prefix := []byte(prefixStr)

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = api.config.Iterator.PreFetchSize

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := (page-1)*per_page + 1
	counter := 0 // Number of objects encountered
	resultsCount := per_page

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		key := string(item.Key()[:])

		// Found a namespace
		counter++

		// Skip this object if its index < intended startingIndex
		if counter < startingIndex {
			continue
		}

		value := item.Value()

		var file = &File{}
		object := file.ToObject(value, key)

		// remove prefix from file name
		object.Id = strings.Replace(object.Id, prefixStr, "", 1)
		respBody = append(respBody, *object)

		if len(respBody) == resultsCount {
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []Object{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
