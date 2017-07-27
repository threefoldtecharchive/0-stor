// Meta package is WIP package for metadata.
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

type Meta struct {
	Size   uint64
	Key    []byte
	Shards []string
}

func (m Meta) Encode(w io.Writer) error {
	return gob.NewEncoder(w).Encode(m)

}

func (m Meta) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := m.Encode(buf)
	return buf.Bytes(), err
}

type Client struct {
	etcdClient *clientv3.Client
}

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

func (c *Client) Close() {
	c.etcdClient.Close()
}

func (c *Client) Put(key string, meta Meta) error {
	buf := new(bytes.Buffer)
	if err := meta.Encode(buf); err != nil {
		return err
	}
	_, err := c.etcdClient.Put(context.TODO(), key, string(buf.Bytes()))
	return err
}

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

func Decode(p []byte) (*Meta, error) {
	var meta Meta
	return &meta, gob.NewDecoder(bytes.NewReader(p)).Decode(&meta)
}
