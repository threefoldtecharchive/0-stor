package api

// Server defines the 0-stor Server API interface.
type Server interface {
	// Listen listens to given addr.
	// The server is going be to started as separate goroutine.
	// It listen to random port if the given addr is empty
	// or ended with ":0"
	Listen(string) error

	// Close closes the server
	Close()

	// Address returns the address this server is listening on.
	// This method should only ever be called, after calling listen already.
	Address() string
}
