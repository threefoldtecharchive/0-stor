package main

import (
	"fmt"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"

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

		prefix := []byte(fmt.Sprintf("%s:", nsid))

		it := api.db.store.NewIterator(opt)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := string(item.Key()[:])

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

		namespaceStats := r.Context().Value("namespaceStats").(*NamespaceStats)
		storeStat.SizeAvailable += namespaceStats.TotalSizeReserved
		storeStat.SizeUsed -= namespaceStats.TotalSizeReserved

		// delete namespacestats
		if err := namespaceStats.Delete(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Save Updated global stats
		if err := storeStat.Save(api.db, api.config); err != nil{
			log.Println("save")
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Delete reservations
		namespace := r.Context().Value("namespace").(NamespaceCreate)
		prefix = []byte(namespace.GetKeyForReservations(api.config))
		it2 := api.db.store.NewIterator(opt)
		defer it2.Close()

		for it2.Seek(prefix); it2.ValidForPrefix(prefix); it2.Next() {
			item := it2.Item()
			key := string(item.Key()[:])
			if err := api.db.Delete(key); err != nil {
				log.Errorln(err.Error())

			}
		}
	}()

	// 204 has no body
	http.Error(w, "", http.StatusNoContent)
}
