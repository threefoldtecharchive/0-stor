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
	"net"
	"runtime"
	"strings"

	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/jwt"

	log "github.com/Sirupsen/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"google.golang.org/grpc"
)

var (
	// DefaultJobCount is the default job count used if the API
	// was created with a job count of 0.
	DefaultJobCount = runtime.NumCPU() * 2
)

const (
	// DefaultMaxMsgsize is the default send/recv msg size of a server
	DefaultMaxMsgsize = 32
)

// Server represents a 0-stor server GRPC Server API.
type Server struct {
	db         db.DB
	grpcServer *grpc.Server
}

// ServerConfig represents all the optional properties,
// that can be configured for a 0-stor server.
// A nil ServerConfig is valid, and will result
// in a server creation using only default options.s
type ServerConfig struct {
	MaxMsgSize int
	JobCount   int
	Verifier   jwt.TokenVerifier
}

// sanitize ensures that all config options
// are in the expected unit. If a property is nil,
// and it has a default value, the default value will be set as well.
func (cfg *ServerConfig) sanitize() {
	if cfg.MaxMsgSize <= 0 {
		cfg.MaxMsgSize = DefaultMaxMsgsize
	}
	cfg.MaxMsgSize *= 1024 * 1024 //Mib to Bytes

	if cfg.JobCount <= 0 {
		cfg.JobCount = DefaultJobCount
	}
}

// New creates a GRPC (server) API, using a given Database,
// and optional also custom server options (e.g. authentication middleware)
// Default maxSizeMsg is equal to DefaultMaxSizeMsg.
// Default jobs is equal to DefaultJobCount.
func New(db db.DB, cfg ServerConfig) (*Server, error) {
	if db == nil {
		panic("no database given")
	}
	cfg.sanitize()

	s := &Server{db: db}

	logrusEntry := log.NewEntry(log.StandardLogger())
	levelOpt := grpc_logrus.WithLevels(CodeToLogrusLevel)

	streamInterceptors := []grpc.StreamServerInterceptor{
		streamStatsInterceptor(),
		grpc_logrus.StreamServerInterceptor(logrusEntry, levelOpt),
	}
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		unaryStatsInterceptor(),
		grpc_logrus.UnaryServerInterceptor(logrusEntry, levelOpt),
	}

	// add our auth interceptor to the front, should a verifier be given
	if cfg.Verifier != nil {
		streamInterceptors = append([]grpc.StreamServerInterceptor{
			streamJWTAuthInterceptor(cfg.Verifier),
		}, streamInterceptors...)
		unaryInterceptors = append([]grpc.UnaryServerInterceptor{
			unaryJWTAuthInterceptor(cfg.Verifier),
		}, unaryInterceptors...)
	}

	// create our grpc server
	s.grpcServer = grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		grpc.MaxRecvMsgSize(cfg.MaxMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxMsgSize),
	)

	// register our different managers
	pb.RegisterObjectManagerServer(s.grpcServer, NewObjectAPI(db, cfg.JobCount))
	pb.RegisterNamespaceManagerServer(s.grpcServer, NewNamespaceAPI(db))

	// return the API server, ready for usage
	return s, nil
}

// Serve implements Server.Serve
func (s *Server) Serve(lis net.Listener) error {
	err := s.grpcServer.Serve(lis)
	if err != nil && !isClosedConnError(err) {
		return err
	}
	return nil
}

// Close implements Server.Close
func (s *Server) Close() error {
	log.Debugln("stop grpc server and all its active listeners")
	s.grpcServer.GracefulStop()
	log.Debugln("closing database")
	return s.db.Close()
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
