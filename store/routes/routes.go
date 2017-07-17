package routes

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
)

func LoggingMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(log.StandardLogger().Out, h)
}

func GetRouter(db db.DB, settings config.Settings) http.Handler {
	r := mux.NewRouter()

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

	api := rest.NewNamespacesAPI(db, settings)
	routes := new(rest.HttpRoutes).GetRoutes(api)
	rest.NamespacesInterfaceRoutes(r, routes)

	router := NewRouter(r)
	router.Use(LoggingMiddleware)

	return router.Handler()
}
