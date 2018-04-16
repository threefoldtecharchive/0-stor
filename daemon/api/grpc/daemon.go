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

package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/datastor/zerodb"
	"github.com/zero-os/0-stor/client/metastor"
	db_utils "github.com/zero-os/0-stor/client/metastor/db/utils"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/processing"
	"github.com/zero-os/0-stor/daemon"
	"github.com/zero-os/0-stor/daemon/api"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Daemon represents a client daemon,
// exposing the client over a GRPC interface.
type Daemon struct {
	grpcServer *grpc.Server
	closer     interface {
		Close() error
	}
}

// Config is used to configure a GRPC daemon manually.
type Config struct {
	// required parameters
	Pipeline   pipeline.Pipeline
	MetaClient *metastor.Client

	MaxMsgSize           int // size in MiB
	DisableLocalFSAccess bool
}

func (cfg *Config) validateAndSanitize() error {
	if cfg.Pipeline == nil {
		return errors.New("no pipeline given, while one is required")
	}
	if cfg.MetaClient == nil {
		return errors.New("no metastor client given, while one is required")
	}

	if cfg.MaxMsgSize <= 0 {
		cfg.MaxMsgSize = DefaultMaxMsgSize
	}
	return nil
}

const (
	// DefaultMaxMsgSize is the default size msg of a server in MiB
	DefaultMaxMsgSize = 32
)

// NewFromConfig creates new daemon with given Config.
func NewFromConfig(cfg daemon.Config, maxMsgSize, jobCount int, disableLocalFSAccess bool) (*Daemon, error) {
	// create data stor cluster
	cluster, err := createDataClusterFromConfig(&cfg)
	if err != nil {
		return nil, err
	}
	// create data pipeline, used for processing of the data
	pipeline, err := pipeline.NewPipeline(cfg.DataStor.Pipeline, cluster, jobCount)
	if err != nil {
		return nil, err
	}

	// create metastor client
	metastorClient, err := createMetastorClientFromConfig(cfg.Namespace, &cfg.MetaStor)
	if err != nil {
		return nil, err
	}

	// create 0-stor master client
	return New(Config{
		Pipeline:             pipeline,
		MetaClient:           metastorClient,
		MaxMsgSize:           maxMsgSize,
		DisableLocalFSAccess: disableLocalFSAccess,
	})
}

func createMetastorClientFromConfig(namespace string, cfg *daemon.MetaStorConfig) (*metastor.Client, error) {
	var (
		err    error
		config metastor.Config
	)

	// create metastor database first,
	// so that then we can create the Metastor client itself

	config.Database, err = db_utils.NewMetaStorDB(cfg.DB.Type, cfg.DB.Config)
	if err != nil {
		return nil, err
	}

	// create the metadata encoding func pair
	config.MarshalFuncPair, err = encoding.NewMarshalFuncPair(cfg.Encoding)
	if err != nil {
		return nil, err
	}

	if len(cfg.Encryption.PrivateKey) == 0 {
		// create potentially insecure metastor storage
		return metastor.NewClientFromConfig([]byte(namespace), config)
	}

	// create the constructor which will create our encrypter-decrypter when needed
	config.ProcessorConstructor = func() (processing.Processor, error) {
		return processing.NewEncrypterDecrypter(
			cfg.Encryption.Type, []byte(cfg.Encryption.PrivateKey))
	}
	// ensure the constructor is valid,
	// as most errors (if not all) are static, and will only fail due to the given input,
	// meaning that if it can be created it now, it should be fine later on as well
	_, err = config.ProcessorConstructor()
	if err != nil {
		return nil, err
	}

	// create our full-configured metastor client,
	// including encryption support for our metadata in binary form
	return metastor.NewClientFromConfig([]byte(namespace), config)
}

func createDataClusterFromConfig(cfg *daemon.Config) (datastor.Cluster, error) {
	// optionally create the global datastor TLS config
	tlsConfig, err := createTLSConfigFromDatastorTLSConfig(&cfg.DataStor.TLS)
	if err != nil {
		return nil, err
	}

	return zerodb.NewCluster(cfg.DataStor.Shards, cfg.Password, cfg.Namespace, tlsConfig)
}

func createTLSConfigFromDatastorTLSConfig(config *client.DataStorTLSConfig) (*tls.Config, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}
	tlsConfig := new(tls.Config)

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	} else {
		log.Warning("TLS is configured to skip verificaitons of certs, " +
			"making the client susceptible to man-in-the-middle attacks!!!")
		tlsConfig.InsecureSkipVerify = true
	}

	if config.RootCA == "" {
		var err error
		tlsConfig.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to create datastor TLS config: %v", err)
		}
	} else {
		tlsConfig.RootCAs = x509.NewCertPool()
		caFile, err := ioutil.ReadFile(config.RootCA)
		if err != nil {
			return nil, err
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caFile) {
			return nil, fmt.Errorf("error reading CA file '%s', while creating datastor TLS config: %v",
				config.RootCA, err)
		}
	}

	return tlsConfig, nil
}

// New creates new daemon with given Config.
func New(cfg Config) (*Daemon, error) {
	// validate our config and sanitize its properties
	err := cfg.validateAndSanitize()
	if err != nil {
		return nil, err
	}

	logrusEntry := log.NewEntry(log.StandardLogger())
	levelOpt := grpc_logrus.WithLevels(CodeToLogrusLevel)

	// create our GRPC server
	maxMsgSize := cfg.MaxMsgSize * 1024 * 1024 // MiB to bytes
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_logrus.StreamServerInterceptor(logrusEntry, levelOpt)),
		grpc.UnaryInterceptor(grpc_logrus.UnaryServerInterceptor(logrusEntry, levelOpt)),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)

	// register the metadata service
	pb.RegisterMetadataServiceServer(grpcServer, newMetadataService(cfg.MetaClient))

	// register the data pipeline service
	pb.RegisterDataServiceServer(grpcServer, newDataService(cfg.Pipeline, cfg.DisableLocalFSAccess))

	// create the master 0-stor client, so we can create the file service
	client := client.NewClient(cfg.MetaClient, cfg.Pipeline)
	pb.RegisterFileServiceServer(grpcServer, newFileService(client, cfg.MetaClient, cfg.DisableLocalFSAccess))

	// return our daemon ready for usage
	return &Daemon{
		grpcServer: grpcServer,
		closer:     client,
	}, nil
}

// Serve implements api.Daemon.Serve
func (d *Daemon) Serve(lis net.Listener) error {
	err := d.grpcServer.Serve(lis)
	if err != nil && !isClosedConnError(err) {
		return err
	}
	return nil
}

// Close implements api.Daemon.Close
func (d *Daemon) Close() error {
	log.Debugln("stop grpc daemon server and all its active listeners")
	d.grpcServer.GracefulStop()
	log.Debugln("closing internal resources")
	return d.closer.Close()
}

// isClosedConnError returns true if the error is from closing listener, cmux.
// copied from golang.org/x/net/http2/http2.go
func isClosedConnError(err error) bool {
	if err == grpc.ErrServerStopped {
		return true
	}
	// 'use of closed network connection' (Go <=1.8)
	// 'use of closed file or network connection' (Go >1.8, internal/poll.ErrClosing)
	// 'mux: listener closed' (cmux.ErrListenerClosed)
	return strings.Contains(err.Error(), "closed")
}

var (
	_ api.Daemon = (*Daemon)(nil)
)
