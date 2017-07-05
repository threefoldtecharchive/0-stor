package main

import (
	"fmt"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"

	"strings"

	log "github.com/Sirupsen/logrus"
)

// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	err := api.db.Delete(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete objects in a namespace
	defer func() {
		opt := badger.DefaultIteratorOptions
		opt.FetchValues = false
		opt.PrefetchSize = api.config.Iterator.PreFetchSize

		prefix := fmt.Sprintf("%s:", nsid)

		it := api.db.store.NewIterator(opt)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key()[:])
			if !strings.Contains(key, prefix) {
				continue
			}

			if err := api.db.Delete(key); err != nil {
				log.Errorln(err.Error())

			}
		}
	}()

	// 204 has no body
	http.Error(w, "", http.StatusNoContent)
}
