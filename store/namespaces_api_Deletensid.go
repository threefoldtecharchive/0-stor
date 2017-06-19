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

		// claim namespaxe size to global stat & delete reservation

		reservationKey := r.Context().Value("reservationKey").(string)
		if err := api.db.Delete(reservationKey); err != nil {
			log.Errorln(err.Error())

		}

		statsBytes := r.Context().Value("stats").([]byte)
		namespaceStats := Stat{}
		namespaceStats.fromBytes(statsBytes)
		size := namespaceStats.SizeReserved

		globalStatBytes, err := api.db.Get(api.config.Stats.CollectionName)
		if err != nil{
			log.Errorln(err.Error())
		}

		globalStats := StoreStat{}
		globalStats.fromBytes(globalStatBytes)
		globalStats.Size += size

		if err:= api.db.Set(api.config.Stats.CollectionName, globalStats.toBytes()); err != nil{
			log.Errorln(err.Error())
		}

		namespaceStatsKey := r.Context().Value("statsKey").(string)
		if err:= api.db.Delete(namespaceStatsKey); err != nil{
			log.Errorln(err.Error())
		}
	}()

	// 204 has no body
	http.Error(w, "", http.StatusNoContent)
}
