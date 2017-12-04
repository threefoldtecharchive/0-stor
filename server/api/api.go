package api

import "github.com/zero-os/0-stor/server/db"

// API defines a 0-stor server API
type API interface {
	// Listen listens to given addr.
	// The server is going be to started as separate goroutine.
	// It listen to random port if the given addr is empty
	// or ended with ":0"
	Listen(string) (string, error)

	// Close closes the server
	Close()

	// Db returns the server's backend database
	DB() db.DB

	// ListenAddress returns the server's listening address
	ListenAddress() string
}
