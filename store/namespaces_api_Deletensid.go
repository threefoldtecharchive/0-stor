package main

import (
	"fmt"
	"net/http"

	"github.com/dgraph-io/badger/badger"
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

		storeStat := StoreStat{}
		if err := storeStat.Get(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		namespaceStats := r.Context().Value("namespaceStats").(NamespaceStats)

		storeStat.SizeAvailable += namespaceStats.TotalSizeReserved

		// delete namespacestats
		if err := namespaceStats.Delete(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Save Updated global stats
		if err := storeStat.Save(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Delete namespace itself
		if err:= api.db.Delete(nsid); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Delete reservations
		prefix = fmt.Sprintf("%s%s", api.config.Reservations.Namespaces.Prefix, nsid)
		it2 := api.db.store.NewIterator(opt)
		defer it2.Close()

		for it2.Rewind(); it2.Valid(); it2.Next() {
			item := it2.Item()
			key := string(item.Key()[:])
			if key > "1@"{
				break
			}

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
