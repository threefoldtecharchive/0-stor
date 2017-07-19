package middleware

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest/models"
)

type NamespaceStatMiddleware struct {
	db db.DB
}

func NewNamespaceStatMiddleware(db db.DB) *NamespaceStatMiddleware {
	return &NamespaceStatMiddleware{
		db: db,
	}
}

func (nm *NamespaceStatMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		go func() {
			nsid := mux.Vars(r)["nsid"]

			nsStats := models.NamespaceStats{Namespace: nsid}

			b, err := nm.db.Get(nsStats.Key())
			if err != nil {
				if err == db.ErrNotFound {
					log.Errorln("namespace stats for (%s) doesn't exist", nsid)
				}

				log.Errorln(err.Error())
				return
			}

			if err = nsStats.Decode(b); err != nil {
				log.Errorln(err.Error())
				return
			}

			nsStats.NrRequests++

			b, err = nsStats.Encode()
			if err != nil {
				log.Errorln(err.Error())
				return
			}

			if err := nm.db.Set(nsStats.Key(), b); err != nil {
				log.Errorln(err.Error())
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}
