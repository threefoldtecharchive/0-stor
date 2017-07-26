package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-stor/server/config"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/goraml"
	"github.com/zero-os/0-stor/server/routes"

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

		db, err := badger.New(settings.DB.Dirs.Data, settings.DB.Dirs.Meta)
		if err != nil {
			log.Fatal(err.Error())
		}

		// if err := ensureStoreStat(settings.DB.Dirs.Data, db); err != nil {
		// 	log.Fatalf("Error checking server stats : %v", err)
		// }

		r := routes.GetRouter(db)

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

//
// func ensureStoreStat(path string, db db.DB) error {
// 	available, err := disk.FreeSpace(path)
// 	if err != nil {
// 		return err
// 	}
//
// 	namespaces, err := db.List(models.NAMESPACE_PREFIX)
// 	if err != nil {
// 		return err
// 	}
//
// 	var totalReserved uint64
// 	for _, namespace := range namespaces {
// 		nsStat := models.NamespaceStats{Namespace: namespace}
// 		b, err := db.Get(nsStat.Key())
// 		if err != nil {
// 			return err
// 		}
// 		if err = nsStat.Decode(b); err != nil {
// 			return err
// 		}
// 		totalReserved += nsStat.TotalSizeReserved
// 	}
//
// 	available = available - totalReserved
//
// 	if available <= 0 {
// 		return fmt.Errorf("total reserved size exceed availale disk space")
// 	}
//
// 	stat := models.StoreStat{
// 		SizeUsed:      totalReserved,
// 		SizeAvailable: available,
// 	}
// 	b, err := stat.Encode()
// 	if err != nil {
// 		return err
// 	}
//
// 	if err = db.Set(stat.Key(), b); err != nil {
// 		return err
// 	}
//
// 	log.Infof("Space reserved : %s", units.BytesSize(float64(totalReserved)))
// 	log.Infof("Space available : %s", units.BytesSize(float64(available)))
//
// 	return nil
// }
