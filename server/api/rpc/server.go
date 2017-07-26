package rpc

import (
	"context"
	"net"

	grpc0 "google.golang.org/grpc"
)

type Server struct {
	ctx        context.Context
	lis        net.Listener
	grpcServer *grpc0.Server
}

func New(bind string) (*Server, error) {
	var err error
	s := &Server{
		ctx: context.Background(),
	}

	s.lis, err = net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	s.grpcServer = grpc0.NewServer()

	return s, nil
}

func (s *Server) GRPCServer() *grpc0.Server {
	return s.grpcServer
}

// Serve stats service listening and service connection
// it's blocking
func (s *Server) Serve() error {
	return s.grpcServer.Serve(s.lis)
}
