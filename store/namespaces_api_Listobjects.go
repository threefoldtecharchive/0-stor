package main

import (
	"encoding/json"
	"net/http"
	"github.com/zaibon/badger/badger"
	"strings"
	"log"
	"github.com/gorilla/mux"
	"fmt"
)

// Listobjects is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) Listobjects(w http.ResponseWriter, r *http.Request) {
	var respBody []Object
	var object Object

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

	startingIndex := 0
	counter := 0 // Number of objects encountered
	resultsCount := 20

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
