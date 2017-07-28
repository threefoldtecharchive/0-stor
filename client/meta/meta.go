// Package meta  is WIP package for metadata.
// The spec need to be fixed first for further development.
package meta

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/coreos/etcd/clientv3"
)

// Meta defines a metadata
type Meta struct {
	Size   uint64
	Key    []byte
	Shards []string
}

// Encode encodes the meta to `gob` format
func (m Meta) Encode(w io.Writer) error {
	return gob.NewEncoder(w).Encode(m)

}

// Bytes returns []byte representation of this meta
// in gob format
func (m Meta) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := m.Encode(buf)
	return buf.Bytes(), err
}

// Client defines client to store metadata
type Client struct {
	etcdClient *clientv3.Client
}

// NewClient creates new meta client
func NewClient(shards []string) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: shards,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		etcdClient: cli,
	}, nil
}

// Close closes meta client, release all resources
func (c *Client) Close() {
	c.etcdClient.Close()
}

// Put stores meta to metadata server
func (c *Client) Put(key string, meta Meta) error {
	buf := new(bytes.Buffer)
	if err := meta.Encode(buf); err != nil {
		return err
	}
	_, err := c.etcdClient.Put(context.TODO(), key, string(buf.Bytes()))
	return err
}

// Get fetch metadata from metadata server
func (c *Client) Get(key string) (*Meta, error) {
	resp, err := c.etcdClient.Get(context.TODO(), key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("invalid number of kvs returned: %v", len(resp.Kvs))
	}

	return Decode(resp.Kvs[0].Value)
}

// Decode decodes metadata
func Decode(p []byte) (*Meta, error) {
	var meta Meta
	return &meta, gob.NewDecoder(bytes.NewReader(p)).Decode(&meta)
}
