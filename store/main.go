package main

import (
	"log"
	"net/http"

	"github.com/zero-os/0-stor/store/goraml"

	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
	"os"
	"encoding/json"
	"os/signal"
	"syscall"
)

type Settings struct {
	Dirs struct {
		Meta string `json:"meta"`
		Data string `json:"data"`
	}`json:"dirs"`
}

func loadSettings() Settings{
	var settings Settings
	configFile, err := os.Open("config.json")
	defer configFile.Close()

	if err != nil{
		log.Fatal(err.Error())
	}

	if err :=json.NewDecoder(configFile).Decode(&settings);err != nil{
		log.Fatal(err.Error())
	}
	return settings
}

func gracefulShutdown(db *Badger){
	log.Println("Gracefully closing 0-stor")
	db.Close()
	os.Exit(0)

}

func main() {

	settings := loadSettings()

	// input validator
	validator.SetValidationFunc("multipleOf", goraml.MultipleOf)

	r := mux.NewRouter()

	// home page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// apidocs
	r.PathPrefix("/apidocs/").Handler(http.StripPrefix("/apidocs/", http.FileServer(http.Dir("./apidocs/"))))

	store := &Badger{}
	store.Init(settings.Dirs.Meta, settings.Dirs.Data)

	db := store.New(settings.Dirs.Meta, settings.Dirs.Data)

	NamespacesInterfaceRoutes(r, &NamespacesAPI{db: db})

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)

	defer func() {
		gracefulShutdown(db)
	}()

	go func() {
		<-c
		gracefulShutdown(db)
	}()

	log.Println("starting server")

	if err := http.ListenAndServe(":5000", r); err != nil{
		log.Fatal(err.Error())

	}
}
