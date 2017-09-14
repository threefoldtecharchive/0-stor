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
	units "github.com/docker/go-units"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/config"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/disk"
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
			Usage:       "disable JWT authentification",
			Destination: &settings.AuthDisabled,
			EnvVar:      "STOR_TESTING",
		},
	}

	app.Before = func(c *cli.Context) error {
		if settings.DebugLog {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}

		if settings.AuthDisabled {
			log.Warning("!! Aunthentification disabled, don't use this mode for production!!!")
		}

		return nil
	}

	app.Action = func(c *cli.Context) {
		log.Infoln(app.Name, "version", app.Version)

		if err := ensureStoreStat(settings.DB.Dirs.Data, settings.DB.Dirs.Meta); err != nil {
			log.Fatalln("Error computing store statistics: %v", err)
		}

		server, err := server.New(settings.DB.Dirs.Data, settings.DB.Dirs.Meta, !settings.AuthDisabled)
		if err != nil {
			log.Fatal(err.Error())
		}

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

		addr, err := server.Listen(settings.BindAddress)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Infof("Server interface: grpc")
		log.Infof("Server listening on %s", addr)

		<-sigChan // block on signal handler
		log.Println("Gracefully closing 0-stor")
		server.Close()

		os.Exit(0)
	}

	app.Run(os.Args)
}

func ensureStoreStat(dataPath, metaPath string) error {
	kv, err := badger.New(dataPath, metaPath)
	if err != nil {
		return err
	}
	defer kv.Close()

	nsMgr := manager.NewNamespaceManager(kv)
	statMgr := manager.NewStoreStatMgr(kv)

	available, err := disk.FreeSpace(dataPath)
	if err != nil {
		return err
	}

	namespaces, err := kv.List([]byte(manager.NAMESPACE_PREFIX))
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
