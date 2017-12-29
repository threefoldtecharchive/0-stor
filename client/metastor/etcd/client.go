package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
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
func (c *Client) SetMetadata(data metastor.Metadata) error {
	if len(data.Key) == 0 {
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

// UpdateMetadata implements metastor.Client.UpdateMetadata
func (c *Client) UpdateMetadata(key []byte, cb metastor.UpdateMetadataFunc) (*metastor.Metadata, error) {
	if cb == nil {
		panic("Metastor (etcd) Client: required UpdateMetadata CB is not given")
	}
	if len(key) == 0 {
		return nil, metastor.ErrNilKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), metaOpTimeout)
	defer cancel()

	var (
		output *metastor.Metadata
		keyStr = string(key)
	)
	resp, err := concurrency.NewSTM(c.etcdClient, func(stm concurrency.STM) error {
		// get the metadata
		var (
			input metastor.Metadata
			value = stm.Get(keyStr)
		)
		if len(value) == 0 {
			return metastor.ErrNotFound
		}
		err := c.unmarshal([]byte(value), &input)
		if err != nil {
			return err
		}

		// update the metadata
		output, err = cb(input)
		if err != nil {
			return err
		}

		// store the metadata
		bytes, err := c.marshal(*output)
		if err != nil {
			return err
		}
		stm.Put(keyStr, string(bytes))
		return nil
	}, concurrency.WithPrefetch(keyStr), concurrency.WithAbortContext(ctx))
	if err != nil {
		return nil, err
	}
	if !resp.Succeeded {
		return nil, fmt.Errorf("update of %s didn't succeed", key)
	}
	return output, nil
}

// GetMetadata implements metastor.Client.GetMetadata
func (c *Client) GetMetadata(key []byte) (*metastor.Metadata, error) {
	if len(key) == 0 {
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

	var data metastor.Metadata
	err = c.unmarshal(resp.Kvs[0].Value, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteMetadata implements metastor.Client.DeleteMetadata
func (c *Client) DeleteMetadata(key []byte) error {
	if len(key) == 0 {
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
	metaOpTimeout = 30 * time.Second
)

var (
	_ metastor.Client = (*Client)(nil)
)
