package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db/badger"
	"github.com/zero-os/0-stor/store/goraml"
	"github.com/zero-os/0-stor/store/rest"
	"github.com/zero-os/0-stor/store/rest/models"

	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
)

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
	settings := &config.Settings{}

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
			Destination: &settings.DB.Dirs.Data,
		},
		cli.StringFlag{
			Name:        "meta",
			Usage:       "Metadata directory",
			Value:       "db/meta",
			Destination: &settings.DB.Dirs.Meta,
		},
		cli.IntFlag{
			Name:        "pagination",
			Usage:       "Default pagination page size",
			Value:       100,
			Destination: &settings.DB.Pagination.PageSize,
		},
		cli.IntFlag{
			Name:        "prefetch",
			Usage:       "Set pre-fetch size",
			Value:       100,
			Destination: &settings.DB.Iterator.PreFetchSize,
		},
	}

	app.Before = func(c *cli.Context) error {
		if configPath != "" {
			settings.Load(configPath)
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

		db, err := badger.New(settings)
		if err != nil {
			log.Fatal(err.Error())
		}

		// TODO: is this the correct location to do that ?
		st := models.StoreStat{}
		exists, err := st.Exists(db, settings)
		if err != nil {
			log.Errorln("Database Error")
			log.Errorln(err.Error())
			return
		}

		state := "[USING CURRENT]"

		if !exists {
			s := StoreStat{}
			s.Save(db, settings)
			state = "[CREATED]"
		}

		log.Printf("Global Stats collection: %v\t%s", settings.Store.Stats.Collection, state)

		apiMan := rest.NewAPIManager(db, settings)
		api := rest.NewNamespacesAPI(db, settings)

		NamespacesInterfaceRoutes(r, api)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		go func() {
			log.Infof("Server listening on %s\n", settings.BindAddress)
			if err := http.ListenAndServe(settings.BindAddress, r); err != nil {
				log.Fatal(err.Error())
			}
		}()

		<-sigChan // block on signal handler
		gracefulShutdown(db)
	}

	app.Run(os.Args)
}
