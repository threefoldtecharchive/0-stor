package client

import (
	"errors"

	"github.com/zero-os/0-stor/client/meta"
)

var (
	errNilMetaClient = errors.New("nil meta client : please specify meta server adddress during client creation")
)

// PutMeta puts metadata to metadata server
func (c *Client) PutMeta(key []byte, md *meta.Data) error {
	if c.metaCli == nil {
		return errNilMetaClient
	}
	if md == nil {
		return c.metaCli.DeleteMetadata(key)
	}
	return c.metaCli.SetMetadata(*md)
}

// GetMeta gets metadata from metadata server
func (c *Client) GetMeta(key []byte) (*meta.Data, error) {
	if c.metaCli == nil {
		return nil, errNilMetaClient
	}
	return c.metaCli.GetMetadata(key)
}
