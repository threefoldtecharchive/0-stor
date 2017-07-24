package stor

import (
	"github.com/zero-os/0-stor-lib/stor/rest"
)

type Client interface {
	Store(key, val []byte) (err error)
	Get(key []byte) (val []byte, err error)
}

func NewClient(addr string) (Client, error) {
	return rest.NewClient(addr), nil
}
