package grpc

import (
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/jwt"
	pb "github.com/zero-os/0-stor/server/schema"
	"google.golang.org/grpc"
)

// API represents a 0-stor server GRPC API
type API struct {
	db         db.DB
	addr       string
	lis        net.Listener
	grpcServer *grpc.Server
}

// New creates a grpc server with given DB data & meta directory.
// If jwtVerifier is nil, JWT authentification is disabled
func New(data, meta string, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (*API, error) {
	db, err := badger.New(data, meta)
	if err != nil {
		return nil, err
	}
	return NewWithDB(db, jwtVerifier, maxSizeMsg)
}

// NewWithDB creates a grpc server with given DB object
func NewWithDB(db db.DB, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (*API, error) {
	maxSizeMsg = maxSizeMsg * 1024 * 1024 //Mib to Bytes

	if db == nil {
		panic("no database given")
	}

	s := &API{db: db}

	// add authenticating middleware if verifier is provided
	if jwtVerifier != nil {
		s.grpcServer = grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
				streamJWTAuthInterceptor(jwtVerifier),
				streamStatsInterceptor(),
			)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
				unaryJWTAuthInterceptor(jwtVerifier),
				unaryStatsInterceptor(),
			)),
			grpc.MaxRecvMsgSize(maxSizeMsg),
			grpc.MaxSendMsgSize(maxSizeMsg),
		)
	} else {
		s.grpcServer = grpc.NewServer(
			grpc.MaxRecvMsgSize(maxSizeMsg),
			grpc.MaxSendMsgSize(maxSizeMsg),
			grpc.StreamInterceptor(streamStatsInterceptor()),
			grpc.UnaryInterceptor(unaryStatsInterceptor()),
		)
	}

	pb.RegisterObjectManagerServer(s.grpcServer, NewObjectAPI(db))
	pb.RegisterNamespaceManagerServer(s.grpcServer, NewNamespaceAPI(db))
	//pb.RegisterReservationManagerServer(s.grpcServer, &rpc.ReservationManager{db})

	return s, nil
}

// Listen implements API.Listen
func (s *API) Listen(addr string) (string, error) {
	var err error
	s.lis, err = net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	go s.grpcServer.Serve(s.lis)
	s.addr = s.lis.Addr().String()

	return s.addr, nil
}

// Close implements API.Close
func (s *API) Close() {
	log.Infoln("stop grpc server")
	s.grpcServer.GracefulStop()
	log.Infoln("closing database")
	s.db.Close()
}

// DB implements API.DB
func (s *API) DB() db.DB {
	return s.db
}

// ListenAddress implements API.ListenAddress
func (s *API) ListenAddress() string {
	return s.addr
}
