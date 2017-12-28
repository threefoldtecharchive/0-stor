package grpc

import (
	"net"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/jwt"
	"google.golang.org/grpc"

	"github.com/grpc-ecosystem/go-grpc-middleware"
)

var (
	// DefaultJobCount is the default job count used if the API
	// was created with a job count of 0.
	DefaultJobCount = runtime.NumCPU() * 2
)

const (
	// DefaultMaxSizeMsg is the default size msg of a server
	DefaultMaxSizeMsg = 32
)

// Server represents a 0-stor server GRPC Server API.
type Server struct {
	db         db.DB
	grpcServer *grpc.Server
}

// New creates a GRPC (server) API, using a given Database,
// and optional also custom server options (e.g. authentication middleware)
// Default maxSizeMsg is equal to DefaultMaxSizeMsg.
// Default jobs is equal to DefaultJobCount.
func New(db db.DB, verifier jwt.TokenVerifier, maxSizeMsg, jobs int) (*Server, error) {
	if db == nil {
		panic("no database given")
	}

	if maxSizeMsg <= 0 {
		maxSizeMsg = DefaultMaxSizeMsg
	}
	maxSizeMsg = maxSizeMsg * 1024 * 1024 //Mib to Bytes

	if jobs <= 0 {
		jobs = DefaultJobCount
	}

	s := &Server{db: db}

	// create our grpc server
	if verifier != nil {
		s.grpcServer = grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
				streamJWTAuthInterceptor(verifier),
				streamStatsInterceptor(),
			)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
				unaryJWTAuthInterceptor(verifier),
				unaryStatsInterceptor(),
			)),
			grpc.MaxRecvMsgSize(maxSizeMsg),
			grpc.MaxSendMsgSize(maxSizeMsg),
		)
	} else {
		s.grpcServer = grpc.NewServer(
			grpc.StreamInterceptor(streamStatsInterceptor()),
			grpc.UnaryInterceptor(unaryStatsInterceptor()),
			grpc.MaxRecvMsgSize(maxSizeMsg),
			grpc.MaxSendMsgSize(maxSizeMsg),
		)
	}

	// register our different managers
	pb.RegisterObjectManagerServer(s.grpcServer, NewObjectAPI(db, jobs))
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
