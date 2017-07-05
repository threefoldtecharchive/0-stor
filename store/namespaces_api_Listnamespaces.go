package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/dgraph-io/badger"
	"strings"
	log "github.com/Sirupsen/logrus"
)

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
		perPageParam = strconv.Itoa(api.config.Pagination.PageSize)
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

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = api.config.Iterator.PreFetchSize

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := (page-1)*perPage + 1
	counter := 0 // Number of namespaces encountered
	resultsCount := perPage

	prefix := []byte(api.config.Namespace.prefix)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()

		// Found a namespace
		counter++

		// Skip this namespace if its index < intended startingIndex
		if counter < startingIndex {
			continue
		}

		value := item.Value()
		var namespace NamespaceCreate
		namespace.FromBytes(value)
		namespace.Label = strings.Replace(namespace.Label, api.config.Namespace.prefix, "", 1)
		respBody = append(respBody, Namespace{
			NamespaceCreate: namespace,
		})

		if len(respBody) == resultsCount {
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []Namespace{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
