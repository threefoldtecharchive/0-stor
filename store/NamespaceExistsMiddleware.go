package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"context"
	log "github.com/Sirupsen/logrus"
	"fmt"
	"time"
	"github.com/zero-os/0-stor/store/librairies/reservation"
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

		now:= time.Now()

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

		statsKey := fmt.Sprintf("%s_%s", nsid, "stats")

		statsBytes, err := nm.db.Get(statsKey)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		stats := Stat{}

		if err := stats.fromBytes(statsBytes); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if now.After(time.Time(stats.ExpireAt)) {
			http.Error(w, "Reservation expired", http.StatusForbidden)
		}

		resKey := stats.Id

		resBytes, err := nm.db.Get(resKey)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		res := reservation.Reservation{}

		if err := res.FromBytes(resBytes); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "namespace", v)
		ctx = context.WithValue(ctx, "stats", stats)
		ctx = context.WithValue(ctx, "reservation", res)
		ctx = context.WithValue(ctx, "statsKey", statsKey)
		ctx = context.WithValue(ctx, "reservationKey", resKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
