package grpc

import (
	"errors"
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
	address    string
	addressCh  chan string
	listener   net.Listener
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

	s := &Server{
		db:        db,
		addressCh: make(chan string, 1),
	}

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

// Listen implements Server.Listen
func (s *Server) Listen(addr string) error {
	if s.listener != nil {
		return errors.New("server is already listening")
	}

	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.addressCh <- s.listener.Addr().String()

	err = s.grpcServer.Serve(s.listener)
	if err != nil && !isClosedConnError(err) {
		return err
	}
	return nil
}

// isClosedConnError returns true if the error is from closing listener, cmux.
// copied from golang.org/x/net/http2/http2.go
func isClosedConnError(err error) bool {
	// 'use of closed network connection' (Go <=1.8)
	// 'use of closed file or network connection' (Go >1.8, internal/poll.ErrClosing)
	// 'mux: listener closed' (cmux.ErrListenerClosed)
	return strings.Contains(err.Error(), "closed")
}

// Close implements Server.Close
func (s *Server) Close() {
	if s.listener != nil {
		log.Infoln("stop listener")
		s.listener.Close()
	}
	log.Infoln("stop grpc server")
	s.grpcServer.GracefulStop()
	log.Infoln("closing database")
	s.db.Close()
}

// Address implements Server.Address
func (s *Server) Address() string {
	if s.address == "" {
		s.address = <-s.addressCh
	}
	return s.address
}
