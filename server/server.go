package server

import (
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/jwt"
	pb "github.com/zero-os/0-stor/server/schema"
	"google.golang.org/grpc"

	log "github.com/Sirupsen/logrus"
)

// StoreServer defines a 0-stor server
type StoreServer interface {
	Listen(string) (string, error)
	Close()
	DB() db.DB
	Addr() string
}

type grpcServer struct {
	db         db.DB
	addr       string
	lis        net.Listener
	grpcServer *grpc.Server
}

// New creates a grpc server with given DB data & meta directory
// if authEnabled is false, JWT authentification is disabled
func New(data, meta string, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (StoreServer, error) {
	db, err := badger.New(data, meta)
	if err != nil {
		return nil, err
	}
	return NewWithDB(db, jwtVerifier, maxSizeMsg)
}

// NewWithDB creates a grpc server with given DB object
func NewWithDB(db db.DB, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (StoreServer, error) {
	maxSizeMsg = maxSizeMsg * 1024 * 1024 //Mib to Bytes

	if db == nil {
		panic("no database given")
	}

	s := &grpcServer{db: db}

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
	//pb.RegisterReservationManagerServer(srv.GRPCServer(), &rpc.ReservationManager{db})

	return s, nil
}

// Listen listens to given addr.
// The server is going be to started as separate goroutine.
// It listen to random port if the given addr is empty
// or ended with ":0"
func (s *grpcServer) Listen(addr string) (string, error) {
	var err error
	s.lis, err = net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	go s.grpcServer.Serve(s.lis)
	s.addr = s.lis.Addr().String()

	return s.addr, nil
}

func (s *grpcServer) Close() {
	log.Infoln("stop grpc server")
	s.grpcServer.GracefulStop()
	log.Infoln("closing database")
	s.db.Close()
}

func (s *grpcServer) DB() db.DB {
	return s.db
}

func (s *grpcServer) Addr() string {
	return s.addr
}
