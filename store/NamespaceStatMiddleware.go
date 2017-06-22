package main

import (
	"net/http"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
)

type NamespaceStatMiddleware struct {
	db *Badger
	config *settings
}

func NewNamespaceStatMiddleware(db *Badger, config *settings) *NamespaceStatMiddleware {
	return &NamespaceStatMiddleware{
		db: db,
		config: config,
	}
}

func (nm *NamespaceStatMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nsid := mux.Vars(r)["nsid"]

		nsStats := NamespaceStats{Namespace:nsid}

		_, err := nsStats.Get(nm.db, nm.config)
		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		nsStats.NrRequests += 1

		if err := nsStats.Save(nm.db, nm.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
