package main

import (
	"net/http"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
	"fmt"
)

type NamespaceStatMiddleware struct {
	db *Badger
}

func NewNamespaceStatMiddleware(db *Badger) *NamespaceStatMiddleware {
	return &NamespaceStatMiddleware{
		db: db,
	}
}

func (nm *NamespaceStatMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nsid := mux.Vars(r)["nsid"]

		statsKey := fmt.Sprintf("%s_stats", nsid)

		value, err := nm.db.Get(statsKey)

		// Database Error
		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// NOT FOUND
		if value == nil{
			log.Errorln("Name space stats not found")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		stat := Stat{}

		if err := stat.fromBytes(value); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return

		}


		stat.NrRequests += 1


		if err := nm.db.Set(statsKey, stat.toBytes()); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
