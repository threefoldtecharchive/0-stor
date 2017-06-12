package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-stor/store/goraml"

	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
)

type settings struct {
	BindAddress string `json:"bind"`
	DebugLog    bool   `json:"debug"`

	Dirs struct {
		Meta string `json:"meta"`
		Data string `json:"data"`
	} `json:"dirs"`

	Iterator struct {
		PreFetchSize int `json:"pre_fetch_size"`
	} `json:"iterator"`

	Pagination struct {
		PageSize int `json:"page_size"`
	}
}

func loadSettings(path string) settings {
	var settings settings
	configFile, err := os.Open(path)
	defer configFile.Close()

	if err != nil {
		log.Fatal(err.Error())
	}

	if err := json.NewDecoder(configFile).Decode(&settings); err != nil {
		log.Fatal(err.Error())
	}
	return settings
}

func gracefulShutdown(db *Badger) {
	log.Println("Gracefully closing 0-stor")
	db.Close()
	os.Exit(0)

}

const version = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "0-stor"
	app.Version = version

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	settings := settings{}
	var configPath string
	// settings := loadSettings()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Usage:       "Path to configuration file",
			Destination: &configPath,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Enable debug logging",
			Destination: &settings.DebugLog,
		},
		cli.StringFlag{
			Name:        "bind, b",
			Usage:       "Bind address",
			Value:       ":8080",
			Destination: &settings.BindAddress,
		},
		cli.StringFlag{
			Name:        "data",
			Usage:       "Data directory",
			Value:       "db/data",
			Destination: &settings.Dirs.Data,
		},
		cli.StringFlag{
			Name:        "meta",
			Usage:       "Metadata directory",
			Value:       "db/meta",
			Destination: &settings.Dirs.Meta,
		},
		cli.IntFlag{
			Name:        "pagination",
			Usage:       "Default pagination page size",
			Value:       100,
			Destination: &settings.Pagination.PageSize,
		},
		cli.IntFlag{
			Name:        "prefetch",
			Usage:       "Set pre-fetch size",
			Value:       100,
			Destination: &settings.Iterator.PreFetchSize,
		},
	}

	app.Before = func(c *cli.Context) error {
		if configPath != "" {
			settings = loadSettings(configPath)
		}

		if settings.DebugLog {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}
		// input validator
		validator.SetValidationFunc("multipleOf", goraml.MultipleOf)

		return nil
	}

	app.Action = func(c *cli.Context) {
		log.Infoln(app.Name, "version", app.Version)

		r := mux.NewRouter()

		// home page
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index.html")
		})

		// apidocs
		r.PathPrefix("/apidocs/").Handler(http.StripPrefix("/apidocs/", http.FileServer(http.Dir("./apidocs/"))))

		db, err := NewBadger(settings.Dirs.Meta, settings.Dirs.Data)
		if err != nil {
			log.Fatal(err.Error())
		}

		NamespacesInterfaceRoutes(r, &NamespacesAPI{db: db, config: &settings})

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		defer gracefulShutdown(db)

		go func() {
			<-sigChan
			gracefulShutdown(db)
		}()

		log.Infof("Server listening on %s\n", settings.BindAddress)

		if err := http.ListenAndServe(settings.BindAddress, r); err != nil {
			log.Fatal(err.Error())

		}
	}

	app.Run(os.Args)

}
