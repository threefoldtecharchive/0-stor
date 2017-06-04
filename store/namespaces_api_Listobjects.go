package main

import (
	"encoding/json"
	"net/http"
	"github.com/zaibon/badger/badger"
	"strings"
	"log"
	"github.com/gorilla/mux"
	"fmt"
	"strconv"
)

// Listobjects is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) Listobjects(w http.ResponseWriter, r *http.Request) { // page := req.FormValue("page")// per_page := req.FormValue("per_page")
	var respBody []Object
	var object Object

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

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	per_page, err := strconv.Atoi(per_pageParam)

	if err != nil{
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]

	value, err := api.db.Get(nsid)

	// Database Error
	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if value == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	prefix := fmt.Sprintf("%s:", nsid)

	opt := badger.IteratorOptions{}
	opt.FetchValues = api.config.Iterator.FetchValues
	opt.PrefetchSize = api.config.Iterator.FetchSize
	opt.Reverse = false

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := (page - 1) * per_page + 1
	counter := 0 // Number of objects encountered
	resultsCount := per_page

	for it.Rewind(); it.Valid(); it.Next(){
		item := it.Item()
		key := string(item.Key()[:])
		/* Skip namespaces and objects not belonging to intended
		   namespace
		 */
		if !strings.Contains(key, prefix){
			continue
		}

		// Found a namespace
		counter++

		// Skip this object if its index < intended startingIndex
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

		// No need to handle errors, we assume data is saved correctly
		json.Unmarshal(value[1:], &object)

		respBody = append(respBody, object)

		if len(respBody) == resultsCount{
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0{
		respBody = []Object{}
	}

	json.NewEncoder(w).Encode(&respBody)
}
