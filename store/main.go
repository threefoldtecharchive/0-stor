package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/db/badger"
	"github.com/zero-os/0-stor/store/goraml"
	"github.com/zero-os/0-stor/store/rest/models"
	"github.com/zero-os/0-stor/store/routes"

	"os"
	"os/signal"
	"syscall"

	"gopkg.in/validator.v2"
)

const version = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "0-stor"
	app.Version = version

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	settings := config.Settings{}

	var configPath string

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
			Value:       ".db/data",
			Destination: &settings.DB.Dirs.Data,
		},
		cli.StringFlag{
			Name:        "meta",
			Usage:       "Metadata directory",
			Value:       ".db/meta",
			Destination: &settings.DB.Dirs.Meta,
		},
		cli.StringFlag{
			Name:        "jwt",
			Usage:       "Key used to signed the jwt produced by the 0-stor",
			Destination: &settings.JWTKey,
		},
	}

	app.Before = func(c *cli.Context) error {
		if configPath != "" {
			settings.Load(configPath)
		}

		if settings.JWTKey == "" {
			log.Errorln("JWT key is empty. Please use the -jwt flag")
			os.Exit(1)
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

		db, err := badger.New(settings.DB.Dirs.Data, settings.DB.Dirs.Meta)
		if err != nil {
			log.Fatal(err.Error())
		}

		if err := ensureStoreStat(db); err != nil {
			log.Fatalf("Error checking store stats : %v", err)
		}

		r := routes.GetRouter(db, settings)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		go func() {
			log.Infof("Server listening on %s\n", settings.BindAddress)
			if err := http.ListenAndServe(settings.BindAddress, r); err != nil {
				log.Fatal(err.Error())
			}
		}()

		<-sigChan // block on signal handler
		log.Println("Gracefully closing 0-stor")
		db.Close()
		os.Exit(0)
	}

	app.Run(os.Args)
}

func ensureStoreStat(db db.DB) error {
	exists, err := db.Exists(models.STORE_STATS_PREFIX)
	if err != nil {
		return err
	}

	state := "[USING CURRENT]"

	if !exists {
		s := models.StoreStat{}
		b, err := s.Encode()
		if err != nil {
			return err
		}

		if err = db.Set(s.Key(), b); err != nil {
			return err
		}

		state = "[CREATED]"
	}

	log.Printf("Global Stats collection: %v\t%s", models.STORE_STATS_PREFIX, state)
	return nil
}
