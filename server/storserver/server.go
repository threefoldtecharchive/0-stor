package storserver

import (
	"net"
	"net/http"
	"strings"

	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/routes"
)

// Server defines a 0-stor server
type Server struct {
	db    *badger.BadgerDB
	route http.Handler
}

// New creates server with given DB data & meta directory
func New(dbDataDir, dbMetaDir string) (*Server, error) {
	db, err := badger.New(dbDataDir, dbMetaDir)
	if err != nil {
		return nil, err
	}

	return &Server{
		route: routes.GetRouter(db),
		db:    db,
	}, nil
}

// Listen listens to given addr.
// The HTTP server is going be to started as separate goroutine.
// It listen to random port if the given addr is empty
// or ended with ":0"
func (s *Server) Listen(addr string) (string, error) {
	// user specify the port
	// listen on that address
	if !strings.HasSuffix(addr, ":0") || addr != "" {
		go http.ListenAndServe(addr, s.route)
		return addr, nil
	}

	// user doesn't specify the port
	// creates our own listener
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	go http.Serve(l, s.route)

	return l.Addr().String(), nil
}

// Close releases all the resources
func (s *Server) Close() {
	s.db.Close()
}
