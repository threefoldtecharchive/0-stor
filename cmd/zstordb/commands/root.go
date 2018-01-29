/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/zero-os/0-stor/cmd"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/jwt"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "zstordb",
	Short: "A generic object store server.",
	Long:  `A generic object store server used by zero-os.`,
	RunE:  rootFunc,
	PreRun: func(*cobra.Command, []string) {
		if rootCfg.DebugLog {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}
		if rootCfg.AuthDisabled {
			log.Warning("!!! Authentication disabled, don't use this mode for production !!!")
		}
	},
}

func rootFunc(*cobra.Command, []string) error {
	cmd.LogVersion()

	dbOpts := badgerdb.DefaultOptions
	dbOpts.SyncWrites = rootCfg.SyncWrite
	dbOpts.Dir = rootCfg.DB.Dirs.Meta
	dbOpts.ValueDir = rootCfg.DB.Dirs.Data

	db, err := badger.NewWithOpts(dbOpts)
	if err != nil {
		log.Errorf("error while opening database files: %v", err)
		return err
	}

	serverConfig := grpc.ServerConfig{
		MaxMsgSize: rootCfg.MaxMsgSize,
		JobCount:   rootCfg.JobCount,
	}
	if !rootCfg.AuthDisabled {
		serverConfig.Verifier = jwt.DefaultVerifier()
	}
	storServer, err := grpc.New(db, serverConfig)
	if err != nil {
		log.Errorf("error while creating database layer: %v", err)
		return err
	}

	if rootCfg.ProfileAddress != "" {
		go func() {
			log.Infof("profiling enabled on %v", rootCfg.ProfileAddress)
			if err := http.ListenAndServe(rootCfg.ProfileAddress, http.DefaultServeMux); err != nil {
				log.Panicf("Failed to enable profiling on %v, err:%v", rootCfg.ProfileAddress, err)
			}
		}()
	}

	if rootCfg.ProfileMode.String() != "" {
		stat, err := os.Stat(rootCfg.ProfileOutput)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err := os.MkdirAll(rootCfg.ProfileOutput, 0660); err != nil {
				return fmt.Errorf("fail to create profile output directory: %v", err)
			}
		}
		if !stat.IsDir() {
			return fmt.Errorf("profile-output (%s) is not a directory", rootCfg.ProfileOutput)
		}

		switch rootCfg.ProfileMode {
		case cmd.ProfileModeCPU:
			defer profile.Start(
				profile.NoShutdownHook,
				profile.ProfilePath(rootCfg.ProfileOutput),
				profile.CPUProfile).Stop()
		case cmd.ProfileModeMem:
			defer profile.Start(
				profile.NoShutdownHook,
				profile.ProfilePath(rootCfg.ProfileOutput),
				profile.MemProfile).Stop()
		case cmd.ProfileModeTrace:
			defer profile.Start(
				profile.NoShutdownHook,
				profile.ProfilePath(rootCfg.ProfileOutput),
				profile.TraceProfile).Stop()
		case cmd.ProfileModeBlock:
			defer profile.Start(
				profile.NoShutdownHook,
				profile.ProfilePath(rootCfg.ProfileOutput),
				profile.BlockProfile).Stop()
		}
	}

	var listener net.Listener
	// create a UNIX/TCP listener for our server
	if rootCfg.TLS.CertFile == "" && rootCfg.TLS.KeyFile == "" {
		listener, err = net.Listen(
			rootCfg.ListenAddress.NetworkProtocol(),
			rootCfg.ListenAddress.String())
	} else {
		log.Info("Server will be configured to be secured using TLS")
		cfg := &tls.Config{
			MinVersion: rootCfg.TLS.MinVersion.VersionTLS(),
			MaxVersion: rootCfg.TLS.MaxVersion.VersionTLS(),
		}
		if rootCfg.TLS.CertLiveReload {
			getter, err := newTLSCertificateGetter()
			if err != nil {
				return err
			}
			cfg.GetCertificate = getter.GetCertificate
		} else {
			cert, err := parseCertificate()
			if err != nil {
				return err
			}
			cfg.Certificates = []tls.Certificate{cert}
		}
		listener, err = tls.Listen(
			rootCfg.ListenAddress.NetworkProtocol(),
			rootCfg.ListenAddress.String(),
			cfg)
	}
	if err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	errChan := make(chan error, 1)

	go func() {
		err := storServer.Serve(listener)
		errChan <- err
	}()

	log.Infof("Server interface: grpc")
	log.Infof("Server listening on %s (net protocol: %s)",
		listener.Addr().String(),
		rootCfg.ListenAddress.NetworkProtocol())

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		return storServer.Close()
	}
}

// used to parse a certificate using the configured cert and key files,
// as well as optionally the given key passphrase, which is required
// only if the used PEM key file is encrypted
func parseCertificate() (tls.Certificate, error) {
	// load key
	keyFile, err := ioutil.ReadFile(rootCfg.TLS.KeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	var certKey []byte
	if block, _ := pem.Decode(keyFile); block != nil {
		if x509.IsEncryptedPEMBlock(block) {
			decryptKey, err := x509.DecryptPEMBlock(block, []byte(rootCfg.TLS.KeyPassphrase))
			if err != nil {
				return tls.Certificate{}, err
			}

			privKey, err := x509.ParsePKCS1PrivateKey(decryptKey)
			if err != nil {
				return tls.Certificate{}, err
			}

			certKey = pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privKey),
			})
		} else {
			certKey = pem.EncodeToMemory(block)
		}
	} else {
		return tls.Certificate{}, fmt.Errorf("Invalid Cert Key")
	}

	// load cert
	certFile, err := ioutil.ReadFile(rootCfg.TLS.CertFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	// load and return certificate, pair cert/key
	return tls.X509KeyPair(certFile, certKey)
}

func newTLSCertificateGetter() (*tlsCertificateGetter, error) {
	cert, err := parseCertificate()
	if err != nil {
		return nil, err
	}

	getter := &tlsCertificateGetter{cert: cert}
	go getter.background()
	return getter, nil
}

// used to create a TLS certificate getter,
// which internal certificate can be hot-reloaded (on the same paths as initially configured),
// by signalling a SIGHUP signal.
type tlsCertificateGetter struct {
	mux  sync.RWMutex
	cert tls.Certificate
	err  error
}

func (getter *tlsCertificateGetter) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	getter.mux.RLock()
	cert, err := getter.cert, getter.err
	getter.mux.RUnlock()
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func (getter *tlsCertificateGetter) background() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	var (
		ctx = context.Background()
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigChan:
			getter.mux.Lock()
			getter.cert, getter.err = parseCertificate()
			getter.mux.Unlock()
		}
	}
}

var rootCfg struct {
	DebugLog       bool
	ListenAddress  cmd.ListenAddress
	ProfileAddress string
	ProfileMode    cmd.ProfileMode
	ProfileOutput  string
	AuthDisabled   bool
	MaxMsgSize     int
	SyncWrite      bool
	JobCount       int

	DB struct {
		Dirs struct {
			Meta string
			Data string
		}
	}

	TLS struct {
		// PEM-encoded files, required
		CertFile, KeyFile string
		// Key Passphrase, only required in case the given PEM key file is encrypted
		KeyPassphrase string
		// min/max supported/accepted TLS version, optional
		MinVersion, MaxVersion cmd.TLSVersion
		// Enable in order to be able to live-reload a cert/key pair on runtime
		// by signaling a SIGHUP signal, optional
		CertLiveReload bool
	}
}

func init() {
	rootCmd.AddCommand(cmd.VersionCmd)

	rootCmd.Flags().BoolVarP(
		&rootCfg.DebugLog, "debug", "D", false, "Enable debug logging.")
	rootCmd.Flags().VarP(
		&rootCfg.ListenAddress, "listen", "L", rootCfg.ListenAddress.Description())
	rootCmd.Flags().StringVar(
		&rootCfg.DB.Dirs.Data, "data-dir", ".db/data", "Directory path used to store the data.")
	rootCmd.Flags().StringVar(
		&rootCfg.DB.Dirs.Meta, "meta-dir", ".db/meta", "Directory path used to store the meta data.")
	rootCmd.Flags().StringVar(
		&rootCfg.ProfileAddress, "profile-addr", "", "Enables profiling of this server as an http service.")
	rootCmd.Flags().VarP(
		&rootCfg.ProfileMode, "profile-mode", "", rootCfg.ProfileMode.Description())
	rootCmd.Flags().StringVar(
		&rootCfg.ProfileOutput, "profile-output", ".", "Path of the directory where profiling files are written")
	rootCmd.Flags().BoolVar(
		&rootCfg.AuthDisabled, "no-auth", false, "Disable JWT authentication.")
	rootCmd.Flags().IntVar(
		&rootCfg.MaxMsgSize, "max-msg-size", 32, "Configure the maximum size of the message GRPC server can receive, in MiB")
	rootCmd.Flags().BoolVar(
		&rootCfg.SyncWrite, "sync-write", false, "Enable synchronous writes in BadgerDB (slows down data load significantly).")
	rootCmd.Flags().IntVarP(
		&rootCfg.JobCount, "jobs", "j", grpc.DefaultJobCount,
		"amount of async jobs to run for heavy GRPC server commands")

	// TLS Flags
	// Required TLS Flags
	rootCmd.Flags().StringVar(&rootCfg.TLS.CertFile, "tls-cert", "",
		"TLS certificate used for this server, paired with the given key")
	rootCmd.Flags().StringVar(&rootCfg.TLS.KeyFile, "tls-key", "",
		"TLS private key used for this server, paired with the given cert")
	// TLS Flags only required if other settings/requirements require it
	rootCmd.Flags().StringVar(&rootCfg.TLS.KeyPassphrase, "tls-key-pass", "",
		"Passphrase of the given TLS private key file, only required if that file is encrypted")
	// Optional TLS Flags
	rootCfg.TLS.MinVersion = cmd.TLSVersion11
	rootCmd.Flags().Var(&rootCfg.TLS.MinVersion, "tls-min-version",
		"Minimum supperted/accepted TLS version")
	rootCfg.TLS.MaxVersion = cmd.TLSVersion12
	rootCmd.Flags().Var(&rootCfg.TLS.MaxVersion, "tls-max-version",
		"Maximum supperted/accepted TLS version")
	rootCmd.Flags().BoolVar(&rootCfg.TLS.CertLiveReload, "tls-live-reload", false,
		"Enable in order to support the live reloading of TLS Cert/Key file pairs, when signaling a SIGHUP signal")
}
