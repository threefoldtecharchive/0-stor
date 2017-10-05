package meta

// Client is the interface needed to be used as metadata store
// for the 0-stor client
type Client interface {
	Put(key string, meta *Meta) error
	Get(key string) (*Meta, error)
	Delete(key string) error
	Close() error
}
