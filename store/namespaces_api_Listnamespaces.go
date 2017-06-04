package main

import (
	"encoding/json"
	"net/http"
	"github.com/zaibon/badger/badger"
	"strings"
	"log"
)

// Listnamespaces is the handler for GET /namespaces
// List all namespaces
func (api NamespacesAPI) Listnamespaces(w http.ResponseWriter, r *http.Request) {
	var respBody []Namespace
	var namespace NamespaceCreate

	opt := badger.IteratorOptions{}
	opt.FetchValues = api.config.Iterator.FetchValues
	opt.PrefetchSize = api.config.Iterator.FetchSize
	opt.Reverse = false

	it := api.db.store.NewIterator(opt)
	defer it.Close()

	startingIndex := 0
	counter := 0 // Number of namespaces encountered
	resultsCount := 20

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


		// No need to handle errors, we assume data is saved correctly
		json.Unmarshal(value, &namespace)

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

	json.NewEncoder(w).Encode(&respBody)
}
