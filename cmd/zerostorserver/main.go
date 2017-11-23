package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	bagerkv "github.com/dgraph-io/badger"
	units "github.com/docker/go-units"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/config"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/fs"
	"github.com/zero-os/0-stor/server/jwt"
	"github.com/zero-os/0-stor/server/manager"
)

var (
	version = "0.0.1"
	// CommitHash represents the Git commit hash at built time
	CommitHash string
	// BuildDate represents the date when this tool suite was built
	BuildDate string
)

func outputVersion() string {
	// Tool Version
	version := "Version: " + version

	// Build (Git) Commit Hash
	if CommitHash != "" {
		version += "\r\nBuild: " + CommitHash
		if BuildDate != "" {
			version += " " + BuildDate
		}
	}

	// Output version and runtime information
	return fmt.Sprintf("%s\r\nRuntime: %s %s\r\n",
		version,
		runtime.Version(), // Go Version
		runtime.GOOS,      // OS Name
	)
}

func main() {
	app := cli.NewApp()
	app.Name = "zerostorserver"
	app.Usage = "Generic object store used by zero-os"
	app.Description = "Generic object store used by zero-os"
	app.Version = outputVersion()

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	settings := config.Settings{}
	var profileAddr string

	app.Flags = []cli.Flag{
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
			Name:        "profile-addr",
			Usage:       "Enables profiling of this server as an http service",
			Value:       "",
			Destination: &profileAddr,
		},
		cli.BoolFlag{
			Name:        "auth-disable",
			Usage:       "Disable JWT authentification",
			Destination: &settings.AuthDisabled,
			EnvVar:      "STOR_TESTING",
		},
		cli.IntFlag{
			Name:        "max-msg-size",
			Usage:       "Configure the maximum size of the message GRPC server can receive, in MiB",
			Destination: &settings.MaxMsgSize,
			Value:       32,
		},
		cli.BoolFlag{
			Name:        "async-write",
			Usage:       "enable asynchronous writes (default: false)",
			Destination: &settings.AsyncWrite,
		},
	}

	app.Before = func(c *cli.Context) error {
		if settings.DebugLog {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}

		if settings.AuthDisabled {
			log.Warning("!! Authentification disabled, don't use this mode for production!!!")
		}

		return nil
	}

	app.Action = func(c *cli.Context) {
		log.Infoln(app.Name, "version", app.Version)

		dbOpts := bagerkv.DefaultOptions
		dbOpts.SyncWrites = !settings.AsyncWrite

		db, err := badger.NewWithOpts(settings.DB.Dirs.Data, settings.DB.Dirs.Meta, dbOpts)
		if err != nil {
			log.Errorf("error opening database files: %v", err)
			os.Exit(1)
		}

		if err := ensureStoreStat(db, settings.DB.Dirs.Data); err != nil {
			log.Fatalf("Error computing store statistics: %v", err)
		}

		var storServer server.StoreServer
		if settings.AuthDisabled {
			storServer, err = server.NewWithDB(db, nil, settings.MaxMsgSize)
		} else {
			storServer, err = server.NewWithDB(db, jwt.DefaultVerifier(), settings.MaxMsgSize)
		}
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			log.Println("Gracefully closing 0-stor")
			storServer.Close()
		}()

		if profileAddr != "" {
			go func() {
				log.Infof("profiling enabled on %v", profileAddr)
				if err := http.ListenAndServe(profileAddr, http.DefaultServeMux); err != nil {
					log.Infof("Failed to enable profiling on %v, err:%v", profileAddr, err)
				}
			}()
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		addr, err := storServer.Listen(settings.BindAddress)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Infof("Server interface: grpc")
		log.Infof("Server listening on %s", addr)

		<-sigChan // block on signal handler
	}

	app.Run(os.Args)
}

func ensureStoreStat(kv db.DB, dataPath string) error {
	nsMgr := manager.NewNamespaceManager(kv)
	statMgr := manager.NewStoreStatMgr(kv)

	available, err := fs.FreeSpace(dataPath)
	if err != nil {
		return err
	}

	namespaces, err := kv.List([]byte(manager.PrefixNamespace))
	if err != nil {
		return err
	}

	var totalReserved uint64
	for _, namespace := range namespaces {
		ns, err := nsMgr.Get(string(namespace))
		if err != nil {
			return err
		}
		totalReserved += ns.Reserved
	}

	available = available - totalReserved

	if available <= 0 {
		return fmt.Errorf("total reserved size exceed availale disk space")
	}

	if err := statMgr.Set(available, totalReserved); err != nil {
		return err
	}

	log.Infof("Space reserved : %s", units.BytesSize(float64(totalReserved)))
	log.Infof("Space available : %s", units.BytesSize(float64(available)))

	return nil
}
