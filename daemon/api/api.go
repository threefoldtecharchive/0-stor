package api

type Daemon interface {
	// Listen listens to given addr.
	Listen(string) error

	// Close closes the daemon
	Close()
}
