package router

import (
	"bytes"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest"
	"github.com/zero-os/0-stor/store/rest/models"
)

func LoggingMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(log.StandardLogger().Out, h)
}

func adapt(h http.Handler, adapters ...func(http.Handler) http.Handler) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

func GetRouter(db db.DB, settings config.Settings, enbleJTW bool) http.Handler {
	r := mux.NewRouter()
	iyo := mux.NewRouter()
	dataAcess := mux.NewRouter()
	reservationAccess := mux.NewRouter()

	// home page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := Asset("sdstor.html")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		datareader := bytes.NewReader(data)
		http.ServeContent(w, r, "index.html", time.Now(), datareader)
	})

	var iyoHandler http.Handler
	var dataAccessHandler http.Handler
	var reservationHandler http.Handler
	if enbleJTW {
		iyoHandler = adapt(iyo, rest.NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler, LoggingMiddleware)
		reservationHandler = adapt(reservationAccess, rest.NewReservationValidMiddleware(db).Handler, LoggingMiddleware)
		dataAccessHandler = adapt(dataAcess, rest.NewDataTokenValidMiddleware(models.ACLEntry{}).Handler, LoggingMiddleware)
	} else {
		iyoHandler = adapt(iyo, LoggingMiddleware)
		reservationHandler = adapt(reservationAccess, LoggingMiddleware)
		dataAccessHandler = adapt(dataAcess, LoggingMiddleware)
	}
	//
	r.PathPrefix("/namespaces").Handler(iyoHandler)
	r.PathPrefix("/namespaces/stat").Handler(iyoHandler)
	r.PathPrefix("/namespaces/{nsid}/reservation").Handler(iyoHandler)

	r.PathPrefix("/namespaces/{nsid}").Handler(reservationHandler)
	r.PathPrefix("/namespaces/{nsid}/acl").Handler(reservationHandler)

	r.PathPrefix("/namespaces/{nsid}/objects").Handler(dataAccessHandler)

	nsAPI := rest.NewNamespacesAPI(db, settings)
	rest.NamespacesInterfaceRoutes(r, nsAPI)

	return r
}
