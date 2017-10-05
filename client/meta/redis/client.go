package redis

import (
	"bytes"

	"github.com/zero-os/0-stor/client/meta"
	"gopkg.in/redis.v3"
)

type Client struct {
	cl *redis.Client
}

func New(addr []string) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr[0],
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, err
	}

	return &Client{
		cl: client,
	}, nil
}

func (c *Client) Put(key string, meta *meta.Meta) error {
	buf := new(bytes.Buffer)
	if err := meta.Encode(buf); err != nil {
		return err
	}

	return c.cl.Set(key, buf.Bytes(), 0).Err()
}
func (c *Client) Get(key string) (*meta.Meta, error) {
	b, err := c.cl.Get(key).Bytes()
	if err != nil {
		return nil, err
	}

	return meta.Decode(b)
}

func (c *Client) Delete(key string) error {
	return c.cl.Del(key).Err()
}

func (c *Client) Close() error {
	return c.cl.Close()
}
