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
	"errors"
	"net"
	"strings"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/etcd"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/daemon/api"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	log "github.com/Sirupsen/logrus"
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
	MetaClient metastor.Client

	// optional parameters
	IYOClient            *itsyouonline.Client
	MaxMsgSize           int
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
		cfg.MaxMsgSize = DefaultMaxSizeMsg
	}
	return nil
}

const (
	// DefaultMaxSizeMsg is the default size msg of a server
	DefaultMaxSizeMsg = 32
)

// NewFromClientConfig creates new daemon with given (client) Config.
func NewFromClientConfig(cfg client.Config, maxMsgSize, jobCount int, disableLocalFSAccess bool) (*Daemon, error) {
	// create IYO client if given
	var iyoClient *itsyouonline.Client
	if cfg.IYO != (itsyouonline.Config{}) {
		var err error
		iyoClient, err = itsyouonline.NewClient(cfg.IYO)
		if err != nil {
			return nil, err
		}
	}

	// create data stor cluster
	cluster, err := createDataClusterFromConfig(&cfg, iyoClient)
	if err != nil {
		return nil, err
	}
	// create data pipeline, used for processing of the data
	pipeline, err := pipeline.NewPipeline(cfg.Pipeline, cluster, jobCount)
	if err != nil {
		return nil, err
	}

	// if no metadata shards are given, return an error,
	// as we require a metastor client
	// TODO: allow a more flexible kind of metastor client configuration,
	// so we can also allow other types of metastor clients,
	// as we do really need one.
	if len(cfg.MetaStor.Shards) == 0 {
		return nil, errors.New("no metadata storage given")
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards, nil)
	if err != nil {
		return nil, err
	}

	// create 0-stor master client
	return New(Config{
		Pipeline:             pipeline,
		MetaClient:           metastorClient,
		IYOClient:            iyoClient,
		MaxMsgSize:           maxMsgSize,
		DisableLocalFSAccess: disableLocalFSAccess,
	})
}

func createDataClusterFromConfig(cfg *client.Config, iyoClient *itsyouonline.Client) (datastor.Cluster, error) {
	if iyoClient == nil {
		// create datastor cluster without the use of IYO-backed JWT Tokens,
		// this will only work if all shards use zstordb servers that
		// do not require any authentication
		return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}

	// create JWT Token Getter (Using the earlier created IYO Client)
	var tokenGetter datastor.JWTTokenGetter
	tokenGetter, err := datastor.JWTTokenGetterUsingIYOClient(cfg.IYO.Organization, iyoClient)
	if err != nil {
		return nil, err
	}
	// create cached token getter from this getter, using the default bucket size and count
	tokenGetter, err = datastor.CachedJWTTokenGetter(tokenGetter, -1, -1)
	if err != nil {
		return nil, err
	}

	// create datastor cluster, with the use of IYO-backed JWT Tokens
	return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
}

// New creates new daemon with given Config.
func New(cfg Config) (*Daemon, error) {
	// validate our config and sanitize its properties
	err := cfg.validateAndSanitize()
	if err != nil {
		return nil, err
	}

	// create our GRPC server
	maxMsgSize := cfg.MaxMsgSize * 1024 * 1024 // MiB to bytes
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)

	// register the metadata service
	pb.RegisterMetadataServiceServer(grpcServer, newMetadataService(cfg.MetaClient))

	// register the data pipeline service
	pb.RegisterDataServiceServer(grpcServer, newDataService(cfg.Pipeline, cfg.DisableLocalFSAccess))

	// create the master 0-stor client, so we can create the file service
	client := client.NewClient(cfg.MetaClient, cfg.Pipeline)
	pb.RegisterFileServiceServer(grpcServer, newFileService(client, cfg.DisableLocalFSAccess))

	// register the optional namespace service
	namespaceClient := namespaceClientFromIYOClient(cfg.IYOClient)
	pb.RegisterNamespaceServiceServer(grpcServer, newNamespaceService(namespaceClient))

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
