package main

import (
	"net/http"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"context"
)

type ReservationValidMiddleware struct {
	db *Badger
	config *settings
}

func NewReservationValidMiddleware(db *Badger, config *settings) *ReservationValidMiddleware {
	return &ReservationValidMiddleware{
		db: db,
		config: config,
	}
}

func (re *ReservationValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("reservation-token")
		if token == ""{
			http.Error(w, "Reservation token is missing", http.StatusUnauthorized)
			return
		}

		nsid := mux.Vars(r)["nsid"]

		res := Reservation{}

		resID, err := res.ValidateReservationToken(token, nsid)

		if err != nil{
			http.Error(w, "Reservation token is invalid", http.StatusUnauthorized)
			return
		}
		res = Reservation{
			Namespace: nsid,
			Reservation: reservation.Reservation{
				Id: resID,
			},
		}

		v, err := res.Get(re.db, re.config)

		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if v == nil{
			http.Error(w, "Reservation token is invalid", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "reservation", v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
