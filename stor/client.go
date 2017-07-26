package stor

import (
	"github.com/zero-os/0-stor-lib/stor/rest"
)

type Client interface {
	Store(key, val []byte) (err error)
	Get(key []byte) (val []byte, err error)
	GetWithStringKey(key string) (val []byte, err error)
}

func NewClient(addr, org, namespace, iyoJWTToken string) (Client, error) {
	return rest.NewClient(addr, org, namespace, iyoJWTToken), nil
}
