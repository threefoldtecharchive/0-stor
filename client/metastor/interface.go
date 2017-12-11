package metastor

import "errors"

var (
	// ErrNotFound is the error returned by metadata clients,
	// in case metadata requested using the GetMetadata,
	// couldn't be found.
	ErrNotFound = errors.New("key couldn't be found")

	// ErrNilKey is the error returned by metadata clients,
	// in case a nil key is given as part of a request.
	ErrNilKey = errors.New("nil key given")
)

// Client defines the client API of a metadata server.
// It is used to set, get and delete metadata.
// It is also used as an optional part of the the main 0-stor client,
// in order to fetch the metadata automatically for a given key.
//
// A Client can always be assumed to be thread safe.
type Client interface {
	// SetMetadata sets the metadata,
	// using the key defined as part of the given metadata.
	//
	// An error is returned in case the metadata couldn't be set.
	SetMetadata(meta Data) error

	// GetMetadata returns the metadata linked to the given key.
	//
	// An error is returned in case the linked data couldn't be found.
	// ErrNotFound is returned in case the key couldn't be found.
	// The returned data will always be non-nil in case no error was returned.
	GetMetadata(key []byte) (*Data, error)

	// DeleteMetadata deletes the metadata linked to the given key.
	// It is not considered an error if the metadata was already deleted.
	//
	// If an error is returned it should be assumed
	// that the data couldn't be deleted and might still exist.
	DeleteMetadata(key []byte) error

	// Close any open resources of this metadata client.
	Close() error
}
