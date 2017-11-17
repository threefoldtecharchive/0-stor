package db

// DB interface is the interface defining how to interact with the key value store.
// DB is threadsafe
type DB interface {
	// Get gets the value of provided key from the kv store.
	Get(key []byte) ([]byte, error)

	// Exists returns wether a key is present in the kv store.
	Exists(key []byte) (bool, error)

	// Filter returns a list of values from keys with the provided prefix
	// from provided starting index and limited by provided count.
	// Pass count of -1 to get all elements starting from the provided index.
	Filter(prefix []byte, start int, count int) ([][]byte, error)

	// List returns a list of keys with provided prefix.
	List(prefix []byte) ([][]byte, error)

	// Set sets a value to a key in the kv store.
	Set(key []byte, value []byte) error

	// Delete deletes the value and key from the kv store.
	Delete(key []byte) error

	// Close closes the connection to the kv store.
	Close() error
}
