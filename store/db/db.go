package db

// DB interface is the interface defining how to interact with the key value store
type DB interface {
	Get(key string) ([]byte, error)
	Exists(key string) (bool, error)
	// GetAllStartingWith(prefix string, start int, count int) ([][]byte, error)
	// ListAllRecordsStartingWith(prefix string) ([]string, error)
	List(prefix string) ([]string, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Close() error
}
