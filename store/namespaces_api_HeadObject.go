package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"github.com/zaibon/badger/badger"
)

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the store
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

	opt := badger.DefaultIteratorOptions
	opt.FetchValues = false
	opt.PrefetchSize = api.config.Iterator.PreFetchSize


	it := api.db.store.NewIterator(opt)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		k := string(item.Key()[:])

		if key == k {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}
	}

	http.Error(w, "Object doesn't exist", http.StatusNotFound)
}
