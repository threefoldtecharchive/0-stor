package middleware

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/jwt"
	"github.com/zero-os/0-stor/store/rest/models"
)

type ReservationValidMiddleware struct {
	db     db.DB
	jwtKey []byte
}

func NewReservationValidMiddleware(db db.DB, jwtKey []byte) *ReservationValidMiddleware {
	return &ReservationValidMiddleware{
		db:     db,
		jwtKey: jwtKey,
	}
}

func (re *ReservationValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("reservation-token")
		if token == "" {
			http.Error(w, "Reservation token is missing", http.StatusUnauthorized)
			return
		}
		namespace := mux.Vars(r)["nsid"]

		resID, err := jwt.ValidateReservationToken(token, namespace, re.jwtKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Reservation token is invalid: %v", err), http.StatusUnauthorized)
			return
		}

		res := models.Reservation{
			Namespace: namespace,
			Id:        resID,
		}
		b, err := re.db.Get(res.Key())
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "Reservation token is invalid", http.StatusUnauthorized)
				return

			}
			log.Errorln(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err = res.Decode(b); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return

		}
		ctx := context.WithValue(r.Context(), "reservation", res)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
