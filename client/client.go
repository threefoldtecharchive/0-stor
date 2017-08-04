package client

import (
	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/pipe"
	"github.com/zero-os/0-stor/client/stor"
)

// Client defines 0-stor client
type Client struct {
	conf *config.Config
	stor.Client
	metaCli *meta.Client

	storWriter block.Writer
	storReader block.Reader
}

// New creates new client from the given config
func New(conf *config.Config) (*Client, error) {
	// stor writer
	storWriter, err := pipe.NewWritePipe(conf, block.NewNilWriter())
	if err != nil {
		return nil, err
	}

	// stor reader
	storReader, err := pipe.NewReadPipe(conf)
	if err != nil {
		return nil, err
	}

	client := Client{
		conf:       conf,
		storWriter: storWriter,
		storReader: storReader,
	}

	if conf.StorClient.Shard != "" {
		// 0-stor client
		storClient, err := stor.NewClient(&conf.StorClient, conf.Organization, conf.Namespace)
		if err != nil {
			return nil, err
		}
		client.Client = storClient
	}
	if len(conf.MetaShards) > 0 {
		// meta client
		metaCli, err := meta.NewClient(conf.MetaShards)
		if err != nil {
			return nil, err
		}
		client.metaCli = metaCli
	}
	return &client, nil
}

// Write writes the key-value to the configured pipes
func (c *Client) Write(key, val []byte) error {
	_, err := c.storWriter.WriteBlock(key, val)
	return err
}

// Read reads value with given key from the configured pipes
func (c *Client) Read(key []byte) ([]byte, error) {
	return c.storReader.ReadBlock(key)
}
