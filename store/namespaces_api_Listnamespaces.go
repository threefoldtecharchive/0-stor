package main

import (
	"encoding/json"
	"net/http"
	"github.com/zaibon/badger/badger"
	"strings"
	"log"
	"strconv"
)

// Listnamespaces is the handler for GET /namespaces
// List all namespaces
func (api NamespacesAPI) Listnamespaces(w http.ResponseWriter, r *http.Request) {
	var respBody []Namespace
	var namespace NamespaceCreate

	// Pagination
	pageParam := r.FormValue("page")
	per_pageParam := r.FormValue("per_page")

	if pageParam == ""{
		pageParam = "1"
	}

	if per_pageParam == ""{
		per_pageParam = strconv.Itoa(api.config.Pagination.PageSize)
	}

	page, err := strconv.Atoi(pageParam)

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	per_page, err := strconv.Atoi(per_pageParam)

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	opt := badger.IteratorOptions{}
	opt.FetchValues = api.config.Iterator.FetchValues
	opt.PrefetchSize = api.config.Iterator.FetchSize
	opt.Reverse = false

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := (page - 1) * per_page + 1
	counter := 0 // Number of namespaces encountered
	resultsCount := per_page

	for it.Rewind(); it.Valid(); it.Next(){
		item := it.Item()
		key := string(item.Key()[:])

		/* Skip keys representing objects
		   namespaces keys can't contain (:)
		 */
		if strings.Contains(key, ":"){
			continue
		}

		// Found a namespace
		counter++

		// Skip this namespace if its index < intended startingIndex
		if counter < startingIndex{
			continue
		}

		value, err := api.db.Get(key)

		// Database Error
		if err != nil{
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}


		if err := json.Unmarshal(value, &namespace); err != nil{
			log.Println("Invalid namespace format")
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		respBody = append(respBody, Namespace{
			NamespaceCreate: namespace,
		})

		if len(respBody) == resultsCount{
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0{
		respBody = []Namespace{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&respBody)
}
