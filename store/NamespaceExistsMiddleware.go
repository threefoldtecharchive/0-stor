package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"context"
	log "github.com/Sirupsen/logrus"
)

type NamespaceExistsMiddleware struct {
	db *Badger
	config *settings
}

func NewNamespaceExistsMiddleware(db *Badger, config *settings) *NamespaceExistsMiddleware {
	return &NamespaceExistsMiddleware{
		db: db,
		config: config,
	}
}

func (nm *NamespaceExistsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		nsid := mux.Vars(r)["nsid"]

		v, err :=  nm.db.Get(nsid)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// NOT FOUND
		if v == nil{
			http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
			return
		}

		ns := NamespaceCreate{}
		ns.FromBytes(v)

		stats, err := ns.GetStatsForNamespace(nm.db, nm.config)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "namespace", ns)
		ctx = context.WithValue(ctx, "namespaceStats", stats)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
