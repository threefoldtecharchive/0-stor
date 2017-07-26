package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	validator "gopkg.in/validator.v2"


	//pb "github.com/zero-os/0-stor/server/api/rpc/store"
	"github.com/zero-os/0-stor/server/config"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/goraml"
	"github.com/zero-os/0-stor/server/api/rpc"
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

		srv, err := rpc.New("localhost:8080")
		if err != nil {
			log.Fatalln(err)
		}

		//pb.RegisterObjectManagerServer(srv.GRPCServer(), rpc.ObjectManager{db})
		//pb.RegisterNamespaceManagerServer(srv.GRPCServer(), rpc.NewNamespaceManager(db))
		//pb.RegisterReservationManagerServer(srv.GRPCServer(), &rpc.ReservationManager{db})

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		go func() {
			log.Infof("Server listening on %s\n", settings.BindAddress)
			if err := srv.Serve(); err != nil {
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
