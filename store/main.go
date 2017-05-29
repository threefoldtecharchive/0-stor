package main

import (
	"log"
	"net/http"

	"github.com/Zero-OS/0-stor/store/goraml"

	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
)

func main() {
	// input validator
	validator.SetValidationFunc("multipleOf", goraml.MultipleOf)

	r := mux.NewRouter()

	// home page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// apidocs
	r.PathPrefix("/apidocs/").Handler(http.StripPrefix("/apidocs/", http.FileServer(http.Dir("./apidocs/"))))

	NamespacesInterfaceRoutes(r, NamespacesAPI{})
	log.Println("Initializing db directories")
	db := &Badger{}
	db.Init()
	log.Printf("Meta dir: %v", settings.Dirs.Meta)
	log.Printf("Data dir: %v", settings.Dirs.Data)
	log.Println("starting server")
	http.ListenAndServe(":5000", r)
}
