package api

import "net"

// Server defines the 0-stor Server API interface.
type Server interface {
	// Serve accepts incoming connections on the listener, lis.
	// This function blocks until the given listener, list, is closed.
	// The given listener, lis, is owned by the Server as soon as this function is called,
	// and the server will close any active listeners as part of its GracefulStop method.
	Serve(lis net.Listener) error

	// Close closes the 0-stor server its resources and stops all it open connections gracefully.
	// It stops the server from accepting new connections and blocks until
	// all established connections and other resources have been closed.
	Close() error
}
