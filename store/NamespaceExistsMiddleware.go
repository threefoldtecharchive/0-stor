package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"context"
	log "github.com/Sirupsen/logrus"
)

type NamespaceExistsMiddleware struct {
	db *Badger
}

func NewNamespaceExistsMiddleware(db *Badger) *NamespaceExistsMiddleware {
	return &NamespaceExistsMiddleware{
		db: db,
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

		ctx := context.WithValue(r.Context(), "namespace", v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
