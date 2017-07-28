package stor

import (
	"github.com/zero-os/0-stor/client/stor/rest"
)

// Client defines client interface to talk with 0-stor server
type Client interface {
	Store(key, val []byte) (storKey string, err error)
	Get(key []byte) (val []byte, err error)
}

// NewClient creates new client, it currently simply returns the REsT client.
func NewClient(addr, org, namespace, iyoJWTToken string) (Client, error) {
	return rest.NewClient(addr, org, namespace, iyoJWTToken), nil
}
