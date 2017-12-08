package etcd

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"
)

// NewClient creates new Metadata client, using an ETCD cluster as storage medium.
// This default constructor uses Proto for the (un)marshaling of metadata values.
func NewClient(endpoints []string) (*Client, error) {
	return NewClientWithEncoding(endpoints, proto.MarshalMetadata, proto.UnmarshalMetadata)
}

// NewClientWithEncoding creates new Metadata client, using an ETCD cluster as storage medium.
// This constructor allows you to use any valid marshal/unmarshal pair.
// All parameters are required.
func NewClientWithEncoding(endpoints []string, marshal encoding.MarshalMetadata, unmarshal encoding.UnmarshalMetadata) (*Client, error) {
	if len(endpoints) == 0 {
		panic("no endpoints given")
	}
	if marshal == nil {
		panic("no metadata-marshal func given")
	}
	if unmarshal == nil {
		panic("no metadata-unmarshal func given")
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: metaOpTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		etcdClient: cli,
		marshal:    marshal,
		unmarshal:  unmarshal,
	}, nil
}

// Client defines client to store metadata,
// using ETCD (v3) as its underlying storage medium.
type Client struct {
	etcdClient *clientv3.Client
	marshal    encoding.MarshalMetadata
	unmarshal  encoding.UnmarshalMetadata
}

// SetMetadata implements metastor.Client.SetMetadata
func (c *Client) SetMetadata(data metastor.Data) error {
	if data.Key == nil {
		return metastor.ErrNilKey
	}

	bytes, err := c.marshal(data)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), metaOpTimeout)
	defer cancel()

	_, err = c.etcdClient.Put(ctx, string(data.Key), string(bytes))
	return err
}

// GetMetadata implements metastor.Client.GetMetadata
func (c *Client) GetMetadata(key []byte) (*metastor.Data, error) {
	if key == nil {
		return nil, metastor.ErrNilKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), metaOpTimeout)
	defer cancel()

	resp, err := c.etcdClient.Get(ctx, string(key))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) < 1 {
		return nil, metastor.ErrNotFound
	}

	var data metastor.Data
	err = c.unmarshal(resp.Kvs[0].Value, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteMetadata implements metastor.Client.DeleteMetadata
func (c *Client) DeleteMetadata(key []byte) error {
	if key == nil {
		return metastor.ErrNilKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), metaOpTimeout)
	defer cancel()

	_, err := c.etcdClient.Delete(ctx, string(key))
	return err
}

// Close implements metastor.Client.Close
func (c *Client) Close() error {
	return c.etcdClient.Close()
}

// Endpoints returns the ETCD endpoints from the ETCD cluster
// used by this client.
func (c *Client) Endpoints() []string {
	return c.etcdClient.Endpoints()
}

const (
	metaOpTimeout = 10 * time.Second
)

var (
	_ metastor.Client = (*Client)(nil)
)
